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

The Based Sequencer implements the `Sequencer` interface from `core/sequencer/sequencing.go`:

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

### Why Epoch-Based

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

```bash
Checkpoint: (DAHeight: 100, TxIndex: 0)
- Ready to fetch epoch starting at DA height 100
```

#### 2. Fetching Transactions

When `GetNextBatch()` is called and we're at an epoch end:

```bash
Request: GetNextBatch(maxBytes: 1MB)
Action: Fetch all transactions from epoch (DA heights 100-109)
Result: currentBatchTxs = [tx1, tx2, tx3, ..., txN] (from entire epoch)
```

#### 3. Processing Transactions

Transactions are processed incrementally, respecting `maxBytes`:

```bash
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

The checkpoint is serialized using Protocol Buffers (`pb.SequencerDACheckpoint`) for efficient storage and cross-version compatibility.
