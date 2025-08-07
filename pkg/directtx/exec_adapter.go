package directtx

import (
	"context"

	"time"

	"github.com/evstack/ev-node/core/execution"
)

var _ execution.Executor = &ExecutionAdapter{}

type DAHeightSource interface {
	GetDAIncludedHeight() uint64
}
type DirectTXProvider interface {
	GetPendingDirectTXs(ctx context.Context, daIncludedHeight uint64, maxBytes uint64) ([][]byte, error)
	MarkTXIncluded(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time) error
	HasMissedDirectTX(ctx context.Context, blockHeight uint64, timestamp time.Time) error
}

type ExecutionAdapter struct {
	nested           execution.Executor
	directTXProvider DirectTXProvider
	daHeightSource   DAHeightSource
	maxBlockBytes    uint64
}

func NewExecutionAdapter(
	nested execution.Executor,
	directTXProvider DirectTXProvider,
	daHeightSource DAHeightSource,
	maxBlockBytes uint64,
) *ExecutionAdapter {
	return &ExecutionAdapter{nested: nested, directTXProvider: directTXProvider, daHeightSource: daHeightSource, maxBlockBytes: maxBlockBytes}
}

func (d *ExecutionAdapter) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) (stateRoot []byte, maxBytes uint64, err error) {
	return d.nested.InitChain(ctx, genesisTime, initialHeight, chainID)
}

func (d *ExecutionAdapter) GetTxs(ctx context.Context) ([][]byte, error) {
	var otherTx [][]byte
	var err error
	if !IsInFallbackMode(ctx) {
		otherTx, err = d.nested.GetTxs(ctx)
		if err != nil {
			return nil, err
		}
	}
	// todo (Alex): should we reserver some capacity for the mempool TX?
	txs, err := d.directTXProvider.GetPendingDirectTXs(ctx, d.daHeightSource.GetDAIncludedHeight(), d.maxBlockBytes)
	if err != nil {
		return nil, err
	}
	return append(txs, otherTx...), nil
}

func (d *ExecutionAdapter) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) (updatedStateRoot []byte, maxBytes uint64, err error) {
	if d.directTXProvider.MarkTXIncluded(ctx, txs, blockHeight, timestamp) != nil {
		return nil, 0, err
	}
	if !IsInFallbackMode(ctx) && // bypass check in fallback mode
		d.directTXProvider.HasMissedDirectTX(ctx, blockHeight, timestamp) != nil {
		return nil, 0, err
	}
	return d.nested.ExecuteTxs(ctx, txs, blockHeight, timestamp, prevStateRoot)
}

func (d *ExecutionAdapter) SetFinal(ctx context.Context, blockHeight uint64) error {
	return d.nested.SetFinal(ctx, blockHeight)
}
