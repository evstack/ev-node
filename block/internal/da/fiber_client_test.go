package da

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// mockFiberClient is a test mock for the FiberClient interface.
type mockFiberClient struct {
	mu        sync.Mutex
	uploads   map[string][]byte // blobID hex -> data
	height    uint64
	uploadErr error
	callCount atomic.Uint64
}

func newMockFiberClient() *mockFiberClient {
	return &mockFiberClient{
		uploads: make(map[string][]byte),
		height:  100,
	}
}

func (m *mockFiberClient) Upload(_ context.Context, namespace, data []byte) (FiberUploadResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.uploadErr != nil {
		return FiberUploadResult{}, m.uploadErr
	}

	m.height++
	callIdx := m.callCount.Add(1)

	// Generate a unique blob ID for each upload
	blobID := make([]byte, 33)
	blobID[0] = 0 // version 0
	hash := sha256.Sum256(append([]byte{byte(callIdx)}, data...))
	copy(blobID[1:], hash[:])

	m.uploads[string(blobID)] = data

	return FiberUploadResult{
		BlobID:  blobID,
		Height:  m.height,
		Promise: []byte("signed-promise-" + string(blobID)),
	}, nil
}

func (m *mockFiberClient) Download(_ context.Context, blobID []byte, _ uint64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, ok := m.uploads[string(blobID)]
	if !ok {
		return nil, errors.New("blob not found")
	}
	return data, nil
}

func (m *mockFiberClient) GetLatestHeight(_ context.Context) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.height, nil
}

func makeTestFiberClient(t *testing.T) (*mockFiberClient, FullClient) {
	t.Helper()
	mock := newMockFiberClient()
	cl := NewFiberClient(FiberConfig{
		Client:         mock,
		Logger:         zerolog.Nop(),
		DefaultTimeout: 5 * time.Second,
		Namespace:      "test-ns",
		DataNamespace:  "test-ns",
	})
	require.NotNil(t, cl)
	return mock, cl
}

func TestFiberClient_NewClient_Nil(t *testing.T) {
	cl := NewFiberClient(FiberConfig{Client: nil})
	assert.Nil(t, cl)
}

func TestFiberClient_Submit_Success(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	res := cl.Submit(context.Background(), [][]byte{[]byte("hello"), []byte("world")}, 0, ns, nil)

	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Len(t, res.IDs, 2)
	require.Equal(t, uint64(2), res.SubmittedCount)
	require.Greater(t, res.Height, uint64(0))
	require.Equal(t, uint64(10), res.BlobSize)
}

func TestFiberClient_Submit_SingleBlob(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	res := cl.Submit(context.Background(), [][]byte{[]byte("single")}, 0, ns, nil)

	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Len(t, res.IDs, 1)
	require.Equal(t, uint64(6), res.BlobSize)
}

func TestFiberClient_Submit_EmptyData(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	res := cl.Submit(context.Background(), [][]byte{}, 0, ns, nil)

	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Empty(t, res.IDs)
	require.Equal(t, uint64(0), res.SubmittedCount)
}

func TestFiberClient_Submit_UploadError(t *testing.T) {
	mock := newMockFiberClient()
	mock.uploadErr = errors.New("upload failed")
	cl := NewFiberClient(FiberConfig{
		Client:         mock,
		Logger:         zerolog.Nop(),
		DefaultTimeout: 5 * time.Second,
		Namespace:      "test-ns",
		DataNamespace:  "test-ns",
	})

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)

	require.Equal(t, datypes.StatusError, res.Code)
	require.Contains(t, res.Message, "fiber upload failed")
}

func TestFiberClient_Submit_CanceledContext(t *testing.T) {
	mock := newMockFiberClient()
	mock.uploadErr = context.Canceled
	cl := NewFiberClient(FiberConfig{
		Client:         mock,
		Logger:         zerolog.Nop(),
		DefaultTimeout: 5 * time.Second,
		Namespace:      "test-ns",
		DataNamespace:  "test-ns",
	})

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)

	require.Equal(t, datypes.StatusContextCanceled, res.Code)
}

func TestFiberClient_Submit_DeadlineExceeded(t *testing.T) {
	mock := newMockFiberClient()
	mock.uploadErr = context.DeadlineExceeded
	cl := NewFiberClient(FiberConfig{
		Client:         mock,
		Logger:         zerolog.Nop(),
		DefaultTimeout: 5 * time.Second,
		Namespace:      "test-ns",
		DataNamespace:  "test-ns",
	})

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)

	require.Equal(t, datypes.StatusContextDeadline, res.Code)
}

func TestFiberClient_Submit_BlobTooLarge(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	largeBlob := make([]byte, 6*1024*1024) // 6MB > 5MB default max
	res := cl.Submit(context.Background(), [][]byte{largeBlob}, 0, ns, nil)

	require.Equal(t, datypes.StatusTooBig, res.Code)
}

func TestFiberClient_Retrieve_Success(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("hello")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	retrieveRes := cl.Retrieve(context.Background(), submitRes.Height, ns)
	require.Equal(t, datypes.StatusSuccess, retrieveRes.Code)
	require.Len(t, retrieveRes.Data, 1)
	require.Equal(t, []byte("hello"), retrieveRes.Data[0])
	require.Equal(t, submitRes.IDs, retrieveRes.IDs)
}

func TestFiberClient_RetrieveBlobs_Success(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("blob1"), []byte("blob2")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	retrieveRes := cl.RetrieveBlobs(context.Background(), submitRes.Height, ns)
	require.Equal(t, datypes.StatusSuccess, retrieveRes.Code)
	require.Len(t, retrieveRes.Data, 2)
	require.Equal(t, []byte("blob1"), retrieveRes.Data[0])
	require.Equal(t, []byte("blob2"), retrieveRes.Data[1])
}

func TestFiberClient_Retrieve_NotFound(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	retrieveRes := cl.Retrieve(context.Background(), 9999, ns)
	require.Equal(t, datypes.StatusNotFound, retrieveRes.Code)
}

func TestFiberClient_Get_Success(t *testing.T) {
	mock, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("getme")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)
	require.Len(t, submitRes.IDs, 1)

	blobs, err := cl.Get(context.Background(), submitRes.IDs, ns)
	require.NoError(t, err)
	require.Len(t, blobs, 1)
	require.Equal(t, []byte("getme"), blobs[0])

	_ = mock // mock stores the data for download
}

func TestFiberClient_Get_EmptyIDs(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	blobs, err := cl.Get(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Nil(t, blobs)
}

func TestFiberClient_Get_InvalidID(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	_, err := cl.Get(context.Background(), []datypes.ID{[]byte{0x01}}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid fiber blob id")
}

func TestFiberClient_Get_DownloadError(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	// Construct a valid-looking ID but with a blob ID that doesn't exist
	fakeBlobID := make([]byte, 33)
	id := makeFiberID(1, fakeBlobID)

	_, err := cl.Get(context.Background(), []datypes.ID{id}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fiber download failed")
}

func TestFiberClient_GetLatestDAHeight(t *testing.T) {
	mock, cl := makeTestFiberClient(t)

	height, err := cl.GetLatestDAHeight(context.Background())
	require.NoError(t, err)
	require.Equal(t, mock.height, height)
}

func TestFiberClient_GetProofs_Success(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("prooftest")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	proofs, err := cl.GetProofs(context.Background(), submitRes.IDs, ns)
	require.NoError(t, err)
	require.Len(t, proofs, 1)
	require.NotEmpty(t, proofs[0]) // Should contain the promise
}

func TestFiberClient_GetProofs_Empty(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	proofs, err := cl.GetProofs(context.Background(), nil, nil)
	require.NoError(t, err)
	require.Empty(t, proofs)
}

func TestFiberClient_Validate_Success(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("validateme")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	proofs, err := cl.GetProofs(context.Background(), submitRes.IDs, ns)
	require.NoError(t, err)

	results, err := cl.Validate(context.Background(), submitRes.IDs, proofs, ns)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.True(t, results[0])
}

func TestFiberClient_Validate_MismatchedLengths(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	_, err := cl.Validate(context.Background(), make([]datypes.ID, 3), make([]datypes.Proof, 2), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must match")
}

func TestFiberClient_Validate_Empty(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	results, err := cl.Validate(context.Background(), nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestFiberClient_Validate_WrongProof(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("validatewrong")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	// Use a wrong proof
	fakeProofs := []datypes.Proof{[]byte("wrong-proof")}
	results, err := cl.Validate(context.Background(), submitRes.IDs, fakeProofs, ns)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.False(t, results[0]) // Wrong proof should fail validation
}

func TestFiberClient_Validate_EmptyProof(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	emptyProofs := []datypes.Proof{[]byte{}}
	results, err := cl.Validate(context.Background(), submitRes.IDs, emptyProofs, ns)
	require.NoError(t, err)
	require.False(t, results[0])
}

func TestFiberClient_Namespaces(t *testing.T) {
	cl := NewFiberClient(FiberConfig{
		Client:                    newMockFiberClient(),
		Logger:                    zerolog.Nop(),
		Namespace:                 "header-ns",
		DataNamespace:             "data-ns",
		ForcedInclusionNamespace:  "forced-ns",
	})
	require.NotNil(t, cl)

	require.Equal(t, datypes.NamespaceFromString("header-ns").Bytes(), cl.GetHeaderNamespace())
	require.Equal(t, datypes.NamespaceFromString("data-ns").Bytes(), cl.GetDataNamespace())
	require.Equal(t, datypes.NamespaceFromString("forced-ns").Bytes(), cl.GetForcedInclusionNamespace())
	require.True(t, cl.HasForcedInclusionNamespace())
}

func TestFiberClient_NoForcedNamespace(t *testing.T) {
	cl := NewFiberClient(FiberConfig{
		Client:        newMockFiberClient(),
		Logger:        zerolog.Nop(),
		Namespace:     "header-ns",
		DataNamespace: "data-ns",
	})
	require.NotNil(t, cl)

	require.Nil(t, cl.GetForcedInclusionNamespace())
	require.False(t, cl.HasForcedInclusionNamespace())
}

func TestFiberClient_Subscribe(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := cl.Subscribe(ctx, nil, false)
	require.NoError(t, err)
	require.NotNil(t, ch)

	// Submit a blob so the index has something
	ns := datypes.NamespaceFromString("test-ns").Bytes()
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("sub-data")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	// The subscribe goroutine polls and should emit the event for the submitted height.
	// Since the mock height starts at 100 and upload increments to 101,
	// the subscribe loop should eventually pick it up.
	select {
	case ev := <-ch:
		require.Equal(t, submitRes.Height, ev.Height)
		require.Len(t, ev.Blobs, 1)
		require.Equal(t, []byte("sub-data"), ev.Blobs[0])
	case <-time.After(5 * time.Second):
		t.Fatal("subscribe did not emit event within timeout")
	}
}

func TestFiberClient_Submit_MultipleBlobs(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	data := [][]byte{[]byte("first"), []byte("second"), []byte("third")}
	res := cl.Submit(context.Background(), data, 0, ns, nil)

	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Len(t, res.IDs, 3)
	require.Equal(t, uint64(3), res.SubmittedCount)

	// Verify all blobs can be retrieved
	retrieveRes := cl.Retrieve(context.Background(), res.Height, ns)
	require.Equal(t, datypes.StatusSuccess, retrieveRes.Code)
	require.Len(t, retrieveRes.Data, 3)
	require.Equal(t, []byte("first"), retrieveRes.Data[0])
	require.Equal(t, []byte("second"), retrieveRes.Data[1])
	require.Equal(t, []byte("third"), retrieveRes.Data[2])
}

func TestFiberClient_SubmitAndDownload(t *testing.T) {
	mock, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()
	data := []byte("download-test")
	submitRes := cl.Submit(context.Background(), [][]byte{data}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)

	// The mock stores the data, so Get should be able to download it
	blobs, err := cl.Get(context.Background(), submitRes.IDs, ns)
	require.NoError(t, err)
	require.Len(t, blobs, 1)
	require.Equal(t, data, blobs[0])

	_ = mock
}

func TestMakeFiberID_RoundTrip(t *testing.T) {
	blobID := make([]byte, 33)
	blobID[0] = 1
	for i := 1; i < 33; i++ {
		blobID[i] = byte(i)
	}

	id := makeFiberID(42, blobID)
	height, extractedBlobID := splitFiberID(id)

	require.Equal(t, uint64(42), height)
	require.Equal(t, blobID, extractedBlobID)
}

func TestSplitFiberID_Invalid(t *testing.T) {
	height, blobID := splitFiberID([]byte{0x01, 0x02})
	require.Equal(t, uint64(0), height)
	require.Nil(t, blobID)
}

func TestFiberClient_DefaultTimeout(t *testing.T) {
	cl := NewFiberClient(FiberConfig{
		Client:        newMockFiberClient(),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	require.NotNil(t, cl)
	fc := cl.(*fiberDAClient)
	require.Equal(t, 60*time.Second, fc.defaultTimeout)
}

func TestFiberClient_FullSubmitRetrieveCycle(t *testing.T) {
	_, cl := makeTestFiberClient(t)

	ns := datypes.NamespaceFromString("test-ns").Bytes()

	// Submit
	submitRes := cl.Submit(context.Background(), [][]byte{[]byte("cycle-data")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, submitRes.Code)
	require.Len(t, submitRes.IDs, 1)
	submittedHeight := submitRes.Height

	// Retrieve
	retrieveRes := cl.Retrieve(context.Background(), submittedHeight, ns)
	require.Equal(t, datypes.StatusSuccess, retrieveRes.Code)
	require.Equal(t, []byte("cycle-data"), retrieveRes.Data[0])

	// Get
	blobs, err := cl.Get(context.Background(), submitRes.IDs, ns)
	require.NoError(t, err)
	require.Equal(t, []byte("cycle-data"), blobs[0])

	// GetProofs + Validate
	proofs, err := cl.GetProofs(context.Background(), submitRes.IDs, ns)
	require.NoError(t, err)
	require.NotEmpty(t, proofs[0])

	valid, err := cl.Validate(context.Background(), submitRes.IDs, proofs, ns)
	require.NoError(t, err)
	require.True(t, valid[0])
}
