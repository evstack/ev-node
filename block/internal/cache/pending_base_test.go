package cache

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func TestPendingHeadersAndData_Flow(t *testing.T) {
	t.Parallel()
	st := memStore(t)
	ctx := context.Background()
	logger := zerolog.Nop()

	// create 3 blocks, with varying number of txs to test data filtering
	chainID := "chain-pending"
	h1, d1 := types.GetRandomBlock(1, 0, chainID) // empty data -> should be filtered out
	h2, d2 := types.GetRandomBlock(2, 1, chainID)
	h3, d3 := types.GetRandomBlock(3, 2, chainID)

	// persist in store and set height
	for _, pair := range []struct {
		h *types.SignedHeader
		d *types.Data
	}{{h1, d1}, {h2, d2}, {h3, d3}} {
		err := st.SaveBlockData(ctx, pair.h, pair.d, &types.Signature{})
		require.NoError(t, err)
	}
	require.NoError(t, st.SetHeight(ctx, 3))

	// construct manager which brings up pending managers
	cfg := tempConfig(t)
	cm, err := NewManager(cfg, st, logger)
	require.NoError(t, err)

	// headers: all 3 should be pending initially
	headers, err := cm.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 3)
	assert.Equal(t, uint64(1), headers[0].Height())
	assert.Equal(t, uint64(3), headers[2].Height())

	// data: empty one filtered, so 2 and 3 only
	signedData, err := cm.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, signedData, 2)
	assert.Equal(t, uint64(2), signedData[0].Data.Height())
	assert.Equal(t, uint64(3), signedData[1].Data.Height())

	// update last submitted heights and re-check
	cm.SetLastSubmittedHeaderHeight(ctx, 1)
	cm.SetLastSubmittedDataHeight(ctx, 2)

	headers, err = cm.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 2)
	assert.Equal(t, uint64(2), headers[0].Height())

	signedData, err = cm.GetPendingData(ctx)
	require.NoError(t, err)
	require.Len(t, signedData, 1)
	assert.Equal(t, uint64(3), signedData[0].Data.Height())

	// numPending views
	assert.Equal(t, uint64(2), cm.NumPendingHeaders())
	assert.Equal(t, uint64(1), cm.NumPendingData())
}

func TestPendingBase_ErrorConditions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	logger := zerolog.Nop()

	// 1) invalid metadata length triggers init error
	err = st.SetMetadata(ctx, store.LastSubmittedHeaderHeightKey, []byte{1, 2})
	require.NoError(t, err)
	_, err = NewPendingHeaders(st, logger)
	require.Error(t, err)

	// 2) lastSubmitted > height yields error from getPending
	// reset metadata to a valid higher value than store height
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, 5)
	require.NoError(t, st.SetMetadata(ctx, store.LastSubmittedHeaderHeightKey, bz))

	// ensure store height stays lower (0)
	ph, err := NewPendingHeaders(st, logger)
	require.NoError(t, err)
	pending, err := ph.getPendingHeaders(ctx)
	assert.Error(t, err)
	assert.Len(t, pending, 0)

	// 3) NewPendingData shares same behavior
	err = st.SetMetadata(ctx, LastSubmittedDataHeightKey, []byte{0xFF}) // invalid length
	require.NoError(t, err)
	_, err = NewPendingData(st, logger)
	require.Error(t, err)
}

func TestPendingBase_PersistLastSubmitted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	logger := zerolog.Nop()

	ph, err := NewPendingHeaders(st, logger)
	require.NoError(t, err)

	// store height 3 to make numPending meaningful
	require.NoError(t, st.SetHeight(ctx, 3))
	assert.Equal(t, uint64(3), ph.numPendingHeaders())

	// set last submitted higher and ensure metadata is written
	ph.setLastSubmittedHeaderHeight(ctx, 2)
	raw, err := st.GetMetadata(ctx, store.LastSubmittedHeaderHeightKey)
	require.NoError(t, err)
	require.Len(t, raw, 8)
	lsh := binary.LittleEndian.Uint64(raw)
	assert.Equal(t, uint64(2), lsh)

	// setting a lower height should not overwrite
	ph.setLastSubmittedHeaderHeight(ctx, 1)
	raw2, err := st.GetMetadata(ctx, store.LastSubmittedHeaderHeightKey)
	require.NoError(t, err)
	assert.Equal(t, raw, raw2)
}
