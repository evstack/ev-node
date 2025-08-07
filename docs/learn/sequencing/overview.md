# Sequencing

 Sequencing is the essential first step for handling your transactions. Think of it as an organizer that takes all incoming transactions, puts them in a clear order, and then groups them into batches. This process is vital for keeping everything consistent and making the chain run. Evolve uses a "Sequencing Interface" with key functions like submitting, retrieving, and verifying these transaction batches, ensuring smooth communication between the chain and the sequencing mechanism, which often acts as a bridge to the underlying network.

## Sequencing Interface {#sequencing-interface}

[Sequencing Interface](https://github.com/evstack/ev-node/blob/main/core/sequencer/sequencing.go#L11) defines a sequencing interface for communicating between any sequencing network and Evolve. The key functions of the interface are defined as shown below.

```go
// Sequencer is a generic interface for a sequencer
type Sequencer interface {
 // SubmitBatchTxs submits a batch of transactions from  to sequencer
 // Id is the unique identifier for the  chain
 // Batch is the batch of transactions to submit
 // returns an error if any from the sequencer
 SubmitBatchTxs(ctx context.Context, req SubmitBatchTxsRequest) (*SubmitBatchTxsResponse, error)

 // GetNextBatch returns the next batch of transactions from sequencer to
 // Id is the unique identifier for the  chain
 // LastBatchHash is the cryptographic hash of the last batch received by the
 // MaxBytes is the maximum number of bytes to return in the batch
 // returns the next batch of transactions and an error if any from the sequencer
 GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error)

 // VerifyBatch verifies a batch of transactions received from the sequencer
 // Id is the unique identifier for the  chain
 // BatchHash is the cryptographic hash of the batch to verify
 // returns a boolean indicating if the batch is valid and an error if any from the sequencer
 VerifyBatch(ctx context.Context, req VerifyBatchResponse) (*VerifyBatchResponse, error)
}
```

It mainly consists of:

* `SubmitBatchTxs` relays the chain transactions from Evolve chain to the sequencing network
* `GetNextBatch` returns the next batch of transactions along with a deterministic timestamp
* `VerifyBatch` validates the sequenced batch

## Sequencing Implementations {#sequencing-implementations}

An implementation of the sequencing interface mainly acts as a middleware that connects Evolve chain and the sequencing layer. It implements the sequencing interface functions described above.
There are several implementations of the sequencer but for now only one is available in Evolve.

* [single-sequencer](/learn/sequencing/single.md) - The simplest and most widely used sequencing model, where a single node (the sequencer) is responsible for ordering transactions and producing blocks.
