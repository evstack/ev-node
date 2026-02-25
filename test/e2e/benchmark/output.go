//go:build evm

package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	e2e "github.com/evstack/ev-node/test/e2e"
	"github.com/stretchr/testify/require"
)

// entry matches the customSmallerIsBetter format for github-action-benchmark.
type entry struct {
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

// resultWriter accumulates benchmark entries and writes them to a JSON file
// when flush is called. Create one early in a test and defer flush so results
// are written regardless of where the test exits.
type resultWriter struct {
	t       testing.TB
	label   string
	entries []entry
}

func newResultWriter(t testing.TB, label string) *resultWriter {
	return &resultWriter{t: t, label: label}
}

// addSpans aggregates trace spans into per-operation avg duration entries.
func (w *resultWriter) addSpans(spans []e2e.TraceSpan) {
	m := e2e.AggregateSpanStats(spans)
	if len(m) == 0 {
		return
	}

	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		s := m[name]
		avg := float64(s.Total.Microseconds()) / float64(s.Count)
		w.entries = append(w.entries, entry{
			Name:  fmt.Sprintf("%s - %s (avg)", w.label, name),
			Unit:  "us",
			Value: avg,
		})
	}
}

// addEntry appends a custom entry to the results.
func (w *resultWriter) addEntry(e entry) {
	w.entries = append(w.entries, e)
}

// flush writes accumulated entries to the path in BENCH_JSON_OUTPUT.
// It is a no-op when the env var is unset or no entries were added.
func (w *resultWriter) flush() {
	outputPath := os.Getenv("BENCH_JSON_OUTPUT")
	if outputPath == "" || len(w.entries) == 0 {
		return
	}

	data, err := json.MarshalIndent(w.entries, "", "  ")
	require.NoError(w.t, err, "failed to marshal benchmark JSON")
	require.NoError(w.t, os.WriteFile(outputPath, data, 0644), "failed to write benchmark JSON to %s", outputPath)
	w.t.Logf("wrote %d benchmark entries to %s", len(w.entries), outputPath)
}
