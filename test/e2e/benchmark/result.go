//go:build evm

package benchmark

import (
	"fmt"
	"sort"
	"testing"
	"time"

	e2e "github.com/evstack/ev-node/test/e2e"
)

// traceResult holds the collected spans from ev-node and (optionally) ev-reth.
type traceResult struct {
	evNode []e2e.TraceSpan
	evReth []e2e.TraceSpan
}

// allSpans returns ev-node and ev-reth spans concatenated.
func (tr *traceResult) allSpans() []e2e.TraceSpan {
	return append(tr.evNode, tr.evReth...)
}

// benchmarkResult collects all metrics from a single benchmark run and
// produces the final set of entries for the result writer.
type benchmarkResult struct {
	prefix  string
	bm      *blockMetrics
	summary *blockMetricsSummary
	traces  *traceResult
}

func newBenchmarkResult(prefix string, bm *blockMetrics, traces *traceResult) *benchmarkResult {
	return &benchmarkResult{
		prefix:  prefix,
		bm:      bm,
		summary: bm.summarize(),
		traces:  traces,
	}
}

// log prints all key benchmark metrics to the test log.
func (r *benchmarkResult) log(t testing.TB, wallClock time.Duration) {
	r.summary.log(t, r.bm.StartBlock, r.bm.EndBlock, r.bm.TotalBlockCount, r.bm.BlockCount, wallClock)

	if overhead, ok := evNodeOverhead(r.traces.evNode); ok {
		t.Logf("ev-node overhead: %.1f%%", overhead)
	}
	if ggas, ok := rethExecutionRate(r.traces.evNode, r.bm.TotalGasUsed); ok {
		t.Logf("ev-reth execution rate: %.3f GGas/s", ggas)
	}
	for _, e := range engineSpanEntries(r.prefix, r.traces.evNode) {
		if e.Name == r.prefix+" - ProduceBlock avg" {
			t.Logf("ProduceBlock avg: %.0fms", e.Value)
		}
	}
	if r.summary.AchievedMGas > 0 {
		t.Logf("seconds_per_gigagas: %.4f", 1000.0/r.summary.AchievedMGas)
	}
}

// entries returns all benchmark metrics as result writer entries.
func (r *benchmarkResult) entries() []entry {
	var out []entry

	out = append(out, r.summary.entries(r.prefix)...)

	if overhead, ok := evNodeOverhead(r.traces.evNode); ok {
		out = append(out, entry{Name: r.prefix + " - ev-node overhead", Unit: "%", Value: overhead})
	}

	if ggas, ok := rethExecutionRate(r.traces.evNode, r.bm.TotalGasUsed); ok {
		out = append(out, entry{Name: r.prefix + " - ev-reth GGas/s", Unit: "GGas/s", Value: ggas})
	}

	out = append(out, engineSpanEntries(r.prefix, r.traces.evNode)...)

	if r.summary.AchievedMGas > 0 {
		out = append(out, entry{
			Name:  r.prefix + " - seconds_per_gigagas",
			Unit:  "s/Ggas",
			Value: 1000.0 / r.summary.AchievedMGas,
		})
	}

	out = append(out, spanAvgEntries(r.prefix, r.traces.allSpans())...)

	return out
}

// spanAvgEntries aggregates trace spans into per-operation avg duration entries.
func spanAvgEntries(prefix string, spans []e2e.TraceSpan) []entry {
	m := e2e.AggregateSpanStats(spans)
	if len(m) == 0 {
		return nil
	}

	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []entry
	for _, name := range names {
		s := m[name]
		avg := float64(s.Total.Microseconds()) / float64(s.Count)
		out = append(out, entry{
			Name:  fmt.Sprintf("%s - %s (avg)", prefix, name),
			Unit:  "us",
			Value: avg,
		})
	}
	return out
}
