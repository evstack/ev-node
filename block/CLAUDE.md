# CLAUDE.md - Block Package

This file provides guidance to Claude Code when working with the block package of ev-node.

## Package Overview

The block package is the core of ev-node's block management system. It handles block creation, validation, synchronization, and submission to the Data Availability (DA) layer. This package implements the block lifecycle from transaction aggregation through to finalization.

## Core Components

### Manager (`manager.go`)
- **Purpose**: Central orchestrator for all block operations
- **Key Responsibilities**:
  - Transaction aggregation into blocks
  - Block production and validation
  - State synchronization
  - DA layer interaction
  - P2P block/header gossiping

### Aggregation (`aggregation.go`, `lazy_aggregation_test.go`)
- **Purpose**: Collects transactions from mempool and creates blocks
- **Modes**:
  - **Normal Mode**: Produces blocks at regular intervals (BlockTime)
  - **Lazy Mode**: Only produces blocks when transactions are present or after LazyBlockTime
- **Key Functions**:
  - `AggregationLoop`: Main loop for block production
  - `lazyAggregationLoop`: Optimized for low-traffic scenarios
  - `normalAggregationLoop`: Regular block production

### Synchronization (`sync.go`, `sync_test.go`)
- **Purpose**: Keeps the node synchronized with the network
- **Key Functions**:
  - `SyncLoop`: Main synchronization loop
  - Processes headers from P2P network
  - Retrieves block data from DA layer
  - Handles header and data caching

### Data Availability (`da_includer.go`, `submitter.go`, `retriever.go`)
- **DA Includer**: Manages DA blob inclusion proofs and validation
- **Submitter**: Handles block submission to the DA layer with retry logic
- **Retriever**: Fetches blocks from the DA layer
- **Key Features**:
  - Multiple DA layer support
  - Configurable retry attempts
  - Batch submission optimization

### Storage (`store.go`, `store_test.go`)
- **Purpose**: Persistent storage for blocks and state
- **Key Features**:
  - Block height tracking
  - Block and commit storage
  - State root management
  - Migration support for namespace changes

### Pending Blocks (`pending_base.go`, `pending_headers.go`, `pending_data.go`)
- **Purpose**: Manages blocks awaiting DA inclusion or validation
- **Components**:
  - **PendingBase**: Base structure for pending blocks
  - **PendingHeaders**: Header queue management
  - **PendingData**: Block data queue management
- **Key Features**:
  - Ordered processing by height
  - Validation before acceptance
  - Memory-efficient caching

### Metrics (`metrics.go`, `metrics_helpers.go`)
- **Purpose**: Performance monitoring and observability
- **Key Metrics**:
  - Block production times
  - Sync status and progress
  - DA layer submission metrics
  - Transaction throughput

## Key Workflows

### Block Production Flow
1. Transactions collected from mempool
2. Block created with proper header and data
3. Block executed through executor
4. Block submitted to DA layer
5. Block gossiped to P2P network

### Synchronization Flow
1. Headers received from P2P network
2. Headers validated and cached
3. Block data retrieved from DA layer
4. Blocks applied to state
5. Sync progress updated

### DA Submission Flow
1. Block prepared for submission
2. Blob created with block data
3. Submission attempted with retries
4. Inclusion proof obtained
5. Block marked as finalized

## Configuration

### Time Parameters
- `BlockTime`: Target time between blocks (default: 1s)
- `DABlockTime`: DA layer block time (default: 6s)
- `LazyBlockTime`: Max time between blocks in lazy mode (default: 60s)

### Limits
- `maxSubmitAttempts`: Max DA submission retries (30)
- `defaultMempoolTTL`: Blocks until tx dropped (25)

## Testing Strategy

### Unit Tests
- Test individual components in isolation
- Mock external dependencies (DA, executor, sequencer)
- Focus on edge cases and error conditions

### Integration Tests
- Test component interactions
- Verify block flow from creation to storage
- Test synchronization scenarios

### Performance Tests (`da_speed_test.go`)
- Measure DA submission performance
- Test batch processing efficiency
- Validate metrics accuracy

## Common Development Tasks

### Adding a New DA Feature
1. Update DA interfaces in `core/da`
2. Modify `da_includer.go` for inclusion logic
3. Update `submitter.go` for submission flow
4. Add retrieval logic in `retriever.go`
5. Update tests and metrics

### Modifying Block Production
1. Update aggregation logic in `aggregation.go`
2. Adjust timing in Manager configuration
3. Update metrics collection
4. Test both normal and lazy modes

### Implementing New Sync Strategy
1. Modify `SyncLoop` in `sync.go`
2. Update pending block handling
3. Adjust cache strategies
4. Test various network conditions

## Error Handling Patterns

- Use structured errors with context
- Retry transient failures (network, DA)
- Log errors with appropriate levels
- Maintain sync state consistency
- Handle node restart gracefully

## Performance Considerations

- Batch DA operations when possible
- Use caching to reduce redundant work
- Optimize header validation path
- Monitor goroutine lifecycle
- Profile memory usage in caches

## Security Considerations

- Validate all headers before processing
- Verify DA inclusion proofs
- Check block signatures
- Prevent DOS through rate limiting
- Validate state transitions

## Debugging Tips

- Enable debug logging for detailed flow
- Monitor metrics for performance issues
- Check pending queues for blockages
- Verify DA layer connectivity
- Inspect cache hit rates

## Code Patterns to Follow

- Use context for cancellation
- Implement graceful shutdown
- Log with structured fields
- Return errors with context
- Use metrics for observability
- Test error conditions thoroughly