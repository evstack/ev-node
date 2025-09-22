//go:build evm
// +build evm

// Package e2e contains shared utilities for EVM end-to-end tests.
//
// This file provides common functionality used across multiple EVM test files:
// - Docker and JWT setup for Reth EVM engines
// - Sequencer and full node initialization
// - P2P connection management
// - Transaction submission and verification utilities
// - Node restart and recovery functions
// - Common constants and configuration values
//
// By centralizing these utilities, we eliminate code duplication and ensure
// consistent behavior across all EVM integration tests.
package e2e

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
)

// evmSingleBinaryPath is the path to the evm-single binary used in tests
var evmSingleBinaryPath string

func init() {
	flag.StringVar(&evmSingleBinaryPath, "evm-binary", "evm-single", "evm-single binary")
}

// getAvailablePort finds an available TCP port on localhost
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// TestPorts holds unique port numbers for each test instance
type TestPorts struct {
	DAPort             string
	RollkitRPCPort     string
	RollkitP2PPort     string
	FullNodeP2PPort    string
	FullNodeRPCPort    string
	SequencerEthURL    string
	SequencerEngineURL string
	FullNodeEthURL     string
	FullNodeEngineURL  string
}

// generateTestPorts creates a set of unique ports for a test instance
// Only generates ports for rollkit components; EVM engine ports remain fixed
func generateTestPorts() (*TestPorts, error) {
	ports := &TestPorts{}

	// Generate unique ports for DA and rollkit components
	daPort, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get DA port: %w", err)
	}

	rollkitRPCPort, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get rollkit RPC port: %w", err)
	}

	rollkitP2PPort, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get rollkit P2P port: %w", err)
	}

	fullNodeP2PPort, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get full node P2P port: %w", err)
	}

	fullNodeRPCPort, err := getAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get full node RPC port: %w", err)
	}

	ports.DAPort = strconv.Itoa(daPort)
	ports.RollkitRPCPort = strconv.Itoa(rollkitRPCPort)
	ports.RollkitP2PPort = strconv.Itoa(rollkitP2PPort)
	ports.FullNodeP2PPort = strconv.Itoa(fullNodeP2PPort)
	ports.FullNodeRPCPort = strconv.Itoa(fullNodeRPCPort)

	return ports, nil
}

// Common constants used across EVM tests
const (
	// Docker configuration
	dockerPath = "../../execution/evm/docker"

	// Port configurations
	SequencerEthPort    = "8545"
	SequencerEnginePort = "8551"
	FullNodeEthPort     = "8555"
	FullNodeEnginePort  = "8561"
	DAPort              = "7980"
	RollkitRPCPort      = "7331"
	RollkitP2PPort      = "7676"
	FullNodeP2PPort     = "7677"
	FullNodeRPCPort     = "46657"

	// URL templates
	SequencerEthURL    = "http://localhost:" + SequencerEthPort
	SequencerEngineURL = "http://localhost:" + SequencerEnginePort
	FullNodeEthURL     = "http://localhost:" + FullNodeEthPort
	FullNodeEngineURL  = "http://localhost:" + FullNodeEnginePort
	DAAddress          = "http://localhost:" + DAPort
	RollkitRPCAddress  = "http://127.0.0.1:" + RollkitRPCPort

	// Test configuration
	DefaultBlockTime   = "150ms"
	DefaultDABlockTime = "1s"
	DefaultTestTimeout = 20 * time.Second
	DefaultChainID     = "1234"
	DefaultGasLimit    = 22000

	// Test account configuration
	TestPrivateKey = "cece4f25ac74deb1468965160c7185e07dff413f23fcadb611b05ca37ab0a52e"
	TestToAddress  = "0x944fDcD1c868E3cC566C78023CcB38A32cDA836E"
	TestPassphrase = "secret"
)

const (
	SlowPollingInterval = 250 * time.Millisecond // Reduced from 500ms

	NodeStartupTimeout = 8 * time.Second // Increased back for CI stability

)

// getNodeP2PAddress uses the net-info command to get the P2P address of a node.
// This is more reliable than parsing logs as it directly queries the node's state.
//
// Parameters:
// - nodeHome: Directory path for the node data
// - rpcPort: Optional RPC port to use (if empty, uses default port)
//
// Returns: The full P2P address (e.g., /ip4/127.0.0.1/tcp/7676/p2p/12D3KooW...)
func getNodeP2PAddress(t *testing.T, sut *SystemUnderTest, nodeHome string, rpcPort ...string) string {
	t.Helper()

	// Build command arguments
	args := []string{"net-info", "--home", nodeHome}

	// Add custom RPC address if provided
	if len(rpcPort) > 0 && rpcPort[0] != "" {
		args = append(args, "--rollkit.rpc.address", "127.0.0.1:"+rpcPort[0])
	}

	// Run net-info command to get node network information
	output, err := sut.RunCmd(evmSingleBinaryPath, args...)
	require.NoError(t, err, "failed to get net-info", output)

	// Parse the output to extract the full P2P address
	lines := strings.Split(output, "\n")

	var p2pAddress string
	for _, line := range lines {
		// Look for the listen address line with the full P2P address
		if strings.Contains(line, "Addr:") && strings.Contains(line, "/p2p/") {
			// Extract everything after "Addr: "
			addrIdx := strings.Index(line, "Addr:")
			if addrIdx != -1 {
				addrPart := line[addrIdx+5:] // Skip "Addr:"
				addrPart = strings.TrimSpace(addrPart)

				// If this line contains the full P2P address format
				if strings.Contains(addrPart, "/ip4/") && strings.Contains(addrPart, "/tcp/") && strings.Contains(addrPart, "/p2p/") {
					// Remove ANSI color codes if present
					ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
					p2pAddress = ansiRegex.ReplaceAllString(addrPart, "")
					break
				}
			}
		}
	}

	require.NotEmpty(t, p2pAddress, "could not extract P2P address from net-info output: %s", output)

	t.Logf("Extracted P2P address: %s", p2pAddress)
	return p2pAddress
}

// setupSequencerNode initializes and starts the sequencer node with proper configuration.
// This function handles:
// - Node initialization with aggregator mode enabled
// - Sequencer-specific configuration (block time, DA layer connection)
// - JWT authentication setup for EVM engine communication
// - Waiting for node to become responsive on the RPC endpoint
//
// Parameters:
// - sequencerHome: Directory path for sequencer node data
// - jwtSecret: JWT secret for authenticating with EVM engine
// - genesisHash: Hash of the genesis block for chain validation
// - ports: TestPorts struct containing unique port assignments
func setupSequencerNode(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string, ports *TestPorts) {
	t.Helper()

	// Initialize sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Resolve EVM endpoints from provided ports and start the sequencer
	seqEthURL := ports.SequencerEthURL
	seqEngineURL := ports.SequencerEngineURL
	args := []string{
		"start",
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
		"--rollkit.da.block_time", DefaultDABlockTime,
		"--rollkit.da.address", "http://localhost:" + ports.DAPort,
		"--rollkit.rpc.address", "127.0.0.1:" + ports.RollkitRPCPort,
		"--rollkit.p2p.listen_address", "/ip4/127.0.0.1/tcp/" + ports.RollkitP2PPort,
		"--evm.engine-url", seqEngineURL,
		"--evm.eth-url", seqEthURL,
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, "http://127.0.0.1:"+ports.RollkitRPCPort, NodeStartupTimeout)
}

// setupSequencerNodeLazy initializes and starts the sequencer node in lazy mode.
// In lazy mode, blocks are only produced when transactions are available,
// not on a regular timer.
func setupSequencerNodeLazy(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string, ports *TestPorts) {
	t.Helper()

	// Initialize sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Resolve EVM endpoints
	seqEthURL := ports.SequencerEthURL
	seqEngineURL := ports.SequencerEngineURL
	args := []string{
		"start",
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.node.lazy_mode=true",
		"--rollkit.node.lazy_block_interval=60s",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
		"--rollkit.da.block_time", DefaultDABlockTime,
		"--rollkit.da.address", "http://localhost:" + ports.DAPort,
		"--rollkit.rpc.address", "127.0.0.1:" + ports.RollkitRPCPort,
		"--rollkit.p2p.listen_address", "/ip4/127.0.0.1/tcp/" + ports.RollkitP2PPort,
		"--evm.engine-url", seqEngineURL,
		"--evm.eth-url", seqEthURL,
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, "http://127.0.0.1:"+ports.RollkitRPCPort, NodeStartupTimeout)
}

// setupFullNode initializes and starts the full node with P2P connection to sequencer.
// This function handles:
// - Full node initialization (non-aggregator mode)
// - Genesis file copying from sequencer to ensure chain consistency
// - P2P configuration to connect with the sequencer node
// - Different EVM engine ports (8555/8561) to avoid conflicts
// - DA layer connection for long-term data availability
//
// Parameters:
// - fullNodeHome: Directory path for full node data
// - sequencerHome: Directory path of sequencer (for genesis file copying)
// - fullNodeJwtSecret: JWT secret for full node's EVM engine
// - genesisHash: Hash of the genesis block for chain validation
// - sequencerP2PAddress: P2P address of the sequencer node to connect to
// - ports: TestPorts struct containing unique port assignments
func setupFullNode(t *testing.T, sut *SystemUnderTest, fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash, sequencerP2PAddress string, ports *TestPorts) {
	t.Helper()

	// Initialize full node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--home", fullNodeHome,
	)
	require.NoError(t, err, "failed to init full node", output)

	// Copy genesis file from sequencer to full node
	sequencerGenesis := filepath.Join(sequencerHome, "config", "genesis.json")
	fullNodeGenesis := filepath.Join(fullNodeHome, "config", "genesis.json")
	genesisData, err := os.ReadFile(sequencerGenesis)
	require.NoError(t, err, "failed to read sequencer genesis file")
	err = os.WriteFile(fullNodeGenesis, genesisData, 0644)
	require.NoError(t, err, "failed to write full node genesis file")

	// Use dynamic EVM endpoints from ports
	fnEthURL := ports.FullNodeEthURL
	fnEngineURL := ports.FullNodeEngineURL

	args := []string{
		"start",
		"--home", fullNodeHome,
		"--evm.jwt-secret", fullNodeJwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.p2p.peers", sequencerP2PAddress,
		"--evm.engine-url", fnEngineURL,
		"--evm.eth-url", fnEthURL,
		"--rollkit.da.block_time", DefaultDABlockTime,
		"--rollkit.da.address", "http://localhost:" + ports.DAPort,
		"--rollkit.rpc.address", "127.0.0.1:" + ports.FullNodeRPCPort,
		"--rollkit.p2p.listen_address", "/ip4/127.0.0.1/tcp/" + ports.FullNodeP2PPort,
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, "http://127.0.0.1:"+ports.FullNodeRPCPort, NodeStartupTimeout)
}

// Global nonce counter to ensure unique nonces across multiple transaction submissions
var globalNonce uint64 = 0

// submitTransactionAndGetBlockNumber submits a transaction to the sequencer and returns inclusion details.
// This function:
// - Creates a random transaction with proper nonce sequencing
// - Submits it to the sequencer's EVM endpoint
// - Waits for the transaction to be included in a block
// - Returns both the transaction hash and the block number where it was included
//
// Returns:
// - Transaction hash for later verification
// - Block number where the transaction was included
//
// This is used in full node sync tests to verify that both nodes
// include the same transaction in the same block number.
func submitTransactionAndGetBlockNumber(t *testing.T, sequencerClient *ethclient.Client) (common.Hash, uint64) {
	t.Helper()

	// Submit transaction to sequencer EVM with unique nonce
	tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &globalNonce)
	require.NoError(t, sequencerClient.SendTransaction(context.Background(), tx))

	// Wait for transaction to be included and get block number
	ctx := context.Background()
	var txBlockNumber uint64
	require.Eventually(t, func() bool {
		receipt, err := sequencerClient.TransactionReceipt(ctx, tx.Hash())
		if err == nil && receipt != nil && receipt.Status == 1 {
			txBlockNumber = receipt.BlockNumber.Uint64()
			return true
		}
		return false
	}, 8*time.Second, SlowPollingInterval)

	return tx.Hash(), txBlockNumber
}

// setupCommonEVMTest performs common setup for EVM tests including DA and EVM engine initialization.
// This helper reduces code duplication across multiple test functions.
//
// Parameters:
// - needsFullNode: whether to set up a full node EVM engine in addition to sequencer
// - daPort: optional DA port to use (if empty, uses default)
//
// Returns: jwtSecret, fullNodeJwtSecret (empty if needsFullNode=false), genesisHash
func setupCommonEVMTest(t *testing.T, sut *SystemUnderTest, needsFullNode bool, _ ...string) (string, string, string, *TestPorts) {
	t.Helper()

	// Reset global nonce for each test to ensure clean state
	globalNonce = 0

	// Construct dynamic test ports (rollkit + DA)
	dynPorts, err := generateTestPorts()
	require.NoError(t, err, "failed to generate dynamic test ports")

	// Start local DA explicitly on the chosen port
	localDABinary := "local-da"
	if evmSingleBinaryPath != "evm-single" {
		localDABinary = filepath.Join(filepath.Dir(evmSingleBinaryPath), "local-da")
	}
	sut.ExecCmd(localDABinary, "-port", dynPorts.DAPort)
	t.Logf("Started local DA on port %s", dynPorts.DAPort)

	rethNode := evm.SetupTestRethEngine(t)

	networkInfo, err := rethNode.GetNetworkInfo(context.Background())
	require.NoError(t, err, "failed to get reth network info")

	seqJWT := rethNode.JWTSecretHex()
	seqEth := "http://localhost:" + networkInfo.External.Ports.RPC
	seqEngine := "http://localhost:" + networkInfo.External.Ports.Engine

	var fnJWT, fnEth, fnEngine string
	if needsFullNode {
		rethFn := evm.SetupTestRethEngineFullNode(t)

		fnInfo, err := rethFn.GetNetworkInfo(context.Background())
		require.NoError(t, err, "failed to get reth network info")

		fnJWT = rethFn.JWTSecretHex()
		fnEth = "http://localhost:" + fnInfo.External.Ports.RPC
		fnEngine = "http://localhost:" + fnInfo.External.Ports.Engine
	}

	// get genesis hash by querying the sequencer ETH endpoint
	genesisHash, err := rethNode.GenesisHash(context.Background())
	require.NoError(t, err, "failed to get genesis hash")

	// Populate ports with both dynamic rollkit ports and dynamic engine endpoints
	dynPorts.SequencerEthURL = seqEth
	dynPorts.SequencerEngineURL = seqEngine
	dynPorts.FullNodeEthURL = fnEth
	dynPorts.FullNodeEngineURL = fnEngine

	return seqJWT, fnJWT, genesisHash, dynPorts
}

// checkTxIncludedAt checks if a transaction was included in a block at the specified EVM endpoint.
// This utility function connects to the provided EVM endpoint and queries for the
// transaction receipt to determine if the transaction was successfully included.
//
// Parameters:
// - txHash: Hash of the transaction to check
// - ethURL: EVM endpoint URL to query (e.g., http://localhost:8545)
//
// Returns: true if transaction is included with success status, false otherwise
func checkTxIncludedAt(t *testing.T, txHash common.Hash, ethURL string) bool {
	t.Helper()
	rpcClient, err := ethclient.Dial(ethURL)
	if err != nil {
		return false
	}
	defer rpcClient.Close()
	receipt, err := rpcClient.TransactionReceipt(context.Background(), txHash)
	return err == nil && receipt != nil && receipt.Status == 1
}

// checkBlockInfoAt retrieves block information at a specific height including state root.
// This function connects to the specified EVM endpoint and queries for the block header
// to get the block hash, state root, transaction count, and other block metadata.
//
// Parameters:
// - ethURL: EVM endpoint URL to query (e.g., http://localhost:8545)
// - blockHeight: Height of the block to retrieve (use nil for latest)
//
// Returns: block hash, state root, transaction count, block number, and error
func checkBlockInfoAt(t *testing.T, ethURL string, blockHeight *uint64) (common.Hash, common.Hash, int, uint64, error) {
	t.Helper()

	ctx := context.Background()
	ethClient, err := ethclient.Dial(ethURL)
	if err != nil {
		return common.Hash{}, common.Hash{}, 0, 0, fmt.Errorf("failed to create ethereum client: %w", err)
	}
	defer ethClient.Close()

	var blockNumber *big.Int
	if blockHeight != nil {
		blockNumber = new(big.Int).SetUint64(*blockHeight)
	}

	// Get the block header
	header, err := ethClient.HeaderByNumber(ctx, blockNumber)
	if err != nil {
		return common.Hash{}, common.Hash{}, 0, 0, fmt.Errorf("failed to get block header: %w", err)
	}

	blockHash := header.Hash()
	stateRoot := header.Root
	blockNum := header.Number.Uint64()

	// Get the full block to count transactions
	block, err := ethClient.BlockByNumber(ctx, header.Number)
	if err != nil {
		return blockHash, stateRoot, 0, blockNum, fmt.Errorf("failed to get full block: %w", err)
	}

	txCount := len(block.Transactions())
	return blockHash, stateRoot, txCount, blockNum, nil
}

// setupSequencerOnlyTest performs setup for EVM sequencer-only tests.
// This helper sets up DA, EVM engine, and sequencer node for tests that don't need full nodes.
//
// Parameters:
// - sut: SystemUnderTest instance for managing test processes
// - nodeHome: Directory path for sequencer node data
//
// Returns: genesisHash for the sequencer
func setupSequencerOnlyTest(t *testing.T, sut *SystemUnderTest, nodeHome string) (string, string) {
	t.Helper()

	// Use common setup (no full node needed)
	jwtSecret, _, genesisHash, ports := setupCommonEVMTest(t, sut, false)

	// Initialize and start sequencer node (use nil ports for backward compatibility)
	setupSequencerNode(t, sut, nodeHome, jwtSecret, genesisHash, ports)
	t.Log("Sequencer node is up")

	return genesisHash, ports.SequencerEthURL
}

// restartDAAndSequencer restarts both the local DA and sequencer node.
// This is used for restart scenarios where all processes were shutdown.
// This function is shared between multiple restart tests.
//
// Parameters:
// - sut: SystemUnderTest instance for managing test processes
// - sequencerHome: Directory path for sequencer node data
// - jwtSecret: JWT secret for sequencer's EVM engine authentication
// - genesisHash: Hash of the genesis block for chain validation
func restartDAAndSequencer(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string) {
	t.Helper()

	// First restart the local DA
	localDABinary := "local-da"
	if evmSingleBinaryPath != "evm-single" {
		localDABinary = filepath.Join(filepath.Dir(evmSingleBinaryPath), "local-da")
	}
	sut.ExecCmd(localDABinary)
	t.Log("Restarted local DA")
	time.Sleep(25 * time.Millisecond)

	// Then restart the sequencer node (without init - node already exists)
	sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
		"--rollkit.da.address", DAAddress,
		"--rollkit.da.block_time", DefaultDABlockTime,
	)

	time.Sleep(SlowPollingInterval)

	sut.AwaitNodeUp(t, RollkitRPCAddress, NodeStartupTimeout)
}

// restartDAAndSequencerLazy restarts both the local DA and sequencer node in lazy mode.
// This is used for restart scenarios where all processes were shutdown and we want
// to restart the sequencer in lazy mode.
// This function is shared between multiple restart tests.
//
// Parameters:
// - sut: SystemUnderTest instance for managing test processes
// - sequencerHome: Directory path for sequencer node data
// - jwtSecret: JWT secret for sequencer's EVM engine authentication
// - genesisHash: Hash of the genesis block for chain validation
func restartDAAndSequencerLazy(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string) {
	t.Helper()

	// First restart the local DA
	localDABinary := "local-da"
	if evmSingleBinaryPath != "evm-single" {
		localDABinary = filepath.Join(filepath.Dir(evmSingleBinaryPath), "local-da")
	}
	sut.ExecCmd(localDABinary)
	t.Log("Restarted local DA")
	time.Sleep(25 * time.Millisecond)

	// Then restart the sequencer node in lazy mode (without init - node already exists)
	sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.node.lazy_mode=true",          // Enable lazy mode
		"--rollkit.node.lazy_block_interval=60s", // Set lazy block interval to 60 seconds to prevent timer-based block production during test
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
		"--rollkit.da.address", DAAddress,
		"--rollkit.da.block_time", DefaultDABlockTime,
	)

	time.Sleep(SlowPollingInterval)

	sut.AwaitNodeUp(t, RollkitRPCAddress, NodeStartupTimeout)
}

// restartSequencerNode starts an existing sequencer node without initialization.
// This is used for restart scenarios where the node has already been initialized.
//
// Parameters:
// - sut: SystemUnderTest instance for managing test processes
// - sequencerHome: Directory path for sequencer node data
// - jwtSecret: JWT secret for sequencer's EVM engine authentication
// - genesisHash: Hash of the genesis block for chain validation
func restartSequencerNode(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string) {
	t.Helper()

	// Start sequencer node (without init - node already exists)
	sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
		"--rollkit.da.address", DAAddress,
		"--rollkit.da.block_time", DefaultDABlockTime,
	)

	time.Sleep(SlowPollingInterval)

	sut.AwaitNodeUp(t, RollkitRPCAddress, NodeStartupTimeout)
}

// verifyNoBlockProduction verifies that no new blocks are being produced over a specified duration.
// This is used to test lazy mode behavior where blocks should only be produced when
// transactions are submitted.
//
// Parameters:
// - client: Ethereum client to monitor for block production
// - duration: How long to monitor for block production
// - nodeName: Human-readable name for logging (e.g., "sequencer", "full node")
//
// This function ensures that during lazy mode idle periods, no automatic block production occurs.
func verifyNoBlockProduction(t *testing.T, client *ethclient.Client, duration time.Duration, nodeName string) {
	t.Helper()

	ctx := context.Background()

	// Get initial height
	initialHeader, err := client.HeaderByNumber(ctx, nil)
	require.NoError(t, err, "Should get initial header from %s", nodeName)
	initialHeight := initialHeader.Number.Uint64()

	t.Logf("Initial %s height: %d, monitoring for %v", nodeName, initialHeight, duration)

	// Monitor for the specified duration
	endTime := time.Now().Add(duration)
	for time.Now().Before(endTime) {
		currentHeader, err := client.HeaderByNumber(ctx, nil)
		require.NoError(t, err, "Should get current header from %s", nodeName)
		currentHeight := currentHeader.Number.Uint64()

		// Verify height hasn't increased
		require.Equal(t, initialHeight, currentHeight,
			"%s should not produce new blocks during idle period (started at %d, now at %d)",
			nodeName, initialHeight, currentHeight)

		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("✅ %s maintained height %d for %v (no new blocks produced)", nodeName, initialHeight, duration)
}
