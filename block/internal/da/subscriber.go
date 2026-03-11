package da

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// SubscriberHandler is the callback interface for subscription consumers.
// Implementations drive the consumer-specific logic (caching, piping events, etc.).
type SubscriberHandler interface {
	// HandleEvent processes a subscription event.
	// isInline is true if the subscriber successfully claimed this height (via CAS).
	// Returning an error when isInline is true instructs the Subscriber to roll back the localDAHeight.
	HandleEvent(ctx context.Context, ev datypes.SubscriptionEvent, isInline bool) error

	// HandleCatchup is called for each height during sequential catchup.
	// The subscriber advances localDAHeight only after this returns (true, nil).
	// Returning (false, nil) rolls back localDAHeight without triggering a backoff.
	// Returning an error rolls back localDAHeight and triggers a backoff retry.
	HandleCatchup(ctx context.Context, height uint64) error
}

// SubscriberConfig holds configuration for creating a Subscriber.
type SubscriberConfig struct {
	Client      Client
	Logger      zerolog.Logger
	Namespaces  [][]byte // subscribe to all, merge into one channel
	DABlockTime time.Duration
	Handler     SubscriberHandler
	// Deprecated: Remove with https://github.com/evstack/ev-node/issues/3142
	FetchBlockTimestamp bool // the timestamp comes with an extra api call before Celestia v0.29.1-mocha.

	StartHeight uint64 // initial localDAHeight
}

// Subscriber is a shared DA subscription primitive that encapsulates the
// follow/catchup lifecycle. It subscribes to one or more DA namespaces,
// tracks the highest seen DA height, and drives sequential catchup via
// callbacks on SubscriberHandler.
//
// Used by both DAFollower (syncing) and asyncBlockRetriever (forced inclusion).
type Subscriber struct {
	client  Client
	logger  zerolog.Logger
	handler SubscriberHandler

	// namespaces to subscribe on. When multiple, they are merged.
	namespaces [][]byte

	// localDAHeight is only written by catchupLoop (via CAS) and read by
	// followLoop to determine whether inline processing is possible.
	localDAHeight atomic.Uint64

	// highestSeenDAHeight is written by followLoop and read by catchupLoop.
	highestSeenDAHeight atomic.Uint64

	// headReached tracks whether the subscriber has caught up to DA head.
	headReached atomic.Bool

	// catchupSignal wakes catchupLoop when a new height is seen above localDAHeight.
	catchupSignal chan struct{}

	// daBlockTime used as backoff and watchdog base.
	daBlockTime time.Duration

	// Deprecated: Remove with https://github.com/evstack/ev-node/issues/3142
	fetchBlockTimestamp bool

	// lifecycle
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSubscriber creates a new Subscriber.
func NewSubscriber(cfg SubscriberConfig) *Subscriber {
	s := &Subscriber{
		client:              cfg.Client,
		logger:              cfg.Logger,
		handler:             cfg.Handler,
		namespaces:          cfg.Namespaces,
		catchupSignal:       make(chan struct{}, 1),
		daBlockTime:         cfg.DABlockTime,
		fetchBlockTimestamp: cfg.FetchBlockTimestamp,
	}
	s.localDAHeight.Store(cfg.StartHeight)
	s.highestSeenDAHeight.Store(cfg.StartHeight)
	s.catchupSignal <- struct{}{}

	if len(s.namespaces) == 0 {
		s.logger.Warn().Msg("no namespaces configured, subscriber will stay idle")
	}
	return s
}

// Start begins the follow and catchup goroutines.
func (s *Subscriber) Start(ctx context.Context) error {
	if len(s.namespaces) == 0 {
		return nil
	}

	ctx, s.cancel = context.WithCancel(ctx)
	s.wg.Add(2)
	go s.followLoop(ctx)
	go s.catchupLoop(ctx)

	return nil
}

// Stop gracefully stops the background goroutines.
func (s *Subscriber) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// LocalDAHeight returns the current local DA height.
func (s *Subscriber) LocalDAHeight() uint64 {
	return s.localDAHeight.Load()
}

// HighestSeenDAHeight returns the highest DA height seen from the subscription.
func (s *Subscriber) HighestSeenDAHeight() uint64 {
	return s.highestSeenDAHeight.Load()
}

// HasReachedHead returns whether the subscriber has caught up to DA head.
func (s *Subscriber) HasReachedHead() bool {
	return s.headReached.Load()
}

// ---------------------------------------------------------------------------
// Follow loop
// ---------------------------------------------------------------------------

// signalCatchup sends a non-blocking signal to wake catchupLoop.
func (s *Subscriber) signalCatchup() {
	select {
	case s.catchupSignal <- struct{}{}:
	default:
	}
}

// followLoop subscribes to DA blob events and keeps highestSeenDAHeight up to date.
func (s *Subscriber) followLoop(ctx context.Context) {
	defer s.wg.Done()

	s.logger.Info().Msg("starting follow loop")
	defer s.logger.Info().Msg("follow loop stopped")

	for {
		if err := s.runSubscription(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			s.logger.Warn().Err(err).Msg("DA subscription failed, reconnecting")
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.backoff()):
			}
		}
	}
}

// runSubscription opens subscriptions on all namespaces (merging if more than one)
// and processes events until a channel is closed or the watchdog times out.
func (s *Subscriber) runSubscription(ctx context.Context) error {
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	ch, err := s.subscribe(subCtx)
	if err != nil {
		return err
	}

	watchdogTimeout := s.watchdogTimeout()
	watchdog := time.NewTimer(watchdogTimeout)
	defer watchdog.Stop()

	for {
		select {
		case <-subCtx.Done():
			return subCtx.Err()
		case ev, ok := <-ch:
			if !ok {
				return errors.New("subscription channel closed")
			}
			s.updateHighest(ev.Height)

			local := s.localDAHeight.Load()
			isInline := ev.Height == local && s.localDAHeight.CompareAndSwap(local, local+1)

			err = s.handler.HandleEvent(subCtx, ev, isInline)
			if isInline {
				if err == nil {
					s.headReached.Store(true)
				} else {
					s.localDAHeight.Store(local)
					s.logger.Debug().Err(err).Uint64("da_height", ev.Height).
						Msg("inline processing skipped/failed, rolling back")
				}
			}

			watchdog.Reset(watchdogTimeout)
		case <-watchdog.C:
			return errors.New("subscription watchdog: no events received, reconnecting")
		}
	}
}

// subscribe opens subscriptions on all configured namespaces. When there are
// multiple distinct namespaces, channels are merged via mergeSubscriptions.
func (s *Subscriber) subscribe(ctx context.Context) (<-chan datypes.SubscriptionEvent, error) {
	if len(s.namespaces) == 0 {
		return nil, errors.New("no namespaces configured")
	}

	// Subscribe to the first namespace.
	ch, err := s.client.Subscribe(ctx, s.namespaces[0], s.fetchBlockTimestamp)
	if err != nil {
		return nil, fmt.Errorf("subscribe namespace 0: %w", err)
	}

	// Subscribe to additional namespaces and merge.
	for i := 1; i < len(s.namespaces); i++ {
		if bytes.Equal(s.namespaces[i], s.namespaces[0]) {
			continue // Same namespace, skip duplicate.
		}
		ch2, err := s.client.Subscribe(ctx, s.namespaces[i], s.fetchBlockTimestamp)
		if err != nil {
			return nil, fmt.Errorf("subscribe namespace %d: %w", i, err)
		}
		ch = mergeSubscriptions(ctx, ch, ch2)
	}

	return ch, nil
}

// mergeSubscriptions fans two subscription channels into one.
func mergeSubscriptions(
	ctx context.Context,
	ch1, ch2 <-chan datypes.SubscriptionEvent,
) <-chan datypes.SubscriptionEvent {
	out := make(chan datypes.SubscriptionEvent, 16)
	go func() {
		defer close(out)
		for ch1 != nil || ch2 != nil {
			var ev datypes.SubscriptionEvent
			var ok bool
			select {
			case <-ctx.Done():
				return
			case ev, ok = <-ch1:
				if !ok {
					ch1 = nil
					continue
				}
			case ev, ok = <-ch2:
				if !ok {
					ch2 = nil
					continue
				}
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// updateHighest atomically bumps highestSeenDAHeight and signals catchup if needed.
func (s *Subscriber) updateHighest(height uint64) {
	for {
		cur := s.highestSeenDAHeight.Load()
		if height <= cur {
			return
		}
		if s.highestSeenDAHeight.CompareAndSwap(cur, height) {
			s.signalCatchup()
			return
		}
	}
}

// catchupLoop waits for signals and sequentially catches up.
func (s *Subscriber) catchupLoop(ctx context.Context) {
	defer s.wg.Done()

	s.logger.Info().Msg("starting catchup loop")
	defer s.logger.Info().Msg("catchup loop stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.catchupSignal:
			s.runCatchup(ctx)
		}
	}
}

// runCatchup sequentially calls HandleCatchup from localDAHeight to highestSeenDAHeight.
func (s *Subscriber) runCatchup(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		local := s.localDAHeight.Load()
		// CAS claims this height — prevents followLoop from inline-processing.
		if !s.localDAHeight.CompareAndSwap(local, local+1) {
			// followLoop already advanced past this height via inline processing.
			continue
		}

		if err := s.handler.HandleCatchup(ctx, local); err != nil {
			// Roll back so we can retry after backoff.
			s.localDAHeight.Store(local)
			if errors.Is(err, datypes.ErrHeightFromFuture) {
				s.headReached.Store(true)
				return
			}
			if !s.shouldContinueCatchup(ctx, err, local) {
				return
			}
		}
	}
}

// shouldContinueCatchup logs the error and backs off before retrying.
func (s *Subscriber) shouldContinueCatchup(ctx context.Context, err error, daHeight uint64) bool {
	if ctx.Err() != nil {
		return false
	}
	s.logger.Warn().Err(err).Uint64("da_height", daHeight).Msg("catchup error, backing off")
	select {
	case <-ctx.Done():
		return false
	case <-time.After(s.backoff()):
		return true
	}
}

func (s *Subscriber) backoff() time.Duration {
	if s.daBlockTime > 0 {
		return s.daBlockTime
	}
	return 2 * time.Second
}

const subscriberWatchdogMultiplier = 3

func (s *Subscriber) watchdogTimeout() time.Duration {
	if s.daBlockTime > 0 {
		return s.daBlockTime * subscriberWatchdogMultiplier
	}
	return 30 * time.Second
}
