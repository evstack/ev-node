# Migration Guide

Migrate an existing Cosmos SDK chain from CometBFT to Evolve while preserving state.

## Overview

The migration process:

1. Add migration modules to your chain
2. Pass governance proposal to halt at upgrade height
3. Export state and run migration
4. Restart with ev-abci

## Phase 1: Add Migration Modules

### Add Migration Manager

The migration manager handles the transition from multi-validator to single-sequencer.

```go
import (
    migrationmngr "github.com/evstack/ev-abci/modules/migrationmngr"
    migrationmngrkeeper "github.com/evstack/ev-abci/modules/migrationmngr/keeper"
    migrationmngrtypes "github.com/evstack/ev-abci/modules/migrationmngr/types"
)
```

Add the keeper to your app and register the module.

### Replace Staking Module

Replace the standard staking module with ev-abci's wrapper to prevent validator updates during migration:

```go
// Replace this:
import "github.com/cosmos/cosmos-sdk/x/staking"

// With this:
import "github.com/evstack/ev-abci/modules/staking"
```

## Phase 2: Governance Proposal

Submit a software upgrade proposal:

```bash
appd tx gov submit-proposal software-upgrade v2-evolve \
  --title "Migrate to Evolve" \
  --description "Upgrade to ev-abci consensus" \
  --upgrade-height <HEIGHT> \
  --from <KEY>
```

Vote on the proposal and wait for it to pass.

## Phase 3: Wire ev-abci

Before the chain halts, update your start command to use ev-abci (see [Integrate ev-abci](/getting-started/cosmos/integrate-ev-abci)).

Rebuild your binary:

```bash
go build -o appd ./cmd/appd
```

**Do not start the node yet.**

## Phase 4: Run Migration

After the chain halts at the upgrade height:

```bash
appd evolve-migrate
```

This command:
- Migrates blocks from CometBFT to Evolve format
- Converts state to Evolve format
- Creates `ev_genesis.json`
- Seeds sync stores

## Phase 5: Restart

Start with ev-abci:

```bash
appd start \
  --evnode.node.aggregator \
  --evnode.da.address <DA_ADDRESS> \
  --evnode.signer.passphrase <PASSPHRASE>
```

The chain continues from the last CometBFT state with the new consensus engine.

## Considerations

- **Downtime**: Chain is halted during migration (typically minutes)
- **Coordination**: All node operators must upgrade simultaneously
- **Rollback**: Keep CometBFT binary and data backup for emergency rollback
- **Vote extensions**: Not supported in Evolve—will have no effect after migration

## Full Node Migration

For non-sequencer nodes, skip the aggregator flag:

```bash
appd start \
  --evnode.da.address <DA_ADDRESS> \
  --evnode.p2p.peers <SEQUENCER_P2P_ID>@<HOST>:<PORT>
```

## Next Steps

- [ev-abci Migration from CometBFT](/ev-abci/migration-from-cometbft) — Detailed migration reference
- [Run a Full Node](/guides/running-nodes/full-node) — Non-sequencer setup
