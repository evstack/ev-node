package pruner

import (
	"context"
	"encoding/binary"
	"errors"
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
	pruner.ctx = ctx
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
	pruner.ctx = ctx
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
		_, err = stateStore.GetStateAtHeight(ctx, h)
		require.ErrorIs(t, err, ds.ErrNotFound)
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
	pruner.ctx = ctx
	// Should return nil (skip pruning) since DA height is not available
	require.NoError(t, pruner.pruneBlocks())

	// Verify no blocks were pruned (all blocks should still be retrievable)
	for h := uint64(1); h <= 100; h++ {
		_, _, err := stateStore.GetBlockData(ctx, h)
		require.NoError(t, err, "expected block data at height %d to still exist (no pruning should have happened)", h)
	}
}

// recordingStore verifies the batch boundaries without allocating thousands of blocks.
type recordingStore struct {
	store.Store
	blockBatches []uint64
	states       []uint64
}

func (s *recordingStore) PruneBlocks(ctx context.Context, height uint64) error {
	s.blockBatches = append(s.blockBatches, height)
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, height)
	return s.SetMetadata(ctx, store.LastPrunedBlockHeightKey, bz)
}

func (s *recordingStore) DeleteStateAtHeight(_ context.Context, height uint64) error {
	s.states = append(s.states, height)
	return nil
}

func TestPrunerCatchupUsesBoundedBatches(t *testing.T) {
	ctx := t.Context()
	st := &recordingStore{Store: store.New(dssync.MutexWrap(ds.NewMapDatastore()))}
	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(100000))
	require.NoError(t, batch.Commit())
	p := New(zerolog.Nop(), st, nil, config.PruningConfig{
		Mode: config.PruningModeAll, KeepRecent: 10,
		Interval: config.DurationWrapper{Duration: 15 * time.Minute},
	}, 100*time.Millisecond, "")
	p.ctx = ctx
	require.NoError(t, p.pruneBlocks())
	require.Equal(t, []uint64{10000, 20000, 30000, 36000}, st.blockBatches)
	require.Len(t, st.states, 36000)

	// The chain grows by 9000 blocks per interval; pruning must exceed that.
	batch, err = st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(109000))
	require.NoError(t, batch.Commit())
	require.NoError(t, p.pruneBlocks())
	require.Equal(t, []uint64{10000, 20000, 30000, 36000, 46000, 56000, 66000, 72000}, st.blockBatches)
	require.Len(t, st.states, 72000)
}

func TestPrunerRecoversSnapshotsAfterBlockOnlyPruning(t *testing.T) {
	for _, mode := range []string{config.PruningModeAll, config.PruningModeMetadata} {
		t.Run(mode, func(t *testing.T) {
			ctx := t.Context()
			st := store.New(dssync.MutexWrap(ds.NewMapDatastore()))
			for h := uint64(1); h <= 5; h++ {
				batch, err := st.NewBatch(ctx)
				require.NoError(t, err)
				require.NoError(t, batch.SetHeight(h))
				require.NoError(t, batch.UpdateState(types.State{LastBlockHeight: h}))
				require.NoError(t, batch.Commit())
			}
			require.NoError(t, st.PruneBlocks(ctx, 4))
			p := New(zerolog.Nop(), st, nil, config.PruningConfig{
				Mode: mode, KeepRecent: 1,
				Interval: config.DurationWrapper{Duration: time.Second},
			}, 100*time.Millisecond, "")
			p.ctx = ctx
			if mode == config.PruningModeAll {
				require.NoError(t, p.pruneBlocks())
			} else {
				require.NoError(t, p.pruneMetadata())
			}
			for h := uint64(1); h <= 4; h++ {
				_, err := st.GetStateAtHeight(ctx, h)
				require.ErrorIs(t, err, ds.ErrNotFound)
			}
			state, err := st.GetState(ctx)
			require.NoError(t, err)
			require.Equal(t, uint64(5), state.LastBlockHeight)
			_, err = st.GetStateAtHeight(ctx, 5)
			require.NoError(t, err)
		})
	}
}

type failingExecPruner struct {
	err   error
	calls []uint64
}

func (e *failingExecPruner) PruneExec(_ context.Context, height uint64) error {
	e.calls = append(e.calls, height)
	return e.err
}

func TestPrunerRetriesExecutionFailure(t *testing.T) {
	ctx := t.Context()
	st := &recordingStore{Store: store.New(dssync.MutexWrap(ds.NewMapDatastore()))}
	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(5))
	require.NoError(t, batch.Commit())
	exec := &failingExecPruner{err: errors.New("execution pruning failed")}
	cfg := config.PruningConfig{Mode: config.PruningModeAll, KeepRecent: 1,
		Interval: config.DurationWrapper{Duration: time.Second}}
	p := New(zerolog.Nop(), st, exec, cfg, 100*time.Millisecond, "")
	p.ctx = ctx
	require.ErrorIs(t, p.pruneBlocks(), exec.err)
	_, err = st.GetMetadata(ctx, store.LastPrunedStateHeightKey)
	require.ErrorIs(t, err, ds.ErrNotFound)

	// A restarted pruner must retry execution despite the advanced block cursor.
	exec.err = nil
	p = New(zerolog.Nop(), st, exec, cfg, 100*time.Millisecond, "")
	p.ctx = ctx
	require.NoError(t, p.pruneBlocks())
	require.Equal(t, []uint64{4, 4}, exec.calls)
	require.Equal(t, []uint64{4}, st.blockBatches)
}

func TestPrunerRespectsDAInclusionBoundary(t *testing.T) {
	ctx := t.Context()
	st := &recordingStore{Store: store.New(dssync.MutexWrap(ds.NewMapDatastore()))}
	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(100))
	require.NoError(t, batch.Commit())
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, 20)
	require.NoError(t, st.SetMetadata(ctx, store.DAIncludedHeightKey, bz))
	p := New(zerolog.Nop(), st, nil, config.PruningConfig{
		Mode: config.PruningModeAll, KeepRecent: 10,
		Interval: config.DurationWrapper{Duration: time.Second},
	}, 100*time.Millisecond, "da")
	p.ctx = ctx
	require.NoError(t, p.pruneBlocks())
	require.Equal(t, []uint64{10}, st.blockBatches)
	require.Equal(t, uint64(10), st.states[len(st.states)-1])
}
