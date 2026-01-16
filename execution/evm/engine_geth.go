package evm

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
)

var (
	_ EngineRPCClient = (*gethEngineClient)(nil)
	_ EthRPCClient    = (*gethEthClient)(nil)
)

const (
	// baseFeeChangeDenominator is the EIP-1559 base fee change denominator.
	baseFeeChangeDenominator = 8
)

// GethBackend holds the in-process geth components.
type GethBackend struct {
	db          ethdb.Database
	chainConfig *params.ChainConfig
	blockchain  *core.BlockChain
	txPool      *txpool.TxPool

	mu sync.Mutex
	// payloads tracks in-flight payload builds
	payloads      map[engine.PayloadID]*payloadBuildState
	nextPayloadID uint64

	// RPC server
	rpcServer   *rpc.Server
	httpServer  *http.Server
	rpcListener net.Listener

	logger zerolog.Logger
}

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

// payloadBuildState tracks the state of a payload being built.
type payloadBuildState struct {
	parentHash   common.Hash
	timestamp    uint64
	prevRandao   common.Hash
	feeRecipient common.Address
	withdrawals  []*types.Withdrawal
	transactions [][]byte
	gasLimit     uint64

	// built payload (populated after getPayload)
	payload *engine.ExecutableData
}

// gethEngineClient implements EngineRPCClient using in-process geth.
type gethEngineClient struct {
	backend *GethBackend
	logger  zerolog.Logger
}

// gethEthClient implements EthRPCClient using in-process geth.
type gethEthClient struct {
	backend *GethBackend
	logger  zerolog.Logger
}

// NewEngineExecutionClientWithGeth creates an EngineClient that uses an in-process
// go-ethereum instance instead of connecting to an external execution engine via RPC.
// If rpcAddress is non-empty, an HTTP JSON-RPC server will be started on that address
// (e.g., "127.0.0.1:8545") exposing standard eth_ methods.
func NewEngineExecutionClientWithGeth(
	genesis *core.Genesis,
	feeRecipient common.Address,
	db ds.Batching,
	rpcAddress string,
	logger zerolog.Logger,
) (*EngineClient, error) {
	if db == nil {
		return nil, errors.New("db is required for EVM execution client")
	}
	if genesis == nil || genesis.Config == nil {
		return nil, errors.New("genesis configuration is required")
	}

	backend, err := newGethBackend(genesis, db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create geth backend: %w", err)
	}

	// Start RPC server if address is provided
	if rpcAddress != "" {
		if err := backend.StartRPCServer(rpcAddress); err != nil {
			backend.Close()
			return nil, fmt.Errorf("failed to start RPC server: %w", err)
		}
	}

	engineClient := &gethEngineClient{
		backend: backend,
		logger:  logger.With().Str("component", "geth-engine").Logger(),
	}

	ethClient := &gethEthClient{
		backend: backend,
		logger:  logger.With().Str("component", "geth-eth").Logger(),
	}

	genesisBlock := backend.blockchain.Genesis()
	genesisHash := genesisBlock.Hash()

	logger.Info().
		Str("genesis_hash", genesisHash.Hex()).
		Str("chain_id", genesis.Config.ChainID.String()).
		Uint64("genesis_gas_limit", genesis.GasLimit).
		Msg("created in-process geth execution client")

	return &EngineClient{
		engineClient:              engineClient,
		ethClient:                 ethClient,
		genesisHash:               genesisHash,
		feeRecipient:              feeRecipient,
		store:                     NewEVMStore(db),
		currentHeadBlockHash:      genesisHash,
		currentSafeBlockHash:      genesisHash,
		currentFinalizedBlockHash: genesisHash,
		blockHashCache:            make(map[uint64]common.Hash),
		logger:                    logger,
	}, nil
}

// newGethBackend creates a new in-process geth backend with persistent storage.
func newGethBackend(genesis *core.Genesis, db ds.Batching, logger zerolog.Logger) (*GethBackend, error) {
	// Wrap the datastore as an ethdb.Database
	ethdb := rawdb.NewDatabase(&wrapper{db})

	// Create trie database
	trieDB := triedb.NewDatabase(ethdb, nil)

	// Ensure blobSchedule is set if Cancun/Prague are enabled
	// TODO: remove and fix genesis.
	if genesis.Config != nil && genesis.Config.BlobScheduleConfig == nil {
		if genesis.Config.CancunTime != nil || genesis.Config.PragueTime != nil {
			genesis.Config.BlobScheduleConfig = &params.BlobScheduleConfig{
				Cancun: params.DefaultCancunBlobConfig,
				Prague: params.DefaultPragueBlobConfig,
			}
			logger.Debug().Msg("auto-populated blobSchedule config for Cancun/Prague forks")
		}
	}

	// Initialize the genesis block
	chainConfig, genesisHash, _, genesisErr := core.SetupGenesisBlockWithOverride(ethdb, trieDB, genesis, nil)
	if genesisErr != nil {
		return nil, fmt.Errorf("failed to setup genesis: %w", genesisErr)
	}

	logger.Info().
		Str("genesis_hash", genesisHash.Hex()).
		Str("chain_id", chainConfig.ChainID.String()).
		Msg("initialized in-process geth with genesis")

	// Create blockchain config
	bcConfig := core.DefaultConfig().WithStateScheme(rawdb.HashScheme)
	consensusEngine := beacon.New(nil)

	// Create the blockchain
	blockchain, err := core.NewBlockChain(ethdb, genesis, consensusEngine, bcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain: %w", err)
	}

	// Log current chain head
	currentHead := blockchain.CurrentBlock()
	if currentHead != nil {
		logger.Info().
			Uint64("height", currentHead.Number.Uint64()).
			Str("hash", currentHead.Hash().Hex()).
			Msg("resuming from existing chain state")
	}

	backend := &GethBackend{
		db:          ethdb,
		chainConfig: chainConfig,
		blockchain:  blockchain,
		payloads:    make(map[engine.PayloadID]*payloadBuildState),
		logger:      logger,
	}

	// Create transaction pool
	txPoolConfig := legacypool.DefaultConfig
	txPoolConfig.NoLocals = true

	legacyPool := legacypool.New(txPoolConfig, blockchain)
	txPool, err := txpool.New(0, blockchain, []txpool.SubPool{legacyPool})
	if err != nil {
		return nil, fmt.Errorf("failed to create tx pool: %w", err)
	}
	backend.txPool = txPool

	return backend, nil
}

// Close shuts down the geth backend gracefully.
func (b *GethBackend) Close() error {
	b.logger.Info().Msg("shutting down geth backend")

	var errs []error

	// Stop RPC server first
	if b.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := b.httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown http server: %w", err))
		}
	}
	if b.rpcServer != nil {
		b.rpcServer.Stop()
	}

	if b.txPool != nil {
		b.txPool.Close()
	}
	if b.blockchain != nil {
		b.blockchain.Stop()
	}
	if b.db != nil {
		if err := b.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
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

// rpcHandler wraps the RPC server with CORS and proper HTTP handling.
type rpcHandler struct {
	rpcServer *rpc.Server
	logger    zerolog.Logger
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

// NetRPCService implements the net_ JSON-RPC namespace.
type NetRPCService struct {
	backend *GethBackend
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

// Web3RPCService implements the web3_ JSON-RPC namespace.
type Web3RPCService struct{}

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

// ForkchoiceUpdated implements EngineRPCClient.
func (g *gethEngineClient) ForkchoiceUpdated(ctx context.Context, fcState engine.ForkchoiceStateV1, attrs map[string]any) (*engine.ForkChoiceResponse, error) {
	// Check context before acquiring lock
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	// Validate the forkchoice state
	headBlock := g.backend.blockchain.GetBlockByHash(fcState.HeadBlockHash)
	if headBlock == nil {
		g.logger.Debug().
			Str("head_hash", fcState.HeadBlockHash.Hex()).
			Msg("head block not found, returning SYNCING")
		return &engine.ForkChoiceResponse{
			PayloadStatus: engine.PayloadStatusV1{
				Status: engine.SYNCING,
			},
		}, nil
	}

	// Update the canonical chain head
	if _, err := g.backend.blockchain.SetCanonical(headBlock); err != nil {
		return nil, fmt.Errorf("failed to set canonical head: %w", err)
	}

	g.logger.Debug().
		Uint64("height", headBlock.NumberU64()).
		Str("hash", headBlock.Hash().Hex()).
		Msg("updated canonical head")

	response := &engine.ForkChoiceResponse{
		PayloadStatus: engine.PayloadStatusV1{
			Status:          engine.VALID,
			LatestValidHash: &fcState.HeadBlockHash,
		},
	}

	// If payload attributes provided, start building a new payload
	if attrs != nil {
		payloadState, err := g.parsePayloadAttributes(fcState.HeadBlockHash, attrs)
		if err != nil {
			return nil, fmt.Errorf("failed to parse payload attributes: %w", err)
		}

		// Generate payload ID deterministically from attributes
		payloadID := g.generatePayloadID(fcState.HeadBlockHash, payloadState)

		g.backend.payloads[payloadID] = payloadState
		response.PayloadID = &payloadID

		g.logger.Info().
			Str("payload_id", payloadID.String()).
			Str("parent_hash", fcState.HeadBlockHash.Hex()).
			Uint64("timestamp", payloadState.timestamp).
			Int("tx_count", len(payloadState.transactions)).
			Str("fee_recipient", payloadState.feeRecipient.Hex()).
			Msg("started payload build")
	}

	return response, nil
}

// generatePayloadID creates a deterministic payload ID from the build parameters.
func (g *gethEngineClient) generatePayloadID(parentHash common.Hash, ps *payloadBuildState) engine.PayloadID {
	g.backend.nextPayloadID++
	var payloadID engine.PayloadID
	binary.BigEndian.PutUint64(payloadID[:], g.backend.nextPayloadID)
	return payloadID
}

// parsePayloadAttributes extracts payload attributes from the map format.
func (g *gethEngineClient) parsePayloadAttributes(parentHash common.Hash, attrs map[string]any) (*payloadBuildState, error) {
	ps := &payloadBuildState{
		parentHash:  parentHash,
		withdrawals: []*types.Withdrawal{},
	}

	// Parse timestamp (required)
	if ts, ok := attrs["timestamp"]; ok {
		switch v := ts.(type) {
		case int64:
			ps.timestamp = uint64(v)
		case uint64:
			ps.timestamp = v
		case float64:
			ps.timestamp = uint64(v)
		default:
			return nil, fmt.Errorf("invalid timestamp type: %T", ts)
		}
	} else {
		return nil, errors.New("timestamp is required in payload attributes")
	}

	// Parse prevRandao (required for PoS)
	if pr, ok := attrs["prevRandao"]; ok {
		switch v := pr.(type) {
		case common.Hash:
			ps.prevRandao = v
		case string:
			ps.prevRandao = common.HexToHash(v)
		case []byte:
			ps.prevRandao = common.BytesToHash(v)
		default:
			return nil, fmt.Errorf("invalid prevRandao type: %T", pr)
		}
	}

	// Parse suggestedFeeRecipient (required)
	if fr, ok := attrs["suggestedFeeRecipient"]; ok {
		switch v := fr.(type) {
		case common.Address:
			ps.feeRecipient = v
		case string:
			ps.feeRecipient = common.HexToAddress(v)
		case []byte:
			ps.feeRecipient = common.BytesToAddress(v)
		default:
			return nil, fmt.Errorf("invalid suggestedFeeRecipient type: %T", fr)
		}
	} else {
		return nil, errors.New("suggestedFeeRecipient is required in payload attributes")
	}

	// Parse transactions (optional)
	if txs, ok := attrs["transactions"]; ok {
		switch v := txs.(type) {
		case []string:
			ps.transactions = make([][]byte, 0, len(v))
			for _, txHex := range v {
				txBytes := common.FromHex(txHex)
				if len(txBytes) > 0 {
					ps.transactions = append(ps.transactions, txBytes)
				}
			}
		case [][]byte:
			ps.transactions = v
		default:
			return nil, fmt.Errorf("invalid transactions type: %T", txs)
		}
	}

	// Parse gasLimit (optional)
	if gl, ok := attrs["gasLimit"]; ok {
		switch v := gl.(type) {
		case uint64:
			ps.gasLimit = v
		case int64:
			ps.gasLimit = uint64(v)
		case float64:
			ps.gasLimit = uint64(v)
		case *uint64:
			if v != nil {
				ps.gasLimit = *v
			}
		default:
			return nil, fmt.Errorf("invalid gasLimit type: %T", gl)
		}
	}

	// Parse withdrawals (optional)
	if w, ok := attrs["withdrawals"]; ok {
		switch v := w.(type) {
		case []*types.Withdrawal:
			ps.withdrawals = v
		case nil:
			// Keep empty slice
		default:
			return nil, fmt.Errorf("invalid withdrawals type: %T", w)
		}
	}

	return ps, nil
}

// GetPayload implements EngineRPCClient.
func (g *gethEngineClient) GetPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	payloadState, ok := g.backend.payloads[payloadID]
	if !ok {
		return nil, fmt.Errorf("unknown payload ID: %s", payloadID.String())
	}

	// Build the block if not already built
	if payloadState.payload == nil {
		startTime := time.Now()
		payload, err := g.buildPayload(ctx, payloadState)
		if err != nil {
			return nil, fmt.Errorf("failed to build payload: %w", err)
		}
		payloadState.payload = payload

		g.logger.Info().
			Str("payload_id", payloadID.String()).
			Uint64("block_number", payload.Number).
			Str("block_hash", payload.BlockHash.Hex()).
			Int("tx_count", len(payload.Transactions)).
			Uint64("gas_used", payload.GasUsed).
			Dur("build_time", time.Since(startTime)).
			Msg("built payload")
	}

	// Remove the payload from pending after retrieval
	delete(g.backend.payloads, payloadID)

	return &engine.ExecutionPayloadEnvelope{
		ExecutionPayload: payloadState.payload,
		BlockValue:       big.NewInt(0),
		BlobsBundle:      &engine.BlobsBundle{},
		Override:         false,
	}, nil
}

// buildPayload constructs an execution payload from the pending state.
func (g *gethEngineClient) buildPayload(ctx context.Context, ps *payloadBuildState) (*engine.ExecutableData, error) {
	parent := g.backend.blockchain.GetBlockByHash(ps.parentHash)
	if parent == nil {
		return nil, fmt.Errorf("parent block not found: %s", ps.parentHash.Hex())
	}

	// Validate block number continuity
	expectedNumber := new(big.Int).Add(parent.Number(), big.NewInt(1))

	// Validate timestamp
	if ps.timestamp <= parent.Time() {
		return nil, fmt.Errorf("invalid timestamp: %d must be greater than parent timestamp %d", ps.timestamp, parent.Time())
	}

	// Calculate base fee for the new block
	var baseFee *big.Int
	if g.backend.chainConfig.IsLondon(expectedNumber) {
		baseFee = calcBaseFee(g.backend.chainConfig, parent.Header())
	}

	gasLimit := ps.gasLimit
	if gasLimit == 0 {
		gasLimit = parent.GasLimit()
	}

	header := &types.Header{
		ParentHash:       ps.parentHash,
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         ps.feeRecipient,
		Root:             common.Hash{}, // Will be set after execution
		TxHash:           types.EmptyTxsHash,
		ReceiptHash:      types.EmptyReceiptsHash,
		Bloom:            types.Bloom{},
		Difficulty:       big.NewInt(0),
		Number:           expectedNumber,
		GasLimit:         gasLimit,
		GasUsed:          0,
		Time:             ps.timestamp,
		Extra:            []byte{},
		MixDigest:        ps.prevRandao,
		Nonce:            types.BlockNonce{},
		BaseFee:          baseFee,
		WithdrawalsHash:  &types.EmptyWithdrawalsHash,
		BlobGasUsed:      new(uint64),
		ExcessBlobGas:    new(uint64),
		ParentBeaconRoot: &common.Hash{},
		RequestsHash:     &types.EmptyRequestsHash,
	}

	// Process transactions
	stateDB, err := g.backend.blockchain.StateAt(parent.Root())
	if err != nil {
		return nil, fmt.Errorf("failed to get parent state: %w", err)
	}

	var (
		txs         types.Transactions
		receipts    []*types.Receipt
		gasUsed     uint64
		txsExecuted int
		txsSkipped  int
	)

	// Create EVM context
	blockContext := core.NewEVMBlockContext(header, g.backend.blockchain, nil)

	// Execute transactions
	gp := new(core.GasPool).AddGas(gasLimit)
	for i, txBytes := range ps.transactions {
		// Check context periodically
		if i%100 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("context cancelled during tx execution: %w", err)
			}
		}

		if len(txBytes) == 0 {
			txsSkipped++
			continue
		}

		var tx types.Transaction
		if err := tx.UnmarshalBinary(txBytes); err != nil {
			g.logger.Debug().
				Int("index", i).
				Err(err).
				Msg("skipping invalid transaction encoding")
			txsSkipped++
			continue
		}

		stateDB.SetTxContext(tx.Hash(), len(txs))

		// Create EVM instance and apply transaction
		receipt, err := applyTransaction(
			g.backend.chainConfig,
			blockContext,
			gp,
			stateDB,
			header,
			&tx,
			&gasUsed,
		)
		if err != nil {
			g.logger.Debug().
				Int("index", i).
				Str("tx_hash", tx.Hash().Hex()).
				Err(err).
				Msg("transaction execution failed, skipping")
			txsSkipped++
			continue
		}

		txs = append(txs, &tx)
		receipts = append(receipts, receipt)
		txsExecuted++
	}

	if txsSkipped > 0 {
		g.logger.Debug().
			Int("executed", txsExecuted).
			Int("skipped", txsSkipped).
			Msg("transaction execution summary")
	}

	// Finalize state
	header.GasUsed = gasUsed
	header.Root = stateDB.IntermediateRoot(g.backend.chainConfig.IsEIP158(header.Number))

	// Calculate transaction and receipt hashes
	header.TxHash = types.DeriveSha(txs, trie.NewListHasher())
	header.ReceiptHash = types.DeriveSha(types.Receipts(receipts), trie.NewListHasher())

	// Calculate bloom filter
	header.Bloom = createBloomFromReceipts(receipts)

	// Calculate withdrawals hash if withdrawals exist
	if len(ps.withdrawals) > 0 {
		wh := types.DeriveSha(types.Withdrawals(ps.withdrawals), trie.NewListHasher())
		header.WithdrawalsHash = &wh
	}

	// Create the block
	block := types.NewBlock(header, &types.Body{
		Transactions: txs,
		Uncles:       nil,
		Withdrawals:  ps.withdrawals,
	}, receipts, trie.NewListHasher())

	// Convert to ExecutableData
	txData := make([][]byte, len(txs))
	for i, tx := range txs {
		data, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tx %d: %w", i, err)
		}
		txData[i] = data
	}

	payload := &engine.ExecutableData{
		ParentHash:    header.ParentHash,
		FeeRecipient:  header.Coinbase,
		StateRoot:     header.Root,
		ReceiptsRoot:  header.ReceiptHash,
		LogsBloom:     header.Bloom[:],
		Random:        header.MixDigest,
		Number:        header.Number.Uint64(),
		GasLimit:      header.GasLimit,
		GasUsed:       header.GasUsed,
		Timestamp:     header.Time,
		ExtraData:     header.Extra,
		BaseFeePerGas: header.BaseFee,
		BlockHash:     block.Hash(),
		Transactions:  txData,
		Withdrawals:   ps.withdrawals,
		BlobGasUsed:   header.BlobGasUsed,
		ExcessBlobGas: header.ExcessBlobGas,
	}

	return payload, nil
}

// NewPayload implements EngineRPCClient.
func (g *gethEngineClient) NewPayload(ctx context.Context, payload *engine.ExecutableData, blobHashes []string, parentBeaconBlockRoot string, executionRequests [][]byte) (*engine.PayloadStatusV1, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	startTime := time.Now()

	// Validate payload
	if payload == nil {
		return nil, errors.New("payload is required")
	}

	// Verify parent exists
	parent := g.backend.blockchain.GetBlockByHash(payload.ParentHash)
	if parent == nil {
		g.logger.Debug().
			Str("parent_hash", payload.ParentHash.Hex()).
			Uint64("block_number", payload.Number).
			Msg("parent block not found, returning SYNCING")
		return &engine.PayloadStatusV1{
			Status: engine.SYNCING,
		}, nil
	}

	// Validate block number
	expectedNumber := parent.NumberU64() + 1
	if payload.Number != expectedNumber {
		g.logger.Warn().
			Uint64("expected", expectedNumber).
			Uint64("got", payload.Number).
			Msg("invalid block number")
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{
			Status:          engine.INVALID,
			LatestValidHash: &parentHash,
		}, nil
	}

	// Validate timestamp
	if payload.Timestamp <= parent.Time() {
		g.logger.Warn().
			Uint64("payload_timestamp", payload.Timestamp).
			Uint64("parent_timestamp", parent.Time()).
			Msg("invalid timestamp")
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{
			Status:          engine.INVALID,
			LatestValidHash: &parentHash,
		}, nil
	}

	// Decode transactions
	var txs types.Transactions
	for i, txData := range payload.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(txData); err != nil {
			g.logger.Warn().
				Int("tx_index", i).
				Err(err).
				Msg("failed to decode transaction")
			parentHash := parent.Hash()
			return &engine.PayloadStatusV1{
				Status:          engine.INVALID,
				LatestValidHash: &parentHash,
			}, nil
		}
		txs = append(txs, &tx)
	}

	gasLimit := payload.GasLimit
	if gasLimit == 0 {
		gasLimit = parent.GasLimit()
	}

	// Reconstruct the header from the payload
	header := &types.Header{
		ParentHash:       payload.ParentHash,
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         payload.FeeRecipient,
		Root:             payload.StateRoot,
		TxHash:           types.DeriveSha(txs, trie.NewListHasher()),
		ReceiptHash:      payload.ReceiptsRoot,
		Bloom:            types.BytesToBloom(payload.LogsBloom),
		Difficulty:       big.NewInt(0),
		Number:           big.NewInt(int64(payload.Number)),
		GasLimit:         gasLimit,
		GasUsed:          payload.GasUsed,
		Time:             payload.Timestamp,
		Extra:            payload.ExtraData,
		MixDigest:        payload.Random,
		Nonce:            types.BlockNonce{},
		BaseFee:          payload.BaseFeePerGas,
		WithdrawalsHash:  &types.EmptyWithdrawalsHash,
		BlobGasUsed:      payload.BlobGasUsed,
		ExcessBlobGas:    payload.ExcessBlobGas,
		ParentBeaconRoot: &common.Hash{},
		RequestsHash:     &types.EmptyRequestsHash,
	}

	if len(payload.Withdrawals) > 0 {
		wh := types.DeriveSha(types.Withdrawals(payload.Withdrawals), trie.NewListHasher())
		header.WithdrawalsHash = &wh
	}

	// Create the block from the payload
	block := types.NewBlock(header, &types.Body{
		Transactions: txs,
		Uncles:       nil,
		Withdrawals:  payload.Withdrawals,
	}, nil, trie.NewListHasher())

	// Verify the block hash matches the payload
	if block.Hash() != payload.BlockHash {
		g.logger.Warn().
			Str("expected", payload.BlockHash.Hex()).
			Str("calculated", block.Hash().Hex()).
			Uint64("block_number", payload.Number).
			Msg("block hash mismatch")
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{
			Status:          engine.INVALID,
			LatestValidHash: &parentHash,
		}, nil
	}

	// Use InsertBlockWithoutSetHead which processes, validates, and commits the block
	// This ensures proper state validation using go-ethereum's internal processor
	_, err := g.backend.blockchain.InsertBlockWithoutSetHead(block, false)
	if err != nil {
		g.logger.Warn().
			Err(err).
			Str("block_hash", block.Hash().Hex()).
			Uint64("block_number", block.NumberU64()).
			Msg("block validation/insertion failed")
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{
			Status:          engine.INVALID,
			LatestValidHash: &parentHash,
		}, nil
	}

	blockHash := block.Hash()

	g.logger.Info().
		Uint64("block_number", block.NumberU64()).
		Str("block_hash", blockHash.Hex()).
		Str("parent_hash", payload.ParentHash.Hex()).
		Int("tx_count", len(txs)).
		Uint64("gas_used", payload.GasUsed).
		Dur("process_time", time.Since(startTime)).
		Msg("new payload validated and inserted")

	return &engine.PayloadStatusV1{
		Status:          engine.VALID,
		LatestValidHash: &blockHash,
	}, nil
}

// HeaderByNumber implements EthRPCClient.
func (g *gethEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	if number == nil {
		// Return current head
		header := g.backend.blockchain.CurrentBlock()
		if header == nil {
			return nil, errors.New("no current block")
		}
		return header, nil
	}

	block := g.backend.blockchain.GetBlockByNumber(number.Uint64())
	if block == nil {
		return nil, fmt.Errorf("block not found at height %d", number.Uint64())
	}
	return block.Header(), nil
}

// GetTxs implements EthRPCClient.
func (g *gethEthClient) GetTxs(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	pending := g.backend.txPool.Pending(txpool.PendingFilter{})
	var result []string
	for _, txs := range pending {
		for _, lazyTx := range txs {
			tx := lazyTx.Tx
			if tx == nil {
				continue
			}
			data, err := tx.MarshalBinary()
			if err != nil {
				g.logger.Debug().
					Str("tx_hash", tx.Hash().Hex()).
					Err(err).
					Msg("failed to marshal pending tx")
				continue
			}
			result = append(result, "0x"+common.Bytes2Hex(data))
		}
	}
	return result, nil
}

// calcBaseFee calculates the base fee for the next block according to EIP-1559.
func calcBaseFee(config *params.ChainConfig, parent *types.Header) *big.Int {
	nextBlockNumber := new(big.Int).Add(parent.Number, big.NewInt(1))

	// If we're before London, return nil
	if !config.IsLondon(nextBlockNumber) {
		return nil
	}

	// Use genesis base fee if this is the first London block
	if !config.IsLondon(parent.Number) {
		return big.NewInt(params.InitialBaseFee)
	}

	// Parent must have base fee
	if parent.BaseFee == nil {
		return big.NewInt(params.InitialBaseFee)
	}

	// Calculate next base fee based on EIP-1559
	var (
		parentGasTarget = parent.GasLimit / 2
		parentGasUsed   = parent.GasUsed
		baseFee         = new(big.Int).Set(parent.BaseFee)
	)

	// Prevent division by zero
	if parentGasTarget == 0 {
		return baseFee
	}

	if parentGasUsed == parentGasTarget {
		return baseFee
	}

	if parentGasUsed > parentGasTarget {
		// Block was more full than target, increase base fee
		gasUsedDelta := new(big.Int).SetUint64(parentGasUsed - parentGasTarget)
		x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
		y := new(big.Int).SetUint64(parentGasTarget)
		z := new(big.Int).Div(x, y)
		baseFeeChangeDenominatorInt := new(big.Int).SetUint64(baseFeeChangeDenominator)
		delta := new(big.Int).Div(z, baseFeeChangeDenominatorInt)
		if delta.Sign() == 0 {
			delta = big.NewInt(1)
		}
		return new(big.Int).Add(baseFee, delta)
	}

	// Block was less full than target, decrease base fee
	gasUsedDelta := new(big.Int).SetUint64(parentGasTarget - parentGasUsed)
	x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
	y := new(big.Int).SetUint64(parentGasTarget)
	z := new(big.Int).Div(x, y)
	baseFeeChangeDenominatorInt := new(big.Int).SetUint64(baseFeeChangeDenominator)
	delta := new(big.Int).Div(z, baseFeeChangeDenominatorInt)
	baseFee = new(big.Int).Sub(baseFee, delta)
	if baseFee.Cmp(big.NewInt(0)) < 0 {
		baseFee = big.NewInt(0)
	}
	return baseFee
}

// applyTransaction executes a transaction and returns the receipt.
func applyTransaction(
	config *params.ChainConfig,
	blockContext vm.BlockContext,
	gp *core.GasPool,
	stateDB *state.StateDB,
	header *types.Header,
	tx *types.Transaction,
	usedGas *uint64,
) (*types.Receipt, error) {
	msg, err := core.TransactionToMessage(tx, types.LatestSigner(config), header.BaseFee)
	if err != nil {
		return nil, fmt.Errorf("failed to convert tx to message: %w", err)
	}

	// Create EVM instance
	txContext := core.NewEVMTxContext(msg)
	evmInstance := vm.NewEVM(blockContext, stateDB, config, vm.Config{})
	evmInstance.SetTxContext(txContext)

	// Apply the transaction
	result, err := core.ApplyMessage(evmInstance, msg, gp)
	if err != nil {
		return nil, fmt.Errorf("failed to apply message: %w", err)
	}

	*usedGas += result.UsedGas

	// Create the receipt
	receipt := &types.Receipt{
		Type:              tx.Type(),
		PostState:         nil,
		CumulativeGasUsed: *usedGas,
		TxHash:            tx.Hash(),
		GasUsed:           result.UsedGas,
		Logs:              stateDB.GetLogs(tx.Hash(), header.Number.Uint64(), common.Hash{}, header.Number.Uint64()),
		BlockNumber:       header.Number,
	}

	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}

	// Set the receipt logs bloom
	receipt.Bloom = types.CreateBloom(receipt)

	// Set contract address if this was a contract creation
	if msg.To == nil {
		receipt.ContractAddress = evmInstance.TxContext.Origin
	}

	return receipt, nil
}

// createBloomFromReceipts creates a bloom filter from multiple receipts.
func createBloomFromReceipts(receipts []*types.Receipt) types.Bloom {
	var bin types.Bloom
	for _, receipt := range receipts {
		bloom := types.CreateBloom(receipt)
		for i := range bin {
			bin[i] |= bloom[i]
		}
	}
	return bin
}
