package store

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/types"
)

func BenchmarkStoreSync(b *testing.B) {
	chainID := "BenchmarkStoreSync"
	testCases := []struct {
		name      string
		blockSize int
	}{
		{"empty", 0},
		{"small", 10},
		{"large", 1_000},
	}

	for _, tc := range testCases {
		header, data := types.GetRandomBlock(1, tc.blockSize, chainID)
		signature := &types.Signature{}

		b.Run(fmt.Sprintf("WithoutSync_%s", tc.name), func(b *testing.B) {
			tmpDir := b.TempDir()
			kv, err := NewDefaultKVStore(tmpDir, "db", "test")
			require.NoError(b, err)
			store := New(kv)
			b.Cleanup(func() { _ = store.Close() })

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch, err := store.NewBatch(b.Context())
				require.NoError(b, err)

				err = batch.SaveBlockData(header, data, signature)
				require.NoError(b, err)

				err = batch.Commit()
				require.NoError(b, err)
			}
		})

		b.Run(fmt.Sprintf("WithSync_%s", tc.name), func(b *testing.B) {
			tmpDir := b.TempDir()
			kv, err := NewDefaultKVStore(tmpDir, "db", "test")
			require.NoError(b, err)
			store := New(kv)
			b.Cleanup(func() { _ = store.Close() })

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				batch, err := store.NewBatch(b.Context())
				require.NoError(b, err)

				err = batch.SaveBlockData(header, data, signature)
				require.NoError(b, err)

				err = batch.Commit()
				require.NoError(b, err)

				err = store.Sync(b.Context())
				require.NoError(b, err)
			}
		})
	}
}
