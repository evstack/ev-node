package da

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/celestiaorg/go-square/v3/share"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBlobAPI struct {
	submitErr   error
	height      uint64
	blobs       []*blobrpc.Blob
	proof       *blobrpc.Proof
	included    bool
	commitProof *blobrpc.CommitmentProof
}

func (m *mockBlobAPI) Submit(ctx context.Context, blobs []*blobrpc.Blob, opts *blobrpc.SubmitOptions) (uint64, error) {
	return m.height, m.submitErr
}

func (m *mockBlobAPI) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blobrpc.Blob, error) {
	return m.blobs, m.submitErr
}

func (m *mockBlobAPI) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blobrpc.Commitment) (*blobrpc.Proof, error) {
	return m.proof, m.submitErr
}

func (m *mockBlobAPI) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blobrpc.Proof, commitment blobrpc.Commitment) (bool, error) {
	return m.included, m.submitErr
}

func (m *mockBlobAPI) GetCommitmentProof(ctx context.Context, height uint64, namespace share.Namespace, shareCommitment []byte) (*blobrpc.CommitmentProof, error) {
	return m.commitProof, m.submitErr
}

func (m *mockBlobAPI) Subscribe(ctx context.Context, namespace share.Namespace) (<-chan *blobrpc.SubscriptionResponse, error) {
	ch := make(chan *blobrpc.SubscriptionResponse)
	close(ch)
	return ch, nil
}

func makeBlobRPCClient(m *mockBlobAPI) *blobrpc.Client {
	var api blobrpc.BlobAPI
	api.Internal.Submit = m.Submit
	api.Internal.GetAll = m.GetAll
	api.Internal.GetProof = m.GetProof
	api.Internal.Included = m.Included
	api.Internal.GetCommitmentProof = m.GetCommitmentProof
	api.Internal.Subscribe = m.Subscribe
	return &blobrpc.Client{Blob: api}
}

func TestCelestiaClient_Submit_ErrorMapping(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	testCases := []struct {
		name       string
		err        error
		wantStatus datypes.StatusCode
	}{
		{"timeout", datypes.ErrTxTimedOut, datypes.StatusNotIncludedInBlock},
		{"alreadyInMempool", datypes.ErrTxAlreadyInMempool, datypes.StatusAlreadyInMempool},
		{"seq", datypes.ErrTxIncorrectAccountSequence, datypes.StatusIncorrectAccountSequence},
		{"tooBig", datypes.ErrBlobSizeOverLimit, datypes.StatusTooBig},
		{"deadline", datypes.ErrContextDeadline, datypes.StatusContextDeadline},
		{"canceled", context.Canceled, datypes.StatusContextCanceled},
		{"other", errors.New("boom"), datypes.StatusError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cl := NewClient(Config{
				Client:        makeBlobRPCClient(&mockBlobAPI{submitErr: tc.err}),
				Logger:        zerolog.Nop(),
				Namespace:     "ns",
				DataNamespace: "ns",
			})
			res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)
			assert.Equal(t, tc.wantStatus, res.Code)
		})
	}
}

func TestCelestiaClient_Submit_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockBlobAPI{height: 10}
	cl := NewClient(Config{
		Client:        makeBlobRPCClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Equal(t, uint64(10), res.Height)
	require.Len(t, res.IDs, 1)
}

func TestCelestiaClient_Submit_InvalidNamespace(t *testing.T) {
	mockAPI := &mockBlobAPI{height: 10}
	cl := NewClient(Config{
		Client:        makeBlobRPCClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, []byte{0x01, 0x02}, nil)
	require.Equal(t, datypes.StatusError, res.Code)
}

func TestCelestiaClient_Retrieve_NotFound(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockBlobAPI{submitErr: datypes.ErrBlobNotFound}
	cl := NewClient(Config{
		Client:        makeBlobRPCClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 5, ns)
	require.Equal(t, datypes.StatusNotFound, res.Code)
}

func TestCelestiaClient_Retrieve_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	b, err := blobrpc.NewBlobV0(share.MustNewV0Namespace([]byte("ns")), []byte("payload"))
	require.NoError(t, err)
	mockAPI := &mockBlobAPI{height: 7, blobs: []*blobrpc.Blob{b}}
	cl := NewClient(Config{
		Client:        makeBlobRPCClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 7, ns)
	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Len(t, res.Data, 1)
	require.Len(t, res.IDs, 1)
}

func TestCelestiaClient_SubmitOptionsMerge(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockBlobAPI{height: 1}
	cl := NewClient(Config{
		Client:        makeBlobRPCClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})

	opts := map[string]any{"signer_address": "celestia1xyz"}
	raw, err := json.Marshal(opts)
	require.NoError(t, err)

	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, raw)
	require.Equal(t, datypes.StatusSuccess, res.Code)
}

func TestClient_RetrieveHeaders(t *testing.T) {
	logger := zerolog.Nop()
	dataLayerHeight := uint64(100)
	mockIDs := [][]byte{[]byte("id1")}
	mockBlobs := [][]byte{[]byte("header-blob")}
	mockTimestamp := time.Now()

	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return &coreda.GetIDsResult{
				IDs:       mockIDs,
				Timestamp: mockTimestamp,
			}, nil
		},
		getFunc: func(ctx context.Context, ids []coreda.ID, namespace []byte) ([]coreda.Blob, error) {
			return mockBlobs, nil
		},
	}

	client := NewClient(Config{
		DA:            mockDAInstance,
		Logger:        logger,
		Namespace:     "test-header-ns",
		DataNamespace: "test-data-ns",
	})

	result := client.RetrieveHeaders(context.Background(), dataLayerHeight)

	assert.Equal(t, coreda.StatusSuccess, result.Code)
	assert.Equal(t, dataLayerHeight, result.Height)
	assert.Equal(t, len(mockBlobs), len(result.Data))
}

func TestClient_RetrieveData(t *testing.T) {
	logger := zerolog.Nop()
	dataLayerHeight := uint64(200)
	mockIDs := [][]byte{[]byte("id1"), []byte("id2")}
	mockBlobs := [][]byte{[]byte("data-blob-1"), []byte("data-blob-2")}
	mockTimestamp := time.Now()

	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return &coreda.GetIDsResult{
				IDs:       mockIDs,
				Timestamp: mockTimestamp,
			}, nil
		},
		getFunc: func(ctx context.Context, ids []coreda.ID, namespace []byte) ([]coreda.Blob, error) {
			return mockBlobs, nil
		},
	}

	client := NewClient(Config{
		DA:            mockDAInstance,
		Logger:        logger,
		Namespace:     "test-header-ns",
		DataNamespace: "test-data-ns",
	})

	result := client.RetrieveData(context.Background(), dataLayerHeight)

	assert.Equal(t, coreda.StatusSuccess, result.Code)
	assert.Equal(t, dataLayerHeight, result.Height)
	assert.Equal(t, len(mockBlobs), len(result.Data))
}

func TestClient_RetrieveBatched(t *testing.T) {
	logger := zerolog.Nop()
	dataLayerHeight := uint64(100)

	// Create 200 IDs to exceed default batch size
	numIDs := 200
	mockIDs := make([][]byte, numIDs)
	for i := range numIDs {
		mockIDs[i] = []byte{byte(i)}
	}

	// Track which batches were requested
	batchCalls := []int{}

	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return &coreda.GetIDsResult{
				IDs:       mockIDs,
				Timestamp: time.Now(),
			}, nil
		},
		getFunc: func(ctx context.Context, ids []coreda.ID, namespace []byte) ([]coreda.Blob, error) {
			batchCalls = append(batchCalls, len(ids))
			// Return a blob for each ID in the batch
			blobs := make([][]byte, len(ids))
			for i := range ids {
				blobs[i] = []byte("blob")
			}
			return blobs, nil
		},
	}

	client := NewClient(Config{
		DA:                mockDAInstance,
		Logger:            logger,
		Namespace:         "test-ns",
		DataNamespace:     "test-data-ns",
		RetrieveBatchSize: 50, // Set smaller batch size for testing
	})

	encodedNamespace := coreda.NamespaceFromString("test-ns")
	result := client.Retrieve(context.Background(), dataLayerHeight, encodedNamespace.Bytes())

	assert.Equal(t, coreda.StatusSuccess, result.Code)
	assert.Equal(t, numIDs, len(result.Data))

	// Should have made 4 batches: 50 + 50 + 50 + 50 = 200
	assert.Equal(t, 4, len(batchCalls))
	assert.Equal(t, 50, batchCalls[0])
	assert.Equal(t, 50, batchCalls[1])
	assert.Equal(t, 50, batchCalls[2])
	assert.Equal(t, 50, batchCalls[3])
}

func TestClient_RetrieveBatched_PartialBatch(t *testing.T) {
	logger := zerolog.Nop()
	dataLayerHeight := uint64(100)

	// Create 175 IDs to test partial batch at the end
	numIDs := 175
	mockIDs := make([][]byte, numIDs)
	for i := range numIDs {
		mockIDs[i] = []byte{byte(i)}
	}

	batchCalls := []int{}

	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return &coreda.GetIDsResult{
				IDs:       mockIDs,
				Timestamp: time.Now(),
			}, nil
		},
		getFunc: func(ctx context.Context, ids []coreda.ID, namespace []byte) ([]coreda.Blob, error) {
			batchCalls = append(batchCalls, len(ids))
			blobs := make([][]byte, len(ids))
			for i := range ids {
				blobs[i] = []byte("blob")
			}
			return blobs, nil
		},
	}

	client := NewClient(Config{
		DA:                mockDAInstance,
		Logger:            logger,
		Namespace:         "test-ns",
		DataNamespace:     "test-data-ns",
		RetrieveBatchSize: 50,
	})

	encodedNamespace := coreda.NamespaceFromString("test-ns")
	result := client.Retrieve(context.Background(), dataLayerHeight, encodedNamespace.Bytes())

	assert.Equal(t, coreda.StatusSuccess, result.Code)
	assert.Equal(t, numIDs, len(result.Data))

	// Should have made 4 batches: 50 + 50 + 50 + 25 = 175
	assert.Equal(t, 4, len(batchCalls))
	assert.Equal(t, 50, batchCalls[0])
	assert.Equal(t, 50, batchCalls[1])
	assert.Equal(t, 50, batchCalls[2])
	assert.Equal(t, 25, batchCalls[3]) // Partial batch
}

func TestClient_RetrieveBatched_ErrorInSecondBatch(t *testing.T) {
	logger := zerolog.Nop()
	dataLayerHeight := uint64(100)

	// Create 200 IDs to require multiple batches
	numIDs := 200
	mockIDs := make([][]byte, numIDs)
	for i := range numIDs {
		mockIDs[i] = []byte{byte(i)}
	}

	batchCallCount := 0

	mockDAInstance := &mockDA{
		getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*coreda.GetIDsResult, error) {
			return &coreda.GetIDsResult{
				IDs:       mockIDs,
				Timestamp: time.Now(),
			}, nil
		},
		getFunc: func(ctx context.Context, ids []coreda.ID, namespace []byte) ([]coreda.Blob, error) {
			batchCallCount++
			// Fail on second batch
			if batchCallCount == 2 {
				return nil, errors.New("network error in batch 2")
			}
			blobs := make([][]byte, len(ids))
			for i := range ids {
				blobs[i] = []byte("blob")
			}
			return blobs, nil
		},
	}

	client := NewClient(Config{
		DA:                mockDAInstance,
		Logger:            logger,
		Namespace:         "test-ns",
		DataNamespace:     "test-data-ns",
		RetrieveBatchSize: 50,
	})

	encodedNamespace := coreda.NamespaceFromString("test-ns")
	result := client.Retrieve(context.Background(), dataLayerHeight, encodedNamespace.Bytes())

	assert.Equal(t, coreda.StatusError, result.Code)
	assert.Assert(t, result.Message != "")
	// Error message should mention the batch range
	assert.Assert(t, len(result.Message) > 0)
}
