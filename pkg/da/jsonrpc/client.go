package jsonrpc

import (
	"context"
	"fmt"
	"net/http"

	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/filecoin-project/go-jsonrpc"
)

// Client dials the celestia-node RPC "blob" namespace.
type Client struct {
	Blob   BlobAPI
	closer jsonrpc.ClientCloser
}

// Close closes the underlying JSON-RPC connection.
func (c *Client) Close() {
	if c != nil && c.closer != nil {
		c.closer()
	}
}

// NewClient connects to the celestia-node RPC endpoint
func NewClient(ctx context.Context, addr, token string, authHeaderName string) (*Client, error) {
	var header http.Header
	if token != "" {
		if authHeaderName == "" {
			authHeaderName = "Authorization"
		}
		header = http.Header{authHeaderName: []string{fmt.Sprintf("Bearer %s", token)}}
	}

	var cl Client
	closer, err := jsonrpc.NewClient(ctx, addr, "blob", &cl.Blob.Internal, header)
	if err != nil {
		return nil, err
	}
	cl.closer = closer
	return &cl, nil
}

// BlobAPI mirrors celestia-node's blob module (nodebuilder/blob/blob.go).
// jsonrpc.NewClient wires Internal.* to RPC stubs.
type BlobAPI struct {
	Internal struct {
		Submit func(
			context.Context,
			[]*Blob,
			*SubmitOptions,
		) (uint64, error) `perm:"write"`
		Get func(
			context.Context,
			uint64,
			libshare.Namespace,
			Commitment,
		) (*Blob, error) `perm:"read"`
		GetAll func(
			context.Context,
			uint64,
			[]libshare.Namespace,
		) ([]*Blob, error) `perm:"read"`
		GetProof func(
			context.Context,
			uint64,
			libshare.Namespace,
			Commitment,
		) (*Proof, error) `perm:"read"`
		Included func(
			context.Context,
			uint64,
			libshare.Namespace,
			*Proof,
			Commitment,
		) (bool, error) `perm:"read"`
		GetCommitmentProof func(
			context.Context,
			uint64,
			libshare.Namespace,
			[]byte,
		) (*CommitmentProof, error) `perm:"read"`
		Subscribe func(
			context.Context,
			libshare.Namespace,
		) (<-chan *SubscriptionResponse, error) `perm:"read"`
	}
}

// Submit sends blobs and returns the height they were included at.
func (api *BlobAPI) Submit(ctx context.Context, blobs []*Blob, opts *SubmitOptions) (uint64, error) {
	return api.Internal.Submit(ctx, blobs, opts)
}

// Get retrieves a blob by commitment under the given namespace and height.
func (api *BlobAPI) Get(ctx context.Context, height uint64, namespace libshare.Namespace, commitment Commitment) (*Blob, error) {
	return api.Internal.Get(ctx, height, namespace, commitment)
}

// GetAll returns all blobs for the given namespaces at the given height.
func (api *BlobAPI) GetAll(ctx context.Context, height uint64, namespaces []libshare.Namespace) ([]*Blob, error) {
	return api.Internal.GetAll(ctx, height, namespaces)
}

// GetProof retrieves proofs in the given namespace at the given height by commitment.
func (api *BlobAPI) GetProof(ctx context.Context, height uint64, namespace libshare.Namespace, commitment Commitment) (*Proof, error) {
	return api.Internal.GetProof(ctx, height, namespace, commitment)
}

// Included checks whether a blob commitment is included at the given height/namespace.
func (api *BlobAPI) Included(ctx context.Context, height uint64, namespace libshare.Namespace, proof *Proof, commitment Commitment) (bool, error) {
	return api.Internal.Included(ctx, height, namespace, proof, commitment)
}

// GetCommitmentProof generates a commitment proof for a share commitment.
func (api *BlobAPI) GetCommitmentProof(ctx context.Context, height uint64, namespace libshare.Namespace, shareCommitment []byte) (*CommitmentProof, error) {
	return api.Internal.GetCommitmentProof(ctx, height, namespace, shareCommitment)
}

// Subscribe streams blobs as they are included for the given namespace.
func (api *BlobAPI) Subscribe(ctx context.Context, namespace libshare.Namespace) (<-chan *SubscriptionResponse, error) {
	return api.Internal.Subscribe(ctx, namespace)
}
