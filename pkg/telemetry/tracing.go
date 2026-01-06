package telemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/evstack/ev-node/pkg/config"
)

// InitTracing initializes a global OpenTelemetry tracer provider if enabled.
// Returns a shutdown function that should be called on process exit.
func InitTracing(ctx context.Context, cfg *config.InstrumentationConfig, logger zerolog.Logger) (func(context.Context) error, error) {
	if !cfg.IsTracingEnabled() {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.TracingServiceName),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	endpoint := cfg.TracingEndpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}
	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp exporter: %w", err)
	}

	ratio := clamp(cfg.TracingSampleRate, 0, 1)
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	// Configure tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	// Set global provider and W3C propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	logger.Info().Str("endpoint", cfg.TracingEndpoint).Str("service", cfg.TracingServiceName).Float64("sample_rate", ratio).Msg("OpenTelemetry tracing initialized")

	return tp.Shutdown, nil
}

func clamp(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}
