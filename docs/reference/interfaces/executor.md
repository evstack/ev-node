# Executor Interface

The Executor interface defines how ev-node communicates with execution layers. Implement this interface to run custom execution environments on Evolve.

## Interface Definition

```go
type Executor interface {
    InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) (stateRoot []byte, err error)
    GetTxs(ctx context.Context) ([][]byte, error)
    ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) (updatedStateRoot []byte, err error)
    SetFinal(ctx context.Context, blockHeight uint64) error
    GetExecutionInfo(ctx context.Context) (ExecutionInfo, error)
    FilterTxs(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) ([]FilterStatus, error)
}
```

## Methods

### InitChain

Initializes the blockchain with genesis parameters.

```go
InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) (stateRoot []byte, err error)
```

**Parameters:**

- `genesisTime` - Chain start timestamp (UTC)
- `initialHeight` - First block height (must be > 0)
- `chainID` - Unique chain identifier

**Returns:**

- `stateRoot` - Hash representing initial state

**Requirements:**

- Must be idempotent (repeated calls return same result)
- Must validate genesis parameters
- Must generate deterministic initial state root

### GetTxs

Fetches transactions from the execution layer's mempool.

```go
GetTxs(ctx context.Context) ([][]byte, error)
```

**Returns:**

- Slice of valid transactions

**Requirements:**

- Return only currently valid transactions
- Do not remove transactions from mempool
- May remove invalid transactions

### ExecuteTxs

Processes transactions to produce a new block state.

```go
ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) (updatedStateRoot []byte, err error)
```

**Parameters:**

- `txs` - Ordered list of transactions
- `blockHeight` - Height of block being created
- `timestamp` - Block timestamp (UTC)
- `prevStateRoot` - Previous block's state root

**Returns:**

- `updatedStateRoot` - New state root after execution

**Requirements:**

- Must be deterministic
- Must handle empty transaction lists
- Must handle malformed transactions gracefully
- Must validate against previous state root

### SetFinal

Marks a block as finalized.

```go
SetFinal(ctx context.Context, blockHeight uint64) error
```

**Parameters:**

- `blockHeight` - Height to finalize

**Requirements:**

- Must be idempotent
- Must verify block exists
- Finalized blocks cannot be reverted

### GetExecutionInfo

Returns current execution layer parameters.

```go
GetExecutionInfo(ctx context.Context) (ExecutionInfo, error)
```

**Returns:**

```go
type ExecutionInfo struct {
    MaxGas uint64  // Maximum gas per block (0 = no gas-based limiting)
}
```

### FilterTxs

Validates and filters transactions for block inclusion.

```go
FilterTxs(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) ([]FilterStatus, error)
```

**Parameters:**

- `txs` - All transactions (force-included + mempool)
- `maxBytes` - Maximum cumulative size (0 = no limit)
- `maxGas` - Maximum cumulative gas (0 = no limit)
- `hasForceIncludedTransaction` - Whether force-included txs are present

**Returns:**

```go
type FilterStatus int

const (
    FilterOK       FilterStatus = iota  // Include in batch
    FilterRemove                        // Invalid, remove
    FilterPostpone                      // Valid but exceeds limits, postpone
)
```

## Optional Interfaces

### HeightProvider

Enables height synchronization checks between ev-node and the execution layer.

```go
type HeightProvider interface {
    GetLatestHeight(ctx context.Context) (uint64, error)
}
```

Useful for detecting desynchronization after crashes or restarts.

### Rollbackable

Enables automatic rollback when execution layer is ahead of consensus.

```go
type Rollbackable interface {
    Rollback(ctx context.Context, targetHeight uint64) error
}
```

Only implement if your execution layer supports in-flight rollback.

## Implementations

| Implementation | Package | Description |
|----------------|---------|-------------|
| ev-reth | `execution/evm` | EVM execution via Engine API |
| ev-abci | `execution/abci` | Cosmos SDK via ABCI |
| testapp | `apps/testapp` | Simple key-value store |

## Implementation Guide

See [Implement Custom Executor](/getting-started/custom/implement-executor) for a step-by-step guide.
