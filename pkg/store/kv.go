package store

import (
	"context"
	"path/filepath"
	"strings"

	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	dsq "github.com/ipfs/go-datastore/query"
	badger4 "github.com/ipfs/go-ds-badger4"
)

// EvPrefix is used in KV store to separate ev-node data from execution environment data (if the same data base is reused)
const EvPrefix = "0"

// NewDefaultKVStore creates instance of default key-value store.
func NewDefaultKVStore(rootDir, dbPath, dbName string) (ds.Batching, error) {
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
