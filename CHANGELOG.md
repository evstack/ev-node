# Changelog

<!--
All notable changes to this module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
-->

## [Unreleased]

### Added

- Node pruning support. [#2984](https://github.com/evstack/ev-node/pull/2984)
  - Two different sort of pruning implemented:
    _Classic pruning_ (`all`): prunes given `HEAD-n` blocks from the databases, including store metadatas.
    _Auto Storage Optimization_ (`metadata`): prunes only the state metadatas, keeps all blocks.
    By using one or the other, you are losing the ability to rollback or replay transactions earlier than `HEAD-n`.
    When using _classic pruning_, you aren't able to fetch blocks prior to `HEAD-n`.

## v1.0.0-rc.4

### Changes

- Skip draining when exec client unavailable. [#3060](https://github.com/evstack/ev-node/pull/3060)

## v1.0.0-rc.3

### Added

- Add DA Hints for P2P transactions. This allows a catching up node to be on sync with both DA and P2P. ([#2891](https://github.com/evstack/ev-node/pull/2891))

### Changes

- Improve `cache.NumPendingData` to not return empty data. Automatically bumps `LastSubmittedHeight` to reflect that. ([#3046](https://github.com/evstack/ev-node/pull/3046))
- **BREAKING** Make pending events cache and tx cache fully ephemeral. Those will be re-fetched on restart. DA Inclusion cache persists until cleared up after DA inclusion has been processed. Persist accross restart using store metadata. ([#3047](https://github.com/evstack/ev-node/pull/3047))
- Replace LRU cache by standard mem cache with manual eviction in `store_adapter`. When P2P blocks were fetched too fast, they would be evicted before being executed [#3051](https://github.com/evstack/ev-node/pull/3051)
- Fix replay logic leading to app hashes by verifying against the wrong block [#3053](https://github.com/evstack/ev-node/pull/3053).

## v1.0.0-rc.2

### Changes

- Improve cache handling when there is a significant backlog of pending headers and data. ([#3030](https://github.com/evstack/ev-node/pull/3030))
- Decrease MaxBytesSize to `5MB` to increase compatibility with public nodes. ([#3030](https://github.com/evstack/ev-node/pull/3030))
- Proper counting of `DASubmitterPendingBlobs` metrics. [#3038](https://github.com/evstack/ev-node/pull/3038)
- Replace `go-header` store by `ev-node` store. This avoid duplication of all blocks in `go-header` and `ev-node` store. Thanks to the cached store from #3030, this should improve p2p performance as well. [#3036](https://github.com/evstack/ev-node/pull/3036)

## v1.0.0-rc.1

### Added

- Added OpenTelemetry tracing support with OTLP export for distributed tracing across ev-node components including block production, syncing, DA submission/retrieval, sequencer, store operations, and RPC layer. Configurable via `instrumentation.tracing`, `instrumentation.tracing_endpoint`, `instrumentation.tracing_service_name`, and `instrumentation.tracing_sample_rate` settings. ([#2956](https://github.com/evstack/ev-node/issues/2956))
- **BREAKING:** Implement forced inclusion and batch sequencing ([#2797](https://github.com/evstack/ev-node/pull/2797))
  **This change requires adding a `da_epoch_forced_inclusion` field to the node's `genesis.json` file.** The recommended value is `100`.
  Full support for this feature will be available in a future release.
- Added `post-tx` command and force inclusion server to submit transactions directly to the DA layer. ([#2888](https://github.com/evstack/ev-node/pull/2888))
  Additionally, modified the core package to support marking transactions as forced included transactions.
  The execution client ought to perform basic validation on those transactions as they have skipped the execution client's mempool.
- Added batching strategies (default stay time-based, unchanged from previous betas). Currently available strategies are `time`, `size`, `immediate` and `adaptive`. [Full documentation can be found here](https://github.com/evstack/ev-node/blob/122486de98d09ecd37d792b88814dcf07238f28a/docs/learn/config.md?plain=1#L521-L597).
- Added `FilterTxs` method to the execution interface. This method is meant to filter txs by size and if the execution clients allows it, by gas. This is useful for force included transactions, as those aren't filtered by the sequencer's mempool.
- Added `GetExecutionInfo` method to the execution interface. This method returns some execution information, such as the maximum gas per block.

### Changed

- **BREAKING:** Renamed `evm-single` to `evm` and `grpc-single` to `evgrpc` for clarity. [#2839](https://github.com/evstack/ev-node/pull/2839). You may need to manually modify your evnode.yaml `signer.signer_path` if your $HOME folder is changed.
- Split cache interface into `CacheManager` and `PendingManager` and created `da` client to easy DA handling. [#2878](https://github.com/evstack/ev-node/pull/2878)
- Improved startup DA retrieval height when cache is cleared or empty. [#2880](https://github.com/evstack/ev-node/pull/2880)

### Removed

- **BREAKING:** Removed unused and confusing metrics from sequencers and block processing, including sequencer-specific metrics (gas price, blob size, transaction status, pending blocks), channel buffer metrics, overly granular error metrics, block production categorization metrics, and sync lag metrics. Essential metrics for DA submission health, block production, and performance monitoring are retained. [#2904](https://github.com/evstack/ev-node/pull/2904)
- **BREAKING**: Removed `core/da` package and replaced DAClient with internal implementation. The DA client is exposed as `block.FullDAClient`, `block.DAClient`, `block.DAVerifier` without leaking implementation details. [#2910](https://github.com/evstack/ev-node/pull/2910)

## v1.0.0-beta.11

### Improvements

- Loosen syncer validation for allowing swapping sequencer and full node state [#2925](https://github.com/evstack/ev-node/pull/2925)

## v1.0.0-beta.10

### Added

- Enhanced health check system with separate liveness (`/health/live`) and readiness (`/health/ready`) HTTP endpoints. Readiness endpoint includes P2P listening check and aggregator block production rate validation (5x block time threshold). ([#2800](https://github.com/evstack/ev-node/pull/2800))
- Added `GetP2PStoreInfo` RPC method to retrieve head/tail metadata for go-header stores used by P2P sync ([#2835](https://github.com/evstack/ev-node/pull/2835))
- Added protobuf definitions for `P2PStoreEntry` and `P2PStoreSnapshot` messages to support P2P store inspection

### Changed

- Improved EVM execution client payload status validation with proper retry logic for SYNCING states in `InitChain`, `ExecuteTxs`, and `SetFinal` methods. The implementation now follows Engine API specification by retrying SYNCING/ACCEPTED status with exponential backoff and failing immediately on INVALID status, preventing unnecessary node shutdowns during transient execution engine sync operations. ([#2863](https://github.com/evstack/ev-node/pull/2863))
- Remove GasPrice and GasMultiplier from DA interface and configuration to use celestia-node's native fee estimation. ([#2822](https://github.com/evstack/ev-node/pull/2822))
- Use cache instead of in memory store for reaper. Persist cache on reload. Autoclean after 24 hours. ([#2811](https://github.com/evstack/ev-node/pull/2811))
- Improved P2P sync service store initialization to be atomic and prevent race conditions ([#2838](https://github.com/evstack/ev-node/pull/2838))
- Enhanced P2P bootstrap behavior to intelligently detect starting height from local store instead of requiring trusted hash
- Relaxed execution layer height validation in block replay to allow execution to be ahead of target height, enabling recovery from manual intervention scenarios

### Removed

- **BREAKING:** Removed `evnode.v1.HealthService` gRPC endpoint. Use HTTP endpoints: `GET /health/live` and `GET /health/ready`. ([#2800](https://github.com/evstack/ev-node/pull/2800))
- **BREAKING:** Removed `TrustedHash` configuration option and `--evnode.node.trusted_hash` flag. Sync service now automatically determines starting height from local store state ([#2838](https://github.com/evstack/ev-node/pull/2838))
- **BREAKING:** Removed unused and confusing metrics from sequencers and block processing, including sequencer-specific metrics (gas price, blob size, transaction status, pending blocks), channel buffer metrics, overly granular error metrics, block production categorization metrics, and sync lag metrics. Essential metrics for DA submission health, block production, and performance monitoring are retained. [#2904](https://github.com/evstack/ev-node/pull/2904)

### Fixed

- Fixed sync service initialization issue when node is not on genesis but has an empty store

## v1.0.0-beta.9

### Added

<!-- New features or capabilities -->

- Added automated upgrade test for the `evm` app that verifies compatibility when moving from v1.0.0-beta.8 to HEAD in CI ([#2780](https://github.com/evstack/ev-node/pull/2780))
- Added execution-layer replay mechanism so nodes can resynchronize by replaying missed batches against the executor ([#2771](https://github.com/evstack/ev-node/pull/2771))
- Added cache-pruning logic that evicts entries once heights are finalized to keep node memory usage bounded ([#2761](https://github.com/evstack/ev-node/pull/2761))
- Added Prometheus gauges and counters that surface DA submission failures, pending blobs, and resend attempts for easier operational monitoring ([#2756](https://github.com/evstack/ev-node/pull/2756))
- Added gRPC execution client implementation for remote execution services using Connect-RPC protocol ([#2490](https://github.com/evstack/ev-node/pull/2490))
- Added `ExecutorService` protobuf definition with InitChain, GetTxs, ExecuteTxs, and SetFinal RPCs ([#2490](https://github.com/evstack/ev-node/pull/2490))
- Added new `grpc` app for running EVNode with a remote execution layer via gRPC ([#2490](https://github.com/evstack/ev-node/pull/2490))

### Changed

<!-- Changes to existing functionality -->

- Hardened signer CLI and block pipeline per security audit: passphrases must be provided via `--evnode.signer.passphrase_file`, JWT secrets must be provided via `--evm.jwt-secret-file`, data/header validation enforces metadata and timestamp checks, and the reaper backs off on failures (BREAKING) ([#2764](https://github.com/evstack/ev-node/pull/2764))
- Added retries around executor `ExecuteTxs` calls to better tolerate transient execution errors ([#2784](https://github.com/evstack/ev-node/pull/2784))
- Increased default `ReadinessMaxBlocksBehind` from 3 to 30 blocks so `/health/ready` stays true during normal batch sync ([#2779](https://github.com/evstack/ev-node/pull/2779))
- Updated EVM execution client to use new `txpoolExt_getTxs` RPC API for retrieving pending transactions as RLP-encoded bytes

### Deprecated

<!-- Features that will be removed in future versions -->

### Removed

<!-- Features that were removed -->

- Removed `LastCommitHash`, `ConsensusHash`, and `LastResultsHash` from the canonical header representation in favor of slim headers (BREAKING; legacy hashes now live under `Header.Legacy`) ([#2766](https://github.com/evstack/ev-node/pull/2766))

### Fixed

<!-- Bug fixes -->

### Security

<!-- Security vulnerability fixes -->

<!--
## Category Guidelines:

### Added
- New features
- New APIs
- New configuration options
- New commands
- New integrations

### Changed
- API changes (breaking or non-breaking)
- Behavior changes
- Performance improvements
- Refactoring (only if it affects users)
- Documentation updates (major ones)
- Default value changes

### Deprecated
- Features planned for removal
- Old APIs being phased out
- Configuration options being replaced

### Removed
- Deleted features
- Removed APIs
- Removed configuration options
- Removed dependencies

### Fixed
- Bug fixes
- Crash fixes
- Memory leak fixes
- Race condition fixes
- Incorrect behavior fixes

### Security
- Security vulnerability patches
- Security hardening
- Authentication/authorization fixes
- Cryptographic updates

## Writing Good Changelog Entries:

DO:
- Start with a verb (Added, Fixed, Changed, etc.)
- Include PR number: "Fixed memory leak in block sync (#1234)"
- Be concise but descriptive
- Focus on WHAT changed and WHY it matters to users
- Group related changes

DON'T:
- Include internal refactoring that doesn't affect users
- Use technical jargon without explanation
- Write from developer perspective
- Include every minor change

## Version Numbering:

Given a version number MAJOR.MINOR.PATCH:

- MAJOR: Incompatible API changes
- MINOR: Backwards-compatible functionality additions
- PATCH: Backwards-compatible bug fixes

Pre-release versions: 0.x.y (anything may change)
-->

<!-- Links -->

- [Unreleased]: https://github.com/evstack/ev-node/compare/v1.0.0-beta.1...HEAD
