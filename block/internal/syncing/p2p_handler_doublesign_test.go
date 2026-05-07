package syncing

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/block/internal/common"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// The processedHeight short-circuit must run AFTER the detector so an
// alternate at an already-applied height still triggers detection.
func TestP2PHandler_DetectsAtAlreadyProcessedHeight(t *testing.T) {
	env := newDSTestEnv(t)

	first := env.signHeaderAtHeight(5, 0x01)
	env.saveHeader(first)

	alt := env.signHeaderAtHeight(5, 0x02)
	require.NotEqual(t, first.Hash().String(), alt.Hash().String())

	headerStoreMock := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.P2PData](t)
	headerStoreMock.EXPECT().
		GetByHeight(mock.Anything, uint64(5)).
		Return(&types.P2PSignedHeader{SignedHeader: alt}, nil).
		Once()

	h := NewP2PHandler(headerStoreMock, dataStoreMock, env.cache, env.gen,
		zerolog.Nop(), env.detectDoubleSign)

	h.SetProcessedHeight(5)

	ch := make(chan common.DAHeightEvent, 1)
	require.NoError(t, h.ProcessHeight(context.Background(), 5, ch))

	captured := env.captured()
	require.Len(t, captured, 1)
	require.Equal(t, alt.Hash().String(), captured[0].AlternateHeader.Hash().String())
	require.Equal(t, types.EvidenceSourceP2P, captured[0].AlternateSource)

	select {
	case evt := <-ch:
		t.Fatalf("expected no event when double-sign fires; got %+v", evt)
	default:
	}
}

// When detection is disabled the legacy short-circuit must still fire.
func TestP2PHandler_LegacyShortCircuitWhenDetectionDisabled(t *testing.T) {
	env := newDSTestEnv(t)

	headerStoreMock := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.P2PData](t)

	h := NewP2PHandler(headerStoreMock, dataStoreMock, env.cache, env.gen,
		zerolog.Nop(), nil)

	h.SetProcessedHeight(5)

	// Mock has no expectation set — a call to GetByHeight would panic.
	ch := make(chan common.DAHeightEvent, 1)
	require.NoError(t, h.ProcessHeight(context.Background(), 5, ch))
}

// A P2P header with the correct proposer but a tampered signature must be
// rejected before the detector runs.
func TestP2PHandler_InvalidSigRejectedBeforeDetector(t *testing.T) {
	env := newDSTestEnv(t)

	good := env.signHeaderAtHeight(5, 0x01)
	pbHdr, err := good.ToProto()
	require.NoError(t, err)
	pbHdr.Signature = append([]byte(nil), good.Signature...)
	pbHdr.Signature[0] ^= 0xff
	bin, err := proto.Marshal(pbHdr)
	require.NoError(t, err)

	forged := new(types.SignedHeader)
	{
		var pbDecoded = pbHdr
		require.NoError(t, proto.Unmarshal(bin, pbDecoded))
		require.NoError(t, forged.FromProto(pbDecoded))
	}

	headerStoreMock := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.P2PData](t)
	headerStoreMock.EXPECT().
		GetByHeight(mock.Anything, uint64(5)).
		Return(&types.P2PSignedHeader{SignedHeader: forged}, nil).
		Once()

	h := NewP2PHandler(headerStoreMock, dataStoreMock, env.cache, env.gen,
		zerolog.Nop(), env.detectDoubleSign)

	ch := make(chan common.DAHeightEvent, 1)
	err = h.ProcessHeight(context.Background(), 5, ch)
	require.Error(t, err)

	require.Empty(t, env.captured())

	_, _, ok := env.cache.GetPendingSignedHeader(5)
	require.False(t, ok)
}

// First P2P observation must populate the pending cache so a later DA blob
// at the same height can be matched against it.
func TestP2PHandler_SetsPendingSignedHeaderOnFirstObservation(t *testing.T) {
	env := newDSTestEnv(t)

	// Provide both header and data so ProcessHeight reaches the emit step.
	first := env.signHeaderAtHeight(5, 0x01)
	first.DataHash = common.DataHashForEmptyTxs

	headerStoreMock := extmocks.NewMockStore[*types.P2PSignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.P2PData](t)
	headerStoreMock.EXPECT().
		GetByHeight(mock.Anything, uint64(5)).
		Return(&types.P2PSignedHeader{SignedHeader: first}, nil).
		Once()
	dataStoreMock.EXPECT().
		GetByHeight(mock.Anything, uint64(5)).
		Return(&types.P2PData{Data: &types.Data{
			Metadata: &types.Metadata{ChainID: env.chainID, Height: 5, Time: first.BaseHeader.Time},
		}}, nil).
		Once()

	h := NewP2PHandler(headerStoreMock, dataStoreMock, env.cache, env.gen,
		zerolog.Nop(), env.detectDoubleSign)

	ch := make(chan common.DAHeightEvent, 1)
	require.NoError(t, h.ProcessHeight(context.Background(), 5, ch))

	got, src, ok := env.cache.GetPendingSignedHeader(5)
	require.True(t, ok)
	require.Equal(t, first.Hash().String(), got.Hash().String())
	require.Equal(t, types.EvidenceSourceP2P, src)
}
