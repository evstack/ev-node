//go:build evm

package benchmark

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/jaeger"

	e2e "github.com/evstack/ev-node/test/e2e"
)

// richSpan captures the full parent-child hierarchy of a span for flowchart rendering.
type richSpan struct {
	traceID      string
	spanID       string
	parentSpanID string
	name         string
	hostName     string
	startTime    time.Time
	duration     time.Duration
}

// traceProvider abstracts trace collection so tests work against a local Jaeger
// instance or a remote VictoriaTraces deployment.
type traceProvider interface {
	collectSpans(ctx context.Context, serviceName string) ([]e2e.TraceSpan, error)
	tryCollectSpans(ctx context.Context, serviceName string) []e2e.TraceSpan
	// uiURL returns a link to view traces for the given service, or empty string if not available.
	uiURL(serviceName string) string
}

// richSpanCollector is an optional interface for providers that support
// collecting spans with full parent-child hierarchy information.
type richSpanCollector interface {
	collectRichSpans(ctx context.Context, serviceName string) ([]richSpan, error)
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

func (j *jaegerTraceProvider) uiURL(_ string) string {
	return j.node.External.QueryURL()
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
	queryURL   string
	t          testing.TB
	startTime  time.Time
	hostFilter string
}

func (v *victoriaTraceProvider) uiURL(serviceName string) string {
	return fmt.Sprintf("%s/select/jaeger?service=%s&start=%d&end=%d",
		strings.TrimRight(v.queryURL, "/"), serviceName,
		v.startTime.UnixMicro(), time.Now().UnixMicro())
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

func (v *victoriaTraceProvider) collectRichSpans(ctx context.Context, serviceName string) ([]richSpan, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	var spans []richSpan
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		var err error
		spans, err = v.fetchAllRichSpans(ctx, serviceName)
		if err != nil {
			return nil, err
		}
		if len(spans) > 0 {
			return spans, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for %s rich traces from %s: %w", serviceName, v.queryURL, ctx.Err())
		case <-ticker.C:
		}
	}
}

// fetchAllRichSpans is like fetchAllSpans but returns richSpan with full hierarchy info.
func (v *victoriaTraceProvider) fetchAllRichSpans(ctx context.Context, serviceName string) ([]richSpan, error) {
	end := time.Now()
	var query string
	if v.hostFilter != "" {
		query = fmt.Sprintf(`_stream:{resource_attr:service.name="%s", resource_attr:host.name="%s"}`, serviceName, v.hostFilter)
	} else {
		query = fmt.Sprintf(`_stream:{resource_attr:service.name="%s"}`, serviceName)
	}
	baseURL := strings.TrimRight(v.queryURL, "/")
	url := fmt.Sprintf("%s/select/logsql/query?query=%s&start=%s&end=%s",
		baseURL,
		neturl.QueryEscape(query),
		v.startTime.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching spans from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	var spans []richSpan
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var row logsqlRichSpan
		if err := json.Unmarshal(line, &row); err != nil {
			continue
		}
		if row.Name == "" {
			continue
		}
		ns, _ := strconv.ParseInt(row.Duration, 10, 64)
		startNs, _ := strconv.ParseInt(row.StartTimeUnixNano, 10, 64)
		spans = append(spans, richSpan{
			traceID:      row.TraceID,
			spanID:       row.SpanID,
			parentSpanID: row.ParentSpanID,
			name:         row.Name,
			hostName:     row.HostName,
			startTime:    time.Unix(0, startNs),
			duration:     time.Duration(ns) * time.Nanosecond,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading response stream: %w", err)
	}
	return spans, nil
}

// fetchAllSpans queries VictoriaTraces via the LogsQL API which supports
// streaming all results without the Jaeger API's 1000 trace limit.
func (v *victoriaTraceProvider) fetchAllSpans(ctx context.Context, serviceName string) ([]e2e.TraceSpan, error) {
	end := time.Now()
	var query string
	if v.hostFilter != "" {
		query = fmt.Sprintf(`_stream:{resource_attr:service.name="%s", resource_attr:host.name="%s"}`, serviceName, v.hostFilter)
	} else {
		query = fmt.Sprintf(`_stream:{resource_attr:service.name="%s"}`, serviceName)
	}
	baseURL := strings.TrimRight(v.queryURL, "/")
	url := fmt.Sprintf("%s/select/logsql/query?query=%s&start=%s&end=%s",
		baseURL,
		neturl.QueryEscape(query),
		v.startTime.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching spans from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	// LogsQL returns newline-delimited JSON, one span per line.
	var spans []e2e.TraceSpan
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var row logsqlSpan
		if err := json.Unmarshal(line, &row); err != nil {
			continue
		}
		if row.Name == "" {
			continue
		}
		spans = append(spans, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading response stream: %w", err)
	}

	v.t.Logf("fetched %d spans for %s in window [%s, %s]",
		len(spans), serviceName,
		v.startTime.Format(time.RFC3339), end.Format(time.RFC3339))
	return spans, nil
}

// logsqlSpan maps the fields returned by VictoriaTraces' LogsQL endpoint.
type logsqlSpan struct {
	Name     string `json:"name"`
	Duration string `json:"duration"` // nanoseconds as string
}

// logsqlRichSpan extends logsqlSpan with hierarchy fields for flowchart rendering.
type logsqlRichSpan struct {
	Name              string `json:"name"`
	Duration          string `json:"duration"`
	TraceID           string `json:"trace_id"`
	SpanID            string `json:"span_id"`
	ParentSpanID      string `json:"parent_span_id"`
	StartTimeUnixNano string `json:"start_time_unix_nano"`
	HostName          string `json:"resource_attr:'host.name"`
}

func (s logsqlSpan) SpanName() string { return s.Name }

func (s logsqlSpan) SpanDuration() time.Duration {
	ns, err := strconv.ParseInt(s.Duration, 10, 64)
	if err != nil {
		return 0
	}
	return time.Duration(ns) * time.Nanosecond
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

