package syncing

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
	"github.com/evstack/ev-node/block/internal/da"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// DAFollower subscribes to DA blob events and drives sequential catchup.
//
// Architecture:
//   - followLoop listens on the subscription channel. When caught up, it processes
//     subscription blobs inline (fast path, no DA re-fetch). Otherwise, it updates
//     highestSeenDAHeight and signals the catchup loop.
//   - catchupLoop sequentially retrieves from localDAHeight → highestSeenDAHeight,
//     piping events to the Syncer's heightInCh.
//
// The two goroutines share only atomic state; no mutexes needed.
type DAFollower interface {
	// Start begins the follow and catchup loops.
	Start(ctx context.Context) error
	// Stop cancels the context and waits for goroutines.
	Stop()
	// HasReachedHead returns true once the catchup loop has processed the
	// DA head at least once. Once true, it stays true.
	HasReachedHead() bool
}

// daFollower is the concrete implementation of DAFollower.
type daFollower struct {
	client    da.Client
	retriever DARetriever
	logger    zerolog.Logger

	// pipeEvent sends a DA height event to the syncer's processLoop.
	pipeEvent func(ctx context.Context, event common.DAHeightEvent) error

	// Namespace to subscribe on (header namespace).
	namespace []byte
	// dataNamespace is the data namespace (may equal namespace when header+data
	// share the same namespace). When different, we subscribe to both and merge.
	dataNamespace []byte

	// localDAHeight is only written by catchupLoop and read by followLoop
	// to determine whether a catchup is needed.
	localDAHeight atomic.Uint64

	// highestSeenDAHeight is written by followLoop and read by catchupLoop.
	highestSeenDAHeight atomic.Uint64

	// headReached tracks whether the follower has caught up to DA head.
	headReached atomic.Bool

	// catchupSignal is sent by followLoop to wake catchupLoop when a new
	// height is seen that is above localDAHeight.
	catchupSignal chan struct{}

	// daBlockTime is used as a backoff before retrying after errors.
	daBlockTime time.Duration

	// lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// DAFollowerConfig holds configuration for creating a DAFollower.
type DAFollowerConfig struct {
	Client        da.Client
	Retriever     DARetriever
	Logger        zerolog.Logger
	PipeEvent     func(ctx context.Context, event common.DAHeightEvent) error
	Namespace     []byte
	DataNamespace []byte // may be nil or equal to Namespace
	StartDAHeight uint64
	DABlockTime   time.Duration
}

// NewDAFollower creates a new daFollower.
func NewDAFollower(cfg DAFollowerConfig) DAFollower {
	dataNs := cfg.DataNamespace
	if len(dataNs) == 0 {
		dataNs = cfg.Namespace
	}
	f := &daFollower{
		client:        cfg.Client,
		retriever:     cfg.Retriever,
		logger:        cfg.Logger.With().Str("component", "da_follower").Logger(),
		pipeEvent:     cfg.PipeEvent,
		namespace:     cfg.Namespace,
		dataNamespace: dataNs,
		catchupSignal: make(chan struct{}, 1),
		daBlockTime:   cfg.DABlockTime,
	}
	f.localDAHeight.Store(cfg.StartDAHeight)
	return f
}

// Start begins the follow and catchup goroutines.
func (f *daFollower) Start(ctx context.Context) error {
	f.ctx, f.cancel = context.WithCancel(ctx)

	f.wg.Add(2)
	go f.followLoop()
	go f.catchupLoop()

	f.logger.Info().
		Uint64("start_da_height", f.localDAHeight.Load()).
		Msg("DA follower started")
	return nil
}

// Stop cancels and waits.
func (f *daFollower) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
	f.wg.Wait()
}

// HasReachedHead returns whether the DA head has been reached at least once.
func (f *daFollower) HasReachedHead() bool {
	return f.headReached.Load()
}

// signalCatchup sends a non-blocking signal to wake catchupLoop.
func (f *daFollower) signalCatchup() {
	select {
	case f.catchupSignal <- struct{}{}:
	default:
		// Already signaled, catchupLoop will pick up the new highestSeen.
	}
}

// followLoop subscribes to DA blob events and keeps highestSeenDAHeight up to date.
// When a new height appears above localDAHeight, it wakes the catchup loop.
func (f *daFollower) followLoop() {
	defer f.wg.Done()

	f.logger.Info().Msg("starting follow loop")
	defer f.logger.Info().Msg("follow loop stopped")

	for {
		if err := f.runSubscription(); err != nil {
			if f.ctx.Err() != nil {
				return
			}
			f.logger.Warn().Err(err).Msg("DA subscription failed, reconnecting")
			select {
			case <-f.ctx.Done():
				return
			case <-time.After(f.backoff()):
			}
		}
	}
}

// runSubscription opens subscriptions on both header and data namespaces (if
// different) and processes events until a channel is closed or an error occurs.
// A watchdog timer triggers if no events arrive within watchdogTimeout(),
// causing a reconnect.
func (f *daFollower) runSubscription() error {
	headerCh, err := f.client.Subscribe(f.ctx, f.namespace)
	if err != nil {
		return fmt.Errorf("subscribe header namespace: %w", err)
	}

	// If namespaces differ, subscribe to the data namespace too and fan-in.
	ch := headerCh
	if !bytes.Equal(f.namespace, f.dataNamespace) {
		dataCh, err := f.client.Subscribe(f.ctx, f.dataNamespace)
		if err != nil {
			return fmt.Errorf("subscribe data namespace: %w", err)
		}
		ch = f.mergeSubscriptions(headerCh, dataCh)
	}

	watchdogTimeout := f.watchdogTimeout()
	watchdog := time.NewTimer(watchdogTimeout)
	defer watchdog.Stop()

	for {
		select {
		case <-f.ctx.Done():
			return f.ctx.Err()
		case ev, ok := <-ch:
			if !ok {
				return errors.New("subscription channel closed")
			}
			f.handleSubscriptionEvent(ev)
			watchdog.Reset(watchdogTimeout)
		case <-watchdog.C:
			return errors.New("subscription watchdog: no events received, reconnecting")
		}
	}
}

// mergeSubscriptions fans two subscription channels into one, concatenating
// blobs from both namespaces when events arrive at the same DA height.
func (f *daFollower) mergeSubscriptions(
	headerCh, dataCh <-chan datypes.SubscriptionEvent,
) <-chan datypes.SubscriptionEvent {
	out := make(chan datypes.SubscriptionEvent, 16)
	go func() {
		defer close(out)
		for headerCh != nil || dataCh != nil {
			var ev datypes.SubscriptionEvent
			var ok bool
			select {
			case <-f.ctx.Done():
				return
			case ev, ok = <-headerCh:
				if !ok {
					headerCh = nil
					continue
				}
			case ev, ok = <-dataCh:
				if !ok {
					dataCh = nil
					continue
				}
			}
			select {
			case out <- ev:
			case <-f.ctx.Done():
				return
			}
		}
	}()
	return out
}

// handleSubscriptionEvent processes a subscription event. When the follower is
// caught up (ev.Height == localDAHeight) and blobs are available, it processes
// them inline — avoiding a DA re-fetch round trip. Otherwise, it just updates
// highestSeenDAHeight and lets catchupLoop handle retrieval.
func (f *daFollower) handleSubscriptionEvent(ev datypes.SubscriptionEvent) {
	// Always record the highest height we've seen from the subscription.
	f.updateHighest(ev.Height)

	// Fast path: process blobs inline when caught up.
	// Only fire when ev.Height == localDAHeight to avoid out-of-order processing.
	if len(ev.Blobs) > 0 && ev.Height == f.localDAHeight.Load() {
		events := f.retriever.ProcessBlobs(f.ctx, ev.Blobs, ev.Height)
		for _, event := range events {
			if err := f.pipeEvent(f.ctx, event); err != nil {
				f.logger.Warn().Err(err).Uint64("da_height", ev.Height).
					Msg("failed to pipe inline event, catchup will retry")
				return // catchupLoop already signaled via updateHighest
			}
		}
		// Only advance if we produced at least one complete event.
		// With split namespaces, the first namespace's blobs may not produce
		// events until the second namespace's blobs arrive at the same height.
		if len(events) != 0 {
			f.localDAHeight.Store(ev.Height + 1)
			f.headReached.Store(true)
			f.logger.Debug().Uint64("da_height", ev.Height).Int("events", len(events)).
				Msg("processed subscription blobs inline (fast path)")
		}
		return
	}

	// Slow path: behind or no blobs — catchupLoop will handle via signal from updateHighest.
}

// updateHighest atomically bumps highestSeenDAHeight and signals catchup if needed.
func (f *daFollower) updateHighest(height uint64) {
	for {
		cur := f.highestSeenDAHeight.Load()
		if height <= cur {
			return
		}
		if f.highestSeenDAHeight.CompareAndSwap(cur, height) {
			f.logger.Debug().
				Uint64("da_height", height).
				Msg("new highest DA height seen from subscription")
			f.signalCatchup()
			return
		}
	}
}

// catchupLoop waits for signals and sequentially retrieves DA heights
// from localDAHeight up to highestSeenDAHeight.
func (f *daFollower) catchupLoop() {
	defer f.wg.Done()

	f.logger.Info().Msg("starting catchup loop")
	defer f.logger.Info().Msg("catchup loop stopped")

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-f.catchupSignal:
			f.runCatchup()
		}
	}
}

// runCatchup sequentially retrieves from localDAHeight to highestSeenDAHeight.
// It handles priority heights first, then sequential heights.
func (f *daFollower) runCatchup() {
	for {
		if f.ctx.Err() != nil {
			return
		}

		// Check for priority heights from P2P hints first.
		if priorityHeight := f.retriever.PopPriorityHeight(); priorityHeight > 0 {
			currentHeight := f.localDAHeight.Load()
			if priorityHeight < currentHeight {
				continue
			}
			f.logger.Debug().
				Uint64("da_height", priorityHeight).
				Msg("fetching priority DA height from P2P hint")
			if err := f.fetchAndPipeHeight(priorityHeight); err != nil {
				if !f.waitOnCatchupError(err, priorityHeight) {
					return
				}
			}
			continue
		}

		// Sequential catchup.
		local := f.localDAHeight.Load()
		highest := f.highestSeenDAHeight.Load()

		if local > highest {
			// Caught up.
			f.headReached.Store(true)
			return
		}

		if err := f.fetchAndPipeHeight(local); err != nil {
			if !f.waitOnCatchupError(err, local) {
				return
			}
			// Retry the same height after backoff.
			continue
		}

		// Advance local height.
		f.localDAHeight.Store(local + 1)
	}
}

// fetchAndPipeHeight retrieves events at a single DA height and pipes them
// to the syncer.
func (f *daFollower) fetchAndPipeHeight(daHeight uint64) error {
	events, err := f.retriever.RetrieveFromDA(f.ctx, daHeight)
	if err != nil {
		switch {
		case errors.Is(err, datypes.ErrBlobNotFound):
			// No blobs at this height — not an error, just skip.
			return nil
		case errors.Is(err, datypes.ErrHeightFromFuture):
			// DA hasn't produced this height yet — mark head reached
			// but return the error to trigger a backoff retry.
			f.headReached.Store(true)
			return err
		default:
			return err
		}
	}

	for _, event := range events {
		if err := f.pipeEvent(f.ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// errCaughtUp is a sentinel used to signal that the catchup loop has reached DA head.
var errCaughtUp = errors.New("caught up with DA head")

// waitOnCatchupError logs the error and backs off before retrying.
// It returns true if the caller should continue (retry), or false if the
// catchup loop should exit (context cancelled or caught-up sentinel).
func (f *daFollower) waitOnCatchupError(err error, daHeight uint64) bool {
	if errors.Is(err, errCaughtUp) {
		f.logger.Debug().Uint64("da_height", daHeight).Msg("DA catchup reached head, waiting for subscription signal")
		return false
	}
	if f.ctx.Err() != nil {
		return false
	}
	f.logger.Warn().Err(err).Uint64("da_height", daHeight).Msg("catchup error, backing off")
	select {
	case <-f.ctx.Done():
		return false
	case <-time.After(f.backoff()):
		return true
	}
}

// backoff returns the configured DA block time or a sane default.
func (f *daFollower) backoff() time.Duration {
	if f.daBlockTime > 0 {
		return f.daBlockTime
	}
	return 2 * time.Second
}

// watchdogTimeout returns how long to wait for a subscription event before
// assuming the subscription is stalled. Defaults to 3× the DA block time.
const watchdogMultiplier = 3

func (f *daFollower) watchdogTimeout() time.Duration {
	if f.daBlockTime > 0 {
		return f.daBlockTime * watchdogMultiplier
	}
	return 30 * time.Second
}
