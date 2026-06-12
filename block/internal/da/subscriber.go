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

	"github.com/evstack/ev-node/block/internal/common"
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
	// The subscriber advances localDAHeight only after this returns nil.
	// Returning an error rolls back localDAHeight and triggers a backoff retry.
	// The returned events are the DAHeightEvents produced for this height
	// (may be nil/empty). The subscriber does not interpret them; they are
	// returned so that higher-level callers (via the Subscriber) can inspect
	// the results without coupling to the handler's internals.
	HandleCatchup(ctx context.Context, height uint64) ([]common.DAHeightEvent, error)
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

	// WalkbackChecker is an optional callback invoked after each successful
	// HandleCatchup call. It receives the DA height just processed and the
	// events returned by the handler. If it returns a non-zero DA height,
	// the subscriber rewinds to that height so it re-fetches on the next
	// iteration. Return 0 to continue normally.
	// This is nil for subscribers that don't need walkback (e.g. async block retriever).
	WalkbackChecker func(daHeight uint64, events []common.DAHeightEvent) uint64
}

// Subscriber is a shared DA subscription primitive that encapsulates the
// follow/catchup lifecycle. It subscribes to one or more DA namespaces,
// tracks the highest seen DA height, and drives sequential catchup via
// callbacks on SubscriberHandler.
//
// Used by both DAFollower (syncing) and asyncBlockRetriever (forced inclusion).
type Subscriber struct {
	client          Client
	logger          zerolog.Logger
	handler         SubscriberHandler
	walkbackChecker func(daHeight uint64, events []common.DAHeightEvent) uint64

	// namespaces to subscribe on. When multiple, they are merged.
	namespaces [][]byte

	// localDAHeight is only written by catchupLoop (via CAS) and read by
	// followLoop to determine whether inline processing is possible.
	localDAHeight atomic.Uint64

	// highestSeenDAHeight is written by followLoop and read by catchupLoop.
	highestSeenDAHeight atomic.Uint64

	// headReached tracks whether the subscriber has caught up to DA head.
	headReached atomic.Bool

	// seenSubscriptionEvent tracks whether followLoop has observed at least one
	// subscription event in this run. Without this, startup catchup can stop
	// too early when highestSeen is still just the initial start height.
	seenSubscriptionEvent atomic.Bool

	// catchupSignal wakes catchupLoop when a new height is seen above localDAHeight.
	catchupSignal chan struct{}

	// daBlockTime used as backoff and watchdog base.
	daBlockTime time.Duration

	// Deprecated: Remove with https://github.com/evstack/ev-node/issues/3142
	fetchBlockTimestamp bool

	// lifecycle
	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewSubscriber creates a new Subscriber.
func NewSubscriber(cfg SubscriberConfig) *Subscriber {
	s := &Subscriber{
		client:              cfg.Client,
		logger:              cfg.Logger,
		handler:             cfg.Handler,
		walkbackChecker:     cfg.WalkbackChecker,
		namespaces:          cfg.Namespaces,
		catchupSignal:       make(chan struct{}, 1),
		daBlockTime:         cfg.DABlockTime,
		fetchBlockTimestamp: cfg.FetchBlockTimestamp,
	}
	s.localDAHeight.Store(cfg.StartHeight)
	s.highestSeenDAHeight.Store(cfg.StartHeight)
	if cfg.StartHeight != 0 {
		s.catchupSignal <- struct{}{}
	}

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

	s.lifecycleMu.Lock()
	if s.cancel != nil {
		s.lifecycleMu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(2)
	s.lifecycleMu.Unlock()

	go s.followLoop(ctx)
	go s.catchupLoop(ctx)

	return nil
}

// Stop gracefully stops the background goroutines.
func (s *Subscriber) Stop() {
	s.lifecycleMu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.lifecycleMu.Unlock()

	if cancel != nil {
		cancel()
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

// RewindTo sets localDAHeight back to the given height and signals the catchup
// loop so that DA heights are re-fetched. This is used when the primary source
// (P2P) stalls and DA needs to take over for the missing range.
func (s *Subscriber) RewindTo(daHeight uint64) {
	for {
		cur := s.localDAHeight.Load()
		if daHeight >= cur {
			return
		}
		if s.localDAHeight.CompareAndSwap(cur, daHeight) {
			s.headReached.Store(false)
			s.signalCatchup()
			return
		}
	}
}

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
			s.seenSubscriptionEvent.Store(true)
			s.updateHighest(ev.Height)

			local := s.localDAHeight.Load()
			isInline := ev.Height == local && s.localDAHeight.CompareAndSwap(local, local+1)

			err = s.handler.HandleEvent(subCtx, ev, isInline)
			if isInline {
				if err == nil {
					highest := s.highestSeenDAHeight.Load()
					// Mark head reached only if this inline event is at or beyond
					// the currently observed head; otherwise catchup is still pending.
					if ev.Height >= highest {
						s.headReached.Store(true)
					}
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
		highest := s.highestSeenDAHeight.Load()

		if s.seenSubscriptionEvent.Load() && local > highest {
			s.headReached.Store(true)
			return
		}

		// CAS claims this height — prevents followLoop from inline-processing.
		if !s.localDAHeight.CompareAndSwap(local, local+1) {
			// followLoop already advanced past this height via inline processing.
			continue
		}

		if events, err := s.handler.HandleCatchup(ctx, local); err != nil {
			// Roll back so we can retry after backoff.
			s.localDAHeight.Store(local)
			if errors.Is(err, datypes.ErrHeightFromFuture) && local >= highest {
				s.headReached.Store(true)
				return
			}
			if !s.shouldContinueCatchup(ctx, err, local) {
				return
			}
		} else if s.walkbackChecker != nil {
			if rewindTo := s.walkbackChecker(local, events); rewindTo > 0 {
				s.RewindTo(rewindTo)
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
