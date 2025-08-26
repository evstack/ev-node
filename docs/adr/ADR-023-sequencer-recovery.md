
# ADR 023: Sequencer Recovery & Liveness — Rafted Conductor vs 1‑Active/1‑Failover

## Changelog

- 2025-08-21: Initial ADR authored; compared approaches and captured failover and escape‑hatch semantics.

## Context

We need a robust, deterministic way to keep L2 block production live when the primary sequencer becomes unhealthy or unreachable, and to **recover leadership** without split‑brain or unsafe reorgs. The solution must integrate cleanly with `ev-node`, be observable, and support zero‑downtime upgrades. This ADR evaluates two designs for the **control plane** that governs which node is allowed to run the sequencer process.

## Alternative Approaches

Considered but not chosen for this iteration:

- **Many replicas, no coordination**: high risk of **simultaneous leaders** (split‑brain) and soft‑confirmation reversals.
- **Full BFT consensus among sequencers**: heavier operational/engineering cost than needed; our fault model is crash‑fault tolerance with honest operators.
- **Outsource ordering to a shared sequencer network**: viable but introduces an external dependency and different SLOs; out of scope for the immediate milestone.
- **Manual failover only**: too slow and error‑prone for production SLOs.

## Decision

> We will operate **1 active + 1 failover** sequencer at all times, regardless of control plane. Two implementation options are approved:

- **Design A — Rafted Conductor (CFT)**: A sidecar *conductor* runs next to each `ev-node`. Conductors form a **Raft** cluster to elect a single leader and **gate** sequencing so only the Raft leader may produce blocks via the Admin Control API. Applicability: use Raft only when there are **≥ 3 sequencers** (prefer odd N: 3, 5, …). Do not use Raft for two-node 1‑active/1‑failover clusters; use Design B in that case.
  *Note:* OP Stack uses a very similar pattern for its sequencer; see `op-conductor` in References.

- **Design B — 1‑Active / 1‑Failover (Lease/Lock)**: One hot standby promotes itself when the active fails by acquiring a **lease/lock** (e.g., Kubernetes Lease or external KV). Strong **fencing** ensures the old leader cannot keep producing after lease loss.

**Why both assume 1A/1F:** Even with Raft, we intentionally keep **n** nodes on hot standby capable of immediate promotion; additional nodes may exist as **read‑only** or **witness** roles to strengthen quorum without enabling extra leaders.

Status of this decision: **Proposed** for implementation and test hardening.

## Detailed Design

### User requirements
- **No split‑brain**: at most one sequencer is active.
- **Deterministic recovery**: new leader starts from a known **unsafe head**.
- **Fast failover**: p50 ≤ 15s, p95 ≤ 45s.
- **Operational clarity**: health metrics, leader identity, and explicit admin controls.
- **Zero‑downtime upgrades**: blue/green leadership transfer.

### Systems affected
- `ev-node` (sequencer control hooks, health surface).
- New sidecar(s): **conductor** (Design A) or **lease‑manager** (Design B).
- RPC ingress (optional **leader‑aware proxy** to route sequencing endpoints only to the leader).
- CI/CD & SRE runbooks, dashboards, alerts.

### New/changed data structures
- **UnsafeHead** record persisted by control plane: `(l2_number, l2_hash, l1_origin, timestamp)`.
- **Design A (Raft)**: replicated **Raft log** entries for `UnsafeHead`, `LeadershipTerm`, and optional `CommitMeta` (batch/DA pointers); periodic snapshots.
- **Design B (Lease)**: a single **Lease** record (Kubernetes Lease or external KV entry) plus a monotonic **lease token** for fencing.

### Admin Control API (Protobuf)

We introduce a separate, authenticated Admin Control API dedicated to sequencing control. This API is not exposed on the public RPC endpoint and binds to a distinct listener (port/interface, e.g., `:8443` on an internal network or loopback-only in single-host deployments). It is used exclusively by the conductor/lease-manager and by privileged operator automation for break-glass procedures.

Service overview:
- StartSequencer: Arms/starts sequencing subject to fencing (valid lease/term) and optionally pins to last persisted UnsafeHead.
- StopSequencer: Hard stop with optional “force” semantics.
- PrepareHandoff / CompleteHandoff: Explicit, auditable, two-phase, blue/green leadership transfer.
- Health / Status: Health probes and machine-readable node + leader state.

Endpoint separation:
- Public JSON-RPC and P2P endpoints remain unchanged.
- Admin Control API is out-of-band and must not be routed through public ingress. It sits behind mTLS and strict network policy.

Protobuf schema (proposed file: `proto/evnode/admin/v1/control.proto`):

```
syntax = "proto3";

package evnode.admin.v1;

option go_package = "github.com/evstack/ev-node/types/pb/evnode/admin/v1;adminv1";

// ControlService governs sequencer lifecycle and health surfaces.
// All operations must be authenticated via mTLS and authorized via RBAC.
service ControlService {
  // StartSequencer starts sequencing if and only if the caller holds leadership/fencing.
  rpc StartSequencer(StartSequencerRequest) returns (StartSequencerResponse);

  // StopSequencer stops sequencing. If force=true, cancels in-flight loops ASAP.
  rpc StopSequencer(StopSequencerRequest) returns (StopSequencerResponse);

  // PrepareHandoff transitions current leader to a safe ready-to-yield state
  // and issues a handoff ticket bound to the current term/unsafe head.
  rpc PrepareHandoff(PrepareHandoffRequest) returns (PrepareHandoffResponse);

  // CompleteHandoff is called by the target node to atomically assume leadership
  // using the handoff ticket. Enforces fencing and continuity from UnsafeHead.
  rpc CompleteHandoff(CompleteHandoffRequest) returns (CompleteHandoffResponse);

  // Health returns node-local liveness and recent errors.
  rpc Health(HealthRequest) returns (HealthResponse);

  // Status returns leader/term, active/standby, and build info.
  rpc Status(StatusRequest) returns (StatusResponse);
}

message UnsafeHead {
  uint64 l2_number = 1;
  bytes  l2_hash   = 2; // 32 bytes
  string l1_origin = 3; // opaque or hash/height string
  int64  timestamp = 4; // unix seconds
}

message LeadershipTerm {
  uint64 term      = 1; // monotonic term/epoch for fencing
  string leader_id = 2; // conductor/node ID
}

message StartSequencerRequest {
  bool   from_unsafe_head = 1;  // if false, uses safe head per policy
  bytes  lease_token      = 2;  // opaque, issued by control plane (Raft/Lease)
  string reason           = 3;  // audit string
  string idempotency_key  = 4;  // optional, de-duplicate retries
  string requester        = 5;  // principal for audit
}
message StartSequencerResponse {
  bool            activated = 1;
  LeadershipTerm  term      = 2;
  UnsafeHead      unsafe    = 3;
}

message StopSequencerRequest {
  bytes  lease_token     = 1;
  bool   force           = 2;
  string reason          = 3;
  string idempotency_key = 4;
  string requester       = 5;
}
message StopSequencerResponse {
  bool stopped = 1;
}

message PrepareHandoffRequest {
  bytes  lease_token     = 1;
  string target_id       = 2; // logical target node ID
  string reason          = 3;
  string idempotency_key = 4;
  string requester       = 5;
}
message PrepareHandoffResponse {
  bytes           handoff_ticket = 1; // opaque, bound to term+unsafe head
  LeadershipTerm  term           = 2;
  UnsafeHead      unsafe         = 3;
}

message CompleteHandoffRequest {
  bytes  handoff_ticket  = 1;
  string requester       = 2;
  string idempotency_key = 3;
}
message CompleteHandoffResponse {
  bool           activated = 1;
  LeadershipTerm term      = 2;
  UnsafeHead     unsafe    = 3;
}

message HealthRequest {}
message HealthResponse {
  bool   healthy     = 1;
  uint64 l2_number   = 2;
  bytes  l2_hash     = 3;
  string l1_origin   = 4;
  uint64 peer_count  = 5;
  uint64 da_height   = 6;
  string last_err    = 7;
}

message StatusRequest {}
message StatusResponse {
  bool   sequencer_active = 1;
  string build_version    = 2;
  string leader_hint      = 3; // optional, human-readable
  string last_err         = 4;
  LeadershipTerm term     = 5;
}
```

Error semantics:
- PERMISSION_DENIED: AuthN/AuthZ failure, missing or invalid mTLS identity.
- FAILED_PRECONDITION: Missing/expired lease or fencing violation; handoff ticket invalid.
- ABORTED: Lost leadership mid-flight; TOCTOU fencing triggered self-stop.
- ALREADY_EXISTS: Start requested but sequencer already active with same term.
- UNAVAILABLE: Local dependencies not ready (DA client, exec engine).

### Efficiency considerations
- **Design A:** Raft heartbeats and snapshotting add small steady‑state overhead; no impact on throughput when healthy.
- **Design B:** Lease renewals are lightweight; performance dominated by `ev-node` itself.

### Expected access patterns
- Reads (RPC, state) should work on all nodes; **writes/sequence endpoints** only on the active leader. If a leader‑aware proxy is deployed, it enforces this automatically.

### Logging/Monitoring/Observability
- Metrics: `leader_id`, `raft_term` (A), `lease_owner` (B), `unsafe_head_advance`, `peer_count`, `rpc_error_rate`, `da_publish_latency`, `backlog`, `leader_election_epoch`, `leader_election_leader_last_seen_ts`, `leader_election_heartbeat_timeout_total`, `leader_election_leader_uptime_ms`.
- Alerts: no unsafe advance > 3× block time; unexpected leader churn; lease lost but sequencer still active (fencing breach).
- Logs: audit all **Start/Stop** decisions and override operations.

### Security considerations
- Lock down **Admin RPC** with mTLS + RBAC; only the sidecar/process account may call Start/Stop.
- Implement **fencing**: leader periodically validates it still holds leadership/lease; otherwise self‑stops.
- Break‑glass overrides must be gated behind separate credentials and produce auditable events.

### Privacy considerations
- None beyond existing node telemetry; no user data added.

### Testing plan
- Kill active sequencer → verify failover within SLO; assert **no double leadership**.
- Partition tests: only Raft majority (A) or lease holder (B) may produce.
- Blue/green: explicit leadership handoff; confirm unsafe head continuity.
- Misconfigured standby → failover should **refuse**; alarms fire.
- Long‑duration outage drills; confirm user‑facing status and catch‑up behavior.

### Change breakdown
- Phase 1: Implement Admin RPC + health surface in `ev-node`; add sidecar skeletons.
- Phase 2: Integrate Design A (Raft) in a 1 sequencer + 2 failover; build dashboards/runbooks.
- Phase 3: Add Design B (Lease) profile for small/test clusters; share common health logic.
- Phase 4: Game days and SLO validation; finalize SRE playbooks.

### Release/compatibility
- **Breaking release?** No — Admin RPCs are additive.

## Status

Proposed

## Consequences

### Positive
- Clear, deterministic leadership with fencing; supports zero‑downtime upgrades.
- Works with `ev-node` via a small, well‑defined Admin RPC.
- Choice of control plane allows right‑sizing ops: Raft for prod; Lease for small/test.

### Negative
- Design A adds Raft operational overhead (quorum management, snapshots).
- Design B has a smaller blast radius but does not generalize to N replicas; stricter reliance on correct fencing.
- Additional components (sidecars, proxies) increase deployment surface.

### Neutral
- Small steady‑state CPU/network overhead for heartbeats/leases; negligible compared to sequencing and DA posting.

## References

- **OP conductor** (industry prior art; similar to Design A):
  - Docs: https://docs.optimism.io/operators/chain-operators/tools/op-conductor
  - README: https://github.com/ethereum-optimism/optimism/blob/develop/op-conductor/README.md

- **`ev-node`** (architecture, sequencing):
  - Repo: https://github.com/evstack/ev-node
  - Quick start: https://ev.xyz/guides/quick-start
  - Discussions/issues on sequencing API & multi-sequencer behavior.

- **Lease-based leader election**:
  - Kubernetes Lease API: https://kubernetes.io/docs/concepts/architecture/leases/
  - client-go leader election helpers: https://pkg.go.dev/k8s.io/client-go/tools/leaderelection
