# DA Interface

The DA (Data Availability) interface defines how ev-node submits and retrieves data from the DA layer.

This page is an overview. The source of truth for exact interfaces is the current DA interface implementation in this repository.

## Client Interface

```go
type Client interface {
    Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) ResultSubmit
    Retrieve(ctx context.Context, height uint64, namespace []byte) ResultRetrieve
    Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)
    GetHeaderNamespace() []byte
    GetDataNamespace() []byte
    GetForcedInclusionNamespace() []byte
    HasForcedInclusionNamespace() bool
}
```

## Methods

### Submit

Submits blobs to the DA layer.

```go
Submit(ctx context.Context, data [][]byte, gasPrice float64, namespace []byte, options []byte) ResultSubmit
```

**Parameters:**

- `data` - Blobs to submit
- `gasPrice` - DA layer gas price
- `namespace` - Target namespace
- `options` - DA-specific options (JSON encoded)

**Returns:**

```go
type ResultSubmit struct {
    BaseResult
}
```

### Retrieve

Retrieves all blobs at a DA height and namespace.

```go
Retrieve(ctx context.Context, height uint64, namespace []byte) ResultRetrieve
```

**Returns:**

```go
type ResultRetrieve struct {
    BaseResult
    Data [][]byte  // Retrieved blobs
}
```

### Get

Retrieves specific blobs by their IDs.

```go
Get(ctx context.Context, ids []ID, namespace []byte) ([]Blob, error)
```

### Namespace Accessors

```go
GetHeaderNamespace() []byte           // Namespace for block headers
GetDataNamespace() []byte             // Namespace for block data
GetForcedInclusionNamespace() []byte  // Namespace for forced inclusion txs
HasForcedInclusionNamespace() bool    // Whether forced inclusion is enabled
```

## Verifier Interface

For sequencers that need to verify batch inclusion:

```go
type Verifier interface {
    GetProofs(ctx context.Context, ids []ID, namespace []byte) ([]Proof, error)
    Validate(ctx context.Context, ids []ID, proofs []Proof, namespace []byte) ([]bool, error)
}
```

## FullClient Interface

Combines Client and Verifier:

```go
type FullClient interface {
    Client
    Verifier
}
```

## Types

### Core Types

```go
type Blob = []byte        // Raw data
type ID = []byte          // Blob identifier (height + commitment)
type Commitment = []byte  // Cryptographic commitment
type Proof = []byte       // Inclusion proof
```

### BaseResult

Common fields for DA operations:

```go
type BaseResult struct {
    Code           StatusCode
    Message        string
    Height         uint64
    SubmittedCount uint64
    BlobSize       uint64
    IDs            [][]byte
    Timestamp      time.Time
}
```

### Status Codes

```go
const (
    StatusUnknown StatusCode = iota
    StatusSuccess
    StatusNotFound
    StatusNotIncludedInBlock
    StatusAlreadyInMempool
    StatusTooBig
    StatusContextDeadline
    StatusError
    StatusIncorrectAccountSequence
    StatusContextCanceled
    StatusHeightFromFuture
)
```

## ID Format

IDs encode both height and commitment:

```go
// ID = height (8 bytes, little-endian) + commitment
func SplitID(id []byte) (height uint64, commitment []byte, error)
```

## Namespaces

DA uses 29-byte namespaces (Celestia format):

- 1 byte version
- 28 bytes identifier

Three namespaces are used:

| Namespace        | Purpose                                 |
|------------------|-----------------------------------------|
| Header           | Block headers                           |
| Data             | Transaction data                        |
| Forced Inclusion | User-submitted censorship-resistant txs |

## Implementations

| Implementation | Package           | Description         |
|----------------|-------------------|---------------------|
| Celestia       | `pkg/da/celestia` | Production DA layer |
| Local DA       | `pkg/da/local`    | Development/testing |

## Configuration

```bash
# Celestia
--evnode.da.address http://localhost:26658
--evnode.da.auth_token <token>
--evnode.da.namespace <hex>
--evnode.da.gas_price 0.01

# Local DA
--evnode.da.address http://localhost:7980
```

## See Also

- [Data Availability Concepts](/concepts/data-availability)
- [Celestia Guide](/guides/da-layers/celestia)
- [Local DA Guide](/guides/da-layers/local-da)
