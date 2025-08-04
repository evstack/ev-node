package sequencer

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"
)

// DirectTxSequencer is an interface for sequencers that can handle direct transactions.
// It extends the Sequencer interface with a method for submitting direct transactions.
type DirectTxSequencer interface {
	Sequencer

	// SubmitDirectTxs adds direct transactions to the sequencer.
	// This method is called by the DirectTxReaper.
	SubmitDirectTxs(ctx context.Context, txs ...DirectTX) error
}

type DirectTX struct {
	TX              []byte
	ID              []byte
	FirstSeenHeight uint64
	// unix time
	FirstSeenTime int64
}

// ValidateBasic performs basic validation of DirectTX fields
func (d *DirectTX) ValidateBasic() error {
	if len(d.TX) == 0 {
		return fmt.Errorf("tx cannot be empty")
	}
	if len(d.ID) == 0 {
		return fmt.Errorf("id cannot be empty")
	}
	if d.FirstSeenHeight == 0 {
		return fmt.Errorf("first seen height cannot be zero")
	}
	if d.FirstSeenTime == 0 {
		return fmt.Errorf("first seen time cannot be zero")
	}
	return nil
}

// Hash Hash on the data
func (d *DirectTX) Hash() ([]byte, error) {
	hasher := sha256.New()
	hashWriteInt(hasher, len(d.ID))
	hasher.Write(d.ID)
	hashWriteInt(hasher, len(d.TX))
	hasher.Write(d.TX)
	return hasher.Sum(nil), nil
}

func hashWriteInt(hasher hash.Hash, data int) {
	txLen := make([]byte, 8) // 8 bytes for uint64
	binary.BigEndian.PutUint64(txLen, uint64(data))
	hasher.Write(txLen)
}
