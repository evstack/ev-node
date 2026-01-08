package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/rpc"
)

// engineRPCClient is the concrete implementation wrapping *rpc.Client.
type engineRPCClient struct {
	client *rpc.Client
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
	var result engine.ExecutionPayloadEnvelope
	err := e.client.CallContext(ctx, &result, "engine_getPayloadV4", payloadID)
	if err != nil {
		return nil, err
	}
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
