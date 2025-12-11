package blob

import (
	"context"

	libshare "github.com/celestiaorg/go-square/v3/share"
)

// BlobModule is the server-side "blob" JSON-RPC interface used by tests/mocks.
type BlobModule interface {
	Submit(context.Context, []*Blob, *SubmitOptions) (uint64, error)
	Get(context.Context, uint64, libshare.Namespace, Commitment) (*Blob, error)
	GetAll(context.Context, uint64, []libshare.Namespace) ([]*Blob, error)
	GetProof(context.Context, uint64, libshare.Namespace, Commitment) (*Proof, error)
	Included(context.Context, uint64, libshare.Namespace, *Proof, Commitment) (bool, error)
	GetCommitmentProof(context.Context, uint64, libshare.Namespace, []byte) (*CommitmentProof, error)
	Subscribe(context.Context, libshare.Namespace) (<-chan *SubscriptionResponse, error)
}
