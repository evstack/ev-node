package syncing

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// TestDAFollower_BackoffOnCatchupError verifies that the DAFollower implements
// proper backoff behavior when encountering different types of DA layer errors.
func TestDAFollower_BackoffOnCatchupError(t *testing.T) {
	tests := map[string]struct {
		daBlockTime    time.Duration
		error          error
		expectsBackoff bool
		exitsCatchup   bool // ErrCaughtUp → clean exit, no backoff, no advance
		description    string
	}{
		"generic_error_triggers_backoff": {
			daBlockTime:    1 * time.Second,
			error:          errors.New("network failure"),
			expectsBackoff: true,
			description:    "Generic DA errors should trigger backoff",
		},
		"height_from_future_stops_catchup": {
			daBlockTime:  500 * time.Millisecond,
			error:        datypes.ErrHeightFromFuture,
			exitsCatchup: true,
			description:  "Height from future should stop catchup (da.ErrCaughtUp), not backoff",
		},
		"blob_not_found_no_backoff": {
			daBlockTime:    1 * time.Second,
			error:          datypes.ErrBlobNotFound,
			expectsBackoff: false,
			description:    "ErrBlobNotFound should not trigger backoff",
		},
		"zero_block_time_fallback": {
			daBlockTime:    0, // Should fallback to 2s
			error:          errors.New("some error"),
			expectsBackoff: true,
			description:    "Zero block time should use 2s fallback",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()

				daRetriever := NewMockDARetriever(t)

				pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

				// Replace manual test usages with Config.
				daClient, eventCh := setupMockDAClient(t)
				follower := NewDAFollower(DAFollowerConfig{
					Client:        daClient,
					Retriever:     daRetriever,
					Logger:        zerolog.Nop(),
					EventSink:     common.EventSinkFunc(pipeEvent),
					Namespace:     []byte("ns"),
					StartDAHeight: 100,
					DABlockTime:   tc.daBlockTime,
				}).(*daFollower)

				ctx, subCancel := context.WithCancel(ctx)
				defer subCancel()

				follower.Start(ctx)
				eventCh <- datypes.SubscriptionEvent{Height: 102}

				var callTimes []time.Time
				callCount := 0

				// First call - returns test error
				daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Run(func(args mock.Arguments) {
						callTimes = append(callTimes, time.Now())
						callCount++
					}).
					Return(nil, tc.error).Once()

				if tc.exitsCatchup {
					// ErrCaughtUp causes runCatchup to exit cleanly after 1 call.
					// No second mock needed.
				} else if tc.expectsBackoff {
					daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
						Run(func(args mock.Arguments) {
							callTimes = append(callTimes, time.Now())
							callCount++
							subCancel()
						}).
						Return(nil, datypes.ErrBlobNotFound).Once()
				} else {
					daRetriever.On("RetrieveFromDA", mock.Anything, uint64(101)).
						Run(func(args mock.Arguments) {
							callTimes = append(callTimes, time.Now())
							callCount++
							subCancel()
						}).
						Return(nil, datypes.ErrBlobNotFound).Once()
				}

				// Start already called above to avoid race on eventCh push vs Start

				if tc.exitsCatchup {
					for !follower.HasReachedHead() && ctx.Err() == nil {
						time.Sleep(time.Millisecond)
					}
				} else {
					<-ctx.Done()
				}

				if tc.exitsCatchup {
					assert.Equal(t, 1, callCount, "should exit after single call (ErrCaughtUp)")
					assert.True(t, follower.HasReachedHead(), "should mark head as reached")
				} else if tc.expectsBackoff {
					require.Len(t, callTimes, 2, "should make exactly 2 calls with backoff")

					timeBetweenCalls := callTimes[1].Sub(callTimes[0])
					expectedDelay := tc.daBlockTime
					if expectedDelay == 0 {
						expectedDelay = 2 * time.Second
					}

					assert.GreaterOrEqual(t, timeBetweenCalls, expectedDelay,
						"second call should be delayed by backoff duration (expected ~%v, got %v)",
						expectedDelay, timeBetweenCalls)
				} else {
					assert.GreaterOrEqual(t, callCount, 2, "should continue without significant delay")
					if len(callTimes) >= 2 {
						timeBetweenCalls := callTimes[1].Sub(callTimes[0])
						assert.Less(t, timeBetweenCalls, 120*time.Millisecond,
							"should not have backoff delay for ErrBlobNotFound")
					}
				}
			})
		})
	}
}

// TestDAFollower_BackoffResetOnSuccess verifies that backoff is properly reset
// after a successful DA retrieval.
func TestDAFollower_BackoffResetOnSuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()

		addr, pub, signer := buildSyncTestSigner(t)
		gen := backoffTestGenesis(addr)

		daRetriever := NewMockDARetriever(t)

		pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

		daClient, eventCh := setupMockDAClient(t)
		follower := NewDAFollower(DAFollowerConfig{
			Client:        daClient,
			Retriever:     daRetriever,
			Logger:        zerolog.Nop(),
			EventSink:     common.EventSinkFunc(pipeEvent),
			Namespace:     []byte("ns"),
			StartDAHeight: 100,
			DABlockTime:   1 * time.Second,
		}).(*daFollower)

		ctx, subCancel := context.WithCancel(ctx)
		defer subCancel()

		follower.Start(ctx)
		eventCh <- datypes.SubscriptionEvent{Height: 105}

		var callTimes []time.Time

		// First call - error (should trigger backoff)
		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
			Run(func(args mock.Arguments) {
				callTimes = append(callTimes, time.Now())
			}).
			Return(nil, errors.New("temporary failure")).Once()

		// Second call - success
		_, header := makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, nil, nil, nil)
		data := &types.Data{
			Metadata: &types.Metadata{
				ChainID: gen.ChainID,
				Height:  1,
				Time:    uint64(time.Now().UnixNano()),
			},
		}
		event := common.DAHeightEvent{
			Header:   header,
			Data:     data,
			DaHeight: 100,
		}

		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
			Run(func(args mock.Arguments) {
				callTimes = append(callTimes, time.Now())
			}).
			Return([]common.DAHeightEvent{event}, nil).Once()

		// Third call - should happen immediately after success (DA height incremented)
		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(101)).
			Run(func(args mock.Arguments) {
				callTimes = append(callTimes, time.Now())
				subCancel()
			}).
			Return(nil, datypes.ErrBlobNotFound).Once()

		<-ctx.Done()

		require.Len(t, callTimes, 3, "should make exactly 3 calls")

		delay1to2 := callTimes[1].Sub(callTimes[0])
		assert.GreaterOrEqual(t, delay1to2, 1*time.Second,
			"should have backed off between error and success (got %v)", delay1to2)

		delay2to3 := callTimes[2].Sub(callTimes[1])
		assert.Less(t, delay2to3, 100*time.Millisecond,
			"should continue immediately after success (got %v)", delay2to3)
	})
}

// TestDAFollower_CatchupThenReachHead verifies the catchup flow:
// sequential fetch from local → highest → mark head reached.
func TestDAFollower_CatchupThenReachHead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		daRetriever := NewMockDARetriever(t)

		pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

		daClient, eventCh := setupMockDAClient(t)
		follower := NewDAFollower(DAFollowerConfig{
			Client:        daClient,
			Retriever:     daRetriever,
			Logger:        zerolog.Nop(),
			EventSink:     common.EventSinkFunc(pipeEvent),
			Namespace:     []byte("ns"),
			StartDAHeight: 3,
			DABlockTime:   500 * time.Millisecond,
		}).(*daFollower)

		follower.Start(ctx)
		eventCh <- datypes.SubscriptionEvent{Height: 5}

		var fetchedHeights []uint64

		for h := uint64(3); h <= 5; h++ {
			daRetriever.On("RetrieveFromDA", mock.Anything, h).
				Run(func(args mock.Arguments) {
					fetchedHeights = append(fetchedHeights, h)
				}).
				Return(nil, datypes.ErrBlobNotFound).Once()
		}

		for !follower.HasReachedHead() && ctx.Err() == nil {
			time.Sleep(time.Millisecond)
		}

		assert.True(t, follower.HasReachedHead(), "should have reached DA head")
		// Heights 3, 4, 5 processed; local now at 6 which > highest (5) → caught up
		assert.Equal(t, []uint64{3, 4, 5}, fetchedHeights, "should have fetched heights sequentially")
	})
}

// TestDAFollower_InlineProcessing verifies the fast path: when the subscription
// delivers blobs at the current localDAHeight, HandleEvent processes
// them inline via ProcessBlobs (not RetrieveFromDA).
func TestDAFollower_InlineProcessing(t *testing.T) {
	t.Run("processes_blobs_inline_when_caught_up", func(t *testing.T) {
		daRetriever := NewMockDARetriever(t)

		var mx sync.Mutex
		var pipedEvents []common.DAHeightEvent
		pipeEvent := func(_ context.Context, ev common.DAHeightEvent) error {
			mx.Lock()
			defer mx.Unlock()
			pipedEvents = append(pipedEvents, ev)
			return nil
		}

		daClient, eventCh := setupMockDAClient(t)
		follower := NewDAFollower(DAFollowerConfig{
			Client:        daClient,
			Retriever:     daRetriever,
			Logger:        zerolog.Nop(),
			EventSink:     common.EventSinkFunc(pipeEvent),
			Namespace:     []byte("ns"),
			StartDAHeight: 10,
			DABlockTime:   500 * time.Millisecond,
		}).(*daFollower)

		blobs := [][]byte{[]byte("header-blob"), []byte("data-blob")}
		expectedEvents := []common.DAHeightEvent{
			{DaHeight: 10, Source: common.SourceDA},
		}

		daRetriever.On("ProcessBlobs", mock.Anything, blobs, uint64(10)).Return(expectedEvents).Once()
		// Mock RetrieveFromDA for height 10 in case background catch-up fires before HandleEvent.
		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(10)).Return(nil, datypes.ErrHeightFromFuture).Maybe()

		// Simulate catchup loop to verify it advances naturally
		require.NoError(t, follower.Start(t.Context()))
		eventCh <- datypes.SubscriptionEvent{Height: 10, Blobs: blobs} // Signal current height

		require.Eventually(t, func() bool { return len(pipedEvents) == 1 }, 10*time.Millisecond, time.Millisecond)

		// Verify: ProcessBlobs was called, events were piped, height advanced
		assert.Equal(t, uint64(10), pipedEvents[0].DaHeight)
		assert.Equal(t, uint64(11), follower.subscriber.LocalDAHeight(), "localDAHeight should advance past processed height")
		assert.True(t, follower.HasReachedHead(), "should mark head as reached after inline processing")
	})

	t.Run("falls_through_to_catchup_when_behind", func(t *testing.T) {
		daRetriever := NewMockDARetriever(t)

		pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

		daClient, subChan := setupMockDAClient(t)
		follower := NewDAFollower(DAFollowerConfig{
			Client:        daClient,
			Retriever:     daRetriever,
			Logger:        zerolog.Nop(),
			EventSink:     common.EventSinkFunc(pipeEvent),
			Namespace:     []byte("ns"),
			StartDAHeight: 10,
			DABlockTime:   1 * time.Millisecond,
		}).(*daFollower)

		blobs := [][]byte{[]byte("blob")}
		subChan <- datypes.SubscriptionEvent{Height: 15, Blobs: blobs}

		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(10)).Return([]common.DAHeightEvent{}, nil).Maybe()

		var updated atomic.Bool
		notifyFn := func(args mock.Arguments) { updated.Store(true) }
		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(11)).Run(notifyFn).Return(nil, datypes.ErrHeightFromFuture).Maybe()
		require.NoError(t, follower.Start(t.Context()))

		require.Eventually(t, func() bool { return updated.Load() }, 10*time.Millisecond, time.Millisecond)
		// ProcessBlobs should NOT have been called
		daRetriever.AssertNotCalled(t, "ProcessBlobs", mock.Anything, mock.Anything, mock.Anything)
		assert.Equal(t, uint64(11), follower.subscriber.LocalDAHeight(), "localDAHeight should not change")
		assert.Equal(t, uint64(15), follower.subscriber.HighestSeenDAHeight(), "highestSeen should be updated")
	})

	t.Run("falls_through_when_no_blobs", func(t *testing.T) {
		daRetriever := NewMockDARetriever(t)

		pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

		daClient, eventCh := setupMockDAClient(t)
		follower := NewDAFollower(DAFollowerConfig{
			Client:        daClient,
			Retriever:     daRetriever,
			Logger:        zerolog.Nop(),
			EventSink:     common.EventSinkFunc(pipeEvent),
			Namespace:     []byte("ns"),
			StartDAHeight: 10,
			DABlockTime:   500 * time.Millisecond,
		}).(*daFollower)

		// Mock RetrieveFromDA because falling through inline processing triggers catchup.
		daRetriever.On("RetrieveFromDA", mock.Anything, uint64(10)).Return(nil, datypes.ErrHeightFromFuture).Maybe()
		require.NoError(t, follower.Start(t.Context()))

		// HandleEvent at current height but no blobs — should fall through
		// Subscriber.HandleEvent updates highest internally even if not processed.
		eventCh <- datypes.SubscriptionEvent{
			Height: 10,
			Blobs:  nil,
		}

		// ProcessBlobs should NOT have been called
		daRetriever.AssertNotCalled(t, "ProcessBlobs", mock.Anything, mock.Anything, mock.Anything)
		require.Eventually(t, func() bool {
			return follower.subscriber.LocalDAHeight() == 10 && follower.subscriber.HighestSeenDAHeight() == 10
		}, 100*time.Millisecond, 2*time.Millisecond)
		assert.Equal(t, uint64(10), follower.subscriber.LocalDAHeight(), "localDAHeight should not change")
		assert.Equal(t, uint64(10), follower.subscriber.HighestSeenDAHeight(), "highestSeen should be updated")
	})
}

// backoffTestGenesis creates a test genesis for the backoff tests.
func backoffTestGenesis(addr []byte) genesis.Genesis {
	return genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Hour),
		ProposerAddress: addr,
		DAStartHeight:   100,
	}
}
