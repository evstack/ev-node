package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

// fakeEngineServer returns an httptest.Server that responds according to the
// provided handler. The handler receives the full JSON-RPC request and returns
// (result JSON, error code, error message).
// If errorCode is 0, a success response is sent.
func fakeEngineServer(t *testing.T, handler func(req jsonRPCRequest) (resultJSON string, errCode int, errMsg string)) *httptest.Server {
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

		resultJSON, errCode, errMsg := handler(req)

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

const (
	validForkchoiceResponseJSON = `{
		"payloadStatus": {
			"status": "VALID",
			"latestValidHash": null,
			"validationError": null
		},
		"payloadId": "0x0000000000000001"
	}`
	validPayloadStatusJSON = `{
		"status": "VALID",
		"latestValidHash": null,
		"validationError": null
	}`
	zeroHashHex = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

func minimalAmsterdamPayloadEnvelopeJSON(t *testing.T) string {
	t.Helper()
	withAmsterdamFields := strings.Replace(
		minimalPayloadEnvelopeJSON,
		`"excessBlobGas": "0x0"`,
		`"excessBlobGas": "0x0",
		"slotNumber": "0x1",
		"blockAccessList": ["0x1234"]`,
		1,
	)
	require.NotEqual(t, minimalPayloadEnvelopeJSON, withAmsterdamFields)
	return withAmsterdamFields
}

func dialTestServer(t *testing.T, serverURL string) *rpc.Client {
	t.Helper()
	client, err := rpc.Dial(serverURL)
	require.NoError(t, err)
	return client
}

func TestGetPayload_PragueChain_UsesV4(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		if req.Method == getPayloadV4Method {
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
	assert.Equal(t, []string{getPayloadV4Method}, calledMethods, "should call V4 only")
	calledMethods = nil
	mu.Unlock()

	// Second call -- still V4 (cached).
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{getPayloadV4Method}, calledMethods, "should still use V4")
	mu.Unlock()
}

func TestGetPayload_OsakaChain_FallsBackToV5(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		if req.Method == getPayloadV5Method {
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
	assert.Equal(t, []string{getPayloadV4Method, getPayloadV5Method}, calledMethods,
		"should try V4 then fall back to V5")
	calledMethods = nil
	mu.Unlock()

	// Second call -- should go directly to V5 (cached).
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{getPayloadV5Method}, calledMethods,
		"should use cached V5 without trying V4")
	mu.Unlock()
}

func TestGetPayload_AmsterdamChain_FallsBackToV6(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		if req.Method == getPayloadV6Method {
			return minimalAmsterdamPayloadEnvelopeJSON(t), 0, ""
		}
		return "", -38005, "Unsupported fork"
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	_, err := client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{getPayloadV4Method, getPayloadV5Method, getPayloadV6Method}, calledMethods,
		"should try V4, V5, then fall back to V6")
	calledMethods = nil
	mu.Unlock()

	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{getPayloadV6Method}, calledMethods,
		"should use cached V6 without trying earlier versions")
	mu.Unlock()
}

func TestGetPayload_ForkUpgrade_SwitchesV4ToV5(t *testing.T) {
	var mu sync.Mutex
	var calledMethods []string
	osakaActive := false

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		active := osakaActive
		mu.Unlock()

		if active {
			// Post-Osaka: V5 works, V4 rejected
			if req.Method == getPayloadV5Method {
				return minimalPayloadEnvelopeJSON, 0, ""
			}
			return "", -38005, "Unsupported fork"
		}
		// Pre-Osaka: V4 works, V5 rejected
		if req.Method == getPayloadV4Method {
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

	mu.Lock()
	assert.Equal(t, []string{getPayloadV4Method}, calledMethods, "pre-upgrade should call V4 only")
	calledMethods = nil
	mu.Unlock()

	// Simulate fork activation.
	mu.Lock()
	osakaActive = true
	mu.Unlock()

	// First post-upgrade call: V4 fails, falls back to V5, caches.
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{getPayloadV4Method, getPayloadV5Method}, calledMethods,
		"first post-upgrade call should try V4 then fall back to V5")
	calledMethods = nil
	mu.Unlock()

	// Subsequent calls: V5 directly (cached).
	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{getPayloadV5Method}, calledMethods,
		"subsequent calls should use cached V5 directly")
	mu.Unlock()
}

func TestGetPayload_NonForkError_Propagated(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		// Return a different error (e.g., unknown payload)
		return "", -38001, "Unknown payload"
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	_, err := client.GetPayload(ctx, engine.PayloadID{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown payload")

	mu.Lock()
	assert.Equal(t, []string{getPayloadV4Method}, calledMethods,
		"non-fork errors should not fall back to alternate versions")
	mu.Unlock()
}

func TestForkchoiceUpdated_AmsterdamAttributes_UsesV4AndPrefersGetPayloadV6(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		switch req.Method {
		case forkchoiceUpdatedV3Method:
			require.Len(t, req.Params, 2)
			var attrs map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(req.Params[1], &attrs))
			require.NotContains(t, attrs, "slotNumber")
			return "", -38005, "Unsupported fork"
		case forkchoiceUpdatedV4Method:
			require.Len(t, req.Params, 2)
			var attrs map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(req.Params[1], &attrs))
			require.Contains(t, attrs, "slotNumber")
			return validForkchoiceResponseJSON, 0, ""
		case getPayloadV6Method:
			return minimalAmsterdamPayloadEnvelopeJSON(t), 0, ""
		default:
			return "", -38005, "Unsupported fork"
		}
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	_, err := client.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{}, map[string]any{
		"timestamp":  uint64(1),
		"slotNumber": uint64(1),
	})
	require.NoError(t, err)

	_, err = client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{forkchoiceUpdatedV3Method, forkchoiceUpdatedV4Method, getPayloadV6Method}, calledMethods)
	mu.Unlock()
}

func TestForkchoiceUpdated_PragueAttributes_UsesV3AndStripsSlotNumber(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		if req.Method != forkchoiceUpdatedV3Method {
			return "", -38005, "Unsupported fork"
		}
		require.Len(t, req.Params, 2)
		var attrs map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(req.Params[1], &attrs))
		require.NotContains(t, attrs, "slotNumber")
		return validForkchoiceResponseJSON, 0, ""
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	_, err := client.ForkchoiceUpdated(context.Background(), engine.ForkchoiceStateV1{}, map[string]any{
		"timestamp":  uint64(1),
		"slotNumber": uint64(1),
	})
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{forkchoiceUpdatedV3Method}, calledMethods)
	mu.Unlock()
}

func TestNewPayload_PraguePayload_UsesV4(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		if req.Method != newPayloadV4Method {
			return "", -38005, "Unsupported fork"
		}

		require.Len(t, req.Params, 4)
		var payload map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(req.Params[0], &payload))
		require.NotContains(t, payload, "blockAccessList")
		return validPayloadStatusJSON, 0, ""
	})
	defer srv.Close()

	var envelope EnginePayloadEnvelope
	require.NoError(t, json.Unmarshal([]byte(minimalPayloadEnvelopeJSON), &envelope))

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	_, err := client.NewPayload(context.Background(), &envelope, []string{}, zeroHashHex, envelope.Requests)
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{newPayloadV4Method}, calledMethods)
	mu.Unlock()
}

func TestNewPayload_AmsterdamPayload_UsesV5AndPreservesBlockAccessList(t *testing.T) {
	var calledMethods []string
	var mu sync.Mutex

	srv := fakeEngineServer(t, func(req jsonRPCRequest) (string, int, string) {
		mu.Lock()
		calledMethods = append(calledMethods, req.Method)
		mu.Unlock()

		switch req.Method {
		case forkchoiceUpdatedV3Method:
			return "", -38005, "Unsupported fork"
		case forkchoiceUpdatedV4Method:
			return validForkchoiceResponseJSON, 0, ""
		case getPayloadV6Method:
			return minimalAmsterdamPayloadEnvelopeJSON(t), 0, ""
		case newPayloadV5Method:
			require.Len(t, req.Params, 4)
			var payload map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(req.Params[0], &payload))
			require.Contains(t, payload, "slotNumber")
			require.Contains(t, payload, "blockAccessList")
			require.JSONEq(t, `["0x1234"]`, string(payload["blockAccessList"]))
			return validPayloadStatusJSON, 0, ""
		default:
			return "", -38005, "Unsupported fork"
		}
	})
	defer srv.Close()

	client := NewEngineRPCClient(dialTestServer(t, srv.URL))
	ctx := context.Background()

	_, err := client.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{}, map[string]any{
		"slotNumber": uint64(1),
	})
	require.NoError(t, err)

	payload, err := client.GetPayload(ctx, engine.PayloadID{})
	require.NoError(t, err)

	_, err = client.NewPayload(ctx, payload, []string{}, zeroHashHex, payload.Requests)
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []string{forkchoiceUpdatedV3Method, forkchoiceUpdatedV4Method, getPayloadV6Method, newPayloadV5Method}, calledMethods)
	mu.Unlock()
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
