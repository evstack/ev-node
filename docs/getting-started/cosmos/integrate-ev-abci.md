# Integrate ev-abci

Manually integrate ev-abci into an existing Cosmos SDK application.

## Overview

ev-abci replaces CometBFT as the consensus engine for your Cosmos SDK chain. Your application logic remains unchanged—only the node startup code changes.

## 1. Add Dependency

```bash
go get github.com/evstack/ev-abci@latest
```

## 2. Modify Your Start Command

Locate your application's entrypoint, typically `cmd/<appd>/main.go` or `cmd/<appd>/root.go`.

Replace the CometBFT server with ev-abci:

```go
package main

import (
    "os"

    "github.com/cosmos/cosmos-sdk/server"
    "github.com/spf13/cobra"

    // Import ev-abci server
    evabci "github.com/evstack/ev-abci/server"

    "your-app/app"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "appd",
        Short: "Your App Daemon",
    }

    // Keep existing commands
    server.AddCommands(rootCmd, app.DefaultNodeHome, app.New, app.MakeEncodingConfig())

    // Replace start command with ev-abci
    startCmd := &cobra.Command{
        Use:   "start",
        Short: "Run the node with ev-abci",
        RunE: func(cmd *cobra.Command, _ []string) error {
            return evabci.StartHandler(cmd, app.New)
        },
    }

    evabci.AddFlags(startCmd)
    rootCmd.AddCommand(startCmd)

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## 3. Build

```bash
go build -o appd ./cmd/appd
```

## 4. Verify

Check that ev-abci flags are available:

```bash
./appd start --help
```

You should see flags like:

```
--evnode.node.aggregator
--evnode.da.address
--evnode.signer.passphrase
```

## 5. Initialize and Run

```bash
# Initialize (same as before)
./appd init mynode --chain-id mychain-1

# Start with ev-abci
./appd start \
  --evnode.node.aggregator \
  --evnode.da.address http://localhost:7980 \
  --evnode.signer.passphrase secret
```

## Key Differences from CometBFT

| Aspect | CometBFT | ev-abci |
|--------|----------|---------|
| Validators | Multiple validators with staking | Single sequencer |
| Consensus | BFT consensus rounds | Sequencer produces blocks |
| Finality | Instant (BFT) | Soft (P2P) → Hard (DA) |
| Block time | ~6s typical | Configurable (100ms+) |

## Next Steps

- [Migration Guide](/getting-started/cosmos/migration-guide) — Migrate existing chain with state
- [ev-abci Overview](/ev-abci/overview) — Architecture details
