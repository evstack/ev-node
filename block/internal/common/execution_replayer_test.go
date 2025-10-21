package common

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

func TestExecutionLayerReplayer_SyncToHeight_ExecutorBehind(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockHeightAwareExecutor(t)
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	// Setup: target height is 100, execution layer is at 99
	targetHeight := uint64(100)
	execHeight := uint64(99)

	mockExec.On("GetLatestHeight", mock.Anything).Return(execHeight, nil)

	now := uint64(time.Now().UnixNano())

	// Setup store to return block data for height 100
	mockStore.EXPECT().GetBlockData(mock.Anything, uint64(100)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height:  100,
					Time:    now,
					ChainID: "test-chain",
				},
				AppHash: []byte("app-hash-100"),
			},
		},
		&types.Data{
			Txs: []types.Tx{[]byte("tx1")},
		},
		nil,
	)

	// Setup store to return previous block for state
	mockStore.EXPECT().GetBlockData(mock.Anything, uint64(99)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height:  99,
					Time:    now - 1000000000,
					ChainID: "test-chain",
				},
				AppHash: []byte("app-hash-99"),
			},
		},
		&types.Data{},
		nil,
	)

	// Setup state
	mockStore.EXPECT().GetState(mock.Anything).Return(
		types.State{
			ChainID:         gen.ChainID,
			InitialHeight:   gen.InitialHeight,
			LastBlockHeight: 99,
			AppHash:         []byte("app-hash-99"),
		},
		nil,
	)

	// Expect ExecuteTxs to be called for height 100
	mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, []byte("app-hash-99")).
		Return([]byte("app-hash-100"), uint64(1000), nil)

	// Execute sync
	err := syncer.SyncToHeight(ctx, targetHeight)
	require.NoError(t, err)

	// Verify expectations
	mockExec.AssertExpectations(t)
}

func TestExecutionLayerReplayer_SyncToHeight_InSync(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockHeightAwareExecutor(t)
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	// Setup: both at height 100
	targetHeight := uint64(100)
	execHeight := uint64(100)

	mockExec.On("GetLatestHeight", mock.Anything).Return(execHeight, nil)

	// Execute sync - should do nothing
	err := syncer.SyncToHeight(ctx, targetHeight)
	require.NoError(t, err)

	// ExecuteTxs should not be called
	mockExec.AssertNotCalled(t, "ExecuteTxs")
}

func TestExecutionLayerReplayer_SyncToHeight_ExecutorAhead(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockHeightAwareExecutor(t)
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	// Setup: target height is 100, execution layer is at 101 (unexpected!)
	targetHeight := uint64(100)
	execHeight := uint64(101)

	mockExec.On("GetLatestHeight", mock.Anything).Return(execHeight, nil)

	// Execute sync - should fail
	err := syncer.SyncToHeight(ctx, targetHeight)
	require.Error(t, err)
	require.Contains(t, err.Error(), "execution layer height (101) is ahead of target height (100)")
}

func TestExecutionLayerReplayer_SyncToHeight_NoHeightProvider(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockExecutor(t) // Regular executor without HeightProvider
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	// Execute sync - should skip silently
	err := syncer.SyncToHeight(ctx, 100)
	require.NoError(t, err)

	// No methods should be called
	mockExec.AssertNotCalled(t, "GetLatestHeight")
	mockExec.AssertNotCalled(t, "ExecuteTxs")
}

func TestExecutionLayerReplayer_SyncToHeight_AtGenesis(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockHeightAwareExecutor(t)
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 10,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	// Target height is below genesis initial height
	targetHeight := uint64(5)

	// Execute sync - should skip
	err := syncer.SyncToHeight(ctx, targetHeight)
	require.NoError(t, err)

	// No calls should be made
	mockExec.AssertNotCalled(t, "GetLatestHeight")
	mockExec.AssertNotCalled(t, "ExecuteTxs")
}

func TestExecutionLayerReplayer_SyncToHeight_MultipleBlocks(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockHeightAwareExecutor(t)
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	// Setup: target height is 100, execution layer is at 97 (need to sync 3 blocks: 98, 99, 100)
	targetHeight := uint64(100)
	execHeight := uint64(97)

	mockExec.On("GetLatestHeight", mock.Anything).Return(execHeight, nil)

	now := uint64(time.Now().UnixNano())

	// Setup mocks for blocks 98, 99, 100
	for height := uint64(98); height <= 100; height++ {
		prevHeight := height - 1

		// Current block data
		mockStore.EXPECT().GetBlockData(mock.Anything, height).Return(
			&types.SignedHeader{
				Header: types.Header{
					BaseHeader: types.BaseHeader{
						Height:  height,
						Time:    now + (height * 1000000000),
						ChainID: "test-chain",
					},
					AppHash: []byte("app-hash-" + string(rune('0'+height))),
				},
			},
			&types.Data{
				Txs: []types.Tx{[]byte("tx-" + string(rune('0'+height)))},
			},
			nil,
		).Once()

		// Previous block data (for getting previous app hash)
		mockStore.EXPECT().GetBlockData(mock.Anything, prevHeight).Return(
			&types.SignedHeader{
				Header: types.Header{
					BaseHeader: types.BaseHeader{
						Height:  prevHeight,
						Time:    now + (prevHeight * 1000000000),
						ChainID: "test-chain",
					},
					AppHash: []byte("app-hash-" + string(rune('0'+prevHeight))),
				},
			},
			&types.Data{},
			nil,
		).Once()

		// State (returns the state of previous block)
		mockStore.EXPECT().GetState(mock.Anything).Return(
			types.State{
				ChainID:         gen.ChainID,
				InitialHeight:   gen.InitialHeight,
				LastBlockHeight: prevHeight,
				AppHash:         []byte("app-hash-" + string(rune('0'+prevHeight))),
			},
			nil,
		).Once()

		// ExecuteTxs for current block
		mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, height, mock.Anything, mock.Anything).
			Return([]byte("app-hash-"+string(rune('0'+height))), uint64(1000), nil).Once()
	}

	// Execute sync
	err := syncer.SyncToHeight(ctx, targetHeight)
	require.NoError(t, err)

	// Verify ExecuteTxs was called 3 times (for blocks 98, 99, 100)
	mockExec.AssertNumberOfCalls(t, "ExecuteTxs", 3)
	mockExec.AssertExpectations(t)
	mockStore.AssertExpectations(t)
}

func TestExecutionLayerReplayer_ReplayBlock_FirstBlock(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockHeightAwareExecutor(t)
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	now := uint64(time.Now().UnixNano())

	// Setup store to return first block (at initial height)
	mockStore.EXPECT().GetBlockData(mock.Anything, uint64(1)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height:  1,
					Time:    now,
					ChainID: "test-chain",
				},
				AppHash: []byte("app-hash-1"),
			},
		},
		&types.Data{
			Txs: []types.Tx{[]byte("tx1")},
		},
		nil,
	)

	// For first block, ExecuteTxs should be called with genesis app hash
	mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(1), mock.Anything, []byte("app-hash-1")).
		Return([]byte("app-hash-1"), uint64(1000), nil)

	// Call replayBlock directly (this is a private method, so we test it through SyncToHeight)
	mockExec.On("GetLatestHeight", mock.Anything).Return(uint64(0), nil)

	err := syncer.SyncToHeight(ctx, 1)
	require.NoError(t, err)

	mockExec.AssertExpectations(t)
	mockStore.AssertExpectations(t)
}

func TestExecutionLayerReplayer_AppHashMismatch(t *testing.T) {
	ctx := context.Background()
	mockExec := mocks.NewMockHeightAwareExecutor(t)
	mockStore := mocks.NewMockStore(t)
	logger := zerolog.Nop()

	gen := genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now().UTC(),
	}

	syncer := NewExecutionLayerReplayer(mockStore, mockExec, gen, logger)

	targetHeight := uint64(100)
	execHeight := uint64(99)

	mockExec.On("GetLatestHeight", mock.Anything).Return(execHeight, nil)

	now := uint64(time.Now().UnixNano())

	mockStore.EXPECT().GetBlockData(mock.Anything, uint64(100)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height:  100,
					Time:    now,
					ChainID: "test-chain",
				},
				AppHash: []byte("expected-app-hash"),
			},
		},
		&types.Data{
			Txs: []types.Tx{[]byte("tx1")},
		},
		nil,
	)

	mockStore.EXPECT().GetBlockData(mock.Anything, uint64(99)).Return(
		&types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					Height:  99,
					Time:    now - 1000000000,
					ChainID: "test-chain",
				},
				AppHash: []byte("app-hash-99"),
			},
		},
		&types.Data{},
		nil,
	)

	mockStore.EXPECT().GetState(mock.Anything).Return(
		types.State{
			ChainID:         gen.ChainID,
			InitialHeight:   gen.InitialHeight,
			LastBlockHeight: 99,
			AppHash:         []byte("app-hash-99"),
		},
		nil,
	)

	// ExecuteTxs returns a different app hash than expected
	mockExec.On("ExecuteTxs", mock.Anything, mock.Anything, uint64(100), mock.Anything, []byte("app-hash-99")).
		Return([]byte("different-app-hash"), uint64(1000), nil)

	// Should fail with mismatch error
	err := syncer.SyncToHeight(ctx, targetHeight)
	require.Error(t, err)
	require.Contains(t, err.Error(), "app hash mismatch")

	mockExec.AssertExpectations(t)
	mockStore.AssertExpectations(t)
}
