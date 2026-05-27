# benchmarks

A containerised benchmark runner that executes `ev-benchmark` on an hourly cron schedule via [supercronic](https://github.com/aptible/supercronic).

## Environment Variables

The following environment variables **must** be provided at runtime:

| Variable | Required | Description | Example |
|---|---|---|---|
| `BENCH_TRACE_QUERY_URL` | No | Base URL of the OTLP trace query service | `http://<otlp-server>:10428` |
| `BENCH_PRIVATE_KEY` | Yes | Private key used to sign benchmark transactions | *(secret)* |
| `BENCH_ETH_RPC_URL` | Yes | Ethereum JSON-RPC endpoint | `http://<load-balancer-host>:8545` |

## Schedule

The benchmark runs **once per hour** (`@hourly`). The cron schedule is defined in [`crontab`](./crontab) and executed by supercronic inside the container.
You can supercharche the `/etc/crontab` to customize the job execution.

## Matrices

Benchmarks are driven by a JSON matrix file that lists which tests to run and the environment variables for each one. The default matrix is [`matrices/baseline.json`](./matrices/baseline.json) and contains the following tests: `TestGasBurner`, `TestStatePressure`, `TestMixedWorkload`, `TestDeFiSimulation`, and `TestERC20Throughput`.

You can supercharge the test suite by supplying your own matrix file. Each entry specifies a `test_name`, an optional `timeout`, and an `env` map of benchmark knobs:

```json
{
  "entries": [
    {
      "test_name": "TestGasBurner",
      "timeout": "15m",
      "env": {
        "BENCH_BLOCK_TIME": "100ms",
        "BENCH_NUM_SPAMMERS": "8",
        "BENCH_COUNT_PER_SPAMMER": "10000"
      }
    }
  ]
}
```

Mount your custom matrix into the container and point `benchmarks` binary at it:

```sh
docker run \
  -v /path/to/your/matrix.json:/root/matrix.json \
  -e BENCH_MATRIX_FILE=/root/matrix.json \
  benchmarks
```

## Build

```sh
docker build -f apps/benchmarks/Dockerfile -t benchmarks .
```
