# Migrate from Single Sequencer to HA Cluster

This guide walks through converting a live single-sequencer chain into a 5-node Raft HA cluster with zero block-production downtime during the cutover window.

## Overview

A single sequencer stores its block production state (latest height, hash, and timestamp) only locally. A Raft cluster shares this state across all nodes via the Raft log. To migrate, you:

1. Prepare four new nodes with the same genesis, signer key, and chain data as the existing node.
2. Reconfigure all five nodes (existing + four new) with Raft enabled.
3. Stop the existing sequencer and start all five nodes together to bootstrap the cluster.

The chain experiences one planned downtime window — the gap between stopping the single sequencer and the Raft cluster electing its first leader, which takes a few seconds.

## Before You Start

### Understand what changes

| | Single sequencer | Raft cluster |
|-|-----------------|-------------|
| Produces blocks | One node always | Elected leader |
| Block production key | Local to one node | Shared across all nodes |
| Raft data directory | Not used | Required, persistent |
| Config flags | No `raft.*` flags | All `raft.*` flags required |
| Restart behavior | Manual recovery | Automatic leader election |

### Requirements

- Five machines that can reach each other on the Raft port (we use `5001`) and P2P port (`26656`)
- A private network or VPN between all nodes (Raft transport is unencrypted)
- The existing sequencer's:
  - Binary (`evm` or your chain binary)
  - Config file (`evnode.yaml`)
  - Signer key
  - Genesis file
  - Block data directory (optional — peers can sync from DA, but copying saves time)
- A scheduled maintenance window of ~5 minutes

---

## Step 1: Measure Network RTT

Before writing any config, measure the maximum RTT between your nodes. The Raft timing parameters must be sized for your actual network:

```bash
# From the existing node, ping each new node
for ip in 10.0.0.2 10.0.0.3 10.0.0.4 10.0.0.5; do
  echo -n "$ip: "
  ping -c 20 $ip | tail -1 | awk -F'/' '{print $5 "ms avg"}'
done
```

For RTT_MAX ≤ 25ms (same-region nodes), use the recommended values in this guide. For higher RTT, adjust using the formulas in the [configuration reference](./overview.md#timing-parameters).

---

## Step 2: Provision the Four New Nodes

On each of the four new machines, install the same ev-node binary version as the existing sequencer.

```bash
# Verify the binary version matches on all machines
./evm version
```

Create the passphrase file before initializing so the signer key is encrypted from the start:

```bash
# On each new node
sudo mkdir -p /etc/ev-node
echo -n "<YOUR_PASSPHRASE>" | sudo tee /etc/ev-node/passphrase > /dev/null
sudo chmod 600 /etc/ev-node/passphrase
```

Initialize each new node's home directory:

```bash
# On each new node
./evm init --evnode.node.aggregator=true --evnode.signer.passphrase_file /etc/ev-node/passphrase
```

---

## Step 3: Copy the Signer Key to All New Nodes

All five nodes must sign blocks with the **same key**. The existing sequencer's key is the one to use — do not generate new keys on the new nodes.

```bash
# On the existing sequencer (node-1)
# Locate the signer key (exact filename depends on your chain)
ls ~/.evm/config/

# Copy to each new node
scp ~/.evm/config/signer.json user@10.0.0.2:~/.evm/config/
scp ~/.evm/config/signer.json user@10.0.0.3:~/.evm/config/
scp ~/.evm/config/signer.json user@10.0.0.4:~/.evm/config/
scp ~/.evm/config/signer.json user@10.0.0.5:~/.evm/config/
```

---

## Step 4: Copy the Genesis File

```bash
scp ~/.evm/config/genesis.json user@10.0.0.2:~/.evm/config/
scp ~/.evm/config/genesis.json user@10.0.0.3:~/.evm/config/
scp ~/.evm/config/genesis.json user@10.0.0.4:~/.evm/config/
scp ~/.evm/config/genesis.json user@10.0.0.5:~/.evm/config/
```

---

## Step 5: Copy Block Data to New Nodes

New nodes can sync their block history from the DA layer or P2P peers after the cluster is running, but copying the existing chain data speeds up the initial sync significantly for long-running chains.

```bash
# Stop the existing sequencer temporarily to get a consistent snapshot
# (you will start it again in step 9)
systemctl stop ev-node   # or kill the process

# Copy the data directory — adjust the path to your chain
rsync -avz ~/.evm/data/ user@10.0.0.2:~/.evm/data/
rsync -avz ~/.evm/data/ user@10.0.0.3:~/.evm/data/
rsync -avz ~/.evm/data/ user@10.0.0.4:~/.evm/data/
rsync -avz ~/.evm/data/ user@10.0.0.5:~/.evm/data/
```

> If your chain uses an EVM execution layer (ev-reth), copy the execution layer database as well. See the [Reth backup guide](../evm/reth-backup.md) for the correct procedure.

After the copy, note the **latest block height** — this is your reference point:

```bash
# Note the height before shutdown — replace 8545 with your EVM RPC port
cast block latest --rpc-url http://10.0.0.1:8545 | jq -r '.number'
```

**Restart the existing sequencer now** so the chain keeps producing blocks while you prepare the remaining nodes (Steps 6–8). The chain will run uninterrupted until the planned cutover in Step 9.

```bash
# On node-1 — restart with your original single-sequencer flags
systemctl start ev-node
```

---

## Step 6: Collect Peer IDs

Before writing the configuration, collect the peer ID from each node. Peer IDs are needed to build the P2P peers list in multiaddr format.

```bash
# Run on each node
./evm net-info
```

Note the `peer_id` value from each node's output — it looks like `12D3KooW...`. You need all five before writing the configuration files.

---

## Step 7: Write the New Configuration on All Five Nodes

Write the following `evnode.yaml` on every node. The `node_id` is the only field that differs.

**Existing sequencer becomes node-1.** Assign `node-2` through `node-5` to the four new machines.

### node-1 (existing sequencer — `~/.evm/config/evnode.yaml`)

```yaml
node:
  aggregator: true
  block_time: "1s"   # keep your existing block_time

raft:
  enable: true
  node_id: "node-1"
  raft_addr: "0.0.0.0:5001"
  raft_dir: "/var/lib/ev-node/raft"
  peers: "node-2@10.0.0.2:5001,node-3@10.0.0.3:5001,node-4@10.0.0.4:5001,node-5@10.0.0.5:5001"

  # Timing — tuned for RTT_MAX ≤ 25ms
  heartbeat_timeout:    "92ms"
  election_timeout:     "368ms"
  leader_lease_timeout: "46ms"
  send_timeout:         "50ms"

  # Log retention
  trailing_logs:      18000
  snapshot_threshold: 5000
  snap_count:         3

p2p:
  listen_address: "/ip4/0.0.0.0/tcp/26656"
  peers: "/ip4/10.0.0.2/tcp/26656/p2p/<PEER_ID_NODE_2>,/ip4/10.0.0.3/tcp/26656/p2p/<PEER_ID_NODE_3>,/ip4/10.0.0.4/tcp/26656/p2p/<PEER_ID_NODE_4>,/ip4/10.0.0.5/tcp/26656/p2p/<PEER_ID_NODE_5>"
```

### node-2 through node-5

Identical except for `node_id`, raft and P2P peers (exclude self from the P2P list):

```yaml
# node-2
node:
  aggregator: true
  block_time: "1s"

raft:
  enable: true
  node_id: "node-2"      # change per node
  raft_addr: "0.0.0.0:5001"
  raft_dir: "/var/lib/ev-node/raft"
  peers: "node-1@10.0.0.1:5001,node-3@10.0.0.3:5001,node-4@10.0.0.4:5001,node-5@10.0.0.5:5001"
  heartbeat_timeout:    "92ms"
  election_timeout:     "368ms"
  leader_lease_timeout: "46ms"
  send_timeout:         "50ms"
  trailing_logs:      18000
  snapshot_threshold: 5000
  snap_count:         3

p2p:
  listen_address: "/ip4/0.0.0.0/tcp/26656"
  peers: "/ip4/10.0.0.1/tcp/26656/p2p/<PEER_ID_NODE_1>,/ip4/10.0.0.3/tcp/26656/p2p/<PEER_ID_NODE_3>,/ip4/10.0.0.4/tcp/26656/p2p/<PEER_ID_NODE_4>,/ip4/10.0.0.5/tcp/26656/p2p/<PEER_ID_NODE_5>"
```

---

## Step 8: Create Raft Data Directories and Passphrase File

Run on every node:

```bash
sudo mkdir -p /var/lib/ev-node/raft
sudo chown $(whoami) /var/lib/ev-node/raft

# Create the passphrase file — the binary reads this at startup
sudo mkdir -p /etc/ev-node
echo -n "<YOUR_PASSPHRASE>" | sudo tee /etc/ev-node/passphrase > /dev/null
sudo chmod 600 /etc/ev-node/passphrase
```

---

## Step 9: The Cutover

This is the planned maintenance window. The chain pauses block production from when you stop the existing sequencer until the new Raft cluster elects its first leader (a few seconds).

### 9a. Stop the existing single sequencer

```bash
# On node-1 (existing sequencer)
# Preferred: use systemd if the node runs as a service
sudo systemctl stop ev-node

# Fallback: stop the process directly
mapfile -t PIDS < <(pgrep -f "evm start")
if [ "${#PIDS[@]}" -ne 1 ]; then
  echo "Expected exactly 1 evm PID, found ${#PIDS[@]}: ${PIDS[*]}"
  exit 1
fi
echo "Stopping PID ${PIDS[0]}"
kill -SIGTERM "${PIDS[0]}"
```

Confirm it has stopped:

```bash
pgrep evm || echo "stopped"
```

### 9b. Start all five nodes simultaneously

The key requirement here is that all nodes must start within a short window of each other. Raft needs a majority (3 out of 5) online to elect a leader. If you start only 2 nodes and wait, the cluster will not elect a leader until the 3rd node joins.

Use a coordination mechanism — a simple approach is to open five terminals (or tmux panes) and fire the start commands in quick succession:

```bash
# On node-1
./evm start \
  --evnode.node.aggregator=true \
  --evnode.raft.enable=true \
  --evnode.raft.node_id="node-1" \
  --evnode.raft.raft_addr="0.0.0.0:5001" \
  --evnode.raft.raft_dir="/var/lib/ev-node/raft" \
  --evnode.raft.peers="node-2@10.0.0.2:5001,node-3@10.0.0.3:5001,node-4@10.0.0.4:5001,node-5@10.0.0.5:5001" \
  --evnode.raft.heartbeat_timeout="92ms" \
  --evnode.raft.election_timeout="368ms" \
  --evnode.raft.leader_lease_timeout="46ms" \
  --evnode.raft.send_timeout="50ms" \
  --evnode.raft.trailing_logs=18000 \
  --evnode.raft.snapshot_threshold=5000 \
  --evnode.p2p.listen_address="/ip4/0.0.0.0/tcp/26656" \
  --evnode.p2p.peers="/ip4/10.0.0.2/tcp/26656/p2p/<PEER_ID_NODE_2>,/ip4/10.0.0.3/tcp/26656/p2p/<PEER_ID_NODE_3>,/ip4/10.0.0.4/tcp/26656/p2p/<PEER_ID_NODE_4>,/ip4/10.0.0.5/tcp/26656/p2p/<PEER_ID_NODE_5>" \
  --evnode.signer.passphrase_file=/etc/ev-node/passphrase \
  --evm.jwt-secret=$(cat /path/to/jwt.hex) \
  --evm.genesis-hash=<YOUR_GENESIS_HASH>
```

```bash
# On node-2 (at the same time, or within a few seconds)
./evm start \
  --evnode.raft.node_id="node-2" \
  # ... same flags, change node_id and p2p.peers
```

Repeat for node-3, node-4, node-5.

---

## Step 10: Verify the Migration Succeeded

### Check leader election

Within seconds of starting, one node will win the election. Look for:

```text
INF raft: election won  tally=3  leader=node-1
INF raft: entering leader state
INF block produced  height=<N+1>
```

where `N` is the last block produced by the old single sequencer.

The followers will show:

```text
INF raft: entering follower state  leader=node-1
INF block applied from raft log  height=<N+1>
```

### Verify block height continuity

The new cluster must continue from exactly where the old sequencer left off. Query the EVM execution layer:

```bash
# From the existing sequencer's last known height (noted in step 5)
LAST_HEIGHT=<height-before-shutdown>

# Query node-1 (or any node) — replace 8545 with your EVM RPC port
NEW_HEIGHT=$(cast block latest --rpc-url http://10.0.0.1:8545 | jq -r '.number')

echo "Last old height: $LAST_HEIGHT"
echo "New cluster height: $NEW_HEIGHT"

# New height should be LAST_HEIGHT + 1 (or a few blocks ahead if it took a moment)
```

### Check all nodes are synced

```bash
for ip in 10.0.0.1 10.0.0.2 10.0.0.3 10.0.0.4 10.0.0.5; do
  echo -n "$ip: height="
  cast block latest --rpc-url http://$ip:8545 | jq -r '.number'
done
```

All nodes should be at the same height (within 1–2 blocks of each other).

---

## Step 11: Set Up Systemd on All Nodes

Once you have confirmed the cluster is healthy, set up systemd for automatic restarts and service management. See the [cluster setup guide](./cluster-setup.md#running-as-a-systemd-service) for a ready-to-use unit file template.

---

## Rollback Plan

If anything goes wrong during the cutover, you can revert to the single sequencer:

1. Stop all five nodes.
2. Wipe the Raft data directories (`/var/lib/ev-node/raft`) on all nodes to clear any bootstrapped cluster state.
3. Remove the Raft configuration from node-1's `evnode.yaml` (or revert to the pre-migration config file).
4. Start node-1 with `raft.enable: false` — it resumes as a single sequencer from the block height it was at when you stopped it.

```bash
# Emergency rollback — revert node-1 to single sequencer
./evm start \
  --evnode.node.aggregator=true \
  --evnode.raft.enable=false \
  --evnode.signer.passphrase_file=/etc/ev-node/passphrase \
  # ... your original flags
```

The chain continues from the last block committed before the cutover. No blocks are lost because the single sequencer's data was never modified.

---

## New Nodes Without Existing Chain Data

If you did not copy the block data in step 5 (or if you are adding nodes long after the chain started), the new nodes will sync historical block data via P2P and the DA layer after joining the cluster. This process runs in the background and does not prevent the cluster from electing a leader or producing new blocks.

Monitor sync progress on a new node:

```bash
# The node will log progress as it fetches historical blocks
journalctl -u ev-node -f | grep "sync\|height"
```

---

## Troubleshooting

### Node-1 starts but no leader is elected

The cluster cannot elect a leader without a quorum (3 out of 5 nodes). Ensure all five nodes are running and can reach each other on port 5001.

### New nodes report height mismatch or divergence panic

This happens if the block data on the new nodes was copied from a different snapshot than the final state of the old sequencer, or if the copy was done while the old sequencer was still running and produced additional blocks during the copy.

Wipe the new nodes' block data and Raft directories, re-copy from the stopped node-1, and retry.

### Block height jumps backward or chain forks

This should not happen if all five nodes are running the same binary version and have the same genesis file and signer key. If you see it:

1. Stop all nodes immediately.
2. Identify which node produced the offending block.
3. Check that its genesis hash and signer key match the other nodes.

### Old single sequencer comes back online accidentally

If the old sequencer (without Raft) is started again after the new cluster is already producing blocks, it will attempt to produce blocks independently, creating a fork. This is why it is important to disable or remove the old single-sequencer startup scripts immediately after the cutover.

With Raft enabled on all five nodes, only the elected leader will produce blocks — there is no risk of a sixth "rogue" leader as long as the old machine is not restarted with the old non-Raft configuration.
