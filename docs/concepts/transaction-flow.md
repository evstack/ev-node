# Transaction flow

Chain users use a light node to communicate with the chain P2P network for two primary reasons:

- submitting transactions
- gossiping headers and fraud proofs

Here's what the typical transaction flow looks like:

## Transaction submission

```mermaid
sequenceDiagram
    participant User
    participant LightNode
    participant FullNode

    User->>LightNode: Submit Transaction
    LightNode->>FullNode: Gossip Transaction
    FullNode-->>User: Refuse (if invalid)
```

## Transaction validation and processing

```mermaid
sequenceDiagram
    participant FullNode
    participant Sequencer

    FullNode->>FullNode: Check Validity
    FullNode->>FullNode: Add to Mempool (if valid)
    FullNode-->>User: Transaction Processed (if valid)
    FullNode->>Sequencer: Inform about Valid Transaction
    Sequencer->>DALayer: Add to Chain Block
```

## Block processing

```mermaid
sequenceDiagram
    participant DALayer
    participant FullNode
    participant Chain

    DALayer->>Chain: Update State
    DALayer->>FullNode: Download & Validate Block
```

To transact, users submit a transaction to their light node, which gossips the transaction to a full node. Before adding the transaction to their mempool, the full node checks its validity. Valid transactions are included in the mempool, while invalid ones are refused, and the user's transaction will not be processed.

If the transaction is valid and has been included in the mempool, the sequencer can add it to a chain block, which is then submitted to the data availability (DA) layer. This results in a successful transaction flow for the user, and the state of the chain is updated accordingly.

After the block is submitted to the DA layer, the full nodes download and validate the block.
