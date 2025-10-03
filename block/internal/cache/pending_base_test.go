package cache

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
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
	require.NoError(t, st.SetHeight(ctx, 3))
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

type someValue struct {
	height uint64
}

func TestPendingBaseIterator_EmitsAllItemsInOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	require.NoError(t, st.SetHeight(ctx, 4))

	items := map[uint64]*someValue{
		2: {height: 2},
		3: {height: 3},
		4: {height: 4},
	}

	pb := &pendingBase[*someValue]{
		logger:  zerolog.Nop(),
		store:   st,
		metaKey: "Iterator-order",
		fetch: func(ctx context.Context, _ store.Store, height uint64) (*someValue, error) {
			if height > 4 {
				return nil, assert.AnError
			}

			return items[height], nil
		},
	}
	pb.lastSubmittedHeight.Store(1)

	iter, err := pb.iterator(ctx)
	require.NoError(t, err)

	var heights []uint64
	for {
		val, ok, err := iter.Next()
		if errors.Is(err, io.EOF) {
			require.False(t, ok)
			break
		}
		require.NoError(t, err)
		require.True(t, ok)
		heights = append(heights, val.height)
	}

	assert.Equal(t, []uint64{2, 3, 4}, heights)
}

func TestPendingBaseIterator_StopsOnFetchError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	require.NoError(t, st.SetHeight(ctx, 3))

	pb := &pendingBase[*someValue]{
		logger:  zerolog.Nop(),
		store:   st,
		metaKey: "Iterator-error",
		fetch: func(ctx context.Context, _ store.Store, height uint64) (*someValue, error) {
			if height == 3 {
				return nil, assert.AnError
			}
			return &someValue{height: height}, nil
		},
	}
	pb.lastSubmittedHeight.Store(1)

	iter, err := pb.iterator(ctx)
	require.NoError(t, err)

	first, ok, err := iter.Next()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, uint64(2), first.height)

	_, _, err = iter.Next()
	require.ErrorIs(t, err, assert.AnError)

	val, ok, err := iter.Next()
	require.ErrorIs(t, err, io.EOF)
	require.False(t, ok)
	assert.Nil(t, val)
}

func TestPendingHeadersIterator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	require.NoError(t, st.SetHeight(ctx, 4))

	items := map[uint64]*types.SignedHeader{
		2: {},
		3: {},
		4: {},
	}

	base := &pendingBase[*types.SignedHeader]{
		logger:  zerolog.Nop(),
		store:   st,
		metaKey: "pending-headers-iterator",
		fetch: func(_ context.Context, _ store.Store, height uint64) (*types.SignedHeader, error) {
			return items[height], nil
		},
	}
	base.lastSubmittedHeight.Store(1)

	ph := &PendingHeaders{base: base}
	iter, err := ph.Iterator(ctx)
	require.NoError(t, err)

	var got []*types.SignedHeader
	for {
		val, ok, err := iter.Next()
		if errors.Is(err, io.EOF) {
			require.False(t, ok)
			break
		}
		require.NoError(t, err)
		require.True(t, ok)
		got = append(got, val)
	}

	assert.Equal(t, []*types.SignedHeader{items[2], items[3], items[4]}, got)
}

func TestPendingDataIterator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	require.NoError(t, st.SetHeight(ctx, 3))

	items := map[uint64]*types.Data{
		2: {Metadata: &types.Metadata{Height: 2}},
		3: {Metadata: &types.Metadata{Height: 3}},
	}

	base := &pendingBase[*types.Data]{
		logger:  zerolog.Nop(),
		store:   st,
		metaKey: "pending-data-iterator",
		fetch: func(_ context.Context, _ store.Store, height uint64) (*types.Data, error) {
			return items[height], nil
		},
	}
	base.lastSubmittedHeight.Store(1)

	pd := &PendingData{base: base}
	iter, err := pd.Iterator(ctx)
	require.NoError(t, err)

	var got []*types.Data
	for {
		val, ok, err := iter.Next()
		if errors.Is(err, io.EOF) {
			require.False(t, ok)
			break
		}
		require.NoError(t, err)
		require.True(t, ok)
		got = append(got, val)
	}

	assert.Equal(t, []*types.Data{items[2], items[3]}, got)
}
