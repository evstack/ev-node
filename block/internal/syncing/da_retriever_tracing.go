package syncing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/block/internal/common"
)

var _ DARetriever = (*tracedDARetriever)(nil)

// tracedDARetriever wraps a DARetriever with OpenTelemetry tracing.
type tracedDARetriever struct {
	inner  DARetriever
	tracer trace.Tracer
}

// WithTracingDARetriever wraps a DARetriever with OpenTelemetry tracing.
func WithTracingDARetriever(inner DARetriever) DARetriever {
	return &tracedDARetriever{
		inner:  inner,
		tracer: otel.Tracer("ev-node/da-retriever"),
	}
}

func (t *tracedDARetriever) RetrieveFromDA(ctx context.Context, daHeight uint64) ([]common.DAHeightEvent, error) {
	ctx, span := t.tracer.Start(ctx, "DARetriever.RetrieveFromDA",
		trace.WithAttributes(
			attribute.Int64("da.height", int64(daHeight)),
		),
	)
	defer span.End()

	events, err := t.inner.RetrieveFromDA(ctx, daHeight)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return events, err
	}

	span.SetAttributes(attribute.Int("event.count", len(events)))

	// add block heights from events
	if len(events) > 0 {
		heights := make([]int64, len(events))
		for i, event := range events {
			heights[i] = int64(event.Header.Height())
		}
		span.SetAttributes(attribute.Int64Slice("block.heights", heights))
	}

	return events, nil
}

func (t *tracedDARetriever) ProcessBlobs(ctx context.Context, blobs [][]byte, daHeight uint64) []common.DAHeightEvent {
	ctx, span := t.tracer.Start(ctx, "DARetriever.ProcessBlobs",
		trace.WithAttributes(
			attribute.Int64("da.height", int64(daHeight)),
			attribute.Int("blob.count", len(blobs)),
		),
	)
	defer span.End()

	events := t.inner.ProcessBlobs(ctx, blobs, daHeight)
	span.SetAttributes(attribute.Int("event.count", len(events)))

	return events
}
