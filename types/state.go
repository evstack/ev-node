package types

import (
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

	// DAHeight identifies DA block containing the latest applied Evolve block.
	DAHeight uint64

	// Merkle root of the results from executing prev block
	LastResultsHash Hash

	// the latest AppHash we've received from calling abci.Commit()
	AppHash []byte
}

func (s *State) NextState(header Header, stateRoot []byte) (State, error) {
	height := header.Height()

	return State{
		Version:         s.Version,
		ChainID:         s.ChainID,
		InitialHeight:   s.InitialHeight,
		LastBlockHeight: height,
		LastBlockTime:   header.Time(),
		AppHash:         stateRoot,
		DAHeight:        s.DAHeight,
	}, nil
}
