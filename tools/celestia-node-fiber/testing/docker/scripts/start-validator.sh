#!/usr/bin/env bash
# start-validator.sh — entrypoint for each validator container.
#
# Reads the validator index from $VAL_INDEX (0..N-1), loads its home from
# the shared volume populated by init-genesis.sh, applies docker-network-
# aware overrides to config.toml/app.toml, then starts celestia-appd and
# the in-process fibre server side-by-side.
set -euo pipefail

VAL_INDEX="${VAL_INDEX:?VAL_INDEX must be set (0..N-1)}"
SHARED="${SHARED:-/shared}"
APP="${APP:-celestia-appd}"
FIBRE_BIN="${FIBRE_BIN:-fibre}"
CHAIN_ID="${CHAIN_ID:-fibre-docker}"

home="$SHARED/val$VAL_INDEX/.celestia-app"
peers=$(cat "$SHARED/peers.txt")
service_name="val$VAL_INDEX"

# Wait for init-genesis to have populated this home.
until [ -f "$home/config/genesis.json" ]; do
    echo "validator-$VAL_INDEX: waiting for genesis on $home..."
    sleep 1
done

# Apply docker-network bindings. config.toml / app.toml are TOML; use sed
# carefully on the keys we need. (A more robust approach would be `dasel`
# or a Go config helper — keeping it minimal here.)
config_toml="$home/config/config.toml"
app_toml="$home/config/app.toml"

sed -i \
    -e 's|^laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|' \
    -e 's|^laddr = "tcp://0.0.0.0:26656"|laddr = "tcp://0.0.0.0:26656"|' \
    -e "s|^persistent_peers = \"\"|persistent_peers = \"$peers\"|" \
    -e "s|^external_address = \"\"|external_address = \"$service_name:26656\"|" \
    -e 's|^priv_validator_grpc_laddr = ""|priv_validator_grpc_laddr = "127.0.0.1:26659"|' \
    "$config_toml"

sed -i \
    -e 's|^minimum-gas-prices = ""|minimum-gas-prices = "0.002utia"|' \
    -e 's|^address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' \
    -e 's|^address = "localhost:9090"|address = "0.0.0.0:9090"|' \
    -e 's|^address = "localhost:9091"|address = "0.0.0.0:9091"|' \
    "$app_toml"

# Start celestia-appd in the background. --force-no-bbr because the
# linux kernel inside docker containers on macOS does not have BBR
# congestion control enabled.
"$APP" start --home "$home" \
    --grpc.address "0.0.0.0:9090" \
    --grpc.enable true \
    --force-no-bbr &
appd_pid=$!

# Wait for the gRPC + privval gRPC ports to be reachable before launching
# fibre. Use bash's /dev/tcp instead of nc (not in slim debian).
until (exec 3<>/dev/tcp/127.0.0.1/9090) 2>/dev/null; do
    sleep 1
done
exec 3<&- 3>&-
until (exec 3<>/dev/tcp/127.0.0.1/26659) 2>/dev/null; do
    sleep 1
done
exec 3<&- 3>&-

# Wait for the chain to produce the first block — fibre requires this
# at startup to detect chain ID, otherwise it errors out and exits.
until height=$("$APP" status --home "$home" --node "tcp://127.0.0.1:26657" 2>/dev/null \
    | jq -r '.sync_info.latest_block_height // 0') \
    && [ "${height:-0}" -ge 1 ]; do
    echo "validator-$VAL_INDEX: waiting for first block (current=${height:-?})..."
    sleep 2
done

# Start the fibre server. Defaults: listens on 0.0.0.0:7980, signs via
# the validator's privval gRPC at 127.0.0.1:26659 (set above via
# priv_validator_grpc_laddr).
"$FIBRE_BIN" start \
    --home "$home/.celestia-fibre" \
    --server-listen-address "0.0.0.0:7980" \
    --app-grpc-address "127.0.0.1:9090" \
    --signer-grpc-address "127.0.0.1:26659" &
fibre_pid=$!

trap 'kill "$appd_pid" "$fibre_pid" 2>/dev/null || true' EXIT
wait "$appd_pid" "$fibre_pid"
