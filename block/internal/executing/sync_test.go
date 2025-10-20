package executing

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// MockHeightAwareExecutor is a mock that implements both Executor and HeightProvider
type MockHeightAwareExecutor struct {
	mock.Mock
}

func (m *MockHeightAwareExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, uint64, error) {
	args := m.Called(ctx, genesisTime, initialHeight, chainID)
	return args.Get(0).([]byte), args.Get(1).(uint64), args.Error(2)
}

func (m *MockHeightAwareExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	args := m.Called(ctx)
	return args.Get(0).([][]byte), args.Error(1)
}

func (m *MockHeightAwareExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, uint64, error) {
	args := m.Called(ctx, txs, blockHeight, timestamp, prevStateRoot)
	return args.Get(0).([]byte), args.Get(1).(uint64), args.Error(2)
}

func (m *MockHeightAwareExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	args := m.Called(ctx, blockHeight)
	return args.Error(0)
}

func (m *MockHeightAwareExecutor) GetLatestHeight(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func TestSyncExecutionLayer_ExecutorBehind(t *testing.T) {
	// Create a mock executor
	mockExec := new(MockHeightAwareExecutor)

	// Setup: ev-node is at height 100, execution layer is at 99
	evNodeHeight := uint64(100)
	execLayerHeight := uint64(99)

	// Mock GetLatestHeight to return 99
	mockExec.On("GetLatestHeight", mock.Anything).Return(execLayerHeight, nil)

	// Mock ExecuteTxs for block 100
	mockExec.On("ExecuteTxs",
		mock.Anything,
		mock.Anything,
		evNodeHeight,
		mock.Anything,
		mock.Anything,
	).Return([]byte("new_state_root"), uint64(1000), nil)

	// Create minimal executor setup
	ctx := context.Background()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now(),
	}

	// Create a mock store
	mockStore := new(MockStore)

	now := uint64(time.Now().UnixNano())

	// Setup mock responses for GetBlockData (height 100)
	mockStore.On("GetBlockData", mock.Anything, uint64(100)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height:  100,
					Time:    now,
					ChainID: "test-chain",
				},
				AppHash: []byte("app_hash_100"),
			},
		},
		&types.Data{
			Txs: []types.Tx{[]byte("tx1"), []byte("tx2")},
		},
		nil,
	)

	// Setup mock responses for GetBlockData (height 99 - previous block)
	mockStore.On("GetBlockData", mock.Anything, uint64(99)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height:  99,
					Time:    now - 1000000000, // 1 second earlier
					ChainID: "test-chain",
				},
				AppHash: []byte("app_hash_99"),
			},
		},
		&types.Data{},
		nil,
	)

	// Setup mock response for GetState (returns current state)
	mockStore.On("GetState", mock.Anything).Return(
		types.State{
			LastBlockHeight: 99,
			AppHash:         []byte("app_hash_99"),
		},
		nil,
	)

	executor := &Executor{
		exec:    mockExec,
		store:   mockStore,
		genesis: gen,
		logger:  testLogger(),
	}

	state := types.State{
		LastBlockHeight: evNodeHeight,
		AppHash:         []byte("app_hash_100"),
	}

	// Execute the sync
	err := executor.syncExecutionLayer(ctx, state)
	require.NoError(t, err)

	// Verify that GetLatestHeight was called
	mockExec.AssertCalled(t, "GetLatestHeight", mock.Anything)

	// Verify that ExecuteTxs was called for block 100
	mockExec.AssertCalled(t, "ExecuteTxs",
		mock.Anything,
		mock.Anything,
		evNodeHeight,
		mock.Anything,
		mock.Anything,
	)
}

func TestSyncExecutionLayer_InSync(t *testing.T) {
	mockExec := new(MockHeightAwareExecutor)

	// Setup: both at height 100
	height := uint64(100)

	mockExec.On("GetLatestHeight", mock.Anything).Return(height, nil)

	ctx := context.Background()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now(),
	}

	mockStore := new(MockStore)

	executor := &Executor{
		exec:    mockExec,
		store:   mockStore,
		genesis: gen,
		logger:  testLogger(),
	}

	state := types.State{
		LastBlockHeight: height,
	}

	err := executor.syncExecutionLayer(ctx, state)
	require.NoError(t, err)

	mockExec.AssertCalled(t, "GetLatestHeight", mock.Anything)
	// ExecuteTxs should NOT be called when in sync
	mockExec.AssertNotCalled(t, "ExecuteTxs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSyncExecutionLayer_ExecutorAhead(t *testing.T) {
	mockExec := new(MockHeightAwareExecutor)

	// Setup: ev-node at 100, execution layer at 101 (should not happen!)
	evNodeHeight := uint64(100)
	execLayerHeight := uint64(101)

	mockExec.On("GetLatestHeight", mock.Anything).Return(execLayerHeight, nil)

	ctx := context.Background()
	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now(),
	}

	mockStore := new(MockStore)

	executor := &Executor{
		exec:    mockExec,
		store:   mockStore,
		genesis: gen,
		logger:  testLogger(),
	}

	state := types.State{
		LastBlockHeight: evNodeHeight,
	}

	err := executor.syncExecutionLayer(ctx, state)
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution layer height (101) is ahead of ev-node (100)")
}

// Helper types for testing
type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetBlockData(ctx context.Context, height uint64) (*types.SignedHeader, *types.Data, error) {
	args := m.Called(ctx, height)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*types.SignedHeader), args.Get(1).(*types.Data), args.Error(2)
}

func (m *MockStore) GetState(ctx context.Context) (types.State, error) {
	args := m.Called(ctx)
	return args.Get(0).(types.State), args.Error(1)
}

func (m *MockStore) GetStateAtHeight(ctx context.Context, height uint64) (types.State, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(types.State), args.Error(1)
}

func (m *MockStore) NewBatch(ctx context.Context) (store.Batch, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(store.Batch), args.Error(1)
}

func (m *MockStore) GetSignature(ctx context.Context, height uint64) (*types.Signature, error) {
	args := m.Called(ctx, height)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Signature), args.Error(1)
}

func (m *MockStore) GetSignatureByHash(ctx context.Context, hash []byte) (*types.Signature, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Signature), args.Error(1)
}

func (m *MockStore) GetBlockByHash(ctx context.Context, hash []byte) (*types.SignedHeader, *types.Data, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*types.SignedHeader), args.Get(1).(*types.Data), args.Error(2)
}

func (m *MockStore) GetHeader(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	args := m.Called(ctx, height)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.SignedHeader), args.Error(1)
}

func (m *MockStore) Height(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockStore) SetMetadata(ctx context.Context, key string, value []byte) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockStore) GetMetadata(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStore) Rollback(ctx context.Context, height uint64, aggregator bool) error {
	args := m.Called(ctx, height, aggregator)
	return args.Error(0)
}

func (m *MockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func testLogger() zerolog.Logger {
	return zerolog.Nop()
}
