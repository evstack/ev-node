# Block Package Architecture

Complete technical reference for the ev-node block package.

## Directory Structure

```
block/
├── public.go                    # Exported types, DA client factory
├── components.go                # Component creation and lifecycle
└── internal/
    ├── common/
    │   ├── errors.go            # Error definitions
    │   ├── event.go             # DAHeightEvent, event types
    │   ├── metrics.go           # Prometheus metrics
    │   ├── options.go           # BlockOptions configuration
    │   ├── expected_interfaces.go
    │   └── replay.go            # Replayer for crash recovery
    ├── executing/
    │   └── executor.go          # Block production loop
    ├── syncing/
    │   ├── syncer.go            # Main sync orchestration
    │   ├── da_retriever.go      # DA block fetching
    │   └── p2p_handler.go       # P2P block coordination
    ├── submitting/
    │   ├── submitter.go         # Main submission loop
    │   └── da_submitter.go      # DA submission with retries
    ├── reaping/
    │   └── reaper.go            # Transaction scraping
    ├── cache/
    │   ├── manager.go           # Unified cache interface
    │   ├── generic_cache.go     # Generic cache impl
    │   ├── pending_headers.go   # Header tracking
    │   └── pending_data.go      # Data tracking
    └── da/
        ├── client.go            # DA client wrapper
        ├── interface.go         # DA interfaces
        ├── async_block_retriever.go
        └── forced_inclusion_retriever.go
```

## Component Lifecycle

All components implement:

```go
type Component interface {
    Start(ctx context.Context) error
    Stop() error
}
```

Startup order:

1. Cache Manager (loads persisted state)
2. Syncer (begins sync workers)
3. Executor (begins production loop) - Aggregator only
4. Reaper (begins tx scraping) - Aggregator only
5. Submitter (begins DA submission)

## Executor (`internal/executing/executor.go`)

Block production for aggregator nodes.

### State

```go
type Executor struct {
    lastState      *atomic.Pointer[types.State]
    sequencer      Sequencer
    exec           Executor
    broadcaster    Broadcaster
    submitter      Submitter
    cache          Cache

    blockTime      time.Duration
    lazyMode       bool
    maxPending     uint64
}
```

### Main Loop

```go
func (e *Executor) executionLoop(ctx context.Context) {
    timer := time.NewTimer(e.blockTime)

    for {
        select {
        case <-ctx.Done():
            return
        case <-timer.C:
            e.produceBlock(ctx)
            timer.Reset(e.blockTime)
        case <-e.txNotifyCh:
            // New txs arrived, produce immediately if not lazy
            if !e.lazyMode {
                e.produceBlock(ctx)
                timer.Reset(e.blockTime)
            }
        }
    }
}
```

### Block Production

```go
func (e *Executor) produceBlock(ctx context.Context) error {
    // 1. Check backpressure
    if e.cache.PendingCount() >= e.maxPending {
        return ErrTooManyPending
    }

    // 2. Get batch from sequencer
    batch, err := e.sequencer.GetNextBatch(ctx)

    // 3. Execute transactions
    stateRoot, gasUsed, err := e.exec.ExecuteTxs(ctx, batch.Txs, ...)

    // 4. Create header
    header := &types.Header{
        Height:          lastState.LastBlockHeight + 1,
        Time:            time.Now().UnixNano(),
        LastHeaderHash:  lastState.LastHeaderHash,
        DataHash:        batch.Txs.Hash(),
        AppHash:         stateRoot,
        ProposerAddress: e.proposer,
    }

    // 5. Sign header
    signedHeader, err := e.signer.SignHeader(header)

    // 6. Create data
    data := &types.Data{Txs: batch.Txs}

    // 7. Update state
    newState := lastState.NextState(header, stateRoot)
    e.lastState.Store(newState)

    // 8. Broadcast to P2P
    e.broadcaster.BroadcastHeader(ctx, signedHeader)
    e.broadcaster.BroadcastData(ctx, data)

    // 9. Queue for DA submission
    e.submitter.AddPending(signedHeader, data)

    return nil
}
```

## Syncer (`internal/syncing/syncer.go`)

Coordinates block synchronization from multiple sources.

### Workers

```go
func (s *Syncer) startSyncWorkers(ctx context.Context) {
    go s.daWorkerLoop(ctx)          // DA retrieval
    go s.pendingWorkerLoop(ctx)     // Pending events
    go s.p2pWorkerLoop(ctx)         // P2P blocks
}
```

### DA Worker

```go
func (s *Syncer) daWorkerLoop(ctx context.Context) {
    for {
        // Get next DA height to retrieve
        height := s.daRetrieverHeight.Load()

        // Retrieve blocks at this DA height
        events, err := s.daRetriever.Retrieve(ctx, height)

        // Send to processing channel
        for _, event := range events {
            s.heightInCh <- event
        }

        // Advance DA height
        s.daRetrieverHeight.Add(1)
    }
}
```

### P2P Worker

```go
func (s *Syncer) p2pWorkerLoop(ctx context.Context) {
    for {
        select {
        case header := <-s.p2pHandler.HeaderCh():
            s.p2pHandler.HandleHeader(header)
        case data := <-s.p2pHandler.DataCh():
            s.p2pHandler.HandleData(data)
        case event := <-s.p2pHandler.EventCh():
            // Complete header+data pair received
            s.heightInCh <- event
        }
    }
}
```

### Process Loop

```go
func (s *Syncer) processLoop(ctx context.Context) {
    for {
        select {
        case event := <-s.heightInCh:
            if err := s.processHeightEvent(ctx, event); err != nil {
                // Log error, continue
            }
        case <-ctx.Done():
            return
        }
    }
}

func (s *Syncer) processHeightEvent(ctx context.Context, event DAHeightEvent) error {
    // 1. Validate header signature
    if err := s.verifyHeader(event.SignedHeader); err != nil {
        return err
    }

    // 2. Validate data hash matches header
    if event.SignedHeader.DataHash != event.Data.Hash() {
        return ErrDataHashMismatch
    }

    // 3. Execute transactions
    stateRoot, _, err := s.exec.ExecuteTxs(ctx, event.Data.Txs, ...)

    // 4. Verify state root
    if stateRoot != event.SignedHeader.AppHash {
        return ErrStateRootMismatch
    }

    // 5. Update state
    newState := s.lastState.NextState(event.SignedHeader.Header, stateRoot)
    s.lastState.Store(newState)

    // 6. Persist to store
    s.store.SaveBlock(event.SignedHeader, event.Data, newState)

    return nil
}
```

## Submitter (`internal/submitting/submitter.go`)

Manages DA submission with retries and inclusion tracking.

### Two Loops

```go
func (s *Submitter) Start(ctx context.Context) error {
    go s.daSubmissionLoop(ctx)        // Submit to DA
    go s.inclusionProcessingLoop(ctx) // Track inclusion
    return nil
}
```

### DA Submission Loop

```go
func (s *Submitter) daSubmissionLoop(ctx context.Context) {
    for {
        // Get pending headers
        headers := s.cache.GetPendingHeaders()
        if len(headers) > 0 {
            if err := s.submitHeaders(ctx, headers); err != nil {
                s.handleSubmitError(err)
                continue
            }
        }

        // Get pending data
        data := s.cache.GetPendingData()
        if len(data) > 0 {
            if err := s.submitData(ctx, data); err != nil {
                s.handleSubmitError(err)
                continue
            }
        }

        time.Sleep(s.submitInterval)
    }
}
```

### Retry Policy

```go
type DASubmitter struct {
    maxRetries     int
    initialBackoff time.Duration
    maxBackoff     time.Duration
}

func (d *DASubmitter) Submit(ctx context.Context, blob []byte) error {
    backoff := d.initialBackoff

    for attempt := 0; attempt < d.maxRetries; attempt++ {
        status, err := d.client.Submit(ctx, blob)

        switch status {
        case StatusSuccess:
            return nil
        case StatusTooBig:
            return d.splitAndSubmit(ctx, blob)
        case StatusAlreadyInMempool:
            return nil // Already submitted
        case StatusNotIncludedInBlock:
            time.Sleep(backoff)
            backoff = min(backoff*2, d.maxBackoff)
            continue
        default:
            return err
        }
    }

    return ErrMaxRetriesExceeded
}
```

## Forced Inclusion (`internal/da/forced_inclusion_retriever.go`)

Prevents sequencer censorship.

### Grace Period Calculation

```go
func (r *ForcedInclusionRetriever) calculateGracePeriod() uint64 {
    // Base period: 1 epoch
    basePeriod := r.epochLength

    // Adjust based on block fullness
    // Higher fullness = longer grace period (congestion tolerance)
    ema := r.blockFullnessEMA.Load()

    if ema > 0.8 {
        // High congestion, extend grace period
        return basePeriod * 2
    }

    return basePeriod
}
```

### Pending TX Tracking

```go
type PendingForcedTx struct {
    Tx            types.Tx
    DAHeight      uint64    // When tx appeared in DA
    GraceDeadline uint64    // DA height deadline for inclusion
}

func (r *ForcedInclusionRetriever) checkPending(currentDAHeight uint64) {
    for _, pending := range r.pendingTxs {
        if currentDAHeight > pending.GraceDeadline {
            // Sequencer failed to include tx
            r.markSequencerMalicious(pending)
            // Force include the tx
            r.forceInclude(pending.Tx)
        }
    }
}
```

## Cache Manager (`internal/cache/manager.go`)

Unified cache for headers, data, and transactions.

### Structure

```go
type Manager struct {
    headerCache    *GenericCache[types.Hash, HeaderEntry]
    dataCache      *GenericCache[types.Hash, DataEntry]
    txCache        *GenericCache[types.Hash, TxEntry]
    pendingEvents  map[uint64]*DAHeightEvent

    cleanupTicker  *time.Ticker
    retentionTime  time.Duration
}
```

### Key Operations

```go
// Header tracking
func (m *Manager) IsHeaderSeen(hash types.Hash) bool
func (m *Manager) SetHeaderSeen(hash types.Hash, height uint64)
func (m *Manager) GetHeaderDAIncluded(hash types.Hash) (uint64, bool)
func (m *Manager) SetHeaderDAIncluded(hash types.Hash, daHeight uint64)

// Transaction deduplication
func (m *Manager) IsTxSeen(hash types.Hash) bool
func (m *Manager) SetTxSeen(hash types.Hash)

// Pending management
func (m *Manager) GetPendingHeaders() []*types.SignedHeader
func (m *Manager) GetPendingData() []*types.Data
func (m *Manager) PendingCount() uint64
```

### Disk Persistence

```go
func (m *Manager) SaveToDisk(path string) error {
    state := &CacheState{
        Headers:  m.headerCache.Entries(),
        Data:     m.dataCache.Entries(),
        Pending:  m.pendingEvents,
    }
    return json.WriteFile(path, state)
}

func (m *Manager) LoadFromDisk(path string) error {
    state, err := json.ReadFile(path)
    // Restore caches from state
}
```

## Replayer (`internal/common/replay.go`)

Syncs execution layer after crash.

```go
func (r *Replayer) Replay(ctx context.Context) error {
    // Get heights
    storeHeight := r.store.GetLastHeight()
    execHeight := r.exec.GetHeight()

    if execHeight >= storeHeight {
        return nil // Already synced
    }

    // Replay missing blocks
    for height := execHeight + 1; height <= storeHeight; height++ {
        header, data, err := r.store.GetBlock(height)
        if err != nil {
            return err
        }

        _, _, err = r.exec.ExecuteTxs(ctx, data.Txs, ...)
        if err != nil {
            return err
        }
    }

    return nil
}
```

## Metrics (`internal/common/metrics.go`)

```go
var (
    Height              = prometheus.NewGauge(...)
    NumTxs              = prometheus.NewGauge(...)
    BlockSizeBytes      = prometheus.NewHistogram(...)
    CommittedHeight     = prometheus.NewGauge(...)
    TxsPerBlock         = prometheus.NewHistogram(...)
    OperationDuration   = prometheus.NewHistogramVec(...)

    // DA metrics
    DASubmitterFailures     = prometheus.NewCounterVec(...)
    DASubmitterLastFailure  = prometheus.NewGauge(...)
    DASubmitterPendingBlobs = prometheus.NewGauge(...)
    DARetrievalAttempts     = prometheus.NewCounter(...)
    DARetrievalSuccesses    = prometheus.NewCounter(...)
    DARetrievalFailures     = prometheus.NewCounter(...)
    DAInclusionHeight       = prometheus.NewGauge(...)

    // Forced inclusion
    ForcedInclusionTxsInGracePeriod = prometheus.NewGauge(...)
    ForcedInclusionTxsMalicious     = prometheus.NewCounter(...)
)
```

## Configuration

Key options in `BlockOptions`:

```go
type BlockOptions struct {
    BlockTime                time.Duration  // Block interval
    LazyBlockInterval        time.Duration  // Lazy mode timeout
    MaxPendingHeadersAndData uint64         // Backpressure limit
    BasedSequencer          bool            // No DA submissions
    DABlockTime             time.Duration   // DA block interval
    ScrapeInterval          time.Duration   // Tx reaping frequency

    // Namespaces
    HeaderNamespace          []byte
    DataNamespace            []byte
    ForcedInclusionNamespace []byte
}
```

## Error Types

```go
var (
    ErrNoHeader           = errors.New("no header found")
    ErrNoData             = errors.New("no data found")
    ErrDataHashMismatch   = errors.New("data hash does not match header")
    ErrStateRootMismatch  = errors.New("state root mismatch after execution")
    ErrInvalidSignature   = errors.New("invalid header signature")
    ErrTooManyPending     = errors.New("too many pending submissions")
    ErrMaxRetriesExceeded = errors.New("max DA submission retries exceeded")
    ErrSequencerMalicious = errors.New("sequencer failed to include forced tx")
)
```

## State Machines

### Executor State Machine

```
┌──────────────┐
│   IDLE       │
└──────┬───────┘
       │ BlockTime elapsed OR TxNotify
       ▼
┌──────────────┐
│ CHECK_PENDING│──── Too many? ───► Wait
└──────┬───────┘
       │ OK
       ▼
┌──────────────┐
│ GET_BATCH    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ EXECUTE_TXS  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ CREATE_BLOCK │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ BROADCAST    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ QUEUE_SUBMIT │───► Back to IDLE
└──────────────┘
```

### Syncer State Machine

```
┌─────────────────────────────────────────┐
│              START                       │
└──────────────────┬──────────────────────┘
                   │
    ┌──────────────┼──────────────┐
    │              │              │
    ▼              ▼              ▼
┌───────┐    ┌───────┐    ┌───────────┐
│  DA   │    │  P2P  │    │ FORCED    │
│WORKER │    │WORKER │    │ INCLUSION │
└───┬───┘    └───┬───┘    └─────┬─────┘
    │            │              │
    └────────────┴──────────────┘
                 │
                 ▼
         ┌──────────────┐
         │ PROCESS_LOOP │
         └──────┬───────┘
                │
                ▼
         ┌──────────────┐
         │   VALIDATE   │──── Invalid? ───► Log, skip
         └──────┬───────┘
                │ Valid
                ▼
         ┌──────────────┐
         │   EXECUTE    │
         └──────┬───────┘
                │
                ▼
         ┌──────────────┐
         │  UPDATE_STATE│
         └──────┬───────┘
                │
                ▼
         ┌──────────────┐
         │    PERSIST   │───► Back to PROCESS_LOOP
         └──────────────┘
```

### Submitter State Machine

```
┌──────────────┐
│    START     │
└──────┬───────┘
       │
       ├─────────────────────┐
       │                     │
       ▼                     ▼
┌──────────────┐    ┌──────────────────┐
│ SUBMIT_LOOP  │    │ INCLUSION_LOOP   │
└──────┬───────┘    └────────┬─────────┘
       │                     │
       ▼                     ▼
┌──────────────┐    ┌──────────────────┐
│ GET_PENDING  │    │ CHECK_DA_HEIGHT  │
└──────┬───────┘    └────────┬─────────┘
       │                     │
       ▼                     │ Included?
┌──────────────┐             │
│   SUBMIT     │             ▼
└──────┬───────┘    ┌──────────────────┐
       │            │ RESET_STATE      │
       │ Failed?    └────────┬─────────┘
       ▼                     │
┌──────────────┐             │
│   RETRY      │             │
│  (backoff)   │             │
└──────┬───────┘             │
       │                     │
       └─────────────────────┘
```
