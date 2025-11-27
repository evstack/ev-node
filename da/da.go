package da

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DA defines the interface for interaction with Data Availability layers.
type DA interface {
	// Get returns Blob for each given ID, or an error.
	Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)

	// GetIDs returns IDs of all Blobs located in DA at given height.
	GetIDs(ctx context.Context, height uint64, namespace []byte) (*GetIDsResult, error)

	// GetProofs returns inclusion Proofs for Blobs specified by their IDs.
	GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error)

	// Commit creates a Commitment for each given Blob.
	Commit(ctx context.Context, blobs []Blob, namespace []byte) ([]Commitment, error)

	// Submit submits the Blobs to Data Availability layer and returns a structured result.
	Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte) ResultSubmit

	// SubmitWithOptions submits the Blobs to Data Availability layer with additional options.
	SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace []byte, options []byte) ResultSubmit

	// Retrieve retrieves all blobs at the given height and returns a structured result.
	Retrieve(ctx context.Context, height uint64, namespace []byte) ResultRetrieve

	// Validate validates Commitments against the corresponding Proofs.
	Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error)
}

// Blob is the data submitted/received from DA interface.
type Blob = []byte

// ID should contain serialized data required by the implementation to find blob in Data Availability layer.
type ID = []byte

// Commitment should contain serialized cryptographic commitment to Blob value.
type Commitment = []byte

// Proof should contain serialized proof of inclusion (publication) of Blob in Data Availability layer.
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

// ResultRetrieve contains batch of block headers returned from DA layer client.
type ResultRetrieve struct {
	BaseResult
	// Data is the block data retrieved from Data Availability Layer.
	Data [][]byte
}

// StatusCode is a type for DA layer return status.
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

// BaseResult contains basic information returned by DA layer.
type BaseResult struct {
	Code           StatusCode
	Message        string
	Height         uint64
	SubmittedCount uint64
	BlobSize       uint64
	IDs            [][]byte
	Timestamp      time.Time
}

// makeID creates an ID from a height and a commitment.
func makeID(height uint64, commitment []byte) []byte {
	id := make([]byte, len(commitment)+8)
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], commitment)
	return id
}

// SplitID splits an ID into a height and a commitment.
func SplitID(id []byte) (uint64, []byte, error) {
	if len(id) < 8 {
		return 0, nil, fmt.Errorf("invalid ID length: %d", len(id))
	}
	commitment := id[8:]
	return binary.LittleEndian.Uint64(id[:8]), commitment, nil
}

// Errors
var (
	ErrBlobNotFound               = errors.New("blob: not found")
	ErrBlobSizeOverLimit          = errors.New("blob: over size limit")
	ErrTxTimedOut                 = errors.New("timed out waiting for tx to be included in a block")
	ErrTxAlreadyInMempool         = errors.New("tx already in mempool")
	ErrTxIncorrectAccountSequence = errors.New("incorrect account sequence")
	ErrContextDeadline            = errors.New("context deadline")
	ErrHeightFromFuture           = errors.New("given height is from the future")
	ErrContextCanceled            = errors.New("context canceled")
)

// StatusCodeToError converts a StatusCode to its corresponding error.
// Returns nil for StatusSuccess or StatusUnknown.
func StatusCodeToError(code StatusCode, message string) error {
	switch code {
	case StatusSuccess, StatusUnknown:
		return nil
	case StatusNotFound:
		return ErrBlobNotFound
	case StatusNotIncludedInBlock:
		return ErrTxTimedOut
	case StatusAlreadyInMempool:
		return ErrTxAlreadyInMempool
	case StatusTooBig:
		return ErrBlobSizeOverLimit
	case StatusContextDeadline:
		return ErrContextDeadline
	case StatusIncorrectAccountSequence:
		return ErrTxIncorrectAccountSequence
	case StatusContextCanceled:
		return ErrContextCanceled
	case StatusHeightFromFuture:
		return ErrHeightFromFuture
	case StatusError:
		return errors.New(message)
	default:
		return errors.New(message)
	}
}

// Namespace constants and types
const (
	// NamespaceVersionIndex is the index of the namespace version in the byte slice
	NamespaceVersionIndex = 0
	// NamespaceVersionSize is the size of the namespace version in bytes
	NamespaceVersionSize = 1
	// NamespaceIDSize is the size of the namespace ID in bytes
	NamespaceIDSize = 28
	// NamespaceSize is the total size of a namespace (version + ID) in bytes
	NamespaceSize = NamespaceVersionSize + NamespaceIDSize

	// NamespaceVersionZero is the only supported user-specifiable namespace version
	NamespaceVersionZero = uint8(0)
	// NamespaceVersionMax is the max namespace version
	NamespaceVersionMax = uint8(255)

	// NamespaceVersionZeroPrefixSize is the number of leading zero bytes required for version 0
	NamespaceVersionZeroPrefixSize = 18
	// NamespaceVersionZeroDataSize is the number of data bytes available for version 0
	NamespaceVersionZeroDataSize = 10
)

// Namespace represents a Celestia namespace
type Namespace struct {
	Version uint8
	ID      [NamespaceIDSize]byte
}

// Bytes returns the namespace as a byte slice
func (n Namespace) Bytes() []byte {
	result := make([]byte, NamespaceSize)
	result[NamespaceVersionIndex] = n.Version
	copy(result[NamespaceVersionSize:], n.ID[:])
	return result
}

// IsValidForVersion0 checks if the namespace is valid for version 0
func (n Namespace) IsValidForVersion0() bool {
	if n.Version != NamespaceVersionZero {
		return false
	}

	for i := range NamespaceVersionZeroPrefixSize {
		if n.ID[i] != 0 {
			return false
		}
	}
	return true
}

// NewNamespaceV0 creates a new version 0 namespace from the provided data
func NewNamespaceV0(data []byte) (*Namespace, error) {
	if len(data) > NamespaceVersionZeroDataSize {
		return nil, fmt.Errorf("data too long for version 0 namespace: got %d bytes, max %d",
			len(data), NamespaceVersionZeroDataSize)
	}

	ns := &Namespace{
		Version: NamespaceVersionZero,
	}

	copy(ns.ID[NamespaceVersionZeroPrefixSize:], data)
	return ns, nil
}

// NamespaceFromBytes creates a namespace from a 29-byte slice
func NamespaceFromBytes(b []byte) (*Namespace, error) {
	if len(b) != NamespaceSize {
		return nil, fmt.Errorf("invalid namespace size: expected %d, got %d", NamespaceSize, len(b))
	}

	ns := &Namespace{
		Version: b[NamespaceVersionIndex],
	}
	copy(ns.ID[:], b[NamespaceVersionSize:])

	if ns.Version == NamespaceVersionZero && !ns.IsValidForVersion0() {
		return nil, fmt.Errorf("invalid version 0 namespace: first %d bytes of ID must be zero",
			NamespaceVersionZeroPrefixSize)
	}

	return ns, nil
}

// NamespaceFromString creates a version 0 namespace from a string identifier
func NamespaceFromString(s string) *Namespace {
	hash := sha256.Sum256([]byte(s))
	ns, _ := NewNamespaceV0(hash[:NamespaceVersionZeroDataSize])
	return ns
}

// HexString returns the hex representation of the namespace
func (n Namespace) HexString() string {
	return "0x" + hex.EncodeToString(n.Bytes())
}

// ParseHexNamespace parses a hex string into a namespace
func ParseHexNamespace(hexStr string) (*Namespace, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")

	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}

	return NamespaceFromBytes(b)
}
