# benchmarks

Standalone load generator for ev-node stress testing. Talks to a [spamoor-daemon](https://github.com/ethpandaops/spamoor) sidecar via HTTP API. Runs on a cron schedule via [supercronic](https://github.com/aptible/supercronic).

## Architecture

```
ev-benchmarks (this binary)  -->  spamoor-daemon  -->  ev-reth RPC
        |                              |
   reads matrix JSON            manages wallets,
   creates/polls spammers       signs & sends txs
```

- **spamoor-daemon** needs: a funded private key + ev-reth RPC URL
- **ev-benchmarks** needs: spamoor-daemon API URL + a matrix JSON file

## Commands

```
ev-benchmarks check                      # send 1 tx to verify spamoor → ev-reth connectivity
ev-benchmarks regular                    # sustained ~1M tx/day (baseline matrix)
ev-benchmarks burst                      # probabilistic 500K tx burst (~15%/invocation)
ev-benchmarks run <matrix.json>          # custom matrix file
```

Global flag: `--spamoor-url` (or `BENCH_SPAMOOR_URL` env, default `http://spamoor-daemon:8080`)

## Quick Start

### 1. Start spamoor-daemon

```sh
docker run -d --name spamoor -p 8080:8080 \
  ethpandaops/spamoor:latest /app/spamoor-daemon \
  --privkey=<funded-private-key> \
  --rpchost=http://<ev-reth-host>:8545 \
  --port=8080 --startup-delay=0
```

### 2. Run benchmarks

```sh
# build
cd apps/benchmarks && go build -o ev-benchmarks .

# run
./ev-benchmarks regular --spamoor-url=http://localhost:8080
```

### Docker Compose

Spins up both spamoor-daemon and benchmarks together:

```sh
export BENCH_PRIVATE_KEY=<funded-private-key>
export BENCH_ETH_RPC_URL=http://<ev-reth-host>:8545
docker compose -f apps/benchmarks/docker-compose.yml up
```

### Smoke Test

```sh
export BENCH_PRIVATE_KEY=<funded-private-key>
export BENCH_ETH_RPC_URL=http://<ev-reth-host>:8545
just bench-smoke
```

## Matrix Format

Each entry specifies a spamoor scenario, tx counts, and optional probability:

```json
{
  "entries": [
    {
      "test_name": "EOATransfer",
      "scenario": "eoatx",
      "timeout": "15m",
      "probability": 1.0,
      "env": {
        "BENCH_NUM_SPAMMERS": "4",
        "BENCH_COUNT_PER_SPAMMER": "10500",
        "BENCH_THROUGHPUT": "200",
        "BENCH_MAX_PENDING": "50000",
        "BENCH_MAX_WALLETS": "200",
        "BENCH_BASE_FEE": "20",
        "BENCH_TIP_FEE": "2"
      }
    }
  ]
}
```

| Field | Description |
|---|---|
| `scenario` | spamoor scenario name (`eoatx`, `gasburnertx`, `erc20tx`, `uniswap-swaps`, etc.) |
| `probability` | 0.0–1.0, chance of running per invocation (omit = always run) |
| `timeout` | max duration per entry (default `15m`) |

## Schedule

Supercronic runs both matrices `@hourly`:
- `regular` — 4 × 10,500 = 42K txs/run → ~1M/day
- `burst` — 10 × 50,000 = 500K txs, 15% chance → ~3-4 bursts/day

## Build

```sh
# binary
cd apps/benchmarks && go build -o ev-benchmarks .

# docker image
docker build -f apps/benchmarks/Dockerfile -t ev-benchmarks:dev .

# via just
just build-benchmarks
just docker-build-benchmarks
```
