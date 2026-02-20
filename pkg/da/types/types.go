package da

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"
)

// BlobClient defines the interface for DA layer operations.
// This is the shared interface that both node (celestia-node) and app (celestia-app) clients implement.
type BlobClient interface {
	// Submit submits blobs to the DA layer.
	Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) ResultSubmit

	// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
	Retrieve(ctx context.Context, height uint64, namespace []byte) ResultRetrieve

	// Get retrieves blobs by their IDs.
	Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)

	// GetLatestDAHeight returns the latest height available on the DA layer.
	GetLatestDAHeight(ctx context.Context) (uint64, error)

	// GetProofs returns inclusion proofs for the provided IDs.
	GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error)

	// Validate validates commitments against the corresponding proofs.
	Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error)
}

// StatusCode mirrors the blob RPC status codes shared with block/internal/da.
type StatusCode uint64

// Data Availability return codes.
const (
	StatusUnknown StatusCode = iota
	StatusSuccess
	StatusNotFound
	StatusNotIncludedInBlock
	StatusAlreadyInMempool
	StatusTooBig
	StatusContextDeadline
	StatusError
	StatusIncorrectAccountSequence
	StatusContextCanceled
	StatusHeightFromFuture
)

// Blob is the data submitted/received from the DA layer.
type Blob = []byte

// ID should contain serialized data required by the implementation to find blob in DA.
type ID = []byte

// Commitment should contain serialized cryptographic commitment to Blob value.
type Commitment = []byte

// Proof should contain serialized proof of inclusion (publication) of Blob in DA.
type Proof = []byte

// GetIDsResult holds the result of GetIDs call: IDs and timestamp of corresponding block.
type GetIDsResult struct {
	IDs       []ID
	Timestamp time.Time
}

// ResultSubmit contains information returned from DA layer after block headers/data submission.
type ResultSubmit struct {
	BaseResult
}

// ResultRetrieve contains batch of block data returned from DA layer client.
type ResultRetrieve struct {
	BaseResult
	// Data is the block data retrieved from Data Availability Layer.
	// If Code is not equal to StatusSuccess, it has to be nil.
	Data [][]byte
}

// BaseResult contains basic information returned by DA layer.
type BaseResult struct {
	// Code is to determine if the action succeeded.
	Code StatusCode
	// Message may contain DA layer specific information (like DA block height/hash, detailed error message, etc)
	Message string
	// Height is the height of the block on Data Availability Layer for given result.
	Height uint64
	// SubmittedCount is the number of successfully submitted blocks.
	SubmittedCount uint64
	// BlobSize is the size of the blob submitted.
	BlobSize uint64
	// IDs is the list of IDs of the blobs submitted.
	IDs [][]byte
	// Timestamp is the timestamp of the posted data on Data Availability Layer.
	Timestamp time.Time
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
