package da_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"

	da "github.com/evstack/ev-node/block/internal/da"
	"github.com/evstack/ev-node/block/internal/da/testclient"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
)

// mockDA is a lightweight datypes.DA implementation for forced inclusion tests.
type mockDA struct {
	submitFunc        func(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte) ([]datypes.ID, error)
	submitWithOptions func(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte, options []byte) ([]datypes.ID, error)
	getIDsFunc        func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error)
	getFunc           func(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error)
}

func (m *mockDA) Submit(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte) ([]datypes.ID, error) {
	if m.submitFunc != nil {
		return m.submitFunc(ctx, blobs, gasPrice, namespace)
	}
	return nil, nil
}

func (m *mockDA) SubmitWithOptions(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte, options []byte) ([]datypes.ID, error) {
	if m.submitWithOptions != nil {
		return m.submitWithOptions(ctx, blobs, gasPrice, namespace, options)
	}
	return nil, nil
}

func (m *mockDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
	if m.getIDsFunc != nil {
		return m.getIDsFunc(ctx, height, namespace)
	}
	return nil, errors.New("getIDs not implemented")
}

func (m *mockDA) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, ids, namespace)
	}
	return nil, errors.New("get not implemented")
}

func (m *mockDA) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	return nil, errors.New("getProofs not implemented")
}

func (m *mockDA) Commit(ctx context.Context, blobs []datypes.Blob, namespace []byte) ([]datypes.Commitment, error) {
	return nil, errors.New("commit not implemented")
}

func (m *mockDA) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	return nil, errors.New("validate not implemented")
}

func TestNewForcedInclusionRetriever(t *testing.T) {
	client := testclient.New(testclient.Config{
		DA:                       &mockDA{},
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

	gen := genesis.Genesis{
		DAStartHeight:          100,
		DAEpochForcedInclusion: 10,
	}

	retriever := da.NewForcedInclusionRetriever(client, gen, zerolog.Nop())
	assert.Assert(t, retriever != nil)
}

func TestForcedInclusionRetriever_RetrieveForcedIncludedTxs_NoNamespace(t *testing.T) {
	client := testclient.New(testclient.Config{
		DA:            &mockDA{},
		Namespace:     "test-ns",
		DataNamespace: "test-data-ns",
		// No forced inclusion namespace
	})

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
	client := testclient.New(testclient.Config{
		DA:                       &mockDA{},
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

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

    mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
			return &datypes.GetIDsResult{
				IDs:       []datypes.ID{[]byte("id1"), []byte("id2"), []byte("id3")},
				Timestamp: time.Now(),
			}, nil
		},
		getFunc: func(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
			return testBlobs, nil
		},
	}

	client := testclient.New(testclient.Config{
		DA:                       mockDAInstance,
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

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
    mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
			return nil, datypes.ErrHeightFromFuture
		},
	}

	client := testclient.New(testclient.Config{
		DA:                       mockDAInstance,
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

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
    mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
			return nil, datypes.ErrBlobNotFound
		},
	}

	client := testclient.New(testclient.Config{
		DA:                       mockDAInstance,
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

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
	callCount := 0
	testBlobsByHeight := map[uint64][][]byte{
		100: {[]byte("tx1"), []byte("tx2")},
		101: {[]byte("tx3")},
		102: {[]byte("tx4"), []byte("tx5"), []byte("tx6")},
	}

    mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
			callCount++
			blobs, exists := testBlobsByHeight[height]
			if !exists {
				return nil, datypes.ErrBlobNotFound
			}
			ids := make([]datypes.ID, len(blobs))
			for i := range blobs {
				ids[i] = []byte("id")
			}
			return &datypes.GetIDsResult{
				IDs:       ids,
				Timestamp: time.Now(),
			}, nil
		},
		getFunc: func(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
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

	client := testclient.New(testclient.Config{
		DA:                       mockDAInstance,
		Namespace:                "test-ns",
		DataNamespace:            "test-data-ns",
		ForcedInclusionNamespace: "test-fi-ns",
	})

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
