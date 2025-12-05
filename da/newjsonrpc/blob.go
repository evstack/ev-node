package newjsonrpc

import (
	"context"

	libshare "github.com/celestiaorg/go-square/v3/share"

	"github.com/evstack/ev-node/pkg/blob"
)

// BlobAPI mirrors celestia-node's blob module (nodebuilder/blob/blob.go).
// jsonrpc.NewClient wires Internal.* to RPC stubs.
type BlobAPI struct {
	Internal struct {
		Submit func(
			context.Context,
			[]*blob.Blob,
			*blob.SubmitOptions,
		) (uint64, error) `perm:"write"`
		Get func(
			context.Context,
			uint64,
			libshare.Namespace,
			blob.Commitment,
		) (*blob.Blob, error) `perm:"read"`
		GetAll func(
			context.Context,
			uint64,
			[]libshare.Namespace,
		) ([]*blob.Blob, error) `perm:"read"`
		GetProof func(
			context.Context,
			uint64,
			libshare.Namespace,
			blob.Commitment,
		) (*blob.Proof, error) `perm:"read"`
		Included func(
			context.Context,
			uint64,
			libshare.Namespace,
			*blob.Proof,
			blob.Commitment,
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
func (api *BlobAPI) Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error) {
	return api.Internal.Submit(ctx, blobs, opts)
}

// Get retrieves a blob by commitment under the given namespace and height.
func (api *BlobAPI) Get(ctx context.Context, height uint64, namespace libshare.Namespace, commitment blob.Commitment) (*blob.Blob, error) {
	return api.Internal.Get(ctx, height, namespace, commitment)
}

// GetAll returns all blobs for the given namespaces at the given height.
func (api *BlobAPI) GetAll(ctx context.Context, height uint64, namespaces []libshare.Namespace) ([]*blob.Blob, error) {
	return api.Internal.GetAll(ctx, height, namespaces)
}

// GetProof retrieves proofs in the given namespace at the given height by commitment.
func (api *BlobAPI) GetProof(ctx context.Context, height uint64, namespace libshare.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	return api.Internal.GetProof(ctx, height, namespace, commitment)
}

// Included checks whether a blob commitment is included at the given height/namespace.
func (api *BlobAPI) Included(ctx context.Context, height uint64, namespace libshare.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
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
