# Executor Interface

The Executor interface is the boundary between ev-node and your execution layer. ev-node calls these methods during block production and synchronization. This page documents each method, its contract, and example implementations.

## Interface Overview

```go
type Executor interface {
    InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, error)
    GetTxs(ctx context.Context) ([][]byte, error)
    ExecuteTxs(ctx context.Context, txs [][]byte, height uint64, timestamp time.Time, prevStateRoot []byte) (execution.ExecuteResult, error)
    SetFinal(ctx context.Context, height uint64) error
    GetExecutionInfo(ctx context.Context) (execution.ExecutionInfo, error)
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
    prevStateRoot []byte,
) (execution.ExecuteResult, error)
```

**Parameters:**

- `txs` — Ordered transactions to execute
- `height` — Block height
- `timestamp` — Block timestamp
- `prevStateRoot` — Previous block's state root

**Returns:**

- `execution.ExecuteResult` containing the new state root and optional next proposer address
- Error only for system failures (not tx failures)

**Responsibilities:**

- Execute each transaction in order
- Update state
- Handle transaction failures gracefully
- Return new state root

**Example:**

```go
func (e *MyExecutor) ExecuteTxs(
    ctx context.Context,
    txs [][]byte,
    height uint64,
    timestamp time.Time,
    prevStateRoot []byte,
) (execution.ExecuteResult, error) {
    for _, txBytes := range txs {
        tx, err := DecodeTx(txBytes)
        if err != nil {
            continue // Skip invalid tx
        }

        if err := e.executeTx(tx); err != nil {
            // Log but continue - tx failure != block failure
            continue
        }
    }

    // Commit state changes
    stateRoot := e.db.Commit()

    return execution.ExecuteResult{
        UpdatedStateRoot: stateRoot,
        // Empty keeps the current proposer.
        NextProposerAddress: nil,
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
    result, err := exec.ExecuteTxs(ctx, txs, 1, time.Now(), initialStateRoot)
    require.NoError(t, err)
    require.NotEmpty(t, result.UpdatedStateRoot)
}
```

## Next Steps

- [Executor Interface Reference](/reference/interfaces/executor) — Full type definitions
- [Testapp Source](https://github.com/evstack/ev-node/tree/main/apps/testapp) — Reference implementation
- [EVM Quickstart](/getting-started/evm/quickstart) — Using the EVM executor (ev-reth)
- [Cosmos SDK Quickstart](/getting-started/cosmos/quickstart) — Using the Cosmos SDK executor (ev-abci)
