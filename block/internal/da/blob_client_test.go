package da

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/celestiaorg/go-square/v3/share"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBlobClient_Submit_ErrorMapping(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	testCases := []struct {
		name       string
		err        error
		wantStatus StatusCode
	}{
		{"timeout", ErrTxTimedOut, StatusNotIncludedInBlock},
		{"alreadyInMempool", ErrTxAlreadyInMempool, StatusAlreadyInMempool},
		{"seq", ErrTxIncorrectAccountSequence, StatusIncorrectAccountSequence},
		{"tooBig", ErrBlobSizeOverLimit, StatusTooBig},
		{"deadline", ErrContextDeadline, StatusContextDeadline},
		{"canceled", context.Canceled, StatusContextCanceled},
		{"other", errors.New("boom"), StatusError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAPI := NewMockBlobAPI(t)
			mockAPI.On("Submit", mock.Anything, mock.Anything, mock.Anything).
				Return(uint64(0), tc.err)

			cl := NewBlobClient(mockAPI, BlobConfig{
				Logger:        zerolog.Nop(),
				Namespace:     "ns",
				DataNamespace: "ns",
			})
			require.NotNil(t, cl)
			res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, nil)
			assert.Equal(t, tc.wantStatus, res.Code)
		})
	}
}

func TestBlobClient_Submit_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := NewMockBlobAPI(t)
	mockAPI.On("Submit", mock.Anything, mock.Anything, mock.Anything).
		Return(uint64(10), nil)
	cl := NewBlobClient(mockAPI, BlobConfig{
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	require.NotNil(t, cl)
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, nil)
	require.Equal(t, StatusSuccess, res.Code)
	require.Equal(t, uint64(10), res.Height)
	require.Len(t, res.IDs, 1)
}

func TestBlobClient_Submit_InvalidNamespace(t *testing.T) {
	mockAPI := NewMockBlobAPI(t)
	cl := NewBlobClient(mockAPI, BlobConfig{
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	require.NotNil(t, cl)
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, []byte{0x01, 0x02}, nil)
	require.Equal(t, StatusError, res.Code)
}

func TestBlobClient_Retrieve_NotFound(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := NewMockBlobAPI(t)
	mockAPI.On("GetAll", mock.Anything, uint64(5), []share.Namespace{share.MustNewV0Namespace([]byte("ns"))}).
		Return([]*blobrpc.Blob(nil), ErrBlobNotFound)
	cl := NewBlobClient(mockAPI, BlobConfig{
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	require.NotNil(t, cl)
	res := cl.Retrieve(context.Background(), 5, ns)
	require.Equal(t, StatusNotFound, res.Code)
}

func TestBlobClient_Retrieve_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	b, err := blobrpc.NewBlobV0(share.MustNewV0Namespace([]byte("ns")), []byte("payload"))
	require.NoError(t, err)
	mockAPI := NewMockBlobAPI(t)
	mockAPI.On("GetAll", mock.Anything, uint64(7), []share.Namespace{share.MustNewV0Namespace([]byte("ns"))}).
		Return([]*blobrpc.Blob{b}, nil)
	cl := NewBlobClient(mockAPI, BlobConfig{
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	require.NotNil(t, cl)
	res := cl.Retrieve(context.Background(), 7, ns)
	require.Equal(t, StatusSuccess, res.Code)
	require.Len(t, res.Data, 1)
	require.Len(t, res.IDs, 1)
}

func TestBlobClient_SubmitOptionsMerge(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := NewMockBlobAPI(t)
	mockAPI.On("Submit", mock.Anything, mock.Anything, mock.Anything).
		Return(uint64(1), nil)
	cl := NewBlobClient(mockAPI, BlobConfig{
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	require.NotNil(t, cl)

	opts := map[string]any{"signer_address": "celestia1xyz"}
	raw, err := json.Marshal(opts)
	require.NoError(t, err)

	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, raw)
	require.Equal(t, StatusSuccess, res.Code)
}
