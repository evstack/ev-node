//go:build docker_e2e && evm

package docker_e2e

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/evstack/evmsingle"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/evstack/ev-node/execution/evm"
	"github.com/stretchr/testify/suite"
)

const (
	// TODO: upgrade from previous released version instead of main
	baseEVMSingleVersion = "main"
	evmChainID           = "1234"
	testPrivateKey       = "cece4f25ac74deb1468965160c7185e07dff413f23fcadb611b05ca37ab0a52e"
	testToAddress        = "0x944fDcD1c868E3cC566C78023CcB38A32cDA836E"
	testGasLimit         = uint64(22000)
)

// EVMSingleUpgradeTestSuite embeds DockerTestSuite to reuse infrastructure setup.
type EVMSingleUpgradeTestSuite struct {
	DockerTestSuite
	rethNode          *reth.Node
	evmSingleChain    *evmsingle.Chain
	evmSingleNode     *evmsingle.Node
	ethClient         *ethclient.Client
	daAddress         string
	evmEngineURL      string
	evmEthURL         string
	evmEthURLExternal string
	jwtSecret         string
	rethGenesisHash   string
	txNonce           uint64
}

func TestEVMSingleUpgradeSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping due to short mode")
	}
	suite.Run(t, new(EVMSingleUpgradeTestSuite))
}

// TestEVMSingleUpgrade tests upgrading an evm node from a stable release
// to a PR-built image while preserving state.
func (s *EVMSingleUpgradeTestSuite) TestEVMSingleUpgrade() {
	ctx := context.Background()

	s.setupDockerEnvironment()

	s.Run("setup_celestia_and_DA_bridge", func() {
		s.celestia = s.CreateChain()
		s.Require().NoError(s.celestia.Start(ctx))
		s.T().Log("Celestia chain started")

		s.daNetwork = s.CreateDANetwork()
		bridgeNode := s.daNetwork.GetBridgeNodes()[0]

		chainID := s.celestia.GetChainID()
		genesisHash := s.getGenesisHash(ctx)
		networkInfo, err := s.celestia.GetNodes()[0].GetNetworkInfo(ctx)
		s.Require().NoError(err)

		s.StartBridgeNode(ctx, bridgeNode, chainID, genesisHash, networkInfo.Internal.Hostname)

		daWallet, err := bridgeNode.GetWallet()
		s.Require().NoError(err)
		s.FundWallet(ctx, daWallet, 100_000_000_00)

		bridgeNetworkInfo, err := bridgeNode.GetNetworkInfo(ctx)
		s.Require().NoError(err)
		s.daAddress = fmt.Sprintf("http://%s:%s", bridgeNetworkInfo.Internal.IP, bridgeNetworkInfo.Internal.Ports.RPC)
		s.T().Log("DA bridge node started and funded")
	})

	s.Run("setup_reth_node", func() {
		s.setupRethNode(ctx)
		s.T().Log("Reth node started")
	})

	s.Run("setup_evm_with_base_version", func() {
		s.setupEVMSingle(ctx, container.NewImage("ghcr.io/evstack/ev-node-evm", baseEVMSingleVersion, ""))
		s.T().Logf("evm started with base version: %s", baseEVMSingleVersion)
	})

	s.Run("create_ethereum_client", func() {
		s.setupEthClient(ctx)
		s.T().Log("Ethereum client connected to Reth")
	})

	var preUpgradeTxHashes []common.Hash

	s.Run("pre_upgrade_submit_transactions_and_verify_state", func() {
		preUpgradeTxHashes = s.submitPreUpgradeTxs(ctx, 20)
		s.T().Logf("Pre-upgrade transactions verified: %d txs", len(preUpgradeTxHashes))
	})

	s.Run("perform_upgrade", func() {
		s.performUpgrade(ctx)
	})

	s.Run("post_upgrade_verify_old_transactions_persist", func() {
		s.verifyOldTxsPersist(ctx, preUpgradeTxHashes)
	})

	s.Run("post_upgrade_submit_new_transactions_and_verify", func() {
		s.submitAndVerifyPostUpgradeTx(ctx)
	})

	s.Run("post_upgrade_verify_account_balances", func() {
		s.verifyAccountBalances(ctx)
	})
}

// setupRethNode creates and starts a Reth node, waiting for it to be ready.
func (s *EVMSingleUpgradeTestSuite) setupRethNode(ctx context.Context) {
	rethNode, err := reth.NewNodeBuilder(s.T()).
		WithGenesis([]byte(reth.DefaultEvolveGenesisJSON())).
		WithDockerClient(s.dockerClient).
		WithDockerNetworkID(s.dockerNetworkID).
		Build(ctx)
	s.Require().NoError(err)

	s.Require().NoError(rethNode.Start(ctx))

	// wait for reth to be ready
	s.Require().Eventually(func() bool {
		networkInfo, err := rethNode.GetNetworkInfo(ctx)
		if err != nil {
			return false
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("0.0.0.0", networkInfo.External.Ports.RPC), time.Second)
		if err != nil {
			return false
		}
		conn.Close()

		// internal URLs for container-to-container communication (evm -> reth)
		s.evmEngineURL = fmt.Sprintf("http://%s:%s", networkInfo.Internal.Hostname, networkInfo.Internal.Ports.Engine)
		s.evmEthURL = fmt.Sprintf("http://%s:%s", networkInfo.Internal.Hostname, networkInfo.Internal.Ports.RPC)
		// external URL for test code -> reth communication
		s.evmEthURLExternal = fmt.Sprintf("http://0.0.0.0:%s", networkInfo.External.Ports.RPC)
		return true
	}, 60*time.Second, 2*time.Second, "reth did not start in time")

	genesisHash, err := rethNode.GenesisHash(ctx)
	s.Require().NoError(err)

	s.rethNode = rethNode
	s.jwtSecret = rethNode.JWTSecretHex()
	s.rethGenesisHash = genesisHash
}

// setupEVMSingle creates and starts an evm node with the specified version.
func (s *EVMSingleUpgradeTestSuite) setupEVMSingle(ctx context.Context, image container.Image) {
	nodeConfig := evmsingle.NewNodeConfigBuilder().
		WithEVMEngineURL(s.evmEngineURL).
		WithEVMETHURL(s.evmEthURL).
		WithEVMJWTSecret(s.jwtSecret).
		WithEVMGenesisHash(s.rethGenesisHash).
		WithEVMBlockTime("1s").
		WithEVMSignerPassphrase("secret").
		WithDAAddress(s.daAddress).
		WithDANamespace("evm-header").
		WithAdditionalStartArgs(
			"--evnode.rpc.address", "0.0.0.0:7331",
			"--evnode.da.data_namespace", "evm-data",
		).
		Build()

	evmSingleChain, err := evmsingle.NewChainBuilder(s.T()).
		WithDockerClient(s.dockerClient).
		WithDockerNetworkID(s.dockerNetworkID).
		WithImage(image).
		WithBinary("evm").
		WithNode(nodeConfig).
		Build(ctx)

	s.Require().NoError(err)
	s.Require().Len(evmSingleChain.Nodes(), 1)

	evmSingleNode := evmSingleChain.Nodes()[0]
	s.Require().NoError(evmSingleNode.Start(ctx))

	// wait for evm to be healthy
	s.waitForEVMSingleHealthy(ctx, evmSingleNode)

	s.evmSingleChain = evmSingleChain
	s.evmSingleNode = evmSingleNode
}

// waitForEVMSingleHealthy waits for evm to respond to health checks.
func (s *EVMSingleUpgradeTestSuite) waitForEVMSingleHealthy(ctx context.Context, node *evmsingle.Node) {
	networkInfo, err := node.GetNetworkInfo(ctx)
	s.Require().NoError(err)

	healthURL := fmt.Sprintf("http://0.0.0.0:%s/health/live", networkInfo.External.Ports.RPC)
	s.Require().Eventually(func() bool {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 60*time.Second, 2*time.Second, "evm did not become healthy")
}

// setupEthClient creates an Ethereum client and verifies connectivity.
func (s *EVMSingleUpgradeTestSuite) setupEthClient(ctx context.Context) {
	// use external URL since test code runs on host, not in docker network
	ethClient, err := ethclient.Dial(s.evmEthURLExternal)
	s.Require().NoError(err)

	// verify connection by getting chain ID
	chainID, err := ethClient.ChainID(ctx)
	s.Require().NoError(err)
	s.Require().Equal(evmChainID, chainID.String())

	s.ethClient = ethClient
}

// submitPreUpgradeTxs submits and verifies multiple transactions before upgrade.
func (s *EVMSingleUpgradeTestSuite) submitPreUpgradeTxs(ctx context.Context, txCount int) []common.Hash {
	var txHashes []common.Hash

	for i := range txCount {
		tx := evm.GetRandomTransaction(s.T(), testPrivateKey, testToAddress, evmChainID, testGasLimit, &s.txNonce)
		err := s.ethClient.SendTransaction(ctx, tx)
		s.Require().NoError(err)
		txHashes = append(txHashes, tx.Hash())
		s.T().Logf("Submitted pre-upgrade tx %d: %s", i, tx.Hash().Hex())
		s.waitForTxIncluded(ctx, tx.Hash())
	}

	return txHashes
}

// waitForTxIncluded waits for a transaction to be included in a block successfully.
func (s *EVMSingleUpgradeTestSuite) waitForTxIncluded(ctx context.Context, txHash common.Hash) {
	s.Require().Eventually(func() bool {
		receipt, err := s.ethClient.TransactionReceipt(ctx, txHash)
		if err != nil {
			return false
		}
		return receipt.Status == 1
	}, 30*time.Second, time.Second, "transaction %s was not included", txHash.Hex())
}

// performUpgrade performs the upgrade by removing the container, updating the image, and restarting.
func (s *EVMSingleUpgradeTestSuite) performUpgrade(ctx context.Context) {
	// remove container but preserve volumes
	err := s.evmSingleNode.Remove(ctx, tastoratypes.WithPreserveVolumes())
	s.Require().NoError(err, "failed to remove container with volume preservation")
	s.T().Log("Removed container with volume preservation")

	// image to upgrade to will be passed via environment variables
	newImage := getEVMSingleImage()

	s.T().Logf("Upgrading to version: %s", newImage.Version)
	s.evmSingleNode.Image = newImage
	for _, node := range s.evmSingleChain.Nodes() {
		node.Image = newImage
	}

	// start with new version on top of the same docker volume.
	err = s.evmSingleNode.Start(ctx)
	s.Require().NoError(err, "failed to start upgraded node")

	// wait for health check
	s.waitForEVMSingleHealthy(ctx, s.evmSingleNode)

	s.T().Logf("Upgraded node started successfully with version: %s", newImage.Version)
}

// verifyOldTxsPersist verifies that all pre-upgrade transactions are still queryable.
func (s *EVMSingleUpgradeTestSuite) verifyOldTxsPersist(ctx context.Context, txHashes []common.Hash) {
	for i, txHash := range txHashes {
		receipt, err := s.ethClient.TransactionReceipt(ctx, txHash)
		s.Require().NoError(err, "failed to query pre-upgrade tx %d: %s", i+1, txHash.Hex())
		s.Require().Equal(uint64(1), receipt.Status, "pre-upgrade tx %d should have successful status", i+1)
		s.T().Logf("Pre-upgrade tx %d still accessible: %s (block %d)", i+1, txHash.Hex(), receipt.BlockNumber.Uint64())
	}

	s.T().Logf("All %d pre-upgrade transactions persisted after upgrade", len(txHashes))
}

// submitAndVerifyPostUpgradeTx submits a new transaction after upgrade and verifies block production.
func (s *EVMSingleUpgradeTestSuite) submitAndVerifyPostUpgradeTx(ctx context.Context) {
	tx := evm.GetRandomTransaction(s.T(), testPrivateKey, testToAddress, evmChainID, testGasLimit, &s.txNonce)
	err := s.ethClient.SendTransaction(ctx, tx)
	s.Require().NoError(err)
	s.T().Logf("Submitted post-upgrade tx: %s", tx.Hash().Hex())

	// wait for transaction to be included
	s.waitForTxIncluded(ctx, tx.Hash())

	// get final receipt to check block number
	receipt, err := s.ethClient.TransactionReceipt(ctx, tx.Hash())
	s.Require().NoError(err)
	s.T().Logf("Post-upgrade tx included in block %d", receipt.BlockNumber.Uint64())

	// verify block production is continuing
	initialBlock, err := s.ethClient.BlockNumber(ctx)
	s.Require().NoError(err)

	time.Sleep(3 * time.Second)

	laterBlock, err := s.ethClient.BlockNumber(ctx)
	s.Require().NoError(err)
	s.Require().Greater(laterBlock, initialBlock, "block height should increase after upgrade")

	s.T().Logf("Post-upgrade block production verified (block %d -> %d)", initialBlock, laterBlock)
}

// verifyAccountBalances verifies that account balances are correct after upgrade.
func (s *EVMSingleUpgradeTestSuite) verifyAccountBalances(ctx context.Context) {
	senderAddr := common.HexToAddress("0xd143C405751162d0F96bEE2eB5eb9C61882a736E")
	balance, err := s.ethClient.BalanceAt(ctx, senderAddr, nil)
	s.Require().NoError(err)
	s.Require().True(balance.Cmp(big.NewInt(0)) > 0, "sender should have non-zero balance")
	s.T().Logf("Sender balance after upgrade: %s wei", balance.String())

	recipientAddr := common.HexToAddress(testToAddress)
	recipientBalance, err := s.ethClient.BalanceAt(ctx, recipientAddr, nil)
	s.Require().NoError(err)
	s.T().Logf("Recipient balance after upgrade: %s wei", recipientBalance.String())

	s.T().Log("Account balances verified after upgrade")
}

// getEVMSingleImage returns the Docker image configuration for evm with the specified version.
func getEVMSingleImage() container.Image {
	repo := strings.TrimSpace(os.Getenv("EVM_IMAGE_REPO"))
	if repo == "" {
		repo = "evm"
	}
	upgradeVersion := strings.TrimSpace(os.Getenv("EVM_NODE_IMAGE_TAG"))
	if upgradeVersion == "" {
		upgradeVersion = "local-dev"
	}

	// evm runs as root (no specific user in Dockerfile), so UIDGID is empty
	return container.NewImage(repo, upgradeVersion, "")
}
