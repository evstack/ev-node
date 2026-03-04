//go:build evm

package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/jaeger"

	e2e "github.com/evstack/ev-node/test/e2e"
)

// traceProvider abstracts trace collection so tests work against a local Jaeger
// instance or a remote VictoriaTraces deployment.
type traceProvider interface {
	collectSpans(ctx context.Context, serviceName string) ([]e2e.TraceSpan, error)
	tryCollectSpans(ctx context.Context, serviceName string) []e2e.TraceSpan
}

// jaegerTraceProvider collects spans from a locally-provisioned Jaeger node.
type jaegerTraceProvider struct {
	node *jaeger.Node
	t    testing.TB
}

func (j *jaegerTraceProvider) collectSpans(ctx context.Context, serviceName string) ([]e2e.TraceSpan, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	ok, err := j.node.External.WaitForTraces(ctx, serviceName, 1, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error waiting for %s traces; UI: %s: %w", serviceName, j.node.External.QueryURL(), err)
	}
	if !ok {
		return nil, fmt.Errorf("expected at least one trace from %s; UI: %s", serviceName, j.node.External.QueryURL())
	}

	traces, err := j.node.External.Traces(ctx, serviceName, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s traces: %w", serviceName, err)
	}
	return toTraceSpans(extractSpansFromTraces(traces)), nil
}

func (j *jaegerTraceProvider) tryCollectSpans(ctx context.Context, serviceName string) []e2e.TraceSpan {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	ok, err := j.node.External.WaitForTraces(ctx, serviceName, 1, 2*time.Second)
	if err != nil || !ok {
		j.t.Logf("warning: could not collect %s traces (err=%v, ok=%v)", serviceName, err, ok)
		return nil
	}

	traces, err := j.node.External.Traces(ctx, serviceName, 10000)
	if err != nil {
		j.t.Logf("warning: failed to fetch %s traces: %v", serviceName, err)
		return nil
	}

	spans := toTraceSpans(extractSpansFromTraces(traces))
	j.t.Logf("collected %d %s spans", len(spans), serviceName)
	return spans
}

// victoriaTraceProvider collects spans from a VictoriaTraces instance via its
// Jaeger-compatible HTTP API.
type victoriaTraceProvider struct {
	queryURL  string
	t         testing.TB
	startTime time.Time
}

func (v *victoriaTraceProvider) collectSpans(ctx context.Context, serviceName string) ([]e2e.TraceSpan, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	var spans []e2e.TraceSpan
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		var err error
		spans, err = v.fetchAllSpans(ctx, serviceName)
		if err != nil {
			return nil, err
		}
		if len(spans) > 0 {
			return spans, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for %s traces from %s: %w", serviceName, v.queryURL, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (v *victoriaTraceProvider) tryCollectSpans(ctx context.Context, serviceName string) []e2e.TraceSpan {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var spans []e2e.TraceSpan
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		var err error
		spans, err = v.fetchAllSpans(ctx, serviceName)
		if err != nil {
			v.t.Logf("warning: failed to fetch %s traces: %v", serviceName, err)
			return nil
		}
		if len(spans) > 0 {
			v.t.Logf("collected %d %s spans", len(spans), serviceName)
			return spans
		}

		select {
		case <-ctx.Done():
			v.t.Logf("warning: timed out waiting for %s traces from %s", serviceName, v.queryURL)
			return nil
		case <-ticker.C:
		}
	}
}

const victoriaPageSize = 1000

// fetchAllSpans paginates through VictoriaTraces using offset until all traces
// in the [startTime, now] window are fetched.
func (v *victoriaTraceProvider) fetchAllSpans(ctx context.Context, serviceName string) ([]e2e.TraceSpan, error) {
	end := time.Now()
	var allSpans []e2e.TraceSpan
	offset := 0

	for {
		traces, err := v.fetchTraces(ctx, serviceName, victoriaPageSize, offset, v.startTime, end)
		if err != nil {
			return nil, err
		}

		batch := toTraceSpans(extractSpansFromTraces(traces))
		allSpans = append(allSpans, batch...)

		if len(traces) < victoriaPageSize {
			break
		}
		offset += len(traces)
	}

	v.t.Logf("fetched %d spans for %s in window [%s, %s]",
		len(allSpans), serviceName,
		v.startTime.Format(time.RFC3339), end.Format(time.RFC3339))
	return allSpans, nil
}

// jaegerAPIResponse is the envelope returned by Jaeger-compatible query APIs.
type jaegerAPIResponse struct {
	Data []any `json:"data"`
}

func (v *victoriaTraceProvider) fetchTraces(ctx context.Context, serviceName string, limit, offset int, start, end time.Time) ([]any, error) {
	url := fmt.Sprintf("%s/select/jaeger/api/traces?service=%s&limit=%d&offset=%d&start=%d&end=%d",
		strings.TrimRight(v.queryURL, "/"), serviceName, limit, offset,
		start.UnixMicro(), end.UnixMicro())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching traces from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	var apiResp jaegerAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", url, err)
	}

	return apiResp.Data, nil
}

// jaegerSpan holds the fields we extract from Jaeger's untyped JSON response.
type jaegerSpan struct {
	operationName string
	duration      float64 // microseconds
}

func (j jaegerSpan) SpanName() string            { return j.operationName }
func (j jaegerSpan) SpanDuration() time.Duration { return time.Duration(j.duration) * time.Microsecond }

// extractSpansFromTraces walks Jaeger's []any response and pulls out span operation names and durations.
func extractSpansFromTraces(traces []any) []jaegerSpan {
	var out []jaegerSpan
	for _, traceEntry := range traces {
		traceMap, ok := traceEntry.(map[string]any)
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
