//go:build e2e

package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/container"
	tastoracosmos "github.com/celestiaorg/tastora/framework/docker/cosmos"
	tastorada "github.com/celestiaorg/tastora/framework/docker/dataavailability"
	"github.com/celestiaorg/tastora/framework/docker/evstack"
	"github.com/celestiaorg/tastora/framework/testutil/query"
	tastoratypes "github.com/celestiaorg/tastora/framework/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	coreda "github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/da/jsonrpc"
	seqcommon "github.com/evstack/ev-node/sequencers/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// TestEvNode_PostsToDA spins up celestia-app, a celestia bridge node and an
// EV Node (aggregator) via tastora, then verifies the EV Node actually posts
// data to DA by confirming blobs exist in the ev-data namespace via the DA
// JSON-RPC client.
func TestEvNode_PostsToDA(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration in short mode")
	}

	configurePrefixOnce.Do(configureCelestiaBech32Prefix)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	uniqueTestName := fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())

	dockerClient, networkID := tastoradocker.Setup(t)
	t.Cleanup(tastoradocker.Cleanup(t, dockerClient))

	encCfg := testutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, transfer.AppModuleBasic{}, gov.AppModuleBasic{})

	// 1) Start celestia-app chain
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

	// 2) Start celestia-node (bridge)
	daImage := container.Image{
		Repository: "ghcr.io/celestiaorg/celestia-node",
		Version:    "v0.28.4-mocha",
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
	sendAmt := sdk.NewInt64Coin(chain.Config.Denom, 5_000_000_000)

	bankSend := banktypes.NewMsgSend(
		faucet.Address,
		bridgeWallet.Address,
		sdk.NewCoins(sendAmt),
	)

	resp, err := chain.BroadcastMessages(ctx, faucet, bankSend)
	require.Zero(t, resp.Code, "broadcast response error should not be zero")
	require.NoErrorf(t, err, "fund bridge wallet")

	amnt, err := query.Balance(ctx, chain.GetNode().GrpcConn, bridgeWallet.FormattedAddress, chain.Config.Denom)
	require.NoError(t, err)
	require.NotZero(t, amnt.Int64(), "bridge wallet should have balance")

	bridgeNetInfo, err := bridge.GetNetworkInfo(ctx)
	require.NoError(t, err, "bridge network info")

	// 4) Start EV Node (aggregator) pointing at DA
	evNodeChain, err := evstack.NewChainBuilderWithTestName(t, uniqueTestName).
		WithChainID("evchain-test").
		WithBinaryName("testapp").
		WithAggregatorPassphrase("12345678").
		WithImage(getEvNodeImage()).
		WithDockerClient(dockerClient).
		WithDockerNetworkID(networkID).
		WithNode(evstack.NewNodeBuilder().WithAggregator(true).Build()).
		Build(ctx)
	require.NoError(t, err, "build ev node chain")

	evNode := evNodeChain.GetNodes()[0]
	require.NoError(t, evNode.Init(ctx), "ev node init")

	authToken, err := bridge.GetAuthToken()
	require.NoError(t, err, "bridge auth token")

	daAddress := fmt.Sprintf("http://%s", bridgeNetInfo.Internal.RPCAddress())
	headerNamespaceStr := "ev-header"
	dataNamespaceStr := "ev-data"
	dataNamespace := coreda.NamespaceFromString(dataNamespaceStr)

	require.NoError(t, evNode.Start(ctx,
		"--evnode.da.address", daAddress,
		"--evnode.da.auth_token", authToken,
		"--evnode.rpc.address", "0.0.0.0:7331",
		"--evnode.da.namespace", headerNamespaceStr,
		"--evnode.da.data_namespace", dataNamespaceStr,
		"--kv-endpoint", "0.0.0.0:8080",
	), "start ev node")

	evNetInfo, err := evNode.GetNetworkInfo(ctx)
	require.NoError(t, err, "ev node network info")
	httpAddr := evNetInfo.External.HTTPAddress()
	require.NotEmpty(t, httpAddr)
	parts := strings.Split(httpAddr, ":")
	require.Len(t, parts, 2)
	host, port := parts[0], parts[1]
	if host == "0.0.0.0" {
		host = "localhost"
	}
	cli, err := newHTTPClient(host, port)
	require.NoError(t, err)

	// 5) Submit a tx to ev-node to trigger block production + DA posting
	key, value := "da-key", "da-value"
	_, err = cli.Post(ctx, "/tx", key, value)
	require.NoError(t, err)

	waitFor(ctx, t, 30*time.Second, 2*time.Second, func() bool {
		res, err := cli.Get(ctx, "/kv?key="+key)
		if err != nil {
			return false
		}
		return string(res) == value
	}, "ev-node should serve the kv value")

	// 6) Assert data landed on DA via celestia-node blob RPC (namespace ev-data)
	daRPCAddr := fmt.Sprintf("http://127.0.0.1:%s", bridgeNetInfo.External.Ports.RPC)
	daClient, err := jsonrpc.NewClient(ctx, zerolog.Nop(), daRPCAddr, authToken, seqcommon.AbsoluteMaxBlobSize)
	require.NoError(t, err, "new da client")
	defer daClient.Close()

	validator := chain.GetNodes()[0].(*tastoracosmos.ChainNode)
	tmRPC, err := validator.GetRPCClient()
	require.NoError(t, err, "tm rpc client")

	var pfbHeight int64
	waitFor(ctx, t, time.Minute, 5*time.Second, func() bool {
		res, err := tmRPC.TxSearch(ctx, "message.action='/celestia.blob.v1.MsgPayForBlobs'", false, nil, nil, "desc")
		if err != nil || len(res.Txs) == 0 {
			return false
		}
		dataNSB64 := base64.StdEncoding.EncodeToString(dataNamespace.Bytes())
		for _, tx := range res.Txs {
			if tx.TxResult.Code != 0 {
				continue
			}
			for _, ev := range tx.TxResult.Events {
				if ev.Type != "celestia.blob.v1.EventPayForBlobs" {
					continue
				}
				for _, attr := range ev.Attributes {
					if string(attr.Key) == "namespaces" && strings.Contains(string(attr.Value), dataNSB64) {
						pfbHeight = tx.Height
						return true
					}
				}
			}
		}
		return false
	}, "expected a PayForBlobs tx on celestia-app")

	waitFor(ctx, t, time.Minute, 5*time.Second, func() bool {
		if pfbHeight == 0 {
			return false
		}
		for h := pfbHeight; h <= pfbHeight+10; h++ {
			ids, err := daClient.DA.GetIDs(ctx, uint64(h), dataNamespace.Bytes())
			if err != nil {
				t.Logf("GetIDs data height=%d err=%v", h, err)
				continue
			}
			if ids != nil && len(ids.IDs) > 0 {
				return true
			}
		}
		return false
	}, "expected blob in DA for namespace ev-data")
}

// newHTTPClient is a small helper to avoid importing the docker_e2e client.
func newHTTPClient(host, port string) (*httpClient, error) {
	return &httpClient{baseURL: fmt.Sprintf("http://%s:%s", host, port)}, nil
}

type httpClient struct {
	baseURL string
}

func (c *httpClient) Get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (c *httpClient) Post(ctx context.Context, path, key, value string) ([]byte, error) {
	body := strings.NewReader(fmt.Sprintf("%s=%s", key, value))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// getEvNodeImage resolves the EV Node image to use for the test.
// Falls back to EV_NODE_IMAGE_REPO:EV_NODE_IMAGE_TAG or evstack:local-dev.
func getEvNodeImage() container.Image {
	repo := strings.TrimSpace(getEnvDefault("EV_NODE_IMAGE_REPO", "evstack"))
	tag := strings.TrimSpace(getEnvDefault("EV_NODE_IMAGE_TAG", "local-dev"))
	return container.NewImage(repo, tag, "10001:10001")
}

func getEnvDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// waitFor polls condition until it returns true, context is cancelled, or timeout expires.
func waitFor(ctx context.Context, t *testing.T, timeout, interval time.Duration, condition func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("%s: context cancelled: %v", msg, ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Fatalf("%s: timed out after %v", msg, timeout)
			}
			if condition() {
				return
			}
		}
	}
}
