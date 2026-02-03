package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/store"
)

type testItem struct{ V int }

// memStore creates an in-memory store for testing
func testMemStore(t *testing.T) store.Store {
	ds, err := store.NewTestInMemoryKVStore()
	require.NoError(t, err)
	return store.New(ds)
}

// TestCache_MaxDAHeight verifies that daHeight tracks the maximum DA height
func TestCache_MaxDAHeight(t *testing.T) {
	c := NewCache[testItem](nil, "")

	// Initially should be 0
	if got := c.daHeight(); got != 0 {
		t.Errorf("initial daHeight = %d, want 0", got)
	}

	// Set items with increasing DA heights
	c.setDAIncluded("hash1", 100, 1)
	if got := c.daHeight(); got != 100 {
		t.Errorf("after setDAIncluded(100): daHeight = %d, want 100", got)
	}

	c.setDAIncluded("hash2", 50, 2) // Lower height shouldn't change max
	if got := c.daHeight(); got != 100 {
		t.Errorf("after setDAIncluded(50): daHeight = %d, want 100", got)
	}

	c.setDAIncluded("hash3", 200, 3)
	if got := c.daHeight(); got != 200 {
		t.Errorf("after setDAIncluded(200): daHeight = %d, want 200", got)
	}
}

// TestCache_MaxDAHeight_WithStore verifies that daHeight is restored from store
func TestCache_MaxDAHeight_WithStore(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	c1 := NewCache[testItem](st, "test/da-included/")

	// Set DA included entries
	c1.setDAIncluded("hash1", 100, 1)
	c1.setDAIncluded("hash2", 200, 2)
	c1.setDAIncluded("hash3", 150, 3)

	if got := c1.daHeight(); got != 200 {
		t.Errorf("after setDAIncluded: daHeight = %d, want 200", got)
	}

	err := c1.SaveToStore(ctx)
	require.NoError(t, err)

	// Create new cache and restore from store
	c2 := NewCache[testItem](st, "test/da-included/")

	// Restore with the known hashes
	err = c2.RestoreFromStore(ctx, []string{"hash1", "hash2", "hash3"})
	require.NoError(t, err)

	if got := c2.daHeight(); got != 200 {
		t.Errorf("after restore: daHeight = %d, want 200", got)
	}

	// Verify individual entries were restored
	daHeight, ok := c2.getDAIncluded("hash1")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)

	daHeight, ok = c2.getDAIncluded("hash2")
	assert.True(t, ok)
	assert.Equal(t, uint64(200), daHeight)

	daHeight, ok = c2.getDAIncluded("hash3")
	assert.True(t, ok)
	assert.Equal(t, uint64(150), daHeight)
}

// TestCache_WithStorePersistence tests that DA inclusion is persisted to store
func TestCache_WithStorePersistence(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	c1 := NewCache[testItem](st, "test/")

	// Set DA inclusion
	c1.setDAIncluded("hash1", 100, 1)
	c1.setDAIncluded("hash2", 200, 2)

	err := c1.SaveToStore(ctx)
	require.NoError(t, err)

	// Create new cache with same store and restore
	c2 := NewCache[testItem](st, "test/")

	err = c2.RestoreFromStore(ctx, []string{"hash1", "hash2", "hash3"})
	require.NoError(t, err)

	// hash1 and hash2 should be restored, hash3 should not exist
	daHeight, ok := c2.getDAIncluded("hash1")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)

	daHeight, ok = c2.getDAIncluded("hash2")
	assert.True(t, ok)
	assert.Equal(t, uint64(200), daHeight)

	_, ok = c2.getDAIncluded("hash3")
	assert.False(t, ok)
}

// TestCache_LargeDataset covers edge cases with height index management at scale.
func TestCache_LargeDataset(t *testing.T) {
	c := NewCache[testItem](nil, "")
	const N = 20000
	// Insert in descending order to exercise insert positions
	for i := N - 1; i >= 0; i-- {
		v := &testItem{V: i}
		c.setItem(uint64(i), v)
	}
	// Delete a range in the middle
	for i := 5000; i < 10000; i += 2 {
		c.getNextItem(uint64(i))
	}
}

// TestCache_BasicOperations tests basic cache operations
func TestCache_BasicOperations(t *testing.T) {
	c := NewCache[testItem](nil, "")

	// Test setItem/getItem
	item := &testItem{V: 42}
	c.setItem(1, item)
	got := c.getItem(1)
	assert.NotNil(t, got)
	assert.Equal(t, 42, got.V)

	// Test getItem for non-existent key
	got = c.getItem(999)
	assert.Nil(t, got)

	// Test setSeen/isSeen
	assert.False(t, c.isSeen("hash1"))
	c.setSeen("hash1", 1)
	assert.True(t, c.isSeen("hash1"))

	// Test removeSeen
	c.removeSeen("hash1")
	assert.False(t, c.isSeen("hash1"))

	// Test setDAIncluded/getDAIncluded
	_, ok := c.getDAIncluded("hash2")
	assert.False(t, ok)

	c.setDAIncluded("hash2", 100, 2)
	daHeight, ok := c.getDAIncluded("hash2")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)

	// Test removeDAIncluded
	c.removeDAIncluded("hash2")
	_, ok = c.getDAIncluded("hash2")
	assert.False(t, ok)
}

// TestCache_GetNextItem tests the atomic get-and-remove operation
func TestCache_GetNextItem(t *testing.T) {
	c := NewCache[testItem](nil, "")

	// Set multiple items
	c.setItem(1, &testItem{V: 1})
	c.setItem(2, &testItem{V: 2})
	c.setItem(3, &testItem{V: 3})

	// Get and remove item at height 2
	got := c.getNextItem(2)
	assert.NotNil(t, got)
	assert.Equal(t, 2, got.V)

	// Item should be removed
	got = c.getNextItem(2)
	assert.Nil(t, got)

	// Other items should still exist
	got = c.getItem(1)
	assert.NotNil(t, got)
	assert.Equal(t, 1, got.V)

	got = c.getItem(3)
	assert.NotNil(t, got)
	assert.Equal(t, 3, got.V)
}

// TestCache_DeleteAllForHeight tests deleting all data for a specific height
func TestCache_DeleteAllForHeight(t *testing.T) {
	c := NewCache[testItem](nil, "")

	// Set items at different heights
	c.setItem(1, &testItem{V: 1})
	c.setItem(2, &testItem{V: 2})
	c.setSeen("hash1", 1)
	c.setSeen("hash2", 2)

	// Delete height 1
	c.deleteAllForHeight(1)

	// Height 1 data should be gone
	assert.Nil(t, c.getItem(1))
	assert.False(t, c.isSeen("hash1"))

	// Height 2 data should still exist
	assert.NotNil(t, c.getItem(2))
	assert.True(t, c.isSeen("hash2"))
}

// TestCacheWithConfig tests creating cache with custom config
func TestCache_WithNilStore(t *testing.T) {
	// Cache without store should work fine
	c := NewCache[testItem](nil, "")
	require.NotNil(t, c)

	// Basic operations should work
	c.setItem(1, &testItem{V: 1})
	got := c.getItem(1)
	assert.NotNil(t, got)
	assert.Equal(t, 1, got.V)

	// DA inclusion should work (just not persisted)
	c.setDAIncluded("hash1", 100, 1)
	daHeight, ok := c.getDAIncluded("hash1")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)
}

// TestCache_SaveToStore tests the SaveToStore method
func TestCache_SaveToStore(t *testing.T) {
	st := testMemStore(t)
	ctx := context.Background()

	c := NewCache[testItem](st, "save-test/")

	// Set some DA included entries
	c.setDAIncluded("hash1", 100, 1)
	c.setDAIncluded("hash2", 200, 2)

	// Save to store (should be a no-op since we persist on setDAIncluded)
	err := c.SaveToStore(ctx)
	require.NoError(t, err)

	// Verify data is in store by creating new cache and restoring
	c2 := NewCache[testItem](st, "save-test/")

	err = c2.RestoreFromStore(ctx, []string{"hash1", "hash2"})
	require.NoError(t, err)

	daHeight, ok := c2.getDAIncluded("hash1")
	assert.True(t, ok)
	assert.Equal(t, uint64(100), daHeight)

	daHeight, ok = c2.getDAIncluded("hash2")
	assert.True(t, ok)
	assert.Equal(t, uint64(200), daHeight)
}
