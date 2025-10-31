# ADR 023: Event-Driven Sync Service Architecture

## Changelog

- 2025-10-31: Initial version documenting event-driven sync design

## Context

The sync service is responsible for synchronizing blocks from two sources: the Data Availability (DA) layer and P2P gossip. Previously, the syncer used a polling-based approach where it would periodically check P2P stores for new data. This created several issues:

1. **Tight Coupling**: The syncer needed direct knowledge of P2P store internals, requiring it to poll and query stores directly
2. **Inefficient Resource Usage**: Constant polling consumed CPU cycles even when no new data was available
3. **Complex Coordination**: Managing the timing between DA retrieval and P2P polling required careful orchestration
4. **Latency**: Polling intervals introduced artificial delays between when data became available and when it was processed

The goal was to decouple the syncer from P2P stores while improving efficiency and reducing latency for block synchronization.

## Alternative Approaches

### Direct Store Polling (Previous Approach)
The original design had the syncer directly polling P2P stores on a timer:
- **Pros**: Simple to understand, direct control over polling frequency
- **Cons**: Tight coupling, inefficient resource usage, artificial latency, complex coordination logic
- **Rejected**: While simpler initially, the tight coupling made the code harder to maintain and test

### Event Bus Architecture
Could have implemented a centralized event bus for all store events:
- **Pros**: Maximum flexibility, could handle many event types, well-known pattern
- **Cons**: Over-engineered for our needs, added complexity and overhead
- **Rejected**: The notification system is simpler and sufficient for our use case

### Callback-Based Approach
Could have used callbacks passed to stores:
- **Pros**: Direct communication, no intermediate components
- **Cons**: Tight coupling through callbacks, harder to test, callback hell
- **Rejected**: Would have created different coupling issues

## Decision

We implemented an **event-driven architecture using a notification system** that decouples P2P stores from the syncer:

1. **Notifier Component**: A publish-subscribe mechanism that propagates store write events
2. **Instrumented Stores**: Wrap P2P stores to publish events on writes without modifying core store logic
3. **Subscription-Based Consumption**: The syncer subscribes to store events and reacts when new data arrives
4. **Event-Triggered Processing**: Instead of polling, the syncer fetches from P2P only when notified of new data

This design follows an **observer pattern** where:
- P2P stores are the subjects (publish events)
- The syncer is the observer (subscribes and reacts)
- The notifier is the event channel (facilitates communication)

## Detailed Design

### Architecture Components

#### 1. Notifier System (`pkg/sync/notifier/`)

**Notifier**: Manages event publication and subscription
```go
type Notifier struct {
    subscribers []*Subscription
    mu          sync.RWMutex
}

func (n *Notifier) Publish(event Event)
func (n *Notifier) Subscribe() *Subscription
```

**Event Structure**:
```go
type Event struct {
    Type      EventType      // Header or Data
    Height    uint64         // Block height
    Hash      string         // Block/header hash
    Source    EventSource    // P2P or Local
    Timestamp time.Time
}
```

**Subscription**: Delivers events to subscribers via buffered channel
```go
type Subscription struct {
    C      chan Event  // Buffered channel (100 events)
    cancel func()
}
```

#### 2. Instrumented Store Pattern (`pkg/sync/sync_service.go`)

**Wrapper Design**: Decorates stores without modifying them
```go
type instrumentedStore[H header.Header[H]] struct {
    header.Store[H]           // Embedded delegate
    publish publishFn[H]      // Callback to publish events
}

func (s *instrumentedStore[H]) Append(ctx context.Context, headers ...H) error {
    // Write to delegate store first
    if err := s.Store.Append(ctx, headers...); err != nil {
        return err
    }

    // Publish event after successful write
    if len(headers) > 0 {
        s.publish(headers)
    }
    return nil
}
```

**Integration**: Applied transparently when notifier is configured
```go
if options.notifier != nil {
    svc.storeView = newInstrumentedStore[H](ss, func(headers []H) {
        svc.publish(headers, syncnotifier.SourceP2P)
    })
} else {
    svc.storeView = ss  // No overhead when notifier not used
}
```

#### 3. Event-Driven Syncer (`block/internal/syncing/syncer.go`)

**Subscription Management**: Subscribe to both header and data stores
```go
func (s *Syncer) startP2PListeners() error {
    // Subscribe to header events
    s.headerSub = s.headerStore.Notifier().Subscribe()
    go s.consumeNotifierEvents(s.headerSub, syncnotifier.EventTypeHeader)

    // Subscribe to data events
    s.dataSub = s.dataStore.Notifier().Subscribe()
    go s.consumeNotifierEvents(s.dataSub, syncnotifier.EventTypeData)

    return nil
}
```

**Event Consumption**: React to store events
```go
func (s *Syncer) consumeNotifierEvents(sub *syncnotifier.Subscription, expected syncnotifier.EventType) {
    for {
        select {
        case evt := <-sub.C:
            if evt.Type != expected {
                continue  // Ignore unexpected event types
            }
            s.tryFetchFromP2P()  // Fetch new data from P2P stores
        case <-s.ctx.Done():
            return
        }
    }
}
```

**Processing Flow**:
```
P2P Store Write → Publish Event → Notifier → Subscription Channel → Syncer → Fetch from P2P → Process Block
```

### Key Design Properties

**Separation of Concerns**:
- P2P stores: Focus on storing and serving headers/data
- Notifier: Handles event distribution
- Syncer: Coordinates block synchronization

**Loose Coupling**:
- Stores don't know about syncer
- Syncer only knows about notifier interface
- Changes to stores don't affect syncer

**Event Filtering**:
- Events include type (Header/Data) for filtering
- Source tracking (P2P/Local) prevents feedback loops
- Height/hash metadata enables decision-making

**Resource Efficiency**:
- No polling when no data available
- Buffered channels (100 events) handle bursts
- Event coalescing: multiple writes trigger single fetch

**Testability**:
- Notifier can be mocked easily
- Events can be injected for testing
- Stores can be tested independently

### Integration with Existing Architecture

**DA Retrieval**: Unchanged, continues polling DA layer
```go
func (s *Syncer) daWorkerLoop() {
    for {
        s.tryFetchFromDA(nextDARequestAt)
        time.Sleep(pollInterval)
    }
}
```

**P2P Processing**: Now event-triggered
```go
func (s *Syncer) tryFetchFromP2P() {
    currentHeight := s.store.Height()

    // Check header store
    newHeaderHeight := s.headerStore.Store().Height()
    if newHeaderHeight > currentHeight {
        s.p2pHandler.ProcessHeaderRange(currentHeight+1, newHeaderHeight, s.heightInCh)
    }

    // Check data store
    newDataHeight := s.dataStore.Store().Height()
    if newDataHeight > currentHeight {
        s.p2pHandler.ProcessDataRange(currentHeight+1, newDataHeight, s.heightInCh)
    }
}
```

**Local Broadcasts**: Prevent network spam
```go
// When syncing from DA, broadcast locally only
s.headerStore.WriteToStoreAndBroadcast(ctx, event.Header,
    pubsub.WithLocalPublication(true))
```

### Event Loss and Recovery

The event-driven design introduces a potential concern: what happens if events are dropped or the system restarts? The architecture handles these scenarios through **store-based recovery**:

#### Dropped Event Scenario

When events are dropped (e.g., subscription buffer full), the system doesn't lose data:

```
Example:
- Event 90 is published but dropped (buffer full)
- Events 91-94 are also dropped
- Event 95 arrives and is successfully delivered
- Syncer processes event 95 and calls tryFetchFromP2P()
```

**Recovery mechanism in tryFetchFromP2P()**:
```go
func (s *Syncer) tryFetchFromP2P() {
    currentHeight := s.store.Height()  // Returns 89 (last processed)

    // P2P store now has up to height 95
    newHeaderHeight := s.headerStore.Store().Height()  // Returns 95
    if newHeaderHeight > currentHeight {
        // Fetches range [90, 95] from store, recovering all dropped events
        s.p2pHandler.ProcessHeaderRange(currentHeight+1, newHeaderHeight, s.heightInCh)
    }
}
```

**Key insight**: Events are notifications, not the data itself. The P2P stores persist all data, so when any new event arrives, we query the store's height and fetch all missing blocks in the range.

#### Restart Scenario

On restart, the system state is as follows:
1. P2P stores have persisted data up to some height (e.g., 100)
2. Syncer's local store has processed blocks up to height 85
3. **No events exist** because the subscription is freshly created

**Recovery depends on triggering mechanism**:

**Scenario A - New P2P data arrives after restart**:
```
- System restarts at height 85
- P2P store has blocks 1-100 persisted
- New block 101 arrives via P2P gossip
- Store write triggers event for height 101
- tryFetchFromP2P() queries store heights and fetches range [86, 101]
- All missed blocks [86-100] are recovered, plus the new block 101
```

**Scenario B - No new P2P data (quiescent period)**:
```
- System restarts at height 85
- P2P store has blocks 1-100 persisted
- No new blocks arrive via P2P gossip
- No events are published
- Syncer waits idle (no polling)
```

**Important caveat**: Until a new event triggers `tryFetchFromP2P()`, the system won't discover existing but unprocessed data in P2P stores. This is a **deliberate trade-off**:

- **Pro**: No CPU waste polling when system is caught up
- **Con**: Restart during quiescent period delays catchup until new P2P activity
- **Mitigation**: DA layer continues polling independently, so blocks will eventually be retrieved from DA if P2P is quiet

#### DA Layer as Safety Net

The DA retrieval loop continues to poll independently:
```go
func (s *Syncer) daWorkerLoop() {
    for {
        s.tryFetchFromDA(nextDARequestAt)
        time.Sleep(pollInterval)
    }
}
```

This ensures that:
- If P2P is quiet after restart, DA retrieval will eventually catch up
- The system can't be permanently stuck, only temporarily delayed
- DA provides the authoritative source of truth regardless of P2P state

#### Design Rationale

The event-driven design prioritizes **efficiency over immediate post-restart catchup**:

1. **Normal operation**: Events trigger immediately when data arrives (optimal)
2. **Dropped events**: Next event recovers all gaps (self-healing)
3. **Restart with P2P data available**: Next P2P event recovers all gaps (self-healing)
4. **Restart during quiescence**: DA polling provides eventual consistency (safety net)

This is acceptable because:
- Restarts are infrequent compared to normal operation
- DA polling ensures eventual consistency
- The efficiency gains during normal operation (no polling) outweigh the restart delay
- Most restarts occur when new blocks are actively being produced, triggering P2P events

## Status

Accepted and Implemented

## Consequences

### Positive

- **Decoupling**: Syncer no longer depends on P2P store internals
- **Efficiency**: No CPU cycles wasted on polling when idle
- **Lower Latency**: Immediate reaction to new data (no polling interval delay)
- **Cleaner Code**: Separation of concerns makes code easier to understand

### Negative

### Neutral


## References

- Initial implementation: commits `94140e3d`, `c60a4256`
- Related ADR: ADR-003 (Peer Discovery) for P2P architecture context
- Go-header library: Used for underlying P2P synchronization primitives
- Observer pattern: Classic design pattern for decoupling publishers and subscribers
