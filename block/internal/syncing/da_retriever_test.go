package syncing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	testmocks "github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// makeSignedHeaderBytes builds a valid SignedHeader and returns its binary encoding and the object
func makeSignedHeaderBytes(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, appHash []byte) ([]byte, *types.SignedHeader) {
	hdr := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: chainID, Height: height, Time: uint64(time.Now().Add(time.Duration(height) * time.Second).UnixNano())},
			AppHash:         appHash,
			ProposerAddress: proposer,
		},
		Signer: types.Signer{PubKey: pub, Address: proposer},
	}
	bz, err := types.DefaultAggregatorNodeSignatureBytesProvider(&hdr.Header)
	require.NoError(t, err)
	sig, err := signer.Sign(bz)
	require.NoError(t, err)
	hdr.Signature = sig
	bin, err := hdr.MarshalBinary()
	require.NoError(t, err)
	return bin, hdr
}

// makeSignedDataBytes builds SignedData containing the provided Data and returns its binary encoding
func makeSignedDataBytes(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, txs int) ([]byte, *types.SignedData) {
	d := &types.Data{Metadata: &types.Metadata{ChainID: chainID, Height: height, Time: uint64(time.Now().UnixNano())}}
	if txs > 0 {
		d.Txs = make(types.Txs, txs)
		for i := 0; i < txs; i++ {
			d.Txs[i] = types.Tx([]byte{byte(height), byte(i)})
		}
	}

	// For DA SignedData, sign the Data payload bytes (matches DA submission logic)
	payload, err := d.MarshalBinary()
	require.NoError(t, err)
	sig, err := signer.Sign(payload)
	require.NoError(t, err)
	sd := &types.SignedData{Data: *d, Signature: sig, Signer: types.Signer{PubKey: pub, Address: proposer}}
	bin, err := sd.MarshalBinary()
	require.NoError(t, err)
	return bin, sd
}

func TestDARetriever_RetrieveFromDA_NotFound(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)

	// GetIDs returns ErrBlobNotFound -> helper maps to StatusNotFound
	mockDA.EXPECT().GetIDs(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("%s: whatever", coreda.ErrBlobNotFound.Error())).Maybe()

	r := NewDARetriever(mockDA, cm, config.DefaultConfig, genesis.Genesis{}, common.DefaultBlockOptions(), zerolog.Nop())
	events, err := r.RetrieveFromDA(context.Background(), 42)
	require.Error(t, err, coreda.ErrBlobNotFound)
	assert.Len(t, events, 0)
}

func TestDARetriever_RetrieveFromDA_HeightFromFuture(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)
	// GetIDs returns ErrHeightFromFuture -> helper maps to StatusHeightFromFuture, fetchBlobs returns error
	mockDA.EXPECT().GetIDs(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("%s: later", coreda.ErrHeightFromFuture.Error())).Maybe()

	r := NewDARetriever(mockDA, cm, config.DefaultConfig, genesis.Genesis{}, common.DefaultBlockOptions(), zerolog.Nop())
	events, derr := r.RetrieveFromDA(context.Background(), 1000)
	assert.Error(t, derr)
	assert.True(t, errors.Is(derr, coreda.ErrHeightFromFuture))
	assert.Nil(t, events)
}

func TestDARetriever_ProcessBlobs_HeaderAndData_Success(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := NewDARetriever(nil, cm, config.DefaultConfig, gen, common.DefaultBlockOptions(), zerolog.Nop())

	// Build one header and one data blob at same height
	_, lastState := types.State{}, types.State{}
	_ = lastState // placeholder to keep parity with helper pattern

	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 2, addr, pub, signer, nil)
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 2, addr, pub, signer, 2)

	events := r.processBlobs(context.Background(), [][]byte{hdrBin, dataBin}, 77)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(2), events[0].Header.Height())
	assert.Equal(t, uint64(2), events[0].Data.Height())
	assert.Equal(t, uint64(77), events[0].DaHeight)
	assert.Equal(t, uint64(77), events[0].HeaderDaIncludedHeight)
}

func TestDARetriever_ProcessBlobs_HeaderOnly_EmptyDataExpected(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := NewDARetriever(nil, cm, config.DefaultConfig, gen, common.DefaultBlockOptions(), zerolog.Nop())

	// Header with no data hash present should trigger empty data creation (per current logic)
	hb, _ := makeSignedHeaderBytes(t, gen.ChainID, 3, addr, pub, signer, nil)

	events := r.processBlobs(context.Background(), [][]byte{hb}, 88)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(3), events[0].Header.Height())
	assert.NotNil(t, events[0].Data)
	assert.Equal(t, uint64(88), events[0].DaHeight)
}

func TestDARetriever_TryDecodeHeaderAndData_Basic(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := NewDARetriever(nil, cm, config.DefaultConfig, gen, common.DefaultBlockOptions(), zerolog.Nop())

	hb, sh := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil)
	gotH := r.tryDecodeHeader(hb, 123)
	require.NotNil(t, gotH)
	assert.Equal(t, sh.Hash().String(), gotH.Hash().String())

	db, sd := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 1)
	gotD := r.tryDecodeData(db, 123)
	require.NotNil(t, gotD)
	assert.Equal(t, sd.Height(), gotD.Height())

	// invalid data fails
	assert.Nil(t, r.tryDecodeHeader([]byte("junk"), 1))
	assert.Nil(t, r.tryDecodeData([]byte("junk"), 1))
}

func TestDARetriever_isEmptyDataExpected(t *testing.T) {
	r := &DARetriever{}
	h := &types.SignedHeader{}
	// when DataHash is nil/empty -> expected empty
	assert.True(t, r.isEmptyDataExpected(h))
	// when equals to predefined emptyTxs hash -> expected empty
	h.DataHash = common.DataHashForEmptyTxs
	assert.True(t, r.isEmptyDataExpected(h))
}

func TestDARetriever_tryDecodeData_InvalidSignatureOrProposer(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	goodAddr, pub, signer := buildSyncTestSigner(t)
	badAddr := []byte("not-the-proposer")
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: badAddr}
	r := NewDARetriever(nil, cm, config.DefaultConfig, gen, common.DefaultBlockOptions(), zerolog.Nop())

	// Signed data is made by goodAddr; retriever expects badAddr -> should be rejected
	db, _ := makeSignedDataBytes(t, gen.ChainID, 7, goodAddr, pub, signer, 1)
	assert.Nil(t, r.tryDecodeData(db, 55))
}

func TestDARetriever_validateBlobResponse(t *testing.T) {
	r := &DARetriever{logger: zerolog.Nop()}
	// StatusSuccess -> nil
	err := r.validateBlobResponse(coreda.ResultRetrieve{BaseResult: coreda.BaseResult{Code: coreda.StatusSuccess}}, 1)
	assert.NoError(t, err)
	// StatusError -> error
	err = r.validateBlobResponse(coreda.ResultRetrieve{BaseResult: coreda.BaseResult{Code: coreda.StatusError, Message: "fail"}}, 1)
	assert.Error(t, err)
	// StatusHeightFromFuture -> specific error
	err = r.validateBlobResponse(coreda.ResultRetrieve{BaseResult: coreda.BaseResult{Code: coreda.StatusHeightFromFuture}}, 1)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, coreda.ErrHeightFromFuture))
}

func TestDARetriever_RetrieveFromDA_TwoNamespaces_Success(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig, st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	// Prepare header/data blobs
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 9, addr, pub, signer, nil)
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 9, addr, pub, signer, 1)

	mockDA := testmocks.NewMockDA(t)
	// Expect GetIDs for both namespaces
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1234), mock.MatchedBy(func(ns []byte) bool { return string(ns) == "nsHdr" })).
		Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("h1")}, Timestamp: time.Now()}, nil).Once()
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool { return string(ns) == "nsHdr" })).
		Return([][]byte{hdrBin}, nil).Once()

	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1234), mock.MatchedBy(func(ns []byte) bool { return string(ns) == "nsData" })).
		Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("d1")}, Timestamp: time.Now()}, nil).Once()
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool { return string(ns) == "nsData" })).
		Return([][]byte{dataBin}, nil).Once()

	cfg := config.DefaultConfig
	cfg.DA.Namespace = "nsHdr"
	cfg.DA.DataNamespace = "nsData"

	r := NewDARetriever(mockDA, cm, cfg, gen, common.DefaultBlockOptions(), zerolog.Nop())

	events, derr := r.RetrieveFromDA(context.Background(), 1234)
	require.NoError(t, derr)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(9), events[0].Header.Height())
	assert.Equal(t, uint64(9), events[0].Data.Height())
}
