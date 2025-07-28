package block

import (
	"context"
	"fmt"
)

// RollbackOneBlock rolls back the last block on the ev-node side.
func (m *Manager) RollbackOneBlock(ctx context.Context) error {
	// Get the current height
	currentHeight, err := m.store.Height(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	if currentHeight <= 1 {
		return fmt.Errorf("cannot rollback, already at genesis block")
	}

	// Get the block data for the previous block
	previousHeight := currentHeight - 1
	previousHeader, previousData, err := m.store.GetBlockData(ctx, previousHeight)
	if err != nil {
		return fmt.Errorf("failed to get previous block data: %w", err)
	}

	// Remove the block data until the previous height
	if err := m.store.Rollback(ctx, previousHeight); err != nil {
		return fmt.Errorf("failed to delete block data until height %d: %w", previousHeight, err)
	}

	// Update the state to reflect the previous block's state
	newState, err := m.applyBlock(ctx, previousHeader.Header, previousData)
	if err != nil {
		return fmt.Errorf("failed to apply previous block: %w", err)
	}

	// Update the state in the store
	if err := m.updateState(ctx, newState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	return nil
}
