package cache

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sync/errgroup"
)

const (
	// DefaultItemsCacheSize is the default size for items cache.
	// Each item (SignedHeader/Data) is ~1-2KB, so 2M entries ≈ 2-4GB max.
	DefaultItemsCacheSize = 2_000_000

	// DefaultHashesCacheSize is the default size for hash tracking.
	// Each hash entry is ~64 bytes, so 4M entries ≈ 256MB max.
	DefaultHashesCacheSize = 4_000_000

	// DefaultDAIncludedCacheSize is the default size for DA inclusion tracking.
	// Each entry is ~72 bytes, so 4M entries ≈ 288MB max.
	DefaultDAIncludedCacheSize = 4_000_000
)

// Cache is a generic cache that maintains items that are seen and hard confirmed.
// Uses bounded LRU caches to prevent unbounded memory growth.
type Cache[T any] struct {
	// itemsByHeight stores items keyed by uint64 height
	itemsByHeight   *lru.Cache[uint64, *T]
	itemsByHeightMu sync.RWMutex

	// hashes tracks whether a given hash has been seen
	hashes   *lru.Cache[string, bool]
	hashesMu sync.RWMutex

	// daIncluded tracks the DA inclusion height for a given hash
	daIncluded   *lru.Cache[string, uint64]
	daIncludedMu sync.RWMutex

	// hashByHeight tracks the hash associated with each height for pruning
	hashByHeight   *lru.Cache[uint64, string]
	hashByHeightMu sync.RWMutex

	// maxDAHeight tracks the maximum DA height seen
	maxDAHeight *atomic.Uint64
}

// CacheConfig holds configuration for cache sizes.
type CacheConfig struct {
	ItemsCacheSize      int
	HashesCacheSize     int
	DAIncludedCacheSize int
}

// DefaultCacheConfig returns the default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		ItemsCacheSize:      DefaultItemsCacheSize,
		HashesCacheSize:     DefaultHashesCacheSize,
		DAIncludedCacheSize: DefaultDAIncludedCacheSize,
	}
}

// NewCache returns a new Cache struct with default sizes
func NewCache[T any]() *Cache[T] {
	cache, _ := NewCacheWithConfig[T](DefaultCacheConfig())
	return cache
}

// NewCacheWithConfig returns a new Cache struct with custom sizes
func NewCacheWithConfig[T any](config CacheConfig) (*Cache[T], error) {
	itemsCache, err := lru.New[uint64, *T](config.ItemsCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create items cache: %w", err)
	}

	hashesCache, err := lru.New[string, bool](config.HashesCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create hashes cache: %w", err)
	}

	daIncludedCache, err := lru.New[string, uint64](config.DAIncludedCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create daIncluded cache: %w", err)
	}

	// hashByHeight must be at least as large as hashes cache to ensure proper pruning.
	// If an entry is evicted from hashByHeight before hashes, the corresponding hash
	// entry can no longer be pruned by height, causing a slow memory leak.
	hashByHeightCache, err := lru.New[uint64, string](config.HashesCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create hashByHeight cache: %w", err)
	}

	return &Cache[T]{
		itemsByHeight: itemsCache,
		hashes:        hashesCache,
		daIncluded:    daIncludedCache,
		hashByHeight:  hashByHeightCache,
		maxDAHeight:   &atomic.Uint64{},
	}, nil
}

// getItem returns an item from the cache by height.
// Returns nil if not found or type mismatch.
func (c *Cache[T]) getItem(height uint64) *T {
	c.itemsByHeightMu.RLock()
	defer c.itemsByHeightMu.RUnlock()

	item, ok := c.itemsByHeight.Get(height)
	if !ok {
		return nil
	}
	return item
}

// setItem sets an item in the cache by height
func (c *Cache[T]) setItem(height uint64, item *T) {
	c.itemsByHeightMu.Lock()
	defer c.itemsByHeightMu.Unlock()

	c.itemsByHeight.Add(height, item)
}

// getNextItem returns the item at the specified height and removes it from cache if found.
// Returns nil if not found.
func (c *Cache[T]) getNextItem(height uint64) *T {
	c.itemsByHeightMu.Lock()
	defer c.itemsByHeightMu.Unlock()

	item, ok := c.itemsByHeight.Get(height)
	if !ok {
		return nil
	}
	c.itemsByHeight.Remove(height)
	return item
}

// isSeen returns true if the hash has been seen
func (c *Cache[T]) isSeen(hash string) bool {
	c.hashesMu.RLock()
	defer c.hashesMu.RUnlock()

	seen, ok := c.hashes.Get(hash)
	if !ok {
		return false
	}
	return seen
}

// setSeen sets the hash as seen and tracks its height for pruning
func (c *Cache[T]) setSeen(hash string, height uint64) {
	c.hashesMu.Lock()
	c.hashes.Add(hash, true)
	c.hashesMu.Unlock()

	c.hashByHeightMu.Lock()
	c.hashByHeight.Add(height, hash)
	c.hashByHeightMu.Unlock()
}

// getDAIncluded returns the DA height if the hash has been DA-included, otherwise it returns 0.
func (c *Cache[T]) getDAIncluded(hash string) (uint64, bool) {
	c.daIncludedMu.RLock()
	defer c.daIncludedMu.RUnlock()

	daHeight, ok := c.daIncluded.Get(hash)
	if !ok {
		return 0, false
	}
	return daHeight, true
}

// setDAIncluded sets the hash as DA-included with the given DA height and tracks block height for pruning
func (c *Cache[T]) setDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	c.daIncludedMu.Lock()
	c.daIncluded.Add(hash, daHeight)
	c.daIncludedMu.Unlock()

	c.hashByHeightMu.Lock()
	c.hashByHeight.Add(blockHeight, hash)
	c.hashByHeightMu.Unlock()

	// Update max DA height if necessary
	for range 1_000 {
		current := c.maxDAHeight.Load()
		if daHeight <= current {
			return
		}
		if c.maxDAHeight.CompareAndSwap(current, daHeight) {
			return
		}
	}
}

// removeDAIncluded removes the DA-included status of the hash
func (c *Cache[T]) removeDAIncluded(hash string) {
	c.daIncludedMu.Lock()
	defer c.daIncludedMu.Unlock()

	c.daIncluded.Remove(hash)
}

// daHeight returns the maximum DA height from all DA-included items.
// Returns 0 if no items are DA-included.
func (c *Cache[T]) daHeight() uint64 {
	return c.maxDAHeight.Load()
}

// removeSeen removes a hash from the seen cache.
func (c *Cache[T]) removeSeen(hash string) {
	c.hashesMu.Lock()
	defer c.hashesMu.Unlock()
	c.hashes.Remove(hash)
}

// forEachHash iterates over all hashes in the seen cache and calls the provided function.
// If the function returns false, iteration stops.
func (c *Cache[T]) forEachHash(fn func(hash string) bool) {
	c.hashesMu.RLock()
	keys := c.hashes.Keys()
	c.hashesMu.RUnlock()

	for _, hash := range keys {
		if !fn(hash) {
			break
		}
	}
}

// deleteAllForHeight removes all items and their associated data from the cache at the given height
func (c *Cache[T]) deleteAllForHeight(height uint64) {
	c.itemsByHeightMu.Lock()
	c.itemsByHeight.Remove(height)
	c.itemsByHeightMu.Unlock()

	c.hashByHeightMu.Lock()
	hash, ok := c.hashByHeight.Get(height)
	if ok {
		c.hashByHeight.Remove(height)
	}
	c.hashByHeightMu.Unlock()

	if ok {
		c.hashesMu.Lock()
		c.hashes.Remove(hash)
		c.hashesMu.Unlock()
		// c.daIncluded.Remove(hash) // we actually do not want to delete the DA-included status here
	}
}

const (
	itemsByHeightFilename = "items_by_height.gob"
	hashesFilename        = "hashes.gob"
	daIncludedFilename    = "da_included.gob"
)

// saveMapGob saves a map to a file using gob encoding.
func saveMapGob[K comparable, V any](filePath string, data map[K]V) (err error) {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	writer := bufio.NewWriter(file)

	defer func() {
		err = errors.Join(err, writer.Flush(), file.Sync(), file.Close())
	}()
	if err := gob.NewEncoder(writer).Encode(data); err != nil {
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
	var wg errgroup.Group
	// prepare items maps
	wg.Go(func() error {
		itemsByHeightMap := make(map[uint64]*T)

		c.itemsByHeightMu.RLock()
		keys := c.itemsByHeight.Keys()
		for _, k := range keys {
			if v, ok := c.itemsByHeight.Peek(k); ok {
				itemsByHeightMap[k] = v
			}
		}
		c.itemsByHeightMu.RUnlock()

		if err := saveMapGob(filepath.Join(folderPath, itemsByHeightFilename), itemsByHeightMap); err != nil {
			return fmt.Errorf("save %s: %w", itemsByHeightFilename, err)
		}
		return nil
	})

	// prepare hashes map
	wg.Go(func() error {
		hashesToSave := make(map[string]bool)

		c.hashesMu.RLock()
		keys := c.hashes.Keys()
		for _, k := range keys {
			if v, ok := c.hashes.Peek(k); ok {
				hashesToSave[k] = v
			}
		}
		c.hashesMu.RUnlock()

		if err := saveMapGob(filepath.Join(folderPath, hashesFilename), hashesToSave); err != nil {
			return fmt.Errorf("save %s: %w", hashesFilename, err)
		}
		return nil
	})

	// prepare daIncluded map
	wg.Go(func() error {
		daIncludedToSave := make(map[string]uint64)

		c.daIncludedMu.RLock()
		keys := c.daIncluded.Keys()
		for _, k := range keys {
			if v, ok := c.daIncluded.Peek(k); ok {
				daIncludedToSave[k] = v
			}
		}
		c.daIncludedMu.RUnlock()

		if err := saveMapGob(filepath.Join(folderPath, daIncludedFilename), daIncludedToSave); err != nil {
			return fmt.Errorf("save %s: %w", daIncludedFilename, err)
		}
		return nil
	})
	return wg.Wait()
}

// LoadFromDisk loads the cache contents from disk from the specified folder.
// It populates the current cache instance. If files are missing, corresponding parts of the cache will be empty.
// It's the caller's responsibility to ensure that type T (and any types it contains)
// are registered with the gob package if necessary (e.g., using gob.Register).
func (c *Cache[T]) LoadFromDisk(folderPath string) error {
	var wg errgroup.Group
	// load items by height
	wg.Go(func() error {
		itemsByHeightMap, err := loadMapGob[uint64, *T](filepath.Join(folderPath, itemsByHeightFilename))
		if err != nil {
			return fmt.Errorf("failed to load %s : %w", itemsByHeightFilename, err)
		}
		c.itemsByHeightMu.Lock()
		for k, v := range itemsByHeightMap {
			c.itemsByHeight.Add(k, v)
		}
		c.itemsByHeightMu.Unlock()
		return nil
	})
	// load hashes
	wg.Go(func() error {
		hashesMap, err := loadMapGob[string, bool](filepath.Join(folderPath, hashesFilename))
		if err != nil {
			return fmt.Errorf("failed to load %s : %w", hashesFilename, err)
		}
		c.hashesMu.Lock()
		for k, v := range hashesMap {
			c.hashes.Add(k, v)
		}
		c.hashesMu.Unlock()
		return nil
	})
	// load daIncluded
	wg.Go(func() error {
		daIncludedMap, err := loadMapGob[string, uint64](filepath.Join(folderPath, daIncludedFilename))
		if err != nil {
			return fmt.Errorf("failed to load %s : %w", daIncludedFilename, err)
		}
		c.daIncludedMu.Lock()
		for k, v := range daIncludedMap {
			c.daIncluded.Add(k, v)
			// Update max DA height during load
			current := c.maxDAHeight.Load()
			if v > current {
				c.maxDAHeight.Store(v)
			}
		}
		c.daIncludedMu.Unlock()
		return nil
	})
	return wg.Wait()
}
