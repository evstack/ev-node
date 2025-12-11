package da

import "time"

// StatusCode mirrors DA status codes used in Celestia blob client.
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
