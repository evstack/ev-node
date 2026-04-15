// Package fibre provides a Go client interface and mock implementation for the
// Fibre DA (Data Availability) gRPC service.
//
// # Design Assumptions
//
//   - The sequencer trusts the encoder to eventually confirm blob inclusion.
//     Upload returns after the blob is uploaded and the PFF transaction is
//     broadcast, NOT after on-chain confirmation. This keeps the sequencer's
//     write path fast (~2s per 128 MB blob).
//
//   - Callers are expected to batch/buffer their data into blobs sized for the
//     protocol maximum (128 MiB - 5 byte header = 134,217,723 bytes).
//     The interface accepts arbitrary sizes but the implementation may batch
//     or reject oversized blobs.
//
//   - Confirmation/finality is intentionally omitted from the initial API.
//     The sequencer does not need it; the read path (Listen + Download) is
//     sufficient for full nodes. A Status or Confirm RPC can be added later
//     if needed without breaking existing callers.
//
//   - Blob ordering is encoded in the blob data itself by the caller.
//     The interface does not impose or guarantee ordering.
//
//   - The interface is the same whether the encoder runs in-process or as an
//     external gRPC service. For in-process use, call the mock or real
//     implementation directly; for external use, connect via gRPC.
package fibremock

import (
	"context"
	"time"
)

// BlobID uniquely identifies an uploaded blob (version byte + 32-byte commitment).
type BlobID []byte

// UploadResult is returned by Upload after the blob is accepted.
type UploadResult struct {
	// BlobID uniquely identifies the uploaded blob.
	BlobID BlobID
	// ExpiresAt is when the blob will be pruned from the DA network.
	// Consumers must download before this time.
	ExpiresAt time.Time
}

// BlobEvent is delivered via Listen when a blob is confirmed on-chain.
type BlobEvent struct {
	// BlobID of the confirmed blob.
	BlobID BlobID
	// Height is the chain height at which the blob was confirmed.
	Height uint64
	// DataSize is the size of the original blob data in bytes (from the PFF).
	// This allows full nodes to know the size before downloading.
	DataSize uint64
}

// DA is the interface for interacting with the Fibre data availability layer.
//
// Implementations include:
//   - MockDA: in-memory mock for testing
//   - (future) gRPC client wrapping the Fibre service
//   - (future) in-process encoder using fibre.Client directly
type DA interface {
	// Upload submits a blob under the given namespace to the DA network.
	// Returns after the blob is uploaded and the payment transaction is broadcast.
	// Does NOT wait for on-chain confirmation (see package doc for rationale).
	//
	// The caller is responsible for batching data to the target blob size.
	Upload(ctx context.Context, namespace []byte, data []byte) (UploadResult, error)

	// Download retrieves and reconstructs a blob by its ID.
	// Returns the original data that was passed to Upload.
	Download(ctx context.Context, blobID BlobID) ([]byte, error)

	// Listen streams confirmed blob events for the given namespace.
	// The returned channel is closed when the context is cancelled.
	// Each event includes the blob ID, confirmation height, and data size.
	Listen(ctx context.Context, namespace []byte) (<-chan BlobEvent, error)
}
