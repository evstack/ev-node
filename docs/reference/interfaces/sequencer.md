# Sequencer Interface

The Sequencer interface defines how ev-node orders transactions for block production. Two implementations are provided: single sequencer and based sequencer.

## Interface Definition

```go
type Sequencer interface {
    SubmitBatchTxs(ctx context.Context, req SubmitBatchTxsRequest) (*SubmitBatchTxsResponse, error)
    GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error)
    VerifyBatch(ctx context.Context, req VerifyBatchRequest) (*VerifyBatchResponse, error)
    SetDAHeight(height uint64)
    GetDAHeight() uint64
}
```

## Methods

### SubmitBatchTxs

Submits a batch of transactions from the executor to the sequencer.

```go
SubmitBatchTxs(ctx context.Context, req SubmitBatchTxsRequest) (*SubmitBatchTxsResponse, error)
```

**Request:**

```go
type SubmitBatchTxsRequest struct {
    Id    []byte   // Chain identifier
    Batch *Batch   // Transactions to submit
}

type Batch struct {
    Transactions [][]byte
}
```

### GetNextBatch

Returns the next batch of transactions for block production.

```go
GetNextBatch(ctx context.Context, req GetNextBatchRequest) (*GetNextBatchResponse, error)
```

**Request:**

```go
type GetNextBatchRequest struct {
    Id            []byte     // Chain identifier
    LastBatchData [][]byte   // Previous batch data
    MaxBytes      uint64     // Maximum batch size
}
```

**Response:**

```go
type GetNextBatchResponse struct {
    Batch     *Batch      // Transactions to include
    Timestamp time.Time   // Block timestamp
    BatchData [][]byte    // Data for verification
}
```

### VerifyBatch

Verifies a batch received from another node during sync.

```go
VerifyBatch(ctx context.Context, req VerifyBatchRequest) (*VerifyBatchResponse, error)
```

**Request:**

```go
type VerifyBatchRequest struct {
    Id        []byte     // Chain identifier
    BatchData [][]byte   // Batch data to verify
}
```

**Response:**

```go
type VerifyBatchResponse struct {
    Status bool  // true if valid
}
```

### SetDAHeight / GetDAHeight

Track the current DA height for forced inclusion retrieval.

```go
SetDAHeight(height uint64)
GetDAHeight() uint64
```

## Batch Type

```go
type Batch struct {
    Transactions [][]byte
}

// Hash returns SHA256 hash of the batch
func (batch *Batch) Hash() ([]byte, error)
```

The hash is computed deterministically:

1. Write transaction count as uint64 (big-endian)
2. For each transaction: write length as uint64, then bytes

## Implementations

### Single Sequencer

Located in `pkg/sequencers/single/`.

- Maintains local mempool
- Supports forced inclusion from DA
- Default for most deployments

### Based Sequencer

Located in `pkg/sequencers/based/`.

- No local mempool
- All transactions come from DA layer
- Maximum censorship resistance

## Configuration

Select sequencer mode via configuration:

```yaml
# Single sequencer (default)
sequencer:
  type: single

# Based sequencer
sequencer:
  type: based
```

## Forced Inclusion

Both sequencer implementations support forced inclusion, but with different behaviors:

| Sequencer | Forced Inclusion Source | Mempool |
|-----------|------------------------|---------|
| Single | DA namespace + local mempool | Yes |
| Based | DA namespace only | No |

The sequencer tracks DA height via `SetDAHeight()` to know which forced inclusion transactions to include.
