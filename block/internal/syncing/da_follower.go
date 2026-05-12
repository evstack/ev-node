package syncing

import (
	"context"
	"errors"
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
}

// daFollower is the concrete implementation of DAFollower.
type daFollower struct {
	subscriber *da.Subscriber
	retriever  DARetriever
	eventSink  common.EventSink
	logger     zerolog.Logger
}

// NewDAFollower creates a new daFollower.
func NewDAFollower(cfg DAFollowerConfig) DAFollower {
	dataNs := cfg.DataNamespace
	if len(dataNs) == 0 {
		dataNs = cfg.Namespace
	}

	f := &daFollower{
		retriever: cfg.Retriever,
		eventSink: cfg.EventSink,
		logger:    cfg.Logger.With().Str("component", "da_follower").Logger(),
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
		return errors.New("skip inline: no complete events")
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

// HandleCatchup retrieves events at a single DA height and pipes them to the event sink.
func (f *daFollower) HandleCatchup(ctx context.Context, daHeight uint64) error {
	events, err := f.retriever.RetrieveFromDA(ctx, daHeight)
	if err != nil {
		if errors.Is(err, datypes.ErrBlobNotFound) {
			return nil
		}
		return err
	}

	for _, event := range events {
		if err := f.eventSink.PipeEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
