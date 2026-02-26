# Migration Manager Module

Coordinates the transition from CometBFT multi-validator consensus to Evolve single-sequencer mode.

## Purpose

The migration manager:

- Stores the designated sequencer address
- Tracks migration height
- Coordinates with the staking wrapper to freeze validators
- Provides the `MsgMigrateToEvolve` message for triggering migration

## Installation

### Add to app.go

```go
import (
    migrationmngr "github.com/evstack/ev-abci/modules/migrationmngr"
    migrationmngrkeeper "github.com/evstack/ev-abci/modules/migrationmngr/keeper"
    migrationmngrtypes "github.com/evstack/ev-abci/modules/migrationmngr/types"
)

// Add store key
keys := sdk.NewKVStoreKeys(
    // ... other keys
    migrationmngrtypes.StoreKey,
)

// Create keeper
app.MigrationManagerKeeper = migrationmngrkeeper.NewKeeper(
    appCodec,
    keys[migrationmngrtypes.StoreKey],
    app.StakingKeeper,
    app.BankKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// Add to module manager
app.ModuleManager = module.NewManager(
    // ... other modules
    migrationmngr.NewAppModule(appCodec, app.MigrationManagerKeeper),
)
```

### Genesis Configuration

```json
{
  "app_state": {
    "migrationmngr": {
      "params": {
        "sequencer_address": "",
        "migration_height": "0"
      }
    }
  }
}
```

## Migration Flow

### 1. Governance Proposal

Submit a proposal to set migration parameters:

```bash
appd tx gov submit-proposal set-sequencer \
  --sequencer-address cosmos1... \
  --migration-height 5000001 \
  --from <KEY>
```

### 2. Vote and Pass

Standard governance voting process.

### 3. Chain Halts

At migration height, the chain halts automatically.

### 4. Run Migration

```bash
appd evolve-migrate
```

### 5. Restart with ev-abci

```bash
appd start \
  --evnode.node.aggregator \
  --evnode.da.address <DA_ADDRESS> \
  --evnode.signer.passphrase <PASSPHRASE>
```

## Messages

### MsgSetMigrationParams

Set migration parameters (governance-gated):

```protobuf
message MsgSetMigrationParams {
  string authority = 1;
  string sequencer_address = 2;
  int64 migration_height = 3;
}
```

### MsgMigrateToEvolve

Trigger the migration (called internally):

```protobuf
message MsgMigrateToEvolve {
  string authority = 1;
}
```

## Queries

```bash
# Get migration params
appd query migrationmngr params

# Get previous validators (post-migration)
appd query migrationmngr previous-validators
```

## State

| Key | Description |
|-----|-------------|
| `params` | Sequencer address and migration height |
| `previous_validators` | Validator set before migration (for reference) |
| `migration_complete` | Boolean flag |

## Next Steps

- [Staking Wrapper](/ev-abci/modules/staking-wrapper) — Freeze validator set
- [Migration from CometBFT](/ev-abci/migration-from-cometbft) — Full migration guide
