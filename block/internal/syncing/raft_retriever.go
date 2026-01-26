package syncing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/raft"
	"github.com/evstack/ev-node/types"
	"github.com/rs/zerolog"
)

// eventProcessor handles DA height events. Used for syncing.
type eventProcessor interface {
	// handle processes a single DA height event.
	handle(ctx context.Context, event common.DAHeightEvent) error
}

// eventProcessorFn adapts a function to an eventProcessor.
type eventProcessorFn func(ctx context.Context, event common.DAHeightEvent) error

// handle calls the wrapped function.
func (e eventProcessorFn) handle(ctx context.Context, event common.DAHeightEvent) error {
	return e(ctx, event)
}

// raftStatePreProcessor is called before processing a raft block state
type raftStatePreProcessor func(ctx context.Context, state *raft.RaftBlockState) error

// raftRetriever retrieves raft blocks and feeds them into the eventProcessor
type raftRetriever struct {
	raftNode              common.RaftNode
	wg                    sync.WaitGroup
	logger                zerolog.Logger
	genesis               genesis.Genesis
	eventProcessor        eventProcessor
	raftBlockPreProcessor raftStatePreProcessor

	mtx    sync.Mutex
	cancel context.CancelFunc
}

// newRaftRetriever constructor
func newRaftRetriever(
	raftNode common.RaftNode,
	genesis genesis.Genesis,
	logger zerolog.Logger,
	eventProcessor eventProcessor,
	raftBlockPreProcessor raftStatePreProcessor,
) *raftRetriever {
	return &raftRetriever{
		raftNode:              raftNode,
		genesis:               genesis,
		logger:                logger,
		eventProcessor:        eventProcessor,
		raftBlockPreProcessor: raftBlockPreProcessor,
	}
}

// Start begins the syncing component
func (r *raftRetriever) Start(ctx context.Context) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	if r.cancel != nil {
		return errors.New("syncer already started")
	}
	ctx, r.cancel = context.WithCancel(ctx)
	applyCh := make(chan raft.RaftApplyMsg, 100)
	r.raftNode.SetApplyCallback(applyCh)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.raftApplyLoop(ctx, applyCh)
	}()
	return nil
}

// Stop gracefully shuts down the raft retriever
func (r *raftRetriever) Stop() {
	r.mtx.Lock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.mtx.Unlock()

	r.wg.Wait()
}

// raftApplyLoop processes blocks received from raft
func (r *raftRetriever) raftApplyLoop(ctx context.Context, applyCh <-chan raft.RaftApplyMsg) {
	r.logger.Info().Msg("starting raft apply loop")
	defer r.logger.Info().Msg("raft apply loop stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-applyCh:
			if err := r.consumeRaftBlock(ctx, msg.State); err != nil {
				r.logger.Error().Err(err).Uint64("height", msg.State.Height).Msg("failed to apply raft block")
			}
		}
	}
}

// consumeRaftBlock applies a block received from raft consensus
func (r *raftRetriever) consumeRaftBlock(ctx context.Context, state *raft.RaftBlockState) error {
	r.logger.Debug().
		Uint64("height", state.Height).
		Hex("raft_hash", state.Hash).
		Uint64("timestamp", state.Timestamp).
		Msg("consuming raft block")
	if err := r.raftBlockPreProcessor(ctx, state); err != nil {
		return err
	}

	// Unmarshal header and data
	var header types.SignedHeader
	if err := header.UnmarshalBinary(state.Header); err != nil {
		return fmt.Errorf("unmarshal header: %w", err)
	}

	// Log the unmarshalled header hash for comparison with raft state hash
	r.logger.Debug().
		Uint64("height", state.Height).
		Hex("raft_hash", state.Hash).
		Hex("header_hash", header.Hash()).
		Str("chain_id", header.ChainID()).
		Hex("data_hash", header.DataHash).
		Hex("app_hash", header.AppHash).
		Msg("raft block hash comparison")

	if err := header.Header.ValidateBasic(); err != nil {
		r.logger.Debug().Err(err).Msg("invalid header structure")
		return nil
	}
	if err := assertExpectedProposer(r.genesis, header.ProposerAddress); err != nil {
		r.logger.Debug().Err(err).Msg("unexpected proposer")
		return nil
	}

	var data types.Data
	if err := data.UnmarshalBinary(state.Data); err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	event := common.DAHeightEvent{
		Header:   &header,
		Data:     &data,
		DaHeight: 0, // raft events don't have DA height context, yet as DA submission is asynchronous
	}
	return r.eventProcessor.handle(ctx, event)
}

// Height returns the current height of the raft node's state.
func (r *raftRetriever) Height() uint64 {
	return r.raftNode.GetState().Height
}
