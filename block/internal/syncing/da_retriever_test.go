package syncing

import (
	"context"
	"fmt"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/cache"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
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

	memDS := dssync.MutexWrap(ds.NewMapDatastore())
	st := store.New(memDS)

	cm, err := cache.NewManager(cfg, st, zerolog.Nop())
	require.NoError(t, err)

	if mockClient == nil {
		mockClient = mocks.NewMockClient(t)
	}
	// default namespace helpers
	mockClient.On("GetHeaderNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	mockClient.On("GetDataNamespace").Return([]byte(cfg.DA.Namespace)).Maybe()
	mockClient.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	mockClient.On("HasForcedInclusionNamespace").Return(false).Maybe()

	return NewDARetriever(mockClient, cm, gen, zerolog.Nop())
}

func TestDARetriever_RetrieveFromDA_NotFound(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("GetHeaderNamespace").Return([]byte("test-ns")).Maybe()
	client.On("GetDataNamespace").Return([]byte("test-ns")).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	client.On("RetrieveBlobs", mock.Anything, uint64(42), []byte("test-ns")).Return(datypes.ResultRetrieve{BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound, Message: fmt.Sprintf("%s: whatever", datypes.ErrBlobNotFound.Error())}}).Once()

	r := newTestDARetriever(t, client, config.DefaultConfig(), genesis.Genesis{})
	events, err := r.RetrieveFromDA(context.Background(), 42)
	assert.Error(t, err)
	assert.Len(t, events, 0)
}

func TestDARetriever_ProcessBlobs_CombinedBlock(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}

	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	d := &types.Data{
		Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 2, Time: uint64(time.Now().UnixNano())},
		Txs:      types.Txs{types.Tx([]byte{2, 0}), types.Tx([]byte{2, 1})},
	}

	dataHash := d.DACommitment()
	hdr := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: gen.ChainID, Height: 2, Time: uint64(time.Now().UnixNano())},
			DataHash:        dataHash,
			ProposerAddress: addr,
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}

	hdrBz, err := hdr.MarshalBinary()
	require.NoError(t, err)
	sig, err := signer.Sign(t.Context(), hdrBz)
	require.NoError(t, err)

	blob, err := common.MarshalBlockBlob(hdr, d, sig)
	require.NoError(t, err)

	events := r.processBlobs([][]byte{blob}, 77)
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

func TestDARetriever_ProcessBlobs_EmptyData(t *testing.T) {
	addr, pub, signer := buildSyncTestSigner(t)
	gen := genesis.Genesis{ChainID: "tchain", InitialHeight: 1, StartTime: time.Now().Add(-time.Second), ProposerAddress: addr}
	r := newTestDARetriever(t, nil, config.DefaultConfig(), gen)

	d := &types.Data{
		Metadata: &types.Metadata{ChainID: gen.ChainID, Height: 3, Time: uint64(time.Now().UnixNano())},
	}
	hdr := &types.SignedHeader{
		Header: types.Header{
			BaseHeader:      types.BaseHeader{ChainID: gen.ChainID, Height: 3, Time: uint64(time.Now().UnixNano())},
			DataHash:        common.DataHashForEmptyTxs,
			ProposerAddress: addr,
		},
		Signer: types.Signer{PubKey: pub, Address: addr},
	}

	hdrBz, err := hdr.MarshalBinary()
	require.NoError(t, err)
	sig, err := signer.Sign(t.Context(), hdrBz)
	require.NoError(t, err)

	blob, err := common.MarshalBlockBlob(hdr, d, sig)
	require.NoError(t, err)

	events := r.processBlobs([][]byte{blob}, 88)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(3), events[0].Header.Height())
	assert.NotNil(t, events[0].Data)
	assert.Equal(t, uint64(88), events[0].DaHeight)
}
