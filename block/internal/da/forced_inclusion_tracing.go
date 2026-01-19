package da

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ForcedInclusionRetrieverAPI defines the interface for retrieving forced inclusion transactions from DA.
type ForcedInclusionRetrieverAPI interface {
	RetrieveForcedIncludedTxs(ctx context.Context, daHeight uint64) (*ForcedInclusionEvent, error)
	Stop()
}

var _ ForcedInclusionRetrieverAPI = (*tracedForcedInclusionRetriever)(nil)

type tracedForcedInclusionRetriever struct {
	inner  ForcedInclusionRetrieverAPI
	tracer trace.Tracer
}

func withTracingForcedInclusionRetriever(inner ForcedInclusionRetrieverAPI) ForcedInclusionRetrieverAPI {
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
