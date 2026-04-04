# Memory & Allocation Optimization Plan

**Date:** 2026-03-30
**Source:** pprof profiles collected from `prd-eden-testnet-node-1:6060`
**Node binary:** `evm` (Build ID: `c66958170e0ea317dee65bab308c02d0cc6b6098`)

---

## Context

A production node on a 62 GiB server is consuming 46 GiB in OS Cache+Buffer, using 2.6 GiB of swap, and the pattern repeats when the server is upsized — available RAM is consumed. The GC is spending ~55% of sampled CPU time in `scanobject`/`findObject`/`gcDrain`, indicating sustained heap pressure rather than a CPU-bound workload.

Profiles collected:

| Profile | Command |
|---------|---------|
| CPU (30s) | `curl http://prd-eden-testnet-node-1:6060/debug/pprof/profile?seconds=30` |
| Heap (live) | `curl http://prd-eden-testnet-node-1:6060/debug/pprof/heap` |
| Allocs (lifetime) | `curl http://prd-eden-testnet-node-1:6060/debug/pprof/allocs` |
| Goroutines | `curl http://prd-eden-testnet-node-1:6060/debug/pprof/goroutine` |

**Follow-up profile:** 2026-04-03, `prd-eden-testnet-node-2:6060` (Build ID: `2b20bbdd78f3b2cecb508bb4ba5a7a12803df841`). In-use heap grew to ~2 GB (up from ~1.05 GB). GC dominance is gone (CPU now 2% utilization, no `scanobject`/`gcDrain` in top) but heap footprint increased. Dominant consumers shifted to `block/internal/cache` (see Issue 6). `go-header Hash.String` holds 302 MB (new Issue 7). Proto marshal is no longer in the top allocators. Gzip/flate share of cumulative allocs rose from 0.8% to 19%. Issues 1, 3, 4, 5 remain open.

**Additional work merged (not in original plan):**
- [#3219](https://github.com/evstack/ev-node/pull/3219) — `Header.MemoizeHash()`: caches computed hash on the struct, avoiding repeated `sha256` + `proto.Marshal` on every `Hash()` call. Partially reduces heap pressure from hash-derived allocations.
- [#3204](https://github.com/evstack/ev-node/pull/3204) — Removed LRU from `block/internal/cache` generic cache; simplified eviction model.

---

## Findings

### 2026-03-30 — node-1 (Build ID: `c66958170e0ea317dee65bab308c02d0cc6b6098`)

#### CPU profile (3.68s samples over 30s)

| flat | cum | function |
|------|-----|----------|
| 16.6% | 17.4% | `runtime.findObject` |
| 11.1% | 41.6% | `runtime.scanobject` |
| 3.5%  | —    | `runtime.(*gcBits).bitp` |
| —     | 41.9%| `runtime.gcDrain` |

GC accounts for ~55% of sampled CPU. The node is not CPU-bound; the GC is responding to sustained allocation pressure.

#### Heap — live objects (~1.05 GB in use)

| MB (flat) | MB (cum) | call site |
|-----------|----------|-----------|
| 312 | 312 | `ristretto/z.Calloc` (off-heap, mmap-based block cache) |
| 101 | 490 | `store.DefaultStore.GetHeader` |
| 83  | 83  | `badger/skl.newArena` (memtables) |
| 82  | 82  | `protobuf consumeBytesSlice` |
| 63  | 63  | LRU `insertValue` |
| 56  | 125 | `types.SignedHeader.FromProto` |
| 9   | 539 | `store.DefaultStore.GetBlockData` |
| 4   | 579 | `store.CachedStore.GetBlockData` |
| 0   | 504 | `go-header/p2p.ExchangeServer.handleRangeRequest` |

#### Allocs — lifetime (~58.5 TB total)

| % of total | call site |
|------------|-----------|
| 63.5% | `proto.MarshalOptions.marshal` |
| 12.1% | `encoding/json.(*Decoder).refill` |
| 6.1%  | `encoding/json.(*decodeState).literalStore` |
| 5.6%  | `encoding/json.(*RawMessage).UnmarshalJSON` |
| 2.8%  | `encoding/hex.DecodeString` |
| 2.0%  | `encoding/hex.EncodeToString` |
| 1.5%  | `go-buffer-pool.(*BufferPool).Get` |
| 0.8%  | `compress/flate.dictDecoder.init` |

The `proto.Marshal` allocations trace to `types.P2PData.MarshalBinary` → `go-header/p2p.ExchangeServer.handleRangeRequest`. For a range request of 128 headers, 128 fresh scratch buffers are allocated.

The JSON/hex allocations trace to `evm.EngineClient.GetTxs` → `rpc.Client.sendHTTP` (go-ethereum). The gzip allocations are from `net/http`'s transparent decompression of EL responses.

---

### 2026-04-03 — node-2 (Build ID: `2b20bbdd78f3b2cecb508bb4ba5a7a12803df841`)

#### CPU profile (610ms samples over 30s — 2% utilization)

GC dominance is gone. Node is largely idle. Top consumers:

| flat | flat% | cum | function |
|------|-------|-----|----------|
| 100ms | 16.4% | 100ms | `syscall.Syscall6` |
| 50ms  | 8.2%  | 50ms  | `edwards25519/field.feMul` (signature verification) |
| 10ms  | 1.6%  | 50ms  | `rpc.(*httpConn).doRequest` (upstream EL calls) |

#### Heap — live objects (~2.0 GB in use)

| MB (flat) | MB (cum) | call site |
|-----------|----------|-----------|
| 514 | 514 | `block/internal/cache.(*Cache).setSeen` |
| 420 | 420 | `block/internal/cache.(*Cache).setDAIncluded` |
| 307 | 307 | `block/internal/cache.HeightPlaceholderKey` (inline) |
| 302 | 302 | `go-header.Hash.String` |
| 148 | 148 | `ristretto/z.Calloc` (off-heap) |
| 83  | 83  | `badger/skl.newArena` (memtables) |
| 75  | 285 | `store.DefaultStore.GetHeader` |
| 38  | 38  | `types.Header.FromProto` |
| 32  | 87  | `types.SignedHeader.FromProto` |
| 13  | 21  | LRU `insertValue` |
| 0   | 229 | `go-header/p2p.ExchangeServer.handleRangeRequest` |

The top four entries — `setSeen`, `setDAIncluded`, `HeightPlaceholderKey`, `Hash.String` — account for ~1.54 GB (77% of heap). These are all within `block/internal/cache` and the header-exchange path. The store LRU is now negligible (13 MB).

#### Allocs — lifetime (~805 GB total)

| % of total | call site |
|------------|-----------|
| 30.2% | `encoding/json.(*Decoder).refill` (cumulative) |
| 24.1% | `compress/flate.NewReader` + `dictDecoder.init` (combined) |
| 21.6% | `ristretto/z.Calloc` |
| 4.6%  | `compress/flate.(*dictDecoder).init` (direct) |
| 3.2%  | `encoding/json.Marshal` |
| 1.5%  | `net/http.Header.Clone` |
| 1.5%  | `encoding/json.(*RawMessage).UnmarshalJSON` |
| 1.5%  | `io.ReadAll` |

Proto marshal no longer appears in the top allocators — the prior dominance (63.5%) has dissipated, likely due to reduced range request volume or binary encoding changes. Gzip/flate rose from 0.8% to ~24% of total allocs; the EL HTTP decompression path is now the single largest allocation source alongside JSON decoding.

---

## Issues and Fixes

### Issue 1 — LRU caches are count-based, not size-bound

**Severity:** High — still unresolved; store LRU holds 13 MB in the 2026-04-03 profile (down from 63 MB) but defaults remain at 200K. Superseded in live-heap dominance by Issue 6.

**Location:** `pkg/store/cached_store.go:11-17`

```go
DefaultHeaderCacheSize    = 200_000
DefaultBlockDataCacheSize = 200_000
```

These are item counts. A `*types.SignedHeader` is typically 300–800 bytes. A `blockDataEntry` includes the full `*types.Data` with all transactions — on a high-TPS chain this can be 50–500 KB per entry. At 200,000 entries, `blockDataCache` has a theoretical maximum in the tens of GB.

**Fix:** Reduce defaults to `2_048` (headers) and `512` (block data). Expose both as fields in `config.Config` so operators can tune without recompiling.

**Files:**
- `pkg/store/cached_store.go` — reduce constants, update `NewCachedStore` to accept config values
- `pkg/config/defaults.go` — add `DefaultStoreCacheHeaderSize = 2048`, `DefaultStoreCacheBlockDataSize = 512`
- `pkg/config/config.go` — add `StoreConfig` struct with `HeaderCacheSize int` and `BlockDataCacheSize int`
- `node/full.go` — pass config values to `NewCachedStore`
- `node/light.go` — same

**Benchmarks:**
```
// pkg/store/cached_store_test.go
BenchmarkCachedStore_GetHeader_HotPath      // cache-hit path — must be 0 allocs/op
BenchmarkCachedStore_GetBlockData_HotPath   // cache-hit path — must be 0 allocs/op
TestCachedStore_MemoryBound                 // runtime.ReadMemStats before/after; assert heap < threshold
```

**Acceptance criteria:**
- Cache-hit path: 0 `allocs/op` (pointer return, no copy)
- Two warm caches at new defaults hold < 50 MB total for realistic block sizes
- Configurable via `[store]` TOML section; documented in operator guide

---

### Issue 2 — Badger `IndexCacheSize` unbounded ✅ COMPLETED

**Severity:** High — index RAM grows proportionally to chain length.

**Resolved:** [#3209](https://github.com/evstack/ev-node/pull/3209) (2026-03-30)

**Location:** `pkg/store/badger_options.go`

`badger.DefaultOptions` sets `BlockCacheSize = 256 MB` but `IndexCacheSize = 0` (unbounded in-memory). As the chain grows, SST block index entries accumulate in RAM without eviction. On a chain with millions of blocks this is several GB.

**Fix applied:**

```go
opts.Options = opts.WithIndexCacheSize(DefaultBadgerIndexCacheSize) // 256 MiB (was 0, unbounded)
```

Note: implemented at 256 MB rather than the originally planned 128 MB.

Operators with more RAM should increase `IndexCacheSize` in config. The tradeoff is that cold reads (index entries not in cache) go to disk — acceptable on NVMe, noticeable on spinning disk.

**Files:**
- `pkg/store/badger_options.go`

**Benchmarks:**
```
// pkg/store/store_test.go
BenchmarkStore_WriteThenRead   // write N blocks, read random order — assert < 5% throughput regression
```

**Acceptance criteria:**
- RSS growth from Badger index is bounded at ~128 MB rather than growing with chain length
- No more than 5% regression in `BenchmarkStore_WriteThenRead` vs baseline

---

### Issue 3 — Proto marshal scratch buffers not pooled (63.5% of all allocations)

**Severity:** High — dominant allocation source, drives GC pressure.

**Location:** `types/p2p_envelope.go` (`P2PData.MarshalBinary`, `P2PSignedHeader.MarshalBinary`), `types/serialization.go` (`SignedHeader.MarshalBinary`, `Data.MarshalBinary`, `Metadata.MarshalBinary`, `Header.MarshalBinary`)

Every call to `proto.Marshal(msg)` allocates a fresh `[]byte` scratch buffer internally. For `handleRangeRequest` serving 128 headers, this is 128 allocations per request.

**Fix:** Replace `proto.Marshal(msg)` with `proto.MarshalOptions{}.MarshalAppend` and a `sync.Pool` of scratch buffers:

```go
var marshalPool = sync.Pool{New: func() any { b := make([]byte, 0, 1024); return &b }}

func marshalProto(msg proto.Message) ([]byte, error) {
    bp := marshalPool.Get().(*[]byte)
    out, err := proto.MarshalOptions{}.MarshalAppend((*bp)[:0], msg)
    marshalPool.Put(bp)
    if err != nil {
        return nil, err
    }
    // Copy — caller must not hold a reference into the pool buffer.
    result := make([]byte, len(out))
    copy(result, out)
    return result, nil
}
```

**Critical:** The pool buffer must not escape. The `copy` before return is mandatory. Run all marshal tests with `-race`.

**Files:**
- `types/p2p_envelope.go`
- `types/serialization.go`
- Optionally extract `marshalProto` to `types/marshal.go` as a package-internal helper

**Benchmarks:**
```
// types/p2p_envelope_bench_test.go  (new file)
BenchmarkP2PData_MarshalBinary            // b.RunParallel to expose pool contention
BenchmarkP2PSignedHeader_MarshalBinary
BenchmarkSignedHeader_MarshalBinary       // types/serialization_bench_test.go
```

**Acceptance criteria:**
- `allocs/op` drops from 6–10 to 1–2 per marshal call
- `B/op` drops by 30–50% (scratch buffer reused; only final copy allocated)
- No race detector findings: `go test -race ./types/...`
- All existing round-trip tests pass: `TestP2PEnvelope_MarshalUnmarshal`, `TestDataBinaryCompatibility`

---

### Issue 4 — `filterTransactions` hex encoding allocates 2× per transaction

**Severity:** Medium — ~20% of EVM RPC allocations.

**Location:** `execution/evm/execution.go` — `filterTransactions`

```go
"0x" + hex.EncodeToString(tx)   // two allocations: hex string + concatenation
```

**Fix:** Use `hex.Encode` directly into a pre-allocated, pool-backed buffer:

```go
var hexBufPool = sync.Pool{New: func() any { b := make([]byte, 0, 512); return &b }}

func encodeHexTx(tx []byte) string {
    bp := hexBufPool.Get().(*[]byte)
    needed := 2 + hex.EncodedLen(len(tx))
    buf := append((*bp)[:0], "0x"...)
    buf = buf[:needed]
    hex.Encode(buf[2:], tx)
    s := string(buf[:needed])
    *bp = buf
    hexBufPool.Put(bp)
    return s
}
```

This reduces two allocations to one (the unavoidable `string()` conversion).

**Files:**
- `execution/evm/execution.go`

**Benchmarks:**

Extend the existing `execution/evm/filter_bench_test.go`:
```
BenchmarkFilterTransactions_HexEncoding   // 1000 txs of 150 bytes each, -benchmem
```

**Acceptance criteria:**
- `allocs/op` drops from `2N` to `N` for N transactions
- `B/op` drops by ~40% for a 1000-tx batch

---

### Issue 5 — Gzip reader allocated per EL HTTP response

**Severity:** Medium — ~1 TB cumulative lifetime allocations; removes the gzip decompressor from the hot path.

**Root cause:** `net/http`'s transport automatically decompresses `Content-Encoding: gzip` responses by allocating a fresh `gzip.Reader` per response. This is in Go's standard library, triggered by go-ethereum's `rpc.Client.sendHTTP`. Not patchable in upstream libraries.

**Fix A (preferred when EL is co-located):** Disable HTTP compression on the EL client transport in `NewEngineExecutionClient`:

```go
rpc.WithHTTPClient(&http.Client{
    Transport: &http.Transport{DisableCompression: true},
})
```

No gzip reader is ever allocated. Valid only when EL and ev-node are on the same host (local compression has no bandwidth benefit).

**Fix B (when EL is remote):** Implement a `sync.Pool`-backed `http.RoundTripper` that calls `(*gzip.Reader).Reset(body)` instead of `gzip.NewReader(body)`. Higher complexity; only needed for remote EL deployments.

**Fix C (best for same-host production):** Switch `engine_url` to an IPC socket path (`/tmp/geth.ipc`). The IPC codec uses raw JSON over a Unix socket — no HTTP framing, no gzip, no JSON decoder allocation at the transport layer.

**Files:**
- `execution/evm/execution.go` — `NewEngineExecutionClient`: add `DisableCompression: true`
- `execution/evm/flags.go` — document IPC URL as the recommended transport for same-host deployments

**Benchmarks:**

Verify via pprof before/after: `compress/flate.(*decompressor)` and `compress/gzip.(*Reader)` must disappear from the top allocation sites in the allocs profile.

**Acceptance criteria:**
- Post-fix allocs profile shows zero `compress/flate` or `compress/gzip` entries
- No correctness regression under round-trip test with a real EL

---

### Issue 6 — `block/internal/cache` unbounded growth (~1.24 GB live)

**Severity:** Critical — new #1 heap consumer as of 2026-04-03 profile.

**Location:** `block/internal/cache/generic_cache.go`, `block/internal/cache/manager.go`

`setSeen`, `setDAIncluded`, and `HeightPlaceholderKey` together hold ~1.24 GB of live heap. The LRU was removed from this cache in [#3204](https://github.com/evstack/ev-node/pull/3204), which simplified eviction but appears to have left these maps growing without a bound. `RestoreFromStore` drives the initial population; ongoing `setSeen`/`setDAIncluded` calls accumulate entries without eviction.

**Fix:** Audit `generic_cache.go` for unbounded map growth. Reintroduce a maximum-size eviction policy (LRU or a fixed-size ring buffer keyed by height) or a TTL-based cleanup for entries older than the finality horizon. The `HeightPlaceholderKey` allocations suggest the key string is being heap-allocated on every lookup — consider interning or using a numeric key directly.

**Files:**
- `block/internal/cache/generic_cache.go`
- `block/internal/cache/manager.go`

**Acceptance criteria:**
- Live heap from `block/internal/cache.*` stays below 100 MB at steady state on a running chain
- `setSeen` / `setDAIncluded` entry count is bounded (log or metric exposed)

---

### Issue 7 — `go-header Hash.String` retaining 302 MB

**Severity:** High — #4 live-heap consumer in 2026-04-03 profile.

**Location:** `go-header/p2p.ExchangeServer.handleRangeRequest` → somewhere caching `Hash.String()` results.

`celestiaorg/go-header.Hash.String()` hex-encodes a `[]byte` hash into a new `string` on every call. 302 MB of live strings means these are being stored (likely as map keys or log fields) and retained. This is separate from `Header.Hash()` computation — memoization in #3219 avoids recomputing the hash but doesn't prevent `String()` from allocating a new hex string each time it is called.

**Fix options:**
1. If strings are used as map keys: store the raw `Hash` (`[]byte`) as a fixed-size `[32]byte` key instead of a hex string. Eliminates the allocation entirely.
2. If strings are used for logging: use `%x` in a format call rather than pre-allocating the string; zerolog's hex field support avoids allocation.
3. If strings must be cached: add `cachedString string` alongside `cachedHash` on `Header` and memoize via a `Hash.MemoizeString()` approach.

**Files:**
- Investigate callers of `Hash.String()` in `block/internal/cache/`, `go-header/p2p/`, and any map keyed by hash string.

**Acceptance criteria:**
- `go-header.Hash.String` disappears from the top-10 live-heap consumers
- No `string(hex.Encode(...))` patterns in hot paths that retain the result

---

## Implementation Sequence

| Phase | Issue | Effort | Expected impact | Status |
|-------|-------|--------|-----------------|--------|
| 1 | `block/internal/cache` eviction bound | 3h | ~1.24 GB → < 100 MB live heap | ⬜ Open (Issue 6) |
| 2 | `Hash.String` retention | 2h | ~302 MB freed | ⬜ Open (Issue 7) |
| 3 | LRU cache defaults + config exposure | 2h | Prevents future regression | ⬜ Open (Issue 1) |
| 4 | Badger `IndexCacheSize` cap | — | Bounds index RAM to ~256 MB | ✅ Done (#3209, Issue 2) |
| 5 | Disable gzip on EL HTTP transport | 1h | ~24% drop in allocation rate | ⬜ Open (Issue 5) |
| 6 | `filterTransactions` hex pool | 2h | ~40% drop in EVM RPC allocs | ⬜ Open (Issue 4) |
| 7 | Proto marshal `sync.Pool` | 4h | Revisit — may no longer be critical | ⬜ Open (Issue 3) |

The ordering above reflects the 2026-04-03 profile. Issue 6 (`block/internal/cache`) is now the highest-leverage fix — it is the #1 live-heap consumer and the root cause of continued RAM growth. Issue 3 (proto pool) should be re-profiled before investing effort; proto marshal no longer appears in the top allocators.

---

## Risks

| Risk | Mitigation |
|------|------------|
| `block/internal/cache` eviction causes re-fetch on cache miss | Profile syncer throughput before/after; ensure DA and P2P retrieval paths tolerate misses without stalling |
| Proto pool buffer escapes | Mandatory `copy` before return; `-race` on all marshal tests |
| LRU reduction slows syncing | Benchmark `BenchmarkSyncerIO` before/after; bump default to 8,192 if regression observed |
| Badger index cap causes cold-read latency | 256 MB is already set — expose via config; document NVMe vs HDD trade-off |
| `DisableCompression` breaks remote EL | Guard behind a config flag; default to `false` for remote URLs, `true` for local |

---

## Running the Benchmarks

```bash
# Phase 1 — store cache
go test -bench=. -benchmem -count=6 ./pkg/store/...

# Phase 2 — badger write/read throughput
go test -bench=BenchmarkStore_WriteThenRead -benchmem -count=6 ./pkg/store/...

# Phase 3 — proto marshal
go test -bench=BenchmarkP2PData_MarshalBinary -benchmem -count=6 -race ./types/...

# Phase 4 — hex encoding
go test -bench=BenchmarkFilterTransactions -benchmem -count=6 ./execution/evm/...

# Full regression
go test ./... -count=1
```

Use `benchstat` to compare before/after:

```bash
go test -bench=. -benchmem -count=10 ./pkg/store/... > before.txt
# apply changes
go test -bench=. -benchmem -count=10 ./pkg/store/... > after.txt
benchstat before.txt after.txt
```
