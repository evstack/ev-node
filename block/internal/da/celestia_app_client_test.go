package da

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/evstack/ev-node/block/internal/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

func TestNewCelestiaAppClient(t *testing.T) {
	logger := zerolog.New(nil)

	tests := []struct {
		name    string
		cfg     CelestiaAppConfig
		wantNil bool
	}{
		{
			name: "valid config",
			cfg: CelestiaAppConfig{
				RPCAddress:    "http://localhost:26657",
				Logger:        logger,
				Namespace:     "test_header",
				DataNamespace: "test_data",
			},
			wantNil: false,
		},
		{
			name: "empty rpc address returns nil",
			cfg: CelestiaAppConfig{
				RPCAddress:    "",
				Logger:        logger,
				Namespace:     "test",
				DataNamespace: "test_data",
			},
			wantNil: true,
		},
		{
			name: "default timeout is set",
			cfg: CelestiaAppConfig{
				RPCAddress:    "http://localhost:26657",
				Logger:        logger,
				Namespace:     "test",
				DataNamespace: "test_data",
				// DefaultTimeout is 0, should be set to 60s
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewCelestiaAppClient(tt.cfg)
			if tt.wantNil {
				assert.Nil(t, client)
			} else {
				assert.NotNil(t, client)
			}
		})
	}
}

func TestCelestiaAppClient_Namespaces(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:               "http://localhost:26657",
		Logger:                   logger,
		Namespace:                "header_ns",
		DataNamespace:            "data_ns",
		ForcedInclusionNamespace: "forced_ns",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	// Cast to access internal methods
	c := client.(*celestiaAppClient)

	// Test namespace getters
	assert.NotNil(t, c.GetHeaderNamespace())
	assert.NotNil(t, c.GetDataNamespace())
	assert.NotNil(t, c.GetForcedInclusionNamespace())
	assert.True(t, c.HasForcedInclusionNamespace())

	// Test with no forced inclusion namespace
	cfg2 := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "header_ns",
		DataNamespace: "data_ns",
		// No ForcedInclusionNamespace
	}
	client2 := NewCelestiaAppClient(cfg2)
	require.NotNil(t, client2)
	c2 := client2.(*celestiaAppClient)

	assert.False(t, c2.HasForcedInclusionNamespace())
	assert.Nil(t, c2.GetForcedInclusionNamespace())
}

func TestCelestiaAppClient_Submit_NotImplemented(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	data := [][]byte{[]byte("test blob")}
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	result := client.Submit(ctx, data, 0, ns, nil)

	assert.Equal(t, datypes.StatusError, result.Code)
	assert.Contains(t, result.Message, "not implemented")
}

func TestCelestiaAppClient_Retrieve_InvalidNamespace(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	// Invalid namespace (too short)
	invalidNs := []byte("short")

	result := client.Retrieve(ctx, 100, invalidNs)

	assert.Equal(t, datypes.StatusError, result.Code)
	assert.Contains(t, result.Message, "invalid namespace")
}

func TestCelestiaAppClient_Get_EmptyIDs(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	blobs, err := client.Get(ctx, nil, ns)

	assert.NoError(t, err)
	assert.Nil(t, blobs)
}

func TestCelestiaAppClient_GetProofs_NotImplemented(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()
	ids := []datypes.ID{[]byte("test_id")}

	proofs, err := client.GetProofs(ctx, ids, ns)

	assert.Error(t, err)
	assert.Nil(t, proofs)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestCelestiaAppClient_Validate_NotImplemented(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()
	ids := []datypes.ID{[]byte("test_id")}
	proofs := []datypes.Proof{[]byte("test_proof")}

	results, err := client.Validate(ctx, ids, proofs, ns)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestCelestiaAppClient_Validate_MismatchedIDsAndProofs(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()
	ids := []datypes.ID{[]byte("id1"), []byte("id2")}
	proofs := []datypes.Proof{[]byte("proof1")} // Mismatched length

	results, err := client.Validate(ctx, ids, proofs, ns)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "must match")
}

func TestCelestiaAppClient_Validate_Empty(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	results, err := client.Validate(ctx, nil, nil, ns)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestCelestiaAppClient_GetLatestDAHeight(t *testing.T) {
	// Create a mock server
	mockResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"sync_info": map[string]interface{}{
				"latest_block_height": "12345",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    server.URL,
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	height, err := client.GetLatestDAHeight(ctx)

	assert.NoError(t, err)
	assert.Equal(t, uint64(12345), height)
}

func TestCelestiaAppClient_GetLatestDAHeight_RPCError(t *testing.T) {
	// Create a mock server that returns an error
	mockResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"error": map[string]interface{}{
			"code":    -32600,
			"message": "Invalid Request",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    server.URL,
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	_, err := client.GetLatestDAHeight(ctx)

	assert.Error(t, err)
}

func TestCelestiaAppClient_Retrieve(t *testing.T) {
	// Create a mock server that returns a block
	txData := base64.StdEncoding.EncodeToString([]byte("mock_tx_data"))
	mockResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"block": map[string]interface{}{
				"header": map[string]interface{}{
					"height": "100",
					"time":   time.Now().Format(time.RFC3339Nano),
				},
				"data": map[string]interface{}{
					"txs": []string{txData},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    server.URL,
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	result := client.Retrieve(ctx, 100, ns)

	// Since extractBlobsFromTx is not fully implemented, we expect StatusNotFound
	assert.Equal(t, datypes.StatusNotFound, result.Code)
	assert.Equal(t, uint64(100), result.Height)
}

func TestCelestiaAppClient_Retrieve_FutureHeight(t *testing.T) {
	// Create a mock server that returns a "future height" error
	mockResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"error": map[string]interface{}{
			"code":    -32603,
			"message": "height 999999 is in the future",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    server.URL,
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	result := client.Retrieve(ctx, 999999, ns)

	assert.Equal(t, datypes.StatusHeightFromFuture, result.Code)
}

func TestCelestiaAppClient_Get_InvalidID(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()
	// Invalid ID (too short)
	ids := []datypes.ID{[]byte("short")}

	_, err := client.Get(ctx, ids, ns)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestCelestiaAppClient_Get_InvalidNamespace(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	// Invalid namespace
	invalidNs := []byte("short")
	ids := []datypes.ID{[]byte("12345678commitment")}

	_, err := client.Get(ctx, ids, invalidNs)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid namespace")
}

func TestCelestiaAppClient_rpcCall(t *testing.T) {
	// Test successful RPC call
	mockResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  map[string]string{"test": "value"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    server.URL,
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)
	c := client.(*celestiaAppClient)

	ctx := context.Background()
	resp, err := c.rpcCall(ctx, "test_method", map[string]string{"param": "value"})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestCelestiaAppClient_rpcCall_HTTPError(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://invalid-address-that-does-not-exist:12345",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)
	c := client.(*celestiaAppClient)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := c.rpcCall(ctx, "test_method", nil)

	assert.Error(t, err)
}

func TestCelestiaAppClient_Submit_BlobTooBig(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	// Create a blob that's too big
	hugeBlob := make([]byte, common.DefaultMaxBlobSize+1)
	data := [][]byte{hugeBlob}

	result := client.Submit(ctx, data, 0, ns, nil)

	assert.Equal(t, datypes.StatusTooBig, result.Code)
	assert.Contains(t, result.Message, "size")
}

func TestCelestiaAppClient_Submit_EmptyBlob(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	// Empty blob
	data := [][]byte{{}}

	result := client.Submit(ctx, data, 0, ns, nil)

	assert.Equal(t, datypes.StatusError, result.Code)
	assert.Contains(t, result.Message, "empty")
}

func TestCelestiaAppClient_extractBlobsFromTx(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)
	c := client.(*celestiaAppClient)

	ns, err := share.NewNamespaceFromBytes(datypes.NamespaceFromString("test_namespace").Bytes())
	require.NoError(t, err)
	txBytes := []byte("mock_transaction_data")

	// This is currently a placeholder - should return empty
	blobs := c.extractBlobsFromTx(txBytes, ns, 100)

	assert.Empty(t, blobs)
}

func TestCelestiaAppClient_GetProofs_EmptyIDs(t *testing.T) {
	logger := zerolog.New(nil)
	cfg := CelestiaAppConfig{
		RPCAddress:    "http://localhost:26657",
		Logger:        logger,
		Namespace:     "test",
		DataNamespace: "test_data",
	}

	client := NewCelestiaAppClient(cfg)
	require.NotNil(t, client)

	ctx := context.Background()
	ns := datypes.NamespaceFromString("test_namespace").Bytes()

	proofs, err := client.GetProofs(ctx, nil, ns)

	assert.NoError(t, err)
	assert.Empty(t, proofs)
}
