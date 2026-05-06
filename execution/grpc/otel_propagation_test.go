package grpc

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/evstack/ev-node/core/execution"
)

func setupTracer(t *testing.T) (*tracetest.SpanRecorder, func()) {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	oldTP := otel.GetTracerProvider()
	oldProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return rec, func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(oldTP)
		otel.SetTextMapPropagator(oldProp)
	}
}

func TestInboundMetadataCreatesChildSpanWithSameTraceID(t *testing.T) {
	rec, cleanup := setupTracer(t)
	defer cleanup()

	tracer := otel.Tracer("test")
	parentCtx, parent := tracer.Start(context.Background(), "parent")
	defer parent.End()
	parentTraceID := parent.SpanContext().TraceID()

	mockExec := &mockExecutor{getTxsFunc: func(ctx context.Context) ([][]byte, error) {
		_, span := tracer.Start(ctx, "server-child")
		span.End()
		return [][]byte{}, nil
	}}

	handler := NewExecutorServiceHandler(mockExec)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	client, err := NewClient(ts.URL)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.GetTxs(parentCtx)
	if err != nil {
		t.Fatalf("GetTxs failed: %v", err)
	}

	var found bool
	for _, s := range rec.Ended() {
		if s.Name() == "server-child" {
			found = true
			if s.SpanContext().TraceID() != parentTraceID {
				t.Fatalf("trace id mismatch: got %s want %s", s.SpanContext().TraceID(), parentTraceID)
			}
		}
	}
	if !found {
		t.Fatalf("server-child span not found")
	}
}

func TestOutboundGRPCCallCarriesTraceparentMetadata(t *testing.T) {
	rec, cleanup := setupTracer(t)
	_ = rec
	defer cleanup()

	tracer := otel.Tracer("test")
	ctx, parent := tracer.Start(context.Background(), "parent")
	defer parent.End()

	gotTraceparent := ""
	captureHeader := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			gotTraceparent = req.Header().Get("traceparent")
			return next(ctx, req)
		}
	})

	mockExec := &mockExecutor{}
	handler := NewExecutorServiceHandler(mockExec, connect.WithInterceptors(captureHeader))
	ts := httptest.NewServer(handler)
	defer ts.Close()

	client, err := NewClient(ts.URL)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if _, err = client.GetTxs(ctx); err != nil {
		t.Fatalf("GetTxs failed: %v", err)
	}
	if gotTraceparent == "" {
		t.Fatalf("expected traceparent metadata to be propagated")
	}
}

func TestOutboundGRPCCallCarriesPropagationHeaders(t *testing.T) {
	rec, cleanup := setupTracer(t)
	_ = rec
	defer cleanup()

	tracer := otel.Tracer("test")
	ctx, parent := tracer.Start(context.Background(), "parent")
	defer parent.End()
	member, err := baggage.NewMember("tenant", "alpha")
	if err != nil {
		t.Fatalf("failed to create baggage member: %v", err)
	}
	bg, err := baggage.New(member)
	if err != nil {
		t.Fatalf("failed to create baggage: %v", err)
	}
	ctx = baggage.ContextWithBaggage(ctx, bg)

	var gotTraceparent string
	var gotBaggage string
	captureHeader := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			gotTraceparent = req.Header().Get("traceparent")
			gotBaggage = req.Header().Get("baggage")
			return next(ctx, req)
		}
	})

	mockExec := &mockExecutor{}
	handler := NewExecutorServiceHandler(mockExec, connect.WithInterceptors(captureHeader))
	ts := httptest.NewServer(handler)
	defer ts.Close()

	client, err := NewClient(ts.URL)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if _, err = client.GetTxs(ctx); err != nil {
		t.Fatalf("GetTxs failed: %v", err)
	}

	if gotTraceparent == "" {
		t.Fatalf("expected traceparent metadata to be propagated")
	}
	if gotBaggage == "" {
		t.Fatalf("expected baggage metadata to be propagated")
	}
}

func TestEndToEndParentChildAcrossServerClientHop(t *testing.T) {
	rec, cleanup := setupTracer(t)
	defer cleanup()

	tracer := otel.Tracer("test")
	var midSpan trace.Span

	downstreamExec := &mockExecutor{getExecutionInfoFunc: func(ctx context.Context) (executionInfo execution.ExecutionInfo, err error) {
		_, span := tracer.Start(ctx, "downstream-child")
		span.End()
		return execution.ExecutionInfo{MaxGas: 1}, nil
	}}
	downstreamHandler := NewExecutorServiceHandler(downstreamExec)
	downstreamSrv := httptest.NewServer(downstreamHandler)
	defer downstreamSrv.Close()
	downstreamClient, err := NewClient(downstreamSrv.URL)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	upstreamExec := &mockExecutor{getTxsFunc: func(ctx context.Context) ([][]byte, error) {
		ctx, span := tracer.Start(ctx, "upstream-mid")
		midSpan = span
		defer span.End()
		_, err := downstreamClient.GetExecutionInfo(ctx)
		if err != nil {
			return nil, err
		}
		return [][]byte{}, nil
	}}
	upstreamHandler := NewExecutorServiceHandler(upstreamExec)
	upstreamSrv := httptest.NewServer(upstreamHandler)
	defer upstreamSrv.Close()

	client, err := NewClient(upstreamSrv.URL)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	rootCtx, root := tracer.Start(context.Background(), "root")
	defer root.End()
	if _, err := client.GetTxs(rootCtx); err != nil {
		t.Fatalf("GetTxs failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	rootTraceID := root.SpanContext().TraceID()
	if midSpan.SpanContext().TraceID() != rootTraceID {
		t.Fatalf("mid span trace id mismatch")
	}
	var found bool
	for _, s := range rec.Ended() {
		if s.Name() == "downstream-child" {
			found = true
			if s.SpanContext().TraceID() != rootTraceID {
				t.Fatalf("downstream trace id mismatch")
			}
		}
	}
	if !found {
		t.Fatalf("downstream-child span not found")
	}
}
