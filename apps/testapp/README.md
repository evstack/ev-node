# Test Application

Reference implementation of a key-value store rollup using ev-node. Includes a KV executor, HTTP server for transaction submission, and a stress test tool targeting 10M req/s.

## Build

```bash
# Build the testapp binary
go build -o testapp .

# Build the stress test tool
go build -o stress-test ./kv/bench/
```

## Quick Start

You need 3 terminals: one for the local DA, one for the testapp node, and one for the stress test.

```bash
# Terminal 1: Start local DA (defaults to localhost:7980)
go run ../../tools/local-da

# Terminal 2: Initialize and start the testapp
./testapp init --evnode.node.aggregator --evnode.signer.passphrase_file examples/passphrase.txt
./testapp start --kv-endpoint localhost:9090 --evnode.node.aggregator --evnode.signer.passphrase_file examples/passphrase.txt

# Terminal 3: Run the stress test
./stress-test --addr localhost:9090 --duration 10s --workers 10000
```

## Commands

| Command            | Description                           |
| ------------------ | ------------------------------------- |
| `testapp init`     | Initialize configuration and genesis  |
| `testapp start`    | Run the node (aliases: `run`, `node`) |
| `testapp rollback` | Rollback state by one height          |
| `testapp version`  | Show version info                     |
| `testapp keys`     | Manage signing keys                   |
| `testapp net-info` | Get info from a running node via RPC  |

### Key Flags for `start`

| Flag                       | Description                                       |
| -------------------------- | ------------------------------------------------- |
| `--kv-endpoint <addr>`     | Enable the KV HTTP server (e.g. `localhost:9090`) |
| `--evnode.node.aggregator` | Run as aggregator (block producer)                |
| `--evnode.node.block_time` | Block interval (default `1s`)                     |
| `--evnode.da.address`      | DA layer address                                  |
| `--home <dir>`             | Data directory (default `~/.testapp`)             |

## HTTP Endpoints

When `--kv-endpoint` is set, the following endpoints are available:

| Method | Path            | Description                                         |
| ------ | --------------- | --------------------------------------------------- |
| POST   | `/tx`           | Submit a transaction (`key=value` body)             |
| GET    | `/kv?key=<key>` | Retrieve latest value for a key                     |
| GET    | `/store`        | List all key-value pairs                            |
| GET    | `/stats`        | Get injected/executed tx counts and blocks produced |

## Stress Test

```bash
./stress-test --addr localhost:9090 --duration 10s --workers 1000
```

| Flag        | Default          | Description            |
| ----------- | ---------------- | ---------------------- |
| `-addr`     | `localhost:9090` | Server host:port       |
| `-duration` | `10s`            | Test duration          |
| `-workers`  | `1000`           | Concurrent TCP workers |

The test sends transactions via raw persistent TCP connections, reports live RPS, and prints a summary table with avg/peak req/s, server-side block stats, and whether the 10M req/s goal was reached.
