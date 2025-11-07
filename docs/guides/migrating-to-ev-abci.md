# Migrating an Existing Chain to ev-abci

This guide is for developers of existing Cosmos SDK chains who want to replace their node's default CometBFT consensus engine with the `ev-abci` implementation. By following these steps, you will migrate your chain to run as an `ev-abci` node while preserving chain state.

## Overview of Migration Process

The migration process involves the following key phases:

1. **Code Preparation:** Add migration module, staking wrapper, and upgrade handler to your existing chain
2. **Governance Proposal:** Create and pass a governance proposal to initiate the migration
3. **State Export:** Export the current chain state at the designated upgrade height
4. **Node Reconfiguration:** Wire the `ev-abci` start handler into your node's entrypoint
5. **Migration Execution:** Run `appd evolve-migrate` to transform the exported state
6. **Chain Restart:** Start the new `ev-abci` node with the migrated state

This document will guide you through each phase.

---

## Phase 1: Code Preparation - Add Migration Module and Staking Wrapper

The first step prepares your existing chain for migration by integrating the necessary modules.

### Step 1: Add Migration Manager Module

Add the `migrationmngr` module to your application. This module manages the transition from a PoS validator set to a sequencer-based model.

*Note: For detailed information about the migration manager, please refer to the [migration manager documentation](https://github.com/evstack/ev-abci/tree/main/modules/migrationmngr).*

In your `app.go` file:

1. Import the migration manager module:

```go
import (
	// ...
	migrationmngr "github.com/evstack/ev-abci/modules/migrationmngr"
	migrationmngrkeeper "github.com/evstack/ev-abci/modules/migrationmngr/keeper"
	migrationmngrtypes "github.com/evstack/ev-abci/modules/migrationmngr/types"
	// ...
)
```

2. Add the migration manager keeper to your app struct
3. Register the module in your module manager
4. Configure the migration manager in your app initialization

### Step 2: Replace Staking Module with Wrapper

**Goal:** Ensure the `migrationmngr` module is the *sole* source of validator set updates during migration.

Replace the standard Cosmos SDK `x/staking` module with the **staking wrapper module** provided in `ev-abci`. The wrapper's `EndBlock` method prevents validator updates from the staking module, delegating that responsibility to the `migrationmngr` module during migration.

In your `app.go` file (and any other files that import the staking module):

**Replace this:**

```go
import (
	// ...
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	// ...
)
```

**With this:**

```go
import (
	// ...
	"github.com/evstack/ev-abci/modules/staking" // The wrapper module
	stakingkeeper "github.com/evstack/ev-abci/modules/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types" // Staking types remain the same
	// ...
)
```

By changing the import path, your application will automatically use the wrapper module. No other changes to your `EndBlocker` method are needed.

---

## Phase 2: Create Upgrade Handler

Create an upgrade handler in your `app.go` that will be triggered when the governance proposal is executed.

```go
func (app *App) setupUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		"v2-migrate-to-evolve", // Upgrade name must match governance proposal
		func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			// The upgrade handler can initialize state for the migration manager if needed
			// The actual migration will happen during the evolve-migrate step
			return app.mm.RunMigrations(ctx, app.configurator, fromVM)
		},
	)
}
```

Call this function in your app initialization code in `app.go`.

---

## Phase 3: Create Governance Proposal for Migration

Create and submit a software upgrade governance proposal to initiate the migration at a specific block height.

```bash
# Create the governance proposal
<appd> tx gov submit-proposal software-upgrade v2-migrate-to-evolve \
  --title "Migrate to Evolve" \
  --description "Upgrade chain to use ev-abci consensus" \
  --upgrade-height <target-height> \
  --from <your-key> \
  --chain-id <chain-id>

# Vote on the proposal (repeat for validators to reach quorum)
<appd> tx gov vote <proposal-id> yes --from <validator-key>
```

Wait for the proposal to pass and for the chain to reach the upgrade height. The chain will halt at the specified height, waiting for the upgrade to be applied.

### Trigger Migration to Evolve

After the upgrade proposal has passed, submit the `MsgMigrateToEvolve` message to initiate the actual migration process. This can be done through a governance proposal or directly if your chain's authority allows it.

```bash
# Submit MsgMigrateToEvolve governance proposal (if using governance)
<appd> tx gov submit-proposal migrate-to-evolve \
  --title "Trigger Migration to Evolve" \
  --description "Execute migration to ev-abci consensus" \
  --from <your-key> \
  --chain-id <chain-id>

# Or submit directly if authority allows (authority address depends on your chain configuration)
<appd> tx migrationmngr migrate-to-evolve \
  --from <authority-key> \
  --chain-id <chain-id>
```

Once this message is processed, the migration manager module will handle the transition from the PoS validator set to the sequencer-based model.

---

## Phase 4: Wire ev-abci Start Handler in root.go

**⚠️ Important:** Complete this phase BEFORE the chain halts at the upgrade height. Do NOT start your node yet - you will start it in Phase 6 after running the migration command.

Modify your node's entrypoint to use the `ev-abci` server commands.

### Locate Your Application's Entrypoint

Open the main entrypoint file for your chain's binary, usually found at `cmd/<your-app-name>/main.go` or `root.go`.

### Modify the Start Command

Add the `ev-abci` start handler to your root command. This is similar to the [Ignite Apps evolve template](https://github.com/ignite/apps/blob/main/evolve/template/init.go#L48-L60).

```go
// cmd/<appd>/main.go (or root.go)
package main

import (
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"

	// Import the ev-abci server package
	evabci_server "github.com/evstack/ev-abci/server"

	"<your-app-path>/app"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "<appd>",
		Short: "Your App Daemon (ev-abci enabled)",
	}

	// Keep existing commands (keys, export, etc.)
	server.AddCommands(rootCmd, app.DefaultNodeHome, app.New, app.MakeEncodingConfig(), tx.DefaultSignModes)

	// --- Wire ev-abci start handler ---
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node with ev-abci",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return server.Start(cmd, evabci_server.StartHandler())
		},
	}

	evabci_server.AddFlags(startCmd)
	rootCmd.AddCommand(startCmd)
	// --- End of ev-abci changes ---

	if err := rootCmd.Execute(); err != nil {
		server.HandleError(err)
		os.Exit(1)
	}
}
```

### Build Your Application

Re-build your application's binary with the updated code:

```sh
go build -o <appd> ./cmd/<appd>
```

**⚠️ Important:** Do NOT start the node yet. Proceed directly to Phase 5 to run the migration command.

---

## Phase 5: Run evolve-migrate

After the chain halts at the upgrade height, run the migration command to transform the CometBFT data to Evolve format.

**⚠️ Critical:** The node must NOT be running when you execute this command. Ensure all node processes are stopped before proceeding.

```bash
# Run the migration command
<appd> evolve-migrate

# Optional: specify the DA height for the Evolve state (defaults to 1)
<appd> evolve-migrate --da-height <height>
```

The `evolve-migrate` command performs the following operations:

1. **Migrates all blocks** from the CometBFT blockstore to the Evolve store
2. **Converts the CometBFT state** to Evolve state format
3. **Creates `ev_genesis.json`** - a minimal genesis file that the node will automatically detect and use on subsequent startups
4. **Saves state** to the ABCI execution store for compatibility
5. **Seeds sync stores** with the latest migrated header and data
6. **Cleans up migration state** from the application database

**Important Notes:**

- The migration processes blocks in reverse order (from latest to earliest)
- If blocks are missing (e.g., due to pruning), they will be skipped. Migration stops if more than the configured maximum number of blocks are missing
- Vote extensions are not supported in Evolve - if they were enabled in your chain, they will have no effect after migration
- The command operates on the data in your node's home directory (e.g., `~/.appd/data/`)
- After successful migration, the `ev_genesis.json` file will be used automatically on node restart

---

## Phase 6: Start New ev-abci Node

Start your node with the migrated state:

```bash
<appd> start
```

Verify that the node starts successfully:

```sh
# Check that ev-abci flags are available
<appd> start --help

# You should see flags like:
# --ev-node.attester-mode
# --ev-node.aggregator
# --ev-node.sequencer-url
# etc.
```

Your node is now running with `ev-abci` instead of CometBFT. The chain continues from the same state but with the new consensus engine.

---

## Summary

The migration process follows these key phases:

1. **Code Preparation:** Modify your chain code to add the migration manager module and staking wrapper
2. **Create Upgrade Handler:** Define the upgrade logic that will be triggered by governance
3. **Governance Proposal:** Submit and pass a software upgrade proposal
4. **Wire Start Handler:** Update your node's entrypoint to use the `ev-abci` start command
5. **Execute Migration:** Run `appd evolve-migrate` to transform the exported state
6. **Restart Chain:** Start the new `ev-abci` node with the migrated state

This approach ensures a smooth migration with minimal downtime and preserves all chain state and history.
