# EV-Reth Txpool Push Subscription for EV-Node

## Summary

Introduce a push-based txpool subscription in ev-reth that streams ordered,
mempool-selected transactions to ev-node, removing the 1s polling bottleneck.
The stream emits snapshots of the ordered tx list using the same selection
logic as `txpoolExt_getTxs`, with backpressure and a heartbeat to guarantee
<= 1s worst-case lag.

## Goals

- Deliver transactions to ev-node as soon as they appear in ev-reth.
- Preserve mempool ordering identical to `txpoolExt_getTxs`.
- Keep worst-case lag at or below 1s (current behavior).
- Avoid DoS amplification by applying backpressure and bounded work.
- Maintain compatibility with existing `txpoolExt_getTxs`.

## Non-Goals

- Guarantee lossless delivery across disconnects (recover via resync).
- Replace sequencing logic or batch queue behavior in ev-node.
- Change transaction ordering semantics or mempool policies.

## Current Behavior (Problem)

- ev-node reaper polls `Executor.GetTxs` on a fixed interval.
- EVM executor uses `txpoolExt_getTxs` (HTTP) to pull an ordered list.
- With 100ms blocks, a 1s poll means txs are only observed once per second.
- Lowering the interval increases RPC load and DoS surface.

## Proposed Architecture

### High-Level

- Add a new JSON-RPC subscription method in ev-reth:
  `txpoolExt_subscribeBestTxs`
- ev-node connects via WebSocket and subscribes.
- Each update is a snapshot of the best-ordered txs, using the same selection
  logic as `txpoolExt_getTxs`.
- ev-node uses the snapshot stream as the primary source of txs, with polling
  fallback when the stream is unavailable.

### Why Snapshot Streaming (Not "New Tx" Events)

Ordering must match the mempool ordering (`best_transactions()`).
Streaming "new txs" in arrival order does not preserve that ordering, and
would force ev-node to re-implement mempool ordering. Snapshot updates avoid
semantic drift and keep ordering consistent with `txpoolExt_getTxs`.

## API Specification

### Method

`txpoolExt_subscribeBestTxs` (WebSocket-only subscription)

### Params (optional object)

```json
{
  "max_bytes": 1939865,
  "max_gas": 30000000,
  "min_emit_interval_ms": 50,
  "max_emit_interval_ms": 1000,
  "include_metadata": true
}
```

| Parameter              | Default                 | Description                                        |
|------------------------|-------------------------|----------------------------------------------------|
| `max_bytes`            | ev-reth config value    | Per-update byte cap                                |
| `max_gas`              | current block gas limit | Per-update gas cap                                 |
| `min_emit_interval_ms` | 50                      | Debounce window for rapid pool updates             |
| `max_emit_interval_ms` | 1000                    | Heartbeat max to guarantee <= 1s lag               |
| `include_metadata`     | true                    | Adds size/gas counters and monotonic `snapshot_id` |

Parameters are fixed for the subscription lifetime. To change parameters,
disconnect and reconnect with new values.

### Payload (when `include_metadata = true`)

```json
{
  "snapshot_id": 42,
  "timestamp_ms": 1705123456789,
  "total_bytes": 102400,
  "total_gas": 15000000,
  "txs": ["0x...", "0x...", "..."]
}
```

| Field          | Type  | Description                                                   |
|----------------|-------|---------------------------------------------------------------|
| `snapshot_id`  | u64   | Monotonic counter, resets on ev-reth restart                  |
| `timestamp_ms` | u64   | Wall clock time (ev-reth system time) when snapshot was built |
| `total_bytes`  | u64   | Sum of tx bytes in this snapshot                              |
| `total_gas`    | u64   | Sum of tx gas limits in this snapshot                         |
| `txs`          | array | RLP-encoded transaction blobs in mempool priority order       |

### Ordering Guarantees

- `txs` order equals the ordering returned by `txpoolExt_getTxs` at the time
  the snapshot is generated.
- A snapshot represents the best ordered selection from the pool at that moment.
- **Ordering is locked on first sight**: once ev-node submits a tx to its batch
  queue, the position is fixed regardless of subsequent mempool reorderings.

## ev-reth Implementation Notes

### Snapshot Generation

- Use `TransactionPool::new_pending_pool_transactions_listener()` as the
  trigger source for updates.
- On event, generate a snapshot by running the same selection logic currently
  used in `txpoolExt_getTxs` (best transactions, max bytes, max gas).
- Iterate `best_transactions()` until `max_bytes` or `max_gas` limits are hit.
  No hard iteration cap; rely on size/gas limits for bounding.
- Transactions exceeding the snapshot limits are silently excluded. They may
  be picked up by polling fallback or future snapshots.

### Debouncing

- Use time-only debouncing: emit after `min_emit_interval_ms` regardless of
  whether updates continue arriving.
- Coalesce multiple pool updates within the debounce window into a single
  snapshot to cap CPU and bandwidth.

### Empty Pool Handling

- Skip emission when the transaction pool is empty.
- Heartbeat still fires at `max_emit_interval_ms` and emits an empty snapshot
  (`txs: []`) when the pool is empty to prove liveness.

### Backpressure

- Keep at most one pending snapshot per subscriber.
- If sink is slow, drop intermediate updates and send the latest snapshot.
- If sink is stalled beyond **5 seconds**, close the subscription immediately
  (abrupt close, no graceful close frame).

### Heartbeat

- If no updates occur within `max_emit_interval_ms`, emit the latest snapshot
  (or empty snapshot if pool is empty) to guarantee <= 1s lag.

### Subscriber Limits

- Enforce an explicit maximum number of concurrent subscribers.
- Default: **5 subscribers**.
- Configurable via ev-reth config.
- Reject new subscriptions with an error when limit is reached.

### Sync Requirement

- Reject subscription requests with an error if ev-reth is still syncing to
  chain tip.
- Subscription is only available when ev-reth is fully synced.

### Resource Safety

- No unbounded queues per subscriber.
- Snapshots are bounded by `max_bytes` and `max_gas`.
- Per-subscriber state is O(1) (snapshot_id, last_emit, dirty flag).

## ev-node Integration

### Config

| Config Key               | Type   | Default            | Description                                        |
|--------------------------|--------|--------------------|----------------------------------------------------|
| `evm.ws-url`             | string | none               | WebSocket URL for ev-reth (required for streaming) |
| `evm.txpool_subscribe`   | bool   | true if ws-url set | Enable txpool subscription                         |
| `evm.txpool_buffer_size` | int    | 3                  | Local snapshot buffer size (3-5 recommended)       |

Single `ws-url` only; high availability is handled at infrastructure level
(load balancer, DNS failover).

### Reaper Behavior

- If streaming is available, consume snapshots and submit new txs.
- If stream is unavailable:
  - Fall back to current polling behavior (`GetTxs`).
  - Retry stream connection with exponential backoff.

### Connection Lifecycle

1. On startup, attempt WebSocket connection to `evm.ws-url`.
2. Subscribe to `txpoolExt_subscribeBestTxs` with configured params.
3. If subscription is rejected (ev-reth syncing, limit reached), fall back to
   polling and retry after backoff.
4. On disconnect, switch to polling immediately and begin reconnection attempts.

### Reconnection Backoff

- **Base interval**: 100ms
- **Multiplier**: 2x on each failure
- **Cap**: 30 seconds
- **Retry**: Unbounded (retry forever while polling provides fallback)

### Snapshot ID Regression Handling

When ev-node detects `snapshot_id` regression (indicating ev-reth restart or
failover):

1. Clear the local snapshot buffer (discard all buffered snapshots).
2. Clear the seen-tx cache.
3. Continue processing from the new snapshot.

### Local Buffer

- Maintain a bounded buffer of **3-5 snapshots** (configurable).
- Always consume snapshots from WebSocket into buffer.
- If buffer is full, drop oldest snapshot.
- Log at WARN level and increment metric when dropping snapshots.

### Fallback Transition

- On stream disconnect: immediately switch to polling.
- On stream reconnect: immediately switch to streaming (trust dedup for overlap).
- No overlap window or drain period required.

### Deduplication

- Maintain seen-tx cache (hash-based).
- For each snapshot, filter txs already seen and submit only new txs.
- **Eviction policy**: Finality-driven. Clear entries when the batch containing
  them is confirmed on DA layer.
- On `snapshot_id` regression, clear the entire seen-tx cache.

### Validation

- Trust ev-reth implicitly; no RLP validation on receipt.
- Malformed data will fail at decode time during actual use.

### Logging

- Log each received snapshot at DEBUG level:
  - `snapshot_id`
  - `tx_count`
  - `receipt_lag_ms`
- Log stream connect/disconnect at INFO level.
- Log snapshot drops at WARN level.

### Semantics

- Ordered txs are provided by ev-reth; ev-node does not reorder.
- Snapshot processing is idempotent due to seen-tx cache.

## Failure Modes and Recovery

| Failure           | ev-reth Behavior                                                      | ev-node Behavior                                                      |
|-------------------|-----------------------------------------------------------------------|-----------------------------------------------------------------------|
| Stream disconnect | N/A                                                                   | Switch to polling, begin reconnect backoff                            |
| ev-reth overload  | Snapshot rate reduced via debounce; slow subscribers dropped after 5s | Polling fallback handles availability                                 |
| Missed updates    | N/A                                                                   | Next snapshot is a full best-txs view; ev-node recovers automatically |
| ev-reth restart   | snapshot_id resets to 0                                               | Detect regression, clear cache and buffer, continue                   |
| ev-reth syncing   | Reject subscription                                                   | Fall back to polling, retry subscription                              |

## Security and DoS Considerations

- Subscription is WS-only; exposure should follow existing RPC security policy.
- Authentication: network-level only (firewall, VPC). No application-level auth.
- Rate limits enforced via `min_emit_interval_ms` and backpressure.
- Subscriber cap prevents resource exhaustion.
- No unbounded memory growth.

## Observability

### ev-reth Metrics

| Metric                           | Type      | Description                           |
|----------------------------------|-----------|---------------------------------------|
| `txpoolext_subscribers`          | gauge     | Current number of active subscribers  |
| `txpoolext_snapshot_emits_total` | counter   | Total snapshots emitted               |
| `txpoolext_snapshot_drop_total`  | counter   | Snapshots dropped due to backpressure |
| `txpoolext_snapshot_build_ms`    | histogram | Time to build a snapshot              |

### ev-node Metrics

| Metric                               | Type      | Description                                |
|--------------------------------------|-----------|--------------------------------------------|
| `reaper_stream_connected`            | gauge     | 1 if stream connected, 0 otherwise         |
| `reaper_stream_reconnects_total`     | counter   | Total reconnection attempts                |
| `reaper_stream_snapshot_lag_ms`      | histogram | Receipt lag (timestamp_ms vs receive time) |
| `reaper_stream_fallback_polls_total` | counter   | Polls made while in fallback mode          |
| `reaper_stream_buffer_drops_total`   | counter   | Snapshots dropped due to full buffer       |

## Compatibility

- `txpoolExt_getTxs` remains unchanged.
- HTTP-only clients unaffected.
- WS subscription is additive and optional.
- No protocol versioning in initial release; add when breaking changes are needed.

## Testing Plan

### Unit Tests (ev-reth)

- Snapshot ordering matches `getTxs`.
- `max_bytes`/`max_gas` enforced.
- Heartbeat interval honored (emits at max_emit_interval_ms).
- Heartbeat emits empty snapshot when pool is empty.
- Backpressure drops intermediate snapshots.
- Subscriber cap enforced.
- Subscription rejected when syncing.

### Unit Tests (ev-node)

- Snapshot buffer bounded correctly.
- Oldest snapshot dropped when buffer full.
- seen-tx cache cleared on snapshot_id regression.
- Buffer cleared on snapshot_id regression.
- Reconnect backoff schedule correct (100ms base, 2x, 30s cap).
- Immediate fallback on disconnect.
- Immediate switch on reconnect.

### Integration Tests

- ev-node receives updates <= 1s.
- Stream disconnect triggers polling fallback.
- Ordering preserved end-to-end.
- seen-tx cache eviction on DA finality.
- Recovery from ev-reth restart (snapshot_id regression).

## Rollout Plan

1. Add ev-reth subscription API and metrics.
2. Add ev-node WS subscription client with fallback polling.
3. Enable subscription by default when `evm.ws-url` is set.
4. Keep existing polling configuration for opt-out and as fallback.

## Summary of Key Decisions

| Decision              | Choice                               |
|-----------------------|--------------------------------------|
| Ordering semantics    | Locked on first sight                |
| snapshot_id lifecycle | Ephemeral, resets on restart         |
| Cache eviction        | Finality-driven                      |
| Fallback transition   | Immediate switch, trust dedup        |
| Stall timeout         | 5s, abrupt close                     |
| Debounce strategy     | Time-only                            |
| Payload format        | Blobs only, ev-node decodes          |
| Oversized tx handling | Silent exclusion                     |
| Authentication        | Network-level only                   |
| Client buffering      | 3-5 snapshots, drop oldest           |
| Multi-reth support    | Single ws-url, infra-level HA        |
| Empty pool emission   | Skip (except heartbeat)              |
| Heartbeat on empty    | Emits empty snapshot                 |
| Reconnect backoff     | 100ms base, 30s cap, unbounded retry |
| Timestamp source      | Wall clock                           |
| Drop visibility       | Log + metric                         |
| Validation            | Trust ev-reth implicitly             |
| Lag measurement       | Receipt lag only                     |
| Param updates         | Reconnect required                   |
| Pre-sync behavior     | Reject subscription                  |
| Snapshot logging      | DEBUG level                          |
| Snapshot build cap    | Scan until limits hit                |
| Subscriber limit      | Explicit cap, default 5              |
| Regression handling   | Clear cache and buffer               |
| Default intervals     | 50ms / 1000ms                        |
| Health exposure       | Metrics only                         |
| Protocol versioning   | None initially                       |
