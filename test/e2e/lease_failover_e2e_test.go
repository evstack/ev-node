//go:build e2e

package e2e

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/lease"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
)

// TestLeaseFailoverE2E runs two node binaries configured to use an HTTP lease backend.
// It forces a lease release on the backend and verifies leadership failover occurs.
func TestLeaseFailoverE2E(t *testing.T) {
	flag.Parse()
	sut := NewSystemUnderTest(t)
	if testing.Verbose() {
		os.Setenv("GOLOG_LOG_LEVEL", "DEBUG")
		t.Cleanup(func() {
			os.Unsetenv("GOLOG_LOG_LEVEL")
		})
	}

	leaseName := "e2e-lease-" + t.Name()
	leaseTerm := 300 * time.Millisecond

	// Start in-process HTTP lease backend used by both nodes
	baseURL, shutdown := startLeaseHTTPTestServer(leaseName)
	t.Cleanup(shutdown)

	workDir := t.TempDir()
	node1Home := filepath.Join(workDir, "node1")
	node2Home := filepath.Join(workDir, "node2")

	// Get JWT secrets and setup common components first
	jwtSecret, fullNodeJwtSecret, genesisHash, testEndpoints := setupCommonEVMTest(t, sut, true)
	leaderElectionConfig := config.LeaderElectionConfig{
		Enabled: true,
		Backend: "http",
		LeaseTerm: config.DurationWrapper{
			Duration: leaseTerm,
		},
		LeaseName:   leaseName,
		BackendAddr: baseURL,
	}
	leaderProcess := setupLeaderSequencerNode(t, sut, node1Home, jwtSecret, genesisHash, testEndpoints, leaderElectionConfig, true, "")
	t.Log("Leading sequencer node is up")

	leaderP2PAddr := getNodeP2PAddress(t, sut, node1Home, testEndpoints.RollkitRPCPort)
	t.Log("Leading sequencer P2P address: ", leaderP2PAddr)

	setupFailoverSequencerFullNode(t, sut, node2Home, node1Home, fullNodeJwtSecret, genesisHash, testEndpoints, leaderElectionConfig, leaderP2PAddr)
	t.Log("Follower sequencer node is up")

	followerP2PAddr := getNodeP2PAddress(t, sut, node2Home, testEndpoints.FullNodeRPCPort)
	t.Log("Follower sequencer P2P address: ", followerP2PAddr)

	// Wait for at least 2 blocks to be produced before starting header detectors
	sut.AwaitNBlocks(t, 2, testEndpoints.GetRollkitRPCAddress(), 6*time.Second)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)

	// Wait until one node acquires leadership
	httpLease := lease.NewHTTPLease(baseURL, leaseName)
	var firstLeader string
	require.Eventually(t, func() bool {
		t.Log("+++ query lease holder")
		ldr, err := httpLease.GetHolder(ctx)
		if err != nil {
			t.Logf("+++ error getting leader: %v\n", err)
			return false
		}
		firstLeader = ldr
		return firstLeader != ""
	}, 10*time.Second, 100*time.Millisecond, "no leader elected")

	// Connect to EVM endpoints for both instances
	seqClient, err := ethclient.Dial(testEndpoints.GetSequencerEthURL())
	require.NoError(t, err, "connect sequencer evm")
	defer seqClient.Close()
	fnClient, err := ethclient.Dial(testEndpoints.GetFullNodeEthURL())
	require.NoError(t, err, "connect follower evm")
	defer fnClient.Close()

	// Submit a tx to the current leader (sequencer) and ensure it propagates
	txHash1, blk1 := submitTransactionAndGetBlockNumber(t, seqClient)
	require.Eventually(t, func() bool {
		rec, err := fnClient.TransactionReceipt(t.Context(), txHash1)
		return err == nil && rec != nil && rec.Status == 1 && rec.BlockNumber.Uint64() == blk1
	}, 20*time.Second, SlowPollingInterval, "tx1 not seen on follower")

	t.Log("+++ killing Current leader: " + firstLeader)
	_ = leaderProcess.Kill()

	lastDABlockOldLeader := queryLastDAHeight(t, 1, jwtSecret, testEndpoints.GetDAAddress())
	t.Log("+++ Last DA block of old leader: ", lastDABlockOldLeader)
	// Expect a different node to take leadership
	require.Eventually(t, func() bool {
		ldr, err := httpLease.GetHolder(ctx)
		if err != nil {
			return false
		}
		return ldr != "" && ldr != firstLeader
	}, 5*time.Second, 100*time.Millisecond, "no failover occurred")

	// After failover, submit a tx to the remaining instance and ensure inclusion
	_, blk2 := submitTxToURL(t, fnClient)
	// Ensure chain progressed across failover
	require.Greater(t, blk2, blk1, "post-failover block should advance")

	require.Eventually(t, func() bool {
		lastDABlockNewLeader := queryLastDAHeight(t, lastDABlockOldLeader, jwtSecret, testEndpoints.GetDAAddress())
		return lastDABlockNewLeader > lastDABlockOldLeader
	}, 2*must(time.ParseDuration(DefaultDABlockTime)), 100*time.Millisecond)

	leaderProcess = setupLeaderSequencerNode(t, sut, node1Home, jwtSecret, genesisHash, testEndpoints, leaderElectionConfig, false, getNodeP2PAddress(t, sut, node1Home, testEndpoints.FullNodeRPCPort))
	t.Log("Restart node1 node to sync with new leader")

	// Wait for several blocks to be produced and propagated through P2P
	sut.AwaitNBlocks(t, 5, testEndpoints.GetRollkitRPCAddress(), 10*time.Second)
}

func setupLeaderSequencerNode(
	t *testing.T,
	sut *SystemUnderTest,
	sequencerHome, jwtSecret, genesisHash string,
	endpoints *TestEndpoints,
	electionConfig config.LeaderElectionConfig,
	runInitBefore bool,
	p2pPeerAddr string,
) *os.Process {
	t.Helper()
	require.NotEmpty(t, endpoints)

	if runInitBefore {
		// Initialize sequencer node
		output, err := sut.RunCmd(evmSingleBinaryPath,
			"init",
			"--chain_id", DefaultChainID,
			"--rollkit.node.aggregator=true",
			"--rollkit.signer.passphrase", TestPassphrase,
			"--home", sequencerHome,
		)
		require.NoError(t, err, "failed to init sequencer", output)
	}
	// Fallback to default ports if none provided
	process := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evnode.log.format", "json",
		"--home", sequencerHome,
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.da.address", endpoints.GetDAAddress(),
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--rollkit.da.block_time", (200 * time.Millisecond).String(),
		"--rollkit.da.namespace", DefaultDANamespace,

		"--rollkit.leader.enabled="+strconv.FormatBool(electionConfig.Enabled),
		"--rollkit.leader.backend="+electionConfig.Backend,
		"--rollkit.leader.lease_term="+electionConfig.LeaseTerm.String(),
		"--rollkit.leader.lease_name="+electionConfig.LeaseName,
		"--rollkit.leader.backend_addr="+electionConfig.BackendAddr,
		"--rollkit.p2p.peers", p2pPeerAddr,

		"--rollkit.rpc.address", endpoints.GetRollkitRPCListen(),
		"--rollkit.p2p.listen_address", endpoints.GetRollkitP2PAddress(),
		"--evm.engine-url", endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", endpoints.GetSequencerEthURL(),
	)
	time.Sleep(SlowPollingInterval)

	sut.AwaitNodeUp(t, endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)
	return process
}

func setupFailoverSequencerFullNode(
	t *testing.T,
	sut *SystemUnderTest,
	fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash string,
	endpoints *TestEndpoints,
	electionConfig config.LeaderElectionConfig,
	peerP2PAddr string,
) *os.Process {
	t.Helper()
	require.NotEmpty(t, endpoints)

	// Initialize full node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--home", fullNodeHome,
	)
	require.NoError(t, err, "failed to init full node", output)

	// Copy genesis file from sequencer to full node
	MustCopyFile(t, filepath.Join(sequencerHome, "config", "genesis.json"),
		filepath.Join(fullNodeHome, "config", "genesis.json"))
	MustCopyFile(t, filepath.Join(sequencerHome, "config", "signer.json"),
		filepath.Join(fullNodeHome, "config", "signer.json"))

	// Fallback to default ports if none provided
	process := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evnode.log.format", "json",
		"--evnode.log.level", "debug",
		"--home", fullNodeHome,
		"--evm.jwt-secret", fullNodeJwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--evnode.signer.signer_path", filepath.Join(fullNodeHome, "config"),
		"--rollkit.da.address", endpoints.GetDAAddress(),
		"--rollkit.da.block_time", (200 * time.Millisecond).String(),
		"--rollkit.da.namespace", DefaultDANamespace,

		"--rollkit.leader.enabled="+strconv.FormatBool(electionConfig.Enabled),
		"--rollkit.leader.backend="+electionConfig.Backend,
		"--rollkit.leader.lease_term="+electionConfig.LeaseTerm.String(),
		"--rollkit.leader.lease_name="+electionConfig.LeaseName,
		"--rollkit.leader.backend_addr="+electionConfig.BackendAddr,

		//"--rollkit.p2p.peers", endpoints.GetRollkitP2PAddress(),
		"--rollkit.p2p.peers", peerP2PAddr,
		"--rollkit.rpc.address", endpoints.GetFullNodeRPCListen(),
		"--rollkit.p2p.listen_address", endpoints.GetFullNodeP2PAddress(),
		"--evm.engine-url", endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", endpoints.GetFullNodeEthURL(),
	)
	time.Sleep(SlowPollingInterval)

	sut.AwaitNodeUp(t, endpoints.GetFullNodeRPCAddress(), NodeStartupTimeout)
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
	client, err := jsonrpc.NewClient(t.Context(), zerolog.New(zerolog.NewTestWriter(t)), daAddress, jwtSecret, 0, 1)
	require.NoError(t, err)
	defer client.Close()
	var lastDABlock = startHeight
	for {
		res, err := client.DA.GetIDs(t.Context(), lastDABlock, coreda.NamespaceFromString(DefaultDANamespace).Bytes())
		if err != nil {
			if strings.Contains(err.Error(), "future") {
				break
			}
			t.Fatal("failed to get IDs:", err)
		}
		if len(res.IDs) != 0 {
			t.Log("+++ DA block: ", lastDABlock, " ids: ", len(res.IDs))
		}
		lastDABlock++
	}
	return lastDABlock
}

func must[T any](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}
