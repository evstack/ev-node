# KV Executor Stress Test

Stress test for the KV executor HTTP server. Sends transactions via raw TCP connections to maximize throughput, targeting 10M req/s.

## Building

```bash
go build -o stress-test ./kv/bench/
```

## Usage

Run the testapp first, then the stress test alongside it:

```bash
# Terminal 1: start the testapp with KV endpoint
./build/testapp start --kv-endpoint localhost:9090

# Terminal 2: run the stress test
./stress-test --addr localhost:9090 --duration 10s --workers 1000
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `localhost:9090` | Server host:port |
| `-duration` | `10s` | Test duration |
| `-workers` | `1000` | Number of concurrent TCP workers |
| `-target-rps` | `10000000` | Target requests per second (goal) |

## Output

The tool prints live progress (req/s) during the run, then a summary table with:

- Avg req/s and peak req/s
- Total requests, successes, and failures
- Server-side stats: blocks produced, txs executed, avg txs per block
- Whether the 10M req/s goal was reached (displays `SUCCESS` if so)

## /stats Endpoint

The KV executor HTTP server exposes a `/stats` endpoint (GET) returning:

```json
{
  "injected_txs": 52345678,
  "executed_txs": 1234567,
  "blocks_produced": 1000
}
```
