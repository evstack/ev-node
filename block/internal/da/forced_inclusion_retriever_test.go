package da_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"gotest.tools/v3/assert"

	da "github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/block/internal/da/testclient"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
)

func TestNewForcedInclusionRetriever(t *testing.T) {
	client := testclient.NewMockClient(t)
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(datypes.NamespaceFromString("test-fi-ns").Bytes()).Maybe()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	assert.Assert(t, retriever != nil)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoNamespace(t *testing.T) {
	client := testclient.NewMockClient(t)
	client.On("HasForcedInclusionNamespace").Return(false).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not configured")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NotAtEpochStart(t *testing.T) {
	client := testclient.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Once()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	// Height 105 is not an epoch start (100, 110, 120, etc. are epoch starts)
	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 105)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)
	assert.Equal(t, event.StartDaHeight, uint64(105))
	assert.Equal(t, event.EndDaHeight, uint64(105))
	assert.Equal(t, len(event.Txs), 0)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_EpochStartSuccess(t *testing.T) {
	testBlobs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
		[]byte("tx3"),
	}

	client := testclient.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Once()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	client.On("RetrieveForcedInclusion", mock.Anything, mock.Anything).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: []datypes.ID{[]byte("id1"), []byte("id2"), []byte("id3")}, Timestamp: time.Now()},
		Data:       testBlobs,
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	// Height 100 is an epoch start
	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)
	assert.Equal(t, event.StartDaHeight, uint64(100))
	assert.Equal(t, event.EndDaHeight, uint64(100))
	assert.Equal(t, len(event.Txs), len(testBlobs))
	assert.DeepEqual(t, event.Txs[0], testBlobs[0])
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_EpochStartNotAvailable(t *testing.T) {
	client := testclient.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Once()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	client.On("RetrieveForcedInclusion", mock.Anything, uint64(109)).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	// Epoch boundaries: [100, 109] - retrieval happens at epoch end (109)
	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 109)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not yet available")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoBlobsAtHeight(t *testing.T) {
	client := testclient.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Once()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	client.On("RetrieveForcedInclusion", mock.Anything, uint64(100)).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)
	assert.Equal(t, len(event.Txs), 0)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_MultiHeightEpoch(t *testing.T) {
	testBlobsByHeight := map[uint64][][]byte{
		100: {[]byte("tx1"), []byte("tx2")},
		101: {[]byte("tx3")},
		102: {[]byte("tx4"), []byte("tx5"), []byte("tx6")},
	}

	client := testclient.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Once()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	client.On("RetrieveForcedInclusion", mock.Anything, uint64(102)).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[102],
	}).Once()
	client.On("RetrieveForcedInclusion", mock.Anything, uint64(100)).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[100],
	}).Once()
	client.On("RetrieveForcedInclusion", mock.Anything, uint64(101)).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[101],
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 3, // Epoch: 100-102
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	// Epoch boundaries: [100, 102] - retrieval happens at epoch end (102)
	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 102)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)
	assert.Equal(t, event.StartDaHeight, uint64(100))
	assert.Equal(t, event.EndDaHeight, uint64(102))

	// Should have collected all txs from all heights
	expectedTxCount := len(testBlobsByHeight[100]) + len(testBlobsByHeight[101]) + len(testBlobsByHeight[102])
	assert.Equal(t, len(event.Txs), expectedTxCount)
}
