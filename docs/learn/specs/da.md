# DA

Evolve provides a generic [data availability interface][da-interface] for modular blockchains. Any DA that implements this interface can be used with Evolve.

## Details

`Client` can connect via JSON-RPC transports using Evolve's [jsonrpc][jsonrpc] implementations. The connection can be configured using the following cli flags:

* `--evolve.da.address`: url address of the DA service (default: "grpc://localhost:26650")
* `--evolve.da.auth_token`: authentication token of the DA service
* `--evolve.da.namespace`: namespace to use when submitting blobs to the DA service

Given a set of blocks to be submitted to DA by the block manager, the `SubmitBlocks` first encodes the blocks using protobuf (the encoded data are called blobs) and invokes the `Submit` method on the underlying DA implementation. On successful submission (`StatusSuccess`), the DA block height which included in the blocks is returned.

To make sure that the serialised blocks don't exceed the underlying DA's blob limits, it fetches the blob size limit by calling `Config` which returns the limit as `uint64` bytes, then includes serialised blocks until the limit is reached. If the limit is reached, it submits the partial set and returns the count of successfully submitted blocks as `SubmittedCount`. The caller should retry with the remaining blocks until all the blocks are submitted. If the first block itself is over the limit, it throws an error.

The `Submit` call may result in an error (`StatusError`) based on the underlying DA implementations on following scenarios:

* the total blobs size exceeds the underlying DA's limits (includes empty blobs)
* the implementation specific failures, e.g., for [celestia-da-json-rpc][jsonrpc], invalid namespace, unable to create the commitment or proof, setting low gas price, etc, could return error.

The `RetrieveBlocks` retrieves the blocks for a given DA height using `GetIDs` and `Get` methods. If there are no blocks available for a given DA height, `StatusNotFound` is returned (which is not an error case). The retrieved blobs are converted back to blocks and returned on successful retrieval.

Both `SubmitBlocks` and `RetrieveBlocks` may be unsuccessful if the DA node and the DA blockchain that the DA implementation is using have failures. For example, failures such as, DA mempool is full, DA submit transaction is nonce clashing with other transaction from the DA submitter account, DA node is not synced, etc.

## References

[1] [da-interface][da-interface]

[2] [jsonrpc][jsonrpc]

[da-interface]: https://github.com/evstack/ev-node/blob/main/core/da/da.go#L11
[jsonrpc]: https://github.com/evstack/ev-node/tree/main/da/jsonrpc
