package evm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ethRPCClient struct {
	client *ethclient.Client
}

func NewEthRPCClient(client *ethclient.Client) EthRPCClient {
	return &ethRPCClient{client: client}
}

func (e *ethRPCClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return e.client.HeaderByNumber(ctx, number)
}

// GetTxs works only on custom execution clients exposing txpoolExt_getTxs.
// Standard Ethereum nodes do not support this RPC method.
func (e *ethRPCClient) GetTxs(ctx context.Context) ([]string, error) {
	var result []string
	err := e.client.Client().CallContext(ctx, &result, "txpoolExt_getTxs")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *ethRPCClient) GetNextProposer(ctx context.Context, number *big.Int) (common.Hash, error) {
	var result common.Hash
	err := e.client.Client().CallContext(ctx, &result, "evolve_getNextProposer", blockNumberArg(number))
	if err != nil {
		return common.Hash{}, err
	}
	return result, nil
}

func blockNumberArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}
