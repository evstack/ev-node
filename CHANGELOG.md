
# Changelog

<!--
All notable changes to this module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
-->

## [Unreleased]

### Added

<!-- New features or capabilities -->
- Added automated upgrade test for the `evm-single` app that verifies compatibility when moving from v1.0.0-beta.8 to HEAD in CI ([#2780](https://github.com/evstack/ev-node/pull/2780))
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
-

### Removed

<!-- Features that were removed -->
- Removed `LastCommitHash`, `ConsensusHash`, and `LastResultsHash` from the canonical header representation in favor of slim headers (BREAKING; legacy hashes now live under `Header.Legacy`) ([#2766](https://github.com/evstack/ev-node/pull/2766))

### Fixed

<!-- Bug fixes -->

### Security

<!-- Security vulnerability fixes -->
-

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
[Unreleased]: https://github.com/evstack/ev-node/compare/v1.0.0-beta.1...HEAD
