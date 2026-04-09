package executor

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestInitChain_Idempotency(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()
	genesisTime := time.Now()
	initialHeight := uint64(1)
	chainID := "test-chain"

	stateRoot1, err := exec.InitChain(ctx, genesisTime, initialHeight, chainID)
	if err != nil {
		t.Fatalf("InitChain failed on first call: %v", err)
	}

	stateRoot2, err := exec.InitChain(ctx, genesisTime, initialHeight, chainID)
	if err != nil {
		t.Fatalf("InitChain failed on second call: %v", err)
	}
	if !reflect.DeepEqual(stateRoot1, stateRoot2) {
		t.Errorf("Genesis state roots do not match: %x vs %x", stateRoot1, stateRoot2)
	}
}

func TestGetTxs(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()

	tx1 := []byte("a=1")
	tx2 := []byte("b=2")
	exec.InjectTx(tx1)
	exec.InjectTx(tx2)

	time.Sleep(10 * time.Millisecond)

	txs, err := exec.GetTxs(ctx)
	if err != nil {
		t.Fatalf("GetTxs returned error: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(txs))
	}

	txsAgain, err := exec.GetTxs(ctx)
	if err != nil {
		t.Fatalf("GetTxs (second call) returned error: %v", err)
	}
	if len(txsAgain) != 0 {
		t.Errorf("Expected 0 transactions on second call (drained), got %d", len(txsAgain))
	}

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
	if string(txsAfterReinject[0]) != "c=3" {
		t.Errorf("Expected tx 'c=3' after re-inject, got %s", string(txsAfterReinject[0]))
	}
}

func TestExecuteTxs_Valid(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()

	txs := [][]byte{
		[]byte("key1=value1"),
		[]byte("key2=value2"),
	}

	stateRoot, err := exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed: %v", err)
	}

	if stateRoot == nil {
		t.Fatal("Expected non-nil state root")
	}

	val, ok := exec.GetStoreValue(ctx, "key1")
	if !ok || val != "value1" {
		t.Errorf("Expected key1=value1, got %q, ok=%v", val, ok)
	}
	val, ok = exec.GetStoreValue(ctx, "key2")
	if !ok || val != "value2" {
		t.Errorf("Expected key2=value2, got %q, ok=%v", val, ok)
	}
}

func TestExecuteTxs_Invalid(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()

	txs := [][]byte{
		[]byte("invalidformat"),
		[]byte("another_invalid_one"),
		[]byte(""),
	}

	stateRoot, err := exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs should handle gibberish gracefully, got error: %v", err)
	}

	if stateRoot == nil {
		t.Error("Expected non-nil state root even with all invalid transactions")
	}

	mixedTxs := [][]byte{
		[]byte("valid_key=valid_value"),
		[]byte("invalidformat"),
		[]byte("another_valid=value2"),
		[]byte(""),
	}

	_, err = exec.ExecuteTxs(ctx, mixedTxs, 2, time.Now(), stateRoot)
	if err != nil {
		t.Fatalf("ExecuteTxs should filter invalid transactions and process valid ones, got error: %v", err)
	}

	val, ok := exec.GetStoreValue(ctx, "valid_key")
	if !ok || val != "valid_value" {
		t.Errorf("Expected valid_key=valid_value, got %q, ok=%v", val, ok)
	}
	val, ok = exec.GetStoreValue(ctx, "another_valid")
	if !ok || val != "value2" {
		t.Errorf("Expected another_valid=value2, got %q, ok=%v", val, ok)
	}
}

func TestSetFinal(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()

	err := exec.SetFinal(ctx, 1)
	if err != nil {
		t.Errorf("Expected nil error for valid blockHeight, got %v", err)
	}

	err = exec.SetFinal(ctx, 0)
	if err == nil {
		t.Error("Expected error for blockHeight 0, got nil")
	}
}

func TestReservedKeysExcludedFromAppHash(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()

	_, err := exec.InitChain(ctx, time.Now(), 1, "test-chain")
	if err != nil {
		t.Fatalf("Failed to initialize chain: %v", err)
	}

	txs := [][]byte{
		[]byte("user/key1=value1"),
		[]byte("user/key2=value2"),
	}
	_, err = exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("Failed to execute transactions: %v", err)
	}

	baselineStateRoot, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute baseline state root: %v", err)
	}

	err = exec.SetFinal(ctx, 5)
	if err != nil {
		t.Fatalf("Failed to set final height: %v", err)
	}

	finalizedHeightExists, err := exec.db.Has(ctx, finalizedHeightKey)
	if err != nil {
		t.Fatalf("Failed to check if finalizedHeight exists: %v", err)
	}
	if !finalizedHeightExists {
		t.Error("Expected finalizedHeight to exist in database")
	}

	stateRootAfterReservedKeyWrite, err := exec.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("Failed to compute state root after writing reserved key: %v", err)
	}

	if !reflect.DeepEqual(baselineStateRoot, stateRootAfterReservedKeyWrite) {
		t.Errorf("State root changed after writing reserved key:\nBefore: %x\nAfter:  %x",
			baselineStateRoot, stateRootAfterReservedKeyWrite)
	}

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

	if reflect.DeepEqual(baselineStateRoot, finalStateRoot) {
		t.Error("Expected state root to change after adding user data")
	}
}

func TestStateRootDeterministic(t *testing.T) {
	exec1 := NewKVExecutor()
	ctx := context.Background()

	txs := [][]byte{
		[]byte("alpha=1"),
		[]byte("beta=2"),
		[]byte("gamma=3"),
	}
	_, err := exec1.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed: %v", err)
	}
	root1, err := exec1.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("computeStateRoot failed: %v", err)
	}

	exec2 := NewKVExecutor()
	_, err = exec2.ExecuteTxs(ctx, [][]byte{
		[]byte("gamma=3"),
		[]byte("alpha=1"),
		[]byte("beta=2"),
	}, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed: %v", err)
	}
	root2, err := exec2.computeStateRoot(ctx)
	if err != nil {
		t.Fatalf("computeStateRoot failed: %v", err)
	}

	if !reflect.DeepEqual(root1, root2) {
		t.Errorf("State roots should be deterministic regardless of insertion order:\n%x\n%x", root1, root2)
	}
}

func TestExecuteTxsDuplicateKeys(t *testing.T) {
	exec := NewKVExecutor()
	ctx := context.Background()

	txs := [][]byte{
		[]byte("key=v1"),
		[]byte("key=v2"),
		[]byte("key=v3"),
	}

	_, err := exec.ExecuteTxs(ctx, txs, 1, time.Now(), []byte(""))
	if err != nil {
		t.Fatalf("ExecuteTxs failed: %v", err)
	}

	val, ok := exec.GetStoreValue(ctx, "key")
	if !ok || val != "v3" {
		t.Errorf("Expected key=v3 (last write wins), got %q, ok=%v", val, ok)
	}
}
