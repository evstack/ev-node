package block

import (
	"context"
	"fmt"
)

// RollbackOneBlock rolls back the last block on the ev-node side.
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

	if err := m.store.Rollback(ctx, height); err != nil {
		return fmt.Errorf("failed to delete block data until height %d: %w", height, err)
	}

	return nil
}
