package cache

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	storepkg "github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

func TestPendingHeaders_BasicFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := memStore(t)

	// create and persist three blocks
	chainID := "ph-basic"
	h1, d1 := types.GetRandomBlock(1, 0, chainID)
	h2, d2 := types.GetRandomBlock(2, 1, chainID)
	h3, d3 := types.GetRandomBlock(3, 2, chainID)

	for _, p := range []struct {
		h *types.SignedHeader
		d *types.Data
	}{{h1, d1}, {h2, d2}, {h3, d3}} {
		require.NoError(t, store.SaveBlockData(ctx, p.h, p.d, &types.Signature{}))
	}
	require.NoError(t, store.SetHeight(ctx, 3))

	pendingHeaders, err := NewPendingHeaders(store, zerolog.Nop())
	require.NoError(t, err)

	// initially all three are pending
	require.Equal(t, uint64(3), pendingHeaders.NumPendingHeaders())
	headers, err := pendingHeaders.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 3)
	require.Equal(t, uint64(1), headers[0].Height())
	require.Equal(t, uint64(3), headers[2].Height())

	// advance last submitted height and verify persistence + filtering
	pendingHeaders.SetLastSubmittedHeaderHeight(ctx, 2)
	metadataRaw, err := store.GetMetadata(ctx, storepkg.LastSubmittedHeaderHeightKey)
	require.NoError(t, err)
	require.Len(t, metadataRaw, 8)
	require.Equal(t, uint64(2), binary.LittleEndian.Uint64(metadataRaw))

	require.Equal(t, uint64(1), pendingHeaders.NumPendingHeaders())
	headers, err = pendingHeaders.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Len(t, headers, 1)
	require.Equal(t, uint64(3), headers[0].Height())

	// New instance should pick up metadata
	pendingHeaders2, err := NewPendingHeaders(store, zerolog.Nop())
	require.NoError(t, err)
	require.Equal(t, uint64(1), pendingHeaders2.NumPendingHeaders())
}

func TestPendingHeaders_EmptyWhenUpToDate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := memStore(t)

	h, d := types.GetRandomBlock(1, 1, "ph-up")
	require.NoError(t, store.SaveBlockData(ctx, h, d, &types.Signature{}))
	require.NoError(t, store.SetHeight(ctx, 1))

	pendingHeaders, err := NewPendingHeaders(store, zerolog.Nop())
	require.NoError(t, err)

	// set last submitted to the current height, so nothing pending
	pendingHeaders.SetLastSubmittedHeaderHeight(ctx, 1)
	require.Equal(t, uint64(0), pendingHeaders.NumPendingHeaders())
	headers, err := pendingHeaders.GetPendingHeaders(ctx)
	require.NoError(t, err)
	require.Empty(t, headers)
}
