# Introduction

Evolve is the fastest way to launch your own modular network — without validator overhead or token lock-in.

Built on Celestia, Evolve offers L1-level control with L2-level performance.

This isn't a toolkit. It's a launch stack.

No fees. No middlemen. No revenue share.

## What is Evolve?

Evolve is a launch stack for L1s. It gives you full control over execution — without CometBFT, validator ops, or lock-in.

It's [open-source](https://github.com/evstack/ev-node), production-ready, and fully composable.

At its core is \`ev-node\`, a modular node that exposes an [Execution interface](https://github.com/evstack/ev-node/blob/main/core/execution/execution.go), — letting you bring any VM or execution logic, including Cosmos SDK or custom-built runtimes.

Evolving from Cosmos SDK?

Migrate without rewriting your stack. Bring your logic and state to Evolve and shed validator overhead — all while gaining performance and execution freedom.

Evolve is how you launch your network. Modular. Production-ready. Yours.

With Evolve, you get:

- Full control over execution \- use any VM
- Low-cost launch — no emissions, no validator inflation
- Speed to traction — from local devnet to testnet in minutes
- Keep sequencer revenue — monetize directly
- Optional L1 validator network for fast finality and staking

Powered by Celestia — toward 1GB blocks, multi-VM freedom, and execution without compromising flexibility or cost.

## What problems is Evolve solving?

### 1\. Scalability and customizability

Deploying your decentralized application as a smart contract on a shared blockchain has many limitations. Your smart contract has to share computational resources with every other application, so scalability is limited.

Plus, you're restricted to the execution environment that the shared blockchain uses, so developer flexibility is limited as well.

### 2\. Security and time to market

Deploying a new chain might sound like the perfect solution for the problems listed above. While it's somewhat true, deploying a new layer 1 chain presents a complex set of challenges and trade-offs for developers looking to build blockchain products.

Deploying a legacy layer 1 has huge barriers to entry: time, capital, token emissions and expertise.

In order to secure the network, developers must bootstrap a sufficiently secure set of validators, incurring the overhead of managing a full consensus network. This requires paying validators with inflationary tokens, putting the network's business sustainability at risk. Network effects are also critical for success, but can be challenging to achieve as the network must gain widespread adoption to be secure and valuable.

In a potential future with millions of chains, it's unlikely all of those chains will be able to sustainably attract a sufficiently secure and decentralized validator set.

## Why Evolve?

Evolve solves the challenges encountered during the deployment of a smart contract or a new layer 1, by minimizing these tradeoffs through the implementation of evolve chains.

With Evolve, developers can benefit from:

- **Shared security**: Chains inherit security from a data availability layer, by posting blocks to it. Chains reduce the trust assumptions placed on chain sequencers by allowing full nodes to download and verify the transactions in the blocks posted by the sequencer. For optimistic or zk-chains, in case of fraudulent blocks, full nodes can generate fraud or zk-proofs, which they can share with the rest of the network, including light nodes. Our roadmap includes the ability for light clients to receive and verify proofs, so that everyday users can enjoy high security guarantees.

- **Scalability:** Evolve chains are deployed on specialized data availability layers like Celestia, which directly leverages the scalability of the DA layer. Additionally, chain transactions are executed off-chain rather than on the data availability layer. This means chains have their own dedicated computational resources, rather than sharing computational resources with other applications.

- **Customizability:** Evolve is built as an open source modular framework, to make it easier for developers to reuse the four main components and customize their chains. These components are data availability layers, execution environments, proof systems, and sequencer schemes.

- **Faster time-to-market:** Evolve eliminates the need to bootstrap a validator set, manage a consensus network, incur high economic costs, and face other trade-offs that come with deploying a legacy layer 1\. Evolve's goal is to make deploying a chain as easy as it is to deploy a smart contract, cutting the time it takes to bring blockchain products to market from months (or even years) to just minutes.

- **Sovereignty**: Evolve also enables developers to deploy chains for cases where communities require sovereignty.

## How can you use Evolve?

As briefly mentioned above, Evolve could be used in many different ways. From chains, to settlement layers, and in the future even to L3s.

### Chain with any VM

Evolve gives developers the flexibility to use pre-existing ABCI-compatible state machines or create a custom state machine tailored to their chain needs. Evolve does not restrict the use of any specific virtual machine, allowing developers to experiment and bring innovative applications to life.

### Cosmos SDK

Similarly to how developers utilize the Cosmos SDK to build a layer 1 chain, the Cosmos SDK could be utilized to create a Evolve-compatible chain. Cosmos-SDK has great [documentation](https://docs.cosmos.network/main) and tooling that developers can leverage to learn.

Another possibility is taking an existing layer 1 built with the Cosmos SDK and deploying it as a Evolve chain. Evolve gives your network a forward path. Migrate seamlessly, keep your logic, and evolve into a modular, high-performance system without CometBFT bottlenecks and zero validator overhead.

### Build a settlement layer

[Settlement layers](https://celestia.org/learn/modular-settlement-layers/settlement-in-the-modular-stack/) are ideal for developers who want to avoid deploying chains. They provide a platform for chains to verify proofs and resolve disputes. Additionally, they act as a hub for chains to facilitate trust-minimized token transfers and liquidity sharing between chains that share the same settlement layer. Think of settlement layers as a special type of execution layer.

## When can you use Evolve?

As of today, Evolve provides a single sequencer, an execution interface (Engine API or ABCI), and a connection to Celestia.

We're currently working on implementing many new and exciting features such as light nodes and state fraud proofs.

Head down to the next section to learn more about what's coming for Evolve. If you're ready to start building, you can skip to the [Guides](/guides/quick-start.md) section.

Spoiler alert, whichever you choose, it's going to be a great rabbit hole\!
