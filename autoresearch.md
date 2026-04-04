# Autoresearch: Block package performance and allocation reduction

## Objective
Reduce memory allocations and improve performance of the block production hot path, specifically the `BlockProducer` interface methods (`ProduceBlock`, `CreateBlock`, `ApplyBlock`). The benchmark lives in `block/internal/executing/executor_benchmark_test.go`.

## Metrics
- **Primary**: `allocs_per_op` (allocs/op, lower is better) — allocation count in `BenchmarkProduceBlock/100_txs`
- **Secondary**: `bytes_per_op` (B/op, lower is better) — memory usage per operation
- **Secondary**: `ns_per_op` (ns/op, lower is better) — execution time

We target the 100_txs benchmark case because it has the most allocations (81 vs 71 for empty) and shows the clearest per-tx allocation pattern.

## How to Run
`./autoresearch.sh` — runs `go test -bench=BenchmarkProduceBlock/100_txs -benchmem -count=3` and reports median of the 3 runs with METRIC lines for allocs_per_op, bytes_per_op, and ns_per_op.

## Files in Scope
- `types/hashing.go` — hash computation; `sha256.New()` allocates ~213 bytes per call; called by `Data.Hash()` and `DACommitment()`
- `types/serialization.go` — `ToProto()`, `MarshalBinary()`, `txsToByteSlices()` all allocate new structs/slices every call
- `block/internal/executing/executor.go` — main block production; `ApplyBlock` converts `Txs` to `[][]byte` every time
- `types/data.go` — `Data` type; `DACommitment()` creates pruned Data allocation
- `types/header.go` — `Header` type
- `types/state.go` — `State` type, `NextState()` 

## Off Limits
- Protobuf definitions (`types/pb/`) — must not change wire format
- Test files except the benchmark
- Public API signatures (keep backward compatibility)

## Constraints
- All tests must pass (`just test ./block/... ./types/...`)
- No new external dependencies
- Must not change protobuf wire format

## What's Been Tried
Nothing yet — starting from baseline.

## Baseline (first run)
Benchmark: `BenchmarkProduceBlock/100_txs`
- **81 allocs/op** (PRIMARY)
- ~25,900 B/op
- ~33,000 ns/op

### Key allocation hotspots identified:
1. **`leafHashOpt()`** — `sha256.New()` allocates a new hash.Hash every call (~213B). Called by `Data.Hash()` and `DACommitment()` every block.
2. **`txsToByteSlices()`** — allocates new `[][]byte` slice every `Data.ToProto()` call.
3. **`Data.ToProto()` / `Header.ToProto()`** — allocate new protobuf structs every serialization.
4. **`DACommitment()`** — creates pruned `&Data{Txs: d.Txs}` allocation before hashing.
5. **`ApplyBlock`** — `make([][]byte, n)` for raw tx conversion every block.
6. **`Data.Hash()`** — allocates byte slice from `MarshalBinary()` + sha256.New().
7. **`Header.HashSlim()`** — same pattern.
