# ev-node Amsterdam Engine API Updates

This note summarizes the ev-node changes needed before enabling Amsterdam in an
ev-reth 2.3 chainspec.

## Current ev-node support

Current ev-node main has Prague/Osaka/Fusaka-style Engine API support:

- `engine_forkchoiceUpdatedV3`
- `engine_getPayloadV4` with fallback/cache to `engine_getPayloadV5`
- `engine_newPayloadV4`

That is enough while the execution chain remains pre-Amsterdam. It is not enough
for Amsterdam payloads.

The checked-in ev-node EVM genesis currently has `pragueTime: 0` and no
`amsterdamTime`, so this is not an immediate break for that default config.

## Why Amsterdam needs more work

Amsterdam introduces an Engine payload shape that ev-node must preserve:

- `PayloadAttributes` require `slotNumber`.
- Built execution payloads include `executionPayload.blockAccessList`.
- Reth 2.3 expects Amsterdam payloads through:
  - `engine_forkchoiceUpdatedV4`
  - `engine_getPayloadV6`
  - `engine_newPayloadV5`

The most important gap is `blockAccessList` passthrough. ev-node currently
decodes `getPayload` responses into go-ethereum's `engine.ExecutionPayloadEnvelope`.
In `go-ethereum v1.17.2`, `engine.ExecutableData` includes `slotNumber`, but the
Engine API envelope does not expose `executionPayload.blockAccessList`. Decoding
an Amsterdam payload into that type will lose the BAL before ev-node submits the
payload back with `newPayload`.

## Required code changes

### 1. Add Amsterdam Engine API method selection

Update `execution/evm/engine_rpc_client.go`:

- Add `engine_forkchoiceUpdatedV4`.
- Add `engine_getPayloadV6`.
- Add `engine_newPayloadV5`.
- Track the selected payload version beyond the current V4/V5 boolean.

Recommended shape:

- Replace `useV5 atomic.Bool` with a small enum or atomic version value.
- Keep unsupported-fork fallback for `getPayload`, but include V6.
- Select `newPayloadV5` whenever the payload has Amsterdam fields.
- Select `forkchoiceUpdatedV4` whenever payload attributes are Amsterdam.

### 2. Add `slotNumber` to Amsterdam payload attributes

Update the payload attributes map in `execution/evm/execution.go`.

Current map includes:

- `timestamp`
- `prevRandao`
- `suggestedFeeRecipient`
- `withdrawals`
- `parentBeaconBlockRoot`
- evolve-specific `transactions`
- evolve-specific `gasLimit`

For Amsterdam-active timestamps, add:

```json
"slotNumber": "<deterministic slot number>"
```

The slot number should come from an explicit chain rule. A reasonable default is
to derive it from rollup block height, but this should be agreed with the
ev-node/ev-reth chain semantics before enabling the fork.

### 3. Preserve `blockAccessList` from `getPayloadV6` to `newPayloadV5`

Do not try to compute the block access list in ev-node. ev-reth builds it.

Add a local Amsterdam-capable payload wrapper that can decode and re-encode the
full `executionPayload` object, including:

- all fields currently in `engine.ExecutableData`
- `slotNumber`
- `blockAccessList`

Implementation options:

- Use a custom struct embedding or mirroring `engine.ExecutableData` and adding
  `BlockAccessList hexutil.Bytes`.
- Or keep the raw `json.RawMessage` for `executionPayload`, decode only the
  fields ev-node needs for `blockNumber`, `timestamp`, `blockHash`, and
  `stateRoot`, then submit the raw payload object unchanged to `newPayloadV5`.

The raw-object approach is safest for Engine API compatibility because it avoids
dropping future EL payload fields.

### 4. Add fork activation awareness

ev-node currently does not have an EVM fork schedule option. It receives:

- ETH RPC URL
- Engine RPC URL
- JWT secret
- genesis hash
- fee recipient

Amsterdam selection needs one of:

- a new flag such as `--evm.amsterdam-time`, kept in sync with ev-reth genesis;
- parsing the ev-reth genesis/chainspec from a configured path;
- an Engine API capability probe plus explicit local state that knows when
  `slotNumber` must be present.

Do not rely only on `getPayload` unsupported-fork fallback. `forkchoiceUpdatedV4`
requires `slotNumber` when starting an Amsterdam build, so ev-node must know the
fork state before asking ev-reth to build a payload.

### 5. Update tracing and logs

Update `execution/evm/engine_rpc_tracing.go` so spans record the actual selected
method:

- `engine_forkchoiceUpdatedV3` or `engine_forkchoiceUpdatedV4`
- `engine_getPayloadV4`, `engine_getPayloadV5`, or `engine_getPayloadV6`
- `engine_newPayloadV4` or `engine_newPayloadV5`

Also update hard-coded log messages in `execution/evm/execution.go` that still
name V3/V4 methods.

## Tests to add

Add or extend tests in `execution/evm/engine_rpc_client_test.go`:

- Prague path uses `getPayloadV4` and `newPayloadV4`.
- Osaka/Fusaka path falls back to and caches `getPayloadV5`.
- Amsterdam path uses `forkchoiceUpdatedV4` with `slotNumber`.
- Amsterdam path uses `getPayloadV6`.
- Amsterdam path submits with `newPayloadV5`.
- `blockAccessList` returned by `getPayloadV6` is present in the submitted
  `newPayloadV5` execution payload.
- Unsupported-fork errors still switch only to valid alternate methods.

Update integration test helpers:

- `execution/evm/test/test_helpers.go` currently hard-codes
  `engine_forkchoiceUpdatedV3` for readiness.
- Any E2E config that enables `amsterdamTime` must use the updated method path.

## Docs to update

Update ev-node docs that currently describe only older Engine API versions:

- `execution/evm/README.md`
- `docs/reference/api/engine-api.md`
- `docs/ev-reth/engine-api.md`

Document that Prague/Osaka/Fusaka support is insufficient for Amsterdam because
Amsterdam adds `slotNumber`, `blockAccessList`, and newer Engine API methods.

## Enablement checklist

Before setting `amsterdamTime` in an ev-reth chainspec:

- [ ] ev-node can call `engine_forkchoiceUpdatedV4`.
- [ ] ev-node sends `slotNumber` in Amsterdam payload attributes.
- [ ] ev-node can call `engine_getPayloadV6`.
- [ ] ev-node preserves `executionPayload.blockAccessList`.
- [ ] ev-node can call `engine_newPayloadV5`.
- [ ] tracing/logs report the selected Engine API version.
- [ ] unit tests cover V4/V5/V6 version selection and BAL passthrough.
- [ ] E2E tests pass against an Amsterdam-enabled ev-reth chainspec.
