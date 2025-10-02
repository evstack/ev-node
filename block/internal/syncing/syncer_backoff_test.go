package syncing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	mocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// TestSyncer_BackoffOnDAError verifies that the syncer implements proper backoff
// behavior when encountering different types of DA layer errors.
func TestSyncer_BackoffOnDAError(t *testing.T) {
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
			error:          coreda.ErrHeightFromFuture,
			expectsBackoff: true,
			description:    "Height from future should trigger backoff",
		},
		"blob_not_found_no_backoff": {
			daBlockTime:    1 * time.Second,
			error:          coreda.ErrBlobNotFound,
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
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Setup syncer
			syncer := setupTestSyncer(t, tc.daBlockTime)
			syncer.ctx = ctx

			// Setup mocks
			daRetriever := newMockdaRetriever(t)
			p2pHandler := newMockp2pHandler(t)
			syncer.daRetriever = daRetriever
			syncer.p2pHandler = p2pHandler

			headerStore := mocks.NewMockStore[*types.SignedHeader](t)
			headerStore.On("Height").Return(uint64(0)).Maybe()
			syncer.headerStore = headerStore

			dataStore := mocks.NewMockStore[*types.Data](t)
			dataStore.On("Height").Return(uint64(0)).Maybe()
			syncer.dataStore = dataStore

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
				// Second call should be delayed due to backoff
				daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
					Run(func(args mock.Arguments) {
						callTimes = append(callTimes, time.Now())
						callCount++
						// Cancel to end test
						cancel()
					}).
					Return(nil, coreda.ErrBlobNotFound).Once()
			} else {
				// For ErrBlobNotFound, DA height should increment
				daRetriever.On("RetrieveFromDA", mock.Anything, uint64(101)).
					Run(func(args mock.Arguments) {
						callTimes = append(callTimes, time.Now())
						callCount++
						cancel()
					}).
					Return(nil, coreda.ErrBlobNotFound).Once()
			}

			// Run sync loop
			done := make(chan struct{})
			go func() {
				defer close(done)
				syncer.syncLoop()
			}()

			<-done

			// Verify behavior
			if tc.expectsBackoff {
				require.Len(t, callTimes, 2, "should make exactly 2 calls with backoff")

				timeBetweenCalls := callTimes[1].Sub(callTimes[0])
				expectedDelay := tc.daBlockTime
				if expectedDelay == 0 {
					expectedDelay = 2 * time.Second
				}

				assert.GreaterOrEqual(t, timeBetweenCalls, expectedDelay-50*time.Millisecond,
					"second call should be delayed by backoff duration (expected ~%v, got %v)",
					expectedDelay, timeBetweenCalls)
			} else {
				assert.GreaterOrEqual(t, callCount, 2, "should continue without significant delay")
				if len(callTimes) >= 2 {
					timeBetweenCalls := callTimes[1].Sub(callTimes[0])
					assert.Less(t, timeBetweenCalls, 100*time.Millisecond,
						"should not have backoff delay for ErrBlobNotFound")
				}
			}
		})
	}
}

// TestSyncer_BackoffResetOnSuccess verifies that backoff is properly reset
// after a successful DA retrieval, allowing the syncer to continue at normal speed.
func TestSyncer_BackoffResetOnSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	syncer := setupTestSyncer(t, 1*time.Second)
	syncer.ctx = ctx

	addr, pub, signer := buildSyncTestSigner(t)
	gen := syncer.genesis

	daRetriever := newMockdaRetriever(t)
	p2pHandler := newMockp2pHandler(t)
	syncer.daRetriever = daRetriever
	syncer.p2pHandler = p2pHandler

	headerStore := mocks.NewMockStore[*types.SignedHeader](t)
	headerStore.On("Height").Return(uint64(0)).Maybe()
	syncer.headerStore = headerStore

	dataStore := mocks.NewMockStore[*types.Data](t)
	dataStore.On("Height").Return(uint64(0)).Maybe()
	syncer.dataStore = dataStore

	var callTimes []time.Time

	// First call - error (should trigger backoff)
	daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
		Run(func(args mock.Arguments) {
			callTimes = append(callTimes, time.Now())
		}).
		Return(nil, errors.New("temporary failure")).Once()

	// Second call - success (should reset backoff and increment DA height)
	_, header := makeSignedHeaderBytes(t, gen.ChainID, 1, addr, pub, signer, nil, nil)
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

	// Third call - should happen immediately after success (DA height incremented to 101)
	daRetriever.On("RetrieveFromDA", mock.Anything, uint64(101)).
		Run(func(args mock.Arguments) {
			callTimes = append(callTimes, time.Now())
			cancel()
		}).
		Return(nil, coreda.ErrBlobNotFound).Once()

	// Start process loop to handle events
	go syncer.processLoop()

	// Run sync loop
	done := make(chan struct{})
	go func() {
		defer close(done)
		syncer.syncLoop()
	}()

	<-done

	require.Len(t, callTimes, 3, "should make exactly 3 calls")

	// Verify backoff between first and second call
	delay1to2 := callTimes[1].Sub(callTimes[0])
	assert.GreaterOrEqual(t, delay1to2, 950*time.Millisecond,
		"should have backed off between error and success (got %v)", delay1to2)

	// Verify no backoff between second and third call (backoff reset)
	delay2to3 := callTimes[2].Sub(callTimes[1])
	assert.Less(t, delay2to3, 100*time.Millisecond,
		"should continue immediately after success (got %v)", delay2to3)
}

// TestSyncer_BackoffBehaviorIntegration tests the complete backoff flow:
// error -> backoff delay -> recovery -> normal operation.
func TestSyncer_BackoffBehaviorIntegration(t *testing.T) {
	// Test simpler backoff behavior: error -> backoff -> success -> continue
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	syncer := setupTestSyncer(t, 500*time.Millisecond)
	syncer.ctx = ctx

	daRetriever := newMockdaRetriever(t)
	p2pHandler := newMockp2pHandler(t)
	syncer.daRetriever = daRetriever
	syncer.p2pHandler = p2pHandler

	headerStore := mocks.NewMockStore[*types.SignedHeader](t)
	headerStore.On("Height").Return(uint64(0)).Maybe()
	syncer.headerStore = headerStore

	dataStore := mocks.NewMockStore[*types.Data](t)
	dataStore.On("Height").Return(uint64(0)).Maybe()
	syncer.dataStore = dataStore

	var callTimes []time.Time

	// First call - error (triggers backoff)
	daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
		Run(func(args mock.Arguments) {
			callTimes = append(callTimes, time.Now())
		}).
		Return(nil, errors.New("network error")).Once()

	// Second call - should be delayed due to backoff
	daRetriever.On("RetrieveFromDA", mock.Anything, uint64(100)).
		Run(func(args mock.Arguments) {
			callTimes = append(callTimes, time.Now())
		}).
		Return(nil, coreda.ErrBlobNotFound).Once()

	// Third call - should continue without delay (DA height incremented)
	daRetriever.On("RetrieveFromDA", mock.Anything, uint64(101)).
		Run(func(args mock.Arguments) {
			callTimes = append(callTimes, time.Now())
			cancel()
		}).
		Return(nil, coreda.ErrBlobNotFound).Once()

	go syncer.processLoop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		syncer.syncLoop()
	}()

	<-done

	require.Len(t, callTimes, 3, "should make exactly 3 calls")

	// First to second call should be delayed (backoff)
	delay1to2 := callTimes[1].Sub(callTimes[0])
	assert.GreaterOrEqual(t, delay1to2, 450*time.Millisecond,
		"should have backoff delay between first and second call (got %v)", delay1to2)

	// Second to third call should be immediate (no backoff after ErrBlobNotFound)
	delay2to3 := callTimes[2].Sub(callTimes[1])
	assert.Less(t, delay2to3, 100*time.Millisecond,
		"should continue immediately after ErrBlobNotFound (got %v)", delay2to3)
}

func setupTestSyncer(t *testing.T, daBlockTime time.Duration) *Syncer {
	t.Helper()

	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)

	cfg := config.DefaultConfig()
	cfg.DA.BlockTime.Duration = daBlockTime

	gen := genesis.Genesis{
		ChainID:         "test-chain",
		InitialHeight:   1,
		StartTime:       time.Now().Add(-time.Hour), // Start in past
		ProposerAddress: addr,
		DAStartHeight:   100,
	}

	syncer := NewSyncer(
		st,
		execution.NewDummyExecutor(),
		nil,
		cm,
		common.NopMetrics(),
		cfg,
		gen,
		nil,
		nil,
		zerolog.Nop(),
		common.DefaultBlockOptions(),
		make(chan error, 1),
	)

	require.NoError(t, syncer.initializeState())
	return syncer
}
