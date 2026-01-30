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

// DataStoreAdapter wraps Store to implement header.Store[*types.Data].
// This allows the ev-node store to be used directly by go-header's P2P infrastructure,
// eliminating the need for a separate go-header store and reducing data duplication.
type DataStoreAdapter struct {
	store Store

	// height caches the current height to avoid repeated context-based lookups.
	// Updated on successful reads and writes.
	height atomic.Uint64

	// mu protects initialization state
	mu          sync.RWMutex
	initialized bool

	// onDeleteFn is called when data is deleted (for rollback scenarios)
	onDeleteFn func(context.Context, uint64) error
}

// Compile-time check that DataStoreAdapter implements header.Store
var _ header.Store[*types.Data] = (*DataStoreAdapter)(nil)

// NewDataStoreAdapter creates a new DataStoreAdapter wrapping the given store.
func NewDataStoreAdapter(store Store) *DataStoreAdapter {
	adapter := &DataStoreAdapter{
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

	_, data, err := a.store.GetBlockData(ctx, height)
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
		return nil, header.ErrNotFound
	}

	// Try height 1 first (most common case)
	_, data, err := a.store.GetBlockData(ctx, 1)
	if err == nil {
		return data, nil
	}

	// Linear scan from 1 to current height to find first data
	for h := uint64(2); h <= height; h++ {
		_, data, err = a.store.GetBlockData(ctx, h)
		if err == nil {
			return data, nil
		}
	}

	return nil, header.ErrNotFound
}

// Get returns data by its hash.
func (a *DataStoreAdapter) Get(ctx context.Context, hash header.Hash) (*types.Data, error) {
	_, data, err := a.store.GetBlockByHash(ctx, hash)
	if err != nil {
		return nil, header.ErrNotFound
	}
	return data, nil
}

// GetByHeight returns data at the given height.
func (a *DataStoreAdapter) GetByHeight(ctx context.Context, height uint64) (*types.Data, error) {
	_, data, err := a.store.GetBlockData(ctx, height)
	if err != nil {
		return nil, header.ErrNotFound
	}
	return data, nil
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
		_, data, err := a.store.GetBlockData(ctx, height)
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
	_, _, err := a.store.GetBlockByHash(ctx, hash)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// HasAt checks if data exists at the given height.
func (a *DataStoreAdapter) HasAt(ctx context.Context, height uint64) bool {
	_, _, err := a.store.GetBlockData(ctx, height)
	return err == nil
}

// Height returns the current height of the store.
func (a *DataStoreAdapter) Height() uint64 {
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

// Append stores data in the store.
// This method is called by go-header's P2P infrastructure when data is received.
// We save the data to the ev-node store to ensure it's available for the syncer.
func (a *DataStoreAdapter) Append(ctx context.Context, dataList ...*types.Data) error {
	if len(dataList) == 0 {
		return nil
	}

	for _, data := range dataList {
		if data == nil || data.IsZero() {
			continue
		}

		// Check if we already have this data
		if a.HasAt(ctx, data.Height()) {
			continue
		}

		// Create a batch and save the data
		// Note: We create a minimal header since we only have the data at this point.
		// The full block will be saved by the syncer when processing from DA.
		batch, err := a.store.NewBatch(ctx)
		if err != nil {
			return fmt.Errorf("failed to create batch for data at height %d: %w", data.Height(), err)
		}

		// Create a minimal header for the data
		// The syncer will overwrite this with complete block data when it processes from DA
		minimalHeader := &types.SignedHeader{
			Header: types.Header{
				BaseHeader: types.BaseHeader{
					ChainID: data.ChainID(),
					Height:  data.Height(),
					Time:    uint64(data.Time().UnixNano()),
				},
				LastHeaderHash: data.LastHeader(),
				DataHash:       data.DACommitment(),
			},
		}

		if err := batch.SaveBlockData(minimalHeader, data, &types.Signature{}); err != nil {
			return fmt.Errorf("failed to save data at height %d: %w", data.Height(), err)
		}

		if err := batch.SetHeight(data.Height()); err != nil {
			return fmt.Errorf("failed to set height for data at height %d: %w", data.Height(), err)
		}

		if err := batch.Commit(); err != nil {
			return fmt.Errorf("failed to commit data at height %d: %w", data.Height(), err)
		}

		// Update cached height
		if data.Height() > a.height.Load() {
			a.height.Store(data.Height())
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

	// Use Append to save the data
	a.mu.Unlock() // Unlock before calling Append to avoid deadlock
	err := a.Append(ctx, d)
	a.mu.Lock() // Re-lock for the initialized flag update

	if err != nil {
		return err
	}

	a.initialized = true
	return nil
}

// Sync ensures all pending writes are flushed.
// Delegates to the underlying store's sync if available.
func (a *DataStoreAdapter) Sync(ctx context.Context) error {
	// The underlying store handles its own syncing
	return nil
}

// DeleteRange deletes data in the range [from, to).
// This is used for rollback operations.
func (a *DataStoreAdapter) DeleteRange(ctx context.Context, from, to uint64) error {
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
	return nil
}

// SetHeight updates the cached height.
// This is useful when the syncer knows the new height after processing a block.
func (a *DataStoreAdapter) SetHeight(height uint64) {
	a.height.Store(height)
}
