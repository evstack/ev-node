//go:build evm
// +build evm

package evm

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

    dockerfw "github.com/celestiaorg/tastora/framework/docker"
    rethfw "github.com/celestiaorg/tastora/framework/docker/evstack/reth"
    dockerclient "github.com/moby/moby/client"
)

// Dynamic endpoints captured when starting reth via Tastora
var (
    sequencerEthURL    string
    sequencerEngineURL string
    fullNodeEthURL     string
    fullNodeEngineURL  string
    dockerCli          *dockerclient.Client
    dockerNetID        string
)

// CurrentSequencerEndpoints returns dynamic Eth/Engine URLs if available
func CurrentSequencerEndpoints() (ethURL, engineURL string) { return sequencerEthURL, sequencerEngineURL }

// CurrentFullNodeEndpoints returns dynamic Eth/Engine URLs if available
func CurrentFullNodeEndpoints() (ethURL, engineURL string) { return fullNodeEthURL, fullNodeEngineURL }

// generateJWTSecret generates a random 32-byte JWT secret and returns it as a hex string.
func generateJWTSecret() (string, error) {
	jwtSecret := make([]byte, 32)
	_, err := rand.Read(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(jwtSecret), nil
}

// SetupTestRethEngine sets up a Reth engine test environment using Docker Compose, writes a JWT secret file, and returns the secret. It also registers cleanup for resources.
func SetupTestRethEngine(t *testing.T, dockerPath, jwtFilename string) (string, string, string) {
    t.Helper()
    // Start a single reth via Tastora to serve as the execution engine paired with the sequencer evm-single
    ctx := context.Background()

    // Setup Docker client/network once per test
    if dockerCli == nil || dockerNetID == "" {
        cli, netID := dockerfw.DockerSetup(t)
        dockerCli, dockerNetID = cli, netID
    }

    // Load genesis used by previous compose path
    dockerAbsPath, err := filepath.Abs(dockerPath)
    require.NoError(t, err)
    genesisPath := filepath.Join(dockerAbsPath, "chain", "genesis.json")
    genesisBz, err := os.ReadFile(genesisPath)
    require.NoError(t, err)

    n, err := rethfw.NewNodeBuilder(t).
        WithDockerClient(dockerCli).
        WithDockerNetworkID(dockerNetID).
        WithGenesis(genesisBz).
        Build(ctx)
    require.NoError(t, err)
    require.NoError(t, n.Start(ctx))

    ni, err := n.GetNetworkInfo(ctx)
    require.NoError(t, err)
    sequencerEthURL = "http://127.0.0.1:" + ni.External.Ports.RPC
    sequencerEngineURL = "http://127.0.0.1:" + ni.External.Ports.Engine
    jwtSecret := n.JWTSecretHex()

    require.NoError(t, waitForRethContainer(t, jwtSecret, sequencerEthURL, sequencerEngineURL))
    return jwtSecret, sequencerEthURL, sequencerEngineURL
}

// waitForRethContainer waits for the Reth container to be ready by polling the provided endpoints with JWT authentication.
func waitForRethContainer(t *testing.T, jwtSecret, ethURL, engineURL string) error {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout waiting for reth container to be ready")
		default:
			rpcReq := strings.NewReader(`{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}`)
			resp, err := client.Post(ethURL, "application/json", rpcReq)
			if err == nil {
				if err := resp.Body.Close(); err != nil {
					return fmt.Errorf("failed to close response body: %w", err)
				}
				if resp.StatusCode == http.StatusOK {
					req, err := http.NewRequest("POST", engineURL, strings.NewReader(`{"jsonrpc":"2.0","method":"engine_getClientVersionV1","params":[],"id":1}`))
					if err != nil {
						return err
					}
					req.Header.Set("Content-Type", "application/json")
					secret, err := decodeSecret(jwtSecret)
					if err != nil {
						return err
					}
					authToken, err := getAuthToken(secret)
					if err != nil {
						return err
					}
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
					resp, err := client.Do(req)
					if err == nil {
						if err := resp.Body.Close(); err != nil {
							return fmt.Errorf("failed to close response body: %w", err)
						}
						if resp.StatusCode == http.StatusOK {
							return nil
						}
					}
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Transaction Helpers

// GetRandomTransaction creates and signs a random Ethereum legacy transaction using the provided private key, recipient, chain ID, gas limit, and nonce.
func GetRandomTransaction(t *testing.T, privateKeyHex, toAddressHex, chainID string, gasLimit uint64, lastNonce *uint64) *types.Transaction {
	t.Helper()
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)
	chainId, ok := new(big.Int).SetString(chainID, 10)
	require.True(t, ok)
	txValue := big.NewInt(1000000000000000000)
	gasPrice := big.NewInt(30000000000)
	toAddress := common.HexToAddress(toAddressHex)
	data := make([]byte, 16)
	_, err = rand.Read(data)
	require.NoError(t, err)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    *lastNonce,
		To:       &toAddress,
		Value:    txValue,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	*lastNonce++
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), privateKey)
	require.NoError(t, err)
	return signedTx
}

// SubmitTransaction submits a signed Ethereum transaction to the local node at http://localhost:8545.
func SubmitTransaction(t *testing.T, tx *types.Transaction) {
    t.Helper()
    url := sequencerEthURL
    if url == "" { url = "http://localhost:8545" }
    rpcClient, err := ethclient.Dial(url)
    require.NoError(t, err)
    defer rpcClient.Close()

    err = rpcClient.SendTransaction(context.Background(), tx)
    require.NoError(t, err)
}

// CheckTxIncluded checks if a transaction with the given hash was included in a block and succeeded.
func CheckTxIncluded(t *testing.T, txHash common.Hash) bool {
    t.Helper()
    url := sequencerEthURL
    if url == "" { url = "http://localhost:8545" }
    rpcClient, err := ethclient.Dial(url)
    if err != nil { return false }
    defer rpcClient.Close()
    receipt, err := rpcClient.TransactionReceipt(context.Background(), txHash)
    return err == nil && receipt != nil && receipt.Status == 1
}

// GetGenesisHash retrieves the hash of the genesis block from the local Ethereum node.
func GetGenesisHash(t *testing.T) string {
    t.Helper()
    client := &http.Client{Timeout: 2 * time.Second}
    data := []byte(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}`)
    url := sequencerEthURL
    if url == "" { url = "http://localhost:8545" }
    resp, err := client.Post(url, "application/json", bytes.NewReader(data))
    require.NoError(t, err)
    defer resp.Body.Close()
    var result struct {
        Result struct {
            Hash string `json:"hash"`
        } `json:"result"`
    }
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
    require.NotEmpty(t, result.Result.Hash)
    return result.Result.Hash
}

// SetupTestRethEngineFullNode sets up a Reth full node test environment using Docker Compose with the full node configuration.
// This function is specifically for setting up full nodes that connect to ports 8555/8561.
func SetupTestRethEngineFullNode(t *testing.T, dockerPath, jwtFilename string) (string, string, string) {
    t.Helper()
    ctx := context.Background()
    // Reuse docker client/network from the first call
    if dockerCli == nil || dockerNetID == "" {
        cli, netID := dockerfw.DockerSetup(t)
        dockerCli, dockerNetID = cli, netID
    }
    dockerAbsPath, err := filepath.Abs(dockerPath)
    require.NoError(t, err)
    genesisPath := filepath.Join(dockerAbsPath, "chain", "genesis.json")
    genesisBz, err := os.ReadFile(genesisPath)
    require.NoError(t, err)

    // Use a different test name suffix to avoid container name collisions
    n, err := rethfw.NewNodeBuilder(t).
        WithTestName(t.Name()+"-full").
        WithDockerClient(dockerCli).
        WithDockerNetworkID(dockerNetID).
        WithGenesis(genesisBz).
        Build(ctx)
    require.NoError(t, err)
    require.NoError(t, n.Start(ctx))

    ni, err := n.GetNetworkInfo(ctx)
    require.NoError(t, err)
    fullNodeEthURL = "http://127.0.0.1:" + ni.External.Ports.RPC
    fullNodeEngineURL = "http://127.0.0.1:" + ni.External.Ports.Engine
    jwtSecret := n.JWTSecretHex()

    require.NoError(t, waitForRethContainer(t, jwtSecret, fullNodeEthURL, fullNodeEngineURL))
    return jwtSecret, fullNodeEthURL, fullNodeEngineURL
}
