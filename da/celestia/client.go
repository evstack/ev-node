package celestia

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"
)

// Client connects to celestia-node's blob API via JSON-RPC.
type Client struct {
	logger      zerolog.Logger
	maxBlobSize uint64
	closer      jsonrpc.ClientCloser

	Internal struct {
		Submit   func(ctx context.Context, blobs []*Blob, opts *SubmitOptions) (uint64, error)       `perm:"write"`
		Get      func(ctx context.Context, height uint64, ns Namespace, c Commitment) (*Blob, error) `perm:"read"`
		GetAll   func(ctx context.Context, height uint64, namespaces []Namespace) ([]*Blob, error)   `perm:"read"`
		GetProof func(ctx context.Context, height uint64, ns Namespace, c Commitment) (*Proof, error) `perm:"read"`
		Included func(ctx context.Context, height uint64, ns Namespace, proof *Proof, c Commitment) (bool, error) `perm:"read"`
	}
}

// NewClient creates a new client connected to celestia-node.
// Token is obtained from: celestia light auth write
func NewClient(
	ctx context.Context,
	logger zerolog.Logger,
	addr string,
	token string,
	maxBlobSize uint64,
) (*Client, error) {
	if addr == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	if maxBlobSize == 0 {
		return nil, fmt.Errorf("maxBlobSize must be greater than 0")
	}

	client := &Client{
		logger:      logger,
		maxBlobSize: maxBlobSize,
	}

	authHeader := http.Header{}
	if token != "" {
		authHeader.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	closer, err := jsonrpc.NewMergeClient(
		ctx,
		addr,
		"blob",
		[]interface{}{&client.Internal},
		authHeader,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON-RPC client: %w", err)
	}

	client.closer = closer

	logger.Info().
		Str("address", addr).
		Uint64("max_blob_size", maxBlobSize).
		Msg("Celestia blob API client created successfully")

	return client, nil
}

// Close closes the connection. Safe to call multiple times.
func (c *Client) Close() {
	if c.closer != nil {
		c.closer()
		c.closer = nil
	}
	c.logger.Debug().Msg("Celestia client connection closed")
}

// Submit submits blobs to Celestia and returns the height at which they were included.
func (c *Client) Submit(ctx context.Context, blobs []*Blob, opts *SubmitOptions) (uint64, error) {
	c.logger.Debug().
		Int("num_blobs", len(blobs)).
		Msg("Submitting blobs to Celestia")

	height, err := c.Internal.Submit(ctx, blobs, opts)
	if err != nil {
		c.logger.Error().
			Err(err).
			Int("num_blobs", len(blobs)).
			Msg("Failed to submit blobs")
		return 0, fmt.Errorf("failed to submit blobs: %w", err)
	}

	c.logger.Info().
		Uint64("height", height).
		Int("num_blobs", len(blobs)).
		Msg("Successfully submitted blobs")

	return height, nil
}

// Get retrieves a single blob by commitment at a given height and namespace.
func (c *Client) Get(ctx context.Context, height uint64, namespace Namespace, commitment Commitment) (*Blob, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// GetAll retrieves all blobs at a given height for the specified namespaces.
func (c *Client) GetAll(ctx context.Context, height uint64, namespaces []Namespace) ([]*Blob, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// GetProof retrieves the inclusion proof for a blob.
func (c *Client) GetProof(ctx context.Context, height uint64, namespace Namespace, commitment Commitment) (*Proof, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// Included checks whether a blob is included in the Celestia block.
func (c *Client) Included(ctx context.Context, height uint64, namespace Namespace, proof *Proof, commitment Commitment) (bool, error) {
	return false, fmt.Errorf("not implemented yet")
}
