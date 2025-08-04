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

func TestSetFinalDoesNotAffectAppHash(t *testing.T) {
	exec, err := NewKVExecutor(t.TempDir(), "testdb")
	if err != nil {
		t.Fatalf("Failed to create KVExecutor: %v", err)
	}
	ctx := context.Background()

	// Initialize chain to set up genesis state
	_, _, err = exec.InitChain(ctx, time.Now(), 1, "test-chain")
	if err != nil {
		t.Fatalf("Failed to initialize chain: %v", err)
	}

	// Add some application data via transactions
	txs := [][]byte{
		[]byte("app/key1=value1"),
		[]byte("app/key2=value2"),
	}
	_, _, err = exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("Failed to execute transactions: %v", err)
	}

	// Compute state root before SetFinal
	stateRootBefore, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute state root before SetFinal: %v", err)
	}

	// Call SetFinal to write /finalizedHeight
	err = exec.SetFinal(ctx, 5)
	if err != nil {
		t.Fatalf("Failed to set final height: %v", err)
	}

	// Verify that /finalizedHeight was actually written to the database
	finalizedHeightExists, err := exec.db.Has(ctx, finalizedHeightKey)
	if err != nil {
		t.Fatalf("Failed to check if finalizedHeight exists: %v", err)
	}
	if !finalizedHeightExists {
		t.Error("Expected /finalizedHeight to exist in database after SetFinal")
	}

	// Verify the stored value
	storedValue, err := exec.db.Get(ctx, finalizedHeightKey)
	if err != nil {
		t.Fatalf("Failed to get finalizedHeight value: %v", err)
	}
	if string(storedValue) != "5" {
		t.Errorf("Expected finalizedHeight value '5', got '%s'", string(storedValue))
	}

	// Compute state root after SetFinal
	stateRootAfter, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute state root after SetFinal: %v", err)
	}

	// State root should be identical (finalizedHeight should not affect it)
	if string(stateRootBefore) != string(stateRootAfter) {
		t.Errorf("State root changed after SetFinal:\nBefore: %s\nAfter:  %s",
			string(stateRootBefore), string(stateRootAfter))
	}

	// Call SetFinal with different height
	err = exec.SetFinal(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to set final height to 10: %v", err)
	}

	// State root should still be the same
	stateRootAfterUpdate, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute state root after SetFinal update: %v", err)
	}

	if string(stateRootBefore) != string(stateRootAfterUpdate) {
		t.Errorf("State root changed after SetFinal update:\nOriginal: %s\nAfter update: %s",
			string(stateRootBefore), string(stateRootAfterUpdate))
	}

	// Verify the updated value
	updatedValue, err := exec.db.Get(ctx, finalizedHeightKey)
	if err != nil {
		t.Fatalf("Failed to get updated finalizedHeight value: %v", err)
	}
	if string(updatedValue) != "10" {
		t.Errorf("Expected updated finalizedHeight value '10', got '%s'", string(updatedValue))
	}

	// Add more application data and verify state root changes only due to app data
	moreTxs := [][]byte{
		[]byte("app/key3=value3"),
	}
	_, _, err = exec.ExecuteTxs(ctx, moreTxs, 2, time.Now(), stateRootAfterUpdate)
	if err != nil {
		t.Fatalf("Failed to execute more transactions: %v", err)
	}

	finalStateRoot, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute final state root: %v", err)
	}

	// This time state root should be different (due to new app data)
	if string(stateRootBefore) == string(finalStateRoot) {
		t.Error("Expected state root to change after adding new application data")
	}

	// But finalizedHeight should still not be included in the state root calculation
	// We can verify this by checking that the state root contains our app data
	stateRootStr := string(finalStateRoot)
	if !strings.Contains(stateRootStr, "app/key1:value1") ||
		!strings.Contains(stateRootStr, "app/key2:value2") ||
		!strings.Contains(stateRootStr, "app/key3:value3") {
		t.Errorf("State root should contain application data: %s", stateRootStr)
	}

	// But should NOT contain finalizedHeight
	if strings.Contains(stateRootStr, "finalizedHeight") {
		t.Errorf("State root should NOT contain finalizedHeight data: %s", stateRootStr)
	}
}
