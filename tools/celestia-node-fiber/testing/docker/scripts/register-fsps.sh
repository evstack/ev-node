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
FIBRE_PORT="${FIBRE_PORT:-7980}"

# Wait until the seed validator has produced a few blocks so the chain
# is healthy enough to accept txs. status command uses the --node flag
# (the home's config.toml laddr is bound to 0.0.0.0 which we can't dial
# from another container).
seed_home="$SHARED/val0/.celestia-app"
until height=$("$APP" status --home "$seed_home" --node "tcp://val0:26657" 2>/dev/null \
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
    # Register a host-reachable address (127.0.0.1:798X) so the test
    # driver running on the docker host can dial each fibre server
    # directly. compose.yaml maps val_i:7980 → host:798$i.
    host_port=$((FIBRE_PORT + i))
    "$APP" tx valaddr set-host "dns:///127.0.0.1:$host_port" \
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
