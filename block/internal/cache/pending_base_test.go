package cache

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/store"
)

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
	pending, err := ph.GetPendingHeaders(ctx)
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
	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(3))
	require.NoError(t, batch.Commit())
	assert.Equal(t, uint64(3), ph.NumPendingHeaders())

	// set last submitted higher and ensure metadata is written
	ph.SetLastSubmittedHeaderHeight(ctx, 2)
	raw, err := st.GetMetadata(ctx, store.LastSubmittedHeaderHeightKey)
	require.NoError(t, err)
	require.Len(t, raw, 8)
	lsh := binary.LittleEndian.Uint64(raw)
	assert.Equal(t, uint64(2), lsh)

	// setting a lower height should not overwrite
	ph.SetLastSubmittedHeaderHeight(ctx, 1)
	raw2, err := st.GetMetadata(ctx, store.LastSubmittedHeaderHeightKey)
	require.NoError(t, err)
	assert.Equal(t, raw, raw2)
}
