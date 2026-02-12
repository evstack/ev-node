package evm

import (
	"context"
	"encoding/binary"
	"testing"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/require"
)

func TestDeleteExecMeta(t *testing.T) {
	t.Parallel()

	store := NewEVMStore(dssync.MutexWrap(ds.NewMapDatastore()))

	ctx := context.Background()
	require.NoError(t, store.SaveExecMeta(ctx, &ExecMeta{
		Height: 1,
		Stage:  ExecStageStarted,
	}))

	require.NoError(t, store.DeleteExecMeta(ctx, 1))

	meta, err := store.GetExecMeta(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, meta)
}

// newTestDatastore creates an in-memory datastore for testing.
func newTestDatastore(t *testing.T) ds.Batching {
	t.Helper()
	// Wrap the in-memory MapDatastore to satisfy the Batching interface.
	return dssync.MutexWrap(ds.NewMapDatastore())
}

func TestPruneExec_PrunesUpToTargetHeight(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestDatastore(t)
	store := NewEVMStore(db)

	// Seed ExecMeta entries at heights 1..5
	for h := uint64(1); h <= 5; h++ {
		meta := &ExecMeta{Height: h}
		require.NoError(t, store.SaveExecMeta(ctx, meta))
	}

	// Sanity: all heights should be present
	for h := uint64(1); h <= 5; h++ {
		meta, err := store.GetExecMeta(ctx, h)
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Equal(t, h, meta.Height)
	}

	// Prune up to height 3
	require.NoError(t, store.PruneExec(ctx, 3))

	// Heights 1..3 should be gone
	for h := uint64(1); h <= 3; h++ {
		meta, err := store.GetExecMeta(ctx, h)
		require.NoError(t, err)
		require.Nil(t, meta)
	}

	// Heights 4..5 should remain
	for h := uint64(4); h <= 5; h++ {
		meta, err := store.GetExecMeta(ctx, h)
		require.NoError(t, err)
		require.NotNil(t, meta)
	}

	// Re-pruning with the same height should be a no-op
	require.NoError(t, store.PruneExec(ctx, 3))
}

func TestPruneExec_TracksLastPrunedHeight(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestDatastore(t)
	store := NewEVMStore(db)

	// Seed ExecMeta entries at heights 1..5
	for h := uint64(1); h <= 5; h++ {
		meta := &ExecMeta{Height: h}
		require.NoError(t, store.SaveExecMeta(ctx, meta))
	}

	// First prune up to 2
	require.NoError(t, store.PruneExec(ctx, 2))

	// Then prune up to 4; heights 3..4 should be deleted in this run
	require.NoError(t, store.PruneExec(ctx, 4))

	// Verify all heights 1..4 are gone, 5 remains
	for h := uint64(1); h <= 4; h++ {
		meta, err := store.GetExecMeta(ctx, h)
		require.NoError(t, err)
		require.Nil(t, meta)
	}

	meta, err := store.GetExecMeta(ctx, 5)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, uint64(5), meta.Height)

	// Ensure last-pruned marker is set to 4
	raw, err := db.Get(ctx, ds.NewKey(lastPrunedExecMetaKey))
	require.NoError(t, err)
	require.Len(t, raw, 8)
	last := binary.BigEndian.Uint64(raw)
	require.Equal(t, uint64(4), last)
}
