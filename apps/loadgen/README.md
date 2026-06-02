# loadgen

Standalone load generator for ev-node stress testing. Talks to a [spamoor-daemon](https://github.com/ethpandaops/spamoor) sidecar via HTTP API. Runs an in-process scheduler with configurable regular and burst workloads.

## Architecture

```
ev-loadgen (this binary)  -->  spamoor-daemon  -->  ev-reth RPC
        |                              |
   reads matrix JSON            manages wallets,
   creates/polls spammers       signs & sends txs
```

- **spamoor-daemon** needs: a funded private key + ev-reth RPC URL
- **ev-loadgen** needs: spamoor-daemon API URL + matrix JSON files

## Commands

```
ev-loadgen start                         # run continuous scheduler (regular + burst)
ev-loadgen check                         # send 1 tx to verify spamoor → ev-reth connectivity
ev-loadgen run <matrix.json>             # one-shot: run a custom matrix file
```

### start flags

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--tx-per-day` | `BENCH_TX_PER_DAY` | `1000000` | sustained txs/day |
| `--interval` | `BENCH_INTERVAL` | `1h` | regular workload frequency |
| `--burst-tx-count` | `BENCH_BURST_TX_COUNT` | `500000` | txs per burst |
| `--burst-per-day` | `BENCH_BURST_PER_DAY` | `2` | bursts per day, randomly spaced |
| `--regular-matrix` | `BENCH_REGULAR_MATRIX` | `/root/baseline.json` | path to regular matrix JSON |
| `--burst-matrix` | `BENCH_BURST_MATRIX` | `/root/burst.json` | path to burst matrix JSON |

Global flag: `--spamoor-url` (or `BENCH_SPAMOOR_URL` env, default `http://spamoor-daemon:8080`)

### Scheduling

- **Regular**: fires immediately at startup, then repeats at `--interval`. Per-run tx count = `tx-per-day / (24h / interval)`. Overrides each matrix entry's `BENCH_COUNT_PER_SPAMMER`.
- **Burst**: at startup + each midnight UTC, generates N random times across the day. Each burst overrides `BENCH_COUNT_PER_SPAMMER` = `burst-tx-count / NumSpammers`.
- **Serialization**: a mutex prevents concurrent spamoor access. If burst fires during regular (or vice versa), it waits for the lock.

## Quick Start

### 1. Start spamoor-daemon

```sh
docker run -d --name spamoor -p 8080:8080 \
  ethpandaops/spamoor:latest /app/spamoor-daemon \
  --privkey=<funded-private-key> \
  --rpchost=http://<ev-reth-host>:8545 \
  --port=8080 --startup-delay=0
```

### 2. Run loadgen

```sh
# build
cd apps/loadgen && go build -o ev-loadgen .

# run with defaults (~1M tx/day, 2 bursts/day)
./ev-loadgen start --spamoor-url=http://localhost:8080

# custom config
./ev-loadgen start \
  --spamoor-url=http://localhost:8080 \
  --tx-per-day=500000 \
  --interval=30m \
  --burst-tx-count=100000 \
  --burst-per-day=4
```

### Docker Compose

Spins up both spamoor-daemon and loadgen together:

```sh
export BENCH_PRIVATE_KEY=<funded-private-key>
export BENCH_ETH_RPC_URL=http://<ev-reth-host>:8545
docker compose -f apps/loadgen/docker-compose.yml up
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
      "env": {
        "BENCH_NUM_SPAMMERS": "4",
        "BENCH_COUNT_PER_SPAMMER": "10500",
        "BENCH_THROUGHPUT": "200",
        "BENCH_MAX_PENDING": "50000",
        "BENCH_MAX_WALLETS": "200",
        "BENCH_BASE_FEE": "500",
        "BENCH_TIP_FEE": "50"
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

When using `start`, the `BENCH_COUNT_PER_SPAMMER` value in the matrix is overridden by the computed per-run count. The matrix value is still used by the `run` command.

## Build

```sh
# binary
cd apps/loadgen && go build -o ev-loadgen .

# docker image
docker build -f apps/loadgen/Dockerfile -t ev-loadgen:dev .

# via just
just build-loadgen
just docker-build-loadgen
```
