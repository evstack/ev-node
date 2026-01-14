package executing

import (
	"context"

	"github.com/evstack/ev-node/types"
)

// BlockProducer defines operations for block production that can be traced.
// The Executor implements this interface, and a tracing decorator can wrap it
// to add OpenTelemetry spans to each operation.
type BlockProducer interface {
	// ProduceBlock creates, validates, and broadcasts a new block.
	ProduceBlock(ctx context.Context) error

	// RetrieveBatch gets the next batch of transactions from the sequencer.
	RetrieveBatch(ctx context.Context) (*BatchData, error)

	// CreateBlock constructs a new block from the given batch data.
	CreateBlock(ctx context.Context, height uint64, batchData *BatchData) (*types.SignedHeader, *types.Data, error)

	// ApplyBlock executes the block transactions and returns the new state.
	ApplyBlock(ctx context.Context, header types.Header, data *types.Data) (types.State, error)

	// ValidateBlock validates block structure and state transitions.
	ValidateBlock(lastState types.State, header *types.SignedHeader, data *types.Data) error
}
