package da

import (
	"context"

	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Client represents the DA client contract.
type Client interface {
	// Submit submits blobs to the DA layer.
	Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit

	// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
	Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve

	// Get retrieves blobs by their IDs. Used for visualization and fetching specific blobs.
	Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error)

	// Namespace accessors.
	GetHeaderNamespace() []byte
	GetDataNamespace() []byte
	GetForcedInclusionNamespace() []byte
	HasForcedInclusionNamespace() bool

	// Subscribe subscribes to blobs in the specified namespace.
	// Returns a channel that receives subscription responses as new blobs are included.
	// Used for follow mode to receive real-time blob notifications.
	Subscribe(ctx context.Context, namespace []byte) (<-chan *blobrpc.SubscriptionResponse, error)

	// LocalHead returns the height the DA node (e.g., Celestia light node) has synced to.
	// This is NOT the ev-node's processed height - it's the DA layer node's local head.
	// Used to determine if ev-node is caught up with the DA layer.
	LocalHead(ctx context.Context) (uint64, error)
}

// Verifier defines the interface for DA proof verification operations.
// This is a subset of the DA interface used by sequencers to verify batch inclusion.
type Verifier interface {
	// GetProofs returns inclusion Proofs for Blobs specified by their IDs.
	GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error)

	// Validate validates Commitments against the corresponding Proofs.
	Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error)
}

// FullClient combines Client and Verifier interfaces.
// This is the complete interface implemented by the concrete DA client.
type FullClient interface {
	Client
	Verifier
}
