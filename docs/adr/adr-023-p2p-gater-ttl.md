# ADR 023: P2P Connection Gater Blocklist

## Changelog

- 2026-04-20: Initial draft

## Context

`pkg/p2p/Client` uses `conngater.BasicConnectionGater` from go-libp2p to enforce peer-level
allow/block rules. The gater is constructed with the node's persistent datastore:

```go
// pkg/p2p/client.go:85
gater, err := conngater.NewBasicConnectionGater(ds)
```

`BasicConnectionGater` writes every `BlockPeer` call to the datastore under its own internal
keys. These entries **survive node restarts indefinitely**. Once a peer ID lands in the
blocklist — regardless of the reason — the node will refuse all outbound and inbound
connections to that peer forever, with no way to recover short of wiping the entire datastore.

This became a production incident on the `edennet-2` testnet (see evstack/ev-node#3267):
a fullnode accumulated stale block entries, which caused every binary builders nodes to be rejected by the
local gater. Header sync never initialized, and the node fell back to unstable and latency prone DA-only sync.

Two distinct categories of blocks exist today but are treated identically:

| Category | Source | Expected lifetime |
|---|---|---|
| **Config blocks** | `p2p.blocked_peers` in node config | As long as the entry stays in config |
| **Dynamic blocks** | Written by internal logic or operator tooling | Should expire; intent may be forgotten |

There is currently no way to:
- Distinguish the two categories in the datastore.
- Set a TTL or expiry on any block entry.
- Inspect which peers are blocked at runtime or at startup.
- Clear dynamic blocks without wiping the full datastore.
- Observe blocklist state via monitoring tooling.

## Alternative Approaches

### A — In-memory gater (no persistence)

Pass `nil` to `NewBasicConnectionGater`. All block state is ephemeral; config-driven
blocks are re-applied on every startup from `p2p.blocked_peers`.

**Pros:** Zero risk of stale state. Simple implementation.

**Cons:** Defeats the purpose of a persistent gater for genuinely malicious peers. A node
that dynamically bans a peer for misbehavior loses the ban on restart, potentially
reconnecting to the same bad actor immediately. Also, the upstream `BasicConnectionGater`
API has no in-memory shortcut that skips its own datastore writes when given `nil` in all
versions — it falls back to an in-memory map, but this is an implementation detail we
cannot rely on across library upgrades.

### B — Wipe gater namespace on startup before loading config blocks

On every start, delete the gater's datastore namespace, then re-apply only the blocks
listed in config.

**Pros:** Config blocks always reflect current intent. No stale entries.

**Cons:** All dynamic blocks (e.g., misbehaviour bans applied at runtime) are lost on every
restart, making runtime banning useless. Also requires knowing the internal datastore key
prefix used by `BasicConnectionGater`, which is a private implementation detail.

### C — TTL-based wrapper with startup reconciliation and observability (chosen)

Wrap `BasicConnectionGater` in a thin `TTLConnectionGater` that:
1. Stores block entries with an expiry timestamp in a separate ev-node-owned datastore
   namespace (not the gater's own namespace).
2. At startup, reads the TTL store, discards expired entries, and calls `BlockPeer` only
   for live entries — plus re-applies config blocks (with no expiry).
3. Provides a periodic background sweep to unblock peers whose TTL has elapsed at runtime.
4. Exposes Prometheus metrics for the current blocklist state, including per-peer labels.

This keeps persistence for intentional runtime bans, gives config blocks permanent status,
automatically heals nodes that accumulated stale entries, and makes the blocklist observable.

## Decision

Implement **Option C**: a `TTLConnectionGater` wrapper around `BasicConnectionGater` with
a default TTL of **24 hour** for dynamic blocks, permanent status for config-sourced blocks,
and Prometheus metrics exposing the full blocklist state.

The implementation is entirely within the `pkg/p2p` package. No external API or consensus
changes are required.

## Detailed Design

### Data model

A new datastore namespace `/p2p/gater/ttl/` stores JSON-encoded records keyed by peer ID:

```
/p2p/gater/ttl/<peerID>  →  {"expires_at": <unix-seconds>, "reason": "<string>", "source": "<config|dynamic>"}
```

`expires_at = 0` is a sentinel meaning "permanent" (used for config blocks).

The existing `BasicConnectionGater` datastore entries (written under its own private
namespace) will be **ignored going forward**. On first startup after the upgrade, the
TTLConnectionGater will not find its own records and will start fresh — effectively
clearing any stale legacy entries automatically.

### `TTLConnectionGater` struct

```go
// pkg/p2p/ttl_gater.go

type blockSource string

const (
    blockSourceConfig  blockSource = "config"
    blockSourceDynamic blockSource = "dynamic"
)

type blockRecord struct {
    ExpiresAt int64       `json:"expires_at"` // Unix seconds; 0 = permanent
    Reason    string      `json:"reason"`
    Source    blockSource `json:"source"`
}

type TTLConnectionGater struct {
    inner      *conngater.BasicConnectionGater
    ds         datastore.Datastore
    defaultTTL time.Duration
    metrics    *GaterMetrics
    logger     zerolog.Logger
    mu         sync.RWMutex
}
```

### Key methods

```go
// BlockPeerWithTTL blocks a peer for the given duration and persists the record.
// Use ttl=0 for a permanent block (config-driven).
func (g *TTLConnectionGater) BlockPeerWithTTL(id peer.ID, ttl time.Duration, reason string, source blockSource) error

// UnblockPeer removes a peer from the blocklist immediately.
func (g *TTLConnectionGater) UnblockPeer(id peer.ID) error

// ListBlocked returns all currently active block records.
func (g *TTLConnectionGater) ListBlocked() map[peer.ID]blockRecord

// InterceptPeerDial delegates to inner gater (which is registered with libp2p).
// TTLConnectionGater implements network.ConnectionGater by delegation.
```

### Startup reconciliation

In `NewClient` (or a dedicated `loadGaterState` helper called from `Start`):

1. Iterate all keys under `/p2p/gater/ttl/`.
2. For each record:
   - If `expires_at != 0` and `now > expires_at`: delete the datastore key and skip.
   - Otherwise: call `inner.BlockPeer(id)`.
3. For each peer in `conf.BlockedPeers`: call `BlockPeerWithTTL(id, 0, "config", blockSourceConfig)` —
   permanent, overwrites any existing TTL entry.
4. For each peer in `conf.AllowedPeers` (currently `UnblockPeer` targets): if a TTL record
   exists, delete it and call `inner.UnblockPeer(id)`.
5. After reconciliation, refresh all Prometheus metrics gauges to reflect current state.

### Background sweep

A goroutine started in `Client.Start` runs on a configurable interval (default: **5 min**):

```go
func (g *TTLConnectionGater) sweepExpired(ctx context.Context, interval time.Duration)
```

On each tick:
- Load all TTL records from the datastore.
- For any expired record: delete the datastore key and call `inner.UnblockPeer(id)`.
- Update Prometheus metrics to reflect the post-sweep state.
- Log the number of entries swept at `DEBUG` level.

### Startup diagnostic logging

After reconciliation, log at `INFO` level:

```
p2p gater loaded: 3 permanent blocks (config), 1 dynamic block (expires in 47m), 0 expired entries pruned
```

For each blocked peer, log at `DEBUG` level: peer ID, expiry time, reason, source.

### Prometheus metrics

Add a `GaterMetrics` struct to `pkg/p2p/metrics.go` alongside the existing `Metrics`:

```go
// GaterMetrics contains metrics for the TTL connection gater blocklist.
type GaterMetrics struct {
    // Total number of currently blocked peers.
    BlockedPeersTotal metrics.Gauge

    // Per-peer blocked status. Label peer_id identifies the peer; label source
    // is either "config" or "dynamic". Value is 1 while blocked, removed when
    // unblocked (set to 0 and the label combination is dropped).
    BlockedPeer metrics.Gauge `metrics_labels:"peer_id,source"`
}
```

Metric names (using the existing `p2p` subsystem):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `<ns>_p2p_gater_blocked_peers_total` | Gauge | — | Current count of blocked peers |
| `<ns>_p2p_gater_blocked_peer` | Gauge | `peer_id`, `source` | 1 if the peer is blocked; label is removed on unblock |

The `BlockedPeer` gauge uses a `peer_id` label containing the full libp2p peer ID string
(e.g. `12D3KooW...`). This allows operators to directly identify blocked peers from
Prometheus/Grafana without needing to correlate with logs.

`GaterMetrics` is constructed in `PrometheusMetrics` and its nop variant in `NopMetrics`.
`TTLConnectionGater` calls `metrics.BlockedPeersTotal.Set(n)` and
`metrics.BlockedPeer.With("peer_id", id.String(), "source", string(record.Source)).Set(1)`
after every block/unblock operation and after each sweep tick.

On unblock, set the gauge to 0:
```go
metrics.BlockedPeer.With("peer_id", id.String(), "source", string(record.Source)).Set(0)
```

### Configuration

Add to `config.P2PConfig`:

```go
// GaterBlockTTL is the default TTL for dynamically blocked peers.
// A value of 0 disables TTL (blocks are permanent unless explicitly unblocked).
// Default: 1h.
GaterBlockTTL    time.Duration `toml:"gater_block_ttl"`

// GaterSweepInterval controls how often expired blocks are pruned at runtime.
// Default: 5m.
GaterSweepInterval time.Duration `toml:"gater_sweep_interval"`
```

### `Client` wiring changes

- `Client.gater` field type changes from `*conngater.BasicConnectionGater` to
  `*TTLConnectionGater`.
- `Client.ConnectionGater()` return type changes accordingly (or returns the inner gater
  for use with `libp2p.ConnectionGater()`).
- `setupBlockedPeers` and `setupAllowedPeers` are replaced by calls to
  `TTLConnectionGater.BlockPeerWithTTL` and `TTLConnectionGater.UnblockPeer`.
- `GaterMetrics` is passed into `TTLConnectionGater` at construction time, sourced from
  the same `PrometheusMetrics` / `NopMetrics` call that provides `Metrics` to the client.

### Migration

No explicit migration needed. On first boot after upgrade:

- The new `/p2p/gater/ttl/` namespace is empty.
- The old `BasicConnectionGater` entries in the datastore still exist but are no longer
  read by ev-node. They remain as inert bytes until a future cleanup pass or datastore wipe.
- Config blocks are re-applied fresh from `conf.BlockedPeers` with permanent TTL.

This means any stale legacy blocks are silently dropped on the first upgrade boot, which is
the desired recovery behavior for nodes in the broken state described in evstack/ev-node#3267.

### Testing

- Unit tests for `TTLConnectionGater`: block, expiry check, sweep, reconciliation with
  mixed permanent + expiring entries.
- Unit tests for `GaterMetrics`: verify gauge values reflect block/unblock/sweep transitions.
- Integration test: create a client with stale gater entries in the datastore (simulating
  the legacy state), start a new client, verify the stale peers are not blocked.
- Existing `pkg/p2p` tests must continue to pass unchanged.

### Breaking changes

None. This is an internal implementation change. The gater datastore key space changes, but
the old keys are simply ignored (not deleted) so the change is safe to roll back by
reverting the binary — the old code will re-read its own namespace and apply whatever was
last written there.

New Prometheus metrics are additive and do not affect existing dashboards or alerts.

## Status

Proposed

## Consequences

### Positive

- Nodes that accumulated stale block entries automatically recover on upgrade without
  operator intervention.
- Dynamic bans (e.g., for misbehaviour) now expire, preventing permanent peer isolation
  caused by transient issues.
- Operators can see exactly which peers are blocked and why at startup (logs) and at
  runtime (Prometheus metrics).
- Config-driven blocks (`p2p.blocked_peers`) remain permanent as long as they are in
  config, matching operator intent.
- Prometheus metrics enable alerting on unexpected blocklist growth and identify specific
  blocked peer IDs without log scraping.

### Negative

- A peer that is dynamically banned for genuine misbehaviour will be unbanned after the
  TTL elapses and could reconnect. Operators who want permanent bans must add the peer to
  `p2p.blocked_peers` in config.
- Adds a new datastore namespace and a background goroutine, slightly increasing
  operational complexity.
- The `BlockedPeer` gauge with a `peer_id` label has high cardinality if many peers are
  blocked simultaneously. In practice the blocklist is expected to remain small (< 100
  entries), so this is acceptable.

### Neutral

- Old `BasicConnectionGater` datastore entries are abandoned in place. They do not affect
  correctness but consume a small amount of disk space until the datastore is pruned or
  wiped.

## References

- evstack/ev-node#3267 — production incident on edennet-2 that motivated this ADR
- [go-libp2p BasicConnectionGater](https://github.com/libp2p/go-libp2p/tree/master/p2p/net/conngater)
- `pkg/p2p/client.go` — current gater construction and peer block/allow helpers
- `pkg/p2p/metrics.go` — existing Prometheus metrics pattern for this package
