# ADR 023: Execution-Owned Proposer Rotation

## Changelog

- 2026-04-24: Initial ADR.

## Status

Proposed

## Context

ev-node originally selected the block proposer from genesis. That made proposer changes a consensus configuration concern and pushed key rotation into a static schedule. This is too rigid for EVM rollups and other execution environments where proposer selection should be governed by execution state.

The replacement design moves proposer selection into the execution environment. ev-node remains responsible for signing, propagating, validating, and persisting blocks, but it consumes proposer updates returned by execution.

## Decision

`Executor.ExecuteTxs` returns an execution result containing:

- `UpdatedStateRoot`: the state root after executing the block.
- `NextProposerAddress`: the address expected to sign the next block.

`GetExecutionInfo` also exposes `NextProposerAddress` for startup. If execution returns an empty proposer at startup, ev-node falls back to `genesis.proposer_address`.

An empty `NextProposerAddress` from `ExecuteTxs` means the proposer is unchanged. ev-node must not write a redundant header field in that case, preserving compatibility with existing headers and hash chains.

When execution returns a non-empty next proposer:

- `State.NextProposerAddress` is updated and used as the expected signer for `LastBlockHeight + 1`.
- Full nodes validate the next block signer against the previous state's `NextProposerAddress`.
- Header encoding remains unchanged. `Header.ProposerAddress` continues to identify the signer of the current block only.

The execution result is the authority for proposer rotation. Header-only paths cannot derive proposer transitions without either replaying execution or using a future proof/certificate mechanism. This preserves header compatibility while keeping the rotation rule deterministic for full nodes.

## EVM System Contract Model

For ev-reth, proposer selection should be implemented as execution state, likely through a system contract. The contract stores the active next proposer address and exposes controlled update methods.

The controlling address can be a multisig or security council. This keeps operational key rotation in execution state instead of requiring a new genesis file or node-side schedule. A future ev-reth implementation should read the contract during block execution and return the selected proposer through `ExecuteTxsResponse.next_proposer_address`.

This ADR does not define the system contract ABI. The contract should be specified with ev-reth because access control, call routing, and predeploy/system-contract conventions are execution-environment details.

## Security Considerations

The security council or multisig becomes the authority for proposer updates. It must use a threshold and operational process appropriate for production signer rotation.

The system contract must restrict writes to the configured authority. Unauthorized proposer updates are consensus-critical because they determine who can sign the next block.

ev-node validates each block's signer against the proposer address stored in the previous state. A malicious proposer cannot rotate the next signer through node-local configuration; the rotation must be derived from execution.

If the execution interface returns an empty proposer, ev-node treats the proposer as unchanged. At startup, empty execution info falls back to genesis so existing execution implementations remain usable.

Compromise of the security council can still rotate the proposer to an attacker. This ADR reduces node configuration risk; it does not eliminate governance-key risk.

## Consequences

Positive:

- Proposer rotation becomes deterministic execution state.
- EVM chains can use a system contract and multisig-controlled rotation.
- Existing chains keep working when execution returns an empty proposer.
- Existing header encoding remains compatible because no new header field is required.

Negative:

- The execution API changes and all execution adapters must return `ExecuteResult`.
- Proposer updates become consensus-critical execution outputs.
- ev-reth needs a separate system-contract design and implementation.
- Header-only/light-client paths cannot follow proposer rotation without execution replay or a later proof design.

## Alternatives Considered

Genesis proposer schedule:

- Rejected. It makes rotation a static node/genesis concern and is not a good fit for security-council or multisig-controlled EVM deployments.

Node-local proposer configuration:

- Rejected. Nodes could disagree about the active proposer unless every operator updates configuration at the same time.

Header commitment for next proposer:

- Rejected for the first version. It would expose rotations to header-only paths, but it changes the signed header and hash encoding. Keeping rotation in execution/state avoids a header compatibility break.
