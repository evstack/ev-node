package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestExtractTraceContext_WithParentTrace(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := tp.Tracer("test")

	// create a parent span to get a valid trace context
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent")
	parentSpanCtx := parentSpan.SpanContext()
	parentSpan.End()

	// inject the parent context into headers
	headers := make(http.Header)
	otel.GetTextMapPropagator().Inject(parentCtx, propagation.HeaderCarrier(headers))

	var extractedSpanCtx trace.SpanContext
	handler := ExtractTraceContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// create a child span using the extracted context
		_, childSpan := tracer.Start(r.Context(), "child")
		extractedSpanCtx = childSpan.SpanContext()
		childSpan.End()
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header = headers
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, extractedSpanCtx.IsValid(), "child span should be valid")
	require.Equal(t, parentSpanCtx.TraceID(), extractedSpanCtx.TraceID(), "child should have same trace ID as parent")
}

func TestExtractTraceContext_WithoutParentTrace(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := tp.Tracer("test")

	var extractedSpanCtx trace.SpanContext
	handler := ExtractTraceContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// create a span - should be a root span since no parent headers
		_, span := tracer.Start(r.Context(), "root")
		extractedSpanCtx = span.SpanContext()
		span.End()
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// no trace headers
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, extractedSpanCtx.IsValid(), "span should be valid")

	// verify it's a root span by checking no parent exists in the recorded spans
	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.False(t, spans[0].Parent().IsValid(), "span should be a root span with no parent")
}
