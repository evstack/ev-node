package store

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/celestiaorg/go-header"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/evstack/ev-node/types"
)

// DataStoreAdapter wraps Store to implement header.Store[*types.Data].
// This allows the ev-node store to be used directly by go-header's P2P infrastructure,
// eliminating the need for a separate go-header store and reducing data duplication.
//
// The adapter maintains an in-memory cache for data received via P2P (through Append).
// This cache allows the go-header syncer and P2P handler to access data before it
// is validated and persisted by the ev-node syncer. Once the ev-node syncer processes
// a block, it writes to the underlying store, and subsequent reads will come from the store.
type DataStoreAdapter struct {
	store Store

	// height caches the current height to avoid repeated context-based lookups.
	// Updated on successful reads and writes.
	height atomic.Uint64

	// mu protects initialization state
	mu          sync.RWMutex
	initialized bool

	// pendingData is an LRU cache for data received via Append that hasn't been
	// written to the store yet. Keyed by height. Using LRU prevents unbounded growth.
	pendingData *lru.Cache[uint64, *types.Data]

	// onDeleteFn is called when data is deleted (for rollback scenarios)
	onDeleteFn func(context.Context, uint64) error
}

// Compile-time check that DataStoreAdapter implements header.Store
var _ header.Store[*types.Data] = (*DataStoreAdapter)(nil)

// NewDataStoreAdapter creates a new DataStoreAdapter wrapping the given store.
func NewDataStoreAdapter(store Store) *DataStoreAdapter {
	// Create LRU cache for pending data - ignore error as size is constant and valid
	pendingCache, _ := lru.New[uint64, *types.Data](defaultPendingCacheSize)

	adapter := &DataStoreAdapter{
		store:       store,
		pendingData: pendingCache,
	}

	// Initialize height from store
	if h, err := store.Height(context.Background()); err == nil && h > 0 {
		adapter.height.Store(h)
		adapter.initialized = true
	}

	return adapter
}

// Start implements header.Store. It initializes the adapter if needed.
func (a *DataStoreAdapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Refresh height from store
	h, err := a.store.Height(ctx)
	if err != nil {
		return err
	}

	if h > 0 {
		a.height.Store(h)
		a.initialized = true
	}

	return nil
}

// Stop implements header.Store. No-op since the underlying store lifecycle
// is managed separately.
func (a *DataStoreAdapter) Stop(ctx context.Context) error {
	return nil
}

// Head returns the data for the highest block in the store.
func (a *DataStoreAdapter) Head(ctx context.Context, _ ...header.HeadOption[*types.Data]) (*types.Data, error) {
	// First check the store height
	storeHeight, err := a.store.Height(ctx)
	if err != nil && storeHeight == 0 {
		// Check pending data
		if a.pendingData.Len() == 0 {
			return nil, header.ErrNotFound
		}

		// Find the highest pending data
		var maxHeight uint64
		var head *types.Data
		for _, h := range a.pendingData.Keys() {
			if d, ok := a.pendingData.Peek(h); ok && h > maxHeight {
				maxHeight = h
				head = d
			}
		}
		if head != nil {
			return head, nil
		}
		return nil, header.ErrNotFound
	}

	// Check if we have higher pending data
	var maxPending uint64
	var pendingHead *types.Data
	for _, h := range a.pendingData.Keys() {
		if d, ok := a.pendingData.Peek(h); ok && h > maxPending {
			maxPending = h
			pendingHead = d
		}
	}

	if maxPending > storeHeight && pendingHead != nil {
		a.height.Store(maxPending)
		return pendingHead, nil
	}

	if storeHeight == 0 {
		return nil, header.ErrNotFound
	}

	a.height.Store(storeHeight)
	_, data, err := a.store.GetBlockData(ctx, storeHeight)
	if err != nil {
		return nil, header.ErrNotFound
	}

	return data, nil
}

// Tail returns the data for the lowest block in the store.
// For ev-node, this is typically the genesis/initial height.
func (a *DataStoreAdapter) Tail(ctx context.Context) (*types.Data, error) {
	height := a.height.Load()
	if height == 0 {
		// Check store
		h, err := a.store.Height(ctx)
		if err != nil || h == 0 {
			return nil, header.ErrNotFound
		}
		height = h
	}

	// Try height 1 first (most common case)
	_, data, err := a.store.GetBlockData(ctx, 1)
	if err == nil {
		return data, nil
	}

	// Check pending for height 1
	if pendingData, ok := a.pendingData.Peek(1); ok {
		return pendingData, nil
	}

	// Linear scan from 1 to current height to find first data
	for h := uint64(2); h <= height; h++ {
		_, data, err = a.store.GetBlockData(ctx, h)
		if err == nil {
			return data, nil
		}
		if pendingData, ok := a.pendingData.Peek(h); ok {
			return pendingData, nil
		}
	}

	return nil, header.ErrNotFound
}

// Get returns data by its hash.
func (a *DataStoreAdapter) Get(ctx context.Context, hash header.Hash) (*types.Data, error) {
	// First try the store
	_, data, err := a.store.GetBlockByHash(ctx, hash)
	if err == nil {
		return data, nil
	}

	// Check pending data - note: this checks data hash, not header hash
	for _, h := range a.pendingData.Keys() {
		if pendingData, ok := a.pendingData.Peek(h); ok && pendingData != nil && bytesEqual(pendingData.Hash(), hash) {
			return pendingData, nil
		}
	}

	return nil, header.ErrNotFound
}

// GetByHeight returns data at the given height.
func (a *DataStoreAdapter) GetByHeight(ctx context.Context, height uint64) (*types.Data, error) {
	// First try the store
	_, data, err := a.store.GetBlockData(ctx, height)
	if err == nil {
		return data, nil
	}

	// Check pending data
	if pendingData, ok := a.pendingData.Peek(height); ok {
		return pendingData, nil
	}

	return nil, header.ErrNotFound
}

// GetRangeByHeight returns data in the range [from.Height()+1, to).
// This follows go-header's convention where 'from' is the trusted data
// and we return data starting from the next height.
func (a *DataStoreAdapter) GetRangeByHeight(ctx context.Context, from *types.Data, to uint64) ([]*types.Data, error) {
	if from == nil {
		return nil, errors.New("from data cannot be nil")
	}

	startHeight := from.Height() + 1
	if startHeight >= to {
		return nil, nil
	}

	return a.GetRange(ctx, startHeight, to)
}

// GetRange returns data in the range [from, to).
func (a *DataStoreAdapter) GetRange(ctx context.Context, from, to uint64) ([]*types.Data, error) {
	if from >= to {
		return nil, nil
	}

	dataList := make([]*types.Data, 0, to-from)
	for height := from; height < to; height++ {
		data, err := a.GetByHeight(ctx, height)
		if err != nil {
			// Return what we have so far
			if len(dataList) > 0 {
				return dataList, nil
			}
			return nil, header.ErrNotFound
		}
		dataList = append(dataList, data)
	}

	return dataList, nil
}

// Has checks if data with the given hash exists.
func (a *DataStoreAdapter) Has(ctx context.Context, hash header.Hash) (bool, error) {
	// Check store first
	_, _, err := a.store.GetBlockByHash(ctx, hash)
	if err == nil {
		return true, nil
	}

	// Check pending data
	for _, h := range a.pendingData.Keys() {
		if pendingData, ok := a.pendingData.Peek(h); ok && pendingData != nil && bytesEqual(pendingData.Hash(), hash) {
			return true, nil
		}
	}

	return false, nil
}

// HasAt checks if data exists at the given height.
func (a *DataStoreAdapter) HasAt(ctx context.Context, height uint64) bool {
	// Check store first
	_, _, err := a.store.GetBlockData(ctx, height)
	if err == nil {
		return true
	}

	// Check pending data
	return a.pendingData.Contains(height)
}

// Height returns the current height of the store.
func (a *DataStoreAdapter) Height() uint64 {
	// Check store first
	if h, err := a.store.Height(context.Background()); err == nil && h > 0 {
		// Also check pending for higher heights
		maxPending := uint64(0)
		for _, height := range a.pendingData.Keys() {
			if height > maxPending {
				maxPending = height
			}
		}

		if maxPending > h {
			a.height.Store(maxPending)
			return maxPending
		}
		a.height.Store(h)
		return h
	}

	// Fall back to cached height or check pending
	height := a.height.Load()
	if height > 0 {
		return height
	}

	for _, h := range a.pendingData.Keys() {
		if h > height {
			height = h
		}
	}
	return height
}

// Append stores data in the pending cache.
// This data is received via P2P and will be available for retrieval
// until the ev-node syncer processes and persists it to the store.
func (a *DataStoreAdapter) Append(ctx context.Context, dataList ...*types.Data) error {
	if len(dataList) == 0 {
		return nil
	}

	for _, data := range dataList {
		if data == nil || data.IsZero() {
			continue
		}

		height := data.Height()

		// Check if already in store
		if _, _, err := a.store.GetBlockData(ctx, height); err == nil {
			// Already persisted, skip
			continue
		}

		// Add to pending cache (LRU will evict oldest if full)
		a.pendingData.Add(height, data)

		// Update cached height
		if height > a.height.Load() {
			a.height.Store(height)
		}
	}

	return nil
}

// Init initializes the store with the first data.
// This is called by go-header when bootstrapping the store with trusted data.
func (a *DataStoreAdapter) Init(ctx context.Context, d *types.Data) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return nil
	}

	if d == nil || d.IsZero() {
		return nil
	}

	// Add to pending cache (LRU will evict oldest if full)
	a.pendingData.Add(d.Height(), d)
	a.height.Store(d.Height())
	a.initialized = true

	return nil
}

// Sync ensures all pending writes are flushed.
// No-op for the adapter as pending data is in-memory cache.
func (a *DataStoreAdapter) Sync(ctx context.Context) error {
	return nil
}

// DeleteRange deletes data in the range [from, to).
// This is used for rollback operations.
func (a *DataStoreAdapter) DeleteRange(ctx context.Context, from, to uint64) error {
	// Remove from pending cache
	for height := from; height < to; height++ {
		a.pendingData.Remove(height)

		if a.onDeleteFn != nil {
			if err := a.onDeleteFn(ctx, height); err != nil {
				return err
			}
		}
	}

	// Update cached height if necessary
	if from <= a.height.Load() {
		a.height.Store(from - 1)
	}

	return nil
}

// OnDelete registers a callback to be invoked when data is deleted.
func (a *DataStoreAdapter) OnDelete(fn func(context.Context, uint64) error) {
	a.onDeleteFn = fn
}

// RefreshHeight updates the cached height from the underlying store.
// This should be called after the syncer processes a new block.
func (a *DataStoreAdapter) RefreshHeight(ctx context.Context) error {
	h, err := a.store.Height(ctx)
	if err != nil {
		return err
	}
	a.height.Store(h)

	// Clean up pending data that is now in store
	for _, height := range a.pendingData.Keys() {
		if height <= h {
			a.pendingData.Remove(height)
		}
	}

	return nil
}

// SetHeight updates the cached height.
// This is useful when the syncer knows the new height after processing a block.
func (a *DataStoreAdapter) SetHeight(height uint64) {
	a.height.Store(height)

	// Clean up pending data at or below this height
	for _, h := range a.pendingData.Keys() {
		if h <= height {
			a.pendingData.Remove(h)
		}
	}
}
