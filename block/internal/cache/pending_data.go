package cache

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

// LastSubmittedDataHeightKey is the key used for persisting the last submitted data height in store.
const LastSubmittedDataHeightKey = "last-submitted-data-height"

// PendingData maintains Data that need to be published to DA layer
//
// Important assertions:
// - data is safely stored in database before submission to DA
// - data is always pushed to DA in order (by height)
// - DA submission of multiple data is atomic - it's impossible to submit only part of a batch
//
// lastSubmittedDataHeight is updated only after receiving confirmation from DA.
// Worst case scenario is when data was successfully submitted to DA, but confirmation was not received (e.g. node was
// restarted, networking issue occurred). In this case data is re-submitted to DA (it's extra cost).
// evolve is able to skip duplicate data so this shouldn't affect full nodes.
// Note: Submission of pending data to DA should account for the DA max blob size.
type PendingData struct {
	base *pendingBase[*types.Data]
}

func fetchData(ctx context.Context, store store.Store, height uint64) (*types.Data, error) {
	_, data, err := store.GetBlockData(ctx, height)
	return data, err
}

// NewPendingData returns a new PendingData struct
func NewPendingData(store store.Store, logger zerolog.Logger) (*PendingData, error) {
	base, err := newPendingBase(store, logger, LastSubmittedDataHeightKey, fetchData)
	if err != nil {
		return nil, err
	}
	return &PendingData{base: base}, nil
}

func (pd *PendingData) init() error {
	return pd.base.init()
}

// GetPendingData returns a sorted slice of pending Data.
func (pd *PendingData) GetPendingData(ctx context.Context) ([]*types.Data, error) {
	return pd.base.getPending(ctx)
}

func (pd *PendingData) NumPendingData() uint64 {
	return pd.base.numPending()
}

func (pd *PendingData) SetLastSubmittedDataHeight(ctx context.Context, newLastSubmittedDataHeight uint64) {
	pd.base.setLastSubmittedHeight(ctx, newLastSubmittedDataHeight)
}
