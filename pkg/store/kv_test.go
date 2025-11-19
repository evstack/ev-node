package store

import (
	"context"
	"testing"

	ds "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultReadOnlyKVStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	rootDir := t.TempDir()
	const dbPath = "db"
	const dbName = "test"

	writable, err := NewDefaultKVStore(rootDir, dbPath, dbName)
	require.NoError(t, err)
	require.NoError(t, writable.Put(ctx, ds.NewKey("/foo"), []byte("bar")))
	require.NoError(t, writable.Close())

	readOnly, err := NewDefaultReadOnlyKVStore(rootDir, dbPath, dbName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = readOnly.Close()
	})

	val, err := readOnly.Get(ctx, ds.NewKey("/foo"))
	require.NoError(t, err)
	require.Equal(t, []byte("bar"), val)

	err = readOnly.Put(ctx, ds.NewKey("/foo"), []byte("baz"))
	require.Error(t, err, "writing to a read-only store should fail")
}
