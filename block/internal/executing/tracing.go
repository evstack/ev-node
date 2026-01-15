package executing

import (
	"context"
	"encoding/hex"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/types"
)

var _ BlockProducer = (*tracedBlockProducer)(nil)

// tracedBlockProducer decorates a BlockProducer with OpenTelemetry spans.
type tracedBlockProducer struct {
	inner  BlockProducer
	tracer trace.Tracer
}

// WithTracingBlockProducer decorates the provided BlockProducer with tracing spans.
func WithTracingBlockProducer(inner BlockProducer) BlockProducer {
	return &tracedBlockProducer{
		inner:  inner,
		tracer: otel.Tracer("ev-node/block-executor"),
	}
}

func (t *tracedBlockProducer) ProduceBlock(ctx context.Context) error {
	ctx, span := t.tracer.Start(ctx, "BlockExecutor.ProduceBlock")
	defer span.End()

	err := t.inner.ProduceBlock(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (t *tracedBlockProducer) RetrieveBatch(ctx context.Context) (*BatchData, error) {
	ctx, span := t.tracer.Start(ctx, "BlockExecutor.RetrieveBatch")
	defer span.End()

	batchData, err := t.inner.RetrieveBatch(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if batchData != nil && batchData.Batch != nil {
		span.SetAttributes(
			attribute.Int("batch.tx_count", len(batchData.Transactions)),
		)
	}
	return batchData, nil
}

func (t *tracedBlockProducer) CreateBlock(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error) {
	txCount := 0
	if batchData != nil && batchData.Batch != nil {
		txCount = len(batchData.Transactions)
	}

	ctx, span := t.tracer.Start(ctx, "BlockExecutor.CreateBlock",
		trace.WithAttributes(
			attribute.Int64("block.height", int64(height)),
			attribute.Int("tx.count", txCount),
		),
	)
	defer span.End()

	header, data, err := t.inner.CreateBlock(ctx, height, batchData)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}
	return header, data, nil
}

func (t *tracedBlockProducer) ApplyBlock(ctx context.Context, header types.Header, data *types.Data) (types.State, error) {
	ctx, span := t.tracer.Start(ctx, "BlockExecutor.ApplyBlock",
		trace.WithAttributes(
			attribute.Int64("block.height", int64(header.Height())),
			attribute.Int("tx.count", len(data.Txs)),
		),
	)
	defer span.End()

	state, err := t.inner.ApplyBlock(ctx, header, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return types.State{}, err
	}

	span.SetAttributes(
		attribute.String("state_root", hex.EncodeToString(state.AppHash)),
	)
	return state, nil
}

func (t *tracedBlockProducer) ValidateBlock(ctx context.Context, lastState types.State, header *types.SignedHeader, data *types.Data) error {
	ctx, span := t.tracer.Start(ctx, "BlockExecutor.ValidateBlock",
		trace.WithAttributes(
			attribute.Int64("block.height", int64(header.Height())),
		),
	)
	defer span.End()

	err := t.inner.ValidateBlock(ctx, lastState, header, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
