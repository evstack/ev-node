package da

import (
	"context"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Client defines the complete interface for DA layer operations.
// This is the interface implemented by the concrete DA client.
type Client interface {
	// Submit submits blobs to the DA layer.
	Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit

	// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
	Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve

	// Get retrieves blobs by their IDs. Used for visualization and fetching specific blobs.
	Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error)

	// GetLatestDAHeight returns the latest height available on the DA layer.
	GetLatestDAHeight(ctx context.Context) (uint64, error)

	// Namespace accessors.
	GetHeaderNamespace() []byte
	GetDataNamespace() []byte
	GetForcedInclusionNamespace() []byte
	HasForcedInclusionNamespace() bool

	// GetProofs returns inclusion Proofs for Blobs specified by their IDs.
	GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error)

	// Validate validates Commitments against the corresponding Proofs.
	Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error)
}
