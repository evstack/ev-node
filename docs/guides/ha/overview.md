# High Availability Sequencer

ev-node supports running your sequencer in a **High Availability (HA)** cluster using the [Raft consensus algorithm](https://raft.github.io/). Instead of a single aggregator node that is a point of failure, multiple nodes form a cluster that automatically elects a leader and recovers from individual node failures without manual intervention and without halting block production.

## Why Raft HA

A single sequencer node means that if the machine crashes, loses power, or needs maintenance, your chain stops producing blocks until the node is back online. With a Raft cluster:

- **Automatic failover** — when the active leader fails, remaining nodes elect a new leader within seconds.
- **No double-signing** — the Raft log guarantees at most one leader at a time and synchronizes block state across all nodes before any block is committed.
- **Graceful restarts** — before shutting down, the leader transfers leadership to a healthy peer so downtime is measured in milliseconds.
- **Fault tolerance** — a 5-node cluster keeps producing blocks as long as at least 3 nodes are reachable; it can absorb 2 simultaneous failures.

## How It Works

Each node in the cluster runs `ev-node` in aggregator mode with Raft enabled. The nodes communicate over a private TCP transport to:

1. **Elect a leader** — using Raft leader election. Only the elected leader produces blocks.
2. **Replicate state** — every block the leader produces is appended to the Raft log and replicated to all followers before it is considered committed.
3. **Apply to FSM** — each node applies committed log entries to its Finite State Machine (FSM), which tracks the latest committed block height, hash, and timestamp.
4. **Detect failure** — followers watch for heartbeats from the leader. If heartbeats stop arriving within the election timeout, a follower starts a new election.
5. **Catch up** — a node that was offline rejoins by receiving a Raft snapshot (fast-forward to the current head) and then fetching any missing historical blocks from peers via P2P.

### Storage

Raft state is stored in the directory specified by `raft.raft_dir`:

| File | Purpose |
|------|---------|
| `raft-log.db` | Raft log entries (BoltDB) |
| `raft-stable.db` | Current term and vote state (BoltDB) |
| `*.snp` | Snapshots of the FSM state |

These files represent the node's **cluster identity**. They must live on persistent storage — loss of this directory is equivalent to removing the node from the cluster.

## Cluster Sizing

Always run an **odd number** of nodes. Raft requires a majority (quorum) to elect a leader and commit entries.

| Nodes | Quorum | Tolerated failures |
|-------|--------|--------------------|
| 3 | 2 | 1 |
| **5** | **3** | **2** |
| 7 | 4 | 3 |

**5 nodes is the recommended production configuration.** It tolerates two simultaneous node failures — enough to absorb a rolling upgrade plus an unexpected crash at the same time — while keeping the cluster size manageable.

## Network Requirements

Raft transport is **plain TCP** with no built-in encryption. Before deploying:

- Run all nodes inside a **private network, VPN, or encrypted mesh** (WireGuard, Tailscale, AWS VPC, etc.).
- **Never expose the Raft port to the public internet.** An attacker with access to the Raft port can send forged messages that disrupt or hijack cluster consensus.
- Ensure low-latency connectivity between nodes. Timeouts must be sized larger than the worst-case round-trip time (RTT) between any two nodes in the cluster.

### Node Placement

**Run all nodes in the same region, spread across different availability zones.**

This is the single most important infrastructure decision for cluster stability. All nodes must have roughly the same RTT to each other. The timing parameters (heartbeat timeout, election timeout) are sized for a single `RTT_MAX` value — if one node has materially higher latency than its peers, it degrades the entire cluster's ability to detect failures and elect leaders reliably.

Specifically:
- **Same region, different AZs** gives uniform 5–30ms RTT and is the validated production topology. Nodes are isolated from AZ-level failures while keeping latency uniform.
- **Cross-region nodes** introduce higher and asymmetric RTT (100ms+). Even a single high-latency node can destabilize the cluster under network stress.

This was observed directly in load testing: a 3-node cluster where one node averaged 99ms RTT (2× higher than its peers at 45–49ms) showed election times up to 284 seconds, three undetected leader elections, and one skipped cycle when 200–500ms of additional latency was injected — the same disruption level where the two lower-latency nodes recovered in under 55 seconds. Moving to a 5-node cluster with uniform ~45ms RTT across all nodes eliminated all undetected elections, reduced the worst-case election time from 284s to 66s, and reduced cascade risk from 10% of cycles to 3%.

If your deployment requires nodes in different regions, increase `heartbeat_timeout` and `election_timeout` to at least 4–5× the worst-case inter-node RTT, and expect slower failover. See the [timing parameters](#timing-parameters) section for tuning formulas.

---

## Configuration Reference

Raft is configured under the `raft` section of `evnode.yaml`, or via `--evnode.raft.*` CLI flags.

### Required Parameters

These must be set on every node for the cluster to form.

#### `raft.enable`

```yaml
raft:
  enable: true
```

**CLI:** `--evnode.raft.enable`  
**Default:** `false`

Enables Raft consensus. Must be `true` on every cluster member. When disabled (the default), the node runs as a traditional single sequencer. Setting this to `true` also requires `node.aggregator: true`.

---

#### `raft.node_id`

```yaml
raft:
  node_id: "node-1"
```

**CLI:** `--evnode.raft.node_id`  
**Default:** _(none, required)_

A string that uniquely identifies this node within the cluster. Every node must have a different `node_id`. The ID is stored in the Raft log and used by other nodes to route messages — **never change it after the cluster is bootstrapped**, as doing so will break the cluster membership records.

Convention: use stable, descriptive names like `node-1`, `node-2`, … `node-5` or names tied to the host (`sequencer-us-east-1`, `sequencer-eu-east-2`).

---

#### `raft.raft_addr`

```yaml
raft:
  raft_addr: "0.0.0.0:5001"
```

**CLI:** `--evnode.raft.raft_addr`  
**Default:** _(none, required)_

The TCP address this node listens on for Raft transport messages from other cluster members. The `0.0.0.0` bind address accepts connections on all interfaces; bind to a specific private IP if you want to restrict which interface is used for cluster traffic.

The port (here `5001`) must be reachable from every other node in the cluster.

The address you advertise in `raft.peers` must resolve to this port from the perspective of other nodes. If you bind to `0.0.0.0` internally, advertise the node's actual private IP in the peers list.

---

#### `raft.raft_dir`

```yaml
raft:
  raft_dir: "/var/lib/ev-node/raft"
```

**CLI:** `--evnode.raft.raft_dir`  
**Default:** `<home>/raft`

The directory where Raft stores its persistent state: log database, stable store, and snapshots. This directory **must be on persistent storage** (not tmpfs, not ephemeral container storage). Losing this directory means the node loses its cluster identity — it cannot rejoin without being reconfigured as a new member.

For Docker deployments, mount this as a named volume. For bare-metal or systemd services, ensure the directory survives reboots.

---

#### `raft.peers`

```yaml
raft:
  peers: "node-2@10.0.0.2:5001,node-3@10.0.0.3:5001,node-4@10.0.0.4:5001,node-5@10.0.0.5:5001"
```

**CLI:** `--evnode.raft.peers`  
**Default:** _(none, required)_

A comma-separated list of the **other** cluster members (exclude the local node), in the format `nodeID@host:port`. The host and port must be the Raft address (`raft_addr`) of each peer as reachable from this node. Do not list the node's own `node_id` in its own `peers` field.

Raft uses this list to:
- Bootstrap the cluster on first start (when no persisted state exists).
- Know which addresses to dial when sending log entries or heartbeats.

> **Limitation — static membership only.** Changing the peer set at runtime (adding or removing nodes without a full cluster restart) is not currently supported. All nodes that will ever participate in the cluster must be listed in `peers` before the cluster is first bootstrapped.

---

#### `raft.bootstrap`

```yaml
raft:
  bootstrap: false
```

**CLI:** `--evnode.raft.bootstrap`  
**Default:** `false`

Compatibility flag retained for older deployments. **You do not need to set this.** ev-node auto-detects the correct startup mode from the state of `raft_dir`:

- If `raft_dir` contains existing Raft state → the node **rejoins** the cluster automatically.
- If `raft_dir` is empty or does not exist → the node **bootstraps** a new cluster from the `peers` list.

Setting `bootstrap: true` explicitly has no additional effect beyond what auto-detection already does.

---

### Timing Parameters

These parameters control how quickly the cluster detects failures and elects a new leader. They must be sized relative to the **maximum round-trip time (RTT) between any two nodes** in the cluster. Too tight and the cluster experiences spurious leader changes; too loose and failover takes longer than necessary.

**To measure your network RTT:**

```bash
# Run from each node to every other node; note the maximum result
ping -c 20 <peer-ip> | tail -1
```

Take the maximum average RTT across all pairs — this is your `RTT_MAX`.

#### `raft.heartbeat_timeout`

```yaml
raft:
  heartbeat_timeout: "92ms"
```

**CLI:** `--evnode.raft.heartbeat_timeout`  
**Default:** `350ms`

The maximum time a follower will wait without receiving a heartbeat from the leader before starting a new election. The leader sends heartbeats more frequently than this value internally; this parameter is purely a follower-side timeout that triggers a new election when crossed.

**Tuning rule:** Set to **4–5× RTT_MAX**. This ensures followers can distinguish a slow network from a dead leader without triggering spurious elections.

- Too low (< 2× RTT_MAX): followers time out due to normal network jitter and start unnecessary elections, causing leadership flapping and brief block production pauses.
- Too high: failover takes longer; the cluster is slower to react to a leader crash.

| RTT_MAX | Recommended heartbeat_timeout |
|---------|-------------------------------|
| 10ms | 40–50ms |
| 23ms | 92ms |
| 50ms | 200–250ms |
| 100ms | 400–500ms |

---

#### `raft.election_timeout`

```yaml
raft:
  election_timeout: "368ms"
```

**CLI:** `--evnode.raft.election_timeout`  
**Default:** `1000ms`

How long a follower waits without receiving a heartbeat before it concludes the leader is dead and starts a new election. Must be greater than or equal to `heartbeat_timeout`.

**Tuning rule:** Set to **4× heartbeat_timeout** (or approximately 16–20× RTT_MAX). The factor of 4 gives the leader several missed heartbeat opportunities before a follower acts — enough to ride out transient packet loss without triggering unnecessary elections.

A larger election timeout means a slower reaction to leader failure (failover takes longer). A smaller election timeout risks false positives: the cluster starts an election while the leader is merely experiencing a brief network delay, causing a term increment and a short pause in block production.

---

#### `raft.leader_lease_timeout`

```yaml
raft:
  leader_lease_timeout: "46ms"
```

**CLI:** `--evnode.raft.leader_lease_timeout`  
**Default:** `175ms`

The duration for which a leader considers its leadership valid after the last successful heartbeat acknowledgment. Leader lease enables local reads from the leader without a round-trip to quorum.

**Tuning rule:** Set to approximately **half of `heartbeat_timeout`** (i.e., ~2× RTT_MAX), and always **strictly less than `election_timeout`**. If `leader_lease_timeout` is close to or exceeds `election_timeout`, a node may believe it is still the leader after followers have already elected a replacement, which can cause split-brain reads.

---

#### `raft.send_timeout`

```yaml
raft:
  send_timeout: "50ms"
```

**CLI:** `--evnode.raft.send_timeout`  
**Default:** `200ms`

The maximum time the leader waits for a single message (log entry, heartbeat) to be delivered to a peer before marking the delivery as failed. A failed send is retried, but repeated failures trigger follower health tracking.

**Tuning rule:** Set to **2–3× RTT_MAX**. This allows for normal network latency plus one retransmission before giving up on a delivery attempt.

---

### Snapshot and Log Retention Parameters

These parameters control how frequently Raft snapshots the FSM state and how many log entries are kept around after a snapshot. They affect both disk usage and how quickly a lagging node can catch up.

#### `raft.snapshot_threshold`

```yaml
raft:
  snapshot_threshold: 5000
```

**CLI:** `--evnode.raft.snapshot_threshold`  
**Default:** `500`

The number of committed log entries that must accumulate before Raft automatically takes a snapshot of the FSM state. After a snapshot, log entries older than the snapshot are compacted away.

**Effect on operations:**
- **Lower values** (e.g., `500`): snapshots are taken frequently, keeping the log small. A restarting node receives a recent snapshot and has fewer log entries to replay, but snapshot writes happen more often, adding brief I/O bursts.
- **Higher values** (e.g., `5000`): less frequent snapshots mean less I/O overhead during normal operation, but a lagging node may have more log entries to replay when catching up.

At 10 block/second, `snapshot_threshold: 5000` takes a snapshot roughly every 8.3 minutes (500 seconds).

---

#### `raft.trailing_logs`

```yaml
raft:
  trailing_logs: 18000
```

**CLI:** `--evnode.raft.trailing_logs`  
**Default:** `200`

The number of log entries to **retain after a snapshot** is taken. These entries act as a catch-up buffer: a node that missed fewer than `trailing_logs` entries since the last snapshot can replay from the log without needing to transfer the full snapshot.

**Effect on operations:**
- **Lower values** (e.g., `200`): tighter disk usage; a node that misses even a few minutes of operation must receive a full snapshot on rejoin.
- **Higher values** (e.g., `18000`): a lagging node can catch up via log replay without needing a full snapshot transfer, reducing the cost of brief outages. At 1 block/second (`block_time: "1s"`), `trailing_logs: 18000` covers ~5 hours; at 10 block/second, ~30 minutes.

Set this high enough to cover your typical maintenance window (restart, upgrade, brief network partition). Scale proportionally with your chain's block rate.

---

#### `raft.snap_count`

```yaml
raft:
  snap_count: 3
```

**CLI:** `--evnode.raft.snap_count`  
**Default:** `3`

The number of snapshot files to retain on disk. Older snapshots are deleted when new ones are created. Keeping 2–3 snapshots provides a rollback option in case the latest snapshot is corrupt.

---

### Recommended Production Configuration

The following configuration is recommended for a **5-node cluster on a network with RTT_MAX ≤ 25ms** (typical for nodes in the same region). It was validated by an extensive sweep of 10 configurations across 150 SIGTERM kill cycles and 50 latency-injection cycles, with zero undetected failures and zero split-brain events recorded.

```yaml
# evnode.yaml — paste this raft section into every node's config
# Replace node_id, raft_addr, and peers with your actual values.

node:
  aggregator: true

raft:
  enable: true
  node_id: "node-1"                    # unique per node
  raft_addr: "0.0.0.0:5001"
  raft_dir: "/var/lib/ev-node/raft"    # must be persistent

  # Remote peers list — different on every node
  peers: >-
    node-2@10.0.0.2:5001,
    node-3@10.0.0.3:5001,
    node-4@10.0.0.4:5001,
    node-5@10.0.0.5:5001

  # Timing — tuned for RTT_MAX ≤ 25ms
  heartbeat_timeout:    "92ms"
  election_timeout:     "368ms"
  leader_lease_timeout: "46ms"
  send_timeout:         "50ms"

  # Log retention
  trailing_logs:      18000
  snapshot_threshold: 5000
  snap_count:         3
```

**Adapting for different RTT values:**

Measure RTT_MAX first and scale the timing parameters:

```text
heartbeat_timeout    = RTT_MAX × 4
election_timeout     = heartbeat_timeout × 4
leader_lease_timeout = heartbeat_timeout / 2
send_timeout         = RTT_MAX × 3
```

---

## Interaction with P2P

Even in a Raft cluster, each node must have P2P configured. Raft handles **hot replication** — it replicates the latest block state to all followers in near real-time. But if a node falls far enough behind that the missing entries have already been compacted out of the Raft log (i.e., it missed more entries than `trailing_logs`), it receives a Raft snapshot to jump to the current head. Historical blocks between the node's last known state and the snapshot are then fetched via the **P2P network or DA layer**.

```yaml
p2p:
  listen_address: "/ip4/0.0.0.0/tcp/26656"
  peers: "/ip4/<PEER_IP>/tcp/26656/p2p/<PEER_ID>,..."
```

Ensure P2P ports are open between nodes in addition to the Raft port.

---

## Monitoring

Track these metrics (available via Prometheus if `metrics.enabled: true`) to catch problems early:

| Signal | What it means |
|--------|---------------|
| Frequent leadership changes | Network instability, asymmetric packet loss, or overloaded nodes |
| Growing applied-index lag | FSM cannot keep up with commits; check CPU and disk I/O |
| Snapshot transfers | Node fell behind `trailing_logs` entries — check network and disk |
| Election timeouts | Heartbeats are being dropped; check MTU, firewall rules, network congestion |

See the [Monitoring guide](../operations/monitoring.md) for the full Prometheus metric list.
