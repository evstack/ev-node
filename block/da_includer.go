package block

import (
	"context"
	"encoding/binary"
	"fmt"

	coreda "github.com/evstack/ev-node/core/da"
	storepkg "github.com/evstack/ev-node/pkg/store"
)

// DAIncluderLoop is responsible for advancing the DAIncludedHeight by checking if blocks after the current height
// have both their header and data marked as DA-included in the caches. If so, it calls setDAIncludedHeight.
func (m *Manager) DAIncluderLoop(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.daIncluderCh:
			// proceed to check for DA inclusion
		}
		currentDAIncluded := m.GetDAIncludedHeight()
		for {
			nextHeight := currentDAIncluded + 1
			daIncluded, err := m.IsDAIncluded(ctx, nextHeight)
			if err != nil {
				// No more blocks to check at this time
				m.logger.Debug().Uint64("height", nextHeight).Err(err).Msg("no more blocks to check at this time")
				break
			}
			if daIncluded {
				m.logger.Debug().Uint64("height", nextHeight).Msg("both header and data are DA-included, advancing height")
				if err := m.SetSequencerHeightToDAHeight(ctx, nextHeight); err != nil {
					errCh <- fmt.Errorf("failed to set sequencer height to DA height: %w", err)
					return
				}
				// Both header and data are DA-included, so we can advance the height
				if err := m.incrementDAIncludedHeight(ctx); err != nil {
					errCh <- fmt.Errorf("error while incrementing DA included height: %w", err)
					return
				}

				currentDAIncluded = nextHeight
			} else {
				// Stop at the first block that is not DA-included
				break
			}
		}
	}
}

// incrementDAIncludedHeight sets the DA included height in the store
// It returns an error if the DA included height is not set.
func (m *Manager) incrementDAIncludedHeight(ctx context.Context) error {
	currentHeight := m.GetDAIncludedHeight()
	newHeight := currentHeight + 1
	m.logger.Debug().Uint64("height", newHeight).Msg("setting final height")
	err := m.exec.SetFinal(ctx, newHeight)
	if err != nil {
		m.logger.Error().Uint64("height", newHeight).Err(err).Msg("failed to set final height")
		return err
	}
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, newHeight)
	m.logger.Debug().Uint64("height", newHeight).Msg("setting DA included height")
	err = m.store.SetMetadata(ctx, storepkg.DAIncludedHeightKey, heightBytes)
	if err != nil {
		m.logger.Error().Uint64("height", newHeight).Err(err).Msg("failed to set DA included height")
		return err
	}
	if !m.daIncludedHeight.CompareAndSwap(currentHeight, newHeight) {
		return fmt.Errorf("failed to set DA included height: %d", newHeight)
	}

	// Update sequencer metrics if the sequencer supports it
	if seq, ok := m.sequencer.(MetricsRecorder); ok {
		seq.RecordMetrics(m.gasPrice, 0, coreda.StatusSuccess, m.pendingHeaders.numPendingHeaders(), newHeight)
	}

	return nil
}
