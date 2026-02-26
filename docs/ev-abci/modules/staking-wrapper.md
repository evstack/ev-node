# Staking Wrapper Module

A wrapper around the Cosmos SDK staking module that prevents validator set changes during migration.

## Purpose

When migrating from CometBFT to Evolve, the validator set must be frozen to allow a clean transition to single-sequencer mode. The staking wrapper:

- Prevents new delegations and undelegations from affecting the validator set
- Blocks validator creation and updates
- Allows the migration manager to perform the final transition

## Installation

Replace your staking module import:

```go
// Before
import "github.com/cosmos/cosmos-sdk/x/staking"

// After
import "github.com/evstack/ev-abci/modules/staking"
```

The wrapper is API-compatible with the standard staking module.

## Behavior

### Normal Operation

Before migration is triggered, the wrapper behaves identically to the standard staking module:

- Delegations work normally
- Validator operations work normally
- Rewards distribution works normally

### During Migration

Once the migration manager signals migration mode:

- `EndBlock` returns an empty validator update set
- Delegation changes are recorded but don't affect validators
- Validator creation/modification is blocked

### After Migration

Post-migration, the staking module becomes read-only for validator operations. The single sequencer is now the only block producer.

## Integration

### app.go

```go
import (
    stakingkeeper "github.com/evstack/ev-abci/modules/staking/keeper"
    stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// In your NewApp function:
app.StakingKeeper = stakingkeeper.NewKeeper(
    appCodec,
    keys[stakingtypes.StoreKey],
    app.AccountKeeper,
    app.BankKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)
```

### Module Manager

```go
import (
    staking "github.com/evstack/ev-abci/modules/staking"
)

// In your module manager:
app.ModuleManager = module.NewManager(
    // ... other modules
    staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
)
```

## Queries

All standard staking queries remain available:

```bash
appd query staking validators
appd query staking delegations <address>
appd query staking pool
```

## Next Steps

- [Migration Manager](/ev-abci/modules/migration-manager) — Coordinate the migration
- [Migration from CometBFT](/ev-abci/migration-from-cometbft) — Full migration guide
