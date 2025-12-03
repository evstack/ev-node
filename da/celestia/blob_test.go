package celestia

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/stretchr/testify/require"

	libshare "github.com/celestiaorg/go-square/v3/share"

	"github.com/evstack/ev-node/pkg/blob"
)

type mockBlobModule struct {
	submitHeight uint64
	submitErr    error

	blob        *blob.Blob
	proof       *blob.Proof
	included    bool
	commitProof *CommitmentProof
}

func (m *mockBlobModule) Submit(_ context.Context, _ []*blob.Blob, _ *blob.SubmitOptions) (uint64, error) {
	return m.submitHeight, m.submitErr
}

func (m *mockBlobModule) Get(_ context.Context, _ uint64, _ libshare.Namespace, _ blob.Commitment) (*blob.Blob, error) {
	return m.blob, nil
}

func (m *mockBlobModule) GetAll(_ context.Context, _ uint64, _ []libshare.Namespace) ([]*blob.Blob, error) {
	return []*blob.Blob{m.blob}, nil
}

func (m *mockBlobModule) GetProof(_ context.Context, _ uint64, _ libshare.Namespace, _ blob.Commitment) (*blob.Proof, error) {
	return m.proof, nil
}

func (m *mockBlobModule) Included(_ context.Context, _ uint64, _ libshare.Namespace, _ *blob.Proof, _ blob.Commitment) (bool, error) {
	return m.included, nil
}

func (m *mockBlobModule) GetCommitmentProof(_ context.Context, _ uint64, _ libshare.Namespace, _ []byte) (*CommitmentProof, error) {
	return m.commitProof, nil
}

func (m *mockBlobModule) Subscribe(_ context.Context, _ libshare.Namespace) (<-chan *SubscriptionResponse, error) {
	ch := make(chan *SubscriptionResponse, 1)
	ch <- &SubscriptionResponse{Height: 11}
	close(ch)
	return ch, nil
}

func newTestServer(t *testing.T, module any) *httptest.Server {
	t.Helper()
	rpc := jsonrpc.NewServer()
	rpc.Register("blob", module)
	return httptest.NewServer(rpc)
}

func TestClient_CallsAreForwarded(t *testing.T) {
	ns := libshare.MustNewV0Namespace([]byte("namespace"))
	blb, err := blob.NewBlobV0(ns, []byte("data"))
	require.NoError(t, err)

	module := &mockBlobModule{
		submitHeight: 7,
		blob:         blb,
		proof:        &blob.Proof{},
		included:     true,
		commitProof:  &CommitmentProof{},
	}
	srv := newTestServer(t, module)
	t.Cleanup(srv.Close)

	client, err := NewClient(context.Background(), srv.URL, "", "")
	require.NoError(t, err)
	t.Cleanup(client.Close)

	height, err := client.Blob.Submit(context.Background(), []*blob.Blob{blb}, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(7), height)

	gotBlob, err := client.Blob.Get(context.Background(), 7, ns, blb.Commitment)
	require.NoError(t, err)
	require.Equal(t, blb.Commitment, gotBlob.Commitment)

	all, err := client.Blob.GetAll(context.Background(), 7, []libshare.Namespace{ns})
	require.NoError(t, err)
	require.Len(t, all, 1)

	pf, err := client.Blob.GetProof(context.Background(), 7, ns, blb.Commitment)
	require.NoError(t, err)
	require.NotNil(t, pf)

	ok, err := client.Blob.Included(context.Background(), 7, ns, pf, blb.Commitment)
	require.NoError(t, err)
	require.True(t, ok)

	cp, err := client.Blob.GetCommitmentProof(context.Background(), 7, ns, []byte("commit"))
	require.NoError(t, err)
	require.NotNil(t, cp)
}
