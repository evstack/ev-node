package syncing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
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
	subscriber *da.Subscriber
	retriever  DARetriever
	eventSink  common.EventSink
	logger     zerolog.Logger
	onSkip     func(context.Context, uint64) error

	// Accessed only by the sequential catch-up goroutine.
	failedHeight  uint64
	fetchFailures uint8

	// Priority queue for P2P hint heights (absorbed from DARetriever refactoring #2).
	priorityMu      sync.Mutex
	priorityHeights []uint64
}

const maxPriorityHeights = 1024

// maxDAFetchAttempts bounds retries of unavailable historical DA heights.
const maxDAFetchAttempts uint8 = 10

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
	// OnSkip persists a height skipped after repeated retrieval failures.
	// Returning an error keeps catch-up at that height until persistence succeeds.
	OnSkip func(context.Context, uint64) error
}

// NewDAFollower creates a new daFollower.
func NewDAFollower(cfg DAFollowerConfig) DAFollower {
	dataNs := cfg.DataNamespace
	if len(dataNs) == 0 {
		dataNs = cfg.Namespace
	}

	f := &daFollower{
		onSkip:          cfg.OnSkip,
		retriever:       cfg.Retriever,
		eventSink:       cfg.EventSink,
		logger:          cfg.Logger.With().Str("component", "da_follower").Logger(),
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
func (f *daFollower) HandleCatchup(ctx context.Context, daHeight uint64) error {
	// 1. Drain stale or future priority heights from P2P hints
	for priorityHeight := f.popPriorityHeight(); priorityHeight != 0; priorityHeight = f.popPriorityHeight() {
		if priorityHeight <= daHeight {
			continue // sequential retrieval handles the current height
		}

		f.logger.Debug().
			Uint64("da_height", priorityHeight).
			Msg("fetching priority DA height from P2P hint")

		if retrievalFailed, err := f.fetchAndPipeHeight(ctx, priorityHeight); err != nil {
			if errors.Is(err, datypes.ErrHeightFromFuture) {
				// Priority hint points to a future height — silently ignore.
				f.logger.Debug().Uint64("priority_da_height", priorityHeight).
					Msg("priority hint is from future, ignoring")
				continue
			}
			if retrievalFailed {
				f.logger.Warn().Err(err).Uint64("priority_da_height", priorityHeight).
					Msg("priority DA retrieval failed, continuing sequential catch-up")
				break
			}
			return err // event delivery failures must still be retried
		}
		break // continue with daHeight
	}

	// 2. Normal sequential fetch. Only retrieval failures count toward the
	// limit: event delivery must succeed, and future heights must remain pending.
	retrievalFailed, err := f.fetchAndPipeHeight(ctx, daHeight)
	if !retrievalFailed {
		f.fetchFailures = 0
		return err
	}
	if ctx.Err() != nil || errors.Is(err, datypes.ErrHeightFromFuture) {
		return err
	}
	if f.failedHeight != daHeight {
		f.failedHeight = daHeight
		f.fetchFailures = 0
	}
	if f.fetchFailures < maxDAFetchAttempts {
		f.fetchFailures++
	}
	if f.fetchFailures < maxDAFetchAttempts {
		return err // subscriber backs off and retries the same height
	}
	// During an outage the error may not identify a future height. Never
	// skip past the observed head just because the transport is unavailable.
	if f.subscriber != nil && daHeight > f.subscriber.HighestSeenDAHeight() {
		return err
	}
	if f.onSkip != nil {
		if skipErr := f.onSkip(ctx, daHeight); skipErr != nil {
			return fmt.Errorf("persist skipped DA height %d: %w", daHeight, skipErr)
		}
	}
	f.logger.Warn().Err(err).Uint64("da_height", daHeight).
		Uint8("attempts", f.fetchFailures).Msg("skipping DA height after repeated retrieval failures")
	f.fetchFailures = 0
	return nil // subscriber advances to the next DA height
}

// fetchAndPipeHeight retrieves and pipes events, reporting whether an error
// came from retrieval so event delivery failures cannot trigger a skipped height.
// It does NOT handle ErrHeightFromFuture — callers must decide how to react
// because the correct response depends on whether this is a normal sequential
// catchup or a priority-hint fetch.
func (f *daFollower) fetchAndPipeHeight(ctx context.Context, daHeight uint64) (retrievalFailed bool, err error) {
	events, err := f.retriever.RetrieveFromDA(ctx, daHeight)
	if err != nil {
		if errors.Is(err, datypes.ErrBlobNotFound) {
			return false, nil
		}
		return true, err
	}

	for _, event := range events {
		if err := f.eventSink.PipeEvent(ctx, event); err != nil {
			return false, err
		}
	}

	return false, nil
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
