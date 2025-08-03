package block

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/evstack/ev-node/types"
)

type DirectTransaction struct {
	TxHash            types.Hash
	FirstSeenDAHeight uint64 // DA block time when the tx was seen
	Included          bool   // Whether it has been included in a block
	IncludedAt        uint64 // Height at which it was included
	TX                []byte
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
		_ = d
	}
	return true
}

var ErrMissingDirectTx = errors.New("missing direct tx")
