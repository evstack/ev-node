# Integration Guide

Integrate ev-abci into a Cosmos SDK application.

## Overview

ev-abci replaces CometBFT as the consensus layer. Your ABCI application logic remains unchanged—only the node startup code changes.

## Prerequisites

- Cosmos SDK v0.50+ application
- Go 1.22+

## Step 1: Add Dependency

```bash
go get github.com/evstack/ev-abci@latest
```

## Step 2: Modify Start Command

Locate your app's entrypoint (typically `cmd/<appd>/root.go` or `main.go`).

### Before (CometBFT)

```go
import (
    "github.com/cosmos/cosmos-sdk/server"
)

// In your root command setup:
server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport)
```

### After (ev-abci)

```go
import (
    "github.com/cosmos/cosmos-sdk/server"
    evabci "github.com/evstack/ev-abci/server"
)

// Keep existing commands for init, genesis, keys, etc.
server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport)

// Replace the start command
startCmd := &cobra.Command{
    Use:   "start",
    Short: "Run the node",
    RunE: func(cmd *cobra.Command, _ []string) error {
        return evabci.StartHandler(cmd, newApp)
    },
}
evabci.AddFlags(startCmd)
rootCmd.AddCommand(startCmd)
```

## Step 3: Build

```bash
go build -o appd ./cmd/appd
```

## Step 4: Verify

Check for ev-abci flags:

```bash
./appd start --help
```

Expected flags:

```
--evnode.node.aggregator     Run as block producer
--evnode.da.address          DA layer address
--evnode.signer.passphrase   Signer passphrase
--evnode.node.block_time     Block production interval
```

## Step 5: Initialize

Standard Cosmos SDK initialization:

```bash
./appd init mynode --chain-id mychain-1
./appd keys add mykey --keyring-backend test
./appd genesis add-genesis-account mykey 1000000000stake --keyring-backend test
./appd genesis gentx mykey 1000000stake --chain-id mychain-1 --keyring-backend test
./appd genesis collect-gentxs
```

## Step 6: Start

```bash
./appd start \
  --evnode.node.aggregator \
  --evnode.da.address http://localhost:7980 \
  --evnode.signer.passphrase secret
```

## Configuration

### ev-node Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--evnode.node.aggregator` | Run as sequencer | `false` |
| `--evnode.node.block_time` | Block interval | `1s` |
| `--evnode.da.address` | DA layer URL | required |
| `--evnode.signer.passphrase` | Signer passphrase | required |
| `--evnode.p2p.peers` | P2P peer addresses | none |

### Full Node (Non-Sequencer)

```bash
./appd start \
  --evnode.da.address http://localhost:7980 \
  --evnode.p2p.peers <SEQUENCER_P2P_ID>@<HOST>:26659
```

## RPC Compatibility

ev-abci provides CometBFT-compatible RPC endpoints. Existing clients work without modification.

See [RPC Compatibility](/ev-abci/rpc-compatibility) for details.

## Next Steps

- [Migration from CometBFT](/ev-abci/migration-from-cometbft) — Migrate existing chain
- [RPC Compatibility](/ev-abci/rpc-compatibility) — Endpoint compatibility
