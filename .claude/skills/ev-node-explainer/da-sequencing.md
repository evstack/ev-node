# Data Availability & Sequencing Architecture

Deep dive into ev-node's DA layer and sequencing system.

## Data Availability Layer

### Overview

The DA layer abstracts blob storage and retrieval. ev-node uses Celestia as the primary DA implementation but the interface is pluggable.

### Directory Structure

```
pkg/da/
├── types/
│   ├── types.go        # Core DA types (Blob, ID, Commitment, Proof)
│   ├── namespace.go    # Namespace handling (29 bytes: version + ID)
│   └── errors.go       # Error definitions
├── selector.go         # Round-robin address selection
└── jsonrpc/            # Celestia JSON-RPC client

block/internal/da/
├── client.go                      # DA client wrapper
├── interface.go                   # Client + Verifier interfaces
├── forced_inclusion_retriever.go  # Epoch-based forced tx retrieval
└── async_block_retriever.go       # Background prefetching
```

### Core Types

```go
// Status codes for DA operations
const (
    StatusSuccess
    StatusNotFound
    StatusNotIncludedInBlock
    StatusAlreadyInMempool
    StatusTooBig
    StatusContextDeadline
    StatusError
    StatusIncorrectAccountSequence
    StatusContextCanceled
    StatusHeightFromFuture
)

// Blob primitives
type Blob = []byte        // Data submitted to DA
type ID = []byte          // Height + commitment to locate blob
type Commitment = []byte  // Cryptographic commitment
type Proof = []byte       // Inclusion proof
```

### Namespace Format

Namespaces are 29 bytes:
- **Version** (1 byte): Protocol version (max 255)
- **ID** (28 bytes): Namespace identifier

Version 0 rules:
- First 18 bytes of ID must be zero
- Leaves 10 bytes for user data

```go
func NewNamespaceV0(id []byte) (Namespace, error) {
    if len(id) > 10 {
        return Namespace{}, ErrInvalidNamespaceLength
    }
    ns := Namespace{Version: 0}
    copy(ns.ID[28-len(id):], id)  // Right-pad zeros
    return ns, nil
}
```

### DA Client Interface

```go
type Client interface {
    // Submit blobs to DA layer
    Submit(ctx context.Context, data [][]byte, gasPrice float64,
           namespace []byte, options []byte) ResultSubmit

    // Retrieve all blobs at height for namespace
    Retrieve(ctx context.Context, height uint64,
             namespace []byte) ResultRetrieve

    // Get specific blobs by ID
    Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)

    // Namespace accessors
    GetHeaderNamespace() []byte
    GetDataNamespace() []byte
    GetForcedInclusionNamespace() []byte
    HasForcedInclusionNamespace() bool
}

type Verifier interface {
    GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error)
    Validate(ctx context.Context, ids []ID, proofs []Proof,
             namespace []byte) ([]bool, error)
}

type FullClient interface {
    Client
    Verifier
}
```

### Submit Flow

```go
func (c *Client) Submit(ctx, data, gasPrice, namespace, options) ResultSubmit {
    // 1. Validate blob size
    for _, blob := range data {
        if len(blob) > DefaultMaxBlobSize {
            return ResultSubmit{Code: StatusTooBig}
        }
    }

    // 2. Create Celestia blobs with namespace
    blobs := make([]*blob.Blob, len(data))
    for i, d := range data {
        blobs[i], _ = blob.NewBlobV0(namespace, d)
    }

    // 3. Submit via RPC
    height, err := c.blobRPC.Submit(ctx, blobs, submitOptions)

    // 4. Return result with IDs
    return ResultSubmit{
        Code:   StatusSuccess,
        Height: height,
        IDs:    createIDs(height, blobs),
    }
}
```

### Retrieve Flow

```go
func (c *Client) Retrieve(ctx, height, namespace) ResultRetrieve {
    // 1. Fetch all blobs at height
    blobs, err := c.blobRPC.GetAll(ctx, height, []Namespace{namespace})

    // 2. Handle errors
    if errors.Is(err, ErrBlobNotFound) {
        return ResultRetrieve{Code: StatusNotFound}
    }
    if errors.Is(err, ErrHeightFromFuture) {
        return ResultRetrieve{Code: StatusHeightFromFuture}
    }

    // 3. Get timestamp from DA header
    header, _ := c.headerRPC.GetByHeight(ctx, height)

    // 4. Extract blob data
    data := make([][]byte, len(blobs))
    for i, b := range blobs {
        data[i] = b.Data()
    }

    return ResultRetrieve{
        Code:      StatusSuccess,
        Height:    height,
        Timestamp: header.Time().UnixNano(),
        Data:      data,
    }
}
```

### Address Selection

For Cosmos SDK compatibility (preventing sequence mismatches):

```go
type RoundRobinSelector struct {
    addresses []string
    counter   atomic.Uint64
}

func (s *RoundRobinSelector) Next() string {
    idx := s.counter.Add(1) % uint64(len(s.addresses))
    return s.addresses[idx]
}
```

---

## Sequencing System

### Overview

Sequencers order transactions for block production. ev-node supports two modes:
- **Single Sequencer**: Hybrid (mempool + forced inclusion)
- **Based Sequencer**: Pure DA (only forced inclusion)

### Directory Structure

```
core/sequencer/
├── sequencing.go    # Core interface
└── dummy.go         # Test implementation

pkg/sequencers/
├── single/
│   ├── sequencer.go # Hybrid sequencer
│   └── queue.go     # Persistent batch queue
├── based/
│   └── sequencer.go # Pure DA sequencer
└── common/
    └── checkpoint.go # Shared checkpoint logic
```

### Core Interface

```go
type Sequencer interface {
    // Submit transactions from reaper to sequencer
    SubmitBatchTxs(ctx, req SubmitBatchTxsRequest) (*SubmitBatchTxsResponse, error)

    // Get next batch for block production
    GetNextBatch(ctx, req GetNextBatchRequest) (*GetNextBatchResponse, error)

    // Verify batch was included in DA
    VerifyBatch(ctx, req VerifyBatchRequest) (*VerifyBatchResponse, error)

    // DA height tracking for forced inclusion
    SetDAHeight(height uint64)
    GetDAHeight() uint64
}
```

### Batch Structure

```go
type Batch struct {
    Transactions [][]byte

    // ForceIncludedMask[i] == true:  From DA (MUST validate)
    // ForceIncludedMask[i] == false: From mempool (already validated)
    // nil: Backward compatibility (validate all)
    ForceIncludedMask []bool
}
```

### Single Sequencer (Hybrid)

Accepts both mempool transactions and forced inclusion from DA.

**Components:**

1. **BatchQueue** - Persistent mempool storage
   ```go
   type BatchQueue struct {
       db        DB
       maxSize   uint64
       nextSeq   uint64  // Starts at 0x8000000000000000
   }

   func (q *BatchQueue) AddBatch(batch [][]byte) error
   func (q *BatchQueue) Next() ([][]byte, error)
   func (q *BatchQueue) Prepend(batch [][]byte) error  // Return unused txs
   ```

2. **Checkpoint** - Tracks position in DA epoch
   ```go
   type Checkpoint struct {
       DAHeight uint64  // Current DA height being processed
       TxIndex  uint64  // Position within epoch
   }
   ```

**GetNextBatch Flow:**

```go
func (s *SingleSequencer) GetNextBatch(ctx, req) (*Response, error) {
    // 1. Check if need next DA epoch
    if s.checkpoint.DAHeight > 0 && len(s.cachedForcedTxs) == 0 {
        s.fetchNextDAEpoch(ctx)
    }

    // 2. Process forced txs from checkpoint
    forcedTxs, forcedBytes := s.processForcedTxs(req.MaxBytes)

    // 3. Get mempool txs (remaining space)
    mempoolTxs := s.queue.Next()
    mempoolTxs = truncateToSize(mempoolTxs, req.MaxBytes - forcedBytes)

    // 4. Return unused mempool txs to queue
    s.queue.Prepend(unusedTxs)

    // 5. Combine batches
    batch := &Batch{
        Transactions:      append(forcedTxs, mempoolTxs...),
        ForceIncludedMask: makeMask(len(forcedTxs), len(mempoolTxs)),
    }

    // 6. Update and persist checkpoint
    s.updateCheckpoint()

    return &Response{Batch: batch, Timestamp: s.timestamp}
}
```

### Based Sequencer (Pure DA)

Only processes forced inclusion transactions. No mempool.

**Key Differences:**

```go
func (s *BasedSequencer) SubmitBatchTxs(ctx, req) (*Response, error) {
    // No-op: Ignores mempool transactions
    return &SubmitBatchTxsResponse{}, nil
}

func (s *BasedSequencer) GetNextBatch(ctx, req) (*Response, error) {
    // Only returns forced inclusion txs
    txs := s.fetchForcedInclusion(ctx)

    // Timestamp spread: prevents duplicate timestamps
    // timestamp = DAEpochEndTime - (remainingTxs * 1ms)
    timestamp := s.calculateSpreadTimestamp()

    return &Response{
        Batch:     &Batch{Transactions: txs},
        Timestamp: timestamp,
    }
}

func (s *BasedSequencer) VerifyBatch(ctx, req) (*Response, error) {
    // Always true: All txs come from DA (already verified)
    return &VerifyBatchResponse{Status: true}, nil
}
```

### Forced Inclusion Flow

```
User submits tx to DA forced-inclusion namespace
         │
         ▼
DA stores tx at height H
         │
         ▼
Sequencer detects epoch boundary
         │
         ▼
ForcedInclusionRetriever.Retrieve(epochStart, epochEnd)
         │
         ├── AsyncBlockRetriever checks cache
         │         │
         │         ├── Cache hit: Return cached block
         │         │
         │         └── Cache miss: Sync fetch from DA
         │
         ▼
Return ForcedInclusionEvent{Txs, Timestamp}
         │
         ▼
Sequencer caches txs, updates checkpoint
         │
         ▼
GetNextBatch returns txs with ForceIncludedMask[i]=true
         │
         ▼
Executor passes mask to execution layer
         │
         ▼
Execution layer validates forced txs (skips mempool validation)
```

### Async Block Retriever

Background prefetching reduces latency:

```go
type AsyncBlockRetriever struct {
    client        DAClient
    cache         map[uint64]*Block  // In-memory cache
    currentHeight atomic.Uint64
    prefetchSize  uint64             // 2x epoch size
    pollInterval  time.Duration      // DA block time
}

func (r *AsyncBlockRetriever) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case <-time.After(r.pollInterval):
                r.prefetch()
            }
        }
    }()
}

func (r *AsyncBlockRetriever) prefetch() {
    current := r.currentHeight.Load()
    end := current + r.prefetchSize

    for h := current; h < end; h++ {
        if _, exists := r.cache[h]; !exists {
            block, _ := r.client.Retrieve(ctx, h, namespace)
            r.cache[h] = block
        }
    }

    // Cleanup old entries
    r.cleanupBefore(current - r.prefetchSize)
}
```

---

## Integration with Block Package

### Executor Integration

```go
func (e *Executor) initializeState() error {
    state := e.store.GetState()

    // Sync sequencer DA height with stored state
    e.sequencer.SetDAHeight(state.DAHeight)

    return nil
}

func (e *Executor) produceBlock(ctx context.Context) error {
    // 1. Get batch from sequencer
    resp, _ := e.sequencer.GetNextBatch(ctx, GetNextBatchRequest{
        Id:       e.genesis.ChainID,
        MaxBytes: DefaultMaxBlobSize,
    })

    // 2. Pass ForceIncludedMask to execution layer
    ctx = WithForceIncludedMask(ctx, resp.Batch.ForceIncludedMask)

    // 3. Execute transactions
    stateRoot, _ := e.exec.ExecuteTxs(ctx, resp.Batch.Transactions, ...)

    // 4. Update state with new DA height
    newState := &State{
        DAHeight: e.sequencer.GetDAHeight(),
        // ...
    }

    // 5. Create and broadcast block
    // ...
}
```

### Configuration

```go
type DAConfig struct {
    Address                  string   // Celestia RPC endpoint
    AuthToken                string   // Auth token
    Namespace                string   // Header namespace
    DataNamespace            string   // Data namespace (optional)
    ForcedInclusionNamespace string   // Forced inclusion namespace
    BlockTime                Duration // DA block time
    SubmitOptions            string   // JSON gas settings
    SigningAddresses         []string // Round-robin addresses
    MaxSubmitAttempts        int      // Retry limit
    RequestTimeout           Duration // Per-request timeout
}

type NodeConfig struct {
    Aggregator      bool     // Enable block production
    BasedSequencer  bool     // Use based sequencer (requires Aggregator)
    BlockTime       Duration // App block time
    LazyMode        bool     // Only produce on txs
}
```

### Genesis Configuration

```go
type Genesis struct {
    DAStartHeight          uint64 // First DA height (0 at genesis)
    DAEpochForcedInclusion uint64 // Epoch size (default 50)
}
```

---

## Key Design Decisions

### 1. ForceIncludedMask Optimization

Distinguishes DA-sourced (untrusted) from mempool (trusted) transactions:
- Execution layer validates forced txs
- Skips redundant validation for mempool txs
- Significant performance improvement

### 2. Epoch-Based Processing

Only retrieves forced inclusion at epoch boundaries:
- Reduces DA queries
- Enables batching
- Checkpoint ensures resumable processing

### 3. Async Prefetching

Background goroutine prefetches 2x epoch size ahead:
- Reduces latency when sequencer needs txs
- Cache misses fall back to sync fetch
- Bounded memory via cleanup

### 4. Namespace Strategy

Three separate namespaces:
- **Header**: Block headers (required)
- **Data**: Transaction data (optional, can share with header)
- **Forced Inclusion**: User-submitted txs for censorship resistance

### 5. Crash Recovery

Both sequencers persist state:
- **Checkpoint**: DAHeight + TxIndex position
- **Queue**: Pending mempool batches
- Protobuf serialization to DB

### 6. Single vs Based Mode

| Aspect | Single | Based |
|--------|--------|-------|
| Mempool | Yes | No |
| Forced Inclusion | Yes | Yes (only source) |
| SubmitBatchTxs | Stores in queue | No-op |
| VerifyBatch | Validates proofs | Always true |
| Use Case | Traditional rollup | High liveness |
