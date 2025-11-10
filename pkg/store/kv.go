package store

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	badger4 "github.com/ipfs/go-ds-badger4"
)

// NewDefaultKVStore creates instance of default key-value store.
func NewDefaultKVStore(rootDir, dbPath, dbName string) (ds.Batching, error) {
	path := filepath.Join(rootify(rootDir, dbPath), dbName)
	return badger4.NewDatastore(path, nil)
}

// PrefixEntries retrieves all entries in the datastore whose keys have the supplied prefix
func PrefixEntries(ctx context.Context, store ds.Datastore, prefix string) (dsq.Results, error) {
	results, err := store.Query(ctx, dsq.Query{Prefix: prefix})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// GenerateKey creates a key from a slice of string fields, joining them with slashes.
func GenerateKey(fields []string) string {
	key := "/" + strings.Join(fields, "/")
	return path.Clean(key)
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
	inMemoryOptions := &badger4.Options{
		Options: badger4.DefaultOptions.WithInMemory(true),
	}
	return badger4.NewDatastore("", inMemoryOptions)
}
