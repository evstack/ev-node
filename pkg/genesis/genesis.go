package genesis

import (
	"fmt"
	"time"
)

const ChainIDFlag = "chain_id"

// Genesis represents the genesis state of the blockchain.
// This genesis struct only contains the fields required by evolve.
// The app state or other fields are not included here.
type Genesis struct {
	ChainID         string    `json:"chain_id"`
	StartTime       time.Time `json:"start_time"`
	InitialHeight   uint64    `json:"initial_height"`
	ProposerAddress []byte    `json:"proposer_address"`
	DAStartHeight   uint64    `json:"da_start_height"`
}

// NewGenesis creates a new Genesis instance.
func NewGenesis(
	chainID string,
	initialHeight uint64,
	startTime time.Time,
	proposerAddress []byte,
) Genesis {
	genesis := Genesis{
		ChainID:         chainID,
		StartTime:       startTime,
		InitialHeight:   initialHeight,
		ProposerAddress: proposerAddress,
		DAStartHeight:   0,
	}

	return genesis
}

// Validate checks if the Genesis object is valid.
func (g Genesis) Validate() error {
	if g.ChainID == "" {
		return fmt.Errorf("invalid or missing chain_id in genesis file")
	}

	if g.InitialHeight < 1 {
		return fmt.Errorf("initial_height must be at least 1, got %d", g.InitialHeight)
	}

	if g.StartTime.IsZero() {
		return fmt.Errorf("start_time cannot be zero time")
	}

	if g.ProposerAddress == nil {
		return fmt.Errorf("proposer_address cannot be nil")
	}

	return nil
}
