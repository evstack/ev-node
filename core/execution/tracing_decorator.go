package execution

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracedExecutor wraps an Executor and records spans for key operations.
type TracedExecutor struct {
	inner  Executor
	tracer trace.Tracer
}

// WithTracing decorates an Executor with OpenTelemetry spans.
func WithTracing(inner Executor) *TracedExecutor {
	return &TracedExecutor{
		inner:  inner,
		tracer: otel.Tracer("ev-node/execution"),
	}
}

func (t *TracedExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, uint64, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.InitChain",
		trace.WithAttributes(
			attribute.String("chain.id", chainID),
			attribute.Int64("initial.height", int64(initialHeight)),
			attribute.Int64("genesis.time_unix", genesisTime.Unix()),
		),
	)
	defer span.End()
	return t.inner.InitChain(ctx, genesisTime, initialHeight, chainID)
}

func (t *TracedExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.GetTxs")
	defer span.End()
	txs, err := t.inner.GetTxs(ctx)
	if err == nil {
		span.SetAttributes(attribute.Int("tx.count", len(txs)))
	} else {
		span.SetStatus(codes.Error, err.Error())
	}
	return txs, err
}

func (t *TracedExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, uint64, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.ExecuteTxs",
		trace.WithAttributes(
			attribute.Int("tx.count", len(txs)),
			attribute.Int64("block.height", int64(blockHeight)),
			attribute.Int64("timestamp", timestamp.Unix()),
		),
	)
	defer span.End()

	stateRoot, maxBytes, err := t.inner.ExecuteTxs(ctx, txs, blockHeight, timestamp, prevStateRoot)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
    
	return stateRoot, maxBytes, err
}

func (t *TracedExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	ctx, span := t.tracer.Start(ctx, "Executor.SetFinal",
		trace.WithAttributes(attribute.Int64("block.height", int64(blockHeight))),
	)
	defer span.End()

	err := t.inner.SetFinal(ctx, blockHeight)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (t *TracedExecutor) GetLatestHeight(ctx context.Context) (uint64, error) {
	hp, ok := t.inner.(HeightProvider)
	if !ok {
		return 0, nil
	}
	ctx, span := t.tracer.Start(ctx, "Executor.GetLatestHeight")
	defer span.End()

	h, err := hp.GetLatestHeight(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return h, err
}
