//go:build evm

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block"
	"github.com/evstack/ev-node/execution/evm"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/da/node"
	da "github.com/evstack/ev-node/pkg/da/types"
)

// enableForceInclusionInGenesis modifies the genesis file to set the force inclusion epoch
// to a small value suitable for testing.
func enableForceInclusionInGenesis(t *testing.T, homeDir string, epoch uint64) {
	t.Helper()
	genesisPath := filepath.Join(homeDir, "config", "genesis.json")
	data, err := os.ReadFile(genesisPath)
	require.NoError(t, err)

	var genesis map[string]interface{}
	err = json.Unmarshal(data, &genesis)
	require.NoError(t, err)

	genesis["da_epoch_forced_inclusion"] = epoch

	newData, err := json.MarshalIndent(genesis, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(genesisPath, newData, 0644)
	require.NoError(t, err)
}

// submitForceInclusionTx sends a raw transaction to the force inclusion server
func submitForceInclusionTx(t *testing.T, fiUrl string, txBytes []byte) {
	t.Helper()
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_sendRawTransaction",
		"params":  []string{"0x" + common.Bytes2Hex(txBytes)},
	}

	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post(fiUrl, "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var res map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	require.Nil(t, res["error"], "RPC returned error: %v", res["error"])
	require.NotNil(t, res["result"], "RPC result is nil")
}

// setupSequencerWithForceInclusion sets up a sequencer node with force inclusion enabled
func setupSequencerWithForceInclusion(t *testing.T, sut *SystemUnderTest, nodeHome string, fiPort int) (string, string) {
	t.Helper()

	// Use common setup (no full node needed initially)
	jwtSecret, _, genesisHash, endpoints, _ := setupCommonEVMTest(t, sut, false)

	// Create passphrase file
	passphraseFile := createPassphraseFile(t, nodeHome)

	// Create JWT secret file
	jwtSecretFile := createJWTSecretFile(t, nodeHome, jwtSecret)

	// Initialize sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", nodeHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Modify genesis to lower the epoch for faster testing (2 DA blocks)
	enableForceInclusionInGenesis(t, nodeHome, 2)

	// Start sequencer with force inclusion server enabled
	fiAddr := fmt.Sprintf("127.0.0.1:%d", fiPort)
	args := []string{
		"start",
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", genesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", nodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
		"--force-inclusion-server", fiAddr,
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)

	return genesisHash, endpoints.GetSequencerEthURL()
}

func TestEvmSequencerForceInclusionE2E(t *testing.T) {
	sut := NewSystemUnderTest(t)
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "sequencer")

	// Get a port for force inclusion server
	fiPort, listener, err := getAvailablePort()
	require.NoError(t, err)
	// We only need the port number for the flag; custom server startup handles binding
	listener.Close()
	fiUrl := fmt.Sprintf("http://127.0.0.1:%d", fiPort)

	// Setup sequencer with force inclusion enabled
	genesisHash, seqEthURL := setupSequencerWithForceInclusion(t, sut, sequencerHome, fiPort)
	t.Logf("Sequencer started with force inclusion server at %s", fiUrl)
	t.Logf("Genesis hash: %s", genesisHash)

	// Connect to sequencer EVM
	client, err := ethclient.Dial(seqEthURL)
	require.NoError(t, err)
	defer client.Close()

	// 1. Send a normal transaction first to ensure chain is moving
	t.Log("Sending normal transaction...")
	var nonce uint64 = 0
	txNormal := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	err = client.SendTransaction(context.Background(), txNormal)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(client, txNormal.Hash())
	}, 15*time.Second, 500*time.Millisecond, "Normal transaction not included")
	t.Log("Normal transaction included")

	// 2. Send a Forced Inclusion transaction
	t.Log("Sending forced inclusion transaction...")
	txForce := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	txBytes, err := txForce.MarshalBinary()
	require.NoError(t, err)

	submitForceInclusionTx(t, fiUrl, txBytes)
	t.Logf("Forced inclusion transaction submitted: %s", txForce.Hash().Hex())

	// Wait for inclusion
	// Force inclusion depends on DA epoch. With epoch=2 and fast DA block time (200ms),
	// this should be reasonably fast, but we allow enough time for robustness.
	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(client, txForce.Hash())
	}, 30*time.Second, 1*time.Second, "Forced inclusion transaction not included")

	t.Log("Forced inclusion transaction included successfully in Sequencer")
}

func TestEvmFullNodeForceInclusionE2E(t *testing.T) {
	sut := NewSystemUnderTest(t)
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "sequencer")
	fullNodeHome := filepath.Join(workDir, "fullnode")

	// Get a port for force inclusion server
	fiPort, listener, err := getAvailablePort()
	require.NoError(t, err)
	// We only need the port number for the flag; custom server startup handles binding
	listener.Close()
	fiUrl := fmt.Sprintf("http://127.0.0.1:%d", fiPort)

	// --- Start Sequencer Setup ---
	// We manually setup sequencer here because we need the force inclusion flag,
	// and we need to capture variables for full node setup.
	jwtSecret, fullNodeJwtSecret, genesisHash, endpoints, _ := setupCommonEVMTest(t, sut, true)

	passphraseFile := createPassphraseFile(t, sequencerHome)
	jwtSecretFile := createJWTSecretFile(t, sequencerHome, jwtSecret)

	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", sequencerHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Set epoch to 2 for fast testing
	enableForceInclusionInGenesis(t, sequencerHome, 2)

	fiAddr := fmt.Sprintf("127.0.0.1:%d", fiPort)
	seqArgs := []string{
		"start",
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", genesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", sequencerHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
		"--force-inclusion-server", fiAddr,
	}
	sut.ExecCmd(evmSingleBinaryPath, seqArgs...)
	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
	t.Log("Sequencer is up with force inclusion enabled")
	// --- End Sequencer Setup ---

	// --- Start Full Node Setup ---
	// Reuse setupFullNode helper which handles genesis copying and node startup
	setupFullNode(t, sut, fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash, endpoints.GetRollkitP2PAddress(), endpoints)
	t.Log("Full node is up")
	// --- End Full Node Setup ---

	// Connect to clients
	seqClient, err := ethclient.Dial(endpoints.GetSequencerEthURL())
	require.NoError(t, err)
	defer seqClient.Close()

	fnClient, err := ethclient.Dial(endpoints.GetFullNodeEthURL())
	require.NoError(t, err)
	defer fnClient.Close()

	var nonce uint64 = 0

	// 1. Send normal tx to sequencer
	t.Log("Sending normal transaction...")
	txNormal := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	err = seqClient.SendTransaction(context.Background(), txNormal)
	require.NoError(t, err)

	// Wait for full node to sync it
	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(fnClient, txNormal.Hash())
	}, 20*time.Second, 500*time.Millisecond, "Normal tx not synced to full node")
	t.Log("Normal tx synced to full node")

	// 2. Send forced inclusion tx
	t.Log("Sending forced inclusion transaction...")
	txForce := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	txBytes, err := txForce.MarshalBinary()
	require.NoError(t, err)

	submitForceInclusionTx(t, fiUrl, txBytes)
	t.Logf("Forced inclusion transaction submitted: %s", txForce.Hash().Hex())

	// Wait for full node to sync it
	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(fnClient, txForce.Hash())
	}, 40*time.Second, 1*time.Second, "Forced inclusion tx not synced to full node")

	t.Log("Forced inclusion tx synced to full node successfully")
}

// setupMaliciousSequencer sets up a sequencer that listens to the WRONG forced inclusion namespace.
// This simulates a malicious sequencer that doesn't retrieve forced inclusion txs from the correct namespace.
func setupMaliciousSequencer(t *testing.T, sut *SystemUnderTest, nodeHome string) (string, string, *TestEndpoints) {
	t.Helper()

	// Use common setup with full node support
	jwtSecret, fullNodeJwtSecret, genesisHash, endpoints, _ := setupCommonEVMTest(t, sut, true)

	passphraseFile := createPassphraseFile(t, nodeHome)
	jwtSecretFile := createJWTSecretFile(t, nodeHome, jwtSecret)

	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", nodeHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Set epoch to 2 for fast testing (force inclusion must happen within 2 DA blocks)
	enableForceInclusionInGenesis(t, nodeHome, 2)

	seqArgs := []string{
		"start",
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", genesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", nodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		// CRITICAL: Set sequencer to listen to WRONG namespace - it won't see forced txs
		"--evnode.da.forced_inclusion_namespace", "wrong-namespace",
		"--evnode.rpc.address", endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
	}
	sut.ExecCmd(evmSingleBinaryPath, seqArgs...)
	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)

	return genesisHash, fullNodeJwtSecret, endpoints
}

// setupFullNodeWithForceInclusionCheck sets up a full node that WILL verify forced inclusion txs
// by reading from DA. This node will detect when the sequencer maliciously skips forced txs.
// The key difference from standard setupFullNode is that we explicitly add the forced_inclusion_namespace flag.
func setupFullNodeWithForceInclusionCheck(t *testing.T, sut *SystemUnderTest, fullNodeHome, sequencerHome, jwtSecret, genesisHash, sequencerP2PAddr string, endpoints *TestEndpoints) {
	t.Helper()

	// Initialize full node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--home", fullNodeHome,
	)
	require.NoError(t, err, "failed to init full node", output)

	// Copy genesis file from sequencer to full node
	MustCopyFile(t, filepath.Join(sequencerHome, "config", "genesis.json"), filepath.Join(fullNodeHome, "config", "genesis.json"))

	// Create JWT secret file for full node
	jwtSecretFile := createJWTSecretFile(t, fullNodeHome, jwtSecret)

	// Start full node WITH forced_inclusion_namespace configured
	// This allows it to retrieve forced txs from DA and detect when they're missing from blocks
	fnArgs := []string{
		"start",
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", genesisHash,
		"--home", fullNodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc", // Enables forced inclusion verification
		"--evnode.rpc.address", endpoints.GetFullNodeRPCListen(),
		"--evnode.p2p.listen_address", endpoints.GetFullNodeP2PAddress(),
		"--evnode.p2p.peers", sequencerP2PAddr,
		"--evm.engine-url", endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", endpoints.GetFullNodeEthURL(),
	}
	sut.ExecCmd(evmSingleBinaryPath, fnArgs...)
	sut.AwaitNodeLive(t, endpoints.GetFullNodeRPCAddress(), NodeStartupTimeout)
}

// TestEvmSyncerMaliciousSequencerForceInclusionE2E tests that a sync node gracefully stops
// when it detects that the sequencer maliciously failed to include a forced inclusion transaction.
//
// This test validates the critical security property that sync nodes can detect and respond to
// malicious/misconfigured sequencer behavior regarding forced inclusion transactions.
//
// Test Architecture:
// - Malicious Sequencer: Configured with WRONG forced_inclusion_namespace ("wrong-namespace")
//   - Does NOT retrieve forced inclusion txs from correct namespace
//   - Simulates a censoring sequencer ignoring forced inclusion
//
// - Honest Sync Node: Configured with CORRECT forced_inclusion_namespace ("forced-inc")
//   - Retrieves forced inclusion txs from DA
//   - Compares them against blocks received from sequencer
//   - Detects when forced txs are missing beyond the grace period
//
// Test Flow:
// 1. Start malicious sequencer (listening to wrong namespace)
// 2. Start sync node that validates forced inclusion (correct namespace)
// 3. Submit forced inclusion tx directly to DA on correct namespace
// 4. Sequencer produces blocks WITHOUT the forced tx (doesn't see it)
// 5. Sync node detects violation after grace period expires
// 6. Sync node stops syncing (in production, would halt with error)
//
// Key Configuration:
// - da_epoch_forced_inclusion: 2 (forced txs must be included within 2 DA blocks)
// - Grace period: Additional buffer for network delays and block fullness
//
// Expected Outcome:
// - Forced tx appears in DA but NOT in sequencer's blocks
// - Sync node stops advancing its block height
// - In production: sync node logs "SEQUENCER IS MALICIOUS" and exits.
//
// Note: This test simulates the scenario by having the sequencer configured to
// listen to the wrong namespace, while we submit directly to the correct namespace.
func TestEvmSyncerMaliciousSequencerForceInclusionE2E(t *testing.T) {
	t.Skip() // Unskip once https://github.com/evstack/ev-node/pull/2963 is merged

	sut := NewSystemUnderTest(t)
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "sequencer")
	fullNodeHome := filepath.Join(workDir, "fullnode")

	// Setup malicious sequencer (listening to wrong forced inclusion namespace)
	genesisHash, fullNodeJwtSecret, endpoints := setupMaliciousSequencer(t, sut, sequencerHome)
	t.Log("Malicious sequencer started listening to WRONG forced inclusion namespace")
	t.Log("NOTE: Sequencer listens to 'wrong-namespace', won't see txs on 'forced-inc'")

	sequencerP2PAddress := getNodeP2PAddress(t, sut, sequencerHome, endpoints.RollkitRPCPort)
	t.Logf("Sequencer P2P address: %s", sequencerP2PAddress)

	// Setup full node that will sync from the sequencer and verify forced inclusion
	setupFullNodeWithForceInclusionCheck(t, sut, fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash, sequencerP2PAddress, endpoints)
	t.Log("Full node (syncer) is up and will verify forced inclusion from DA")

	// Connect to clients
	seqClient, err := ethclient.Dial(endpoints.GetSequencerEthURL())
	require.NoError(t, err)
	defer seqClient.Close()

	fnClient, err := ethclient.Dial(endpoints.GetFullNodeEthURL())
	require.NoError(t, err)
	defer fnClient.Close()

	var nonce uint64 = 0

	// 1. Send a normal transaction first to ensure chain is moving
	t.Log("Sending normal transaction to establish baseline...")
	txNormal := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	err = seqClient.SendTransaction(context.Background(), txNormal)
	require.NoError(t, err)

	// Wait for full node to sync it
	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(fnClient, txNormal.Hash())
	}, 20*time.Second, 500*time.Millisecond, "Normal tx not synced to full node")
	t.Log("Normal tx synced successfully")

	// 2. Submit forced inclusion transaction directly to DA (correct namespace)
	t.Log("Submitting forced inclusion transaction directly to DA on correct namespace...")
	txForce := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	txBytes, err := txForce.MarshalBinary()
	require.NoError(t, err)

	// Create a blobrpc client and DA client to submit directly to the correct forced inclusion namespace
	// The sequencer is listening to "wrong-namespace" so it won't see this
	ctx := context.Background()
	blobClient, err := NewClient(ctx, endpoints.GetDAAddress(), "", "")
	require.NoError(t, err, "Failed to create blob RPC client")
	defer blobClient.Close()

	daClient := block.NewDAClient(
		blobClient,
		config.Config{
			DA: config.DAConfig{
				Namespace:                "evm-e2e",
				DataNamespace:            "evm-e2e-data",
				ForcedInclusionNamespace: "forced-inc", // Correct namespace
			},
		},
		zerolog.Nop(),
	)

	// Submit transaction to DA on the forced inclusion namespace
	result := daClient.Submit(ctx, [][]byte{txBytes}, -1, daClient.GetForcedInclusionNamespace(), nil)
	require.Equal(t, da.StatusSuccess, result.Code, "Failed to submit to DA: %s", result.Message)
	t.Logf("Forced inclusion transaction submitted to DA: %s", txForce.Hash().Hex())

	// 3. Wait a moment for the forced tx to be written to DA
	time.Sleep(1 * time.Second)

	// 4. The malicious sequencer will NOT include the forced transaction in blocks
	// because it's listening to "wrong-namespace" instead of "forced-inc"
	// Send normal transactions to advance the chain past the epoch boundary and grace period.
	t.Log("Advancing chain to trigger malicious behavior detection...")
	for i := 0; i < 15; i++ {
		txExtra := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
		err = seqClient.SendTransaction(context.Background(), txExtra)
		require.NoError(t, err)
		time.Sleep(400 * time.Millisecond)
	}

	// 5. The sync node should detect the malicious behavior and stop syncing
	// With epoch=2, after ~2 DA blocks the forced tx should be included
	// The grace period gives some buffer, but eventually the violation is detected
	t.Log("Monitoring sync node for malicious behavior detection...")

	// Track whether the sync node stops advancing
	var lastHeight uint64
	var lastSeqHeight uint64
	stoppedSyncing := false
	consecutiveStops := 0

	// Monitor for up to 60 seconds
	for i := 0; i < 120; i++ {
		time.Sleep(500 * time.Millisecond)

		// Verify forced tx is NOT on the malicious sequencer
		if evm.CheckTxIncluded(seqClient, txForce.Hash()) {
			t.Fatal("Malicious sequencer incorrectly included the forced tx")
		}

		// Get sequencer height
		seqHeader, seqErr := seqClient.HeaderByNumber(context.Background(), nil)
		if seqErr == nil {
			seqHeight := seqHeader.Number.Uint64()
			if seqHeight > lastSeqHeight {
				t.Logf("Sequencer height: %d (was: %d) - producing blocks", seqHeight, lastSeqHeight)
				lastSeqHeight = seqHeight
			}
		}

		// Get full node height
		fnHeader, fnErr := fnClient.HeaderByNumber(context.Background(), nil)
		if fnErr != nil {
			t.Logf("Full node error (may have stopped): %v", fnErr)
			stoppedSyncing = true
			break
		}

		currentHeight := fnHeader.Number.Uint64()

		// Check if sync node stopped advancing
		if lastHeight > 0 && currentHeight == lastHeight {
			consecutiveStops++
			t.Logf("Full node height unchanged at %d (count: %d)", currentHeight, consecutiveStops)

			// If height hasn't changed for 10 consecutive checks (~5s), it's stopped
			if consecutiveStops >= 10 {
				t.Log("✅ Full node stopped syncing - malicious behavior detected!")
				stoppedSyncing = true
				break
			}
		} else if currentHeight > lastHeight {
			consecutiveStops = 0
			t.Logf("Full node height: %d (was: %d)", currentHeight, lastHeight)
		}

		lastHeight = currentHeight

		// Log gap between sequencer and sync node
		if seqErr == nil && lastSeqHeight > currentHeight {
			gap := lastSeqHeight - currentHeight
			if gap > 10 {
				t.Logf("⚠️  Sync node falling behind - gap: %d blocks", gap)
			}
		}
	}

	// Verify expected behavior
	require.True(t, stoppedSyncing,
		"Sync node should have stopped syncing after detecting malicious behavior")

	require.False(t, evm.CheckTxIncluded(seqClient, txForce.Hash()),
		"Malicious sequencer should NOT have included the forced inclusion transaction")
}
