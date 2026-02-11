package evm

import (
	"context"
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
