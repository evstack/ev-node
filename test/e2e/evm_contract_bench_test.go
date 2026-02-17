//go:build evm

package e2e

import (
	"context"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// BenchmarkEvmContractRoundtrip measures the store → retrieve roundtrip latency
// against a real reth node with a pre-deployed contract.
//
// All transaction generation happens during setup. The timed loop exclusively
// measures: SendTransaction → wait for receipt → eth_call retrieve → verify.
//
// Run with (after building local-da and evm binaries):
//
//	PATH="/path/to/binaries:$PATH" go test -tags evm \
//	  -bench BenchmarkEvmContractRoundtrip -benchmem -benchtime=5x \
//	  -run='^$' -timeout=10m --evm-binary=/path/to/evm .
func BenchmarkEvmContractRoundtrip(b *testing.B) {
	workDir := b.TempDir()
	sequencerHome := filepath.Join(workDir, "evm-bench-sequencer")

	client, _, cleanup := setupTestSequencer(b, sequencerHome)
	defer cleanup()

	ctx := b.Context()
	privateKey, err := crypto.HexToECDSA(TestPrivateKey)
	require.NoError(b, err)
	chainID, ok := new(big.Int).SetString(DefaultChainID, 10)
	require.True(b, ok)
	signer := types.NewEIP155Signer(chainID)

	// Deploy contract once during setup.
	contractAddr, nonce := deployContract(b, ctx, client, StorageContractBytecode, 0, privateKey, chainID)

	// Pre-build signed store(42) transactions for all iterations.
	storeData, err := hexutil.Decode("0x000000000000000000000000000000000000000000000000000000000000002a")
	require.NoError(b, err)

	const maxIter = 1024
	signedTxs := make([]*types.Transaction, maxIter)
	for i := range maxIter {
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce + uint64(i),
			To:       &contractAddr,
			Value:    big.NewInt(0),
			Gas:      500000,
			GasPrice: big.NewInt(30000000000),
			Data:     storeData,
		})
		signedTxs[i], err = types.SignTx(tx, signer, privateKey)
		require.NoError(b, err)
	}

	expected := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000002a").Bytes()
	callMsg := ethereum.CallMsg{To: &contractAddr, Data: []byte{}}

	b.ResetTimer()
	b.ReportAllocs()

	var i int
	for b.Loop() {
		require.Less(b, i, maxIter, "increase maxIter for longer benchmark runs")

		// 1. Submit pre-signed store(42) transaction.
		err = client.SendTransaction(ctx, signedTxs[i])
		require.NoError(b, err)

		// 2. Wait for inclusion.
		waitForReceipt(b, ctx, client, signedTxs[i].Hash())

		// 3. Retrieve and verify.
		result, err := client.CallContract(ctx, callMsg, nil)
		require.NoError(b, err)
		require.Equal(b, expected, result, "retrieve() should return 42")

		i++
	}
}

// waitForReceipt polls for a transaction receipt until it is available.
func waitForReceipt(t testing.TB, ctx context.Context, client *ethclient.Client, txHash common.Hash) *types.Receipt {
	t.Helper()
	var receipt *types.Receipt
	var err error
	require.Eventually(t, func() bool {
		receipt, err = client.TransactionReceipt(ctx, txHash)
		return err == nil && receipt != nil
	}, 2*time.Second, 50*time.Millisecond, "transaction %s not included", txHash.Hex())
	return receipt
}
