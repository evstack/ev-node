# Based Sequencer

## Overview

The Based Sequencer is a sequencer implementation that retrieves transactions exclusively from the Data Availability (DA) layer via the forced inclusion mechanism. Unlike traditional sequencers that accept transactions from mempools or external sources, the based sequencer only processes transactions that have been posted to the DA layer's forced inclusion namespace.

## What is a Based Sequencer?

A "based" sequencer (also known as "based rollup") is a rollup architecture where transaction ordering is derived entirely from the base layer (DA layer) rather than from a centralized sequencer. This provides several benefits:

- **Censorship Resistance**: Users can submit transactions directly to DA, bypassing the sequencer
- **Decentralization**: No single entity controls transaction ordering
- **Liveness**: The rollup continues operating as long as the DA layer is available
- **Trustless**: Users don't need to trust the sequencer to include their transactions

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Based Sequencer                          │
│                                                               │
│  ┌────────────────┐         ┌─────────────────┐            │
│  │  Transaction   │         │   DA Retriever  │            │
│  │     Queue      │◄────────│   (Interface)   │            │
│  └────────────────┘         └─────────────────┘            │
│         │                            │                       │
│         │                            │                       │
│         ▼                            ▼                       │
│  ┌────────────────┐         ┌─────────────────┐            │
│  │  GetNextBatch  │         │  Fetch Forced   │            │
│  │    (Method)    │────────►│  Inclusion Txs  │            │
│  └────────────────┘         └─────────────────┘            │
│                                      │                       │
└──────────────────────────────────────┼───────────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │   DA Layer      │
                              │  (Forced Inc.   │
                              │   Namespace)    │
                              └─────────────────┘
```

## Features

- **DA-Only Transaction Source**: Fetches transactions exclusively from DA forced inclusion namespace
- **Batch Size Management**: Respects MaxBytes limits when creating batches
- **Transaction Queue**: Buffers transactions when they exceed batch size limits
- **DA Height Tracking**: Maintains synchronization with DA layer height
- **Concurrent-Safe**: Thread-safe operations with mutex protection
- **Automatic Height Management**: Handles "height from future" errors gracefully

## Interface Compliance

The Based Sequencer implements the `core/sequencer.Sequencer` interface:

```go
type Sequencer interface {
    SubmitBatchTxs(ctx context.Context, req SubmitBatchTxsRequest) (*SubmitBatchTxsResponse, error)
    GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error)
    VerifyBatch(ctx context.Context, req VerifyBatchRequest) (*VerifyBatchResponse, error)
}
```

## Configuration

The based sequencer uses the following configuration from `config.Config`:

- `DA.ForcedInclusionNamespace`: Namespace for forced inclusion transactions
- `DA.ForcedInclusionDAEpoch`: Number of DA blocks to scan per fetch

If `ForcedInclusionNamespace` is not configured, the sequencer returns empty batches.

## Performance Considerations

- **Batching**: Transactions are batched to reduce DA queries
- **Queue**: In-memory queue prevents repeated DA fetches
- **Mutex Protection**: Thread-safe but may block on concurrent access
- **DA Epoch**: Configure `ForcedInclusionDAEpoch` to balance freshness vs. efficiency

## Comparison to Traditional Sequencer

| Feature               | Traditional Sequencer | Based Sequencer |
| --------------------- | --------------------- | --------------- |
| Transaction Source    | Mempool, RPC          | DA Layer Only   |
| Censorship Resistance | Low                   | High            |
| Centralization        | High                  | Low             |
| Latency               | Low                   | Higher          |
| MEV Opportunity       | High                  | Low             |
| Trust Requirements    | High                  | Low             |
