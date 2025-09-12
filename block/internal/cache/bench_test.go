package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	storepkg "github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

/*
goos: darwin
goarch: arm64
pkg: github.com/evstack/ev-node/block/internal/cache
cpu: Apple M1 Pro
BenchmarkManager_GetPendingHeaders/N=1000-10 	     278	   3922717 ns/op	5064666 B/op	70818 allocs/op
BenchmarkManager_GetPendingHeaders/N=10000-10         	      28	  40704543 ns/op	50639803 B/op	709864 allocs/op
BenchmarkManager_GetPendingData/N=1000-10             	     279	   4258291 ns/op	5869716 B/op	73824 allocs/op
BenchmarkManager_GetPendingData/N=10000-10            	      26	  45428974 ns/op	58719067 B/op	  739926 allocs/op
BenchmarkManager_PendingEventsSnapshot-10             	     336	   3251530 ns/op	 2365497 B/op	     285 allocs/op
PASS
ok  	github.com/evstack/ev-node/block/internal/cache	25.834s
*/

func benchSetupStore(b *testing.B, n int, txsPer int, chainID string) storepkg.Store {
	ds, err := storepkg.NewDefaultInMemoryKVStore()
	if err != nil {
		b.Fatal(err)
	}
	st := storepkg.New(ds)
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

func benchNewManager(b *testing.B, st storepkg.Store) Manager {
	cfg := config.DefaultConfig
	cfg.RootDir = b.TempDir()
	m, err := NewManager(cfg, st, zerolog.Nop())
	if err != nil {
		b.Fatal(err)
	}
	return m
}

// Helpers to create pending managers with either iterator path (using datastore query)
// or a manual one-by-one iterator (baseline for comparison).
func benchNewPendingData(b *testing.B, st storepkg.Store, useIter bool) *PendingData {
	var iter func(ctx context.Context, s storepkg.Store, after uint64) ([]*types.Data, error)
	if useIter {
		iter = func(ctx context.Context, s storepkg.Store, after uint64) ([]*types.Data, error) {
			if ds, ok := s.(*storepkg.DefaultStore); ok {
				return ds.DataAfterHeight(ctx, after)
			}
			return nil, fmt.Errorf("iterator not supported")
		}
	} else {
		iter = func(ctx context.Context, s storepkg.Store, after uint64) ([]*types.Data, error) {
			// Manual one-by-one retrieval baseline
			h, err := s.Height(ctx)
			if err != nil {
				return nil, err
			}
			if after >= h {
				return nil, nil
			}
			out := make([]*types.Data, 0, h-after)
			for i := after + 1; i <= h; i++ {
				_, d, err := s.GetBlockData(ctx, i)
				if err != nil {
					return out, err
				}
				out = append(out, d)
			}
			return out, nil
		}
	}
	base, err := newPendingBase(st, zerolog.Nop(), LastSubmittedDataHeightKey, iter)
	if err != nil {
		b.Fatal(err)
	}
	return &PendingData{base: base}
}

func benchNewPendingHeaders(b *testing.B, st storepkg.Store, useIter bool) *PendingHeaders {
	var iter func(ctx context.Context, s storepkg.Store, after uint64) ([]*types.SignedHeader, error)
	if useIter {
		iter = func(ctx context.Context, s storepkg.Store, after uint64) ([]*types.SignedHeader, error) {
			if ds, ok := s.(*storepkg.DefaultStore); ok {
				return ds.HeadersAfterHeight(ctx, after)
			}
			return nil, fmt.Errorf("iterator not supported")
		}
	} else {
		iter = func(ctx context.Context, s storepkg.Store, after uint64) ([]*types.SignedHeader, error) {
			// Manual one-by-one retrieval baseline
			h, err := s.Height(ctx)
			if err != nil {
				return nil, err
			}
			if after >= h {
				return nil, nil
			}
			out := make([]*types.SignedHeader, 0, h-after)
			for i := after + 1; i <= h; i++ {
				hd, err := s.GetHeader(ctx, i)
				if err != nil {
					return out, err
				}
				out = append(out, hd)
			}
			return out, nil
		}
	}
	base, err := newPendingBase(st, zerolog.Nop(), storepkg.LastSubmittedHeaderHeightKey, iter)
	if err != nil {
		b.Fatal(err)
	}
	return &PendingHeaders{base: base}
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
		m.SetPendingEvent(uint64(i), &common.DAHeightEvent{Header: h, Data: d, DaHeight: uint64(i)})
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
