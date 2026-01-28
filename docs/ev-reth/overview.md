# ev-reth Overview

ev-reth is a modified [reth](https://github.com/paradigmxyz/reth) Ethereum execution client optimized for Evolve rollups.

## What is ev-reth?

ev-reth extends reth with:

- **Engine API integration** — Driven by ev-node for block production
- **Rollup-specific features** — Base fee redirect, deploy allowlist, custom precompiles
- **Configurable chain parameters** — Contract size limits, custom gas settings

## Architecture

```
┌─────────────────────────────────────────┐
│               ev-node                    │
│         (consensus + DA + P2P)          │
└─────────────────┬───────────────────────┘
                  │ Engine API
                  │ (JWT authenticated)
┌─────────────────▼───────────────────────┐
│               ev-reth                    │
│           (EVM execution)               │
│  ┌───────────┐  ┌───────────────────┐   │
│  │ State DB  │  │ Transaction Pool  │   │
│  └───────────┘  └───────────────────┘   │
│  ┌───────────────────────────────────┐  │
│  │          EVM + Precompiles        │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

ev-node drives ev-reth through the Engine API:
1. ev-node calls `engine_forkchoiceUpdated` with payload attributes
2. ev-reth builds a block from pending transactions
3. ev-node calls `engine_getPayload` to retrieve the block
4. ev-node broadcasts and submits to DA
5. ev-node calls `engine_newPayload` to finalize

## Features

| Feature | Description |
|---------|-------------|
| [Base Fee Redirect](/ev-reth/features/base-fee-redirect) | Send base fees to treasury instead of burning |
| [Deploy Allowlist](/ev-reth/features/deploy-allowlist) | Restrict who can deploy contracts |
| [Contract Size Limits](/ev-reth/features/contract-size-limits) | Increase max contract size beyond 24KB |
| [Mint Precompile](/ev-reth/features/mint-precompile) | Native token minting for bridges |

## When to Use ev-reth

Use ev-reth when you want:

- Full EVM compatibility
- Ethereum tooling (Foundry, Hardhat, etc.)
- Standard wallet support (MetaMask, etc.)
- High-performance Rust execution

## Repository

- GitHub: [github.com/evstack/ev-reth](https://github.com/evstack/ev-reth)
- Based on: [paradigmxyz/reth](https://github.com/paradigmxyz/reth)

## Next Steps

- [EVM Quickstart](/getting-started/evm/quickstart) — Get started
- [Configuration](/ev-reth/configuration) — Chainspec and settings
- [Engine API](/ev-reth/engine-api) — How ev-node communicates with ev-reth
