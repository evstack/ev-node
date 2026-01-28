# ev-abci Overview

ev-abci is an ABCI adapter that allows Cosmos SDK applications to run on Evolve instead of CometBFT.

## What is ev-abci?

ev-abci provides:

- **Drop-in replacement** — Swap CometBFT for Evolve with minimal code changes
- **ABCI compatibility** — Your existing Cosmos SDK modules work unchanged
- **CometBFT RPC compatibility** — Existing clients and tooling continue to work
- **Migration tooling** — Migrate existing chains from CometBFT to Evolve

## Architecture

```
┌─────────────────────────────────────────┐
│            Your Cosmos App              │
│  ┌─────────────────────────────────┐    │
│  │     Cosmos SDK Modules          │    │
│  │  (bank, staking, gov, etc.)     │    │
│  └─────────────────────────────────┘    │
│                  │ ABCI                 │
│  ┌───────────────▼─────────────────┐    │
│  │           ev-abci               │    │
│  │   (ABCI adapter + RPC server)   │    │
│  └───────────────┬─────────────────┘    │
└──────────────────┼──────────────────────┘
                   │ Executor Interface
┌──────────────────▼──────────────────────┐
│              ev-node                    │
│       (consensus + DA + P2P)            │
└─────────────────────────────────────────┘
```

ev-abci implements the Executor interface, translating ev-node's calls into ABCI calls to your application.

## Key Differences from CometBFT

| Aspect | CometBFT | ev-abci |
|--------|----------|---------|
| Validators | Multiple validators with staking | Single sequencer |
| Consensus | BFT consensus rounds | Sequencer produces blocks |
| Finality | Instant (BFT) | Soft (P2P) → Hard (DA) |
| Block time | ~6s typical | Configurable (100ms+) |
| Vote extensions | Supported | Not supported |

## Benefits

- **No validator coordination** — Single sequencer eliminates consensus overhead
- **Faster blocks** — No BFT round-trips, blocks as fast as 100ms
- **DA-secured** — Security from data availability, not validator set
- **Simpler operations** — No validator management, slashing, or jailing

## Trade-offs

- **Single sequencer** — One node produces blocks (with forced inclusion for censorship resistance)
- **Different finality model** — Soft confirmation before DA finality
- **No vote extensions** — ABCI++ vote extensions not available

## Modules

ev-abci includes helper modules for migration:

- [Staking Wrapper](/ev-abci/modules/staking-wrapper) — Prevents validator updates during migration
- [Migration Manager](/ev-abci/modules/migration-manager) — Handles validator set transition

## Repository

- GitHub: [github.com/evstack/ev-abci](https://github.com/evstack/ev-abci)

## Next Steps

- [Cosmos SDK Quickstart](/getting-started/cosmos/quickstart) — Get started
- [Integration Guide](/ev-abci/integration-guide) — Manual integration
- [Migration from CometBFT](/ev-abci/migration-from-cometbft) — Migrate existing chain
