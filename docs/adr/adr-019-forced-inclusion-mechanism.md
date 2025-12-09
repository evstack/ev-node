# ADR 019: Forced Inclusion Mechanism

## Changelog

- 2025-03-24: Initial draft
- 2025-04-23: Renumbered from ADR-018 to ADR-019 to maintain chronological order.
- 2025-11-10: Updated to reflect actual implementation
- 2025-12-09: Added documentation for ForcedInclusionGracePeriod parameter

## Context

In a single-sequencer rollup architecture, users depend entirely on the sequencer to include their transactions in blocks. This creates several problems:

1. **Censorship Risk**: A malicious or coerced sequencer can selectively exclude transactions
2. **Liveness Failure**: If the sequencer goes offline, no new transactions can be processed
3. **Centralization**: Users must trust a single entity to behave honestly
4. **No Recourse**: Users have no alternative path to submit transactions if the sequencer refuses them

While eventual solutions like decentralized sequencer networks exist, they introduce significant complexity. We need a simpler mechanism that provides censorship resistance and liveness guarantees while maintaining the performance benefits of a single sequencer.

## Alternative Approaches

### Decentralized Sequencer

A fully decentralized sequencer network would eliminate single points of failure but requires:

- Complex consensus mechanisms
- Increased latency due to coordination
- More infrastructure and operational complexity

### Automatic Sequencer Failover

Implementing automatic failover to backup sequencers when the primary goes down requires:

- Complex monitoring and health checks
- Coordination between sequencers to prevent forks
- Does not solve censorship issues with a malicious sequencer

## Decision

We implement a **forced inclusion mechanism** that allows users to submit transactions directly to the Data Availability (DA) layer. This approach provides:

1. **Censorship Resistance**: Users can always bypass the sequencer by posting to DA
2. **Verifiable Inclusion**: Full nodes verify that sequencers include all forced transactions
3. **Based Rollup Option**: A based sequencer mode for fully DA-driven transaction ordering
4. **Simplicity**: No complex timing mechanisms or fallback modes

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Actions                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Normal Path:                    Forced Inclusion Path:         │
│  Submit tx to Sequencer  ────►   Submit tx directly to DA       │
│       (Fast)                          (Censorship-resistant)     │
│                                                                  │
└──────────┬────────────────────────────────────┬─────────────────┘
           │                                     │
           ▼                                     ▼
    ┌─────────────┐                    ┌──────────────────┐
    │  Sequencer  │                    │    DA Layer      │
    │  (Mempool)  │                    │ (Forced Inc. NS) │
    └──────┬──────┘                    └─────────┬────────┘
           │                                     │
           │  1. Fetch forced inc. txs           │
           │◄────────────────────────────────────┘
           │
           │  2. Prepend forced txs to batch
           │
           ▼
    ┌─────────────┐
    │    Block    │
    │  Production │
    └──────┬──────┘
           │
           │  3. Submit block to DA
           │
           ▼
    ┌─────────────┐
    │  DA Layer   │
    └──────┬──────┘
           │
           │  4. Full nodes retrieve block
           │
           ▼
    ┌─────────────────────┐
    │   Full Nodes        │
    │  (Verification)     │
    │                     │
    │  5. Verify forced   │
    │     inc. txs are    │
    │     included        │
    └─────────────────────┘
```

### Key Components

1. **Forced Inclusion Namespace**: A dedicated DA namespace where users can post transactions
2. **DA Retriever**: Fetches forced inclusion transactions from DA using epoch-based scanning
3. **Single Sequencer**: Enhanced to include forced transactions from DA in every batch
4. **Based Sequencer**: Alternative sequencer that ONLY retrieves transactions from DA
5. **Verification**: Full nodes validate that blocks include all forced transactions

## Detailed Design

### User Requirements

Users can submit transactions in two ways:

1. **Normal Path**: Submit to sequencer's mempool/RPC (fast, low cost)
2. **Forced Inclusion Path**: Submit directly to DA forced inclusion namespace (censorship-resistant)

No additional requirements or monitoring needed from users.

### Systems Affected

1. **DA Layer**: New namespace for forced inclusion transactions
2. **Sequencer (Single)**: Fetches and includes forced transactions
3. **Sequencer (Based)**: New sequencer type that only uses DA transactions
4. **DA Retriever**: New component for fetching forced transactions
5. **Syncer**: Verifies forced transaction inclusion in blocks
6. **Configuration**: New fields for forced inclusion settings

### Data Structures

#### Forced Inclusion Event

```go
type ForcedIncludedEvent struct {
    Txs           [][]byte  // Forced inclusion transactions
    StartDaHeight uint64    // Start of DA height range
    EndDaHeight   uint64    // End of DA height range
}
```

#### DA Retriever Interface

```go
type DARetriever interface {
    // Retrieve forced inclusion transactions from DA at specified height
    RetrieveForcedIncludedTxsFromDA(ctx context.Context, daHeight uint64) (*ForcedIncludedEvent, error)
}
```

### APIs and Interfaces

#### DA Retriever

The DA Retriever component handles fetching forced inclusion transactions:

```go
type daRetriever struct {
    da                         coreda.DA
    cache                      cache.CacheManager
    genesis                    genesis.Genesis
    logger                     zerolog.Logger
    namespaceForcedInclusionBz []byte
    hasForcedInclusionNs       bool
    daEpochSize                uint64
}

// RetrieveForcedIncludedTxsFromDA fetches forced inclusion transactions
// Only fetches at epoch boundaries to prevent redundant DA queries
func (r *daRetriever) RetrieveForcedIncludedTxsFromDA(
    ctx context.Context,
    daHeight uint64,
) (*ForcedIncludedEvent, error)
```

#### Single Sequencer Extension

The single sequencer is enhanced to fetch and include forced transactions:

```go
type Sequencer struct {
    // ... existing fields ...
    fiRetriever               ForcedInclusionRetriever
    genesis                   genesis.Genesis
    daHeight                  atomic.Uint64
    pendingForcedInclusionTxs []pendingForcedInclusionTx
    queue                     *BatchQueue
}

type pendingForcedInclusionTx struct {
    Data           []byte
    OriginalHeight uint64
}

func (s *Sequencer) GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error) {
    // 1. Fetch forced inclusion transactions from DA
    forcedEvent, err := s.fiRetriever.RetrieveForcedIncludedTxs(ctx, s.daHeight.Load())

    // 2. Process forced txs with size validation and pending queue
    forcedTxs := s.processForcedInclusionTxs(forcedEvent, req.MaxBytes)

    // 3. Get batch from mempool queue
    batch, err := s.queue.Next(ctx)

    // 4. Prepend forced txs and trim batch to fit MaxBytes
    if len(forcedTxs) > 0 {
        forcedTxsSize := calculateSize(forcedTxs)
        remainingBytes := req.MaxBytes - forcedTxsSize

        // Trim batch transactions to fit
        trimmedBatchTxs := trimToSize(batch.Transactions, remainingBytes)

        // Return excluded txs to front of queue
        if len(trimmedBatchTxs) < len(batch.Transactions) {
            excludedBatch := batch.Transactions[len(trimmedBatchTxs):]
            s.queue.Prepend(ctx, Batch{Transactions: excludedBatch})
        }

        batch.Transactions = append(forcedTxs, trimmedBatchTxs...)
    }

    return &GetNextBatchResponse{Batch: batch}
}

// processForcedInclusionTxs validates and queues forced txs
func (s *Sequencer) processForcedInclusionTxs(event *ForcedInclusionEvent, maxBytes uint64) [][]byte {
    var validatedTxs [][]byte
    var newPendingTxs []pendingForcedInclusionTx
    currentSize := 0

    // Process pending txs from previous epochs first
    for _, pendingTx := range s.pendingForcedInclusionTxs {
        if !ValidateBlobSize(pendingTx.Data) {
            continue // Skip blobs exceeding absolute DA limit
        }
        if WouldExceedCumulativeSize(currentSize, len(pendingTx.Data), maxBytes) {
            newPendingTxs = append(newPendingTxs, pendingTx)
            continue
        }
        validatedTxs = append(validatedTxs, pendingTx.Data)
        currentSize += len(pendingTx.Data)
    }

    // Process new txs from this epoch
    for _, tx := range event.Txs {
        if !ValidateBlobSize(tx) {
            continue // Skip blobs exceeding absolute DA limit
        }
        if WouldExceedCumulativeSize(currentSize, len(tx), maxBytes) {
            newPendingTxs = append(newPendingTxs, pendingForcedInclusionTx{
                Data:           tx,
                OriginalHeight: event.StartDaHeight,
            })
            continue
        }
        validatedTxs = append(validatedTxs, tx)
        currentSize += len(tx)
    }

    s.pendingForcedInclusionTxs = newPendingTxs
    return validatedTxs
}
```

#### Based Sequencer

A new sequencer implementation that ONLY retrieves transactions from DA:

```go
type BasedSequencer struct {
    fiRetriever ForcedInclusionRetriever
    da          coreda.DA
    config      config.Config
    genesis     genesis.Genesis
    logger      zerolog.Logger
    mu          sync.RWMutex
    daHeight    uint64
    txQueue     [][]byte  // Buffer for transactions exceeding batch size
}

func (s *BasedSequencer) GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error) {


    // Always fetch forced inclusion transactions from DA
    forcedEvent, err := s.fiRetriever.RetrieveForcedIncludedTxs(ctx, s.daHeight)
    if err != nil && !errors.Is(err, ErrHeightFromFuture) {
        return nil, err
    }

    // Validate and add transactions to queue
    for _, tx := range forcedEvent.Txs {
        if ValidateBlobSize(tx) {
            s.txQueue = append(s.txQueue, tx)
        }
    }

    // Create batch from queue respecting MaxBytes
    batch := s.createBatchFromQueue(req.MaxBytes)

    return &GetNextBatchResponse{Batch: batch}
}

// SubmitBatchTxs is a no-op for based sequencer
func (s *BasedSequencer) SubmitBatchTxs(ctx context.Context, req SubmitBatchTxsRequest) (*SubmitBatchTxsResponse, error) {
    // Based sequencer ignores submitted transactions
    return &SubmitBatchTxsResponse{}, nil
}
```

#### Syncer Verification

Full nodes verify forced inclusion in the sync process with support for transaction smoothing across multiple blocks and a configurable grace period:

```go
func (s *Syncer) verifyForcedInclusionTxs(currentState State, data *Data) error {
    // 1. Retrieve forced inclusion transactions from DA for current epoch
    forcedEvent, err := s.daRetriever.RetrieveForcedIncludedTxsFromDA(s.ctx, currentState.DAHeight)
    if err != nil {
        return err
    }

    // 2. Build map of transactions in current block
    blockTxMap := make(map[string]struct{})
    for _, tx := range data.Txs {
        blockTxMap[hashTx(tx)] = struct{}{}
    }

    // 3. Check if any pending forced inclusion txs from previous epochs are included
    var stillPending []pendingForcedInclusionTx
    s.pendingForcedInclusionTxs.Range(func(key, value any) bool {
        pending := value.(pendingForcedInclusionTx)
        if _, ok := blockTxMap[pending.TxHash]; ok {
            // Transaction was included - remove from pending
            s.pendingForcedInclusionTxs.Delete(key)
        } else {
            stillPending = append(stillPending, pending)
        }
        return true
    })

    // 4. Process new forced inclusion transactions from current epoch
    for _, forcedTx := range forcedEvent.Txs {
        txHash := hashTx(forcedTx)
        if _, ok := blockTxMap[txHash]; !ok {
            // Transaction not included yet - add to pending for deferral within epoch
            stillPending = append(stillPending, pendingForcedInclusionTx{
                Data:       forcedTx,
                EpochStart: forcedEvent.StartDaHeight,
                EpochEnd:   forcedEvent.EndDaHeight,
                TxHash:     txHash,
            })
        }
    }

    // 5. Check for malicious behavior: pending txs past their grace boundary
    // Grace period provides tolerance for temporary DA unavailability
    var maliciousTxs, remainingPending []pendingForcedInclusionTx
    for _, pending := range stillPending {
        // Calculate grace boundary: epoch end + (grace periods × epoch size)
        graceBoundary := pending.EpochEnd + (s.genesis.ForcedInclusionGracePeriod * s.genesis.DAEpochForcedInclusion)

        // If current DA height is past the grace boundary, these txs MUST have been included
        if currentState.DAHeight > graceBoundary {
            maliciousTxs = append(maliciousTxs, pending)
        } else {
            remainingPending = append(remainingPending, pending)
        }
    }

    // 6. Update pending map with only remaining valid pending txs
    pendingForcedInclusionTxs = remainingPending

    // 7. Reject block if sequencer censored forced txs past grace boundary
    if len(maliciousTxs) > 0 {
        return fmt.Errorf("sequencer is malicious: %d forced inclusion transactions past grace boundary (grace_periods=%d) not included", len(maliciousTxs), s.genesis.ForcedInclusionGracePeriod)
    }

    return nil
}
```

**Key Verification Features**:

1. **Pending Transaction Tracking**: Maintains a map of forced inclusion transactions that haven't been included yet
2. **Epoch-Based Deferral**: Allows transactions to be deferred (smoothed) across multiple blocks within the same epoch
3. **Strict Epoch Boundary Enforcement**: Once `currentState.DAHeight > pending.EpochEnd`, all pending transactions from that epoch MUST have been included
4. **Censorship Detection**: Identifies malicious sequencers that fail to include forced transactions after epoch boundaries

**Smoothing Example**:

```
Epoch [100-109] contains 3MB of forced inclusion transactions

Block at DA height 100:
  - Includes 2MB of forced txs (partial)
  - Remaining 1MB added to pending map with EpochEnd=109
  - ✅ Valid - within epoch boundary

Block at DA height 105:
  - Includes remaining 1MB from pending
  - Pending map cleared for those txs
  - ✅ Valid - within epoch boundary

Block at DA height 110 (next epoch):
  - If any txs from epoch [100-109] still pending
  - ❌ MALICIOUS - epoch boundary violated
  - Block rejected, sequencer flagged
```

### Implementation Details

#### Epoch-Based Fetching

To avoid excessive DA queries, the DA Retriever uses epoch-based fetching:

- **Epoch Size**: Configurable number of DA blocks (e.g., 10)
- **Epoch Boundaries**: Deterministically calculated based on `DAStartHeight`
- **Fetch Timing**: Only fetch at epoch start to prevent duplicate fetches

```go
// Calculate epoch boundaries
func (r *daRetriever) calculateEpochBoundaries(daHeight uint64) (start, end uint64) {
    epochNum := r.calculateEpochNumber(daHeight)
    start = r.genesis.DAStartHeight + (epochNum-1)*r.daEpochSize
    end = r.genesis.DAStartHeight + epochNum*r.daEpochSize - 1
    return start, end
}

// Only fetch at epoch start
if daHeight != epochStart {
    return &ForcedIncludedEvent{Txs: [][]byte{}}
}

// Fetch all heights in epoch range
for height := epochStart; height <= epochEnd; height++ {
    // Fetch forced inclusion blobs from this DA height
}
```

#### Height From Future Handling

When DA height is not yet available:

```go
if errors.Is(err, coreda.ErrHeightFromFuture) {
    // Keep current DA height, return empty batch
    // Retry same height on next call
    return &ForcedIncludedEvent{Txs: [][]byte{}}, nil
}
```

#### Grace Period for Forced Inclusion

The grace period mechanism provides tolerance for chain congestion while maintaining censorship resistance:

**Problem**: If the DA layer experiences temporary unavailability or the chain congestion, the sequencer may be unable to fetch forced inclusion transactions from a completed epoch. Without a grace period, full nodes would immediately flag the sequencer as malicious.

**Solution**: The `ForcedInclusionGracePeriod` parameter allows forced inclusion transactions from epoch N to be included during epochs N+1 through N+k (where k is the grace period) before being flagged as malicious.

**Grace Boundary Calculation**:

```go
graceBoundary := epochEnd + (ForcedInclusionGracePeriod * DAEpochForcedInclusion)

// Example with ForcedInclusionGracePeriod = 1, DAEpochForcedInclusion = 50:
// - Epoch N ends at DA height 100
// - Grace boundary = 100 + (1 * 50) = 150
// - Transaction must be included by DA height 150
// - If not included by height 151+, sequencer is malicious
```

**Configuration Recommendations**:

- **Production (default)**: `ForcedInclusionGracePeriod = 1`
  - Tolerates ~1 epoch of DA unavailability (e.g., 50 DA blocks)
  - Balances censorship resistance with reliability
- **High Security / Reliable DA**: `ForcedInclusionGracePeriod = 0`
  - Strict enforcement, no tolerance
  - Requires 99.9%+ DA uptime
  - Immediate detection of censorship
- **Unreliable DA**: `ForcedInclusionGracePeriod = 2+`
  - Higher tolerance for DA outages
  - Reduced censorship resistance (longer time to detect malicious behavior)

**Verification Logic**:

1. Forced inclusion transactions from epoch N are tracked with their epoch boundaries
2. Transactions not immediately included are added to pending queue
3. Each block, full nodes check if pending transactions are past their grace boundary
4. If `currentDAHeight > graceBoundary`, the sequencer is flagged as malicious
5. Transactions within the grace period remain in pending queue without error

**Benefits**:

- Prevents false positives during temporary DA outages
- Maintains censorship resistance (transactions must be included within grace window)
- Configurable trade-off between reliability and security
- Allows networks to adapt to their DA layer's reliability characteristics

**Examples and Edge Cases**:

Configuration: `DAEpochForcedInclusion = 50`, `ForcedInclusionGracePeriod = 1`

_Example 1: Normal Inclusion (Within Same Epoch)_

```
- Forced tx submitted to DA at height 75 (epoch 51-100)
- Sequencer fetches at height 101 (next epoch start)
- Sequencer includes tx in block at DA height 105
- Result: ✅ Valid - included within same epoch
```

_Example 2: Grace Period Usage (Included in Next Epoch)_

```
- Forced tx submitted to DA at height 75 (epoch 51-100)
- Sequencer fetches at height 101
- DA temporarily unavailable, sequencer cannot fetch
- Sequencer includes tx at DA height 125 (epoch 101-150)
- Grace boundary = 100 + (1 × 50) = 150
- Result: ✅ Valid - within grace period
```

_Example 3: Malicious Sequencer (Past Grace Boundary)_

```
- Forced tx submitted to DA at height 75 (epoch 51-100)
- Sequencer fetches at height 101
- Sequencer deliberately omits tx
- Block produced at DA height 151 (past grace boundary 150)
- Full node detects: currentDAHeight (151) > graceBoundary (150)
- Result: ❌ Block rejected, sequencer flagged as malicious
```

_Example 4: Strict Mode (Grace Period = 0)_

```
- ForcedInclusionGracePeriod = 0
- Forced tx submitted at height 75 (epoch 51-100)
- Sequencer must include by height 100 (epoch end)
- Block at height 101 without tx is rejected
- Result: Immediate censorship detection, requires high DA reliability
```

_Example 5: Multiple Pending Transactions_

```
- Tx A from epoch ending at height 100, grace boundary 150
- Tx B from epoch ending at height 150, grace boundary 200
- Current DA height: 155
- Tx A not included: ❌ Past grace boundary - malicious
- Tx B not included: ✅ Within grace period - still pending
- Result: Block rejected due to Tx A
```

_Example 6: Extended Grace Period (Grace Period = 2)_

```
- ForcedInclusionGracePeriod = 2
- Forced tx submitted at height 75 (epoch 51-100)
- Grace boundary = 100 + (2 × 50) = 200
- Sequencer has until DA height 200 to include tx
- Result: More tolerance but delayed censorship detection
```

#### Size Validation and Max Bytes Handling

Both sequencers enforce strict size limits to prevent DoS and ensure batches never exceed the DA layer's limits:

```go
// Size validation utilities
const AbsoluteMaxBlobSize = 1.5 * 1024 * 1024 // 1.5MB DA layer limit

// ValidateBlobSize checks against absolute DA layer limit
func ValidateBlobSize(blob []byte) bool {
    return uint64(len(blob)) <= AbsoluteMaxBlobSize
}

// WouldExceedCumulativeSize checks against per-batch limit
func WouldExceedCumulativeSize(currentSize int, blobSize int, maxBytes uint64) bool {
    return uint64(currentSize)+uint64(blobSize) > maxBytes
}
```

**Key Behaviors**:

- **Absolute validation**: Blobs exceeding 2MB are permanently rejected
- **Batch size limits**: `req.MaxBytes` is NEVER exceeded in any batch
- **Transaction preservation**:
  - Single sequencer: Trimmed batch txs returned to queue via `Prepend()`
  - Based sequencer: Excess txs remain in `txQueue` for next batch
  - Forced txs that don't fit go to `pendingForcedInclusionTxs` (single) or stay in `txQueue` (based)

#### Transaction Queue Management

The based sequencer uses a simplified queue to handle transactions:

```go
func (s *BasedSequencer) createBatchFromQueue(maxBytes uint64) *Batch {
    var batch [][]byte
    var totalBytes uint64

    for i, tx := range s.txQueue {
        txSize := uint64(len(tx))
        // Always respect maxBytes, even for first transaction
        if totalBytes+txSize > maxBytes {
            // Would exceed max bytes, keep remaining in queue
            s.txQueue = s.txQueue[i:]
            break
        }

        batch = append(batch, tx)
        totalBytes += txSize

        // Clear queue if we processed everything
        if i == len(s.txQueue)-1 {
            s.txQueue = s.txQueue[:0]
        }
    }

    return &Batch{Transactions: batch}
}
```

**Note**: The based sequencer is simpler than the single sequencer - it doesn't need a separate pending queue because `txQueue` naturally handles all transaction buffering.

### Configuration

```go
type Genesis struct {
    ChainID                string
    StartTime              time.Time
    InitialHeight          uint64
    ProposerAddress        []byte
    DAStartHeight          uint64
    // Number of DA blocks to scan per forced inclusion fetch
    // Higher values reduce DA queries but increase latency
    // Lower values increase DA queries but improve responsiveness
    DAEpochForcedInclusion uint64
    // Number of additional epochs allowed for including forced inclusion transactions
    // before marking the sequencer as malicious. Provides tolerance for temporary DA unavailability.
    // Value of 0: Strict enforcement (no grace period) - requires 99.9% DA uptime
    // Value of 1: Transactions from epoch N can be included through epoch N+1 (recommended)
    // Value of 2+: Higher tolerance for unreliable DA environments
    ForcedInclusionGracePeriod uint64
}

type DAConfig struct {
    // ... existing fields ...

    // Namespace for forced inclusion transactions
    ForcedInclusionNamespace string
}

type NodeConfig struct {
    // ... existing fields ...

    // Run node with based sequencer (requires aggregator mode)
    BasedSequencer bool
}
```

### Configuration Examples

#### Traditional Sequencer with Forced Inclusion

```yaml
# genesis.json
{
  "chain_id": "my-rollup",
  "forced_inclusion_da_epoch": 10,  # Scan 10 DA blocks at a time
  "forced_inclusion_grace_period": 1  # Allow 1 epoch grace period (recommended for production)
}

# config.toml
[da]
forced_inclusion_namespace = "0x0000000000000000000000000000000000000000000000000000666f72636564"

[node]
aggregator = true
based_sequencer = false # Use traditional sequencer
```

#### Based Sequencer (DA-Only)

```yaml
# genesis.json
{
  "chain_id": "my-rollup",
  "forced_inclusion_da_epoch": 5,  # Scan 5 DA blocks at a time
  "forced_inclusion_grace_period": 1  # Allow 1 epoch grace period (balances reliability and censorship detection)
}

# config.toml
[da]
forced_inclusion_namespace = "0x0000000000000000000000000000000000000000000000000000666f72636564"

[node]
aggregator = true
based_sequencer = true # Use based sequencer
```

### Sequencer Operation Flows

#### Single Sequencer Flow

1. Timer triggers GetNextBatch
2. Fetch forced inclusion txs from DA (via DA Retriever)
   - Only at epoch boundaries
   - Scan epoch range for forced transactions
3. Get batch from mempool queue
4. Prepend forced txs to batch
5. Return batch for block production

#### Based Sequencer Flow

1. Timer triggers GetNextBatch
2. Check transaction queue for buffered txs
3. If queue empty or epoch boundary:
   - Fetch forced inclusion txs from DA
   - Add to queue
4. Create batch from queue (respecting MaxBytes)
5. Return batch for block production

### Full Node Verification Flow

1. Receive block from DA or P2P
2. Before applying block:
   a. Fetch forced inclusion txs from DA at block's DA height (epoch-based)
   b. Build map of transactions in block
   c. Check if pending forced txs from previous epochs are included
   d. Add any new forced txs not yet included to pending queue
   e. Calculate grace boundary for each pending tx:
   graceBoundary = epochEnd + (ForcedInclusionGracePeriod × DAEpochForcedInclusion)
   f. Check if any pending txs are past their grace boundary
   g. If txs past grace boundary are not included: reject block, flag malicious proposer
   h. If txs within grace period: keep in pending queue, allow block
3. Apply block if verification passes

**Grace Period Example** (with `ForcedInclusionGracePeriod = 1`, `DAEpochForcedInclusion = 50`):

- Forced tx appears in epoch ending at DA height 100
- Grace boundary = 100 + (1 × 50) = 150
- Transaction can be included at any DA height from 101 to 150
- At DA height 151+, if not included, sequencer is flagged as malicious

### Efficiency Considerations

1. **Epoch-Based Fetching**: Reduces DA queries by batching multiple DA heights
2. **Deterministic Epochs**: All nodes calculate same epoch boundaries
3. **Fetch at Epoch Start**: Prevents duplicate fetches as DA height progresses
4. **Transaction Queue**: Buffers excess transactions across multiple blocks
5. **Conditional Fetching**: Only when forced inclusion namespace is configured
6. **Size Pre-validation**: Invalid blobs rejected early, before batch construction
7. **Efficient Queue Operations**:
   - Single sequencer: `Prepend()` reuses space before head position
   - Based sequencer: Simple slice operations for queue management

**DA Query Frequency**:

Every `DAEpochForcedInclusion` DA blocks

### Security Considerations

1. **Malicious Proposer Detection**: Full nodes reject blocks missing forced transactions
2. **No Timing Attacks**: Epoch boundaries are deterministic, no time-based logic
3. **Blob Size Limits**: Two-tier size validation prevents DoS
   - Absolute limit (1.5MB): Blobs exceeding this are permanently rejected
   - Batch limit (`MaxBytes`): Ensures no batch exceeds DA submission limits
4. **Graceful Degradation**: Continues operation if forced inclusion not configured
5. **Height Validation**: Handles "height from future" errors without state corruption
6. **Transaction Preservation**: No valid transactions are lost due to size constraints
7. **Strict MaxBytes Enforcement**: Batches NEVER exceed `req.MaxBytes`, preventing DA layer rejections

**Attack Vectors**:

### Security Considerations

- **Censorship**: Mitigated by forced inclusion verification with grace period
  - Transactions must be included within grace window (epoch + grace period)
  - Full nodes detect and reject blocks from malicious sequencers
  - Grace period = 0 provides immediate detection but requires high DA reliability
  - Grace period = 1+ balances censorship resistance with operational tolerance
- **DA Spam**: Limited by DA layer's native spam protection and two-tier blob size limits
- **Block Withholding**: Full nodes can fetch and verify from DA independently
- **Oversized Batches**: Prevented by strict size validation at multiple levels
- **Grace Period Attacks**:
  - Malicious sequencer cannot indefinitely delay forced transactions
  - Grace boundary is deterministic and enforced by all full nodes
  - Longer grace periods extend time to detect censorship (trade-off)

## Status

Accepted and Implemented

## Consequences

### Positive

1. **Censorship Resistance**: Users have guaranteed path to include transactions
2. **Verifiable**: Full nodes enforce forced inclusion, detecting malicious sequencers
3. **Simple Design**: No complex timing mechanisms or fallback modes
4. **Based Rollup Option**: Fully DA-driven transaction ordering available (simplified implementation)
5. **Optional**: Forced inclusion can be disabled for permissioned deployments
6. **Efficient**: Epoch-based fetching minimizes DA queries
7. **Flexible**: Configurable epoch size and grace period allow tuning latency vs reliability
8. **Robust Size Handling**: Two-tier size validation prevents DoS and DA rejections
9. **Transaction Preservation**: All valid transactions are preserved in queues, nothing is lost
10. **Strict MaxBytes Compliance**: Batches never exceed limits, preventing DA submission failures
11. **DA Fault Tolerance**: Grace period prevents false positives during temporary DA unavailability

### Negative

1. **Increased Latency**: Forced transactions subject to epoch boundaries
2. **DA Dependency**: Requires DA layer to support multiple namespaces
3. **Higher DA Costs**: Users pay DA posting fees for forced inclusion
4. **Additional Complexity**: New component (DA Retriever) and verification logic with grace period tracking
5. **Epoch Configuration**: Requires setting `DAEpochForcedInclusion` and `ForcedInclusionGracePeriod` in genesis (consensus parameters)
6. **Grace Period Trade-off**: Longer grace periods delay censorship detection but improve operational reliability

### Neutral

1. **Two Sequencer Types**: Choice between single (hybrid) and based (DA-only)
2. **Privacy Model Unchanged**: Forced inclusion has same privacy as normal path
3. **Monitoring**: Operators should monitor forced inclusion namespace usage and grace period metrics
4. **Documentation**: Users need guidance on when to use forced inclusion and grace period implications
5. **Genesis Parameters**: `DAEpochForcedInclusion` and `ForcedInclusionGracePeriod` are consensus parameters fixed at genesis

## References

- [Evolve Single Sequencer ADR-013](https://github.com/evstack/ev-node/blob/main/docs/adr/adr-013-single-sequencer.md)
- [Evolve Minimal Header ADR-015](https://github.com/evstack/ev-node/blob/main/docs/adr/adr-015-rollkit-minimal-header.md)
- [L2 Beat Stages Framework](https://forum.l2beat.com/t/the-stages-framework/291#p-516-stage-1-requirements-3)
- [GitHub Issue #1914: Add Forced Inclusion Mechanism from the DA layer](https://github.com/evstack/ev-node/issues/1914)
