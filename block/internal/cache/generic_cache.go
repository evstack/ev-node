package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Cache is a generic cache that maintains items that are seen and hard confirmed
type Cache[T any] struct {
	// itemsByHeight stores items keyed by uint64 height
	itemsByHeight *sync.Map
	// itemsByHash stores items keyed by string hash (optional)
	itemsByHash *sync.Map
	// hashes tracks whether a given hash has been seen
	hashes *sync.Map
	// daIncluded tracks the DA inclusion height for a given hash
	daIncluded *sync.Map

	// ordered index of heights for iteration
	heightIndexMu sync.RWMutex
	heightKeys    []uint64 // kept in ascending order, unique
}

// NewCache returns a new Cache struct
func NewCache[T any]() *Cache[T] {
	return &Cache[T]{
		itemsByHeight: new(sync.Map),
		itemsByHash:   new(sync.Map),
		hashes:        new(sync.Map),
		daIncluded:    new(sync.Map),
	}
}

// GetItem returns an item from the cache by height.
// Returns nil if not found or type mismatch.
func (c *Cache[T]) GetItem(height uint64) *T {
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

// SetItem sets an item in the cache by height
func (c *Cache[T]) SetItem(height uint64, item *T) {
	// check if key already exists to avoid duplicate index entries
	if _, ok := c.itemsByHeight.Load(height); !ok {
		c.insertHeight(height)
	}
	c.itemsByHeight.Store(height, item)
}

// DeleteItem deletes an item from the cache by height
func (c *Cache[T]) DeleteItem(height uint64) {
	c.itemsByHeight.Delete(height)
	c.deleteHeight(height)
}

// RangeByHeight iterates over items keyed by height in an unspecified order and calls fn for each.
// If fn returns false, iteration stops early.
func (c *Cache[T]) RangeByHeight(fn func(height uint64, item *T) bool) {
	c.itemsByHeight.Range(func(k, v any) bool {
		height, ok := k.(uint64)
		if !ok {
			return true
		}
		it, ok := v.(*T)
		if !ok {
			return true
		}
		return fn(height, it)
	})
}

// RangeByHeightAsc iterates items by ascending height order.
func (c *Cache[T]) RangeByHeightAsc(fn func(height uint64, item *T) bool) {
	keys := c.snapshotHeightsAsc()
	for _, h := range keys {
		if v, ok := c.itemsByHeight.Load(h); ok {
			it, ok := v.(*T)
			if !ok {
				continue
			}
			if !fn(h, it) {
				return
			}
		}
	}
}

// RangeByHeightDesc iterates items by descending height order.
func (c *Cache[T]) RangeByHeightDesc(fn func(height uint64, item *T) bool) {
	keys := c.snapshotHeightsDesc()
	for _, h := range keys {
		if v, ok := c.itemsByHeight.Load(h); ok {
			it, ok := v.(*T)
			if !ok {
				continue
			}
			if !fn(h, it) {
				return
			}
		}
	}
}

// GetItemByHash returns an item from the cache by string hash key.
// Returns nil if not found or type mismatch.
func (c *Cache[T]) GetItemByHash(hash string) *T {
	item, ok := c.itemsByHash.Load(hash)
	if !ok {
		return nil
	}
	val, ok := item.(*T)
	if !ok {
		return nil
	}
	return val
}

// SetItemByHash sets an item in the cache by string hash key
func (c *Cache[T]) SetItemByHash(hash string, item *T) {
	c.itemsByHash.Store(hash, item)
}

// DeleteItemByHash deletes an item in the cache by string hash key
func (c *Cache[T]) DeleteItemByHash(hash string) {
	c.itemsByHash.Delete(hash)
}

// RangeByHash iterates over all items keyed by string hash and calls fn for each.
// If fn returns false, iteration stops early.
func (c *Cache[T]) RangeByHash(fn func(hash string, item *T) bool) {
	c.itemsByHash.Range(func(k, v any) bool {
		key, ok := k.(string)
		if !ok {
			return true
		}
		it, ok := v.(*T)
		if !ok {
			return true
		}
		return fn(key, it)
	})
}

// IsSeen returns true if the hash has been seen
func (c *Cache[T]) IsSeen(hash string) bool {
	seen, ok := c.hashes.Load(hash)
	if !ok {
		return false
	}
	return seen.(bool)
}

// SetSeen sets the hash as seen
func (c *Cache[T]) SetSeen(hash string) {
	c.hashes.Store(hash, true)
}

// IsDAIncluded returns true if the hash has been DA-included
func (c *Cache[T]) IsDAIncluded(hash string) bool {
	_, ok := c.daIncluded.Load(hash)
	return ok
}

// GetDAIncludedHeight returns the DA height at which the hash was DA included
func (c *Cache[T]) GetDAIncludedHeight(hash string) (uint64, bool) {
	daIncluded, ok := c.daIncluded.Load(hash)
	if !ok {
		return 0, false
	}
	return daIncluded.(uint64), true
}

// SetDAIncluded sets the hash as DA-included with the given DA height
func (c *Cache[T]) SetDAIncluded(hash string, daHeight uint64) {
	c.daIncluded.Store(hash, daHeight)
}

const (
	itemsByHeightFilename = "items_by_height.gob"
	itemsByHashFilename   = "items_by_hash.gob"
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
	itemsByHashMap := make(map[string]*T)

	c.itemsByHeight.Range(func(k, v any) bool {
		if hk, ok := k.(uint64); ok {
			if it, ok := v.(*T); ok {
				itemsByHeightMap[hk] = it
			}
		}
		return true
	})
	c.itemsByHash.Range(func(k, v any) bool {
		if hk, ok := k.(string); ok {
			if it, ok := v.(*T); ok {
				itemsByHashMap[hk] = it
			}
		}
		return true
	})

	if err := saveMapGob(filepath.Join(folderPath, itemsByHeightFilename), itemsByHeightMap); err != nil {
		return err
	}
	if err := saveMapGob(filepath.Join(folderPath, itemsByHashFilename), itemsByHashMap); err != nil {
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
		c.insertHeight(k)
	}

	// load items by hash
	itemsByHashMap, err := loadMapGob[string, *T](filepath.Join(folderPath, itemsByHashFilename))
	if err != nil {
		return fmt.Errorf("failed to load items by hash: %w", err)
	}
	for k, v := range itemsByHashMap {
		c.itemsByHash.Store(k, v)
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

// insertHeight inserts h into heightKeys, keeping ascending order and uniqueness.
func (c *Cache[T]) insertHeight(h uint64) {
	c.heightIndexMu.Lock()
	defer c.heightIndexMu.Unlock()
	i := sort.Search(len(c.heightKeys), func(i int) bool { return c.heightKeys[i] >= h })
	if i < len(c.heightKeys) && c.heightKeys[i] == h {
		return
	}
	c.heightKeys = append(c.heightKeys, 0)
	copy(c.heightKeys[i+1:], c.heightKeys[i:])
	c.heightKeys[i] = h
}

// deleteHeight removes h if present.
func (c *Cache[T]) deleteHeight(h uint64) {
	c.heightIndexMu.Lock()
	defer c.heightIndexMu.Unlock()
	i := sort.Search(len(c.heightKeys), func(i int) bool { return c.heightKeys[i] >= h })
	if i < len(c.heightKeys) && c.heightKeys[i] == h {
		copy(c.heightKeys[i:], c.heightKeys[i+1:])
		c.heightKeys = c.heightKeys[:len(c.heightKeys)-1]
	}
}

func (c *Cache[T]) snapshotHeightsAsc() []uint64 {
	c.heightIndexMu.RLock()
	defer c.heightIndexMu.RUnlock()
	out := make([]uint64, len(c.heightKeys))
	copy(out, c.heightKeys)
	return out
}

func (c *Cache[T]) snapshotHeightsDesc() []uint64 {
	c.heightIndexMu.RLock()
	defer c.heightIndexMu.RUnlock()
	n := len(c.heightKeys)
	out := make([]uint64, n)
	for i := 0; i < n; i++ {
		out[i] = c.heightKeys[n-1-i]
	}
	return out
}
