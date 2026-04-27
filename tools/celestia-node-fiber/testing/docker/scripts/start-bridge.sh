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

# celestia-node only accepts presets (celestia, mocha, arabica, ...) for
# --p2p.network. For a private chain we must set CELESTIA_CUSTOM=<netID>
# before invoking the binary; that registers the network at runtime so
# the same name passes flag validation.
export CELESTIA_CUSTOM="$NETWORK"

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
# it up without --auth-token plumbing. CELESTIA_CUSTOM prints a warning
# to stdout before the JWT, so grep for the actual token (three base64
# segments separated by dots).
celestia bridge auth admin --p2p.network "$NETWORK" --node.store "$HOME_DIR" \
    | grep -E '^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$' \
    > "$SHARED/bridge-admin-jwt.txt"

exec celestia bridge start \
    --p2p.network "$NETWORK" \
    --node.store "$HOME_DIR" \
    --core.ip "$CORE_IP" \
    --core.port "$CORE_GRPC_PORT" \
    --rpc.addr 0.0.0.0 \
    --rpc.port 26658
