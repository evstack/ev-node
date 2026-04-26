#!/usr/bin/env bash
# init-genesis.sh — runs once in the bootstrap container.
#
# Generates a 4-validator celestia-app genesis under /shared, with each
# validator's home dir at /shared/val<n>/.celestia-app. All four homes
# share the same genesis.json so the chain has a coherent validator set.
# Each validator's priv_validator_key.json + node_key.json are unique
# per home dir.
#
# After this script exits, validator entrypoints can pick up their home
# from the shared volume.
set -euo pipefail

CHAIN_ID="${CHAIN_ID:-fibre-docker}"
NUM_VALIDATORS="${NUM_VALIDATORS:-4}"
SHARED="${SHARED:-/shared}"
APP="${APP:-celestia-appd}"
KEYRING_BACKEND="test"
STAKE="100000000000utia"          # 100k TIA per validator self-stake
INITIAL_BALANCE="1000000000000utia"
CLIENT_ACCOUNT="${CLIENT_ACCOUNT:-default-fibre}"
CLIENT_BALANCE="${CLIENT_BALANCE:-1000000000000utia}"

mkdir -p "$SHARED"

# Validator 0 is the seed home: we initialize there, add genesis accounts,
# then copy the resulting genesis.json into every other validator's home.
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    home="$SHARED/val$i/.celestia-app"
    mkdir -p "$home"
    "$APP" init "validator-$i" --chain-id "$CHAIN_ID" --home "$home" >/dev/null
done

seed_home="$SHARED/val0/.celestia-app"

# Add a validator key and genesis account to each home, then copy the
# pubkey/account into the seed home so it ends up in genesis.
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    home="$SHARED/val$i/.celestia-app"
    "$APP" keys add "validator" \
        --keyring-backend "$KEYRING_BACKEND" --home "$home" --output json \
        > "$SHARED/val$i/validator.key.json"
    addr=$("$APP" keys show "validator" -a \
        --keyring-backend "$KEYRING_BACKEND" --home "$home")
    "$APP" genesis add-genesis-account "$addr" "$INITIAL_BALANCE" \
        --keyring-backend "$KEYRING_BACKEND" --home "$seed_home"
done

# Add the client signer account to the seed genesis with a generous balance
# so the test driver can fund its escrow without worrying about gas.
"$APP" keys add "$CLIENT_ACCOUNT" \
    --keyring-backend "$KEYRING_BACKEND" --home "$seed_home" \
    --output json > "$SHARED/client.key.json"
client_addr=$("$APP" keys show "$CLIENT_ACCOUNT" -a \
    --keyring-backend "$KEYRING_BACKEND" --home "$seed_home")
"$APP" genesis add-genesis-account "$client_addr" "$CLIENT_BALANCE" \
    --keyring-backend "$KEYRING_BACKEND" --home "$seed_home"

# Generate gentxs from each validator's home, collect them in seed_home.
mkdir -p "$seed_home/config/gentx"
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    home="$SHARED/val$i/.celestia-app"
    if [ "$i" -ne 0 ]; then
        # Other validators need the seed's genesis.json before they can
        # produce a valid gentx.
        cp "$seed_home/config/genesis.json" "$home/config/genesis.json"
        # Their account also needs to exist in their own keyring + genesis.
        # Re-add account: gentx requires it.
        addr=$("$APP" keys show "validator" -a \
            --keyring-backend "$KEYRING_BACKEND" --home "$home")
        "$APP" genesis add-genesis-account "$addr" "$INITIAL_BALANCE" \
            --keyring-backend "$KEYRING_BACKEND" --home "$home" || true
    fi
    "$APP" genesis gentx "validator" "$STAKE" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend "$KEYRING_BACKEND" \
        --home "$home" \
        --output-document "$seed_home/config/gentx/gentx-val$i.json"
done

# Collect every gentx into the seed genesis.
"$APP" genesis collect-gentxs --home "$seed_home"
"$APP" genesis validate --home "$seed_home"

# Distribute the final genesis.json to every other validator's home.
for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
    home="$SHARED/val$i/.celestia-app"
    cp "$seed_home/config/genesis.json" "$home/config/genesis.json"
done

# Persistent peers: each validator advertises itself by its docker
# service name (val0, val1, ...) on the standard P2P port 26656.
PEERS=""
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    home="$SHARED/val$i/.celestia-app"
    nodeid=$("$APP" comet show-node-id --home "$home")
    PEERS="${PEERS}${PEERS:+,}${nodeid}@val$i:26656"
done
echo "$PEERS" > "$SHARED/peers.txt"

# TODO: write per-validator config tweaks (laddr / external_address /
# persistent_peers / minimum-gas-prices) into each home's config.toml /
# app.toml. The validator entrypoint expects these to be present already.
# Either inline here with sed/jq, or have the entrypoint apply them on
# startup.

echo "init-genesis: done. peers=$PEERS"
