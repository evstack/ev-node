package celestia

import (
	"fmt"

	"github.com/celestiaorg/go-square/merkle"
	"github.com/celestiaorg/go-square/v3/inclusion"
	libshare "github.com/celestiaorg/go-square/v3/share"
)

// subtreeRootThreshold matches the value used by celestia-app.
// This determines the size of subtrees when computing blob commitments.
const subtreeRootThreshold = 64

// CreateCommitment computes the commitment for a blob.
// The commitment is computed using the same algorithm as celestia-node:
// 1. Split the blob data into shares
// 2. Build a Merkle tree over the shares
// 3. Return the Merkle root
func CreateCommitment(data []byte, namespace []byte) (Commitment, error) {
	// Create namespace from bytes
	ns, err := libshare.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create a blob with share version 0 (default)
	blob, err := libshare.NewBlob(ns, data, libshare.ShareVersionZero, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob: %w", err)
	}

	// Compute commitment using the same function as celestia-node
	commitment, err := inclusion.CreateCommitment(blob, merkle.HashFromByteSlices, subtreeRootThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to create commitment: %w", err)
	}

	return commitment, nil
}

// CreateCommitments computes commitments for multiple blobs.
func CreateCommitments(data [][]byte, namespace []byte) ([]Commitment, error) {
	commitments := make([]Commitment, len(data))
	for i, d := range data {
		commitment, err := CreateCommitment(d, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create commitment for blob %d: %w", i, err)
		}
		commitments[i] = commitment
	}
	return commitments, nil
}
