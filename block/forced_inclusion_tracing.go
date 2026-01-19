package block

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ ForcedInclusionRetriever = (*tracedForcedInclusionRetriever)(nil)

type tracedForcedInclusionRetriever struct {
	inner  ForcedInclusionRetriever
	tracer trace.Tracer
}

// WithTracingForcedInclusionRetriever wraps a ForcedInclusionRetriever with OpenTelemetry tracing.
func WithTracingForcedInclusionRetriever(inner ForcedInclusionRetriever) ForcedInclusionRetriever {
	return &tracedForcedInclusionRetriever{
		inner:  inner,
		tracer: otel.Tracer("ev-node/forced-inclusion"),
	}
}

func (t *tracedForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error) {
	ctx, span := t.tracer.Start(ctx, "ForcedInclusionRetriever.RetrieveForcedIncludedTxs",
		trace.WithAttributes(
			attribute.Int64("da.height", int64(daHeight)),
		),
	)
	defer span.End()

	event, err := t.inner.RetrieveForcedIncludedTxs(ctx, daHeight)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return event, err
	}

	if event != nil {
		span.SetAttributes(
			attribute.Int64("event.start_da_height", int64(event.StartDaHeight)),
			attribute.Int64("event.end_da_height", int64(event.EndDaHeight)),
			attribute.Int("event.tx_count", len(event.Txs)),
		)
	}

	return event, nil
}

func (t *tracedForcedInclusionRetriever) Stop() {
	t.inner.Stop()
}
