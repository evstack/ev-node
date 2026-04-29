package store

import (
	"context"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/types"
)

const (
	// DefaultHeaderCacheSize is the default number of headers to cache in memory.
	DefaultHeaderCacheSize = 200_000

	// DefaultBlockDataCacheSize is the default number of block data entries to cache.
	DefaultBlockDataCacheSize = 200_000

	asyncWriteBufferSize = 8192

	// batchWindow is the time the write goroutine waits after receiving the first
	// op before flushing. This allows bursts of metadata writes (e.g. 3-4 per
	// height in the submitter) to be coalesced into a single Badger WriteBatch.
	batchWindow = 100 * time.Microsecond
)

type asyncWriteOp struct {
	key      string
	value    []byte
	isDelete bool
}

// CachedStore wraps a Store with LRU caching for frequently accessed data.
// The underlying LRU cache is thread-safe, so no additional synchronization is needed.
// Metadata writes (SetMetadata, DeleteMetadata) are processed asynchronously via a
// buffered channel to avoid blocking Badger's write pipeline for critical operations
// like block production (batch commits).
type CachedStore struct {
	Store

	headerCache    *lru.Cache[uint64, *types.SignedHeader]
	blockDataCache *lru.Cache[uint64, *blockDataEntry]

	writeCh chan asyncWriteOp
	done    chan struct{}
	stopMu  sync.RWMutex
	stopped bool
	logger  zerolog.Logger
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
		writeCh:        make(chan asyncWriteOp, asyncWriteBufferSize),
		done:           make(chan struct{}),
		logger:         zerolog.Nop(),
	}

	for _, opt := range opts {
		if err := opt(cs); err != nil {
			return nil, err
		}
	}

	cs.startWriteLoop()

	return cs, nil
}

func (cs *CachedStore) startWriteLoop() {
	go func() {
		defer close(cs.done)
		for op := range cs.writeCh {
			ops := []asyncWriteOp{op}

			timer := time.NewTimer(batchWindow)
		collect:
			for {
				select {
				case op, ok := <-cs.writeCh:
					if !ok {
						timer.Stop()
						break collect
					}
					ops = append(ops, op)
				case <-timer.C:
					break collect
				}
			}

			last := make(map[string]asyncWriteOp, len(ops))
			for _, o := range ops {
				last[o.key] = o
			}

			var puts []MetadataKV
			var deletes []string
			for _, o := range last {
				if o.isDelete {
					deletes = append(deletes, o.key)
				} else {
					puts = append(puts, MetadataKV{Key: o.key, Value: o.value})
				}
			}

			if err := cs.BatchMetadata(context.Background(), puts, deletes); err != nil {
				for _, o := range ops {
					cs.logger.Error().Err(err).Str("key", o.key).
						Bool("delete", o.isDelete).
						Msg("async metadata batch write failed")
				}
			}
		}
	}()
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

	header.MemoizeHash()

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

	header.MemoizeHash()

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

// PruneBlocks wraps the underlying store's PruneBlocks and invalidates caches
// up to the height that we prune
func (cs *CachedStore) PruneBlocks(ctx context.Context, height uint64) error {
	if err := cs.Store.PruneBlocks(ctx, height); err != nil {
		return err
	}

	// Invalidate cache for pruned heights
	cs.InvalidateRange(1, height)
	return nil
}

// SetMetadata queues an asynchronous metadata write. The write is persisted
// by the background goroutine via BatchMetadata. If the store has been stopped,
// the write falls back to synchronous execution on the underlying store.
func (cs *CachedStore) SetMetadata(ctx context.Context, key string, value []byte) error {
	cs.stopMu.RLock()
	defer cs.stopMu.RUnlock()

	if cs.stopped {
		return cs.Store.SetMetadata(ctx, key, value)
	}
	valueCopy := append([]byte(nil), value...)
	cs.writeCh <- asyncWriteOp{key: key, value: valueCopy}
	return nil
}

// DeleteMetadata queues an asynchronous metadata delete. If the store has been
// stopped, the delete falls back to synchronous execution.
func (cs *CachedStore) DeleteMetadata(ctx context.Context, key string) error {
	cs.stopMu.RLock()
	defer cs.stopMu.RUnlock()

	if cs.stopped {
		return cs.Store.DeleteMetadata(ctx, key)
	}
	cs.writeCh <- asyncWriteOp{key: key, isDelete: true}
	return nil
}

// Close drains pending async writes, then closes the underlying store.
func (cs *CachedStore) Close() error {
	cs.stopMu.Lock()
	cs.stopped = true
	close(cs.writeCh)
	cs.stopMu.Unlock()
	<-cs.done

	cs.ClearCache()
	return cs.Store.Close()
}
