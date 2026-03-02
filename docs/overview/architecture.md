# Architecture

Evolve uses a modular architecture where each component has a well-defined interface and can be swapped independently. This document provides an overview of how the pieces fit together.

## System Overview

```text
┌─────────────────────────────────────────────────────────────────┐
│                          Client Apps                            │
│                    (wallets, dapps, indexers)                   │
└─────────────────────────────┬───────────────────────────────────┘
                              │ JSON-RPC / gRPC
┌─────────────────────────────▼───────────────────────────────────┐
│                           ev-node                               │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌──────────────┐  │
│  │  Block    │  │ Sequencer │  │    P2P    │  │    Sync      │  │
│  │ Components│  │           │  │  Network  │  │   Services   │  │
│  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘  └───────┬──────┘  │
└────────┼──────────────┼──────────────┼────────────────┼─────────┘
         │              │              │                │
         │ Executor     │ Sequencer    │ libp2p        │ DA Client
         ▼              ▼              ▼                ▼
┌────────────────┐ ┌──────────┐ ┌─────────────────────────────────┐
│    Executor    │ │Sequencer │ │          DA Layer               │
│  (ev-reth or   │ │(single,  │ │         (Celestia)              │
│   ev-abci)     │ │ based)   │ │                                 │
└────────────────┘ └──────────┘ └─────────────────────────────────┘
```

## Core Design Principles

1. **Zero-dependency core** — The `core/` package contains only interfaces with no external dependencies. This keeps the API stable and allows any implementation.

2. **Modular components** — Executor, Sequencer, and DA layer are all pluggable. Swap them without changing ev-node.

3. **Separation of concerns** — Block production, syncing, and DA submission run as independent components that communicate through well-defined channels.

4. **Two operating modes** — Nodes run as either an Aggregator (produces blocks) or Sync-only (follows chain).

## Block Components

The block package is the heart of ev-node. It's organized into specialized components:

| Component | Responsibility | Runs On |
|-----------|---------------|---------|
| **Executor** | Produces blocks by getting batches from sequencer and executing via execution layer | Aggregator only |
| **Reaper** | Scrapes transactions from execution layer mempool and submits to sequencer | Aggregator only |
| **Syncer** | Coordinates block sync from DA layer and P2P network | All nodes |
| **Submitter** | Submits blocks to DA layer and tracks inclusion | Aggregator only |
| **Cache** | Manages in-memory state for headers, data, and pending submissions | All nodes |

### Component Interaction

```text
                    ┌─────────────┐
                    │   Reaper    │
                    │  (tx scrape)│
                    └──────┬──────┘
                           │ Submit batch
                           ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Sequencer  │◄───│  Executor   │───►│ Broadcaster │
│             │    │(block prod) │    │   (P2P)     │
└─────────────┘    └──────┬──────┘    └─────────────┘
                          │
                          │ Queue for submission
                          ▼
                   ┌─────────────┐
                   │  Submitter  │───► DA Layer
                   │             │
                   └──────┬──────┘
                          │
                          │ Track inclusion
                          ▼
                   ┌─────────────┐
                   │    Cache    │
                   └─────────────┘
```

## Node Types

Evolve supports several node configurations:

| Type | Block Production | Full Validation | DA Submission | Use Case |
|------|-----------------|-----------------|---------------|----------|
| **Aggregator** | Yes | Yes | Yes | Block producer (sequencer) |
| **Full Node** | No | Yes | No | RPC provider, validator |
| **Light Node** | No | Headers only | No | Mobile, embedded clients |
| **Attester** | No | Yes | No | Soft consensus participant |

### Aggregator

The aggregator (also called sequencer node) produces blocks:

1. Reaper collects transactions from execution layer
2. Executor gets ordered batch from sequencer
3. Executor calls execution layer to process transactions
4. Executor creates and signs block (header + data)
5. Broadcaster gossips block to P2P network
6. Submitter queues block for DA submission

### Full Node

Full nodes sync and validate without producing blocks:

1. Syncer receives blocks from DA layer and/or P2P
2. Validates header signatures and data hashes
3. Executes transactions via execution layer
4. Verifies resulting state root matches header
5. Persists validated blocks to local store

## Data Flow

### Block Production (Aggregator)

```text
User Tx → Execution Layer Mempool
                 │
                 ▼
         Reaper scrapes txs
                 │
                 ▼
         Sequencer orders batch
                 │
                 ▼
         Executor.ExecuteTxs()
                 │
                 ├──► SignedHeader + Data
                 │
                 ├──► P2P Broadcast (soft confirmation)
                 │
                 └──► Submitter Queue
                           │
                           ▼
                      DA Layer (hard confirmation)
```

### Block Sync (Non-Aggregator)

```text
┌────────────────────────────────────────┐
│              Syncer                    │
├────────────┬────────────┬──────────────┤
│ DA Worker  │ P2P Worker │Forced Incl.  │
│            │            │   Worker     │
└─────┬──────┴─────┬──────┴───────┬──────┘
      │            │              │
      └────────────┴──────────────┘
                   │
                   ▼
           processHeightEvent()
                   │
                   ▼
           Validate → Execute → Persist
```

## P2P Network

Built on libp2p with:

- **GossipSub** for transaction and block propagation
- **Kademlia DHT** for peer discovery
- **Topics**: `{chainID}-tx`, `{chainID}-header`, `{chainID}-data`

Nodes discover peers through:

1. Bootstrap/seed nodes
2. DHT peer exchange
3. PEX (peer exchange protocol)

## Storage

ev-node uses a key-value store (badger) for:

- **Headers** — Indexed by height and hash
- **Data** — Transaction lists indexed by height
- **State** — Last committed height, app hash, DA height
- **Pending** — Blocks awaiting DA inclusion

## Further Reading

- [Block Lifecycle](/concepts/block-lifecycle) — Detailed block processing flow
- [Sequencing](/concepts/sequencing) — How transaction ordering works
- [Data Availability](/concepts/data-availability) — DA layer integration
- [Executor Interface](/reference/interfaces/executor) — Full interface reference
