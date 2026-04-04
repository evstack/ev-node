package types

import (
	"testing"
)

// BenchmarkHeaderHash_NoMemo measures the cost of the old 3× call pattern with no
// memoization: each call re-marshals every field via ToProto → proto.Marshal → sha256.
func BenchmarkHeaderHash_NoMemo(b *testing.B) {
	h := GetRandomHeader("bench-chain", GetRandomBytes(32))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = h.Hash()
		_ = h.Hash()
		_ = h.Hash()
	}
}

// BenchmarkHeaderHash_Memoized measures the cost of the same 3× call pattern after
// explicit memoization: first call pays full cost, subsequent two are cache hits.
func BenchmarkHeaderHash_Memoized(b *testing.B) {
	h := GetRandomHeader("bench-chain", GetRandomBytes(32))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		h.InvalidateHash()
		_ = h.MemoizeHash() // compute and store
		_ = h.Hash()        // cache hit
		_ = h.Hash()        // cache hit
	}
}

// BenchmarkHeaderHash_Single is a baseline: cost of one Hash() call with a cold cache.
func BenchmarkHeaderHash_Single(b *testing.B) {
	h := GetRandomHeader("bench-chain", GetRandomBytes(32))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = h.Hash()
	}
}
