// Package node provides a DA client that communicates with celestia-node
// via JSON-RPC.
package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/filecoin-project/go-jsonrpc"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// Client dials the celestia-node RPC "blob" and "header" namespaces.
type Client struct {
	Blob   BlobAPI
	Header HeaderAPI
	closer jsonrpc.ClientCloser
}

// Close closes the underlying JSON-RPC connection.
func (c *Client) Close() {
	if c != nil && c.closer != nil {
		c.closer()
	}
}

// Ensure Client implements the datypes.BlobClient interface.
var _ datypes.BlobClient = (*Client)(nil)

// Submit submits blobs to the DA layer via celestia-node.
func (c *Client) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) datypes.ResultSubmit {
	ns, err := libshare.NewNamespaceFromBytes(namespace)
	if err != nil {
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "invalid namespace: " + err.Error(),
			},
		}
	}

	// Build blobs
	blobs := make([]*Blob, len(data))
	for i, raw := range data {
		blob, err := NewBlobV0(ns, raw)
		if err != nil {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusError,
					Message: "failed to build blob: " + err.Error(),
				},
			}
		}
		blobs[i] = blob
	}

	// Parse options
	var opts SubmitOptions
	if len(options) > 0 {
		// Options would be parsed here if needed
	}

	// Submit via node client
	height, err := c.Blob.Submit(ctx, blobs, &opts)
	if err != nil {
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "failed to submit blobs: " + err.Error(),
			},
		}
	}

	// Build IDs
	ids := make([]datypes.ID, len(blobs))
	for i, b := range blobs {
		ids[i] = MakeID(height, b.Commitment)
	}

	return datypes.ResultSubmit{
		BaseResult: datypes.BaseResult{
			Code:           datypes.StatusSuccess,
			IDs:            ids,
			SubmittedCount: uint64(len(ids)),
			Height:         height,
		},
	}
}

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
func (c *Client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	ns, err := libshare.NewNamespaceFromBytes(namespace)
	if err != nil {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "invalid namespace: " + err.Error(),
			},
		}
	}

	blobs, err := c.Blob.GetAll(ctx, height, []libshare.Namespace{ns})
	if err != nil {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: "failed to get blobs: " + err.Error(),
			},
		}
	}

	if len(blobs) == 0 {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusNotFound,
				Message: "no blobs found",
			},
		}
	}

	// Extract IDs and data
	ids := make([]datypes.ID, len(blobs))
	data := make([]datypes.Blob, len(blobs))
	for i, b := range blobs {
		ids[i] = MakeID(height, b.Commitment)
		data[i] = b.Data()
	}

	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:   datypes.StatusSuccess,
			Height: height,
			IDs:    ids,
		},
		Data: data,
	}
}

// Get retrieves blobs by their IDs.
func (c *Client) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	ns, err := libshare.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, err
	}

	result := make([]datypes.Blob, 0, len(ids))
	for _, id := range ids {
		height, commitment := SplitID(id)
		if commitment == nil {
			continue
		}

		blob, err := c.Blob.Get(ctx, height, ns, commitment)
		if err != nil {
			return nil, err
		}
		if blob != nil {
			result = append(result, blob.Data())
		}
	}

	return result, nil
}

// GetLatestDAHeight returns the latest height available on the DA layer.
func (c *Client) GetLatestDAHeight(ctx context.Context) (uint64, error) {
	header, err := c.Header.NetworkHead(ctx)
	if err != nil {
		return 0, err
	}
	return header.Height, nil
}

// GetProofs returns inclusion proofs for the provided IDs.
func (c *Client) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	if len(ids) == 0 {
		return []datypes.Proof{}, nil
	}

	ns, err := libshare.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, err
	}

	proofs := make([]datypes.Proof, len(ids))
	for i, id := range ids {
		height, commitment := SplitID(id)
		if commitment == nil {
			return nil, fmt.Errorf("invalid ID: nil commitment")
		}

		proof, err := c.Blob.GetProof(ctx, height, ns, commitment)
		if err != nil {
			return nil, fmt.Errorf("failed to get proof for height %d: %w", height, err)
		}

		// Serialize proof to JSON
		proofBytes, err := json.Marshal(proof)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal proof: %w", err)
		}
		proofs[i] = proofBytes
	}

	return proofs, nil
}

// Validate validates commitments against the corresponding proofs.
func (c *Client) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	if len(ids) != len(proofs) {
		return nil, fmt.Errorf("ids and proofs length mismatch: %d vs %d", len(ids), len(proofs))
	}

	if len(ids) == 0 {
		return []bool{}, nil
	}

	ns, err := libshare.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, err
	}

	results := make([]bool, len(ids))
	for i, id := range ids {
		height, commitment := SplitID(id)
		if commitment == nil {
			results[i] = false
			continue
		}

		// Deserialize proof from JSON
		var proof Proof
		if err := json.Unmarshal(proofs[i], &proof); err != nil {
			results[i] = false
			continue
		}

		included, err := c.Blob.Included(ctx, height, ns, &proof, commitment)
		if err != nil {
			results[i] = false
		} else {
			results[i] = included
		}
	}

	return results, nil
}

// NewClient connects to the celestia-node RPC endpoint
func NewClient(ctx context.Context, addr, token string, authHeaderName string) (*Client, error) {
	var httpHeader http.Header
	if token != "" {
		if authHeaderName == "" {
			authHeaderName = "Authorization"
		}
		httpHeader = http.Header{authHeaderName: []string{fmt.Sprintf("Bearer %s", token)}}
	}

	var cl Client

	// Connect to the blob namespace
	blobCloser, err := jsonrpc.NewClient(ctx, addr, "blob", &cl.Blob.Internal, httpHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to blob namespace: %w", err)
	}

	// Connect to the header namespace
	headerCloser, err := jsonrpc.NewClient(ctx, addr, "header", &cl.Header.Internal, httpHeader)
	if err != nil {
		blobCloser()
		return nil, fmt.Errorf("failed to connect to header namespace: %w", err)
	}

	// Create a combined closer that closes both connections
	cl.closer = func() {
		blobCloser()
		headerCloser()
	}

	return &cl, nil
}

// BlobAPI mirrors celestia-node's blob module (nodebuilder/blob/blob.go).
// jsonrpc.NewClient wires Internal.* to RPC stubs.
type BlobAPI struct {
	Internal struct {
		Submit func(
			context.Context,
			[]*Blob,
			*SubmitOptions,
		) (uint64, error) `perm:"write"`
		Get func(
			context.Context,
			uint64,
			libshare.Namespace,
			Commitment,
		) (*Blob, error) `perm:"read"`
		GetAll func(
			context.Context,
			uint64,
			[]libshare.Namespace,
		) ([]*Blob, error) `perm:"read"`
		GetProof func(
			context.Context,
			uint64,
			libshare.Namespace,
			Commitment,
		) (*Proof, error) `perm:"read"`
		Included func(
			context.Context,
			uint64,
			libshare.Namespace,
			*Proof,
			Commitment,
		) (bool, error) `perm:"read"`
		GetCommitmentProof func(
			context.Context,
			uint64,
			libshare.Namespace,
			[]byte,
		) (*CommitmentProof, error) `perm:"read"`
		Subscribe func(
			context.Context,
			libshare.Namespace,
		) (<-chan *SubscriptionResponse, error) `perm:"read"`
	}
}

// Submit sends blobs and returns the height they were included at.
func (api *BlobAPI) Submit(ctx context.Context, blobs []*Blob, opts *SubmitOptions) (uint64, error) {
	return api.Internal.Submit(ctx, blobs, opts)
}

// Get retrieves a blob by commitment under the given namespace and height.
func (api *BlobAPI) Get(ctx context.Context, height uint64, namespace libshare.Namespace, commitment Commitment) (*Blob, error) {
	return api.Internal.Get(ctx, height, namespace, commitment)
}

// GetAll returns all blobs for the given namespaces at the given height.
func (api *BlobAPI) GetAll(ctx context.Context, height uint64, namespaces []libshare.Namespace) ([]*Blob, error) {
	return api.Internal.GetAll(ctx, height, namespaces)
}

// GetProof retrieves proofs in the given namespace at the given height by commitment.
func (api *BlobAPI) GetProof(ctx context.Context, height uint64, namespace libshare.Namespace, commitment Commitment) (*Proof, error) {
	return api.Internal.GetProof(ctx, height, namespace, commitment)
}

// Included checks whether a blob commitment is included at the given height/namespace.
func (api *BlobAPI) Included(ctx context.Context, height uint64, namespace libshare.Namespace, proof *Proof, commitment Commitment) (bool, error) {
	return api.Internal.Included(ctx, height, namespace, proof, commitment)
}

// GetCommitmentProof generates a commitment proof for a share commitment.
func (api *BlobAPI) GetCommitmentProof(ctx context.Context, height uint64, namespace libshare.Namespace, shareCommitment []byte) (*CommitmentProof, error) {
	return api.Internal.GetCommitmentProof(ctx, height, namespace, shareCommitment)
}

// Subscribe streams blobs as they are included for the given namespace.
func (api *BlobAPI) Subscribe(ctx context.Context, namespace libshare.Namespace) (<-chan *SubscriptionResponse, error) {
	return api.Internal.Subscribe(ctx, namespace)
}
