package execution

import (
	"bytes"
	"context"
	"crypto/sha512"
	"fmt"
	"slices"
	"sync"
	"time"
)

//---------------------
// DummyExecutor
//---------------------

// DummyExecutor is a dummy implementation of the Executor interface for testing
type DummyExecutor struct {
	mu           sync.RWMutex // Add mutex for thread safety
	stateRoot    []byte
	pendingRoots map[uint64][]byte
	injectedTxs  [][]byte
}

// NewDummyExecutor creates a new dummy DummyExecutor instance
func NewDummyExecutor() *DummyExecutor {
	return &DummyExecutor{
		stateRoot:    []byte{1, 2, 3},
		pendingRoots: make(map[uint64][]byte),
	}
}

// InitChain initializes the chain state with the given genesis time, initial height, and chain ID.
// It returns the state root hash and an error if the initialization fails.
func (e *DummyExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	hash := sha512.New()
	hash.Write(e.stateRoot)
	e.stateRoot = hash.Sum(nil)
	return e.stateRoot, nil
}

// GetTxs returns the list of transactions within the DummyExecutor instance and an error if any.
func (e *DummyExecutor) GetTxs(context.Context) ([][]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	txs := make([][]byte, len(e.injectedTxs))
	copy(txs, e.injectedTxs) // Create a copy to avoid external modifications
	return txs, nil
}

// InjectTx adds a transaction to the internal list of injected transactions in the DummyExecutor instance.
func (e *DummyExecutor) InjectTx(tx []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.injectedTxs = append(e.injectedTxs, tx)
}

// ExecuteTxs simulate execution of transactions.
func (e *DummyExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	hash := sha512.New()
	hash.Write(prevStateRoot)
	for _, tx := range txs {
		hash.Write(tx)
	}
	pending := hash.Sum(nil)
	e.pendingRoots[blockHeight] = pending
	e.removeExecutedTxs(txs)
	return pending, nil
}

// SetFinal marks block at given height as finalized.
func (e *DummyExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if pending, ok := e.pendingRoots[blockHeight]; ok {
		e.stateRoot = pending
		delete(e.pendingRoots, blockHeight)
		return nil
	}
	return fmt.Errorf("cannot set finalized block at height %d", blockHeight)
}

func (e *DummyExecutor) removeExecutedTxs(txs [][]byte) {
	e.injectedTxs = slices.DeleteFunc(e.injectedTxs, func(tx []byte) bool {
		return slices.ContainsFunc(txs, func(t []byte) bool { return bytes.Equal(tx, t) })
	})
}

// GetStateRoot returns the current state root in a thread-safe manner
func (e *DummyExecutor) GetStateRoot() []byte {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.stateRoot
}

// GetExecutionInfo returns execution layer parameters.
// For DummyExecutor, returns MaxGas=0 indicating no gas-based filtering.
func (e *DummyExecutor) GetExecutionInfo(ctx context.Context) (ExecutionInfo, error) {
	return ExecutionInfo{MaxGas: 0}, nil
}

// FilterTxs validates force-included transactions and applies size filtering.
// For DummyExecutor, no tx validation is done but size filtering is applied.
func (e *DummyExecutor) FilterTxs(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) ([]FilterStatus, error) {
	result := make([]FilterStatus, len(txs))

	var cumulativeBytes uint64
	limitReached := false

	for i, tx := range txs {
		// Skip empty transactions
		if len(tx) == 0 {
			result[i] = FilterRemove
			continue
		}

		txBytes := uint64(len(tx))

		// Skip tx that can never make it in a block (too big)
		if maxBytes > 0 && txBytes > maxBytes {
			result[i] = FilterRemove
			continue
		}

		// Once limit is reached, postpone remaining txs
		if limitReached {
			result[i] = FilterPostpone
			continue
		}

		// Check size limit
		if maxBytes > 0 && cumulativeBytes+txBytes > maxBytes {
			limitReached = true
			result[i] = FilterPostpone
			continue
		}

		cumulativeBytes += txBytes
		result[i] = FilterOK
	}

	return result, nil
}
