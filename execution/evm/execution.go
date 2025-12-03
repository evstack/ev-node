package evm

import (
	"context"
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
)

const (
	// MaxPayloadStatusRetries is the maximum number of retries for SYNCING status.
	// According to the Engine API specification, SYNCING indicates temporary unavailability
	// and should be retried with exponential backoff.
	MaxPayloadStatusRetries = 3
	// InitialRetryBackoff is the initial backoff duration for retries.
	// The backoff doubles on each retry attempt (exponential backoff).
	InitialRetryBackoff = 1 * time.Second
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

	mu                        sync.Mutex  // Mutex to protect concurrent access to block hashes
	currentHeadBlockHash      common.Hash // Store last non-finalized HeadBlockHash
	currentSafeBlockHash      common.Hash // Store last non-finalized SafeBlockHash
	currentFinalizedBlockHash common.Hash // Store last finalized block hash

	logger zerolog.Logger
}

// NewEngineExecutionClient creates a new instance of EngineAPIExecutionClient
func NewEngineExecutionClient(
	ethURL,
	engineURL string,
	jwtSecret string,
	genesisHash common.Hash,
	feeRecipient common.Address,
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
		currentHeadBlockHash:      genesisHash,
		currentSafeBlockHash:      genesisHash,
		currentFinalizedBlockHash: genesisHash,
		logger:                    zerolog.Nop(),
	}, nil
}

// SetLogger allows callers to attach a structured logger.
func (c *EngineClient) SetLogger(l zerolog.Logger) {
	c.logger = l
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

// ExecuteTxs executes the given transactions at the specified block height and timestamp
func (c *EngineClient) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) (updatedStateRoot []byte, maxBytes uint64, err error) {
	forceIncludedMask := execution.GetForceIncludedMask(ctx)

	// Filter out invalid transactions to handle gibberish gracefully
	validTxs := make([]string, 0, len(txs))
	for i, tx := range txs {
		if len(tx) == 0 {
			c.logger.Debug().
				Int("tx_index", i).
				Uint64("block_height", blockHeight).
				Msg("skipping empty transaction")
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
			c.logger.Warn().
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
		c.logger.Info().
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

	args := engine.ForkchoiceStateV1{
		HeadBlockHash:      prevBlockHash,
		SafeBlockHash:      prevBlockHash,
		FinalizedBlockHash: prevBlockHash,
	}

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

	// forkchoice update
	blockHash := payloadResult.ExecutionPayload.BlockHash
	err = c.setFinal(ctx, blockHash, false)
	if err != nil {
		return nil, 0, err
	}

	return payloadResult.ExecutionPayload.StateRoot.Bytes(), payloadResult.ExecutionPayload.GasUsed, nil
}

func (c *EngineClient) setFinal(ctx context.Context, blockHash common.Hash, isFinal bool) error {
	c.mu.Lock()
	// Update block hashes based on finalization status
	if isFinal {
		c.currentFinalizedBlockHash = blockHash
	} else {
		c.currentHeadBlockHash = blockHash
		c.currentSafeBlockHash = blockHash
	}

	// Construct forkchoice state
	args := engine.ForkchoiceStateV1{
		HeadBlockHash:      c.currentHeadBlockHash,
		SafeBlockHash:      c.currentSafeBlockHash,
		FinalizedBlockHash: c.currentFinalizedBlockHash,
	}
	c.mu.Unlock()

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
				Msg("forkchoiceUpdatedV3 returned non-VALID status")
			return err
		}
		return nil
	}, MaxPayloadStatusRetries, InitialRetryBackoff, "setFinal")
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
