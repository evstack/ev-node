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

	// rich spans include parent-child hierarchy for flowchart rendering.
	// empty when the trace provider doesn't support rich span collection.
	evNodeRich []richSpan
	evRethRich []richSpan

	// resource attributes extracted from trace spans (OTEL_RESOURCE_ATTRIBUTES).
	evNodeAttrs *resourceAttrs
	evRethAttrs *resourceAttrs
}

// displayFlowcharts renders ASCII flowcharts from rich spans. Falls back to
// flat trace reports when rich spans are not available.
func (tr *traceResult) displayFlowcharts(t testing.TB, serviceName string) {
	if len(tr.evNodeRich) > 0 {
		printFlowcharts(t, tr.evNodeRich)
		printAggregateFlowcharts(t, tr.evNodeRich)
	} else {
		e2e.PrintTraceReport(t, serviceName, tr.evNode)
	}

	if len(tr.evRethRich) > 0 {
		t.Logf("ev-reth: collected %d rich spans", len(tr.evRethRich))
		printAggregateFlowcharts(t, tr.evRethRich)
	} else if len(tr.evReth) > 0 {
		e2e.PrintTraceReport(t, "ev-reth", tr.evReth)
	}
}

// allSpans returns ev-node and ev-reth spans concatenated into a new slice.
func (tr *traceResult) allSpans() []e2e.TraceSpan {
	out := make([]e2e.TraceSpan, 0, len(tr.evNode)+len(tr.evReth))
	out = append(out, tr.evNode...)
	out = append(out, tr.evReth...)
	return out
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
	if stats := e2e.AggregateSpanStats(r.traces.evNode); stats != nil {
		if pb, ok := stats[spanProduceBlock]; ok && pb.Count > 0 {
			avg := pb.Total / time.Duration(pb.Count)
			t.Logf("ProduceBlock avg: %dms", avg.Milliseconds())
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
