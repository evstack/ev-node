# DA

Rollkit provides a wrapper for [go-da][go-da], a generic data availability interface for modular blockchains, called `DAClient` with wrapper functionalities like `SubmitBlocks` and `RetrieveBlocks` to help block manager interact with DA more easily.

## Details

`DAClient` can connect via either gRPC or JSON-RPC transports using the [go-da][go-da] [proxy/grpc][proxy/grpc] or [proxy/jsonrpc][proxy/jsonrpc] implementations. The connection can be configured using the following cli flags:

* `--rollkit.da_address`: url address of the DA service (default: "grpc://localhost:26650")
* `--rollkit.da_auth_token`: authentication token of the DA service
* `--rollkit.da_namespace`: namespace to use when submitting blobs to the DA service

Given a set of blocks to be submitted to DA by the block manager, the `SubmitBlocks` first encodes the blocks using protobuf (the encoded data are called blobs) and invokes the `Submit` method on the underlying DA implementation. On successful submission (`StatusSuccess`), the DA block height which included in the blocks is returned.

To make sure that the serialised blocks don't exceed the underlying DA's blob limits, it fetches the blob size limit by calling `Config` which returns the limit as `uint64` bytes, then includes serialised blocks until the limit is reached. If the limit is reached, it submits the partial set and returns the count of successfully submitted blocks as `SubmittedCount`. The caller should retry with the remaining blocks until all the blocks are submitted. If the first block itself is over the limit, it throws an error.

The `Submit` call may result in an error (`StatusError`) based on the underlying DA implementations on following scenarios:

* the total blobs size exceeds the underlying DA's limits (includes empty blobs)
* the implementation specific failures, e.g., for [celestia-da-json-rpc][proxy/jsonrpc], invalid namespace, unable to create the commitment or proof, setting low gas price, etc, could return error.

The `RetrieveBlocks` retrieves the blocks for a given DA height using [go-da][go-da] `GetIDs` and `Get` methods. If there are no blocks available for a given DA height, `StatusNotFound` is returned (which is not an error case). The retrieved blobs are converted back to blocks and returned on successful retrieval.

Both `SubmitBlocks` and `RetrieveBlocks` may be unsuccessful if the DA node and the DA blockchain that the DA implementation is using have failures. For example, failures such as, DA mempool is full, DA submit transaction is nonce clashing with other transaction from the DA submitter account, DA node is not synced, etc.

## Implementation

See [da implementation]

## References

[1] [go-da][go-da]

[2] [proxy/grpc][proxy/grpc]

[3] [proxy/jsonrpc][proxy/jsonrpc]

[da implementation]: https://github.com/rollkit/rollkit/blob/main/da/jsonrpc/client.go
[go-da]: https://github.com/rollkit/go-da
[proxy/grpc]: https://github.com/rollkit/go-da/tree/main/proxy/grpc
[proxy/jsonrpc]: https://github.com/rollkit/go-da/tree/main/proxy/jsonrpc
