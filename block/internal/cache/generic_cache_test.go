package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type testItem struct{ V int }

func init() {
	gob.Register(&testItem{})
}

// TestCache_TypeSafety ensures methods gracefully handle invalid underlying types.
func TestCache_TypeSafety(t *testing.T) {
	c := NewCache[testItem]()

	// Inject invalid value types directly into maps (bypassing typed methods)
	c.itemsByHeight.Store(uint64(1), "not-a-*testItem")

	if got := c.getItem(1); got != nil {
		t.Fatalf("expected nil for invalid stored type, got %#v", got)
	}

	// Range should skip invalid entries and not panic
	ran := false
	c.rangeByHeight(func(_ uint64, _ *testItem) bool { ran = true; return true })
	_ = ran // ensure no panic
}

// TestCache_SaveLoad_ErrorPaths covers SaveToDisk and LoadFromDisk error scenarios.
func TestCache_SaveLoad_ErrorPaths(t *testing.T) {
	c := NewCache[testItem]()
	for i := 0; i < 5; i++ {
		v := &testItem{V: i}
		c.setItem(uint64(i), v)
		c.setSeen(fmt.Sprintf("s%d", i), uint64(i))
		c.setDAIncluded(fmt.Sprintf("d%d", i), uint64(i), uint64(i))
	}

	// Normal save/load roundtrip
	dir := t.TempDir()
	if err := c.SaveToDisk(dir); err != nil {
		t.Fatalf("SaveToDisk failed: %v", err)
	}
	c2 := NewCache[testItem]()
	if err := c2.LoadFromDisk(dir); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}
	// Spot-check a few values
	if got := c2.getItem(3); got == nil || got.V != 3 {
		t.Fatalf("roundtrip getItem mismatch: got %#v", got)
	}

	_, c2OK := c2.getDAIncluded("d2")

	if !c2.isSeen("s1") || !c2OK {
		t.Fatalf("roundtrip auxiliary maps mismatch")
	}

	// SaveToDisk error: path exists as a file
	filePath := filepath.Join(t.TempDir(), "not_a_dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := c.SaveToDisk(filePath); err == nil {
		t.Fatalf("expected error when saving to a file path, got nil")
	}

	// LoadFromDisk error: corrupt gob
	badDir := t.TempDir()
	badFile := filepath.Join(badDir, itemsByHeightFilename)
	if err := os.WriteFile(badFile, []byte("not a gob"), 0o600); err != nil {
		t.Fatalf("failed to write bad gob: %v", err)
	}
	c3 := NewCache[testItem]()
	if err := c3.LoadFromDisk(badDir); err == nil {
		t.Fatalf("expected error when loading corrupted gob, got nil")
	}
}

// TestCache_LargeDataset covers edge cases with height index management at scale.
func TestCache_LargeDataset(t *testing.T) {
	c := NewCache[testItem]()
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

// TestCache_PruneOldEntries tests pruning of old entries from the cache
func TestCache_PruneOldEntries(t *testing.T) {
	c := NewCache[testItem]()

	// Add items at various heights
	for i := uint64(1); i <= 100; i++ {
		c.setItem(i, &testItem{V: int(i)})
		c.setSeen(fmt.Sprintf("hash-%d", i), i)
		c.setDAIncluded(fmt.Sprintf("hash-%d", i), i, i)
	}

	// Prune everything below height 50 with retention of 10
	// This should keep heights 40-100 (61 items)
	c.pruneOldEntries(50, 10)

	// Verify items below height 40 are removed
	for i := uint64(1); i < 40; i++ {
		if item := c.getItem(i); item != nil {
			t.Errorf("expected item at height %d to be pruned, but found %v", i, item)
		}
	}

	// Verify items >= 40 are kept
	for i := uint64(40); i <= 100; i++ {
		if item := c.getItem(i); item == nil {
			t.Errorf("expected item at height %d to be kept, but it was pruned", i)
		}
	}

	// Note: Hashes are currently NOT pruned - this is a known memory leak
	// We verify that old hashes still exist to document current behavior
	if !c.isSeen("hash-50") {
		t.Error("expected hash-50 to still be marked as seen")
	}
	if _, ok := c.getDAIncluded("hash-50"); !ok {
		t.Error("expected hash-50 to still have DA inclusion")
	}
}

// TestCache_PruneOldEntries_EdgeCases tests edge cases for pruning
func TestCache_PruneOldEntries_EdgeCases(t *testing.T) {
	t.Run("prune with current height less than retention", func(t *testing.T) {
		c := NewCache[testItem]()
		for i := uint64(1); i <= 5; i++ {
			c.setItem(i, &testItem{V: int(i)})
		}
		// With retention of 10 and current height 5, nothing should be pruned
		c.pruneOldEntries(5, 10)

		// All items should remain
		for i := uint64(1); i <= 5; i++ {
			if item := c.getItem(i); item == nil {
				t.Errorf("expected item at height %d to be kept", i)
			}
		}
	})

	t.Run("prune with zero retention", func(t *testing.T) {
		c := NewCache[testItem]()
		for i := uint64(1); i <= 10; i++ {
			c.setItem(i, &testItem{V: int(i)})
		}
		// With zero retention, only items >= current height should remain
		c.pruneOldEntries(5, 0)

		for i := uint64(1); i < 5; i++ {
			if item := c.getItem(i); item != nil {
				t.Errorf("expected item at height %d to be pruned", i)
			}
		}
		for i := uint64(5); i <= 10; i++ {
			if item := c.getItem(i); item == nil {
				t.Errorf("expected item at height %d to be kept", i)
			}
		}
	})

	t.Run("prune empty cache", func(t *testing.T) {
		c := NewCache[testItem]()
		// Should not panic
		c.pruneOldEntries(100, 10)
	})

	t.Run("prune with current height zero", func(t *testing.T) {
		c := NewCache[testItem]()
		for i := uint64(1); i <= 5; i++ {
			c.setItem(i, &testItem{V: int(i)})
		}
		// Should not panic or prune anything
		c.pruneOldEntries(0, 10)

		// All items should remain
		for i := uint64(1); i <= 5; i++ {
			if item := c.getItem(i); item == nil {
				t.Errorf("expected item at height %d to be kept", i)
			}
		}
	})
}

// TestCache_HashMapPruning verifies that hash maps are properly pruned
// to prevent memory leaks in long-running chains.
func TestCache_HashMapPruning(t *testing.T) {
	c := NewCache[testItem]()

	// Simulate a long-running chain with 10,000 blocks
	const totalBlocks = 10000
	const retentionWindow = 1000

	// Track how many hashes we add
	hashCount := 0

	for i := uint64(1); i <= totalBlocks; i++ {
		c.setItem(i, &testItem{V: int(i)})
		c.setSeen(fmt.Sprintf("hash-%d", i), i)
		c.setDAIncluded(fmt.Sprintf("hash-%d", i), i, i)
		hashCount++

		// Prune every 100 blocks to simulate real usage
		if i%100 == 0 && i > retentionWindow {
			c.pruneOldEntries(i, retentionWindow)
		}
	}

	// Count how many items remain in itemsByHeight (should be ~1000)
	itemCount := 0
	c.itemsByHeight.Range(func(k, v any) bool {
		itemCount++
		return true
	})

	// Count how many hashes remain (this will be ALL 10,000 - MEMORY LEAK!)
	seenHashCount := 0
	c.hashes.Range(func(k, v any) bool {
		seenHashCount++
		return true
	})

	daIncludedCount := 0
	c.daIncluded.Range(func(k, v any) bool {
		daIncludedCount++
		return true
	})

	t.Logf("After processing %d blocks with retention %d:", totalBlocks, retentionWindow)
	t.Logf("  itemsByHeight count: %d (expected ~%d)", itemCount, retentionWindow)
	t.Logf("  hashes count: %d (expected ~%d)", seenHashCount, retentionWindow)
	t.Logf("  daIncluded count: %d (expected ~%d)", daIncludedCount, retentionWindow)

	// Verify all maps are pruned correctly
	maxAllowed := int(retentionWindow * 1.1) // Allow 10% margin

	if itemCount > maxAllowed {
		t.Errorf("itemsByHeight not pruned correctly: got %d, want ~%d", itemCount, retentionWindow)
	}

	if seenHashCount > maxAllowed {
		t.Errorf("hashes not pruned correctly: got %d, want ~%d", seenHashCount, retentionWindow)
	}

	if daIncludedCount > maxAllowed {
		t.Errorf("daIncluded not pruned correctly: got %d, want ~%d", daIncludedCount, retentionWindow)
	}

	// Calculate memory saved by pruning
	bytesPerEntry := 100 // Approximate bytes per hash entry
	expectedWithoutPruning := hashCount
	actualEntries := seenHashCount
	savedEntries := expectedWithoutPruning - actualEntries
	savedMB := float64(savedEntries*bytesPerEntry) / (1024 * 1024)
	t.Logf("Memory saved by pruning: %.2f MB", savedMB)
	t.Logf("At 1 block/sec, pruning prevents %.2f MB/month memory growth",
		savedMB*30*24*60*60/float64(totalBlocks))
}
