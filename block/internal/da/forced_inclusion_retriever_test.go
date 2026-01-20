package da

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"gotest.tools/v3/assert"

	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
)

// mockSubscribe sets up a Subscribe mock that returns a channel that blocks until context is cancelled.
func mockSubscribe(client *mocks.MockClient) {
	client.On("Subscribe", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, _ []byte) <-chan *blobrpc.SubscriptionResponse {
			ch := make(chan *blobrpc.SubscriptionResponse)
			go func() {
				<-ctx.Done()
				close(ch)
			}()
			return ch
		},
		nil,
	).Maybe()
	client.On("LocalHead", mock.Anything).Return(uint64(0), nil).Maybe()
}

func TestNewForcedInclusionRetriever(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("GetForcedInclusionNamespace").Return(datypes.NamespaceFromString("test-fi-ns").Bytes()).Maybe()
	mockSubscribe(client)

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	assert.Assert(t, retriever != nil)
	retriever.Stop()
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoNamespace(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("HasForcedInclusionNamespace").Return(false).Maybe()
	client.On("GetForcedInclusionNamespace").Return([]byte(nil)).Maybe()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
	ctx := context.Background()

	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not configured")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NotAtEpochStart(t *testing.T) {
	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
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

	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)
	client.On("Retrieve", mock.Anything, mock.Anything, fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: []datypes.ID{[]byte("id1"), []byte("id2"), []byte("id3")}, Timestamp: time.Now()},
		Data:       testBlobs,
	}).Maybe()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
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
	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)

	// Mock the first height in epoch as not available
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
	ctx := context.Background()

	// Epoch boundaries: [100, 109] - now tries to fetch all blocks in epoch
	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 109)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not yet available")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoBlobsAtHeight(t *testing.T) {
	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
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

	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)
	client.On("Retrieve", mock.Anything, uint64(102), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[102],
	}).Once()
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[100],
	}).Once()
	client.On("Retrieve", mock.Anything, uint64(101), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[101],
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 3, // Epoch: 100-102
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
	ctx := context.Background()

	// Epoch boundaries: [100, 102] - retrieval happens at epoch end (102)
	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 102)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)
	assert.Equal(t, event.StartDaHeight, uint64(100))
	assert.Equal(t, event.EndDaHeight, uint64(102))
	assert.Assert(t, event.Timestamp.After(time.Time{}))

	// Should have collected all txs from all heights
	expectedTxCount := len(testBlobsByHeight[100]) + len(testBlobsByHeight[101]) + len(testBlobsByHeight[102])
	assert.Equal(t, len(event.Txs), expectedTxCount)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_ErrorHandling(t *testing.T) {
	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:    datypes.StatusError,
			Message: "test error",
		},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
	ctx := context.Background()

	// Should return error so caller can retry without skipping the epoch
	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.Assert(t, err != nil)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_EmptyBlobsSkipped(t *testing.T) {
	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       [][]byte{[]byte("tx1"), {}, []byte("tx2"), nil, []byte("tx3")},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)
	// Should skip empty and nil blobs
	assert.Equal(t, len(event.Txs), 3)
	assert.DeepEqual(t, event.Txs[0], []byte("tx1"))
	assert.DeepEqual(t, event.Txs[1], []byte("tx2"))
	assert.DeepEqual(t, event.Txs[2], []byte("tx3"))
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_OrderPreserved(t *testing.T) {
	// Test that transactions are returned in height order even when fetched out of order
	testBlobsByHeight := map[uint64][][]byte{
		100: {[]byte("tx-100-1"), []byte("tx-100-2")},
		101: {[]byte("tx-101-1")},
		102: {[]byte("tx-102-1"), []byte("tx-102-2")},
	}

	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	mockSubscribe(client)
	// Return heights out of order to test ordering is preserved
	client.On("Retrieve", mock.Anything, uint64(102), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[102],
	}).Once()
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[100],
	}).Once()
	client.On("Retrieve", mock.Anything, uint64(101), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, Timestamp: time.Now()},
		Data:       testBlobsByHeight[101],
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 3, // Epoch: 100-102
	}

	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), config.DefaultConfig(), gen.DAStartHeight, gen.DAEpochForcedInclusion)
	defer retriever.Stop()
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 102)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)

	// Verify transactions are in height order: 100, 100, 101, 102, 102
	expectedOrder := [][]byte{
		[]byte("tx-100-1"),
		[]byte("tx-100-2"),
		[]byte("tx-101-1"),
		[]byte("tx-102-1"),
		[]byte("tx-102-2"),
	}
	assert.Equal(t, len(event.Txs), len(expectedOrder))
	for i, expected := range expectedOrder {
		assert.DeepEqual(t, event.Txs[i], expected)
	}
}
