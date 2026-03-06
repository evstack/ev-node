package cache

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"
	ds "github.com/ipfs/go-datastore"

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

// snapshotEntry is one record in the persisted snapshot.
// Encoded as 16 bytes: [blockHeight uint64 LE][daHeight uint64 LE].
type snapshotEntry struct {
	blockHeight uint64
	daHeight    uint64
}

const snapshotEntrySize = 16 // bytes per snapshotEntry

// Cache tracks seen blocks and DA inclusion status using bounded LRU caches.
type Cache[T any] struct {
	// itemsByHeight stores items keyed by uint64 height.
	// Mutex needed for atomic get-and-remove in getNextItem.
	itemsByHeight   *lru.Cache[uint64, *T]
	itemsByHeightMu sync.Mutex

	// hashes tracks whether a given hash has been seen
	hashes *lru.Cache[string, bool]

	// daIncluded maps hash → daHeight. Hash may be a real content hash or a
	// height placeholder (see HeightPlaceholderKey) immediately after restore.
	daIncluded *lru.Cache[string, uint64]

	// hashByHeight maps blockHeight → hash, used for pruning and height-based
	// lookups. Protected by hashByHeightMu only in deleteAllForHeight where a
	// read-then-remove must be atomic.
	hashByHeight   *lru.Cache[uint64, string]
	hashByHeightMu sync.Mutex

	// maxDAHeight tracks the maximum DA height seen
	maxDAHeight *atomic.Uint64

	store store.Store // nil = ephemeral, no persistence
	// storeKeyPrefix is the prefix used for store keys
	storeKeyPrefix string
}

func (c *Cache[T]) snapshotKey() string {
	return c.storeKeyPrefix + "__snap"
}

// NewCache creates a Cache. When store and keyPrefix are set, mutations
// persist a snapshot so RestoreFromStore can recover in-flight state.
func NewCache[T any](s store.Store, keyPrefix string) *Cache[T] {
	// LRU cache creation only fails if size <= 0, which won't happen with our defaults
	itemsCache, _ := lru.New[uint64, *T](DefaultItemsCacheSize)
	hashesCache, _ := lru.New[string, bool](DefaultHashesCacheSize)
	daIncludedCache, _ := lru.New[string, uint64](DefaultDAIncludedCacheSize)
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

// getItem returns an item from the cache by height.
func (c *Cache[T]) getItem(height uint64) *T {
	item, ok := c.itemsByHeight.Get(height)
	if !ok {
		return nil
	}
	return item
}

// setItem sets an item in the cache by height.
func (c *Cache[T]) setItem(height uint64, item *T) {
	c.itemsByHeight.Add(height, item)
}

// getNextItem returns and removes the item at height, or nil if absent.
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

// isSeen returns true if the hash has been seen.
func (c *Cache[T]) isSeen(hash string) bool {
	seen, ok := c.hashes.Get(hash)
	return ok && seen
}

// setSeen sets the hash as seen and tracks its height for pruning.
func (c *Cache[T]) setSeen(hash string, height uint64) {
	c.hashes.Add(hash, true)
	c.hashByHeight.Add(height, hash)
}

// getDAIncluded returns the DA height if the hash has been DA-included.
func (c *Cache[T]) getDAIncluded(hash string) (uint64, bool) {
	return c.daIncluded.Get(hash)
}

// getDAIncludedByHeight resolves DA height via the height→hash index.
// Works for both real hashes (steady state) and snapshot placeholders
// (post-restart, before the DA retriever re-fires the real hash).
func (c *Cache[T]) getDAIncludedByHeight(blockHeight uint64) (uint64, bool) {
	hash, ok := c.hashByHeight.Get(blockHeight)
	if !ok {
		return 0, false
	}
	return c.getDAIncluded(hash)
}

// setDAIncluded records DA inclusion and persists the snapshot.
// If a previous entry already exists at blockHeight (e.g. a placeholder from
// RestoreFromStore), it is evicted from daIncluded to avoid orphan leaks.
func (c *Cache[T]) setDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	if prev, ok := c.hashByHeight.Get(blockHeight); ok && prev != hash {
		c.daIncluded.Remove(prev)
	}
	c.daIncluded.Add(hash, daHeight)
	c.hashByHeight.Add(blockHeight, hash)
	c.setMaxDAHeight(daHeight)
	_ = c.persistSnapshot(context.Background())
}

// removeDAIncluded removes the DA-included status of the hash from the cache
// and rewrites the window snapshot.
func (c *Cache[T]) removeDAIncluded(hash string) {
	c.daIncluded.Remove(hash)
	_ = c.persistSnapshot(context.Background())
}

// daHeight returns the maximum DA height from all DA-included items.
func (c *Cache[T]) daHeight() uint64 {
	return c.maxDAHeight.Load()
}

// setMaxDAHeight sets the maximum DA height if the provided value is greater.
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

// deleteAllForHeight removes all items and their associated data from the
// cache at the given height and rewrites the window snapshot.
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
		// Remove from daIncluded without triggering a snapshot write — we will
		// write the snapshot once below after the removal is applied.
		c.daIncluded.Remove(hash)
	}

	_ = c.persistSnapshot(context.Background())
}

// persistSnapshot writes all current in-flight [blockHeight, daHeight] pairs
// to the store under a single key. Called on every mutation; payload is tiny
// (typically <10 entries × 16 bytes).
func (c *Cache[T]) persistSnapshot(ctx context.Context) error {
	if c.store == nil || c.storeKeyPrefix == "" {
		return nil
	}

	heights := c.hashByHeight.Keys()
	entries := make([]snapshotEntry, 0, len(heights))
	for _, h := range heights {
		hash, ok := c.hashByHeight.Peek(h)
		if !ok {
			continue
		}
		daH, ok := c.daIncluded.Peek(hash)
		if !ok {
			continue
		}
		entries = append(entries, snapshotEntry{blockHeight: h, daHeight: daH})
	}

	return c.store.SetMetadata(ctx, c.snapshotKey(), encodeSnapshot(entries))
}

// encodeSnapshot serialises a slice of snapshotEntry values into a byte slice.
func encodeSnapshot(entries []snapshotEntry) []byte {
	buf := make([]byte, len(entries)*snapshotEntrySize)
	for i, e := range entries {
		off := i * snapshotEntrySize
		binary.LittleEndian.PutUint64(buf[off:], e.blockHeight)
		binary.LittleEndian.PutUint64(buf[off+8:], e.daHeight)
	}
	return buf
}

// decodeSnapshot deserialises a byte slice produced by encodeSnapshot.
// Returns nil if the slice is empty or not a multiple of snapshotEntrySize.
func decodeSnapshot(buf []byte) []snapshotEntry {
	if len(buf) == 0 || len(buf)%snapshotEntrySize != 0 {
		return nil
	}
	entries := make([]snapshotEntry, len(buf)/snapshotEntrySize)
	for i := range entries {
		off := i * snapshotEntrySize
		entries[i].blockHeight = binary.LittleEndian.Uint64(buf[off:])
		entries[i].daHeight = binary.LittleEndian.Uint64(buf[off+8:])
	}
	return entries
}

// RestoreFromStore loads the in-flight snapshot with a single store read.
// Each entry is installed as a height placeholder; real hashes replace them
// once the DA retriever re-fires SetHeaderDAIncluded after startup.
// Missing snapshot key is treated as a no-op (fresh node or pre-snapshot version).
func (c *Cache[T]) RestoreFromStore(ctx context.Context) error {
	if c.store == nil || c.storeKeyPrefix == "" {
		return nil
	}

	buf, err := c.store.GetMetadata(ctx, c.snapshotKey())
	if err != nil {
		if errors.Is(err, ds.ErrNotFound) {
			return nil // key absent = fresh node or pre-snapshot version, nothing to restore
		}
		return fmt.Errorf("reading cache snapshot from store: %w", err)
	}

	for _, e := range decodeSnapshot(buf) {
		placeholder := HeightPlaceholderKey(c.storeKeyPrefix, e.blockHeight)
		c.daIncluded.Add(placeholder, e.daHeight)
		c.hashByHeight.Add(e.blockHeight, placeholder)
		c.setMaxDAHeight(e.daHeight)
	}

	return nil
}

// HeightPlaceholderKey returns a store key for a height-indexed DA inclusion
// entry used when the real content hash is unavailable (e.g. after restore).
// Format: "<prefix>__h/<height_hex_16>" — cannot collide with real 64-char hashes.
func HeightPlaceholderKey(prefix string, height uint64) string {
	const hexDigits = "0123456789abcdef"
	buf := make([]byte, len(prefix)+4+16)
	n := copy(buf, prefix)
	n += copy(buf[n:], "__h/")
	for i := 15; i >= 0; i-- {
		buf[n+i] = hexDigits[height&0xf]
		height >>= 4
	}
	return string(buf)
}

// SaveToStore flushes the current snapshot to the store.
func (c *Cache[T]) SaveToStore(ctx context.Context) error {
	if c.store == nil {
		return nil
	}
	if err := c.persistSnapshot(ctx); err != nil {
		return fmt.Errorf("saving cache snapshot: %w", err)
	}
	return nil
}

// ClearFromStore deletes the snapshot key from the store.
func (c *Cache[T]) ClearFromStore(ctx context.Context) error {
	if c.store == nil {
		return nil
	}
	if err := c.store.DeleteMetadata(ctx, c.snapshotKey()); err != nil {
		return fmt.Errorf("clearing cache snapshot: %w", err)
	}
	return nil
}
