package blob

// NOTE: This file is a trimmed copy of celestia-node's blob/blob.go
// at release v0.28.4 (commit tag v0.28.4). We keep only the JSON-
// compatible surface used by ev-node to avoid pulling celestia-app /
// Cosmos-SDK dependencies. See pkg/da/blob/README.md for update guidance.

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/celestiaorg/go-square/merkle"
	"github.com/celestiaorg/go-square/v3/inclusion"
	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/celestiaorg/nmt"
)

// Commitment is the Merkle subtree commitment for a blob.
type Commitment []byte

// Proof is a set of NMT proofs used to verify a blob inclusion.
// This mirrors celestia-node's blob.Proof shape.
type Proof []*nmt.Proof

// DefaultMaxBlobSize is the default maximum blob size used by celestia-app (32 MiB).
const DefaultMaxBlobSize = 32 * 1_048_576 // bytes

// subtreeRootThreshold is copied from celestia-app/v6 appconsts.SubtreeRootThreshold.
// It controls the branching factor when generating commitments.
const subtreeRootThreshold = 64

// Blob represents application-specific binary data that can be submitted to Celestia.
// It is intentionally compatible with celestia-node's blob.Blob JSON shape.
type Blob struct {
	*libshare.Blob `json:"blob"`

	Commitment Commitment `json:"commitment"`

	// index is the index of the blob's first share in the EDS.
	// Only blobs retrieved from the chain will have this set; default is -1.
	index int
}

// NewBlobV0 builds a version 0 blob (the only version we currently need).
func NewBlobV0(namespace libshare.Namespace, data []byte) (*Blob, error) {
	return NewBlob(libshare.ShareVersionZero, namespace, data, nil)
}

// NewBlob constructs a new blob from the provided namespace, data, signer, and share version.
// This is a lightly adapted copy of celestia-node/blob.NewBlob.
func NewBlob(shareVersion uint8, namespace libshare.Namespace, data, signer []byte) (*Blob, error) {
	if err := namespace.ValidateForBlob(); err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	libBlob, err := libshare.NewBlob(namespace, data, shareVersion, signer)
	if err != nil {
		return nil, fmt.Errorf("build blob: %w", err)
	}

	com, err := inclusion.CreateCommitment(libBlob, merkle.HashFromByteSlices, subtreeRootThreshold)
	if err != nil {
		return nil, fmt.Errorf("create commitment: %w", err)
	}

	return &Blob{
		Blob:       libBlob,
		Commitment: com,
		index:      -1,
	}, nil
}

// Namespace returns the blob namespace.
func (b *Blob) Namespace() libshare.Namespace {
	return b.Blob.Namespace()
}

// Index returns the blob's first share index in the EDS (or -1 if unknown).
func (b *Blob) Index() int {
	return b.index
}

// MarshalJSON matches celestia-node's blob JSON encoding.
func (b *Blob) MarshalJSON() ([]byte, error) {
	type jsonBlob struct {
		Namespace    []byte     `json:"namespace"`
		Data         []byte     `json:"data"`
		ShareVersion uint8      `json:"share_version"`
		Commitment   Commitment `json:"commitment"`
		Signer       []byte     `json:"signer,omitempty"`
		Index        int        `json:"index"`
	}

	jb := &jsonBlob{
		Namespace:    b.Namespace().Bytes(),
		Data:         b.Data(),
		ShareVersion: b.ShareVersion(),
		Commitment:   b.Commitment,
		Signer:       b.Signer(),
		Index:        b.index,
	}
	return json.Marshal(jb)
}

// UnmarshalJSON matches celestia-node's blob JSON decoding.
func (b *Blob) UnmarshalJSON(data []byte) error {
	type jsonBlob struct {
		Namespace    []byte     `json:"namespace"`
		Data         []byte     `json:"data"`
		ShareVersion uint8      `json:"share_version"`
		Commitment   Commitment `json:"commitment"`
		Signer       []byte     `json:"signer,omitempty"`
		Index        int        `json:"index"`
	}

	var jb jsonBlob
	if err := json.Unmarshal(data, &jb); err != nil {
		return err
	}

	ns, err := libshare.NewNamespaceFromBytes(jb.Namespace)
	if err != nil {
		return err
	}

	blob, err := NewBlob(jb.ShareVersion, ns, jb.Data, jb.Signer)
	if err != nil {
		return err
	}

	blob.Commitment = jb.Commitment
	blob.index = jb.Index
	*b = *blob
	return nil
}

// MakeID constructs a blob ID by prefixing the commitment with the height (little endian).
func MakeID(height uint64, commitment Commitment) []byte {
	id := make([]byte, 8+len(commitment))
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], commitment)
	return id
}

// SplitID splits a blob ID into height and commitment.
// If the ID is malformed, it returns height 0 and nil commitment.
func SplitID(id []byte) (uint64, Commitment) {
	if len(id) <= 8 {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(id[:8]), id[8:]
}

// EqualCommitment compares the blob's commitment with the provided one.
func (b *Blob) EqualCommitment(com Commitment) bool {
	return bytes.Equal(b.Commitment, com)
}
