package syncing

import (
	"bytes"
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

func TestDARetriever_RetrieveFromDA_Invalid(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	assert.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)

	mockDA.EXPECT().GetIDs(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("just invalid")).Maybe()

	r := NewDARetriever(mockDA, cm, config.DefaultConfig(), genesis.Genesis{}, zerolog.Nop())
	events, err := r.RetrieveFromDA(context.Background(), 42)
	assert.Error(t, err)
	assert.Len(t, events, 0)
}

func TestDARetriever_RetrieveFromDA_NotFound(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	assert.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)

	// GetIDs returns ErrBlobNotFound -> helper maps to StatusNotFound
	mockDA.EXPECT().GetIDs(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("%s: whatever", coreda.ErrBlobNotFound.Error())).Maybe()

	r := NewDARetriever(mockDA, cm, config.DefaultConfig(), genesis.Genesis{}, zerolog.Nop())
	events, err := r.RetrieveFromDA(context.Background(), 42)
	assert.True(t, errors.Is(err, coreda.ErrBlobNotFound))
	assert.Len(t, events, 0)
}

func TestDARetriever_RetrieveFromDA_HeightFromFuture(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)
	// GetIDs returns ErrHeightFromFuture -> helper maps to StatusHeightFromFuture, fetchBlobs returns error
	mockDA.EXPECT().GetIDs(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("%s: later", coreda.ErrHeightFromFuture.Error())).Maybe()

	r := NewDARetriever(mockDA, cm, config.DefaultConfig(), genesis.Genesis{}, zerolog.Nop())
	events, derr := r.RetrieveFromDA(context.Background(), 1000)
	assert.Error(t, derr)
	assert.True(t, errors.Is(derr, coreda.ErrHeightFromFuture))
	assert.Nil(t, events)
}

func TestDARetriever_RetrieveFromDA_Timeout(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)

	// Mock GetIDs to hang longer than the timeout
	mockDA.EXPECT().GetIDs(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, height uint64, namespace []byte) {
			<-ctx.Done()
		}).
		Return(nil, context.DeadlineExceeded).Maybe()

	r := NewDARetriever(mockDA, cm, config.DefaultConfig(), genesis.Genesis{}, zerolog.Nop())

	start := time.Now()
	events, err := r.RetrieveFromDA(context.Background(), 42)
	duration := time.Since(start)

	// Verify error is returned and contains deadline exceeded information
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DA retrieval failed")
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Len(t, events, 0)

	// Verify timeout occurred approximately at expected time (with some tolerance)
	assert.Greater(t, duration, 9*time.Second, "should timeout after approximately 10 seconds")
	assert.Less(t, duration, 12*time.Second, "should not take much longer than timeout")
}

func TestDARetriever_RetrieveFromDA_TimeoutFast(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)

	// Mock GetIDs to immediately return context deadline exceeded
	mockDA.EXPECT().GetIDs(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, context.DeadlineExceeded).Maybe()

	r := NewDARetriever(mockDA, cm, config.DefaultConfig(), genesis.Genesis{}, zerolog.Nop())

	events, err := r.RetrieveFromDA(context.Background(), 42)

	// Verify error is returned and contains deadline exceeded information
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DA retrieval failed")
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Len(t, events, 0)
}

func TestDARetriever_ProcessBlobs_HeaderAndData_Success(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := NewDARetriever(nil, cm, config.DefaultConfig(), gen, zerolog.Nop())

	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 2, addr, pub, signer, 2)
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 2, addr, pub, signer, nil, &data.Data, nil)

	events := r.processBlobs(context.Background(), [][]byte{hdrBin, dataBin}, 77)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(2), events[0].Header.Height())
	assert.Equal(t, uint64(2), events[0].Data.Height())
	assert.Equal(t, uint64(77), events[0].DaHeight)

	hHeight, ok := r.cache.GetHeaderDAIncluded(events[0].Header.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(77), hHeight)

	dHeight, ok := r.cache.GetDataDAIncluded(events[0].Data.DACommitment().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(77), dHeight)
}

func TestDARetriever_ProcessBlobs_HeaderOnly_EmptyDataExpected(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := NewDARetriever(nil, cm, config.DefaultConfig(), gen, zerolog.Nop())

	// Header with no data hash present should trigger empty data creation (per current logic)
	hb, _ := makeSignedHeaderBytes(t, gen.ChainID, 3, addr, pub, signer, nil, nil, nil)

	events := r.processBlobs(context.Background(), [][]byte{hb}, 88)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(3), events[0].Header.Height())
	assert.NotNil(t, events[0].Data)
	assert.Equal(t, uint64(88), events[0].DaHeight)

	hHeight, ok := r.cache.GetHeaderDAIncluded(events[0].Header.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(88), hHeight)

	// empty data is not marked as data included (the submitter components does handle the empty data case)
	_, ok = r.cache.GetDataDAIncluded(events[0].Data.DACommitment().String())
	assert.False(t, ok)
}

func TestDARetriever_TryDecodeHeaderAndData_Basic(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := NewDARetriever(nil, cm, config.DefaultConfig(), gen, zerolog.Nop())

	hb, sh := makeSignedHeaderBytes(t, gen.ChainID, 5, addr, pub, signer, nil, nil, nil)
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

func TestDARetriever_tryDecodeData_InvalidSignatureOrProposer(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	goodAddr, pub, signer := buildSyncTestSigner(t)
	badAddr := []byte("not-the-proposer")
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: badAddr}
	r := NewDARetriever(nil, cm, config.DefaultConfig(), gen, zerolog.Nop())

	// Signed data is made by goodAddr; retriever expects badAddr -> should be rejected
	db, _ := makeSignedDataBytes(t, gen.ChainID, 7, goodAddr, pub, signer, 1)
	assert.Nil(t, r.tryDecodeData(db, 55))
}

func TestDARetriever_validateBlobResponse(t *testing.T) {
	r := &daRetriever{logger: zerolog.Nop()}
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
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	// Prepare header/data blobs
	dataBin, data := makeSignedDataBytes(t, gen.ChainID, 9, addr, pub, signer, 1)
	hdrBin, _ := makeSignedHeaderBytes(t, gen.ChainID, 9, addr, pub, signer, nil, &data.Data, nil)

	cfg := config.DefaultConfig()
	cfg.DA.Namespace = "nsHdr"
	cfg.DA.DataNamespace = "nsData"

	namespaceBz := coreda.NamespaceFromString(cfg.DA.GetNamespace()).Bytes()
	namespaceDataBz := coreda.NamespaceFromString(cfg.DA.GetDataNamespace()).Bytes()

	mockDA := testmocks.NewMockDA(t)
	// Expect GetIDs for both namespaces
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1234), mock.MatchedBy(func(ns []byte) bool { return bytes.Equal(ns, namespaceBz) })).
		Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("h1")}, Timestamp: time.Now()}, nil).Once()
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool { return bytes.Equal(ns, namespaceBz) })).
		Return([][]byte{hdrBin}, nil).Once()

	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1234), mock.MatchedBy(func(ns []byte) bool { return bytes.Equal(ns, namespaceDataBz) })).
		Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("d1")}, Timestamp: time.Now()}, nil).Once()
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool { return bytes.Equal(ns, namespaceDataBz) })).
		Return([][]byte{dataBin}, nil).Once()

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	events, derr := r.RetrieveFromDA(context.Background(), 1234)
	require.NoError(t, derr)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(9), events[0].Header.Height())
	assert.Equal(t, uint64(9), events[0].Data.Height())
}

func TestDARetriever_ProcessBlobs_CrossDAHeightMatching(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := NewDARetriever(nil, cm, config.DefaultConfig(), gen, zerolog.Nop())

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

	// Verify pending maps are cleared
	require.NotContains(t, r.pendingHeaders, uint64(5), "header should be removed from pending")
	require.NotContains(t, r.pendingData, uint64(5), "data should be removed from pending")
}

func TestDARetriever_ProcessBlobs_MultipleHeadersCrossDAHeightMatching(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := NewDARetriever(nil, cm, config.DefaultConfig(), gen, zerolog.Nop())

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

func TestDARetriever_RetrieveForcedIncludedTxsFromDA_Success(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 5678}

	// Prepare forced inclusion transaction data
	dataBin, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 3)

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"
	cfg.DA.ForcedInclusionDAEpoch = 1 // Epoch size of 1

	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	mockDA := testmocks.NewMockDA(t)
	// With DAStartHeight=5678, epoch size=1, daHeight=5678 -> epoch boundaries are [5678, 5678]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(5678), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

	// Fetch epoch start data
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin}, nil).Once()

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 5678)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Txs, 1) // Only fetched once since start == end
	assert.Equal(t, dataBin, result.Txs[0])
}

func TestDARetriever_FetchForcedIncludedTxs_NoNamespaceConfigured(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 0}

	cfg := config.DefaultConfig()
	// Leave ForcedInclusionNamespace empty

	r := NewDARetriever(nil, cm, cfg, gen, zerolog.Nop())

	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 1234)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestDARetriever_FetchForcedIncludedTxs_NotFound(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 9999}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"
	cfg.DA.ForcedInclusionDAEpoch = 1 // Epoch size of 1

	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	mockDA := testmocks.NewMockDA(t)
	// With DAStartHeight=9999, epoch size=1, daHeight=9999 -> epoch boundaries are [9999, 9999]
	// Check epoch start only (end check is skipped when same as start)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(9999), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 9999)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Txs)
}

func TestDARetriever_RetrieveForcedIncludedTxsFromDA_ExceedsMaxBlobSize(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 1000}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"
	cfg.DA.ForcedInclusionDAEpoch = 3 // Epoch size of 3

	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	// Create signed data blobs that will exceed DefaultMaxBlobSize when accumulated
	// DefaultMaxBlobSize is 1.5MB = 1,572,864 bytes
	// Each 700KB tx becomes ~719KB blob, so 2 blobs = ~1.44MB (fits), 3 blobs = ~2.16MB (exceeds)
	d1 := &types.Data{
		Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 10, Time: uint64(time.Now().UnixNano())},
		Txs:      make(types.Txs, 1),
	}
	d1.Txs[0] = make([]byte, 700*1024) // 700KB transaction

	payload1, err := d1.MarshalBinary()
	require.NoError(t, err)
	sig1, err := signer.Sign(payload1)
	require.NoError(t, err)
	sd1 := &types.SignedData{Data: *d1, Signature: sig1, Signer: types.Signer{PubKey: pub, Address: addr}}
	dataBin1, err := sd1.MarshalBinary()
	require.NoError(t, err)

	d2 := &types.Data{
		Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 11, Time: uint64(time.Now().UnixNano())},
		Txs:      make(types.Txs, 1),
	}
	d2.Txs[0] = make([]byte, 700*1024) // 700KB transaction

	payload2, err := d2.MarshalBinary()
	require.NoError(t, err)
	sig2, err := signer.Sign(payload2)
	require.NoError(t, err)
	sd2 := &types.SignedData{Data: *d2, Signature: sig2, Signer: types.Signer{PubKey: pub, Address: addr}}
	dataBin2, err := sd2.MarshalBinary()
	require.NoError(t, err)

	d3 := &types.Data{
		Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 12, Time: uint64(time.Now().UnixNano())},
		Txs:      make(types.Txs, 1),
	}
	d3.Txs[0] = make([]byte, 700*1024) // 700KB transaction

	payload3, err := d3.MarshalBinary()
	require.NoError(t, err)
	sig3, err := signer.Sign(payload3)
	require.NoError(t, err)
	sd3 := &types.SignedData{Data: *d3, Signature: sig3, Signer: types.Signer{PubKey: pub, Address: addr}}
	dataBin3, err := sd3.MarshalBinary()
	require.NoError(t, err)

	mockDA := testmocks.NewMockDA(t)

	// With DAStartHeight=1000, epoch size=3, daHeight=1000 -> epoch boundaries are [1000, 1002]
	// Check epoch start
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1000), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

	// Check epoch end
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1002), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi3")}, Timestamp: time.Now()}, nil).Once()

	// Fetch epoch start data
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin1}, nil).Once()

	// Second height in epoch - should succeed
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1001), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi2")}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin2}, nil).Once()

	// Fetch epoch end data - should be retrieved but skipped due to size limit
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin3}, nil).Once()

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 1000)

	// Should succeed but skip the third blob due to size limit (using continue)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should only have 2 transactions, third is skipped due to size
	require.Len(t, result.Txs, 2)
	assert.Equal(t, dataBin1, result.Txs[0])
	assert.Equal(t, dataBin2, result.Txs[1])

	// Verify total size is within limits
	totalSize := len(dataBin1) + len(dataBin2)
	assert.LessOrEqual(t, totalSize, int(common.DefaultMaxBlobSize))

	// Verify that adding the third would have exceeded the limit
	totalSizeWithThird := totalSize + len(dataBin3)
	assert.Greater(t, totalSizeWithThird, int(common.DefaultMaxBlobSize))
}

func TestDARetriever_RetrieveForcedIncludedTxsFromDA_NotAtEpochStart(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 100}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"
	cfg.DA.ForcedInclusionDAEpoch = 10

	mockDA := testmocks.NewMockDA(t)

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	// With DAStartHeight=100, epoch size=10, daHeight=105 -> epoch boundaries are [100, 109]
	// But daHeight=105 is NOT the epoch start, so it should be a no-op
	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 105)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Txs)
	require.Equal(t, uint64(105), result.StartDaHeight)
	require.Equal(t, uint64(105), result.EndDaHeight)
}

func TestDARetriever_RetrieveForcedIncludedTxsFromDA_EpochStartFromFuture(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 1000}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"
	cfg.DA.ForcedInclusionDAEpoch = 10

	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	mockDA := testmocks.NewMockDA(t)
	// With DAStartHeight=1000, epoch size=10, daHeight=1000 -> epoch boundaries are [1000, 1009]
	// Mock that height 1000 (epoch start) is from the future
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1000), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(nil, fmt.Errorf("%s: not yet available", coreda.ErrHeightFromFuture.Error())).Once()

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 1000)
	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, coreda.ErrHeightFromFuture))
	require.Contains(t, err.Error(), "epoch start height 1000 not yet available")
}

func TestDARetriever_RetrieveForcedIncludedTxsFromDA_EpochEndFromFuture(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, _, _ := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 1000}

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"
	cfg.DA.ForcedInclusionDAEpoch = 10

	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	mockDA := testmocks.NewMockDA(t)
	// With DAStartHeight=1000, epoch size=10, daHeight=1000 -> epoch boundaries are [1000, 1009]
	// Epoch start is available but epoch end (1009) is from the future
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1000), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().GetIDs(mock.Anything, uint64(1009), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(nil, fmt.Errorf("%s: not yet available", coreda.ErrHeightFromFuture.Error())).Once()

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 1000)
	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, errors.Is(err, coreda.ErrHeightFromFuture))
	require.Contains(t, err.Error(), "epoch end height 1009 not yet available")
}

func TestDARetriever_RetrieveForcedIncludedTxsFromDA_CompleteEpoch(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	st := store.New(ds)
	cm, err := cache.NewManager(config.DefaultConfig(), st, zerolog.Nop())
	require.NoError(t, err)

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr, DAStartHeight: 2000}

	// Prepare forced inclusion transaction data
	dataBin1, _ := makeSignedDataBytes(t, gen.ChainID, 10, addr, pub, signer, 2)
	dataBin2, _ := makeSignedDataBytes(t, gen.ChainID, 11, addr, pub, signer, 1)
	dataBin3, _ := makeSignedDataBytes(t, gen.ChainID, 12, addr, pub, signer, 1)

	cfg := config.DefaultConfig()
	cfg.DA.ForcedInclusionNamespace = "nsForcedInclusion"
	cfg.DA.ForcedInclusionDAEpoch = 3

	namespaceForcedInclusionBz := coreda.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	mockDA := testmocks.NewMockDA(t)

	// With DAStartHeight=2000, epoch size=3, daHeight=2000 -> epoch boundaries are [2000, 2002]
	// All heights available

	// Check epoch start (2000)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(2000), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi1")}, Timestamp: time.Now()}, nil).Once()

	// Check epoch end (2002)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(2002), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi3")}, Timestamp: time.Now()}, nil).Once()

	// Fetch epoch start data
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin1}, nil).Once()

	// Fetch middle height (2001)
	mockDA.EXPECT().GetIDs(mock.Anything, uint64(2001), mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return(&coreda.GetIDsResult{IDs: [][]byte{[]byte("fi2")}, Timestamp: time.Now()}, nil).Once()

	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin2}, nil).Once()

	// Fetch epoch end data
	mockDA.EXPECT().Get(mock.Anything, mock.Anything, mock.MatchedBy(func(ns []byte) bool {
		return bytes.Equal(ns, namespaceForcedInclusionBz)
	})).Return([][]byte{dataBin3}, nil).Once()

	r := NewDARetriever(mockDA, cm, cfg, gen, zerolog.Nop())

	result, err := r.RetrieveForcedIncludedTxsFromDA(context.Background(), 2000)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Txs, 3)
	require.Equal(t, dataBin1, result.Txs[0])
	require.Equal(t, dataBin2, result.Txs[1])
	require.Equal(t, dataBin3, result.Txs[2])
	require.Equal(t, uint64(2000), result.StartDaHeight)
	require.Equal(t, uint64(2002), result.EndDaHeight)
}
