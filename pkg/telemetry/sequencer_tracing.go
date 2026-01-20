package telemetry

import (
	"context"
	"encoding/hex"

	coresequencer "github.com/evstack/ev-node/core/sequencer"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ coresequencer.Sequencer = (*tracedSequencer)(nil)

// tracedSequencer decorates a Sequencer with OpenTelemetry spans.
type tracedSequencer struct {
	inner  coresequencer.Sequencer
	tracer trace.Tracer
}

// WithTracingSequencer decorates the provided Sequencer with tracing spans.
func WithTracingSequencer(inner coresequencer.Sequencer) coresequencer.Sequencer {
	return &tracedSequencer{
		inner:  inner,
		tracer: otel.Tracer("ev-node/sequencer"),
	}
}

func (t *tracedSequencer) SubmitBatchTxs(ctx context.Context, req coresequencer.SubmitBatchTxsRequest) (*coresequencer.SubmitBatchTxsResponse, error) {
	txCount := 0
	totalBytes := 0
	if req.Batch != nil {
		txCount = len(req.Batch.Transactions)
		for _, tx := range req.Batch.Transactions {
			totalBytes += len(tx)
		}
	}

	ctx, span := t.tracer.Start(ctx, "Sequencer.SubmitBatchTxs",
		trace.WithAttributes(
			attribute.String("chain.id", hex.EncodeToString(req.Id)),
			attribute.Int("tx.count", txCount),
			attribute.Int("batch.size_bytes", totalBytes),
		),
	)
	defer span.End()

	res, err := t.inner.SubmitBatchTxs(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return res, nil
}

func (t *tracedSequencer) GetNextBatch(ctx context.Context, req coresequencer.GetNextBatchRequest) (*coresequencer.GetNextBatchResponse, error) {
	ctx, span := t.tracer.Start(ctx, "Sequencer.GetNextBatch",
		trace.WithAttributes(
			attribute.String("chain.id", hex.EncodeToString(req.Id)),
			attribute.Int64("max_bytes", int64(req.MaxBytes)),
		),
	)
	defer span.End()

	res, err := t.inner.GetNextBatch(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if res.Batch != nil {
		txCount := len(res.Batch.Transactions)
		forcedCount := 0
		for _, forced := range res.Batch.ForceIncludedMask {
			if forced {
				forcedCount++
			}
		}
		totalBytes := 0
		for _, tx := range res.Batch.Transactions {
			totalBytes += len(tx)
		}

		span.SetAttributes(
			attribute.Int("tx.count", txCount),
			attribute.Int("forced_inclusion.count", forcedCount),
			attribute.Int("batch.size_bytes", totalBytes),
			attribute.Int64("timestamp", res.Timestamp.Unix()),
		)
	}

	return res, nil
}

func (t *tracedSequencer) VerifyBatch(ctx context.Context, req coresequencer.VerifyBatchRequest) (*coresequencer.VerifyBatchResponse, error) {
	ctx, span := t.tracer.Start(ctx, "Sequencer.VerifyBatch",
		trace.WithAttributes(
			attribute.String("chain.id", hex.EncodeToString(req.Id)),
			attribute.Int("batch_data.count", len(req.BatchData)),
		),
	)
	defer span.End()

	res, err := t.inner.VerifyBatch(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Bool("verified", res.Status),
	)

	return res, nil
}

func (t *tracedSequencer) SetDAHeight(height uint64) {
	t.inner.SetDAHeight(height)
}

func (t *tracedSequencer) GetDAHeight() uint64 {
	return t.inner.GetDAHeight()
}
