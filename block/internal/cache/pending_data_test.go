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

	// three blocks with transactions
	chainID := "pd-basic"
	h1, d1 := types.GetRandomBlock(1, 1, chainID)
	h2, d2 := types.GetRandomBlock(2, 1, chainID)
	h3, d3 := types.GetRandomBlock(3, 1, chainID)

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

	// initially all 3 data items are pending
	require.Equal(t, uint64(3), pendingData.NumPendingData())
	pendingDataList, _, err := pendingData.GetPendingData(ctx)
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
	pendingDataList, _, err = pendingData.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, pendingDataList, 2)
	require.Equal(t, uint64(2), pendingDataList[0].Height())
}

func TestPendingData_AdvancesPastEmptyData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := memStore(t)

	// Create blocks: non-empty, empty, empty, non-empty
	chainID := "pd-empty"
	h1, d1 := types.GetRandomBlock(1, 1, chainID) // 1 tx
	h2, d2 := types.GetRandomBlock(2, 0, chainID) // empty
	h3, d3 := types.GetRandomBlock(3, 0, chainID) // empty
	h4, d4 := types.GetRandomBlock(4, 1, chainID) // 1 tx

	for i, p := range []struct {
		h *types.SignedHeader
		d *types.Data
	}{{h1, d1}, {h2, d2}, {h3, d3}, {h4, d4}} {
		batch, err := store.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(p.h, p.d, &types.Signature{}))
		require.NoError(t, batch.SetHeight(uint64(i+1)))
		require.NoError(t, batch.Commit())
	}

	pendingData, err := NewPendingData(store, zerolog.Nop())
	require.NoError(t, err)

	// Initially 4 pending - height 1 is non-empty so no advancing happens yet
	require.Equal(t, uint64(4), pendingData.NumPendingData())
	require.Equal(t, uint64(0), pendingData.GetLastSubmittedDataHeight())

	// Submit height 1 (non-empty)
	pendingData.SetLastSubmittedDataHeight(ctx, 1)
	require.Equal(t, uint64(1), pendingData.GetLastSubmittedDataHeight())

	// NumPendingData advances past empty blocks 2 and 3, leaving only height 4
	require.Equal(t, uint64(1), pendingData.NumPendingData())

	// Should have advanced to height 3 (past empty blocks 2 and 3)
	require.Equal(t, uint64(3), pendingData.GetLastSubmittedDataHeight())

	pendingDataList, _, err := pendingData.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, pendingDataList, 1)
	require.Equal(t, uint64(4), pendingDataList[0].Height())
}

func TestPendingData_AdvancesPastAllEmptyToEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := memStore(t)

	// Create blocks: non-empty, empty, empty (all remaining are empty)
	chainID := "pd-all-empty"
	h1, d1 := types.GetRandomBlock(1, 1, chainID) // 1 tx
	h2, d2 := types.GetRandomBlock(2, 0, chainID) // empty
	h3, d3 := types.GetRandomBlock(3, 0, chainID) // empty

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

	// Submit height 1
	pendingData.SetLastSubmittedDataHeight(ctx, 1)
	require.Equal(t, uint64(1), pendingData.GetLastSubmittedDataHeight())

	// NumPendingData advances past empty blocks to end of store
	require.Equal(t, uint64(0), pendingData.NumPendingData())

	// Should have advanced to height 3
	require.Equal(t, uint64(3), pendingData.GetLastSubmittedDataHeight())
}

func TestPendingData_AdvancesPastEmptyAtStart(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := memStore(t)

	// Create blocks: empty, empty, non-empty
	chainID := "pd-empty-start"
	h1, d1 := types.GetRandomBlock(1, 0, chainID) // empty
	h2, d2 := types.GetRandomBlock(2, 0, chainID) // empty
	h3, d3 := types.GetRandomBlock(3, 1, chainID) // 1 tx

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

	// NumPendingData should advance past empty blocks 1 and 2, leaving only height 3
	require.Equal(t, uint64(1), pendingData.NumPendingData())

	// Should have advanced to height 2 (past empty blocks 1 and 2)
	require.Equal(t, uint64(2), pendingData.GetLastSubmittedDataHeight())

	pendingDataList, _, err := pendingData.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, pendingDataList, 1)
	require.Equal(t, uint64(3), pendingDataList[0].Height())
}

func TestPendingData_InitFromMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ds, err := store.NewTestInMemoryKVStore()
	require.NoError(t, err)
	store := store.New(ds)

	// set metadata to last submitted = 2 before creating PendingData
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, 2)
	require.NoError(t, store.SetMetadata(ctx, LastSubmittedDataHeightKey, bz))

	// store height is 3, with a non-empty block at height 3
	h3, d3 := types.GetRandomBlock(3, 1, "test-chain")
	batch, err := store.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SaveBlockData(h3, d3, &types.Signature{}))
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
	pending, _, err := pendingData.GetPendingData(ctx)
	require.Error(t, err)
	require.Nil(t, pending)
}
