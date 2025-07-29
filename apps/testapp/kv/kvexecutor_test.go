package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	ds "github.com/ipfs/go-datastore"
)

func TestInitChain_Idempotency(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()
	genesisTime := time.Now()
	initialHeight := uint64(1)
	chainID := "test-chain"

	// First call initializes genesis state
	stateRoot1, maxBytes1, err := exec.InitChain(ctx, genesisTime, initialHeight, chainID)
	if err != nil {
		t.Fatalf("InitChain failed on first call: %v", err)
	}
	if maxBytes1 != 1024 {
		t.Errorf("Expected maxBytes 1024, got %d", maxBytes1)
	}

	// Second call should return the same genesis state root
	stateRoot2, maxBytes2, err := exec.InitChain(ctx, genesisTime, initialHeight, chainID)
	if err != nil {
		t.Fatalf("InitChain failed on second call: %v", err)
	}
	if !bytes.Equal(stateRoot1, stateRoot2) {
		t.Errorf("Genesis state roots do not match: %s vs %s", stateRoot1, stateRoot2)
	}
	if maxBytes2 != 1024 {
		t.Errorf("Expected maxBytes 1024, got %d", maxBytes2)
	}
}

func TestGetTxs(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Inject transactions using the InjectTx method which sends to the channel
	tx1 := []byte("a=1")
	tx2 := []byte("b=2")
	exec.InjectTx(tx1)
	exec.InjectTx(tx2)

	// Allow a brief moment for transactions to be processed by the channel if needed,
	// though for buffered channels it should be immediate unless full.
	time.Sleep(10 * time.Millisecond)

	// First call to GetTxs should retrieve the injected transactions
	txs, err := exec.GetTxs(ctx)
	if err != nil {
		t.Fatalf("GetTxs returned error on first call: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("Expected 2 transactions on first call, got %d", len(txs))
	}

	// Verify the content (order might not be guaranteed depending on channel internals, but likely FIFO here)
	foundTx1 := false
	foundTx2 := false
	for _, tx := range txs {
		if reflect.DeepEqual(tx, tx1) {
			foundTx1 = true
		}
		if reflect.DeepEqual(tx, tx2) {
			foundTx2 = true
		}
	}
	if !foundTx1 || !foundTx2 {
		t.Errorf("Did not retrieve expected transactions. Got: %v", txs)
	}

	// Second call to GetTxs should return no transactions as the channel was drained
	txsAfterDrain, err := exec.GetTxs(ctx)
	if err != nil {
		t.Fatalf("GetTxs returned error on second call: %v", err)
	}
	if len(txsAfterDrain) != 0 {
		t.Errorf("Expected 0 transactions after drain, got %d", len(txsAfterDrain))
	}

	// Test injecting again after draining
	tx3 := []byte("c=3")
	exec.InjectTx(tx3)
	time.Sleep(10 * time.Millisecond)
	txsAfterReinject, err := exec.GetTxs(ctx)
	if err != nil {
		t.Fatalf("GetTxs returned error after re-inject: %v", err)
	}
	if len(txsAfterReinject) != 1 {
		t.Errorf("Expected 1 transaction after re-inject, got %d", len(txsAfterReinject))
	}
	if !reflect.DeepEqual(txsAfterReinject[0], tx3) {
		t.Errorf("Expected tx 'c=3' after re-inject, got %s", string(txsAfterReinject[0]))
	}
}

func TestExecuteTxs_Valid(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Prepare valid transactions
	txs := [][]byte{
		[]byte("key1=value1"),
		[]byte("key2=value2"),
	}

	stateRoot, maxBytes, err := exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed: %v", err)
	}
	if maxBytes != 1024 {
		t.Errorf("Expected maxBytes 1024, got %d", maxBytes)
	}

	// Check that stateRoot contains the updated key-value pairs
	rootStr := string(stateRoot)
	if !strings.Contains(rootStr, "key1:value1;") || !strings.Contains(rootStr, "key2:value2;") {
		t.Errorf("State root does not contain expected key-values: %s", rootStr)
	}
}

func TestExecuteTxs_Invalid(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Prepare an invalid transaction (missing '=')
	txs := [][]byte{
		[]byte("invalidformat"),
	}

	_, _, err = exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err == nil {
		t.Fatal("Expected error for malformed transaction, got nil")
	}
}

func TestSetFinal(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Test with valid blockHeight
	err = exec.SetFinal(ctx, 1)
	if err != nil {
		t.Errorf("Expected nil error for valid blockHeight, got %v", err)
	}

	// Test with invalid blockHeight (zero)
	err = exec.SetFinal(ctx, 0)
	if err == nil {
		t.Error("Expected error for blockHeight 0, got nil")
	}
}

func TestRollback_Success(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Execute transactions at different heights
	txsHeight1 := [][]byte{
		[]byte("key1=value1"),
		[]byte("key2=value2"),
	}
	_, _, err = exec.ExecuteTxs(ctx, txsHeight1, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed for height 1: %v", err)
	}

	txsHeight2 := [][]byte{
		[]byte("key3=value3"),
		[]byte("key4=value4"),
	}
	_, _, err = exec.ExecuteTxs(ctx, txsHeight2, 2, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed for height 2: %v", err)
	}

	txsHeight3 := [][]byte{
		[]byte("key5=value5"),
	}
	_, _, err = exec.ExecuteTxs(ctx, txsHeight3, 3, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed for height 3: %v", err)
	}

	// Verify all keys exist before rollback
	if value, exists := exec.GetStoreValue(ctx, "/height/1/key1"); !exists || value != "value1" {
		t.Errorf("Expected key1=value1 at height 1, got exists=%v, value=%s", exists, value)
	}
	if value, exists := exec.GetStoreValue(ctx, "/height/2/key3"); !exists || value != "value3" {
		t.Errorf("Expected key3=value3 at height 2, got exists=%v, value=%s", exists, value)
	}
	if value, exists := exec.GetStoreValue(ctx, "/height/3/key5"); !exists || value != "value5" {
		t.Errorf("Expected key5=value5 at height 3, got exists=%v, value=%s", exists, value)
	}

	// Rollback to height 2
	err = exec.Rollback(ctx, 2)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify keys at height 1 and 2 still exist
	if value, exists := exec.GetStoreValue(ctx, "/height/1/key1"); !exists || value != "value1" {
		t.Errorf("Expected key1=value1 at height 1 after rollback, got exists=%v, value=%s", exists, value)
	}
	if value, exists := exec.GetStoreValue(ctx, "/height/2/key3"); !exists || value != "value3" {
		t.Errorf("Expected key3=value3 at height 2 after rollback, got exists=%v, value=%s", exists, value)
	}

	// Verify keys at height 3 are removed
	if value, exists := exec.GetStoreValue(ctx, "/height/3/key5"); exists {
		t.Errorf("Expected key5 at height 3 to be removed after rollback, but got value=%s", value)
	}
}

func TestRollback_InvalidHeight(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Test rollback to height 0
	err = exec.Rollback(ctx, 0)
	if err == nil {
		t.Error("Expected error for rollback to height 0, got nil")
	}
	if !strings.Contains(err.Error(), "cannot rollback to height 0") {
		t.Errorf("Expected error message about invalid height 0, got: %v", err)
	}
}

func TestRollback_MultipleHeights(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Execute transactions at heights 1-5
	for height := uint64(1); height <= 5; height++ {
		txs := [][]byte{
			[]byte(fmt.Sprintf("key%d=value%d", height, height)),
		}
		_, _, err = exec.ExecuteTxs(ctx, txs, height, time.Now(), []byte(""))
		if err != nil {
			t.Fatalf("ExecuteTxs failed for height %d: %v", height, err)
		}
	}

	// Rollback to height 2
	err = exec.Rollback(ctx, 2)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify keys at heights 1-2 still exist
	for height := uint64(1); height <= 2; height++ {
		key := fmt.Sprintf("/height/%d/key%d", height, height)
		expectedValue := fmt.Sprintf("value%d", height)
		if value, exists := exec.GetStoreValue(ctx, key); !exists || value != expectedValue {
			t.Errorf("Expected %s=%s after rollback, got exists=%v, value=%s", key, expectedValue, exists, value)
		}
	}

	// Verify keys at heights 3-5 are removed
	for height := uint64(3); height <= 5; height++ {
		key := fmt.Sprintf("/height/%d/key%d", height, height)
		if value, exists := exec.GetStoreValue(ctx, key); exists {
			t.Errorf("Expected key at %s to be removed after rollback, but got value=%s", key, value)
		}
	}
}

func TestRollback_WithFinalizedHeight(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Execute transactions at heights 1-3
	for height := uint64(1); height <= 3; height++ {
		txs := [][]byte{
			[]byte(fmt.Sprintf("key%d=value%d", height, height)),
		}
		_, _, err = exec.ExecuteTxs(ctx, txs, height, time.Now(), []byte(""))
		if err != nil {
			t.Fatalf("ExecuteTxs failed for height %d: %v", height, err)
		}
	}

	// Set finalized height to 3
	err = exec.SetFinal(ctx, 3)
	if err != nil {
		t.Fatalf("SetFinal failed: %v", err)
	}

	// Rollback to height 1
	err = exec.Rollback(ctx, 1)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify finalized height was updated to rollback height
	finalizedHeightBytes, err := exec.db.Get(ctx, ds.NewKey("/finalizedHeight"))
	if err != nil {
		t.Fatalf("Failed to get finalized height: %v", err)
	}
	finalizedHeight := string(finalizedHeightBytes)
	if finalizedHeight != "1" {
		t.Errorf("Expected finalized height to be updated to 1, got %s", finalizedHeight)
	}
}

func TestRollback_EmptyState(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Rollback on empty state should succeed
	err = exec.Rollback(ctx, 1)
	if err != nil {
		t.Errorf("Rollback on empty state should succeed, got error: %v", err)
	}
}

func TestRollback_ContextCancellation(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Rollback should respect context cancellation
	err = exec.Rollback(ctx, 1)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestRollback_StateRootAfterRollback(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Execute transactions at height 1
	txsHeight1 := [][]byte{
		[]byte("key1=value1"),
		[]byte("key2=value2"),
	}
	stateRoot1, _, err := exec.ExecuteTxs(ctx, txsHeight1, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed for height 1: %v", err)
	}

	// Execute transactions at height 2
	txsHeight2 := [][]byte{
		[]byte("key3=value3"),
	}
	stateRoot2, _, err := exec.ExecuteTxs(ctx, txsHeight2, 2, time.Now(), stateRoot1)
	if err != nil {
		t.Fatalf("ExecuteTxs failed for height 2: %v", err)
	}

	// Verify state root changed after height 2
	if bytes.Equal(stateRoot1, stateRoot2) {
		t.Error("State root should change after executing transactions at height 2")
	}

	// Rollback to height 1
	err = exec.Rollback(ctx, 1)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Compute state root after rollback - should match height 1 state
	newStateRoot, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute state root after rollback: %v", err)
	}

	// State root after rollback should match original height 1 state
	if !bytes.Equal(stateRoot1, newStateRoot) {
		t.Errorf("State root after rollback should match height 1 state, got different roots")
	}

	// Verify height 2 key is no longer in state root
	rootStr := string(newStateRoot)
	if strings.Contains(rootStr, "key3:value3;") {
		t.Error("State root should not contain height 2 transactions after rollback")
	}

	// Verify height 1 keys are still in state root
	if !strings.Contains(rootStr, "key1:value1;") || !strings.Contains(rootStr, "key2:value2;") {
		t.Error("State root should still contain height 1 transactions after rollback")
	}
}
