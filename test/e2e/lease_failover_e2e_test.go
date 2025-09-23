//go:build e2e

package e2e

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/lease"
	"github.com/stretchr/testify/require"
)

// TestLeaseFailoverE2E runs two node binaries configured to use an HTTP lease backend.
// It forces a lease release on the backend and verifies leadership failover occurs.
func TestLeaseFailoverE2E(t *testing.T) {
	flag.Parse()

	sut := NewSystemUnderTest(t)

	// Start local DA to keep nodes healthy
	localDABinary := filepath.Join(filepath.Dir(binaryPath), "local-da")
	sut.ExecCmd(localDABinary)
	// give DA a moment to initialize
	time.Sleep(500 * time.Millisecond)

	leaseName := "e2e-lease-" + t.Name()
	leaseTerm := 300 * time.Millisecond

	// Start in-process HTTP lease backend used by both nodes
	baseURL, shutdown := startLeaseHTTPTestServer(leaseName)
	t.Cleanup(shutdown)

	workDir := "./testnet"
	//workDir := t.TempDir()
	node1Home := filepath.Join(workDir, "node1")
	node2Home := filepath.Join(workDir, "node2")

	// Get JWT secrets and setup common components first
	jwtSecret, fullNodeJwtSecret, genesisHash := setupCommonEVMTest(t, sut, true)
	_ = fullNodeJwtSecret
	leaderElectionConfig := config.LeaderElectionConfig{
		Enabled: true,
		Backend: "http",
		LeaseTerm: config.DurationWrapper{
			Duration: leaseTerm,
		},
		LeaseName:   leaseName,
		BackendAddr: baseURL,
	}
	leaderProcess := setupLeaderSequencerNode(t, sut, node1Home, jwtSecret, genesisHash, nil, leaderElectionConfig)
	t.Log("Sequencer1 node is up")

	// Get P2P address and setup full node
	sequencer1P2PAddress := getNodeP2PAddress(t, sut, node1Home)
	t.Logf("Sequencer1 P2P address: %s", sequencer1P2PAddress)

	setupFailoverAggregator(
		t,
		sut,
		node2Home,
		node1Home,
		fullNodeJwtSecret,
		genesisHash,
		sequencer1P2PAddress,
		nil,
		leaderElectionConfig,
	)
	t.Log("Sequencer2 node is up")

	sequencer2P2PAddress := getNodeP2PAddress(t, sut, node1Home)
	t.Logf("Sequencer2 P2P address: %s", sequencer2P2PAddress)

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

	t.Log("+++ killing Current leader: " + firstLeader)
	_ = leaderProcess.Kill()

	// Expect a different node to take leadership
	require.Eventually(t, func() bool {
		ldr, err := httpLease.GetHolder(ctx)
		t.Logf("+++ leader: %s\n", ldr)
		if err != nil {
			return false
		}
		return ldr != "" && ldr != firstLeader
	}, 10*time.Second, 100*time.Millisecond, "no failover occurred")
}

func setupLeaderSequencerNode(t *testing.T, sut *SystemUnderTest, sequencerHome, jwtSecret, genesisHash string, ports *TestPorts, electionConfig config.LeaderElectionConfig, ) *os.Process {
	t.Helper()

	// Initialize sequencer node
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
	)
	require.NoError(t, err, "failed to init sequencer", output)

	if ports != nil {
		t.Fatal("not implemented")
		return nil
	}
	// Fallback to default ports if none provided
	process := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evm.jwt-secret", jwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--home", sequencerHome,
		"--rollkit.da.address", DAAddress,
		"--rollkit.da.block_time", DefaultDABlockTime,

		"--rollkit.leader.enabled="+strconv.FormatBool(electionConfig.Enabled),
		"--rollkit.leader.backend="+electionConfig.Backend,
		"--rollkit.leader.lease_term="+electionConfig.LeaseTerm.String(),
		"--rollkit.leader.lease_name="+electionConfig.LeaseName,
		"--rollkit.leader.backend_addr="+electionConfig.BackendAddr,
	)
	sut.AwaitNodeUp(t, RollkitRPCAddress, NodeStartupTimeout)
	return process
}

func setupFailoverAggregator(
	t *testing.T,
	sut *SystemUnderTest,
	fullNodeHome, sequencerHome, fullNodeJwtSecret, genesisHash, sequencerP2PAddress string,
	ports *TestPorts,
	electionConfig config.LeaderElectionConfig,
) *os.Process {
	t.Helper()

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

	if ports != nil {
		sut.t.Fatal("not implemented")
		return nil
	}
	// Fallback to default ports if none provided
	process := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--home", fullNodeHome,
		"--evm.jwt-secret", fullNodeJwtSecret,
		"--evm.genesis-hash", genesisHash,
		"--rollkit.node.block_time", DefaultBlockTime,
		"--rollkit.node.aggregator=true",
		"--rollkit.signer.passphrase", TestPassphrase,
		"--evnode.signer.signer_path", filepath.Join(fullNodeHome, "config"),
		"--rollkit.da.address", DAAddress,
		"--rollkit.da.block_time", DefaultDABlockTime,

		"--rollkit.leader.enabled="+strconv.FormatBool(electionConfig.Enabled),
		"--rollkit.leader.backend="+electionConfig.Backend,
		"--rollkit.leader.lease_term="+electionConfig.LeaseTerm.String(),
		"--rollkit.leader.lease_name="+electionConfig.LeaseName,
		"--rollkit.leader.backend_addr="+electionConfig.BackendAddr,

		"--rollkit.rpc.address", "127.0.0.1:"+FullNodeRPCPort,
		"--rollkit.p2p.listen_address", "/ip4/127.0.0.1/tcp/"+FullNodeP2PPort,
		"--rollkit.p2p.peers", sequencerP2PAddress,
		"--evm.engine-url", FullNodeEngineURL,
		"--evm.eth-url", FullNodeEthURL,
	)
	sut.AwaitNodeUp(t, "http://127.0.0.1:"+FullNodeRPCPort, NodeStartupTimeout)
	return process
}
