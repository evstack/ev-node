package evm

import (
	"encoding/hex"
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
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog"
)

// EthRPCService implements the eth_ JSON-RPC namespace.
type EthRPCService struct {
	backend *GethBackend
	logger  zerolog.Logger
}

// TxPoolExtService implements the txpoolExt_ JSON-RPC namespace.
type TxPoolExtService struct {
	backend *GethBackend
	logger  zerolog.Logger
}

// NetRPCService implements the net_ JSON-RPC namespace.
type NetRPCService struct {
	backend *GethBackend
}

// Web3RPCService implements the web3_ JSON-RPC namespace.
type Web3RPCService struct{}

// rpcHandler wraps the RPC server with CORS and proper HTTP handling.
type rpcHandler struct {
	rpcServer *rpc.Server
	logger    zerolog.Logger
}

// FilterQuery represents a filter for logs.
type FilterQuery struct {
	FromBlock *big.Int         `json:"fromBlock"`
	ToBlock   *big.Int         `json:"toBlock"`
	Addresses []common.Address `json:"address"`
	Topics    [][]common.Hash  `json:"topics"`
}

// TransactionArgs represents the arguments to construct a transaction.
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

// StartRPCServer starts the JSON-RPC server on the specified address.
// If address is empty, the server is not started.
func (b *GethBackend) StartRPCServer(address string) error {
	if address == "" {
		return nil
	}

	b.rpcServer = rpc.NewServer()

	// Register eth_ namespace
	ethService := &EthRPCService{backend: b, logger: b.logger.With().Str("rpc", "eth").Logger()}
	if err := b.rpcServer.RegisterName("eth", ethService); err != nil {
		return fmt.Errorf("failed to register eth service: %w", err)
	}

	// Register txpoolExt_ namespace for compatibility
	txpoolService := &TxPoolExtService{backend: b, logger: b.logger.With().Str("rpc", "txpoolExt").Logger()}
	if err := b.rpcServer.RegisterName("txpoolExt", txpoolService); err != nil {
		return fmt.Errorf("failed to register txpoolExt service: %w", err)
	}

	// Register net_ namespace
	netService := &NetRPCService{backend: b}
	if err := b.rpcServer.RegisterName("net", netService); err != nil {
		return fmt.Errorf("failed to register net service: %w", err)
	}

	// Register web3_ namespace
	web3Service := &Web3RPCService{}
	if err := b.rpcServer.RegisterName("web3", web3Service); err != nil {
		return fmt.Errorf("failed to register web3 service: %w", err)
	}

	// Create HTTP handler with CORS support
	handler := &rpcHandler{
		rpcServer: b.rpcServer,
		logger:    b.logger,
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	b.rpcListener = listener

	b.httpServer = &http.Server{
		Handler:      handler,
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

func (h *rpcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle GET requests with a simple response
	if r.Method == "GET" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":"ev-node in-process geth RPC","id":null}`))
		return
	}

	// Handle POST requests via the RPC server
	if r.Method == "POST" {
		h.rpcServer.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Version returns the network ID.
func (s *NetRPCService) Version() string {
	return s.backend.chainConfig.ChainID.String()
}

// Listening returns true if the node is listening for connections.
func (s *NetRPCService) Listening() bool {
	return true
}

// PeerCount returns the number of connected peers.
func (s *NetRPCService) PeerCount() hexutil.Uint {
	return 0
}

// ClientVersion returns the client version.
func (s *Web3RPCService) ClientVersion() string {
	return "ev-node/geth/1.0.0"
}

// Sha3 returns the Keccak-256 hash of the input.
func (s *Web3RPCService) Sha3(input hexutil.Bytes) hexutil.Bytes {
	hash := common.BytesToHash(input)
	return hash[:]
}

// ChainId returns the chain ID.
func (s *EthRPCService) ChainId() *hexutil.Big {
	return (*hexutil.Big)(s.backend.chainConfig.ChainID)
}

// BlockNumber returns the current block number.
func (s *EthRPCService) BlockNumber() hexutil.Uint64 {
	header := s.backend.blockchain.CurrentBlock()
	if header == nil {
		return 0
	}
	return hexutil.Uint64(header.Number.Uint64())
}

// GetBlockByNumber returns block information by number.
func (s *EthRPCService) GetBlockByNumber(blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	var block *types.Block
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		header := s.backend.blockchain.CurrentBlock()
		if header == nil {
			return nil, nil
		}
		block = s.backend.blockchain.GetBlock(header.Hash(), header.Number.Uint64())
	} else {
		block = s.backend.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return nil, nil
	}
	return s.formatBlock(block, fullTx), nil
}

// GetBlockByHash returns block information by hash.
func (s *EthRPCService) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block := s.backend.blockchain.GetBlockByHash(hash)
	if block == nil {
		return nil, nil
	}
	return s.formatBlock(block, fullTx), nil
}

// formatBlock formats a block for JSON-RPC response.
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

	result["uncles"] = []common.Hash{}
	return result
}

// formatTransaction formats a transaction for JSON-RPC response.
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
		"input":            hexutil.Bytes(tx.Data()),
	}

	if tx.Type() == types.LegacyTxType {
		result["gasPrice"] = (*hexutil.Big)(tx.GasPrice())
	} else {
		result["gasPrice"] = (*hexutil.Big)(tx.GasPrice())
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

// GetBalance returns the balance of an account.
func (s *EthRPCService) GetBalance(address common.Address, blockNr rpc.BlockNumber) (*hexutil.Big, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(stateDB.GetBalance(address).ToBig()), nil
}

// GetTransactionCount returns the nonce of an account.
func (s *EthRPCService) GetTransactionCount(address common.Address, blockNr rpc.BlockNumber) (hexutil.Uint64, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(stateDB.GetNonce(address)), nil
}

// GetCode returns the code at an address.
func (s *EthRPCService) GetCode(address common.Address, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}
	return stateDB.GetCode(address), nil
}

// GetStorageAt returns the storage value at a position.
func (s *EthRPCService) GetStorageAt(address common.Address, position hexutil.Big, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}
	value := stateDB.GetState(address, common.BigToHash((*big.Int)(&position)))
	return value[:], nil
}

// stateAtBlock returns the state at the given block number.
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

// Call executes a call without creating a transaction.
func (s *EthRPCService) Call(args TransactionArgs, blockNr rpc.BlockNumber) (hexutil.Bytes, error) {
	stateDB, err := s.stateAtBlock(blockNr)
	if err != nil {
		return nil, err
	}

	header := s.backend.blockchain.CurrentBlock()
	if header == nil {
		return nil, errors.New("no current block")
	}

	msg := args.ToMessage(header.BaseFee)
	blockContext := core.NewEVMBlockContext(header, s.backend.blockchain, nil)
	txContext := core.NewEVMTxContext(msg)
	evm := vm.NewEVM(blockContext, stateDB, s.backend.chainConfig, vm.Config{})
	evm.SetTxContext(txContext)

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

// EstimateGas estimates the gas needed for a transaction.
func (s *EthRPCService) EstimateGas(args TransactionArgs, blockNr *rpc.BlockNumber) (hexutil.Uint64, error) {
	blockNumber := rpc.LatestBlockNumber
	if blockNr != nil {
		blockNumber = *blockNr
	}

	stateDB, err := s.stateAtBlock(blockNumber)
	if err != nil {
		return 0, err
	}

	header := s.backend.blockchain.CurrentBlock()
	if header == nil {
		return 0, errors.New("no current block")
	}

	// Use a high gas limit for estimation
	hi := header.GasLimit
	if args.Gas != nil && uint64(*args.Gas) < hi {
		hi = uint64(*args.Gas)
	}

	lo := uint64(21000)

	// Binary search for the optimal gas
	for lo+1 < hi {
		mid := (lo + hi) / 2
		args.Gas = (*hexutil.Uint64)(&mid)

		msg := args.ToMessage(header.BaseFee)
		stateCopy := stateDB.Copy()
		blockContext := core.NewEVMBlockContext(header, s.backend.blockchain, nil)
		txContext := core.NewEVMTxContext(msg)
		evm := vm.NewEVM(blockContext, stateCopy, s.backend.chainConfig, vm.Config{})
		evm.SetTxContext(txContext)

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

// GasPrice returns the current gas price.
func (s *EthRPCService) GasPrice() *hexutil.Big {
	header := s.backend.blockchain.CurrentBlock()
	if header == nil || header.BaseFee == nil {
		return (*hexutil.Big)(big.NewInt(params.InitialBaseFee))
	}
	// Return base fee + 1 gwei tip
	tip := big.NewInt(1e9)
	return (*hexutil.Big)(new(big.Int).Add(header.BaseFee, tip))
}

// MaxPriorityFeePerGas returns the suggested priority fee.
func (s *EthRPCService) MaxPriorityFeePerGas() *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(1e9)) // 1 gwei
}

// SendRawTransaction sends a signed transaction.
func (s *EthRPCService) SendRawTransaction(encodedTx hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(encodedTx); err != nil {
		return common.Hash{}, err
	}

	errs := s.backend.txPool.Add([]*types.Transaction{tx}, true)
	if len(errs) > 0 && errs[0] != nil {
		return common.Hash{}, errs[0]
	}

	s.logger.Info().
		Str("tx_hash", tx.Hash().Hex()).
		Msg("received raw transaction")

	return tx.Hash(), nil
}

// GetTransactionByHash returns transaction info by hash.
func (s *EthRPCService) GetTransactionByHash(hash common.Hash) (map[string]interface{}, error) {
	// Search in recent blocks
	currentBlock := s.backend.blockchain.CurrentBlock()
	if currentBlock == nil {
		return nil, nil
	}

	// Search backwards through blocks
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

// GetTransactionReceipt returns the receipt of a transaction.
func (s *EthRPCService) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	// Search in recent blocks
	currentBlock := s.backend.blockchain.CurrentBlock()
	if currentBlock == nil {
		return nil, nil
	}

	// Search backwards through blocks
	for i := currentBlock.Number.Uint64(); i > 0 && i > currentBlock.Number.Uint64()-1000; i-- {
		block := s.backend.blockchain.GetBlockByNumber(i)
		if block == nil {
			continue
		}
		for idx, tx := range block.Transactions() {
			if tx.Hash() == hash {
				receipts := s.backend.blockchain.GetReceiptsByHash(block.Hash())
				if len(receipts) <= idx {
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

				return result, nil
			}
		}
	}
	return nil, nil
}

// GetLogs returns logs matching the filter criteria.
func (s *EthRPCService) GetLogs(filter FilterQuery) ([]*types.Log, error) {
	var fromBlock, toBlock uint64

	if filter.FromBlock == nil {
		fromBlock = s.backend.blockchain.CurrentBlock().Number.Uint64()
	} else {
		fromBlock = filter.FromBlock.Uint64()
	}

	if filter.ToBlock == nil {
		toBlock = s.backend.blockchain.CurrentBlock().Number.Uint64()
	} else {
		toBlock = filter.ToBlock.Uint64()
	}

	var logs []*types.Log
	for i := fromBlock; i <= toBlock; i++ {
		block := s.backend.blockchain.GetBlockByNumber(i)
		if block == nil {
			continue
		}
		receipts := s.backend.blockchain.GetReceiptsByHash(block.Hash())
		for _, receipt := range receipts {
			for _, log := range receipt.Logs {
				if s.matchLog(log, filter) {
					logs = append(logs, log)
				}
			}
		}
	}
	return logs, nil
}

// matchLog checks if a log matches the filter criteria.
func (s *EthRPCService) matchLog(log *types.Log, filter FilterQuery) bool {
	if len(filter.Addresses) > 0 {
		found := false
		for _, addr := range filter.Addresses {
			if log.Address == addr {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	for i, topics := range filter.Topics {
		if len(topics) == 0 {
			continue
		}
		if i >= len(log.Topics) {
			return false
		}
		found := false
		for _, topic := range topics {
			if log.Topics[i] == topic {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// ToMessage converts TransactionArgs to a core.Message.
func (args *TransactionArgs) ToMessage(baseFee *big.Int) *core.Message {
	var from common.Address
	if args.From != nil {
		from = *args.From
	}

	var to *common.Address
	if args.To != nil {
		to = args.To
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

	msg := &core.Message{
		From:            from,
		To:              to,
		Value:           value,
		GasLimit:        gas,
		GasPrice:        gasPrice,
		GasFeeCap:       gasPrice,
		GasTipCap:       big.NewInt(0),
		Data:            data,
		SkipNonceChecks: true,
	}
	return msg
}

// GetTxs returns pending transactions (for txpoolExt compatibility).
func (s *TxPoolExtService) GetTxs() ([]string, error) {
	pending := s.backend.txPool.Pending(txpool.PendingFilter{})
	var result []string
	for _, txs := range pending {
		for _, lazyTx := range txs {
			tx := lazyTx.Tx
			if tx == nil {
				continue
			}
			data, err := tx.MarshalBinary()
			if err != nil {
				continue
			}
			result = append(result, "0x"+hex.EncodeToString(data))
		}
	}
	return result, nil
}
