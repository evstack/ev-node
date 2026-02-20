// Package factory provides a factory for creating DA clients based on configuration.
// It automatically detects whether to use celestia-node or celestia-app client
// based on the address and aggregator mode.
package factory

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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
// based on the address port and aggregator mode.
//
// Detection rules:
//   - If IsAggregator is true, always uses celestia-node client
//   - Port 26657 (or any port) -> celestia-app client (CometBFT RPC)
//   - Port 26658 -> celestia-node client (blob module)
//   - Any other port -> celestia-node client (assumed to be custom node endpoint)
//
// Note: celestia-app client (port 26657) does not support proof operations
// (GetProofs/Validate). Use celestia-node client for full functionality.
func NewClient(ctx context.Context, cfg Config) (datypes.BlobClient, error) {
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

	// Auto-detect based on address
	clientType := detectClientType(cfg.Address)

	switch clientType {
	case ClientTypeApp:
		cfg.Logger.Debug().
			Str("address", cfg.Address).
			Msg("auto-detected celestia-app client (port 26657)")
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

// detectClientType determines which client to use based on the address.
// Port 26657 is celestia-app (CometBFT RPC).
// Port 26658 or any other port is treated as celestia-node.
func detectClientType(address string) ClientType {
	// Parse the URL to extract port
	u, err := url.Parse(address)
	if err != nil {
		// If parsing fails, try adding scheme
		u, err = url.Parse("http://" + address)
		if err != nil {
			// Can't parse, default to node client
			return ClientTypeNode
		}
	}

	// If the "scheme" looks like a hostname (localhost or contains digits),
	// then url.Parse misinterpreted the address and we need to re-parse with a scheme.
	if u.Host == "" && (u.Scheme == "localhost" || containsDigits(u.Scheme)) {
		u, _ = url.Parse("http://" + address)
	}

	// Extract port
	port := u.Port()
	if port == "" {
		// No port specified, check scheme
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			// Default to node client
			return ClientTypeNode
		}
	}

	// Port 26657 is celestia-app CometBFT RPC
	if port == "26657" {
		return ClientTypeApp
	}

	// All other ports (including 26658) are treated as celestia-node
	return ClientTypeNode
}

// containsDigits checks if a string contains any digits.
func containsDigits(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
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

// IsNodeAddress checks if the given address is likely a celestia-node endpoint.
// Returns true for any address that is NOT port 26657.
func IsNodeAddress(address string) bool {
	return detectClientType(address) == ClientTypeNode
}

// IsAppAddress checks if the given address is likely a celestia-app endpoint.
// Returns true only for port 26657.
func IsAppAddress(address string) bool {
	return detectClientType(address) == ClientTypeApp
}

// ValidateAddress checks if the address is valid and returns the detected client type.
func ValidateAddress(address string) (ClientType, error) {
	if strings.TrimSpace(address) == "" {
		return ClientTypeAuto, fmt.Errorf("DA address cannot be empty")
	}

	// Try to parse as URL
	u, err := url.Parse(address)
	if err != nil {
		// Try with http:// prefix
		u, err = url.Parse("http://" + address)
		if err != nil {
			return ClientTypeAuto, fmt.Errorf("invalid DA address format: %w", err)
		}
	}

	if u.Host == "" {
		return ClientTypeAuto, fmt.Errorf("DA address must include host")
	}

	return detectClientType(address), nil
}
