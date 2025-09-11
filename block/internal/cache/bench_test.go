package cache

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

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
	cfg := config.DefaultConfig
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
			for i := 0; i < b.N; i++ {
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
			for i := 0; i < b.N; i++ {
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

func BenchmarkManager_PendingEventsSnapshot(b *testing.B) {
	st := benchSetupStore(b, 1_000, 1, "bench-events")
	m := benchNewManager(b, st)
	for i := 1; i <= 50_000; i++ {
		h, d := types.GetRandomBlock(uint64(i), 1, "bench-events")
		m.SetPendingEvent(uint64(i), &DAHeightEvent{Header: h, Data: d, DaHeight: uint64(i)})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ev := m.GetPendingEvents()
		if len(ev) == 0 {
			b.Fatal("unexpected empty events")
		}
	}
}

func BenchmarkHeaderCache_RangeAsc(b *testing.B) {
	st := benchSetupStore(b, 1, 1, "bench-range")
	m := benchNewManager(b, st)
	impl := m.(*implementation)
	for i := 1; i <= 50_000; i++ {
		h, _ := types.GetRandomBlock(uint64(i), 1, "bench-range")
		impl.SetHeader(uint64(i), h)
	}
	sum := 0
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum = 0
		impl.headerCache.RangeByHeightAsc(func(height uint64, sh *types.SignedHeader) bool {
			sum += int(height)
			return true
		})
	}
	if sum == 42 {
		b.Fatal("avoid eliminate")
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
