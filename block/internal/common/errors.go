package common

import (
	"errors"
)

// These errors are used by block components.
var (
	// ErrNotProposer is used when the node is not a proposer
	ErrNotProposer = errors.New("not a proposer")

	// ErrNoBatch indicate no batch is available for creating block
	ErrNoBatch = errors.New("no batch to process")

	// ErrNoTransactionsInBatch is used when no transactions are found in batch
	ErrNoTransactionsInBatch = errors.New("no transactions found in batch")

	// ErrHeightFromFutureStr is the error message for height from future returned by da
	ErrHeightFromFutureStr = errors.New("given height is from the future")
)
