package evm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/core/execution"
	"github.com/evstack/ev-node/pkg/store"
)

const (
	// MaxPayloadStatusRetries is the maximum number of retries for SYNCING status.
	// According to the Engine API specification, SYNCING indicates temporary unavailability
	// and should be retried with exponential backoff.
	MaxPayloadStatusRetries = 3
	// InitialRetryBackoff is the initial backoff duration for retries.
	// The backoff doubles on each retry attempt (exponential backoff).
	InitialRetryBackoff = 1 * time.Second
	// SafeBlockLag is the number of blocks the safe block lags behind the head.
	// This provides a buffer for reorg protection - safe blocks won't be reorged
	// under normal operation. A value of 2 means when head is at block N,
	// safe is at block N-2.
	SafeBlockLag = 2
)

var (
	// ErrInvalidPayloadStatus indicates that the execution engine returned a permanent
	// failure status (INVALID or unknown status). This error should not be retried.
	ErrInvalidPayloadStatus = errors.New("invalid payload status")
	// ErrPayloadSyncing indicates that the execution engine is temporarily syncing.
	// According to the Engine API specification, this is a transient condition that
	// should be handled with retry logic rather than immediate failure.
	ErrPayloadSyncing = errors.New("payload syncing")
)

// Ensure EngineAPIExecutionClient implements the execution.Execute interface
var _ execution.Executor = (*EngineClient)(nil)

// validatePayloadStatus checks the payload status and returns appropriate errors.
// It implements the Engine API specification's status handling:
//   - VALID: Operation succeeded, return nil
//   - SYNCING/ACCEPTED: Temporary unavailability, return ErrPayloadSyncing for retry
//   - INVALID: Permanent failure, return ErrInvalidPayloadStatus (no retry)
//   - Unknown: Treat as permanent failure (no retry)
func validatePayloadStatus(status engine.PayloadStatusV1) error {
	switch status.Status {
	case engine.VALID:
		return nil
	case engine.SYNCING, engine.ACCEPTED:
		// SYNCING and ACCEPTED indicate temporary unavailability - should retry
		return ErrPayloadSyncing
	case engine.INVALID:
		// INVALID is a permanent failure - should not retry
		return ErrInvalidPayloadStatus
	default:
		// Unknown status - treat as invalid
		return ErrInvalidPayloadStatus
	}
}

// retryWithBackoffOnPayloadStatus executes a function with exponential backoff retry logic.
// It implements the Engine API specification's recommendation to retry SYNCING
// status with exponential backoff. The function:
//   - Retries only on ErrPayloadSyncing (transient failures)
//   - Fails immediately on ErrInvalidPayloadStatus (permanent failures)
//   - Respects context cancellation for graceful shutdown
//   - Uses exponential backoff that doubles on each attempt
func retryWithBackoffOnPayloadStatus(ctx context.Context, fn func() error, maxRetries int, initialBackoff time.Duration, operation string) error {
	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		// Don't retry on invalid status
		if errors.Is(err, ErrInvalidPayloadStatus) {
			return err
		}

		// Only retry on syncing status
		if !errors.Is(err, ErrPayloadSyncing) {
			return err
		}

		// Check if we've exhausted retries
		if attempt >= maxRetries {
			return fmt.Errorf("max retries (%d) exceeded for %s: %w", maxRetries, operation, err)
		}

		// Wait with exponential backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry for %s: %w", operation, ctx.Err())
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return fmt.Errorf("max retries (%d) exceeded for %s", maxRetries, operation)
}

// EngineClient represents a client that interacts with an Ethereum execution engine
// through the Engine API. It manages connections to both the engine and standard Ethereum
// APIs, and maintains state related to block processing.
type EngineClient struct {
	engineClient  *rpc.Client       // Client for Engine API calls
	ethClient     *ethclient.Client // Client for standard Ethereum API calls
	genesisHash   common.Hash       // Hash of the genesis block
	initialHeight uint64
	feeRecipient  common.Address // Address to receive transaction fees

	// store provides persistence for ExecMeta to enable idempotent execution
	// and crash recovery. Optional - if nil, ExecMeta tracking is disabled.
	store store.Store

	mu                        sync.Mutex             // Mutex to protect concurrent access to block hashes
	currentHeadBlockHash      common.Hash            // Store last non-finalized HeadBlockHash
	currentHeadHeight         uint64                 // Height of the current head block (for safe lag calculation)
	currentSafeBlockHash      common.Hash            // Store last non-finalized SafeBlockHash
	currentFinalizedBlockHash common.Hash            // Store last finalized block hash
	blockHashCache            map[uint64]common.Hash // height -> hash cache for safe block lookups

	// executeMu serializes all ExecuteTxs calls to prevent concurrent block builds
	// that could create sibling blocks (fork explosion). This is separate from mu
	// to avoid holding locks during long RPC calls.
	executeMu sync.Mutex

	logger zerolog.Logger
}

// NewEngineExecutionClient creates a new instance of EngineAPIExecutionClient.
// The store parameter is optional - if provided, ExecMeta tracking is enabled
// for idempotent execution and crash recovery.
func NewEngineExecutionClient(
	ethURL,
	engineURL string,
	jwtSecret string,
	genesisHash common.Hash,
	feeRecipient common.Address,
	evStore store.Store,
) (*EngineClient, error) {
	ethClient, err := ethclient.Dial(ethURL)
	if err != nil {
		return nil, err
	}

	secret, err := decodeSecret(jwtSecret)
	if err != nil {
		return nil, err
	}

	engineClient, err := rpc.DialOptions(context.Background(), engineURL,
		rpc.WithHTTPAuth(func(h http.Header) error {
			authToken, err := getAuthToken(secret)
			if err != nil {
				return err
			}

			if authToken != "" {
				h.Set("Authorization", "Bearer "+authToken)
			}
			return nil
		}))
	if err != nil {
		return nil, err
	}

	return &EngineClient{
		engineClient:              engineClient,
		ethClient:                 ethClient,
		genesisHash:               genesisHash,
		feeRecipient:              feeRecipient,
		store:                     evStore,
		currentHeadBlockHash:      genesisHash,
		currentSafeBlockHash:      genesisHash,
		currentFinalizedBlockHash: genesisHash,
		blockHashCache:            make(map[uint64]common.Hash),
		logger:                    zerolog.Nop(),
	}, nil
}

// SetLogger allows callers to attach a structured logger.
func (c *EngineClient) SetLogger(l zerolog.Logger) {
	c.logger = l
}

// SetStore allows callers to attach a store for ExecMeta tracking.
// This enables idempotent execution and crash recovery features.
// Must be called before ExecuteTxs if ExecMeta tracking is desired.
func (c *EngineClient) SetStore(s store.Store) {
	c.store = s
}

// InitChain initializes the blockchain with the given genesis parameters
func (c *EngineClient) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, uint64, error) {
	if initialHeight != 1 {
		return nil, 0, fmt.Errorf("initialHeight must be 1, got %d", initialHeight)
	}

	// Acknowledge the genesis block with retry logic for SYNCING status
	err := retryWithBackoffOnPayloadStatus(ctx, func() error {
		var forkchoiceResult engine.ForkChoiceResponse
		err := c.engineClient.CallContext(ctx, &forkchoiceResult, "engine_forkchoiceUpdatedV3",
			engine.ForkchoiceStateV1{
				HeadBlockHash:      c.genesisHash,
				SafeBlockHash:      c.genesisHash,
				FinalizedBlockHash: c.genesisHash,
			},
			nil,
		)
		if err != nil {
			return fmt.Errorf("engine_forkchoiceUpdatedV3 failed: %w", err)
		}

		// Validate payload status
		if err := validatePayloadStatus(forkchoiceResult.PayloadStatus); err != nil {
			c.logger.Warn().
				Str("status", forkchoiceResult.PayloadStatus.Status).
				Str("latestValidHash", forkchoiceResult.PayloadStatus.LatestValidHash.Hex()).
				Interface("validationError", forkchoiceResult.PayloadStatus.ValidationError).
				Msg("InitChain: engine_forkchoiceUpdatedV3 returned non-VALID status")
			return err
		}

		return nil
	}, MaxPayloadStatusRetries, InitialRetryBackoff, "InitChain")
	if err != nil {
		return nil, 0, err
	}

	_, stateRoot, gasLimit, _, err := c.getBlockInfo(ctx, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get block info: %w", err)
	}

	c.initialHeight = initialHeight

	return stateRoot[:], gasLimit, nil
}

// GetTxs retrieves transactions from the current execution payload
func (c *EngineClient) GetTxs(ctx context.Context) ([][]byte, error) {
	var result []string
	err := c.ethClient.Client().CallContext(ctx, &result, "txpoolExt_getTxs")
	if err != nil {
		return nil, fmt.Errorf("failed to get tx pool content: %w", err)
	}

	txs := make([][]byte, 0, len(result))
	for _, rlpHex := range result {
		if !strings.HasPrefix(rlpHex, "0x") || len(rlpHex) < 3 {
			return nil, fmt.Errorf("invalid hex format for transaction: %s", rlpHex)
		}
		txBytes := common.FromHex(rlpHex)
		if len(txBytes) == 0 && len(rlpHex) > 2 {
			return nil, fmt.Errorf("failed to decode hex transaction: %s", rlpHex)
		}
		txs = append(txs, txBytes)
	}

	return txs, nil
}

// ExecuteTxs executes the given transactions at the specified block height and timestamp.
// This method is serialized via executeMu to prevent concurrent block builds that could
// create sibling blocks (fork explosion).
//
// ExecMeta tracking (if store is configured):
// - Checks for already-promoted blocks to enable idempotent execution
// - Saves ExecMeta with payloadID after forkchoiceUpdatedV3 for crash recovery
// - Updates ExecMeta to "promoted" after successful execution
func (c *EngineClient) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) (updatedStateRoot []byte, maxBytes uint64, err error) {
	// Serialize all ExecuteTxs calls to prevent concurrent sibling block creation
	c.executeMu.Lock()
	defer c.executeMu.Unlock()

	// Check ExecMeta for idempotent execution (if store is configured)
	if c.store != nil {
		execMeta, err := c.store.GetExecMeta(ctx, blockHeight)
		if err == nil && execMeta != nil {
			// If we already have a promoted block at this height, return the stored StateRoot
			if execMeta.Stage == store.ExecStagePromoted && len(execMeta.StateRoot) > 0 {
				c.logger.Info().
					Uint64("height", blockHeight).
					Str("stage", execMeta.Stage).
					Msg("ExecuteTxs: reusing already-promoted execution (idempotent)")
				return execMeta.StateRoot, 0, nil
			}

			// If we have a started execution with a payloadID, try to resume
			if execMeta.Stage == store.ExecStageStarted && len(execMeta.PayloadID) > 0 {
				c.logger.Info().
					Uint64("height", blockHeight).
					Str("stage", execMeta.Stage).
					Msg("ExecuteTxs: found in-progress execution with payloadID, attempting resume")

				stateRoot, err := c.ResumePayload(ctx, execMeta.PayloadID)
				if err == nil {
					c.logger.Info().
						Uint64("height", blockHeight).
						Msg("ExecuteTxs: successfully resumed payload")
					return stateRoot, 0, nil
				}
				// Resume failed - log and fall through to EL-level idempotency check
				c.logger.Warn().Err(err).
					Uint64("height", blockHeight).
					Msg("ExecuteTxs: failed to resume payload, falling back to EL-level check")
			}
		}
	}

	// Idempotency check: if EL already has a block at this height with matching
	// timestamp, return its state root instead of building a new block.
	// This handles retries and crash recovery without creating sibling blocks.
	existingBlockHash, existingStateRoot, _, existingTimestamp, err := c.getBlockInfo(ctx, blockHeight)
	if err == nil && existingBlockHash != (common.Hash{}) {
		// Block exists at this height - check if timestamp matches
		if existingTimestamp == uint64(timestamp.Unix()) {
			c.logger.Info().
				Uint64("height", blockHeight).
				Str("blockHash", existingBlockHash.Hex()).
				Str("stateRoot", existingStateRoot.Hex()).
				Msg("ExecuteTxs: reusing existing block at height (EL idempotency)")

			// Update head to point to this existing block
			if err := c.setHead(ctx, existingBlockHash); err != nil {
				c.logger.Warn().Err(err).Msg("ExecuteTxs: failed to update head to existing block")
				// Continue anyway - the block exists and we can return its state root
			}

			// Update ExecMeta to promoted if store is configured
			if c.store != nil {
				c.saveExecMeta(ctx, blockHeight, timestamp.Unix(), nil, existingBlockHash[:], existingStateRoot.Bytes(), txs, store.ExecStagePromoted)
			}

			return existingStateRoot.Bytes(), 0, nil
		}
		// Timestamp mismatch - this is unexpected, log a warning but proceed
		// This could happen if there's a genuine reorg needed
		c.logger.Warn().
			Uint64("height", blockHeight).
			Uint64("existingTimestamp", existingTimestamp).
			Int64("requestedTimestamp", timestamp.Unix()).
			Msg("ExecuteTxs: block exists at height but timestamp differs")
	}

	forceIncludedMask := execution.GetForceIncludedMask(ctx)

	// Filter out invalid transactions to handle gibberish gracefully
	validTxs := make([]string, 0, len(txs))
	for i, tx := range txs {
		if len(tx) == 0 {
			continue
		}

		// Skip validation for mempool transactions (already validated when added to mempool)
		// Force-included transactions from DA MUST be validated as they come from untrusted sources
		if forceIncludedMask != nil && i < len(forceIncludedMask) && !forceIncludedMask[i] {
			validTxs = append(validTxs, "0x"+hex.EncodeToString(tx))
			continue
		}

		// Validate that the transaction can be parsed as an Ethereum transaction
		var ethTx types.Transaction
		if err := ethTx.UnmarshalBinary(tx); err != nil {
			c.logger.Debug().
				Int("tx_index", i).
				Uint64("block_height", blockHeight).
				Err(err).
				Str("tx_hex", "0x"+hex.EncodeToString(tx)).
				Msg("filtering out invalid transaction (gibberish)")
			continue
		}

		validTxs = append(validTxs, "0x"+hex.EncodeToString(tx))
	}

	if len(validTxs) < len(txs) {
		c.logger.Debug().
			Int("total_txs", len(txs)).
			Int("valid_txs", len(validTxs)).
			Int("filtered_txs", len(txs)-len(validTxs)).
			Uint64("block_height", blockHeight).
			Msg("filtered out invalid transactions")
	}
	txsPayload := validTxs

	prevBlockHash, _, prevGasLimit, _, err := c.getBlockInfo(ctx, blockHeight-1)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get block info: %w", err)
	}

	// Use tracked safe/finalized state rather than prevBlockHash to avoid
	// regressing these values. Head must be prevBlockHash to build on top of it.
	c.mu.Lock()
	args := engine.ForkchoiceStateV1{
		HeadBlockHash:      prevBlockHash,
		SafeBlockHash:      c.currentSafeBlockHash,
		FinalizedBlockHash: c.currentFinalizedBlockHash,
	}
	c.mu.Unlock()

	// update forkchoice to get the next payload id
	// Create evolve-compatible payloadtimestamp.Unix()
	evPayloadAttrs := map[string]any{
		// Standard Ethereum payload attributes (flattened) - using camelCase as expected by JSON
		"timestamp":             timestamp.Unix(),
		"prevRandao":            c.derivePrevRandao(blockHeight),
		"suggestedFeeRecipient": c.feeRecipient,
		"withdrawals":           []*types.Withdrawal{},
		// V3 requires parentBeaconBlockRoot
		"parentBeaconBlockRoot": common.Hash{}.Hex(), // Use zero hash for evolve
		// evolve-specific fields
		"transactions": txsPayload,
		"gasLimit":     prevGasLimit, // Use camelCase to match JSON conventions
	}

	c.logger.Debug().
		Uint64("height", blockHeight).
		Int("tx_count", len(txs)).
		Msg("engine_forkchoiceUpdatedV3")

	// Call forkchoice update with retry logic for SYNCING status
	var payloadID *engine.PayloadID
	err = retryWithBackoffOnPayloadStatus(ctx, func() error {
		var forkchoiceResult engine.ForkChoiceResponse
		err := c.engineClient.CallContext(ctx, &forkchoiceResult, "engine_forkchoiceUpdatedV3", args, evPayloadAttrs)
		if err != nil {
			return fmt.Errorf("forkchoice update failed: %w", err)
		}

		// Validate payload status
		if err := validatePayloadStatus(forkchoiceResult.PayloadStatus); err != nil {
			c.logger.Warn().
				Str("status", forkchoiceResult.PayloadStatus.Status).
				Str("latestValidHash", forkchoiceResult.PayloadStatus.LatestValidHash.Hex()).
				Interface("validationError", forkchoiceResult.PayloadStatus.ValidationError).
				Uint64("blockHeight", blockHeight).
				Msg("ExecuteTxs: engine_forkchoiceUpdatedV3 returned non-VALID status")
			return err
		}

		if forkchoiceResult.PayloadID == nil {
			c.logger.Error().
				Str("status", forkchoiceResult.PayloadStatus.Status).
				Str("latestValidHash", forkchoiceResult.PayloadStatus.LatestValidHash.Hex()).
				Interface("validationError", forkchoiceResult.PayloadStatus.ValidationError).
				Interface("forkchoiceState", args).
				Interface("payloadAttributes", evPayloadAttrs).
				Uint64("blockHeight", blockHeight).
				Msg("returned nil PayloadID")

			return fmt.Errorf("returned nil PayloadID - (status: %s, latestValidHash: %s)",
				forkchoiceResult.PayloadStatus.Status,
				forkchoiceResult.PayloadStatus.LatestValidHash.Hex())
		}

		payloadID = forkchoiceResult.PayloadID
		return nil
	}, MaxPayloadStatusRetries, InitialRetryBackoff, "ExecuteTxs forkchoice")
	if err != nil {
		return nil, 0, err
	}

	// Save ExecMeta with payloadID for crash recovery (Stage="started")
	// This allows resuming the payload build if we crash before completing
	if c.store != nil {
		c.saveExecMeta(ctx, blockHeight, timestamp.Unix(), payloadID[:], nil, nil, txs, store.ExecStageStarted)
	}

	// get payload
	var payloadResult engine.ExecutionPayloadEnvelope
	err = c.engineClient.CallContext(ctx, &payloadResult, "engine_getPayloadV4", *payloadID)
	if err != nil {
		return nil, 0, fmt.Errorf("get payload failed: %w", err)
	}

	// Submit payload with retry logic for SYNCING status
	var newPayloadResult engine.PayloadStatusV1
	err = retryWithBackoffOnPayloadStatus(ctx, func() error {
		err := c.engineClient.CallContext(ctx, &newPayloadResult, "engine_newPayloadV4",
			payloadResult.ExecutionPayload,
			[]string{},          // No blob hashes
			common.Hash{}.Hex(), // Use zero hash for parentBeaconBlockRoot (same as in payload attributes)
			[][]byte{},          // No execution requests
		)
		if err != nil {
			return fmt.Errorf("new payload submission failed: %w", err)
		}

		// Validate payload status
		if err := validatePayloadStatus(newPayloadResult); err != nil {
			c.logger.Warn().
				Str("status", newPayloadResult.Status).
				Str("latestValidHash", newPayloadResult.LatestValidHash.Hex()).
				Interface("validationError", newPayloadResult.ValidationError).
				Uint64("blockHeight", blockHeight).
				Msg("engine_newPayloadV4 returned non-VALID status")
			return err
		}
		return nil
	}, MaxPayloadStatusRetries, InitialRetryBackoff, "ExecuteTxs newPayload")
	if err != nil {
		return nil, 0, err
	}

	// forkchoice update - advance head and lag safe by SafeBlockLag blocks
	blockHash := payloadResult.ExecutionPayload.BlockHash

	// Cache block hash for safe block lookups (avoids RPC calls in SetSafeByHeight)
	c.cacheBlockHash(blockHeight, blockHash)

	err = c.setFinalWithHeight(ctx, blockHash, blockHeight, false)
	if err != nil {
		return nil, 0, err
	}

	// Update ExecMeta to "promoted" after successful execution
	if c.store != nil {
		c.saveExecMeta(ctx, blockHeight, timestamp.Unix(), payloadID[:], blockHash[:], payloadResult.ExecutionPayload.StateRoot.Bytes(), txs, store.ExecStagePromoted)
	}

	return payloadResult.ExecutionPayload.StateRoot.Bytes(), payloadResult.ExecutionPayload.GasUsed, nil
}

// setHead updates the head block hash without changing safe or finalized.
// This is used when reusing an existing block (idempotency check).
func (c *EngineClient) setHead(ctx context.Context, blockHash common.Hash) error {
	c.mu.Lock()
	c.currentHeadBlockHash = blockHash
	// Note: safe and finalized are NOT updated - they advance separately via derivation
	args := engine.ForkchoiceStateV1{
		HeadBlockHash:      c.currentHeadBlockHash,
		SafeBlockHash:      c.currentSafeBlockHash,
		FinalizedBlockHash: c.currentFinalizedBlockHash,
	}
	c.mu.Unlock()

	return c.doForkchoiceUpdate(ctx, args, "setHead")
}

func (c *EngineClient) setFinal(ctx context.Context, blockHash common.Hash, isFinal bool) error {
	return c.setFinalWithHeight(ctx, blockHash, 0, isFinal)
}

// setFinalWithHeight updates forkchoice state with optional safe block lagging.
// When isFinal=false and headHeight > SafeBlockLag, the safe block is automatically
// set to headHeight - SafeBlockLag blocks behind head.
func (c *EngineClient) setFinalWithHeight(ctx context.Context, blockHash common.Hash, headHeight uint64, isFinal bool) error {
	c.mu.Lock()
	// Update block hashes based on finalization status
	if isFinal {
		// When finalizing, also advance safe to at least the finalized block
		c.currentFinalizedBlockHash = blockHash
		c.currentSafeBlockHash = blockHash
	} else {
		// Update head
		c.currentHeadBlockHash = blockHash
		c.currentHeadHeight = headHeight
	}
	c.mu.Unlock()

	// If advancing head (not finalizing) and we have height info, update safe with lag
	if !isFinal && headHeight > SafeBlockLag {
		safeHeight := headHeight - SafeBlockLag
		if err := c.SetSafeByHeight(ctx, safeHeight); err != nil {
			c.logger.Warn().
				Err(err).
				Uint64("headHeight", headHeight).
				Uint64("safeHeight", safeHeight).
				Msg("setFinalWithHeight: failed to update safe block (continuing)")
			// Don't fail - safe update is best-effort
		}
	}

	c.mu.Lock()
	// Construct forkchoice state
	args := engine.ForkchoiceStateV1{
		HeadBlockHash:      c.currentHeadBlockHash,
		SafeBlockHash:      c.currentSafeBlockHash,
		FinalizedBlockHash: c.currentFinalizedBlockHash,
	}
	c.mu.Unlock()

	return c.doForkchoiceUpdate(ctx, args, "setFinal")
}

// doForkchoiceUpdate performs the actual forkchoice update RPC call with retry logic.
func (c *EngineClient) doForkchoiceUpdate(ctx context.Context, args engine.ForkchoiceStateV1, operation string) error {
	// Call forkchoice update with retry logic for SYNCING status
	err := retryWithBackoffOnPayloadStatus(ctx, func() error {
		var forkchoiceResult engine.ForkChoiceResponse
		err := c.engineClient.CallContext(ctx, &forkchoiceResult, "engine_forkchoiceUpdatedV3", args, nil)
		if err != nil {
			return fmt.Errorf("forkchoice update failed: %w", err)
		}

		// Validate payload status
		if err := validatePayloadStatus(forkchoiceResult.PayloadStatus); err != nil {
			c.logger.Warn().
				Str("status", forkchoiceResult.PayloadStatus.Status).
				Str("latestValidHash", forkchoiceResult.PayloadStatus.LatestValidHash.Hex()).
				Interface("validationError", forkchoiceResult.PayloadStatus.ValidationError).
				Str("operation", operation).
				Msg("forkchoiceUpdatedV3 returned non-VALID status")
			return err
		}
		return nil
	}, MaxPayloadStatusRetries, InitialRetryBackoff, operation)
	if err != nil {
		return err
	}

	return nil
}

// SetFinal marks the block at the given height as finalized
func (c *EngineClient) SetFinal(ctx context.Context, blockHeight uint64) error {
	blockHash, _, _, _, err := c.getBlockInfo(ctx, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to get block info: %w", err)
	}
	return c.setFinal(ctx, blockHash, true)
}

// SetSafe explicitly sets the safe block hash.
// This allows the derivation layer to advance the safe block independently of head.
// Safe indicates a block that is unlikely to be reorged (e.g., confirmed by DA).
func (c *EngineClient) SetSafe(ctx context.Context, blockHash common.Hash) error {
	c.mu.Lock()
	c.currentSafeBlockHash = blockHash
	args := engine.ForkchoiceStateV1{
		HeadBlockHash:      c.currentHeadBlockHash,
		SafeBlockHash:      c.currentSafeBlockHash,
		FinalizedBlockHash: c.currentFinalizedBlockHash,
	}
	c.mu.Unlock()

	return c.doForkchoiceUpdate(ctx, args, "SetSafe")
}

// SetSafeByHeight sets the safe block by looking up the block hash at the given height.
// Uses cached block hashes when available to avoid RPC calls. Falls back to RPC on cache miss
// (e.g., during restart before cache is warmed).
// Returns nil if the height doesn't exist yet (block not produced).
func (c *EngineClient) SetSafeByHeight(ctx context.Context, height uint64) error {
	// Try cache first (avoids RPC call in normal operation)
	c.mu.Lock()
	blockHash, ok := c.blockHashCache[height]
	c.mu.Unlock()

	if !ok {
		// Cache miss - fallback to RPC (happens during restart/recovery)
		var err error
		blockHash, _, _, _, err = c.getBlockInfo(ctx, height)
		if err != nil {
			// Block doesn't exist yet - this is expected during early blocks
			c.logger.Debug().
				Uint64("height", height).
				Err(err).
				Msg("SetSafeByHeight: block not found, skipping safe update")
			return nil
		}
	}
	return c.SetSafe(ctx, blockHash)
}

// cacheBlockHash stores a block hash in the cache for later safe block lookups.
// The cache is bounded to prevent unbounded memory growth - old entries are pruned.
func (c *EngineClient) cacheBlockHash(height uint64, hash common.Hash) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.blockHashCache[height] = hash

	// Prune old entries to keep cache bounded (keep last ~10 blocks)
	// This is sufficient since SafeBlockLag is only 2
	const maxCacheSize = 10
	if len(c.blockHashCache) > maxCacheSize {
		minHeight := height - maxCacheSize
		for h := range c.blockHashCache {
			if h < minHeight {
				delete(c.blockHashCache, h)
			}
		}
	}
}

// SetFinalized explicitly sets the finalized block hash.
// This allows the derivation layer to advance finalization independently.
// Finalized indicates a block that will never be reorged (e.g., included in DA with sufficient confirmations).
func (c *EngineClient) SetFinalized(ctx context.Context, blockHash common.Hash) error {
	c.mu.Lock()
	c.currentFinalizedBlockHash = blockHash
	// When finalizing, also ensure safe is at least at finalized
	if c.currentSafeBlockHash == (common.Hash{}) || c.currentSafeBlockHash == c.genesisHash {
		c.currentSafeBlockHash = blockHash
	}
	args := engine.ForkchoiceStateV1{
		HeadBlockHash:      c.currentHeadBlockHash,
		SafeBlockHash:      c.currentSafeBlockHash,
		FinalizedBlockHash: c.currentFinalizedBlockHash,
	}
	c.mu.Unlock()

	return c.doForkchoiceUpdate(ctx, args, "SetFinalized")
}

// ResumePayload resumes an in-progress payload build using a stored payloadID.
// This is used for crash recovery when we have a payloadID but haven't yet
// retrieved and submitted the payload to the EL.
//
// The method:
// 1. Calls engine_getPayloadV4 with the stored payloadID
// 2. Submits the payload via engine_newPayloadV4
// 3. Updates forkchoice to set the new block as head
//
// Returns the state root from the payload, or an error if resumption fails.
// Implements the execution.PayloadResumer interface.
func (c *EngineClient) ResumePayload(ctx context.Context, payloadIDBytes []byte) (stateRoot []byte, err error) {
	// Serialize to prevent concurrent block builds
	c.executeMu.Lock()
	defer c.executeMu.Unlock()

	// Convert bytes to PayloadID
	if len(payloadIDBytes) != 8 {
		return nil, fmt.Errorf("ResumePayload: invalid payloadID length %d, expected 8", len(payloadIDBytes))
	}
	var payloadID engine.PayloadID
	copy(payloadID[:], payloadIDBytes)

	c.logger.Info().
		Str("payloadID", payloadID.String()).
		Msg("ResumePayload: attempting to resume in-progress payload")

	// Step 1: Get the payload using the stored payloadID
	var payloadResult engine.ExecutionPayloadEnvelope
	err = c.engineClient.CallContext(ctx, &payloadResult, "engine_getPayloadV4", payloadID)
	if err != nil {
		return nil, fmt.Errorf("ResumePayload: get payload failed: %w", err)
	}

	// Step 2: Submit the payload with retry logic for SYNCING status
	var newPayloadResult engine.PayloadStatusV1
	err = retryWithBackoffOnPayloadStatus(ctx, func() error {
		err := c.engineClient.CallContext(ctx, &newPayloadResult, "engine_newPayloadV4",
			payloadResult.ExecutionPayload,
			[]string{},          // No blob hashes
			common.Hash{}.Hex(), // Use zero hash for parentBeaconBlockRoot
			[][]byte{},          // No execution requests
		)
		if err != nil {
			return fmt.Errorf("new payload submission failed: %w", err)
		}

		// Validate payload status
		if err := validatePayloadStatus(newPayloadResult); err != nil {
			c.logger.Warn().
				Str("status", newPayloadResult.Status).
				Str("latestValidHash", newPayloadResult.LatestValidHash.Hex()).
				Interface("validationError", newPayloadResult.ValidationError).
				Msg("ResumePayload: engine_newPayloadV4 returned non-VALID status")
			return err
		}
		return nil
	}, MaxPayloadStatusRetries, InitialRetryBackoff, "ResumePayload newPayload")
	if err != nil {
		return nil, err
	}

	// Step 3: Update forkchoice to set the new block as head (with safe lag)
	blockHash := payloadResult.ExecutionPayload.BlockHash
	blockHeight := payloadResult.ExecutionPayload.Number
	err = c.setFinalWithHeight(ctx, blockHash, blockHeight, false)
	if err != nil {
		return nil, fmt.Errorf("ResumePayload: forkchoice update failed: %w", err)
	}

	c.logger.Info().
		Str("blockHash", blockHash.Hex()).
		Str("stateRoot", payloadResult.ExecutionPayload.StateRoot.Hex()).
		Uint64("blockHeight", blockHeight).
		Msg("ResumePayload: successfully resumed payload")

	// Update ExecMeta to "promoted" after successful resume
	if c.store != nil {
		c.saveExecMeta(ctx, blockHeight, int64(payloadResult.ExecutionPayload.Timestamp), payloadIDBytes, blockHash[:], payloadResult.ExecutionPayload.StateRoot.Bytes(), nil, store.ExecStagePromoted)
	}

	return payloadResult.ExecutionPayload.StateRoot.Bytes(), nil
}

func (c *EngineClient) derivePrevRandao(blockHeight uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(blockHeight))
}

func (c *EngineClient) getBlockInfo(ctx context.Context, height uint64) (common.Hash, common.Hash, uint64, uint64, error) {
	header, err := c.ethClient.HeaderByNumber(ctx, new(big.Int).SetUint64(height))

	if err != nil {
		return common.Hash{}, common.Hash{}, 0, 0, fmt.Errorf("failed to get block at height %d: %w", height, err)
	}

	return header.Hash(), header.Root, header.GasLimit, header.Time, nil
}

// saveExecMeta persists execution metadata to the store for crash recovery.
// This is a best-effort operation - failures are logged but don't fail the execution.
func (c *EngineClient) saveExecMeta(ctx context.Context, height uint64, timestamp int64, payloadID []byte, blockHash []byte, stateRoot []byte, txs [][]byte, stage string) {
	if c.store == nil {
		return
	}

	execMeta := &store.ExecMeta{
		Height:        height,
		Timestamp:     timestamp,
		PayloadID:     payloadID,
		BlockHash:     blockHash,
		StateRoot:     stateRoot,
		Stage:         stage,
		UpdatedAtUnix: time.Now().Unix(),
	}

	// Compute tx hash for sanity checks on retry
	if len(txs) > 0 {
		h := sha256.New()
		for _, tx := range txs {
			h.Write(tx)
		}
		execMeta.TxHash = h.Sum(nil)
	}

	batch, err := c.store.NewBatch(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Uint64("height", height).Msg("saveExecMeta: failed to create batch")
		return
	}

	if err := batch.SaveExecMeta(execMeta); err != nil {
		c.logger.Warn().Err(err).Uint64("height", height).Msg("saveExecMeta: failed to save exec meta")
		return
	}

	if err := batch.Commit(); err != nil {
		c.logger.Warn().Err(err).Uint64("height", height).Msg("saveExecMeta: failed to commit batch")
		return
	}

	c.logger.Debug().
		Uint64("height", height).
		Str("stage", stage).
		Msg("saveExecMeta: saved execution metadata")
}

// GetLatestHeight returns the current block height of the execution layer
func (c *EngineClient) GetLatestHeight(ctx context.Context) (uint64, error) {
	header, err := c.ethClient.HeaderByNumber(ctx, nil) // nil = latest block
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	return header.Number.Uint64(), nil
}

// decodeSecret decodes a hex-encoded JWT secret string into a byte slice.
func decodeSecret(jwtSecret string) ([]byte, error) {
	secret, err := hex.DecodeString(strings.TrimPrefix(jwtSecret, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT secret: %w", err)
	}
	return secret, nil
}

// getAuthToken creates a JWT token signed with the provided secret, valid for 1 hour.
func getAuthToken(jwtSecret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 1).Unix(), // Expires in 1 hour
		"iat": time.Now().Unix(),
	})

	// Sign the token with the decoded secret
	authToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}
	return authToken, nil
}
