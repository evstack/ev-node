package cache

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func TestPendingData_BasicFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := memStore(t)

	// three blocks; PendingData should return all data (no filtering here)
	chainID := "pd-basic"
	h1, d1 := types.GetRandomBlock(1, 0, chainID)
	h2, d2 := types.GetRandomBlock(2, 1, chainID)
	h3, d3 := types.GetRandomBlock(3, 2, chainID)

	for i, p := range []struct {
		h *types.SignedHeader
		d *types.Data
	}{{h1, d1}, {h2, d2}, {h3, d3}} {
		batch, err := store.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(p.h, p.d, &types.Signature{}))
		require.NoError(t, batch.SetHeight(uint64(i+1)))
		require.NoError(t, batch.Commit())
	}

	pendingData, err := NewPendingData(store, zerolog.Nop())
	require.NoError(t, err)

	// initially all 3 data items are pending, incl. empty
	require.Equal(t, uint64(3), pendingData.NumPendingData())
	pendingDataList, err := pendingData.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, pendingDataList, 3)
	require.Equal(t, uint64(1), pendingDataList[0].Height())
	require.Equal(t, uint64(3), pendingDataList[2].Height())

	// set last submitted and verify persistence
	pendingData.SetLastSubmittedDataHeight(ctx, 1)
	metadataRaw, err := store.GetMetadata(ctx, LastSubmittedDataHeightKey)
	require.NoError(t, err)
	require.Len(t, metadataRaw, 8)
	require.Equal(t, uint64(1), binary.LittleEndian.Uint64(metadataRaw))

	require.Equal(t, uint64(2), pendingData.NumPendingData())
	pendingDataList, err = pendingData.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, pendingDataList, 2)
	require.Equal(t, uint64(2), pendingDataList[0].Height())
}

func TestPendingData_InitFromMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ds, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	store := store.New(ds)

	// set metadata to last submitted = 2 before creating PendingData
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, 2)
	require.NoError(t, store.SetMetadata(ctx, LastSubmittedDataHeightKey, bz))

	// store height is 3
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(3))
	require.NoError(t, batch.Commit())

	pendingData, err := NewPendingData(store, zerolog.Nop())
	require.NoError(t, err)
	require.Equal(t, uint64(1), pendingData.NumPendingData())
}

func TestPendingData_GetPending_PropagatesFetchError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := memStore(t)

	// Set height to 1 but do not save any block data
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(1))
	require.NoError(t, batch.Commit())

	pendingData, err := NewPendingData(store, zerolog.Nop())
	require.NoError(t, err)

	// fetching pending should propagate the not-found error from store
	pending, err := pendingData.GetPendingData(ctx)
	require.Error(t, err)
	require.Empty(t, pending)
}
