//go:build !ignore
// +build !ignore

package syncing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/block/internal/da"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/blob"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	signerpkg "github.com/evstack/ev-node/pkg/signer"
	"github.com/evstack/ev-node/types"
)

// newTestDARetriever creates a DA retriever for testing with the given DA implementation
func newTestDARetriever(t *testing.T, api da.BlobAPI, cfg config.Config, gen genesis.Genesis) *daRetriever {
	t.Helper()
	if cfg.DA.Namespace == "" {
		cfg.DA.Namespace = "test-ns"
	}
	if cfg.DA.DataNamespace == "" {
		cfg.DA.DataNamespace = "test-data-ns"
	}

	cm, err := cache.NewCacheManager(cfg, zerolog.Nop())
	require.NoError(t, err)

	blobAPI := api
	if blobAPI == nil {
		blobAPI = da.NewLocalBlobAPI(common.DefaultMaxBlobSize)
	}
	daClient := da.NewClient(da.Config{
		BlobAPI:        blobAPI,
		Logger:         zerolog.Nop(),
		Namespace:      cfg.DA.Namespace,
		DataNamespace:  cfg.DA.DataNamespace,
		DefaultTimeout: 10 * time.Second,
	})

	return NewDARetriever(daClient, cm, gen, zerolog.Nop())
}

type stubBlobAPI struct {
	submitFn   func(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error)
	getAllFn   func(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error)
	getProofFn func(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error)
	includedFn func(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error)
}

func (s stubBlobAPI) Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error) {
	if s.submitFn != nil {
		return s.submitFn(ctx, blobs, opts)
	}
	return 0, nil
}

func (s stubBlobAPI) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	if s.getAllFn != nil {
		return s.getAllFn(ctx, height, namespaces)
	}
	return []*blob.Blob{}, nil
}

func (s stubBlobAPI) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	if s.getProofFn != nil {
		return s.getProofFn(ctx, height, namespace, commitment)
	}
	return &blob.Proof{}, nil
}

func (s stubBlobAPI) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	if s.includedFn != nil {
		return s.includedFn(ctx, height, namespace, proof, commitment)
	}
	return true, nil
}

// makeSignedDataBytes builds SignedData containing the provided Data and returns its binary encoding
func makeSignedDataBytes(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, txs int) ([]byte, *types.SignedData) {
	return makeSignedDataBytesWithTime(t, chainID, height, proposer, pub, signer, txs, uint64(time.Now().UnixNano()))
}

func makeSignedDataBytesWithTime(t *testing.T, chainID string, height uint64, proposer []byte, pub crypto.PubKey, signer signerpkg.Signer, txs int, timestamp uint64) ([]byte, *types.SignedData) {
	d := &types.Data{Metadata: &types.Metadata{ChainID: chainID, Height: height, Time: timestamp}}
	if txs > 0 {
		d.Txs = make(types.Txs, txs)
		for i := 0; i < txs; i++ {
			d.Txs[i] = types.Tx([]byte{byte(height), byte(i)})
		}
	}

	// For DA SignedData, sign the Data payload bytes (matches DA submission logic)
	payload, _ := d.MarshalBinary()
	sig, err := signer.Sign(payload)
	require.NoError(t, err)
	sd := &types.SignedData{Data: *d, Signature: sig, Signer: types.Signer{PubKey: pub, Address: proposer}}
	bin, err := sd.MarshalBinary()
	require.NoError(t, err)
	return bin, sd
}

func TestDARetriever_RetrieveFromDA_Invalid(t *testing.T) {
	api := stubBlobAPI{
		getAllFn: func(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
			return nil, errors.New("just invalid")
		},
	}

	r := newTestDARetriever(t, api, config.DefaultConfig(), genesis.Genesis{})
	events, err := r.RetrieveFromDA(context.Background(), 42)
	assert.Error(t, err)
	assert.Len(t, events, 0)
}

func TestDARetriever_RetrieveFromDA_NotFound(t *testing.T) {
	api := stubBlobAPI{
		getAllFn: func(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
			return nil, errors.New(datypes.ErrBlobNotFound.Error())
		},
	}

	r := newTestDARetriever(t, api, config.DefaultConfig(), genesis.Genesis{})
	events, err := r.RetrieveFromDA(context.Background(), 42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blob: not found")
	assert.Len(t, events, 0)
}

func TestDARetriever_RetrieveFromDA_HeightFromFuture(t *testing.T) {
	api := stubBlobAPI{
		getAllFn: func(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
			return nil, errors.New(datypes.ErrHeightFromFuture.Error())
		},
	}

	r := newTestDARetriever(t, api, config.DefaultConfig(), genesis.Genesis{})
	events, derr := r.RetrieveFromDA(context.Background(), 1000)
	assert.Error(t, derr)
	assert.Contains(t, derr.Error(), "height from future")
	assert.Nil(t, events)
}

func TestDARetriever_RetrieveFromDA_Timeout(t *testing.T) {
	t.Skip("Skipping flaky timeout test - timing is now controlled by DA client")

	api := stubBlobAPI{
		getAllFn: func(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
			<-ctx.Done()
			return nil, context.DeadlineExceeded
		},
	}

	r := newTestDARetriever(t, api, config.DefaultConfig(), genesis.Genesis{})

	start := time.Now()
	events, err := r.RetrieveFromDA(context.Background(), 42)
	duration := time.Since(start)

	// Verify error is returned and contains deadline exceeded information
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DA retrieval failed")
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Len(t, events, 0)

	// Verify timeout occurred approximately at expected time (with some tolerance)
	// DA client has a 30-second default timeout
	assert.Greater(t, duration, 29*time.Second, "should timeout after approximately 30 seconds")
	assert.Less(t, duration, 35*time.Second, "should not take much longer than timeout")
}

func TestDARetriever_RetrieveFromDA_TimeoutFast(t *testing.T) {

	api := stubBlobAPI{
		getAllFn: func(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
			return nil, context.DeadlineExceeded
		},
	}

	r := newTestDARetriever(t, api, config.DefaultConfig(), genesis.Genesis{})

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

	hHeight, ok := r.cache.GetHeaderDAIncluded(events[0].Header.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(77), hHeight)

	dHeight, ok := r.cache.GetDataDAIncluded(events[0].Data.DACommitment().String())
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

	hHeight, ok := r.cache.GetHeaderDAIncluded(events[0].Header.Hash().String())
	assert.True(t, ok)
	assert.Equal(t, uint64(88), hHeight)

	// empty data is not marked as data included (the submitter components does handle the empty data case)
	_, ok = r.cache.GetDataDAIncluded(events[0].Data.DACommitment().String())
	assert.False(t, ok)
}

func TestDARetriever_TryDecodeHeaderAndData_Basic(t *testing.T) {

	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

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

	goodAddr, pub, signer := buildSyncTestSigner(t)
	badAddr := []byte("not-the-proposer")
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: badAddr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	// Signed data is made by goodAddr; retriever expects badAddr -> should be rejected
	db, _ := makeSignedDataBytes(t, gen.ChainID, 7, goodAddr, pub, signer, 1)
	assert.Nil(t, r.tryDecodeData(db, 55))
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

	namespaceBz := coreda.NamespaceFromString(cfg.DA.GetNamespace()).Bytes()
	namespaceDataBz := coreda.NamespaceFromString(cfg.DA.GetDataNamespace()).Bytes()

	hdrNS, err := share.NewNamespaceFromBytes(namespaceBz)
	require.NoError(t, err)
	dataNS, err := share.NewNamespaceFromBytes(namespaceDataBz)
	require.NoError(t, err)

	hdrBlob, err := blob.NewBlobV0(hdrNS, hdrBin)
	require.NoError(t, err)
	dataBlob, err := blob.NewBlobV0(dataNS, dataBin)
	require.NoError(t, err)

	api := stubBlobAPI{
		getAllFn: func(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
			require.Equal(t, uint64(1234), height)
			require.Len(t, namespaces, 1)
			switch string(namespaces[0].Bytes()) {
			case string(hdrNS.Bytes()):
				return []*blob.Blob{hdrBlob}, nil
			case string(dataNS.Bytes()):
				return []*blob.Blob{dataBlob}, nil
			default:
				return []*blob.Blob{}, nil
			}
		},
	}

	r := newTestDARetriever(t, api, cfg, gen)

	events, derr := r.RetrieveFromDA(context.Background(), 1234)
	require.NoError(t, derr)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(9), events[0].Header.Height())
	assert.Equal(t, uint64(9), events[0].Data.Height())
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

	// Verify pending maps are cleared
	require.NotContains(t, r.pendingHeaders, uint64(5), "header should be removed from pending")
	require.NotContains(t, r.pendingData, uint64(5), "data should be removed from pending")
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
