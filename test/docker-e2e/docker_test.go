//go:build docker_e2e

package docker_e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/math"
	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/cosmos"
	da "github.com/celestiaorg/tastora/framework/docker/dataavailability"
	"github.com/celestiaorg/tastora/framework/docker/evstack"
	"github.com/celestiaorg/tastora/framework/testutil/sdkacc"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/suite"
)

const (
	// testChainID is the chain ID used for testing.
	// it must be the string "test" as it is handled explicitly in app/node.
	testChainID = "test"
	// celestiaAppVersion specifies the tag of the celestia-app image to deploy in tests.
	celestiaAppVersion = "v5.0.2"
)

func init() {
	sdkConf := sdk.GetConfig()
	sdkConf.SetBech32PrefixForAccount("celestia", "celestiapub")
	sdkConf.Seal()
}

func TestDockerSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping due to short mode")
	}
	suite.Run(t, new(DockerTestSuite))
}

type DockerTestSuite struct {
	suite.Suite
	celestia        *cosmos.Chain
	daNetwork       *da.Network
	evNodeChain     *evstack.Chain
	dockerClient    *dockerclient.Client
	dockerNetworkID string
}

// setupDockerEnvironment sets up the basic Docker environment
func (s *DockerTestSuite) setupDockerEnvironment() {
	t := s.T()
	client, network := tastoradocker.DockerSetup(t)

	// Store client and network ID in the suite for later use
	s.dockerClient = client
	s.dockerNetworkID = network
}

// getGenesisHash returns the genesis hash of the given chain node.
func (s *DockerTestSuite) getGenesisHash(ctx context.Context) string {
	node := s.celestia.GetNodes()[0]
	c, err := node.GetRPCClient()
	s.Require().NoError(err, "failed to get node client")

	first := int64(1)
	block, err := c.Block(ctx, &first)
	s.Require().NoError(err, "failed to get block")

	genesisHash := block.Block.Header.Hash().String()
	s.Require().NotEmpty(genesisHash, "genesis hash is empty")
	return genesisHash
}

// SetupDockerResources creates chains using the builder pattern.
// none of the resources are started.
func (s *DockerTestSuite) SetupDockerResources() {
	s.setupDockerEnvironment()
	s.celestia = s.CreateChain()
	s.daNetwork = s.CreateDANetwork()
	s.evNodeChain = s.CreateEvolveChain()
}

// CreateChain creates a chain using the ChainBuilder pattern.
func (s *DockerTestSuite) CreateChain() *cosmos.Chain {
	ctx := context.Background()
	t := s.T()
	encConfig := testutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{})

	// Create chain using ChainBuilder pattern
	chain, err := cosmos.NewChainBuilder(t).
		WithName("celestia").
		WithChainID(testChainID).
		WithBinaryName("celestia-appd").
		WithBech32Prefix("celestia").
		WithDenom("utia").
		WithCoinType("118").
		WithGasPrices("0.025utia").
		WithGasAdjustment(1.3).
		WithEncodingConfig(&encConfig).
		WithImage(container.NewImage("ghcr.io/celestiaorg/celestia-app", celestiaAppVersion, "10001:10001")).
		WithAdditionalStartArgs(
			"--force-no-bbr",
			"--grpc.enable",
			"--grpc.address",
			"0.0.0.0:9090",
			"--rpc.grpc_laddr=tcp://0.0.0.0:9098",
			"--timeout-commit", "1s",
		).
		WithDockerClient(s.dockerClient).
		WithDockerNetworkID(s.dockerNetworkID).
		WithNode(cosmos.NewChainNodeConfigBuilder().
			WithNodeType(tastoratypes.NodeTypeValidator).
			Build()).
		Build(ctx)

	s.Require().NoError(err)
	return chain
}

// CreateDANetwork creates a DA network using the builder pattern
func (s *DockerTestSuite) CreateDANetwork() *da.Network {
	ctx := context.Background()
	t := s.T()

	bridgeNodeConfig := da.NewNodeBuilder().
		WithNodeType(tastoratypes.BridgeNode).
		Build()

	daNetwork, err := da.NewNetworkBuilder(t).
		WithDockerClient(s.dockerClient).
		WithDockerNetworkID(s.dockerNetworkID).
		WithImage(container.NewImage("ghcr.io/celestiaorg/celestia-node", "v0.25.3", "10001:10001")).
		WithNode(bridgeNodeConfig).
		Build(ctx)
	s.Require().NoError(err)

	return daNetwork
}

// CreateEvolveChain creates an evstack chain using the builder pattern
func (s *DockerTestSuite) CreateEvolveChain() *evstack.Chain {
	ctx := context.Background()
	t := s.T()

	aggregatorNodeConfig := evstack.NewNodeBuilder().
		WithAggregator(true).
		Build()

	evNodeChain, err := evstack.NewChainBuilder(t).
		WithChainID("evchain-test").
		WithBinaryName("testapp").
		WithAggregatorPassphrase("12345678").
		WithImage(getEvNodeImage()).
		WithDockerClient(s.dockerClient).
		WithDockerNetworkID(s.dockerNetworkID).
		WithNode(aggregatorNodeConfig).
		Build(ctx)
	s.Require().NoError(err)

	return evNodeChain
}

// StartBridgeNode initializes and starts a bridge node within the data availability network using the given parameters.
func (s *DockerTestSuite) StartBridgeNode(ctx context.Context, bridgeNode *da.Node, chainID string, genesisHash string, celestiaNodeHostname string) {
	s.Require().Equal(tastoratypes.BridgeNode, bridgeNode.GetType())
	err := bridgeNode.Start(ctx,
		da.WithChainID(chainID),
		da.WithAdditionalStartArguments("--p2p.network", chainID, "--core.ip", celestiaNodeHostname, "--rpc.addr", "0.0.0.0"),
		da.WithEnvironmentVariables(
			map[string]string{
				"CELESTIA_CUSTOM": tastoratypes.BuildCelestiaCustomEnvVar(chainID, genesisHash, ""),
				"P2P_NETWORK":     chainID,
			},
		),
	)
	s.Require().NoError(err)
}

// FundWallet transfers the specified amount of utia from the faucet wallet to the target wallet.
func (s *DockerTestSuite) FundWallet(ctx context.Context, wallet *tastoratypes.Wallet, amount int64) {
	fromAddress, err := sdkacc.AddressFromWallet(s.celestia.GetFaucetWallet())
	s.Require().NoError(err)

	toAddress, err := sdk.AccAddressFromBech32(wallet.GetFormattedAddress())
	s.Require().NoError(err)

	bankSend := banktypes.NewMsgSend(fromAddress, toAddress, sdk.NewCoins(sdk.NewCoin("utia", math.NewInt(amount))))
	_, err = s.celestia.BroadcastMessages(ctx, s.celestia.GetFaucetWallet(), bankSend)
	s.Require().NoError(err)
}

// StartEVNode initializes and starts an Ev node.
func (s *DockerTestSuite) StartEVNode(ctx context.Context, bridgeNode *da.Node, evNode *evstack.Node) {
	s.StartEVNodeWithNamespace(ctx, bridgeNode, evNode, "ev-header", "ev-data")
}

// StartEVNodeWithNamespace initializes and starts an EV node with a specific namespace.
func (s *DockerTestSuite) StartEVNodeWithNamespace(ctx context.Context, bridgeNode *da.Node, evNode *evstack.Node, headerNamespace, dataNamespace string) {
	err := evNode.Init(ctx)
	s.Require().NoError(err)

	bridgeNetworkInfo, err := bridgeNode.GetNetworkInfo(ctx)
	s.Require().NoError(err)

	authToken, err := bridgeNode.GetAuthToken()
	s.Require().NoError(err)

	bridgeRPCAddress := bridgeNetworkInfo.Internal.RPCAddress()
	daAddress := fmt.Sprintf("http://%s", bridgeRPCAddress)
	err = evNode.Start(ctx,
		"--evnode.da.address", daAddress,
		"--evnode.da.gas_price", "0.025",
		"--evnode.da.auth_token", authToken,
		"--evnode.rpc.address", "0.0.0.0:7331", // bind to 0.0.0.0 so rpc is reachable from test host.
		"--evnode.da.namespace", headerNamespace,
		"--evnode.da.data_namespace", dataNamespace,
		"--kv-endpoint", "0.0.0.0:8080",
	)
	s.Require().NoError(err)
}

// getEvNodeImage returns the Docker image configuration for EV Node
// Uses EV_NODE_IMAGE_REPO and EV_NODE_IMAGE_TAG environment variables if set
// Defaults to locally built image using a unique tag to avoid registry conflicts
func getEvNodeImage() container.Image {
	repo := strings.TrimSpace(os.Getenv("EV_NODE_IMAGE_REPO"))
	if repo == "" {
		repo = "evstack"
	}

	tag := strings.TrimSpace(os.Getenv("EV_NODE_IMAGE_TAG"))
	if tag == "" {
		tag = "local-dev"
	}

	return container.NewImage(repo, tag, "10001:10001")
}
