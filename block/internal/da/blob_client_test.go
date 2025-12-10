package da

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/celestiaorg/go-square/v3/share"
	blobrpc "github.com/evstack/ev-node/da/jsonrpc/blob"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCelestiaBlobAPI struct {
	submitErr   error
	height      uint64
	blobs       []*blobrpc.Blob
	proof       *blobrpc.Proof
	included    bool
	commitProof *blobrpc.CommitmentProof
}

func (m *mockCelestiaBlobAPI) Submit(ctx context.Context, blobs []*blobrpc.Blob, opts *blobrpc.SubmitOptions) (uint64, error) {
	return m.height, m.submitErr
}

func (m *mockCelestiaBlobAPI) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blobrpc.Blob, error) {
	return m.blobs, m.submitErr
}

func (m *mockCelestiaBlobAPI) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blobrpc.Commitment) (*blobrpc.Proof, error) {
	return m.proof, m.submitErr
}

func (m *mockCelestiaBlobAPI) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blobrpc.Proof, commitment blobrpc.Commitment) (bool, error) {
	return m.included, m.submitErr
}

func (m *mockCelestiaBlobAPI) GetCommitmentProof(ctx context.Context, height uint64, namespace share.Namespace, shareCommitment []byte) (*blobrpc.CommitmentProof, error) {
	return m.commitProof, m.submitErr
}

func (m *mockCelestiaBlobAPI) Subscribe(ctx context.Context, namespace share.Namespace) (<-chan *blobrpc.SubscriptionResponse, error) {
	ch := make(chan *blobrpc.SubscriptionResponse)
	close(ch)
	return ch, nil
}

func makeCelestiaClient(m *mockCelestiaBlobAPI) *blobrpc.Client {
	var api blobrpc.BlobAPI
	api.Internal.Submit = m.Submit
	api.Internal.GetAll = m.GetAll
	api.Internal.GetProof = m.GetProof
	api.Internal.Included = m.Included
	api.Internal.GetCommitmentProof = m.GetCommitmentProof
	api.Internal.Subscribe = m.Subscribe
	return &blobrpc.Client{Blob: api}
}

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
			cl := NewBlobClient(BlobConfig{
				Client:        makeCelestiaClient(&mockCelestiaBlobAPI{submitErr: tc.err}),
				Logger:        zerolog.Nop(),
				Namespace:     "ns",
				DataNamespace: "ns",
			})
			res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, nil)
			assert.Equal(t, tc.wantStatus, res.Code)
		})
	}
}

func TestBlobClient_Submit_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockCelestiaBlobAPI{height: 10}
	cl := NewBlobClient(BlobConfig{
		Client:        makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, nil)
	require.Equal(t, StatusSuccess, res.Code)
	require.Equal(t, uint64(10), res.Height)
	require.Len(t, res.IDs, 1)
}

func TestBlobClient_Submit_InvalidNamespace(t *testing.T) {
	mockAPI := &mockCelestiaBlobAPI{height: 10}
	cl := NewBlobClient(BlobConfig{
		Client:        makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, []byte{0x01, 0x02}, nil)
	require.Equal(t, StatusError, res.Code)
}

func TestBlobClient_Retrieve_NotFound(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockCelestiaBlobAPI{submitErr: ErrBlobNotFound}
	cl := NewBlobClient(BlobConfig{
		Client:        makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 5, ns)
	require.Equal(t, StatusNotFound, res.Code)
}

func TestBlobClient_Retrieve_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	b, err := blobrpc.NewBlobV0(share.MustNewV0Namespace([]byte("ns")), []byte("payload"))
	require.NoError(t, err)
	mockAPI := &mockCelestiaBlobAPI{height: 7, blobs: []*blobrpc.Blob{b}}
	cl := NewBlobClient(BlobConfig{
		Client:        makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 7, ns)
	require.Equal(t, StatusSuccess, res.Code)
	require.Len(t, res.Data, 1)
	require.Len(t, res.IDs, 1)
}

func TestBlobClient_SubmitOptionsMerge(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockCelestiaBlobAPI{height: 1}
	cl := NewBlobClient(BlobConfig{
		Client:        makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})

	opts := map[string]any{"signer_address": "celestia1xyz"}
	raw, err := json.Marshal(opts)
	require.NoError(t, err)

	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, raw)
	require.Equal(t, StatusSuccess, res.Code)
}
