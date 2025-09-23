//go:build evm
// +build evm

package evm

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/tastora/framework/docker"
	"github.com/celestiaorg/tastora/framework/docker/evstack/reth"
	dockerclient "github.com/moby/moby/client"
)

// Test-scoped Docker client/network mapping to avoid conflicts between tests
var (
	dockerClients  = make(map[string]*dockerclient.Client)
	dockerNetworks = make(map[string]string)
	dockerMutex    sync.RWMutex
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// getTestScopedDockerSetup returns a Docker client and network ID that are scoped to the specific test.
func getTestScopedDockerSetup(t *testing.T) (*dockerclient.Client, string) {
	t.Helper()

	testKey := t.Name()
	dockerMutex.Lock()
	defer dockerMutex.Unlock()

	dockerCli, exists := dockerClients[testKey]
	if !exists {
		cli, netID := docker.DockerSetup(t)
		dockerClients[testKey] = cli
		dockerNetworks[testKey] = netID
		dockerCli = cli
	}
	dockerNetID := dockerNetworks[testKey]

	return dockerCli, dockerNetID
}

// SetupTestRethNode creates a single Reth node for testing purposes.
func SetupTestRethNode(t *testing.T) *reth.Node {
	t.Helper()
	ctx := context.Background()

	dockerCli, dockerNetID := getTestScopedDockerSetup(t)

	n, err := reth.NewNodeBuilderWithTestName(t, fmt.Sprintf("%s-%s", t.Name(), randomString(6))).
		WithDockerClient(dockerCli).
		WithDockerNetworkID(dockerNetID).
		WithGenesis([]byte(reth.DefaultEvolveGenesisJSON())).
		Build(ctx)
	t.Cleanup(func() {
		_ = n.Remove(context.Background())
	})

	require.NoError(t, err)
	require.NoError(t, n.Start(ctx))

	ni, err := n.GetNetworkInfo(ctx)
	require.NoError(t, err)
	ethURL := "http://127.0.0.1:" + ni.External.Ports.RPC
	engineURL := "http://127.0.0.1:" + ni.External.Ports.Engine
	jwtSecret := n.JWTSecretHex()

	require.NoError(t, waitForRethContainer(t, jwtSecret, ethURL, engineURL))
	return n
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

// CheckTxIncluded checks if a transaction with the given hash was included in a block and succeeded.
func CheckTxIncluded(client *ethclient.Client, txHash common.Hash) bool {
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	return err == nil && receipt != nil && receipt.Status == 1
}
