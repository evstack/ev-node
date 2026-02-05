package cache

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/evstack/ev-node/pkg/store"
)

const (
	// DefaultItemsCacheSize is the default size for items cache.
	DefaultItemsCacheSize = 200_000

	// DefaultHashesCacheSize is the default size for hash tracking.
	DefaultHashesCacheSize = 200_000

	// DefaultDAIncludedCacheSize is the default size for DA inclusion tracking.
	DefaultDAIncludedCacheSize = 200_000
)

// Cache is a generic cache that maintains items that are seen and hard confirmed.
// Uses bounded thread-safe LRU caches to prevent unbounded memory growth.
type Cache[T any] struct {
	// itemsByHeight stores items keyed by uint64 height.
	// Mutex needed for atomic get-and-remove in getNextItem.
	itemsByHeight   *lru.Cache[uint64, *T]
	itemsByHeightMu sync.Mutex

	// hashes tracks whether a given hash has been seen
	hashes *lru.Cache[string, bool]

	// daIncluded tracks the DA inclusion height for a given hash
	daIncluded *lru.Cache[string, uint64]

	// hashByHeight tracks the hash associated with each height for pruning.
	// Mutex needed for atomic operations in deleteAllForHeight.
	hashByHeight   *lru.Cache[uint64, string]
	hashByHeightMu sync.Mutex

	// maxDAHeight tracks the maximum DA height seen
	maxDAHeight *atomic.Uint64

	// store is used for persisting DA inclusion data (optional, can be nil for ephemeral caches)
	store store.Store
	// storeKeyPrefix is the prefix used for store keys
	storeKeyPrefix string
}

// NewCache returns a new Cache struct with default sizes.
// If store and keyPrefix are provided, DA inclusion data will be persisted to the store for populating the cache on restarts.
func NewCache[T any](s store.Store, keyPrefix string) *Cache[T] {
	// LRU cache creation only fails if size <= 0, which won't happen with our defaults
	itemsCache, _ := lru.New[uint64, *T](DefaultItemsCacheSize)
	hashesCache, _ := lru.New[string, bool](DefaultHashesCacheSize)
	daIncludedCache, _ := lru.New[string, uint64](DefaultDAIncludedCacheSize)
	// hashByHeight must be at least as large as hashes cache to ensure proper pruning.
	hashByHeightCache, _ := lru.New[uint64, string](DefaultHashesCacheSize)

	return &Cache[T]{
		itemsByHeight:  itemsCache,
		hashes:         hashesCache,
		daIncluded:     daIncludedCache,
		hashByHeight:   hashByHeightCache,
		maxDAHeight:    &atomic.Uint64{},
		store:          s,
		storeKeyPrefix: keyPrefix,
	}
}

// storeKey returns the store key for a given hash.
func (c *Cache[T]) storeKey(hash string) string {
	return c.storeKeyPrefix + hash
}

// encodeDAInclusion encodes daHeight and blockHeight into a 16-byte value.
func encodeDAInclusion(daHeight, blockHeight uint64) []byte {
	value := make([]byte, 16) // 8 bytes for daHeight + 8 bytes for blockHeight
	binary.LittleEndian.PutUint64(value[0:8], daHeight)
	binary.LittleEndian.PutUint64(value[8:16], blockHeight)
	return value
}

// decodeDAInclusion decodes a 16-byte value into daHeight and blockHeight.
func decodeDAInclusion(value []byte) (daHeight, blockHeight uint64, ok bool) {
	if len(value) != 16 {
		return 0, 0, false
	}
	daHeight = binary.LittleEndian.Uint64(value[0:8])
	blockHeight = binary.LittleEndian.Uint64(value[8:16])
	return daHeight, blockHeight, true
}

// getItem returns an item from the cache by height.
// Returns nil if not found or type mismatch.
func (c *Cache[T]) getItem(height uint64) *T {
	item, ok := c.itemsByHeight.Get(height)
	if !ok {
		return nil
	}
	return item
}

// setItem sets an item in the cache by height
func (c *Cache[T]) setItem(height uint64, item *T) {
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
	seen, ok := c.hashes.Get(hash)
	if !ok {
		return false
	}
	return seen
}

// setSeen sets the hash as seen and tracks its height for pruning
func (c *Cache[T]) setSeen(hash string, height uint64) {
	c.hashes.Add(hash, true)
	c.hashByHeight.Add(height, hash)
}

// getDAIncluded returns the DA height if the hash has been DA-included, otherwise it returns 0.
func (c *Cache[T]) getDAIncluded(hash string) (uint64, bool) {
	daHeight, ok := c.daIncluded.Get(hash)
	if !ok {
		return 0, false
	}
	return daHeight, true
}

// setDAIncluded sets the hash as DA-included with the given DA height and tracks block height for pruning.
func (c *Cache[T]) setDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	c.daIncluded.Add(hash, daHeight)
	c.hashByHeight.Add(blockHeight, hash)

	// Persist to store if configured (for SIGKILL protection)
	if c.store != nil {
		_ = c.store.SetMetadata(context.Background(), c.storeKey(hash), encodeDAInclusion(daHeight, blockHeight))
	}

	// Update max DA height if necessary
	c.setMaxDAHeight(daHeight)
}

// removeDAIncluded removes the DA-included status of the hash from cache and store.
func (c *Cache[T]) removeDAIncluded(hash string) {
	c.daIncluded.Remove(hash)
	if c.store != nil {
		_ = c.store.DeleteMetadata(context.Background(), c.storeKey(hash))
	}
}

// daHeight returns the maximum DA height from all DA-included items.
// Returns 0 if no items are DA-included.
func (c *Cache[T]) daHeight() uint64 {
	return c.maxDAHeight.Load()
}

// setMaxDAHeight sets the maximum DA height if the provided value is greater
// than the current value.
func (c *Cache[T]) setMaxDAHeight(daHeight uint64) {
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

// removeSeen removes a hash from the seen cache.
func (c *Cache[T]) removeSeen(hash string) {
	c.hashes.Remove(hash)
}

// deleteAllForHeight removes all items and their associated data from the cache and store at the given height.
func (c *Cache[T]) deleteAllForHeight(height uint64) {
	c.itemsByHeight.Remove(height)

	c.hashByHeightMu.Lock()
	hash, ok := c.hashByHeight.Get(height)
	if ok {
		c.hashByHeight.Remove(height)
	}
	c.hashByHeightMu.Unlock()

	if ok {
		c.hashes.Remove(hash)
		c.removeDAIncluded(hash)
	}
}

// RestoreFromStore loads DA inclusion data from the store into the in-memory cache.
// This should be called during initialization to restore persisted state.
// It directly queries store metadata keys with the cache's prefix, avoiding iteration through all blocks.
func (c *Cache[T]) RestoreFromStore(ctx context.Context) error {
	if c.store == nil || c.storeKeyPrefix == "" {
		return nil // No store configured or no prefix, nothing to restore
	}

	// Query all metadata entries with our prefix directly
	entries, err := c.store.GetMetadataByPrefix(ctx, c.storeKeyPrefix)
	if err != nil {
		return fmt.Errorf("failed to query metadata by prefix %q: %w", c.storeKeyPrefix, err)
	}

	for _, entry := range entries {
		// Extract the hash from the key by removing the prefix
		hash := strings.TrimPrefix(entry.Key, c.storeKeyPrefix)
		if hash == entry.Key || hash == "" {
			// Prefix not found or empty hash - skip invalid entry
			continue
		}

		daHeight, blockHeight, ok := decodeDAInclusion(entry.Value)
		if !ok {
			continue // Invalid data, skip
		}

		c.daIncluded.Add(hash, daHeight)
		c.hashByHeight.Add(blockHeight, hash)

		// Update max DA height
		current := c.maxDAHeight.Load()
		if daHeight > current {
			c.maxDAHeight.Store(daHeight)
		}
	}

	return nil
}

// SaveToStore persists all current DA inclusion entries to the store.
// This can be called before shutdown to ensure all data is persisted.
func (c *Cache[T]) SaveToStore(ctx context.Context) error {
	if c.store == nil {
		return nil // No store configured
	}

	keys := c.daIncluded.Keys()
	for _, hash := range keys {
		daHeight, ok := c.daIncluded.Peek(hash)
		if !ok {
			continue
		}

		// We need to find the block height for this hash
		// Since we track hash by height, we need to iterate
		var blockHeight uint64
		heightKeys := c.hashByHeight.Keys()
		for _, h := range heightKeys {
			if storedHash, ok := c.hashByHeight.Peek(h); ok && storedHash == hash {
				blockHeight = h
				break
			}
		}

		if err := c.store.SetMetadata(ctx, c.storeKey(hash), encodeDAInclusion(daHeight, blockHeight)); err != nil {
			return fmt.Errorf("failed to save DA inclusion for hash %s: %w", hash, err)
		}
	}

	return nil
}

// ClearFromStore removes all DA inclusion entries from the store for this cache.
func (c *Cache[T]) ClearFromStore(ctx context.Context, hashes []string) error {
	if c.store == nil {
		return nil
	}

	for _, hash := range hashes {
		if err := c.store.DeleteMetadata(ctx, c.storeKey(hash)); err != nil {
			return fmt.Errorf("failed to delete DA inclusion for hash %s: %w", hash, err)
		}
	}

	return nil
}
