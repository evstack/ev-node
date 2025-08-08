package block

import (
	"context"
	"fmt"
	"time"
)

// AggregationLoop is responsible for aggregating transactions into blocks.
func (m *Manager) AggregationLoop(ctx context.Context, errCh chan<- error) {
	initialHeight := m.genesis.InitialHeight //nolint:gosec
	height, err := m.store.Height(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("error while getting store height")
		return
	}
	var delay time.Duration

	if height < initialHeight {
		delay = time.Until(m.genesis.GenesisDAStartTime.Add(m.config.Node.BatchRetrievalInterval.Duration))
	} else {
		lastBlockTime := m.getLastBlockTime()
		delay = time.Until(lastBlockTime.Add(m.config.Node.BatchRetrievalInterval.Duration))
	}

	if delay > 0 {
		m.logger.Info().Dur("delay", delay).Msg("waiting to produce block")
		time.Sleep(delay)
	}

	// batchTimer is used to signal when to retrieve batches from the sequencer
	// based on the batch retrieval interval. This drives the block production cycle.
	batchTimer := time.NewTimer(0)
	defer batchTimer.Stop()

	// Lazy Sequencer mode.
	// In Lazy Sequencer mode, blocks are built only when there are
	// transactions or every LazyBlockTime.
	if m.config.Node.LazyMode {
		if err := m.lazyAggregationLoop(ctx, batchTimer); err != nil {
			errCh <- fmt.Errorf("error in lazy aggregation loop: %w", err)
		}
		return
	}

	if err := m.normalAggregationLoop(ctx, batchTimer); err != nil {
		errCh <- fmt.Errorf("error in normal aggregation loop: %w", err)
	}
}

func (m *Manager) lazyAggregationLoop(ctx context.Context, batchTimer *time.Timer) error {
	// lazyTimer triggers block publication even during inactivity
	lazyTimer := time.NewTimer(0)
	defer lazyTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-lazyTimer.C:
			m.logger.Debug().Msg("Lazy timer triggered block production")

			start := time.Now()
			if err := m.publishBlock(ctx); err != nil && ctx.Err() == nil {
				return fmt.Errorf("error while publishing block: %w", err)
			}
			lazyTimer.Reset(getRemainingSleep(start, m.config.Node.LazyBlockInterval.Duration))

		case <-batchTimer.C:
			if m.txsAvailable {
				m.logger.Debug().Msg("Batch timer triggered block production (lazy mode)")

				start := time.Now()
				if err := m.publishBlock(ctx); err != nil && ctx.Err() == nil {
					return fmt.Errorf("error while publishing block: %w", err)
				}

				m.txsAvailable = false
			} else {
				m.logger.Debug().Msg("Batch timer fired but no transactions available")
			}
			// Reset timer for next batch retrieval attempt
			batchTimer.Reset(m.config.Node.BatchRetrievalInterval.Duration)

		case <-m.txNotifyCh:
			m.txsAvailable = true
		}
	}
}


func (m *Manager) normalAggregationLoop(ctx context.Context, batchTimer *time.Timer) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-batchTimer.C:
			m.logger.Debug().Msg("Batch timer triggered block production")

			// Define the start time for the block production period
			start := time.Now()

			if err := m.publishBlock(ctx); err != nil && ctx.Err() == nil {
				return fmt.Errorf("error while publishing block: %w", err)
			}

			// Reset the batchTimer to signal the next batch retrieval
			// period based on the batch retrieval interval.
			batchTimer.Reset(getRemainingSleep(start, m.config.Node.BatchRetrievalInterval.Duration))

		case <-m.txNotifyCh:
			// Transaction notifications are intentionally ignored in normal mode
			// to avoid triggering block production outside the scheduled intervals.
			// We just update the txsAvailable flag for tracking purposes
			m.txsAvailable = true
		}
	}
}

func getRemainingSleep(start time.Time, interval time.Duration) time.Duration {
	elapsed := time.Since(start)

	if elapsed < interval {
		return interval - elapsed
	}

	return time.Millisecond
}
