package sequencer

import (
	"context"
)

// DirectTxSequencer is an interface for sequencers that can handle direct transactions.
// It extends the Sequencer interface with a method for submitting direct transactions.
type DirectTxSequencer interface {
	Sequencer

	// SubmitDirectTxs adds direct transactions to the sequencer.
	// This method is called by the DirectTxReaper.
	SubmitDirectTxs(ctx context.Context, txs [][]byte) error
}
