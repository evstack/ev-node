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
	"syscall"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/block"
	"github.com/evstack/ev-node/execution/evm"
	"github.com/evstack/ev-node/pkg/config"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
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
	dcli, netID := tastoradocker.Setup(t)
	env := setupCommonEVMEnv(t, sut, dcli, netID)

	// Create passphrase file
	passphraseFile := createPassphraseFile(t, nodeHome)

	// Create JWT secret file
	jwtSecretFile := createJWTSecretFile(t, nodeHome, env.SequencerJWT)

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
		"--evm.genesis-hash", env.GenesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", nodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", env.Endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", env.Endpoints.GetSequencerEthURL(),
		"--force-inclusion-server", fiAddr,
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, env.Endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)

	return env.GenesisHash, env.Endpoints.GetSequencerEthURL()
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
	dockerClient, networkID := tastoradocker.Setup(t)
	env := setupCommonEVMEnv(t, sut, dockerClient, networkID, WithFullNode())

	passphraseFile := createPassphraseFile(t, sequencerHome)
	jwtSecretFile := createJWTSecretFile(t, sequencerHome, env.SequencerJWT)

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
		"--evm.genesis-hash", env.GenesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", sequencerHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", env.Endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", env.Endpoints.GetSequencerEthURL(),
		"--force-inclusion-server", fiAddr,
	}
	sut.ExecCmd(evmSingleBinaryPath, seqArgs...)
	sut.AwaitNodeUp(t, env.Endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
	t.Log("Sequencer is up with force inclusion enabled")
	// --- End Sequencer Setup ---

	// --- Start Full Node Setup ---
	// Reuse setupFullNode helper which handles genesis copying and node startup
	setupFullNode(t, sut, fullNodeHome, sequencerHome, env.FullNodeJWT, env.GenesisHash, env.Endpoints.GetRollkitP2PAddress(), env.Endpoints)
	t.Log("Full node is up")
	// --- End Full Node Setup ---

	// Connect to clients
	seqClient, err := ethclient.Dial(env.Endpoints.GetSequencerEthURL())
	require.NoError(t, err)
	defer seqClient.Close()

	fnClient, err := ethclient.Dial(env.Endpoints.GetFullNodeEthURL())
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
	dockerClient, networkID := tastoradocker.Setup(t)
	env := setupCommonEVMEnv(t, sut, dockerClient, networkID, WithFullNode())
	// Use env fields inline below to reduce local vars

	passphraseFile := createPassphraseFile(t, nodeHome)
	jwtSecretFile := createJWTSecretFile(t, nodeHome, env.SequencerJWT)

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
		"--evm.genesis-hash", env.GenesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", nodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		// CRITICAL: Set sequencer to listen to WRONG namespace - it won't see forced txs
		"--evnode.da.forced_inclusion_namespace", "wrong-namespace",
		"--evnode.rpc.address", env.Endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", env.Endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", env.Endpoints.GetSequencerEthURL(),
	}
	sut.ExecCmd(evmSingleBinaryPath, seqArgs...)
	sut.AwaitNodeUp(t, env.Endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)

	return env.GenesisHash, env.FullNodeJWT, env.Endpoints
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
		"--evm.engine-url", endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", endpoints.GetFullNodeEthURL(),
	}
	// Only add P2P peers if a peer address is provided (disabled for malicious sequencer test)
	if sequencerP2PAddr != "" {
		fnArgs = append(fnArgs, "--evnode.p2p.peers", sequencerP2PAddr)
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
	sut := NewSystemUnderTest(t)
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "sequencer")
	fullNodeHome := filepath.Join(workDir, "fullnode")

	// Setup malicious sequencer (listening to wrong forced inclusion namespace)
	genesisHash, fullNodeJwtSecret, endpoints := setupMaliciousSequencer(t, sut, sequencerHome)
	t.Log("Malicious sequencer started listening to WRONG forced inclusion namespace")
	t.Log("NOTE: Sequencer listens to 'wrong-namespace', won't see txs on 'forced-inc'")

	// Disable P2P sync - the full node will sync blocks directly from DA.
	setupFullNodeWithForceInclusionCheck(t, sut, fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash, "", endpoints)
	t.Log("Full node (syncer) is up and will verify forced inclusion from DA (P2P disabled)")

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
	blobClient, err := blobrpc.NewClient(ctx, endpoints.GetDAAddress(), "", "")
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

// setDAStartHeightInGenesis modifies the genesis file to set da_start_height.
// This is needed because the based sequencer requires non-zero DAStartHeight,
// and catch-up detection via CalculateEpochNumber also depends on it.
func setDAStartHeightInGenesis(t *testing.T, homeDir string, height uint64) {
	t.Helper()
	genesisPath := filepath.Join(homeDir, "config", "genesis.json")
	data, err := os.ReadFile(genesisPath)
	require.NoError(t, err)

	var genesis map[string]interface{}
	err = json.Unmarshal(data, &genesis)
	require.NoError(t, err)

	genesis["da_start_height"] = height

	newData, err := json.MarshalIndent(genesis, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(genesisPath, newData, 0644)
	require.NoError(t, err)
}

// TestEvmSequencerCatchUpBasedSequencerE2E tests that when a sequencer restarts after
// extended downtime (multiple DA epochs), it correctly enters catch-up mode, replays
// missed forced inclusion transactions from DA (matching what a based sequencer would
// produce), and then resumes normal operation.
func TestEvmSequencerCatchUpBasedSequencerE2E(t *testing.T) {
	sut := NewSystemUnderTest(t)
	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "sequencer")
	fullNodeHome := filepath.Join(workDir, "fullnode")

	t.Log("Phase 1: Setup - Start Sequencer and Sync Node")

	dockerClient, networkID := tastoradocker.Setup(t)
	env := setupCommonEVMEnv(t, sut, dockerClient, networkID, WithFullNode())

	// Create passphrase and JWT secret files for sequencer
	seqPassphraseFile := createPassphraseFile(t, sequencerHome)
	seqJwtSecretFile := createJWTSecretFile(t, sequencerHome, env.SequencerJWT)

	// Initialize sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", seqPassphraseFile,
		"--home", sequencerHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Modify genesis: enable force inclusion with epoch=2, set da_start_height=1
	enableForceInclusionInGenesis(t, sequencerHome, 2)
	setDAStartHeightInGenesis(t, sequencerHome, 1)

	// Copy genesis to full node (will be used when restarting as based sequencer)
	output, err = sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--home", fullNodeHome,
	)
	require.NoError(t, err, "failed to init full node", output)
	MustCopyFile(t,
		filepath.Join(sequencerHome, "config", "genesis.json"),
		filepath.Join(fullNodeHome, "config", "genesis.json"),
	)

	// Start sequencer with forced inclusion namespace
	seqProcess := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret-file", seqJwtSecretFile,
		"--evm.genesis-hash", env.GenesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", seqPassphraseFile,
		"--home", sequencerHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", env.Endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", env.Endpoints.GetSequencerEthURL(),
		"--evnode.log.level", "error",
	)
	sut.AwaitNodeUp(t, env.Endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
	t.Log("Sequencer is up with force inclusion enabled")

	// Get sequencer P2P address for sync node to connect to
	sequencerP2PAddress := getNodeP2PAddress(t, sut, sequencerHome, env.Endpoints.RollkitRPCPort)
	t.Logf("Sequencer P2P address: %s", sequencerP2PAddress)

	// Create JWT secret file for full node
	fnJwtSecretFile := createJWTSecretFile(t, fullNodeHome, env.FullNodeJWT)

	// Start sync node (full node) - connects to sequencer via P2P
	fnProcess := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret-file", fnJwtSecretFile,
		"--evm.genesis-hash", env.GenesisHash,
		"--home", fullNodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetFullNodeRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetFullNodeP2PAddress(),
		"--evnode.p2p.peers", sequencerP2PAddress,
		"--evm.engine-url", env.Endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", env.Endpoints.GetFullNodeEthURL(),
		"--evnode.log.level", "error",
	)
	sut.AwaitNodeLive(t, env.Endpoints.GetFullNodeRPCAddress(), NodeStartupTimeout)
	t.Log("Sync node (full node) is up and syncing from sequencer")

	t.Log("Phase 2: Send Transactions and Wait for Sync")

	seqClient, err := ethclient.Dial(env.Endpoints.GetSequencerEthURL())
	require.NoError(t, err)
	defer seqClient.Close()

	fnClient, err := ethclient.Dial(env.Endpoints.GetFullNodeEthURL())
	require.NoError(t, err)
	defer fnClient.Close()

	ctx := context.Background()
	var nonce uint64 = 0

	// Submit 2 normal transactions to sequencer
	var normalTxHashes []common.Hash
	for i := 0; i < 2; i++ {
		tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
		err = seqClient.SendTransaction(ctx, tx)
		require.NoError(t, err)
		normalTxHashes = append(normalTxHashes, tx.Hash())
		t.Logf("Submitted normal tx %d: %s (nonce=%d)", i+1, tx.Hash().Hex(), tx.Nonce())
	}

	// Wait for sync node to sync the transactions
	for i, txHash := range normalTxHashes {
		require.Eventually(t, func() bool {
			return evm.CheckTxIncluded(fnClient, txHash)
		}, 20*time.Second, 500*time.Millisecond, "Normal tx %d not synced to full node", i+1)
		t.Logf("Normal tx %d synced to full node", i+1)
	}

	t.Log("Phase 3: Stop Sequencer and Wait for Sync Node to Catch Up")

	// Record sequencer's height BEFORE stopping (RPC may be unavailable after SIGTERM).
	seqHeader, err := seqClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	seqFinalHeight := seqHeader.Number.Uint64()
	t.Logf("Sequencer at height: %d before shutdown", seqFinalHeight)

	// Stop sequencer so it stops producing new blocks.
	err = seqProcess.Signal(syscall.SIGTERM)
	require.NoError(t, err, "failed to stop sequencer process")
	time.Sleep(1 * time.Second)

	// Wait for the full node to sync up to the sequencer's final height.
	// Both nodes MUST be at the same height so that the based sequencer
	// later produces blocks identical to what the single sequencer's catch-up
	// will reproduce (same DA checkpoint, same state, same timestamps).
	require.Eventually(t, func() bool {
		fnH, err := fnClient.HeaderByNumber(ctx, nil)
		if err != nil {
			return false
		}
		return fnH.Number.Uint64() >= seqFinalHeight
	}, 30*time.Second, 500*time.Millisecond, "Full node should catch up to sequencer height %d", seqFinalHeight)

	fnHeader, err := fnClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	t.Logf("Full node caught up to height: %d (sequencer was at %d)", fnHeader.Number.Uint64(), seqFinalHeight)

	// Stop sync node process
	err = fnProcess.Signal(syscall.SIGTERM)
	require.NoError(t, err, "failed to stop full node process")
	time.Sleep(1 * time.Second)
	t.Log("Both sequencer and sync node stopped at same height")

	t.Log("Phase 4: Restart Sync Node as Based Sequencer")

	// Restart the same full node as a based sequencer
	// Reuse the same home directory and data, just add the --based_sequencer flag
	basedSeqProcess := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evnode.node.aggregator=true",
		"--evnode.node.based_sequencer=true",
		"--evm.jwt-secret-file", fnJwtSecretFile,
		"--evm.genesis-hash", env.GenesisHash,
		"--home", fullNodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetFullNodeRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetFullNodeP2PAddress(),
		"--evm.engine-url", env.Endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", env.Endpoints.GetFullNodeEthURL(),
		"--evnode.log.level", "error",
	)
	sut.AwaitNodeLive(t, env.Endpoints.GetFullNodeRPCAddress(), NodeStartupTimeout)
	t.Log("Sync node restarted as based sequencer")

	// Reconnect to based sequencer
	basedSeqClient, err := ethclient.Dial(env.Endpoints.GetFullNodeEthURL())
	require.NoError(t, err)
	defer basedSeqClient.Close()

	t.Log("Phase 5: Submit Forced Inclusion Transactions to DA")

	blobClient, err := blobrpc.NewClient(ctx, env.Endpoints.GetDAAddress(), "", "")
	require.NoError(t, err, "Failed to create blob RPC client")
	defer blobClient.Close()

	daClient := block.NewDAClient(
		blobClient,
		config.Config{
			DA: config.DAConfig{
				Namespace:                DefaultDANamespace,
				ForcedInclusionNamespace: "forced-inc",
			},
		},
		zerolog.Nop(),
	)

	// Create and submit 3 forced inclusion txs to DA
	var forcedTxHashes []common.Hash
	for i := 0; i < 3; i++ {
		txForce := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
		txBytes, err := txForce.MarshalBinary()
		require.NoError(t, err)

		result := daClient.Submit(ctx, [][]byte{txBytes}, -1, daClient.GetForcedInclusionNamespace(), nil)
		require.Equal(t, da.StatusSuccess, result.Code, "Failed to submit forced tx %d to DA: %s", i+1, result.Message)

		forcedTxHashes = append(forcedTxHashes, txForce.Hash())
		t.Logf("Submitted forced inclusion tx %d to DA: %s (nonce=%d)", i+1, txForce.Hash().Hex(), txForce.Nonce())
	}

	t.Log("Advancing DA past multiple epochs...")
	time.Sleep(6 * time.Second)

	t.Log("Phase 6: Verify Based Sequencer Includes Forced Txs")

	// Wait for based sequencer to include forced inclusion txs
	for i, txHash := range forcedTxHashes {
		require.Eventually(t, func() bool {
			return evm.CheckTxIncluded(basedSeqClient, txHash)
		}, 30*time.Second, 1*time.Second,
			"Forced inclusion tx %d (%s) not included in based sequencer", i+1, txHash.Hex())
		t.Logf("Based sequencer included forced tx %d: %s", i+1, txHash.Hex())
	}
	t.Log("All forced inclusion txs verified on based sequencer")

	// Get the based sequencer's block height after including forced txs
	basedSeqHeader, err := basedSeqClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	basedSeqFinalHeight := basedSeqHeader.Number.Uint64()
	t.Logf("Based sequencer final height: %d", basedSeqFinalHeight)

	t.Log("Phase 7: Stop Based Sequencer and Restart as Normal Sync Node")

	// Stop based sequencer
	err = basedSeqProcess.Signal(syscall.SIGTERM)
	require.NoError(t, err, "failed to stop based sequencer process")
	time.Sleep(1 * time.Second)

	// Restart as normal sync node (without --based_sequencer flag, with --p2p.peers to connect to sequencer)
	fnProcess = sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret-file", fnJwtSecretFile,
		"--evm.genesis-hash", env.GenesisHash,
		"--home", fullNodeHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetFullNodeRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetFullNodeP2PAddress(),
		"--evnode.p2p.peers", sequencerP2PAddress,
		"--evnode.clear_cache",
		"--evm.engine-url", env.Endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", env.Endpoints.GetFullNodeEthURL(),
		"--evnode.log.level", "debug",
	)
	sut.AwaitNodeLive(t, env.Endpoints.GetFullNodeRPCAddress(), NodeStartupTimeout)
	t.Log("Sync node restarted as normal full node")

	// Reconnect to sync node
	fnClient, err = ethclient.Dial(env.Endpoints.GetFullNodeEthURL())
	require.NoError(t, err)

	fnClientHeader, err := fnClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	t.Logf("Sync node restarted at height: %d", fnClientHeader.Number.Uint64())

	t.Log("Phase 8: Restart Original Sequencer")

	// Restart the original sequencer
	seqProcess = sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret-file", seqJwtSecretFile,
		"--evm.genesis-hash", env.GenesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", seqPassphraseFile,
		"--home", sequencerHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", env.Endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", env.Endpoints.GetSequencerEthURL(),
		"--evnode.log.level", "error",
	)
	sut.AwaitNodeUp(t, env.Endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
	t.Log("Sequencer restarted successfully")

	// Reconnect to sequencer
	seqClient, err = ethclient.Dial(env.Endpoints.GetSequencerEthURL())
	require.NoError(t, err)

	t.Log("Phase 9: Verify Sequencer Catches Up")

	// Wait for sequencer to catch up and include forced txs
	for i, txHash := range forcedTxHashes {
		require.Eventually(t, func() bool {
			return evm.CheckTxIncluded(seqClient, txHash)
		}, 30*time.Second, 1*time.Second,
			"Forced inclusion tx %d (%s) should be included after catch-up", i+1, txHash.Hex())
		t.Logf("Sequencer caught up with forced tx %d: %s", i+1, txHash.Hex())
	}
	t.Log("All forced inclusion txs verified on sequencer after catch-up")

	// Verify sequencer produces blocks and reaches same height as based sequencer
	require.Eventually(t, func() bool {
		seqHeader, err := seqClient.HeaderByNumber(ctx, nil)
		if err != nil {
			return false
		}
		return seqHeader.Number.Uint64() >= basedSeqFinalHeight
	}, 30*time.Second, 1*time.Second, "Sequencer should catch up to based sequencer height")

	seqHeader, err = seqClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	t.Logf("Sequencer caught up to height: %d", seqHeader.Number.Uint64())

	// ===== PHASE 10: Verify Nodes Are in Sync =====
	t.Log("Phase 10: Verify Nodes Are in Sync")

	// Wait for sync node to catch up to sequencer
	require.Eventually(t, func() bool {
		seqHeader, err1 := seqClient.HeaderByNumber(ctx, nil)
		fnHeader, err2 := fnClient.HeaderByNumber(ctx, nil)
		if err1 != nil || err2 != nil {
			return false
		}

		syncHeaderNb, seqHeaderNb := fnHeader.Number.Uint64(), seqHeader.Number.Uint64()
		t.Logf("Sync node height is %d and seq node height is %d", syncHeaderNb, seqHeaderNb)

		return syncHeaderNb >= seqHeaderNb
	}, 30*time.Second, 1*time.Second, "Sync node should catch up to sequencer")

	// Verify both nodes have all forced inclusion txs
	for i, txHash := range forcedTxHashes {
		seqIncluded := evm.CheckTxIncluded(seqClient, txHash)
		fnIncluded := evm.CheckTxIncluded(fnClient, txHash)
		require.True(t, seqIncluded, "Forced tx %d should be on sequencer", i+1)
		require.True(t, fnIncluded, "Forced tx %d should be on sync node", i+1)
		t.Logf("Forced tx %d verified on both nodes: %s", i+1, txHash.Hex())
	}

	// Send a new transaction and verify both nodes get it
	txFinal := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	err = seqClient.SendTransaction(ctx, txFinal)
	require.NoError(t, err)
	t.Logf("Submitted final tx: %s (nonce=%d)", txFinal.Hash().Hex(), txFinal.Nonce())

	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(seqClient, txFinal.Hash()) && evm.CheckTxIncluded(fnClient, txFinal.Hash())
	}, 30*time.Second, 20*time.Millisecond, "Final tx should be included on both nodes")
	t.Log("Final tx included on both nodes - nodes are in sync")
}

// TestEvmBasedSequencerBaselineE2E tests the based sequencer.
// This test validates that a fresh based sequencer can:
// 1. Start from genesis (not restarted from a full node)
// 2. Retrieve forced inclusion transactions from DA
// 3. Include those transactions in produced blocks
func TestEvmBasedSequencerBaselineE2E(t *testing.T) {
	sut := NewSystemUnderTest(t)
	workDir := t.TempDir()
	basedSeqHome := filepath.Join(workDir, "based_sequencer")

	t.Log("Setting up fresh based sequencer from genesis")

	dockerClient, networkID := tastoradocker.Setup(t)
	env := setupCommonEVMEnv(t, sut, dockerClient, networkID, WithFullNode())

	// Create passphrase and JWT secret files
	passphraseFile := createPassphraseFile(t, basedSeqHome)
	jwtSecretFile := createJWTSecretFile(t, basedSeqHome, env.SequencerJWT)

	// Initialize based sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--evnode.node.aggregator=true",
		"--evnode.node.based_sequencer=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", basedSeqHome,
	)
	require.NoError(t, err, "failed to init based sequencer", output)

	// Modify genesis: enable force inclusion with epoch=2, set da_start_height=1
	enableForceInclusionInGenesis(t, basedSeqHome, 2)
	setDAStartHeightInGenesis(t, basedSeqHome, 1)

	// Start based sequencer
	sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evnode.node.aggregator=true",
		"--evnode.node.based_sequencer=true",
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", env.GenesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", basedSeqHome,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", env.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.forced_inclusion_namespace", "forced-inc",
		"--evnode.rpc.address", env.Endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", env.Endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", env.Endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", env.Endpoints.GetSequencerEthURL(),
		"--evnode.log.level", "debug",
	)
	sut.AwaitNodeUp(t, env.Endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
	t.Log("Based sequencer is up")

	// Connect to based sequencer
	basedSeqClient, err := ethclient.Dial(env.Endpoints.GetSequencerEthURL())
	require.NoError(t, err)
	defer basedSeqClient.Close()

	ctx := context.Background()

	t.Log("Submitting forced inclusion transactions to DA")
	blobClient, err := blobrpc.NewClient(ctx, env.Endpoints.GetDAAddress(), "", "")
	require.NoError(t, err, "Failed to create blob RPC client")
	defer blobClient.Close()

	daClient := block.NewDAClient(
		blobClient,
		config.Config{
			DA: config.DAConfig{
				Namespace:                DefaultDANamespace,
				ForcedInclusionNamespace: "forced-inc",
			},
		},
		zerolog.Nop(),
	)

	// Create and submit 3 forced inclusion txs to DA
	var forcedTxHashes []common.Hash
	var nonce uint64 = 0
	for i := 0; i < 3; i++ {
		txForce := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
		txBytes, err := txForce.MarshalBinary()
		require.NoError(t, err)

		result := daClient.Submit(ctx, [][]byte{txBytes}, -1, daClient.GetForcedInclusionNamespace(), nil)
		require.Equal(t, da.StatusSuccess, result.Code, "Failed to submit forced tx %d to DA: %s", i+1, result.Message)

		forcedTxHashes = append(forcedTxHashes, txForce.Hash())
		t.Logf("Submitted forced inclusion tx %d to DA: %s (nonce=%d)", i+1, txForce.Hash().Hex(), txForce.Nonce())
	}

	// Advance DA past epoch boundary by submitting dummy data
	// With epoch=2, we need at least 2 DA blocks per epoch
	t.Log("Advancing DA past epoch boundary...")
	time.Sleep(4 * time.Second)

	// ===== VERIFY BASED SEQUENCER INCLUDES FORCED TXS =====
	t.Log("Waiting for based sequencer to include forced inclusion txs")

	for i, txHash := range forcedTxHashes {
		require.Eventually(t, func() bool {
			return evm.CheckTxIncluded(basedSeqClient, txHash)
		}, 30*time.Second, 1*time.Second,
			"Forced inclusion tx %d (%s) not included in based sequencer", i+1, txHash.Hex())
		t.Logf("Based sequencer included forced tx %d: %s", i+1, txHash.Hex())
	}

	// Verify blocks are being produced
	header, err := basedSeqClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	t.Logf("Based sequencer height: %d", header.Number.Uint64())
}
