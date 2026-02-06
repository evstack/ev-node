# Based Sequencing

Based sequencing is a decentralized sequencing model where transaction ordering is determined by the base layer (Celestia) rather than a centralized sequencer. In this model, **every full node acts as its own proposer** by independently and deterministically deriving the next batch of transactions directly from the base layer.

## How Based Sequencing Works

### Transaction Submission

Users submit transactions to the base layer's forced inclusion namespace. These transactions are posted as blobs to the DA layer, where they become part of the canonical transaction ordering.

```text
User → Base Layer (DA) → Full Nodes retrieve and execute
```

### Deterministic Batch Construction

All full nodes independently construct identical batches by:

1. **Retrieving forced inclusion transactions** from the base layer at epoch boundaries
2. **Applying forkchoice rules** to determine batch composition:
   - `MaxBytes`: Maximum byte size per batch (respects block size limits)
   - DA epoch boundaries
3. **Smoothing large transactions** across multiple blocks when necessary

### Epoch-Based Processing

Forced inclusion transactions are retrieved in epochs defined by `DAEpochForcedInclusion`. For example, with an epoch size of 10:

- DA heights 100-109 form one epoch
- DA heights 110-119 form the next epoch
- Transactions from each epoch must be included before the epoch ends

Epochs durations determine the block time in based sequencing.
Additionally, because no headers are published, the lazy mode has no effect. The block time is a factor of the DA layer's block time.

## Block Smoothing

When forced inclusion transactions exceed the `MaxBytes` limit for a single block, they can be "smoothed" across multiple blocks within the same epoch. This ensures that:

- Large transactions don't block the chain
- All transactions are eventually included
- The system remains censorship-resistant

### Example

```text
Epoch [100, 104]:
  - Block 1: Includes 1.5 MB of forced inclusion txs (partial)
  - Block 2: Includes remaining 0.5 MB + new regular txs
  - All epoch transactions included before DA height 105
```

## Trust Assumptions

Based sequencing minimizes trust assumptions:

- **No trusted sequencer** - ordering comes from the base layer
- **No proposer selection** - every full node derives blocks independently
- **Deterministic consensus** - all honest nodes converge on the same chain
- **Base layer security** - inherits the security guarantees of the DA layer
- **No malicious actor concern** - invalid blocks are automatically rejected by validation rules

## Comparison with Single Sequencer

| Feature               | Based Sequencing              | Single Sequencer              |
| --------------------- | ----------------------------- | ----------------------------- |
| Decentralization      | ✅ Fully decentralized        | ❌ Single point of control    |
| Censorship Resistance | ✅ Guaranteed by base layer   | ⚠️ Guaranteed by base layer   |
| Latency               | ⚠️ Depends on DA layer (~12s) | ✅ Low latency (configurable) |
| Block Time Control    | ❌ Factor of DA block time    | ✅ Configurable by sequencer  |
| Trust Assumptions     | ✅ Minimal (only DA layer)    | ❌ Trust the sequencer        |

## Further Reading

- [Data Availability](../data-availability.md) - Understanding the DA layer
- [Transaction Flow](../transaction-flow.md) - How transactions move through the system
