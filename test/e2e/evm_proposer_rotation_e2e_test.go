//go:build evm

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/container"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	evmtest "github.com/evstack/ev-node/execution/evm/test"
	"github.com/evstack/ev-node/pkg/rpc/client"
	filesigner "github.com/evstack/ev-node/pkg/signer/file"
)

const proposerControlPrecompileAddress = "0x000000000000000000000000000000000000F101"

func TestEvmFullNodeCanBecomeProposerAfterExecutionRotation(t *testing.T) {
	if os.Getenv("EV_RETH_PROPOSER_IMAGE_TAG") == "" {
		t.Skip("set EV_RETH_PROPOSER_IMAGE_TAG to an ev-reth image built with proposer-control support")
	}

	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "sequencer")
	fullNodeHome := filepath.Join(workDir, "fullnode")
	sut := NewSystemUnderTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dockerClient, networkID := tastoradocker.Setup(t)
	admin := privateKeyAddress(t, TestPrivateKey)
	evmGenesis := proposerControlGenesis(t, admin, common.Hash{})
	env := SetupCommonEVMEnv(t, sut, dockerClient, networkID,
		WithFullNode(),
		WithRethOpts(
			withProposerControlGenesis(evmGenesis),
			withProposerControlImage(t),
		),
	)

	SetupSequencerNode(t, sut, sequencerHome, env.SequencerJWT, env.GenesisHash, env.Endpoints)
	sequencerAddress := evNodeSignerAddress(t, sequencerHome)

	fullNodePassphraseFile := initNodeWithSigner(t, sut, fullNodeHome)
	MustCopyFile(t,
		filepath.Join(sequencerHome, "config", "genesis.json"),
		filepath.Join(fullNodeHome, "config", "genesis.json"),
	)
	fullNodeAddress := evNodeSignerAddress(t, fullNodeHome)
	require.NotEqual(t, sequencerAddress, fullNodeAddress)

	sequencerP2PAddress := getNodeP2PAddress(t, sut, sequencerHome, env.Endpoints.RollkitRPCPort)
	fullNodeJWTSecretFile := createJWTSecretFile(t, fullNodeHome, env.FullNodeJWT)
	fullNodeProcess := startFullNodeProcess(t, sut, fullNodeHome, fullNodeJWTSecretFile, fullNodePassphraseFile, env.GenesisHash, sequencerP2PAddress, env.Endpoints)

	seqClient, err := ethclient.Dial(env.Endpoints.GetSequencerEthURL())
	require.NoError(t, err)
	defer seqClient.Close()

	fnClient, err := ethclient.Dial(env.Endpoints.GetFullNodeEthURL())
	require.NoError(t, err)
	defer fnClient.Close()

	waitForEvNodeNextProposer(t, ctx, env.Endpoints.GetRollkitRPCAddress(), sequencerAddress, 20*time.Second)
	waitForEvNodeNextProposer(t, ctx, env.Endpoints.GetFullNodeRPCAddress(), sequencerAddress, 20*time.Second)

	nextProposer := common.BytesToHash(fullNodeAddress)
	tx := setNextProposerTx(t, TestPrivateKey, nextProposer, 0)
	require.NoError(t, seqClient.SendTransaction(ctx, tx))
	waitForEVMTransaction(t, ctx, seqClient, tx.Hash(), 30*time.Second)

	waitForEvNodeNextProposer(t, ctx, env.Endpoints.GetRollkitRPCAddress(), fullNodeAddress, 30*time.Second)
	waitForEvNodeNextProposer(t, ctx, env.Endpoints.GetFullNodeRPCAddress(), fullNodeAddress, 30*time.Second)
	requireRawNextProposer(t, ctx, env.Endpoints.GetSequencerEthURL(), nextProposer)
	requireRawNextProposer(t, ctx, env.Endpoints.GetFullNodeEthURL(), nextProposer)

	heightBeforeSwitch := currentEvNodeHeight(t, ctx, env.Endpoints.GetFullNodeRPCAddress())

	require.NoError(t, fullNodeProcess.Signal(syscall.SIGTERM))
	time.Sleep(time.Second)

	fullNodeProposerProcess := startAggregatorProcess(
		t,
		sut,
		fullNodeHome,
		fullNodeJWTSecretFile,
		fullNodePassphraseFile,
		env.GenesisHash,
		env.Endpoints.GetFullNodeRPCListen(),
		env.Endpoints.GetFullNodeP2PAddress(),
		env.Endpoints.GetFullNodeEngineURL(),
		env.Endpoints.GetFullNodeEthURL(),
		env.Endpoints,
	)
	t.Cleanup(func() {
		_ = fullNodeProposerProcess.Signal(syscall.SIGTERM)
	})

	waitForEvNodeHeightAbove(t, ctx, env.Endpoints.GetFullNodeRPCAddress(), heightBeforeSwitch, 30*time.Second)
	latestHeight := currentEvNodeHeight(t, ctx, env.Endpoints.GetFullNodeRPCAddress())
	latestBlock, err := client.NewClient(env.Endpoints.GetFullNodeRPCAddress()).GetBlockByHeight(ctx, latestHeight)
	require.NoError(t, err)
	require.Equal(t, fullNodeAddress, latestBlock.Block.Header.Header.ProposerAddress)
}

func initNodeWithSigner(t *testing.T, sut *SystemUnderTest, home string) string {
	t.Helper()

	passphraseFile := createPassphraseFile(t, home)
	output, err := sut.RunCmd(evmSingleBinaryPath,
		"init",
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--home", home,
	)
	require.NoError(t, err, "failed to init node with signer", output)
	return passphraseFile
}

func evNodeSignerAddress(t *testing.T, home string) []byte {
	t.Helper()

	signer, err := filesigner.LoadFileSystemSigner(filepath.Join(home, "config"), []byte(TestPassphrase))
	require.NoError(t, err)
	addr, err := signer.GetAddress()
	require.NoError(t, err)
	require.Len(t, addr, 32)
	return append([]byte(nil), addr...)
}

func startFullNodeProcess(
	t *testing.T,
	sut *SystemUnderTest,
	fullNodeHome string,
	fullNodeJWTSecretFile string,
	passphraseFile string,
	genesisHash string,
	sequencerP2PAddress string,
	endpoints *TestEndpoints,
) *os.Process {
	t.Helper()

	process := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evnode.log.level", "debug",
		"--evnode.log.format", "json",
		"--home", fullNodeHome,
		"--evm.jwt-secret-file", fullNodeJWTSecretFile,
		"--evnode.node.aggregator=false",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--evm.genesis-hash", genesisHash,
		"--evnode.p2p.peers", sequencerP2PAddress,
		"--evm.engine-url", endpoints.GetFullNodeEngineURL(),
		"--evm.eth-url", endpoints.GetFullNodeEthURL(),
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.batching_strategy", "immediate",
		"--evnode.rpc.address", endpoints.GetFullNodeRPCListen(),
		"--evnode.p2p.listen_address", endpoints.GetFullNodeP2PAddress(),
	)
	sut.AwaitNodeLive(t, endpoints.GetFullNodeRPCAddress(), NodeStartupTimeout)
	return process
}

func startAggregatorProcess(
	t *testing.T,
	sut *SystemUnderTest,
	home string,
	jwtSecretFile string,
	passphraseFile string,
	genesisHash string,
	rpcListen string,
	p2pListen string,
	engineURL string,
	ethURL string,
	endpoints *TestEndpoints,
) *os.Process {
	t.Helper()

	process := sut.ExecCmd(evmSingleBinaryPath,
		"start",
		"--evnode.log.level", "debug",
		"--evnode.log.format", "json",
		"--home", home,
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", genesisHash,
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.node.aggregator=true",
		"--evnode.signer.passphrase_file", passphraseFile,
		"--evm.engine-url", engineURL,
		"--evm.eth-url", ethURL,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.batching_strategy", "immediate",
		"--evnode.rpc.address", rpcListen,
		"--evnode.p2p.listen_address", p2pListen,
	)
	sut.AwaitNodeUp(t, "http://"+rpcListen, NodeStartupTimeout)
	return process
}

func waitForEvNodeNextProposer(t *testing.T, ctx context.Context, rpcAddr string, want []byte, timeout time.Duration) {
	t.Helper()

	evClient := client.NewClient(rpcAddr)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		state, err := evClient.GetState(ctx)
		require.NoError(t, err)
		require.True(t, bytes.Equal(want, state.NextProposerAddress), "got %x, want %x", state.NextProposerAddress, want)
	}, timeout, 500*time.Millisecond)
}

func currentEvNodeHeight(t *testing.T, ctx context.Context, rpcAddr string) uint64 {
	t.Helper()

	state, err := client.NewClient(rpcAddr).GetState(ctx)
	require.NoError(t, err)
	return state.LastBlockHeight
}

func waitForEvNodeHeightAbove(t *testing.T, ctx context.Context, rpcAddr string, height uint64, timeout time.Duration) {
	t.Helper()

	evClient := client.NewClient(rpcAddr)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		state, err := evClient.GetState(ctx)
		require.NoError(t, err)
		require.Greater(t, state.LastBlockHeight, height)
	}, timeout, 500*time.Millisecond)
}

func waitForEVMTransaction(t *testing.T, ctx context.Context, ethClient *ethclient.Client, hash common.Hash, timeout time.Duration) {
	t.Helper()

	require.Eventually(t, func() bool {
		receipt, err := ethClient.TransactionReceipt(ctx, hash)
		return err == nil && receipt != nil && receipt.Status == ethTypes.ReceiptStatusSuccessful
	}, timeout, 500*time.Millisecond)
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

func withProposerControlGenesis(genesis []byte) evmtest.RethNodeOpt {
	return func(b *reth.NodeBuilder) {
		b.WithGenesis(genesis)
	}
}

func withProposerControlImage(t *testing.T) evmtest.RethNodeOpt {
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

func setNextProposerTx(t *testing.T, privateKeyHex string, nextProposer common.Hash, nonce uint64) *ethTypes.Transaction {
	t.Helper()

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)
	chainID, ok := new(big.Int).SetString(DefaultChainID, 10)
	require.True(t, ok)

	to := common.HexToAddress(proposerControlPrecompileAddress)
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

func requireRawNextProposer(t *testing.T, ctx context.Context, ethURL string, want common.Hash) {
	t.Helper()

	rpcClient, err := rpc.Dial(ethURL)
	require.NoError(t, err)
	defer rpcClient.Close()

	var got common.Hash
	require.NoError(t, rpcClient.CallContext(ctx, &got, "evolve_getNextProposer", "latest"))
	require.Equal(t, want, got)
}
