package executor

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
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
	stateRoot1, err := exec.InitChain(ctx, genesisTime, initialHeight, chainID)
	if err != nil {
		t.Fatalf("InitChain failed on first call: %v", err)
	}

	// Second call should return the same genesis state root
	stateRoot2, err := exec.InitChain(ctx, genesisTime, initialHeight, chainID)
	if err != nil {
		t.Fatalf("InitChain failed on second call: %v", err)
	}
	if !bytes.Equal(stateRoot1, stateRoot2) {
		t.Errorf("Genesis state roots do not match: %s vs %s", stateRoot1, stateRoot2)
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

	txs, err := exec.GetTxs(ctx)
	if err != nil {
		t.Fatalf("GetTxs returned error: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(txs))
	}
	if !reflect.DeepEqual(txs[0], tx1) {
		t.Errorf("Expected first tx 'a=1', got %s", string(txs[0]))
	}
	if !reflect.DeepEqual(txs[1], tx2) {
		t.Errorf("Expected second tx 'b=2', got %s", string(txs[1]))
	}

	// GetTxs should drain the channel, so a second call should return empty or nil
	txsAgain, err := exec.GetTxs(ctx)
	if err != nil {
		t.Fatalf("GetTxs (second call) returned error: %v", err)
	}
	if len(txsAgain) != 0 {
		t.Errorf("Expected 0 transactions on second call (drained), got %d", len(txsAgain))
	}

	// Inject another transaction and verify it's available
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

	stateRoot, err := exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed: %v", err)
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

	// According to the Executor interface: "Must handle gracefully gibberish transactions"
	// Invalid transactions should be filtered out, not cause errors

	// Prepare invalid transactions (missing '=')
	txs := [][]byte{
		[]byte("invalidformat"),
		[]byte("another_invalid_one"),
		[]byte(""),
	}

	stateRoot, err := exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs should handle gibberish gracefully, got error: %v", err)
	}

	// State root should still be computed (empty block is valid)
	if stateRoot == nil {
		t.Error("Expected non-nil state root even with all invalid transactions")
	}

	// Test mix of valid and invalid transactions
	mixedTxs := [][]byte{
		[]byte("valid_key=valid_value"),
		[]byte("invalidformat"),
		[]byte("another_valid=value2"),
		[]byte(""),
	}

	stateRoot2, err := exec.ExecuteTxs(ctx, mixedTxs, 2, time.Now(), stateRoot)
	if err != nil {
		t.Fatalf("ExecuteTxs should filter invalid transactions and process valid ones, got error: %v", err)
	}

	// State root should contain only the valid transactions
	rootStr := string(stateRoot2)
	if !strings.Contains(rootStr, "valid_key:valid_value") || !strings.Contains(rootStr, "another_valid:value2") {
		t.Errorf("State root should contain valid transactions: %s", rootStr)
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

func TestReservedKeysExcludedFromAppHash(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Initialize chain to set up genesis state (this writes genesis reserved keys)
	_, err = exec.InitChain(ctx, time.Now(), 1, "test-chain")
	if err != nil {
		t.Fatalf("Failed to initialize chain: %v", err)
	}

	// Add some application data
	txs := [][]byte{
		[]byte("user/key1=value1"),
		[]byte("user/key2=value2"),
	}
	_, err = exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("Failed to execute transactions: %v", err)
	}

	// Compute baseline state root
	baselineStateRoot, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute baseline state root: %v", err)
	}

	// Write to finalizedHeight (a reserved key)
	err = exec.SetFinal(ctx, 5)
	if err != nil {
		t.Fatalf("Failed to set final height: %v", err)
	}

	// Verify finalizedHeight was written
	finalizedHeightExists, err := exec.db.Has(ctx, finalizedHeightKey)
	if err != nil {
		t.Fatalf("Failed to check if finalizedHeight exists: %v", err)
	}
	if !finalizedHeightExists {
		t.Error("Expected finalizedHeight to exist in database")
	}

	// State root should be unchanged (reserved keys excluded from calculation)
	stateRootAfterReservedKeyWrite, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute state root after writing reserved key: %v", err)
	}

	if string(baselineStateRoot) != string(stateRootAfterReservedKeyWrite) {
		t.Errorf("State root changed after writing reserved key:\nBefore: %s\nAfter:  %s",
			string(baselineStateRoot), string(stateRootAfterReservedKeyWrite))
	}

	// Verify state root contains only user data, not reserved keys
	stateRootStr := string(stateRootAfterReservedKeyWrite)
	if !strings.Contains(stateRootStr, "user/key1:value1") ||
		!strings.Contains(stateRootStr, "user/key2:value2") {
		t.Errorf("State root should contain user data: %s", stateRootStr)
	}

	// Verify reserved keys are NOT in state root
	for key := range reservedKeys {
		keyStr := key.String()
		if strings.Contains(stateRootStr, keyStr) {
			t.Errorf("State root should NOT contain reserved key %s: %s", keyStr, stateRootStr)
		}
	}

	// Verify that adding user data DOES change the state root
	moreTxs := [][]byte{
		[]byte("user/key3=value3"),
	}
	_, err = exec.ExecuteTxs(ctx, moreTxs, 2, time.Now(), stateRootAfterReservedKeyWrite)
	if err != nil {
		t.Fatalf("Failed to execute more transactions: %v", err)
	}

	finalStateRoot, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute final state root: %v", err)
	}

	if string(baselineStateRoot) == string(finalStateRoot) {
		t.Error("Expected state root to change after adding user data")
	}
}
