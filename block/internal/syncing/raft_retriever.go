package syncing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
	"github.com/rs/zerolog"
)

type eventProcessor interface {
	handle(ctx context.Context, event common.DAHeightEvent) error
}
type eventProcessorFn func(ctx context.Context, event common.DAHeightEvent) error

func (e eventProcessorFn) handle(ctx context.Context, event common.DAHeightEvent) error {
	return e(ctx, event)
}

type raftRetriever struct {
	// Raft consensus
	raftNode       common.RaftNode
	wg             sync.WaitGroup
	logger         zerolog.Logger
	genesis        genesis.Genesis
	eventProcessor eventProcessor

	mtx    sync.Mutex
	cancel context.CancelFunc
}

func newRaftRetriever(
	raftNode common.RaftNode,
	genesis genesis.Genesis,
	logger zerolog.Logger,
	eventProcessor eventProcessor,
) *raftRetriever {
	return &raftRetriever{
		raftNode:       raftNode,
		genesis:        genesis,
		logger:         logger,
		eventProcessor: eventProcessor,
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
	applyCh := make(chan common.RaftApplyMsg, 100)
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
func (r *raftRetriever) raftApplyLoop(ctx context.Context, applyCh <-chan common.RaftApplyMsg) {
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
func (r *raftRetriever) consumeRaftBlock(ctx context.Context, state *common.RaftBlockState) error {
	r.logger.Debug().Uint64("height", state.Height).Msg("applying raft block")

	// Unmarshal header and data
	var header types.SignedHeader
	if err := header.UnmarshalBinary(state.Header); err != nil {
		return fmt.Errorf("unmarshal header: %w", err)
	}

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
		DaHeight: 0, // raft events don't have DA height context
	}
	return r.eventProcessor.handle(ctx, event)
}
