# Rotate proposer key

Use this guide to rotate a sequencer proposer key without restarting the chain. The active proposer is selected from `proposer_schedule` in `genesis.json` based on block height.

## Before you start

- This is a coordinated network upgrade. Every node must run a binary that supports `proposer_schedule`.
- Every node must use the same updated `genesis.json` before the activation height.
- `ev-node` loads `genesis.json` when the node starts. Updating the file on disk is not enough; you must restart nodes after replacing it.
- The old proposer key remains valid until the block before the activation height. If the old key cannot safely produce until then, stop the sequencer and coordinate operator recovery first.

## How proposer rotation is stored in genesis

`proposer_address` and `proposer_schedule[].address` are base64-encoded strings in JSON.

```json
{
  "initial_height": 1,
  "proposer_address": "0FQmA4Hn9dn8m4ZpM4+fV4e8KhkWjI4V2Vt1j9Qm5pA=",
  "proposer_schedule": [
    {
      "start_height": 1,
      "address": "0FQmA4Hn9dn8m4ZpM4+fV4e8KhkWjI4V2Vt1j9Qm5pA="
    },
    {
      "start_height": 125000,
      "address": "Y7z5v9mQm4Nw6mD0a2yR9kD2B0qv5iJj1Q1R7gD4B7Q="
    }
  ]
}
```

Rules enforced by `ev-node`:

- `proposer_schedule[0].start_height` must equal `initial_height`
- schedule entries must be strictly increasing by `start_height`
- if `proposer_address` is set, it must match the first schedule entry

Keep all earlier schedule entries. Fresh full nodes need them to validate historical blocks.

## 1. Pick an activation height

Choose an activation height `H` far enough in the future that you can distribute the updated genesis and restart every non-producing node before the cutover.

```bash
ACTIVATION_HEIGHT=125000
GENESIS="$HOME/.evnode/config/genesis.json"
INITIAL_HEIGHT="$(jq -r '.initial_height' "$GENESIS")"
```

## 2. Get the current and replacement proposer public keys

For a file-based signer, the signer public key is stored in `signer.json` as base64. You only put the derived address into genesis, but you still need the public key once to compute that address.

```bash
OLD_SIGNER_DIR="$HOME/.evnode/config"
NEW_SIGNER_DIR="/secure/path/new-signer"

OLD_PROPOSER_PUBKEY="$(jq -r '.pub_key' "$OLD_SIGNER_DIR/signer.json")"
NEW_PROPOSER_PUBKEY="$(jq -r '.pub_key' "$NEW_SIGNER_DIR/signer.json")"
```

If you use a KMS-backed signer, export the replacement Ed25519 public key from your signer flow and base64-encode the raw public key bytes in the same format. The runtime configuration stays the same as in the [AWS KMS signer guide](./aws-kms-signer.md).

## 3. Derive proposer addresses from the public keys

`ev-node` derives the proposer address as `sha256(raw_pubkey_bytes)`. The helper below prints the address in the base64 format used by `genesis.json`.

```bash
proposer_address() {
  python3 - "$1" <<'PY'
import base64
import hashlib
import sys

pub_key = base64.b64decode(sys.argv[1])
address = hashlib.sha256(pub_key).digest()
print(base64.b64encode(address).decode())
PY
}

OLD_PROPOSER_ADDRESS="$(proposer_address "$OLD_PROPOSER_PUBKEY")"
NEW_PROPOSER_ADDRESS="$(proposer_address "$NEW_PROPOSER_PUBKEY")"
```

## 4. Update `genesis.json`

### If your chain only has `proposer_address` today

Create an explicit schedule with the current proposer at `initial_height` and the new proposer at `ACTIVATION_HEIGHT`.

```bash
jq \
  --arg old_addr "$OLD_PROPOSER_ADDRESS" \
  --arg new_addr "$NEW_PROPOSER_ADDRESS" \
  --argjson initial_height "$INITIAL_HEIGHT" \
  --argjson activation_height "$ACTIVATION_HEIGHT" \
  '
  .proposer_address = $old_addr
  | .proposer_schedule = [
      {
        start_height: $initial_height,
        address: $old_addr
      },
      {
        start_height: $activation_height,
        address: $new_addr
      }
    ]
  ' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"
```

### If your chain already has `proposer_schedule`

Append the new entry. Do not replace older entries, and make sure `ACTIVATION_HEIGHT` is greater than the last scheduled `start_height`.

```bash
jq \
  --arg new_addr "$NEW_PROPOSER_ADDRESS" \
  --argjson activation_height "$ACTIVATION_HEIGHT" \
  '
  .proposer_schedule += [
    {
      start_height: $activation_height,
      address: $new_addr
    }
  ]
  ' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"
```

Verify the result before you distribute it:

```bash
jq '{initial_height, proposer_address, proposer_schedule}' "$GENESIS"
```

## 5. Distribute the updated genesis and restart followers

Copy the same `genesis.json` to every full node, replica, and failover node. Restart them after copying the file so they load the updated schedule.

Do this before the chain reaches `ACTIVATION_HEIGHT`.

## 6. Cut over the sequencer

Wait until the chain reaches `ACTIVATION_HEIGHT - 1`, then stop the old sequencer and start it with the replacement signer.

Example with a file-based signer:

```bash
evnode start \
  --home "$HOME/.evnode" \
  --evnode.node.aggregator \
  --evnode.signer.signer_type file \
  --evnode.signer.signer_path "$NEW_SIGNER_DIR" \
  --evnode.signer.passphrase "$SIGNER_PASSPHRASE"
```

If you run a custom chain binary such as `gmd` or `appd`, use the same start command you already use for the sequencer and only change the signer configuration.

## 7. Verify the first post-upgrade block

Fetch the header at `ACTIVATION_HEIGHT` or the next produced block and confirm that it carries the new proposer address.

```bash
curl -s http://127.0.0.1:26657/header \
  -H 'Content-Type: application/json' \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"header\",\"params\":{\"height\":\"${ACTIVATION_HEIGHT}\"},\"id\":1}" \
  | jq .
```

Some RPC clients render binary fields as hex instead of base64. If needed, convert the base64 genesis address before comparing:

```bash
python3 - "$NEW_PROPOSER_ADDRESS" <<'PY'
import base64
import sys

print("0x" + base64.b64decode(sys.argv[1]).hex())
PY
```

If the node at `ACTIVATION_HEIGHT` is still signed by the old key, stop block production and check three things first:

1. every node was restarted after receiving the updated genesis
2. `proposer_schedule` contains the new entry at the intended height
3. the sequencer is actually running with the replacement signer
