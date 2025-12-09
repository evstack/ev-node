package da

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// mockDA is a simple mock implementation of datypes.DA for testing
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
	return nil, errors.New("not implemented")
}

func (m *mockDA) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, ids, namespace)
	}
	return nil, errors.New("not implemented")
}

func (m *mockDA) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDA) Commit(ctx context.Context, blobs []datypes.Blob, namespace []byte) ([]datypes.Commitment, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDA) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	return nil, errors.New("not implemented")
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "with all namespaces",
			cfg: Config{
				DA:                       &mockDA{},
				Logger:                   zerolog.Nop(),
				DefaultTimeout:           5 * time.Second,
				Namespace:                "test-ns",
				DataNamespace:            "test-data-ns",
				ForcedInclusionNamespace: "test-fi-ns",
			},
		},
		{
			name: "without forced inclusion namespace",
			cfg: Config{
				DA:             &mockDA{},
				Logger:         zerolog.Nop(),
				DefaultTimeout: 5 * time.Second,
				Namespace:      "test-ns",
				DataNamespace:  "test-data-ns",
			},
		},
		{
			name: "with default timeout",
			cfg: Config{
				DA:            &mockDA{},
				Logger:        zerolog.Nop(),
				Namespace:     "test-ns",
				DataNamespace: "test-data-ns",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)
			assert.Assert(t, client != nil)
			assert.Assert(t, client.da != nil)
			assert.Assert(t, len(client.namespaceBz) > 0)
			assert.Assert(t, len(client.namespaceDataBz) > 0)

			if tt.cfg.ForcedInclusionNamespace != "" {
				assert.Assert(t, client.hasForcedInclusionNs)
				assert.Assert(t, len(client.namespaceForcedInclusionBz) > 0)
			} else {
				assert.Assert(t, !client.hasForcedInclusionNs)
			}

			expectedTimeout := tt.cfg.DefaultTimeout
			if expectedTimeout == 0 {
				expectedTimeout = 60 * time.Second
			}
			assert.Equal(t, client.defaultTimeout, expectedTimeout)
		})
	}
}

func TestClient_HasForcedInclusionNamespace(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{
			name: "with forced inclusion namespace",
			cfg: Config{
				DA:                       &mockDA{},
				Logger:                   zerolog.Nop(),
				Namespace:                "test-ns",
				DataNamespace:            "test-data-ns",
				ForcedInclusionNamespace: "test-fi-ns",
			},
			expected: true,
		},
		{
			name: "without forced inclusion namespace",
			cfg: Config{
				DA:            &mockDA{},
				Logger:        zerolog.Nop(),
				Namespace:     "test-ns",
				DataNamespace: "test-data-ns",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)
			assert.Equal(t, client.HasForcedInclusionNamespace(), tt.expected)
		})
	}
}

func TestClient_GetNamespaces(t *testing.T) {
	cfg := Config{
		DA:                       &mockDA{},
		Logger:                   zerolog.Nop(),
		Namespace:                "test-header",
		DataNamespace:            "test-data",
		ForcedInclusionNamespace: "test-fi",
	}

	client := NewClient(cfg)

	headerNs := client.GetHeaderNamespace()
	assert.Assert(t, len(headerNs) > 0)

	dataNs := client.GetDataNamespace()
	assert.Assert(t, len(dataNs) > 0)

	fiNs := client.GetForcedInclusionNamespace()
	assert.Assert(t, len(fiNs) > 0)

	// Namespaces should be different
	assert.Assert(t, string(headerNs) != string(dataNs))
	assert.Assert(t, string(headerNs) != string(fiNs))
	assert.Assert(t, string(dataNs) != string(fiNs))
}

func TestClient_RetrieveForcedInclusion_NotConfigured(t *testing.T) {
	cfg := Config{
		DA:            &mockDA{},
		Logger:        zerolog.Nop(),
		Namespace:     "test-ns",
		DataNamespace: "test-data-ns",
	}

	client := NewClient(cfg)
	ctx := context.Background()

	result := client.RetrieveForcedInclusion(ctx, 100)
	assert.Equal(t, result.Code, datypes.StatusError)
	assert.Assert(t, result.Message != "")
}

func TestClient_Submit(t *testing.T) {
	logger := zerolog.Nop()

	testCases := []struct {
		name           string
		data           [][]byte
		gasPrice       float64
		options        []byte
		submitErr      error
		submitIDs      [][]byte
		expectedCode   datypes.StatusCode
		expectedErrMsg string
		expectedIDs    [][]byte
		expectedCount  uint64
	}{
		{
			name:          "successful submission",
			data:          [][]byte{[]byte("blob1"), []byte("blob2")},
			gasPrice:      1.0,
			options:       []byte("opts"),
			submitIDs:     [][]byte{[]byte("id1"), []byte("id2")},
			expectedCode:  datypes.StatusSuccess,
			expectedIDs:   [][]byte{[]byte("id1"), []byte("id2")},
			expectedCount: 2,
		},
		{
			name:           "context canceled error",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitErr:      context.Canceled,
			expectedCode:   datypes.StatusContextCanceled,
			expectedErrMsg: "submission canceled",
		},
		{
			name:           "tx timed out error",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitErr:      datypes.ErrTxTimedOut,
			expectedCode:   datypes.StatusNotIncludedInBlock,
			expectedErrMsg: "failed to submit blobs: " + datypes.ErrTxTimedOut.Error(),
		},
		{
			name:           "tx already in mempool error",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitErr:      datypes.ErrTxAlreadyInMempool,
			expectedCode:   datypes.StatusAlreadyInMempool,
			expectedErrMsg: "failed to submit blobs: " + datypes.ErrTxAlreadyInMempool.Error(),
		},
		{
			name:           "incorrect account sequence error",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitErr:      datypes.ErrTxIncorrectAccountSequence,
			expectedCode:   datypes.StatusIncorrectAccountSequence,
			expectedErrMsg: "failed to submit blobs: " + datypes.ErrTxIncorrectAccountSequence.Error(),
		},
		{
			name:           "blob size over limit error",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitErr:      datypes.ErrBlobSizeOverLimit,
			expectedCode:   datypes.StatusTooBig,
			expectedErrMsg: "failed to submit blobs: " + datypes.ErrBlobSizeOverLimit.Error(),
		},
		{
			name:           "context deadline error",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitErr:      datypes.ErrContextDeadline,
			expectedCode:   datypes.StatusContextDeadline,
			expectedErrMsg: "failed to submit blobs: " + datypes.ErrContextDeadline.Error(),
		},
		{
			name:           "generic submission error",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitErr:      errors.New("some generic error"),
			expectedCode:   datypes.StatusError,
			expectedErrMsg: "failed to submit blobs: some generic error",
		},
		{
			name:           "no IDs returned for non-empty data",
			data:           [][]byte{[]byte("blob1")},
			gasPrice:       1.0,
			options:        []byte("opts"),
			submitIDs:      [][]byte{},
			expectedCode:   datypes.StatusError,
			expectedErrMsg: "failed to submit blobs: no IDs returned despite non-empty input",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDAInstance := &mockDA{
				submitWithOptions: func(ctx context.Context, blobs []datypes.Blob, gasPrice float64, namespace []byte, options []byte) ([]datypes.ID, error) {
					return tc.submitIDs, tc.submitErr
				},
			}

			client := NewClient(Config{
				DA:            mockDAInstance,
				Logger:        logger,
				Namespace:     "test-namespace",
				DataNamespace: "test-data-namespace",
			})

			encodedNamespace := datypes.NamespaceFromString("test-namespace")
			result := client.Submit(context.Background(), tc.data, tc.gasPrice, encodedNamespace.Bytes(), tc.options)

			assert.Equal(t, tc.expectedCode, result.Code)
			if tc.expectedErrMsg != "" {
				assert.Assert(t, result.Message != "")
			}
			if tc.expectedIDs != nil {
				assert.Equal(t, len(tc.expectedIDs), len(result.IDs))
			}
			if tc.expectedCount != 0 {
				assert.Equal(t, tc.expectedCount, result.SubmittedCount)
			}
		})
	}
}

func TestClient_Retrieve(t *testing.T) {
	logger := zerolog.Nop()
	dataLayerHeight := uint64(100)
	mockIDs := [][]byte{[]byte("id1"), []byte("id2")}
	mockBlobs := [][]byte{[]byte("blobA"), []byte("blobB")}
	mockTimestamp := time.Now()

	testCases := []struct {
		name           string
		getIDsResult   *datypes.GetIDsResult
		getIDsErr      error
		getBlobsErr    error
		expectedCode   datypes.StatusCode
		expectedErrMsg string
		expectedIDs    [][]byte
		expectedData   [][]byte
		expectedHeight uint64
	}{
		{
			name: "successful retrieval",
			getIDsResult: &datypes.GetIDsResult{
				IDs:       mockIDs,
				Timestamp: mockTimestamp,
			},
			expectedCode:   datypes.StatusSuccess,
			expectedIDs:    mockIDs,
			expectedData:   mockBlobs,
			expectedHeight: dataLayerHeight,
		},
		{
			name:           "blob not found error during GetIDs",
			getIDsErr:      datypes.ErrBlobNotFound,
			expectedCode:   datypes.StatusNotFound,
			expectedErrMsg: datypes.ErrBlobNotFound.Error(),
			expectedHeight: dataLayerHeight,
		},
		{
			name:           "height from future error during GetIDs",
			getIDsErr:      datypes.ErrHeightFromFuture,
			expectedCode:   datypes.StatusHeightFromFuture,
			expectedErrMsg: datypes.ErrHeightFromFuture.Error(),
			expectedHeight: dataLayerHeight,
		},
		{
			name:           "generic error during GetIDs",
			getIDsErr:      errors.New("failed to connect to DA"),
			expectedCode:   datypes.StatusError,
			expectedErrMsg: "failed to get IDs: failed to connect to DA",
			expectedHeight: dataLayerHeight,
		},
		{
			name:           "GetIDs returns nil result",
			getIDsResult:   nil,
			expectedCode:   datypes.StatusNotFound,
			expectedErrMsg: datypes.ErrBlobNotFound.Error(),
			expectedHeight: dataLayerHeight,
		},
		{
			name: "GetIDs returns empty IDs",
			getIDsResult: &datypes.GetIDsResult{
				IDs:       [][]byte{},
				Timestamp: mockTimestamp,
			},
			expectedCode:   datypes.StatusNotFound,
			expectedErrMsg: datypes.ErrBlobNotFound.Error(),
			expectedHeight: dataLayerHeight,
		},
		{
			name: "error during Get (blobs retrieval)",
			getIDsResult: &datypes.GetIDsResult{
				IDs:       mockIDs,
				Timestamp: mockTimestamp,
			},
			getBlobsErr:    errors.New("network error during blob retrieval"),
			expectedCode:   datypes.StatusError,
			expectedErrMsg: "failed to get blobs for batch 0-1: network error during blob retrieval",
			expectedHeight: dataLayerHeight,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDAInstance := &mockDA{
				getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
					return tc.getIDsResult, tc.getIDsErr
				},
				getFunc: func(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
					if tc.getBlobsErr != nil {
						return nil, tc.getBlobsErr
					}
					return mockBlobs, nil
				},
			}

			client := NewClient(Config{
				DA:             mockDAInstance,
				Logger:         logger,
				Namespace:      "test-namespace",
				DataNamespace:  "test-data-namespace",
				DefaultTimeout: 5 * time.Second,
			})

			encodedNamespace := datypes.NamespaceFromString("test-namespace")
			result := client.Retrieve(context.Background(), dataLayerHeight, encodedNamespace.Bytes())

			assert.Equal(t, tc.expectedCode, result.Code)
			assert.Equal(t, tc.expectedHeight, result.Height)
			if tc.expectedErrMsg != "" {
				assert.Assert(t, result.Message != "")
			}
			if tc.expectedIDs != nil {
				assert.Equal(t, len(tc.expectedIDs), len(result.IDs))
			}
			if tc.expectedData != nil {
				assert.Equal(t, len(tc.expectedData), len(result.Data))
			}
		})
	}
}

func TestClient_Retrieve_Timeout(t *testing.T) {
	logger := zerolog.Nop()
	dataLayerHeight := uint64(100)
	encodedNamespace := datypes.NamespaceFromString("test-namespace")

	t.Run("timeout during GetIDs", func(t *testing.T) {
		mockDAInstance := &mockDA{
			getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
				<-ctx.Done() // Wait for context cancellation
				return nil, context.DeadlineExceeded
			},
		}

		client := NewClient(Config{
			DA:             mockDAInstance,
			Logger:         logger,
			Namespace:      "test-namespace",
			DataNamespace:  "test-data-namespace",
			DefaultTimeout: 1 * time.Millisecond,
		})

		result := client.Retrieve(context.Background(), dataLayerHeight, encodedNamespace.Bytes())

		assert.Equal(t, datypes.StatusError, result.Code)
		assert.Assert(t, result.Message != "")
	})

	t.Run("timeout during Get", func(t *testing.T) {
		mockIDs := [][]byte{[]byte("id1")}
		mockTimestamp := time.Now()

		mockDAInstance := &mockDA{
			getIDsFunc: func(ctx context.Context, height uint64, namespace []byte) (*datypes.GetIDsResult, error) {
				return &datypes.GetIDsResult{
					IDs:       mockIDs,
					Timestamp: mockTimestamp,
				}, nil
			},
			getFunc: func(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
				<-ctx.Done() // Wait for context cancellation
				return nil, context.DeadlineExceeded
			},
		}

		client := NewClient(Config{
			DA:             mockDAInstance,
			Logger:         logger,
			Namespace:      "test-namespace",
			DataNamespace:  "test-data-namespace",
			DefaultTimeout: 1 * time.Millisecond,
		})

		result := client.Retrieve(context.Background(), dataLayerHeight, encodedNamespace.Bytes())

		assert.Equal(t, datypes.StatusError, result.Code)
		assert.Assert(t, result.Message != "")
	})
}
