package da

import (
	"context"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Client represents the DA client contract.
type Client interface {
	Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit
	Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve
	RetrieveHeaders(ctx context.Context, height uint64) datypes.ResultRetrieve
	RetrieveData(ctx context.Context, height uint64) datypes.ResultRetrieve
	RetrieveForcedInclusion(ctx context.Context, height uint64) datypes.ResultRetrieve
	Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error)
	GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error)
	Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error)

	GetHeaderNamespace() []byte
	GetDataNamespace() []byte
	GetForcedInclusionNamespace() []byte
	HasForcedInclusionNamespace() bool
}
