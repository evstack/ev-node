package execution

import (
	"context"
	"time"
)

// ExecutionInfo contains execution layer parameters that may change per block.
type ExecutionInfo struct {
	// MaxGas is the maximum gas allowed for transactions in a block.
	// For non-gas-based execution layers, this should be 0.
	MaxGas uint64
}

// Executor defines the interface that execution clients must implement to be compatible with Evolve.
// This interface enables the separation between consensus and execution layers, allowing for modular
// and pluggable execution environments.
//
// Note: if you are modifying this interface, ensure that all implementations are compatible (evm, abci, protobuf/grpc, etc.)
type Executor interface {
	// InitChain initializes a new blockchain instance with genesis parameters.
	// Requirements:
	// - Must generate initial state root representing empty/genesis state
	// - Must validate and store genesis parameters for future reference
	// - Must ensure idempotency (repeated calls with identical parameters should return same results)
	// - Must return error if genesis parameters are invalid
	//
	// Parameters:
	// - ctx: Context for timeout/cancellation control
	// - genesisTime: timestamp marking chain start time in UTC
	// - initialHeight: First block height (must be > 0)
	// - chainID: Unique identifier string for the blockchain
	//
	// Returns:
	// - stateRoot: Hash representing initial state
	// - err: Any initialization errors
	InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) (stateRoot []byte, err error)

	// GetTxs fetches available transactions from the execution layer's mempool.
	// Requirements:
	// - Must return currently valid transactions only
	// - Must handle empty mempool case gracefully
	// - Must respect context cancellation/timeout
	// - Should perform basic transaction validation
	// - Should not remove transactions from mempool
	// - May remove invalid transactions from mempool
	//
	// Parameters:
	// - ctx: Context for timeout/cancellation control
	//
	// Returns:
	// - [][]byte: Slice of valid transactions
	// - error: Any errors during transaction retrieval
	GetTxs(ctx context.Context) ([][]byte, error)

	// ExecuteTxs processes transactions to produce a new block state.
	// Requirements:
	// - Must validate state transition against previous state root
	// - Must handle empty transaction list
	// - Must handle gracefully gibberish transactions
	// - Must maintain deterministic execution
	// - Must respect context cancellation/timeout
	// - The rest of the rules are defined by the specific execution layer
	//
	// Parameters:
	// - ctx: Context for timeout/cancellation control
	// - txs: Ordered list of transactions to execute
	// - blockHeight: Height of block being created (must be > 0)
	// - timestamp: Block creation time in UTC
	// - prevStateRoot: Previous block's state root hash
	//
	// Returns:
	// - updatedStateRoot: New state root after executing transactions
	// - err: Any execution errors
	ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) (updatedStateRoot []byte, err error)

	// SetFinal marks a block as finalized at the specified height.
	// Requirements:
	// - Must verify block exists at specified height
	// - Must be idempotent
	// - Must maintain finality guarantees (no reverting finalized blocks)
	// - Must respect context cancellation/timeout
	// - Should clean up any temporary state/resources
	//
	// Parameters:
	// - ctx: Context for timeout/cancellation control
	// - blockHeight: Height of block to finalize
	//
	// Returns:
	// - error: Any errors during finalization
	SetFinal(ctx context.Context, blockHeight uint64) error

	// GetExecutionInfo returns current execution layer parameters.
	// The height parameter allows querying info for a specific block height.
	// Use height=0 to get parameters for the next block (based on latest state).
	//
	// For non-gas-based execution layers, return ExecutionInfo{MaxGas: 0}.
	//
	// Parameters:
	// - ctx: Context for timeout/cancellation control
	// - height: Block height to query (0 for next block parameters)
	//
	// Returns:
	// - info: Current execution parameters
	// - error: Any errors during retrieval
	GetExecutionInfo(ctx context.Context, height uint64) (ExecutionInfo, error)

	// FilterDATransactions validates and filters force-included transactions from DA.
	// It performs execution-layer specific validation (e.g., EVM tx parsing, gas checks)
	// and returns transactions that are valid and fit within the gas limit.
	//
	// The function filters out:
	// - Invalid/unparseable transactions (gibberish)
	// - Transactions that would exceed the cumulative gas limit
	//
	// For non-gas-based execution layers, return all valid transactions with nil remainingTxs.
	//
	// Parameters:
	// - ctx: Context for timeout/cancellation control
	// - txs: Raw transactions from DA to validate
	// - maxGas: Maximum cumulative gas allowed for these transactions
	//
	// Returns:
	// - validTxs: Transactions that passed validation and fit within maxGas
	// - remainingTxs: Valid transactions that didn't fit due to gas limit (for re-queuing)
	// - err: Any errors during filtering (not validation errors, which result in filtering)
	FilterDATransactions(ctx context.Context, txs [][]byte, maxGas uint64) (validTxs [][]byte, remainingTxs [][]byte, err error)
}

// HeightProvider is an optional interface that execution clients can implement
// to support height synchronization checks between ev-node and the execution layer.
type HeightProvider interface {
	// GetLatestHeight returns the current block height of the execution layer.
	// This is useful for detecting desynchronization between ev-node and the execution layer
	// after crashes or restarts.
	//
	// Parameters:
	// - ctx: Context for timeout/cancellation control
	//
	// Returns:
	// - height: Current block height of the execution layer
	// - error: Any errors during height retrieval
	GetLatestHeight(ctx context.Context) (uint64, error)
}
