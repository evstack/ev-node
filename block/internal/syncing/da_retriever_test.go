package syncing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/test/mocks"
	"github.com/evstack/ev-node/types"
)

// newTestDARetriever creates a DA retriever for testing with the given DA implementation
func newTestDARetriever(t *testing.T, mockClient *mocks.MockClient, cfg config.Config, gen genesis.Genesis) *daRetriever {
	t.Helper()
	if cfg.DA.Namespace == "" {
		cfg.DA.Namespace = "test-ns"
	}
	if cfg.DA.DataNamespace == "" {
		cfg.DA.DataNamespace = "test-data-ns"
	}

	// Create an in-memory store for the cache
	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)

	cm, err := cache.NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	if mockClient == nil {
		mockClient = mocks.NewMockClient(t)
	}
	// default namespace helpers
	mockClient.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte(cfg.DA.DataNamespace)).Maybe()
	mockClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	mockClient.On("HasForcedInclusionNamespace").Return(false).Maybe()

	return NewDARetriever(mockClient, cm, gen, zerolog.Nop())
}

// makeSignedDataBytes builds SignedData containing the provided Data and returns its binary encoding
func makeSignedDataBytes(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, txs int) ([]byte, *types.SignedData) {
	return makeSignedDataBytesWithTime(t, chainID, height, proposer, pub, signer, txs, uint64(time.Now().UnixNano()))
}

func makeSignedDataBytesWithTime(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, txs int, timestamp uint64) ([]byte, *types.SignedData) {
	d := &types.Data{Metadata: &types.Metadata{ChainID: chainID, Height: height, Time: timestamp}}
	if txs > 0 {
		d.Txs = make(types.Txs, txs)
		for i := range txs {
			d.Txs[i] = types.Tx([]byte{byte(height), byte(i)})
		}
	}

	// For DA SignedData, sign the Data payload bytes (matches DA submission logic)
	payload, _ := d.MarshalBinary()
	sig, err := signer.Sign(t.Context(), payload)
	require.NoError(t, err)
	sd := &types.SignedData{Data: *d, Signature: sig, Signer: types.Signer{PubKey: pub, Address: proposer}}
	bin, err := sd.MarshalBinary()
	require.NoError(t, err)
	return bin, sd
}

func TestDARetriever_RetrieveFromDA_Invalid(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte("test-ns")).Maybe()
	client.On("GetDataNamespace").Return([]byte("test-data-ns")).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	client.On("RetrieveBlobs", mock.Anything, uint64(42), []byte("test-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess}}).Once()
	client.On("RetrieveBlobs", mock.Anything, uint64(42), []byte("test-data-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "just invalid"}}).Once()

	r := newTestDARetriever(t, client, config.DefaultConfig(), genesis.Genesis{})
	events, err := r.RetrieveFromDA(context.Background(), 42)
	assert.Error(t, err)
	assert.Len(t, events, 0)
}

func TestDARetriever_RetrieveFromDA_NotFound(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte("test-ns")).Maybe()
	client.On("GetDataNamespace").Return([]byte("test-data-ns")).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	client.On("RetrieveBlobs", mock.Anything, uint64(42), []byte("test-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess}}).Once()
	client.On("RetrieveBlobs", mock.Anything, uint64(42), []byte("test-data-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Message: fmt.Sprintf("%s: whatever", datypes.ErrBlobNotFound.Error())}}).Once()

	r := newTestDARetriever(t, client, config.DefaultConfig(), genesis.Genesis{})
	events, err := r.RetrieveFromDA(context.Background(), 42)
	assert.True(t, errors.Is(err, datypes.ErrBlobNotFound))
	assert.Len(t, events, 0)
}

func TestDARetriever_RetrieveFromDA_HeightFromFuture(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte("test-ns")).Maybe()
	client.On("GetDataNamespace").Return([]byte("test-data-ns")).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	client.On("RetrieveBlobs", mock.Anything, uint64(1000), []byte("test-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess}}).Once()
	client.On("RetrieveBlobs", mock.Anything, uint64(1000), []byte("test-data-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture, Message: fmt.Sprintf("%s: later", datypes.ErrHeightFromFuture.Error())}}).Once()

	r := newTestDARetriever(t, client, config.DefaultConfig(), genesis.Genesis{})
	events, derr := r.RetrieveFromDA(context.Background(), 1000)
	assert.Error(t, derr)
	assert.True(t, errors.Is(derr, datypes.ErrHeightFromFuture))
	assert.Nil(t, events)
}

func TestDARetriever_RetrieveFromDA_TimeoutFast(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte("test-ns")).Maybe()
	client.On("GetDataNamespace").Return([]byte("test-data-ns")).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	client.On("RetrieveBlobs", mock.Anything, uint64(42), []byte("test-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: context.DeadlineExceeded.Error()}}).Once()
	client.On("RetrieveBlobs", mock.Anything, uint64(42), []byte("test-data-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusContextDeadline, Message: context.DeadlineExceeded.Error()}}).Once()

	r := newTestDARetriever(t, client, config.DefaultConfig(), genesis.Genesis{})

	events, err := r.RetrieveFromDA(context.Background(), 42)

	// Verify error is returned and contains deadline exceeded information
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DA retrieval failed")
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Len(t, events, 0)
}

func TestDARetriever_ProcessBlobs_HeaderAndData_Success(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 2, addr, pub, signer, 2)
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 2, addr, pub, signer, nil, &data.Data, nil)

	events := r.processBlobs(context.Background(), [][]byte{hdrBin, dataBin}, 77)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(2), events[0].Header.Height())
	assert.Equal(t, uint64(2), events[0].Data.Height())
	assert.Equal(t, uint64(77), events[0].DaHeight)

	hHeight, ok := r.cache.GetHeaderDAIncludedByHash(events[0].Header.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(77), hHeight)

	dHeight, ok := r.cache.GetDataDAIncludedByHash(events[0].Data.DACommitment().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(77), dHeight)
}

func TestDARetriever_ProcessBlobs_HeaderOnly_EmptyDataExpected(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	// Header with no data hash present should trigger empty data creation (per current logic)
	hb, _ := makeSignedHeaderBytes(t, gen.ChainID, 3, addr, pub, signer, nil, nil, nil)

	events := r.processBlobs(context.Background(), [][]byte{hb}, 88)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(3), events[0].Header.Height())
	assert.NotNil(t, events[0].Data)
	assert.Equal(t, uint64(88), events[0].DaHeight)

	hHeight, ok := r.cache.GetHeaderDAIncludedByHash(events[0].Header.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(88), hHeight)

	// empty data is not marked as data included (the submitter component does handle the empty data case)
	_, ok = r.cache.GetDataDAIncludedByHash(events[0].Data.DACommitment().String())
	assert.False(t, ok)
}

func TestDARetriever_TryDecodeHeaderAndData_Basic(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	hb, sh := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, nil, nil)
	gotH, _ := r.tryDecodeHeader(hb)
	require.NotNil(t, gotH)
	assert.Equal(t, sh.Hash().String(), gotH.Hash().String())

	db, sd := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 1)
	gotD, _ := r.tryDecodeData(db)
	require.NotNil(t, gotD)
	assert.Equal(t, sd.Height(), gotD.Height())

	// invalid data fails
	junkH, _ := r.tryDecodeHeader([]byte("junk"))
	assert.Nil(t, junkH)
	junkD, _ := r.tryDecodeData([]byte("junk"))
	assert.Nil(t, junkD)
}

func TestDARetriever_tryDecodeData_InvalidSignature(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	_, signedData := makeSignedDataBytes(t, gen.ChainID, 7, addr, pub, signer, 1)
	require.NotEmpty(t, signedData.Signature)
	signedData.Signature[0] ^= 0x01
	db, err := signedData.MarshalBinary()
	require.NoError(t, err)

	got, _ := r.tryDecodeData(db)
	assert.Nil(t, got)
}

func TestDARetriever_tryDecodeData_NilMetadata(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	// A validly-signed blob whose Data carries no metadata must be rejected,
	// not dereferenced: the accessors below (DACommitment, Height) assume
	// Metadata is set.
	d := &types.Data{Txs: types.Txs{types.Tx([]byte{0x01})}}
	payload, err := d.MarshalBinary()
	require.NoError(t, err)
	sig, err := signer.Sign(t.Context(), payload)
	require.NoError(t, err)
	sd := &types.SignedData{Data: *d, Signature: sig, Signer: types.Signer{PubKey: pub, Address: addr}}
	db, err := sd.MarshalBinary()
	require.NoError(t, err)

	got, _ := r.tryDecodeData(db)
	assert.Nil(t, got)
}

func TestDARetriever_validateBlobResponse(t *testing.T) {
	r := &daRetriever{logger: zerolog.Nop()}
	// StatusSuccess -> nil
	err := r.validateBlobResponse(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess}}, 1)
	assert.NoError(t, err)
	// StatusError -> error
	err = r.validateBlobResponse(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusError, Message: "fail"}}, 1)
	assert.Error(t, err)
	// StatusHeightFromFuture -> specific error
	err = r.validateBlobResponse(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture}}, 1)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, datypes.ErrHeightFromFuture))
}

func TestDARetriever_RetrieveFromDA_TwoNamespaces_Success(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	// Prepare header/data blobs
	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 9, addr, pub, signer, 1)
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 9, addr, pub, signer, nil, &data.Data, nil)

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "nsHdr"
	cfg.DA.DataNamespace = "nsData"

	client := mocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte("nsHdr")).Maybe()
	client.On("GetDataNamespace").Return([]byte("nsData")).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	client.On("RetrieveBlobs", mock.Anything, uint64(1234), []byte("nsHdr")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("h1")}, Timestamp: time.Now()},
		Data:       [][]byte{hdrBin},
	}).Once()
	client.On("RetrieveBlobs", mock.Anything, uint64(1234), []byte("nsData")).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: [][]byte{[]byte("d1")}, Timestamp: time.Now()},
		Data:       [][]byte{dataBin},
	}).Once()

	r := newTestDARetriever(t, client, cfg, gen)

	events, derr := r.RetrieveFromDA(context.Background(), 1234)
	require.NoError(t, derr)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(9), events[0].Header.Height())
	assert.Equal(t, uint64(9), events[0].Data.Height())
}

func TestDARetriever_ProcessBlobs_OutsiderDataDoesNotClaimHeight(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	_, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	outsiderBin, _ := makeSignedDataBytes(t, gen.ChainID, 5, nil, outsiderPub, outsiderSigner, 3)
	require.Empty(t, r.processBlobs(context.Background(), [][]byte{outsiderBin}, 10))

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 2)
	hdrBin, hdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data.Data, nil)
	events := r.processBlobs(context.Background(), [][]byte{hdrBin, dataBin}, 11)

	require.Len(t, events, 1)
	require.Equal(t, data.DACommitment(), events[0].Data.DACommitment())
	require.NoError(t, types.Validate(hdr, events[0].Data))
}

func TestDARetriever_ProcessBlobs_OutsiderDataDoesNotReplaceEmptyData(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	_, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	outsiderBin, _ := makeSignedDataBytes(t, gen.ChainID, 5, nil, outsiderPub, outsiderSigner, 3)
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, nil, nil)
	events := r.processBlobs(context.Background(), [][]byte{outsiderBin, hdrBin}, 10)

	require.Len(t, events, 1)
	require.Empty(t, events[0].Data.Txs)
}

func TestDARetriever_ProcessBlobs_OutsiderHeaderDoesNotClaimHeight(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	outsiderAddr, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 2)
	hdrBin, hdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data.Data, nil)
	// Signed by the outsider, and a second one that only copies the proposer address.
	outsiderBin, outsiderHdr := makeSignedHeaderBytes(t, gen.ChainID, 5, outsiderAddr, outsiderPub, outsiderSigner, nil, &data.Data, nil)
	spoofedBin, spoofedHdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, outsiderPub, outsiderSigner, nil, &data.Data, nil)

	require.Empty(t, r.processBlobs(context.Background(), [][]byte{outsiderBin, spoofedBin, dataBin}, 10))
	for _, h := range []*types.SignedHeader{outsiderHdr, spoofedHdr} {
		_, included := r.cache.GetHeaderDAIncludedByHash(h.Hash().String())
		require.False(t, included)
	}

	events := r.processBlobs(context.Background(), [][]byte{hdrBin}, 11)
	require.Len(t, events, 1)
	require.Equal(t, hdr.Hash().String(), events[0].Header.Hash().String())
}

func TestDARetriever_ProcessBlobs_SyncedHeightMarksOnlyStoredBlock(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	outsiderAddr, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 2)
	hdrBin, hdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data.Data, nil)
	fakeDataBin, fakeData := makeSignedDataBytes(t, gen.ChainID, 5, outsiderAddr, outsiderPub, outsiderSigner, 3)
	fakeHdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 5, outsiderAddr, outsiderPub, outsiderSigner, nil, &fakeData.Data, nil)

	r.setSyncedHeaderProvider(func(height uint64) (*types.SignedHeader, bool, error) {
		return hdr, height <= 5, nil
	})

	// A self-consistent block from an outsider for a height the node holds.
	require.Empty(t, r.processBlobs(context.Background(), [][]byte{fakeHdrBin, fakeDataBin}, 10))
	_, headerIncluded := r.cache.GetHeaderDAIncludedByHeight(5)
	_, dataIncluded := r.cache.GetDataDAIncludedByHeight(5)
	require.False(t, headerIncluded)
	require.False(t, dataIncluded)
	require.Empty(t, r.pendingHeaders)
	require.Empty(t, r.pendingData)

	// The stored block is still reported; the syncer skips it as already processed.
	events := r.processBlobs(context.Background(), [][]byte{hdrBin, dataBin}, 11)
	require.Len(t, events, 1)
	require.Equal(t, hdr.Hash().String(), events[0].Header.Hash().String())
	daHeight, headerIncluded := r.cache.GetHeaderDAIncludedByHash(hdr.Hash().String())
	require.True(t, headerIncluded)
	require.Equal(t, uint64(11), daHeight)
	_, dataIncluded = r.cache.GetDataDAIncludedByHash(data.Data.DACommitment().String())
	require.True(t, dataIncluded)
}

// resignHeader reposts a real header under another key: same header hash and
// header signature, different signer. It returns the copy as a legacy blob and
// wrapped in an envelope that key signed.
func resignHeader(t *testing.T, real *types.SignedHeader, pub crypto.PubKey, signer signerpkg.Signer) [][]byte {
	t.Helper()
	addr, err := signer.GetAddress()
	require.NoError(t, err)
	copied := &types.SignedHeader{Header: real.Header, Signature: real.Signature, Signer: types.Signer{PubKey: pub, Address: addr}}
	payload, err := copied.MarshalBinary()
	require.NoError(t, err)
	envelopeSig, err := signer.Sign(t.Context(), payload)
	require.NoError(t, err)
	envelope, err := copied.MarshalDAEnvelope(envelopeSig)
	require.NoError(t, err)
	return [][]byte{payload, envelope}
}

func TestDARetriever_ProcessBlobs_ResignedHeaderCopyDoesNotReplaceRealHeader(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	_, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 2)
	hdrBin, hdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data.Data, nil)
	copies := resignHeader(t, hdr, outsiderPub, outsiderSigner)

	require.Empty(t, r.processBlobs(context.Background(), append(copies, dataBin), 10))
	require.False(t, r.strictMode)

	events := r.processBlobs(context.Background(), [][]byte{hdrBin}, 11)
	require.Len(t, events, 1)
	require.Equal(t, addr, types.KeyAddress(events[0].Header.Signer.PubKey))
}

func TestDARetriever_ProcessBlobs_SyncedHeightIgnoresResignedHeaderCopy(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	_, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	_, hdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, nil, nil)
	r.setSyncedHeaderProvider(func(height uint64) (*types.SignedHeader, bool, error) {
		return hdr, height <= 5, nil
	})

	require.Empty(t, r.processBlobs(context.Background(), resignHeader(t, hdr, outsiderPub, outsiderSigner), 10))
	_, included := r.cache.GetHeaderDAIncludedByHeight(5)
	require.False(t, included)
	require.False(t, r.strictMode)
}

func TestDARetriever_ProcessBlobs_SyncedHeightRejectsDataWithWrongMetadata(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	_, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	_, data := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 2)
	_, hdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data.Data, nil)
	r.setSyncedHeaderProvider(func(height uint64) (*types.SignedHeader, bool, error) {
		return hdr, height <= 5, nil
	})

	// Same transactions as the stored block, different timestamp.
	wrongTimeBin, _ := makeSignedDataBytesWithTime(t, gen.ChainID, 5, nil, outsiderPub, outsiderSigner, 2, uint64(data.Time().UnixNano())+1)
	require.Empty(t, r.processBlobs(context.Background(), [][]byte{wrongTimeBin}, 10))
	_, included := r.cache.GetDataDAIncludedByHeight(5)
	require.False(t, included)
}

func TestDARetriever_ProcessBlobs_PrefersDataSignedByProposer(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	_, outsiderPub, outsiderSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 2)
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data.Data, nil)

	// The real transactions and metadata the header checks, with a forged LastDataHash.
	forged := types.Data{Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 5, Time: uint64(data.Time().UnixNano()), LastDataHash: types.Hash("forged")}, Txs: data.Txs}
	payload, err := forged.MarshalBinary()
	require.NoError(t, err)
	sig, err := outsiderSigner.Sign(t.Context(), payload)
	require.NoError(t, err)
	forgedBin, err := (&types.SignedData{Data: forged, Signature: sig, Signer: types.Signer{PubKey: outsiderPub}}).MarshalBinary()
	require.NoError(t, err)

	events := r.processBlobs(context.Background(), [][]byte{forgedBin, dataBin, hdrBin}, 10)
	require.Len(t, events, 1)
	require.Equal(t, data.Data.Hash(), events[0].Data.Hash())
}

func TestDARetriever_ProcessBlobs_KeepsCandidatesWhenSyncedHeaderLoadFails(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	hdrBin, hdr := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, nil, nil)
	loadErr := errors.New("store unavailable")
	r.setSyncedHeaderProvider(func(uint64) (*types.SignedHeader, bool, error) {
		return hdr, true, loadErr
	})

	require.Empty(t, r.processBlobs(context.Background(), [][]byte{hdrBin}, 10))
	_, included := r.cache.GetHeaderDAIncludedByHeight(5)
	require.False(t, included)

	loadErr = nil
	require.Len(t, r.processBlobs(context.Background(), nil, 11), 1)
	_, included = r.cache.GetHeaderDAIncludedByHeight(5)
	require.True(t, included)
}

func TestDARetriever_ProcessBlobs_CrossDAHeightMatching(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	// Create header and data for the same block height but from different DA heights
	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 2)
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data.Data, nil)

	// Process header from DA height 100 first
	events1 := r.processBlobs(context.Background(), [][]byte{hdrBin}, 100)
	require.Len(t, events1, 0, "should not create event yet - data is missing")

	// Verify header is stored in pending headers
	require.Contains(t, r.pendingHeaders, uint64(5), "header should be stored as pending")

	// Process data from DA height 102
	events2 := r.processBlobs(context.Background(), [][]byte{dataBin}, 102)
	require.Len(t, events2, 1, "should create event when matching data arrives")

	event := events2[0]
	assert.Equal(t, uint64(5), event.Header.Height())
	assert.Equal(t, uint64(5), event.Data.Height())
	assert.Equal(t, uint64(102), event.DaHeight, "DaHeight should be the height where data was processed")

	// Verify the header is consumed, while data remains available until the
	// candidate block is accepted by the syncer.
	require.NotContains(t, r.pendingHeaders, uint64(5), "header should be removed from pending")
	require.Contains(t, r.pendingData, uint64(5), "data should remain pending until accepted")

	r.removePendingData(5)
	require.NotContains(t, r.pendingData, uint64(5), "accepted data should be removed from pending")
}

func TestDARetriever_ProcessBlobs_KeepsDataForLaterHeaderAfterCandidateEvent(t *testing.T) {
	expectedAddr, expectedPub, expectedSigner := buildSyncTestSigner(t)
	wrongAddr, wrongPub, wrongSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: expectedAddr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, expectedAddr, expectedPub, expectedSigner, 2)
	wrongHeaderBin, wrongHeader := makeSignedHeaderBytes(t, gen.ChainID, 5, wrongAddr, wrongPub, wrongSigner, nil, &data.Data, nil)
	correctHeaderBin, correctHeader := makeSignedHeaderBytes(t, gen.ChainID, 5, expectedAddr, expectedPub, expectedSigner, nil, &data.Data, nil)

	events := r.processBlobs(context.Background(), [][]byte{wrongHeaderBin, dataBin}, 100)
	require.Empty(t, events, "a header from an unexpected proposer must not be emitted")
	_, wrongIncluded := r.cache.GetHeaderDAIncludedByHash(wrongHeader.Hash().String())
	require.False(t, wrongIncluded, "a header from an unexpected proposer must not be marked DA included")
	require.Contains(t, r.pendingData, uint64(5), "data should stay available for the expected proposer's header")

	events = r.processBlobs(context.Background(), [][]byte{correctHeaderBin}, 101)
	require.Len(t, events, 1)
	require.Equal(t, correctHeader.Hash().String(), events[0].Header.Hash().String())
	require.Equal(t, data.Data.DACommitment().String(), events[0].Data.DACommitment().String())
}

func TestDARetriever_ProcessBlobs_RejectsUnexpectedProposerBeforePendingHeader(t *testing.T) {
	expectedAddr, expectedPub, expectedSigner := buildSyncTestSigner(t)
	wrongAddr, wrongPub, wrongSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: expectedAddr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)
	r.setExpectedProposerProvider(func(height uint64) ([]byte, bool) {
		if height != 5 {
			return nil, false
		}
		return expectedAddr, true
	})

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, expectedAddr, expectedPub, expectedSigner, 1)
	wrongHeaderBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 5, wrongAddr, wrongPub, wrongSigner, nil, &data.Data, nil)
	correctHeaderBin, correctHeader := makeSignedHeaderBytes(t, gen.ChainID, 5, expectedAddr, expectedPub, expectedSigner, nil, &data.Data, nil)

	events := r.processBlobs(context.Background(), [][]byte{wrongHeaderBin}, 100)
	require.Empty(t, events)

	events = r.processBlobs(context.Background(), [][]byte{correctHeaderBin, dataBin}, 101)
	require.Len(t, events, 1, "an unexpected proposer's header must not block the expected one")
	require.Equal(t, correctHeader.Hash().String(), events[0].Header.Hash().String())
}

func TestDARetriever_ProcessBlobs_ReplacesFutureHeaderOnceExpectedProposerIsKnown(t *testing.T) {
	expectedAddr, expectedPub, expectedSigner := buildSyncTestSigner(t)
	wrongAddr, wrongPub, wrongSigner := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: expectedAddr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)
	expectedKnown := false
	r.setExpectedProposerProvider(func(height uint64) ([]byte, bool) {
		if height != 5 || !expectedKnown {
			return nil, false
		}
		return expectedAddr, true
	})

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 5, expectedAddr, expectedPub, expectedSigner, 1)
	wrongHeaderBin, wrongHeader := makeSignedHeaderBytes(t, gen.ChainID, 5, wrongAddr, wrongPub, wrongSigner, nil, &data.Data, nil)
	correctHeaderBin, correctHeader := makeSignedHeaderBytes(t, gen.ChainID, 5, expectedAddr, expectedPub, expectedSigner, nil, &data.Data, nil)

	events := r.processBlobs(context.Background(), [][]byte{wrongHeaderBin}, 100)
	require.Empty(t, events)
	require.Equal(t, wrongHeader.Hash().String(), r.pendingHeaders[5][0].header.Hash().String())

	expectedKnown = true
	events = r.processBlobs(context.Background(), [][]byte{correctHeaderBin, dataBin}, 101)
	require.Len(t, events, 1)
	require.Equal(t, correctHeader.Hash().String(), events[0].Header.Hash().String())
	require.Equal(t, data.Data.DACommitment().String(), events[0].Data.DACommitment().String())
}

func TestDARetriever_ProcessBlobs_MultipleHeadersCrossDAHeightMatching(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	// Create multiple headers and data for different block heights
	data3Bin, data3 := makeSignedDataBytes(t, gen.ChainID, 3, addr, pub, signer, 1)
	data4Bin, data4 := makeSignedDataBytes(t, gen.ChainID, 4, addr, pub, signer, 2)
	data5Bin, data5 := makeSignedDataBytes(t, gen.ChainID, 5, addr, pub, signer, 1)

	hdr3Bin, _ := makeSignedHeaderBytes(t, gen.ChainID, 3, addr, pub, signer, nil, &data3.Data, nil)
	hdr4Bin, _ := makeSignedHeaderBytes(t, gen.ChainID, 4, addr, pub, signer, nil, &data4.Data, nil)
	hdr5Bin, _ := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, &data5.Data, nil)

	// Process multiple headers from DA height 200 - should be stored as pending
	events1 := r.processBlobs(context.Background(), [][]byte{hdr3Bin, hdr4Bin, hdr5Bin}, 200)
	require.Len(t, events1, 0, "should not create events yet - all data is missing")

	// Verify all headers are stored in pending
	require.Contains(t, r.pendingHeaders, uint64(3), "header 3 should be pending")
	require.Contains(t, r.pendingHeaders, uint64(4), "header 4 should be pending")
	require.Contains(t, r.pendingHeaders, uint64(5), "header 5 should be pending")

	// Process some data from DA height 203 - should create partial events
	events2 := r.processBlobs(context.Background(), [][]byte{data3Bin, data5Bin}, 203)
	require.Len(t, events2, 2, "should create events for heights 3 and 5")

	// Sort events by height for consistent testing
	if events2[0].Header.Height() > events2[1].Header.Height() {
		events2[0], events2[1] = events2[1], events2[0]
	}

	// Verify event for height 3
	assert.Equal(t, uint64(3), events2[0].Header.Height())
	assert.Equal(t, uint64(3), events2[0].Data.Height())
	assert.Equal(t, uint64(203), events2[0].DaHeight)

	// Verify event for height 5
	assert.Equal(t, uint64(5), events2[1].Header.Height())
	assert.Equal(t, uint64(5), events2[1].Data.Height())
	assert.Equal(t, uint64(203), events2[1].DaHeight)
	r.removePendingData(3)
	r.removePendingData(5)

	// Verify header 4 is still pending (no matching data yet)
	require.Contains(t, r.pendingHeaders, uint64(4), "header 4 should still be pending")
	require.NotContains(t, r.pendingHeaders, uint64(3), "header 3 should be removed from pending")
	require.NotContains(t, r.pendingHeaders, uint64(5), "header 5 should be removed from pending")

	// Process remaining data from DA height 205
	events3 := r.processBlobs(context.Background(), [][]byte{data4Bin}, 205)
	require.Len(t, events3, 1, "should create event for height 4")

	// Verify final event for height 4
	assert.Equal(t, uint64(4), events3[0].Header.Height())
	assert.Equal(t, uint64(4), events3[0].Data.Height())
	assert.Equal(t, uint64(205), events3[0].DaHeight)
	r.removePendingData(4)

	// Verify all pending maps are now clear
	require.NotContains(t, r.pendingHeaders, uint64(4), "header 4 should be removed from pending")
	require.Len(t, r.pendingHeaders, 0, "all headers should be processed")
	require.Len(t, r.pendingData, 0, "all data should be processed")
}

func Test_isEmptyDataExpected(t *testing.T) {
	h := &types.SignedHeader{}
	// when DataHash is nil/empty -> expected empty
	assert.True(t, isEmptyDataExpected(h))
	// when equals to predefined emptyTxs hash -> expected empty
	h.DataHash = common.DataHashForEmptyTxs
	assert.True(t, isEmptyDataExpected(h))
}
