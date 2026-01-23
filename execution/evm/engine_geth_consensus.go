package evm

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// sovereignBeacon wraps the standard beacon consensus engine but allows
// equal timestamps (timestamp >= parent.timestamp) instead of requiring
// strictly increasing timestamps (timestamp > parent.timestamp).
// This enables subsecond block times for sovereign rollups while keeping
// all other beacon consensus rules intact.
type sovereignBeacon struct {
	consensus.Engine
}

// newSovereignBeacon creates a beacon consensus engine that allows equal timestamps.
func newSovereignBeacon() *sovereignBeacon {
	return &sovereignBeacon{
		Engine: beacon.New(nil),
	}
}

// VerifyHeader checks whether a header conforms to the consensus rules.
// This override allows equal timestamps for subsecond block times.
func (sb *sovereignBeacon) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Get parent header
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	// Check timestamp - allow equal (>=) instead of strictly greater (>)
	if header.Time < parent.Time {
		return errors.New("invalid timestamp: must be >= parent timestamp")
	}

	// Verify difficulty is zero (PoS requirement)
	if header.Difficulty.Cmp(common.Big0) != 0 {
		return errors.New("invalid difficulty: must be zero for PoS")
	}

	// Verify nonce is zero (PoS requirement)
	if header.Nonce != (types.BlockNonce{}) {
		return errors.New("invalid nonce: must be zero for PoS")
	}

	// Verify uncle hash is empty (PoS requirement)
	if header.UncleHash != types.EmptyUncleHash {
		return errors.New("invalid uncle hash: must be empty for PoS")
	}

	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, params.MaxGasLimit)
	}

	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify the header's EIP-1559 attributes.
	if err := eip1559.VerifyEIP1559Header(chain.Config(), parent, header); err != nil {
		return err
	}

	return nil
}

// VerifyHeaders verifies a batch of headers concurrently.
func (sb *sovereignBeacon) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for _, header := range headers {
			select {
			case <-abort:
				return
			case results <- sb.VerifyHeader(chain, header):
			}
		}
	}()

	return abort, results
}
