package store

import (
	"context"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/evstack/ev-node/types"
)

const (
	// DefaultHeaderCacheSize is the default number of headers to cache in memory.
	DefaultHeaderCacheSize = 200_000

	// DefaultBlockDataCacheSize is the default number of block data entries to cache.
	DefaultBlockDataCacheSize = 200_000
)

// CachedStore wraps a Store with LRU caching for frequently accessed data.
// The underlying LRU cache is thread-safe, so no additional synchronization is needed.
type CachedStore struct {
	Store

	headerCache    *lru.Cache[uint64, *types.SignedHeader]
	blockDataCache *lru.Cache[uint64, *blockDataEntry]
}

type blockDataEntry struct {
	header *types.SignedHeader
	data   *types.Data
}

// CachedStoreOption configures a CachedStore.
type CachedStoreOption func(*CachedStore) error

// WithHeaderCacheSize sets the header cache size.
func WithHeaderCacheSize(size int) CachedStoreOption {
	return func(cs *CachedStore) error {
		cache, err := lru.New[uint64, *types.SignedHeader](size)
		if err != nil {
			return err
		}
		cs.headerCache = cache
		return nil
	}
}

// WithBlockDataCacheSize sets the block data cache size.
func WithBlockDataCacheSize(size int) CachedStoreOption {
	return func(cs *CachedStore) error {
		cache, err := lru.New[uint64, *blockDataEntry](size)
		if err != nil {
			return err
		}
		cs.blockDataCache = cache
		return nil
	}
}

// NewCachedStore creates a new CachedStore wrapping the given store.
func NewCachedStore(store Store, opts ...CachedStoreOption) (*CachedStore, error) {
	headerCache, err := lru.New[uint64, *types.SignedHeader](DefaultHeaderCacheSize)
	if err != nil {
		return nil, err
	}

	blockDataCache, err := lru.New[uint64, *blockDataEntry](DefaultBlockDataCacheSize)
	if err != nil {
		return nil, err
	}

	cs := &CachedStore{
		Store:          store,
		headerCache:    headerCache,
		blockDataCache: blockDataCache,
	}

	for _, opt := range opts {
		if err := opt(cs); err != nil {
			return nil, err
		}
	}

	return cs, nil
}

// GetHeader returns the header at the given height, using the cache if available.
func (cs *CachedStore) GetHeader(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	// Try cache first
	if header, ok := cs.headerCache.Get(height); ok {
		return header, nil
	}

	// Cache miss, fetch from underlying store
	header, err := cs.Store.GetHeader(ctx, height)
	if err != nil {
		return nil, err
	}

	// Add to cache
	cs.headerCache.Add(height, header)

	return header, nil
}

// GetBlockData returns block header and data at given height, using cache if available.
func (cs *CachedStore) GetBlockData(ctx context.Context, height uint64) (*types.SignedHeader, *types.Data, error) {
	// Try cache first
	if entry, ok := cs.blockDataCache.Get(height); ok {
		return entry.header, entry.data, nil
	}

	// Cache miss, fetch from underlying store
	header, data, err := cs.Store.GetBlockData(ctx, height)
	if err != nil {
		return nil, nil, err
	}

	// Add to cache
	cs.blockDataCache.Add(height, &blockDataEntry{header: header, data: data})

	// Also add header to header cache
	cs.headerCache.Add(height, header)

	return header, data, nil
}

// InvalidateRange removes headers in the given range from the cache.
func (cs *CachedStore) InvalidateRange(fromHeight, toHeight uint64) {
	for h := fromHeight; h <= toHeight; h++ {
		cs.headerCache.Remove(h)
		cs.blockDataCache.Remove(h)
	}
}

// ClearCache clears all cached entries.
func (cs *CachedStore) ClearCache() {
	cs.headerCache.Purge()
	cs.blockDataCache.Purge()
}

// Rollback wraps the underlying store's Rollback and invalidates affected cache entries.
func (cs *CachedStore) Rollback(ctx context.Context, height uint64, aggregator bool) error {
	currentHeight, err := cs.Height(ctx)
	if err != nil {
		return err
	}

	// First do the rollback
	if err := cs.Store.Rollback(ctx, height, aggregator); err != nil {
		return err
	}

	// Then invalidate cache entries for rolled back heights
	cs.InvalidateRange(height+1, currentHeight)

	return nil
}

// Close closes the underlying store.
func (cs *CachedStore) Close() error {
	cs.ClearCache()
	return cs.Store.Close()
}
