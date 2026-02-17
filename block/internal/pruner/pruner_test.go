package pruner

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type execMetaAdapter struct {
	existing map[uint64]struct{}
}

func (e *execMetaAdapter) PruneExec(ctx context.Context, height uint64) error {
	for h := range e.existing {
		if h < height {
			delete(e.existing, h)
		}
	}

	return nil
}

func TestPrunerPruneMetadata(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	kv := dssync.MutexWrap(ds.NewMapDatastore())
	stateStore := store.New(kv)

	for height := uint64(1); height <= 5; height++ {
		batch, err := stateStore.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SetHeight(height))
		require.NoError(t, batch.UpdateState(types.State{LastBlockHeight: height}))
		require.NoError(t, batch.Commit())
	}

	execAdapter := &execMetaAdapter{existing: map[uint64]struct{}{1: {}, 2: {}, 3: {}}}
	cfg := config.PruningConfig{
		Mode:       config.PruningModeMetadata,
		Interval:   config.DurationWrapper{Duration: 1 * time.Second},
		KeepRecent: 1,
	}

	pruner := New(zerolog.New(zerolog.NewTestWriter(t)), stateStore, execAdapter, cfg, 100*time.Millisecond, "") // Empty DA address
	require.NoError(t, pruner.pruneMetadata())

	_, err := stateStore.GetStateAtHeight(ctx, 1)
	require.ErrorIs(t, err, ds.ErrNotFound)

	_, err = stateStore.GetStateAtHeight(ctx, 5)
	require.NoError(t, err)

	_, exists := execAdapter.existing[1]
	require.False(t, exists)
}

func TestPrunerPruneBlocksWithoutDA(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	kv := dssync.MutexWrap(ds.NewMapDatastore())
	stateStore := store.New(kv)

	// Create blocks without setting DAIncludedHeightKey (simulating node without DA)
	for height := uint64(1); height <= 100; height++ {
		header := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: height}}}
		data := &types.Data{}
		sig := types.Signature([]byte{byte(height)})

		batch, err := stateStore.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(header, data, &sig))
		require.NoError(t, batch.SetHeight(height))
		require.NoError(t, batch.UpdateState(types.State{LastBlockHeight: height}))
		require.NoError(t, batch.Commit())
	}

	execAdapter := &execMetaAdapter{existing: make(map[uint64]struct{})}
	for h := uint64(1); h <= 100; h++ {
		execAdapter.existing[h] = struct{}{}
	}

	// Test with empty DA address (DA disabled) - should prune successfully
	cfg := config.PruningConfig{
		Mode:       config.PruningModeAll,
		Interval:   config.DurationWrapper{Duration: 1 * time.Second},
		KeepRecent: 10,
	}

	pruner := New(zerolog.New(zerolog.NewTestWriter(t)), stateStore, execAdapter, cfg, 100*time.Millisecond, "") // Empty DA address = DA disabled
	require.NoError(t, pruner.pruneBlocks())

	// Verify blocks were pruned (batch size is 40 blocks: 1s interval / 100ms block time * 4)
	// So we expect to prune from height 1 up to min(0 + 40, 90) = 40
	height, err := stateStore.Height(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), height)

	// Verify old blocks were pruned (up to height 40)
	for h := uint64(1); h <= 40; h++ {
		_, _, err := stateStore.GetBlockData(ctx, h)
		require.Error(t, err, "expected block data at height %d to be pruned", h)
	}

	// Verify blocks after batch were kept
	for h := uint64(41); h <= 100; h++ {
		_, _, err := stateStore.GetBlockData(ctx, h)
		require.NoError(t, err, "expected block data at height %d to be kept", h)
	}

	// Verify exec metadata was also pruned (strictly less than 40)
	for h := uint64(1); h < 40; h++ {
		_, exists := execAdapter.existing[h]
		require.False(t, exists, "expected exec metadata at height %d to be pruned", h)
	}
}

func TestPrunerPruneBlocksWithDAEnabled(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	kv := dssync.MutexWrap(ds.NewMapDatastore())
	stateStore := store.New(kv)

	// Create blocks without setting DAIncludedHeightKey
	for height := uint64(1); height <= 100; height++ {
		header := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: height}}}
		data := &types.Data{}
		sig := types.Signature([]byte{byte(height)})

		batch, err := stateStore.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(header, data, &sig))
		require.NoError(t, batch.SetHeight(height))
		require.NoError(t, batch.UpdateState(types.State{LastBlockHeight: height}))
		require.NoError(t, batch.Commit())
	}

	// Test with DA address provided (DA enabled) - should skip pruning when DA height is not available
	cfg := config.PruningConfig{
		Mode:       config.PruningModeAll,
		Interval:   config.DurationWrapper{Duration: 1 * time.Second},
		KeepRecent: 10,
	}

	pruner := New(zerolog.New(zerolog.NewTestWriter(t)), stateStore, nil, cfg, 100*time.Millisecond, "localhost:1234") // DA enabled
	// Should return nil (skip pruning) since DA height is not available
	require.NoError(t, pruner.pruneBlocks())

	// Verify no blocks were pruned (all blocks should still be retrievable)
	for h := uint64(1); h <= 100; h++ {
		_, _, err := stateStore.GetBlockData(ctx, h)
		require.NoError(t, err, "expected block data at height %d to still exist (no pruning should have happened)", h)
	}
}

// TestPrunerPruneBlocksWithDAWhenStoreHeightLessThanDAHeight is a regression test that verifies
// pruning behavior when DA is enabled and store height is less than DA inclusion height.
// This ensures the pruner uses min(storeHeight, daInclusionHeight) as the upper bound.
func TestPrunerPruneBlocksWithDAWhenStoreHeightLessThanDAHeight(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	kv := dssync.MutexWrap(ds.NewMapDatastore())
	stateStore := store.New(kv)

	// Create blocks up to height 100
	for height := uint64(1); height <= 100; height++ {
		header := &types.SignedHeader{Header: types.Header{BaseHeader: types.BaseHeader{Height: height}}}
		data := &types.Data{}
		sig := types.Signature([]byte{byte(height)})

		batch, err := stateStore.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(header, data, &sig))
		require.NoError(t, batch.SetHeight(height))
		require.NoError(t, batch.UpdateState(types.State{LastBlockHeight: height}))
		require.NoError(t, batch.Commit())
	}

	// Set DA inclusion height to 150 (higher than store height of 100)
	// This simulates the case where DA has confirmed blocks beyond what's in the local store
	daInclusionHeight := uint64(150)
	daInclusionHeightBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(daInclusionHeightBz, daInclusionHeight)
	require.NoError(t, stateStore.SetMetadata(ctx, store.DAIncludedHeightKey, daInclusionHeightBz))

	execAdapter := &execMetaAdapter{existing: make(map[uint64]struct{})}
	for h := uint64(1); h <= 100; h++ {
		execAdapter.existing[h] = struct{}{}
	}

	// Test with DA enabled - should use min(storeHeight=100, daInclusionHeight=150) = 100
	cfg := config.PruningConfig{
		Mode:       config.PruningModeAll,
		Interval:   config.DurationWrapper{Duration: 1 * time.Second},
		KeepRecent: 10,
	}

	pruner := New(zerolog.New(zerolog.NewTestWriter(t)), stateStore, execAdapter, cfg, 100*time.Millisecond, "localhost:1234") // DA enabled
	require.NoError(t, pruner.pruneBlocks())

	// Verify blocks were pruned correctly
	// Upper bound should be min(100, 150) = 100
	// Target height = 100 - 10 (KeepRecent) = 90
	// Batch size = 40 blocks (1s interval / 100ms block time * 4)
	// So we expect to prune from height 1 up to min(0 + 40, 90) = 40

	height, err := stateStore.Height(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), height)

	// Verify old blocks were pruned (up to height 40)
	for h := uint64(1); h <= 40; h++ {
		_, _, err := stateStore.GetBlockData(ctx, h)
		require.Error(t, err, "expected block data at height %d to be pruned", h)
	}

	// Verify blocks after batch were kept
	for h := uint64(41); h <= 100; h++ {
		_, _, err := stateStore.GetBlockData(ctx, h)
		require.NoError(t, err, "expected block data at height %d to be kept", h)
	}

	// Verify exec metadata was also pruned (strictly less than 40)
	for h := uint64(1); h < 40; h++ {
		_, exists := execAdapter.existing[h]
		require.False(t, exists, "expected exec metadata at height %d to be pruned", h)
	}
}
