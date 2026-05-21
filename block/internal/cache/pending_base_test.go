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

func TestPendingBase_ErrorConditions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewTestInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	logger := zerolog.Nop()

	// 1) invalid metadata length triggers init error
	err = st.SetMetadata(ctx, store.LastSubmittedHeaderHeightKey, []byte{1, 2})
	require.NoError(t, err)
	_, err = NewPendingHeaders(st, logger)
	require.Error(t, err)

	// 2) lastSubmitted > height yields error from getPending
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, 5)
	require.NoError(t, st.SetMetadata(ctx, store.LastSubmittedHeaderHeightKey, bz))

	ph, err := NewPendingHeaders(st, logger)
	require.NoError(t, err)
	pending, _, err := ph.GetPendingHeaders(ctx)
	assert.Error(t, err)
	assert.Len(t, pending, 0)

	// 3) NewPendingData shares same behavior
	err = st.SetMetadata(ctx, LastSubmittedDataHeightKey, []byte{0xFF})
	require.NoError(t, err)
	_, err = NewPendingData(st, logger)
	require.Error(t, err)
}

func TestPendingBase_PersistLastSubmitted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dsKV, err := store.NewTestInMemoryKVStore()
	require.NoError(t, err)
	st := store.New(dsKV)
	logger := zerolog.Nop()

	ph, err := NewPendingHeaders(st, logger)
	require.NoError(t, err)

	batch, err := st.NewBatch(ctx)
	require.NoError(t, err)
	require.NoError(t, batch.SetHeight(3))
	require.NoError(t, batch.Commit())
	assert.Equal(t, uint64(3), ph.NumPendingHeaders())

	ph.SetLastSubmittedHeaderHeight(ctx, 2)
	raw, err := st.GetMetadata(ctx, store.LastSubmittedHeaderHeightKey)
	require.NoError(t, err)
	require.Len(t, raw, 8)
	lsh := binary.LittleEndian.Uint64(raw)
	assert.Equal(t, uint64(2), lsh)

	ph.SetLastSubmittedHeaderHeight(ctx, 1)
	raw2, err := st.GetMetadata(ctx, store.LastSubmittedHeaderHeightKey)
	require.NoError(t, err)
	assert.Equal(t, raw, raw2)
}

func TestPendingBase_InFlightClaim(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testMemStore(t)
	chainID := "inflight"

	for _, h := range []uint64{1, 2, 3, 4, 5} {
		hdr, data := types.GetRandomBlock(h, int(h-1), chainID)
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(hdr, data, &types.Signature{}))
		require.NoError(t, batch.SetHeight(h))
		require.NoError(t, batch.Commit())
	}

	ph, err := NewPendingHeaders(st, zerolog.Nop())
	require.NoError(t, err)

	// Claim all 5 items
	headers, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 5)

	// All claimed, nothing pending
	assert.Equal(t, uint64(0), ph.NumPendingHeaders())

	// Second call returns nothing (all claimed)
	headers2, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	assert.Empty(t, headers2)

	// Simulate failure: reset the claim range
	ph.ResetInFlightHeaderRange(1, 5)

	// Items are available again
	assert.Equal(t, uint64(5), ph.NumPendingHeaders())
	headers3, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers3, 5)
}

func TestPendingBase_InFlightPartialAdvance(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testMemStore(t)
	chainID := "inflight-partial"

	for _, h := range []uint64{1, 2, 3, 4, 5} {
		hdr, data := types.GetRandomBlock(h, int(h-1), chainID)
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(hdr, data, &types.Signature{}))
		require.NoError(t, batch.SetHeight(h))
		require.NoError(t, batch.Commit())
	}

	ph, err := NewPendingHeaders(st, zerolog.Nop())
	require.NoError(t, err)

	// Claim all items [1..5]
	headers, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 5)

	// Partial success: [1..3] submitted, claim trimmed to [4..5]
	ph.SetLastSubmittedHeaderHeight(ctx, 3)

	// Claim [4..5] still active, so getPending returns nothing
	headers2, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	assert.Empty(t, headers2)
	assert.Equal(t, uint64(0), ph.NumPendingHeaders())

	// Add new items at heights 6-8
	for _, h := range []uint64{6, 7, 8} {
		hdr, data := types.GetRandomBlock(h, int(h-1), chainID)
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(hdr, data, &types.Signature{}))
		require.NoError(t, batch.SetHeight(h))
		require.NoError(t, batch.Commit())
	}

	// Claim [4..5] still active, but items [6..8] are available
	assert.Equal(t, uint64(3), ph.NumPendingHeaders())
	headers3, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers3, 3)
	assert.Equal(t, uint64(6), headers3[0].Height())
	assert.Equal(t, uint64(8), headers3[2].Height())

	// Retry of [4..5] succeeds
	ph.SetLastSubmittedHeaderHeight(ctx, 5)

	// Claims [6..8] active, lastHeight=5, all covered
	assert.Equal(t, uint64(0), ph.NumPendingHeaders())
}

func TestPendingBase_InFlightGapReexposure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testMemStore(t)
	chainID := "inflight-gap"

	// Start with items [1..5]
	for _, h := range []uint64{1, 2, 3, 4, 5} {
		hdr, data := types.GetRandomBlock(h, int(h-1), chainID)
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(hdr, data, &types.Signature{}))
		require.NoError(t, batch.SetHeight(h))
		require.NoError(t, batch.Commit())
	}

	ph, err := NewPendingHeaders(st, zerolog.Nop())
	require.NoError(t, err)

	// Claim A: all available items [1..5]
	hA, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, hA, 5)

	// Add items [6..15]
	for _, h := range []uint64{6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		hdr, data := types.GetRandomBlock(h, int(h-1), chainID)
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(hdr, data, &types.Signature{}))
		require.NoError(t, batch.SetHeight(h))
		require.NoError(t, batch.Commit())
	}

	// Claim B: items [6..15] (claim A still covers [1..5])
	hB, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, hB, 10)
	assert.Equal(t, uint64(6), hB[0].Height())

	// A succeeds
	ph.SetLastSubmittedHeaderHeight(ctx, 5)
	// B succeeds (lastHeight jumps to 15)
	ph.SetLastSubmittedHeaderHeight(ctx, 15)

	// Now simulate: a retry of items [8..10] from claim B had failed earlier
	// and the retry loop also fails. Reset the sub-range.
	// Since [8..10] is below lastHeight=15, it becomes a gap.
	ph.ResetInFlightHeaderRange(8, 10)

	// Gap [8..10] should be available
	assert.Equal(t, uint64(3), ph.NumPendingHeaders())

	hRetry, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, hRetry, 3)
	assert.Equal(t, uint64(8), hRetry[0].Height())
	assert.Equal(t, uint64(10), hRetry[2].Height())
}

func TestPendingBase_InFlightResetOnFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testMemStore(t)
	chainID := "inflight-reset"

	for _, h := range []uint64{1, 2, 3} {
		hdr, data := types.GetRandomBlock(h, int(h-1), chainID)
		batch, err := st.NewBatch(ctx)
		require.NoError(t, err)
		require.NoError(t, batch.SaveBlockData(hdr, data, &types.Signature{}))
		require.NoError(t, batch.SetHeight(h))
		require.NoError(t, batch.Commit())
	}

	ph, err := NewPendingHeaders(st, zerolog.Nop())
	require.NoError(t, err)

	// Claim all items
	_, _, err = ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), ph.NumPendingHeaders())

	// Simulate failure and reset
	ph.ResetInFlightHeaderRange(1, 3)
	assert.Equal(t, uint64(3), ph.NumPendingHeaders())

	// Claim again
	headers, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 3)

	// Partial success: [1] submitted, claim trimmed to [2..3]
	ph.SetLastSubmittedHeaderHeight(ctx, 1)
	// Failure of remaining [2..3]: reset the trimmed claim
	ph.ResetInFlightHeaderRange(2, 3)

	// Items from height 2 onward should be available
	assert.Equal(t, uint64(2), ph.NumPendingHeaders())
	headers2, _, err := ph.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers2, 2)
	assert.Equal(t, uint64(2), headers2[0].Height())
}
