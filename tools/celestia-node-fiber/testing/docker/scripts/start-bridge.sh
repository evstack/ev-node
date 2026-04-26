#!/usr/bin/env bash
# start-bridge.sh — entrypoint for the celestia-node bridge container.
#
# Initializes the bridge home, configures it to talk to val0's gRPC and
# CometBFT RPC, generates an admin JWT (written to a shared file so the
# test driver can read it), and starts the bridge.
set -euo pipefail

NETWORK="${NETWORK:-fibre-docker}"
SHARED="${SHARED:-/shared}"
HOME_DIR="${HOME_DIR:-/home/celestia/.celestia-bridge}"
CORE_IP="${CORE_IP:-val0}"
CORE_GRPC_PORT="${CORE_GRPC_PORT:-9090}"
CORE_RPC_PORT="${CORE_RPC_PORT:-26657}"

if [ ! -f "$HOME_DIR/config.toml" ]; then
    celestia bridge init --p2p.network "$NETWORK" --node.store "$HOME_DIR"
fi

# Wait for the FSP registration step to finish so blob.Subscribe has
# something meaningful to emit.
until [ -f "$SHARED/setup.done" ]; do
    echo "bridge: waiting for FSP registration..."
    sleep 2
done

# Drop an admin JWT into the shared volume so the test driver can pick
# it up without --auth-token plumbing.
celestia bridge auth admin --p2p.network "$NETWORK" --node.store "$HOME_DIR" \
    > "$SHARED/bridge-admin-jwt.txt"

exec celestia bridge start \
    --p2p.network "$NETWORK" \
    --node.store "$HOME_DIR" \
    --core.ip "$CORE_IP" \
    --core.grpc.port "$CORE_GRPC_PORT" \
    --core.rpc.port "$CORE_RPC_PORT" \
    --rpc.addr 0.0.0.0 \
    --rpc.port 26658
