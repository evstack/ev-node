package telemetry

import (
	"net/http"

	"github.com/evstack/ev-node/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// otelRoundTripper injects W3C Trace Context headers into outbound HTTP requests
// using the request's context. It does not create spans; propagation only.
type otelRoundTripper struct {
	base       http.RoundTripper
	propagator propagation.TextMapPropagator
}

func (t *otelRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.propagator != nil {
		t.propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
	}
	return t.base.RoundTrip(req)
}

// NewPropagatingHTTPClient returns an http.Client whose Transport injects
// W3C Trace Context headers (traceparent, tracestate) from the request context.
// If base is nil, http.DefaultTransport is used.
func NewPropagatingHTTPClient(base http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: &otelRoundTripper{
			base:       base,
			propagator: otel.GetTextMapPropagator(),
		},
	}
}

// RPCHTTPClientFromConfig returns an HTTP client that injects W3C trace context
// headers when tracing is enabled in the provided config. Returns nil if
// tracing is disabled so callers can decide how to build RPC client options.
func RPCHTTPClientFromConfig(cfg *config.InstrumentationConfig) *http.Client {
	if cfg == nil || !cfg.IsTracingEnabled() {
		return nil
	}
	return NewPropagatingHTTPClient(http.DefaultTransport)
}
