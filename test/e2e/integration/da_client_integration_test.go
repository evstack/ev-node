package integration

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	libshare "github.com/celestiaorg/go-square/v3/share"
	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	tastoraconsts "github.com/celestiaorg/tastora/framework/docker/consts"
	"github.com/celestiaorg/tastora/framework/docker/container"
	tastoracosmos "github.com/celestiaorg/tastora/framework/docker/cosmos"
	tastorada "github.com/celestiaorg/tastora/framework/docker/dataavailability"
	"github.com/celestiaorg/tastora/framework/testutil/wait"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govmodule "github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	blobrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
)

var configurePrefixOnce sync.Once

func configureCelestiaBech32Prefix() {
	conf := sdk.GetConfig()
	conf.SetBech32PrefixForAccount("celestia", "celestiapub")
}

func TestClient_SubmitAndGetBlobAgainstRealNode(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration in short mode")
	}

	configurePrefixOnce.Do(configureCelestiaBech32Prefix)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	uniqueTestName := fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())

	dockerClient, networkID := tastoradocker.Setup(t)
	t.Cleanup(tastoradocker.Cleanup(t, dockerClient))

	encCfg := testutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, transfer.AppModuleBasic{}, govmodule.AppModuleBasic{})

	chainImage := container.Image{
		Repository: "ghcr.io/celestiaorg/celestia-app",
		Version:    "v5.0.10",
		UIDGID:     "10001:10001",
	}

	chainBuilder := tastoracosmos.NewChainBuilderWithTestName(t, uniqueTestName).
		WithDockerClient(dockerClient).
		WithDockerNetworkID(networkID).
		WithImage(chainImage).
		WithEncodingConfig(&encCfg).
		WithAdditionalStartArgs(
			"--force-no-bbr",
			"--grpc.enable",
			"--grpc.address", "0.0.0.0:9090",
			"--rpc.grpc_laddr=tcp://0.0.0.0:9098",
			"--rpc.laddr=tcp://0.0.0.0:26657",
			"--timeout-commit", "1s",
			"--minimum-gas-prices", "0utia",
		).
		WithNode(tastoracosmos.NewChainNodeConfigBuilder().Build())

	chain, err := chainBuilder.Build(ctx)
	require.NoError(t, err, "build celestia-app chain")
	require.NoError(t, chain.Start(ctx), "start celestia-app chain")

	chainID := chain.GetChainID()

	genesisHash, err := fetchGenesisHash(ctx, chain)
	require.NoError(t, err, "genesis hash")

	chainNetInfo, err := chain.GetNodes()[0].GetNetworkInfo(ctx)
	require.NoError(t, err, "chain network info")
	coreHost := chainNetInfo.Internal.Hostname

	daImage := container.Image{
		Repository: "ghcr.io/celestiaorg/celestia-node",
		Version:    "v0.26.4",
		UIDGID:     "10001:10001",
	}

	daNetwork, err := tastorada.NewNetworkBuilderWithTestName(t, uniqueTestName).
		WithDockerClient(dockerClient).
		WithDockerNetworkID(networkID).
		WithImage(daImage).
		WithNodes(tastorada.NewNodeBuilder().WithNodeType(tastoratypes.BridgeNode).Build()).
		Build(ctx)
	require.NoError(t, err, "build da network")

	bridge := daNetwork.GetBridgeNodes()[0]
	err = bridge.Start(ctx,
		tastorada.WithChainID(chainID),
		tastorada.WithAdditionalStartArguments(
			"--p2p.network", chainID,
			"--core.ip", coreHost,
			"--rpc.addr", "0.0.0.0",
		),
		tastorada.WithEnvironmentVariables(map[string]string{
			"CELESTIA_CUSTOM": tastoratypes.BuildCelestiaCustomEnvVar(chainID, genesisHash, ""),
			"P2P_NETWORK":     chainID,
		}),
	)
	require.NoError(t, err, "start bridge node")

	bridgeWallet, err := bridge.GetWallet()
	require.NoError(t, err, "bridge wallet")

	faucet := chain.GetFaucetWallet()
	require.NotNil(t, faucet, "faucet wallet")

	validatorNode := chain.GetNodes()[0].(*tastoracosmos.ChainNode)

	// wait until the node is serving RPC before funding
	err = wait.ForCondition(ctx, 2*time.Minute, time.Second, func() (bool, error) {
		c, err := validatorNode.GetRPCClient()
		if err != nil {
			return false, nil
		}
		if _, err := c.Status(ctx); err != nil {
			return false, nil
		}
		h, err := validatorNode.Height(ctx)
		if err != nil {
			return false, nil
		}
		return h >= 3, nil
	})
	require.NoError(t, err, "validator RPC ready")

	// Wait for a few blocks to ensure app is live
	err = wait.ForCondition(ctx, 1*time.Minute, time.Second, func() (bool, error) {
		h, err := validatorNode.Height(ctx)
		if err != nil {
			return false, nil
		}
		return h >= 2, nil
	})
	require.NoError(t, err, "chain progressed to height>=2")

	// fund the bridge wallet via CLI to avoid BroadcastMessages JSON-RPC decoding bug
	faucetKey := tastoraconsts.FaucetAccountKeyName
	sendAmt := sdk.NewInt64Coin(chain.Config.Denom, 5_000_000_000)
	rpcNode := fmt.Sprintf("tcp://%s:26657", coreHost)

	cmd := []string{
		validatorNode.BinaryName,
		"tx", "bank", "send",
		faucetKey,
		bridgeWallet.FormattedAddress,
		sendAmt.String(), // e.g. 5000000000utia
		"--chain-id", chainID,
		"--home", validatorNode.HomeDir(),
		"--keyring-backend", "test",
		"--node", rpcNode,
		"--fees", fmt.Sprintf("1000%s", chain.Config.Denom),
		"--broadcast-mode", "sync",
		"--yes",
	}
	stdout, stderr, err := validatorNode.Exec(ctx, cmd, nil)
	require.NoErrorf(t, err, "fund bridge wallet via CLI: %s", string(stderr))
	require.Contains(t, string(stdout), "code: 0", "bank send succeeded")

	bankQuery := banktypes.NewQueryClient(chain.GetNode().GrpcConn)
	err = wait.ForCondition(ctx, 2*time.Minute, time.Second, func() (bool, error) {
		bal, err := bankQuery.Balance(ctx, &banktypes.QueryBalanceRequest{
			Address: bridgeWallet.FormattedAddress,
			Denom:   chain.Config.Denom,
		})
		if err != nil {
			return false, nil
		}
		return bal.Balance != nil && bal.Balance.Amount.GT(sdkmath.NewInt(0)), nil
	})
	require.NoError(t, err, "bridge wallet funded")

	bridgeNetInfo, err := bridge.GetNetworkInfo(ctx)
	require.NoError(t, err, "bridge network info")

	rpcAddr := fmt.Sprintf("http://127.0.0.1:%s", bridgeNetInfo.External.Ports.RPC)

	// wait for celestia-node RPC port to become reachable
	err = wait.ForCondition(ctx, 2*time.Minute, time.Second, func() (bool, error) {
		hostPort := fmt.Sprintf("127.0.0.1:%s", bridgeNetInfo.External.Ports.RPC)
		conn, err := net.DialTimeout("tcp", hostPort, 2*time.Second)
		if err != nil {
			return false, nil
		}
		_ = conn.Close()
		return true, nil
	})
	require.NoError(t, err, "bridge RPC reachable")

	client, err := blobrpc.NewClient(ctx, rpcAddr, "", "")
	require.NoError(t, err, "new da client")
	t.Cleanup(client.Close)

	ns := libshare.MustNewV0Namespace([]byte("evnode"))
	blb, err := blobrpc.NewBlobV0(ns, []byte("integration-data"))
	require.NoError(t, err, "build blob")

	var height uint64
	for attempt := 0; attempt < 3; attempt++ {
		height, err = client.Blob.Submit(ctx, []*blobrpc.Blob{blb}, &blobrpc.SubmitOptions{
			GasPrice:      0.1,
			IsGasPriceSet: true,
			Gas:           2_000_000,
			KeyName:       bridgeWallet.KeyName,
		})
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	require.NoError(t, err, "submit blob")
	require.Greater(t, height, uint64(0), "height returned")

	err = wait.ForCondition(ctx, 3*time.Minute, 2*time.Second, func() (bool, error) {
		got, err := client.Blob.Get(ctx, height, ns, blb.Commitment)
		if err != nil || got == nil {
			return false, nil
		}
		return bytes.Equal(got.Data(), blb.Data()), nil
	})
	require.NoError(t, err, "blob retrievable from celestia node")
}

func fetchGenesisHash(ctx context.Context, chain *tastoracosmos.Chain) (string, error) {
	node := chain.GetNodes()[0]
	rpc, err := node.GetRPCClient()
	if err != nil {
		return "", err
	}

	first := int64(1)
	block, err := rpc.Block(ctx, &first)
	if err != nil {
		return "", err
	}

	return block.Block.Header.Hash().String(), nil
}
