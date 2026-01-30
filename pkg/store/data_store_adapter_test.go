package store

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/celestiaorg/go-header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/types"
)

// computeDataIndexHash computes the hash used for indexing in the store.
// The store indexes by sha256(signedHeader.MarshalBinary()), so for data tests
// we need to use the header hash from the saved block.
func computeDataIndexHash(h *types.SignedHeader) []byte {
	blob, _ := h.MarshalBinary()
	hash := sha256.Sum256(blob)
	return hash[:]
}

func TestDataStoreAdapter_NewDataStoreAdapter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	adapter := NewDataStoreAdapter(store)
	require.NotNil(t, adapter)

	// Initially, height should be 0
	assert.Equal(t, uint64(0), adapter.Height())

	// Head should return ErrNotFound when empty
	_, err = adapter.Head(ctx)
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestDataStoreAdapter_AppendAndRetrieve(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Create test data
	_, d1 := types.GetRandomBlock(1, 2, "test-chain")
	_, d2 := types.GetRandomBlock(2, 2, "test-chain")

	// Append data
	err = adapter.Append(ctx, d1, d2)
	require.NoError(t, err)

	// Check height is updated
	assert.Equal(t, uint64(2), adapter.Height())

	// Retrieve by height
	retrieved, err := adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, d1.Height(), retrieved.Height())

	retrieved, err = adapter.GetByHeight(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, d2.Height(), retrieved.Height())

	// Head should return the latest
	head, err := adapter.Head(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), head.Height())
}

func TestDataStoreAdapter_Get(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	// First save via the underlying store to get proper header hash
	h1, d1 := types.GetRandomBlock(1, 2, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &h1.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	// Create adapter after data is in store
	adapter := NewDataStoreAdapter(store)

	// Get by hash - need to use the index hash (sha256 of marshaled SignedHeader)
	hash := computeDataIndexHash(h1)
	retrieved, err := adapter.Get(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, d1.Height(), retrieved.Height())

	// Get non-existent hash
	_, err = adapter.Get(ctx, []byte("nonexistent"))
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestDataStoreAdapter_Has(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	h1, d1 := types.GetRandomBlock(1, 2, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &h1.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	// Create adapter after data is in store
	adapter := NewDataStoreAdapter(store)

	// Has should return true for existing data - use index hash
	has, err := adapter.Has(ctx, computeDataIndexHash(h1))
	require.NoError(t, err)
	assert.True(t, has)

	// Has should return false for non-existent
	has, err = adapter.Has(ctx, []byte("nonexistent"))
	require.NoError(t, err)
	assert.False(t, has)
}

func TestDataStoreAdapter_HasAt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	_, d1 := types.GetRandomBlock(1, 2, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1))

	// HasAt should return true for existing height
	assert.True(t, adapter.HasAt(ctx, 1))

	// HasAt should return false for non-existent height
	assert.False(t, adapter.HasAt(ctx, 2))
}

func TestDataStoreAdapter_GetRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Create and append multiple data blocks
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1, d2, d3))

	// GetRange [1, 3) should return data 1 and 2
	dataList, err := adapter.GetRange(ctx, 1, 3)
	require.NoError(t, err)
	require.Len(t, dataList, 2)
	assert.Equal(t, uint64(1), dataList[0].Height())
	assert.Equal(t, uint64(2), dataList[1].Height())

	// GetRange with from >= to should return nil
	dataList, err = adapter.GetRange(ctx, 3, 3)
	require.NoError(t, err)
	assert.Nil(t, dataList)
}

func TestDataStoreAdapter_GetRangeByHeight(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1, d2, d3))

	// GetRangeByHeight from d1 to 4 should return data 2 and 3
	dataList, err := adapter.GetRangeByHeight(ctx, d1, 4)
	require.NoError(t, err)
	require.Len(t, dataList, 2)
	assert.Equal(t, uint64(2), dataList[0].Height())
	assert.Equal(t, uint64(3), dataList[1].Height())
}

func TestDataStoreAdapter_Init(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")

	// Init should save the data
	err = adapter.Init(ctx, d1)
	require.NoError(t, err)

	// Verify it's stored
	retrieved, err := adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, d1.Height(), retrieved.Height())

	// Init again should be a no-op (already initialized)
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	err = adapter.Init(ctx, d2)
	require.NoError(t, err)

	// Height 2 should not be stored since Init was already done
	assert.False(t, adapter.HasAt(ctx, 2))
}

func TestDataStoreAdapter_Tail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Tail on empty store should return ErrNotFound
	_, err = adapter.Tail(ctx)
	assert.ErrorIs(t, err, header.ErrNotFound)

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1, d2))

	// Tail should return the first data
	tail, err := adapter.Tail(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), tail.Height())
}

func TestDataStoreAdapter_StartStop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Start should not error
	err = adapter.Start(ctx)
	require.NoError(t, err)

	// Stop should not error
	err = adapter.Stop(ctx)
	require.NoError(t, err)
}

func TestDataStoreAdapter_DeleteRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1, d2, d3))

	assert.Equal(t, uint64(3), adapter.Height())

	// DeleteRange should update cached height
	err = adapter.DeleteRange(ctx, 2, 4)
	require.NoError(t, err)

	// Cached height should be updated to 1
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestDataStoreAdapter_OnDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1, d2))

	// Track deleted heights
	var deletedHeights []uint64
	adapter.OnDelete(func(ctx context.Context, height uint64) error {
		deletedHeights = append(deletedHeights, height)
		return nil
	})

	err = adapter.DeleteRange(ctx, 1, 3)
	require.NoError(t, err)

	assert.Equal(t, []uint64{1, 2}, deletedHeights)
}

func TestDataStoreAdapter_RefreshHeight(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Save a block directly to the underlying store
	h1, d1 := types.GetRandomBlock(1, 1, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &types.Signature{}))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	// Adapter height may be stale
	// RefreshHeight should update it
	err = adapter.RefreshHeight(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestDataStoreAdapter_SetHeight(t *testing.T) {
	t.Parallel()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	adapter.SetHeight(42)
	assert.Equal(t, uint64(42), adapter.Height())
}

func TestDataStoreAdapter_AppendSkipsExisting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	_, d1 := types.GetRandomBlock(1, 2, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1))

	// Append the same data again should not error (skips existing)
	err = adapter.Append(ctx, d1)
	require.NoError(t, err)

	// Height should still be 1
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestDataStoreAdapter_AppendNilData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Append with nil and empty should not error
	err = adapter.Append(ctx)
	require.NoError(t, err)

	var nilData *types.Data
	err = adapter.Append(ctx, nilData)
	require.NoError(t, err)

	assert.Equal(t, uint64(0), adapter.Height())
}

func TestDataStoreAdapter_Sync(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Sync should not error
	err = adapter.Sync(ctx)
	require.NoError(t, err)
}

func TestDataStoreAdapter_HeightRefreshFromStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	// Save data directly to store before creating adapter
	h1, d1 := types.GetRandomBlock(1, 1, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &types.Signature{}))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	// Create adapter - it should pick up the height from store
	adapter := NewDataStoreAdapter(store)
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestDataStoreAdapter_GetByHeightNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	_, err = adapter.GetByHeight(ctx, 999)
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestDataStoreAdapter_InitWithNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Init with nil should not error but also not mark as initialized
	err = adapter.Init(ctx, nil)
	require.NoError(t, err)

	// Should still return ErrNotFound
	_, err = adapter.Head(ctx)
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestDataStoreAdapter_ContextTimeout(t *testing.T) {
	t.Parallel()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Create a context that's already canceled
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure context is expired

	// Operations should still work with in-memory store
	// but this tests the context is being passed through
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	// Note: In-memory store doesn't actually check context, but this verifies
	// the adapter passes the context through
	_ = adapter.Append(ctx, d1)
}

func TestDataStoreAdapter_GetRangePartial(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Only append data for heights 1 and 2, not 3
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1, d2))

	// GetRange [1, 5) should return data 1 and 2 (partial result)
	dataList, err := adapter.GetRange(ctx, 1, 5)
	require.NoError(t, err)
	require.Len(t, dataList, 2)
	assert.Equal(t, uint64(1), dataList[0].Height())
	assert.Equal(t, uint64(2), dataList[1].Height())
}

func TestDataStoreAdapter_GetRangeEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// GetRange on empty store should return ErrNotFound
	_, err = adapter.GetRange(ctx, 1, 5)
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestDataStoreAdapter_MultipleAppends(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store)

	// Append data in multiple batches
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d1))
	assert.Equal(t, uint64(1), adapter.Height())

	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d2))
	assert.Equal(t, uint64(2), adapter.Height())

	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, d3))
	assert.Equal(t, uint64(3), adapter.Height())

	// Verify all data is retrievable
	for h := uint64(1); h <= 3; h++ {
		assert.True(t, adapter.HasAt(ctx, h))
	}
}
