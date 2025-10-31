package syncing

import (
	"context"
	crand "crypto/rand"
	"errors"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	storemocks "github.com/evstack/ev-node/test/mocks"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// buildTestSigner returns an address, pubkey and signer suitable for tests.
func buildTestSigner(t *testing.T) ([]byte, crypto.PubKey, signerpkg.Signer) {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err, "failed to generate ed25519 key for test signer")
	n, err := noop.NewNoopSigner(priv)
	require.NoError(t, err, "failed to create noop signer from private key")
	addr, err := n.GetAddress()
	require.NoError(t, err, "failed to derive address from signer")
	pub, err := n.GetPublic()
	require.NoError(t, err, "failed to derive public key from signer")
	return addr, pub, n
}

// p2pMakeSignedHeader creates a minimally valid SignedHeader for P2P tests.
func p2pMakeSignedHeader(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer) *types.SignedHeader {
	t.Helper()
	hdr := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: chainID, Height: height, Time: uint64(time.Now().UnixNano())},
			ProposerAddress: proposer,
		},
		Signer: types.Signer{PubKey: pub, Address: proposer},
	}
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&hdr.Header)
	require.NoError(t, err, "failed to get signature bytes for header")
	sig, err := signer.Sign(bz)
	require.NoError(t, err, "failed to sign header bytes")
	hdr.Signature = sig
	return hdr
}

// P2PTestData aggregates dependencies used by P2P handler tests.
type P2PTestData struct {
	Handler      *P2PHandler
	HeaderStore  *extmocks.MockStore[*types.SignedHeader]
	DataStore    *extmocks.MockStore[*types.Data]
	Cache        cache.Manager
	Genesis      genesis.Genesis
	ProposerAddr []byte
	ProposerPub  crypto.PubKey
	Signer       signerpkg.Signer
}

// setupP2P constructs a P2PHandler with mocked go-header stores and a real cache.
func setupP2P(t *testing.T) *P2PTestData {
	t.Helper()
	proposerAddr, proposerPub, signer := buildTestSigner(t)

	gen := genesis.Genesis{ChainID: "p2p-test", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: proposerAddr}

	headerStoreMock := extmocks.NewMockStore[*types.SignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.Data](t)

	storeMock := storemocks.NewMockStore(t)
	storeMock.EXPECT().GetMetadata(mock.Anything, "last-submitted-header-height").Return(nil, ds.ErrNotFound).Maybe()
	storeMock.EXPECT().GetMetadata(mock.Anything, "last-submitted-data-height").Return(nil, ds.ErrNotFound).Maybe()
	storeMock.EXPECT().Height(mock.Anything).Return(uint64(0), nil).Maybe()
	storeMock.EXPECT().SetMetadata(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	cfg := config.Config{
		RootDir:    t.TempDir(),
		ClearCache: true,
	}
	cacheManager, err := cache.NewManager(cfg, storeMock, zerolog.Nop())
	require.NoError(t, err, "failed to create cache manager")

	handler := NewP2PHandler(headerStoreMock, dataStoreMock, cacheManager, gen, zerolog.Nop())
	return &P2PTestData{
		Handler:      handler,
		HeaderStore:  headerStoreMock,
		DataStore:    dataStoreMock,
		Cache:        cacheManager,
		Genesis:      gen,
		ProposerAddr: proposerAddr,
		ProposerPub:  proposerPub,
		Signer:       signer,
	}
}

func collectEvents(t *testing.T, ch <-chan common.DAHeightEvent, timeout time.Duration) []common.DAHeightEvent {
	t.Helper()
	var events []common.DAHeightEvent
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case evt := <-ch:
			events = append(events, evt)
		case <-timer.C:
			return events
		default:
			if len(events) == 0 {
				select {
				case evt := <-ch:
					events = append(events, evt)
				default:
					return events
				}
			} else {
				return events
			}
		}
	}
}

func TestP2PHandler_ProcessRange_EmitsEventWhenHeaderAndDataPresent(t *testing.T) {
	p := setupP2P(t)
	ctx := context.Background()

	require.Equal(t, string(p.Genesis.ProposerAddress), string(p.ProposerAddr))

	header := p2pMakeSignedHeader(t, p.Genesis.ChainID, 5, p.ProposerAddr, p.ProposerPub, p.Signer)
	data := makeData(p.Genesis.ChainID, 5, 1)
	header.DataHash = data.DACommitment()
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&header.Header)
	require.NoError(t, err)
	sig, err := p.Signer.Sign(bz)
	require.NoError(t, err)
	header.Signature = sig

	p.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(5)).Return(header, nil).Once()
	p.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(5)).Return(data, nil).Once()

	ch := make(chan common.DAHeightEvent, 1)
	p.Handler.ProcessHeaderRange(ctx, 5, 5, ch)

	events := collectEvents(t, ch, 50*time.Millisecond)
	require.Len(t, events, 1)
	require.Equal(t, uint64(5), events[0].Header.Height())
	require.NotNil(t, events[0].Data)
}

func TestP2PHandler_ProcessRange_SkipsWhenDataMissing(t *testing.T) {
	p := setupP2P(t)
	ctx := context.Background()

	header := p2pMakeSignedHeader(t, p.Genesis.ChainID, 7, p.ProposerAddr, p.ProposerPub, p.Signer)
	data := makeData(p.Genesis.ChainID, 7, 1)
	header.DataHash = data.DACommitment()
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&header.Header)
	require.NoError(t, err)
	sig, err := p.Signer.Sign(bz)
	require.NoError(t, err)
	header.Signature = sig

	p.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(7)).Return(header, nil).Once()
	p.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(7)).Return(nil, errors.New("missing")).Once()

	ch := make(chan common.DAHeightEvent, 1)
	p.Handler.ProcessHeaderRange(ctx, 7, 7, ch)

	require.Empty(t, collectEvents(t, ch, 50*time.Millisecond))
}

func TestP2PHandler_ProcessRange_SkipsWhenHeaderMissing(t *testing.T) {
	p := setupP2P(t)
	ctx := context.Background()

	p.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(9)).Return(nil, errors.New("missing")).Once()

	ch := make(chan common.DAHeightEvent, 1)
	p.Handler.ProcessHeaderRange(ctx, 9, 9, ch)

	require.Empty(t, collectEvents(t, ch, 50*time.Millisecond))
	p.DataStore.AssertNotCalled(t, "GetByHeight", mock.Anything, uint64(9))
}

func TestP2PHandler_ProcessRange_SkipsOnProposerMismatch(t *testing.T) {
	p := setupP2P(t)
	ctx := context.Background()

	badAddr, pub, signer := buildTestSigner(t)
	require.NotEqual(t, string(p.Genesis.ProposerAddress), string(badAddr))

	header := p2pMakeSignedHeader(t, p.Genesis.ChainID, 11, badAddr, pub, signer)
	header.DataHash = common.DataHashForEmptyTxs

	p.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(11)).Return(header, nil).Once()

	ch := make(chan common.DAHeightEvent, 1)
	p.Handler.ProcessHeaderRange(ctx, 11, 11, ch)

	require.Empty(t, collectEvents(t, ch, 50*time.Millisecond))
	p.DataStore.AssertNotCalled(t, "GetByHeight", mock.Anything, uint64(11))
}

func TestP2PHandler_ProcessRange_UsesProcessedHeightToSkip(t *testing.T) {
	p := setupP2P(t)
	ctx := context.Background()

	// Mark up to height 5 as processed.
	p.Handler.SetProcessedHeight(5)

	header := p2pMakeSignedHeader(t, p.Genesis.ChainID, 6, p.ProposerAddr, p.ProposerPub, p.Signer)
	data := makeData(p.Genesis.ChainID, 6, 1)
	header.DataHash = data.DACommitment()
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&header.Header)
	require.NoError(t, err)
	sig, err := p.Signer.Sign(bz)
	require.NoError(t, err)
	header.Signature = sig

	p.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(6)).Return(header, nil).Once()
	p.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(6)).Return(data, nil).Once()

	ch := make(chan common.DAHeightEvent, 1)
	p.Handler.ProcessHeaderRange(ctx, 4, 6, ch)

	events := collectEvents(t, ch, 50*time.Millisecond)
	require.Len(t, events, 1)
	require.Equal(t, uint64(6), events[0].Header.Height())
}

func TestP2PHandler_OnHeightProcessedPreventsDuplicates(t *testing.T) {
	p := setupP2P(t)
	ctx := context.Background()

	header := p2pMakeSignedHeader(t, p.Genesis.ChainID, 8, p.ProposerAddr, p.ProposerPub, p.Signer)
	data := makeData(p.Genesis.ChainID, 8, 0)
	header.DataHash = data.DACommitment()
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&header.Header)
	require.NoError(t, err)
	sig, err := p.Signer.Sign(bz)
	require.NoError(t, err)
	header.Signature = sig

	p.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(8)).Return(header, nil).Once()
	p.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(8)).Return(data, nil).Once()

	ch := make(chan common.DAHeightEvent, 1)
	p.Handler.ProcessHeaderRange(ctx, 8, 8, ch)

	events := collectEvents(t, ch, 50*time.Millisecond)
	require.Len(t, events, 1)

	// Mark the height as processed; a subsequent range should skip lookups.
	p.Handler.SetProcessedHeight(8)

	p.HeaderStore.AssertExpectations(t)
	p.DataStore.AssertExpectations(t)

	// No additional expectations set; if the handler queried the stores again the mock would fail.
	p.Handler.ProcessHeaderRange(ctx, 7, 8, ch)
	require.Empty(t, collectEvents(t, ch, 50*time.Millisecond))
}
