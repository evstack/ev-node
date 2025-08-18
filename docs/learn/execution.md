# Execution Layers in Evolve

Evolve is designed to be modular and flexible, allowing different execution layers to be plugged in. Evolve defines a general-purpose execution interface ([see execution.go](https://github.com/evstack/ev-node/blob/main/core/execution/execution.go)) that enables developers to integrate any compatible application as the chain's execution layer.

This means you can use a variety of Cosmos SDK or Reth compatible applications as the execution environment for your chain: choose the execution environment that best fits your use case.

## Supported Execution Layers

### Cosmos SDK Execution Layer

Evolve natively supports Cosmos SDK-based applications as the execution layer for a chain via the ABCI (Application Blockchain Interface) protocol. The Cosmos SDK provides a rich set of modules for staking, governance, IBC, and more, and is widely used in the Cosmos ecosystem. This integration allows developers to leverage the full power and flexibility of the Cosmos SDK when building their chain applications.

- [Cosmos SDK Documentation](https://docs.cosmos.network/)
- [Cosmos SDK ABCI Documentation](https://docs.cosmos.network/main/build/abci/introduction)
- [Evolve ABCI Adapter](https://github.com/evstack/ev-abci)

### Reth

Reth is a high-performance Ethereum execution client written in Rust. Evolve can integrate Reth as an execution layer, enabling Ethereum-compatible chains to process EVM transactions and maintain Ethereum-like state. This allows developers to build chains that leverage the Ethereum ecosystem, tooling, and smart contracts, while benefiting from Evolve's modular consensus and data availability.

For more information about Reth, see the official documentation:

- [Reth GitHub Repository](https://github.com/paradigmxyz/reth)
- [Evolve Reth Integration](https://github.com/evstack/ev-reth)

## How It Works

- Evolve acts as the consensus and uses Celestia as its data availability layer.
- The execution layer (Cosmos SDK app or Reth) processes transactions and maintains application state.

For more details on integrating an execution layer with Evolve, see the respective documentation links above.
