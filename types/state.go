package types

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

// InitStateVersion sets the Consensus.Block and Software versions,
// but leaves the Consensus.App version blank.
// The Consensus.App version will be set during the Handshake, once
// we hear from the app what protocol version it is running.
var InitStateVersion = Version{
	Block: 11, // Block version is set to 11, to be compatible with CometBFT blocks for IBC.
	App:   0,
}

// State contains information about current state of the blockchain.
type State struct {
	Version Version

	// immutable
	ChainID       string
	InitialHeight uint64

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight uint64
	LastBlockTime   time.Time

	// LastHeaderHash is the hash of the header of the last block
	LastHeaderHash Hash

	// DAHeight identifies DA block containing the latest applied Evolve block for a syncing node.
	// In the case of an aggregator, this corresponds as the last fetched DA block height for forced included transactions.
	DAHeight uint64

	// the latest AppHash we've received from calling abci.Commit()
	AppHash []byte

	// NextProposerAddress is the proposer expected to sign LastBlockHeight+1.
	// It is initialized from genesis and then updated from execution results.
	NextProposerAddress []byte
}

func (s *State) NextState(header Header, stateRoot []byte, nextProposerAddress ...[]byte) (State, error) {
	height := header.Height()
	nextProposer := header.NextProposerAddress
	if len(nextProposerAddress) > 0 && len(nextProposerAddress[0]) > 0 {
		nextProposer = nextProposerAddress[0]
	}
	if len(nextProposer) == 0 {
		nextProposer = header.ProposerAddress
	}

	return State{
		Version:             s.Version,
		ChainID:             s.ChainID,
		InitialHeight:       s.InitialHeight,
		LastBlockHeight:     height,
		LastBlockTime:       header.Time(),
		AppHash:             stateRoot,
		LastHeaderHash:      header.Hash(),
		DAHeight:            s.DAHeight,
		NextProposerAddress: cloneBytes(nextProposer),
	}, nil
}

// AssertValidForNextState performs common validation of a header and data against the current state.
// It assumes any context-specific basic header checks and verifier setup have already been performed
func (s State) AssertValidForNextState(header *SignedHeader, data *Data) error {
	if err := s.AssertValidSequence(header); err != nil {
		return err
	}

	if err := Validate(header, data); err != nil {
		return fmt.Errorf("header-data validation failed: %w", err)
	}
	if len(s.NextProposerAddress) > 0 && !bytes.Equal(header.ProposerAddress, s.NextProposerAddress) {
		return fmt.Errorf("unexpected proposer - got: %x, want: %x", header.ProposerAddress, s.NextProposerAddress)
	}
	return nil
}

var (
	basedSequencerTracking sync.Once
	lastHeaderHashErrCount = 0
)

// AssertValidSequence performs lightweight state-sequence validation for self-produced blocks.
func (s State) AssertValidSequence(header *SignedHeader) error {
	if header.ChainID() != s.ChainID {
		return fmt.Errorf("invalid chain ID - got %s, want %s", header.ChainID(), s.ChainID)
	}

	if len(s.LastHeaderHash) == 0 { // initial state
		return nil
	}

	if expdHeight := s.LastBlockHeight + 1; header.Height() != expdHeight {
		return fmt.Errorf("invalid block height - got: %d, want: %d", header.Height(), expdHeight)
	}

	if headerTime := header.Time(); s.LastBlockTime.After(headerTime) {
		return fmt.Errorf("invalid block time - got: %v, last: %v", headerTime, s.LastBlockTime)
	}

	// Trick to support the switch from a base sequencer to a normal syncing node.
	// Based sequencers do not sign ehaders, meaning the last header hash will be different from the
	// newly derived header hash when a base sequencer have switched to a syncing node
	if !bytes.Equal(header.LastHeaderHash, s.LastHeaderHash) {
		if lastHeaderHashErrCount == 1 {
			return fmt.Errorf("invalid last header hash - got: %x, want: %x", header.LastHeaderHash, s.LastHeaderHash)
		}

		basedSequencerTracking.Do(func() {
			lastHeaderHashErrCount++
		})
	}

	if !bytes.Equal(header.AppHash, s.AppHash) {
		return fmt.Errorf("invalid last app hash - got: %x, want: %x", header.AppHash, s.AppHash)
	}

	return nil
}
