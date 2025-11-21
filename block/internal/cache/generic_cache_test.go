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

// TestCache_MaxDAHeight verifies that daHeight tracks the maximum DA height
func TestCache_MaxDAHeight(t *testing.T) {
	c := NewCache[testItem]()

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

	// Test persistence
	dir := t.TempDir()
	if err := c.SaveToDisk(dir); err != nil {
		t.Fatalf("SaveToDisk failed: %v", err)
	}

	c2 := NewCache[testItem]()
	if err := c2.LoadFromDisk(dir); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	if got := c2.daHeight(); got != 200 {
		t.Errorf("after load: daHeight = %d, want 200", got)
	}
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
