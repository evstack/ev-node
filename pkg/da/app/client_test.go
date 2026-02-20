package app

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// setupMockServer creates a mock CometBFT RPC server for testing
func setupMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(func() { server.Close() })
	return server
}

// createTestClient creates a Client pointing to the mock server
func createTestClient(serverURL string) *Client {
	return NewClient(Config{
		RPCAddress:     serverURL,
		Logger:         zerolog.Nop(),
		DefaultTimeout: 5 * time.Second,
	})
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		expectNil  bool
		expectAddr string
	}{
		{
			name: "valid config",
			cfg: Config{
				RPCAddress:     "http://localhost:26657",
				Logger:         zerolog.Nop(),
				DefaultTimeout: 60 * time.Second,
			},
			expectNil:  false,
			expectAddr: "http://localhost:26657",
		},
		{
			name:       "empty address returns nil",
			cfg:        Config{RPCAddress: ""},
			expectNil:  true,
			expectAddr: "",
		},
		{
			name: "default timeout",
			cfg: Config{
				RPCAddress: "http://localhost:26657",
				Logger:     zerolog.Nop(),
				// DefaultTimeout is 0, should use default
			},
			expectNil:  false,
			expectAddr: "http://localhost:26657",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)
			if tt.expectNil {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
				assert.Equal(t, tt.expectAddr, client.rpcAddress)
				assert.NotNil(t, client.httpClient)
			}
		})
	}
}

func TestClient_Submit(t *testing.T) {
	client := createTestClient("http://localhost:26657")
	ctx := context.Background()

	// Test with invalid namespace
	t.Run("invalid namespace", func(t *testing.T) {
		result := client.Submit(ctx, [][]byte{[]byte("test")}, 0.0, []byte("invalid"), nil)
		assert.Equal(t, datypes.StatusError, result.Code)
		assert.Contains(t, result.Message, "invalid namespace")
	})

	// Test with empty blob
	t.Run("empty blob", func(t *testing.T) {
		ns := share.MustNewV0Namespace([]byte("testns"))
		result := client.Submit(ctx, [][]byte{{}}, 0.0, ns.Bytes(), nil)
		assert.Equal(t, datypes.StatusError, result.Code)
		assert.Contains(t, result.Message, "empty")
	})

	// Test with blob too big
	t.Run("blob too big", func(t *testing.T) {
		ns := share.MustNewV0Namespace([]byte("testns"))
		bigBlob := make([]byte, defaultMaxBlobSize+1)
		result := client.Submit(ctx, [][]byte{bigBlob}, 0.0, ns.Bytes(), nil)
		assert.Equal(t, datypes.StatusTooBig, result.Code)
	})

	// Test that Submit returns not implemented error
	t.Run("not implemented", func(t *testing.T) {
		ns := share.MustNewV0Namespace([]byte("testns"))
		result := client.Submit(ctx, [][]byte{[]byte("test data")}, 0.0, ns.Bytes(), nil)
		assert.Equal(t, datypes.StatusError, result.Code)
		assert.Contains(t, result.Message, "not implemented")
		assert.Contains(t, result.Message, "transaction signing")
	})
}

func TestClient_Retrieve(t *testing.T) {
	ctx := context.Background()
	ns := share.MustNewV0Namespace([]byte("testns"))

	t.Run("invalid namespace", func(t *testing.T) {
		client := createTestClient("http://localhost:26657")
		result := client.Retrieve(ctx, 100, []byte("invalid"))
		assert.Equal(t, datypes.StatusError, result.Code)
		assert.Contains(t, result.Message, "invalid namespace")
	})

	t.Run("block not found", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Return error for block request
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &rpcError{
					Code:    -32603,
					Message: "Internal error: height 100 is not available, lowest height is 1",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		result := client.Retrieve(ctx, 100, ns.Bytes())
		assert.Equal(t, datypes.StatusNotFound, result.Code)
	})

	t.Run("future height", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &rpcError{
					Code:    -32603,
					Message: "height 999999 is from future",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		result := client.Retrieve(ctx, 999999, ns.Bytes())
		assert.Equal(t, datypes.StatusHeightFromFuture, result.Code)
	})

	t.Run("no blobs found", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			blockResp := blockResult{
				Block: struct {
					Header struct {
						Height string `json:"height"`
						Time   string `json:"time"`
					} `json:"header"`
					Data struct {
						Txs []string `json:"txs"`
					} `json:"data"`
				}{
					Header: struct {
						Height string `json:"height"`
						Time   string `json:"time"`
					}{
						Height: "100",
						Time:   time.Now().Format(time.RFC3339Nano),
					},
					Data: struct {
						Txs []string `json:"txs"`
					}{
						Txs: []string{}, // No transactions
					},
				},
			}
			resultBytes, _ := json.Marshal(blockResp)
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  resultBytes,
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		result := client.Retrieve(ctx, 100, ns.Bytes())
		assert.Equal(t, datypes.StatusNotFound, result.Code)
		assert.Contains(t, result.Message, "not found")
	})

	t.Run("successful retrieval", func(t *testing.T) {
		// Create a simple blob transaction
		// This is a simplified test - real blob txs would be more complex
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			blockResp := blockResult{
				Block: struct {
					Header struct {
						Height string `json:"height"`
						Time   string `json:"time"`
					} `json:"header"`
					Data struct {
						Txs []string `json:"txs"`
					} `json:"data"`
				}{
					Header: struct {
						Height string `json:"height"`
						Time   string `json:"time"`
					}{
						Height: "100",
						Time:   time.Now().Format(time.RFC3339Nano),
					},
					Data: struct {
						Txs []string `json:"txs"`
					}{
						// Regular transaction (not a blob tx)
						Txs: []string{base64.StdEncoding.EncodeToString([]byte("regular tx"))},
					},
				},
			}
			resultBytes, _ := json.Marshal(blockResp)
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  resultBytes,
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		result := client.Retrieve(ctx, 100, ns.Bytes())
		// Should return not found since we didn't include a real blob tx
		assert.Equal(t, datypes.StatusNotFound, result.Code)
	})
}

func TestClient_Get(t *testing.T) {
	ctx := context.Background()
	ns := share.MustNewV0Namespace([]byte("testns"))

	t.Run("empty ids", func(t *testing.T) {
		client := createTestClient("http://localhost:26657")
		blobs, err := client.Get(ctx, nil, ns.Bytes())
		require.NoError(t, err)
		assert.Nil(t, blobs)
	})

	t.Run("invalid namespace", func(t *testing.T) {
		client := createTestClient("http://localhost:26657")
		id := make([]byte, 10)
		binary.LittleEndian.PutUint64(id, 100)
		_, err := client.Get(ctx, []datypes.ID{id}, []byte("invalid"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid namespace")
	})

	t.Run("invalid blob id", func(t *testing.T) {
		client := createTestClient("http://localhost:26657")
		// ID too short
		_, err := client.Get(ctx, []datypes.ID{[]byte("short")}, ns.Bytes())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid blob id")
	})
}

func TestClient_GetLatestDAHeight(t *testing.T) {
	ctx := context.Background()

	t.Run("successful status fetch", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			statusResp := statusResult{
				SyncInfo: struct {
					LatestBlockHeight string `json:"latest_block_height"`
				}{
					LatestBlockHeight: "12345",
				},
			}
			resultBytes, _ := json.Marshal(statusResp)
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  resultBytes,
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		height, err := client.GetLatestDAHeight(ctx)
		require.NoError(t, err)
		assert.Equal(t, uint64(12345), height)
	})

	t.Run("rpc error", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &rpcError{
					Code:    -32603,
					Message: "Internal error",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		_, err := client.GetLatestDAHeight(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Internal error")
	})

	t.Run("invalid height format", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			statusResp := statusResult{
				SyncInfo: struct {
					LatestBlockHeight string `json:"latest_block_height"`
				}{
					LatestBlockHeight: "not-a-number",
				},
			}
			resultBytes, _ := json.Marshal(statusResp)
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  resultBytes,
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		_, err := client.GetLatestDAHeight(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse block height")
	})
}

func TestClient_GetProofs(t *testing.T) {
	ctx := context.Background()
	client := createTestClient("http://localhost:26657")
	ns := share.MustNewV0Namespace([]byte("testns"))

	t.Run("returns not supported error", func(t *testing.T) {
		id := make([]byte, 10)
		binary.LittleEndian.PutUint64(id, 100)
		proofs, err := client.GetProofs(ctx, []datypes.ID{id}, ns.Bytes())
		require.Error(t, err)
		assert.Nil(t, proofs)
		assert.Contains(t, err.Error(), "not supported")
		assert.Contains(t, err.Error(), "proof generation")
	})

	t.Run("empty ids returns error", func(t *testing.T) {
		// App client returns error for all GetProofs calls, even with empty IDs
		proofs, err := client.GetProofs(ctx, nil, ns.Bytes())
		require.Error(t, err)
		assert.Nil(t, proofs)
		assert.Contains(t, err.Error(), "not supported")
	})
}

func TestClient_Validate(t *testing.T) {
	ctx := context.Background()
	client := createTestClient("http://localhost:26657")
	ns := share.MustNewV0Namespace([]byte("testns"))

	t.Run("returns not supported error", func(t *testing.T) {
		id := make([]byte, 10)
		binary.LittleEndian.PutUint64(id, 100)
		proof := datypes.Proof([]byte("proof"))
		results, err := client.Validate(ctx, []datypes.ID{id}, []datypes.Proof{proof}, ns.Bytes())
		require.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "not supported")
		assert.Contains(t, err.Error(), "proof validation")
	})

	t.Run("empty ids returns error", func(t *testing.T) {
		// App client returns error for all Validate calls, even with empty IDs
		results, err := client.Validate(ctx, nil, nil, ns.Bytes())
		require.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "not supported")
	})
}

func TestExtractBlobsFromTx(t *testing.T) {
	client := createTestClient("http://localhost:26657")
	ns := share.MustNewV0Namespace([]byte("testns"))

	t.Run("non-blob transaction", func(t *testing.T) {
		txBytes := []byte("regular transaction data")
		blobs := client.extractBlobsFromTx(txBytes, ns, 100)
		assert.Empty(t, blobs)
	})

	t.Run("invalid blob tx", func(t *testing.T) {
		// Invalid blob tx format
		txBytes := []byte{0x00, 0x01, 0x02, 0x03}
		blobs := client.extractBlobsFromTx(txBytes, ns, 100)
		assert.Empty(t, blobs)
	})
}

func TestRPCCall(t *testing.T) {
	ctx := context.Background()

	t.Run("successful call", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{"test": "result"}`),
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		resp, err := client.rpcCall(ctx, "test_method", map[string]string{"key": "value"})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		// Compare the unmarshaled JSON to avoid whitespace issues
		var expected, actual map[string]interface{}
		require.NoError(t, json.Unmarshal(json.RawMessage(`{"test": "result"}`), &expected))
		require.NoError(t, json.Unmarshal(resp.Result, &actual))
		assert.Equal(t, expected, actual)
	})

	t.Run("rpc error response", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			resp := rpcResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &rpcError{
					Code:    -32600,
					Message: "Invalid Request",
					Data:    "extra data",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		client := createTestClient(server.URL)
		_, err := client.rpcCall(ctx, "test_method", nil)
		require.Error(t, err)
		rpcErr, ok := err.(*rpcError)
		require.True(t, ok)
		assert.Equal(t, -32600, rpcErr.Code)
		assert.Equal(t, "Invalid Request", rpcErr.Message)
	})

	t.Run("http error", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		client := createTestClient(server.URL)
		_, err := client.rpcCall(ctx, "test_method", nil)
		require.Error(t, err)
	})

	t.Run("invalid json response", func(t *testing.T) {
		server := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not valid json"))
		})

		client := createTestClient(server.URL)
		_, err := client.rpcCall(ctx, "test_method", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal RPC response")
	})
}

func TestRPCTypes(t *testing.T) {
	t.Run("rpcError Error() method", func(t *testing.T) {
		err := &rpcError{
			Code:    -32600,
			Message: "Invalid Request",
			Data:    "extra data",
		}
		assert.Equal(t, "RPC error -32600: Invalid Request: extra data", err.Error())
	})
}

func TestClient_InterfaceCompliance(t *testing.T) {
	// Verify that Client implements datypes.BlobClient
	client := &Client{}
	var _ datypes.BlobClient = client
}

// Benchmark tests
func BenchmarkExtractBlobsFromTx(b *testing.B) {
	client := createTestClient("http://localhost:26657")
	ns := share.MustNewV0Namespace([]byte("testns"))
	txBytes := []byte("regular transaction data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.extractBlobsFromTx(txBytes, ns, 100)
	}
}
