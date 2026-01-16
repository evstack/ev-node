package evm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
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
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
)

var (
	_ EngineRPCClient = (*gethEngineClient)(nil)
	_ EthRPCClient    = (*gethEthClient)(nil)
)

// GethBackend holds the in-process geth components.
type GethBackend struct {
	db          ethdb.Database
	chainConfig *params.ChainConfig
	blockchain  *core.BlockChain
	txPool      *txpool.TxPool

	mu sync.Mutex
	// payloadBuilding tracks in-flight payload builds
	payloads      map[engine.PayloadID]*payloadBuildState
	nextPayloadID uint64
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
//
// This is useful for:
// - Testing without needing to run a separate geth/reth process
// - Embedded rollup nodes that want a single binary
// - Development and debugging with full control over the EVM
//
// Parameters:
//   - genesis: The genesis configuration for the chain
//   - feeRecipient: Address to receive transaction fees
//   - db: Datastore for execution metadata (crash recovery)
//   - logger: Logger for the client
//
// Returns an EngineClient that behaves identically to the RPC-based client
// but executes everything in-process.
func NewEngineExecutionClientWithGeth(
	genesis *core.Genesis,
	feeRecipient common.Address,
	db ds.Batching,
	logger zerolog.Logger,
) (*EngineClient, error) {
	if db == nil {
		return nil, errors.New("db is required for EVM execution client")
	}
	if genesis == nil {
		return nil, errors.New("genesis configuration is required")
	}

	backend, err := newGethBackend(genesis, db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create geth backend: %w", err)
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

// newGethBackend creates a new in-process geth backend.
func newGethBackend(genesis *core.Genesis, db ds.Batching, logger zerolog.Logger) (*GethBackend, error) {
	ethdb := rawdb.NewDatabase(db)

	// Create trie database
	trieDB := triedb.NewDatabase(ethdb, nil)

	// Ensure blobSchedule is set if Cancun/Prague are enabled
	// This is required by go-ethereum v1.16+
	if genesis.Config != nil && genesis.Config.BlobScheduleConfig == nil {
		// Check if Cancun or Prague are enabled (time-based forks)
		if genesis.Config.CancunTime != nil || genesis.Config.PragueTime != nil {
			genesis.Config.BlobScheduleConfig = &params.BlobScheduleConfig{
				Cancun: params.DefaultCancunBlobConfig,
				Prague: params.DefaultPragueBlobConfig,
			}
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

	// Create the consensus engine (beacon/PoS)
	consensusEngine := beacon.New(nil)

	// Create blockchain config
	bcConfig := core.DefaultConfig().WithStateScheme(rawdb.HashScheme)

	// Create the blockchain
	blockchain, err := core.NewBlockChain(ethdb, genesis, consensusEngine, bcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain: %w", err)
	}

	backend := &GethBackend{
		db:          ethdb,
		chainConfig: chainConfig,
		blockchain:  blockchain,
		payloads:    make(map[engine.PayloadID]*payloadBuildState),
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

// Close shuts down the geth backend.
func (b *GethBackend) Close() error {
	if b.txPool != nil {
		b.txPool.Close()
	}
	if b.blockchain != nil {
		b.blockchain.Stop()
	}
	if b.db != nil {
		b.db.Close()
	}
	return nil
}

// ForkchoiceUpdated implements EngineRPCClient.
func (g *gethEngineClient) ForkchoiceUpdated(ctx context.Context, fcState engine.ForkchoiceStateV1, attrs map[string]any) (*engine.ForkChoiceResponse, error) {
	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	// Validate the forkchoice state
	headBlock := g.backend.blockchain.GetBlockByHash(fcState.HeadBlockHash)
	if headBlock == nil {
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

		// Generate payload ID
		g.backend.nextPayloadID++
		var payloadID engine.PayloadID
		payloadID[0] = byte(g.backend.nextPayloadID >> 56)
		payloadID[1] = byte(g.backend.nextPayloadID >> 48)
		payloadID[2] = byte(g.backend.nextPayloadID >> 40)
		payloadID[3] = byte(g.backend.nextPayloadID >> 32)
		payloadID[4] = byte(g.backend.nextPayloadID >> 24)
		payloadID[5] = byte(g.backend.nextPayloadID >> 16)
		payloadID[6] = byte(g.backend.nextPayloadID >> 8)
		payloadID[7] = byte(g.backend.nextPayloadID)

		g.backend.payloads[payloadID] = payloadState
		response.PayloadID = &payloadID

		g.logger.Info().
			Str("payload_id", payloadID.String()).
			Uint64("timestamp", payloadState.timestamp).
			Int("tx_count", len(payloadState.transactions)).
			Msg("started payload build")
	}

	return response, nil
}

// parsePayloadAttributes extracts payload attributes from the map format.
func (g *gethEngineClient) parsePayloadAttributes(parentHash common.Hash, attrs map[string]any) (*payloadBuildState, error) {
	ps := &payloadBuildState{
		parentHash:  parentHash,
		withdrawals: []*types.Withdrawal{},
	}

	// Parse timestamp
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
	}

	// Parse prevRandao
	if pr, ok := attrs["prevRandao"]; ok {
		switch v := pr.(type) {
		case common.Hash:
			ps.prevRandao = v
		case string:
			ps.prevRandao = common.HexToHash(v)
		default:
			return nil, fmt.Errorf("invalid prevRandao type: %T", pr)
		}
	}

	// Parse suggestedFeeRecipient
	if fr, ok := attrs["suggestedFeeRecipient"]; ok {
		switch v := fr.(type) {
		case common.Address:
			ps.feeRecipient = v
		case string:
			ps.feeRecipient = common.HexToAddress(v)
		default:
			return nil, fmt.Errorf("invalid suggestedFeeRecipient type: %T", fr)
		}
	}

	// Parse transactions
	if txs, ok := attrs["transactions"]; ok {
		switch v := txs.(type) {
		case []string:
			ps.transactions = make([][]byte, len(v))
			for i, txHex := range v {
				ps.transactions[i] = common.FromHex(txHex)
			}
		case [][]byte:
			ps.transactions = v
		}
	}

	// Parse gasLimit
	if gl, ok := attrs["gasLimit"]; ok {
		switch v := gl.(type) {
		case uint64:
			ps.gasLimit = v
		case int64:
			ps.gasLimit = uint64(v)
		case float64:
			ps.gasLimit = uint64(v)
		default:
			return nil, fmt.Errorf("invalid gasLimit type: %T", gl)
		}
	}

	// Parse withdrawals (optional)
	if w, ok := attrs["withdrawals"]; ok {
		if withdrawals, ok := w.([]*types.Withdrawal); ok {
			ps.withdrawals = withdrawals
		}
	}

	return ps, nil
}

// GetPayload implements EngineRPCClient.
func (g *gethEngineClient) GetPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	payloadState, ok := g.backend.payloads[payloadID]
	if !ok {
		return nil, fmt.Errorf("unknown payload ID: %s", payloadID.String())
	}

	// Build the block if not already built
	if payloadState.payload == nil {
		payload, err := g.buildPayload(payloadState)
		if err != nil {
			return nil, fmt.Errorf("failed to build payload: %w", err)
		}
		payloadState.payload = payload
	}

	return &engine.ExecutionPayloadEnvelope{
		ExecutionPayload: payloadState.payload,
		BlockValue:       big.NewInt(0),
		BlobsBundle:      &engine.BlobsBundle{},
		Override:         false,
	}, nil
}

// buildPayload constructs an execution payload from the pending state.
func (g *gethEngineClient) buildPayload(ps *payloadBuildState) (*engine.ExecutableData, error) {
	parent := g.backend.blockchain.GetBlockByHash(ps.parentHash)
	if parent == nil {
		return nil, fmt.Errorf("parent block not found: %s", ps.parentHash.Hex())
	}

	// Calculate base fee for the new block
	var baseFee *big.Int
	if g.backend.chainConfig.IsLondon(new(big.Int).Add(parent.Number(), big.NewInt(1))) {
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
		Number:           new(big.Int).Add(parent.Number(), big.NewInt(1)),
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
		txs      types.Transactions
		receipts []*types.Receipt
		gasUsed  uint64
	)

	// Create EVM context
	blockContext := core.NewEVMBlockContext(header, g.backend.blockchain, nil)

	// Execute transactions
	gp := new(core.GasPool).AddGas(gasLimit)
	for i, txBytes := range ps.transactions {
		if len(txBytes) == 0 {
			continue
		}

		var tx types.Transaction
		if err := tx.UnmarshalBinary(txBytes); err != nil {
			g.logger.Debug().
				Int("index", i).
				Err(err).
				Msg("skipping invalid transaction")
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
			continue
		}

		txs = append(txs, &tx)
		receipts = append(receipts, receipt)
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
			return nil, fmt.Errorf("failed to marshal tx: %w", err)
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
	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	// Verify parent exists
	parent := g.backend.blockchain.GetBlockByHash(payload.ParentHash)
	if parent == nil {
		return &engine.PayloadStatusV1{
			Status: engine.SYNCING,
		}, nil
	}

	// Decode transactions
	var txs types.Transactions
	for _, txData := range payload.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(txData); err != nil {
			return &engine.PayloadStatusV1{
				Status: engine.INVALID,
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
	return &engine.PayloadStatusV1{
		Status:          engine.VALID,
		LatestValidHash: &blockHash,
	}, nil
}

// HeaderByNumber implements EthRPCClient.
func (g *gethEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
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
	pending := g.backend.txPool.Pending(txpool.PendingFilter{})
	var result []string
	for _, txs := range pending {
		for _, lazyTx := range txs {
			// Resolve the lazy transaction to get the actual transaction
			tx := lazyTx.Tx
			if tx == nil {
				continue
			}
			data, err := tx.MarshalBinary()
			if err != nil {
				continue
			}
			result = append(result, "0x"+common.Bytes2Hex(data))
		}
	}
	return result, nil
}

// calcBaseFee calculates the base fee for the next block.
func calcBaseFee(config *params.ChainConfig, parent *types.Header) *big.Int {
	// If we're before London, return nil
	if !config.IsLondon(new(big.Int).Add(parent.Number, big.NewInt(1))) {
		return nil
	}

	// Use genesis base fee if this is the first London block
	if !config.IsLondon(parent.Number) {
		return big.NewInt(params.InitialBaseFee)
	}

	// Calculate next base fee based on EIP-1559
	var (
		parentGasTarget = parent.GasLimit / 2
		parentGasUsed   = parent.GasUsed
		baseFee         = new(big.Int).Set(parent.BaseFee)
	)

	if parentGasUsed == parentGasTarget {
		return baseFee
	}

	if parentGasUsed > parentGasTarget {
		// Block was more full than target, increase base fee
		gasUsedDelta := new(big.Int).SetUint64(parentGasUsed - parentGasTarget)
		x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
		y := new(big.Int).SetUint64(parentGasTarget)
		z := new(big.Int).Div(x, y)
		baseFeeChangeDenominator := new(big.Int).SetUint64(8)
		delta := new(big.Int).Div(z, baseFeeChangeDenominator)
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
	baseFeeChangeDenominator := new(big.Int).SetUint64(8)
	delta := new(big.Int).Div(z, baseFeeChangeDenominator)
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
		return nil, err
	}

	// Create EVM instance
	txContext := core.NewEVMTxContext(msg)
	evmInstance := vm.NewEVM(blockContext, stateDB, config, vm.Config{})
	evmInstance.SetTxContext(txContext)

	// Apply the transaction
	result, err := core.ApplyMessage(evmInstance, msg, gp)
	if err != nil {
		return nil, err
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
		receipt.ContractAddress = evmInstance.Origin
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
