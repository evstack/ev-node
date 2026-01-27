package evm

import (
	"context"
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

// GethBackend is the in-process geth execution engine.
type GethBackend struct {
	db          ethdb.Database
	chainConfig *params.ChainConfig
	blockchain  *core.BlockChain
	txPool      *txpool.TxPool

	mu             sync.Mutex
	pendingPayload *pendingPayload

	rpcServer   *rpc.Server
	httpServer  *http.Server
	rpcListener net.Listener
	logger      zerolog.Logger
}

// pendingPayload represents a single in-flight payload build.
type pendingPayload struct {
	id           engine.PayloadID
	parentHash   common.Hash
	timestamp    uint64
	prevRandao   common.Hash
	feeRecipient common.Address
	withdrawals  []*types.Withdrawal
	transactions [][]byte
	gasLimit     uint64
	built        *engine.ExecutableData
}

type gethEngineClient struct {
	backend *GethBackend
	logger  zerolog.Logger
}

type gethEthClient struct {
	backend *GethBackend
	logger  zerolog.Logger
}

// NewEngineExecutionClientWithGeth creates an EngineClient using in-process geth.
func NewEngineExecutionClientWithGeth(
	genesis *core.Genesis,
	feeRecipient common.Address,
	db ds.Batching,
	rpcAddress string,
	logger zerolog.Logger,
) (*EngineClient, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	if genesis == nil || genesis.Config == nil {
		return nil, errors.New("genesis configuration is required")
	}

	backend, err := newGethBackend(genesis, db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create geth backend: %w", err)
	}

	if rpcAddress != "" {
		if err := backend.StartRPCServer(rpcAddress); err != nil {
			backend.Close()
			return nil, fmt.Errorf("failed to start RPC server: %w", err)
		}
	}

	genesisBlock := backend.blockchain.Genesis()
	genesisHash := genesisBlock.Hash()

	logger.Info().
		Str("genesis_hash", genesisHash.Hex()).
		Str("chain_id", genesis.Config.ChainID.String()).
		Msg("created in-process geth execution client")

	return &EngineClient{
		engineClient:              &gethEngineClient{backend: backend, logger: logger.With().Str("component", "geth-engine").Logger()},
		ethClient:                 &gethEthClient{backend: backend, logger: logger.With().Str("component", "geth-eth").Logger()},
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

func newGethBackend(genesis *core.Genesis, db ds.Batching, logger zerolog.Logger) (*GethBackend, error) {
	ethdb := rawdb.NewDatabase(&wrapper{db})
	trieDB := triedb.NewDatabase(ethdb, nil)

	// Auto-populate blob config for Cancun/Prague if needed
	if genesis.Config != nil && genesis.Config.BlobScheduleConfig == nil {
		if genesis.Config.CancunTime != nil || genesis.Config.PragueTime != nil {
			genesis.Config.BlobScheduleConfig = &params.BlobScheduleConfig{
				Cancun: params.DefaultCancunBlobConfig,
				Prague: params.DefaultPragueBlobConfig,
			}
		}
	}

	chainConfig, genesisHash, _, err := core.SetupGenesisBlockWithOverride(ethdb, trieDB, genesis, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to setup genesis: %w", err)
	}

	logger.Info().Str("genesis_hash", genesisHash.Hex()).Msg("initialized genesis")

	bcConfig := core.DefaultConfig().WithStateScheme(rawdb.HashScheme)
	blockchain, err := core.NewBlockChain(ethdb, genesis, newSovereignBeacon(), bcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain: %w", err)
	}

	if head := blockchain.CurrentBlock(); head != nil {
		logger.Info().Uint64("height", head.Number.Uint64()).Msg("resuming from chain state")
	}

	backend := &GethBackend{
		db:          ethdb,
		chainConfig: chainConfig,
		blockchain:  blockchain,
		logger:      logger,
	}

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

// Close shuts down the backend.
func (b *GethBackend) Close() error {
	b.logger.Info().Msg("shutting down geth backend")

	if b.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = b.httpServer.Shutdown(ctx)
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
		return b.db.Close()
	}
	return nil
}

// ForkchoiceUpdated handles forkchoice updates and optionally starts payload building.
func (g *gethEngineClient) ForkchoiceUpdated(ctx context.Context, fcState engine.ForkchoiceStateV1, attrs map[string]any) (*engine.ForkChoiceResponse, error) {
	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	headBlock := g.backend.blockchain.GetBlockByHash(fcState.HeadBlockHash)
	if headBlock == nil {
		return &engine.ForkChoiceResponse{
			PayloadStatus: engine.PayloadStatusV1{Status: engine.SYNCING},
		}, nil
	}

	if _, err := g.backend.blockchain.SetCanonical(headBlock); err != nil {
		return nil, fmt.Errorf("failed to set canonical head: %w", err)
	}

	response := &engine.ForkChoiceResponse{
		PayloadStatus: engine.PayloadStatusV1{
			Status:          engine.VALID,
			LatestValidHash: &fcState.HeadBlockHash,
		},
	}

	if attrs != nil {
		payload, err := g.parsePayloadAttributes(fcState.HeadBlockHash, attrs)
		if err != nil {
			return nil, fmt.Errorf("invalid payload attributes: %w", err)
		}
		g.backend.pendingPayload = payload
		response.PayloadID = &payload.id

		g.logger.Info().
			Str("payload_id", payload.id.String()).
			Str("parent", fcState.HeadBlockHash.Hex()).
			Uint64("timestamp", payload.timestamp).
			Int("txs", len(payload.transactions)).
			Msg("started payload build")
	}

	return response, nil
}

func (g *gethEngineClient) parsePayloadAttributes(parentHash common.Hash, attrs map[string]any) (*pendingPayload, error) {
	p := &pendingPayload{
		parentHash:  parentHash,
		withdrawals: []*types.Withdrawal{},
	}

	// Generate simple sequential payload ID
	p.id = engine.PayloadID{byte(time.Now().UnixNano() & 0xFF)}

	// Timestamp (required)
	if ts, ok := attrs["timestamp"]; ok {
		switch v := ts.(type) {
		case int64:
			p.timestamp = uint64(v)
		case uint64:
			p.timestamp = v
		case float64:
			p.timestamp = uint64(v)
		default:
			return nil, fmt.Errorf("invalid timestamp type: %T", ts)
		}
	} else {
		return nil, errors.New("timestamp required")
	}

	// PrevRandao
	if pr, ok := attrs["prevRandao"]; ok {
		switch v := pr.(type) {
		case common.Hash:
			p.prevRandao = v
		case string:
			p.prevRandao = common.HexToHash(v)
		case []byte:
			p.prevRandao = common.BytesToHash(v)
		}
	}

	// Fee recipient (required)
	if fr, ok := attrs["suggestedFeeRecipient"]; ok {
		switch v := fr.(type) {
		case common.Address:
			p.feeRecipient = v
		case string:
			p.feeRecipient = common.HexToAddress(v)
		case []byte:
			p.feeRecipient = common.BytesToAddress(v)
		}
	} else {
		return nil, errors.New("suggestedFeeRecipient required")
	}

	// Transactions
	if txs, ok := attrs["transactions"]; ok {
		switch v := txs.(type) {
		case []string:
			for _, txHex := range v {
				if txBytes := common.FromHex(txHex); len(txBytes) > 0 {
					p.transactions = append(p.transactions, txBytes)
				}
			}
		case [][]byte:
			p.transactions = v
		}
	}

	// Gas limit
	if gl, ok := attrs["gasLimit"]; ok {
		switch v := gl.(type) {
		case uint64:
			p.gasLimit = v
		case int64:
			p.gasLimit = uint64(v)
		case float64:
			p.gasLimit = uint64(v)
		case *uint64:
			if v != nil {
				p.gasLimit = *v
			}
		}
	}

	// Withdrawals
	if w, ok := attrs["withdrawals"]; ok {
		if withdrawals, ok := w.([]*types.Withdrawal); ok {
			p.withdrawals = withdrawals
		}
	}

	return p, nil
}

// GetPayload builds and returns the pending payload.
func (g *gethEngineClient) GetPayload(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	p := g.backend.pendingPayload
	if p == nil || p.id != payloadID {
		return nil, fmt.Errorf("unknown payload ID: %s", payloadID.String())
	}

	if p.built == nil {
		built, err := g.buildPayload(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("failed to build payload: %w", err)
		}
		p.built = built

		g.logger.Info().
			Uint64("number", built.Number).
			Str("hash", built.BlockHash.Hex()).
			Int("txs", len(built.Transactions)).
			Uint64("gas_used", built.GasUsed).
			Msg("built payload")
	}

	// Clear pending after retrieval
	g.backend.pendingPayload = nil

	return &engine.ExecutionPayloadEnvelope{
		ExecutionPayload: p.built,
		BlockValue:       big.NewInt(0),
		BlobsBundle:      &engine.BlobsBundle{},
	}, nil
}

func (g *gethEngineClient) buildPayload(ctx context.Context, p *pendingPayload) (*engine.ExecutableData, error) {
	parent := g.backend.blockchain.GetBlockByHash(p.parentHash)
	if parent == nil {
		return nil, fmt.Errorf("parent not found: %s", p.parentHash.Hex())
	}

	if p.timestamp < parent.Time() {
		return nil, fmt.Errorf("timestamp %d < parent %d", p.timestamp, parent.Time())
	}

	number := new(big.Int).Add(parent.Number(), big.NewInt(1))
	gasLimit := p.gasLimit
	if gasLimit == 0 {
		gasLimit = parent.GasLimit()
	}

	var baseFee *big.Int
	if g.backend.chainConfig.IsLondon(number) {
		baseFee = calcBaseFee(g.backend.chainConfig, parent.Header())
	}

	header := &types.Header{
		ParentHash:       p.parentHash,
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         p.feeRecipient,
		Number:           number,
		GasLimit:         gasLimit,
		Time:             p.timestamp,
		MixDigest:        p.prevRandao,
		Difficulty:       big.NewInt(0),
		BaseFee:          baseFee,
		WithdrawalsHash:  &types.EmptyWithdrawalsHash,
		BlobGasUsed:      new(uint64),
		ExcessBlobGas:    new(uint64),
		ParentBeaconRoot: &common.Hash{},
		RequestsHash:     &types.EmptyRequestsHash,
	}

	stateDB, err := g.backend.blockchain.StateAt(parent.Root())
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	var (
		txs      types.Transactions
		receipts []*types.Receipt
		gasUsed  uint64
	)

	blockContext := core.NewEVMBlockContext(header, g.backend.blockchain, nil)
	gp := new(core.GasPool).AddGas(gasLimit)

	for _, txBytes := range p.transactions {
		if len(txBytes) == 0 {
			continue
		}

		var tx types.Transaction
		if err := tx.UnmarshalBinary(txBytes); err != nil {
			continue
		}

		stateDB.SetTxContext(tx.Hash(), len(txs))
		receipt, err := applyTransaction(g.backend.chainConfig, blockContext, gp, stateDB, header, &tx, &gasUsed)
		if err != nil {
			g.logger.Debug().Str("tx", tx.Hash().Hex()).Err(err).Msg("tx failed")
			continue
		}

		txs = append(txs, &tx)
		receipts = append(receipts, receipt)
	}

	// Process withdrawals
	for _, w := range p.withdrawals {
		amount := new(big.Int).SetUint64(w.Amount)
		amount.Mul(amount, big.NewInt(params.GWei))
		stateDB.AddBalance(w.Address, uint256.MustFromBig(amount), tracing.BalanceIncreaseWithdrawal)
	}

	header.GasUsed = gasUsed
	header.Root = stateDB.IntermediateRoot(g.backend.chainConfig.IsEIP158(number))
	header.TxHash = types.DeriveSha(txs, trie.NewListHasher())
	header.ReceiptHash = types.DeriveSha(types.Receipts(receipts), trie.NewListHasher())
	header.Bloom = createBloom(receipts)

	if len(p.withdrawals) > 0 {
		wh := types.DeriveSha(types.Withdrawals(p.withdrawals), trie.NewListHasher())
		header.WithdrawalsHash = &wh
	}

	block := types.NewBlock(header, &types.Body{
		Transactions: txs,
		Withdrawals:  p.withdrawals,
	}, receipts, trie.NewListHasher())

	txData := make([][]byte, len(txs))
	for i, tx := range txs {
		txData[i], _ = tx.MarshalBinary()
	}

	return &engine.ExecutableData{
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
		Withdrawals:   p.withdrawals,
		BlobGasUsed:   header.BlobGasUsed,
		ExcessBlobGas: header.ExcessBlobGas,
	}, nil
}

// NewPayload validates and inserts a new block.
func (g *gethEngineClient) NewPayload(ctx context.Context, payload *engine.ExecutableData, blobHashes []string, parentBeaconBlockRoot string, executionRequests [][]byte) (*engine.PayloadStatusV1, error) {
	g.backend.mu.Lock()
	defer g.backend.mu.Unlock()

	if payload == nil {
		return nil, errors.New("payload required")
	}

	parent := g.backend.blockchain.GetBlockByHash(payload.ParentHash)
	if parent == nil {
		return &engine.PayloadStatusV1{Status: engine.SYNCING}, nil
	}

	if payload.Number != parent.NumberU64()+1 {
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &parentHash}, nil
	}

	if payload.Timestamp < parent.Time() {
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &parentHash}, nil
	}

	var txs types.Transactions
	for _, txData := range payload.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(txData); err != nil {
			parentHash := parent.Hash()
			return &engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &parentHash}, nil
		}
		txs = append(txs, &tx)
	}

	gasLimit := payload.GasLimit
	if gasLimit == 0 {
		gasLimit = parent.GasLimit()
	}

	// Build the header from payload data
	// NOTE: We must set TxHash and ReceiptHash from the payload, not recompute them,
	// because types.NewBlock would overwrite them based on the passed transactions/receipts.
	withdrawalsHash := types.EmptyWithdrawalsHash
	if len(payload.Withdrawals) > 0 {
		withdrawalsHash = types.DeriveSha(types.Withdrawals(payload.Withdrawals), trie.NewListHasher())
	}

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
		BaseFee:          payload.BaseFeePerGas,
		WithdrawalsHash:  &withdrawalsHash,
		BlobGasUsed:      payload.BlobGasUsed,
		ExcessBlobGas:    payload.ExcessBlobGas,
		ParentBeaconRoot: &common.Hash{},
		RequestsHash:     &types.EmptyRequestsHash,
	}

	// Compute block hash directly from header - don't use types.NewBlock which overwrites hashes
	block := types.NewBlockWithHeader(header).WithBody(types.Body{
		Transactions: txs,
		Withdrawals:  payload.Withdrawals,
	})

	if block.Hash() != payload.BlockHash {
		g.logger.Warn().
			Str("expected", payload.BlockHash.Hex()).
			Str("got", block.Hash().Hex()).
			Msg("block hash mismatch")
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &parentHash}, nil
	}

	if _, err := g.backend.blockchain.InsertBlockWithoutSetHead(block, false); err != nil {
		g.logger.Warn().Err(err).Str("hash", block.Hash().Hex()).Msg("block insertion failed")
		parentHash := parent.Hash()
		return &engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &parentHash}, nil
	}

	blockHash := block.Hash()
	g.logger.Info().
		Uint64("number", block.NumberU64()).
		Str("hash", blockHash.Hex()).
		Int("txs", len(txs)).
		Msg("payload validated")

	return &engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &blockHash}, nil
}

// HeaderByNumber returns a header by number.
func (g *gethEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number == nil {
		header := g.backend.blockchain.CurrentBlock()
		if header == nil {
			return nil, errors.New("no current block")
		}
		return header, nil
	}
	block := g.backend.blockchain.GetBlockByNumber(number.Uint64())
	if block == nil {
		return nil, fmt.Errorf("block %d not found", number.Uint64())
	}
	return block.Header(), nil
}

// GetTxs returns pending transactions.
func (g *gethEthClient) GetTxs(ctx context.Context) ([]string, error) {
	pending := g.backend.txPool.Pending(txpool.PendingFilter{})
	var result []string
	for _, txs := range pending {
		for _, lazyTx := range txs {
			if tx := lazyTx.Tx; tx != nil {
				if data, err := tx.MarshalBinary(); err == nil {
					result = append(result, "0x"+common.Bytes2Hex(data))
				}
			}
		}
	}
	return result, nil
}

// calcBaseFee calculates EIP-1559 base fee.
func calcBaseFee(config *params.ChainConfig, parent *types.Header) *big.Int {
	next := new(big.Int).Add(parent.Number, big.NewInt(1))
	if !config.IsLondon(next) {
		return nil
	}
	if !config.IsLondon(parent.Number) || parent.BaseFee == nil {
		return big.NewInt(params.InitialBaseFee)
	}

	target := parent.GasLimit / 2
	if target == 0 {
		return new(big.Int).Set(parent.BaseFee)
	}

	baseFee := new(big.Int).Set(parent.BaseFee)
	if parent.GasUsed == target {
		return baseFee
	}

	if parent.GasUsed > target {
		delta := new(big.Int).SetUint64(parent.GasUsed - target)
		delta.Mul(delta, parent.BaseFee)
		delta.Div(delta, new(big.Int).SetUint64(target))
		delta.Div(delta, big.NewInt(8))
		if delta.Sign() == 0 {
			delta = big.NewInt(1)
		}
		return baseFee.Add(baseFee, delta)
	}

	delta := new(big.Int).SetUint64(target - parent.GasUsed)
	delta.Mul(delta, parent.BaseFee)
	delta.Div(delta, new(big.Int).SetUint64(target))
	delta.Div(delta, big.NewInt(8))
	baseFee.Sub(baseFee, delta)
	if baseFee.Sign() < 0 {
		return big.NewInt(0)
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

	evm := vm.NewEVM(blockContext, stateDB, config, vm.Config{})
	evm.SetTxContext(core.NewEVMTxContext(msg))

	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	*usedGas += result.UsedGas

	receipt := &types.Receipt{
		Type:              tx.Type(),
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

	receipt.Bloom = types.CreateBloom(receipt)

	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(msg.From, tx.Nonce())
	}

	return receipt, nil
}

func createBloom(receipts []*types.Receipt) types.Bloom {
	var bloom types.Bloom
	for _, r := range receipts {
		b := types.CreateBloom(r)
		for i := range bloom {
			bloom[i] |= b[i]
		}
	}
	return bloom
}
