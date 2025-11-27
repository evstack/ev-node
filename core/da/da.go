package da

import (
	"context"
	"encoding/binary"
	"fmt"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// DA defines very generic interface for interaction with Data Availability layers.
// Deprecated: use blob client directly.
type DA interface {
	Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)
	GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error)
	GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error)
	Commit(ctx context.Context, blobs []Blob, namespace []byte) ([]Commitment, error)
	Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte) ([]ID, error)
	SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ([]ID, error)
	Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error)
}

// Deprecated: use pkg/da/types equivalents.
type (
	Blob           = datypes.Blob
	ID             = datypes.ID
	Commitment     = datypes.Commitment
	Proof          = datypes.Proof
	GetIDsResult   = datypes.GetIDsResult
	ResultSubmit   = datypes.ResultSubmit
	ResultRetrieve = datypes.ResultRetrieve
	StatusCode     = datypes.StatusCode
	BaseResult     = datypes.BaseResult
)

// Deprecated: use pkg/da/types constants.
const (
	StatusUnknown                  = datypes.StatusUnknown
	StatusSuccess                  = datypes.StatusSuccess
	StatusNotFound                 = datypes.StatusNotFound
	StatusNotIncludedInBlock       = datypes.StatusNotIncludedInBlock
	StatusAlreadyInMempool         = datypes.StatusAlreadyInMempool
	StatusTooBig                   = datypes.StatusTooBig
	StatusContextDeadline          = datypes.StatusContextDeadline
	StatusError                    = datypes.StatusError
	StatusIncorrectAccountSequence = datypes.StatusIncorrectAccountSequence
	StatusContextCanceled          = datypes.StatusContextCanceled
	StatusHeightFromFuture         = datypes.StatusHeightFromFuture
)

// makeID creates an ID from a height and a commitment.
func makeID(height uint64, commitment []byte) []byte {
	id := make([]byte, len(commitment)+8)
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], commitment)
	return id
}

// SplitID splits an ID into a height and a commitment.
// if len(id) <= 8, it returns 0 and nil.
func SplitID(id []byte) (uint64, []byte, error) {
	if len(id) <= 8 {
		return 0, nil, fmt.Errorf("invalid ID length: %d", len(id))
	}
	commitment := id[8:]
	return binary.LittleEndian.Uint64(id[:8]), commitment, nil
}
