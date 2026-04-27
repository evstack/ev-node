package store

import (
	"context"
	"path/filepath"
	"strings"

	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	dsq "github.com/ipfs/go-datastore/query"
	dssync "github.com/ipfs/go-datastore/sync"
	badger4 "github.com/ipfs/go-ds-badger4"
)

// EvPrefix is used in KV store to separate ev-node data from execution environment data (if the same data base is reused)
const EvPrefix = "0"

// NewDefaultKVStore creates instance of default key-value store.
//
// HACK(fiber-throughput): swapped to a pure in-memory map for the
// Fibre throughput investigation. The real issue this surfaces is
// architectural, not a Badger bug: block.executing.Executor.ProduceBlock
// calls store.batch.Commit() synchronously inside the producer, so
// the storage engine's write rate is a hard ceiling on block
// production. With 128 MiB blocks × ~1 b/s the on-disk path drives
// ~150 MB/s of value-log writes plus heavy compaction; the producer
// blocks on Badger long before the DA submitter is the bottleneck.
//
// Don't revert this in place — fix the underlying design instead.
// Options worth weighing:
//   - Move the block save off the producer hot path (async commit
//     with a bounded queue). Block durability is not required to
//     advance state, only to recover after restart.
//   - For Fibre-only rollups specifically: skip local persistence
//     entirely. Fibre IS the storage; a node can re-sync from the
//     chain on restart. This removes the question.
//   - If we keep persisting, pick a write-optimised backend that
//     handles 100s of MB/s of large-value writes without compaction
//     stalls. Badger v4 with these tunings still hit a .vlog
//     rotation race under sustained load.
//
// NewDefaultKVStoreOnDisk preserved below as the literal Badger
// constructor for any caller that explicitly wants disk-backed
// state today; the production wiring should switch to one of the
// three options above before this directory is dropped.
func NewDefaultKVStore(_, _, _ string) (ds.Batching, error) {
	return dssync.MutexWrap(ds.NewMapDatastore()), nil
}

// NewDefaultKVStoreOnDisk is the original Badger-backed constructor,
// preserved for the duration of the throughput-cleanup window.
func NewDefaultKVStoreOnDisk(rootDir, dbPath, dbName string) (ds.Batching, error) {
	path := filepath.Join(rootify(rootDir, dbPath), dbName)
	return badger4.NewDatastore(path, BadgerOptions())
}

// NewPrefixKVStore creates a new key-value store with a prefix applied to all keys.
func NewPrefixKVStore(kvStore ds.Batching, prefix string) ds.Batching {
	return ktds.Wrap(kvStore, ktds.PrefixTransform{Prefix: ds.NewKey(prefix)})
}

// NewEvNodeKVStore creates a new key-value store with EvPrefix prefix applied to all keys.
func NewEvNodeKVStore(kvStore ds.Batching) ds.Batching {
	return NewPrefixKVStore(kvStore, EvPrefix)
}

// GetPrefixEntries retrieves all entries in the datastore whose keys have the supplied prefix
func GetPrefixEntries(ctx context.Context, store ds.Datastore, prefix string) (dsq.Results, error) {
	results, err := store.Query(ctx, dsq.Query{Prefix: prefix})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GenerateKey creates a key from a slice of string fields, joining them with slashes.
func GenerateKey(fields []string) string {
	// Pre-calculate total size to avoid re-allocation.
	n := 0
	for _, f := range fields {
		n += 1 + len(f) // '/' + field
	}
	var b strings.Builder
	b.Grow(n)
	for _, f := range fields {
		b.WriteByte('/')
		b.WriteString(f)
	}
	return b.String()
}

// rootify works just like in cosmos-sdk
func rootify(rootDir, dbPath string) string {
	if filepath.IsAbs(dbPath) {
		return dbPath
	}
	return filepath.Join(rootDir, dbPath)
}

// NewTestInMemoryKVStore builds KVStore that works in-memory (without accessing disk).
func NewTestInMemoryKVStore() (ds.Batching, error) {
	inMemoryOptions := BadgerOptions()
	inMemoryOptions.Options = inMemoryOptions.WithInMemory(true)
	return badger4.NewDatastore("", inMemoryOptions)
}
