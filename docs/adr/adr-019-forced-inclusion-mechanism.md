# ADR 019: Forced Inclusion Mechanism

## Changelog

- 2025-03-24: Initial draft
- 2025-04-23: Renumbered from ADR-018 to ADR-019 to maintain chronological order.
- 2025-11-10: Updated to reflect actual implementation

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
    daRetriever DARetriever
    genesis     genesis.Genesis
    mu          sync.RWMutex
    daHeight    uint64
}

func (s *Sequencer) GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error) {
    // 1. Fetch forced inclusion transactions from DA
    forcedEvent, err := s.daRetriever.RetrieveForcedIncludedTxsFromDA(ctx, s.daHeight)

    // 2. Get batch from mempool
    batch, err := s.queue.Next(ctx)

    // 3. Prepend forced transactions to batch
    if len(forcedEvent.Txs) > 0 {
        batch.Transactions = append(forcedEvent.Txs, batch.Transactions...)
    }

    return &GetNextBatchResponse{Batch: batch}
}
```

#### Based Sequencer

A new sequencer implementation that ONLY retrieves transactions from DA:

```go
type BasedSequencer struct {
    daRetriever DARetriever
    da          coreda.DA
    config      config.Config
    genesis     genesis.Genesis
    logger      zerolog.Logger
    mu          sync.RWMutex
    daHeight    uint64
    txQueue     [][]byte  // Buffer for transactions exceeding batch size
}

func (s *BasedSequencer) GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error) {
    // Fetch forced inclusion transactions from DA
    forcedEvent, err := s.daRetriever.RetrieveForcedIncludedTxsFromDA(ctx, s.daHeight)

    // Add transactions to queue
    s.txQueue = append(s.txQueue, forcedEvent.Txs...)

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

Full nodes verify forced inclusion in the sync process:

```go
func (s *Syncer) verifyForcedInclusionTxs(currentState State, data *Data) error {
    // 1. Retrieve forced inclusion transactions from DA
    forcedEvent, err := s.daRetriever.RetrieveForcedIncludedTxsFromDA(s.ctx, currentState.DAHeight)
    if err != nil {
        return err
    }

    // 2. Build map of transactions in block
    blockTxMap := make(map[string]struct{})
    for _, tx := range data.Txs {
        blockTxMap[string(tx)] = struct{}{}
    }

    // 3. Verify all forced transactions are included
    for _, forcedTx := range forcedEvent.Txs {
        if _, ok := blockTxMap[string(forcedTx)]; !ok {
            return errMaliciousProposer
        }
    }

    return nil
}
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

#### Transaction Queue Management

The based sequencer uses a queue to handle transactions exceeding batch size:

```go
func (s *BasedSequencer) createBatchFromQueue(maxBytes uint64) *Batch {
    var batch [][]byte
    var totalBytes uint64

    for i, tx := range s.txQueue {
        txSize := uint64(len(tx))
        if totalBytes+txSize > maxBytes && len(batch) > 0 {
            // Would exceed max bytes, stop here
            s.txQueue = s.txQueue[i:]
            break
        }

        batch = append(batch, tx)
        totalBytes += txSize
    }

    return &Batch{Transactions: batch}
}
```

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
  "forced_inclusion_da_epoch": 10  # Scan 10 DA blocks at a time
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
  "forced_inclusion_da_epoch": 5  # Scan 5 DA blocks at a time
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

```
1. Timer triggers GetNextBatch
2. Fetch forced inclusion txs from DA (via DA Retriever)
   - Only at epoch boundaries
   - Scan epoch range for forced transactions
3. Get batch from mempool queue
4. Prepend forced txs to batch
5. Return batch for block production
```

#### Based Sequencer Flow

```
1. Timer triggers GetNextBatch
2. Check transaction queue for buffered txs
3. If queue empty or epoch boundary:
   - Fetch forced inclusion txs from DA
   - Add to queue
4. Create batch from queue (respecting MaxBytes)
5. Return batch for block production
```

### Full Node Verification Flow

```
1. Receive block from DA or P2P
2. Before applying block:
   a. Fetch forced inclusion txs from DA at block's DA height
   b. Build map of transactions in block
   c. Verify all forced txs are in block
   d. If missing: reject block, flag malicious proposer
3. Apply block if verification passes
```

### Efficiency Considerations

1. **Epoch-Based Fetching**: Reduces DA queries by batching multiple DA heights
2. **Deterministic Epochs**: All nodes calculate same epoch boundaries
3. **Fetch at Epoch Start**: Prevents duplicate fetches as DA height progresses
4. **Transaction Queue**: Buffers excess transactions across multiple blocks
5. **Conditional Fetching**: Only when forced inclusion namespace is configured

**DA Query Frequency**:

Every `DAEpochForcedInclusion` DA blocks

### Security Considerations

1. **Malicious Proposer Detection**: Full nodes reject blocks missing forced transactions
2. **No Timing Attacks**: Epoch boundaries are deterministic, no time-based logic
3. **Blob Size Limits**: Enforces maximum blob size to prevent DoS
4. **Graceful Degradation**: Continues operation if forced inclusion not configured
5. **Height Validation**: Handles "height from future" errors without state corruption

**Attack Vectors**:

- **Censorship**: Mitigated by forced inclusion verification
- **DA Spam**: Limited by DA layer's native spam protection and blob size limits
- **Block Withholding**: Full nodes can fetch and verify from DA independently

### Testing Strategy

#### Unit Tests

1. **DA Retriever**:
   - Epoch boundary calculations
   - Height from future handling
   - Blob size validation
   - Empty epoch handling

2. **Single Sequencer**:
   - Forced transaction prepending
   - DA height tracking
   - Error handling

3. **Based Sequencer**:
   - Queue management
   - Batch size limits
   - DA-only operation

4. **Syncer Verification**:
   - All forced txs included (pass)
   - Missing forced txs (fail)
   - No forced txs (pass)

#### Integration Tests

1. **Single Sequencer Integration**:
   - Submit to mempool and forced inclusion
   - Verify both included in block
   - Forced txs appear first

2. **Based Sequencer Integration**:
   - Submit only to DA forced inclusion
   - Verify block production
   - Mempool submissions ignored

3. **Verification Flow**:
   - Full node rejects block missing forced tx
   - Full node accepts block with all forced txs

#### End-to-End Tests

1. **User Flow**:
   - User submits tx to forced inclusion namespace
   - Sequencer includes tx in next epoch
   - Full nodes verify inclusion

2. **Based Rollup**:
   - Start network with based sequencer
   - Submit transactions to DA
   - Verify block production and finalization

3. **Censorship Resistance**:
   - Sequencer ignores specific transaction
   - User submits to forced inclusion
   - Transaction included in next epoch
   - Attempting to exclude causes block rejection

### Breaking Changes

1. **Sequencer Initialization**: Requires `DARetriever` and `Genesis` parameters
2. **Configuration**: New fields in `DAConfig` and `NodeConfig`
3. **Syncer**: New verification step in block processing

**Migration Path**:

- Forced inclusion is optional (enabled when namespace configured)
- Existing deployments work without configuration changes
- Can enable incrementally per network

## Status

Accepted and Implemented

## Consequences

### Positive

1. **Censorship Resistance**: Users have guaranteed path to include transactions
2. **Verifiable**: Full nodes enforce forced inclusion, detecting malicious sequencers
3. **Simple Design**: No complex timing mechanisms or fallback modes
4. **Based Rollup Option**: Fully DA-driven transaction ordering available
5. **Optional**: Forced inclusion can be disabled for permissioned deployments
6. **Efficient**: Epoch-based fetching minimizes DA queries
7. **Flexible**: Configurable epoch size allows tuning latency vs efficiency

### Negative

1. **Increased Latency**: Forced transactions subject to epoch boundaries
2. **DA Dependency**: Requires DA layer to support multiple namespaces
3. **Higher DA Costs**: Users pay DA posting fees for forced inclusion
4. **Additional Complexity**: New component (DA Retriever) and verification logic
5. **Epoch Configuration**: Requires setting `DAEpochForcedInclusion` in genesis (consensus parameter)

### Neutral

1. **Two Sequencer Types**: Choice between single (hybrid) and based (DA-only)
2. **Privacy Model Unchanged**: Forced inclusion has same privacy as normal path
3. **Monitoring**: Operators should monitor forced inclusion namespace usage
4. **Documentation**: Users need guidance on when to use forced inclusion
5. **Genesis Parameter**: `DAEpochForcedInclusion` is a consensus parameter fixed at genesis

## References

- [Evolve Single Sequencer ADR-013](https://github.com/evstack/ev-node/blob/main/docs/adr/adr-013-single-sequencer.md)
- [Evolve Minimal Header ADR-015](https://github.com/evstack/ev-node/blob/main/docs/adr/adr-015-rollkit-minimal-header.md)
- [L2 Beat Stages Framework](https://forum.l2beat.com/t/the-stages-framework/291#p-516-stage-1-requirements-3)
- [GitHub Issue #1914: Add Forced Inclusion Mechanism from the DA layer](https://github.com/evstack/ev-node/issues/1914)
