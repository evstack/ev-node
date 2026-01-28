# Raft Implementation & Production Configuration

This guide details the Raft consensus implementation in `ev-node`, used for High Availability (HA) of the Sequencer/Aggregator. It is targeted at experienced DevOps and developers configuring production environments.

## Overview

`ev-node` uses the [HashiCorp Raft](https://github.com/hashicorp/raft) implementation to manage leader election and state replication when running in **Aggregator Mode**. 

*   **Role**: Ensures only one active Aggregator (Leader) produces blocks at a time.
*   **Failover**: Automatically elects a new leader if the current leader fails.
*   **Safety**: Synchronizes the block production state to prevent double-signing or fork divergence.

### Architecture

*   **Transport**: TCP-based transport for inter-node communication.
*   **Storage**: [BoltDB](https://github.com/etcd-io/bbolt) is used for both the Raft Log (`raft-log.db`) and Stable Store (`raft-stable.db`). Snapshots are stored as files.
*   **FSM (Finite State Machine)**: The State Machine applies `RaftBlockState` (Protobuf) containing the latest block height, hash, and timestamp.
*   **Safety Checks**:
    *   **Startup**: Nodes check for divergence between local block store and Raft state.
    *   **Leadership Transfer**: Before becoming leader, a node waits for its FSM to catch up (`waitForMsgsLanded`) to prevent proposing blocks from a stale state.
    *   **Shutdown**: The leader attempts to transfer leadership gracefully before shutting down to minimize downtime.

## Configuration

Raft is configured via CLI flags or the `config.toml` file under the `[raft]` (or `[rollkit.raft]`) section.

### Essential Flags

| Flag | Config Key | Description | Production Value |
|------|------------|-------------|------------------|
| `--evnode.raft.enable` | `raft.enable` | Enable Raft consensus. | `true` |
| `--evnode.raft.node_id` | `raft.node_id` | **Unique** identifier for the node. | e.g., `node-01` |
| `--evnode.raft.raft_addr` | `raft.raft_addr` | TCP address for Raft transport. | `0.0.0.0:5001` (Bind to private IP) |
| `--evnode.raft.raft_dir` | `raft.raft_dir` | Directory for Raft data. | `/data/raft` (Must be persistent) |
| `--evnode.raft.peers` | `raft.peers` | Comma-separated list of peer addresses in format `nodeID@host:port`. | `node-1@10.0.0.1:5001,node-2@10.0.0.2:5001,node-3@10.0.0.3:5001` |
| `--evnode.raft.bootstrap` | `raft.bootstrap` | Bootstrap the cluster. **Required** for initial setup. | `true` (See Limitations) |

### Timeout Tuning

Raft timeouts should be tuned relative to your **Block Time** (`--evnode.node.block_time`) to utilize the fast failover capabilities without causing instability.

| Flag | Default | Recommended Tuning |
|------|---------|--------------------|
| `--evnode.raft.heartbeat_timeout` | `1s` | **10-30% of Leader Lease**. For sub-second block times, lower to `50ms-100ms`. |
| `--evnode.raft.leader_lease_timeout` | `500ms` | **Must be < Election Timeout**. Use `500ms` for 1s block times. For slower chains (e.g., 10s blocks), increase to `1s-2s` to tolerate network jitter. |
| `--evnode.raft.send_timeout` | `1s` | Should be `> 2x RTT`. |

**Relation to Block Time**:
Ideally, a failover should complete within `2 * BlockTime` to minimize user impact.
*   **Fast Chain (BlockTime < 1s)**: Tighten timeouts. Heartbeat `50ms`, Lease `250ms`.
*   **Standard Chain (BlockTime = 1s)**: Heartbeat `100ms`, Lease `500ms`.
*   **Slow Chain (BlockTime > 5s)**: Defaults are usually sufficient (`1s` heartbeat).

> **Warning**: Setting timeouts too low (< RTT + Jitter) will cause leadership flapping and halted block production.

## Production Deployment Principles

### 1. Static Peering & Bootstrap
Current implementation requires **Bootstrap Mode** (`--evnode.raft.bootstrap=true`) for all nodes participating in the cluster initialization.
*   **All nodes** should list the full set of peers in `--evnode.raft.peers`.
*   The `peers` list format is strict: `NodeID@Host:Port`.
*   **Limitation**: Dynamic addition of peers (Run-time Membership Changes) via RPC/CLI is not currently exposed. The cluster membership is static based on the initial bootstrap configuration.

### 2. Infrastructure Requirements
*   **Encrypted Network (CRITICAL)**: Raft traffic is **unencrypted** (plain TCP). You **MUST** run the cluster inside a private network, VPN, or encrypted mesh (e.g., WireGuard, Tailscale). **Never expose Raft ports to the public internet**; doing so allows attackers to hijack the cluster consensus.
*   **Cluster Size**: Run an **odd number** of nodes (3 or 5) to tolerate failures (3 nodes tolerate 1 failure; 5 nodes tolerate 2).
*   **Storage**: The `--evnode.raft.raft_dir` **MUST** be mounted on persistent storage. Loss of this directory will cause the node to lose its identity and commit history, effectively removing it from the cluster.
*   **Network**: Raft requires low-latency, reliable connectivity. Ensure firewall rules allow TCP traffic on `raft_addr`.

### 3. P2P Interaction & Catch-Up
Raft and P2P work in parallel to ensure reliability:
*   **Hot Replication (Raft)**: New blocks produced by the leader are replicated via the Raft transport (Header + Data) to all followers. This ensures low-latency propagation of the chain tip.
*   **Catch-Up (P2P)**: If a node falls behind (e.g., disconnected for longer than the Raft log retention), it will receive a **Raft Snapshot** to update its consensus state to the latest head. However, the *historical blocks* between its local state and the new head are fetched via the **P2P Network** (or DA).
    *   **Implication**: You must ensure P2P connectivity (`--p2p.listen_address` and `--p2p.peers`) is configured even for Raft nodes, to allow them to backfill missing data from peers.

### 4. Lifecycle Management
*   **Rolling Restarts**: You can restart nodes one by one. The `ev-node` implementation handles graceful shutdown (leadership transfer) to minimize impact.
*   **State Divergence**: If a node falls too far behind or its local store conflicts with Raft (e.g., due to catastrophic disk failure), it may panic on startup to protect safety. In such cases, a manual extensive recovery (wiping state and re-syncing) may be required.

### 4. Monitoring
Monitor the following metrics (propagated via Prometheus if enabled):
*   **Leadership Changes**: Frequent changes indicate network instability or overloaded nodes.
*   **Applied Index vs Commit Index**: A growing lag indicates the FSM cannot keep up.

## Example Command

```bash
./ev-node start \
  --node.aggregator \
  --raft.enable \
  --raft.node_id="node-1" \
  --raft.raft_addr="0.0.0.0:5001" \
  --raft.raft_dir="/var/lib/ev-node/raft" \
  --raft.bootstrap=true \
  --raft.peers="node-1@10.0.1.1:5001,node-2@10.0.1.2:5001,node-3@10.0.1.3:5001" \
  --p2p.listen_address="/ip4/0.0.0.0/tcp/26656" \
  ...other flags
```
