# DA Debug Tool

A professional debugging tool for querying and inspecting Data Availability (DA) layer data in ev-node.

## Overview

The `da-debug` tool provides a command-line interface to interact with DA layers for debugging purposes. It offers two main commands: `query` for inspecting specific DA heights and `search` for finding blobs containing specific blockchain heights.

## Installation

Install using `go install`:

```bash
go install github.com/evstack/ev-node/tools/da-debug@main
```

After installation, the `da-debug` binary will be available in your `$GOPATH/bin` directory.

## Commands

### Query Command

Query and decode blobs at a specific DA height and namespace.

```bash
da-debug query <height> <namespace> [flags]
```

**Flags:**

- `--filter-height uint`: Filter blobs by specific blockchain height (0 = no filter)

**Examples:**

```bash
# Basic query
da-debug query 100 "my-rollup"

# Query with height filter (only show blobs containing height 50)
da-debug query 100 "my-rollup" --filter-height 50

# Query with hex namespace
da-debug query 500 "0x000000000000000000000000000000000000000000000000000000746573743031"
```

### Search Command

Search through multiple DA heights to find blobs containing a specific blockchain height.

```bash
da-debug search <start-da-height> <namespace> --target-height <height> [flags]
```

**Flags:**

- `--target-height uint`: Target blockchain height to search for (required)
- `--range uint`: Number of DA heights to search (default: 10)

**Examples:**

```bash
# Search for blockchain height 1000 starting from DA height 500
da-debug search 500 "my-rollup" --target-height 1000

# Search with custom range of 20 DA heights
da-debug search 500 "my-rollup" --target-height 1000 --range 20

# Search with hex namespace
da-debug search 100 "0x000000000000000000000000000000000000000000000000000000746573743031" --target-height 50 --range 5
```

## Global Flags

All commands support these global flags:

<!-- markdown-link-check-disable -->
- `--da-url string`: DA layer JSON-RPC URL (default: "http://localhost:7980")
<!-- markdown-link-check-enable -->
- `--auth-token string`: Authentication token for DA layer
- `--timeout duration`: Request timeout (default: 30s)
- `--verbose`: Enable verbose logging
- `--max-blob-size uint`: Maximum blob size in bytes (default: 1970176)
- `--gas-price float`: Gas price for DA operations (default: 0.0)
- `--gas-multiplier float`: Gas multiplier for DA operations (default: 1.0)
- `--no-color`: Disable colored output

## Namespace Format

Namespaces can be provided in two formats:

1. **Hex String**: A 29-byte hex string (with or without `0x` prefix)
   - Example: `0x000000000000000000000000000000000000000000000000000000746573743031`

2. **String Identifier**: Any string that gets automatically converted to a valid namespace
   - Example: `"my-app"` or `"test-namespace"`
   - The string is hashed and converted to a valid version 0 namespace
