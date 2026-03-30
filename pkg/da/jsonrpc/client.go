package jsonrpc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"
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

// httpToWS converts an HTTP(S) URL to a WebSocket URL.
func httpToWS(addr string) string {
	addr = strings.Replace(addr, "https://", "wss://", 1)
	addr = strings.Replace(addr, "http://", "ws://", 1)
	return addr
}

// NewClient connects to the DA RPC endpoint using the address as-is.
// Uses HTTP by default (lazy connection — only connects on first RPC call).
// Does NOT support channel-based subscriptions (e.g. Subscribe).
// For subscription support, use NewWSClient instead.
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

// NewWSClient connects to the DA RPC endpoint over WebSocket.
// Automatically converts http:// to ws:// (and https:// to wss://).
// Supports channel-based subscriptions (e.g. Subscribe).
// Note: WebSocket connections are eager — they connect at creation time
// if it fails, we fallback to non websocket connection for the whole runtime process.
func NewWSClient(ctx context.Context, logger zerolog.Logger, addr, token string, authHeaderName string) (*Client, error) {
	client, err := NewClient(ctx, httpToWS(addr), token, authHeaderName)
	if err != nil {
		logger.Warn().Err(err).Msg("DA websocket connection failed, falling back to DA polling")
		return NewClient(ctx, addr, token, authHeaderName)
	}

	return client, nil
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
