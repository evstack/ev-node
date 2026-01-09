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
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
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
	jwtSecret, _, genesisHash, endpoints := setupCommonEVMTest(t, sut, false)

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
	fiPort, err := getAvailablePort()
	require.NoError(t, err)
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
	fiPort, err := getAvailablePort()
	require.NoError(t, err)
	fiUrl := fmt.Sprintf("http://127.0.0.1:%d", fiPort)

	// --- Start Sequencer Setup ---
	// We manually setup sequencer here because we need the force inclusion flag,
	// and we need to capture variables for full node setup.
	jwtSecret, fullNodeJwtSecret, genesisHash, endpoints := setupCommonEVMTest(t, sut, true)

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
