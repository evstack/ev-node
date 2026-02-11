package pruner

import (
	"context"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type execMetaAdapter struct {
	existing map[uint64]struct{}
}

func (e *execMetaAdapter) PruneExecMeta(ctx context.Context, height uint64) error {
	delete(e.existing, height)
	return nil
}

func TestPrunerPrunesRecoveryHistory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	kv := dssync.MutexWrap(ds.NewMapDatastore())
	stateStore := store.New(kv)

	for height := uint64(1); height <= 3; height++ {
		batch, err := stateStore.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SetHeight(height))
		require.NoError(t, batch.UpdateState(types.State{LastBlockHeight: height}))
		require.NoError(t, batch.Commit())
	}

	execAdapter := &execMetaAdapter{existing: map[uint64]struct{}{1: {}, 2: {}, 3: {}}}

	recoveryPruner := New(stateStore, execAdapter, 2, time.Minute, zerolog.Nop())
	require.NoError(t, recoveryPruner.pruneOnce(ctx))

	_, err := stateStore.GetStateAtHeight(ctx, 1)
	require.ErrorIs(t, err, ds.ErrNotFound)

	_, exists := execAdapter.existing[1]
	require.False(t, exists)
}
