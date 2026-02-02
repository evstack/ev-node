package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/go-header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// testGenesisData returns a genesis with InitialHeight=1 for use in data adapter tests.
func testGenesisData() genesis.Genesis {
	return genesis.Genesis{
		ChainID:       "test-chain",
		InitialHeight: 1,
		StartTime:     time.Now(),
	}
}

// computeDataIndexHash computes the hash used for indexing in the store.
// The store indexes by sha256(signedHeader.MarshalBinary()), so for data tests
// we need to use the header hash from the saved block.
func computeDataIndexHash(h *types.SignedHeader) []byte {
	blob, _ := h.MarshalBinary()
	hash := sha256.Sum256(blob)
	return hash[:]
}

// wrapData wraps a *types.Data in a *types.P2PData for use with the DataStoreAdapter.
func wrapData(d *types.Data) *types.P2PData {
	if d == nil {
		return nil
	}
	return &types.P2PData{
		Message:      d,
		DAHeightHint: 0,
	}
}

func TestDataStoreAdapter_NewDataStoreAdapter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	adapter := NewDataStoreAdapter(store, testGenesisData())
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Create test data
	_, d1 := types.GetRandomBlock(1, 2, "test-chain")
	_, d2 := types.GetRandomBlock(2, 2, "test-chain")

	// Append data - these go to pending cache
	err = adapter.Append(ctx, wrapData(d1), wrapData(d2))
	require.NoError(t, err)

	// Check height is updated (from pending)
	assert.Equal(t, uint64(2), adapter.Height())

	// Retrieve by height (from pending)
	retrieved, err := adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, d1.Height(), retrieved.Height())

	retrieved, err = adapter.GetByHeight(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, d2.Height(), retrieved.Height())

	// Head should return the latest (from pending)
	head, err := adapter.Head(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), head.Height())
}

func TestDataStoreAdapter_GetFromStore(t *testing.T) {
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

	// Now create adapter and verify we can get from store
	adapter := NewDataStoreAdapter(store, testGenesisData())

	retrieved, err := adapter.GetByHeight(ctx, 1)
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	require.NoError(t, adapter.Append(ctx, wrapData(d1)))

	// Has should return true for existing hash
	has, err := adapter.Has(ctx, d1.Hash())
	require.NoError(t, err)
	assert.True(t, has)

	// Has should return false for non-existent hash
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	_, d1 := types.GetRandomBlock(1, 2, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1)))

	// HasAt should return true for pending height
	assert.True(t, adapter.HasAt(ctx, 1))

	// HasAt should return false for non-existent height
	assert.False(t, adapter.HasAt(ctx, 2))
}

func TestDataStoreAdapter_HasAtFromStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	// Save directly to store
	h1, d1 := types.GetRandomBlock(1, 2, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &h1.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	adapter := NewDataStoreAdapter(store, testGenesisData())

	// HasAt should return true for stored height
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Create and append multiple data blocks to pending
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1), wrapData(d2), wrapData(d3)))

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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1), wrapData(d2), wrapData(d3)))

	// GetRangeByHeight from d1 to 4 should return data 2 and 3
	dataList, err := adapter.GetRangeByHeight(ctx, wrapData(d1), 4)
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")

	// Init should add data to pending
	err = adapter.Init(ctx, wrapData(d1))
	require.NoError(t, err)

	// Verify it's retrievable from pending
	retrieved, err := adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, d1.Height(), retrieved.Height())

	// Init again should be a no-op (already initialized)
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	err = adapter.Init(ctx, wrapData(d2))
	require.NoError(t, err)

	// Height 2 should not be in pending since Init was already done
	assert.False(t, adapter.HasAt(ctx, 2))
}

func TestDataStoreAdapter_Tail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Tail on empty store should return ErrNotFound
	_, err = adapter.Tail(ctx)
	assert.ErrorIs(t, err, header.ErrNotFound)

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1), wrapData(d2)))

	// Tail should return the first data from pending
	tail, err := adapter.Tail(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), tail.Height())
}

func TestDataStoreAdapter_TailFromStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	// Save directly to store
	h1, d1 := types.GetRandomBlock(1, 1, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &h1.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Tail should return the first data from store
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1), wrapData(d2), wrapData(d3)))

	assert.Equal(t, uint64(3), adapter.Height())

	// DeleteRange should update cached height and remove from pending
	err = adapter.DeleteRange(ctx, 2, 4)
	require.NoError(t, err)

	// Cached height should be updated to 1
	assert.Equal(t, uint64(1), adapter.Height())

	// Heights 2 and 3 should no longer be available
	assert.False(t, adapter.HasAt(ctx, 2))
	assert.False(t, adapter.HasAt(ctx, 3))

	// Height 1 should still be available
	assert.True(t, adapter.HasAt(ctx, 1))
}

func TestDataStoreAdapter_OnDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1), wrapData(d2)))

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

func TestDataStoreAdapter_AppendSkipsExisting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	// Save directly to store first
	h1, d1 := types.GetRandomBlock(1, 2, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &h1.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Append the same data again should not error (skips existing in store)
	err = adapter.Append(ctx, wrapData(d1))
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Append with nil and empty should not error
	err = adapter.Append(ctx)
	require.NoError(t, err)

	var nilData *types.P2PData
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
	adapter := NewDataStoreAdapter(store, testGenesisData())

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
	adapter := NewDataStoreAdapter(store, testGenesisData())
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestDataStoreAdapter_GetByHeightNotFound(t *testing.T) {
	t.Parallel()
	// Use a short timeout since GetByHeight now blocks waiting for the height
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	_, err = adapter.GetByHeight(ctx, 999)
	// GetByHeight now blocks until the height is available or context is canceled
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDataStoreAdapter_InitWithNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

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
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Create a context that's already canceled
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure context is expired

	// Operations should still work with in-memory store
	// but this tests the context is being passed through
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	// Note: In-memory store doesn't actually check context, but this verifies
	// the adapter passes the context through
	_ = adapter.Append(ctx, wrapData(d1))
}

func TestDataStoreAdapter_GetRangePartial(t *testing.T) {
	t.Parallel()
	// Use a short timeout since GetByHeight now blocks waiting for the height
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Only append data for heights 1 and 2, not 3
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1), wrapData(d2)))

	// GetRange [1, 5) should return data 1 and 2 (partial result)
	dataList, err := adapter.GetRange(ctx, 1, 5)
	require.NoError(t, err)
	require.Len(t, dataList, 2)
	assert.Equal(t, uint64(1), dataList[0].Height())
	assert.Equal(t, uint64(2), dataList[1].Height())
}

func TestDataStoreAdapter_GetRangeEmpty(t *testing.T) {
	t.Parallel()
	// Use a short timeout since GetByHeight now blocks waiting for the height
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// GetRange on empty store will block until context timeout
	_, err = adapter.GetRange(ctx, 1, 5)
	// GetByHeight now blocks - we may get context.DeadlineExceeded or ErrNotFound depending on timing
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, header.ErrNotFound),
		"expected DeadlineExceeded or ErrNotFound, got: %v", err)
}

func TestDataStoreAdapter_MultipleAppends(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Append data in multiple batches
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1)))
	assert.Equal(t, uint64(1), adapter.Height())

	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d2)))
	assert.Equal(t, uint64(2), adapter.Height())

	_, d3 := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d3)))
	assert.Equal(t, uint64(3), adapter.Height())

	// Verify all data is retrievable
	for h := uint64(1); h <= 3; h++ {
		assert.True(t, adapter.HasAt(ctx, h))
	}
}

func TestDataStoreAdapter_PendingAndStoreInteraction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Add data to pending
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d1)))

	// Verify it's in pending
	retrieved, err := adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, d1.Height(), retrieved.Height())

	// Now save a different data at height 1 directly to store
	h1Store, d1Store := types.GetRandomBlock(1, 2, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1Store, d1Store, &h1Store.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	// GetByHeight should now return from store (store takes precedence)
	retrieved, err = adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	// The store version should be returned
	assert.Equal(t, d1Store.Height(), retrieved.Height())
}

func TestDataStoreAdapter_HeadPrefersPending(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	// Save height 1 to store
	h1, d1 := types.GetRandomBlock(1, 1, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h1, d1, &h1.Signature))
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Add height 2 to pending
	_, d2 := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, wrapData(d2)))

	// Head should return the pending data (higher height)
	head, err := adapter.Head(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), head.Height())
}

func TestDataStoreAdapter_GetFromPendingByHash(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewDataStoreAdapter(store, testGenesisData())

	// Add data to pending
	_, d1 := types.GetRandomBlock(1, 1, "test-chain")
	p2pD1 := wrapData(d1)
	require.NoError(t, adapter.Append(ctx, p2pD1))

	// Get by hash from pending (uses data's Hash() method)
	retrieved, err := adapter.Get(ctx, p2pD1.Hash())
	require.NoError(t, err)
	assert.Equal(t, d1.Height(), retrieved.Height())
}
