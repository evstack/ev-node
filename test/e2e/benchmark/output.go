//go:build evm

package benchmark

import (
	"encoding/json"
	"os"
	"testing"

	e2e "github.com/evstack/ev-node/test/e2e"
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
	w.entries = append(w.entries, spanAvgEntries(w.label, spans)...)
}

// addEntry appends a custom entry to the results.
func (w *resultWriter) addEntry(e entry) {
	w.entries = append(w.entries, e)
}

// addEntries appends multiple entries to the results.
func (w *resultWriter) addEntries(entries []entry) {
	w.entries = append(w.entries, entries...)
}

// flush writes accumulated entries to the path in BENCH_JSON_OUTPUT.
// It is a no-op when the env var is unset or no entries were added.
func (w *resultWriter) flush() {
	outputPath := os.Getenv("BENCH_JSON_OUTPUT")
	if outputPath == "" || len(w.entries) == 0 {
		return
	}

	data, err := json.MarshalIndent(w.entries, "", "  ")
	if err != nil {
		w.t.Logf("WARNING: failed to marshal benchmark JSON: %v", err)
		return
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		w.t.Logf("WARNING: failed to write benchmark JSON to %s: %v", outputPath, err)
		return
	}
	w.t.Logf("wrote %d benchmark entries to %s", len(w.entries), outputPath)
}
