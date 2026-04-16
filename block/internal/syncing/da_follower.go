package syncing

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// DAFollower follows DA blob events and drives sequential catchup
// using a shared da.Subscriber for the subscription plumbing.
type DAFollower interface {
	Start(ctx context.Context) error
	Stop()
	HasReachedHead() bool
	// QueuePriorityHeight queues a DA height for priority retrieval (from P2P hints).
	QueuePriorityHeight(daHeight uint64)
}

// daFollower is the concrete implementation of DAFollower.
type daFollower struct {
	subscriber    *da.Subscriber
	retriever     DARetriever
	eventSink     common.EventSink
	logger        zerolog.Logger
	nodeHeightFn  func() uint64
	p2pStalledFn  func() bool
	startDAHeight uint64

	// walkbackActive is set when the follower detects a gap between the
	// DA events it just processed and the node's current block height.
	// While active, every DA height (even empty ones) triggers a rewind
	// so the subscriber walks backwards until the gap is filled.
	walkbackActive atomic.Bool

	// Priority queue for P2P hint heights (absorbed from DARetriever refactoring #2).
	priorityMu      sync.Mutex
	priorityHeights []uint64
}

const maxPriorityHeights = 1024

// DAFollowerConfig holds configuration for creating a DAFollower.
type DAFollowerConfig struct {
	Client        da.Client
	Retriever     DARetriever
	Logger        zerolog.Logger
	EventSink     common.EventSink
	Namespace     []byte
	DataNamespace []byte // may be nil or equal to Namespace
	StartDAHeight uint64
	DABlockTime   time.Duration
	// NodeHeight returns the node's current block height. Used together
	// with P2PStalled to detect gaps that need a DA walkback.
	NodeHeight func() uint64
	// P2PStalled returns true when the P2P sync worker has failed to
	// deliver blocks. The follower only walks back when P2P is stalled.
	P2PStalled func() bool
}

// NewDAFollower creates a new daFollower.
func NewDAFollower(cfg DAFollowerConfig) DAFollower {
	dataNs := cfg.DataNamespace
	if len(dataNs) == 0 {
		dataNs = cfg.Namespace
	}

	f := &daFollower{
		retriever:       cfg.Retriever,
		eventSink:       cfg.EventSink,
		logger:          cfg.Logger.With().Str("component", "da_follower").Logger(),
		nodeHeightFn:    cfg.NodeHeight,
		p2pStalledFn:    cfg.P2PStalled,
		startDAHeight:   cfg.StartDAHeight,
		priorityHeights: make([]uint64, 0),
	}

	f.subscriber = da.NewSubscriber(da.SubscriberConfig{
		Client:      cfg.Client,
		Logger:      cfg.Logger,
		Namespaces:  [][]byte{cfg.Namespace, dataNs},
		DABlockTime: cfg.DABlockTime,
		Handler:     f,
		StartHeight: cfg.StartDAHeight,
	})

	return f
}

// Start begins the follow and catchup goroutines.
func (f *daFollower) Start(ctx context.Context) error {
	return f.subscriber.Start(ctx)
}

// Stop gracefully stops the background goroutines.
func (f *daFollower) Stop() {
	f.subscriber.Stop()
}

// HasReachedHead returns whether the follower has caught up to DA head.
func (f *daFollower) HasReachedHead() bool {
	return f.subscriber.HasReachedHead()
}

// HandleEvent processes a subscription event. When the follower is
// caught up (ev.Height == localDAHeight) and blobs are available, it processes
// them inline — avoiding a DA re-fetch round trip. Otherwise, it just lets
// the catchup loop handle retrieval.
func (f *daFollower) HandleEvent(ctx context.Context, ev datypes.SubscriptionEvent, isInline bool) error {
	if !isInline {
		return nil // skip: let subscriber just update highestSeenDAHeight
	}
	if len(ev.Blobs) == 0 {
		return errors.New("skip inline: no blobs") // subscriber rolls back, catch-up loop will retry
	}

	events := f.retriever.ProcessBlobs(ctx, ev.Blobs, ev.Height)
	if len(events) == 0 {
		return errors.New("skip inline: no complete events") // Split namespace, subscriber rolls back
	}

	for _, event := range events {
		if err := f.eventSink.PipeEvent(ctx, event); err != nil {
			f.logger.Warn().Err(err).Uint64("da_height", ev.Height).
				Msg("failed to pipe inline event, catchup will retry")
			return err // Actual pipe failure, subscriber rolls back
		}
	}

	f.logger.Debug().Uint64("da_height", ev.Height).Int("events", len(events)).
		Msg("processed subscription blobs inline (fast path)")
	return nil
}

// HandleCatchup retrieves events at a single DA height and pipes them
// to the event sink. Checks priority heights first.
//
// When a node-height callback is configured, HandleCatchup detects gaps
// between the block heights it just fetched and the node's current height.
// If the smallest block height is above nodeHeight+1 the subscriber is
// rewound by one DA height so it re-fetches the previous height on the
// next iteration. This "walk-back" continues automatically through empty
// DA heights until blocks contiguous with the node are found.
func (f *daFollower) HandleCatchup(ctx context.Context, daHeight uint64) error {
	// 1. Drain stale or future priority heights from P2P hints
	for priorityHeight := f.popPriorityHeight(); priorityHeight != 0; priorityHeight = f.popPriorityHeight() {
		if priorityHeight < daHeight {
			continue // skip stale hints without yielding back to the catchup loop
		}

		f.logger.Debug().
			Uint64("da_height", priorityHeight).
			Msg("fetching priority DA height from P2P hint")

		if _, err := f.fetchAndPipeHeight(ctx, priorityHeight); err != nil {
			if errors.Is(err, datypes.ErrHeightFromFuture) {
				f.logger.Debug().Uint64("priority_da_height", priorityHeight).
					Msg("priority hint is from future, ignoring")
				continue
			}
			return err
		}
		break // continue with daHeight
	}

	// 2. Normal sequential fetch
	events, err := f.fetchAndPipeHeight(ctx, daHeight)
	if err != nil {
		return err
	}

	// 3. Self-correction: walk back when P2P has stalled and DA blocks skip
	//    past the node height. Only active when P2P is confirmed stalled to
	//    avoid unnecessary rewinds during normal DA catchup.
	p2pStalled := f.p2pStalledFn != nil && f.p2pStalledFn()
	if p2pStalled && f.nodeHeightFn != nil && daHeight > f.startDAHeight {
		nodeHeight := f.nodeHeightFn()

		needsWalkback := f.walkbackActive.Load()
		if len(events) > 0 {
			minHeight := events[0].Header.Height()
			for _, e := range events[1:] {
				if e.Header.Height() < minHeight {
					minHeight = e.Header.Height()
				}
			}
			if minHeight <= nodeHeight+1 {
				f.walkbackActive.Store(false)
				return nil
			}
			needsWalkback = true
		}

		if needsWalkback {
			f.walkbackActive.Store(true)
			f.logger.Info().
				Uint64("da_height", daHeight).
				Uint64("node_height", nodeHeight).
				Int("events", len(events)).
				Msg("P2P stalled with gap between DA blocks and node height, walking DA follower back")
			f.subscriber.RewindTo(daHeight - 1)
		}
	} else if !p2pStalled {
		f.walkbackActive.Store(false)
	}

	return nil
}

// fetchAndPipeHeight retrieves events at a single DA height and pipes them.
// It does NOT handle ErrHeightFromFuture — callers must decide how to react
// because the correct response depends on whether this is a normal sequential
// catchup or a priority-hint fetch.
func (f *daFollower) fetchAndPipeHeight(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
	events, err := f.retriever.RetrieveFromDA(ctx, daHeight)
	if err != nil {
		if errors.Is(err, datypes.ErrBlobNotFound) {
			return nil, nil
		}
		return nil, err
	}

	for _, event := range events {
		if err := f.eventSink.PipeEvent(ctx, event); err != nil {
			return nil, err
		}
	}

	return events, nil
}

// QueuePriorityHeight queues a DA height for priority retrieval.
func (f *daFollower) QueuePriorityHeight(daHeight uint64) {
	f.priorityMu.Lock()
	defer f.priorityMu.Unlock()

	idx, found := slices.BinarySearch(f.priorityHeights, daHeight)
	if found {
		return
	}

	// Keep the queue bounded. When full, prefer lower (sooner) heights.
	if len(f.priorityHeights) >= maxPriorityHeights {
		last := f.priorityHeights[len(f.priorityHeights)-1]
		if daHeight >= last {
			return
		}
		f.priorityHeights = f.priorityHeights[:len(f.priorityHeights)-1]
	}

	f.priorityHeights = slices.Insert(f.priorityHeights, idx, daHeight)
}

// popPriorityHeight returns the next priority height to fetch, or 0 if none.
func (f *daFollower) popPriorityHeight() uint64 {
	f.priorityMu.Lock()
	defer f.priorityMu.Unlock()

	if len(f.priorityHeights) == 0 {
		return 0
	}
	height := f.priorityHeights[0]
	f.priorityHeights = f.priorityHeights[1:]
	if len(f.priorityHeights) == 0 {
		f.priorityHeights = nil
	}
	return height
}
