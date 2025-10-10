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

	// Prune everything below height 50 (DA included height)
	// This should keep only heights >= 50
	c.pruneOldEntries(50)

	// Verify items below height 50 are removed
	for i := uint64(1); i < 50; i++ {
		if item := c.getItem(i); item != nil {
			t.Errorf("expected item at height %d to be pruned, but found %v", i, item)
		}
	}

	// Verify items >= 50 are kept
	for i := uint64(50); i <= 100; i++ {
		if item := c.getItem(i); item == nil {
			t.Errorf("expected item at height %d to be kept, but it was pruned", i)
		}
	}

	// Verify that hashes are pruned correctly
	if c.isSeen("hash-10") {
		t.Error("expected hash-10 to be pruned")
	}
	if !c.isSeen("hash-50") {
		t.Error("expected hash-50 to remain (at DA included height)")
	}
	if !c.isSeen("hash-51") {
		t.Error("expected hash-51 to still be marked as seen")
	}
}

// TestCache_PruneOldEntries_EdgeCases tests edge cases for pruning
func TestCache_PruneOldEntries_EdgeCases(t *testing.T) {
	t.Run("prune below current height", func(t *testing.T) {
		c := NewCache[testItem]()
		for i := uint64(1); i <= 10; i++ {
			c.setItem(i, &testItem{V: int(i)})
		}
		// Prune at height 5, only items >= 5 should remain
		c.pruneOldEntries(5)

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
		c.pruneOldEntries(100)
	})

	t.Run("prune with current height zero", func(t *testing.T) {
		c := NewCache[testItem]()
		for i := uint64(1); i <= 5; i++ {
			c.setItem(i, &testItem{V: int(i)})
		}
		// Should not panic or prune anything
		c.pruneOldEntries(0)

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

	// Track how many hashes we add
	hashCount := 0

	for i := uint64(1); i <= totalBlocks; i++ {
		c.setItem(i, &testItem{V: int(i)})
		c.setSeen(fmt.Sprintf("hash-%d", i), i)
		c.setDAIncluded(fmt.Sprintf("hash-%d", i), i, i)
		hashCount++

		// Prune periodically to simulate real usage (keep last 100 blocks as "DA included")
		if i%100 == 0 && i >= 100 {
			// Prune up to i-100, keeping the last 100 blocks
			c.pruneOldEntries(i - 100)
		}
	}

	// Final prune to keep only the last 100 blocks (DA included height simulation)
	c.pruneOldEntries(totalBlocks - 100)

	// Count how many items remain in itemsByHeight (should be ~100, the last 100 blocks)
	itemCount := 0
	c.itemsByHeight.Range(func(k, v any) bool {
		itemCount++
		return true
	})

	// Count how many hashes remain (should also be ~100, the last 100 blocks)
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

	expectedRemaining := uint64(100)
	t.Logf("After processing %d blocks with pruning up to height %d:", totalBlocks, totalBlocks-100)
	t.Logf("  itemsByHeight count: %d (expected %d)", itemCount, expectedRemaining)
	t.Logf("  hashes count: %d (expected %d)", seenHashCount, expectedRemaining)
	t.Logf("  daIncluded count: %d (expected %d)", daIncludedCount, expectedRemaining)

	// Verify pruning kept approximately the right amount (allow some margin)
	maxAllowed := int(expectedRemaining * 11 / 10) // Allow 10% margin
	minExpected := int(expectedRemaining)

	if itemCount < minExpected || itemCount > maxAllowed {
		t.Errorf("itemsByHeight not pruned correctly: got %d, want ~%d", itemCount, expectedRemaining)
	}

	if seenHashCount < minExpected || seenHashCount > maxAllowed {
		t.Errorf("hashes not pruned correctly: got %d, want ~%d", seenHashCount, expectedRemaining)
	}

	if daIncludedCount < minExpected || daIncludedCount > maxAllowed {
		t.Errorf("daIncluded not pruned correctly: got %d, want ~%d", daIncludedCount, expectedRemaining)
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
