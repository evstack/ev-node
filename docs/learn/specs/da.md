# DA

Evolve uses Celestia as its data availability layer through the [data availability interface][da-interface].

## Details

The Celestia DA client connects directly to a Celestia node using the blob API. The connection can be configured using the following cli flags:

* `--rollkit.da.address`: url address of the DA service (default: "grpc://localhost:26650")
* `--rollkit.da.auth_token`: authentication token of the DA service
* `--rollkit.da.namespace`: namespace to use when submitting blobs to the DA service (deprecated)
* `--rollkit.da.header_namespace`: namespace to use when submitting headers to the DA service
* `--rollkit.da.data_namespace`: namespace to use when submitting data to the DA service

The Submitter component now submits headers and data separately to the DA layer using different namespaces:

* **Headers**: Submitted to the namespace specified by `--rollkit.da.header_namespace` (or falls back to `--rollkit.da.namespace` if not set)
* **Data**: Submitted to the namespace specified by `--rollkit.da.data_namespace` (or falls back to `--rollkit.da.namespace` if not set)

Each submission first encodes the headers or data using protobuf (the encoded data are called blobs) and invokes the `Submit` method on the underlying DA implementation with the appropriate namespace. On successful submission (`StatusSuccess`), the DA block height which included the blobs is returned.

To make sure that the serialised blocks don't exceed the underlying DA's blob limits, it fetches the blob size limit by calling `Config` which returns the limit as `uint64` bytes, then includes serialised blocks until the limit is reached. If the limit is reached, it submits the partial set and returns the count of successfully submitted blocks as `SubmittedCount`. The caller should retry with the remaining blocks until all the blocks are submitted. If the first block itself is over the limit, it throws an error.

The `Submit` call may result in an error (`StatusError`) in the following scenarios:

* the total blobs size exceeds Celestia's blob size limits (includes empty blobs)
* Celestia-specific failures, e.g., invalid namespace, unable to create the commitment or proof, setting low gas price, etc.

The retrieval process now supports both legacy single-namespace mode and separate namespace mode:

1. **Legacy Mode Support**: For backward compatibility, the system first attempts to retrieve from the legacy namespace if migration has not been completed.

2. **Separate Namespace Retrieval**: The system retrieves headers and data separately:
   * Headers are retrieved from the `HeaderNamespace`
   * Data is retrieved from the `DataNamespace`
   * Results from both namespaces are combined

3. **Namespace Migration**: The system automatically detects and tracks namespace migration:
   * When data is found in new namespaces, migration is marked as complete
   * Migration state is persisted to optimize future retrievals
   * Once migration is complete, legacy namespace checks are skipped

If there are no blocks available for a given DA height in any namespace, `StatusNotFound` is returned (which is not an error case). The retrieved blobs are converted back to headers and data, then combined into complete blocks for processing.

Both header/data submission and retrieval operations may be unsuccessful if the Celestia node or the Celestia network have failures. For example, mempool is full, transaction nonce conflicts, node is not synced, etc.

## Namespace Separation Benefits

The separation of headers and data into different namespaces provides several advantages:

* **Improved Scalability**: Headers and data can be processed independently, allowing for more efficient resource utilization
* **Flexible Data Availability**: Different availability guarantees can be applied to headers vs data
* **Optimized Retrieval**: Clients can retrieve only the data they need (e.g., light clients may only need headers)
* **Backward Compatibility**: The system maintains support for legacy single-namespace deployments while enabling gradual migration

## References

[1] [da-interface][da-interface]

[da-interface]: https://github.com/evstack/ev-node/blob/main/core/da/da.go#L11
