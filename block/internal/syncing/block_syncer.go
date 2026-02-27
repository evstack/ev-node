package syncing

import (
	"context"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/types"
)

// BlockSyncer defines operations for block synchronization that can be traced.
// The Syncer implements this interface, and a tracing decorator can wrap it
// to add OpenTelemetry spans to each operation.
type BlockSyncer interface {
	// TrySyncNextBlock attempts to sync the next available block from DA or P2P.
	TrySyncNextBlock(ctx context.Context, event *common.DAHeightEvent) error

	// ApplyBlock executes block transactions and returns the new state.
	ApplyBlock(ctx context.Context, header types.Header, data *types.Data, currentState types.State) (types.State, error)

	// ValidateBlock validates block structure and state transitions.
	ValidateBlock(ctx context.Context, currState types.State, data *types.Data, header *types.SignedHeader) error

	// VerifyForcedInclusionTxs verifies that forced inclusion transactions are properly handled.
	VerifyForcedInclusionTxs(ctx context.Context, daHeight uint64, data *types.Data) error
}
