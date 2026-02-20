package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// mockNodeServer creates a mock celestia-node server
func mockNodeServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// node.Ready response
		if req.Method == "node.Ready" {
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  true,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Return method not found for other methods
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "method not found",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

// mockAppServer creates a mock celestia-app server
func mockAppServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// status response
		if req.Method == "status" {
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"node_info": map[string]interface{}{
						"network": "celestia-local",
					},
					"sync_info": map[string]interface{}{
						"latest_block_height": "100",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Return method not found for other methods
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "method not found",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestDetectClientType(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		setupServer  func() *httptest.Server
		expectedType ClientType
	}{
		{
			name:         "detects celestia-node endpoint",
			setupServer:  mockNodeServer,
			expectedType: ClientTypeNode,
		},
		{
			name:         "detects celestia-app endpoint",
			setupServer:  mockAppServer,
			expectedType: ClientTypeApp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			result := detectClientType(ctx, server.URL)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestDetectClientType_NoServer(t *testing.T) {
	ctx := context.Background()

	// When no server is running, should default to node client
	result := detectClientType(ctx, "http://localhost:59999")
	assert.Equal(t, ClientTypeNode, result)
}

func TestIsNodeAddress(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		expected    bool
	}{
		{
			name:        "node endpoint returns true",
			setupServer: mockNodeServer,
			expected:    true,
		},
		{
			name:        "app endpoint returns false",
			setupServer: mockAppServer,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			result := IsNodeAddress(ctx, server.URL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAppAddress(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		expected    bool
	}{
		{
			name:        "node endpoint returns false",
			setupServer: mockNodeServer,
			expected:    false,
		},
		{
			name:        "app endpoint returns true",
			setupServer: mockAppServer,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			result := IsAppAddress(ctx, server.URL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAddress(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		address     string
		expectError bool
		expectType  ClientType
	}{
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
			clientType, err := ValidateAddress(ctx, tt.address)
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
		Address:      "http://localhost:26657",
		Logger:       zerolog.Nop(),
		IsAggregator: true,
	}

	ctx := context.Background()

	// The node client is created successfully even without a real server
	// (connection happens lazily on first call)
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify it's a node client by checking that Validate doesn't return "not supported" error
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
		expectUnsupported bool // true if we expect "not supported" error
	}{
		{
			name:              "force node",
			forceType:         ClientTypeNode,
			expectUnsupported: false,
		},
		{
			name:              "force app",
			forceType:         ClientTypeApp,
			expectUnsupported: true,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Address:   "http://localhost:59999", // Non-existent server
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
				if err != nil {
					assert.NotContains(t, err.Error(), "not supported")
				}
			}
		})
	}
}

func TestNewClient_AutoDetection(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		setupServer       func() *httptest.Server
		expectUnsupported bool // true if we expect app client (which doesn't support proofs)
	}{
		{
			name:              "detects app client",
			setupServer:       mockAppServer,
			expectUnsupported: true,
		},
		{
			name:              "detects node client",
			setupServer:       mockNodeServer,
			expectUnsupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			cfg := Config{
				Address: server.URL,
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
				if err != nil {
					assert.NotContains(t, err.Error(), "not supported")
				}
			}
		})
	}
}

func TestNewClient_AutoDetection_NoServer(t *testing.T) {
	ctx := context.Background()

	// When no server is running, defaults to node client
	cfg := Config{
		Address: "http://localhost:59999",
		Logger:  zerolog.Nop(),
	}

	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify it's a node client (doesn't return "not supported")
	_, err = client.GetProofs(ctx, nil, nil)
	if err != nil {
		assert.NotContains(t, err.Error(), "not supported")
	}
}
