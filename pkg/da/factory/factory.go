// Package factory provides a factory for creating DA clients based on configuration.
// It automatically detects whether to use celestia-node or celestia-app client
// by making HTTP requests to the endpoint.
package factory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	daapp "github.com/evstack/ev-node/pkg/da/app"
	danode "github.com/evstack/ev-node/pkg/da/node"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// ClientType indicates which type of DA client to use.
type ClientType int

const (
	// ClientTypeAuto automatically detects the client type based on address.
	ClientTypeAuto ClientType = iota
	// ClientTypeNode forces the use of celestia-node client.
	ClientTypeNode
	// ClientTypeApp forces the use of celestia-app client.
	ClientTypeApp
)

// Config contains configuration for creating a DA client.
type Config struct {
	// Address is the DA endpoint address (e.g., "http://localhost:26657" or "http://localhost:26658")
	Address string
	// AuthToken is the authentication token for celestia-node (optional)
	AuthToken string
	// AuthHeaderName is the name of the auth header (optional, defaults to "Authorization")
	AuthHeaderName string
	// Logger for logging
	Logger zerolog.Logger
	// ForceType forces a specific client type (optional, defaults to auto-detection)
	ForceType ClientType
	// IsAggregator indicates if the node is running in aggregator mode.
	// When true, always uses celestia-node client.
	IsAggregator bool
}

// NewClient creates a new DA client based on the configuration.
// It automatically detects whether to use celestia-node or celestia-app client
// by making HTTP requests to the endpoint.
//
// Detection rules:
//   - If IsAggregator is true, always uses celestia-node client
//   - Makes HTTP request to detect service type (celestia-node vs celestia-app)
//   - Falls back to port-based detection if HTTP detection fails
//
// Note: celestia-app client does not support proof operations
// (GetProofs/Validate). Use celestia-node client for full functionality.
func NewClient(ctx context.Context, cfg Config) (datypes.BlobClient, error) {
	// Sanitize address to remove any surrounding quotes that might have been included from config
	cfg.Address = strings.Trim(strings.TrimSpace(cfg.Address), "\"'")

	// Always use node client in aggregator mode
	if cfg.IsAggregator {
		cfg.Logger.Debug().
			Str("address", cfg.Address).
			Msg("aggregator mode enabled, using celestia-node client")
		return createNodeClient(ctx, cfg)
	}

	// Check if type is forced
	switch cfg.ForceType {
	case ClientTypeNode:
		cfg.Logger.Debug().Msg("forced celestia-node client")
		return createNodeClient(ctx, cfg)
	case ClientTypeApp:
		cfg.Logger.Debug().Msg("forced celestia-app client")
		return createAppClient(cfg), nil
	}

	// Auto-detect based on HTTP request
	clientType := detectClientType(ctx, cfg.Address)

	switch clientType {
	case ClientTypeApp:
		cfg.Logger.Debug().
			Str("address", cfg.Address).
			Msg("auto-detected celestia-app client")
		return createAppClient(cfg), nil
	case ClientTypeNode:
		cfg.Logger.Debug().
			Str("address", cfg.Address).
			Msg("auto-detected celestia-node client")
		return createNodeClient(ctx, cfg)
	default:
		return nil, fmt.Errorf("unable to detect client type for address: %s", cfg.Address)
	}
}

// detectClientType determines which client to use by making HTTP requests to detect the service type.
func detectClientType(ctx context.Context, address string) ClientType {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}

	// Try celestia-node detection first (JSON-RPC over HTTP)
	// node.Ready is a lightweight endpoint that requires only "read" permission
	if isNodeEndpoint(ctx, address) {
		return ClientTypeNode
	}

	// Try celestia-app detection (CometBFT RPC)
	// /status endpoint returns node information
	if isAppEndpoint(ctx, address) {
		return ClientTypeApp
	}

	// Default to node client if detection fails
	return ClientTypeNode
}

// isNodeEndpoint checks if the address is a celestia-node endpoint
// by calling the node.Ready JSON-RPC method.
// It returns true only if the method returns a successful result (not an error).
func isNodeEndpoint(ctx context.Context, address string) bool {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "node.Ready",
		"params":  []interface{}{},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", address, bytes.NewReader(reqBytes))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// Check if response contains a successful JSON-RPC result
	// We require a successful result (no error) to confirm it's celestia-node
	var rpcResp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return false
	}

	// Must be a valid JSON-RPC 2.0 response with no error and a result
	if rpcResp.JSONRPC == "2.0" && rpcResp.Error == nil && rpcResp.Result != nil {
		return true
	}

	return false
}

// isAppEndpoint checks if the address is a celestia-app endpoint
// by calling the /status CometBFT RPC endpoint.
func isAppEndpoint(ctx context.Context, address string) bool {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "status",
		"params":  []interface{}{},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", address, bytes.NewReader(reqBytes))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// Check if response contains CometBFT status fields
	var rpcResp struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			NodeInfo struct {
				Network string `json:"network"`
			} `json:"node_info"`
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return false
	}

	// If we got a valid response with node_info, it's celestia-app
	if rpcResp.JSONRPC == "2.0" && rpcResp.Result.NodeInfo.Network != "" {
		return true
	}

	return false
}

// createNodeClient creates a celestia-node client.
func createNodeClient(ctx context.Context, cfg Config) (datypes.BlobClient, error) {
	client, err := danode.NewClient(ctx, cfg.Address, cfg.AuthToken, cfg.AuthHeaderName)
	if err != nil {
		return nil, fmt.Errorf("failed to create celestia-node client: %w", err)
	}
	return client, nil
}

// createAppClient creates a celestia-app client.
func createAppClient(cfg Config) datypes.BlobClient {
	return daapp.NewClient(daapp.Config{
		RPCAddress:     cfg.Address,
		Logger:         cfg.Logger,
		DefaultTimeout: 0, // Use default
	})
}

// IsNodeAddress checks if the given address is a celestia-node endpoint.
// It makes an HTTP request to detect the service type.
func IsNodeAddress(ctx context.Context, address string) bool {
	return detectClientType(ctx, address) == ClientTypeNode
}

// IsAppAddress checks if the given address is a celestia-app endpoint.
// It makes an HTTP request to detect the service type.
func IsAppAddress(ctx context.Context, address string) bool {
	return detectClientType(ctx, address) == ClientTypeApp
}
