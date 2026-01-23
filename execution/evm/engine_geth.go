package evm

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
)

var (
	_ EngineRPCClient = (*gethEngineClient)(nil)
	_ EthRPCClient    = (*gethEthClient)(nil)
)

// baseFeeChangeDenominator is the EIP-1559 base fee change denominator.
const baseFeeChangeDenominator = 8

// payloadTTL is how long a payload can remain in the map before being cleaned up.
const payloadTTL = 60 * time.Second

// maxPayloads is the maximum number of payloads to keep in memory.
const maxPayloads = 10

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

// payloadBuildState tracks the state of a payload being built.
type payloadBuildState struct {
	parentHash   common.Hash
	timestamp    uint64
	prevRandao   common.Hash
	feeRecipient common.Address
	withdrawals  []*types.Withdrawal
	transactions [][]byte
	gasLimit     uint64

	// createdAt tracks when this payload was created for TTL cleanup
	createdAt time.Time

	// built payload (populated after getPayload)
	payload *engine.ExecutableData

	// buildErr stores any error that occurred during payload build
	buildErr error
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
	// Use sovereign beacon consensus that allows equal timestamps for subsecond block times
	consensusEngine := newSovereignBeacon()

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

		// Generate deterministic payload ID from attributes
		payloadID := g.generatePayloadID(payloadState)

		// Check if we already have this payload (idempotency)
		if existing, ok := g.backend.payloads[payloadID]; ok {
			// Reuse existing payload if it hasn't errored
			if existing.buildErr == nil {
				response.PayloadID = &payloadID
				g.logger.Debug().
					Str("payload_id", payloadID.String()).
					Msg("reusing existing payload")
				return response, nil
			}
			// Previous build failed, remove it and try again
			delete(g.backend.payloads, payloadID)
		}

		// Clean up old payloads before adding new one
		g.cleanupStalePayloads()

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

// generatePayloadID creates a deterministic payload ID from the payload attributes.
// This ensures that the same attributes always produce the same ID for idempotency.
func (g *gethEngineClient) generatePayloadID(ps *payloadBuildState) engine.PayloadID {
	h := sha256.New()
	h.Write(ps.parentHash[:])
	binary.Write(h, binary.BigEndian, ps.timestamp)
	h.Write(ps.prevRandao[:])
	h.Write(ps.feeRecipient[:])
	binary.Write(h, binary.BigEndian, ps.gasLimit)
	// Include transaction count and first tx hash for uniqueness
	binary.Write(h, binary.BigEndian, uint64(len(ps.transactions)))
	for _, tx := range ps.transactions {
		h.Write(tx)
	}
	sum := h.Sum(nil)
	var id engine.PayloadID
	copy(id[:], sum[:8])
	return id
}

// cleanupStalePayloads removes payloads that have exceeded their TTL or when we have too many.
func (g *gethEngineClient) cleanupStalePayloads() {
	now := time.Now()
	var staleIDs []engine.PayloadID

	// Find stale payloads
	for id, ps := range g.backend.payloads {
		if now.Sub(ps.createdAt) > payloadTTL {
			staleIDs = append(staleIDs, id)
		}
	}

	// Remove stale payloads
	for _, id := range staleIDs {
		delete(g.backend.payloads, id)
		g.logger.Debug().
			Str("payload_id", id.String()).
			Msg("cleaned up stale payload")
	}

	// If still too many payloads, remove oldest ones
	for len(g.backend.payloads) >= maxPayloads {
		var oldestID engine.PayloadID
		var oldestTime time.Time
		first := true
		for id, ps := range g.backend.payloads {
			if first || ps.createdAt.Before(oldestTime) {
				oldestID = id
				oldestTime = ps.createdAt
				first = false
			}
		}
		if !first {
			delete(g.backend.payloads, oldestID)
			g.logger.Debug().
				Str("payload_id", oldestID.String()).
				Msg("evicted oldest payload due to limit")
		}
	}
}

// parsePayloadAttributes extracts payload attributes from the map format.
func (g *gethEngineClient) parsePayloadAttributes(parentHash common.Hash, attrs map[string]any) (*payloadBuildState, error) {
	ps := &payloadBuildState{
		parentHash:  parentHash,
		withdrawals: []*types.Withdrawal{},
		createdAt:   time.Now(),
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

	// Return cached error if previous build failed
	if payloadState.buildErr != nil {
		delete(g.backend.payloads, payloadID)
		return nil, fmt.Errorf("payload build previously failed: %w", payloadState.buildErr)
	}

	// Build the payload if not already built
	if payloadState.payload == nil {
		buildStartTime := time.Now()
		payload, err := g.buildPayload(ctx, payloadState)
		if err != nil {
			// Cache the error so we don't retry on the same payload
			payloadState.buildErr = err
			return nil, fmt.Errorf("failed to build payload: %w", err)
		}
		payloadState.payload = payload

		g.logger.Info().
			Str("payload_id", payloadID.String()).
			Uint64("block_number", payload.Number).
			Str("block_hash", payload.BlockHash.Hex()).
			Int("tx_count", len(payload.Transactions)).
			Uint64("gas_used", payload.GasUsed).
			Dur("build_time", time.Since(buildStartTime)).
			Msg("built payload")
	}

	// Remove the payload from pending after retrieval - caller has it now
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

	if ps.timestamp < parent.Time() {
		return nil, fmt.Errorf("invalid timestamp: %d must be >= parent timestamp %d", ps.timestamp, parent.Time())
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

	// Process withdrawals (EIP-4895) - credit ETH to withdrawal recipients
	// Withdrawals are processed after all transactions, crediting the specified
	// amount (in Gwei) to each recipient address.
	if len(ps.withdrawals) > 0 {
		for _, withdrawal := range ps.withdrawals {
			// Withdrawal amount is in Gwei, convert to Wei (multiply by 1e9)
			amount := new(big.Int).SetUint64(withdrawal.Amount)
			amount.Mul(amount, big.NewInt(params.GWei))
			stateDB.AddBalance(withdrawal.Address, uint256.MustFromBig(amount), tracing.BalanceIncreaseWithdrawal)
		}
		g.logger.Debug().
			Int("count", len(ps.withdrawals)).
			Msg("processed withdrawals")
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

	validationStart := time.Now()

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
	if payload.Timestamp < parent.Time() {
		g.logger.Warn().
			Uint64("payload_timestamp", payload.Timestamp).
			Uint64("parent_timestamp", parent.Time()).
			Msg("invalid timestamp: must be >= parent timestamp")
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
		Dur("process_time", time.Since(validationStart)).
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
		receipt.ContractAddress = crypto.CreateAddress(msg.From, tx.Nonce())
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
