# Custom Executor Quickstart

Build a minimal custom executor to understand how ev-node integrates with execution layers.

This page is a fast entry point. For a method-by-method implementation walkthrough, continue with [Implement Executor Interface](/getting-started/custom/implement-executor).

## Prerequisites

- Go 1.22+
- Familiarity with Go interfaces

## 1. Start Local DA

```bash
go install github.com/evstack/ev-node/tools/local-da@latest
local-da
```

Keep this running.

## 2. Clone ev-node

```bash
git clone https://github.com/evstack/ev-node.git
cd ev-node
```

## 3. Explore the Testapp

ev-node includes a reference executor in `apps/testapp/`. This is a minimal key-value store:

```bash
ls apps/testapp/
```

Key files:

- `executor.go` — Implements the Executor interface
- `main.go` — Wires everything together

## 4. Build and Run

```bash
make build

./build/testapp init --evnode.node.aggregator --evnode.signer.passphrase secret

./build/testapp start --evnode.signer.passphrase secret
```

You should see blocks being produced.

## 5. Understand the Executor Interface

The core interface your executor must implement:

```go
type Executor interface {
    // Initialize chain state from genesis
    InitChain(ctx context.Context, genesis Genesis) (stateRoot []byte, err error)

    // Return pending transactions from mempool
    GetTxs(ctx context.Context) (txs [][]byte, err error)

    // Execute transactions and return new state root
    ExecuteTxs(ctx context.Context, txs [][]byte, height uint64, timestamp time.Time) (*ExecutionResult, error)

    // Mark a height as DA-finalized
    SetFinal(ctx context.Context, height uint64) error
}
```

## 6. Create Your Own Executor

Create a new file `my_executor.go`:

```go
package main

import (
    "context"
    "time"

    "github.com/evstack/ev-node/core/execution"
)

type MyExecutor struct {
    state map[string]string
}

func NewMyExecutor() *MyExecutor {
    return &MyExecutor{state: make(map[string]string)}
}

func (e *MyExecutor) InitChain(ctx context.Context, genesis execution.Genesis) ([]byte, error) {
    // Initialize from genesis
    return []byte("genesis-root"), nil
}

func (e *MyExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
    // Return pending transactions
    return nil, nil
}

func (e *MyExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, height uint64, timestamp time.Time) (*execution.ExecutionResult, error) {
    // Process transactions, update state
    for _, tx := range txs {
        // Your logic here
        _ = tx
    }

    return &execution.ExecutionResult{
        StateRoot: []byte("new-root"),
        GasUsed:   0,
    }, nil
}

func (e *MyExecutor) SetFinal(ctx context.Context, height uint64) error {
    // Height is now DA-finalized
    return nil
}
```

## 7. Wire It Up

See `apps/testapp/main.go` for how to create a full node with your executor:

```go
executor := NewMyExecutor()

node, err := node.NewFullNode(
    ctx,
    config,
    executor,
    // ... other options
)
```

## Next Steps

- [Implement Executor](/getting-started/custom/implement-executor) — Deep dive into each method
- [Executor Interface Reference](/reference/interfaces/executor) — Full interface documentation
- [Testapp Source](https://github.com/evstack/ev-node/tree/main/apps/testapp) — Reference implementation
