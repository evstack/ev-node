package factory

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

func TestDetectClientType(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected ClientType
	}{
		{
			name:     "celestia-app default port",
			address:  "http://localhost:26657",
			expected: ClientTypeApp,
		},
		{
			name:     "celestia-app with IP",
			address:  "http://127.0.0.1:26657",
			expected: ClientTypeApp,
		},
		{
			name:     "celestia-node default port",
			address:  "http://localhost:26658",
			expected: ClientTypeNode,
		},
		{
			name:     "celestia-node with IP",
			address:  "http://127.0.0.1:26658",
			expected: ClientTypeNode,
		},
		{
			name:     "custom port",
			address:  "http://localhost:8080",
			expected: ClientTypeNode,
		},
		{
			name:     "https with port 26657",
			address:  "https://example.com:26657",
			expected: ClientTypeApp,
		},
		{
			name:     "https with custom port",
			address:  "https://example.com:443",
			expected: ClientTypeNode,
		},
		{
			name:     "http default port",
			address:  "http://localhost",
			expected: ClientTypeNode,
		},
		{
			name:     "https default port",
			address:  "https://localhost",
			expected: ClientTypeNode,
		},
		{
			name:     "without scheme port 26657",
			address:  "localhost:26657",
			expected: ClientTypeApp,
		},
		{
			name:     "without scheme port 26658",
			address:  "localhost:26658",
			expected: ClientTypeNode,
		},
		{
			name:     "invalid URL defaults to node",
			address:  "://invalid",
			expected: ClientTypeNode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectClientType(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNodeAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected bool
	}{
		{"port 26657", "http://localhost:26657", false},
		{"port 26658", "http://localhost:26658", true},
		{"custom port", "http://localhost:8080", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNodeAddress(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAppAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected bool
	}{
		{"port 26657", "http://localhost:26657", true},
		{"port 26658", "http://localhost:26658", false},
		{"custom port", "http://localhost:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAppAddress(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expectError bool
		expectType  ClientType
	}{
		{
			name:        "valid node address",
			address:     "http://localhost:26658",
			expectError: false,
			expectType:  ClientTypeNode,
		},
		{
			name:        "valid app address",
			address:     "http://localhost:26657",
			expectError: false,
			expectType:  ClientTypeApp,
		},
		{
			name:        "empty address",
			address:     "",
			expectError: true,
		},
		{
			name:        "whitespace address",
			address:     "   ",
			expectError: true,
		},
		{
			name:        "address without host",
			address:     "http://",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientType, err := ValidateAddress(tt.address)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectType, clientType)
			}
		})
	}
}

func TestNewClient_AggregatorMode(t *testing.T) {
	// In aggregator mode, should always use node client regardless of address
	cfg := Config{
		Address:      "http://localhost:26657", // This would normally be app client
		Logger:       zerolog.Nop(),
		IsAggregator: true,
	}

	// Create a context for the test
	ctx := context.Background()

	// The node client is created successfully even without a real server
	// (connection happens lazily on first call)
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify it's a node client by checking that Validate doesn't return "not supported" error
	// Call with non-empty IDs to trigger actual validation logic
	ids := []datypes.ID{[]byte("test-id")}
	proofs := []datypes.Proof{[]byte("test-proof")}
	_, err = client.Validate(ctx, ids, proofs, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	// Should fail because no server is running, not because it's unsupported
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "not supported")
}

func TestNewClient_ForceType(t *testing.T) {
	tests := []struct {
		name              string
		forceType         ClientType
		address           string
		expectUnsupported bool // true if we expect "not supported" error
	}{
		{
			name:              "force node",
			forceType:         ClientTypeNode,
			address:           "http://localhost:26657", // Would normally be app
			expectUnsupported: false,                    // Node client supports all operations
		},
		{
			name:              "force app",
			forceType:         ClientTypeApp,
			address:           "http://localhost:26658", // Would normally be node
			expectUnsupported: true,                     // App client doesn't support proofs
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Address:   tt.address,
				Logger:    zerolog.Nop(),
				ForceType: tt.forceType,
			}

			client, err := NewClient(ctx, cfg)
			require.NoError(t, err)
			assert.NotNil(t, client)

			// Verify the right client type by checking GetProofs behavior
			_, err = client.GetProofs(ctx, nil, nil)
			if tt.expectUnsupported {
				// App client returns "not supported" error
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not supported")
			} else {
				// Node client doesn't return "not supported" (will fail to connect instead)
				// If err is nil (empty ids case), that's fine too
				if err != nil {
					assert.NotContains(t, err.Error(), "not supported")
				}
			}
		})
	}
}

func TestNewClient_AutoDetection(t *testing.T) {
	tests := []struct {
		name              string
		address           string
		expectUnsupported bool // true if we expect app client (which doesn't support proofs)
	}{
		{
			name:              "port 26657 uses app client",
			address:           "http://localhost:26657",
			expectUnsupported: true,
		},
		{
			name:              "port 26658 uses node client",
			address:           "http://localhost:26658",
			expectUnsupported: false,
		},
		{
			name:              "custom port uses node client",
			address:           "http://localhost:8080",
			expectUnsupported: false,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Address: tt.address,
				Logger:  zerolog.Nop(),
			}

			client, err := NewClient(ctx, cfg)
			require.NoError(t, err)
			assert.NotNil(t, client)

			// Verify the right client type by checking GetProofs behavior
			_, err = client.GetProofs(ctx, nil, nil)
			if tt.expectUnsupported {
				// App client returns "not supported" error
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not supported")
			} else {
				// Node client doesn't return "not supported" (will fail to connect instead)
				// If err is nil (empty ids case), that's fine too
				if err != nil {
					assert.NotContains(t, err.Error(), "not supported")
				}
			}
		})
	}
}
