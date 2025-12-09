package datypes

import (
	"context"
	"time"

	coreda "github.com/evstack/ev-node/core/da"
)

// DA defines the data-availability interface using the shared datypes types.
// This mirrors the legacy core/da interface to allow a gradual migration.
type DA interface {
	Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)
	GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error)
	GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error)
	Commit(ctx context.Context, blobs []Blob, namespace []byte) ([]Commitment, error)
	Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte) ([]ID, error)
	SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ([]ID, error)
	Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error)
}

// DummyDA wraps the legacy in-memory DA implementation and satisfies the datypes.DA interface.
type DummyDA struct {
	*coreda.DummyDA
}

// NewDummyDA creates an in-memory DA for tests.
func NewDummyDA(maxBlobSize uint64, blockTime time.Duration) *DummyDA {
	return &DummyDA{DummyDA: coreda.NewDummyDA(maxBlobSize, blockTime)}
}

// GetIDs converts the legacy return type into the shared datypes variant.
func (d *DummyDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error) {
	res, err := d.DummyDA.GetIDs(ctx, height, namespace)
	if res == nil {
		return nil, err
	}
	return &GetIDsResult{IDs: res.IDs, Timestamp: res.Timestamp}, err
}

// WrapCoreDA adapts a core DA implementation to the datypes.DA interface.
func WrapCoreDA(da coreda.DA) DA {
	return coreDAWrapper{da: da}
}

type coreDAWrapper struct {
	da coreda.DA
}

// Ensure wrappers implement DA.
var _ DA = coreDAWrapper{}
var _ DA = (*DummyDA)(nil)

func (w coreDAWrapper) Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error) {
	return w.da.Get(ctx, ids, namespace)
}

func (w coreDAWrapper) GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error) {
	res, err := w.da.GetIDs(ctx, height, namespace)
	if res == nil {
		return nil, err
	}
	return &GetIDsResult{IDs: res.IDs, Timestamp: res.Timestamp}, err
}

func (w coreDAWrapper) GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error) {
	return w.da.GetProofs(ctx, ids, namespace)
}

func (w coreDAWrapper) Commit(ctx context.Context, blobs []Blob, namespace []byte) ([]Commitment, error) {
	return w.da.Commit(ctx, blobs, namespace)
}

func (w coreDAWrapper) Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte) ([]ID, error) {
	return w.da.Submit(ctx, blobs, gasPrice, namespace)
}

func (w coreDAWrapper) SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ([]ID, error) {
	return w.da.SubmitWithOptions(ctx, blobs, gasPrice, namespace, options)
}

func (w coreDAWrapper) Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error) {
	return w.da.Validate(ctx, ids, proofs, namespace)
}
