# Migrating an Existing Chain to ev-abci

This guide is for developers of existing Cosmos SDK chains who want to replace their node's default CometBFT consensus engine and `start` command with the `ev-abci` implementation. By following these steps, you will reconfigure your chain to run as an `ev-abci` node.

## Overview of Changes

The migration process involves two key modifications:

1.  **Node Entrypoint (`main.go`):** Replacing the default `start` and `init` commands with the `ev-abci` versions.
2.  **Application Logic (`app.go`):** For certain modules like `migrationmngr`, you must add logic to your `app.go` file to correctly manage the chain's state during state-machine transitions.

This document will guide you through both parts.

---

## Part 1: Modifying the Node Entrypoint (`main.go`)

The first step is to change your application's main entrypoint to use the `ev-abci` server commands.

### 1. Locate Your Application's Entrypoint

Open the main entrypoint file for your chain's binary. This is usually found at `cmd/<your-app-name>/main.go`.

### 2. Modify `main.go`

You will need to add a new import and then overwrite the default `start` and `init` commands with the ones provided by `ev-abci`.

Below is a comprehensive example of what your `main.go` might look like before and after the changes.

#### Before Changes

A typical `main.go` file uses the default `server.AddCommands` to add all the standard node commands.

```go
// cmd/<appd>/main.go
package main

import (
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"

	"<your-app-path>/app"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "<appd>",
		Short: "Your App Daemon",
	}

	server.AddCommands(rootCmd, app.DefaultNodeHome, app.New, app.MakeEncodingConfig(), tx.DefaultSignModes)

	if err := rootCmd.Execute(); err != nil {
		server.HandleError(err)
		os.Exit(1)
	}
}
```

#### After Changes

To migrate, you will continue to use `server.AddCommands` to get useful commands like `keys` and `export`, but you will then overwrite the `start` and `init` commands with the `ev-abci` versions.

```go
// cmd/<appd>/main.go
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

	server.AddCommands(rootCmd, app.DefaultNodeHome, app.New, app.MakeEncodingConfig(), tx.DefaultSignModes)

	// --- Overwrite with ev-abci commands ---
	rootCmd.AddCommand(evabci_server.InitCmd())

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

---

## Part 2: Modifying the Application Logic (`app.go`)

If your chain utilizes the `migrationmngr` module to transition from a PoS validator set to a sequencer-based model, you must ensure that the standard `staking` module does not send conflicting validator updates during the migration.

*Note: In order to understand the migration manager in depth, please refer to the [migration manager documentation](https://github.com/evstack/ev-abci/tree/main/modules/migrationmngr).*

### For Chains Using the `migrationmngr` Module

**Goal:** To ensure the `migrationmngr` module is the *sole* source of validator set updates during a migration.

Instead of manually adding conditional logic to your `app.go`, you simply need to replace the standard Cosmos SDK `x/staking` module with the **staking wrapper module** provided in `ev-abci`. The wrapper's `EndBlock` method is hardcoded to prevent it from sending validator updates, cleanly delegating that responsibility to the `migrationmngr` module when a migration is active.

#### Action: Swap Staking Module Imports

In your `app.go` file (and any other files that import the staking module), replace the import for the Cosmos SDK's `x/staking` module with the one from `ev-abci`.

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

By changing the import path, your application's dependency injection system will automatically use the wrapper module. No other changes to your `EndBlocker` method are needed. This is the only required modification to `app.go` for the migration to work correctly.

---

## Part 3: Build and Verification

### 1. Re-build Your Application

After making these changes, re-build your application's binary:

```sh
go build -o <appd> ./cmd/<appd>
```

### 2. Verify the Changes

To verify that the `main.go` changes were successful, check the help text for the new `start` command. It should now include the `ev-abci` specific flags.

```sh
./<appd> start --help
```

You should see flags like `--ev-node.attester-mode`, `--ev-node.aggregator`, and others from the `ev-node` and `network` modules.

Your node is now fully configured to run using `ev-abci`. When you run the `start` command, it will no longer start a CometBFT instance but will instead start the `ev-abci` service that connects to a Rollkit sequencer, and it will correctly handle any on-chain migrations.
