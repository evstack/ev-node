# Sequencing

Sequencing is the process of determining the order of transactions in a blockchain. In rollups, the sequencer is the entity responsible for collecting transactions from users, ordering them, and producing blocks that are eventually posted to the data availability (DA) layer.

Transaction ordering matters because it determines execution outcomes. Two transactions that touch the same state can produce different results depending on which executes first. The sequencer's ordering decisions directly impact users, particularly in DeFi where transaction order can mean the difference between a successful trade and a failed one.

## The Role of the Sequencer

A sequencer performs three core functions:

1. **Transaction collection** — Accepting transactions from users and holding them in a mempool
2. **Ordering** — Deciding which transactions to include and in what order
3. **Block production** — Bundling ordered transactions into blocks and publishing them

In traditional L1 blockchains, these functions are distributed across validators through consensus. In rollups, sequencing can be handled differently depending on the design goals.

## Single Sequencer

The simplest approach is a single sequencer: one designated node that orders all transactions.

```
User → Sequencer → Block → DA Layer
```

**Advantages:**

- **Low latency** — No consensus required means block times can be very fast (sub-second)
- **Simple operation** — One node, one source of truth for ordering
- **Predictable performance** — No coordination overhead

**Disadvantages:**

- **Centralization** — Single point of control over transaction ordering
- **Censorship risk** — The sequencer can refuse to include specific transactions
- **Liveness dependency** — If the sequencer goes down, the chain halts
- **MEV extraction** — The sequencer has full visibility and can reorder for profit

Most production rollups today use single sequencers because the performance benefits are significant and the trust assumptions are often acceptable for their use cases.

## Based Sequencing

Based sequencing (also called "based rollups") delegates transaction ordering to the underlying DA layer. Instead of a dedicated sequencer, users submit transactions directly to the DA layer, and all rollup nodes independently derive the same ordering from DA blocks.

```
User → DA Layer → All Nodes Derive Same Order
```

**Advantages:**

- **Decentralization** — No privileged sequencer role
- **Censorship resistance** — Inherits the censorship resistance of the DA layer
- **Liveness** — Chain stays live as long as the DA layer is live
- **Shared security** — Ordering is secured by the DA layer's consensus

**Disadvantages:**

- **Higher latency** — Block times are bounded by DA layer block times (e.g., ~12s for Ethereum)
- **MEV leakage** — MEV flows to DA layer validators rather than the rollup
- **Complexity** — Requires deterministic derivation rules that all nodes must follow

Based sequencing is compelling for applications that prioritize decentralization over speed.

## Hybrid Approaches

### Forced Inclusion

Forced inclusion is a mechanism that combines the performance of single sequencing with censorship resistance guarantees. It works as follows:

1. Users normally submit transactions to the sequencer for fast inclusion
2. If censored, users can submit transactions directly to the DA layer
3. The sequencer must include DA-submitted transactions within a defined time window
4. Failure to include triggers penalties or allows the chain to transition to based mode

This gives users an escape hatch while maintaining the benefits of centralized sequencing for the common case.

### Shared Sequencing

Multiple rollups can share a sequencer or sequencer network. This enables:

- **Atomic cross-rollup transactions** — Transactions that span multiple rollups can be ordered atomically
- **Shared MEV** — Revenue from cross-rollup MEV can be distributed
- **Reduced costs** — Infrastructure costs are amortized across chains

Shared sequencing is an active area of research and development.

## MEV Considerations

Maximal Extractable Value (MEV) is the profit a sequencer can extract by reordering, inserting, or censoring transactions. Common MEV strategies include:

- **Frontrunning** — Inserting a transaction before a target transaction
- **Backrunning** — Inserting a transaction immediately after a target
- **Sandwich attacks** — Combining frontrunning and backrunning around a target

The sequencing design determines who captures MEV:

| Design            | MEV Captured By          |
|-------------------|--------------------------|
| Single sequencer  | Sequencer operator       |
| Based sequencing  | DA layer validators      |
| Shared sequencing | Shared sequencer network |

Some rollups implement MEV mitigation through encrypted mempools, fair ordering protocols, or MEV redistribution to users.

## Choosing a Sequencing Model

| Factor                 | Single Sequencer          | Based Sequencer     |
|------------------------|---------------------------|---------------------|
| Block time             | Sub-second possible       | DA layer block time |
| Censorship resistance  | Requires forced inclusion | Native              |
| Liveness               | Sequencer must be online  | DA layer liveness   |
| MEV control            | Sequencer controlled      | DA layer controlled |
| Operational complexity | Lower                     | Higher              |

The right choice depends on your application's priorities. High-frequency trading applications might prefer single sequencing for speed. Applications handling high-value, censorship-sensitive transactions might prefer based sequencing for its guarantees.

## Learn More

- [Forced Inclusion](/guides/advanced/forced-inclusion) — Implementing censorship resistance with single sequencing
- [Based Sequencing](/guides/advanced/based-sequencing) — Running a based rollup
- [Sequencer Interface](/reference/interfaces/sequencer) — Implementation reference
