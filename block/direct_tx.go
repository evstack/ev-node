package block

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/evstack/ev-node/types"
	"sync"
)

type ForcedInclusionConfig struct {
	MaxInclusionDelay uint64 // Max inclusion time in DA block time units
	MinDADelay        uint64 // Minimum number of DA blocks before including a direct tx
}

type DirectTransaction struct {
	TxHash            types.Hash
	FirstSeenDAHeight uint64 // DA block time when the tx was seen
	Included          bool   // Whether it has been included in a block
	IncludedAt        uint64 // Height at which it was included
	TX                []byte
}

type DirectTxTracker struct {
	config ForcedInclusionConfig
	mu     sync.RWMutex
	//txs                   map[string]DirectTransaction // hash -> tx
	txs                   []DirectTransaction // ordered by da height and position in blob
	latestSeenDABlockTime uint64
	latestDAHeight        uint64
}

func (m *Manager) handlePotentialDirectTXs(ctx context.Context, bz []byte, daHeight uint64) bool {
	var unsignedData types.Data // todo (Alex): we need some type to separate from noise
	err := unsignedData.UnmarshalBinary(bz)
	if err != nil {
		m.logger.Debug("failed to unmarshal unsigned data, error", err)
		return false
	}
	if len(unsignedData.Txs) == 0 {
		m.logger.Debug("ignoring empty unsigned data, daHeight: ", daHeight)
		return false
	}
	if unsignedData.Metadata.ChainID != m.genesis.ChainID {
		m.logger.Debug("ignoring unsigned data from different chain, daHeight: ", daHeight)
		return false
	}
	//Early validation to reject junk data
	//if !m.isValidSignedData(&unsignedData) {
	//	m.logger.Debug("invalid data signature, daHeight: ", daHeight)
	//	return false
	//}
	h := m.headerCache.GetItem(daHeight)
	if h == nil {
		panic("header not found in cache") // todo (Alex): not sure if headers are always available before data. better assume not
		//m.logger.Debug("header not found in cache, height:", daHeight)
		//return false
	}
	for _, tx := range unsignedData.Txs {
		txHash := sha256.New().Sum(tx)
		d := DirectTransaction{
			TxHash:            txHash,
			TX:                unsignedData.Txs[0],
			FirstSeenDAHeight: daHeight,
			Included:          false,
			IncludedAt:        0,
		}
		m.directTXTracker.mu.Lock()
		m.directTXTracker.txs = append(m.directTXTracker.txs, d)
		m.directTXTracker.mu.Unlock()
	}
	return true
}

var ErrMissingDirectTx = errors.New("missing direct tx")

func (m *Manager) getPendingDirectTXs(_ context.Context, maxBytes int) ([][]byte, error) {
	remaining := maxBytes
	currBlockTime := m.directTXTracker.latestSeenDABlockTime
	var res [][]byte

	m.directTXTracker.mu.Lock()
	defer m.directTXTracker.mu.Unlock()
	for _, tx := range m.directTXTracker.txs {
		if tx.Included {
			continue
		}

		if currBlockTime-tx.FirstSeenDAHeight > m.directTXTracker.config.MaxInclusionDelay {
			// should have been forced included already.
			// what should we do now? stop the world
			return nil, ErrMissingDirectTx
		}
		if m.directTXTracker.latestDAHeight-tx.FirstSeenDAHeight < m.directTXTracker.config.MinDADelay {
			// we can stop here as following tx are newer
			break
		}
		if len(tx.TX) > remaining {
			break
		}
		res = append(res, tx.TX)
	}
	return res, nil
}
