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

	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	"github.com/evstack/ev-node/execution/evm"
	evmtest "github.com/evstack/ev-node/execution/evm/test"
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

// TestEndpoints holds unique port numbers for each test instance
type TestEndpoints struct {
	DAPort              string
	RollkitRPCPort      string
	RollkitP2PPort      string
	FullNodeP2PPort     string
	FullNodeRPCPort     string
	SequencerEthPort    string
	SequencerEnginePort string
	FullNodeEthPort     string
	FullNodeEnginePort  string
}

// GetSequencerEthURL returns the complete ETH URL for the sequencer
func (te *TestEndpoints) GetSequencerEthURL() string {
	return "http://127.0.0.1:" + te.SequencerEthPort
}

// GetSequencerEngineURL returns the complete engine URL for the sequencer
func (te *TestEndpoints) GetSequencerEngineURL() string {
	return "http://127.0.0.1:" + te.SequencerEnginePort
}

// GetFullNodeEthURL returns the complete ETH URL for the full node
func (te *TestEndpoints) GetFullNodeEthURL() string {
	return "http://127.0.0.1:" + te.FullNodeEthPort
}

// GetFullNodeEngineURL returns the complete engine URL for the full node
func (te *TestEndpoints) GetFullNodeEngineURL() string {
	return "http://127.0.0.1:" + te.FullNodeEnginePort
}

// GetDAAddress returns the complete DA address
func (te *TestEndpoints) GetDAAddress() string {
	return "http://127.0.0.1:" + te.DAPort
}

// GetRollkitRPCAddress returns the complete rollkit RPC address
func (te *TestEndpoints) GetRollkitRPCAddress() string {
	return "http://127.0.0.1:" + te.RollkitRPCPort
}

// GetRollkitRPCListen returns the rollkit RPC listen address for CLI usage
func (te *TestEndpoints) GetRollkitRPCListen() string {
	return "127.0.0.1:" + te.RollkitRPCPort
}

// GetFullNodeRPCAddress returns the complete full node RPC address
func (te *TestEndpoints) GetFullNodeRPCAddress() string {
	return "http://127.0.0.1:" + te.FullNodeRPCPort
}

// GetFullNodeRPCListen returns the full node RPC listen address for CLI usage
func (te *TestEndpoints) GetFullNodeRPCListen() string {
	return "127.0.0.1:" + te.FullNodeRPCPort
}

// GetRollkitP2PAddress returns the complete rollkit P2P listen address
func (te *TestEndpoints) GetRollkitP2PAddress() string {
	return "/ip4/127.0.0.1/tcp/" + te.RollkitP2PPort
}

// GetFullNodeP2PAddress returns the complete full node P2P listen address
func (te *TestEndpoints) GetFullNodeP2PAddress() string {
	return "/ip4/127.0.0.1/tcp/" + te.FullNodeP2PPort
}

// generateTestEndpoints creates a set of unique ports for a test instance
// Only generates ports for rollkit components; EVM engine ports will be set dynamically
func generateTestEndpoints() (*TestEndpoints, error) {
	endpoints := &TestEndpoints{}

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

	endpoints.DAPort = strconv.Itoa(daPort)
	endpoints.RollkitRPCPort = strconv.Itoa(rollkitRPCPort)
	endpoints.RollkitP2PPort = strconv.Itoa(rollkitP2PPort)
	endpoints.FullNodeP2PPort = strconv.Itoa(fullNodeP2PPort)
	endpoints.FullNodeRPCPort = strconv.Itoa(fullNodeRPCPort)

	return endpoints, nil
}

// Common constants used across EVM tests
const (
	// Port configurations
	DAPort         = "7980"
	RollkitRPCPort = "7331"

	DAAddress         = "http://127.0.0.1:" + DAPort
	RollkitRPCAddress = "http://127.0.0.1:" + RollkitRPCPort

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
// - endpoints: TestEndpoints struct containing unique port assignments
func setupSequencerNode(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string, endpoints *TestEndpoints) {
	t.Helper()

	// Initialize sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Use helper methods to get complete URLs
	args := []string{
		"start",
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
		"--rollkit.da.block_time", DefaultDABlockTime,
		"--rollkit.da.address", endpoints.GetDAAddress(),
		"--rollkit.rpc.address", endpoints.GetRollkitRPCListen(),
		"--rollkit.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
}

// setupSequencerNodeLazy initializes and starts the sequencer node in lazy mode.
// In lazy mode, blocks are only produced when transactions are available,
// not on a regular timer.
func setupSequencerNodeLazy(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string, endpoints *TestEndpoints) {
	t.Helper()

	// Initialize sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	// Use helper methods to get complete URLs
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
		"--rollkit.da.address", endpoints.GetDAAddress(),
		"--rollkit.rpc.address", endpoints.GetRollkitRPCListen(),
		"--rollkit.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
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
// - endpoints: TestEndpoints struct containing unique port assignments
func setupFullNode(t *testing.T, sut *SystemUnderTest, fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash, sequencerP2PAddress string, endpoints *TestEndpoints) {
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

	// Use helper methods to get complete URLs
	args := []string{
		"start",
		"--home", fullNodeHome,
		"--evm.jwt-secret", fullNodeJwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.p2p.peers", sequencerP2PAddress,
		"--evm.engine-url", endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", endpoints.GetFullNodeEthURL(),
		"--rollkit.da.block_time", DefaultDABlockTime,
		"--rollkit.da.address", endpoints.GetDAAddress(),
		"--rollkit.rpc.address", endpoints.GetFullNodeRPCListen(),
		"--rollkit.p2p.listen_address", endpoints.GetFullNodeP2PAddress(),
	}
	sut.ExecCmd(evmSingleBinaryPath, args...)
	sut.AwaitNodeUp(t, endpoints.GetFullNodeRPCAddress(), NodeStartupTimeout)
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
func setupCommonEVMTest(t *testing.T, sut *SystemUnderTest, needsFullNode bool, _ ...string) (string, string, string, *TestEndpoints) {
	t.Helper()

	// Reset global nonce for each test to ensure clean state
	globalNonce = 0

	// Construct dynamic test ports (rollkit + DA)
	dynEndpoints, err := generateTestEndpoints()
	require.NoError(t, err, "failed to generate dynamic test endpoints")

	// Start local DA explicitly on the chosen port
	localDABinary := "local-da"
	if evmSingleBinaryPath != "evm-single" {
		localDABinary = filepath.Join(filepath.Dir(evmSingleBinaryPath), "local-da")
	}
	sut.ExecCmd(localDABinary, "-port", dynEndpoints.DAPort)
	t.Logf("Started local DA on port %s", dynEndpoints.DAPort)

	rethNode := evmtest.SetupTestRethNode(t)

	networkInfo, err := rethNode.GetNetworkInfo(context.Background())
	require.NoError(t, err, "failed to get reth network info")

	seqJWT := rethNode.JWTSecretHex()

	var fnJWT string
	var rethFn *reth.Node
	if needsFullNode {
		rethFn = evmtest.SetupTestRethNode(t)
		fnJWT = rethFn.JWTSecretHex()
	}

	// get genesis hash by querying the sequencer ETH endpoint
	genesisHash, err := rethNode.GenesisHash(context.Background())
	require.NoError(t, err, "failed to get genesis hash")

	// Populate endpoints with both dynamic rollkit ports and dynamic engine ports
	dynEndpoints.SequencerEthPort = networkInfo.External.Ports.RPC
	dynEndpoints.SequencerEnginePort = networkInfo.External.Ports.Engine
	if needsFullNode {
		fnInfo, err := rethFn.GetNetworkInfo(context.Background())
		require.NoError(t, err, "failed to get full node reth network info")
		dynEndpoints.FullNodeEthPort = fnInfo.External.Ports.RPC
		dynEndpoints.FullNodeEnginePort = fnInfo.External.Ports.Engine
	}

	return seqJWT, fnJWT, genesisHash, dynEndpoints
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
	jwtSecret, _, genesisHash, endpoints := setupCommonEVMTest(t, sut, false)

	// Initialize and start sequencer node
	setupSequencerNode(t, sut, nodeHome, jwtSecret, genesisHash, endpoints)
	t.Log("Sequencer node is up")

	return genesisHash, endpoints.GetSequencerEthURL()
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
func restartDAAndSequencer(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string, endpoints *TestEndpoints) {
	t.Helper()

	// First restart the local DA
	localDABinary := "local-da"
	if evmSingleBinaryPath != "evm-single" {
		localDABinary = filepath.Join(filepath.Dir(evmSingleBinaryPath), "local-da")
	}
	sut.ExecCmd(localDABinary, "-port", endpoints.DAPort)
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
		"--rollkit.da.address", endpoints.GetDAAddress(),
		"--rollkit.da.block_time", DefaultDABlockTime,
		"--rollkit.rpc.address", endpoints.GetRollkitRPCListen(),
		"--rollkit.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
	)

	time.Sleep(SlowPollingInterval)

	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
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
func restartDAAndSequencerLazy(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string, endpoints *TestEndpoints) {
	t.Helper()

	// First restart the local DA
	localDABinary := "local-da"
	if evmSingleBinaryPath != "evm-single" {
		localDABinary = filepath.Join(filepath.Dir(evmSingleBinaryPath), "local-da")
	}
	sut.ExecCmd(localDABinary, "-port", endpoints.DAPort)
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
		"--rollkit.da.address", endpoints.GetDAAddress(),
		"--rollkit.da.block_time", DefaultDABlockTime,
		"--rollkit.rpc.address", endpoints.GetRollkitRPCListen(),
		"--rollkit.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
	)

	time.Sleep(SlowPollingInterval)

	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
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
