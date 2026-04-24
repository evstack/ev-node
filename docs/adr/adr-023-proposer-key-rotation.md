# ADR 023: Proposer Key Rotation via Height-Based Schedule

## Changelog

- 2026-04-23: Implemented proposer key rotation through a height-indexed proposer schedule

## Context

ev-node historically treated the proposer as a single static identity embedded in genesis via `proposer_address`.
That assumption leaked into block production, DA submission, and sync validation. As a result, rotating a compromised
or operationally obsolete proposer key required out-of-band coordination and effectively behaved like a manual
re-genesis from the point of view of node operators.

This was suboptimal for three reasons:

1. It made proposer rotation operationally risky and easy to get wrong.
2. Fresh nodes syncing from genesis had no protocol-visible record of when the proposer changed.
3. Validation only pinned the proposer address, not the scheduled public key that should be producing blocks.

## Alternative Approaches

### 1. Manual key swap only

Operators can stop the sequencer, swap the local signer, redistribute config, and restart nodes.
This is insufficient because the chain itself does not encode when the proposer changed, so historical sync
and validation become ambiguous.

### 2. Re-issue a new genesis on each rotation

This treats every proposer rotation like a chain restart: a new `chain_id`, state reset back to `initial_height`,
and existing block history discarded. It is operationally heavy, conflates upgrades with rotations, and breaks
continuity for nodes syncing historical data.

### 3. Height-indexed proposer schedule in genesis (Chosen)

Record proposer changes as an ordered schedule indexed by activation height. The `genesis.json` file is updated
with a new schedule entry and redistributed, but the chain keeps its `chain_id`, continues from the current
height, preserves all block history, and fresh nodes can still validate the entire chain end-to-end across
rotation boundaries. The rollout is still coordinated — every node must receive the updated `genesis.json` and
restart before the activation height — but none of the chain's state or provenance is reset.

## Decision

ev-node now supports proposer rotation through a `proposer_schedule` field in genesis.

### What this is not

This is **not** a re-genesis. Re-genesis — in the sense we mean it above — would involve issuing a new `chain_id`,
resetting height to `initial_height`, and discarding existing block history. Proposer key rotation does none of
that: the `chain_id` is unchanged, block height keeps progressing, all previous blocks remain valid, and fresh
nodes can sync the chain from genesis across any number of rotation boundaries.

The `genesis.json` file itself is updated (a new `proposer_schedule` entry is appended) and operators must
restart every node to reload it. The file changes; the chain's state does not.

Each entry declares:

- `start_height`
- `address`
- `pub_key` (optional; when present, it must match `address`)

The active proposer for block height `h` is the last entry whose `start_height <= h`.

The legacy `proposer_address` field remains for backward compatibility. When no explicit schedule is present,
ev-node derives an implicit single-entry schedule beginning at `initial_height`.

When an explicit schedule is present:

- the first entry must start at `initial_height`
- entries must be strictly increasing by `start_height`
- if `pub_key` is present, the entry's `address` must match it
- entries without `pub_key` are interpreted by `address` only
- `proposer_address`, when present, must match the first schedule entry's `address`

## Detailed Design

### Data model

Genesis gains:

```json
"proposer_schedule": [
  {
    "start_height": 1,
    "address": "...",
    "pub_key": "..."
  },
  {
    "start_height": 1250000,
    "address": "..."
  }
]
```

The existing `proposer_address` field is retained as a compatibility field and is normalized to the first
scheduled proposer when a schedule is present.

### Validation rules

The proposer schedule is now consulted in all proposer-sensitive paths:

1. executor startup accepts any signer that appears somewhere in the schedule
2. block creation resolves the proposer for the exact height being produced
3. DA submission validates the configured signer against the scheduled proposer for each signed data height
4. sync validation validates incoming headers and signed data against the scheduled proposer for their heights

This makes proposer rotation protocol-visible for both live nodes and nodes syncing historical data.

### Operational procedure

For a planned rotation:

1. Choose activation height `H`
2. Add a new `proposer_schedule` entry with `start_height = H`
3. Distribute the updated genesis/config to node operators
4. Upgrade follower/full nodes before activation
5. Stop the old sequencer before `H`
6. Start the new sequencer with the replacement key at or after `H`

The old proposer remains valid for heights `< H`, and the new proposer becomes valid at heights `>= H`.

### Security considerations

This design improves safety by allowing validation against the scheduled public key when one is pinned.
It does not solve emergency rotation authorization by itself; a future design can add a separate upgrade authority
or rotation certificate flow if the network needs signer replacement without prior static scheduling.

### Testing

Coverage includes:

- genesis schedule validation and height resolution
- sync acceptance of scheduled proposer rotation
- DA submission using a rotated proposer key at the configured height
- executor block creation using the proposer scheduled for the produced height

## Status

Implemented

## Consequences

### Positive

- proposer rotation is now part of the chain configuration rather than an operator convention
- fresh nodes can validate historical proposer changes from genesis
- sync and DA validation can pin scheduled public keys, not just addresses
- routine key rotation no longer requires a chain restart

### Negative

- proposer schedule changes are consensus-visible and require coordinated rollout
- operators must distribute updated genesis/config before activation height
- emergency rotation still requires prior scheduling or a later authority-based mechanism

### Neutral

- legacy single-proposer deployments continue to work without defining `proposer_schedule`

## References

- [pkg/genesis/genesis.go](../../pkg/genesis/genesis.go)
- [pkg/genesis/proposer_schedule.go](../../pkg/genesis/proposer_schedule.go)
- [block/internal/executing/executor.go](../../block/internal/executing/executor.go)
- [block/internal/syncing/assert.go](../../block/internal/syncing/assert.go)
