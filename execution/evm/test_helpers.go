package evm

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"math/big"
	"math/rand"
	"testing"
)

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
