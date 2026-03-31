//go:build evm

package benchmark

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"testing"
	"time"

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

// traceProvider abstracts trace collection so tests work against different
// tracing backends (e.g. VictoriaTraces).
type traceProvider interface {
	collectSpans(ctx context.Context, serviceName string) ([]e2e.TraceSpan, error)
	tryCollectSpans(ctx context.Context, serviceName string) []e2e.TraceSpan
	// uiURL returns a link to view traces for the given service, or empty string if not available.
	uiURL(serviceName string) string
	// resetStartTime sets the trace collection window start to now.
	resetStartTime()
}

// richSpanCollector is an optional interface for providers that support
// collecting spans with full parent-child hierarchy information.
type richSpanCollector interface {
	collectRichSpans(ctx context.Context, serviceName string) ([]richSpan, error)
}

// resourceAttrCollector is an optional interface for providers that can
// extract OTEL resource attributes from trace spans.
type resourceAttrCollector interface {
	fetchResourceAttrs(ctx context.Context, serviceName string) *resourceAttrs
}

// victoriaTraceProvider collects spans from a VictoriaTraces instance via its
// LogsQL streaming API.
type victoriaTraceProvider struct {
	queryURL  string
	t         testing.TB
	startTime time.Time
}

func (v *victoriaTraceProvider) resetStartTime() {
	v.startTime = time.Now()
}

func (v *victoriaTraceProvider) uiURL(serviceName string) string {
	query := fmt.Sprintf(`_stream:{resource_attr:service.name="%s"}`, serviceName)
	return fmt.Sprintf("%s/select/vmui/#/query?query=%s&start=%s&end=%s",
		strings.TrimRight(v.queryURL, "/"),
		neturl.QueryEscape(query),
		v.startTime.Format(time.RFC3339),
		time.Now().Format(time.RFC3339))
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

// fetchLogStream opens a streaming LogsQL query against VictoriaTraces and
// returns a scanner over the newline-delimited JSON response. The caller must
// close the returned io.Closer when done.
func (v *victoriaTraceProvider) fetchLogStream(ctx context.Context, serviceName string) (*bufio.Scanner, io.Closer, error) {
	end := time.Now()
	query := fmt.Sprintf(`_stream:{resource_attr:service.name="%s"}`, serviceName)
	baseURL := strings.TrimRight(v.queryURL, "/")
	url := fmt.Sprintf("%s/select/logsql/query?query=%s&start=%s&end=%s",
		baseURL,
		neturl.QueryEscape(query),
		v.startTime.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching spans from %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return scanner, resp.Body, nil
}

// fetchAllRichSpans returns richSpan with full hierarchy info for flowchart rendering.
func (v *victoriaTraceProvider) fetchAllRichSpans(ctx context.Context, serviceName string) ([]richSpan, error) {
	scanner, body, err := v.fetchLogStream(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var spans []richSpan
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
			hostName:     extractHostName(line),
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
	scanner, body, err := v.fetchLogStream(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var spans []e2e.TraceSpan
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
		v.startTime.Format(time.RFC3339), time.Now().Format(time.RFC3339))
	return spans, nil
}

// fetchResourceAttrs queries a single span and extracts OTEL resource attributes
// from it. Uses limit=1 to avoid streaming the full span set on long-lived
// instances. Returns nil if no spans are available.
func (v *victoriaTraceProvider) fetchResourceAttrs(ctx context.Context, serviceName string) *resourceAttrs {
	end := time.Now()
	query := fmt.Sprintf(`_stream:{resource_attr:service.name="%s"}`, serviceName)
	baseURL := strings.TrimRight(v.queryURL, "/")
	url := fmt.Sprintf("%s/select/logsql/query?query=%s&start=%s&end=%s&limit=1",
		baseURL,
		neturl.QueryEscape(query),
		v.startTime.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		v.t.Logf("warning: failed to create resource attrs request: %v", err)
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		v.t.Logf("warning: failed to fetch resource attrs: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		v.t.Logf("warning: unexpected status %d fetching resource attrs", resp.StatusCode)
		return nil
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		attrs := extractResourceAttrs(line)
		return &attrs
	}
	return nil
}

// logsqlSpan maps the fields returned by VictoriaTraces' LogsQL endpoint.
type logsqlSpan struct {
	Name     string `json:"name"`
	Duration string `json:"duration"` // nanoseconds as string
}

// logsqlRichSpan extends logsqlSpan with hierarchy fields for flowchart rendering.
type logsqlRichSpan struct {
	logsqlSpan
	TraceID           string `json:"trace_id"`
	SpanID            string `json:"span_id"`
	ParentSpanID      string `json:"parent_span_id"`
	StartTimeUnixNano string `json:"start_time_unix_nano"`
}

// extractHostName pulls the host.name value from a raw JSON line.
// VictoriaTraces encodes it as resource_attr:'host.name which Go's
// struct tags can't match due to the single quote.
func extractHostName(line []byte) string {
	return extractResourceAttr(line, "host.name")
}

// extractResourceAttr pulls a specific resource attribute from a raw LogsQL JSON line.
// VictoriaTraces encodes resource attributes with keys like resource_attr:'host.name
// which can't be mapped via struct tags.
func extractResourceAttr(line []byte, attr string) string {
	var raw map[string]string
	if err := json.Unmarshal(line, &raw); err != nil {
		return ""
	}
	for k, v := range raw {
		if strings.Contains(k, attr) {
			return v
		}
	}
	return ""
}

// resourceAttrs holds OTEL resource attributes extracted from trace spans.
type resourceAttrs struct {
	HostName    string `json:"host_name,omitempty"`
	HostCPU     string `json:"host_cpu,omitempty"`
	HostMemory  string `json:"host_memory,omitempty"`
	HostType    string `json:"host_type,omitempty"`
	OSName      string `json:"os_name,omitempty"`
	OSVersion   string `json:"os_version,omitempty"`
	ServiceType string `json:"service_type,omitempty"`
}

// extractResourceAttrs pulls all known OTEL resource attributes from a raw LogsQL JSON line.
func extractResourceAttrs(line []byte) resourceAttrs {
	return resourceAttrs{
		HostName:    extractResourceAttr(line, "host.name"),
		HostCPU:     extractResourceAttr(line, "host.cpu"),
		HostMemory:  extractResourceAttr(line, "host.memory"),
		HostType:    extractResourceAttr(line, "host.type"),
		OSName:      extractResourceAttr(line, "os.name"),
		OSVersion:   extractResourceAttr(line, "os.version"),
		ServiceType: extractResourceAttr(line, "service.type"),
	}
}

func (s logsqlSpan) SpanName() string { return s.Name }

func (s logsqlSpan) SpanDuration() time.Duration {
	ns, err := strconv.ParseInt(s.Duration, 10, 64)
	if err != nil {
		return 0
	}
	return time.Duration(ns) * time.Nanosecond
}

