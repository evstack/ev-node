# Bootstrap a 5-Node HA Cluster from Scratch

This tutorial walks you through setting up a production-ready 5-node ev-node Raft cluster from zero. By the end you will have five sequencer nodes that automatically elect a leader, replicate block state, and survive individual node failures.

## Prerequisites

- Five machines (VMs, bare metal, or containers) with:
  - Network connectivity to each other on the Raft port (we use `5001`) and P2P port (`26656`)
  - Persistent storage for the Raft data directory
  - A working ev-node binary (see the [quickstart guide](../quick-start.md))
- A private network, VPN, or encrypted mesh between all nodes (Raft transport is plain TCP — never expose the Raft port publicly)
- A shared genesis file for your chain (see [Create Genesis](../create-genesis.md))
- A signer key on each node (all nodes must share the same signing identity so block hashes match regardless of which node is the current leader)

### Node addresses used in this guide

| Node | Private IP | Raft address | P2P port |
|------|------------|--------------|----------|
| node-1 | 10.0.0.1 | 10.0.0.1:5001 | 26656 |
| node-2 | 10.0.0.2 | 10.0.0.2:5001 | 26656 |
| node-3 | 10.0.0.3 | 10.0.0.3:5001 | 26656 |
| node-4 | 10.0.0.4 | 10.0.0.4:5001 | 26656 |
| node-5 | 10.0.0.5 | 10.0.0.5:5001 | 26656 |

Replace these with your actual IP addresses throughout the guide.

P2P peers use the libp2p multiaddr format, which includes each node's peer ID:

```text
/ip4/<ip>/tcp/<port>/p2p/<peer-id>
```

You will collect peer IDs in Step 3 after initializing each node.

---

## Step 1: Measure Network RTT

The Raft timing parameters must be sized for your network. Run the following from each node to every other node and note the highest average RTT you observe:

```bash
# Example: from node-1, ping all peers
for ip in 10.0.0.2 10.0.0.3 10.0.0.4 10.0.0.5; do
  echo -n "$ip: "
  ping -c 20 $ip | tail -1 | awk -F'/' '{print $5 "ms avg"}'
done
```

Repeat from each node. Take the single highest value across all measurements — this is your `RTT_MAX`.

For nodes within the same region or data center, `RTT_MAX` is typically 5–30ms. For the configuration file below we assume `RTT_MAX ≤ 25ms`. If your measurement is higher, adjust the timing parameters using the formulas in the [configuration reference](./overview.md#timing-parameters).

---

## Step 2: Verify Network Connectivity

Confirm that the Raft port and P2P port are reachable between nodes before starting anything:

```bash
# From node-2, verify node-1's Raft port is reachable
nc -zv 10.0.0.1 5001

# From node-2, verify node-1's P2P port is reachable
nc -zv 10.0.0.1 26656
```

Do this for every node pair in both directions. If any check fails, fix your firewall rules before proceeding.

---

## Step 3: Initialize Each Node

Run this on every node. Each node gets its own home directory where the config, keys, and data live.

First, create a passphrase file that only root and the service account can read. This file is referenced by the binary at runtime — the passphrase never appears in process listings or logs.

```bash
# Run on every node
sudo mkdir -p /etc/ev-node
echo -n "<YOUR_PASSPHRASE>" | sudo tee /etc/ev-node/passphrase > /dev/null
sudo chmod 600 /etc/ev-node/passphrase
sudo chown ev-node:ev-node /etc/ev-node/passphrase
```

Then initialize the node:

```bash
# Run on every node (the binary name depends on your chain)
./evm init --evnode.node.aggregator=true --evnode.signer.passphrase_file /etc/ev-node/passphrase
```

This creates the home directory structure (default `~/.evm`) with a `config/evnode.yaml` file and generates the signer key.

After initializing each node, retrieve its peer ID — you will need all five when writing the configuration in Step 5:

```bash
# Run on each node after init
./evm net-info
```

Note the `peer_id` value from each node's output. It looks like `12D3KooW...`. You will need all five peer IDs before writing the configuration files.

> **Shared signer key:** All cluster nodes must sign blocks with the same key so that block hashes produced by any leader are identical. Copy the key material from node-1 to all other nodes after initialization:
>
> ```bash
> # On node-1: locate the signer key
> ls ~/.evm/config/
>
> # Secure-copy it to each peer
> scp ~/.evm/config/signer.json user@10.0.0.2:~/.evm/config/
> scp ~/.evm/config/signer.json user@10.0.0.3:~/.evm/config/
> scp ~/.evm/config/signer.json user@10.0.0.4:~/.evm/config/
> scp ~/.evm/config/signer.json user@10.0.0.5:~/.evm/config/
> ```

---

## Step 4: Distribute the Genesis File

Every node must start with the same genesis file. Create it on node-1 (see [Create Genesis](../create-genesis.md)) then copy it to all peers:

```bash
scp ~/.evm/config/genesis.json user@10.0.0.2:~/.evm/config/
scp ~/.evm/config/genesis.json user@10.0.0.3:~/.evm/config/
scp ~/.evm/config/genesis.json user@10.0.0.4:~/.evm/config/
scp ~/.evm/config/genesis.json user@10.0.0.5:~/.evm/config/
```

---

## Step 5: Write the Configuration Files

Write the following `evnode.yaml` on each node. `raft.node_id` is unique per node; `raft.peers` and `p2p.peers` must each exclude the local node — everything else is identical.

### node-1 (`~/.evm/config/evnode.yaml`)

```yaml
node:
  aggregator: true
  block_time: "1s"

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

  # Log retention — covers ~5 hours of absence at 1 block/s
  trailing_logs:      18000
  snapshot_threshold: 5000
  snap_count:         3

p2p:
  listen_address: "/ip4/0.0.0.0/tcp/26656"
  peers: "/ip4/10.0.0.2/tcp/26656/p2p/<PEER_ID_NODE_2>,/ip4/10.0.0.3/tcp/26656/p2p/<PEER_ID_NODE_3>,/ip4/10.0.0.4/tcp/26656/p2p/<PEER_ID_NODE_4>,/ip4/10.0.0.5/tcp/26656/p2p/<PEER_ID_NODE_5>"
```

### node-2 (`~/.evm/config/evnode.yaml`)

`raft.peers` must omit the local node. Because `raft_addr` is `0.0.0.0:5001` (a wildcard), the self-exclusion check in the bootstrap code compares addresses literally — it will not recognise `node-2@10.0.0.2:5001` as itself and will add node-2 twice, causing a startup error. Always list only the **other** nodes.

```yaml
node:
  aggregator: true
  block_time: "1s"

raft:
  enable: true
  node_id: "node-2"
  raft_addr: "0.0.0.0:5001"
  raft_dir: "/var/lib/ev-node/raft"
  peers: "node-1@10.0.0.1:5001,node-3@10.0.0.3:5001,node-4@10.0.0.4:5001,node-5@10.0.0.5:5001"

  # Timing — tuned for RTT_MAX ≤ 25ms
  heartbeat_timeout:    "92ms"
  election_timeout:     "368ms"
  leader_lease_timeout: "46ms"
  send_timeout:         "50ms"

  # Log retention — covers ~5 hours of absence at 1 block/s
  trailing_logs:      18000
  snapshot_threshold: 5000
  snap_count:         3

p2p:
  listen_address: "/ip4/0.0.0.0/tcp/26656"
  peers: "/ip4/10.0.0.1/tcp/26656/p2p/<PEER_ID_NODE_1>,/ip4/10.0.0.3/tcp/26656/p2p/<PEER_ID_NODE_3>,/ip4/10.0.0.4/tcp/26656/p2p/<PEER_ID_NODE_4>,/ip4/10.0.0.5/tcp/26656/p2p/<PEER_ID_NODE_5>"
```

Repeat for node-3 through node-5, updating `node_id`, `raft.peers` (exclude the local node), and `p2p.peers` (exclude the local node).

---

## Step 6: Create the Raft Data Directories

```bash
# Run on every node
sudo mkdir -p /var/lib/ev-node/raft
sudo chown $(whoami) /var/lib/ev-node/raft
```

For Docker deployments, this is handled by the named volume — skip this step.

---

## Step 7: Start All Nodes Simultaneously

Raft requires a majority of configured peers to be online before it can elect a leader. For a 5-node cluster, you need at least 3 nodes to be up before a leader can be elected and blocks can be produced.

Start all five nodes as close together as possible. The order does not matter but they should all be up within a few seconds of each other.

```bash
# Run this on each node, substituting the correct binary name and flags for your chain
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
  --evnode.raft.snap_count=3 \
  --evnode.p2p.listen_address="/ip4/0.0.0.0/tcp/26656" \
  --evnode.p2p.peers="/ip4/10.0.0.2/tcp/26656/p2p/<PEER_ID_NODE_2>,/ip4/10.0.0.3/tcp/26656/p2p/<PEER_ID_NODE_3>,/ip4/10.0.0.4/tcp/26656/p2p/<PEER_ID_NODE_4>,/ip4/10.0.0.5/tcp/26656/p2p/<PEER_ID_NODE_5>" \
  --evnode.signer.passphrase_file=/etc/ev-node/passphrase \
  --evm.jwt-secret=$(cat /path/to/jwt.hex) \
  --evm.genesis-hash=<YOUR_GENESIS_HASH>
```

Adjust flags for your execution layer (e.g., remove EVM flags if you are running a Cosmos SDK chain).

---

## Step 8: Verify the Cluster Is Healthy

### Watch the logs

Within a few seconds of starting, you should see one node win the election:

```text
INF raft: entering candidate state  node=node-1
INF raft: election won               tally=3
INF raft: entering leader state      leader=node-1
INF block produced                   height=1 hash=0xabc...
```

The other nodes will log:

```text
INF raft: entering follower state  leader=node-1
INF block applied from raft log    height=1 hash=0xabc...
```

### Check node health

Verify each node's HTTP API is responding:

```bash
# ev-node exposes its health endpoint on port 7331 by default
curl http://10.0.0.1:7331/health/ready

# Check which node is the current leader
curl http://10.0.0.1:7331/raft/node | jq '{node_id, is_leader}'
```

### Check block production

For EVM chains, query the execution layer to confirm blocks are being produced:

```bash
# Run from any node; the height should increase each time
cast block latest --rpc-url http://10.0.0.1:8545
```

Repeat after a few seconds — the block number should be increasing.

### Verify all nodes are synced

Query each node; all five should report the same block height (or within 1–2 blocks of each other):

```bash
for ip in 10.0.0.1 10.0.0.2 10.0.0.3 10.0.0.4 10.0.0.5; do
  echo -n "$ip: height="
  cast block latest --rpc-url http://$ip:8545 | jq -r '.number'
done
```

---

## Step 9: Test Failover

With all five nodes running and producing blocks, simulate a leader failure:

```bash
# Identify the current leader from its logs, then on that machine.
# Preferred: use the systemd unit if ev-node runs as a service
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

Within `election_timeout` (368ms in this configuration), the remaining four nodes will elect a new leader and resume block production. Measure the actual gap in your logs:

```bash
# Look for the last block before the kill and first block after
grep "block produced\|block applied" /var/log/ev-node/node-2.log | tail -20
```

The gap should be well under 1 second in most cases (a few election cycles at most).

---

## Running as a Systemd Service

For production, manage each node with systemd.

### Create the passphrase file

If you did not create the passphrase file in Step 3, do it now. The file must exist on every node before the service starts:

```bash
# Run on every node (skip if you already did this in Step 3)
sudo mkdir -p /etc/ev-node
echo -n "<YOUR_PASSPHRASE>" | sudo tee /etc/ev-node/passphrase > /dev/null
sudo chmod 600 /etc/ev-node/passphrase
sudo chown ev-node:ev-node /etc/ev-node/passphrase
```

### Unit file

```ini
# /etc/systemd/system/ev-node.service
[Unit]
Description=ev-node HA sequencer
After=network-online.target
Wants=network-online.target

[Service]
User=ev-node
ExecStart=/usr/local/bin/evm start \
  --evnode.node.aggregator=true \
  --evnode.raft.enable=true \
  --evnode.raft.node_id=node-1 \
  --evnode.raft.raft_addr=0.0.0.0:5001 \
  --evnode.raft.raft_dir=/var/lib/ev-node/raft \
  --evnode.raft.peers=node-2@10.0.0.2:5001,node-3@10.0.0.3:5001,node-4@10.0.0.4:5001,node-5@10.0.0.5:5001 \
  --evnode.raft.heartbeat_timeout=92ms \
  --evnode.raft.election_timeout=368ms \
  --evnode.raft.leader_lease_timeout=46ms \
  --evnode.raft.send_timeout=50ms \
  --evnode.raft.trailing_logs=18000 \
  --evnode.raft.snapshot_threshold=5000 \
  --evnode.p2p.listen_address=/ip4/0.0.0.0/tcp/26656 \
  --evnode.p2p.peers=/ip4/10.0.0.2/tcp/26656/p2p/<PEER_ID_NODE_2>,/ip4/10.0.0.3/tcp/26656/p2p/<PEER_ID_NODE_3>,/ip4/10.0.0.4/tcp/26656/p2p/<PEER_ID_NODE_4>,/ip4/10.0.0.5/tcp/26656/p2p/<PEER_ID_NODE_5> \
  --evnode.signer.passphrase_file=/etc/ev-node/passphrase
Restart=on-failure
RestartSec=5s

# Give the process time to transfer leadership before systemd kills it
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable ev-node
sudo systemctl start ev-node
sudo journalctl -u ev-node -f
```

`TimeoutStopSec=30` gives the node enough time to perform a graceful leadership transfer on `SIGTERM` before systemd sends `SIGKILL`. Do not set this too short.

---

## Performing a Rolling Restart

To restart nodes without taking the cluster offline (e.g., for a config change or binary upgrade):

1. Restart one non-leader node at a time and wait for it to rejoin before touching the next.
2. For the leader node, restart it last. `ev-node` will transfer leadership to a peer before shutting down.

```bash
# Restart non-leader nodes first, one at a time.
# After each restart, wait until the node confirms it has rejoined before touching the next.

ssh user@10.0.0.2 "sudo systemctl restart ev-node"
ssh user@10.0.0.2 "sudo journalctl -u ev-node --since '1 min ago' -f | grep -m1 'follower state\|leader state'"

ssh user@10.0.0.3 "sudo systemctl restart ev-node"
ssh user@10.0.0.3 "sudo journalctl -u ev-node --since '1 min ago' -f | grep -m1 'follower state\|leader state'"

ssh user@10.0.0.4 "sudo systemctl restart ev-node"
ssh user@10.0.0.4 "sudo journalctl -u ev-node --since '1 min ago' -f | grep -m1 'follower state\|leader state'"

ssh user@10.0.0.5 "sudo systemctl restart ev-node"
ssh user@10.0.0.5 "sudo journalctl -u ev-node --since '1 min ago' -f | grep -m1 'follower state\|leader state'"

# Restart the leader last — ev-node transfers leadership before shutting down
ssh user@10.0.0.1 "sudo systemctl restart ev-node"
ssh user@10.0.0.1 "sudo journalctl -u ev-node --since '1 min ago' -f | grep -m1 'follower state\|leader state'"
```

The `grep -m1` exits as soon as the node logs `entering follower state` or `entering leader state`, confirming it has rejoined the cluster. Only then proceed to the next node.

---

## Troubleshooting

### Cluster does not elect a leader

Check that:
- At least 3 out of 5 nodes are running and can reach each other on port 5001.
- The `peers` list on every node is identical and all addresses are correct.
- No firewall rule is blocking TCP on port 5001.

```bash
# Quick connectivity check from node-2 to node-1 Raft port
nc -zv 10.0.0.1 5001
```

### Node panics on startup with "state divergence"

This means the node's local block store is ahead of or behind the Raft consensus state in a way that cannot be reconciled automatically. This typically happens when a node's `raft_dir` was wiped but the block database was not (or vice versa).

Stop the node, wipe both `raft_dir` and the node's block data directory, then restart. The node will receive a Raft snapshot and rebuild from there.

### Spurious elections / leadership flapping

Symptoms: frequent `election won` and `entering follower state` lines in the logs, block production pausing briefly every few seconds.

Causes:
- `heartbeat_timeout` is too short for your network RTT — increase it.
- Network congestion or packet loss between nodes.
- Node CPU is saturated and cannot process heartbeats in time.

As a quick diagnostic, check the RTT between nodes while the cluster is running:

```bash
ping -c 100 10.0.0.2 | tail -5
```

If the max RTT is close to or above `heartbeat_timeout`, increase `heartbeat_timeout` and `election_timeout` proportionally.
