package jsonrpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/blob"
)

// API exposes the blob RPC methods used by the node.
type API struct {
	Logger      zerolog.Logger
	MaxBlobSize uint64
	Internal    struct {
		Submit   func(context.Context, []*blob.Blob, *blob.SubmitOptions) (uint64, error)                   `perm:"write"`
		GetAll   func(context.Context, uint64, []share.Namespace) ([]*blob.Blob, error)                     `perm:"read"`
		GetProof func(context.Context, uint64, share.Namespace, blob.Commitment) (*blob.Proof, error)       `perm:"read"`
		Included func(context.Context, uint64, share.Namespace, *blob.Proof, blob.Commitment) (bool, error) `perm:"read"`
	}
}

// Client is the jsonrpc client for the blob namespace implementing block/internal/da.BlobAPI.
type Client struct {
	API    API
	closer multiClientCloser
}

// BlobAPI exposes the methods needed by block/internal/da.
func (c *Client) Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error) {
	if c.API.MaxBlobSize > 0 {
		for i, b := range blobs {
			if b == nil {
				return 0, fmt.Errorf("blob %d is nil", i)
			}
			if uint64(len(b.Data())) > c.API.MaxBlobSize {
				c.API.Logger.Warn().
					Int("index", i).
					Int("size", len(b.Data())).
					Uint64("max", c.API.MaxBlobSize).
					Msg("blob rejected: size over limit")
				return 0, fmt.Errorf("blob %d exceeds max blob size %d bytes", i, c.API.MaxBlobSize)
			}
		}
	}
	return c.API.Internal.Submit(ctx, blobs, opts)
}

func (c *Client) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	return c.API.Internal.GetAll(ctx, height, namespaces)
}

func (c *Client) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	return c.API.Internal.GetProof(ctx, height, namespace, commitment)
}

func (c *Client) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	return c.API.Internal.Included(ctx, height, namespace, proof, commitment)
}

// multiClientCloser is a wrapper struct to close clients across multiple namespaces.
type multiClientCloser struct {
	closers []jsonrpc.ClientCloser
}

// register adds a new closer to the multiClientCloser
func (m *multiClientCloser) register(closer jsonrpc.ClientCloser) {
	m.closers = append(m.closers, closer)
}

// closeAll closes all saved clients.
func (m *multiClientCloser) closeAll() {
	for _, closer := range m.closers {
		closer()
	}
}

// Close closes the connections to all namespaces registered on the staticClient.
func (c *Client) Close() {
	c.closer.closeAll()
}

// NewClient creates a new Client with one connection to the blob namespace.
func NewClient(ctx context.Context, logger zerolog.Logger, addr, token string, maxBlobSize uint64) (*Client, error) {
	authHeader := http.Header{}
	if token != "" {
		authHeader.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return newClient(ctx, logger, addr, authHeader, maxBlobSize)
}

func newClient(ctx context.Context, logger zerolog.Logger, addr string, authHeader http.Header, maxBlobSize uint64) (*Client, error) {
	var multiCloser multiClientCloser
	var client Client
	client.API.Logger = logger
	client.API.MaxBlobSize = maxBlobSize

	for name, module := range moduleMap(&client) {
		closer, err := jsonrpc.NewMergeClient(ctx, addr, name, []interface{}{module}, authHeader)
		if err != nil {
			multiCloser.closeAll()
			return nil, err
		}
		multiCloser.register(closer)
	}

	client.closer = multiCloser
	return &client, nil
}

func moduleMap(client *Client) map[string]interface{} {
	return map[string]interface{}{
		"blob": &client.API.Internal,
	}
}
