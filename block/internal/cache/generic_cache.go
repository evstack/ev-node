package cache

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	ds "github.com/ipfs/go-datastore"

	"github.com/evstack/ev-node/pkg/store"
)

// snapshotEntry is one record in the persisted snapshot.
// Encoded as 16 bytes: [blockHeight uint64 LE][daHeight uint64 LE].
type snapshotEntry struct {
	blockHeight uint64
	daHeight    uint64
}

const snapshotEntrySize = 16 // bytes per snapshotEntry

// Cache tracks seen blocks and DA inclusion status.
type Cache struct {
	mu sync.RWMutex

	hashes       map[string]bool
	daIncluded   map[string]uint64
	hashByHeight map[uint64]string
	maxDAHeight  *atomic.Uint64

	store          store.Store
	storeKeyPrefix string
}

func (c *Cache) snapshotKey() string {
	return c.storeKeyPrefix + "__snap"
}

// NewCache creates a Cache. When store and keyPrefix are set, mutations
// persist a snapshot so RestoreFromStore can recover in-flight state.
func NewCache(s store.Store, keyPrefix string) *Cache {
	return &Cache{
		hashes:         make(map[string]bool),
		daIncluded:     make(map[string]uint64),
		hashByHeight:   make(map[uint64]string),
		maxDAHeight:    &atomic.Uint64{},
		store:          s,
		storeKeyPrefix: keyPrefix,
	}
}

func (c *Cache) isSeen(hash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hashes[hash]
}

// areSeen checks which hashes have been seen. Returns a boolean slice
// parallel to the input where result[i] is true if hashes[i] is in the
// cache. Acquires the read lock once for the entire batch.
func (c *Cache) areSeen(hashes []string) []bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]bool, len(hashes))
	for i, h := range hashes {
		result[i] = c.hashes[h]
	}
	return result
}

func (c *Cache) setSeen(hash string, height uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.hashByHeight[height]; ok && existing == hash {
		c.hashes[existing] = true
		return
	}
	c.hashes[hash] = true
	c.hashByHeight[height] = hash
}

func (c *Cache) removeSeen(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.hashes, hash)
}

// setSeenBatch marks all hashes as seen under a single write lock.
// For height 0 (transactions), the hashByHeight bookkeeping is skipped
// since all txs share the same sentinel height — the map lookup and
// overwrite on every entry is pure overhead with no benefit.
func (c *Cache) setSeenBatch(hashes []string, height uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if height == 0 {
		for _, h := range hashes {
			c.hashes[h] = true
		}
		return
	}

	// currently not used, but there for completeness against setSeen
	for _, h := range hashes {
		if existing, ok := c.hashByHeight[height]; ok && existing == h {
			c.hashes[existing] = true
			continue
		}
		c.hashes[h] = true
		c.hashByHeight[height] = h
	}
}

func (c *Cache) getDAIncluded(hash string) (uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.daIncluded[hash]
	return v, ok
}

func (c *Cache) getDAIncludedByHeight(blockHeight uint64) (uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	hash, ok := c.hashByHeight[blockHeight]
	if !ok {
		return 0, false
	}
	v, exists := c.daIncluded[hash]
	return v, exists
}

// setDAIncluded records DA inclusion in memory.
// If a previous entry already exists at blockHeight (e.g. a placeholder from
// RestoreFromStore), it is evicted from daIncluded to avoid orphan leaks.
func (c *Cache) setDAIncluded(hash string, daHeight uint64, blockHeight uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.hashByHeight[blockHeight]; ok && prev != hash {
		delete(c.daIncluded, prev)
	}
	c.daIncluded[hash] = daHeight
	c.hashByHeight[blockHeight] = hash
	c.setMaxDAHeight(daHeight)
}

func (c *Cache) removeDAIncluded(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.daIncluded, hash)
}

func (c *Cache) daHeight() uint64 {
	return c.maxDAHeight.Load()
}

func (c *Cache) setMaxDAHeight(daHeight uint64) {
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

func (c *Cache) deleteAllForHeight(height uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hash, ok := c.hashByHeight[height]
	if !ok {
		return
	}
	delete(c.hashByHeight, height)
	delete(c.hashes, hash)
	delete(c.daIncluded, hash)
}

func (c *Cache) daIncludedLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.daIncluded)
}

// persistSnapshot writes all current in-flight [blockHeight, daHeight] pairs to the store under a single key.
// Only called explicitly via SaveToStore. NEVER CALL IT ON HOT-PATH TO AVOID BAGER WRITE AMPLIFICATION.
func (c *Cache) persistSnapshot(ctx context.Context) error {
	if c.store == nil || c.storeKeyPrefix == "" {
		return nil
	}

	c.mu.RLock()
	entries := make([]snapshotEntry, 0, len(c.hashByHeight))
	for h, hash := range c.hashByHeight {
		daH, ok := c.daIncluded[hash]
		if !ok {
			continue
		}
		entries = append(entries, snapshotEntry{blockHeight: h, daHeight: daH})
	}
	c.mu.RUnlock()

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
func (c *Cache) RestoreFromStore(ctx context.Context) error {
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

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, e := range decodeSnapshot(buf) {
		placeholder := HeightPlaceholderKey(c.storeKeyPrefix, e.blockHeight)
		c.daIncluded[placeholder] = e.daHeight
		c.hashByHeight[e.blockHeight] = placeholder
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
func (c *Cache) SaveToStore(ctx context.Context) error {
	if c.store == nil {
		return nil
	}
	if err := c.persistSnapshot(ctx); err != nil {
		return fmt.Errorf("saving cache snapshot: %w", err)
	}
	return nil
}

// ClearFromStore deletes the snapshot key from the store.
func (c *Cache) ClearFromStore(ctx context.Context) error {
	if c.store == nil {
		return nil
	}
	if err := c.store.DeleteMetadata(ctx, c.snapshotKey()); err != nil {
		return fmt.Errorf("clearing cache snapshot: %w", err)
	}
	return nil
}
