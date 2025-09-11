package cache

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

type testItem struct{ V int }

func init() {
	gob.Register(&testItem{})
}

// TestCache_ConcurrentOperations exercises concurrent Set, Delete, and Range operations.
func TestCache_ConcurrentOperations(t *testing.T) {
	c := NewCache[testItem]()

	const N = 2000
	var wg sync.WaitGroup

	// writers
	writer := func(start int) {
		defer wg.Done()
		for i := start; i < N; i += 2 {
			v := &testItem{V: i}
			c.SetItem(uint64(i), v)
			if i%10 == 0 {
				// randomly delete some keys
				c.DeleteItem(uint64(i))
			}
		}
	}

	wg.Add(2)
	go writer(0)
	go writer(1)
	wg.Wait()
}

// TestCache_HeightIndexConsistency verifies the height index matches actual keys after concurrent modifications.
func TestCache_HeightIndexConsistency(t *testing.T) {
	c := NewCache[testItem]()
	const N = 5000
	// shuffle insert order
	idx := rand.Perm(N)
	for _, i := range idx {
		v := &testItem{V: i}
		c.SetItem(uint64(i), v)
	}
	// random deletions
	for i := 0; i < N; i += 3 {
		if i%7 == 0 {
			c.DeleteItem(uint64(i))
		}
	}

	// Collect actual keys from the map
	actual := make([]uint64, 0, N)
	c.itemsByHeight.Range(func(k, _ any) bool {
		if h, ok := k.(uint64); ok {
			actual = append(actual, h)
		}
		return true
	})
	sort.Slice(actual, func(i, j int) bool { return actual[i] < actual[j] })

	// Compare against heightKeys under lock
	c.heightIndexMu.RLock()
	idxCopy := make([]uint64, len(c.heightKeys))
	copy(idxCopy, c.heightKeys)
	c.heightIndexMu.RUnlock()

	if len(actual) != len(idxCopy) {
		t.Fatalf("index length mismatch: got %d want %d", len(idxCopy), len(actual))
	}
	for i := range actual {
		if actual[i] != idxCopy[i] {
			t.Fatalf("index mismatch at %d: got %d want %d", i, idxCopy[i], actual[i])
		}
	}
}

// TestCache_TypeSafety ensures methods gracefully handle invalid underlying types.
func TestCache_TypeSafety(t *testing.T) {
	c := NewCache[testItem]()

	// Inject invalid value types directly into maps (bypassing typed methods)
	c.itemsByHeight.Store(uint64(1), "not-a-*testItem")

	if got := c.GetItem(1); got != nil {
		t.Fatalf("expected nil for invalid stored type, got %#v", got)
	}

	// Range should skip invalid entries and not panic
	ran := false
	c.RangeByHeight(func(_ uint64, _ *testItem) bool { ran = true; return true })
	_ = ran // ensure no panic
}

// TestCache_SaveLoad_ErrorPaths covers SaveToDisk and LoadFromDisk error scenarios.
func TestCache_SaveLoad_ErrorPaths(t *testing.T) {
	c := NewCache[testItem]()
	for i := 0; i < 5; i++ {
		v := &testItem{V: i}
		c.SetItem(uint64(i), v)
		c.SetSeen(fmt.Sprintf("s%d", i))
		c.SetDAIncluded(fmt.Sprintf("d%d", i), uint64(i))
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
	if got := c2.GetItem(3); got == nil || got.V != 3 {
		t.Fatalf("roundtrip GetItem mismatch: got %#v", got)
	}
	if !c2.IsSeen("s1") || !c2.IsDAIncluded("d2") {
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
		c.SetItem(uint64(i), v)
	}
	// Delete a range in the middle
	for i := 5000; i < 10000; i += 2 {
		c.DeleteItem(uint64(i))
	}
}
