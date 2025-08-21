package block

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/evstack/ev-node/types"
)

// SyncLoop is responsible for syncing blocks.
//
// SyncLoop processes headers gossiped in P2P network to know what's the latest block height, block data is retrieved from DA layer.
func (m *Manager) SyncLoop(ctx context.Context, errCh chan<- error) {
	daTicker := time.NewTicker(m.config.DA.BlockTime.Duration)
	defer daTicker.Stop()
	blockTicker := time.NewTicker(m.config.Node.BlockTime.Duration)
	defer blockTicker.Stop()

	// Create ticker for periodic metrics updates
	metricsTicker := time.NewTicker(30 * time.Second)
	defer metricsTicker.Stop()

	for {
		select {
		case <-daTicker.C:
			m.sendNonBlockingSignalToRetrieveCh()
		case <-blockTicker.C:
			m.sendNonBlockingSignalToHeaderStoreCh()
			m.sendNonBlockingSignalToDataStoreCh()
		case heightEvent := <-m.heightInCh:
			// Only validated headers and data are sent to heightEvent
			// we can safely assume that headerEvent.header is valid
			header := heightEvent.Header
			daHeight := heightEvent.DAHeight
			headerHash := header.Hash().String()
			headerHeight := header.Height()

			m.logger.Debug().
				Uint64("height", headerHeight).
				Uint64("daHeight", daHeight).
				Str("hash", headerHash).
				Msg("header retrieved")

			height, err := m.store.Height(ctx)
			if err != nil {
				m.logger.Error().Err(err).Msg("error while getting store height")
				continue
			}

			if headerHeight <= height || m.headerCache.IsSeen(headerHash) {
				m.logger.Debug().Uint64("height", headerHeight).Str("block_hash", headerHash).Msg("header already seen")
				continue
			}

			m.headerCache.SetItem(headerHeight, header)
			// Record header synced metric
			m.recordSyncMetrics("header_synced")

			// check if the dataHash is dataHashForEmptyTxs
			// if not, proceed to retrieve the data
			// otherwise, handle empty data for try next block to succeed
			var dataHash string
			if !bytes.Equal(header.DataHash, dataHashForEmptyTxs) {
				data := heightEvent.Data
				if len(data.Txs) == 0 || data.Metadata == nil {
					break
				}

				daHeight := heightEvent.DAHeight
				dataHash = data.DACommitment().String()
				dataHeight := data.Metadata.Height

				m.logger.Debug().
					Uint64("daHeight", daHeight).
					Str("hash", dataHash).
					Uint64("height", dataHeight).
					Int("txs", len(data.Txs)).
					Msg("data retrieved")

				if m.dataCache.IsSeen(dataHash) {
					m.logger.Debug().Str("data_hash", dataHash).Msg("data already seen")
					continue
				}

				if dataHeight <= height {
					m.logger.Debug().Uint64("height", dataHeight).Str("data_hash", dataHash).Msg("data already seen")
					continue
				}

				m.dataCache.SetItem(dataHeight, data)

				// Record data synced metric
				m.recordSyncMetrics("data_synced")
			} else {
				m.handleEmptyDataHash(ctx, &header.Header)
			}

			if err = m.trySyncNextBlock(ctx, daHeight); err != nil {
				errCh <- fmt.Errorf("failed to sync next block: %w", err)
				return
			}

			m.headerCache.SetSeen(headerHash)
			m.dataCache.SetSeen(dataHash)
		case <-metricsTicker.C:
			// Update channel metrics periodically
			m.updateChannelMetrics()
			m.updatePendingMetrics()
		case <-ctx.Done():
			return
		}
	}
}

// trySyncNextBlock tries to execute as many blocks as possible from the blockCache.
//
//	Note: the blockCache contains only valid blocks that are not yet synced
//
// For every block, to be able to apply block at height h, we need to have its Commit. It is contained in block at height h+1.
// If commit for block h+1 is available, we proceed with sync process, and remove synced block from sync cache.
func (m *Manager) trySyncNextBlock(ctx context.Context, daHeight uint64) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		currentHeight, err := m.store.Height(ctx)
		if err != nil {
			return err
		}
		h := m.headerCache.GetItem(currentHeight + 1)
		if h == nil {
			m.logger.Debug().Uint64("height", currentHeight+1).Msg("header not found in cache")
			return nil
		}
		d := m.dataCache.GetItem(currentHeight + 1)
		if d == nil {
			m.logger.Debug().Uint64("height", currentHeight+1).Msg("data not found in cache")
			return nil
		}

		hHeight := h.Height()
		m.logger.Info().Uint64("height", hHeight).Msg("syncing header and data")

		// set the custom verifier to ensure proper signature validation
		h.SetCustomVerifierForSyncNode(m.syncNodeSignaturePayloadProvider)

		newState, err := m.applyBlock(ctx, h.Header, d)
		if err != nil {
			return fmt.Errorf("failed to apply block: %w", err)
		}

		// validate the received block after applying
		// a custom verification function can depend on the state of the blockchain
		if err := m.Validate(ctx, h, d); err != nil {
			return fmt.Errorf("failed to validate block: %w", err)
		}

		if err = m.updateState(ctx, newState); err != nil {
			return fmt.Errorf("failed to save updated state: %w", err)
		}

		if err = m.store.SaveBlockData(ctx, h, d, &h.Signature); err != nil {
			return fmt.Errorf("failed to save block: %w", err)
		}

		// Height gets updated
		if err = m.store.SetHeight(ctx, hHeight); err != nil {
			return err
		}

		// Record sync metrics
		m.recordSyncMetrics("block_applied")

		if daHeight > newState.DAHeight {
			newState.DAHeight = daHeight
		}

		m.headerCache.DeleteItem(currentHeight + 1)
		m.dataCache.DeleteItem(currentHeight + 1)
		if !bytes.Equal(h.DataHash, dataHashForEmptyTxs) {
			m.dataCache.SetSeen(h.DataHash.String())
		}
		m.headerCache.SetSeen(h.Hash().String())
	}
}

func (m *Manager) handleEmptyDataHash(ctx context.Context, header *types.Header) {
	headerHeight := header.Height()
	var lastDataHash types.Hash
	if headerHeight > 1 {
		_, lastData, err := m.store.GetBlockData(ctx, headerHeight-1)
		if err != nil {
			m.logger.Debug().Uint64("current_height", headerHeight).Uint64("previous_height", headerHeight-1).Err(err).Msg("previous block not applied yet")
		}
		if lastData != nil {
			lastDataHash = lastData.Hash()
		}
	}
	metadata := &types.Metadata{
		ChainID:      header.ChainID(),
		Height:       headerHeight,
		Time:         header.BaseHeader.Time,
		LastDataHash: lastDataHash,
	}
	d := &types.Data{
		Metadata: metadata,
	}
	m.dataCache.SetItem(headerHeight, d)
}

func (m *Manager) sendNonBlockingSignalToHeaderStoreCh() {
	m.sendNonBlockingSignalWithMetrics(m.headerStoreCh, "header_store")
}

func (m *Manager) sendNonBlockingSignalToDataStoreCh() {
	m.sendNonBlockingSignalWithMetrics(m.dataStoreCh, "data_store")
}

func (m *Manager) sendNonBlockingSignalToRetrieveCh() {
	m.sendNonBlockingSignalWithMetrics(m.retrieveCh, "retrieve")
}

func (m *Manager) sendNonBlockingSignalToDAIncluderCh() {
	m.sendNonBlockingSignalWithMetrics(m.daIncluderCh, "da_includer")
}

// Updates the state stored in manager's store along the manager's lastState
func (m *Manager) updateState(ctx context.Context, s types.State) error {
	m.logger.Debug().Interface("newState", s).Msg("updating state")
	m.lastStateMtx.Lock()
	defer m.lastStateMtx.Unlock()
	err := m.store.UpdateState(ctx, s)
	if err != nil {
		return err
	}
	m.lastState = s
	m.metrics.Height.Set(float64(s.LastBlockHeight))
	return nil
}
