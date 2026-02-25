//go:build evm

package benchmark

import (
	"testing"
	"time"

	e2e "github.com/evstack/ev-node/test/e2e"
	"github.com/stretchr/testify/require"
)

// jaegerSpan holds the fields we extract from Jaeger's untyped JSON response.
type jaegerSpan struct {
	operationName string
	duration      float64 // microseconds
}

func (j jaegerSpan) SpanName() string           { return j.operationName }
func (j jaegerSpan) SpanDuration() time.Duration { return time.Duration(j.duration) * time.Microsecond }

// extractSpansFromTraces walks Jaeger's []any response and pulls out span operation names and durations.
func extractSpansFromTraces(traces []any) []jaegerSpan {
	var out []jaegerSpan
	for _, t := range traces {
		traceMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		spans, ok := traceMap["spans"].([]any)
		if !ok {
			continue
		}
		for _, s := range spans {
			spanMap, ok := s.(map[string]any)
			if !ok {
				continue
			}
			name, _ := spanMap["operationName"].(string)
			dur, _ := spanMap["duration"].(float64)
			if name != "" {
				out = append(out, jaegerSpan{operationName: name, duration: dur})
			}
		}
	}
	return out
}

func toTraceSpans(spans []jaegerSpan) []e2e.TraceSpan {
	out := make([]e2e.TraceSpan, len(spans))
	for i, s := range spans {
		out[i] = s
	}
	return out
}

// assertSpanNames verifies that all expected span names appear in the trace data.
func assertSpanNames(t testing.TB, spans []e2e.TraceSpan, expected []string, label string) {
	t.Helper()
	opNames := make(map[string]struct{}, len(spans))
	for _, span := range spans {
		opNames[span.SpanName()] = struct{}{}
	}
	for _, name := range expected {
		require.Contains(t, opNames, name, "expected span %q not found in %s traces", name, label)
	}
}
