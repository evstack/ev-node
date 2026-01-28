//go:build docker_e2e && evm

package docker_e2e

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/evstack/evmsingle"
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

	reth           RethSetup
	daAddress      string
	evmSingleChain *evmsingle.Chain
	evmSingleNode  *evmsingle.Node
	ethClient      *ethclient.Client
	txNonce        uint64
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
		s.daAddress = s.SetupCelestiaAndDABridge(ctx)
		s.T().Log("DA bridge node started and funded")
	})

	s.Run("setup_reth_node", func() {
		s.reth = s.SetupRethNode(ctx)
		s.T().Log("Reth node started")
	})

	s.Run("setup_evm_with_base_version", func() {
		s.setupEVMSingle(ctx, container.NewImage("ghcr.io/evstack/ev-node-evm", baseEVMSingleVersion, ""))
		s.T().Logf("evm started with base version: %s", baseEVMSingleVersion)
	})

	s.Run("create_ethereum_client", func() {
		s.ethClient = s.SetupEthClient(ctx, s.reth.EthURLExternal, evmChainID)
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

// setupEVMSingle creates and starts an evm node with the specified version.
func (s *EVMSingleUpgradeTestSuite) setupEVMSingle(ctx context.Context, image container.Image) {
	nodeConfig := evmsingle.NewNodeConfigBuilder().
		WithEVMEngineURL(s.reth.EngineURL).
		WithEVMETHURL(s.reth.EthURL).
		WithEVMJWTSecret(s.reth.JWTSecret).
		WithEVMGenesisHash(s.reth.GenesisHash).
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
	s.WaitForEVMHealthy(ctx, evmSingleNode)

	s.evmSingleChain = evmSingleChain
	s.evmSingleNode = evmSingleNode
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
		s.WaitForTxIncluded(ctx, s.ethClient, tx.Hash())
	}

	return txHashes
}

// performUpgrade performs the upgrade by removing the container, updating the image, and restarting.
func (s *EVMSingleUpgradeTestSuite) performUpgrade(ctx context.Context) {
	err := s.evmSingleNode.Remove(ctx, tastoratypes.WithPreserveVolumes())
	s.Require().NoError(err, "failed to remove container with volume preservation")
	s.T().Log("Removed container with volume preservation")

	newImage := getEVMSingleImage()

	s.T().Logf("Upgrading to version: %s", newImage.Version)
	s.evmSingleNode.Image = newImage
	for _, node := range s.evmSingleChain.Nodes() {
		node.Image = newImage
	}

	err = s.evmSingleNode.Start(ctx)
	s.Require().NoError(err, "failed to start upgraded node")
	s.WaitForEVMHealthy(ctx, s.evmSingleNode)

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

	s.WaitForTxIncluded(ctx, s.ethClient, tx.Hash())

	receipt, err := s.ethClient.TransactionReceipt(ctx, tx.Hash())
	s.Require().NoError(err)
	s.T().Logf("Post-upgrade tx included in block %d", receipt.BlockNumber.Uint64())

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

	return container.NewImage(repo, upgradeVersion, "")
}
