package cache

import (
	"errors"
	"io"
	"math/rand/v2"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

/*
goos: darwin
goarch: arm64
pkg: github.com/evstack/ev-node/block/internal/cache
cpu: Apple M2 Max
BenchmarkManager_GetPendingHeaders
BenchmarkManager_GetPendingHeaders/N=1000
BenchmarkManager_GetPendingHeaders/N=1000-12 	     		 524	   2184969 ns/op	 3552674 B/op	   46915 allocs/op
BenchmarkManager_GetPendingHeaders/N=10000
BenchmarkManager_GetPendingHeaders/N=10000-12         	      46	  22664514 ns/op	35523048 B/op	  469926 allocs/op
BenchmarkPendingHeaders_Iterator
BenchmarkPendingHeaders_Iterator/N=1000
BenchmarkPendingHeaders_Iterator/N=1000-12            	     476	   2289244 ns/op	 3648499 B/op	   46915 allocs/op
BenchmarkPendingHeaders_Iterator/N=10000
BenchmarkPendingHeaders_Iterator/N=10000-12           	      43	  23622464 ns/op	36481617 B/op	  469936 allocs/op
BenchmarkManager_GetPendingData
BenchmarkManager_GetPendingData/N=1000
BenchmarkManager_GetPendingData/N=1000-12             	     273	   4134386 ns/op	 5867378 B/op	   73821 allocs/op
BenchmarkManager_GetPendingData/N=10000
BenchmarkManager_GetPendingData/N=10000-12            	      24	  43703116 ns/op	58664308 B/op	  739879 allocs/op
BenchmarkPendingData_Iterator
BenchmarkPendingData_Iterator/N=1000
BenchmarkPendingData_Iterator/N=1000-12               	     261	   4232198 ns/op	 5858848 B/op	   72821 allocs/op
BenchmarkPendingData_Iterator/N=10000
BenchmarkPendingData_Iterator/N=10000-12              	      26	  45327748 ns/op	58648039 B/op	  729901 allocs/op
BenchmarkManager_PendingEventsSnapshot
BenchmarkManager_PendingEventsSnapshot-12             	89357942	        13.42 ns/op	       0 B/op	       0 allocs/op
PASS
*/

func benchSetupStore(b *testing.B, n int, txsPer int, chainID string) store.Store {
	rootDir := b.TempDir()
	ds, err := store.NewDefaultKVStore(rootDir, chainID, b.Name())
	require.NoError(b, err)
	st := store.New(ds)
	ctx := b.Context()
	for i := 1; i <= n; i++ {
		h, d := types.GetRandomBlock(uint64(i), txsPer, chainID)
		if err := st.SaveBlockData(ctx, h, d, &types.Signature{}); err != nil {
			b.Fatal(err)
		}
	}
	if err := st.SetHeight(ctx, uint64(n)); err != nil {
		b.Fatal(err)
	}
	return st
}

func benchNewManager(b *testing.B, st store.Store) Manager {
	cfg := config.DefaultConfig()
	cfg.RootDir = b.TempDir()
	m, err := NewManager(cfg, st, zerolog.Nop())
	if err != nil {
		b.Fatal(err)
	}
	return m
}

func BenchmarkManager_GetPendingHeaders(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(benchName(n), func(b *testing.B) {
			st := benchSetupStore(b, n, 1, "bench-headers")
			m := benchNewManager(b, st)
			ctx := b.Context()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				hs, err := m.GetPendingHeaders(ctx)
				if err != nil {
					b.Fatal(err)
				}
				if len(hs) == 0 {
					b.Fatal("unexpected empty headers")
				}
			}
		})
	}
}

func BenchmarkPendingHeaders_Iterator(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(benchName(n), func(b *testing.B) {
			st := benchSetupStore(b, n, 1, "bench-headers-iter")
			ph, err := NewPendingHeaders(st, zerolog.Nop())
			if err != nil {
				b.Fatal(err)
			}
			ctx := b.Context()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				iter, err := ph.Iterator(ctx)
				if err != nil {
					b.Fatal(err)
				}
				count := 0
				for {
					val, ok, err := iter.Next()
					if errors.Is(err, io.EOF) {
						if ok {
							b.Fatal("expected ok=false on EOF")
						}
						break
					}
					if err != nil {
						b.Fatal(err)
					}
					if !ok {
						b.Fatal("expected ok=true before EOF")
					}
					if val == nil {
						b.Fatal("unexpected nil header")
					}
					count++
				}
				if count == 0 {
					b.Fatal("unexpected empty iteration result")
				}
			}
		})
	}
}

func BenchmarkPendingHeaders_Iterator2(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(benchName(n), func(b *testing.B) {
			st := benchSetupStore(b, n, 1, "bench-headers-iter")
			ph, err := NewPendingHeaders(st, zerolog.Nop())
			if err != nil {
				b.Fatal(err)
			}
			ctx := b.Context()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				iter := ph.Query(ctx)

				count := 0
				for val, err := range iter {
					require.NoError(b, err)
					require.NotNil(b, val)
					count++
				}
				require.NotEmpty(b, count)
			}
		})
	}
}

func BenchmarkManager_GetPendingData(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(benchName(n), func(b *testing.B) {
			st := benchSetupStore(b, n, 2, "bench-data") // ensure data not filtered
			m := benchNewManager(b, st)
			ctx := b.Context()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				ds, err := m.GetPendingData(ctx)
				if err != nil {
					b.Fatal(err)
				}
				if len(ds) == 0 {
					b.Fatal("unexpected empty data")
				}
			}
		})
	}
}

func BenchmarkPendingData_Iterator(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(benchName(n), func(b *testing.B) {
			st := benchSetupStore(b, n, 2, "bench-data-iter")
			pd, err := NewPendingData(st, zerolog.Nop())
			if err != nil {
				b.Fatal(err)
			}
			ctx := b.Context()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				iter, err := pd.Iterator(ctx)
				if err != nil {
					b.Fatal(err)
				}
				count := 0
				for {
					val, ok, err := iter.Next()
					if errors.Is(err, io.EOF) {
						if ok {
							b.Fatal("expected ok=false on EOF")
						}
						break
					}
					if err != nil {
						b.Fatal(err)
					}
					if !ok {
						b.Fatal("expected ok=true before EOF")
					}
					if len(val.Txs) == 0 {
						continue
					}
					count++
				}
				if count == 0 {
					b.Fatal("unexpected empty iteration result")
				}
			}
		})
	}
}

func BenchmarkManager_PendingEventsSnapshot(b *testing.B) {
	st := benchSetupStore(b, 1_000, 1, "bench-events")
	m := benchNewManager(b, st)
	for i := 1; i <= 50_000; i++ {
		h, d := types.GetRandomBlock(uint64(i), 1, "bench-events")
		m.SetPendingEvent(uint64(i), &common.DAHeightEvent{Header: h, Data: d, DaHeight: uint64(i)})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Test getting next pending event at various heights
		height := rand.N(uint64(50_000)) + 1 //nolint:gosec // this is a benchmark test
		_ = m.GetNextPendingEvent(height)
	}
}

func benchName(n int) string {
	// simple itoa without fmt to avoid allocations
	if n == 0 {
		return "N=0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return "N=" + string(buf[i:])
}
