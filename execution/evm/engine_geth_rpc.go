package evm

import (
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog"
)

// EthRPCService implements essential eth_ JSON-RPC methods for block explorers.
type EthRPCService struct {
	backend *GethBackend
	logger  zerolog.Logger
}

// NetRPCService implements the net_ JSON-RPC namespace.
type NetRPCService struct {
	backend *GethBackend
}

// Web3RPCService implements the web3_ JSON-RPC namespace.
type Web3RPCService struct{}

// TransactionArgs represents transaction call arguments.
type TransactionArgs struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Data                 *hexutil.Bytes  `json:"data"`
	Input                *hexutil.Bytes  `json:"input"`
	Nonce                *hexutil.Uint64 `json:"nonce"`
}

// StartRPCServer starts a minimal JSON-RPC server for block explorer compatibility.
func (b *GethBackend) StartRPCServer(address string) error {
	if address == "" {
		return nil
	}

	b.rpcServer = rpc.NewServer()

	if err := b.rpcServer.RegisterName("eth", &EthRPCService{backend: b, logger: b.logger.With().Str("rpc", "eth").Logger()}); err != nil {
		return fmt.Errorf("failed to register eth: %w", err)
	}
	if err := b.rpcServer.RegisterName("net", &NetRPCService{backend: b}); err != nil {
		return fmt.Errorf("failed to register net: %w", err)
	}
	if err := b.rpcServer.RegisterName("web3", &Web3RPCService{}); err != nil {
		return fmt.Errorf("failed to register web3: %w", err)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	b.rpcListener = listener

	b.httpServer = &http.Server{
		Handler:      &rpcHandler{rpcServer: b.rpcServer},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		b.logger.Info().Str("address", address).Msg("starting JSON-RPC server")
		if err := b.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			b.logger.Error().Err(err).Msg("JSON-RPC server error")
		}
	}()

	return nil
}

type rpcHandler struct {
	rpcServer *rpc.Server
}

func (h *rpcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		return
	}
	if r.Method == "GET" {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":"ev-node RPC","id":null}`))
		return
	}
	if r.Method == "POST" {
		h.rpcServer.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// --- Web3 namespace ---

func (s *Web3RPCService) ClientVersion() string {
	return "ev-node/1.0.0"
}

func (s *Web3RPCService) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}

// --- Net namespace ---

func (s *NetRPCService) Version() string {
	return s.backend.chainConfig.ChainID.String()
}

func (s *NetRPCService) Listening() bool {
	return true
}

func (s *NetRPCService) PeerCount() hexutil.Uint {
	return 0
}

// --- Eth namespace ---

func (s *EthRPCService) ChainId() *hexutil.Big {
	return (*hexutil.Big)(s.backend.chainConfig.ChainID)
}

func (s *EthRPCService) BlockNumber() hexutil.Uint64 {
	if header := s.backend.blockchain.CurrentBlock(); header != nil {
		return hexutil.Uint64(header.Number.Uint64())
	}
	return 0
}

func (s *EthRPCService) GasPrice() *hexutil.Big {
	header := s.backend.blockchain.CurrentBlock()
	if header == nil || header.BaseFee == nil {
		return (*hexutil.Big)(big.NewInt(params.InitialBaseFee))
	}
	return (*hexutil.Big)(new(big.Int).Add(header.BaseFee, big.NewInt(1e9)))
}

func (s *EthRPCService) MaxPriorityFeePerGas() *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(1e9))
}

func (s *EthRPCService) Syncing() (interface{}, error) {
	return false, nil
}

func (s *EthRPCService) Accounts() []common.Address {
	return []common.Address{}
}

func (s *EthRPCService) GetBalance(address common.Address, blockNr rpc.BlockNumber) (*hexutil.Big, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(stateDB.GetBalance(address).ToBig()), nil
}

func (s *EthRPCService) GetTransactionCount(address common.Address, blockNr rpc.BlockNumber) (hexutil.Uint64, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(stateDB.GetNonce(address)), nil
}

func (s *EthRPCService) GetCode(address common.Address, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}
	return stateDB.GetCode(address), nil
}

func (s *EthRPCService) GetStorageAt(address common.Address, position hexutil.Big, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}
	value := stateDB.GetState(address, common.BigToHash((*big.Int)(&position)))
	return value[:], nil
}

func (s *EthRPCService) GetBlockByNumber(blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	var block *types.Block
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		if header := s.backend.blockchain.CurrentBlock(); header != nil {
			block = s.backend.blockchain.GetBlock(header.Hash(), header.Number.Uint64())
		}
	} else {
		block = s.backend.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return nil, nil
	}
	return s.formatBlock(block, fullTx), nil
}

func (s *EthRPCService) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block := s.backend.blockchain.GetBlockByHash(hash)
	if block == nil {
		return nil, nil
	}
	return s.formatBlock(block, fullTx), nil
}

func (s *EthRPCService) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	block := s.backend.blockchain.GetBlockByHash(hash)
	if block == nil {
		return nil
	}
	n := hexutil.Uint(len(block.Transactions()))
	return &n
}

func (s *EthRPCService) GetBlockTransactionCountByNumber(blockNr rpc.BlockNumber) *hexutil.Uint {
	var block *types.Block
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		if header := s.backend.blockchain.CurrentBlock(); header != nil {
			block = s.backend.blockchain.GetBlock(header.Hash(), header.Number.Uint64())
		}
	} else {
		block = s.backend.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return nil
	}
	n := hexutil.Uint(len(block.Transactions()))
	return &n
}

func (s *EthRPCService) GetTransactionByHash(hash common.Hash) (map[string]interface{}, error) {
	currentBlock := s.backend.blockchain.CurrentBlock()
	if currentBlock == nil {
		return nil, nil
	}

	for i := currentBlock.Number.Uint64(); i > 0 && i > currentBlock.Number.Uint64()-1000; i-- {
		block := s.backend.blockchain.GetBlockByNumber(i)
		if block == nil {
			continue
		}
		for idx, tx := range block.Transactions() {
			if tx.Hash() == hash {
				return s.formatTransaction(tx, block.Hash(), block.NumberU64(), uint64(idx)), nil
			}
		}
	}
	return nil, nil
}

func (s *EthRPCService) GetTransactionByBlockHashAndIndex(hash common.Hash, index hexutil.Uint) map[string]interface{} {
	block := s.backend.blockchain.GetBlockByHash(hash)
	if block == nil {
		return nil
	}
	txs := block.Transactions()
	if int(index) >= len(txs) {
		return nil
	}
	return s.formatTransaction(txs[index], block.Hash(), block.NumberU64(), uint64(index))
}

func (s *EthRPCService) GetTransactionByBlockNumberAndIndex(blockNr rpc.BlockNumber, index hexutil.Uint) map[string]interface{} {
	var block *types.Block
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		if header := s.backend.blockchain.CurrentBlock(); header != nil {
			block = s.backend.blockchain.GetBlock(header.Hash(), header.Number.Uint64())
		}
	} else {
		block = s.backend.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return nil
	}
	txs := block.Transactions()
	if int(index) >= len(txs) {
		return nil
	}
	return s.formatTransaction(txs[index], block.Hash(), block.NumberU64(), uint64(index))
}

func (s *EthRPCService) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	currentBlock := s.backend.blockchain.CurrentBlock()
	if currentBlock == nil {
		return nil, nil
	}

	for i := currentBlock.Number.Uint64(); i > 0 && i > currentBlock.Number.Uint64()-1000; i-- {
		block := s.backend.blockchain.GetBlockByNumber(i)
		if block == nil {
			continue
		}
		for idx, tx := range block.Transactions() {
			if tx.Hash() == hash {
				receipts := s.backend.blockchain.GetReceiptsByHash(block.Hash())
				if idx >= len(receipts) {
					return nil, nil
				}
				receipt := receipts[idx]
				signer := types.LatestSignerForChainID(s.backend.chainConfig.ChainID)
				from, _ := types.Sender(signer, tx)

				result := map[string]interface{}{
					"transactionHash":   hash,
					"transactionIndex":  hexutil.Uint64(idx),
					"blockHash":         block.Hash(),
					"blockNumber":       (*hexutil.Big)(block.Number()),
					"from":              from,
					"to":                tx.To(),
					"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
					"gasUsed":           hexutil.Uint64(receipt.GasUsed),
					"contractAddress":   nil,
					"logs":              receipt.Logs,
					"logsBloom":         receipt.Bloom,
					"status":            hexutil.Uint(receipt.Status),
					"effectiveGasPrice": (*hexutil.Big)(tx.GasPrice()),
					"type":              hexutil.Uint(tx.Type()),
				}
				if receipt.ContractAddress != (common.Address{}) {
					result["contractAddress"] = receipt.ContractAddress
				}
				if result["logs"] == nil {
					result["logs"] = []*types.Log{}
				}
				return result, nil
			}
		}
	}
	return nil, nil
}

func (s *EthRPCService) GetLogs(args map[string]interface{}) ([]*types.Log, error) {
	currentBlock := s.backend.blockchain.CurrentBlock()
	if currentBlock == nil {
		return []*types.Log{}, nil
	}

	fromBlock := currentBlock.Number.Uint64()
	toBlock := currentBlock.Number.Uint64()

	if fb, ok := args["fromBlock"]; ok {
		if fbStr, ok := fb.(string); ok && fbStr != "latest" && fbStr != "pending" {
			if n, err := hexutil.DecodeUint64(fbStr); err == nil {
				fromBlock = n
			}
		}
	}
	if tb, ok := args["toBlock"]; ok {
		if tbStr, ok := tb.(string); ok && tbStr != "latest" && tbStr != "pending" {
			if n, err := hexutil.DecodeUint64(tbStr); err == nil {
				toBlock = n
			}
		}
	}

	var addresses []common.Address
	if addr, ok := args["address"]; ok {
		switch v := addr.(type) {
		case string:
			addresses = append(addresses, common.HexToAddress(v))
		case []interface{}:
			for _, a := range v {
				if s, ok := a.(string); ok {
					addresses = append(addresses, common.HexToAddress(s))
				}
			}
		}
	}

	var logs []*types.Log
	for i := fromBlock; i <= toBlock && i <= fromBlock+1000; i++ {
		block := s.backend.blockchain.GetBlockByNumber(i)
		if block == nil {
			continue
		}
		receipts := s.backend.blockchain.GetReceiptsByHash(block.Hash())
		for _, receipt := range receipts {
			for _, log := range receipt.Logs {
				if len(addresses) == 0 || containsAddress(addresses, log.Address) {
					logs = append(logs, log)
				}
			}
		}
	}
	return logs, nil
}

func (s *EthRPCService) Call(args TransactionArgs, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}

	header := s.backend.blockchain.CurrentBlock()
	if header == nil {
		return nil, errors.New("no current block")
	}

	msg := args.toMessage(header.BaseFee)
	blockContext := core.NewEVMBlockContext(header, s.backend.blockchain, nil)
	evm := vm.NewEVM(blockContext, stateDB, s.backend.chainConfig, vm.Config{})
	evm.SetTxContext(core.NewEVMTxContext(msg))

	gp := new(core.GasPool).AddGas(header.GasLimit)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Return(), nil
}

func (s *EthRPCService) EstimateGas(args TransactionArgs, blockNr *rpc.BlockNumber) (hexutil.Uint64, error) {
	bn := rpc.LatestBlockNumber
	if blockNr != nil {
		bn = *blockNr
	}

	stateDB, err := s.stateAtBlock(bn)
	if err != nil {
		return 0, err
	}

	header := s.backend.blockchain.CurrentBlock()
	if header == nil {
		return 0, errors.New("no current block")
	}

	hi := header.GasLimit
	if args.Gas != nil && uint64(*args.Gas) < hi {
		hi = uint64(*args.Gas)
	}
	lo := uint64(21000)

	for lo+1 < hi {
		mid := (lo + hi) / 2
		args.Gas = (*hexutil.Uint64)(&mid)

		msg := args.toMessage(header.BaseFee)
		stateCopy := stateDB.Copy()
		blockContext := core.NewEVMBlockContext(header, s.backend.blockchain, nil)
		evm := vm.NewEVM(blockContext, stateCopy, s.backend.chainConfig, vm.Config{})
		evm.SetTxContext(core.NewEVMTxContext(msg))

		gp := new(core.GasPool).AddGas(mid)
		result, err := core.ApplyMessage(evm, msg, gp)
		if err != nil || result.Failed() {
			lo = mid
		} else {
			hi = mid
		}
	}

	return hexutil.Uint64(hi), nil
}

func (s *EthRPCService) SendRawTransaction(encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encodedTx); err != nil {
		return common.Hash{}, err
	}

	errs := s.backend.txPool.Add([]*types.Transaction{tx}, true)
	if len(errs) > 0 && errs[0] != nil {
		return common.Hash{}, errs[0]
	}

	s.logger.Info().Str("tx_hash", tx.Hash().Hex()).Msg("received transaction")
	return tx.Hash(), nil
}

func (s *EthRPCService) FeeHistory(blockCount hexutil.Uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (map[string]interface{}, error) {
	if blockCount == 0 || blockCount > 1024 {
		blockCount = 1024
	}

	var endBlock uint64
	if lastBlock == rpc.LatestBlockNumber || lastBlock == rpc.PendingBlockNumber {
		endBlock = s.backend.blockchain.CurrentBlock().Number.Uint64()
	} else {
		endBlock = uint64(lastBlock)
	}

	startBlock := endBlock - uint64(blockCount) + 1
	if startBlock > endBlock {
		startBlock = 0
	}

	baseFees := make([]*hexutil.Big, 0)
	gasUsedRatios := make([]float64, 0)

	for i := startBlock; i <= endBlock; i++ {
		block := s.backend.blockchain.GetBlockByNumber(i)
		if block == nil {
			continue
		}
		header := block.Header()

		if header.BaseFee != nil {
			baseFees = append(baseFees, (*hexutil.Big)(header.BaseFee))
		} else {
			baseFees = append(baseFees, (*hexutil.Big)(big.NewInt(0)))
		}

		if header.GasLimit > 0 {
			gasUsedRatios = append(gasUsedRatios, float64(header.GasUsed)/float64(header.GasLimit))
		} else {
			gasUsedRatios = append(gasUsedRatios, 0)
		}
	}

	if len(baseFees) > 0 {
		baseFees = append(baseFees, baseFees[len(baseFees)-1])
	}

	return map[string]interface{}{
		"oldestBlock":   (*hexutil.Big)(new(big.Int).SetUint64(startBlock)),
		"baseFeePerGas": baseFees,
		"gasUsedRatio":  gasUsedRatios,
	}, nil
}

// GetUncleCountByBlockHash returns 0 (no uncles in PoS).
func (s *EthRPCService) GetUncleCountByBlockHash(hash common.Hash) *hexutil.Uint {
	n := hexutil.Uint(0)
	return &n
}

// GetUncleCountByBlockNumber returns 0 (no uncles in PoS).
func (s *EthRPCService) GetUncleCountByBlockNumber(blockNr rpc.BlockNumber) *hexutil.Uint {
	n := hexutil.Uint(0)
	return &n
}

// --- Helpers ---

func (s *EthRPCService) stateAtBlock(blockNr rpc.BlockNumber) (*state.StateDB, error) {
	var header *types.Header
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		header = s.backend.blockchain.CurrentBlock()
	} else {
		block := s.backend.blockchain.GetBlockByNumber(uint64(blockNr))
		if block == nil {
			return nil, fmt.Errorf("block %d not found", blockNr)
		}
		header = block.Header()
	}
	if header == nil {
		return nil, errors.New("no current block")
	}
	return s.backend.blockchain.StateAt(header.Root)
}

func (s *EthRPCService) formatBlock(block *types.Block, fullTx bool) map[string]interface{} {
	header := block.Header()
	result := map[string]interface{}{
		"number":           (*hexutil.Big)(header.Number),
		"hash":             block.Hash(),
		"parentHash":       header.ParentHash,
		"nonce":            header.Nonce,
		"sha3Uncles":       header.UncleHash,
		"logsBloom":        header.Bloom,
		"transactionsRoot": header.TxHash,
		"stateRoot":        header.Root,
		"receiptsRoot":     header.ReceiptHash,
		"miner":            header.Coinbase,
		"difficulty":       (*hexutil.Big)(header.Difficulty),
		"extraData":        hexutil.Bytes(header.Extra),
		"size":             hexutil.Uint64(block.Size()),
		"gasLimit":         hexutil.Uint64(header.GasLimit),
		"gasUsed":          hexutil.Uint64(header.GasUsed),
		"timestamp":        hexutil.Uint64(header.Time),
		"mixHash":          header.MixDigest,
		"totalDifficulty":  (*hexutil.Big)(big.NewInt(0)),
		"uncles":           []common.Hash{},
	}

	if header.BaseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(header.BaseFee)
	}

	txs := block.Transactions()
	if fullTx {
		txList := make([]map[string]interface{}, len(txs))
		for i, tx := range txs {
			txList[i] = s.formatTransaction(tx, block.Hash(), header.Number.Uint64(), uint64(i))
		}
		result["transactions"] = txList
	} else {
		txHashes := make([]common.Hash, len(txs))
		for i, tx := range txs {
			txHashes[i] = tx.Hash()
		}
		result["transactions"] = txHashes
	}

	return result
}

func (s *EthRPCService) formatTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) map[string]interface{} {
	signer := types.LatestSignerForChainID(s.backend.chainConfig.ChainID)
	from, _ := types.Sender(signer, tx)

	result := map[string]interface{}{
		"hash":             tx.Hash(),
		"nonce":            hexutil.Uint64(tx.Nonce()),
		"blockHash":        blockHash,
		"blockNumber":      (*hexutil.Big)(new(big.Int).SetUint64(blockNumber)),
		"transactionIndex": hexutil.Uint64(index),
		"from":             from,
		"to":               tx.To(),
		"value":            (*hexutil.Big)(tx.Value()),
		"gas":              hexutil.Uint64(tx.Gas()),
		"gasPrice":         (*hexutil.Big)(tx.GasPrice()),
		"input":            hexutil.Bytes(tx.Data()),
	}

	if tx.Type() != types.LegacyTxType {
		result["maxFeePerGas"] = (*hexutil.Big)(tx.GasFeeCap())
		result["maxPriorityFeePerGas"] = (*hexutil.Big)(tx.GasTipCap())
		result["type"] = hexutil.Uint64(tx.Type())
	}

	v, r, ss := tx.RawSignatureValues()
	result["v"] = (*hexutil.Big)(v)
	result["r"] = (*hexutil.Big)(r)
	result["s"] = (*hexutil.Big)(ss)

	return result
}

func (args *TransactionArgs) toMessage(baseFee *big.Int) *core.Message {
	var from common.Address
	if args.From != nil {
		from = *args.From
	}

	var gas uint64 = 50000000
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}

	var value *big.Int
	if args.Value != nil {
		value = (*big.Int)(args.Value)
	} else {
		value = big.NewInt(0)
	}

	var data []byte
	if args.Data != nil {
		data = *args.Data
	} else if args.Input != nil {
		data = *args.Input
	}

	var gasPrice *big.Int
	if args.GasPrice != nil {
		gasPrice = (*big.Int)(args.GasPrice)
	} else if baseFee != nil {
		gasPrice = new(big.Int).Add(baseFee, big.NewInt(1e9))
	} else {
		gasPrice = big.NewInt(params.InitialBaseFee)
	}

	return &core.Message{
		From:            from,
		To:              args.To,
		Value:           value,
		GasLimit:        gas,
		GasPrice:        gasPrice,
		GasFeeCap:       gasPrice,
		GasTipCap:       big.NewInt(0),
		Data:            data,
		SkipNonceChecks: true,
	}
}

func containsAddress(addrs []common.Address, addr common.Address) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}
