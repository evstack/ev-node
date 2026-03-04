package evm

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/rpc"
)

// engineErrUnsupportedFork is the Engine API error code for "Unsupported fork".
// Defined in the Engine API specification.
const engineErrUnsupportedFork = -38005

// Engine API method names for GetPayload versions.
const (
	getPayloadV4Method = "engine_getPayloadV4"
	getPayloadV5Method = "engine_getPayloadV5"
)

var _ EngineRPCClient = (*engineRPCClient)(nil)

// engineRPCClient is the concrete implementation wrapping *rpc.Client.
// It auto-detects whether to use engine_getPayloadV4 (Prague) or
// engine_getPayloadV5 (Osaka) by caching the last successful version
// and falling back on "Unsupported fork" errors.
type engineRPCClient struct {
	client *rpc.Client
	// useV5 tracks whether GetPayload should prefer V5 (Osaka).
	// Starts false (V4/Prague). Flips automatically on unsupported-fork errors.
	useV5 atomic.Bool
}

// NewEngineRPCClient creates a new Engine API client.
func NewEngineRPCClient(client *rpc.Client) EngineRPCClient {
	return &engineRPCClient{client: client}
}

func (e *engineRPCClient) ForkchoiceUpdated(ctx context.Context, state engine.ForkchoiceStateV1, args map[string]any) (*engine.ForkChoiceResponse, error) {
	var result engine.ForkChoiceResponse
	err := e.client.CallContext(ctx, &result, "engine_forkchoiceUpdatedV3", state, args)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *engineRPCClient) GetPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	method := getPayloadV4Method
	altMethod := getPayloadV5Method
	if e.useV5.Load() {
		method = getPayloadV5Method
		altMethod = getPayloadV4Method
	}

	var result engine.ExecutionPayloadEnvelope
	err := e.client.CallContext(ctx, &result, method, payloadID)
	if err == nil {
		return &result, nil
	}

	if !isUnsupportedForkErr(err) {
		return nil, fmt.Errorf("%s payload %s: %w", method, payloadID, err)
	}

	// Primary method returned "Unsupported fork" -- try the other version.
	err = e.client.CallContext(ctx, &result, altMethod, payloadID)
	if err != nil {
		return nil, fmt.Errorf("%s fallback after %s unsupported fork, payload %s: %w", altMethod, method, payloadID, err)
	}

	// The alt method worked -- cache it for future calls.
	e.useV5.Store(altMethod == getPayloadV5Method)
	return &result, nil
}

// GetPayloadMethod returns the Engine API method name currently used by GetPayload.
// This allows wrappers (e.g. tracing) to report the resolved version.
func (e *engineRPCClient) GetPayloadMethod() string {
	if e.useV5.Load() {
		return getPayloadV5Method
	}
	return getPayloadV4Method
}

func (e *engineRPCClient) NewPayload(ctx context.Context, payload *engine.ExecutableData, blobHashes []string, parentBeaconBlockRoot string, executionRequests [][]byte) (*engine.PayloadStatusV1, error) {
	var result engine.PayloadStatusV1
	err := e.client.CallContext(ctx, &result, "engine_newPayloadV4", payload, blobHashes, parentBeaconBlockRoot, executionRequests)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// isUnsupportedForkErr reports whether err is an Engine API "Unsupported fork"
// JSON-RPC error (code -38005).
func isUnsupportedForkErr(err error) bool {
	var rpcErr rpc.Error
	return errors.As(err, &rpcErr) && rpcErr.ErrorCode() == engineErrUnsupportedFork
}
