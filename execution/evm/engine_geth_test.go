package evm

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGenesis() *core.Genesis {
	testKey, _ := crypto.GenerateKey()
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	return &core.Genesis{
		Config:     params.AllDevChainProtocolChanges,
		Difficulty: big.NewInt(0),
		GasLimit:   30_000_000,
		Alloc: types.GenesisAlloc{
			testAddr: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))},
		},
		Timestamp: uint64(time.Now().Unix()),
	}
}

func testDatastore() ds.Batching {
	return sync.MutexWrap(ds.NewMapDatastore())
}

func TestNewEngineExecutionClientWithGeth(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotEqual(t, common.Hash{}, client.genesisHash)
	assert.Equal(t, feeRecipient, client.feeRecipient)
}

func TestNewEngineExecutionClientWithGeth_NilDB(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	_, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, nil, "", logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db is required")
}

func TestNewEngineExecutionClientWithGeth_NilGenesis(t *testing.T) {
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	_, err := NewEngineExecutionClientWithGeth(nil, feeRecipient, db, "", logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "genesis configuration is required")
}

func TestGethEngineClient_InitChain(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	genesisTime := time.Now()

	stateRoot, err := client.InitChain(ctx, genesisTime, 1, "1337")
	require.NoError(t, err)
	assert.NotEmpty(t, stateRoot)
}

func TestGethEngineClient_GetLatestHeight(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	height, err := client.GetLatestHeight(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), height)
}

func TestGethEthClient_HeaderByNumber(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)

	ctx := context.Background()

	header, err := client.ethClient.HeaderByNumber(ctx, big.NewInt(0))
	require.NoError(t, err)
	assert.NotNil(t, header)
	assert.Equal(t, uint64(0), header.Number.Uint64())

	latestHeader, err := client.ethClient.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, latestHeader)
}

func TestGethEthClient_GetTxs_EmptyPool(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	txs, err := client.GetTxs(ctx)
	require.NoError(t, err)
	assert.Empty(t, txs)
}

func TestGethEngineClient_ExecuteTxs_EmptyBlock(t *testing.T) {
	genesis := testGenesis()
	db := testDatastore()
	logger := zerolog.Nop()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	client, err := NewEngineExecutionClientWithGeth(genesis, feeRecipient, db, "", logger)
	require.NoError(t, err)

	ctx := context.Background()
	genesisTime := time.Now()

	stateRoot, err := client.InitChain(ctx, genesisTime, 1, "1337")
	require.NoError(t, err)

	newStateRoot, err := client.ExecuteTxs(ctx, [][]byte{}, 1, genesisTime.Add(time.Second*12), stateRoot)
	require.NoError(t, err)
	assert.NotEmpty(t, newStateRoot)
}

func TestGethEngineClient_ForkchoiceUpdated(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{backend: backend, logger: logger}
	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, resp.PayloadStatus.Status)
	assert.Nil(t, resp.PayloadID)
}

func TestGethEngineClient_ForkchoiceUpdated_WithPayloadAttributes(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{backend: backend, logger: logger}
	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, resp.PayloadStatus.Status)
	assert.NotNil(t, resp.PayloadID)
}

func TestGethEngineClient_GetPayload(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{backend: backend, logger: logger}
	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp.PayloadID)

	envelope, err := engineClient.GetPayload(ctx, *resp.PayloadID)
	require.NoError(t, err)
	assert.NotNil(t, envelope)
	assert.NotNil(t, envelope.ExecutionPayload)
	assert.Equal(t, uint64(1), envelope.ExecutionPayload.Number)
	assert.Equal(t, feeRecipient, envelope.ExecutionPayload.FeeRecipient)
}

func TestGethEngineClient_NewPayload(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{backend: backend, logger: logger}
	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp.PayloadID)

	envelope, err := engineClient.GetPayload(ctx, *resp.PayloadID)
	require.NoError(t, err)

	status, err := engineClient.NewPayload(ctx, envelope.ExecutionPayload, nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, status.Status)
}

func TestGethEngineClient_ForkchoiceUpdated_UnknownHead(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{backend: backend, logger: logger}
	ctx := context.Background()

	unknownHash := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      unknownHash,
		SafeBlockHash:      unknownHash,
		FinalizedBlockHash: unknownHash,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, engine.SYNCING, resp.PayloadStatus.Status)
}

func TestGethBackend_Close(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)

	err = backend.Close()
	require.NoError(t, err)
}

func TestCalcBaseFee(t *testing.T) {
	config := params.AllDevChainProtocolChanges

	// At exactly 50% full, base fee stays the same
	parent := &types.Header{
		Number:   big.NewInt(100),
		GasLimit: 30_000_000,
		GasUsed:  15_000_000,
		BaseFee:  big.NewInt(1_000_000_000),
	}
	baseFee := calcBaseFee(config, parent)
	require.NotNil(t, baseFee)
	assert.Equal(t, parent.BaseFee, baseFee)
}

func TestCalcBaseFee_OverTarget(t *testing.T) {
	config := params.AllDevChainProtocolChanges

	parent := &types.Header{
		Number:   big.NewInt(100),
		GasLimit: 30_000_000,
		GasUsed:  20_000_000, // >50% full
		BaseFee:  big.NewInt(1_000_000_000),
	}
	baseFee := calcBaseFee(config, parent)
	require.NotNil(t, baseFee)
	assert.Greater(t, baseFee.Int64(), parent.BaseFee.Int64())
}

func TestCalcBaseFee_UnderTarget(t *testing.T) {
	config := params.AllDevChainProtocolChanges

	parent := &types.Header{
		Number:   big.NewInt(100),
		GasLimit: 30_000_000,
		GasUsed:  5_000_000, // <50% full
		BaseFee:  big.NewInt(1_000_000_000),
	}
	baseFee := calcBaseFee(config, parent)
	require.NotNil(t, baseFee)
	assert.Less(t, baseFee.Int64(), parent.BaseFee.Int64())
}

func TestParsePayloadAttributes(t *testing.T) {
	logger := zerolog.Nop()
	engineClient := &gethEngineClient{logger: logger}

	parentHash := common.HexToHash("0xabcd")
	feeRecipient := common.HexToAddress("0x1234")
	timestamp := int64(1234567890)

	attrs := map[string]any{
		"timestamp":             timestamp,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{"0xaabbcc", "0xddeeff"},
		"gasLimit":              uint64(30_000_000),
	}

	payload, err := engineClient.parsePayloadAttributes(parentHash, attrs)
	require.NoError(t, err)
	assert.Equal(t, parentHash, payload.parentHash)
	assert.Equal(t, uint64(timestamp), payload.timestamp)
	assert.Equal(t, feeRecipient, payload.feeRecipient)
	assert.Equal(t, uint64(30_000_000), payload.gasLimit)
	assert.Len(t, payload.transactions, 2)
}

func TestCreateBloom(t *testing.T) {
	bloom := createBloom([]*types.Receipt{})
	assert.Equal(t, types.Bloom{}, bloom)

	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, Logs: []*types.Log{}}
	bloom = createBloom([]*types.Receipt{receipt})
	assert.NotNil(t, bloom)
}

func TestGethEngineClient_GetPayloadRemovesFromPending(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{backend: backend, logger: logger}
	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           []*types.Withdrawal{},
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp.PayloadID)

	assert.NotNil(t, backend.pendingPayload)

	envelope, err := engineClient.GetPayload(ctx, *resp.PayloadID)
	require.NoError(t, err)
	assert.NotNil(t, envelope)

	assert.Nil(t, backend.pendingPayload)

	_, err = engineClient.GetPayload(ctx, *resp.PayloadID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown payload ID")
}

func TestGethEngineClient_WithdrawalProcessing(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	engineClient := &gethEngineClient{backend: backend, logger: logger}
	ctx := context.Background()
	genesisBlock := backend.blockchain.Genesis()
	feeRecipient := common.HexToAddress("0x1234567890123456789012345678901234567890")

	withdrawalAddr1 := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	withdrawalAddr2 := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	stateDB, err := backend.blockchain.StateAt(genesisBlock.Root())
	require.NoError(t, err)
	assert.True(t, stateDB.GetBalance(withdrawalAddr1).IsZero())
	assert.True(t, stateDB.GetBalance(withdrawalAddr2).IsZero())

	withdrawals := []*types.Withdrawal{
		{Index: 0, Validator: 1, Address: withdrawalAddr1, Amount: 1000000000}, // 1 ETH in Gwei
		{Index: 1, Validator: 2, Address: withdrawalAddr2, Amount: 500000000},  // 0.5 ETH in Gwei
	}

	attrs := map[string]any{
		"timestamp":             time.Now().Unix() + 12,
		"prevRandao":            common.Hash{1, 2, 3},
		"suggestedFeeRecipient": feeRecipient,
		"transactions":          []string{},
		"gasLimit":              uint64(30_000_000),
		"withdrawals":           withdrawals,
	}

	resp, err := engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisBlock.Hash(),
		SafeBlockHash:      genesisBlock.Hash(),
		FinalizedBlockHash: genesisBlock.Hash(),
	}, attrs)
	require.NoError(t, err)
	require.NotNil(t, resp.PayloadID)

	envelope, err := engineClient.GetPayload(ctx, *resp.PayloadID)
	require.NoError(t, err)
	assert.Len(t, envelope.ExecutionPayload.Withdrawals, 2)

	status, err := engineClient.NewPayload(ctx, envelope.ExecutionPayload, nil, "", nil)
	require.NoError(t, err)
	assert.Equal(t, engine.VALID, status.Status)

	_, err = engineClient.ForkchoiceUpdated(ctx, engine.ForkchoiceStateV1{
		HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
		SafeBlockHash:      envelope.ExecutionPayload.BlockHash,
		FinalizedBlockHash: envelope.ExecutionPayload.BlockHash,
	}, nil)
	require.NoError(t, err)

	newBlock := backend.blockchain.GetBlockByHash(envelope.ExecutionPayload.BlockHash)
	require.NotNil(t, newBlock)

	newStateDB, err := backend.blockchain.StateAt(newBlock.Root())
	require.NoError(t, err)

	expectedBalance1 := new(big.Int).Mul(big.NewInt(1000000000), big.NewInt(1e9))
	expectedBalance2 := new(big.Int).Mul(big.NewInt(500000000), big.NewInt(1e9))

	assert.Equal(t, expectedBalance1.String(), newStateDB.GetBalance(withdrawalAddr1).ToBig().String())
	assert.Equal(t, expectedBalance2.String(), newStateDB.GetBalance(withdrawalAddr2).ToBig().String())
}

func TestGethEngineClient_ContractCreationAddress(t *testing.T) {
	sender := common.HexToAddress("0x1234567890123456789012345678901234567890")
	nonce := uint64(0)
	expectedAddr := crypto.CreateAddress(sender, nonce)

	expectedAddr2 := crypto.CreateAddress(sender, nonce)
	assert.Equal(t, expectedAddr, expectedAddr2)

	differentAddr := crypto.CreateAddress(sender, nonce+1)
	assert.NotEqual(t, expectedAddr, differentAddr)

	differentSender := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	differentAddr2 := crypto.CreateAddress(differentSender, nonce)
	assert.NotEqual(t, expectedAddr, differentAddr2)
}

func TestWeb3Sha3(t *testing.T) {
	service := &Web3RPCService{}

	input := hexutil.Bytes("hello")
	result := service.Sha3(input)

	expected := common.FromHex("0x1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8")
	assert.Equal(t, expected, []byte(result))
}

func TestEthRPCService_Accounts(t *testing.T) {
	service := &EthRPCService{}
	assert.Empty(t, service.Accounts())
}

func TestEthRPCService_Syncing(t *testing.T) {
	service := &EthRPCService{}
	result, err := service.Syncing()
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestEthRPCService_FeeHistory(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	result, err := service.FeeHistory(10, rpc.LatestBlockNumber, []float64{25, 50, 75})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result["oldestBlock"])
}

func TestEthRPCService_UnclesMethods(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	count := service.GetUncleCountByBlockHash(common.Hash{})
	assert.NotNil(t, count)
	assert.Equal(t, hexutil.Uint(0), *count)

	count = service.GetUncleCountByBlockNumber(rpc.LatestBlockNumber)
	assert.NotNil(t, count)
	assert.Equal(t, hexutil.Uint(0), *count)
}

func TestEthRPCService_BlockTransactionCount(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	genesisHash := backend.blockchain.Genesis().Hash()
	count := service.GetBlockTransactionCountByHash(genesisHash)
	assert.NotNil(t, count)
	assert.Equal(t, hexutil.Uint(0), *count)

	count = service.GetBlockTransactionCountByNumber(0)
	assert.NotNil(t, count)
	assert.Equal(t, hexutil.Uint(0), *count)
}

func TestEthRPCService_GetBlockByNumber(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	block, err := service.GetBlockByNumber(0, false)
	require.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, (*hexutil.Big)(big.NewInt(0)), block["number"])

	block, err = service.GetBlockByNumber(rpc.LatestBlockNumber, true)
	require.NoError(t, err)
	assert.NotNil(t, block)
}

func TestEthRPCService_GetBlockByHash(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	genesisHash := backend.blockchain.Genesis().Hash()
	block, err := service.GetBlockByHash(genesisHash, false)
	require.NoError(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, genesisHash, block["hash"])
}

func TestEthRPCService_GasPrice(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	gasPrice := service.GasPrice()
	assert.NotNil(t, gasPrice)
	assert.Greater(t, (*big.Int)(gasPrice).Int64(), int64(0))
}

func TestEthRPCService_ChainId(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	chainId := service.ChainId()
	assert.NotNil(t, chainId)
}

func TestEthRPCService_BlockNumber(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &EthRPCService{backend: backend, logger: logger}

	blockNumber := service.BlockNumber()
	assert.Equal(t, hexutil.Uint64(0), blockNumber)
}

func TestNetRPCService(t *testing.T) {
	genesis := testGenesis()
	logger := zerolog.Nop()

	backend, err := newGethBackend(genesis, ds.NewMapDatastore(), logger)
	require.NoError(t, err)
	defer backend.Close()

	service := &NetRPCService{backend: backend}

	assert.NotEmpty(t, service.Version())
	assert.True(t, service.Listening())
	assert.Equal(t, hexutil.Uint(0), service.PeerCount())
}

func TestWeb3RPCService(t *testing.T) {
	service := &Web3RPCService{}

	assert.NotEmpty(t, service.ClientVersion())

	hash := service.Sha3([]byte("test"))
	assert.Len(t, hash, 32)
}
