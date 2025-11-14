package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Cache is a generic cache that maintains items that are seen and hard confirmed
type Cache[T any] struct {
	// itemsByHeight stores items keyed by uint64 height
	itemsByHeight *sync.Map
	// hashes tracks whether a given hash has been seen
	hashes *sync.Map
	// daIncluded tracks the DA inclusion height for a given hash
	daIncluded *sync.Map
	// hashByHeight tracks the hash associated with each height for pruning
	hashByHeight *sync.Map
}

// NewCache returns a new Cache struct
func NewCache[T any]() *Cache[T] {
	return &Cache[T]{
		itemsByHeight: new(sync.Map),
		hashes:        new(sync.Map),
		daIncluded:    new(sync.Map),
		hashByHeight:  new(sync.Map),
	}
}

// getItem returns an item from the cache by height.
// Returns nil if not found or type mismatch.
func (c *Cache[T]) getItem(height uint64) *T {
	item, ok := c.itemsByHeight.Load(height)
	if !ok {
		return nil
	}
	val, ok := item.(*T)
	if !ok {
		return nil
	}
	return val
}

// setItem sets an item in the cache by height
func (c *Cache[T]) setItem(height uint64, item *T) {
	c.itemsByHeight.Store(height, item)
}

// getNextItem returns the item at the specified height and removes it from cache if found.
// Returns nil if not found.
func (c *Cache[T]) getNextItem(height uint64) *T {
	item, loaded := c.itemsByHeight.LoadAndDelete(height)
	if !loaded {
		return nil
	}
	val, ok := item.(*T)
	if !ok {
		return nil
	}
	return val
}

// isSeen returns true if the hash has been seen
func (c *Cache[T]) isSeen(hash string) bool {
	seen, ok := c.hashes.Load(hash)
	if !ok {
		return false
	}
	return seen.(bool)
}

// setSeen sets the hash as seen and tracks its height for pruning
func (c *Cache[T]) setSeen(hash string, height uint64) {
	c.hashes.Store(hash, true)
	c.hashByHeight.Store(height, hash)
}

// getDAIncluded returns the DA height if the hash has been DA-included, otherwise it returns 0.
func (c *Cache[T]) getDAIncluded(hash string) (uint64, bool) {
	daIncluded, ok := c.daIncluded.Load(hash)
	if !ok {
		return 0, false
	}
	return daIncluded.(uint64), true
}

// setDAIncluded sets the hash as DA-included with the given DA height and tracks block height for pruning
func (c *Cache[T]) setDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	c.daIncluded.Store(hash, daHeight)
	c.hashByHeight.Store(blockHeight, hash)
}

// removeDAIncluded removes the DA-included status of the hash
func (c *Cache[T]) removeDAIncluded(hash string) {
	c.daIncluded.Delete(hash)
}

// deleteAllForHeight removes all items and their associated data from the cache at the given height
func (c *Cache[T]) deleteAllForHeight(height uint64) {
	c.itemsByHeight.Delete(height)
	hash, ok := c.hashByHeight.Load(height)
	if ok {
		c.hashes.Delete(hash)
		c.hashByHeight.Delete(height)
		// c.daIncluded.Delete(hash) // we actually do not want to delete the DA-included status here
	}
}

const (
	itemsByHeightFilename = "items_by_height.gob"
	hashesFilename        = "hashes.gob"
	daIncludedFilename    = "da_included.gob"
)

// saveMapGob saves a map to a file using gob encoding.
func saveMapGob[K comparable, V any](filePath string, data map[K]V) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode to file %s: %w", filePath, err)
	}
	return nil
}

// loadMapGob loads a map from a file using gob encoding.
// if the file does not exist, it returns an empty map and no error.
func loadMapGob[K comparable, V any](filePath string) (map[K]V, error) {
	m := make(map[K]V)
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil // return empty map if file not found
		}
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&m); err != nil {
		return nil, fmt.Errorf("failed to decode file %s: %w", filePath, err)
	}
	return m, nil
}

// SaveToDisk saves the cache contents to disk in the specified folder.
// It's the caller's responsibility to ensure that type T (and any types it contains)
// are registered with the gob package if necessary (e.g., using gob.Register).
func (c *Cache[T]) SaveToDisk(folderPath string) error {
	if err := os.MkdirAll(folderPath, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", folderPath, err)
	}

	// prepare items maps
	itemsByHeightMap := make(map[uint64]*T)

	c.itemsByHeight.Range(func(k, v any) bool {
		if hk, ok := k.(uint64); ok {
			if it, ok := v.(*T); ok {
				itemsByHeightMap[hk] = it
			}
		}
		return true
	})

	if err := saveMapGob(filepath.Join(folderPath, itemsByHeightFilename), itemsByHeightMap); err != nil {
		return err
	}

	// prepare hashes map
	hashesToSave := make(map[string]bool)
	c.hashes.Range(func(k, v any) bool {
		keyStr, okKey := k.(string)
		valBool, okVal := v.(bool)
		if okKey && okVal {
			hashesToSave[keyStr] = valBool
		}
		return true
	})
	if err := saveMapGob(filepath.Join(folderPath, hashesFilename), hashesToSave); err != nil {
		return err
	}

	// prepare daIncluded map
	daIncludedToSave := make(map[string]uint64)
	c.daIncluded.Range(func(k, v any) bool {
		keyStr, okKey := k.(string)
		valUint64, okVal := v.(uint64)
		if okKey && okVal {
			daIncludedToSave[keyStr] = valUint64
		}
		return true
	})
	return saveMapGob(filepath.Join(folderPath, daIncludedFilename), daIncludedToSave)
}

// LoadFromDisk loads the cache contents from disk from the specified folder.
// It populates the current cache instance. If files are missing, corresponding parts of the cache will be empty.
// It's the caller's responsibility to ensure that type T (and any types it contains)
// are registered with the gob package if necessary (e.g., using gob.Register).
func (c *Cache[T]) LoadFromDisk(folderPath string) error {
	// load items by height
	itemsByHeightMap, err := loadMapGob[uint64, *T](filepath.Join(folderPath, itemsByHeightFilename))
	if err != nil {
		return fmt.Errorf("failed to load items by height: %w", err)
	}
	for k, v := range itemsByHeightMap {
		c.itemsByHeight.Store(k, v)
	}

	// load hashes
	hashesMap, err := loadMapGob[string, bool](filepath.Join(folderPath, hashesFilename))
	if err != nil {
		return fmt.Errorf("failed to load hashes: %w", err)
	}
	for k, v := range hashesMap {
		c.hashes.Store(k, v)
	}

	// load daIncluded
	daIncludedMap, err := loadMapGob[string, uint64](filepath.Join(folderPath, daIncludedFilename))
	if err != nil {
		return fmt.Errorf("failed to load daIncluded: %w", err)
	}
	for k, v := range daIncludedMap {
		c.daIncluded.Store(k, v)
	}

	return nil
}
