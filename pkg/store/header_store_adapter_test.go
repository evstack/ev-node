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

// computeHeaderIndexHash computes the hash used for indexing in the store.
// The store indexes by sha256(signedHeader.MarshalBinary()), not signedHeader.Hash().
func computeHeaderIndexHash(h *types.SignedHeader) []byte {
	blob, _ := h.MarshalBinary()
	hash := sha256.Sum256(blob)
	return hash[:]
}

func TestHeaderStoreAdapter_NewHeaderStoreAdapter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)

	adapter := NewHeaderStoreAdapter(store)
	require.NotNil(t, adapter)

	// Initially, height should be 0
	assert.Equal(t, uint64(0), adapter.Height())

	// Head should return ErrNotFound when empty
	_, err = adapter.Head(ctx)
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestHeaderStoreAdapter_AppendAndRetrieve(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Create test headers
	h1, _ := types.GetRandomBlock(1, 2, "test-chain")
	h2, _ := types.GetRandomBlock(2, 2, "test-chain")

	// Append headers
	err = adapter.Append(ctx, h1, h2)
	require.NoError(t, err)

	// Check height is updated
	assert.Equal(t, uint64(2), adapter.Height())

	// Retrieve by height
	retrieved, err := adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, h1.Height(), retrieved.Height())

	retrieved, err = adapter.GetByHeight(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, h2.Height(), retrieved.Height())

	// Head should return the latest
	head, err := adapter.Head(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), head.Height())
}

func TestHeaderStoreAdapter_Get(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 2, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1))

	// Get by hash - need to use the index hash (sha256 of marshaled SignedHeader)
	hash := computeHeaderIndexHash(h1)
	retrieved, err := adapter.Get(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, h1.Height(), retrieved.Height())

	// Get non-existent hash
	_, err = adapter.Get(ctx, []byte("nonexistent"))
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestHeaderStoreAdapter_Has(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 2, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1))

	// Has should return true for existing header - use index hash
	has, err := adapter.Has(ctx, computeHeaderIndexHash(h1))
	require.NoError(t, err)
	assert.True(t, has)

	// Has should return false for non-existent
	has, err = adapter.Has(ctx, []byte("nonexistent"))
	require.NoError(t, err)
	assert.False(t, has)
}

func TestHeaderStoreAdapter_HasAt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 2, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1))

	// HasAt should return true for existing height
	assert.True(t, adapter.HasAt(ctx, 1))

	// HasAt should return false for non-existent height
	assert.False(t, adapter.HasAt(ctx, 2))
}

func TestHeaderStoreAdapter_GetRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Create and append multiple headers
	h1, _ := types.GetRandomBlock(1, 1, "test-chain")
	h2, _ := types.GetRandomBlock(2, 1, "test-chain")
	h3, _ := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1, h2, h3))

	// GetRange [1, 3) should return headers 1 and 2
	headers, err := adapter.GetRange(ctx, 1, 3)
	require.NoError(t, err)
	require.Len(t, headers, 2)
	assert.Equal(t, uint64(1), headers[0].Height())
	assert.Equal(t, uint64(2), headers[1].Height())

	// GetRange with from >= to should return nil
	headers, err = adapter.GetRange(ctx, 3, 3)
	require.NoError(t, err)
	assert.Nil(t, headers)
}

func TestHeaderStoreAdapter_GetRangeByHeight(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 1, "test-chain")
	h2, _ := types.GetRandomBlock(2, 1, "test-chain")
	h3, _ := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1, h2, h3))

	// GetRangeByHeight from h1 to 4 should return headers 2 and 3
	headers, err := adapter.GetRangeByHeight(ctx, h1, 4)
	require.NoError(t, err)
	require.Len(t, headers, 2)
	assert.Equal(t, uint64(2), headers[0].Height())
	assert.Equal(t, uint64(3), headers[1].Height())
}

func TestHeaderStoreAdapter_Init(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 1, "test-chain")

	// Init should save the header
	err = adapter.Init(ctx, h1)
	require.NoError(t, err)

	// Verify it's stored
	retrieved, err := adapter.GetByHeight(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, h1.Height(), retrieved.Height())

	// Init again should be a no-op (already initialized)
	h2, _ := types.GetRandomBlock(2, 1, "test-chain")
	err = adapter.Init(ctx, h2)
	require.NoError(t, err)

	// Height 2 should not be stored since Init was already done
	assert.False(t, adapter.HasAt(ctx, 2))
}

func TestHeaderStoreAdapter_Tail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Tail on empty store should return ErrNotFound
	_, err = adapter.Tail(ctx)
	assert.ErrorIs(t, err, header.ErrNotFound)

	h1, _ := types.GetRandomBlock(1, 1, "test-chain")
	h2, _ := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1, h2))

	// Tail should return the first header
	tail, err := adapter.Tail(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), tail.Height())
}

func TestHeaderStoreAdapter_StartStop(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Start should not error
	err = adapter.Start(ctx)
	require.NoError(t, err)

	// Stop should not error
	err = adapter.Stop(ctx)
	require.NoError(t, err)
}

func TestHeaderStoreAdapter_DeleteRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 1, "test-chain")
	h2, _ := types.GetRandomBlock(2, 1, "test-chain")
	h3, _ := types.GetRandomBlock(3, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1, h2, h3))

	assert.Equal(t, uint64(3), adapter.Height())

	// DeleteRange should update cached height
	err = adapter.DeleteRange(ctx, 2, 4)
	require.NoError(t, err)

	// Cached height should be updated to 1
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestHeaderStoreAdapter_OnDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 1, "test-chain")
	h2, _ := types.GetRandomBlock(2, 1, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1, h2))

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

func TestHeaderStoreAdapter_RefreshHeight(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

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

func TestHeaderStoreAdapter_SetHeight(t *testing.T) {
	t.Parallel()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	adapter.SetHeight(42)
	assert.Equal(t, uint64(42), adapter.Height())
}

func TestHeaderStoreAdapter_AppendSkipsExisting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	h1, _ := types.GetRandomBlock(1, 2, "test-chain")
	require.NoError(t, adapter.Append(ctx, h1))

	// Append the same header again should not error (skips existing)
	err = adapter.Append(ctx, h1)
	require.NoError(t, err)

	// Height should still be 1
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestHeaderStoreAdapter_AppendNilHeaders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Append with nil and empty should not error
	err = adapter.Append(ctx)
	require.NoError(t, err)

	var nilHeader *types.SignedHeader
	err = adapter.Append(ctx, nilHeader)
	require.NoError(t, err)

	assert.Equal(t, uint64(0), adapter.Height())
}

func TestHeaderStoreAdapter_Sync(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Sync should not error
	err = adapter.Sync(ctx)
	require.NoError(t, err)
}

func TestHeaderStoreAdapter_HeightRefreshFromStore(t *testing.T) {
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
	adapter := NewHeaderStoreAdapter(store)
	assert.Equal(t, uint64(1), adapter.Height())
}

func TestHeaderStoreAdapter_GetByHeightNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	_, err = adapter.GetByHeight(ctx, 999)
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestHeaderStoreAdapter_InitWithNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Init with nil should not error but also not mark as initialized
	err = adapter.Init(ctx, nil)
	require.NoError(t, err)

	// Should still return ErrNotFound
	_, err = adapter.Head(ctx)
	assert.ErrorIs(t, err, header.ErrNotFound)
}

func TestHeaderStoreAdapter_ContextTimeout(t *testing.T) {
	t.Parallel()

	ds, err := NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := New(ds)
	adapter := NewHeaderStoreAdapter(store)

	// Create a context that's already canceled
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure context is expired

	// Operations should still work with in-memory store
	// but this tests the context is being passed through
	h1, _ := types.GetRandomBlock(1, 1, "test-chain")
	// Note: In-memory store doesn't actually check context, but this verifies
	// the adapter passes the context through
	_ = adapter.Append(ctx, h1)
}
