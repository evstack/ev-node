package store

import (
	"runtime"

	badger4 "github.com/ipfs/go-ds-badger4"
)

// BadgerOptions returns ev-node tuned Badger options for the node workload.
// These defaults favor write throughput for append-heavy usage.
func BadgerOptions() *badger4.Options {
	opts := badger4.DefaultOptions

	// Disable conflict detection to reduce write overhead; ev-node does not rely
	// on Badger's multi-writer conflict checks for correctness.
	opts.Options = opts.WithDetectConflicts(false)
	// Allow more L0 tables before compaction kicks in to smooth bursty ingest.
	opts.Options = opts.WithNumLevelZeroTables(10)
	// Stall threshold is raised to avoid write throttling under heavy load.
	opts.Options = opts.WithNumLevelZeroTablesStall(20)
	// Scale compaction workers to available CPUs without over-saturating.
	opts.Options = opts.WithNumCompactors(compactorCount())

	return &opts
}

func compactorCount() int {
	// Badger defaults to 4; keep a modest range to avoid compaction thrash.
	count := runtime.NumCPU()
	if count < 4 {
		return 4
	}
	if count > 8 {
		return 8
	}
	return count
}
