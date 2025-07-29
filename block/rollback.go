package block

import (
	"context"
	"fmt"
)

// Rollback rolls back to given height on the ev-node side.
func (m *Manager) Rollback(ctx context.Context, height uint64) error {
	if height == 0 {
		return fmt.Errorf("cannot rollback, already at genesis block")
	}

	currentHeight, err := m.store.Height(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	if height >= currentHeight {
		return fmt.Errorf("cannot rollback to height %d, current height is %d", height, currentHeight)
	}

	// rollback execution environment state
	if err := m.exec.Rollback(ctx, height); err != nil {
		return fmt.Errorf("failed to rollback execution environment to height %d: %w", height, err)
	}

	// rollback ev-node store
	if err := m.store.Rollback(ctx, height); err != nil {
		return fmt.Errorf("failed to delete block data until height %d: %w", height, err)
	}

	return nil
}
