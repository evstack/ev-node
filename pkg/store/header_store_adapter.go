package store

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/celestiaorg/go-header"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/evstack/ev-node/types"
)

// defaultPendingCacheSize is the default size for the pending headers/data LRU cache.
const defaultPendingCacheSize = 1000

// HeaderStoreAdapter wraps Store to implement header.Store[*types.SignedHeader].
// This allows the ev-node store to be used directly by go-header's P2P infrastructure,
// eliminating the need for a separate go-header store and reducing data duplication.
//
// The adapter maintains an in-memory cache for headers received via P2P (through Append).
// This cache allows the go-header syncer and P2P handler to access headers before they
// are validated and persisted by the ev-node syncer. Once the ev-node syncer processes
// a block, it writes to the underlying store, and subsequent reads will come from the store.
type HeaderStoreAdapter struct {
	store Store

	// height caches the current height to avoid repeated context-based lookups.
	// Updated on successful reads and writes.
	height atomic.Uint64

	// mu protects initialization state
	mu          sync.RWMutex
	initialized bool

	// pendingHeaders is an LRU cache for headers received via Append that haven't been
	// written to the store yet. Keyed by height. Using LRU prevents unbounded growth.
	pendingHeaders *lru.Cache[uint64, *types.SignedHeader]

	// onDeleteFn is called when headers are deleted (for rollback scenarios)
	onDeleteFn func(context.Context, uint64) error
}

// Compile-time check that HeaderStoreAdapter implements header.Store
var _ header.Store[*types.SignedHeader] = (*HeaderStoreAdapter)(nil)

// NewHeaderStoreAdapter creates a new HeaderStoreAdapter wrapping the given store.
func NewHeaderStoreAdapter(store Store) *HeaderStoreAdapter {
	// Create LRU cache for pending headers - ignore error as size is constant and valid
	pendingCache, _ := lru.New[uint64, *types.SignedHeader](defaultPendingCacheSize)

	adapter := &HeaderStoreAdapter{
		store:          store,
		pendingHeaders: pendingCache,
	}

	// Initialize height from store
	if h, err := store.Height(context.Background()); err == nil && h > 0 {
		adapter.height.Store(h)
		adapter.initialized = true
	}

	return adapter
}

// Start implements header.Store. It initializes the adapter if needed.
func (a *HeaderStoreAdapter) Start(ctx context.Context) error {
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
func (a *HeaderStoreAdapter) Stop(ctx context.Context) error {
	return nil
}

// Head returns the highest header in the store.
func (a *HeaderStoreAdapter) Head(ctx context.Context, _ ...header.HeadOption[*types.SignedHeader]) (*types.SignedHeader, error) {
	// First check the store height
	storeHeight, err := a.store.Height(ctx)
	if err != nil && storeHeight == 0 {
		// Check pending headers
		if a.pendingHeaders.Len() == 0 {
			return nil, header.ErrNotFound
		}

		// Find the highest pending header
		var maxHeight uint64
		var head *types.SignedHeader
		for _, h := range a.pendingHeaders.Keys() {
			if hdr, ok := a.pendingHeaders.Peek(h); ok && h > maxHeight {
				maxHeight = h
				head = hdr
			}
		}
		if head != nil {
			return head, nil
		}
		return nil, header.ErrNotFound
	}

	// Check if we have a higher pending header
	var maxPending uint64
	var pendingHead *types.SignedHeader
	for _, h := range a.pendingHeaders.Keys() {
		if hdr, ok := a.pendingHeaders.Peek(h); ok && h > maxPending {
			maxPending = h
			pendingHead = hdr
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
	hdr, err := a.store.GetHeader(ctx, storeHeight)
	if err != nil {
		return nil, header.ErrNotFound
	}

	return hdr, nil
}

// Tail returns the lowest header in the store.
// For ev-node, this is typically the genesis/initial height.
func (a *HeaderStoreAdapter) Tail(ctx context.Context) (*types.SignedHeader, error) {
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
	hdr, err := a.store.GetHeader(ctx, 1)
	if err == nil {
		return hdr, nil
	}

	// Check pending for height 1
	if pendingHdr, ok := a.pendingHeaders.Peek(1); ok {
		return pendingHdr, nil
	}

	// Linear scan from 1 to current height to find first header
	for h := uint64(2); h <= height; h++ {
		hdr, err = a.store.GetHeader(ctx, h)
		if err == nil {
			return hdr, nil
		}
		if pendingHdr, ok := a.pendingHeaders.Peek(h); ok {
			return pendingHdr, nil
		}
	}

	return nil, header.ErrNotFound
}

// Get returns a header by its hash.
func (a *HeaderStoreAdapter) Get(ctx context.Context, hash header.Hash) (*types.SignedHeader, error) {
	// First try the store
	hdr, _, err := a.store.GetBlockByHash(ctx, hash)
	if err == nil {
		return hdr, nil
	}

	// Check pending headers
	for _, h := range a.pendingHeaders.Keys() {
		if pendingHdr, ok := a.pendingHeaders.Peek(h); ok && pendingHdr != nil && bytes.Equal(pendingHdr.Hash(), hash) {
			return pendingHdr, nil
		}
	}

	return nil, header.ErrNotFound
}

// GetByHeight returns a header at the given height.
func (a *HeaderStoreAdapter) GetByHeight(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	// First try the store
	hdr, err := a.store.GetHeader(ctx, height)
	if err == nil {
		return hdr, nil
	}

	// Check pending headers
	if pendingHdr, ok := a.pendingHeaders.Peek(height); ok {
		return pendingHdr, nil
	}

	return nil, header.ErrNotFound
}

// GetRangeByHeight returns headers in the range [from.Height()+1, to).
// This follows go-header's convention where 'from' is the trusted header
// and we return headers starting from the next height.
func (a *HeaderStoreAdapter) GetRangeByHeight(ctx context.Context, from *types.SignedHeader, to uint64) ([]*types.SignedHeader, error) {
	if from == nil {
		return nil, errors.New("from header cannot be nil")
	}

	startHeight := from.Height() + 1
	if startHeight >= to {
		return nil, nil
	}

	return a.GetRange(ctx, startHeight, to)
}

// GetRange returns headers in the range [from, to).
func (a *HeaderStoreAdapter) GetRange(ctx context.Context, from, to uint64) ([]*types.SignedHeader, error) {
	if from >= to {
		return nil, nil
	}

	headers := make([]*types.SignedHeader, 0, to-from)
	for height := from; height < to; height++ {
		hdr, err := a.GetByHeight(ctx, height)
		if err != nil {
			// Return what we have so far
			if len(headers) > 0 {
				return headers, nil
			}
			return nil, header.ErrNotFound
		}
		headers = append(headers, hdr)
	}

	return headers, nil
}

// Has checks if a header with the given hash exists.
func (a *HeaderStoreAdapter) Has(ctx context.Context, hash header.Hash) (bool, error) {
	// Check store first
	_, _, err := a.store.GetBlockByHash(ctx, hash)
	if err == nil {
		return true, nil
	}

	// Check pending headers
	for _, h := range a.pendingHeaders.Keys() {
		if pendingHdr, ok := a.pendingHeaders.Peek(h); ok && pendingHdr != nil && bytes.Equal(pendingHdr.Hash(), hash) {
			return true, nil
		}
	}

	return false, nil
}

// HasAt checks if a header exists at the given height.
func (a *HeaderStoreAdapter) HasAt(ctx context.Context, height uint64) bool {
	// Check store first
	_, err := a.store.GetHeader(ctx, height)
	if err == nil {
		return true
	}

	// Check pending headers
	return a.pendingHeaders.Contains(height)
}

// Height returns the current height of the store.
func (a *HeaderStoreAdapter) Height() uint64 {
	// Check store first
	if h, err := a.store.Height(context.Background()); err == nil && h > 0 {
		// Also check pending for higher heights
		maxPending := uint64(0)
		for _, height := range a.pendingHeaders.Keys() {
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

	for _, h := range a.pendingHeaders.Keys() {
		if h > height {
			height = h
		}
	}
	return height
}

// Append stores headers in the pending cache.
// These headers are received via P2P and will be available for retrieval
// until the ev-node syncer processes and persists them to the store.
func (a *HeaderStoreAdapter) Append(ctx context.Context, headers ...*types.SignedHeader) error {
	if len(headers) == 0 {
		return nil
	}

	for _, hdr := range headers {
		if hdr == nil || hdr.IsZero() {
			continue
		}

		height := hdr.Height()

		// Check if already in store
		if _, err := a.store.GetHeader(ctx, height); err == nil {
			// Already persisted, skip
			continue
		}

		// Add to pending cache (LRU will evict oldest if full)
		a.pendingHeaders.Add(height, hdr)

		// Update cached height
		if height > a.height.Load() {
			a.height.Store(height)
		}
	}

	return nil
}

// Init initializes the store with the first header.
// This is called by go-header when bootstrapping the store with a trusted header.
func (a *HeaderStoreAdapter) Init(ctx context.Context, h *types.SignedHeader) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return nil
	}

	if h == nil || h.IsZero() {
		return nil
	}

	// Add to pending cache (LRU will evict oldest if full)
	a.pendingHeaders.Add(h.Height(), h)
	a.height.Store(h.Height())
	a.initialized = true

	return nil
}

// Sync ensures all pending writes are flushed.
// No-op for the adapter as pending data is in-memory cache.
func (a *HeaderStoreAdapter) Sync(ctx context.Context) error {
	return nil
}

// DeleteRange deletes headers in the range [from, to).
// This is used for rollback operations.
func (a *HeaderStoreAdapter) DeleteRange(ctx context.Context, from, to uint64) error {
	// Remove from pending cache
	for height := from; height < to; height++ {
		a.pendingHeaders.Remove(height)

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

// OnDelete registers a callback to be invoked when headers are deleted.
func (a *HeaderStoreAdapter) OnDelete(fn func(context.Context, uint64) error) {
	a.onDeleteFn = fn
}
