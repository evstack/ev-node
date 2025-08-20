package block

import (
	"context"
	"fmt"

	"github.com/evstack/ev-node/types"
)

// HeaderStoreRetrieveLoop is responsible for retrieving headers from the Header Store.
func (m *Manager) HeaderStoreRetrieveLoop(ctx context.Context) {
	// height is always > 0
	initialHeight, err := m.store.Height(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to get initial store height for DataStoreRetrieveLoop")
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
			headers, err := m.getHeadersFromHeaderStore(ctx, lastHeaderStoreHeight+1, headerStoreHeight)
			if err != nil {
				m.logger.Error().Uint64("lastHeaderHeight", lastHeaderStoreHeight).Uint64("headerStoreHeight", headerStoreHeight).Str("errors", err.Error()).Msg("failed to get headers from Header Store")
				continue
			}
			daHeight := m.daHeight.Load()
			for _, header := range headers {
				// Check for shut down event prior to logging
				// and sending header to headerInCh. The reason
				// for checking for the shutdown event
				// separately is due to the inconsistent nature
				// of the select statement when multiple cases
				// are satisfied.
				select {
				case <-ctx.Done():
					return
				default:
				}

				// early validation to reject junk headers
				if !m.isUsingExpectedSingleSequencer(header) {
					continue
				}

				// set custom verifier to do correct header verification
				header.SetCustomVerifier(m.signaturePayloadProvider)

				// validate header and its signature validity
				if err := header.ValidateBasic(); err != nil {
					continue
				}

				m.logger.Debug().Uint64("headerHeight", header.Height()).Uint64("daHeight", daHeight).Msg("header retrieved from p2p header sync")
				m.headerInCh <- NewHeaderEvent{header, daHeight}
			}
		}
		lastHeaderStoreHeight = headerStoreHeight
	}
}

// DataStoreRetrieveLoop is responsible for retrieving data from the Data Store.
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
			data, err := m.getDataFromDataStore(ctx, lastDataStoreHeight+1, dataStoreHeight)
			if err != nil {
				m.logger.Error().Uint64("lastDataStoreHeight", lastDataStoreHeight).Uint64("dataStoreHeight", dataStoreHeight).Str("errors", err.Error()).Msg("failed to get data from Data Store")
				continue
			}
			daHeight := m.daHeight.Load()
			for _, d := range data {
				// Check for shut down event prior to logging
				// and sending header to dataInCh. The reason
				// for checking for the shutdown event
				// separately is due to the inconsistent nature
				// of the select statement when multiple cases
				// are satisfied.
				select {
				case <-ctx.Done():
					return
				default:
				}
				// TODO: remove junk if possible
				m.logger.Debug().Uint64("dataHeight", d.Metadata.Height).Uint64("daHeight", daHeight).Msg("data retrieved from p2p data sync")
				m.dataInCh <- NewDataEvent{d, daHeight}
			}
		}
		lastDataStoreHeight = dataStoreHeight
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
