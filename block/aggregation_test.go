package block

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/cache"
	"github.com/evstack/ev-node/pkg/config"
	genesispkg "github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// TestAggregationLoop_Normal_BasicInterval verifies that the aggregation loop publishes blocks at the expected interval under normal conditions.
func TestAggregationLoop_Normal_BasicInterval(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	blockTime := 50 * time.Millisecond
	waitTime := blockTime*4 + blockTime/2

	mockStore := mocks.NewMockStore(t)
	mockStore.On("Height", mock.Anything).Return(uint64(1), nil).Maybe()
	mockStore.On("GetState", mock.Anything).Return(types.State{LastBlockTime: time.Now().Add(-blockTime)}, nil).Maybe()

	mockExec := mocks.NewMockExecutor(t)
	mockSeq := mocks.NewMockSequencer(t)
	mockDAC := mocks.NewMockDA(t)
	logger := zerolog.Nop()

	m := &Manager{
		store:     mockStore,
		exec:      mockExec,
		sequencer: mockSeq,
		da:        mockDAC,
		logger:    logger,
		config: config.Config{
			Node: config.NodeConfig{
				BlockTime: config.DurationWrapper{Duration: blockTime},
				LazyMode:  false,
			},
			DA: config.DAConfig{
				BlockTime: config.DurationWrapper{Duration: 1 * time.Second},
			},
		},
		genesis: genesispkg.Genesis{
			InitialHeight: 1,
		},
		lastState: types.State{
			LastBlockTime: time.Now().Add(-blockTime),
		},
		lastStateMtx: &sync.RWMutex{},
		metrics:      NopMetrics(),
		headerCache:  cache.NewCache[types.SignedHeader](),
		dataCache:    cache.NewCache[types.Data](),
	}

	var publishTimes []time.Time
	var publishLock sync.Mutex
	mockPublishBlock := func(ctx context.Context) error {
		publishLock.Lock()
		defer publishLock.Unlock()
		publishTimes = append(publishTimes, time.Now())
		m.logger.Debug().Time("time", publishTimes[len(publishTimes)-1]).Msg("Mock publishBlock called")
		return nil
	}
	m.publishBlock = mockPublishBlock

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.AggregationLoop(ctx, make(chan<- error))
		m.logger.Info().Msg("AggregationLoop exited")
	}()

	m.logger.Info().Dur("duration", waitTime).Msg("Waiting for blocks...")
	time.Sleep(waitTime)

	m.logger.Info().Msg("Cancelling context")
	cancel()
	m.logger.Info().Msg("Waiting for WaitGroup")
	wg.Wait()
	m.logger.Info().Msg("WaitGroup finished")

	publishLock.Lock()
	defer publishLock.Unlock()

	m.logger.Info().Int("count", len(publishTimes)).Any("times", publishTimes).Msg("Recorded publish times")

	expectedCallsLow := int(waitTime/blockTime) - 1
	expectedCallsHigh := int(waitTime/blockTime) + 1
	require.GreaterOrEqualf(len(publishTimes), expectedCallsLow, "Expected at least %d calls, got %d", expectedCallsLow, len(publishTimes))
	require.LessOrEqualf(len(publishTimes), expectedCallsHigh, "Expected at most %d calls, got %d", expectedCallsHigh, len(publishTimes))

	if len(publishTimes) > 1 {
		for i := 1; i < len(publishTimes); i++ {
			interval := publishTimes[i].Sub(publishTimes[i-1])
			m.logger.Debug().Int("index", i).Dur("interval", interval).Msg("Checking interval")
			tolerance := blockTime / 2
			assert.True(WithinDuration(t, blockTime, interval, tolerance), "Interval %d (%v) not within tolerance (%v) of blockTime (%v)", i, interval, tolerance, blockTime)
		}
	}
}

// TestAggregationLoop_Normal_PublishBlockError verifies that the aggregation loop handles errors from publishBlock gracefully.
func TestAggregationLoop_Normal_PublishBlockError(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	blockTime := 50 * time.Millisecond
	waitTime := blockTime*4 + blockTime/2

	mockStore := mocks.NewMockStore(t)
	mockStore.On("Height", mock.Anything).Return(uint64(1), nil).Maybe()
	mockStore.On("GetState", mock.Anything).Return(types.State{LastBlockTime: time.Now().Add(-blockTime)}, nil).Maybe()

	mockExec := mocks.NewMockExecutor(t)
	mockSeq := mocks.NewMockSequencer(t)
	mockDAC := mocks.NewMockDA(t)

	logger := zerolog.Nop()

	// Create a basic Manager instance
	m := &Manager{
		store:     mockStore,
		exec:      mockExec,
		sequencer: mockSeq,
		da:        mockDAC,
		logger:    logger,
		config: config.Config{
			Node: config.NodeConfig{
				BlockTime: config.DurationWrapper{Duration: blockTime},
				LazyMode:  false,
			},
			DA: config.DAConfig{
				BlockTime: config.DurationWrapper{Duration: 1 * time.Second},
			},
		},
		genesis: genesispkg.Genesis{
			InitialHeight: 1,
		},
		lastState: types.State{
			LastBlockTime: time.Now().Add(-blockTime),
		},
		lastStateMtx: &sync.RWMutex{},
		metrics:      NopMetrics(),
		headerCache:  cache.NewCache[types.SignedHeader](),
		dataCache:    cache.NewCache[types.Data](),
	}

	var publishCalls atomic.Int64
	var publishTimes []time.Time
	var publishLock sync.Mutex
	expectedErr := errors.New("failed to publish block")

	mockPublishBlock := func(ctx context.Context) error {
		callNum := publishCalls.Add(1)
		publishLock.Lock()
		publishTimes = append(publishTimes, time.Now())
		publishLock.Unlock()

		if callNum == 1 {
			m.logger.Debug().Int64("call", callNum).Msg("Mock publishBlock returning error")
			return expectedErr
		}
		m.logger.Debug().Int64("call", callNum).Msg("Mock publishBlock returning nil")
		return nil
	}
	m.publishBlock = mockPublishBlock

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.AggregationLoop(ctx, errCh)
		m.logger.Info().Msg("AggregationLoop exited")
	}()

	time.Sleep(waitTime)

	cancel()
	wg.Wait()

	publishLock.Lock()
	defer publishLock.Unlock()

	calls := publishCalls.Load()
	require.Equal(calls, int64(1))
	require.ErrorContains(<-errCh, expectedErr.Error())
	require.Equal(len(publishTimes), 1, "Expected only one publish time after error")
}
