package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/types"
)

func createTestStoreWithBlocks(t *testing.T, numBlocks int) Store {
	t.Helper()
	ctx := context.Background()
	chainID := t.Name()

	kv, err := NewTestInMemoryKVStore()
	require.NoError(t, err)

	s := New(kv)

	batch, err := s.NewBatch(ctx)
	require.NoError(t, err)

	for i := 1; i <= numBlocks; i++ {
		header, data := types.GetRandomBlock(uint64(i), 5, chainID)
		sig := types.Signature([]byte("test-signature"))
		err = batch.SaveBlockData(header, data, &sig)
		require.NoError(t, err)
		err = batch.SetHeight(uint64(i))
		require.NoError(t, err)
	}

	err = batch.Commit()
	require.NoError(t, err)

	return s
}

func TestCachedStore_GetHeader_CacheHit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create underlying store with test data
	store := createTestStoreWithBlocks(t, 5)

	// Wrap with cached store
	cachedStore, err := NewCachedStore(store)
	require.NoError(t, err)

	// First call fetches from store
	header1, err := cachedStore.GetHeader(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, header1)

	// Second call should return cached header (same reference)
	header2, err := cachedStore.GetHeader(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, header2)

	// Headers should be the same reference (cached)
	assert.Same(t, header1, header2, "should return same cached header")
}

func TestCachedStore_GetHeader_MultipleHeights(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := createTestStoreWithBlocks(t, 10)

	cachedStore, err := NewCachedStore(store)
	require.NoError(t, err)

	// Load multiple heights
	headers := make([]*types.SignedHeader, 5)
	for h := uint64(1); h <= 5; h++ {
		header, err := cachedStore.GetHeader(ctx, h)
		require.NoError(t, err)
		require.NotNil(t, header)
		assert.Equal(t, h, header.Height())
		headers[h-1] = header
	}

	// Re-access all heights - should return same cached references
	for h := uint64(1); h <= 5; h++ {
		header, err := cachedStore.GetHeader(ctx, h)
		require.NoError(t, err)
		assert.Same(t, headers[h-1], header, "should return cached header for height %d", h)
	}
}

func TestCachedStore_GetBlockData_CacheHit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := createTestStoreWithBlocks(t, 5)

	cachedStore, err := NewCachedStore(store)
	require.NoError(t, err)

	// First call should fetch from store
	header1, data1, err := cachedStore.GetBlockData(ctx, 2)
	require.NoError(t, err)
	require.NotNil(t, header1)
	require.NotNil(t, data1)

	// Second call should return cached data
	header2, data2, err := cachedStore.GetBlockData(ctx, 2)
	require.NoError(t, err)

	// Should be same references
	assert.Same(t, header1, header2)
	assert.Same(t, data1, data2)
}

func TestCachedStore_CustomCacheSize(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := createTestStoreWithBlocks(t, 10)

	// Create with small cache size of 3
	cachedStore, err := NewCachedStore(store, WithHeaderCacheSize(3))
	require.NoError(t, err)

	// Load 5 headers - LRU should evict oldest entries
	for h := uint64(1); h <= 5; h++ {
		_, err = cachedStore.GetHeader(ctx, h)
		require.NoError(t, err)
	}

	// Height 5 should still be cached (most recent)
	header5a, err := cachedStore.GetHeader(ctx, 5)
	require.NoError(t, err)
	header5b, err := cachedStore.GetHeader(ctx, 5)
	require.NoError(t, err)
	assert.Same(t, header5a, header5b, "height 5 should be cached")

	// Height 1 was evicted, so fetching it again will get a new object
	header1a, err := cachedStore.GetHeader(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, header1a)
}

func TestCachedStore_Rollback_InvalidatesCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := createTestStoreWithBlocks(t, 10)

	cachedStore, err := NewCachedStore(store)
	require.NoError(t, err)

	// Load headers 5-10 into cache
	cachedHeaders := make(map[uint64]*types.SignedHeader)
	for h := uint64(5); h <= 10; h++ {
		header, err := cachedStore.GetHeader(ctx, h)
		require.NoError(t, err)
		cachedHeaders[h] = header
	}

	// Rollback to height 7
	err = cachedStore.Rollback(ctx, 7, false)
	require.NoError(t, err)

	// Heights 5, 6, 7 should still be accessible
	for h := uint64(5); h <= 7; h++ {
		header, err := cachedStore.GetHeader(ctx, h)
		require.NoError(t, err)
		require.NotNil(t, header)
	}

	// Heights 8, 9, 10 should not exist anymore (rolled back)
	for h := uint64(8); h <= 10; h++ {
		_, err := cachedStore.GetHeader(ctx, h)
		assert.Error(t, err, "height %d should not exist after rollback", h)
	}
}

func TestNewCachedStore_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid header cache size", func(t *testing.T) {
		store := createTestStoreWithBlocks(t, 1)
		_, err := NewCachedStore(store, WithHeaderCacheSize(-1))
		assert.Error(t, err)
	})

	t.Run("invalid block data cache size", func(t *testing.T) {
		store := createTestStoreWithBlocks(t, 1)
		_, err := NewCachedStore(store, WithBlockDataCacheSize(-1))
		assert.Error(t, err)
	})
}

func TestCachedStore_DelegatesHeight(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	store := createTestStoreWithBlocks(t, 5)

	cachedStore, err := NewCachedStore(store)
	require.NoError(t, err)

	height, err := cachedStore.Height(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(5), height)
}

func TestCachedStore_DelegatesGetState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	chainID := t.Name()

	kv, err := NewTestInMemoryKVStore()
	require.NoError(t, err)

	s := New(kv)

	// Create a block and update state
	batch, err := s.NewBatch(ctx)
	require.NoError(t, err)

	header, data := types.GetRandomBlock(1, 5, chainID)
	sig := types.Signature([]byte("test-signature"))
	err = batch.SaveBlockData(header, data, &sig)
	require.NoError(t, err)
	err = batch.SetHeight(1)
	require.NoError(t, err)

	state := types.State{
		ChainID:         chainID,
		InitialHeight:   1,
		LastBlockHeight: 1,
		DAHeight:        100,
	}
	err = batch.UpdateState(state)
	require.NoError(t, err)

	err = batch.Commit()
	require.NoError(t, err)

	cachedStore, err := NewCachedStore(s)
	require.NoError(t, err)

	retrievedState, err := cachedStore.GetState(ctx)
	require.NoError(t, err)
	assert.Equal(t, chainID, retrievedState.ChainID)
	assert.Equal(t, uint64(1), retrievedState.LastBlockHeight)
}

func TestCachedStore_Close(t *testing.T) {
	t.Parallel()

	store := createTestStoreWithBlocks(t, 3)

	cachedStore, err := NewCachedStore(store)
	require.NoError(t, err)

	// Close should not error
	err = cachedStore.Close()
	require.NoError(t, err)
}
