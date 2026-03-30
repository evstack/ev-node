package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgstore "github.com/evstack/ev-node/pkg/store"
)

// testMemStore creates an in-memory store for testing.
func testMemStore(t *testing.T) pkgstore.Store {
	t.Helper()
	ds, err := pkgstore.NewTestInMemoryKVStore()
	require.NoError(t, err)
	return pkgstore.New(ds)
}

// writeSnapshot directly encodes and writes a snapshot into the store under
// the cache's snapshot key (storeKeyPrefix + "__snap").
func writeSnapshot(t *testing.T, st pkgstore.Store, storeKeyPrefix string, entries []snapshotEntry) {
	t.Helper()
	buf := encodeSnapshot(entries)
	require.NoError(t, st.SetMetadata(context.Background(), storeKeyPrefix+"__snap", buf))
}

// ---------------------------------------------------------------------------
// MaxDAHeight
// ---------------------------------------------------------------------------

// TestCache_MaxDAHeight verifies that daHeight tracks the maximum DA height
// across successive setDAIncluded calls.
func TestCache_MaxDAHeight(t *testing.T) {
	c := NewCache(nil, "")

	assert.Equal(t, uint64(0), c.daHeight(), "initial daHeight should be 0")

	c.setDAIncluded("hash1", 100, 1)
	assert.Equal(t, uint64(100), c.daHeight(), "after setDAIncluded(100)")

	c.setDAIncluded("hash2", 50, 2)
	assert.Equal(t, uint64(100), c.daHeight(), "after setDAIncluded(50)")

	c.setDAIncluded("hash3", 200, 3)
	assert.Equal(t, uint64(200), c.daHeight(), "after setDAIncluded(200)")
}

// ---------------------------------------------------------------------------
// RestoreFromStore
// ---------------------------------------------------------------------------

// TestCache_RestoreFromStore_EmptyChain verifies that RestoreFromStore is a
// no-op on a brand-new node (no snapshot key in the store).
func TestCache_RestoreFromStore_EmptyChain(t *testing.T) {
	st := testMemStore(t)

	c := NewCache(st, "hdr/")
	require.NoError(t, c.RestoreFromStore(context.Background()))

	assert.Equal(t, 0, c.daIncludedLen(), "no entries expected on empty chain")
	assert.Equal(t, uint64(0), c.daHeight())
}

// TestCache_RestoreFromStore_FullyFinalized verifies that when the persisted
// snapshot contains no entries (all blocks finalized, window empty) nothing is
// loaded but maxDAHeight is still zero (no in-flight state).
func TestCache_RestoreFromStore_FullyFinalized(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	// Simulate a previous run that had all blocks finalized: the snapshot is
	// empty (persistSnapshot writes an empty buf when daIncluded is empty).
	writeSnapshot(t, st, "hdr/", nil)

	c := NewCache(st, "hdr/")
	require.NoError(t, c.RestoreFromStore(ctx))

	assert.Equal(t, 0, c.daIncludedLen(), "no in-flight entries expected")
	assert.Equal(t, uint64(0), c.daHeight(), "no in-flight entries means daHeight is 0")
}

// TestCache_RestoreFromStore_InFlightWindow verifies that the in-flight entries
// encoded in the snapshot are fully recovered on restore.
func TestCache_RestoreFromStore_InFlightWindow(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	writeSnapshot(t, st, "hdr/", []snapshotEntry{
		{blockHeight: 4, daHeight: 13},
		{blockHeight: 5, daHeight: 14},
	})

	c := NewCache(st, "hdr/")
	require.NoError(t, c.RestoreFromStore(ctx))

	assert.Equal(t, 2, c.daIncludedLen(), "exactly the in-flight snapshot entries should be loaded")
	assert.Equal(t, uint64(14), c.daHeight(), "maxDAHeight should reflect the highest in-flight DA height")

	hash4, ok := c.getDAIncludedByHeight(4)
	require.True(t, ok, "height 4 should exist in daIncluded")
	assert.Equal(t, uint64(13), hash4)

	hash5, ok := c.getDAIncludedByHeight(5)
	require.True(t, ok, "height 5 should exist in daIncluded")
	assert.Equal(t, uint64(14), hash5)
}

// TestCache_RestoreFromStore_SingleEntry verifies a snapshot with one in-flight
// entry is correctly decoded.
func TestCache_RestoreFromStore_SingleEntry(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	writeSnapshot(t, st, "hdr/", []snapshotEntry{
		{blockHeight: 3, daHeight: 20},
	})

	c := NewCache(st, "hdr/")
	require.NoError(t, c.RestoreFromStore(ctx))

	assert.Equal(t, 1, c.daIncludedLen(), "one entry should be in-flight")
	assert.Equal(t, uint64(20), c.daHeight())

	_, ok := c.getDAIncludedByHeight(4)
	assert.False(t, ok, "height 4 was not in snapshot")
	_, ok = c.getDAIncludedByHeight(5)
	assert.False(t, ok, "height 5 was not in snapshot")
}

func TestCache_RestoreFromStore_NilStore(t *testing.T) {
	c := NewCache(nil, "")
	require.NoError(t, c.RestoreFromStore(context.Background()))
	assert.Equal(t, 0, c.daIncludedLen())
}

// TestCache_RestoreFromStore_PlaceholderOverwrittenByRealHash
// when a real content-hash entry is written after restore it overwrites the
// height-indexed placeholder, leaving exactly one entry per height.
func TestCache_RestoreFromStore_PlaceholderOverwrittenByRealHash(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	writeSnapshot(t, st, "hdr/", []snapshotEntry{
		{blockHeight: 3, daHeight: 99},
	})

	c := NewCache(st, "hdr/")
	require.NoError(t, c.RestoreFromStore(ctx))

	assert.Equal(t, 1, c.daIncludedLen(), "one placeholder for height 3")

	c.setDAIncluded("realHash_height3", 99, 3)

	daH, ok := c.getDAIncluded("realHash_height3")
	require.True(t, ok)
	assert.Equal(t, uint64(99), daH)

	daH2, ok := c.getDAIncludedByHeight(3)
	require.True(t, ok)
	assert.Equal(t, uint64(99), daH2)
}

// TestCache_RestoreFromStore_RoundTrip verifies that SaveToStore persists a
// snapshot that a freshly-constructed cache can fully recover.
func TestCache_RestoreFromStore_RoundTrip(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	c1 := NewCache(st, "rt/")
	c1.setDAIncluded("hashA", 10, 1)
	c1.setDAIncluded("hashB", 20, 2)
	c1.setDAIncluded("hashC", 30, 3)
	c1.removeDAIncluded("hashB")
	require.NoError(t, c1.SaveToStore(ctx))

	c2 := NewCache(st, "rt/")
	require.NoError(t, c2.RestoreFromStore(ctx))

	assert.Equal(t, 2, c2.daIncludedLen(), "only non-deleted entries should be restored")
	assert.Equal(t, uint64(30), c2.daHeight())

	_, ok := c2.getDAIncludedByHeight(1)
	assert.True(t, ok, "height 1 placeholder should exist")
	_, ok = c2.getDAIncludedByHeight(2)
	assert.False(t, ok, "height 2 was removed, should not exist")
	_, ok = c2.getDAIncludedByHeight(3)
	assert.True(t, ok, "height 3 placeholder should exist")
}

// ---------------------------------------------------------------------------
// Basic operations
// ---------------------------------------------------------------------------

func TestCache_BasicOperations(t *testing.T) {
	c := NewCache(nil, "")

	assert.False(t, c.isSeen("hash1"))
	c.setSeen("hash1", 1)
	assert.True(t, c.isSeen("hash1"))
	c.removeSeen("hash1")
	assert.False(t, c.isSeen("hash1"))

	_, ok := c.getDAIncluded("hash2")
	assert.False(t, ok)
	c.setDAIncluded("hash2", 100, 2)
	daHeight, ok := c.getDAIncluded("hash2")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)
	c.removeDAIncluded("hash2")
	_, ok = c.getDAIncluded("hash2")
	assert.False(t, ok)
}

func TestCache_DeleteAllForHeight(t *testing.T) {
	c := NewCache(nil, "")

	c.setSeen("hash1", 1)
	c.setSeen("hash2", 2)

	c.deleteAllForHeight(1)

	assert.False(t, c.isSeen("hash1"))
	assert.True(t, c.isSeen("hash2"))
}

func TestCache_WithNilStore(t *testing.T) {
	c := NewCache(nil, "")
	require.NotNil(t, c)

	c.setDAIncluded("hash1", 100, 1)
	daHeight, ok := c.getDAIncluded("hash1")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)
}

// ---------------------------------------------------------------------------
// SaveToStore / ClearFromStore
// ---------------------------------------------------------------------------

func TestCache_SaveToStore(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	c := NewCache(st, "save-test/")
	c.setDAIncluded("hash1", 100, 1)
	c.setDAIncluded("hash2", 200, 2)

	require.NoError(t, c.SaveToStore(ctx))

	raw, err := st.GetMetadata(ctx, "save-test/__snap")
	require.NoError(t, err)
	assert.Len(t, raw, 2*snapshotEntrySize, "snapshot should contain 2 entries of 16 bytes each")

	_, err = st.GetMetadata(ctx, "save-test/hash1")
	assert.Error(t, err, "per-hash keys should not exist in the snapshot design")
}

func TestCache_ClearFromStore(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	c := NewCache(st, "clear-test/")
	c.setDAIncluded("hash1", 100, 1)
	c.setDAIncluded("hash2", 200, 2)
	require.NoError(t, c.SaveToStore(ctx))

	_, err := st.GetMetadata(ctx, "clear-test/__snap")
	require.NoError(t, err, "snapshot key should exist before ClearFromStore")

	require.NoError(t, c.ClearFromStore(ctx))

	_, err = st.GetMetadata(ctx, "clear-test/__snap")
	assert.Error(t, err, "snapshot key should have been removed from store")
}

// ---------------------------------------------------------------------------
// heightPlaceholderKey
// ---------------------------------------------------------------------------

func TestHeightPlaceholderKey(t *testing.T) {
	k0 := HeightPlaceholderKey("pfx/", 0)
	k1 := HeightPlaceholderKey("pfx/", 1)
	kMax := HeightPlaceholderKey("pfx/", ^uint64(0))

	assert.NotEqual(t, k0, k1)
	assert.NotEqual(t, k1, kMax)

	assert.Contains(t, k0, "pfx/")
	assert.Contains(t, k1, "pfx/")
	assert.Contains(t, kMax, "pfx/")

	assert.NotEqual(t, HeightPlaceholderKey("a/", 1), HeightPlaceholderKey("b/", 1))
}

// TestCache_NoPlaceholderLeakAfterRefire verifies that when the DA retriever
// re-fires setDAIncluded with the real content hash after a restart, the
// snapshot placeholder that RestoreFromStore installed is evicted from
// daIncluded.  Without the eviction in setDAIncluded, every restart cycle
// would leak one orphaned placeholder key per in-flight block.
func TestCache_NoPlaceholderLeakAfterRefire(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	c1 := NewCache(st, "pfx/")
	c1.setDAIncluded("realHash3", 99, 3)
	require.NoError(t, c1.SaveToStore(ctx))

	c2 := NewCache(st, "pfx/")
	require.NoError(t, c2.RestoreFromStore(ctx))

	placeholder := HeightPlaceholderKey("pfx/", 3)
	_, placeholderPresent := c2.getDAIncluded(placeholder)
	require.True(t, placeholderPresent, "placeholder must be present immediately after restore")
	assert.Equal(t, 1, c2.daIncludedLen(), "only one entry expected before re-fire")

	c2.setDAIncluded("realHash3", 99, 3)

	daH, ok := c2.getDAIncluded("realHash3")
	require.True(t, ok, "real hash must be present after re-fire")
	assert.Equal(t, uint64(99), daH)

	_, placeholderPresent = c2.getDAIncluded(placeholder)
	assert.False(t, placeholderPresent, "placeholder must be evicted after real hash is written")

	assert.Equal(t, 1, c2.daIncludedLen(), "exactly one daIncluded entry after re-fire — no orphan")
}

// TestCache_RestartIdempotent verifies that multiple successive restarts all
// yield a correctly functioning cache — i.e. the snapshot written after a
// re-fire is identical in semantics to the original, so a second (or third)
// restart still loads the right DA height via the placeholder fallback and the
// snapshot never grows stale or accumulates phantom entries.
func TestCache_RestartIdempotent(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	const realHash = "realHashH5"
	const blockH = uint64(5)
	const daH = uint64(42)

	c1 := NewCache(st, "pfx/")
	c1.setDAIncluded(realHash, daH, blockH)
	require.NoError(t, c1.SaveToStore(ctx))

	for restart := 1; restart <= 3; restart++ {
		cR := NewCache(st, "pfx/")
		require.NoError(t, cR.RestoreFromStore(ctx), "restart %d: RestoreFromStore", restart)

		assert.Equal(t, 1, cR.daIncludedLen(), "restart %d: one placeholder entry", restart)
		assert.Equal(t, daH, cR.daHeight(), "restart %d: daHeight correct", restart)

		gotDAH, ok := cR.getDAIncludedByHeight(blockH)
		require.True(t, ok, "restart %d: height-based lookup must succeed", restart)
		assert.Equal(t, daH, gotDAH, "restart %d: height-based DA height correct", restart)

		cR.setDAIncluded(realHash, daH, blockH)
		require.NoError(t, cR.SaveToStore(ctx), "restart %d: SaveToStore", restart)

		_, realPresent := cR.getDAIncluded(realHash)
		assert.True(t, realPresent, "restart %d: real hash present after re-fire", restart)
		assert.Equal(t, 1, cR.daIncludedLen(), "restart %d: no orphan after re-fire", restart)
	}
}

// ---------------------------------------------------------------------------
// Cleanup via deleteAllForHeight
// ---------------------------------------------------------------------------

func TestCache_DeleteAllForHeight_CleansHashAndDA(t *testing.T) {
	c := NewCache(nil, "")

	c.setDAIncluded("hash1", 100, 1)
	c.setSeen("hash1", 1)
	c.setDAIncluded("hash2", 200, 2)
	c.setSeen("hash2", 2)

	assert.Equal(t, 2, c.daIncludedLen())
	assert.True(t, c.isSeen("hash1"))
	assert.True(t, c.isSeen("hash2"))

	c.deleteAllForHeight(1)

	assert.Equal(t, 1, c.daIncludedLen())
	assert.False(t, c.isSeen("hash1"))
	assert.True(t, c.isSeen("hash2"))

	_, ok := c.getDAIncludedByHeight(1)
	assert.False(t, ok)
	_, ok = c.getDAIncludedByHeight(2)
	assert.True(t, ok)
}
