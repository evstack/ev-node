# Implement Executor Interface

Deep dive into each method of the Executor interface.

## Interface Overview

```go
type Executor interface {
    InitChain(ctx context.Context, genesis Genesis) ([]byte, error)
    GetTxs(ctx context.Context) ([][]byte, error)
    ExecuteTxs(ctx context.Context, txs [][]byte, height uint64, timestamp time.Time) (*ExecutionResult, error)
    SetFinal(ctx context.Context, height uint64) error
}
```

## InitChain

Called once when the chain starts for the first time.

```go
func (e *MyExecutor) InitChain(ctx context.Context, genesis Genesis) ([]byte, error)
```

**Parameters:**
- `genesis` — Contains initial state, chain ID, and configuration

**Returns:**
- Initial state root (hash of genesis state)
- Error if initialization fails

**Responsibilities:**
- Parse genesis data
- Initialize state storage
- Set up initial accounts/balances
- Return deterministic state root

**Example:**

```go
func (e *MyExecutor) InitChain(ctx context.Context, genesis Genesis) ([]byte, error) {
    // Parse genesis
    var state GenesisState
    if err := json.Unmarshal(genesis.AppState, &state); err != nil {
        return nil, err
    }

    // Initialize state
    for addr, balance := range state.Balances {
        e.db.Set([]byte(addr), []byte(balance))
    }

    // Compute and return state root
    return e.db.Hash(), nil
}
```

## GetTxs

Called by the sequencer to get pending transactions for the next block.

```go
func (e *MyExecutor) GetTxs(ctx context.Context) ([][]byte, error)
```

**Returns:**
- Slice of transaction bytes from your mempool
- Error if retrieval fails

**Responsibilities:**
- Return transactions ready for inclusion
- Optionally prioritize by fee, nonce, etc.
- Remove invalid transactions

**Example:**

```go
func (e *MyExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
    txs := e.mempool.GetPending(100) // Get up to 100 txs
    return txs, nil
}
```

## ExecuteTxs

The core execution method. Called for every block.

```go
func (e *MyExecutor) ExecuteTxs(
    ctx context.Context,
    txs [][]byte,
    height uint64,
    timestamp time.Time,
) (*ExecutionResult, error)
```

**Parameters:**
- `txs` — Ordered transactions to execute
- `height` — Block height
- `timestamp` — Block timestamp

**Returns:**
- `ExecutionResult` containing new state root and gas used
- Error only for system failures (not tx failures)

**Responsibilities:**
- Execute each transaction in order
- Update state
- Track gas usage
- Handle transaction failures gracefully
- Return new state root

**Example:**

```go
func (e *MyExecutor) ExecuteTxs(
    ctx context.Context,
    txs [][]byte,
    height uint64,
    timestamp time.Time,
) (*ExecutionResult, error) {
    var totalGas uint64

    for _, txBytes := range txs {
        tx, err := DecodeTx(txBytes)
        if err != nil {
            continue // Skip invalid tx
        }

        gas, err := e.executeTx(tx)
        if err != nil {
            // Log but continue - tx failure != block failure
            continue
        }

        totalGas += gas
    }

    // Commit state changes
    stateRoot := e.db.Commit()

    return &ExecutionResult{
        StateRoot: stateRoot,
        GasUsed:   totalGas,
    }, nil
}
```

## SetFinal

Called when a block is confirmed on the DA layer.

```go
func (e *MyExecutor) SetFinal(ctx context.Context, height uint64) error
```

**Parameters:**
- `height` — The block height that is now DA-finalized

**Responsibilities:**
- Mark state as finalized
- Prune old state if desired
- Trigger any finality-dependent logic

**Example:**

```go
func (e *MyExecutor) SetFinal(ctx context.Context, height uint64) error {
    // Mark height as final
    e.finalHeight = height

    // Optionally prune old state
    if height > 100 {
        e.db.Prune(height - 100)
    }

    return nil
}
```

## State Management Tips

1. **Determinism** — ExecuteTxs must be deterministic. Same inputs must produce same state root.

2. **Atomicity** — Either all state changes for a block commit, or none do.

3. **Crash recovery** — State should be recoverable after crash. ev-node will replay blocks if needed.

4. **Gas metering** — Track computational cost to prevent DoS.

## Testing

Test your executor in isolation:

```go
func TestExecuteTxs(t *testing.T) {
    exec := NewMyExecutor()

    // Initialize
    _, err := exec.InitChain(ctx, genesis)
    require.NoError(t, err)

    // Execute
    result, err := exec.ExecuteTxs(ctx, txs, 1, time.Now())
    require.NoError(t, err)
    require.NotEmpty(t, result.StateRoot)
}
```

## Next Steps

- [Executor Interface Reference](/reference/interfaces/executor) — Full type definitions
- [Testapp Source](https://github.com/evstack/ev-node/tree/main/apps/testapp) — Working example
