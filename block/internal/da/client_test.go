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
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	b, err := blobrpc.NewBlobV0(share.MustNewV0Namespace([]byte("ns")), []byte("payload"))
	require.NoError(t, err)
	module := mocks.NewMockBlobModule(t)
	module.On("GetAll", mock.Anything, uint64(7), mock.Anything).Return([]*blobrpc.Blob{b}, nil)

	cl := NewClient(Config{
		Client:        makeBlobRPCClient(module),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 7, ns)
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

func TestClient_RetrieveHeaders(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("header-ns"))
	blb, err := blobrpc.NewBlobV0(ns, []byte("header-blob"))
	require.NoError(t, err)
	module := mocks.NewMockBlobModule(t)
	module.On("GetAll", mock.Anything, uint64(42), mock.Anything).Return([]*blobrpc.Blob{blb}, nil)

	client := NewClient(Config{
		Client:        makeBlobRPCClient(module),
		Logger:        zerolog.Nop(),
		Namespace:     "header-ns",
		DataNamespace: "data-ns",
	})

	result := client.RetrieveHeaders(context.Background(), 42)

	assert.Equal(t, datypes.StatusSuccess, result.Code)
	assert.Equal(t, uint64(42), result.Height)
	assert.Equal(t, 1, len(result.Data))
}

func TestClient_RetrieveData(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("data-ns"))
	blb1, err := blobrpc.NewBlobV0(ns, []byte("data-blob-1"))
	require.NoError(t, err)
	blb2, err := blobrpc.NewBlobV0(ns, []byte("data-blob-2"))
	require.NoError(t, err)
	module := mocks.NewMockBlobModule(t)
	module.On("GetAll", mock.Anything, uint64(99), mock.Anything).Return([]*blobrpc.Blob{blb1, blb2}, nil)

	client := NewClient(Config{
		Client:        makeBlobRPCClient(module),
		Logger:        zerolog.Nop(),
		Namespace:     "header-ns",
		DataNamespace: "data-ns",
	})

	result := client.RetrieveData(context.Background(), 99)

	assert.Equal(t, datypes.StatusSuccess, result.Code)
	assert.Equal(t, uint64(99), result.Height)
	assert.Equal(t, 2, len(result.Data))
}
