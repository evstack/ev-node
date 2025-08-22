package block

import (
	"bytes"
	"context"
	"fmt"

	"github.com/evstack/ev-node/types"
)

// HeaderStoreRetrieveLoop is responsible for retrieving headers from the Header Store.
// It retrieves both header and corresponding data before sending to heightInCh for validation.
func (m *Manager) HeaderStoreRetrieveLoop(ctx context.Context) {
	// height is always > 0
	initialHeight, err := m.store.Height(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to get initial store height for HeaderStoreRetrieveLoop")
		return
	}
	lastHeaderStoreHeight := initialHeight
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.headerStoreCh:
		}
		headerStoreHeight := m.headerStore.Height()
		if headerStoreHeight > lastHeaderStoreHeight {
			m.processHeaderStoreRange(ctx, lastHeaderStoreHeight+1, headerStoreHeight)
		}
		lastHeaderStoreHeight = headerStoreHeight
	}
}

// DataStoreRetrieveLoop is responsible for retrieving data from the Data Store.
// It retrieves both data and corresponding header before sending to heightInCh for validation.
func (m *Manager) DataStoreRetrieveLoop(ctx context.Context) {
	// height is always > 0
	initialHeight, err := m.store.Height(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to get initial store height for DataStoreRetrieveLoop")
		return
	}
	lastDataStoreHeight := initialHeight
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.dataStoreCh:
		}
		dataStoreHeight := m.dataStore.Height()
		if dataStoreHeight > lastDataStoreHeight {
			m.processDataStoreRange(ctx, lastDataStoreHeight+1, dataStoreHeight)
		}
		lastDataStoreHeight = dataStoreHeight
	}
}

// processHeaderStoreRange processes headers from header store and retrieves corresponding data
func (m *Manager) processHeaderStoreRange(ctx context.Context, startHeight, endHeight uint64) {
	headers, err := m.getHeadersFromHeaderStore(ctx, startHeight, endHeight)
	if err != nil {
		m.logger.Error().Uint64("startHeight", startHeight).Uint64("endHeight", endHeight).Str("errors", err.Error()).Msg("failed to get headers from Header Store")
		return
	}
	daHeight := m.daHeight.Load()
	for _, header := range headers {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// early validation to reject junk headers
		if ok, _ := m.isUsingExpectedSingleSequencer(header.ProposerAddress); !ok {
			continue
		}

		// set custom verifier to do correct header verification
		header.SetCustomVerifierForSyncNode(m.syncNodeSignaturePayloadProvider)

		// Get corresponding data for this header
		var data *types.Data
		if bytes.Equal(header.DataHash, dataHashForEmptyTxs) {
			// Create empty data for headers with empty data hash
			data = m.createEmptyDataForHeader(ctx, header)
		} else {
			// Try to get data from data store
			retrievedData, err := m.dataStore.GetByHeight(ctx, header.Height())
			if err != nil {
				m.logger.Debug().Uint64("height", header.Height()).Err(err).Msg("could not retrieve data for header from data store")
				continue
			}
			data = retrievedData
		}

		// we need to wait until the previous height has been executed in order to continue syncing
		if err := m.waitForHeight(ctx, header.Height()-1); err != nil {
			m.logger.Error().Err(err).Msg("failed to wait for previous height")
			return
		}

		// validate header and its signature validity with data
		if err := header.ValidateBasicWithData(data); err != nil {
			m.logger.Debug().Uint64("height", header.Height()).Err(err).Msg("header validation with data failed")
			continue
		}

		m.sendCompleteHeightEvent(ctx, header, data, daHeight, "p2p header sync")
	}
}

// processDataStoreRange processes data from data store and retrieves corresponding headers
func (m *Manager) processDataStoreRange(ctx context.Context, startHeight, endHeight uint64) {
	data, err := m.getDataFromDataStore(ctx, startHeight, endHeight)
	if err != nil {
		m.logger.Error().Uint64("startHeight", startHeight).Uint64("endHeight", endHeight).Str("errors", err.Error()).Msg("failed to get data from Data Store")
		return
	}
	daHeight := m.daHeight.Load()
	for _, d := range data {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get corresponding header for this data
		header, err := m.headerStore.GetByHeight(ctx, d.Metadata.Height)
		if err != nil {
			m.logger.Debug().Uint64("height", d.Metadata.Height).Err(err).Msg("could not retrieve header for data from header store")
			continue
		}

		// early validation to reject junk headers
		if ok, _ := m.isUsingExpectedSingleSequencer(header.ProposerAddress); !ok {
			continue
		}

		// set custom verifier to do correct header verification
		header.SetCustomVerifierForSyncNode(m.syncNodeSignaturePayloadProvider)

		// we need to wait until the previous height has been executed in order to continue syncing
		if err := m.waitForHeight(ctx, header.Height()-1); err != nil {
			m.logger.Error().Err(err).Msg("failed to wait for previous height")
			return
		}

		// validate header and its signature validity with data
		if err := header.ValidateBasicWithData(d); err != nil {
			m.logger.Debug().Uint64("height", d.Metadata.Height).Err(err).Msg("header validation with data failed")
			continue
		}

		m.sendCompleteHeightEvent(ctx, header, d, daHeight, "p2p data sync")
	}
}

// sendCompleteHeightEvent sends a complete height event with both header and data
func (m *Manager) sendCompleteHeightEvent(ctx context.Context, header *types.SignedHeader, data *types.Data, daHeight uint64, source string) {
	heightEvent := NewHeightEvent{
		Header:   header,
		Data:     data,
		DAHeight: daHeight,
	}

	select {
	case <-ctx.Done():
		return
	case m.heightInCh <- heightEvent:
		m.logger.Debug().
			Uint64("height", header.Height()).
			Uint64("daHeight", daHeight).
			Str("source", source).
			Msg("sent complete height event with header and data")
	default:
		m.logger.Warn().
			Uint64("height", header.Height()).
			Uint64("daHeight", daHeight).
			Str("source", source).
			Msg("heightInCh backlog full, dropping complete event")
	}
}

// createEmptyDataForHeader creates empty data for headers with empty data hash
func (m *Manager) createEmptyDataForHeader(ctx context.Context, header *types.SignedHeader) *types.Data {
	headerHeight := header.Height()
	var lastDataHash types.Hash

	if headerHeight > 1 {
		_, lastData, err := m.store.GetBlockData(ctx, headerHeight-1)
		if err != nil {
			m.logger.Debug().Uint64("current_height", headerHeight).Uint64("previous_height", headerHeight-1).Msg(fmt.Sprintf("previous block not available, using empty last data hash: %s", err.Error()))
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

	return &types.Data{
		Metadata: metadata,
	}
}

func (m *Manager) getHeadersFromHeaderStore(ctx context.Context, startHeight, endHeight uint64) ([]*types.SignedHeader, error) {
	if startHeight > endHeight {
		return nil, fmt.Errorf("startHeight (%d) is greater than endHeight (%d)", startHeight, endHeight)
	}
	headers := make([]*types.SignedHeader, endHeight-startHeight+1)
	for i := startHeight; i <= endHeight; i++ {
		header, err := m.headerStore.GetByHeight(ctx, i)
		if err != nil {
			return nil, err
		}
		headers[i-startHeight] = header
	}
	return headers, nil
}

func (m *Manager) getDataFromDataStore(ctx context.Context, startHeight, endHeight uint64) ([]*types.Data, error) {
	if startHeight > endHeight {
		return nil, fmt.Errorf("startHeight (%d) is greater than endHeight (%d)", startHeight, endHeight)
	}
	data := make([]*types.Data, endHeight-startHeight+1)
	for i := startHeight; i <= endHeight; i++ {
		d, err := m.dataStore.GetByHeight(ctx, i)
		if err != nil {
			return nil, err
		}
		data[i-startHeight] = d
	}
	return data, nil
}
