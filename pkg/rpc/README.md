# Evolve RPC

This package provides a Remote Procedure Call (RPC) interface for the Evolve store package, implementing ADR-017.

## Overview

The RPC implementation uses [Connect-Go](https://connectrpc.com/docs/go/getting-started/) to create a modern, lightweight RPC layer that supports both gRPC and HTTP/1.1 protocols. This allows clients to interact with a Evolve node's store remotely.

## Directory Structure

```txt
pkg/rpc/
  ├── client/       # Client implementation
  │   └── client.go
  └── server/       # Server implementation
      └── server.go
```

## Usage

### Server

To start a Store RPC server:

```go
import (
    "context"
    "log"

    "github.com/evstack/ev-node/pkg/rpc/server"
    "github.com/evstack/ev-node/pkg/store"
)

func main() {
    // Create a store instance
    myStore := store.NewKVStore(...)

    // Start the RPC server
    log.Fatal(server.StartServer(myStore, "localhost:7331"))
}
```

### Client

To use the Store RPC client:

```go
import (
    "context"
    "fmt"

    "github.com/evstack/ev-node/pkg/rpc/client"
)

func main() {
    // Create a client
    storeClient := client.NewStoreClient("http://localhost:7331")

    // Use the client to interact with the store
    ctx := context.Background()

    // Get the current height
    height, err := storeClient.GetHeight(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Current height: %d\n", height)

    // Get a block
    block err := storeClient.GetBlockByHeight(ctx, height)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Block at height %d", block)
}
```

## Features

The RPC service provides the following methods:

- `GetHeight`: Returns the current height of the store
- `GetBlock`: Returns a block by height or hash
- `GetState`: Returns the current state
- `GetMetadata`: Returns metadata for a specific key
- `SetMetadata`: Sets metadata for a specific key

## Protocol Buffers

The service is defined in `proto/evolve/v1/rpc.proto`. The protocol buffer definitions are compiled using the standard evolve build process.
