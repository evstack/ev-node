#!/usr/bin/env bash
# run-bench.sh — convenience wrapper around `fiber-bench` for the
# common case: build the binary if missing, ensure a key exists,
# print the address, then start a run.
#
# Usage:
#   CONSENSUS_GRPC=139.59.229.101:9091 \
#   CHAIN_ID=talis-evnode \
#     ./run-bench.sh [duration] [workers]
#
# All optional flags pass through via FIBER_BENCH_ARGS.
set -euo pipefail

cd "$(dirname "$0")/../../.."

CONSENSUS_GRPC="${CONSENSUS_GRPC:-}"
CHAIN_ID="${CHAIN_ID:-}"
KEYRING_DIR="${KEYRING_DIR:-$HOME/.fiber-bench/keyring}"
KEY_NAME="${KEY_NAME:-bench}"
DURATION="${1:-${DURATION:-2m}}"
WORKERS="${2:-${WORKERS:-32}}"
TX_SIZE="${TX_SIZE:-200}"
BLOCK_TIME="${BLOCK_TIME:-1s}"
BATCHING="${BATCHING:-immediate}"
HOME_DIR="${HOME_DIR:-$HOME/.fiber-bench/node}"

if [[ -z "$CONSENSUS_GRPC" || -z "$CHAIN_ID" ]]; then
    echo "ERROR: CONSENSUS_GRPC and CHAIN_ID must be set" >&2
    echo "  example: CONSENSUS_GRPC=host:9091 CHAIN_ID=talis-evnode $0" >&2
    exit 1
fi

BIN="$(pwd)/bin/fiber-bench"
mkdir -p "$(dirname "$BIN")"

if [[ ! -x "$BIN" || -n "${REBUILD:-}" ]]; then
    echo "==> building fiber-bench (-tags fibre)"
    go build -tags fibre -o "$BIN" ./cmd/fiber-bench/
fi

# Create the bench key if missing — idempotent: `keys add` errors if the
# key exists, so we only run it on a fresh keyring.
if ! "$BIN" keys show "$KEY_NAME" --keyring-dir "$KEYRING_DIR" >/dev/null 2>&1; then
    echo "==> creating bench key '$KEY_NAME' at $KEYRING_DIR"
    "$BIN" keys add "$KEY_NAME" --keyring-dir "$KEYRING_DIR"
    echo
    echo "Top up the address above and run:"
    echo "  $BIN escrow deposit --consensus-grpc $CONSENSUS_GRPC \\"
    echo "      --keyring-dir $KEYRING_DIR --key-name $KEY_NAME --amount 50000000"
    echo
    echo "Then re-run this script."
    exit 0
fi

echo "==> bench account:"
"$BIN" keys show "$KEY_NAME" --keyring-dir "$KEYRING_DIR"

echo "==> escrow:"
"$BIN" escrow query --consensus-grpc "$CONSENSUS_GRPC" \
    --keyring-dir "$KEYRING_DIR" --key-name "$KEY_NAME" || true

echo "==> starting bench: duration=$DURATION workers=$WORKERS tx_size=$TX_SIZE block_time=$BLOCK_TIME batching=$BATCHING"
exec "$BIN" run \
    --evnode.da.fiber.consensus_address "$CONSENSUS_GRPC" \
    --evnode.da.fiber.consensus_chain_id "$CHAIN_ID" \
    --evnode.da.fiber.key_name "$KEY_NAME" \
    --keyring-dir "$KEYRING_DIR" \
    --home "$HOME_DIR" \
    --duration "$DURATION" \
    --workers "$WORKERS" \
    --tx-size "$TX_SIZE" \
    --evnode.node.block_time "$BLOCK_TIME" \
    --evnode.da.batching_strategy "$BATCHING" \
    ${FIBER_BENCH_ARGS:-}
