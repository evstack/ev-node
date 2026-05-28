# Build ev-benchmarks binary
[group('bench')]
build-benchmarks:
    @echo "--> Building ev-benchmarks"
    @mkdir -p {{ build_dir }}
    @cd apps/benchmarks && go build -o {{ build_dir }}/ev-benchmarks .
    @echo "    Check the binary with: {{ build_dir }}/ev-benchmarks"

# Build ev-benchmarks Docker image
[group('bench')]
docker-build-benchmarks:
    @echo "--> Building ev-benchmarks Docker image"
    @docker build -f apps/benchmarks/Dockerfile -t ev-benchmarks:dev .
    @echo "--> Docker image built: ev-benchmarks:dev"

# Smoke test: start spamoor + benchmarks, verify txs are submitted, then tear down.
# Requires BENCH_PRIVATE_KEY and BENCH_ETH_RPC_URL env vars.
[group('bench')]
bench-smoke:
    #!/usr/bin/env bash
    set -euo pipefail

    : "${BENCH_PRIVATE_KEY:?Set BENCH_PRIVATE_KEY}"
    : "${BENCH_ETH_RPC_URL:?Set BENCH_ETH_RPC_URL}"

    export BENCH_PRIVATE_KEY BENCH_ETH_RPC_URL

    echo "--> Building ev-benchmarks binary"
    (cd apps/benchmarks && go build -o {{ build_dir }}/ev-benchmarks .)

    echo "--> Starting spamoor-daemon via docker compose"
    docker compose -f apps/benchmarks/docker-compose.yml up -d spamoor-daemon
    trap 'echo "--> Tearing down"; docker compose -f apps/benchmarks/docker-compose.yml down' EXIT

    echo "--> Waiting for spamoor-daemon to be healthy"
    for i in $(seq 1 30); do
        if curl -sf http://localhost:8080/metrics > /dev/null 2>&1; then
            echo "    spamoor-daemon ready"
            break
        fi
        if [ "$i" -eq 30 ]; then
            echo "ERROR: spamoor-daemon did not become healthy"
            exit 1
        fi
        sleep 1
    done

    echo "--> Running smoke matrix (small tx count)"
    cat > /tmp/bench-smoke.json <<'EOF'
    {
      "entries": [
        {
          "test_name": "SmokeEOA",
          "scenario": "eoatx",
          "timeout": "5m",
          "env": {
            "BENCH_NUM_SPAMMERS": "1",
            "BENCH_COUNT_PER_SPAMMER": "10",
            "BENCH_THROUGHPUT": "10",
            "BENCH_MAX_PENDING": "100",
            "BENCH_MAX_WALLETS": "10",
            "BENCH_BASE_FEE": "500",
            "BENCH_TIP_FEE": "50"
          }
        }
      ]
    }
    EOF

    {{ build_dir }}/ev-benchmarks run --spamoor-url=http://localhost:8080 /tmp/bench-smoke.json
    echo "--> Smoke test passed: transactions submitted successfully"
