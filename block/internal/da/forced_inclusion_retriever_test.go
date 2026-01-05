package da

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"gotest.tools/v3/assert"

	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/test/mocks"
)

// createTestAsyncFetcher creates a minimal async fetcher for tests (without starting it)
func createTestAsyncFetcher(client Client, gen genesis.Genesis) *AsyncEpochFetcher {
	return NewAsyncEpochFetcher(
		client,
		zerolog.Nop(),
		gen.DAStartHeight,
		gen.DAEpochForcedInclusion,
		1,             // prefetch 1 epoch
		1*time.Second, // poll interval (doesn't matter for tests)
	)
}

func TestNewForcedInclusionRetriever(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(datypes.NamespaceFromString("test-fi-ns").Bytes()).Maybe()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)
	assert.Assert(t, retriever != nil)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoNamespace(t *testing.T) {
	client := mocks.NewMockClient(t)
	client.On("HasForcedInclusionNamespace").Return(false).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)
	ctx := context.Background()

	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not configured")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NotAtEpochStart(t *testing.T) {
	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Once()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)
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
	client.On("Retrieve", mock.Anything, mock.Anything, fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusSuccess, IDs: []datypes.ID{[]byte("id1"), []byte("id2"), []byte("id3")}, Timestamp: time.Now()},
		Data:       testBlobs,
	}).Maybe()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)
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
	client.On("Retrieve", mock.Anything, uint64(109), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusHeightFromFuture},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)
	ctx := context.Background()

	// Epoch boundaries: [100, 109] - retrieval happens at epoch end (109)
	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 109)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not yet available")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoBlobsAtHeight(t *testing.T) {
	client := mocks.NewMockClient(t)
	fiNs := datypes.NamespaceFromString("test-fi-ns").Bytes()
	client.On("HasForcedInclusionNamespace").Return(true).Maybe()
	client.On("GetForcedInclusionNamespace").Return(fiNs).Maybe()
	client.On("Retrieve", mock.Anything, uint64(100), fiNs).Return(datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{Code: datypes.StatusNotFound},
	}).Once()

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)
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

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)
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

func TestForcedInclusionRetriever_processForcedInclusionBlobs(t *testing.T) {
	client := mocks.NewMockClient(t)

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	asyncFetcher := createTestAsyncFetcher(client, gen)
	retriever := NewForcedInclusionRetriever(client, zerolog.Nop(), gen.DAStartHeight, gen.DAEpochForcedInclusion, asyncFetcher)

	tests := []struct {
		name            string
		result          datypes.ResultRetrieve
		height          uint64
		expectedTxCount int
		expectError     bool
	}{
		{
			name: "success with blobs",
			result: datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code: datypes.StatusSuccess,
				},
				Data: [][]byte{[]byte("tx1"), []byte("tx2")},
			},
			height:          100,
			expectedTxCount: 2,
			expectError:     false,
		},
		{
			name: "not found",
			result: datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code: datypes.StatusNotFound,
				},
			},
			height:          100,
			expectedTxCount: 0,
			expectError:     false,
		},
		{
			name: "error status",
			result: datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusError,
					Message: "test error",
				},
			},
			height:      100,
			expectError: true,
		},
		{
			name: "empty blobs are skipped",
			result: datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code: datypes.StatusSuccess,
				},
				Data: [][]byte{[]byte("tx1"), {}, []byte("tx2")},
			},
			height:          100,
			expectedTxCount: 2,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &ForcedInclusionEvent{
				Txs: [][]byte{},
			}

			err := retriever.processForcedInclusionBlobs(event, tt.result, tt.height)

			if tt.expectError {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, len(event.Txs), tt.expectedTxCount)
				assert.Equal(t, event.Timestamp, time.Time{})
			}
		})
	}
}
