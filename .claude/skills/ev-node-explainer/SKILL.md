---
name: ev-node-explainer
description: Explains ev-node architecture, components, and internal workings. Use when the user asks how ev-node works, wants to understand the block package, needs architecture explanations, or asks about block production, syncing, DA submission, or forced inclusion.
---

# ev-node Architecture Explainer

ev-node is a sovereign rollup framework that allows building rollups on any Data Availability (DA) layer. It follows a modular architecture where components can be swapped.

## Core Principles

1. **Zero-dependency core** - `core/` contains only interfaces, no external deps
2. **Modular components** - Executor, Sequencer, DA are pluggable
3. **Two operating modes** - Aggregator (produces blocks) and Sync-only (follows chain)
4. **Separation of concerns** - Block production, syncing, and DA submission are independent

## Package Overview

| Package | Responsibility |
|---------|---------------|
| `core/` | Interfaces only (Executor, Sequencer) |
| `types/` | Data structures (Header, Data, State, SignedHeader) |
| `block/` | Block lifecycle management |
| `execution/` | Execution layer implementations (EVM, ABCI) |
| `node/` | Node initialization and orchestration |
| `pkg/p2p/` | libp2p-based networking |
| `pkg/store/` | Persistent storage |
| `pkg/da/` | DA layer abstraction |

## Block Package Deep Dive

The block package is the most complex part of ev-node. See [block-architecture.md](block-architecture.md) for the complete breakdown.

### Component Summary

```
Components struct:
├── Executor    - Block production (Aggregator only)
├── Reaper      - Transaction scraping (Aggregator only)
├── Syncer      - Block synchronization
├── Submitter   - DA submission and inclusion
└── Cache       - Unified state caching
```

### Entry Points

- `NewAggregatorComponents()` - Full node that produces and syncs blocks
- `NewSyncComponents()` - Non-aggregator that only syncs

### Key Data Types

**Header** - Block metadata (height, time, hashes, proposer)
**Data** - Transaction list with metadata
**SignedHeader** - Header with proposer signature
**State** - Chain state (last block, app hash, DA height)

## Block Production Flow (Aggregator)

```
Sequencer.GetNextBatch()
    │
    ▼
Executor.ExecuteTxs()
    │
    ├──► SignedHeader + Data
    │
    ├──► P2P Broadcast
    │
    └──► Submitter Queue
            │
            ▼
        DA Layer
```

## Block Sync Flow (Non-Aggregator)

```
┌─────────────────────────────────────┐
│           Syncer                     │
├─────────────┬─────────────┬─────────┤
│ DA Worker   │ P2P Worker  │ Forced  │
│             │             │ Incl.   │
└──────┬──────┴──────┬──────┴────┬────┘
       │             │           │
       └─────────────┴───────────┘
                  │
                  ▼
          processHeightEvent()
                  │
                  ▼
          ExecuteTxs → Update State
```

## Forced Inclusion

Forced inclusion prevents sequencer censorship:

1. User submits tx directly to DA layer
2. Syncer detects tx in forced-inclusion namespace
3. Grace period starts (adjusts based on block fullness)
4. If not included by sequencer within grace period → sequencer marked malicious
5. Tx gets included regardless

## Key Files

| File | Purpose |
|------|---------|
| `block/public.go` | Exported types and factories |
| `block/components.go` | Component creation |
| `block/internal/executing/executor.go` | Block production |
| `block/internal/syncing/syncer.go` | Sync orchestration |
| `block/internal/submitting/submitter.go` | DA submission |
| `block/internal/cache/manager.go` | Unified cache |

## Common Questions

### How does block production work?

The Executor runs `executionLoop()`:
1. Wait for block time or new transactions
2. Get batch from sequencer
3. Execute via execution layer
4. Create SignedHeader + Data
5. Broadcast to P2P
6. Queue for DA submission

### How does syncing work?

The Syncer coordinates three workers:
- **DA Worker** - Fetches confirmed blocks from DA
- **P2P Worker** - Receives gossiped blocks
- **Forced Inclusion** - Monitors for censored txs

All feed into `processHeightEvent()` which validates and executes.

### What happens if DA submission fails?

Submitter has retry logic with exponential backoff. Status codes:
- `TooBig` - Splits blob into chunks
- `AlreadyInMempool` - Skips (duplicate)
- `NotIncludedInBlock` - Retries with backoff
- `ContextCanceled` - Request canceled

### How is state recovered after crash?

The Replayer syncs execution layer from disk:
1. Load last committed height from store
2. Check execution layer height
3. Replay any missing blocks
4. Ensure consistency before starting

## Architecture Diagrams

For detailed component diagrams and state machines, see [block-architecture.md](block-architecture.md).
