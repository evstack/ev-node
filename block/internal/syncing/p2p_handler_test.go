package syncing

import (
	"context"
	crand "crypto/rand"
	"errors"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
    "github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/signer/noop"
	"github.com/evstack/ev-node/pkg/store"
	extmocks "github.com/evstack/ev-node/test/mocks/external"
	"github.com/evstack/ev-node/types"
)

// buildTestSigner returns an address, pubkey and signer suitable for tests
func buildTestSigner(t *testing.T) (addr []byte, pub crypto.PubKey, s signerpkg.Signer) {
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

// p2pMakeData creates Data with the given number of txs
func p2pMakeData(t *testing.T, chainID string, height uint64, txs int) *types.Data {
	t.Helper()
	d := &types.Data{Metadata: &types.Metadata{ChainID: chainID, Height: height, Time: uint64(time.Now().UnixNano())}}
	if txs > 0 {
		d.Txs = make(types.Txs, txs)
		for i := 0; i < txs; i++ {
			d.Txs[i] = []byte{byte(height), byte(i)}
		}
	}
	return d
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

// setupP2P constructs a P2PHandler with mocked go-header stores and in-memory cache/store
func setupP2P(t *testing.T) *P2PTestData {
	t.Helper()
	datastore := dssync.MutexWrap(ds.NewMapDatastore())
	stateStore := store.New(datastore)
    cacheManager, err := cache.NewManager(config.DefaultConfig, stateStore, zerolog.Nop())
    require.NoError(t, err, "failed to create cache manager")

	proposerAddr, proposerPub, signer := buildTestSigner(t)

	gen := genesis.Genesis{ChainID: "p2p-test", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: proposerAddr}

	headerStoreMock := extmocks.NewMockStore[*types.SignedHeader](t)
	dataStoreMock := extmocks.NewMockStore[*types.Data](t)

	handler := NewP2PHandler(headerStoreMock, dataStoreMock, cacheManager, gen, common.DefaultBlockOptions(), zerolog.Nop())
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

func TestP2PHandler_ProcessHeaderRange_HeaderAndDataHappyPath(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	// Signed header at height 5 with non-empty data
    require.Equal(t, string(p2pData.Genesis.ProposerAddress), string(p2pData.ProposerAddr), "test signer must match genesis proposer for P2P validation")
	signedHeader := p2pMakeSignedHeader(t, p2pData.Genesis.ChainID, 5, p2pData.ProposerAddr, p2pData.ProposerPub, p2pData.Signer)
	blockData := p2pMakeData(t, p2pData.Genesis.ChainID, 5, 1)
	signedHeader.DataHash = blockData.DACommitment()

	// Re-sign after setting DataHash so signature matches header bytes
    bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&signedHeader.Header)
    require.NoError(t, err, "failed to get signature bytes after setting DataHash")
    sig, err := p2pData.Signer.Sign(bz)
    require.NoError(t, err, "failed to re-sign header after setting DataHash")
	signedHeader.Signature = sig

	// Sanity: header should validate with data using default sync verifier
    require.NoError(t, signedHeader.ValidateBasicWithData(blockData), "header+data must validate before handler processes them")

	p2pData.HeaderStore.EXPECT().GetByHeight(ctx, uint64(5)).Return(signedHeader, nil).Once()
	p2pData.DataStore.EXPECT().GetByHeight(ctx, uint64(5)).Return(blockData, nil).Once()

	events := p2pData.Handler.ProcessHeaderRange(ctx, 5, 5)
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
	blockData := p2pMakeData(t, p2pData.Genesis.ChainID, 7, 1)
	signedHeader.DataHash = blockData.DACommitment()

	p2pData.HeaderStore.EXPECT().GetByHeight(ctx, uint64(7)).Return(signedHeader, nil).Once()
	p2pData.DataStore.EXPECT().GetByHeight(ctx, uint64(7)).Return(nil, errors.New("not found")).Once()

    events := p2pData.Handler.ProcessHeaderRange(ctx, 7, 7)
    require.Len(t, events, 0)
}

func TestP2PHandler_ProcessDataRange_HeaderMissing(t *testing.T) {
	p2pData := setupP2P(t)
	ctx := context.Background()

	blockData := p2pMakeData(t, p2pData.Genesis.ChainID, 9, 1)
	p2pData.DataStore.EXPECT().GetByHeight(ctx, uint64(9)).Return(blockData, nil).Once()
	p2pData.HeaderStore.EXPECT().GetByHeight(ctx, uint64(9)).Return(nil, errors.New("no header")).Once()

    events := p2pData.Handler.ProcessDataRange(ctx, 9, 9)
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

	p2pData.HeaderStore.EXPECT().GetByHeight(ctx, uint64(4)).Return(signedHeader, nil).Once()

    events := p2pData.Handler.ProcessHeaderRange(ctx, 4, 4)
    require.Len(t, events, 0)
}
