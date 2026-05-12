# Divergence from Main Branch

This branch (`perf/block-optimization`) introduces breaking changes to maximize performance in the `block/` package. It is **not wire-compatible** with the main branch.

## 1. Combined Header+Data Blobs

**Main**: Headers and data are submitted as separate blobs to separate DA namespaces (`HeaderNamespace`, `DataNamespace`). On retrieval, the DA retriever fetches from both namespaces, decodes headers and data independently, then matches them by block height.

**This branch**: Headers and data are combined into a single blob using a custom binary encoding (`common.MarshalBlockBlob`/`UnmarshalBlockBlob`). Each blob contains the proto-encoded header, proto-encoded data, and the envelope signature, separated by length prefixes with a magic number prefix (`0x45564E44`). Only the `HeaderNamespace` is used.

### Why
- Eliminates matching overhead (no separate header/data pending maps)
- Halves DA submission round trips (one blob per block instead of two)
- Simplifies DA inclusion tracking (single check per block)
- Removes the `DAHeaderEnvelope` protobuf wrapper and the separate `SignedData` protobuf wrapper

## 2. Custom Binary Blob Encoding

**Main**: DA blobs use protobuf encoding (`DAHeaderEnvelope` for headers, `SignedData` for data). Each involves allocating proto message structs, converting Go types to proto types, and calling `proto.Marshal`.

**This branch**: The combined blob wrapper uses a custom binary format: `[magic 4B][header_len 4B][header_bytes][data_len 4B][data_bytes][sig_len 4B][sig_bytes]`. Individual header and data fields are still proto-encoded internally (hash computation requires it), but the envelope wrapper avoids all proto overhead.

### Why
- Zero allocation for the blob wrapper (direct length-prefixed binary)
- No proto message pool management for the envelope
- No `ToProto`/`FromProto` conversion for the DA envelope or signed data
- Simpler and faster encode/decode path

## 3. P2P Sync Removed

**Main**: Full nodes sync from both P2P (via `go-header` `HeaderSyncService`/`DataSyncService`) and DA. The executor broadcasts produced blocks to P2P peers. P2P events include DA height hints for targeted DA retrieval. The syncer runs a P2P worker loop alongside the DA follower.

**This branch**: All P2P sync is removed. Nodes sync exclusively from DA. No P2P broadcasting, no P2P stores, no P2P handler, no DA height hints.

### Removed code
- `syncing/p2p_handler.go` — entire file deleted
- `syncing/syncer_mock.go` — P2P handler mock deleted
- `common/expected_interfaces.go` — `HeaderP2PBroadcaster`/`DataP2PBroadcaster` types removed
- P2P broadcasting in `executing/executor.go` removed
- P2P worker loop in syncer removed
- `SourceP2P` event source removed
- `DaHeightHints` field removed from `DAHeightEvent`
- `headerStore`/`dataStore` parameters removed from `NewSyncer` and component constructors
- `headerSyncService`/`dataSyncService` parameters removed from aggregator component constructors
- `DAHintAppender` interface removed from DA submitter
- Separate `SubmitHeaders`/`SubmitData` replaced with single `SubmitBlocks`

### Why
- Removes network overhead from P2P gossip
- Eliminates the complexity of two sync sources competing
- DA is the single source of truth, reducing consistency issues
- Removes libp2p dependency from the block package's hot path
- Simplifies the syncer from 3 worker loops to 2 (process loop + pending worker)

## 4. DA Submitter Simplified

**Main**: `DASubmitterAPI` has two methods: `SubmitHeaders` and `SubmitData`, each with separate batching strategies, mutex locks, and retry loops.

**This branch**: `DASubmitterAPI` has a single `SubmitBlocks` method that takes headers and data together, creates combined blobs, signs them, and submits to a single namespace. One batching strategy, one mutex, one retry loop.

### Why
- Halves the submission loop complexity
- Eliminates the envelope cache (no more retry-signing concern)
- Single retry loop instead of two
- Combined blobs submitted atomically — no partial header-without-data states

## Migration Notes

- Existing DA data from main branch is **not readable** by this branch (different blob format)
- This branch requires a fresh start or a migration tool
- The `P2PSignedHeader` and `P2PData` types still exist in `types/` but are no longer used by the block package
- External consumers of `NewSyncComponents` and `NewAggregatorWithCatchupComponents` must update their call sites
