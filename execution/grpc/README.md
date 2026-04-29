# gRPC Execution Client

This package provides a gRPC-based implementation of the Evolve execution interface. It allows Evolve to communicate with execution clients via gRPC using the Connect-RPC framework.

## Overview

The gRPC execution client enables separation between the consensus layer (Evolve) and the execution layer by providing a process boundary for communication. Execution clients can run on different machines over TCP, or on the same machine over a Unix domain socket to avoid TCP/IP overhead.

## Usage

### Client

To connect to a remote execution service:

```go
import (
    "github.com/evstack/ev-node/execution/grpc"
)

// Create a new gRPC client over TCP
client, err := grpc.NewClient("http://localhost:50051")
if err != nil {
    return err
}

// Or connect to an executor on the same machine over a Unix domain socket
client, err = grpc.NewClient("unix:///tmp/evolve-executor.sock")
if err != nil {
    return err
}

// Use the client as an execution.Executor
ctx := context.Background()
stateRoot, err := client.InitChain(ctx, time.Now(), 1, "my-chain")
```

### Server

To serve an execution implementation via gRPC:

```go
import (
    "net/http"
    "github.com/evstack/ev-node/execution/grpc"
)

// Wrap your executor implementation
handler := grpc.NewExecutorServiceHandler(myExecutor)

// Start the HTTP server
http.ListenAndServe(":50051", handler)
```

To serve on a Unix domain socket:

```go
import "github.com/evstack/ev-node/execution/grpc"

err := grpc.ListenAndServeUnix("/tmp/evolve-executor.sock", myExecutor)
```

## Protocol

The gRPC service is defined in `proto/evnode/v1/execution.proto` and provides the following methods:

- `InitChain`: Initialize the blockchain with genesis parameters
- `GetTxs`: Fetch transactions from the mempool
- `ExecuteTxs`: Execute transactions and update state
- `SetFinal`: Mark a block as finalized
- `GetExecutionInfo`: Return current execution limits
- `FilterTxs`: Validate and filter force-included transactions

## Features

- Full implementation of the `execution.Executor` interface
- Support for HTTP/1.1 and HTTP/2 (via h2c)
- Support for Unix domain socket connections with `unix:///path/to/socket`
- gRPC reflection for debugging and service discovery
- Compression for efficient data transfer
- Contiguous `tx_batch` transaction encoding to reduce per-transaction protobuf overhead
- Comprehensive error handling and validation

## Testing

Run the tests with:

```bash
go test ./execution/grpc/...
```

The package includes comprehensive unit tests for both client and server implementations.
