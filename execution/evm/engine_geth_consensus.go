package evm

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// maxExtraDataSize is the maximum allowed size for block extra data (32 bytes).
	maxExtraDataSize = 32

	// gasLimitBoundDivisor is the bound divisor for gas limit changes between blocks.
	// Gas limit can only change by 1/1024 per block.
	gasLimitBoundDivisor = 1024

	// minGasLimit is the minimum gas limit allowed for blocks.
	minGasLimit = 5000
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

	// Verify extra data size limit
	if len(header.Extra) > maxExtraDataSize {
		return fmt.Errorf("invalid extra data size: have %d, max %d", len(header.Extra), maxExtraDataSize)
	}

	// Verify gas limit bounds
	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, params.MaxGasLimit)
	}
	if header.GasLimit < minGasLimit {
		return fmt.Errorf("invalid gasLimit: have %v, min %v", header.GasLimit, minGasLimit)
	}

	// Verify gas limit change is within bounds (can only change by 1/1024 per block)
	diff := int64(header.GasLimit) - int64(parent.GasLimit)
	if diff < 0 {
		diff = -diff
	}
	limit := parent.GasLimit / gasLimitBoundDivisor
	if uint64(diff) >= limit {
		return fmt.Errorf("invalid gas limit: have %d, want %d ± %d", header.GasLimit, parent.GasLimit, limit-1)
	}

	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify the header's EIP-1559 attributes.
	if err := eip1559.VerifyEIP1559Header(chain.Config(), parent, header); err != nil {
		return err
	}

	// Verify EIP-4844 blob gas fields if Cancun is active
	config := chain.Config()
	if config.IsCancun(header.Number, header.Time) {
		if err := eip4844.VerifyEIP4844Header(config, parent, header); err != nil {
			return err
		}
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
