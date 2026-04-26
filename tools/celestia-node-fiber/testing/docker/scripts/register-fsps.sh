#!/usr/bin/env bash
# register-fsps.sh — runs once after validators are producing blocks.
#
# Submits MsgSetFibreProviderInfo for each validator so the chain's
# valaddr module maps consensus address → fibre server address. The
# `dns:///` URI prefix is required by the fibre client's gRPC dialer
# (a bare host:port fails URL parsing — same gotcha documented in
# tools/talis/fibre_setup.go).
#
# Also funds the test client account's escrow so MsgPayForFibre can
# settle in the docker network.
set -euo pipefail

NUM_VALIDATORS="${NUM_VALIDATORS:-4}"
SHARED="${SHARED:-/shared}"
APP="${APP:-celestia-appd}"
CHAIN_ID="${CHAIN_ID:-fibre-docker}"
FEES="${FEES:-5000utia}"
ESCROW_AMOUNT="${ESCROW_AMOUNT:-50000000utia}"
CLIENT_ACCOUNT="${CLIENT_ACCOUNT:-default-fibre}"
FIBRE_PORT="${FIBRE_PORT:-26659}"

# Wait until the seed validator has produced a few blocks so the chain
# is healthy enough to accept txs.
seed_home="$SHARED/val0/.celestia-app"
until height=$("$APP" status --home "$seed_home" 2>/dev/null \
    | jq -r '.sync_info.latest_block_height // 0') \
    && [ "${height:-0}" -ge 3 ]; do
    echo "register-fsps: waiting for chain to reach height 3 (current=${height:-?})..."
    sleep 2
done

# Register each validator's fibre host. We submit each tx from the
# validator's own keyring, hitting that validator's local gRPC.
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    home="$SHARED/val$i/.celestia-app"
    val_oper=$("$APP" keys show "validator" --bech val -a \
        --keyring-backend test --home "$home")
    # MsgSetFibreProviderInfo via the valaddr tx CLI. Each fibre server
    # is reachable inside the compose network at val$i:$FIBRE_PORT.
    "$APP" tx valaddr set-host "dns:///val$i:$FIBRE_PORT" \
        --from validator --keyring-backend test --home "$home" \
        --chain-id "$CHAIN_ID" --node "tcp://val$i:26657" \
        --fees "$FEES" --yes
    echo "register-fsps: registered val$i ($val_oper)"
    sleep 6  # allow inclusion in the next block
done

# Fund the client account's escrow.
client_addr=$("$APP" keys show "$CLIENT_ACCOUNT" -a \
    --keyring-backend test --home "$seed_home")
"$APP" tx fibre deposit-to-escrow "$ESCROW_AMOUNT" \
    --from "$CLIENT_ACCOUNT" --keyring-backend test --home "$seed_home" \
    --chain-id "$CHAIN_ID" --node "tcp://val0:26657" \
    --fees "$FEES" --yes
echo "register-fsps: deposited $ESCROW_AMOUNT into $client_addr's escrow"

touch "$SHARED/setup.done"
echo "register-fsps: complete; setup.done flag written"
