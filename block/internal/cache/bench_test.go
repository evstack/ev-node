package cache

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"testing"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

/*
goos: darwin
goarch: arm64
pkg: github.com/evstack/ev-node/block/internal/cache
cpu: Apple M4
BenchmarkManager_GetPendingHeaders/N=1000-10         194           5763894 ns/op         3552719 B/op      46916 allocs/op
BenchmarkManager_GetPendingHeaders/N=10000-10                 21          53211540 ns/op        35524178 B/op     469941 allocs/op
BenchmarkManager_GetPendingData/N=1000-10                    100          10699778 ns/op         5868643 B/op      73823 allocs/op
BenchmarkManager_GetPendingData/N=10000-10                    13         138766532 ns/op        58667709 B/op     739950 allocs/op
BenchmarkPendingData_Iterator/N=1000-10                       91          12462366 ns/op         5849299 B/op      72828 allocs/op
BenchmarkPendingData_Iterator/N=10000-10                      12         111749198 ns/op        58643530 B/op     729975 allocs/op
BenchmarkPendingHeaders_Iterator/N=1000-10                   204           6132184 ns/op         3648970 B/op      46924 allocs/op
BenchmarkPendingHeaders_Iterator/N=10000-10                   25          53847862 ns/op        36485675 B/op     470017 allocs/op
BenchmarkManager_PendingEventsSnapshot-10               30719360                35.37 ns/op            0 B/op          0 allocs/op
PASS
ok      github.com/evstack/ev-node/block/internal/cache 23.693s
*/

func benchSetupStore(b *testing.B, n int, txsPer int, chainID string) store.Store {
	ds, err := store.NewDefaultInMemoryKVStore()
	if err != nil {
		b.Fatal(err)
	}
	st := store.New(ds)
	ctx := context.Background()
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
			ctx := context.Background()
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

func BenchmarkManager_GetPendingData(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(benchName(n), func(b *testing.B) {
			st := benchSetupStore(b, n, 2, "bench-data") // ensure data not filtered
			m := benchNewManager(b, st)
			ctx := context.Background()
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
			ctx := context.Background()
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

func BenchmarkPendingHeaders_Iterator(b *testing.B) {
	for _, n := range []int{1_000, 10_000} {
		b.Run(benchName(n), func(b *testing.B) {
			st := benchSetupStore(b, n, 1, "bench-headers-iter")
			ph, err := NewPendingHeaders(st, zerolog.Nop())
			if err != nil {
				b.Fatal(err)
			}
			ctx := context.Background()
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
