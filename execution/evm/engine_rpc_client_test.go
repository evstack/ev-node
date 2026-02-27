package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonRPCRequest is a minimal JSON-RPC request for test inspection.
type jsonRPCRequest struct {
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
	ID     json.RawMessage   `json:"id"`
}

// fakeEngineServer returns an httptest.Server that responds to engine_getPayloadV4
// and engine_getPayloadV5 according to the provided handler. The handler receives
// the method name and returns (result JSON, error code, error message).
// If errorCode is 0, a success response is sent.
func fakeEngineServer(t *testing.T, handler func(method string) (resultJSON string, errCode int, errMsg string)) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Logf("failed to decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		resultJSON, errCode, errMsg := handler(req.Method)

		w.Header().Set("Content-Type", "application/json")
		if errCode != 0 {
			resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":%d,"message":"%s"}}`,
				req.ID, errCode, errMsg)
			_, _ = w.Write([]byte(resp))
		} else {
			resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, resultJSON)
			_, _ = w.Write([]byte(resp))
		}
	}))
}

// minimalPayloadEnvelopeJSON is a minimal valid ExecutionPayloadEnvelope JSON
// that go-ethereum can unmarshal without error.
const minimalPayloadEnvelopeJSON = `{
	"executionPayload": {
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"feeRecipient": "0x0000000000000000000000000000000000000000",
		"stateRoot": "0x0000000000000000000000000000000000000000000000000000000000000001",
		"receiptsRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"prevRandao": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"blockNumber": "0x1",
		"gasLimit": "0x1000000",
		"gasUsed": "0x0",
		"timestamp": "0x1",
		"extraData": "0x",
		"baseFeePerGas": "0x1",
		"blockHash": "0x0000000000000000000000000000000000000000000000000000000000000002",
		"transactions": [],
		"blobGasUsed": "0x0",
		"excessBlobGas": "0x0"
	},
	"blockValue": "0x0",
	"blobsBundle": {
		"commitments": [],
		"proofs": [],
		"blobs": []
	},
	"executionRequests": [],
	"shouldOverrideBuilder": false
}`

func dialTestServer(t *testing.T, serverURL string) *rpc.Client {
	t.Helper()
	client, err := rpc.Dial(serverURL)
	require.NoError(t, err)
	return client
}

func TestGetPayload_PragueChain_UsesV4(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(method string) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, method)
		mu.Unlock()

		if method == "engine_getPayloadV4" {
			return minimalPayloadEnvelopeJSON, 0, ""
		}
		return "", -38005, "Unsupported fork"
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	// First call -- should use V4 directly, succeed.
	_, err := client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{"engine_getPayloadV4"}, calledMethods, "should call V4 only")
	calledMethods = nil
	mu.Unlock()

	// Second call -- still V4 (cached).
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{"engine_getPayloadV4"}, calledMethods, "should still use V4")
	mu.Unlock()
}

func TestGetPayload_OsakaChain_FallsBackToV5(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(method string) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, method)
		mu.Unlock()

		if method == "engine_getPayloadV5" {
			return minimalPayloadEnvelopeJSON, 0, ""
		}
		return "", -38005, "Unsupported fork"
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	// First call -- V4 fails with -38005, falls back to V5.
	_, err := client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{"engine_getPayloadV4", "engine_getPayloadV5"}, calledMethods,
		"should try V4 then fall back to V5")
	calledMethods = nil
	mu.Unlock()

	// Second call -- should go directly to V5 (cached).
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{"engine_getPayloadV5"}, calledMethods,
		"should use cached V5 without trying V4")
	mu.Unlock()
}

func TestGetPayload_ForkUpgrade_SwitchesV4ToV5(t *testing.T) {
	var mu sync.Mutex
	osakaActive := false

	srv := fakeEngineServer(t, func(method string) (string, int, string) {
		mu.Lock()
		active := osakaActive
		mu.Unlock()

		if active {
			// Post-Osaka: V5 works, V4 rejected
			if method == "engine_getPayloadV5" {
				return minimalPayloadEnvelopeJSON, 0, ""
			}
			return "", -38005, "Unsupported fork"
		}
		// Pre-Osaka: V4 works, V5 rejected
		if method == "engine_getPayloadV4" {
			return minimalPayloadEnvelopeJSON, 0, ""
		}
		return "", -38005, "Unsupported fork"
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	// Pre-upgrade: V4 works.
	_, err := client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	// Simulate fork activation.
	mu.Lock()
	osakaActive = true
	mu.Unlock()

	// First post-upgrade call: V4 fails, falls back to V5, caches.
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	// Subsequent calls: V5 directly.
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)
}

func TestGetPayload_NonForkError_Propagated(t *testing.T) {
	srv := fakeEngineServer(t, func(method string) (string, int, string) {
		// Return a different error (e.g., unknown payload)
		return "", -38001, "Unknown payload"
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	_, err := client.GetPayload(ctx, engine.PayloadID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown payload")
}

func TestIsUnsupportedForkErr(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"generic error", fmt.Errorf("something went wrong"), false},
		{"unsupported fork code", &testRPCError{code: -38005, msg: "Unsupported fork"}, true},
		{"different code", &testRPCError{code: -38001, msg: "Unknown payload"}, false},
		{"wrapped unsupported fork", fmt.Errorf("call failed: %w", &testRPCError{code: -38005, msg: "Unsupported fork"}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isUnsupportedForkErr(tt.err))
		})
	}
}

// testRPCError implements rpc.Error for testing.
type testRPCError struct {
	code int
	msg  string
}

func (e *testRPCError) Error() string  { return e.msg }
func (e *testRPCError) ErrorCode() int { return e.code }
