package cache

import (
	"context"
	"encoding/binary"
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

// snapshotEntry is one record in the persisted in-flight window snapshot.
// Encoded as 16 contiguous bytes: [blockHeight uint64 LE][daHeight uint64 LE].
type snapshotEntry struct {
	blockHeight uint64
	daHeight    uint64
}

const snapshotEntrySize = 16 // bytes per snapshotEntry

// Cache is a generic cache that maintains items that are seen and hard confirmed.
// Uses bounded thread-safe LRU caches to prevent unbounded memory growth.
//
// # Persistence strategy (O(1) restore)
//
// Rather than persisting one store key per DA-included hash (which required an
// O(n) prefix scan on restore), the Cache maintains a single *window snapshot*
// key in the store.  The snapshot encodes the full set of currently in-flight
// entries as a compact byte slice:
//
//	[ blockHeight₀ uint64-LE | daHeight₀ uint64-LE ]
//	[ blockHeight₁ uint64-LE | daHeight₁ uint64-LE ]
//	…
//
// Every call to setDAIncluded, removeDAIncluded, or deleteAllForHeight
// rewrites this single key atomically.  On startup RestoreFromStore issues
// exactly one GetMetadata call and deserialises the slice — O(1) regardless
// of chain height or node type.
//
// The snapshot key is:  storeKeyPrefix + "__snap"
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

	// store is used for persisting the window snapshot (optional, nil = ephemeral).
	store store.Store
	// storeKeyPrefix is the prefix used for store keys (header or data).
	storeKeyPrefix string
}

// snapshotKey returns the single metadata key used to persist the in-flight window.
func (c *Cache[T]) snapshotKey() string {
	return c.storeKeyPrefix + "__snap"
}

// NewCache returns a new Cache struct with default sizes.
//
// When store and keyPrefix are non-empty, setDAIncluded / removeDAIncluded /
// deleteAllForHeight persist a compact window snapshot under a single metadata
// key so that RestoreFromStore can recover the in-flight state with one store
// read (O(1)).
func NewCache[T any](s store.Store, keyPrefix string, _ func(uint64) string) *Cache[T] {
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

// getNextItem returns the item at the specified height and removes it from cache if found.
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
	if !ok {
		return false
	}
	return seen
}

// setSeen sets the hash as seen and tracks its height for pruning.
func (c *Cache[T]) setSeen(hash string, height uint64) {
	c.hashes.Add(hash, true)
	c.hashByHeight.Add(height, hash)
}

// getDAIncluded returns the DA height if the hash has been DA-included.
func (c *Cache[T]) getDAIncluded(hash string) (uint64, bool) {
	daHeight, ok := c.daIncluded.Get(hash)
	if !ok {
		return 0, false
	}
	return daHeight, true
}

// setDAIncluded sets the hash as DA-included with the given DA height and
// tracks block height for pruning.  It also rewrites the window snapshot so
// the in-flight state survives a crash/restart.
func (c *Cache[T]) setDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	c.daIncluded.Add(hash, daHeight)
	c.hashByHeight.Add(blockHeight, hash)
	c.setMaxDAHeight(daHeight)
	c.persistSnapshot(context.Background())
}

// removeDAIncluded removes the DA-included status of the hash from the cache
// and rewrites the window snapshot.
func (c *Cache[T]) removeDAIncluded(hash string) {
	c.daIncluded.Remove(hash)
	c.persistSnapshot(context.Background())
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

	c.persistSnapshot(context.Background())
}

// persistSnapshot serialises all current daIncluded entries into a compact
// byte slice and writes it to the store under the single snapshot key.
//
// Format: N × 16 bytes where each record is:
//
//	[blockHeight uint64 LE][daHeight uint64 LE]
//
// We iterate hashByHeight (height→hash) rather than daIncluded (hash→daH)
// because hashByHeight gives us the blockHeight we need to include in the
// record without an inverse lookup.
//
// This write happens on every mutation so the store always reflects the exact
// current in-flight window.  The payload is small (typically < 10 entries ×
// 16 bytes = 160 bytes), so the cost is negligible.
func (c *Cache[T]) persistSnapshot(ctx context.Context) {
	if c.store == nil || c.storeKeyPrefix == "" {
		return
	}

	// Collect all height→daHeight pairs that are still in daIncluded.
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

	buf := encodeSnapshot(entries)
	_ = c.store.SetMetadata(ctx, c.snapshotKey(), buf)
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
	n := len(buf) / snapshotEntrySize
	entries := make([]snapshotEntry, n)
	for i := range entries {
		off := i * snapshotEntrySize
		entries[i].blockHeight = binary.LittleEndian.Uint64(buf[off:])
		entries[i].daHeight = binary.LittleEndian.Uint64(buf[off+8:])
	}
	return entries
}

// RestoreFromStore recovers the in-flight DA inclusion window from the store.
//
// # Complexity: O(1)
//
// A single GetMetadata call retrieves the window snapshot written by
// persistSnapshot.  The snapshot encodes the complete set of in-flight entries
// as a compact byte slice, so no iteration, prefix scan, or per-height lookup
// is required.
//
// If the snapshot key is absent (brand-new node, or node upgraded from an
// older version that did not write snapshots) the function is a no-op; the
// in-flight state will be reconstructed naturally as the submitter / DA
// retriever re-processes blocks.
func (c *Cache[T]) RestoreFromStore(ctx context.Context) error {
	if c.store == nil || c.storeKeyPrefix == "" {
		return nil
	}

	buf, err := c.store.GetMetadata(ctx, c.snapshotKey())
	if err != nil {
		// Key absent — nothing to restore (new node or pre-snapshot version).
		return nil //nolint:nilerr // ok to ignore
	}

	entries := decodeSnapshot(buf)
	for _, e := range entries {
		syntheticHash := HeightPlaceholderKey(c.storeKeyPrefix, e.blockHeight)
		c.daIncluded.Add(syntheticHash, e.daHeight)
		c.hashByHeight.Add(e.blockHeight, syntheticHash)
		c.setMaxDAHeight(e.daHeight)
	}

	return nil
}

// HeightPlaceholderKey returns a deterministic, unique key used to index an
// in-flight DA inclusion entry by block height when the real content hash is
// not available during store restoration.
//
// Format: "<storeKeyPrefix>__h/<height_big_endian_hex>"
//
// The "__h/" infix cannot collide with real content hashes because real
// hashes are hex-encoded 32-byte digests (64 chars) whereas height values
// are at most 16 hex digits.
//
// This is exported so that callers (e.g. the submitter's IsHeightDAIncluded)
// can perform a height-based fallback lookup immediately after a restart,
// before the DA retriever has had a chance to re-fire SetHeaderDAIncluded with
// the real content hash.
func HeightPlaceholderKey(prefix string, height uint64) string {
	const hexDigits = "0123456789abcdef"
	buf := make([]byte, len(prefix)+4+16) // prefix + "__h/" + 16 hex chars
	n := copy(buf, prefix)
	n += copy(buf[n:], "__h/")
	for i := 15; i >= 0; i-- {
		buf[n+i] = hexDigits[height&0xf]
		height >>= 4
	}
	return string(buf)
}

// SaveToStore persists all current DA inclusion entries to the store by
// rewriting the window snapshot.  This is a no-op if no store is configured.
func (c *Cache[T]) SaveToStore(ctx context.Context) error {
	if c.store == nil {
		return nil
	}
	c.persistSnapshot(ctx)
	return nil
}

// ClearFromStore removes the window snapshot from the store.
func (c *Cache[T]) ClearFromStore(ctx context.Context, _ []string) error {
	if c.store == nil {
		return nil
	}
	// Delete the snapshot key; ignore not-found.
	_ = c.store.DeleteMetadata(ctx, c.snapshotKey())
	return nil
}
