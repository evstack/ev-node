//go:build evm

package test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	"github.com/celestiaorg/tastora/framework/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
)

const proposerControlPrecompile = "0x000000000000000000000000000000000000F101"

func TestEngineExecutionReturnsNextProposerFromEvolveRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	admin := privateKeyAddress(t, TEST_PRIVATE_KEY)
	initialProposer := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	nextProposer := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	genesis := proposerControlGenesis(t, admin, initialProposer)

	dockerClient, dockerNetworkID := tastoradocker.Setup(t)
	rethNode := SetupTestRethNode(t, dockerClient, dockerNetworkID,
		withProposerControlImage(t),
		func(b *reth.NodeBuilder) {
			b.WithGenesis(genesis)
		},
	)

	ni, err := rethNode.GetNetworkInfo(ctx)
	require.NoError(t, err)
	ethURL := "http://127.0.0.1:" + ni.External.Ports.RPC
	engineURL := "http://127.0.0.1:" + ni.External.Ports.Engine
	genesisHash, err := rethNode.GenesisHash(ctx)
	require.NoError(t, err)
	requireRawNextProposer(t, ctx, ethURL, initialProposer)

	executionClient, err := evm.NewEngineExecutionClient(
		ethURL,
		engineURL,
		rethNode.JWTSecretHex(),
		common.HexToHash(genesisHash),
		common.Address{},
		dssync.MutexWrap(ds.NewMapDatastore()),
		false,
		zerolog.Nop(),
	)
	require.NoError(t, err)

	info, err := executionClient.GetExecutionInfo(ctx)
	requireProposerControlSupported(t, err)
	require.Equal(t, initialProposer.Bytes(), info.NextProposerAddress)

	genesisStateRoot, err := executionClient.InitChain(ctx, time.Now().UTC().Truncate(time.Second), 1, CHAIN_ID)
	require.NoError(t, err)

	tx := setNextProposerTx(t, TEST_PRIVATE_KEY, nextProposer)
	ethClient := createEthClient(t, ethURL)
	defer ethClient.Close()
	require.NoError(t, ethClient.SendTransaction(ctx, tx))

	payload, err := executionClient.GetTxs(ctx)
	require.NoError(t, err)
	require.Len(t, payload, 1)

	result, err := executionClient.ExecuteTxs(ctx, payload, 1, time.Now().UTC().Truncate(time.Second).Add(time.Second), genesisStateRoot)
	require.NoError(t, err)
	require.Equal(t, nextProposer.Bytes(), result.NextProposerAddress)

	info, err = executionClient.GetExecutionInfo(ctx)
	require.NoError(t, err)
	require.Equal(t, nextProposer.Bytes(), info.NextProposerAddress)
}

func TestTwoEngineNodesObserveRepeatedProposerRotation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	admin := privateKeyAddress(t, TEST_PRIVATE_KEY)
	proposerA := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	proposerB := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	genesis := proposerControlGenesis(t, admin, proposerA)

	dockerClient, dockerNetworkID := tastoradocker.Setup(t)
	nodeA := setupProposerControlNode(t, dockerClient, dockerNetworkID, genesis)
	nodeB := setupProposerControlNode(t, dockerClient, dockerNetworkID, genesis)

	clientA, ethURLA := newEngineClientForRethNode(t, ctx, nodeA)
	clientB, ethURLB := newEngineClientForRethNode(t, ctx, nodeB)

	requireRawNextProposer(t, ctx, ethURLA, proposerA)
	requireRawNextProposer(t, ctx, ethURLB, proposerA)

	genesisTime := time.Now().UTC().Truncate(time.Second)
	prevStateRootA, err := clientA.InitChain(ctx, genesisTime, 1, CHAIN_ID)
	require.NoError(t, err)
	prevStateRootB, err := clientB.InitChain(ctx, genesisTime, 1, CHAIN_ID)
	require.NoError(t, err)
	require.Equal(t, prevStateRootA, prevStateRootB)

	type executionNode struct {
		client *evm.EngineClient
		ethURL string
	}
	nodes := map[common.Hash]executionNode{
		proposerA: {client: clientA, ethURL: ethURLA},
		proposerB: {client: clientB, ethURL: ethURLB},
	}

	currentProposer := proposerA
	rotations := []common.Hash{proposerB, proposerA, proposerB, proposerA}
	baseTimestamp := genesisTime.Add(time.Second)
	for i, nextProposer := range rotations {
		blockHeight := uint64(i + 1)
		proposerNode := nodes[currentProposer]

		tx := setNextProposerTxWithNonce(t, TEST_PRIVATE_KEY, uint64(i), nextProposer)
		ethClient := createEthClient(t, proposerNode.ethURL)
		require.NoError(t, ethClient.SendTransaction(ctx, tx))
		ethClient.Close()

		payload, err := proposerNode.client.GetTxs(ctx)
		require.NoError(t, err)
		require.Len(t, payload, 1)

		blockTimestamp := baseTimestamp.Add(time.Duration(i) * time.Second)
		resultA, err := clientA.ExecuteTxs(ctx, payload, blockHeight, blockTimestamp, prevStateRootA)
		require.NoError(t, err)
		resultB, err := clientB.ExecuteTxs(ctx, payload, blockHeight, blockTimestamp, prevStateRootB)
		require.NoError(t, err)

		require.Equal(t, resultA.UpdatedStateRoot, resultB.UpdatedStateRoot)
		require.Equal(t, nextProposer.Bytes(), resultA.NextProposerAddress)
		require.Equal(t, nextProposer.Bytes(), resultB.NextProposerAddress)

		require.NoError(t, clientA.SetFinal(ctx, blockHeight))
		require.NoError(t, clientB.SetFinal(ctx, blockHeight))
		requireRawNextProposer(t, ctx, ethURLA, nextProposer)
		requireRawNextProposer(t, ctx, ethURLB, nextProposer)

		prevStateRootA = resultA.UpdatedStateRoot
		prevStateRootB = resultB.UpdatedStateRoot
		currentProposer = nextProposer
	}
}

func proposerControlGenesis(t *testing.T, admin common.Address, initialNextProposer common.Hash) []byte {
	t.Helper()

	var genesis map[string]any
	require.NoError(t, json.Unmarshal([]byte(reth.DefaultEvolveGenesisJSON()), &genesis))
	config, ok := genesis["config"].(map[string]any)
	require.True(t, ok)
	evolve, ok := config["evolve"].(map[string]any)
	if !ok {
		evolve = make(map[string]any)
		config["evolve"] = evolve
	}
	evolve["proposerControlAdmin"] = admin.Hex()
	evolve["proposerControlActivationHeight"] = float64(0)
	evolve["initialNextProposer"] = initialNextProposer.Hex()

	bz, err := json.MarshalIndent(genesis, "", "  ")
	require.NoError(t, err)
	return bz
}

func setupProposerControlNode(t *testing.T, dockerClient types.TastoraDockerClient, dockerNetworkID string, genesis []byte) *reth.Node {
	t.Helper()

	return SetupTestRethNode(t, dockerClient, dockerNetworkID,
		withProposerControlImage(t),
		func(b *reth.NodeBuilder) {
			b.WithGenesis(genesis)
		},
	)
}

func newEngineClientForRethNode(t *testing.T, ctx context.Context, node *reth.Node) (*evm.EngineClient, string) {
	t.Helper()

	ni, err := node.GetNetworkInfo(ctx)
	require.NoError(t, err)
	ethURL := "http://127.0.0.1:" + ni.External.Ports.RPC
	engineURL := "http://127.0.0.1:" + ni.External.Ports.Engine
	genesisHash, err := node.GenesisHash(ctx)
	require.NoError(t, err)

	client, err := evm.NewEngineExecutionClient(
		ethURL,
		engineURL,
		node.JWTSecretHex(),
		common.HexToHash(genesisHash),
		common.Address{},
		dssync.MutexWrap(ds.NewMapDatastore()),
		false,
		zerolog.Nop(),
	)
	require.NoError(t, err)
	return client, ethURL
}

func withProposerControlImage(t *testing.T) RethNodeOpt {
	t.Helper()

	repo := os.Getenv("EV_RETH_PROPOSER_IMAGE_REPO")
	tag := os.Getenv("EV_RETH_PROPOSER_IMAGE_TAG")
	if repo == "" && tag == "" {
		return nil
	}
	if repo == "" {
		repo = reth.DefaultImage().Repository
	}
	if tag == "" {
		t.Fatal("EV_RETH_PROPOSER_IMAGE_TAG must be set when overriding the ev-reth image")
	}
	return func(b *reth.NodeBuilder) {
		b.WithImage(container.NewImage(repo, tag, ""))
	}
}

func setNextProposerTx(t *testing.T, privateKeyHex string, nextProposer common.Hash) *ethTypes.Transaction {
	t.Helper()

	return setNextProposerTxWithNonce(t, privateKeyHex, 0, nextProposer)
}

func setNextProposerTxWithNonce(t *testing.T, privateKeyHex string, nonce uint64, nextProposer common.Hash) *ethTypes.Transaction {
	t.Helper()

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)
	chainID, ok := new(big.Int).SetString(CHAIN_ID, 10)
	require.True(t, ok)

	to := common.HexToAddress(proposerControlPrecompile)
	tx := ethTypes.NewTx(&ethTypes.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    big.NewInt(0),
		Gas:      100_000,
		GasPrice: big.NewInt(30_000_000_000),
		Data:     setNextProposerCalldata(nextProposer),
	})
	signed, err := ethTypes.SignTx(tx, ethTypes.NewEIP155Signer(chainID), privateKey)
	require.NoError(t, err)
	return signed
}

func setNextProposerCalldata(nextProposer common.Hash) []byte {
	selector := crypto.Keccak256([]byte("setNextProposer(bytes32)"))[:4]
	return append(selector, nextProposer.Bytes()...)
}

func privateKeyAddress(t *testing.T, privateKeyHex string) common.Address {
	t.Helper()

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)
	return crypto.PubkeyToAddress(privateKey.PublicKey)
}

func requireProposerControlSupported(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}

	var rpcErr rpc.Error
	if errors.As(err, &rpcErr) && rpcErr.ErrorCode() == -32601 {
		t.Skip("ev-reth image does not expose evolve_getNextProposer; set EV_RETH_PROPOSER_IMAGE_TAG to an image built from the proposer-control branch")
	}
	if strings.Contains(err.Error(), "proposerControl") {
		t.Skipf("ev-reth image does not support proposer-control chainspec fields: %v", err)
	}
	require.NoError(t, err)
}

func requireRawNextProposer(t *testing.T, ctx context.Context, ethURL string, want common.Hash) {
	t.Helper()

	client, err := rpc.Dial(ethURL)
	require.NoError(t, err)
	defer client.Close()

	var got common.Hash
	err = client.CallContext(ctx, &got, "evolve_getNextProposer", "latest")
	requireProposerControlSupported(t, err)
	require.Equal(t, want, got)
}
