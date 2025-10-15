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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	evmtest "github.com/evstack/ev-node/execution/evm/test"
	rpcclient "github.com/evstack/ev-node/pkg/rpc/client"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/evstack/ev-node/execution/evm"
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

	//workDir := t.TempDir()
	workDir := "/Users/alex/workspace/rollkit/rollkit/test/e2e/testnet"

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
	initChain(t, sut, bootstrapDir)

	clusterNodes := &raftClusterNodes{
		nodes: make(map[string]*nodeDetails),
	}

	// Start node1 (bootstrap node)
	go func() {
		proc := setupRaftSequencerNode(t, sut, workDir, "node1", node1RaftAddr, jwtSecret, genesisHash, testEndpoints.GetDAAddress(),
			bootstrapDir, raftCluster, "", testEndpoints.GetRollkitRPCListen(), testEndpoints.GetRollkitP2PAddress(),
			testEndpoints.GetSequencerEngineURL(), testEndpoints.GetSequencerEthURL(), true)
		t.Log("Node1 is up")
		clusterNodes.Set("node1", testEndpoints.GetRollkitRPCAddress(), proc, testEndpoints.GetSequencerEthURL())
	}()
	node1P2PAddr := ""

	// Start node2 (follower)
	go func() {
		t.Log("Starting Node2")
		proc := setupRaftSequencerNode(t, sut, workDir, "node2", node2RaftAddr, fullNodeJwtSecret, genesisHash, testEndpoints.GetDAAddress(),
			bootstrapDir, raftCluster, node1P2PAddr, testEndpoints.GetFullNodeRPCListen(), testEndpoints.GetFullNodeP2PAddress(),
			testEndpoints.GetFullNodeEngineURL(), testEndpoints.GetFullNodeEthURL(), true)
		t.Log("Node2 is up")

		clusterNodes.Set("node2", testEndpoints.GetFullNodeRPCAddress(), proc, testEndpoints.GetFullNodeEthURL())
	}()

	// Start node3 (follower)
	node3EthAddr := fmt.Sprintf("http://127.0.0.1:%s", fullNode3EthPort)
	go func() {
		t.Log("Starting Node3")
		p2pPeerList := ""
		node3RPCListen := fmt.Sprintf("127.0.0.1:%d", mustGetAvailablePort(t))
		node3P2PAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", mustGetAvailablePort(t))
		proc := setupRaftSequencerNode(t, sut, workDir, "node3", node3RaftAddr, jwtSecret3, genesisHash, testEndpoints.GetDAAddress(),
			bootstrapDir, raftCluster, p2pPeerList,
			node3RPCListen, node3P2PAddr,
			fmt.Sprintf("http://127.0.0.1:%s", fullNode3EnginePort), node3EthAddr, true)
		t.Log("Node3 is up")
		clusterNodes.Set("node3", "http://"+node3RPCListen, proc, node3EthAddr)
	}()
	sut.AwaitNodeUp(t, testEndpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
	// Wait for at least 2 blocks to be produced
	sut.AwaitNBlocks(t, 2, testEndpoints.GetRollkitRPCAddress(), 6*time.Second)

	// Submit a tx and ensure it propagates to all nodes
	leaderNode := clusterNodes.Leader(t)
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
	time.Sleep(1 * time.Second)
	lastDABlockOldLeader := queryLastDAHeight(t, daStartHeight, jwtSecret, testEndpoints.GetDAAddress())
	t.Log("+++ Last DA block by old leader: ", lastDABlockOldLeader)
	leaderElectionStart := time.Now()

	// Wait for new leader election - submit tx to node2
	var newLeader string
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		newLeader = clusterNodes.Leader(collect)
	}, 5*time.Second, 200*time.Millisecond)
	leaderFailoverTime := time.Since(leaderElectionStart)
	t.Logf("+++ New leader: %s within %s\n", newLeader, leaderFailoverTime)
	require.NotEqual(t, oldLeader, newLeader)

	_, blk2 := submitTxToURL(t, clusterNodes.Details(newLeader).ethClient(t))
	require.Greater(t, blk2, blk1, "post-failover block should advance")

	// Verify DA progress
	var lastDABlockNewLeader uint64
	require.Eventually(t, func() bool {
		lastDABlockNewLeader = queryLastDAHeight(t, lastDABlockOldLeader, jwtSecret, testEndpoints.GetDAAddress())
		return lastDABlockNewLeader > lastDABlockOldLeader
	}, 2*must(time.ParseDuration(DefaultDABlockTime)), 100*time.Millisecond)
	t.Logf("+++ Last DA block by new leader: %d\n", lastDABlockNewLeader)

	// Restart node1 to rejoin cluster
	// todo: re-using peers here and in the server impl is a quick hack. must not be the same config later!
	raftClusterRPCs := []string{clusterNodes.Details("node2").rpcAddr, clusterNodes.Details("node3").rpcAddr}
	node1Process := setupRaftSequencerNode(t, sut, workDir, "node1", node1RaftAddr, jwtSecret, genesisHash, testEndpoints.GetDAAddress(),
		"", raftClusterRPCs, "",
		testEndpoints.GetRollkitRPCListen(), testEndpoints.GetRollkitP2PAddress(),
		testEndpoints.GetSequencerEngineURL(), testEndpoints.GetSequencerEthURL(), false)
	t.Log("Restarted node1 to sync with cluster")
	clusterNodes.Set("node1", testEndpoints.GetRollkitRPCAddress(), node1Process, testEndpoints.GetSequencerEthURL())

	// Wait for several blocks to be produced and propagated
	sut.AwaitNBlocks(t, 5, testEndpoints.GetRollkitRPCAddress(), 10*time.Second)

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
	verifyBlocks(t, daStartHeight, lastDABlockNewLeader, jwtSecret, testEndpoints.GetDAAddress(), genesisHeight, state.LastBlockHeight)

	// Cleanup processes
	clusterNodes.killAll()
	t.Logf("Completed leader change in: %s", leaderFailoverTime)
}

// verifyNoDoubleSigning checks that no two blocks at the same height have different hashes across nodes
func verifyNoDoubleSigning(t *testing.T, clusterNodes *raftClusterNodes, genesisHeight uint64, lastBlockHeight uint64) {
	t.Helper()
	// Compare block hashes across nodes
	for height := genesisHeight; height <= lastBlockHeight; height++ {
		nodeByHash := make(map[common.Hash][]string, 1)
		for nodeName, node := range clusterNodes.allNodes() {
			if !node.running.Load() {
				continue
			}
			var header *ethtypes.Header
			require.Eventually(t, func() bool {
				header, _ = node.ethClient(t).HeaderByNumber(t.Context(), big.NewInt(int64(height)))
				return header != nil
			}, 2*time.Second, 100*time.Millisecond)
			nodeByHash[header.Hash()] = append(nodeByHash[header.Hash()], nodeName)
		}
		if !assert.Len(t, nodeByHash, 1, "double signing detected at height %d: %v", height, nodeByHash) {
			for _, nodes := range nodeByHash {
				rsp, err := clusterNodes.Details(nodes[0]).rpcClient(t).GetBlockByHeight(t.Context(), height)
				require.NoError(t, err)
				t.Logf("%s: %v", nodes[0], rsp.Block)
			}
			//t.FailNow()
		}
	}
}

// verifyBlocks checks that DA block heights form a continuous sequence without gaps
func verifyBlocks(t *testing.T, daStartHeight, lastDABlock uint64, jwtSecret string, daAddress string, genesisHeight, lastEVBlock uint64) {
	t.Helper()
	client, err := jsonrpc.NewClient(t.Context(), zerolog.Nop(), daAddress, jwtSecret, 0, 1, 0)
	require.NoError(t, err)
	defer client.Close()

	namespace := coreda.NamespaceFromString(DefaultDANamespace).Bytes()
	evHeightsToEvBlockParts := make(map[uint64]int)
	deduplicationCache := make(map[string]uint64) // mixed header and data hashes

	lastEVHeight := genesisHeight
	// Verify each block is present exactly once
	for daHeight := daStartHeight; daHeight <= lastDABlock; daHeight++ {
		res, err := client.DA.GetIDs(t.Context(), daHeight, namespace)
		require.NoError(t, err, "height %d/%d", daHeight, lastDABlock)
		require.NotEmpty(t, res.IDs)

		blobs, err := client.DA.Get(t.Context(), res.IDs, namespace)
		require.NoError(t, err)

		for _, blob := range blobs {
			if evHeight, hash := extractBlockHeight(t, blob); evHeight != 0 {
				t.Logf("extracting block height from blob (da height %d): %x", daHeight, evHeight)
				if height, ok := deduplicationCache[hash.String()]; ok {
					require.Equal(t, evHeight, height)
					continue
				}
				_ = lastEVHeight
				//require.GreaterOrEqual(t, evHeight, lastEVHeight)
				lastEVHeight = evHeight
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
func extractBlockHeight(t *testing.T, blob []byte) (uint64, types.Hash) {
	t.Helper()
	var headerPb pb.SignedHeader
	if err := proto.Unmarshal(blob, &headerPb); err == nil {
		var signedHeader types.SignedHeader
		if err := signedHeader.FromProto(&headerPb); err == nil {
			if err := signedHeader.Header.ValidateBasic(); err == nil {
				return signedHeader.Height(), signedHeader.Hash()
			} else {
				t.Logf("invalid header: %v", err)
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
			return signedData.Height(), signedData.Hash()
		}
	} else {
		t.Logf("failed to unmarshal signed data: %v", err)
	}
	return 0, nil
}

func initChain(t *testing.T, sut *SystemUnderTest, workDir string) {
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--chain_id", DefaultChainID,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", workDir,
	)
	require.NoError(t, err, "failed to init node", output)
}
func setupRaftSequencerNode(
	t *testing.T,
	sut *SystemUnderTest,
	workDir, nodeID, raftAddr, jwtSecret, genesisHash, daAddress string,
	bootstrapDir string,
	raftPeers []string,
	p2pPeers, rpcAddr, p2pAddr, engineURL, ethURL string,
	bootstrap bool,
) *os.Process {
	t.Helper()
	nodeHome := filepath.Join(workDir, nodeID)
	raftDir := filepath.Join(nodeHome, "raft")

	if bootstrap {
		initChain(t, sut, nodeHome)

		// Copy genesis and signer files for non-genGenesis nodes
		MustCopyFile(t, filepath.Join(bootstrapDir, "config", "genesis.json"),
			filepath.Join(nodeHome, "config", "genesis.json"))
		MustCopyFile(t, filepath.Join(bootstrapDir, "config", "signer.json"),
			filepath.Join(nodeHome, "config", "signer.json"))
	}

	// Start node with raft configuration
	process := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evnode.log.format", "json",
		//"--evnode.log.level", "DEBUG",
		"--home", nodeHome,
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.da.address", daAddress,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--evnode.signer.signer_path", filepath.Join(nodeHome, "config"),
		"--rollkit.da.block_time", (200 * time.Millisecond).String(),
		"--rollkit.da.namespace", DefaultDANamespace,

		"--evnode.raft.enable=true",
		"--evnode.raft.node_id="+nodeID,
		"--evnode.raft.raft_addr="+raftAddr,
		"--evnode.raft.raft_dir="+raftDir,
		"--evnode.raft.bootstrap="+strconv.FormatBool(bootstrap),
		"--evnode.raft.peers="+strings.Join(raftPeers, ","),
		"--evnode.raft.snap_count=10",
		"--evnode.raft.send_timeout=300ms",
		"--evnode.raft.heartbeat_timeout=300ms",

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

func queryLastDAHeight(t *testing.T, startHeight uint64, jwtSecret string, daAddress string) uint64 {
	t.Helper()
	logger := zerolog.Nop()
	if testing.Verbose() {
		logger = zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.DebugLevel)
	}
	client, err := jsonrpc.NewClient(t.Context(), logger, daAddress, jwtSecret, 0, 1, 0)
	require.NoError(t, err)
	defer client.Close()
	var lastDABlock = startHeight
	for {
		res, err := client.DA.GetIDs(t.Context(), lastDABlock, coreda.NamespaceFromString(DefaultDANamespace).Bytes())
		if err != nil {
			if strings.Contains(err.Error(), "future") {
				return lastDABlock - 1
			}
			t.Fatal("failed to get IDs:", err)
		}
		if len(res.IDs) != 0 && testing.Verbose() {
			t.Log("+++ DA block: ", lastDABlock, " ids: ", len(res.IDs))
		}
		lastDABlock++
	}
}

type nodeDetails struct {
	rpcAddr string
	process *os.Process
	ethAddr string

	extClientOnce sync.Once
	xEthClient    atomic.Pointer[ethclient.Client]
	xRPCClient    atomic.Pointer[rpcclient.Client]
	running       atomic.Bool
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

func (d *nodeDetails) Kill() (err error) {
	err = d.process.Kill()
	d.running.Store(false)
	return
}

type raftClusterNodes struct {
	mx    sync.Mutex
	nodes map[string]*nodeDetails
}

func (c *raftClusterNodes) Set(node string, listen string, proc *os.Process, eth string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	d := &nodeDetails{rpcAddr: listen, process: proc, ethAddr: eth}
	d.running.Store(true)
	c.nodes[node] = d
}

func (c *raftClusterNodes) Leader(t require.TestingT) string {
	node, _ := leader(t, c.allNodes())
	return node
}

func (c *raftClusterNodes) Details(node string) *nodeDetails {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.nodes[node]
}

func (c *raftClusterNodes) Followers(t require.TestingT) map[string]*nodeDetails {
	all := c.allNodes()
	leader, _ := leader(t, all)
	delete(all, leader)
	return all
}

func (c *raftClusterNodes) killAll() {
	for _, d := range c.allNodes() {
		_ = d.Kill()
	}
}

// allNodes returns snapshot of nodes map
func (c *raftClusterNodes) allNodes() map[string]*nodeDetails {
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
