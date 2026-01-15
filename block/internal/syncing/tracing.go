package syncing

import (
	"context"
	"encoding/hex"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/types"
)

var _ BlockSyncer = (*tracedBlockSyncer)(nil)

// tracedBlockSyncer decorates a BlockSyncer with OpenTelemetry spans.
type tracedBlockSyncer struct {
	inner  BlockSyncer
	tracer trace.Tracer
}

// WithTracingBlockSyncer decorates the provided BlockSyncer with tracing spans.
func WithTracingBlockSyncer(inner BlockSyncer) BlockSyncer {
	return &tracedBlockSyncer{
		inner:  inner,
		tracer: otel.Tracer("ev-node/block-syncer"),
	}
}

func (t *tracedBlockSyncer) TrySyncNextBlock(ctx context.Context, event *common.DAHeightEvent) error {
	ctx, span := t.tracer.Start(ctx, "BlockSyncer.SyncBlock",
		trace.WithAttributes(
			attribute.Int64("block.height", int64(event.Header.Height())),
			attribute.Int64("da.height", int64(event.DaHeight)),
			attribute.String("source", string(event.Source)),
		),
	)
	defer span.End()

	err := t.inner.TrySyncNextBlock(ctx, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (t *tracedBlockSyncer) ApplyBlock(ctx context.Context, header types.Header, data *types.Data, currentState types.State) (types.State, error) {
	ctx, span := t.tracer.Start(ctx, "BlockSyncer.ApplyBlock",
		trace.WithAttributes(
			attribute.Int64("block.height", int64(header.Height())),
			attribute.Int("tx.count", len(data.Txs)),
		),
	)
	defer span.End()

	state, err := t.inner.ApplyBlock(ctx, header, data, currentState)
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

func (t *tracedBlockSyncer) ValidateBlock(ctx context.Context, currState types.State, data *types.Data, header *types.SignedHeader) error {
	ctx, span := t.tracer.Start(ctx, "BlockSyncer.ValidateBlock",
		trace.WithAttributes(
			attribute.Int64("block.height", int64(header.Height())),
		),
	)
	defer span.End()

	err := t.inner.ValidateBlock(ctx, currState, data, header)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (t *tracedBlockSyncer) VerifyForcedInclusionTxs(ctx context.Context, currentState types.State, data *types.Data) error {
	ctx, span := t.tracer.Start(ctx, "BlockSyncer.VerifyForcedInclusion",
		trace.WithAttributes(
			attribute.Int64("block.height", int64(data.Height())),
			attribute.Int64("da.height", int64(currentState.DAHeight)),
		),
	)
	defer span.End()

	err := t.inner.VerifyForcedInclusionTxs(ctx, currentState, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
