//go:build docker_e2e

package docker_e2e

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/cosmos"
	da "github.com/celestiaorg/tastora/framework/docker/dataavailability"
	"github.com/celestiaorg/tastora/framework/docker/evstack/evmsingle"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	"github.com/celestiaorg/tastora/framework/testutil/sdkacc"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/evstack/ev-node/execution/evm"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

const (
	// baseEVMSingleVersion is the stable release version to start with before upgrading.
	baseEVMSingleVersion = "v1.0.0"

	// test account configuration (matches genesis accounts in reth/genesis.go)
	testPrivateKey = "cece4f25ac74deb1468965160c7185e07dff413f23fcadb611b05ca37ab0a52e"
	testToAddress  = "0x944fDcD1c868E3cC566C78023CcB38A32cDA836E"
	testGasLimit   = uint64(22000)
)

// TestEVMSingleUpgrade tests upgrading an evm-single node from a stable release
// to a PR-built image while preserving state.
//
// Test Flow:
// 1. Deploy Celestia chain, DA bridge node, Reth, and evm-single (stable version)
// 2. Submit Ethereum transactions and verify state before upgrade
// 3. Stop evm-single, update image, remove with volume preservation, restart
// 4. Verify old transactions persist and new transactions work after upgrade
func TestEVMSingleUpgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping due to short mode")
	}

	ctx := context.Background()

	// setup docker environment
	dockerClient, dockerNetworkID := tastoradocker.DockerSetup(t)

	var (
		celestiaChain   *cosmos.Chain
		daNetwork       *da.Network
		bridgeNode      *da.Node
		rethNode        *reth.Node
		evmSingleChain  *evmsingle.Chain
		evmSingleNode   *evmsingle.Node
		ethClient       *ethclient.Client
		daAddress       string
		evmEngineURL    string
		evmEthURL       string
		jwtSecret       string
		rethGenesisHash string
		txNonce         uint64 // track nonce for transactions
	)

	t.Run("setup celestia chain", func(t *testing.T) {
		celestiaChain = createCelestiaChain(t, dockerClient, dockerNetworkID)
		err := celestiaChain.Start(ctx)
		require.NoError(t, err)
		t.Log("Celestia chain started")
	})

	t.Run("setup DA bridge node", func(t *testing.T) {
		chainID := celestiaChain.GetChainID()
		daNetwork = createDANetwork(t, dockerClient, dockerNetworkID)

		// get genesis hash and celestia node info
		genesisHash := getGenesisHashFromChain(t, ctx, celestiaChain)
		networkInfo, err := celestiaChain.GetNodes()[0].GetNetworkInfo(ctx)
		require.NoError(t, err)
		celestiaNodeHostname := networkInfo.Internal.Hostname

		// start bridge node
		bridgeNode = daNetwork.GetBridgeNodes()[0]
		err = bridgeNode.Start(ctx,
			da.WithChainID(chainID),
			da.WithAdditionalStartArguments("--p2p.network", chainID, "--core.ip", celestiaNodeHostname, "--rpc.addr", "0.0.0.0"),
			da.WithEnvironmentVariables(
				map[string]string{
					"CELESTIA_CUSTOM": tastoratypes.BuildCelestiaCustomEnvVar(chainID, genesisHash, ""),
					"P2P_NETWORK":     chainID,
				},
			),
		)
		require.NoError(t, err)

		// fund DA wallet
		daWallet, err := bridgeNode.GetWallet()
		require.NoError(t, err)
		fundWallet(t, ctx, celestiaChain, daWallet, 100_000_000_00)

		// get DA address for evm-single
		bridgeNetworkInfo, err := bridgeNode.GetNetworkInfo(ctx)
		require.NoError(t, err)
		daAddress = fmt.Sprintf("http://%s:%s", bridgeNetworkInfo.Internal.IP, bridgeNetworkInfo.Internal.Ports.RPC)

		// get auth token for DA
		authToken, err := bridgeNode.GetAuthToken()
		require.NoError(t, err)
		require.NotEmpty(t, authToken, "DA auth token should not be empty")

		t.Log("DA bridge node started and funded")
	})

	t.Run("setup reth node", func(t *testing.T) {
		var err error
		rethNode, err = reth.NewNodeBuilderWithTestName(t, t.Name()).
			WithDockerClient(dockerClient).
			WithDockerNetworkID(dockerNetworkID).
			WithGenesis([]byte(reth.DefaultEvolveGenesisJSON())).
			Build(ctx)
		require.NoError(t, err)

		err = rethNode.Start(ctx)
		require.NoError(t, err)

		// wait for reth to be ready
		require.Eventually(t, func() bool {
			ec, err := rethNode.GetEthClient(ctx)
			if err != nil {
				return false
			}
			if _, err := ec.BlockNumber(ctx); err != nil {
				return false
			}
			return true
		}, 45*time.Second, 1*time.Second, "reth JSON-RPC did not become ready")

		// get reth genesis hash
		rethGenesisHash, err = rethNode.GenesisHash(ctx)
		require.NoError(t, err)

		// ensure engine port is open
		rethNetworkInfo, err := rethNode.GetNetworkInfo(ctx)
		require.NoError(t, err)
		engineHost := fmt.Sprintf("0.0.0.0:%s", rethNetworkInfo.External.Ports.Engine)
		require.Eventually(t, func() bool {
			c, err := net.DialTimeout("tcp", engineHost, 2*time.Second)
			if err != nil {
				return false
			}
			_ = c.Close()
			return true
		}, 45*time.Second, 1*time.Second, "reth Engine port did not open")

		// get reth connection info for evm-single (use internal addresses for container-to-container)
		evmEthURL = fmt.Sprintf("http://%s:%s", rethNetworkInfo.Internal.Hostname, rethNetworkInfo.Internal.Ports.RPC)
		evmEngineURL = fmt.Sprintf("http://%s:%s", rethNetworkInfo.Internal.Hostname, rethNetworkInfo.Internal.Ports.Engine)
		jwtSecret = rethNode.JWTSecretHex()

		t.Log("Reth node started")
	})

	t.Run("setup evm-single with base version", func(t *testing.T) {
		var err error
		evmSingleChain, err = evmsingle.NewChainBuilder(t).
			WithImage(getEVMSingleImage(baseEVMSingleVersion)).
			WithDockerClient(dockerClient).
			WithDockerNetworkID(dockerNetworkID).
			WithNode(
				evmsingle.NewNodeConfigBuilder().
					WithEVMEngineURL(evmEngineURL).
					WithEVMETHURL(evmEthURL).
					WithEVMJWTSecret(jwtSecret).
					WithEVMSignerPassphrase("secret").
					WithEVMBlockTime("1s").
					WithEVMGenesisHash(rethGenesisHash).
					WithDAAddress(daAddress).
					Build(),
			).
			Build(ctx)
		require.NoError(t, err)

		err = evmSingleChain.Start(ctx)
		require.NoError(t, err)

		evmSingleNode = evmSingleChain.Nodes()[0]

		// health check
		networkInfo, err := evmSingleNode.GetNetworkInfo(ctx)
		require.NoError(t, err)
		healthURL := fmt.Sprintf("http://0.0.0.0:%s/evnode.v1.HealthService/Livez", networkInfo.External.Ports.RPC)
		require.Eventually(t, func() bool {
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, healthURL, bytes.NewBufferString("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 60*time.Second, 2*time.Second, "evm-single did not become healthy")

		t.Logf("evm-single started with base version: %s", baseEVMSingleVersion)
	})

	t.Run("create ethereum client", func(t *testing.T) {
		// connect to reth via external port for test host access
		rethNetworkInfo, err := rethNode.GetNetworkInfo(ctx)
		require.NoError(t, err)

		ethURL := fmt.Sprintf("http://127.0.0.1:%s", rethNetworkInfo.External.Ports.RPC)
		ethClient, err = ethclient.Dial(ethURL)
		require.NoError(t, err)

		// verify connection
		chainID, err := ethClient.ChainID(ctx)
		require.NoError(t, err)
		require.Equal(t, testChainID, chainID.String())

		t.Log("Ethereum client connected to Reth")
	})

	var preUpgradeTxHashes []common.Hash

	t.Run("pre-upgrade: submit transactions and verify state", func(t *testing.T) {
		// submit first transaction
		tx1 := evm.GetRandomTransaction(t, testPrivateKey, testToAddress, testChainID, testGasLimit, &txNonce)
		err := ethClient.SendTransaction(ctx, tx1)
		require.NoError(t, err)
		preUpgradeTxHashes = append(preUpgradeTxHashes, tx1.Hash())
		t.Logf("Submitted pre-upgrade tx 1: %s", tx1.Hash().Hex())

		// wait for tx1 to be mined
		require.Eventually(t, func() bool {
			receipt, err := ethClient.TransactionReceipt(ctx, tx1.Hash())
			if err != nil {
				return false
			}
			return receipt.Status == 1 // success
		}, 30*time.Second, time.Second, "transaction 1 was not mined")

		// submit second transaction
		tx2 := evm.GetRandomTransaction(t, testPrivateKey, testToAddress, testChainID, testGasLimit, &txNonce)
		err = ethClient.SendTransaction(ctx, tx2)
		require.NoError(t, err)
		preUpgradeTxHashes = append(preUpgradeTxHashes, tx2.Hash())
		t.Logf("Submitted pre-upgrade tx 2: %s", tx2.Hash().Hex())

		// wait for tx2 to be mined
		require.Eventually(t, func() bool {
			receipt, err := ethClient.TransactionReceipt(ctx, tx2.Hash())
			if err != nil {
				return false
			}
			return receipt.Status == 1
		}, 30*time.Second, time.Second, "transaction 2 was not mined")

		t.Logf("Pre-upgrade transactions verified: %d txs", len(preUpgradeTxHashes))
	})

	t.Run("perform upgrade", func(t *testing.T) {
		// verify node is running before upgrade
		err := evmSingleNode.ContainerLifecycle.Running(ctx)
		require.NoError(t, err, "evm-single should be running before upgrade")

		// stop the node
		err = evmSingleNode.StopContainer(ctx)
		require.NoError(t, err, "failed to stop evm-single")
		t.Log("Stopped evm-single node")

		// get upgrade version from environment variable (set by CI)
		upgradeVersion := strings.TrimSpace(os.Getenv("EV_NODE_IMAGE_TAG"))
		if upgradeVersion == "" {
			upgradeVersion = "local-dev" // fallback for local testing
		}
		t.Logf("Upgrading to version: %s", upgradeVersion)

		// update image version
		newImage := getEVMSingleImage(upgradeVersion)
		evmSingleNode.Image = newImage
		for _, node := range evmSingleChain.Nodes() {
			node.Image = newImage
		}

		// remove container but preserve volumes
		err = evmSingleNode.Remove(ctx, tastoratypes.WithPreserveVolumes())
		require.NoError(t, err, "failed to remove container with volume preservation")
		t.Log("Removed container with volume preservation")

		// recreate and start with new version
		err = evmSingleNode.Start(ctx)
		require.NoError(t, err, "failed to start upgraded node")

		// verify node is running after upgrade
		err = evmSingleNode.ContainerLifecycle.Running(ctx)
		require.NoError(t, err, "evm-single should be running after upgrade")

		// health check
		networkInfo, err := evmSingleNode.GetNetworkInfo(ctx)
		require.NoError(t, err)
		healthURL := fmt.Sprintf("http://0.0.0.0:%s/evnode.v1.HealthService/Livez", networkInfo.External.Ports.RPC)
		require.Eventually(t, func() bool {
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, healthURL, bytes.NewBufferString("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 60*time.Second, 2*time.Second, "upgraded evm-single did not become healthy")

		t.Logf("Upgraded node started successfully with version: %s", upgradeVersion)
	})

	t.Run("post-upgrade: verify old transactions persist", func(t *testing.T) {
		// verify all pre-upgrade transactions are still queryable
		for i, txHash := range preUpgradeTxHashes {
			receipt, err := ethClient.TransactionReceipt(ctx, txHash)
			require.NoError(t, err, "failed to query pre-upgrade tx %d: %s", i+1, txHash.Hex())
			require.Equal(t, uint64(1), receipt.Status, "pre-upgrade tx %d should have successful status", i+1)
			t.Logf("Pre-upgrade tx %d still accessible: %s (block %d)", i+1, txHash.Hex(), receipt.BlockNumber.Uint64())
		}

		t.Logf("All %d pre-upgrade transactions persisted after upgrade", len(preUpgradeTxHashes))
	})

	t.Run("post-upgrade: submit new transactions and verify", func(t *testing.T) {
		// submit new transaction after upgrade
		tx := evm.GetRandomTransaction(t, testPrivateKey, testToAddress, testChainID, testGasLimit, &txNonce)
		err := ethClient.SendTransaction(ctx, tx)
		require.NoError(t, err)
		t.Logf("Submitted post-upgrade tx: %s", tx.Hash().Hex())

		// wait for transaction to be mined
		require.Eventually(t, func() bool {
			receipt, err := ethClient.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return false
			}
			return receipt.Status == 1
		}, 30*time.Second, time.Second, "post-upgrade transaction was not mined")

		// get final receipt to check block number
		receipt, err := ethClient.TransactionReceipt(ctx, tx.Hash())
		require.NoError(t, err)
		t.Logf("Post-upgrade tx mined in block %d", receipt.BlockNumber.Uint64())

		// verify block production is continuing by checking block height increases
		initialBlock, err := ethClient.BlockNumber(ctx)
		require.NoError(t, err)

		time.Sleep(3 * time.Second) // wait for a few blocks

		laterBlock, err := ethClient.BlockNumber(ctx)
		require.NoError(t, err)
		require.Greater(t, laterBlock, initialBlock, "block height should increase after upgrade")

		t.Logf("Post-upgrade block production verified (block %d -> %d)", initialBlock, laterBlock)
	})

	t.Run("post-upgrade: verify account balances", func(t *testing.T) {
		// verify sender account still has balance
		senderAddr := common.HexToAddress("0xd143C405751162d0F96bEE2eB5eb9C61882a736E")
		balance, err := ethClient.BalanceAt(ctx, senderAddr, nil)
		require.NoError(t, err)
		require.True(t, balance.Cmp(big.NewInt(0)) > 0, "sender should have non-zero balance")
		t.Logf("Sender balance after upgrade: %s wei", balance.String())

		// verify recipient account balance
		recipientAddr := common.HexToAddress(testToAddress)
		recipientBalance, err := ethClient.BalanceAt(ctx, recipientAddr, nil)
		require.NoError(t, err)
		t.Logf("Recipient balance after upgrade: %s wei", recipientBalance.String())

		t.Log("Account balances verified after upgrade")
	})
}

// createCelestiaChain creates a Celestia chain for testing.
func createCelestiaChain(t *testing.T, dockerClient *dockerclient.Client, dockerNetworkID string) *cosmos.Chain {
	ctx := context.Background()
	encConfig := getEncodingConfig()

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
			"--grpc.address", "0.0.0.0:9090",
			"--rpc.grpc_laddr=tcp://0.0.0.0:9098",
			"--timeout-commit", "1s",
		).
		WithDockerClient(dockerClient).
		WithDockerNetworkID(dockerNetworkID).
		WithNode(cosmos.NewChainNodeConfigBuilder().
			WithNodeType(tastoratypes.NodeTypeValidator).
			Build()).
		Build(ctx)

	require.NoError(t, err)
	return chain
}

// createDANetwork creates a DA network with a bridge node.
func createDANetwork(t *testing.T, dockerClient *dockerclient.Client, dockerNetworkID string) *da.Network {
	ctx := context.Background()

	bridgeNodeConfig := da.NewNodeBuilder().
		WithNodeType(tastoratypes.BridgeNode).
		Build()

	daNetwork, err := da.NewNetworkBuilder(t).
		WithDockerClient(dockerClient).
		WithDockerNetworkID(dockerNetworkID).
		WithImage(container.NewImage("ghcr.io/celestiaorg/celestia-node", "v0.25.3", "10001:10001")).
		WithNode(bridgeNodeConfig).
		Build(ctx)
	require.NoError(t, err)

	return daNetwork
}

// getGenesisHashFromChain returns the genesis hash of the given chain.
func getGenesisHashFromChain(t *testing.T, ctx context.Context, chain *cosmos.Chain) string {
	node := chain.GetNodes()[0]
	c, err := node.GetRPCClient()
	require.NoError(t, err, "failed to get node client")

	first := int64(1)
	block, err := c.Block(ctx, &first)
	require.NoError(t, err, "failed to get block")

	genesisHash := block.Block.Header.Hash().String()
	require.NotEmpty(t, genesisHash, "genesis hash is empty")
	return genesisHash
}

// fundWallet transfers the specified amount of utia from the faucet wallet to the target wallet.
func fundWallet(t *testing.T, ctx context.Context, chain *cosmos.Chain, wallet *tastoratypes.Wallet, amount int64) {
	fromAddress, err := sdkacc.AddressFromWallet(chain.GetFaucetWallet())
	require.NoError(t, err)

	toAddress, err := sdk.AccAddressFromBech32(wallet.GetFormattedAddress())
	require.NoError(t, err)

	bankSend := banktypes.NewMsgSend(fromAddress, toAddress, sdk.NewCoins(sdk.NewCoin("utia", math.NewInt(amount))))
	_, err = chain.BroadcastMessages(ctx, chain.GetFaucetWallet(), bankSend)
	require.NoError(t, err)
}

// getEVMSingleImage returns the Docker image configuration for evm-single with the specified version.
func getEVMSingleImage(version string) container.Image {
	repo := strings.TrimSpace(os.Getenv("EV_NODE_IMAGE_REPO"))
	if repo == "" {
		repo = "evstack"
	}

	return container.NewImage(repo, version, "10001:10001")
}

// getEncodingConfig returns the encoding config for Celestia.
func getEncodingConfig() testutil.TestEncodingConfig {
	return testutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{})
}
