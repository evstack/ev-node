package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ EngineRPCClient = (*tracedEngineRPCClient)(nil)

// tracedEngineRPCClient wraps an EngineRPCClient and records spans.
type tracedEngineRPCClient struct {
	inner  EngineRPCClient
	tracer trace.Tracer
}

// withTracingEngineRPCClient decorates an EngineRPCClient with OpenTelemetry spans.
func withTracingEngineRPCClient(inner EngineRPCClient) EngineRPCClient {
	return &tracedEngineRPCClient{
		inner:  inner,
		tracer: otel.Tracer("ev-node/execution/engine-rpc"),
	}
}

func (t *tracedEngineRPCClient) ForkchoiceUpdated(ctx context.Context, state engine.ForkchoiceStateV1, args map[string]any) (*engine.ForkChoiceResponse, error) {
	ctx, span := t.tracer.Start(ctx, "Engine.ForkchoiceUpdated",
		trace.WithAttributes(
			attribute.String("method", "engine_forkchoiceUpdatedV3"),
			attribute.String("head_block_hash", state.HeadBlockHash.Hex()),
			attribute.String("safe_block_hash", state.SafeBlockHash.Hex()),
			attribute.String("finalized_block_hash", state.FinalizedBlockHash.Hex()),
		),
	)
	defer span.End()

	result, err := t.inner.ForkchoiceUpdated(ctx, state, args)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	attributes := []attribute.KeyValue{
		attribute.String("payload_status", result.PayloadStatus.Status),
	}

	if result.PayloadID != nil {
		attributes = append(attributes, attribute.String("payload_id", result.PayloadID.String()))
	}

	if result.PayloadStatus.LatestValidHash != nil {
		attributes = append(attributes, attribute.String("latest_valid_hash", result.PayloadStatus.LatestValidHash.Hex()))
	}

	span.SetAttributes(
		attributes...,
	)

	return result, nil
}

func (t *tracedEngineRPCClient) GetPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	ctx, span := t.tracer.Start(ctx, "Engine.GetPayload",
		trace.WithAttributes(
			attribute.String("method", "engine_getPayloadV4"),
			attribute.String("payload_id", payloadID.String()),
		),
	)
	defer span.End()

	result, err := t.inner.GetPayload(ctx, payloadID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int64("block_number", int64(result.ExecutionPayload.Number)),
		attribute.String("block_hash", result.ExecutionPayload.BlockHash.Hex()),
		attribute.String("state_root", result.ExecutionPayload.StateRoot.Hex()),
		attribute.Int("tx_count", len(result.ExecutionPayload.Transactions)),
		attribute.Int64("gas_used", int64(result.ExecutionPayload.GasUsed)),
	)

	return result, nil
}

func (t *tracedEngineRPCClient) NewPayload(ctx context.Context, payload *engine.ExecutableData, blobHashes []string, parentBeaconBlockRoot string, executionRequests [][]byte) (*engine.PayloadStatusV1, error) {
	ctx, span := t.tracer.Start(ctx, "Engine.NewPayload",
		trace.WithAttributes(
			attribute.String("method", "engine_newPayloadV4"),
			attribute.Int64("block_number", int64(payload.Number)),
			attribute.String("block_hash", payload.BlockHash.Hex()),
			attribute.String("parent_hash", payload.ParentHash.Hex()),
			attribute.Int("tx_count", len(payload.Transactions)),
			attribute.Int64("gas_used", int64(payload.GasUsed)),
		),
	)
	defer span.End()

	result, err := t.inner.NewPayload(ctx, payload, blobHashes, parentBeaconBlockRoot, executionRequests)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	attributes := []attribute.KeyValue{attribute.String("payload_status", result.Status)}

	if result.LatestValidHash != nil {
		attributes = append(attributes, attribute.String("latest_valid_hash", result.LatestValidHash.Hex()))
	}

	span.SetAttributes(attributes...)

	return result, nil
}
