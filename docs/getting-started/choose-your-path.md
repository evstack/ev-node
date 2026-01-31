# Choose Your Path

Evolve supports three execution environments. Your choice depends on your existing codebase, target users, and development resources.

## Quick Comparison

|                      | EVM (ev-reth)                      | Cosmos SDK (ev-abci)        | Custom Executor     |
|----------------------|------------------------------------|-----------------------------|---------------------|
| **Best for**         | New chains, DeFi, NFTs             | Existing Cosmos chains      | Novel VMs, research |
| **Language**         | Solidity, Vyper                    | Go                          | Any                 |
| **Wallet support**   | MetaMask, Rainbow, all EVM wallets | Keplr, Leap, Cosmos wallets | Build your own      |
| **Block explorer**   | Blockscout, any EVM explorer       | Mintscan, Ping.pub          | Build your own      |
| **Tooling maturity** | Excellent                          | Good                        | None                |
| **Setup complexity** | Low                                | Medium                      | High                |
| **Migration path**   | Deploy existing contracts          | Migrate existing chain      | N/A                 |

## EVM (ev-reth)

Use ev-reth if you want Ethereum compatibility.

### Pros

- **Wallet ecosystem** — MetaMask, Rainbow, Rabby, and every EVM wallet works out of the box. Users don't need new software.
- **Developer tooling** — Foundry, Hardhat, Remix, Tenderly, and the entire Ethereum toolchain works unchanged.
- **Contract portability** — Deploy existing Solidity/Vyper contracts without modification.
- **Block explorers** — Blockscout, Etherscan-compatible APIs, and standard indexers work immediately.
- **RPC compatibility** — Standard Ethereum JSON-RPC means existing frontend code works.

### Cons

- **EVM constraints** — Bound by EVM gas model and execution semantics.

### When to choose EVM

- Building a new chain and want maximum user/developer reach
- Need access to EVM DeFi tooling (Uniswap, lending protocols, etc.)
- Want users to connect with wallets they already have

**→ [EVM Quickstart](/getting-started/evm/quickstart)**

## Cosmos SDK (ev-abci)

Use ev-abci if you have an existing Cosmos chain or want Cosmos SDK modules.

### Pros

- **Migration path** — Existing Cosmos SDK chains can migrate without rewriting application logic.
- **Cosmos tooling** — Ignite CLI, Cosmos SDK modules, and familiar Go development.
- **Custom modules** — Build application-specific logic beyond what smart contracts allow.
- **Established wallets** — Keplr, Leap, and Cosmos wallets have strong user bases.

### Cons

- **Smaller wallet ecosystem** — Fewer wallets than EVM, though major ones are well-supported.
- **Migration complexity** — Moving from CometBFT requires careful migration.
- **Different mental model** — Cosmos SDK modules differ significantly from smart contracts.

### When to choose Cosmos SDK

- Have an existing Cosmos SDK chain running on CometBFT
- Want to shed validator overhead while keeping your application logic
- Prefer Go over Solidity for application development

**→ [Cosmos SDK Quickstart](/getting-started/cosmos/quickstart)**

## Custom Executor

Use a custom executor if you need something neither EVM nor Cosmos SDK provides.

### Pros

- **Maximum flexibility** — Implement any state machine, any VM, any execution model.
- **Performance optimization** — Tailor execution to your specific use case.
- **Novel designs** — Build zkVMs, specialized rollups, or research prototypes.

### Cons

- **No wallet support** — You must build or integrate wallet connectivity.
- **No tooling** — No block explorers, no development frameworks, no debugging tools.
- **High development cost** — Everything beyond ev-node itself is your responsibility.
- **No ecosystem** — Users and developers must learn your custom environment.

### When to choose Custom

- Building a novel VM (zkVM, MoveVM, etc.)
- Research or experimental chains
- Highly specialized state machines (gaming, specific financial instruments)
- Have resources to build full tooling stack

**→ [Custom Executor Quickstart](/getting-started/custom/quickstart)**

## Decision Tree

```
Do you have an existing Cosmos SDK chain?
├── Yes → Cosmos SDK (ev-abci)
└── No
    │
    Do you need a custom VM or non-standard execution?
    ├── Yes → Custom Executor
    └── No
        │
        Do you want maximum wallet/tooling support?
        ├── Yes → EVM (ev-reth)
        └── No
            │
            Do you prefer Go over Solidity?
            ├── Yes → Cosmos SDK (ev-abci)
            └── No → EVM (ev-reth)
```

## Switching Later

- **EVM → Cosmos SDK**: Not practical. Different execution models, would require chain restart.
- **Cosmos SDK → EVM**: Not practical. Same reason.
- **Custom → Either**: Possible if you design for it, but significant work.

Choose based on your long-term needs. The execution environment is a foundational decision.
