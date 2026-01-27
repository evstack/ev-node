//go:build docker_e2e && evm

package docker_e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/evstack/evmsingle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/evstack/ev-node/execution/evm"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

const (
	compatChainID    = "1234"
	compatPrivateKey = "cece4f25ac74deb1468965160c7185e07dff413f23fcadb611b05ca37ab0a52e"
	compatToAddress  = "0x944fDcD1c868E3cC566C78023CcB38A32cDA836E"
	compatGasLimit   = uint64(22000)
)

// EVMCompatTestSuite tests cross-version compatibility between different ev-node-evm versions.
type EVMCompatTestSuite struct {
	DockerTestSuite

	rethCfg       RethSetupConfig
	daAddress     string
	sequencerNode *evmsingle.Node
	fullNodeNode  *evmsingle.Node
	ethClient     *ethclient.Client
	txNonce       uint64
}

func TestEVMCompatSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping due to short mode")
	}
	suite.Run(t, new(EVMCompatTestSuite))
}

func (s *EVMCompatTestSuite) TestCrossVersionSync() {
	ctx := context.Background()
	s.setupDockerEnvironment()

	s.Run("setup_infrastructure", func() {
		s.daAddress = s.SetupCelestiaAndDABridge(ctx)
		s.T().Log("Celestia and DA bridge started")

		s.rethCfg = s.SetupRethNode(ctx)
		s.T().Log("Reth node started")
	})

	s.Run("setup_sequencer", func() {
		s.setupSequencer(ctx, getSequencerImage())
	})

	var preSyncTxHashes []common.Hash
	s.Run("submit_transactions", func() {
		s.ethClient = s.SetupEthClient(ctx, s.rethCfg.EthURLExternal, compatChainID)
		preSyncTxHashes = s.submitTransactions(ctx, 10)
	})

	sequencerHeight, err := s.ethClient.BlockNumber(ctx)
	s.Require().NoError(err)

	s.Run("setup_fullnode_and_sync", func() {
		s.setupFullNode(ctx, getFullNodeImage())
		s.waitForSync(ctx, sequencerHeight)
	})

	s.Run("verify_sync", func() {
		s.verifyTransactions(ctx, preSyncTxHashes)

		// submit more transactions and verify ongoing sync
		postSyncTxHashes := s.submitTransactions(ctx, 5)
		latestHeight, err := s.ethClient.BlockNumber(ctx)
		s.Require().NoError(err)
		s.waitForSync(ctx, latestHeight)
		s.verifyTransactions(ctx, postSyncTxHashes)
	})
}

func (s *EVMCompatTestSuite) setupSequencer(ctx context.Context, image container.Image) {
	s.T().Logf("Setting up sequencer: %s:%s", image.Repository, image.Version)

	nodeConfig := evmsingle.NewNodeConfigBuilder().
		WithEVMEngineURL(s.rethCfg.EngineURL).
		WithEVMETHURL(s.rethCfg.EthURL).
		WithEVMJWTSecret(s.rethCfg.JWTSecret).
		WithEVMGenesisHash(s.rethCfg.GenesisHash).
		WithEVMBlockTime("1s").
		WithEVMSignerPassphrase("secret").
		WithDAAddress(s.daAddress).
		WithDANamespace("compat-header").
		WithAdditionalStartArgs(
			"--evnode.da.data_namespace", "compat-data",
			"--evnode.p2p.listen_address", "/ip4/0.0.0.0/tcp/26656",
		).
		Build()

	chain, err := evmsingle.NewChainBuilder(s.T()).
		WithDockerClient(s.dockerClient).
		WithDockerNetworkID(s.dockerNetworkID).
		WithImage(image).
		WithBinary("evm").
		WithName("seq").
		WithNode(nodeConfig).
		Build(ctx)
	s.Require().NoError(err)

	s.sequencerNode = chain.Nodes()[0]
	s.Require().NoError(s.sequencerNode.Start(ctx))
	s.WaitForEVMHealthy(ctx, s.sequencerNode)
	s.T().Log("Sequencer started")
}

func (s *EVMCompatTestSuite) setupFullNode(ctx context.Context, image container.Image) {
	s.T().Logf("Setting up full node: %s:%s", image.Repository, image.Version)

	sequencerP2PAddr := s.getSequencerP2PAddress(ctx)

	genesis, err := s.sequencerNode.ReadFile(ctx, "config/genesis.json")
	s.Require().NoError(err)

	nodeConfig := evmsingle.NewNodeConfigBuilder().
		WithEVMEngineURL(s.rethCfg.EngineURL).
		WithEVMETHURL(s.rethCfg.EthURL).
		WithEVMJWTSecret(s.rethCfg.JWTSecret).
		WithEVMGenesisHash(s.rethCfg.GenesisHash).
		WithDAAddress(s.daAddress).
		WithDANamespace("compat-header").
		WithAdditionalStartArgs(
			"--evnode.da.data_namespace", "compat-data",
			"--evnode.p2p.listen_address", "/ip4/0.0.0.0/tcp/26656",
			"--rollkit.p2p.peers", sequencerP2PAddr,
		).
		Build()

	chain, err := evmsingle.NewChainBuilder(s.T()).
		WithDockerClient(s.dockerClient).
		WithDockerNetworkID(s.dockerNetworkID).
		WithImage(image).
		WithBinary("evm").
		WithName("fn").
		WithNode(nodeConfig).
		Build(ctx)
	s.Require().NoError(err)

	s.fullNodeNode = chain.Nodes()[0]
	s.Require().NoError(s.fullNodeNode.WriteFile(ctx, "config/genesis.json", genesis))
	s.Require().NoError(s.fullNodeNode.Start(ctx))
	s.WaitForEVMHealthy(ctx, s.fullNodeNode)
	s.T().Log("Full node started")
}

type nodeKeyJSON struct {
	PrivKeyBytes []byte `json:"priv_key"`
	PubKeyBytes  []byte `json:"pub_key"`
}

func (s *EVMCompatTestSuite) getSequencerP2PAddress(ctx context.Context) string {
	nodeKeyData, err := s.sequencerNode.ReadFile(ctx, "config/node_key.json")
	s.Require().NoError(err)

	var nk nodeKeyJSON
	s.Require().NoError(json.Unmarshal(nodeKeyData, &nk))

	pubKey, err := crypto.UnmarshalEd25519PublicKey(nk.PubKeyBytes)
	s.Require().NoError(err)

	peerID, err := peer.IDFromPublicKey(pubKey)
	s.Require().NoError(err)

	networkInfo, err := s.sequencerNode.GetNetworkInfo(ctx)
	s.Require().NoError(err)

	return fmt.Sprintf("/dns4/%s/tcp/26656/p2p/%s", networkInfo.Internal.Hostname, peerID.String())
}

func (s *EVMCompatTestSuite) submitTransactions(ctx context.Context, count int) []common.Hash {
	var txHashes []common.Hash
	for i := range count {
		tx := evm.GetRandomTransaction(s.T(), compatPrivateKey, compatToAddress, compatChainID, compatGasLimit, &s.txNonce)
		s.Require().NoError(s.ethClient.SendTransaction(ctx, tx))
		txHashes = append(txHashes, tx.Hash())
		s.T().Logf("Submitted tx %d: %s", i, tx.Hash().Hex())
		s.WaitForTxIncluded(ctx, s.ethClient, tx.Hash())
	}
	s.T().Logf("Submitted %d transactions", len(txHashes))
	return txHashes
}

func (s *EVMCompatTestSuite) waitForSync(ctx context.Context, targetHeight uint64) {
	s.Require().Eventually(func() bool {
		height, err := s.ethClient.BlockNumber(ctx)
		return err == nil && height >= targetHeight
	}, 120*time.Second, 2*time.Second, "full node did not sync to height %d", targetHeight)
	s.T().Logf("Full node synced to height %d", targetHeight)
}

func (s *EVMCompatTestSuite) verifyTransactions(ctx context.Context, txHashes []common.Hash) {
	for i, txHash := range txHashes {
		receipt, err := s.ethClient.TransactionReceipt(ctx, txHash)
		s.Require().NoError(err, "failed to query tx %d: %s", i, txHash.Hex())
		s.Require().Equal(uint64(1), receipt.Status)
	}
	s.T().Logf("Verified %d transactions", len(txHashes))
}

func getSequencerImage() container.Image {
	repo := strings.TrimSpace(os.Getenv("SEQUENCER_EVM_IMAGE_REPO"))
	if repo == "" {
		repo = "ghcr.io/evstack/ev-node-evm"
	}
	tag := strings.TrimSpace(os.Getenv("SEQUENCER_EVM_IMAGE_TAG"))
	if tag == "" {
		tag = "main"
	}
	return container.NewImage(repo, tag, "")
}

func getFullNodeImage() container.Image {
	repo := strings.TrimSpace(os.Getenv("FULLNODE_EVM_IMAGE_REPO"))
	if repo == "" {
		repo = "ghcr.io/evstack/ev-node-evm"
	}
	tag := strings.TrimSpace(os.Getenv("FULLNODE_EVM_IMAGE_TAG"))
	if tag == "" {
		tag = "main"
	}
	return container.NewImage(repo, tag, "")
}
