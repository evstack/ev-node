package block

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	storepkg "github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// setupStoreWithBlocks creates a store with dummy block and state data up to the given height
func setupStoreWithBlocks(t *testing.T, chainID string, maxHeight uint64) storepkg.Store {
	t.Helper()
	require := require.New(t)

	es, err := storepkg.NewDefaultInMemoryKVStore()
	require.NoError(err)
	store := storepkg.New(es)

	ctx := context.Background()

	// Create and store dummy blocks and states up to maxHeight
	for h := uint64(1); h <= maxHeight; h++ {
		header, data := types.GetRandomBlock(h, 2, chainID)
		sig := &header.Signature

		// Save block data
		err := store.SaveBlockData(ctx, header, data, sig)
		require.NoError(err)

		// Set height (this triggers state storage at the current height)
		err = store.SetHeight(ctx, h)
		require.NoError(err)

		// Create and update state for this height
		state := types.State{
			ChainID:         chainID,
			InitialHeight:   1,
			LastBlockHeight: h,
			LastBlockTime:   header.Time(),
			AppHash:         header.AppHash,
		}
		err = store.UpdateState(ctx, state)
		require.NoError(err)
	}

	return store
}

// TestRollback_Success verifies that rollback successfully reverts one block
func TestRollback_Success(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	ctx := context.Background()
	chainID := "test-rollback-success"
	currentHeight := uint64(10)
	rollbackToHeight := uint64(9)

	// Create real store with dummy data
	store := setupStoreWithBlocks(t, chainID, currentHeight)
	mockExecutor := mocks.NewMockExecutor(t)
	mockExecutor.On("Rollback", ctx, rollbackToHeight).Return(nil)

	m := &Manager{
		store: store,
		exec:  mockExecutor,
	}

	// Verify current height
	height, err := store.Height(ctx)
	require.NoError(err)
	require.Equal(currentHeight, height)

	// Execute rollback
	err = m.Rollback(ctx, rollbackToHeight)
	require.NoError(err)

	// Verify rollback succeeded
	newHeight, err := store.Height(ctx)
	require.NoError(err)
	require.Equal(rollbackToHeight, newHeight)
}

// TestRollback_CannotRollbackGenesis verifies that rollback fails when trying to rollback to height 0
func TestRollback_CannotRollbackGenesis(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	mockStore := mocks.NewMockStore(t)
	m := &Manager{
		store: mockStore,
	}

	ctx := context.Background()
	rollbackToHeight := uint64(0)

	err := m.Rollback(ctx, rollbackToHeight)

	// Assertions
	require.Error(err)
	require.Contains(err.Error(), "cannot rollback, already at genesis block")
	mockStore.AssertNotCalled(t, "Height", mock.Anything)
	mockStore.AssertNotCalled(t, "Rollback", mock.Anything, mock.Anything)
}

// TestRollback_GetHeightError verifies that rollback fails when getting current height fails
func TestRollback_GetHeightError(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	mockStore := mocks.NewMockStore(t)
	m := &Manager{
		store: mockStore,
	}

	ctx := context.Background()
	rollbackToHeight := uint64(5)
	expectedErr := errors.New("failed to get height")

	// Mock height call to return error
	mockStore.On("Height", ctx).Return(uint64(0), expectedErr).Once()

	// Execute rollback
	err := m.Rollback(ctx, rollbackToHeight)
	require.Error(err)
	require.Contains(err.Error(), "failed to get current height")
	require.ErrorIs(err, expectedErr)
	mockStore.AssertExpectations(t)
	mockStore.AssertNotCalled(t, "Rollback", mock.Anything, mock.Anything)
}

// TestRollback_CannotRollbackToSameHeight verifies that rollback fails when trying to rollback to current height
func TestRollback_CannotRollbackToSameHeight(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	mockStore := mocks.NewMockStore(t)
	m := &Manager{
		store: mockStore,
	}

	ctx := context.Background()
	currentHeight := uint64(10)
	rollbackToHeight := currentHeight

	mockStore.On("Height", ctx).Return(currentHeight, nil).Once()

	err := m.Rollback(ctx, rollbackToHeight)
	require.Error(err)
	require.Contains(err.Error(), "cannot rollback to height 10, current height is 10")
	mockStore.AssertExpectations(t)
	mockStore.AssertNotCalled(t, "Rollback", mock.Anything, mock.Anything)
}

// TestRollback_CannotRollbackToHigherHeight verifies that rollback fails when trying to rollback to a higher height
func TestRollback_CannotRollbackToHigherHeight(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	mockStore := mocks.NewMockStore(t)
	m := &Manager{
		store: mockStore,
	}

	ctx := context.Background()
	currentHeight := uint64(10)
	rollbackToHeight := uint64(15)

	mockStore.On("Height", ctx).Return(currentHeight, nil).Once()

	err := m.Rollback(ctx, rollbackToHeight)
	require.Error(err)
	require.Contains(err.Error(), "cannot rollback to height 15, current height is 10")
	mockStore.AssertExpectations(t)
	mockStore.AssertNotCalled(t, "Rollback", mock.Anything, mock.Anything)
}

// TestRollback_StoreRollbackError verifies that rollback fails when store rollback operation fails
func TestRollback_StoreRollbackError(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	mockStore := mocks.NewMockStore(t)
	mockExecutor := mocks.NewMockExecutor(t)

	m := &Manager{
		store: mockStore,
		exec:  mockExecutor,
	}

	ctx := context.Background()
	currentHeight := uint64(10)
	rollbackToHeight := uint64(8)
	expectedErr := errors.New("failed to rollback store")

	mockStore.On("Height", ctx).Return(currentHeight, nil).Once()
	mockStore.On("Rollback", ctx, rollbackToHeight).Return(expectedErr).Once()
	mockExecutor.On("Rollback", ctx, rollbackToHeight).Return(nil)

	err := m.Rollback(ctx, rollbackToHeight)
	require.Error(err)
	require.Contains(err.Error(), "failed to delete block data until height 8")
	require.ErrorIs(err, expectedErr)
	mockStore.AssertExpectations(t)
}

// TestRollback_MultipleBlocks verifies that rollback can revert multiple blocks
func TestRollback_MultipleBlocks(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	ctx := context.Background()
	chainID := "test-rollback-multiple"
	currentHeight := uint64(15)
	rollbackToHeight := uint64(10) // Rollback 5 blocks

	// Create real store with dummy data
	store := setupStoreWithBlocks(t, chainID, currentHeight)
	mockExecutor := mocks.NewMockExecutor(t)
	mockExecutor.On("Rollback", ctx, rollbackToHeight).Return(nil)

	m := &Manager{
		store: store,
		exec:  mockExecutor,
	}

	// Verify current height
	height, err := store.Height(ctx)
	require.NoError(err)
	require.Equal(currentHeight, height)

	// Verify blocks exist up to current height
	for h := uint64(1); h <= currentHeight; h++ {
		_, _, err := store.GetBlockData(ctx, h)
		require.NoError(err, "block at height %d should exist", h)
	}

	// Execute rollback
	err = m.Rollback(ctx, rollbackToHeight)
	require.NoError(err)

	// Verify rollback succeeded
	newHeight, err := store.Height(ctx)
	require.NoError(err)
	require.Equal(rollbackToHeight, newHeight)

	// Verify blocks exist only up to rollback height
	for h := uint64(1); h <= rollbackToHeight; h++ {
		_, _, err := store.GetBlockData(ctx, h)
		require.NoError(err, "block at height %d should still exist after rollback", h)
	}

	// Verify blocks above rollback height are removed
	for h := rollbackToHeight + 1; h <= currentHeight; h++ {
		_, _, err := store.GetBlockData(ctx, h)
		require.Error(err, "block at height %d should be removed after rollback", h)
	}
}
