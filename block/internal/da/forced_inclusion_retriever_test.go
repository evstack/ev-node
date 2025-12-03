package da

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"

	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/genesis"
)

func TestNewForcedInclusionRetriever(t *testing.T) {
	client := NewClient(Config{
		DA:                       &mockDA{},
		Logger:                   zerolog.Nop(),
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	assert.Assert(t, retriever != nil)
	assert.Equal(t, retriever.daEpochSize, uint64(10))
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoNamespace(t *testing.T) {
	client := NewClient(Config{
		DA:            &mockDA{},
		Logger:        zerolog.Nop(),
		Namespace:     "test-ns",
		DataNamespace: "test-data-ns",
		// No forced inclusion namespace
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not configured")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NotAtEpochStart(t *testing.T) {
	client := NewClient(Config{
		DA:                       &mockDA{},
		Logger:                   zerolog.Nop(),
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())
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

	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return &coreda.GetIDsResult{
				IDs:       []coreda.ID{[]byte("id1"), []byte("id2"), []byte("id3")},
				Timestamp: time.Now(),
			}, nil
		},
		getFunc: func(ctx context.Context, ids []coreda.ID, namespace []byte) ([]coreda.Blob, error) {
			return testBlobs, nil
		},
	}

	client := NewClient(Config{
		DA:                       mockDAInstance,
		Logger:                   zerolog.Nop(),
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())
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
	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return nil, coreda.ErrHeightFromFuture
		},
	}

	client := NewClient(Config{
		DA:                       mockDAInstance,
		Logger:                   zerolog.Nop(),
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	// Epoch boundaries: [100, 109] - retrieval happens at epoch end (109)
	_, err := retriever.RetrieveForcedIncludedTxs(ctx, 109)
	assert.Assert(t, err != nil)
	assert.ErrorContains(t, err, "not yet available")
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoBlobsAtHeight(t *testing.T) {
	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return nil, coreda.ErrBlobNotFound
		},
	}

	client := NewClient(Config{
		DA:                       mockDAInstance,
		Logger:                   zerolog.Nop(),
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 1, // Single height epoch
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	ctx := context.Background()

	event, err := retriever.RetrieveForcedIncludedTxs(ctx, 100)
	assert.NilError(t, err)
	assert.Assert(t, event != nil)
	assert.Equal(t, len(event.Txs), 0)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_MultiHeightEpoch(t *testing.T) {
	callCount := 0
	testBlobsByHeight := map[uint64][][]byte{
		100: {[]byte("tx1"), []byte("tx2")},
		101: {[]byte("tx3")},
		102: {[]byte("tx4"), []byte("tx5"), []byte("tx6")},
	}

	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			callCount++
			blobs, exists := testBlobsByHeight[height]
			if !exists {
				return nil, coreda.ErrBlobNotFound
			}
			ids := make([]coreda.ID, len(blobs))
			for i := range blobs {
				ids[i] = []byte("id")
			}
			return &coreda.GetIDsResult{
				IDs:       ids,
				Timestamp: time.Now(),
			}, nil
		},
		getFunc: func(ctx context.Context, ids []coreda.ID, namespace []byte) ([]coreda.Blob, error) {
			// Return blobs based on current call count
			switch callCount {
			case 1:
				return testBlobsByHeight[100], nil
			case 2:
				return testBlobsByHeight[101], nil
			case 3:
				return testBlobsByHeight[102], nil
			default:
				return nil, errors.New("unexpected call")
			}
		},
	}

	client := NewClient(Config{
		DA:                       mockDAInstance,
		Logger:                   zerolog.Nop(),
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 3, // Epoch: 100-102
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())
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

func TestForcedInclusionRetriever_processForcedInclusionBlobs(t *testing.T) {
	client := NewClient(Config{
		DA:                       &mockDA{},
		Logger:                   zerolog.Nop(),
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := NewForcedInclusionRetriever(client, gen, zerolog.Nop())

	tests := []struct {
		name            string
		result          coreda.ResultRetrieve
		height          uint64
		expectedTxCount int
		expectError     bool
	}{
		{
			name: "success with blobs",
			result: coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code: coreda.StatusSuccess,
				},
				Data: [][]byte{[]byte("tx1"), []byte("tx2")},
			},
			height:          100,
			expectedTxCount: 2,
			expectError:     false,
		},
		{
			name: "not found",
			result: coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code: coreda.StatusNotFound,
				},
			},
			height:          100,
			expectedTxCount: 0,
			expectError:     false,
		},
		{
			name: "error status",
			result: coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code:    coreda.StatusError,
					Message: "test error",
				},
			},
			height:      100,
			expectError: true,
		},
		{
			name: "empty blobs are skipped",
			result: coreda.ResultRetrieve{
				BaseResult: coreda.BaseResult{
					Code: coreda.StatusSuccess,
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
			}
		})
	}
}
