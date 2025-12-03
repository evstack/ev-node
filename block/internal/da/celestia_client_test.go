package da

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	celestia "github.com/evstack/ev-node/da/celestia"
	"github.com/evstack/ev-node/pkg/blob"
)

type mockCelestiaBlobAPI struct {
	submitErr   error
	height      uint64
	blobs       []*blob.Blob
	proof       *blob.Proof
	included    bool
	commitProof *celestia.CommitmentProof
}

func (m *mockCelestiaBlobAPI) Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error) {
	return m.height, m.submitErr
}

func (m *mockCelestiaBlobAPI) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	return m.blobs, m.submitErr
}

func (m *mockCelestiaBlobAPI) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	return m.proof, m.submitErr
}

func (m *mockCelestiaBlobAPI) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	return m.included, m.submitErr
}

func (m *mockCelestiaBlobAPI) GetCommitmentProof(ctx context.Context, height uint64, namespace share.Namespace, shareCommitment []byte) (*celestia.CommitmentProof, error) {
	return m.commitProof, m.submitErr
}

func (m *mockCelestiaBlobAPI) Subscribe(ctx context.Context, namespace share.Namespace) (<-chan *celestia.SubscriptionResponse, error) {
	ch := make(chan *celestia.SubscriptionResponse)
	close(ch)
	return ch, nil
}

func makeCelestiaClient(m *mockCelestiaBlobAPI) *celestia.Client {
	var api celestia.BlobAPI
	api.Internal.Submit = m.Submit
	api.Internal.GetAll = m.GetAll
	api.Internal.GetProof = m.GetProof
	api.Internal.Included = m.Included
	api.Internal.GetCommitmentProof = m.GetCommitmentProof
	api.Internal.Subscribe = m.Subscribe
	return &celestia.Client{Blob: api}
}

func TestCelestiaClient_Submit_ErrorMapping(t *testing.T) {
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
			cl := NewCelestiaBlob(CelestiaBlobConfig{
				Celestia:      makeCelestiaClient(&mockCelestiaBlobAPI{submitErr: tc.err}),
				Logger:        zerolog.Nop(),
				Namespace:     "ns",
				DataNamespace: "ns",
			})
			res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, nil)
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
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, nil)
	require.Equal(t, StatusSuccess, res.Code)
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
	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, []byte{0x01, 0x02}, nil)
	require.Equal(t, StatusError, res.Code)
}

func TestCelestiaClient_Retrieve_NotFound(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	mockAPI := &mockCelestiaBlobAPI{submitErr: ErrBlobNotFound}
	cl := NewCelestiaBlob(CelestiaBlobConfig{
		Celestia:      makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 5, ns)
	require.Equal(t, StatusNotFound, res.Code)
}

func TestCelestiaClient_Retrieve_Success(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns")).Bytes()
	b, err := blob.NewBlobV0(share.MustNewV0Namespace([]byte("ns")), []byte("payload"))
	require.NoError(t, err)
	mockAPI := &mockCelestiaBlobAPI{height: 7, blobs: []*blob.Blob{b}}
	cl := NewCelestiaBlob(CelestiaBlobConfig{
		Celestia:      makeCelestiaClient(mockAPI),
		Logger:        zerolog.Nop(),
		Namespace:     "ns",
		DataNamespace: "ns",
	})
	res := cl.Retrieve(context.Background(), 7, ns)
	require.Equal(t, StatusSuccess, res.Code)
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

	res := cl.Submit(context.Background(), [][]byte{[]byte("data")}, ns, raw)
	require.Equal(t, StatusSuccess, res.Code)
}

func TestLocalBlobAPI_SubmitAndGetAll(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	api := NewLocalBlobAPI(1024)

	b, err := blob.NewBlobV0(ns, []byte("payload"))
	require.NoError(t, err)

	height, err := api.Submit(context.Background(), []*blob.Blob{b}, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), height)

	blobs, err := api.GetAll(context.Background(), height, []share.Namespace{ns})
	require.NoError(t, err)
	require.Len(t, blobs, 1)
	assert.Equal(t, b.Data(), blobs[0].Data())
}

func TestLocalBlobAPI_MaxSize(t *testing.T) {
	ns := share.MustNewV0Namespace([]byte("ns"))
	api := NewLocalBlobAPI(2)

	b, err := blob.NewBlobV0(ns, []byte("long"))
	require.NoError(t, err)

	_, err = api.Submit(context.Background(), []*blob.Blob{b}, nil)
	require.Error(t, err)
}
