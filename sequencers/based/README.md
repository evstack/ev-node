# Based Sequencer

## Overview

The Based Sequencer is a sequencer implementation that exclusively retrieves transactions from the Data Availability (DA) layer via the forced inclusion mechanism. Unlike other sequencer types, it does not accept transactions from a mempool or reaper - it treats the DA layer as a transaction queue.

This design ensures that all transactions are force-included from DA, making the sequencer completely "based" on the DA layer's transaction ordering.

## Architecture

### Core Components

1. **ForcedInclusionRetriever**: Fetches transactions from DA at epoch boundaries
2. **CheckpointStore**: Persists processing position to enable crash recovery
3. **BasedSequencer**: Orchestrates transaction retrieval and batch creation

### Key Interfaces

The Based Sequencer implements the `Sequencer` interface from `core/sequencer.go`:

- `SubmitBatchTxs()` - No-op for based sequencer (transactions are not accepted)
- `GetNextBatch()` - Retrieves the next batch from DA via forced inclusion
- `VerifyBatch()` - Always returns true (all transactions come from DA)

## Epoch-Based Transaction Retrieval

### How Epochs Work

Transactions are retrieved from DA in **epochs**, not individual DA blocks. An epoch is a range of DA blocks defined by `DAEpochForcedInclusion` in the genesis configuration.

**Example**: If `DAStartHeight = 100` and `DAEpochForcedInclusion = 10`:
- Epoch 1: DA heights 100-109
- Epoch 2: DA heights 110-119
- Epoch 3: DA heights 120-129

### Epoch Boundary Fetching

The `ForcedInclusionRetriever` only returns transactions when queried at the **epoch end** (the last DA height in an epoch):

```go
// When NOT at epoch end -> returns empty transactions
if daHeight != epochEnd {
    return &ForcedInclusionEvent{
        StartDaHeight: daHeight,
        EndDaHeight:   daHeight,
        Txs:           [][]byte{},
    }, nil
}

// When AT epoch end -> fetches entire epoch
// Retrieves ALL transactions from epochStart to epochEnd (inclusive)
```

When at an epoch end, the retriever fetches transactions from **all DA blocks in that epoch**:

1. Fetches forced inclusion blobs from `epochStart`
2. Fetches forced inclusion blobs from each height between start and end
3. Fetches forced inclusion blobs from `epochEnd`
4. Returns all transactions as a single `ForcedInclusionEvent`

### Why Epoch-Based?

- **Efficiency**: Reduces the number of DA queries
- **Batching**: Allows processing multiple DA blocks worth of transactions together
- **Determinism**: Clear boundaries for when to fetch from DA
- **Gas optimization**: Fewer DA reads means lower operational costs

## Checkpoint System

### Purpose

The checkpoint system tracks the exact position in the transaction stream to enable crash recovery and ensure no transactions are lost or duplicated.

### Checkpoint Structure

```go
type Checkpoint struct {
    // DAHeight is the DA block height currently being processed
    DAHeight uint64
    
    // TxIndex is the index of the next transaction to process
    // within the DA block's forced inclusion batch
    TxIndex uint64
}
```

### How Checkpoints Work

#### 1. Initial State
```
Checkpoint: (DAHeight: 100, TxIndex: 0)
- Ready to fetch epoch starting at DA height 100
```

#### 2. Fetching Transactions
When `GetNextBatch()` is called and we're at an epoch end:
```
Request: GetNextBatch(maxBytes: 1MB)
Action: Fetch all transactions from epoch (DA heights 100-109)
Result: currentBatchTxs = [tx1, tx2, tx3, ..., txN] (from entire epoch)
```

#### 3. Processing Transactions
Transactions are processed incrementally, respecting `maxBytes`:
```
Batch 1: [tx1, tx2] (fits in maxBytes)
Checkpoint: (DAHeight: 100, TxIndex: 2)

Batch 2: [tx3, tx4, tx5]
Checkpoint: (DAHeight: 100, TxIndex: 5)

... continue until all transactions from DA height 100 are consumed

Checkpoint: (DAHeight: 101, TxIndex: 0)
- Moved to next DA height within the same epoch
```

#### 4. Checkpoint Persistence

**Critical**: The checkpoint is persisted to disk **after every batch** of transactions is processed:

```go
if txCount > 0 {
    s.checkpoint.TxIndex += txCount
    
    // Move to next DA height when current one is exhausted
    if s.checkpoint.TxIndex >= uint64(len(s.currentBatchTxs)) {
        s.checkpoint.DAHeight++
        s.checkpoint.TxIndex = 0
        s.currentBatchTxs = nil
        s.SetDAHeight(s.checkpoint.DAHeight)
    }
    
    // Persist checkpoint to disk
    if err := s.checkpointStore.Save(ctx, s.checkpoint); err != nil {
        return nil, fmt.Errorf("failed to save checkpoint: %w", err)
    }
}
```

### Crash Recovery Behavior

#### Scenario: Crash Mid-Epoch

**Setup**:
- Epoch 1 spans DA heights 100-109
- At DA height 109, fetched all transactions from the epoch
- Processed transactions up to DA height 105, TxIndex 3
- **Crash occurs**

**On Restart**:

1. **Load Checkpoint**: `(DAHeight: 105, TxIndex: 3)`
2. **Lost Cache**: `currentBatchTxs` is empty (in-memory only)
3. **Attempt Fetch**: `RetrieveForcedIncludedTxs(105)`
4. **Result**: Empty (105 is not an epoch end)
5. **Continue**: Increment DA height, keep trying
6. **Eventually**: Reach DA height 109 (epoch end)
7. **Re-fetch**: Retrieve **entire epoch** again (DA heights 100-109)
8. **Resume**: Use checkpoint to skip already-processed transactions

#### Important Implications

**The entire epoch will be re-fetched after a crash**, even with fine-grained checkpoints.

**Why?**
- Transactions are only available at epoch boundaries
- In-memory cache (`currentBatchTxs`) is lost on restart
- Must wait until the next epoch end to fetch transactions again

**What the checkpoint prevents**:
- ✅ Re-execution of already processed transactions
- ✅ Correct resumption within a DA block's transaction list
- ✅ No transaction loss or duplication

**What the checkpoint does NOT prevent**:
- ❌ Re-fetching the entire epoch from DA
- ❌ Re-validation of previously fetched transactions

### Checkpoint Storage

The checkpoint is stored using a key-value datastore:

```go
// Checkpoint key in the datastore
checkpointKey = ds.NewKey("/based/checkpoint")

// Operations
checkpoint, err := checkpointStore.Load(ctx)    // Load from disk
err := checkpointStore.Save(ctx, checkpoint)    // Save to disk
err := checkpointStore.Delete(ctx)              // Delete from disk
```

The checkpoint is serialized using Protocol Buffers (`pb.BasedCheckpoint`) for efficient storage and cross-version compatibility.

## Transaction Processing Flow

### Full Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│ 1. GetNextBatch() called                                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Check: Do we have cached transactions?                   │
│    - currentBatchTxs empty OR all consumed?                 │
└──────────────────────┬──────────────────────────────────────┘
                       │ YES
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. fetchNextDABatch(checkpoint.DAHeight)                    │
│    - Calls RetrieveForcedIncludedTxs()                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. Is current DAHeight an epoch end?                        │
└──────┬──────────────────────────────────────┬───────────────┘
       │ NO                                    │ YES
       ▼                                       ▼
┌──────────────────┐              ┌──────────────────────────┐
│ Return empty     │              │ Fetch entire epoch from  │
│ transactions     │              │ DA (all heights in epoch)│
└──────┬───────────┘              └──────────┬───────────────┘
       │                                     │
       │                                     ▼
       │                          ┌──────────────────────────┐
       │                          │ Validate blob sizes      │
       │                          │ Cache in currentBatchTxs │
       │                          └──────────┬───────────────┘
       │                                     │
       └─────────────────┬───────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. createBatchFromCheckpoint(maxBytes)                      │
│    - Start from checkpoint.TxIndex                          │
│    - Add transactions until maxBytes reached                │
│    - Mark all as force-included                             │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. Update checkpoint                                         │
│    - checkpoint.TxIndex += len(batch.Transactions)          │
│    - If consumed all txs from current DA height:            │
│      * checkpoint.DAHeight++                                │
│      * checkpoint.TxIndex = 0                               │
│      * Clear currentBatchTxs cache                          │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. Persist checkpoint to disk                               │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. Return batch to executor for processing                  │
└─────────────────────────────────────────────────────────────┘
```

### Error Handling

**DA Height from Future**:
```go
if errors.Is(err, coreda.ErrHeightFromFuture) {
    // Stay at current position
    // Will retry on next call
    s.logger.Debug().Msg("DA height from future, waiting for DA to produce block")
    return nil
}
```

**Forced Inclusion Not Configured**:
```go
if errors.Is(err, block.ErrForceInclusionNotConfigured) {
    return errors.New("forced inclusion not configured")
}
```

**Invalid Blob Size**:
```go
if !seqcommon.ValidateBlobSize(tx) {
    s.logger.Warn().Msg("forced inclusion blob exceeds absolute maximum size - skipping")
    skippedTxs++
    continue
}
```

## Relationship with Executor

### DA Height Synchronization

The executor maintains a separate `DAHeight` field in the blockchain state:

```go
// In executor.go
newState.DAHeight = e.sequencer.GetDAHeight()

// State is saved after EVERY block
if err := batch.UpdateState(newState); err != nil {
    return fmt.Errorf("failed to update state: %w", err)
}
```

**Key Differences**:

| Aspect | Based Sequencer Checkpoint | Executor State |
|--------|---------------------------|----------------|
| **Frequency** | After every batch | After every block |
| **Granularity** | DAHeight + TxIndex | DAHeight only |
| **Purpose** | Track position in DA transaction stream | Track blockchain state |
| **Storage** | Checkpoint datastore | State datastore |
| **Scope** | Sequencer-specific | Chain-wide |

### Initialization Flow

On startup:

1. **Executor** loads State from disk
2. **Executor** calls `sequencer.SetDAHeight(state.DAHeight)`
3. **Based Sequencer** loads checkpoint from disk
4. If no checkpoint exists, initializes with current DA height
5. Both systems now synchronized

## Configuration

### Genesis Parameters

```go
type Genesis struct {
    // Starting DA height for the chain
    DAStartHeight uint64
    
    // Number of DA blocks per epoch for forced inclusion
    // Set to 0 to disable epochs (all blocks in epoch 1)
    DAEpochForcedInclusion uint64
}
```

### Example Configurations

**Frequent Fetching** (small epochs):
```go
DAStartHeight: 1000
DAEpochForcedInclusion: 1  // Fetch every DA block
```

**Batched Fetching** (larger epochs):
```go
DAStartHeight: 1000
DAEpochForcedInclusion: 100  // Fetch every 100 DA blocks
```

**Single Epoch** (no epoch boundaries):
```go
DAStartHeight: 1000
DAEpochForcedInclusion: 0  // All blocks in one epoch
```

## Performance Considerations

### Memory Usage

- `currentBatchTxs` holds all transactions from all DA heights in the current epoch
- With large epochs and many transactions, memory usage can be significant
- Example: Epoch size 100, 1000 txs/block, 1KB/tx = ~100MB

### DA Query Efficiency

**Pros**:
- Fewer DA queries (one per epoch instead of per block)
- Reduced DA layer costs

**Cons**:
- Longer wait times between transaction fetches
- Larger re-fetch overhead on crash recovery

### Crash Recovery Trade-offs

**Fine-grained checkpoints** (current approach):
- ✅ No transaction re-execution after crash
- ✅ Fast recovery within cached transactions
- ❌ Entire epoch re-fetched from DA
- ❌ All transactions re-validated

**Alternative** (epoch-level checkpoints):
- ✅ Simpler implementation
- ❌ All transactions in epoch re-executed after crash
- ❌ Longer recovery time

The current design prioritizes **no re-execution** over DA re-fetching, as execution is typically more expensive than fetching.

## Testing

### Unit Tests

- `checkpoint_test.go`: Tests checkpoint persistence operations
- `sequencer_test.go`: Tests sequencer batch retrieval logic

### Integration Testing

To test the based sequencer with a real DA layer:

```bash
# Run with based sequencer configuration
make run-n NODES=1 SEQUENCER_TYPE=based

# Simulate crash recovery
# 1. Stop node mid-epoch
# 2. Check checkpoint value
# 3. Restart node
# 4. Verify correct resumption
```

## Debugging

### Log Messages

**Checkpoint Loading**:
```
loaded based sequencer checkpoint from DB da_height=105 tx_index=3
```

**DA Fetching**:
```
fetching forced inclusion transactions from DA da_height=109
```

**Not at Epoch End**:
```
not at epoch end - returning empty transactions da_height=105 epoch_end=109
```

**Transactions Retrieved**:
```
fetched forced inclusion transactions from DA valid_tx_count=150 skipped_tx_count=2 da_height_start=100 da_height_end=109
```

**Checkpoint Resumption**:
```
resuming from checkpoint within DA block tx_index=3
```

### Common Issues

**Problem**: No transactions being processed
- **Check**: Are you at an epoch end? Transactions only arrive at epoch boundaries.
- **Check**: Is forced inclusion configured? Look for `ErrForceInclusionNotConfigured`.

**Problem**: Transactions re-executed after restart
- **Check**: Is checkpoint being persisted? Look for checkpoint save errors.
- **Check**: Is checkpoint being loaded on restart?

**Problem**: Slow recovery after crash
- **Cause**: Entire epoch is re-fetched from DA.
- **Solution**: Reduce epoch size for faster recovery (at cost of more DA queries).

## Future Improvements

### Potential Optimizations

1. **Persistent Transaction Cache**: Store fetched transactions on disk to avoid re-fetching entire epoch after crash
2. **Progressive Fetching**: Fetch DA blocks incrementally within an epoch instead of all at once
3. **Compression**: Compress checkpoint data for faster I/O
4. **Parallel Validation**: Validate transactions from multiple DA heights concurrently

### Design Alternatives

1. **Streaming Model**: Instead of epoch boundaries, stream transactions as DA blocks become available
2. **Hybrid Checkpointing**: Save both fine-grained position and transaction cache
3. **Two-Phase Commit**: Separate checkpoint updates from transaction processing for better crash consistency

## References

- Core interfaces: `core/sequencer.go`
- Forced inclusion: `block/internal/da/forced_inclusion_retriever.go`
- Epoch calculations: `types/epoch.go`
- Executor integration: `block/internal/executing/executor.go`
