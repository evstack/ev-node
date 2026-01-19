package syncing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/block/internal/da"
)

// forcedInclusionRetriever defines the interface for retrieving forced inclusion
// transactions from DA. This local interface is defined to avoid import cycles
// since block/ imports syncing/.
type forcedInclusionRetriever interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*da.ForcedInclusionEvent, error)
	Stop()
}

var _ forcedInclusionRetriever = (*tracedForcedInclusionRetriever)(nil)

type tracedForcedInclusionRetriever struct {
	inner  forcedInclusionRetriever
	tracer trace.Tracer
}

// withTracingForcedInclusionRetriever wraps a forcedInclusionRetriever with OpenTelemetry tracing.
func withTracingForcedInclusionRetriever(inner forcedInclusionRetriever) forcedInclusionRetriever {
	return &tracedForcedInclusionRetriever{
		inner:  inner,
		tracer: otel.Tracer("ev-node/forced-inclusion"),
	}
}

func (t *tracedForcedInclusionRetriever) RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*da.ForcedInclusionEvent, error) {
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
