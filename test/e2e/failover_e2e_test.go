//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	evmtest "github.com/evstack/ev-node/execution/evm/test"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	coreda "github.com/evstack/ev-node/pkg/da/types"
	rpcclient "github.com/evstack/ev-node/pkg/rpc/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/execution/evm"
	"github.com/evstack/ev-node/pkg/rpc/client"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// TestLeaseFailoverE2E runs three node binaries configured to use raft consensus.
// It forces a leader shutdown and verifies leadership failover occurs in the raft cluster.
func TestLeaseFailoverE2E(t *testing.T) {
	flag.Parse()
	sut := NewSystemUnderTest(t)
	if testing.Verbose() {
		os.Setenv("GOLOG_LOG_LEVEL", "DEBUG")
		t.Cleanup(func() {
			os.Unsetenv("GOLOG_LOG_LEVEL")
		})
	}

	workDir := t.TempDir()

	// Get JWT secrets and setup common components first
	jwtSecret, fullNodeJwtSecret, genesisHash, testEndpoints := setupCommonEVMTest(t, sut, true)
	rethFn := evmtest.SetupTestRethNode(t)
	jwtSecret3 := rethFn.JWTSecretHex()
	fnInfo, err := rethFn.GetNetworkInfo(context.Background())
	require.NoError(t, err, "failed to get full node reth network info")
	fullNode3EthPort := fnInfo.External.Ports.RPC
	fullNode3EnginePort := fnInfo.External.Ports.Engine

	// Allocate raft ports for all nodes
	node1RaftPort := mustGetAvailablePort(t)
	node2RaftPort := mustGetAvailablePort(t)
	node3RaftPort := mustGetAvailablePort(t)

	// Setup raft addresses
	node1RaftAddr := fmt.Sprintf("127.0.0.1:%d", node1RaftPort)
	node2RaftAddr := fmt.Sprintf("127.0.0.1:%d", node2RaftPort)
	node3RaftAddr := fmt.Sprintf("127.0.0.1:%d", node3RaftPort)
	raftCluster := []string{"node1@" + node1RaftAddr, "node2@" + node2RaftAddr, "node3@" + node3RaftAddr}

	bootstrapDir := filepath.Join(workDir, "bootstrap")
	passphraseFile := initChain(t, sut, bootstrapDir)

	clusterNodes := &raftClusterNodes{
		nodes: make(map[string]*nodeDetails),
	}
	node1P2PAddr := testEndpoints.GetRollkitP2PAddress()
	node2P2PAddr := testEndpoints.GetFullNodeP2PAddress()
	node3P2PAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", mustGetAvailablePort(t))

	// Start node1 (bootstrap mode)
	go func() {
		p2pPeers := node2P2PAddr + "," + node3P2PAddr
		proc := setupRaftSequencerNode(t, sut, workDir, "node1", node1RaftAddr, jwtSecret, genesisHash, testEndpoints.GetDAAddress(),
			bootstrapDir, raftCluster, p2pPeers, testEndpoints.GetRollkitRPCListen(), testEndpoints.GetRollkitP2PAddress(),
			testEndpoints.GetSequencerEngineURL(), testEndpoints.GetSequencerEthURL(), true, passphraseFile)
		clusterNodes.Set("node1", testEndpoints.GetRollkitRPCAddress(), proc, testEndpoints.GetSequencerEthURL(), node1RaftAddr, testEndpoints.GetRollkitP2PAddress(), testEndpoints.GetSequencerEngineURL(), testEndpoints.GetSequencerEthURL())
		t.Log("Node1 is up")
	}()

	// Start node2 (bootstrap node)
	go func() {
		t.Log("Starting Node2")
		p2pPeers := node1P2PAddr + "," + node3P2PAddr
		proc := setupRaftSequencerNode(t, sut, workDir, "node2", node2RaftAddr, fullNodeJwtSecret, genesisHash, testEndpoints.GetDAAddress(), bootstrapDir, raftCluster, p2pPeers, testEndpoints.GetFullNodeRPCListen(), testEndpoints.GetFullNodeP2PAddress(), testEndpoints.GetFullNodeEngineURL(), testEndpoints.GetFullNodeEthURL(), true, passphraseFile)
		clusterNodes.Set("node2", testEndpoints.GetFullNodeRPCAddress(), proc, testEndpoints.GetFullNodeEthURL(), node2RaftAddr, testEndpoints.GetFullNodeP2PAddress(), testEndpoints.GetFullNodeEngineURL(), testEndpoints.GetFullNodeEthURL())
		t.Log("Node2 is up")
	}()

	// Start node3 (bootstrap node)
	node3EthAddr := fmt.Sprintf("http://127.0.0.1:%s", fullNode3EthPort)
	go func() {
		t.Log("Starting Node3")
		p2pPeers := node1P2PAddr + "," + node2P2PAddr
		node3RPCListen := fmt.Sprintf("127.0.0.1:%d", mustGetAvailablePort(t))
		ethEngineURL := fmt.Sprintf("http://127.0.0.1:%s", fullNode3EnginePort)
		proc := setupRaftSequencerNode(t, sut, workDir, "node3", node3RaftAddr, jwtSecret3, genesisHash, testEndpoints.GetDAAddress(), bootstrapDir, raftCluster, p2pPeers, node3RPCListen, node3P2PAddr, ethEngineURL, node3EthAddr, true, passphraseFile)
		clusterNodes.Set("node3", "http://"+node3RPCListen, proc, node3EthAddr, node3RaftAddr, node3P2PAddr, ethEngineURL, node3EthAddr)
		t.Log("Node3 is up")
	}()

	var leaderNode string
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		leaderNode = clusterNodes.Leader(collect)
	}, 5*time.Second, 200*time.Millisecond)

	sut.AwaitNodeUp(t, clusterNodes.Details(leaderNode).rpcAddr, NodeStartupTimeout)
	// Wait for at least 2 blocks to be produced
	sut.AwaitNBlocks(t, 2, clusterNodes.Details(leaderNode).rpcAddr, 6*time.Second)

	// Submit a tx and ensure it propagates to all nodes
	txHash1, blk1 := submitTransactionAndGetBlockNumber(t, clusterNodes.Details(leaderNode).ethClient(t))

	for node, details := range clusterNodes.Followers(t) {
		require.Eventually(t, func() bool {
			rec, err := details.ethClient(t).TransactionReceipt(t.Context(), txHash1)
			return err == nil && rec != nil && rec.Status == 1 && rec.BlockNumber.Uint64() == blk1
		}, 20*time.Second, SlowPollingInterval, "tx1 not seen on "+node)
	}

	oldLeader := leaderNode
	t.Logf("+++ Killing current leader (%s)\n", oldLeader)
	_ = clusterNodes.Details(oldLeader).Kill()

	const daStartHeight = 1
	lastDABlockOldLeader := queryLastDAHeight(t, daStartHeight, jwtSecret, testEndpoints.GetDAAddress())
	t.Log("+++ Last DA block by old leader: ", lastDABlockOldLeader)
	leaderElectionStart := time.Now()

	// Wait for new leader election - submit tx to node2
	var newLeader string
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		newLeader = clusterNodes.Leader(collect)
	}, 5*time.Second, 200*time.Millisecond)
	require.NotEqual(t, oldLeader, newLeader)
	t.Logf("+++ New leader: %s within ~%s\n", newLeader, time.Since(leaderElectionStart))

	// submit TX to new leader
	_, blk2 := submitTxToURL(t, clusterNodes.Details(newLeader).ethClient(t))
	require.Greater(t, blk2, blk1, "post-failover block should advance")

	// Verify DA progress
	var lastDABlockNewLeader uint64
	require.Eventually(t, func() bool {
		lastDABlockNewLeader = queryLastDAHeight(t, lastDABlockOldLeader, jwtSecret, testEndpoints.GetDAAddress())
		return lastDABlockNewLeader > lastDABlockOldLeader
	}, 2*must(time.ParseDuration(DefaultDABlockTime)), 100*time.Millisecond)
	t.Logf("+++ Last DA block by new leader: %d\n", lastDABlockNewLeader)

	// Restart oldLeader to rejoin cluster
	var raftClusterRPCs []string
	for _, f := range clusterNodes.AllNodes() {
		if f.IsRunning() {
			raftClusterRPCs = append(raftClusterRPCs, f.rpcAddr)
		}
	}
	oldDetails := clusterNodes.Details(oldLeader)
	restartedNodeProcess := setupRaftSequencerNode(t, sut, workDir, oldLeader, oldDetails.raftAddr, jwtSecret, genesisHash, testEndpoints.GetDAAddress(), "", raftCluster, clusterNodes.Details(newLeader).p2pAddr, oldDetails.rpcAddr, oldDetails.p2pAddr, oldDetails.engineURL, oldDetails.ethAddr, false, passphraseFile)
	t.Log("Restarted old leader to sync with cluster: " + oldLeader)

	if IsNodeUp(t, oldDetails.rpcAddr, NodeStartupTimeout) {
		clusterNodes.Set(oldLeader, oldDetails.rpcAddr, restartedNodeProcess, oldDetails.ethAddr, oldDetails.raftAddr, "", oldDetails.engineURL, oldDetails.ethAddr)
	} else {
		t.Log("+++ old leader did not recover on restart. Skipping node verification")
	}
	targetHeight := blk2 + 1
	// give node some time to catch up
	for range 10 {
		if bn, _ := clusterNodes.Details(oldLeader).ethClient(t).BlockNumber(t.Context()); bn > targetHeight {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Ensure at least two nodes are actually producing/serving blocks beyond the last known height
	minReady, ready := 2, 0
	running := make(map[string]struct{})
	require.Eventually(t, func() bool {
		ready = 0
		for name, n := range clusterNodes.AllNodes() {
			if !n.IsRunning() {
				continue
			}
			bn, err := n.ethClient(t).BlockNumber(t.Context())
			if err != nil {
				t.Logf("err node %s : %s", name, err)
				continue
			}
			if bn >= targetHeight {
				ready++
				running[name] = struct{}{}
			}
		}
		return ready >= minReady
	}, 20*time.Second, 200*time.Millisecond, "expected at least 2 raft nodes serving height >= %d", targetHeight)
	t.Logf("+++ %d/3 nodes are up and running: %s / old: %s", ready, slices.Collect(maps.Keys(running)), oldLeader)

	t.Log("+++ Verifying no double-signing...")
	var state *pb.State
	require.Eventually(t, func() bool {
		state, err = clusterNodes.Details(newLeader).rpcClient(t).GetState(t.Context())
		return err == nil
	}, time.Second, 100*time.Millisecond)

	lastDABlockNewLeader = queryLastDAHeight(t, lastDABlockNewLeader, jwtSecret, testEndpoints.GetDAAddress())

	genesisHeight := state.InitialHeight
	verifyNoDoubleSigning(t, clusterNodes, genesisHeight, state.LastBlockHeight)

	// wait for the next DA block to ensure all blocks are propagated
	require.Eventually(t, func() bool {
		before := lastDABlockNewLeader
		lastDABlockNewLeader = queryLastDAHeight(t, lastDABlockNewLeader, jwtSecret, testEndpoints.GetDAAddress())
		return before < lastDABlockNewLeader
	}, 2*must(time.ParseDuration(DefaultDABlockTime)), 100*time.Millisecond)

	t.Log("+++ Verifying no DA gaps...")
	verifyDABlocks(t, daStartHeight, lastDABlockNewLeader, jwtSecret, testEndpoints.GetDAAddress(), genesisHeight, state.LastBlockHeight)

	// Cleanup processes
	clusterNodes.killAll()
	t.Logf("Completed leader change in: %s", time.Since(leaderElectionStart))
}

// TestHASequencerRollingRestartE2E tests graceful rolling restart of a 3-node HA sequencer cluster.
// It starts 3 nodes, gracefully stops them one at a time with ~100 blocks downtime each,
// then restarts them sequentially and verifies cluster health, no double-signing, and no DA gaps.
func TestHASequencerRollingRestartE2E(t *testing.T) {
	flag.Parse()
	sut := NewSystemUnderTest(t)
	if testing.Verbose() {
		os.Setenv("GOLOG_LOG_LEVEL", "DEBUG")
		t.Cleanup(func() {
			os.Unsetenv("GOLOG_LOG_LEVEL")
		})
	}

	workDir := t.TempDir()

	// Get JWT secrets and setup common components first
	jwtSecret, fullNodeJwtSecret, genesisHash, testEndpoints := setupCommonEVMTest(t, sut, true)
	rethFn := evmtest.SetupTestRethNode(t)
	jwtSecret3 := rethFn.JWTSecretHex()
	fnInfo, err := rethFn.GetNetworkInfo(context.Background())
	require.NoError(t, err, "failed to get full node reth network info")
	fullNode3EthPort := fnInfo.External.Ports.RPC
	fullNode3EnginePort := fnInfo.External.Ports.Engine

	// Allocate raft ports for all nodes
	node1RaftPort := mustGetAvailablePort(t)
	node2RaftPort := mustGetAvailablePort(t)
	node3RaftPort := mustGetAvailablePort(t)

	// Setup raft addresses
	node1RaftAddr := fmt.Sprintf("127.0.0.1:%d", node1RaftPort)
	node2RaftAddr := fmt.Sprintf("127.0.0.1:%d", node2RaftPort)
	node3RaftAddr := fmt.Sprintf("127.0.0.1:%d", node3RaftPort)
	raftCluster := []string{"node1@" + node1RaftAddr, "node2@" + node2RaftAddr, "node3@" + node3RaftAddr}

	bootstrapDir := filepath.Join(workDir, "bootstrap")
	passphraseFile := initChain(t, sut, bootstrapDir)

	clusterNodes := &raftClusterNodes{
		nodes: make(map[string]*nodeDetails),
	}
	node1P2PAddr := testEndpoints.GetRollkitP2PAddress()
	node2P2PAddr := testEndpoints.GetFullNodeP2PAddress()
	node3P2PAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", mustGetAvailablePort(t))

	// Start node1 (bootstrap mode)
	go func() {
		p2pPeers := node2P2PAddr + "," + node3P2PAddr
		proc := setupRaftSequencerNode(t, sut, workDir, "node1", node1RaftAddr, jwtSecret, genesisHash, testEndpoints.GetDAAddress(),
			bootstrapDir, raftCluster, p2pPeers, testEndpoints.GetRollkitRPCListen(), testEndpoints.GetRollkitP2PAddress(),
			testEndpoints.GetSequencerEngineURL(), testEndpoints.GetSequencerEthURL(), true, passphraseFile)
		clusterNodes.Set("node1", testEndpoints.GetRollkitRPCAddress(), proc, testEndpoints.GetSequencerEthURL(), node1RaftAddr, testEndpoints.GetRollkitP2PAddress(), testEndpoints.GetSequencerEngineURL(), testEndpoints.GetSequencerEthURL())
		t.Log("Node1 is up")
	}()

	// Start node2 (bootstrap node)
	go func() {
		t.Log("Starting Node2")
		p2pPeers := node1P2PAddr + "," + node3P2PAddr
		proc := setupRaftSequencerNode(t, sut, workDir, "node2", node2RaftAddr, fullNodeJwtSecret, genesisHash, testEndpoints.GetDAAddress(), bootstrapDir, raftCluster, p2pPeers, testEndpoints.GetFullNodeRPCListen(), testEndpoints.GetFullNodeP2PAddress(), testEndpoints.GetFullNodeEngineURL(), testEndpoints.GetFullNodeEthURL(), true, passphraseFile)
		clusterNodes.Set("node2", testEndpoints.GetFullNodeRPCAddress(), proc, testEndpoints.GetFullNodeEthURL(), node2RaftAddr, testEndpoints.GetFullNodeP2PAddress(), testEndpoints.GetFullNodeEngineURL(), testEndpoints.GetFullNodeEthURL())
		t.Log("Node2 is up")
	}()

	// Start node3 (bootstrap node)
	node3EthAddr := fmt.Sprintf("http://127.0.0.1:%s", fullNode3EthPort)
	go func() {
		t.Log("Starting Node3")
		p2pPeers := node1P2PAddr + "," + node2P2PAddr
		node3RPCListen := fmt.Sprintf("127.0.0.1:%d", mustGetAvailablePort(t))
		ethEngineURL := fmt.Sprintf("http://127.0.0.1:%s", fullNode3EnginePort)
		proc := setupRaftSequencerNode(t, sut, workDir, "node3", node3RaftAddr, jwtSecret3, genesisHash, testEndpoints.GetDAAddress(), bootstrapDir, raftCluster, p2pPeers, node3RPCListen, node3P2PAddr, ethEngineURL, node3EthAddr, true, passphraseFile)
		clusterNodes.Set("node3", "http://"+node3RPCListen, proc, node3EthAddr, node3RaftAddr, node3P2PAddr, ethEngineURL, node3EthAddr)
		t.Log("Node3 is up")
	}()

	var leaderNode string
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		leaderNode = clusterNodes.Leader(collect)
	}, 5*time.Second, 200*time.Millisecond)

	sut.AwaitNodeUp(t, clusterNodes.Details(leaderNode).rpcAddr, NodeStartupTimeout)
	// Wait for at least 5 blocks to be produced before starting shutdown sequence
	sut.AwaitNBlocks(t, 5, clusterNodes.Details(leaderNode).rpcAddr, 10*time.Second)

	const daStartHeight = 1
	initialDAHeight := queryLastDAHeight(t, daStartHeight, jwtSecret, testEndpoints.GetDAAddress())
	t.Logf("+++ Initial DA height: %d", initialDAHeight)

	// Calculate downtime per node
	const blocksPerDowntime = 30
	downtimePerNode := time.Duration(blocksPerDowntime) * must(time.ParseDuration(DefaultBlockTime))

	// ===== ROLLING SHUTDOWN PHASE =====
	t.Log("+++ Starting rolling shutdown phase...")
	nodeOrder := []string{"node1", "node2", "node3"}

	for i, nodeName := range nodeOrder {
		nodeDetails := clusterNodes.Details(nodeName)
		if nodeDetails == nil || !nodeDetails.IsRunning() {
			t.Logf("Node %s not running, skipping shutdown", nodeName)
			continue
		}

		t.Logf("+++ [%d/3] Gracefully stopping %s", i+1, nodeName)
		_ = nodeDetails.GracefulStop()

		// Wait for ~100 blocks worth of time
		t.Logf("+++ Waiting %v (~%d blocks) before next action", downtimePerNode, blocksPerDowntime)
		time.Sleep(downtimePerNode)
	}

	t.Log("+++ All nodes stopped. Starting rolling restart phase...")

	// ===== ROLLING RESTART PHASE =====
	// Helper to get JWT secret for a node
	getNodeJWT := func(nodeName string) string {
		switch nodeName {
		case "node1":
			return jwtSecret
		case "node2":
			return fullNodeJwtSecret
		case "node3":
			return jwtSecret3
		}
		return ""
	}

	// Helper to get P2P peers for a node
	getP2PPeers := func(nodeName string) string {
		switch nodeName {
		case "node1":
			return node2P2PAddr + "," + node3P2PAddr
		case "node2":
			return node1P2PAddr + "," + node3P2PAddr
		case "node3":
			return node1P2PAddr + "," + node2P2PAddr
		}
		return ""
	}

	// Helper to restart a node
	restartNode := func(nodeName string) {
		nodeDetails := clusterNodes.Details(nodeName)
		nodeJWT := getNodeJWT(nodeName)
		p2pPeers := getP2PPeers(nodeName)

		restartedProc := setupRaftSequencerNode(t, sut, workDir, nodeName, nodeDetails.raftAddr, nodeJWT, genesisHash,
			testEndpoints.GetDAAddress(), "", raftCluster, p2pPeers,
			strings.TrimPrefix(nodeDetails.rpcAddr, "http://"), nodeDetails.p2pAddr,
			nodeDetails.engineURL, nodeDetails.ethAddr, false, passphraseFile)

		clusterNodes.Set(nodeName, nodeDetails.rpcAddr, restartedProc, nodeDetails.ethAddr,
			nodeDetails.raftAddr, nodeDetails.p2pAddr, nodeDetails.engineURL, nodeDetails.ethAddr)
	}

	// Initial restart of all nodes
	for i, nodeName := range nodeOrder {
		t.Logf("+++ [%d/3] Restarting %s", i+1, nodeName)
		restartNode(nodeName)

		// Wait for node to come up
		if IsNodeUp(t, clusterNodes.Details(nodeName).rpcAddr, NodeStartupTimeout) {
			t.Logf("+++ %s is back up", nodeName)
		} else {
			t.Logf("+++ %s did not recover on initial restart (expected for state mismatch)", nodeName)
		}

		// Brief pause between restarts
		time.Sleep(2 * time.Second)
	}

	// Retry loop: keep restarting any nodes that are not responsive until all are online
	// This handles the expected case where nodes may die due to state mismatch
	// Note: We use IsNodeUp to check actual responsiveness, not IsRunning() which only
	// tracks if we started the process (doesn't detect crashes)
	const maxRestartAttempts = 10
	const restartRetryInterval = 3 * time.Second
	const nodeCheckTimeout = 5 * time.Second

	for attempt := 1; attempt <= maxRestartAttempts; attempt++ {
		// Check which nodes are not actually responsive (using IsNodeUp, not IsRunning)
		var deadNodes []string
		for _, nodeName := range nodeOrder {
			nodeDetails := clusterNodes.Details(nodeName)
			if nodeDetails == nil || !IsNodeUp(t, nodeDetails.rpcAddr, nodeCheckTimeout) {
				deadNodes = append(deadNodes, nodeName)
			}
		}

		if len(deadNodes) == 0 {
			t.Log("+++ All nodes are responsive")
			break
		}

		t.Logf("+++ Attempt %d/%d: Nodes not responsive: %v, restarting...", attempt, maxRestartAttempts, deadNodes)

		for _, nodeName := range deadNodes {
			restartNode(nodeName)
			t.Logf("+++ Restarted %s", nodeName)
		}

		// Wait a bit before checking again
		time.Sleep(restartRetryInterval)
	}

	// Final check: ensure all nodes are actually responsive
	var finalDeadNodes []string
	for _, nodeName := range nodeOrder {
		nodeDetails := clusterNodes.Details(nodeName)
		if nodeDetails == nil || !IsNodeUp(t, nodeDetails.rpcAddr, nodeCheckTimeout) {
			finalDeadNodes = append(finalDeadNodes, nodeName)
		}
	}
	require.Empty(t, finalDeadNodes, "some nodes failed to come online after %d restart attempts: %v", maxRestartAttempts, finalDeadNodes)

	// Wait for leader election after all nodes are up
	t.Log("+++ Waiting for leader election after restarts...")
	var newLeader string
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		newLeader = clusterNodes.Leader(collect)
	}, 10*time.Second, 500*time.Millisecond)
	t.Logf("+++ New leader after restart: %s", newLeader)

	// Submit a transaction to verify block production is working
	t.Log("+++ Verifying block production by submitting transaction...")
	txHash, txBlock := submitTxToURL(t, clusterNodes.Details(newLeader).ethClient(t))
	t.Logf("+++ Transaction %s included in block %d", txHash.Hex(), txBlock)

	// ===== VERIFICATION PHASE =====
	t.Log("+++ Starting verification phase...")

	// Ensure all three nodes are actually producing/serving blocks
	minReady, ready := 3, 0
	running := make(map[string]struct{})
	require.Eventually(t, func() bool {
		ready = 0
		for name, n := range clusterNodes.AllNodes() {
			if !n.IsRunning() {
				continue
			}
			bn, err := n.ethClient(t).BlockNumber(t.Context())
			if err != nil {
				t.Logf("err node %s : %s", name, err)
				continue
			}
			if bn >= txBlock {
				ready++
				running[name] = struct{}{}
			}
		}
		return ready >= minReady
	}, 30*time.Second, 500*time.Millisecond, "expected all 3 raft nodes serving height >= %d", txBlock)
	t.Logf("+++ %d/3 nodes are up and running: %s", ready, slices.Collect(maps.Keys(running)))

	// Verify no double-signing
	t.Log("+++ Verifying no double-signing...")
	var state *pb.State
	require.Eventually(t, func() bool {
		state, err = clusterNodes.Details(newLeader).rpcClient(t).GetState(t.Context())
		return err == nil
	}, time.Second, 100*time.Millisecond)

	lastDABlock := queryLastDAHeight(t, initialDAHeight, jwtSecret, testEndpoints.GetDAAddress())

	genesisHeight := state.InitialHeight
	verifyNoDoubleSigning(t, clusterNodes, genesisHeight, state.LastBlockHeight)
	t.Log("+++ No double-signing detected ✓")

	// Wait for the next DA block to ensure all blocks are propagated
	require.Eventually(t, func() bool {
		before := lastDABlock
		lastDABlock = queryLastDAHeight(t, lastDABlock, jwtSecret, testEndpoints.GetDAAddress())
		return before < lastDABlock
	}, 2*must(time.ParseDuration(DefaultDABlockTime)), 100*time.Millisecond)

	// Verify no DA gaps
	t.Log("+++ Verifying no DA gaps...")
	verifyDABlocks(t, daStartHeight, lastDABlock, jwtSecret, testEndpoints.GetDAAddress(), genesisHeight, state.LastBlockHeight)
	t.Log("+++ No DA gaps detected ✓")

	// Cleanup processes
	clusterNodes.killAll()
	t.Log("+++ TestHASequencerRollingRestartE2E completed successfully")
}

// verifyNoDoubleSigning checks that no two blocks at the same height have different hashes across nodes
func verifyNoDoubleSigning(t *testing.T, clusterNodes *raftClusterNodes, genesisHeight uint64, lastBlockHeight uint64) {
	t.Helper()
	// Compare block hashes across nodes
	for height := genesisHeight; height <= lastBlockHeight; height++ {
		nodeByHash := make(map[common.Hash][]string, 1)
		headerByNode := make(map[string]*ethtypes.Header)
		for nodeName, node := range clusterNodes.AllNodes() {
			if !node.running.Load() {
				continue
			}
			var header *ethtypes.Header
			require.Eventually(t, func() bool {
				header, _ = node.ethClient(t).HeaderByNumber(t.Context(), big.NewInt(int64(height)))
				return header != nil
			}, 2*time.Second, 100*time.Millisecond, nodeName)
			nodeByHash[header.Hash()] = append(nodeByHash[header.Hash()], nodeName)
			headerByNode[nodeName] = header
		}
		if !assert.Len(t, nodeByHash, 1, "double signing detected at height %d: %v", height, nodeByHash) {
			for hash, nodes := range nodeByHash {
				for _, nodeName := range nodes {
					hdr := headerByNode[nodeName]
					t.Logf("%s (hash=%s): Number=%d Time=%d ParentHash=%s Root=%s TxHash=%s ReceiptHash=%s GasLimit=%d GasUsed=%d BaseFee=%v ExtraData=%x",
						nodeName, hash.Hex(), hdr.Number.Uint64(), hdr.Time, hdr.ParentHash.Hex(), hdr.Root.Hex(),
						hdr.TxHash.Hex(), hdr.ReceiptHash.Hex(), hdr.GasLimit, hdr.GasUsed, hdr.BaseFee, hdr.Extra)
				}
			}
			t.FailNow()
		}
	}
}

// verifyDABlocks checks that DA block heights form a continuous sequence without gaps
func verifyDABlocks(t *testing.T, daStartHeight, lastDABlock uint64, jwtSecret string, daAddress string, genesisHeight, lastEVBlock uint64) {
	t.Helper()
	blobClient, err := blobrpc.NewClient(t.Context(), daAddress, jwtSecret, "")
	require.NoError(t, err)
	defer blobClient.Close()

	ns, err := libshare.NewNamespaceFromBytes(coreda.NamespaceFromString(DefaultDANamespace).Bytes())
	require.NoError(t, err)
	evHeightsToEvBlockParts := make(map[uint64]int)
	deduplicationCache := make(map[string]uint64) // mixed header and data hashes

	// Verify each block is present exactly once
	for daHeight := daStartHeight; daHeight <= lastDABlock; daHeight++ {
		blobs, err := blobClient.Blob.GetAll(t.Context(), daHeight, []libshare.Namespace{ns})
		require.NoError(t, err, "height %d/%d", daHeight, lastDABlock)
		require.NotEmpty(t, blobs)

		for _, blob := range blobs {
			if evHeight, hash, blobType := extractBlockHeight(t, blob.Data()); evHeight != 0 {
				t.Logf("extracting block height from blob (da height %d): %4d (%s)", daHeight, evHeight, blobType)
				if height, ok := deduplicationCache[hash.String()]; ok {
					require.Equal(t, evHeight, height)
					continue
				}
				require.GreaterOrEqual(t, evHeight, genesisHeight)
				deduplicationCache[hash.String()] = evHeight
				evHeightsToEvBlockParts[evHeight]++
			}
		}
	}

	for h := genesisHeight; h <= lastEVBlock; h++ {
		// can be 1 or 2 blobs per block if data is not empty
		require.NotEmpty(t, evHeightsToEvBlockParts[h], "missing block on DA for height %d/%d", h, lastEVBlock)
		require.Less(t, evHeightsToEvBlockParts[h], 3, "duplicate block on DA for height %d/%d", h, lastEVBlock)
	}
}

// extractBlockHeight attempts to decode a blob as SignedHeader or SignedData and extract the block height
func extractBlockHeight(t *testing.T, blob []byte) (uint64, types.Hash, string) {
	t.Helper()
	if len(blob) == 0 {
		t.Log("empty blob, skipping")
		return 0, nil, ""
	}
	var headerPb pb.SignedHeader
	if err := proto.Unmarshal(blob, &headerPb); err == nil {
		var signedHeader types.SignedHeader
		if err := signedHeader.FromProto(&headerPb); err == nil {
			if err := signedHeader.Header.ValidateBasic(); err == nil {
				return signedHeader.Height(), signedHeader.Hash(), "header"
			} else {
				jsonBZ, _ := json.MarshalIndent(signedHeader.Header, "", "  ")
				t.Logf("invalid header: %v: %s", err, string(jsonBZ))
			}
		} else {
			t.Logf("failed to unmarshal signed header: %v", err)
		}
	} else {
		t.Logf("failed to unmarshal blob: %v", err)
	}

	var signedData types.SignedData
	if err := signedData.UnmarshalBinary(blob); err == nil {
		if signedData.Metadata != nil {
			return signedData.Height(), signedData.Hash(), "data"
		}
	} else {
		t.Logf("failed to unmarshal signed data: %v", err)
	}
	return 0, nil, ""
}

func initChain(t *testing.T, sut *SystemUnderTest, workDir string) string {
	passphraseFile := createPassphraseFile(t, workDir)
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--chain_id", DefaultChainID,
		"--rollkit.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", workDir,
	)
	require.NoError(t, err, "failed to init node", output)
	return passphraseFile
}
func setupRaftSequencerNode(
	t *testing.T,
	sut *SystemUnderTest,
	workDir, nodeID, raftAddr, jwtSecret, genesisHash, daAddress, bootstrapDir string,
	allRaftClusterMembers []string,
	p2pPeers, rpcAddr, p2pAddr, engineURL, ethURL string,
	bootstrap bool,
	passphraseFile string,
) *os.Process {
	t.Helper()
	nodeHome := filepath.Join(workDir, nodeID)
	raftDir := filepath.Join(nodeHome, "raft")

	jwtSecretFile := filepath.Join(nodeHome, "jwt-secret.hex")
	if bootstrap {
		initChain(t, sut, nodeHome)
		jwtSecretFile = createJWTSecretFile(t, nodeHome, jwtSecret)

		// Copy genesis and signer files for non-genGenesis nodes
		MustCopyFile(t, filepath.Join(bootstrapDir, "config", "genesis.json"),
			filepath.Join(nodeHome, "config", "genesis.json"))
		MustCopyFile(t, filepath.Join(bootstrapDir, "config", "signer.json"),
			filepath.Join(nodeHome, "config", "signer.json"))
	}
	if strings.HasPrefix(rpcAddr, "http://") {
		rpcAddr = rpcAddr[7:]
	}
	raftPeers := slices.DeleteFunc(slices.Clone(allRaftClusterMembers), func(v string) bool { return strings.Contains(v, nodeID+"@") || strings.TrimSpace(v) == "" })

	// Start node with raft configuration
	process := sut.ExecCmdWithLogPrefix(nodeID, evmSingleBinaryPath,
		"start",
		"--evnode.log.format", "json",
		//"--evnode.log.level", "DEBUG",
		"--home", nodeHome,
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.da.address", daAddress,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--evnode.signer.signer_path", filepath.Join(nodeHome, "config"),
		"--rollkit.da.block_time", (200 * time.Millisecond).String(),
		"--rollkit.da.namespace", DefaultDANamespace,

		"--evnode.raft.enable=true",
		"--evnode.raft.node_id="+nodeID,
		"--evnode.raft.raft_addr="+raftAddr,
		"--evnode.raft.raft_dir="+raftDir,
		"--evnode.raft.bootstrap=true",
		"--evnode.raft.peers="+strings.Join(raftPeers, ","),
		"--evnode.raft.snap_count=10",
		"--evnode.raft.send_timeout=100ms",
		"--evnode.raft.leader_lease_timeout=500ms", // 5x heatbeat interval (we use block time)
		"--evnode.raft.heartbeat_timeout=1000ms",   // 2x lease timeout

		"--rollkit.p2p.peers", p2pPeers,
		"--rollkit.rpc.address", rpcAddr,
		"--rollkit.p2p.listen_address", p2pAddr,
		"--evm.engine-url", engineURL,
		"--evm.eth-url", ethURL,
	)
	time.Sleep(SlowPollingInterval)

	//sut.AwaitNodeUp(t, "http://"+rpcAddr, NodeStartupTimeout)
	return process
}

// submitTxToURL submits a tx to the specified EVM endpoint and waits for inclusion.
func submitTxToURL(t *testing.T, client *ethclient.Client) (common.Hash, uint64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	priv, err := crypto.HexToECDSA(TestPrivateKey)
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(priv.PublicKey)

	nonce, err := client.PendingNonceAt(ctx, from)
	require.NoError(t, err)
	ln := nonce

	tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &ln)
	require.NoError(t, client.SendTransaction(ctx, tx))

	var blk uint64
	require.Eventually(t, func() bool {
		rec, err := client.TransactionReceipt(t.Context(), tx.Hash())
		if err == nil && rec != nil && rec.Status == 1 {
			blk = rec.BlockNumber.Uint64()
			return true
		}
		return false
	}, 20*time.Second, SlowPollingInterval)

	return tx.Hash(), blk
}

const defaultMaxBlobSize = 2 * 1024 * 1024 // 2MB

func queryLastDAHeight(t *testing.T, startHeight uint64, jwtSecret string, daAddress string) uint64 {
	t.Helper()
	blobClient, err := blobrpc.NewClient(t.Context(), daAddress, jwtSecret, "")
	require.NoError(t, err)
	defer blobClient.Close()
	ns, err := libshare.NewNamespaceFromBytes(coreda.NamespaceFromString(DefaultDANamespace).Bytes())
	require.NoError(t, err)
	var lastDABlock = startHeight
	for {
		blobs, err := blobClient.Blob.GetAll(t.Context(), lastDABlock, []libshare.Namespace{ns})
		if err != nil {
			if strings.Contains(err.Error(), "future") {
				return lastDABlock - 1
			}
			t.Fatal("failed to get blobs:", err)
		}
		if len(blobs) != 0 && testing.Verbose() {
			t.Log("+++ DA block: ", lastDABlock, " blobs: ", len(blobs))
		}
		lastDABlock++
	}
}

type nodeDetails struct {
	raftAddr string
	rpcAddr  string
	process  *os.Process
	ethAddr  string

	extClientOnce sync.Once
	xEthClient    atomic.Pointer[ethclient.Client]
	xRPCClient    atomic.Pointer[rpcclient.Client]
	running       atomic.Bool
	p2pAddr       string
	engineURL     string
	ethURL        string
}

func (d *nodeDetails) ethClient(t *testing.T) *ethclient.Client {
	t.Helper()
	d.initExtClients(t)
	return d.xEthClient.Load()
}

func (d *nodeDetails) rpcClient(t *testing.T) *rpcclient.Client {
	t.Helper()
	d.initExtClients(t)
	return d.xRPCClient.Load()

}

func (d *nodeDetails) initExtClients(t *testing.T) {
	require.NotNil(t, d)
	d.extClientOnce.Do(func() {
		client, err := ethclient.Dial(d.ethAddr)
		require.NoError(t, err)
		d.xEthClient.Store(client)
		t.Cleanup(client.Close)
		rpcClient := rpcclient.NewClient(d.rpcAddr)
		require.NotNil(t, rpcClient)
		d.xRPCClient.Store(rpcClient)
	})
}

func (d *nodeDetails) IsRunning() bool {
	return d.running.Load()
}

func (d *nodeDetails) Kill() (err error) {
	err = d.process.Kill()
	d.running.Store(false)
	return
}

// GracefulStop sends SIGTERM to the process for a graceful shutdown
func (d *nodeDetails) GracefulStop() (err error) {
	err = d.process.Signal(syscall.SIGTERM)
	d.running.Store(false)
	return
}

type raftClusterNodes struct {
	mx    sync.Mutex
	nodes map[string]*nodeDetails
}

func (c *raftClusterNodes) Set(node string, listen string, proc *os.Process, eth string, raftAddr string, p2pAddr string, engineURL string, ethURL string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	d := &nodeDetails{raftAddr: raftAddr, rpcAddr: listen, process: proc, ethAddr: eth, p2pAddr: p2pAddr, engineURL: engineURL, ethURL: ethURL}
	d.running.Store(true)
	c.nodes[node] = d
}

func (c *raftClusterNodes) Leader(t require.TestingT) string {
	node, _ := leader(t, c.AllNodes())
	return node
}

func (c *raftClusterNodes) Details(node string) *nodeDetails {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.nodes[node]
}

func (c *raftClusterNodes) Followers(t require.TestingT) map[string]*nodeDetails {
	all := c.AllNodes()
	leader, _ := leader(t, all)
	delete(all, leader)
	return all
}

func (c *raftClusterNodes) killAll() {
	for _, d := range c.AllNodes() {
		_ = d.Kill()
	}
}

// allNodes returns snapshot of nodes map
func (c *raftClusterNodes) AllNodes() map[string]*nodeDetails {
	c.mx.Lock()
	defer c.mx.Unlock()
	return maps.Clone(c.nodes)
}

// fails when no leader is found
func leader(t require.TestingT, nodes map[string]*nodeDetails) (string, *nodeDetails) {
	client := &http.Client{Timeout: 1 * time.Second}
	type nodeStatus struct {
		IsLeader bool `json:"is_leader"`
	}
	for node, details := range nodes {
		if !details.running.Load() {
			continue
		}
		resp, err := client.Get(details.rpcAddr + "/raft/node")
		require.NoError(t, err)
		defer resp.Body.Close()

		var status nodeStatus
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))

		if status.IsLeader {
			return node, details
		}
	}

	t.Errorf("no leader found")
	return "", nil
}

func must[T any](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}

// IsNodeUp waits until a node is operational by validating it produces blocks.
// Returns true if node is up within the specified timeout.
// Unlike AwaitNodeUp, this method Does not fail tests.
func IsNodeUp(t *testing.T, rpcAddr string, timeout time.Duration) bool {
	t.Helper()
	t.Logf("Query node is up: %s", rpcAddr)
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	ticker := time.Tick(min(timeout/10, 200*time.Millisecond))
	c := client.NewClient(rpcAddr)
	require.NotNil(t, c)
	var lastBlock uint64
	for {
		select {
		case <-ticker:
			switch s, err := c.GetState(ctx); {
			case err != nil: // ignore
			case lastBlock == 0:
				lastBlock = s.LastBlockHeight
			case lastBlock < s.LastBlockHeight:
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}
