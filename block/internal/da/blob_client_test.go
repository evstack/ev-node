package da

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/celestiaorg/go-square/v3/share"
	blobrpc "github.com/evstack/ev-node/da/jsonrpc/blob"
	datypes "github.com/evstack/ev-node/pkg/da/types"
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
			cl := NewCelestiaBlob(CelestiaBlobConfig{
				Celestia:      makeCelestiaClient(&mockCelestiaBlobAPI{submitErr: tc.err}),
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
	mockAPI := &mockCelestiaBlobAPI{height: 10}
	cl := NewCelestiaBlob(CelestiaBlobConfig{
		Celestia:      makeCelestiaClient(mockAPI),
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
	mockAPI := &mockCelestiaBlobAPI{height: 10}
	cl := NewCelestiaBlob(CelestiaBlobConfig{
		Celestia:      makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, 0, []byte{0x01, 0x02}, nil)
	require.Equal(t, datypes.StatusError, res.Code)
}

func TestCelestiaClient_Retrieve_NotFound(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockCelestiaBlobAPI{submitErr: datypes.ErrBlobNotFound}
	cl := NewCelestiaBlob(CelestiaBlobConfig{
		Celestia:      makeCelestiaClient(mockAPI),
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
	mockAPI := &mockCelestiaBlobAPI{height: 7, blobs: []*blobrpc.Blob{b}}
	cl := NewCelestiaBlob(CelestiaBlobConfig{
		Celestia:      makeCelestiaClient(mockAPI),
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
	mockAPI := &mockCelestiaBlobAPI{height: 1}
	cl := NewCelestiaBlob(CelestiaBlobConfig{
		Celestia:      makeCelestiaClient(mockAPI),
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
