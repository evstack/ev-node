package da

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/celestiaorg/go-square/v3/share"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	"github.com/evstack/ev-node/pkg/da/jsonrpc/mocks"
	datypes "github.com/evstack/ev-node/pkg/da/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func makeBlobRPCClient(module *mocks.MockBlobModule) *blobrpc.Client {
	var api blobrpc.BlobAPI
	api.Internal.Submit = module.Submit
	api.Internal.Get = module.Get
	api.Internal.GetAll = module.GetAll
	api.Internal.GetProof = module.GetProof
	api.Internal.Included = module.Included
	api.Internal.GetCommitmentProof = module.GetCommitmentProof
	api.Internal.Subscribe = module.Subscribe
	return &blobrpc.Client{Blob: api}
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
			module := mocks.NewMockBlobModule(t)
			module.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(uint64(0), tc.err)

			cl := NewClient(Config{
				Client:        makeBlobRPCClient(module),
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
	module := mocks.NewMockBlobModule(t)
	module.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(uint64(10), nil)

	cl := NewClient(Config{
		Client:        makeBlobRPCClient(module),
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
	module := mocks.NewMockBlobModule(t)
	cl := NewClient(Config{
		Client:        makeBlobRPCClient(module),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, []byte{0x01, 0x02}, nil)
	require.Equal(t, datypes.StatusError, res.Code)
}

func TestClient_Retrieve_NotFound(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	module := mocks.NewMockBlobModule(t)
	module.On("GetAll", mock.Anything, mock.Anything, mock.Anything).Return([]*blobrpc.Blob(nil), datypes.ErrBlobNotFound)

	cl := NewClient(Config{
		Client:        makeBlobRPCClient(module),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 5, ns)
	require.Equal(t, datypes.StatusNotFound, res.Code)
}

func TestClient_Retrieve_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	nsBz := ns.Bytes()
	b, err := blobrpc.NewBlobV0(ns, []byte("payload"))
	require.NoError(t, err)
	module := mocks.NewMockBlobModule(t)
	// GetIDs calls GetAll to get blob IDs
	module.On("GetAll", mock.Anything, uint64(7), mock.Anything).Return([]*blobrpc.Blob{b}, nil)
	// Retrieve then calls Get in batches to fetch the actual data
	module.On("Get", mock.Anything, uint64(7), ns, b.Commitment).Return(b, nil)

	cl := NewClient(Config{
		Client:        makeBlobRPCClient(module),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 7, nsBz)
	require.Equal(t, datypes.StatusSuccess, res.Code)
	require.Len(t, res.Data, 1)
	require.Len(t, res.IDs, 1)
}

func TestClient_SubmitOptionsMerge(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	module := mocks.NewMockBlobModule(t)
	module.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(uint64(1), nil)

	cl := NewClient(Config{
		Client:        makeBlobRPCClient(module),
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


// TestClient_BatchProcessing tests the batching behavior for Get, GetProofs, and Validate.
// Tests core batching logic (multiple batches, context cancellation, error propagation)
// once using Get, then verifies GetProofs and Validate work correctly with batching.
// Note: These test internal methods on the concrete client type, not the interface.
func TestClient_BatchProcessing(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	nsBz := ns.Bytes()

	t.Run("Get processes multiple batches in order", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)

		// Create 5 blobs with batch size of 2 (3 batches: 2+2+1)
		blobs := make([]*blobrpc.Blob, 5)
		ids := make([]datypes.ID, 5)
		for i := 0; i < 5; i++ {
			blb, err := blobrpc.NewBlobV0(ns, []byte{byte(i)})
			require.NoError(t, err)
			blobs[i] = blb
			ids[i] = blobrpc.MakeID(uint64(100+i), blb.Commitment)
			module.On("Get", mock.Anything, uint64(100+i), ns, blb.Commitment).Return(blb, nil).Once()
		}

		cl := NewClient(Config{
			Client:            makeBlobRPCClient(module),
			Logger:            zerolog.Nop(),
			Namespace:         "ns",
			DataNamespace:     "ns",
			RetrieveBatchSize: 2,
		}).(*client)

		result, err := cl.Get(context.Background(), ids, nsBz)
		require.NoError(t, err)
		require.Len(t, result, 5)
		for i := 0; i < 5; i++ {
			assert.Equal(t, blobs[i].Data(), result[i])
		}
	})

	t.Run("Get respects context cancellation", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)
		blb, _ := blobrpc.NewBlobV0(ns, []byte{0})
		ids := []datypes.ID{blobrpc.MakeID(100, blb.Commitment)}

		cl := NewClient(Config{
			Client:            makeBlobRPCClient(module),
			Logger:            zerolog.Nop(),
			Namespace:         "ns",
			DataNamespace:     "ns",
			RetrieveBatchSize: 2,
		}).(*client)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := cl.Get(ctx, ids, nsBz)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("Get propagates errors from batch", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)
		blb, _ := blobrpc.NewBlobV0(ns, []byte{0})
		ids := []datypes.ID{blobrpc.MakeID(100, blb.Commitment)}
		module.On("Get", mock.Anything, uint64(100), ns, blb.Commitment).Return(nil, errors.New("network error")).Once()

		cl := NewClient(Config{
			Client:        makeBlobRPCClient(module),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		}).(*client)

		_, err := cl.Get(context.Background(), ids, nsBz)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "network error")
	})

	t.Run("GetProofs batches correctly", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)

		ids := make([]datypes.ID, 5)
		for i := 0; i < 5; i++ {
			blb, _ := blobrpc.NewBlobV0(ns, []byte{byte(i)})
			ids[i] = blobrpc.MakeID(uint64(200+i), blb.Commitment)
			module.On("GetProof", mock.Anything, uint64(200+i), ns, blb.Commitment).Return(&blobrpc.Proof{}, nil).Once()
		}

		cl := NewClient(Config{
			Client:            makeBlobRPCClient(module),
			Logger:            zerolog.Nop(),
			Namespace:         "ns",
			DataNamespace:     "ns",
			RetrieveBatchSize: 2,
		}).(*client)

		proofs, err := cl.GetProofs(context.Background(), ids, nsBz)
		require.NoError(t, err)
		require.Len(t, proofs, 5)
	})

	t.Run("Validate batches correctly with mixed results", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)

		ids := make([]datypes.ID, 5)
		proofs := make([]datypes.Proof, 5)
		for i := 0; i < 5; i++ {
			blb, _ := blobrpc.NewBlobV0(ns, []byte{byte(i)})
			ids[i] = blobrpc.MakeID(uint64(300+i), blb.Commitment)
			proofBz, _ := json.Marshal(&blobrpc.Proof{})
			proofs[i] = proofBz
			module.On("Included", mock.Anything, uint64(300+i), ns, mock.Anything, blb.Commitment).Return(i%2 == 0, nil).Once()
		}

		cl := NewClient(Config{
			Client:            makeBlobRPCClient(module),
			Logger:            zerolog.Nop(),
			Namespace:         "ns",
			DataNamespace:     "ns",
			RetrieveBatchSize: 2,
		}).(*client)

		results, err := cl.Validate(context.Background(), ids, proofs, nsBz)
		require.NoError(t, err)
		require.Len(t, results, 5)
		for i := 0; i < 5; i++ {
			assert.Equal(t, i%2 == 0, results[i])
		}
	})

	t.Run("Validate continues on inclusion check error", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)

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

		module.On("Included", mock.Anything, uint64(400), ns, mock.Anything, blb0.Commitment).Return(true, nil).Once()
		module.On("Included", mock.Anything, uint64(401), ns, mock.Anything, blb1.Commitment).Return(false, errors.New("check failed")).Once()

		cl := NewClient(Config{
			Client:        makeBlobRPCClient(module),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		}).(*client)

		results, err := cl.Validate(context.Background(), ids, proofs, nsBz)
		require.NoError(t, err) // Validate logs errors but doesn't fail
		assert.True(t, results[0])
		assert.False(t, results[1])
	})

	t.Run("Validate rejects mismatched ids and proofs", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)
		cl := NewClient(Config{
			Client:        makeBlobRPCClient(module),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		}).(*client)

		_, err := cl.Validate(context.Background(), make([]datypes.ID, 3), make([]datypes.Proof, 2), nsBz)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must match")
	})
}

func TestClient_BatchSize_Configuration(t *testing.T) {
	t.Run("defaults to DefaultRetrieveBatchSize", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)
		cl := NewClient(Config{
			Client:        makeBlobRPCClient(module),
			Logger:        zerolog.Nop(),
			Namespace:     "ns",
			DataNamespace: "ns",
		})
		assert.Equal(t, DefaultRetrieveBatchSize, cl.(*client).batchSize)
	})

	t.Run("respects custom batch size", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)
		cl := NewClient(Config{
			Client:            makeBlobRPCClient(module),
			Logger:            zerolog.Nop(),
			Namespace:         "ns",
			DataNamespace:     "ns",
			RetrieveBatchSize: 50,
		})
		assert.Equal(t, 50, cl.(*client).batchSize)
	})

	t.Run("negative batch size defaults", func(t *testing.T) {
		module := mocks.NewMockBlobModule(t)
		cl := NewClient(Config{
			Client:            makeBlobRPCClient(module),
			Logger:            zerolog.Nop(),
			Namespace:         "ns",
			DataNamespace:     "ns",
			RetrieveBatchSize: -1,
		})
		assert.Equal(t, DefaultRetrieveBatchSize, cl.(*client).batchSize)
	})
}
