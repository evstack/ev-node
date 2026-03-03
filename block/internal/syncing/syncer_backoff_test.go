package syncing

import (
	"context"
	"errors"
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
		description    string
	}{
		"generic_error_triggers_backoff": {
			daBlockTime:    1 * time.Second,
			error:          errors.New("network failure"),
			expectsBackoff: true,
			description:    "Generic DA errors should trigger backoff",
		},
		"height_from_future_triggers_backoff": {
			daBlockTime:    500 * time.Millisecond,
			error:          datypes.ErrHeightFromFuture,
			expectsBackoff: true,
			description:    "Height from future should trigger backoff",
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
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				defer cancel()

				daRetriever := NewMockDARetriever(t)
				daRetriever.On("PopPriorityHeight").Return(uint64(0)).Maybe()

				pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

				follower := NewDAFollower(DAFollowerConfig{
					Retriever:     daRetriever,
					Logger:        zerolog.Nop(),
					PipeEvent:     pipeEvent,
					Namespace:     []byte("ns"),
					StartDAHeight: 100,
					DABlockTime:   tc.daBlockTime,
				}).(*daFollower)

				follower.ctx, follower.cancel = context.WithCancel(ctx)
				follower.highestSeenDAHeight.Store(102)

				var callTimes []time.Time
				callCount := 0

				// First call - returns test error
				daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Run(func(args mock.Arguments) {
						callTimes = append(callTimes, time.Now())
						callCount++
					}).
					Return(nil, tc.error).Once()

				if tc.expectsBackoff {
					daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
						Run(func(args mock.Arguments) {
							callTimes = append(callTimes, time.Now())
							callCount++
							cancel()
						}).
						Return(nil, datypes.ErrBlobNotFound).Once()
				} else {
					daRetriever.On("RetrieveFromDA", mock.Anything, uint64(101)).
						Run(func(args mock.Arguments) {
							callTimes = append(callTimes, time.Now())
							callCount++
							cancel()
						}).
						Return(nil, datypes.ErrBlobNotFound).Once()
				}

				go follower.runCatchup()
				<-ctx.Done()

				if tc.expectsBackoff {
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
		daRetriever.On("PopPriorityHeight").Return(uint64(0)).Maybe()

		pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

		follower := NewDAFollower(DAFollowerConfig{
			Retriever:     daRetriever,
			Logger:        zerolog.Nop(),
			PipeEvent:     pipeEvent,
			Namespace:     []byte("ns"),
			StartDAHeight: 100,
			DABlockTime:   1 * time.Second,
		}).(*daFollower)

		follower.ctx, follower.cancel = context.WithCancel(ctx)
		follower.highestSeenDAHeight.Store(105)

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
				cancel()
			}).
			Return(nil, datypes.ErrBlobNotFound).Once()

		go follower.runCatchup()
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
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()

		daRetriever := NewMockDARetriever(t)
		daRetriever.On("PopPriorityHeight").Return(uint64(0)).Maybe()

		pipeEvent := func(_ context.Context, _ common.DAHeightEvent) error { return nil }

		follower := NewDAFollower(DAFollowerConfig{
			Retriever:     daRetriever,
			Logger:        zerolog.Nop(),
			PipeEvent:     pipeEvent,
			Namespace:     []byte("ns"),
			StartDAHeight: 3,
			DABlockTime:   500 * time.Millisecond,
		}).(*daFollower)

		follower.ctx, follower.cancel = context.WithCancel(ctx)
		follower.highestSeenDAHeight.Store(5)

		var fetchedHeights []uint64

		for h := uint64(3); h <= 5; h++ {
			h := h
			daRetriever.On("RetrieveFromDA", mock.Anything, h).
				Run(func(args mock.Arguments) {
					fetchedHeights = append(fetchedHeights, h)
				}).
				Return(nil, datypes.ErrBlobNotFound).Once()
		}

		follower.runCatchup()

		assert.True(t, follower.HasReachedHead(), "should have reached DA head")
		// Heights 3, 4, 5 processed; local now at 6 which > highest (5) → caught up
		assert.Equal(t, []uint64{3, 4, 5}, fetchedHeights, "should have fetched heights sequentially")
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
