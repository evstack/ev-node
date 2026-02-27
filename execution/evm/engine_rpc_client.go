package evm

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/rpc"
)

// engineErrUnsupportedFork is the Engine API error code for "Unsupported fork".
// Defined in the Engine API specification.
const engineErrUnsupportedFork = -38005

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
	method := "engine_getPayloadV4"
	altMethod := "engine_getPayloadV5"
	if e.useV5.Load() {
		method = "engine_getPayloadV5"
		altMethod = "engine_getPayloadV4"
	}

	var result engine.ExecutionPayloadEnvelope
	err := e.client.CallContext(ctx, &result, method, payloadID)
	if err == nil {
		return &result, nil
	}

	if !isUnsupportedForkErr(err) {
		return nil, err
	}

	// Primary method returned "Unsupported fork" -- try the other version.
	err = e.client.CallContext(ctx, &result, altMethod, payloadID)
	if err != nil {
		return nil, err
	}

	// The alt method worked -- cache it for future calls.
	e.useV5.Store(altMethod == "engine_getPayloadV5")
	return &result, nil
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
