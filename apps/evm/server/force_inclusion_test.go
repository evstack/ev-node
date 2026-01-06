package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/evstack/ev-node/pkg/config"
	da "github.com/evstack/ev-node/pkg/da/types"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"
)

// mockDA implements block/internal/da.Client for testing
type mockDA struct {
	submitFunc func(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) da.ResultSubmit
}

func (m *mockDA) Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) da.ResultSubmit {
	if m.submitFunc != nil {
		return m.submitFunc(ctx, data, gasPrice, namespace, options)
	}

	return da.ResultSubmit{BaseResult: da.BaseResult{Code: da.StatusSuccess, Height: 1}}
}

func (m *mockDA) Retrieve(ctx context.Context, height uint64, namespace []byte) da.ResultRetrieve {
	return da.ResultRetrieve{}
}

func (m *mockDA) RetrieveHeaders(ctx context.Context, height uint64) da.ResultRetrieve {
	return da.ResultRetrieve{}
}

func (m *mockDA) RetrieveData(ctx context.Context, height uint64) da.ResultRetrieve {
	return da.ResultRetrieve{}
}

func (m *mockDA) RetrieveForcedInclusion(ctx context.Context, height uint64) da.ResultRetrieve {
	return da.ResultRetrieve{}
}

func (m *mockDA) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	return nil, nil
}

func (m *mockDA) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	return nil, nil
}

func (m *mockDA) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	return nil, nil
}

func (m *mockDA) GetHeaderNamespace() []byte {
	return []byte("header")
}

func (m *mockDA) GetDataNamespace() []byte {
	return []byte("data")
}

func (m *mockDA) GetForcedInclusionNamespace() []byte {
	return []byte("forced")
}

func (m *mockDA) HasForcedInclusionNamespace() bool {
	return true
}

func TestForceInclusionServer_handleSendRawTransaction_Success(t *testing.T) {
	testHeight := uint64(100)

	mockDAClient := &mockDA{
		submitFunc: func(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) da.ResultSubmit {
			assert.Equal(t, 1, len(data))
			return da.ResultSubmit{BaseResult: da.BaseResult{Code: da.StatusSuccess, Height: testHeight}}
		},
	}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{
		DAStartHeight:          50,
		DAEpochForcedInclusion: 10,
	}

	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	// Create test request
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_sendRawTransaction",
		Params:  json.RawMessage(`["0x02f873010185012a05f20085012a05f2008252089400000000000000000000000000000000000000008080c080a0abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890aba0dcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210fe"]`),
	}

	reqJSON, err := json.Marshal(reqBody)
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqJSON))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsonResp JSONRPCResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NilError(t, err)
	assert.Equal(t, "2.0", jsonResp.JSONRPC)
	assert.Assert(t, jsonResp.Error == nil)
	assert.Assert(t, jsonResp.Result != nil)
}

func TestForceInclusionServer_handleSendRawTransaction_InvalidParams(t *testing.T) {
	mockDAClient := &mockDA{}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	tests := []struct {
		name   string
		params json.RawMessage
	}{
		{
			name:   "empty params",
			params: json.RawMessage(`[]`),
		},
		{
			name:   "too many params",
			params: json.RawMessage(`["0x123", "0x456"]`),
		},
		{
			name:   "invalid hex",
			params: json.RawMessage(`["not-hex"]`),
		},
		{
			name:   "no 0x prefix",
			params: json.RawMessage(`["123456"]`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "eth_sendRawTransaction",
				Params:  tt.params,
			}

			reqJSON, err := json.Marshal(reqBody)
			assert.NilError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqJSON))
			w := httptest.NewRecorder()

			server.handleJSONRPC(w, req)

			resp := w.Result()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var jsonResp JSONRPCResponse
			err = json.NewDecoder(resp.Body).Decode(&jsonResp)
			assert.NilError(t, err)
			assert.Assert(t, jsonResp.Error != nil)
			assert.Equal(t, InvalidParams, jsonResp.Error.Code)
		})
	}
}

func TestForceInclusionServer_handleSendRawTransaction_DAError(t *testing.T) {
	mockDAClient := &mockDA{
		submitFunc: func(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) da.ResultSubmit {
			return da.ResultSubmit{BaseResult: da.BaseResult{Code: da.StatusError, Message: da.ErrBlobSizeOverLimit.Error()}}
		},
	}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_sendRawTransaction",
		Params:  json.RawMessage(`["0x02f873"]`),
	}

	reqJSON, err := json.Marshal(reqBody)
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqJSON))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsonResp JSONRPCResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NilError(t, err)
	assert.Assert(t, jsonResp.Error != nil)
	assert.Equal(t, InternalError, jsonResp.Error.Code)
}

func TestForceInclusionServer_handleJSONRPC_MethodNotFound(t *testing.T) {
	mockDAClient := &mockDA{}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_getBalance",
		Params:  json.RawMessage(`[]`),
	}

	reqJSON, err := json.Marshal(reqBody)
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqJSON))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsonResp JSONRPCResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NilError(t, err)
	assert.Assert(t, jsonResp.Error != nil)
	assert.Equal(t, MethodNotFound, jsonResp.Error.Code)
}

func TestForceInclusionServer_handleJSONRPC_InvalidJSON(t *testing.T) {
	mockDAClient := &mockDA{}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsonResp JSONRPCResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NilError(t, err)
	assert.Assert(t, jsonResp.Error != nil)
	assert.Equal(t, ParseError, jsonResp.Error.Code)
}

func TestForceInclusionServer_StartStop(t *testing.T) {
	mockDAClient := &mockDA{}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("127.0.0.1:0", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	ctx := context.Background()
	err = server.Start(ctx)
	assert.NilError(t, err)

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Stop the server
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Stop(stopCtx)
	assert.NilError(t, err)
}

func TestDecodeHexTx(t *testing.T) {
	mockDAClient := &mockDA{}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid hex with 0x prefix",
			input:       "0x1234567890abcdef",
			expectError: false,
		},
		{
			name:        "valid hex with 0X prefix",
			input:       "0X1234567890ABCDEF",
			expectError: false,
		},
		{
			name:        "no 0x prefix",
			input:       "1234567890abcdef",
			expectError: true,
		},
		{
			name:        "invalid hex characters",
			input:       "0x123xyz",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "0x",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.decodeHexTx(tt.input)
			if tt.expectError {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
				assert.Assert(t, result != nil)
			}
		})
	}
}

func TestForceInclusionServer_ProxyToExecutionRPC(t *testing.T) {
	mockDAClient := &mockDA{}

	// Create a mock execution RPC server
	mockExecutionRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NilError(t, err)

		// Return a mock response
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  "0x1234567890abcdef",
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		assert.NilError(t, err)
	}))
	defer mockExecutionRPC.Close()

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, mockExecutionRPC.URL)
	assert.NilError(t, err)

	// Test proxying eth_getBalance
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_getBalance",
		Params:  json.RawMessage(`["0x1234567890123456789012345678901234567890", "latest"]`),
	}

	reqJSON, err := json.Marshal(reqBody)
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqJSON))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsonResp JSONRPCResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NilError(t, err)
	assert.Equal(t, "2.0", jsonResp.JSONRPC)
	assert.Assert(t, jsonResp.Error == nil)
	assert.Equal(t, "0x1234567890abcdef", jsonResp.Result)
}

func TestForceInclusionServer_ProxyWithoutExecutionRPC(t *testing.T) {
	mockDAClient := &mockDA{}

	cfg := config.Config{
		DA: config.DAConfig{
			ForcedInclusionNamespace: "0x0000000000000000000000000000000000000000000000000000666f72636564",
		},
	}

	gen := genesis.Genesis{}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	// Create server without execution RPC URL
	server, err := NewForceInclusionServer("", mockDAClient, cfg, gen, logger, "")
	assert.NilError(t, err)

	// Test that unknown methods return method not found when no execution RPC is configured
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_getBalance",
		Params:  json.RawMessage(`[]`),
	}

	reqJSON, err := json.Marshal(reqBody)
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqJSON))
	w := httptest.NewRecorder()

	server.handleJSONRPC(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsonResp JSONRPCResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.NilError(t, err)
	assert.Assert(t, jsonResp.Error != nil)
	assert.Equal(t, MethodNotFound, jsonResp.Error.Code)
}
