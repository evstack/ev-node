package jsonrpc

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"
)

// Client dials the celestia-node RPC "blob" and "header" namespaces.
type Client struct {
	Blob        BlobAPI
	Header      HeaderAPI
	IsWebSocket atomic.Bool

	mu          sync.Mutex
	closer      jsonrpc.ClientCloser
	retryCancel context.CancelFunc // stops the background WS retry loop
}

// Close closes the underlying JSON-RPC connection and stops any
// background WebSocket retry loop.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.retryCancel != nil {
		c.retryCancel()
		c.retryCancel = nil
	}
	closer := c.closer
	c.mu.Unlock()
	if closer != nil {
		closer()
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
// WebSocket connections are eager — they connect at creation time.
// If the initial WS dial fails, it falls back to HTTP polling and spawns a
// background goroutine that periodically retries the WS connection. When
// the WS endpoint becomes reachable, the transport is transparently upgraded.
func NewWSClient(ctx context.Context, logger zerolog.Logger, addr, token string, authHeaderName string) (*Client, error) {
	client, err := NewClient(ctx, httpToWS(addr), token, authHeaderName)
	if err != nil {
		logger.Warn().Err(err).Msg("DA websocket connection failed, falling back to DA polling")
		client, err = NewClient(ctx, addr, token, authHeaderName)
		if err != nil {
			return nil, err
		}
		client.IsWebSocket.Store(false)

		// Retry WS in the background so transient outages don't force a permanent downgrade.
		retryCtx, retryCancel := context.WithCancel(context.Background())
		client.retryCancel = retryCancel
		go client.retryWSLoop(retryCtx, logger, addr, token, authHeaderName)

		return client, nil
	}

	client.IsWebSocket.Store(true)
	return client, nil
}

const wsRetryInterval = 30 * time.Second

// retryWSLoop periodically attempts to re-establish a WebSocket connection.
// When successful, it swaps the transport in-place and exits.
func (c *Client) retryWSLoop(ctx context.Context, logger zerolog.Logger, addr, token, authHeaderName string) {
	ticker := time.NewTicker(wsRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.tryUpgradeWS(ctx, logger, addr, token, authHeaderName) {
				return
			}
		}
	}
}

// tryUpgradeWS attempts to open a WS connection and, if successful, swaps
// the transport internals so subsequent calls use WebSocket. Returns true
// when the upgrade succeeds (or the client is already on WS).
func (c *Client) tryUpgradeWS(ctx context.Context, logger zerolog.Logger, addr, token, authHeaderName string) bool {
	wsClient, err := NewClient(ctx, httpToWS(addr), token, authHeaderName)
	if err != nil {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Another goroutine may have already upgraded.
	if c.IsWebSocket.Load() {
		wsClient.Close()
		return true
	}

	// Swap function pointers from the new WS client into the active client.
	c.Blob.Internal = wsClient.Blob.Internal
	c.Header.Internal = wsClient.Header.Internal

	// Close the old HTTP connections and wire the new closer.
	oldCloser := c.closer
	c.closer = func() {
		wsClient.closer()
		if oldCloser != nil {
			oldCloser()
		}
	}

	c.IsWebSocket.Store(true)
	logger.Info().Msg("DA websocket connection restored, switching back from HTTP polling")
	return true
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
