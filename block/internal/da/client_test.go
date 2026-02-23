package da

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	"github.com/evstack/ev-node/pkg/da/jsonrpc/mocks"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

func makeBlobRPCClient(blobModule *mocks.MockBlobModule, headerModule *mocks.MockHeaderModule) *blobrpc.Client {
	var blobAPI blobrpc.BlobAPI
	blobAPI.Internal.Submit = blobModule.Submit
	blobAPI.Internal.Get = blobModule.Get
	blobAPI.Internal.GetAll = blobModule.GetAll
	blobAPI.Internal.GetProof = blobModule.GetProof
	blobAPI.Internal.Included = blobModule.Included
	blobAPI.Internal.GetCommitmentProof = blobModule.GetCommitmentProof
	blobAPI.Internal.Subscribe = blobModule.Subscribe

	var headerAPI blobrpc.HeaderAPI
	if headerModule != nil {
		headerAPI.Internal.GetByHeight = headerModule.GetByHeight
		headerAPI.Internal.LocalHead = headerModule.LocalHead
		headerAPI.Internal.NetworkHead = headerModule.NetworkHead
	}

	return &blobrpc.Client{Blob: blobAPI, Header: headerAPI}
}

// makeDefaultHeaderModule creates a header module mock that returns a fixed timestamp.
func makeDefaultHeaderModule(t *testing.T) *mocks.MockHeaderModule {
	headerModule := mocks.NewMockHeaderModule(t)
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	headerModule.On("GetByHeight", mock.Anything, mock.Anything).Return(&blobrpc.Header{
		Header: blobrpc.RawHeader{
			Time: fixedTime,
		},
	}, nil).Maybe()
	return headerModule
}

func TestClient_Submit_ErrorMapping(t *testing.T) {
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
			blobModule := mocks.NewMockBlobModule(t)
			blobModule.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(uint64(0), tc.err)

			cl := NewClient(Config{
				DA:            makeBlobRPCClient(blobModule, nil),
				Logger:        zerolog.Nop(),
				Namespace:     "ns",
				DataNamespace: "ns",
			})
			res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)
			assert.Equal(t, tc.wantStatus, res.Code)
		})
	}
}

func TestClient_Submit_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	blobModule := mocks.NewMockBlobModule(t)
	blobModule.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(uint64(10), nil)

	cl := NewClient(Config{
		DA:            makeBlobRPCClient(blobModule, nil),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, nil)
	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Equal(t, uint64(10), res.Height)
	require.Len(t, res.IDs, 1)
}

func TestClient_Submit_InvalidNamespace(t *testing.T) {
	blobModule := mocks.NewMockBlobModule(t)
	cl := NewClient(Config{
		DA:            makeBlobRPCClient(blobModule, nil),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, []byte{0x01, 0x02}, nil)
	require.Equal(t, datypes.StatusError, res.Code)
}

func TestClient_Retrieve_NotFound(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	blobModule := mocks.NewMockBlobModule(t)
	blobModule.On("GetAll", mock.Anything, mock.Anything, mock.Anything).Return([]*blobrpc.Blob(nil), datypes.ErrBlobNotFound)
	headerModule := makeDefaultHeaderModule(t)

	cl := NewClient(Config{
		DA:            makeBlobRPCClient(blobModule, headerModule),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 5, ns)
	require.Equal(t, datypes.StatusNotFound, res.Code)
	// Verify timestamp is the fixed time from the header module
	expectedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	require.Equal(t, expectedTime, res.Timestamp)
}

func TestClient_Retrieve_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	nsBz := ns.Bytes()
	b, err := blobrpc.NewBlobV0(ns, []byte("payload"))
	require.NoError(t, err)
	blobModule := mocks.NewMockBlobModule(t)
	// GetAll returns all blobs at the height/namespace
	blobModule.On("GetAll", mock.Anything, uint64(7), mock.Anything).Return([]*blobrpc.Blob{b}, nil)
	headerModule := makeDefaultHeaderModule(t)

	cl := NewClient(Config{
		DA:            makeBlobRPCClient(blobModule, headerModule),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 7, nsBz)
	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Len(t, res.Data, 1)
	require.Len(t, res.IDs, 1)
	// Verify timestamp is the fixed time from the header module
	expectedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	require.Equal(t, expectedTime, res.Timestamp)
}

func TestClient_SubmitOptionsMerge(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	blobModule := mocks.NewMockBlobModule(t)
	blobModule.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(uint64(1), nil)

	cl := NewClient(Config{
		DA:            makeBlobRPCClient(blobModule, nil),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})

	opts := map[string]any{"signer_address": "signer1xyz"}
	raw, err := json.Marshal(opts)
	require.NoError(t, err)

	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, ns, raw)
	require.Equal(t, datypes.StatusSuccess, res.Code)
}

// TestClient_Get tests the Get method.
func TestClient_Get(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	nsBz := ns.Bytes()

	t.Run("Get fetches blobs by IDs", func(t *testing.T) {
		blobModule := mocks.NewMockBlobModule(t)

		blobs := make([]*blobrpc.Blob, 3)
		ids := make([]datypes.ID, 3)
		for i := range 3 {
			blb, err := blobrpc.NewBlobV0(ns, []byte{byte(i)})
			require.NoError(t, err)
			blobs[i] = blb
			ids[i] = blobrpc.MakeID(uint64(100+i), blb.Commitment)
			blobModule.On("Get", mock.Anything, uint64(100+i), ns, blb.Commitment).Return(blb, nil).Once()
		}

		cl := NewClient(Config{
			DA:            makeBlobRPCClient(blobModule, nil),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		})

		result, err := cl.Get(context.Background(), ids, nsBz)
		require.NoError(t, err)
		require.Len(t, result, 3)
		for i := range 3 {
			assert.Equal(t, blobs[i].Data(), result[i])
		}
	})

	t.Run("Get propagates errors", func(t *testing.T) {
		blobModule := mocks.NewMockBlobModule(t)
		blb, _ := blobrpc.NewBlobV0(ns, []byte{0})
		ids := []datypes.ID{blobrpc.MakeID(100, blb.Commitment)}
		blobModule.On("Get", mock.Anything, uint64(100), ns, blb.Commitment).Return(nil, errors.New("network error")).Once()

		cl := NewClient(Config{
			DA:            makeBlobRPCClient(blobModule, nil),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		})

		_, err := cl.Get(context.Background(), ids, nsBz)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "network error")
	})
}

// TestClient_GetProofs tests the GetProofs method.
func TestClient_GetProofs(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	nsBz := ns.Bytes()

	blobModule := mocks.NewMockBlobModule(t)

	ids := make([]datypes.ID, 3)
	for i := range 3 {
		blb, _ := blobrpc.NewBlobV0(ns, []byte{byte(i)})
		ids[i] = blobrpc.MakeID(uint64(200+i), blb.Commitment)
		blobModule.On("GetProof", mock.Anything, uint64(200+i), ns, blb.Commitment).Return(&blobrpc.Proof{}, nil).Once()
	}

	cl := NewClient(Config{
		DA:            makeBlobRPCClient(blobModule, nil),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})

	proofs, err := cl.GetProofs(context.Background(), ids, nsBz)
	require.NoError(t, err)
	require.Len(t, proofs, 3)
}

// TestClient_Validate tests the Validate method.
func TestClient_Validate(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	nsBz := ns.Bytes()

	t.Run("Validate with mixed results", func(t *testing.T) {
		blobModule := mocks.NewMockBlobModule(t)

		ids := make([]datypes.ID, 3)
		proofs := make([]datypes.Proof, 3)
		for i := range 3 {
			blb, _ := blobrpc.NewBlobV0(ns, []byte{byte(i)})
			ids[i] = blobrpc.MakeID(uint64(300+i), blb.Commitment)
			proofBz, _ := json.Marshal(&blobrpc.Proof{})
			proofs[i] = proofBz
			blobModule.On("Included", mock.Anything, uint64(300+i), ns, mock.Anything, blb.Commitment).Return(i%2 == 0, nil).Once()
		}

		cl := NewClient(Config{
			DA:            makeBlobRPCClient(blobModule, nil),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		})

		results, err := cl.Validate(context.Background(), ids, proofs, nsBz)
		require.NoError(t, err)
		require.Len(t, results, 3)
		for i := range 3 {
			assert.Equal(t, i%2 == 0, results[i])
		}
	})

	t.Run("Validate continues on inclusion check error", func(t *testing.T) {
		blobModule := mocks.NewMockBlobModule(t)

		blb0, _ := blobrpc.NewBlobV0(ns, []byte{0})
		blb1, _ := blobrpc.NewBlobV0(ns, []byte{1})
		ids := []datypes.ID{
			blobrpc.MakeID(400, blb0.Commitment),
			blobrpc.MakeID(401, blb1.Commitment),
		}
		proofs := make([]datypes.Proof, 2)
		for i := range proofs {
			proofs[i], _ = json.Marshal(&blobrpc.Proof{})
		}

		blobModule.On("Included", mock.Anything, uint64(400), ns, mock.Anything, blb0.Commitment).Return(true, nil).Once()
		blobModule.On("Included", mock.Anything, uint64(401), ns, mock.Anything, blb1.Commitment).Return(false, errors.New("check failed")).Once()

		cl := NewClient(Config{
			DA:            makeBlobRPCClient(blobModule, nil),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		})

		results, err := cl.Validate(context.Background(), ids, proofs, nsBz)
		require.NoError(t, err)
		assert.True(t, results[0])
		assert.False(t, results[1])
	})

	t.Run("Validate rejects mismatched ids and proofs", func(t *testing.T) {
		blobModule := mocks.NewMockBlobModule(t)
		cl := NewClient(Config{
			DA:            makeBlobRPCClient(blobModule, nil),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		})

		_, err := cl.Validate(context.Background(), make([]datypes.ID, 3), make([]datypes.Proof, 2), nsBz)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must match")
	})
}
