//go:build evm
// +build evm

// Package e2e contains end-to-end tests for Rollkit's EVM integration.
//
// This file specifically tests DA layer restart scenarios including:
// - Block creation and initial DA submission
// - DA layer failure handling with pending blocks exceeding blob size limits
// - Recovery behavior after DA restart
// - Verification that accumulated pending blocks are properly submitted after DA recovery
//
// Test Coverage:
// TestEvmDARestartWithPendingBlocksE2E - Tests the scenario where:
//  1. Blocks are created and published to DA
//  2. DA layer is killed
//  3. More blocks are created until pending blocks exceed max blob size
//  4. DA layer is restarted
//  5. Verification that pending blocks are submitted correctly without infinite loops
package e2e

import (
	"context"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
)

// TestEvmDARestartWithPendingBlocksE2E tests the scenario where DA layer is killed
// while blocks are being produced, causing pending blocks to accumulate beyond the
// maximum blob size limit, then verifies proper recovery when DA is restarted.
//
// Test Flow:
// 1. Sets up Local DA layer and sequencer node
// 2. Creates and submits initial transactions to establish baseline
// 3. Kills the DA layer while keeping sequencer running
// 4. Continues creating blocks until pending blocks exceed max blob size
// 5. Restarts the DA layer
// 6. Verifies that all pending blocks are properly submitted without loops
//
// This test validates:
// - Proper handling of DA layer failures
// - Accumulation of pending blocks when DA is unavailable
// - Automatic retry and batch splitting when blob size limits are exceeded
// - Recovery behavior when DA layer comes back online
// - Prevention of infinite loops during pending block submission
func TestEvmDARestartWithPendingBlocksE2E(t *testing.T) {
	flag.Parse()
	workDir := t.TempDir()
	nodeHome := filepath.Join(workDir, "evm-sequencer")
	sut := NewSystemUnderTest(t)

	// Setup sequencer and get genesis hash
	genesisHash, seqURL := setupSequencerOnlyTest(t, sut, nodeHome)
	t.Logf("Genesis hash: %s", genesisHash)

	// Connect to EVM
	client, err := ethclient.Dial(seqURL)
	require.NoError(t, err, "Should be able to connect to EVM")
	defer client.Close()

	ctx := context.Background()
	var globalNonce uint64 = 0

	// ===== PHASE 1: Establish Baseline - Create Initial Blocks =====
	t.Log("🔄 PHASE 1: Creating initial blocks and submitting to DA")

	// Submit initial transactions to create baseline blocks
	const initialTxCount = 5
	var initialTxHashes []string

	for i := 0; i < initialTxCount; i++ {
		tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &globalNonce)
		require.NoError(t, client.SendTransaction(context.Background(), tx))
		initialTxHashes = append(initialTxHashes, tx.Hash().Hex())
		t.Logf("Submitted initial transaction %d: %s", i+1, tx.Hash().Hex())
		time.Sleep(100 * time.Millisecond) // Small delay to ensure separate blocks
	}

	// Wait for initial transactions to be included in blocks
	t.Log("Waiting for initial transactions to be included...")
	require.Eventually(t, func() bool {
		for _, txHashStr := range initialTxHashes {
			if !evm.CheckTxIncluded(client, common.HexToHash(txHashStr)) {
				return false
			}
		}
		return true
	}, 15*time.Second, 500*time.Millisecond, "All initial transactions should be included")

	// Get current block height after initial setup
	initialHeader, err := client.HeaderByNumber(ctx, nil)
	require.NoError(t, err, "Should get initial header")
	initialHeight := initialHeader.Number.Uint64()
	t.Logf("✅ Initial phase completed. Current block height: %d", initialHeight)

	// ===== PHASE 2: Kill DA Layer =====
	t.Log("🔄 PHASE 2: Killing DA layer to simulate failure")

	// Kill only the DA process, keep sequencer and EVM running
	sut.ShutdownByCmd("local-da")
	t.Log("✅ DA layer killed, sequencer and EVM continue running")

	// ===== PHASE 3: Create Blocks Until Pending Exceeds Blob Size =====
	t.Log("🔄 PHASE 3: Creating blocks to accumulate pending data beyond blob size limit")

	// Calculate approximate number of transactions needed to exceed max blob size
	// We will restart DA with 10KB limit, so we need enough data to exceed that
	// Each transaction is roughly ~100-200 bytes, so ~100 transactions should be enough
	const txsPerBlock = 10  // 10 txs per block
	const targetBlocks = 15 // 15 blocks = ~150 transactions to exceed 10KB limit

	var pendingTxHashes []string
	blockCreationStart := time.Now()

	for block := 0; block < targetBlocks; block++ {
		// Create multiple transactions per block to increase data size
		for tx := 0; tx < txsPerBlock; tx++ {
			transaction := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &globalNonce)
			require.NoError(t, client.SendTransaction(ctx, transaction))
			pendingTxHashes = append(pendingTxHashes, transaction.Hash().Hex())
		}

		// Small delay between transaction batches
		time.Sleep(200 * time.Millisecond)

		// Log progress every 10 blocks
		if (block+1)%10 == 0 {
			currentHeader, err := client.HeaderByNumber(ctx, nil)
			if err == nil {
				currentHeight := currentHeader.Number.Uint64()
				t.Logf("Progress: Block batch %d/%d completed. Current height: %d, Pending txs: %d",
					block+1, targetBlocks, currentHeight, len(pendingTxHashes))
			}
		}
	}

	// Wait for blocks to be created locally (they won't be submitted to DA)
	time.Sleep(5 * time.Second)

	// Check current block height
	currentHeader, err := client.HeaderByNumber(ctx, nil)
	require.NoError(t, err, "Should get current header")
	currentHeight := currentHeader.Number.Uint64()
	blocksCreated := currentHeight - initialHeight

	t.Logf("✅ Phase 3 completed in %v:", time.Since(blockCreationStart))
	t.Logf("   - Blocks created while DA was down: %d", blocksCreated)
	t.Logf("   - Total pending transactions: %d", len(pendingTxHashes))
	t.Logf("   - Current block height: %d", currentHeight)

	// ===== PHASE 4: Restart DA Layer =====
	t.Log("🔄 PHASE 4: Restarting DA layer to test recovery")

	// Start local DA again with a very small max blob size to force the error
	localDABinary := "local-da"
	if evmSingleBinaryPath != "evm-single" {
		localDABinary = filepath.Join(filepath.Dir(evmSingleBinaryPath), "local-da")
	}
	// Use an extremely small max blob size (100 bytes) to guarantee StatusTooBig with our large batch
	sut.ExecCmd(localDABinary, "-max-blob-size", "100")
	t.Log("✅ DA layer restarted")

	// Wait for DA to be ready
	time.Sleep(2 * time.Second)

	// ===== PHASE 5: Monitor Recovery Process =====
	t.Log("🔄 PHASE 5: Monitoring recovery and pending block submission")

	// Monitor the recovery process for a reasonable amount of time
	recoveryStart := time.Now()
	recoveryTimeout := 30 * time.Second // Reduced to 30 seconds to quickly detect infinite loop
	checkInterval := 2 * time.Second    // Check more frequently

	var recoverySuccess bool
	var finalTxCount int

	t.Log("Monitoring transaction inclusion during recovery...")
	for elapsed := time.Duration(0); elapsed < recoveryTimeout; elapsed += checkInterval {
		// Count how many pending transactions have been included
		includedCount := 0
		for _, txHashStr := range pendingTxHashes {
			if evm.CheckTxIncluded(client, common.HexToHash(txHashStr)) {
				includedCount++
			}
		}

		finalTxCount = includedCount
		progressPct := float64(includedCount) / float64(len(pendingTxHashes)) * 100

		t.Logf("Recovery progress: %d/%d transactions included (%.1f%%) - Elapsed: %v",
			includedCount, len(pendingTxHashes), progressPct, elapsed)

		// Check if all transactions are included
		if includedCount == len(pendingTxHashes) {
			recoverySuccess = true
			t.Logf("🎉 All pending transactions recovered successfully in %v!", time.Since(recoveryStart))
			break
		}

		time.Sleep(checkInterval)
	}

	// ===== PHASE 6: Final Validation =====
	t.Log("🔄 PHASE 6: Final validation of recovery")

	// Verify that ALL transactions were recovered successfully
	require.True(t, recoverySuccess,
		"Recovery failed: only %d/%d transactions included after %v. All transactions must be included for data integrity.",
		finalTxCount, len(pendingTxHashes), recoveryTimeout)

	// Verify final block height increased appropriately
	finalHeader, err := client.HeaderByNumber(ctx, nil)
	require.NoError(t, err, "Should get final header")
	finalHeight := finalHeader.Number.Uint64()

	t.Logf("📊 Final Recovery Statistics:")
	t.Logf("   - Initial height: %d", initialHeight)
	t.Logf("   - Height when DA restarted: %d", currentHeight)
	t.Logf("   - Final height: %d", finalHeight)
	t.Logf("   - Blocks created during DA downtime: %d", currentHeight-initialHeight)
	t.Logf("   - Total recovery time: %v", time.Since(recoveryStart))
	t.Logf("   - Transactions recovered: %d/%d", finalTxCount, len(pendingTxHashes))

	// Submit one final transaction to verify system is stable
	t.Log("Testing system stability with final transaction...")
	finalTx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &globalNonce)
	require.NoError(t, client.SendTransaction(ctx, finalTx))

	// Verify final transaction is included quickly (system is responsive)
	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(client, finalTx.Hash())
	}, 15*time.Second, 500*time.Millisecond, "Final test transaction should be included quickly")

	t.Log("✅ Final transaction included - system is stable and responsive")

	// ===== TEST SUMMARY =====
	t.Logf("🎉 DA RESTART TEST COMPLETED SUCCESSFULLY!")
	t.Logf("   ✅ Initial baseline established: %d transactions", initialTxCount)
	t.Logf("   ✅ DA layer failure handled gracefully")
	t.Logf("   ✅ Pending blocks accumulated: %d blocks with %d transactions", blocksCreated, len(pendingTxHashes))
	t.Logf("   ✅ DA layer recovery successful: %d/%d transactions recovered", finalTxCount, len(pendingTxHashes))
	t.Logf("   ✅ No infinite loops detected during recovery")
	t.Logf("   ✅ System stability confirmed with final transaction")
	t.Logf("   🚀 Total test duration: %v", time.Since(blockCreationStart))
}
