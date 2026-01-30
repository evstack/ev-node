package store

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"

	"github.com/celestiaorg/go-header"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

// defaultPendingCacheSize is the default size for the pending headers/data LRU cache.
const defaultPendingCacheSize = 1000

// StoreGetter abstracts the store access methods for different types (headers vs data).
type StoreGetter[H header.Header[H]] interface {
	// GetByHeight retrieves an item by its height.
	GetByHeight(ctx context.Context, height uint64) (H, error)
	// GetByHash retrieves an item by its hash.
	GetByHash(ctx context.Context, hash []byte) (H, error)
	// Height returns the current height of the store.
	Height(ctx context.Context) (uint64, error)
	// HasAt checks if an item exists at the given height.
	HasAt(ctx context.Context, height uint64) bool
}

// StoreAdapter is a generic adapter that wraps Store to implement header.Store[H].
// This allows the ev-node store to be used directly by go-header's P2P infrastructure,
// eliminating the need for a separate go-header store and reducing data duplication.
//
// The adapter maintains an in-memory cache for items received via P2P (through Append).
// This cache allows the go-header syncer and P2P handler to access items before they
// are validated and persisted by the ev-node syncer. Once the ev-node syncer processes
// a block, it writes to the underlying store, and subsequent reads will come from the store.
type StoreAdapter[H header.Header[H]] struct {
	getter  StoreGetter[H]
	genesis genesis.Genesis

	// height caches the current height to avoid repeated context-based lookups.
	// Updated on successful reads and writes.
	height atomic.Uint64

	// mu protects initialization state
	mu          sync.RWMutex
	initialized bool

	// pending is an LRU cache for items received via Append that haven't been
	// written to the store yet. Keyed by height. Using LRU prevents unbounded growth.
	pending *lru.Cache[uint64, H]

	// onDeleteFn is called when items are deleted (for rollback scenarios)
	onDeleteFn func(context.Context, uint64) error
}

// NewStoreAdapter creates a new StoreAdapter wrapping the given store getter.
// The genesis is used to determine the initial height for efficient Tail lookups.
func NewStoreAdapter[H header.Header[H]](getter StoreGetter[H], gen genesis.Genesis) *StoreAdapter[H] {
	// Create LRU cache for pending items - ignore error as size is constant and valid
	pendingCache, _ := lru.New[uint64, H](defaultPendingCacheSize)

	adapter := &StoreAdapter[H]{
		getter:  getter,
		genesis: gen,
		pending: pendingCache,
	}

	// Initialize height from store
	if h, err := getter.Height(context.Background()); err == nil && h > 0 {
		adapter.height.Store(h)
		adapter.initialized = true
	}

	return adapter
}

// Start implements header.Store. It initializes the adapter if needed.
func (a *StoreAdapter[H]) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Refresh height from store
	h, err := a.getter.Height(ctx)
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
func (a *StoreAdapter[H]) Stop(ctx context.Context) error {
	return nil
}

// pendingHead returns the highest item in the pending cache and its height.
// Returns zero value and 0 if pending cache is empty.
func (a *StoreAdapter[H]) pendingHead() (H, uint64) {
	var maxHeight uint64
	var head H
	for _, h := range a.pending.Keys() {
		if item, ok := a.pending.Peek(h); ok && h > maxHeight {
			maxHeight = h
			head = item
		}
	}
	return head, maxHeight
}

// Head returns the highest item in the store.
func (a *StoreAdapter[H]) Head(ctx context.Context, _ ...header.HeadOption[H]) (H, error) {
	var zero H

	storeHeight, _ := a.getter.Height(ctx)
	pendingHead, pendingHeight := a.pendingHead()

	// Prefer pending if it's higher than store
	if pendingHeight > storeHeight {
		a.height.Store(pendingHeight)
		return pendingHead, nil
	}

	// Try to get from store
	if storeHeight > 0 {
		a.height.Store(storeHeight)
		if item, err := a.getter.GetByHeight(ctx, storeHeight); err == nil {
			return item, nil
		}
	}

	// Fall back to pending if store failed
	if pendingHeight > 0 {
		a.height.Store(pendingHeight)
		return pendingHead, nil
	}

	return zero, header.ErrNotFound
}

// Tail returns the lowest item in the store.
// For ev-node, this is typically the genesis/initial height.
// If pruning has occurred, it walks up from initialHeight to find the first available item.
// TODO(@julienrbrt): Optimize this when pruning is enabled.
func (a *StoreAdapter[H]) Tail(ctx context.Context) (H, error) {
	var zero H

	height := a.height.Load()
	if height == 0 {
		// Check store
		h, err := a.getter.Height(ctx)
		if err != nil || h == 0 {
			return zero, header.ErrNotFound
		}
		height = h
	}

	initialHeight := a.genesis.InitialHeight
	if initialHeight == 0 {
		initialHeight = 1
	}

	// Try initialHeight first (most common case - no pruning)
	item, err := a.getter.GetByHeight(ctx, initialHeight)
	if err == nil {
		return item, nil
	}

	// Check pending for initialHeight
	if pendingItem, ok := a.pending.Peek(initialHeight); ok {
		return pendingItem, nil
	}

	// Walk up from initialHeight to find the first available item (pruning case)
	for h := initialHeight + 1; h <= height; h++ {
		item, err = a.getter.GetByHeight(ctx, h)
		if err == nil {
			return item, nil
		}
		if pendingItem, ok := a.pending.Peek(h); ok {
			return pendingItem, nil
		}
	}

	return zero, header.ErrNotFound
}

// Get returns an item by its hash.
func (a *StoreAdapter[H]) Get(ctx context.Context, hash header.Hash) (H, error) {
	var zero H

	// First try the store
	item, err := a.getter.GetByHash(ctx, hash)
	if err == nil {
		return item, nil
	}

	// Check pending items
	for _, h := range a.pending.Keys() {
		if pendingItem, ok := a.pending.Peek(h); ok && !pendingItem.IsZero() && bytes.Equal(pendingItem.Hash(), hash) {
			return pendingItem, nil
		}
	}

	return zero, header.ErrNotFound
}

// GetByHeight returns an item at the given height.
func (a *StoreAdapter[H]) GetByHeight(ctx context.Context, height uint64) (H, error) {
	var zero H

	// First try the store
	item, err := a.getter.GetByHeight(ctx, height)
	if err == nil {
		return item, nil
	}

	// Check pending items
	if pendingItem, ok := a.pending.Peek(height); ok {
		return pendingItem, nil
	}

	return zero, header.ErrNotFound
}

// GetRangeByHeight returns items in the range [from.Height()+1, to).
// This follows go-header's convention where 'from' is the trusted item
// and we return items starting from the next height.
func (a *StoreAdapter[H]) GetRangeByHeight(ctx context.Context, from H, to uint64) ([]H, error) {
	if from.IsZero() {
		return nil, header.ErrNotFound
	}

	startHeight := from.Height() + 1
	if startHeight >= to {
		return nil, nil
	}

	return a.GetRange(ctx, startHeight, to)
}

// GetRange returns items in the range [from, to).
func (a *StoreAdapter[H]) GetRange(ctx context.Context, from, to uint64) ([]H, error) {
	if from >= to {
		return nil, nil
	}

	items := make([]H, 0, to-from)
	for height := from; height < to; height++ {
		item, err := a.GetByHeight(ctx, height)
		if err != nil {
			// Return what we have so far
			if len(items) > 0 {
				return items, nil
			}
			return nil, header.ErrNotFound
		}
		items = append(items, item)
	}

	return items, nil
}

// Has checks if an item with the given hash exists.
func (a *StoreAdapter[H]) Has(ctx context.Context, hash header.Hash) (bool, error) {
	// Check store first
	_, err := a.getter.GetByHash(ctx, hash)
	if err == nil {
		return true, nil
	}

	// Check pending items
	for _, h := range a.pending.Keys() {
		if pendingItem, ok := a.pending.Peek(h); ok && !pendingItem.IsZero() && bytes.Equal(pendingItem.Hash(), hash) {
			return true, nil
		}
	}

	return false, nil
}

// HasAt checks if an item exists at the given height.
func (a *StoreAdapter[H]) HasAt(ctx context.Context, height uint64) bool {
	// Check store first
	if a.getter.HasAt(ctx, height) {
		return true
	}

	// Check pending items
	return a.pending.Contains(height)
}

// Height returns the current height of the store.
func (a *StoreAdapter[H]) Height() uint64 {
	// Check store first
	if h, err := a.getter.Height(context.Background()); err == nil && h > 0 {
		// Also check pending for higher heights
		maxPending := uint64(0)
		for _, height := range a.pending.Keys() {
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

	for _, h := range a.pending.Keys() {
		if h > height {
			height = h
		}
	}
	return height
}

// Append stores items in the pending cache.
// These items are received via P2P and will be available for retrieval
// until the ev-node syncer processes and persists them to the store.
func (a *StoreAdapter[H]) Append(ctx context.Context, items ...H) error {
	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if item.IsZero() {
			continue
		}

		height := item.Height()

		// Check if already in store
		if a.getter.HasAt(ctx, height) {
			// Already persisted, skip
			continue
		}

		// Add to pending cache (LRU will evict oldest if full)
		a.pending.Add(height, item)

		// Update cached height
		if height > a.height.Load() {
			a.height.Store(height)
		}
	}

	return nil
}

// Init initializes the store with the first item.
// This is called by go-header when bootstrapping the store with a trusted item.
func (a *StoreAdapter[H]) Init(ctx context.Context, item H) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return nil
	}

	if item.IsZero() {
		return nil
	}

	// Add to pending cache (LRU will evict oldest if full)
	a.pending.Add(item.Height(), item)
	a.height.Store(item.Height())
	a.initialized = true

	return nil
}

// Sync ensures all pending writes are flushed.
// No-op for the adapter as pending data is in-memory cache.
func (a *StoreAdapter[H]) Sync(ctx context.Context) error {
	return nil
}

// DeleteRange deletes items in the range [from, to).
// This is used for rollback operations.
func (a *StoreAdapter[H]) DeleteRange(ctx context.Context, from, to uint64) error {
	// Remove from pending cache
	for height := from; height < to; height++ {
		a.pending.Remove(height)

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

// OnDelete registers a callback to be invoked when items are deleted.
func (a *StoreAdapter[H]) OnDelete(fn func(context.Context, uint64) error) {
	a.onDeleteFn = fn
}

// HeaderStoreGetter implements StoreGetter for *types.SignedHeader.
type HeaderStoreGetter struct {
	store Store
}

// NewHeaderStoreGetter creates a new HeaderStoreGetter.
func NewHeaderStoreGetter(store Store) *HeaderStoreGetter {
	return &HeaderStoreGetter{store: store}
}

// GetByHeight implements StoreGetter.
func (g *HeaderStoreGetter) GetByHeight(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	return g.store.GetHeader(ctx, height)
}

// GetByHash implements StoreGetter.
func (g *HeaderStoreGetter) GetByHash(ctx context.Context, hash []byte) (*types.SignedHeader, error) {
	hdr, _, err := g.store.GetBlockByHash(ctx, hash)
	return hdr, err
}

// Height implements StoreGetter.
func (g *HeaderStoreGetter) Height(ctx context.Context) (uint64, error) {
	return g.store.Height(ctx)
}

// HasAt implements StoreGetter.
func (g *HeaderStoreGetter) HasAt(ctx context.Context, height uint64) bool {
	_, err := g.store.GetHeader(ctx, height)
	return err == nil
}

// DataStoreGetter implements StoreGetter for *types.Data.
type DataStoreGetter struct {
	store Store
}

// NewDataStoreGetter creates a new DataStoreGetter.
func NewDataStoreGetter(store Store) *DataStoreGetter {
	return &DataStoreGetter{store: store}
}

// GetByHeight implements StoreGetter.
func (g *DataStoreGetter) GetByHeight(ctx context.Context, height uint64) (*types.Data, error) {
	_, data, err := g.store.GetBlockData(ctx, height)
	return data, err
}

// GetByHash implements StoreGetter.
func (g *DataStoreGetter) GetByHash(ctx context.Context, hash []byte) (*types.Data, error) {
	_, data, err := g.store.GetBlockByHash(ctx, hash)
	return data, err
}

// Height implements StoreGetter.
func (g *DataStoreGetter) Height(ctx context.Context) (uint64, error) {
	return g.store.Height(ctx)
}

// HasAt implements StoreGetter.
func (g *DataStoreGetter) HasAt(ctx context.Context, height uint64) bool {
	_, _, err := g.store.GetBlockData(ctx, height)
	return err == nil
}

// Type aliases for convenience
type HeaderStoreAdapter = StoreAdapter[*types.SignedHeader]
type DataStoreAdapter = StoreAdapter[*types.Data]

// NewHeaderStoreAdapter creates a new StoreAdapter for headers.
// The genesis is used to determine the initial height for efficient Tail lookups.
func NewHeaderStoreAdapter(store Store, gen genesis.Genesis) *HeaderStoreAdapter {
	return NewStoreAdapter[*types.SignedHeader](NewHeaderStoreGetter(store), gen)
}

// NewDataStoreAdapter creates a new StoreAdapter for data.
// The genesis is used to determine the initial height for efficient Tail lookups.
func NewDataStoreAdapter(store Store, gen genesis.Genesis) *DataStoreAdapter {
	return NewStoreAdapter[*types.Data](NewDataStoreGetter(store), gen)
}
