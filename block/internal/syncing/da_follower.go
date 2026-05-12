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

type DAFollower interface {
	Start(ctx context.Context) error
	Stop()
	HasReachedHead() bool
}

type daFollower struct {
	subscriber *da.Subscriber
	retriever  DARetriever
	eventSink  common.EventSink
	logger     zerolog.Logger
}

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

type DAFollowerConfig struct {
	Client        da.Client
	Retriever     DARetriever
	Logger        zerolog.Logger
	EventSink     common.EventSink
	Namespace     []byte
	DataNamespace []byte
	StartDAHeight uint64
	DABlockTime   time.Duration
}

func (f *daFollower) Start(ctx context.Context) error {
	return f.subscriber.Start(ctx)
}

func (f *daFollower) Stop() {
	f.subscriber.Stop()
}

func (f *daFollower) HasReachedHead() bool {
	return f.subscriber.HasReachedHead()
}

func (f *daFollower) HandleEvent(ctx context.Context, ev datypes.SubscriptionEvent, isInline bool) error {
	if !isInline {
		return nil
	}
	if len(ev.Blobs) == 0 {
		return errors.New("skip inline: no blobs")
	}

	events := f.retriever.ProcessBlobs(ctx, ev.Blobs, ev.Height)
	if len(events) == 0 {
		return errors.New("skip inline: no complete events")
	}

	for _, event := range events {
		if err := f.eventSink.PipeEvent(ctx, event); err != nil {
			f.logger.Warn().Err(err).Uint64("da_height", ev.Height).
				Msg("failed to pipe inline event, catchup will retry")
			return err
		}
	}

	f.logger.Debug().Uint64("da_height", ev.Height).Int("events", len(events)).
		Msg("processed subscription blobs inline (fast path)")
	return nil
}

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
