package newjsonrpc

import "github.com/evstack/ev-node/pkg/blob"

// CommitmentProof matches celestia-node's blob.CommitmentProof JSON shape.
// We keep only the fields we need on the client side.
type CommitmentProof struct {
	SubtreeRoots [][]byte `json:"subtree_roots,omitempty"`
}

// SubscriptionResponse mirrors celestia-node's blob.SubscriptionResponse.
type SubscriptionResponse struct {
	Blobs  []*blob.Blob `json:"blobs"`
	Height uint64       `json:"height"`
}
