package evm

import (
	"context"
	"testing"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/stretchr/testify/require"
)

func TestSaveExecMetaPrunesHistory(t *testing.T) {
	t.Parallel()

	store := NewEVMStore(dssync.MutexWrap(ds.NewMapDatastore()))
	store.SetExecMetaRetention(2)

	ctx := context.Background()
	for height := uint64(1); height <= 3; height++ {
		require.NoError(t, store.SaveExecMeta(ctx, &ExecMeta{
			Height: height,
			Stage:  ExecStageStarted,
		}))
	}

	meta, err := store.GetExecMeta(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, meta)

	meta, err = store.GetExecMeta(ctx, 2)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, uint64(2), meta.Height)

	meta, err = store.GetExecMeta(ctx, 3)
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.Equal(t, uint64(3), meta.Height)
}
