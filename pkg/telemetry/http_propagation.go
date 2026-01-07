package telemetry

import (
	"net/http"

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
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // carries trace_id/span_id and sampling
		propagation.Baggage{},      // optional key/value context that can flow with the trace
	)
	return &http.Client{
		Transport: &otelRoundTripper{
			base:       base,
			propagator: prop,
		},
	}
}
