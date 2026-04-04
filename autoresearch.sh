#!/bin/bash
# Fast benchmark runner for block production performance.
# Pre-check: syntax compilation under 1s.
# Runs the benchmark 3 times and reports the median for 100_txs case.

set -euo pipefail

# Fast syntax check
go build ./block/internal/executing/ >/dev/null 2>&1 || exit 1

run_bench() {
    go test -run=^$ -bench=BenchmarkProduceBlock/100_txs -benchmem -count=1 \
        ./block/internal/executing/ 2>&1 | grep '100_txs'
}

# Run 3 times, collect results
declare -a lines=()
for i in 1 2 3; do
    lines+=("$(run_bench)")
done

# Sort by allocs (field before "allocs/op"), pick median
sorted=$(for line in "${lines[@]}"; do echo "$line"; done | sort -t'	' -k1)
median=$(echo "$sorted" | sed -n '2p')

# Extract numbers using awk
# Format: BenchmarkProduceBlock/100_txs-10   	   35155	     33517 ns/op	   25932 B/op	      81 allocs/op
ns=$(echo "$median" | awk '{for(i=1;i<=NF;i++){if($i=="ns/op")print $(i-1)}}')
bytes=$(echo "$median" | awk '{for(i=1;i<=NF;i++){if($i=="B/op")print $(i-1)}}')
allocs=$(echo "$median" | awk '{for(i=1;i<=NF;i++){if($i=="allocs/op")print $(i-1)}}')

echo "METRIC allocs_per_op=$allocs"
echo "METRIC bytes_per_op=$bytes"
echo "METRIC ns_per_op=$ns"
