package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
	rpc "github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

// HealthStatus represents the health status of a node
type HealthStatus int32

const (
	// HealthStatus_UNKNOWN represents an unknown health status
	HealthStatus_UNKNOWN HealthStatus = 0
	// HealthStatus_PASS represents a healthy node
	HealthStatus_PASS HealthStatus = 1
	// HealthStatus_WARN represents a degraded but still serving node
	HealthStatus_WARN HealthStatus = 2
	// HealthStatus_FAIL represents a failed node
	HealthStatus_FAIL HealthStatus = 3
)

func (h HealthStatus) String() string {
	switch h {
	case HealthStatus_PASS:
		return "PASS"
	case HealthStatus_WARN:
		return "WARN"
	case HealthStatus_FAIL:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

// Client is the client for StoreService, P2PService, and ConfigService
type Client struct {
	storeClient  rpc.StoreServiceClient
	p2pClient    rpc.P2PServiceClient
	configClient rpc.ConfigServiceClient
	baseURL      string
	httpClient   *http.Client
}

// NewClient creates a new RPC client
func NewClient(baseURL string) *Client {
	httpClient := http.DefaultClient
	storeClient := rpc.NewStoreServiceClient(httpClient, baseURL, connect.WithGRPC())
	p2pClient := rpc.NewP2PServiceClient(httpClient, baseURL, connect.WithGRPC())
	configClient := rpc.NewConfigServiceClient(httpClient, baseURL, connect.WithGRPC())

	return &Client{
		storeClient:  storeClient,
		p2pClient:    p2pClient,
		configClient: configClient,
		baseURL:      baseURL,
		httpClient:   httpClient,
	}
}

// GetBlockByHeight returns the full GetBlockResponse for a block by height
func (c *Client) GetBlockByHeight(ctx context.Context, height uint64) (*pb.GetBlockResponse, error) {
	req := connect.NewRequest(&pb.GetBlockRequest{
		Identifier: &pb.GetBlockRequest_Height{
			Height: height,
		},
	})

	resp, err := c.storeClient.GetBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
}

// GetBlockByHash returns the full GetBlockResponse for a block by hash
func (c *Client) GetBlockByHash(ctx context.Context, hash []byte) (*pb.GetBlockResponse, error) {
	req := connect.NewRequest(&pb.GetBlockRequest{
		Identifier: &pb.GetBlockRequest_Hash{
			Hash: hash,
		},
	})

	resp, err := c.storeClient.GetBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg, nil
}

// GetState returns the current state
func (c *Client) GetState(ctx context.Context) (*pb.State, error) {
	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := c.storeClient.GetState(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg.State, nil
}

// GetMetadata returns metadata for a specific key
func (c *Client) GetMetadata(ctx context.Context, key string) ([]byte, error) {
	req := connect.NewRequest(&pb.GetMetadataRequest{
		Key: key,
	})

	resp, err := c.storeClient.GetMetadata(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg.Value, nil
}

// GetPeerInfo returns information about the connected peers
func (c *Client) GetPeerInfo(ctx context.Context) ([]*pb.PeerInfo, error) {
	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := c.p2pClient.GetPeerInfo(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg.Peers, nil
}

// GetNetInfo returns information about the network
func (c *Client) GetNetInfo(ctx context.Context) (*pb.NetInfo, error) {
	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := c.p2pClient.GetNetInfo(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Msg.NetInfo, nil
}

// GetHealth calls the /health/live HTTP endpoint and returns the HealthStatus
func (c *Client) GetHealth(ctx context.Context) (HealthStatus, error) {
	healthURL := fmt.Sprintf("%s/health/live", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return HealthStatus_UNKNOWN, fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return HealthStatus_UNKNOWN, fmt.Errorf("failed to get health: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HealthStatus_UNKNOWN, fmt.Errorf("failed to read health response: %w", err)
	}

	// Parse the text response
	status := strings.TrimSpace(string(body))
	switch status {
	case "OK":
		return HealthStatus_PASS, nil
	case "WARN":
		return HealthStatus_WARN, nil
	case "FAIL":
		return HealthStatus_FAIL, nil
	default:
		return HealthStatus_UNKNOWN, fmt.Errorf("unknown health status: %s", status)
	}
}

// GetNamespace returns the namespace configuration for this network
func (c *Client) GetNamespace(ctx context.Context) (*pb.GetNamespaceResponse, error) {
	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := c.configClient.GetNamespace(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
