package celestia

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
)

// Client dials the celestia-node RPC "blob" namespace.
type Client struct {
	Blob   BlobAPI
	closer jsonrpc.ClientCloser
}

// Close closes the underlying JSON-RPC connection.
func (c *Client) Close() {
	if c != nil && c.closer != nil {
		c.closer()
	}
}

// NewClient connects to the celestia-node RPC endpoint (namespace "blob" by default).
// addr should include scheme, e.g. http://127.0.0.1:26658.
// token, if non-empty, is sent as Bearer token using perms.AuthKey header.
// For future additional RPC namespaces, we would need one jsonrpc.Client per namespace
// (mirroring celestia-node's api/rpc/client.moduleMap). Here we only wire "blob".
func NewClient(ctx context.Context, addr, token string, authHeaderName string) (*Client, error) {
	var header http.Header
	if token != "" {
		if authHeaderName == "" {
			authHeaderName = "Authorization"
		}
		header = http.Header{authHeaderName: []string{fmt.Sprintf("Bearer %s", token)}}
	}

	var cl Client
	closer, err := jsonrpc.NewClient(ctx, addr, "blob", &cl.Blob.Internal, header)
	if err != nil {
		return nil, err
	}
	cl.closer = closer
	return &cl, nil
}
