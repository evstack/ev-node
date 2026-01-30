package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/celestiaorg/go-header"

	"github.com/evstack/ev-node/types"
)

// HeaderStoreAdapter wraps Store to implement header.Store[*types.SignedHeader].
// This allows the ev-node store to be used directly by go-header's P2P infrastructure,
// eliminating the need for a separate go-header store and reducing data duplication.
type HeaderStoreAdapter struct {
	store Store

	// height caches the current height to avoid repeated context-based lookups.
	// Updated on successful reads and writes.
	height atomic.Uint64

	// mu protects initialization state
	mu          sync.RWMutex
	initialized bool

	// onDeleteFn is called when headers are deleted (for rollback scenarios)
	onDeleteFn func(context.Context, uint64) error
}

// Compile-time check that HeaderStoreAdapter implements header.Store
var _ header.Store[*types.SignedHeader] = (*HeaderStoreAdapter)(nil)

// NewHeaderStoreAdapter creates a new HeaderStoreAdapter wrapping the given store.
func NewHeaderStoreAdapter(store Store) *HeaderStoreAdapter {
	adapter := &HeaderStoreAdapter{
		store: store,
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
	height := a.height.Load()
	if height == 0 {
		// Try to refresh from store
		h, err := a.store.Height(ctx)
		if err != nil {
			return nil, header.ErrNotFound
		}
		if h == 0 {
			return nil, header.ErrNotFound
		}
		a.height.Store(h)
		height = h
	}

	hdr, err := a.store.GetHeader(ctx, height)
	if err != nil {
		return nil, header.ErrNotFound
	}

	return hdr, nil
}

// Tail returns the lowest header in the store.
// For ev-node, this is typically the genesis/initial height.
func (a *HeaderStoreAdapter) Tail(ctx context.Context) (*types.SignedHeader, error) {
	// Start from height 1 and find the first available header
	// This is a simple implementation; could be optimized with metadata
	height := a.height.Load()
	if height == 0 {
		return nil, header.ErrNotFound
	}

	// Try height 1 first (most common case)
	hdr, err := a.store.GetHeader(ctx, 1)
	if err == nil {
		return hdr, nil
	}

	// Linear scan from 1 to current height to find first header
	for h := uint64(2); h <= height; h++ {
		hdr, err = a.store.GetHeader(ctx, h)
		if err == nil {
			return hdr, nil
		}
	}

	return nil, header.ErrNotFound
}

// Get returns a header by its hash.
func (a *HeaderStoreAdapter) Get(ctx context.Context, hash header.Hash) (*types.SignedHeader, error) {
	hdr, _, err := a.store.GetBlockByHash(ctx, hash)
	if err != nil {
		return nil, header.ErrNotFound
	}
	return hdr, nil
}

// GetByHeight returns a header at the given height.
func (a *HeaderStoreAdapter) GetByHeight(ctx context.Context, height uint64) (*types.SignedHeader, error) {
	hdr, err := a.store.GetHeader(ctx, height)
	if err != nil {
		return nil, header.ErrNotFound
	}
	return hdr, nil
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
		hdr, err := a.store.GetHeader(ctx, height)
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
	_, _, err := a.store.GetBlockByHash(ctx, hash)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// HasAt checks if a header exists at the given height.
func (a *HeaderStoreAdapter) HasAt(ctx context.Context, height uint64) bool {
	_, err := a.store.GetHeader(ctx, height)
	return err == nil
}

// Height returns the current height of the store.
func (a *HeaderStoreAdapter) Height() uint64 {
	height := a.height.Load()
	if height == 0 {
		// Try to refresh from store
		if h, err := a.store.Height(context.Background()); err == nil {
			a.height.Store(h)
			return h
		}
	}
	return height
}

// Append stores headers in the store.
// This method is called by go-header's P2P infrastructure when headers are received.
// We save the headers to the ev-node store to ensure they're available for the syncer.
func (a *HeaderStoreAdapter) Append(ctx context.Context, headers ...*types.SignedHeader) error {
	if len(headers) == 0 {
		return nil
	}

	for _, hdr := range headers {
		if hdr == nil || hdr.IsZero() {
			continue
		}

		// Check if we already have this header
		if a.HasAt(ctx, hdr.Height()) {
			continue
		}

		// Create a batch and save the header
		// Note: We create empty data since we only have the header at this point.
		// The full block data will be saved by the syncer when processing from DA.
		batch, err := a.store.NewBatch(ctx)
		if err != nil {
			return fmt.Errorf("failed to create batch for header at height %d: %w", hdr.Height(), err)
		}

		// Save header with empty data and signature
		// The syncer will overwrite this with complete block data when it processes from DA
		emptyData := &types.Data{
			Metadata: &types.Metadata{
				ChainID:      hdr.ChainID(),
				Height:       hdr.Height(),
				Time:         uint64(hdr.Time().UnixNano()),
				LastDataHash: hdr.LastHeader(),
			},
			Txs: nil,
		}

		if err := batch.SaveBlockData(hdr, emptyData, &hdr.Signature); err != nil {
			return fmt.Errorf("failed to save header at height %d: %w", hdr.Height(), err)
		}

		if err := batch.SetHeight(hdr.Height()); err != nil {
			return fmt.Errorf("failed to set height for header at height %d: %w", hdr.Height(), err)
		}

		if err := batch.Commit(); err != nil {
			return fmt.Errorf("failed to commit header at height %d: %w", hdr.Height(), err)
		}

		// Update cached height
		if hdr.Height() > a.height.Load() {
			a.height.Store(hdr.Height())
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

	// Use Append to save the header
	a.mu.Unlock() // Unlock before calling Append to avoid deadlock
	err := a.Append(ctx, h)
	a.mu.Lock() // Re-lock for the initialized flag update

	if err != nil {
		return err
	}

	a.initialized = true
	return nil
}

// Sync ensures all pending writes are flushed.
// Delegates to the underlying store's sync if available.
func (a *HeaderStoreAdapter) Sync(ctx context.Context) error {
	// The underlying store handles its own syncing
	return nil
}

// DeleteRange deletes headers in the range [from, to).
// This is used for rollback operations.
func (a *HeaderStoreAdapter) DeleteRange(ctx context.Context, from, to uint64) error {
	// Rollback is handled by the ev-node store's Rollback method
	// This is called during store cleanup operations
	if a.onDeleteFn != nil {
		for height := from; height < to; height++ {
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

// RefreshHeight updates the cached height from the underlying store.
// This should be called after the syncer processes a new block.
func (a *HeaderStoreAdapter) RefreshHeight(ctx context.Context) error {
	h, err := a.store.Height(ctx)
	if err != nil {
		return err
	}
	a.height.Store(h)
	return nil
}

// SetHeight updates the cached height.
// This is useful when the syncer knows the new height after processing a block.
func (a *HeaderStoreAdapter) SetHeight(height uint64) {
	a.height.Store(height)
}
