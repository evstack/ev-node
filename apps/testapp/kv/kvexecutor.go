package executor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/evstack/ev-node/pkg/store"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

var (
	genesisInitializedKey = ds.NewKey("/genesis/initialized")
	genesisStateRootKey   = ds.NewKey("/genesis/stateroot")
	heightKeyPrefix       = ds.NewKey("/height")
	finalizedHeightKey    = ds.NewKey("/finalizedHeight")
	// Define a buffer size for the transaction channel
	txChannelBufferSize = 10000
	// reservedKeys defines the set of keys that should be excluded from state root calculation
	// and protected from transaction modifications
	reservedKeys = map[ds.Key]bool{
		genesisInitializedKey: true,
		genesisStateRootKey:   true,
		finalizedHeightKey:    true,
	}
)

// KVExecutor is a simple key-value store backed by go-datastore that implements the Executor interface
// for testing purposes. It uses a buffered channel as a mempool for transactions.
// It also includes fields to track genesis initialization persisted in the datastore.
type KVExecutor struct {
	db     ds.Batching
	txChan chan []byte // Buffered channel for transactions
}

// NewKVExecutor creates a new instance of KVExecutor with initialized store and mempool channel.
func NewKVExecutor(rootdir, dbpath string) (*KVExecutor, error) {
	datastore, err := store.NewDefaultKVStore(rootdir, dbpath, "executor")
	if err != nil {
		return nil, err
	}
	return &KVExecutor{
		db:     datastore,
		txChan: make(chan []byte, txChannelBufferSize),
	}, nil
}

// GetStoreValue is a helper for the HTTP interface to retrieve the value for a key from the database.
// It searches across all block heights to find the latest value for the given key.
func (k *KVExecutor) GetStoreValue(ctx context.Context, key string) (string, bool) {
	// Query all keys to find height-prefixed versions of this key
	q := query.Query{}
	results, err := k.db.Query(ctx, q)
	if err != nil {
		fmt.Printf("Error querying DB for key '%s': %v\n", key, err)
		return "", false
	}
	defer results.Close()

	heightPrefix := heightKeyPrefix.String()
	var latestValue string
	var latestHeight uint64
	found := false

	for result := range results.Next() {
		if result.Error != nil {
			fmt.Printf("Error iterating query results for key '%s': %v\n", key, result.Error)
			return "", false
		}

		resultKey := result.Key
		// Check if this is a height-prefixed key that matches our target key
		if strings.HasPrefix(resultKey, heightPrefix+"/") {
			// Extract height and actual key: /height/{height}/{actual_key}
			parts := strings.Split(strings.TrimPrefix(resultKey, heightPrefix+"/"), "/")
			if len(parts) >= 2 {
				var keyHeight uint64
				if _, err := fmt.Sscanf(parts[0], "%d", &keyHeight); err == nil {
					// Reconstruct the actual key by joining all parts after the height
					actualKey := strings.Join(parts[1:], "/")
					if actualKey == key {
						// This key matches - check if it's the latest height
						if !found || keyHeight > latestHeight {
							latestHeight = keyHeight
							latestValue = string(result.Value)
							found = true
						}
					}
				}
			}
		}
	}

	if !found {
		return "", false
	}

	return latestValue, true
}

// computeStateRoot computes a deterministic state root by querying all keys, sorting them,
// and concatenating key-value pairs from the database.
func (k *KVExecutor) computeStateRoot(ctx context.Context) ([]byte, error) {
	q := query.Query{KeysOnly: true}
	results, err := k.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys for state root: %w", err)
	}
	defer results.Close()

	keys := make([]string, 0)
	for result := range results.Next() {
		if result.Error != nil {
			return nil, fmt.Errorf("error iterating query results: %w", result.Error)
		}
		// Exclude reserved keys from the state root calculation
		dsKey := ds.NewKey(result.Key)
		if reservedKeys[dsKey] {
			continue
		}
		keys = append(keys, result.Key)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, key := range keys {
		valueBytes, err := k.db.Get(ctx, ds.NewKey(key))
		if err != nil {
			// This shouldn't happen if the key came from the query, but handle defensively
			return nil, fmt.Errorf("failed to get value for key '%s' during state root computation: %w", key, err)
		}
		sb.WriteString(fmt.Sprintf("%s:%s;", key, string(valueBytes)))
	}
	return []byte(sb.String()), nil
}

// InitChain initializes the chain state with genesis parameters.
// It checks the database to see if genesis was already performed.
// If not, it computes the state root from the current DB state and persists genesis info.
func (k *KVExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, uint64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	initialized, err := k.db.Has(ctx, genesisInitializedKey)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to check genesis initialization status: %w", err)
	}

	if initialized {
		genesisRoot, err := k.db.Get(ctx, genesisStateRootKey)
		if err != nil {
			return nil, 0, fmt.Errorf("genesis initialized but failed to retrieve state root: %w", err)
		}
		return genesisRoot, 1024, nil // Assuming 1024 is a constant gas value
	}

	// Genesis not initialized. Compute state root from the current DB state.
	// Note: The DB might not be empty if restarting, this reflects the state *at genesis time*.
	stateRoot, err := k.computeStateRoot(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compute initial state root for genesis: %w", err)
	}

	// Persist genesis state root and initialized flag
	batch, err := k.db.Batch(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create batch for genesis persistence: %w", err)
	}
	err = batch.Put(ctx, genesisStateRootKey, stateRoot)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to put genesis state root in batch: %w", err)
	}
	err = batch.Put(ctx, genesisInitializedKey, []byte("true")) // Store a marker value
	if err != nil {
		return nil, 0, fmt.Errorf("failed to put genesis initialized flag in batch: %w", err)
	}
	err = batch.Commit(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to commit genesis persistence batch: %w", err)
	}

	return stateRoot, 1024, nil // Assuming 1024 is a constant gas value
}

// GetTxs retrieves available transactions from the mempool channel.
// It drains the channel in a non-blocking way.
func (k *KVExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Drain the channel efficiently
	txs := make([][]byte, 0, len(k.txChan)) // Pre-allocate roughly
	for {
		select {
		case tx := <-k.txChan:
			txs = append(txs, tx)
		default: // Channel is empty or context is done
			// Check context again in case it was cancelled during drain
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				if len(txs) == 0 {
					return nil, nil // Return nil slice if no transactions were retrieved
				}
				return txs, nil
			}
		}
	}
}

// ExecuteTxs processes each transaction assumed to be in the format "key=value".
// It updates the database accordingly using a batch and removes the executed transactions from the mempool.
// If a transaction is malformed, an error is returned, and the database is not changed.
func (k *KVExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, uint64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	batch, err := k.db.Batch(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create database batch: %w", err)
	}

	// Process transactions and stage them in the batch
	for _, tx := range txs {
		parts := strings.SplitN(string(tx), "=", 2)
		if len(parts) != 2 {
			return nil, 0, errors.New("malformed transaction; expected format key=value")
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, 0, errors.New("empty key in transaction")
		}

		dsKey := getTxKey(blockHeight, key)

		// Prevent writing reserved keys via transactions
		if reservedKeys[dsKey] {
			return nil, 0, fmt.Errorf("transaction attempts to modify reserved key: %s", key)
		}

		err = batch.Put(ctx, dsKey, []byte(value))
		if err != nil {
			// This error is unlikely for Put unless the context is cancelled.
			return nil, 0, fmt.Errorf("failed to stage put operation in batch for key '%s': %w", key, err)
		}
	}

	// Commit the batch to apply all changes atomically
	err = batch.Commit(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to commit transaction batch: %w", err)
	}

	// Compute the new state root *after* successful commit
	stateRoot, err := k.computeStateRoot(ctx)
	if err != nil {
		// This is problematic, state was changed but root calculation failed.
		// May need more robust error handling or recovery logic.
		return nil, 0, fmt.Errorf("failed to compute state root after executing transactions: %w", err)
	}

	return stateRoot, 1024, nil
}

// SetFinal marks a block as finalized at the specified height.
func (k *KVExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate blockHeight
	if blockHeight == 0 {
		return errors.New("invalid blockHeight: cannot be zero")
	}

	return k.db.Put(ctx, finalizedHeightKey, []byte(fmt.Sprintf("%d", blockHeight)))
}

// InjectTx adds a transaction to the mempool channel.
// Uses a non-blocking send to avoid blocking the caller if the channel is full.
func (k *KVExecutor) InjectTx(tx []byte) {
	select {
	case k.txChan <- tx:
		// Transaction successfully sent to channel
	default:
		// Channel is full, transaction is dropped. Log this event.
		fmt.Printf("Warning: Transaction channel buffer full. Dropping transaction.\n")
		// Consider adding metrics here
	}
}

// Rollback reverts the state to the previous block height.
func (k *KVExecutor) Rollback(ctx context.Context, height uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate height constraints
	if height == 0 {
		return fmt.Errorf("cannot rollback to height 0: invalid height")
	}

	// Create a batch for atomic rollback operation
	batch, err := k.db.Batch(ctx)
	if err != nil {
		return fmt.Errorf("failed to create batch for rollback: %w", err)
	}

	// Query all keys to find those with height > target height
	q := query.Query{}
	results, err := k.db.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("failed to query keys for rollback: %w", err)
	}
	defer results.Close()

	keysToDelete := make([]ds.Key, 0)
	heightPrefix := heightKeyPrefix.String()

	for result := range results.Next() {
		if result.Error != nil {
			return fmt.Errorf("error iterating query results during rollback: %w", result.Error)
		}

		key := result.Key
		// Check if this is a height-prefixed key
		if strings.HasPrefix(key, heightPrefix+"/") {
			// Extract height from key: /height/{height}/{actual_key} (see getTxKey)
			parts := strings.Split(strings.TrimPrefix(key, heightPrefix+"/"), "/")
			if len(parts) > 0 {
				var keyHeight uint64
				if _, err := fmt.Sscanf(parts[0], "%d", &keyHeight); err == nil {
					// If this key's height is greater than target, mark for deletion
					if keyHeight > height {
						keysToDelete = append(keysToDelete, ds.NewKey(key))
					}
				}
			}
		}
	}

	// Delete all keys with height > target height
	for _, key := range keysToDelete {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = batch.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to stage delete operation for key '%s' during rollback: %w", key.String(), err)
		}
	}

	// Update finalized height if necessary - it should not exceed rollback height
	finalizedHeightKey := ds.NewKey("/finalizedHeight")
	if finalizedHeightBytes, err := k.db.Get(ctx, finalizedHeightKey); err == nil {
		var finalizedHeight uint64
		if _, err := fmt.Sscanf(string(finalizedHeightBytes), "%d", &finalizedHeight); err == nil {
			if finalizedHeight > height {
				err = batch.Put(ctx, finalizedHeightKey, fmt.Appendf([]byte{}, "%d", height))
				if err != nil {
					return fmt.Errorf("failed to update finalized height during rollback: %w", err)
				}
			}
		}
	}

	// Commit the batch atomically
	err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit rollback batch: %w", err)
	}

	return nil
}

func getTxKey(height uint64, txKey string) ds.Key {
	return heightKeyPrefix.Child(ds.NewKey(fmt.Sprintf("%d/%s", height, txKey)))
}
