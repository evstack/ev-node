package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/core/execution"
)

// tracedExecutor wraps a core execution.Executor and records spans for key operations.
type tracedExecutor struct {
	inner  execution.Executor
	tracer trace.Tracer
}

// WithTracingExecutor decorates an Executor with OpenTelemetry spans.
func WithTracingExecutor(inner execution.Executor) execution.Executor {
	return &tracedExecutor{
		inner:  inner,
		tracer: otel.Tracer("ev-node/execution"),
	}
}

func (t *tracedExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.InitChain",
		trace.WithAttributes(
			attribute.String("chain.id", chainID),
			attribute.Int64("initial.height", int64(initialHeight)),
			attribute.Int64("genesis.time_unix", genesisTime.Unix()),
		),
	)
	defer span.End()

	stateRoot, err := t.inner.InitChain(ctx, genesisTime, initialHeight, chainID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return stateRoot, err
}

func (t *tracedExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.GetTxs")
	defer span.End()

	txs, err := t.inner.GetTxs(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(attribute.Int("tx.count", len(txs)))
	}
	return txs, err
}

func (t *tracedExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.ExecuteTxs",
		trace.WithAttributes(
			attribute.Int("tx.count", len(txs)),
			attribute.Int64("block.height", int64(blockHeight)),
			attribute.Int64("timestamp", timestamp.Unix()),
		),
	)
	defer span.End()

	stateRoot, err := t.inner.ExecuteTxs(ctx, txs, blockHeight, timestamp, prevStateRoot)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return stateRoot, err
}

func (t *tracedExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	ctx, span := t.tracer.Start(ctx, "Executor.SetFinal",
		trace.WithAttributes(attribute.Int64("block.height", int64(blockHeight))),
	)
	defer span.End()

	err := t.inner.SetFinal(ctx, blockHeight)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

// If the underlying executor implements HeightProvider, forward it while tracing.
func (t *tracedExecutor) GetLatestHeight(ctx context.Context) (uint64, error) {
	hp, ok := t.inner.(execution.HeightProvider)
	if !ok {
		return 0, nil
	}
	ctx, span := t.tracer.Start(ctx, "Executor.GetLatestHeight")
	defer span.End()

	height, err := hp.GetLatestHeight(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return height, err
}

// GetExecutionInfo forwards to the inner executor with tracing.
func (t *tracedExecutor) GetExecutionInfo(ctx context.Context) (execution.ExecutionInfo, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.GetExecutionInfo")
	defer span.End()

	info, err := t.inner.GetExecutionInfo(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(attribute.Int64("max_gas", int64(info.MaxGas)))
	}
	return info, err
}

// FilterTxs forwards to the inner executor with tracing.
func (t *tracedExecutor) FilterTxs(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) ([]execution.FilterStatus, error) {
	ctx, span := t.tracer.Start(ctx, "Executor.FilterTxs",
		trace.WithAttributes(
			attribute.Int("input_tx_count", len(txs)),
			attribute.Bool("has_force_included", hasForceIncludedTransaction),
			attribute.Int64("max_bytes", int64(maxBytes)),
			attribute.Int64("max_gas", int64(maxGas)),
		),
	)
	defer span.End()

	result, err := t.inner.FilterTxs(ctx, txs, maxBytes, maxGas, hasForceIncludedTransaction)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if result != nil {
		// Count statuses
		okCount, removeCount, postponeCount := 0, 0, 0
		for _, status := range result {
			switch status {
			case execution.FilterOK:
				okCount++
			case execution.FilterRemove:
				removeCount++
			case execution.FilterPostpone:
				postponeCount++
			}
		}
		span.SetAttributes(
			attribute.Int("ok_count", okCount),
			attribute.Int("remove_count", removeCount),
			attribute.Int("postpone_count", postponeCount),
		)
	}
	return result, err
}
