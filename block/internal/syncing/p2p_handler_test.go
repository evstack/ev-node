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

// buildTestSigner returns an address, pubkey and signer suitable for tests
func buildTestSigner(t *testing.T) ([]byte, crypto.PubKey, signerpkg.Signer) {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(crand.Reader)
	require.NoError(t, err, "failed to generate ed25519 key for test signer")
	n, err := noop.NewNoopSigner(priv)
	require.NoError(t, err, "failed to create noop signer from private key")
	a, err := n.GetAddress()
	require.NoError(t, err, "failed to derive address from signer")
	p, err := n.GetPublic()
	require.NoError(t, err, "failed to derive public key from signer")
	return a, p, n
}

// p2pMakeSignedHeader creates a minimally valid SignedHeader for P2P tests
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

// P2PTestData aggregates all dependencies used by P2P handler tests.
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

// setupP2P constructs a P2PHandler with mocked go-header stores and real cache
func setupP2P(t *testing.T) *P2PTestData {
	t.Helper()
	proposerAddr, proposerPub, signer := buildTestSigner(t)

	gen := genesis.Genesis{ChainID: "p2p-test", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: proposerAddr}

	headerStoreMock := extmocks.NewMockStore[*types.SignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.Data](t)

	// Create a real cache manager for tests
	storeMock := storemocks.NewMockStore(t)
	// Mock the methods that cache manager initialization will call
	// Return ErrNotFound for non-existent metadata keys
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

// collectEvents reads events from a channel with a timeout
func collectEvents(t *testing.T, ch <-chan common.DAHeightEvent, timeout time.Duration) []common.DAHeightEvent {
	t.Helper()
	var events []common.DAHeightEvent
	deadline := time.After(timeout)
	for {
		select {
		case event := <-ch:
			events = append(events, event)
		case <-deadline:
			return events
		case <-time.After(10 * time.Millisecond):
			// Give it a moment to check if more events are coming
			select {
			case event := <-ch:
				events = append(events, event)
			default:
				return events
			}
		}
	}
}

func TestP2PHandler_ProcessHeaderRange_HeaderAndDataHappyPath(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	// Signed header at height 5 with non-empty data
	require.Equal(t, string(p2pData.Genesis.ProposerAddress), string(p2pData.ProposerAddr), "test signer must match genesis proposer for P2P validation")
	signedHeader := p2pMakeSignedHeader(t, p2pData.Genesis.ChainID, 5, p2pData.ProposerAddr, p2pData.ProposerPub, p2pData.Signer)
	blockData := makeData(p2pData.Genesis.ChainID, 5, 1)
	signedHeader.DataHash = blockData.DACommitment()

	// Re-sign after setting DataHash so signature matches header bytes
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&signedHeader.Header)
	require.NoError(t, err, "failed to get signature bytes after setting DataHash")
	sig, err := p2pData.Signer.Sign(bz)
	require.NoError(t, err, "failed to re-sign header after setting DataHash")
	signedHeader.Signature = sig

	// Sanity: header should validate with data using default sync verifier
	require.NoError(t, signedHeader.ValidateBasicWithData(blockData), "header+data must validate before handler processes them")

	p2pData.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(5)).Return(signedHeader, nil).Once()
	p2pData.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(5)).Return(blockData, nil).Once()

	// Create channel and process
	ch := make(chan common.DAHeightEvent, 10)
	p2pData.Handler.ProcessHeaderRange(ctx, 5, 5, ch)

	events := collectEvents(t, ch, 100*time.Millisecond)
	require.Len(t, events, 1, "expected one event for the provided header/data height")
	require.Equal(t, uint64(5), events[0].Header.Height())
	require.NotNil(t, events[0].Data)
	require.Equal(t, uint64(5), events[0].Data.Height())
}

func TestP2PHandler_ProcessHeaderRange_MissingData_NonEmptyHash(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	require.Equal(t, string(p2pData.Genesis.ProposerAddress), string(p2pData.ProposerAddr), "test signer must match genesis proposer for P2P validation")
	signedHeader := p2pMakeSignedHeader(t, p2pData.Genesis.ChainID, 7, p2pData.ProposerAddr, p2pData.ProposerPub, p2pData.Signer)

	// Non-empty data: set header.DataHash to a commitment; expect data store lookup to fail and event skipped
	blockData := makeData(p2pData.Genesis.ChainID, 7, 1)
	signedHeader.DataHash = blockData.DACommitment()

	p2pData.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(7)).Return(signedHeader, nil).Once()
	p2pData.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(7)).Return(nil, errors.New("not found")).Once()

	// Create channel and process
	ch := make(chan common.DAHeightEvent, 10)
	p2pData.Handler.ProcessHeaderRange(ctx, 7, 7, ch)

	events := collectEvents(t, ch, 100*time.Millisecond)
	require.Len(t, events, 0)
}

func TestP2PHandler_ProcessDataRange_HeaderMissing(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	blockData := makeData(p2pData.Genesis.ChainID, 9, 1)
	p2pData.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(9)).Return(blockData, nil).Once()
	p2pData.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(9)).Return(nil, errors.New("no header")).Once()

	// Create channel and process
	ch := make(chan common.DAHeightEvent, 10)
	p2pData.Handler.ProcessDataRange(ctx, 9, 9, ch)

	events := collectEvents(t, ch, 100*time.Millisecond)
	require.Len(t, events, 0)
}

func TestP2PHandler_ProposerMismatch_Rejected(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	// Build a header with a different proposer
	badAddr, pub, signer := buildTestSigner(t)
	require.NotEqual(t, string(p2pData.Genesis.ProposerAddress), string(badAddr), "negative test requires mismatched proposer")
	signedHeader := p2pMakeSignedHeader(t, p2pData.Genesis.ChainID, 4, badAddr, pub, signer)
	signedHeader.DataHash = common.DataHashForEmptyTxs

	p2pData.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(4)).Return(signedHeader, nil).Once()

	// Create channel and process
	ch := make(chan common.DAHeightEvent, 10)
	p2pData.Handler.ProcessHeaderRange(ctx, 4, 4, ch)

	events := collectEvents(t, ch, 100*time.Millisecond)
	require.Len(t, events, 0)
}

func TestP2PHandler_ProcessHeaderRange_MultipleHeightsHappyPath(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	// Build two consecutive heights with valid headers and data
	// Height 5
	header5 := p2pMakeSignedHeader(t, p2pData.Genesis.ChainID, 5, p2pData.ProposerAddr, p2pData.ProposerPub, p2pData.Signer)
	data5 := makeData(p2pData.Genesis.ChainID, 5, 1)
	header5.DataHash = data5.DACommitment()
	// Re-sign after setting DataHash to keep signature valid
	bz5, err := types.DefaultAggregatorNodeSignatureBytesProvider(&header5.Header)
	require.NoError(t, err, "failed to get signature bytes for height 5")
	sig5, err := p2pData.Signer.Sign(bz5)
	require.NoError(t, err, "failed to sign header for height 5")
	header5.Signature = sig5
	require.NoError(t, header5.ValidateBasicWithData(data5), "header/data invalid for height 5")

	// Height 6
	header6 := p2pMakeSignedHeader(t, p2pData.Genesis.ChainID, 6, p2pData.ProposerAddr, p2pData.ProposerPub, p2pData.Signer)
	data6 := makeData(p2pData.Genesis.ChainID, 6, 2)
	header6.DataHash = data6.DACommitment()
	bz6, err := types.DefaultAggregatorNodeSignatureBytesProvider(&header6.Header)
	require.NoError(t, err, "failed to get signature bytes for height 6")
	sig6, err := p2pData.Signer.Sign(bz6)
	require.NoError(t, err, "failed to sign header for height 6")
	header6.Signature = sig6
	require.NoError(t, header6.ValidateBasicWithData(data6), "header/data invalid for height 6")

	// Expectations for both heights
	p2pData.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(5)).Return(header5, nil).Once()
	p2pData.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(5)).Return(data5, nil).Once()
	p2pData.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(6)).Return(header6, nil).Once()
	p2pData.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(6)).Return(data6, nil).Once()

	// Create channel and process
	ch := make(chan common.DAHeightEvent, 10)
	p2pData.Handler.ProcessHeaderRange(ctx, 5, 6, ch)

	events := collectEvents(t, ch, 100*time.Millisecond)
	require.Len(t, events, 2, "expected two events for heights 5 and 6")
	require.Equal(t, uint64(5), events[0].Header.Height(), "first event should be height 5")
	require.Equal(t, uint64(6), events[1].Header.Height(), "second event should be height 6")
	require.NotNil(t, events[0].Data, "event for height 5 must include data")
	require.NotNil(t, events[1].Data, "event for height 6 must include data")
}

func TestP2PHandler_ProcessDataRange_HeaderValidateHeaderFails(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	// Data exists at height 3
	blockData := makeData(p2pData.Genesis.ChainID, 3, 1)
	p2pData.DataStore.EXPECT().GetByHeight(mock.Anything, uint64(3)).Return(blockData, nil).Once()

	// Header proposer does not match genesis -> validateHeader should fail
	badAddr, pub, signer := buildTestSigner(t)
	require.NotEqual(t, string(p2pData.Genesis.ProposerAddress), string(badAddr), "negative test requires mismatched proposer")
	badHeader := p2pMakeSignedHeader(t, p2pData.Genesis.ChainID, 3, badAddr, pub, signer)
	p2pData.HeaderStore.EXPECT().GetByHeight(mock.Anything, uint64(3)).Return(badHeader, nil).Once()

	// Create channel and process
	ch := make(chan common.DAHeightEvent, 10)
	p2pData.Handler.ProcessDataRange(ctx, 3, 3, ch)

	events := collectEvents(t, ch, 100*time.Millisecond)
	require.Len(t, events, 0, "validateHeader failure should drop event")
}
