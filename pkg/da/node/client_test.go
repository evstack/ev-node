package node_test

import (
	"context"
	"net/http/httptest"
	"testing"

	fcjsonrpc "github.com/filecoin-project/go-jsonrpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/evstack/ev-node/pkg/da/node"
	"github.com/evstack/ev-node/pkg/da/node/mocks"
)

func newTestServer(t *testing.T, module any) *httptest.Server {
	t.Helper()
	rpc := fcjsonrpc.NewServer()
	rpc.Register("blob", module)
	return httptest.NewServer(rpc)
}

func TestClient_CallsAreForwarded(t *testing.T) {
	ns := libshare.MustNewV0Namespace([]byte("namespace"))
	blb, err := node.NewBlobV0(ns, []byte("data"))
	require.NoError(t, err)

	module := mocks.NewMockBlobModule(t)
	module.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(uint64(7), nil)
	module.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(blb, nil)
	module.On("GetAll", mock.Anything, mock.Anything, mock.Anything).Return([]*node.Blob{blb}, nil)
	module.On("GetProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&node.Proof{}, nil)
	module.On("Included", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	module.On("GetCommitmentProof", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&node.CommitmentProof{}, nil)

	srv := newTestServer(t, module)
	t.Cleanup(srv.Close)

	client, err := node.NewClient(context.Background(), srv.URL, "", "")
	require.NoError(t, err)
	t.Cleanup(client.Close)

	height, err := client.Blob.Submit(context.Background(), []*node.Blob{blb}, nil)
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
