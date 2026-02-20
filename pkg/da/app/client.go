// Package app provides a DA client that communicates directly with celestia-app
// via CometBFT RPC, without requiring celestia-node.
package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/celestiaorg/go-square/merkle"
	"github.com/celestiaorg/go-square/v3/inclusion"
	"github.com/celestiaorg/go-square/v3/share"
	"github.com/celestiaorg/go-square/v3/tx"
	"github.com/rs/zerolog"

	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// subtreeRootThreshold is the default threshold for subtree roots.
// This matches the value used in celestia-app.
const subtreeRootThreshold = 64

// defaultMaxBlobSize is the default maximum blob size (5MB).
const defaultMaxBlobSize = 5 * 1024 * 1024

// Config contains configuration for the celestia-app DA client.
type Config struct {
	// RPCAddress is the CometBFT RPC endpoint (e.g., "http://localhost:26657")
	RPCAddress string
	// Logger for logging
	Logger zerolog.Logger
	// DefaultTimeout for RPC calls
	DefaultTimeout time.Duration
}

// Client implements the datypes.BlobClient interface using celestia-app's CometBFT RPC.
// This client communicates directly with celestia-app without requiring celestia-node.
type Client struct {
	rpcAddress     string
	logger         zerolog.Logger
	defaultTimeout time.Duration
	httpClient     *http.Client
}

// Ensure Client implements the datypes.BlobClient interface.
var _ datypes.BlobClient = (*Client)(nil)

// NewClient creates a new DA client that communicates directly with celestia-app.
func NewClient(cfg Config) *Client {
	if cfg.RPCAddress == "" {
		return nil
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 60 * time.Second
	}

	return &Client{
		rpcAddress:     cfg.RPCAddress,
		logger:         cfg.Logger.With().Str("component", "celestia_app_client").Logger(),
		defaultTimeout: cfg.DefaultTimeout,
		httpClient:     &http.Client{Timeout: cfg.DefaultTimeout},
	}
}

// Submit submits blobs to the DA layer via celestia-app.
// Note: This requires transaction signing which is not implemented in this basic version.
// For full implementation, integration with a signer/keyring would be needed.
func (c *Client) Submit(ctx context.Context, data [][]byte, _ float64, namespace []byte, options []byte) datypes.ResultSubmit {
	// Calculate blob size
	var blobSize uint64
	for _, b := range data {
		blobSize += uint64(len(b))
	}

	// Validate namespace
	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return datypes.ResultSubmit{
			BaseResult: datypes.BaseResult{
				Code:    datypes.StatusError,
				Message: fmt.Sprintf("invalid namespace: %v", err),
			},
		}
	}

	// Check blob sizes
	for i, raw := range data {
		if uint64(len(raw)) > defaultMaxBlobSize {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusTooBig,
					Message: datypes.ErrBlobSizeOverLimit.Error(),
				},
			}
		}
		// Validate blob data
		if len(raw) == 0 {
			return datypes.ResultSubmit{
				BaseResult: datypes.BaseResult{
					Code:    datypes.StatusError,
					Message: fmt.Sprintf("blob %d is empty", i),
				},
			}
		}
	}

	// TODO: Implement actual blob submission
	// This requires:
	// 1. Creating a MsgPayForBlobs transaction
	// 2. Signing the transaction
	// 3. Broadcasting via /broadcast_tx_commit or /broadcast_tx_sync
	//
	// For now, return an error indicating this needs to be implemented
	// with proper transaction signing infrastructure.
	c.logger.Error().
		Int("blob_count", len(data)).
		Str("namespace", ns.String()).
		Msg("Submit not implemented - requires transaction signing infrastructure")

	return datypes.ResultSubmit{
		BaseResult: datypes.BaseResult{
			Code:           datypes.StatusError,
			Message:        "Submit not implemented: requires transaction signing. Use celestia-node client for submission or implement signer integration",
			SubmittedCount: 0,
			Height:         0,
			Timestamp:      time.Now(),
			BlobSize:       blobSize,
		},
	}
}

// Retrieve retrieves blobs from the DA layer at the specified height and namespace.
// It fetches the block via CometBFT RPC and extracts blob transactions.
func (c *Client) Retrieve(ctx context.Context, height uint64, namespace []byte) datypes.ResultRetrieve {
	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusError,
				Message:   fmt.Sprintf("invalid namespace: %v", err),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	// Fetch block from celestia-app
	blockResult, err := c.getBlock(ctx, height)
	if err != nil {
		// Handle specific errors
		if strings.Contains(err.Error(), "height") && strings.Contains(err.Error(), "future") {
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusHeightFromFuture,
					Message:   datypes.ErrHeightFromFuture.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		if strings.Contains(err.Error(), "is not available, lowest height is") {
			c.logger.Debug().Uint64("height", height).Err(err).Msg("block is pruned or unavailable")
			return datypes.ResultRetrieve{
				BaseResult: datypes.BaseResult{
					Code:      datypes.StatusNotFound,
					Message:   datypes.ErrBlobNotFound.Error(),
					Height:    height,
					Timestamp: time.Now(),
				},
			}
		}
		c.logger.Error().Err(err).Uint64("height", height).Msg("failed to get block")
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusError,
				Message:   fmt.Sprintf("failed to get block: %s", err.Error()),
				Height:    height,
				Timestamp: time.Now(),
			},
		}
	}

	// Extract timestamp from block
	blockTime := time.Now()
	if blockResult.Block.Header.Time != "" {
		if t, err := time.Parse(time.RFC3339Nano, blockResult.Block.Header.Time); err == nil {
			blockTime = t
		}
	}

	// Extract blobs from block transactions
	var blobs [][]byte
	var ids []datypes.ID

	for _, tx := range blockResult.Block.Data.Txs {
		// Decode base64 transaction
		txBytes, err := base64.StdEncoding.DecodeString(tx)
		if err != nil {
			// Try raw bytes if not base64
			txBytes = []byte(tx)
		}

		// Check if this is a blob transaction and extract blobs for the requested namespace
		extractedBlobs := c.extractBlobsFromTx(txBytes, ns, height)
		for _, blob := range extractedBlobs {
			blobs = append(blobs, blob.Data)
			ids = append(ids, blob.ID)
		}
	}

	if len(blobs) == 0 {
		c.logger.Debug().Uint64("height", height).Str("namespace", ns.String()).Msg("no blobs found at height")
		return datypes.ResultRetrieve{
			BaseResult: datypes.BaseResult{
				Code:      datypes.StatusNotFound,
				Message:   datypes.ErrBlobNotFound.Error(),
				Height:    height,
				Timestamp: blockTime,
			},
		}
	}

	c.logger.Debug().
		Uint64("height", height).
		Int("blob_count", len(blobs)).
		Str("namespace", ns.String()).
		Msg("retrieved blobs from block")

	return datypes.ResultRetrieve{
		BaseResult: datypes.BaseResult{
			Code:      datypes.StatusSuccess,
			Height:    height,
			IDs:       ids,
			Timestamp: blockTime,
		},
		Data: blobs,
	}
}

// extractedBlob represents a blob extracted from a transaction
type extractedBlob struct {
	Data []byte
	ID   datypes.ID
}

// extractBlobsFromTx attempts to extract blobs from a transaction.
// It parses the BlobTx format from go-square and filters by namespace.
func (c *Client) extractBlobsFromTx(txBytes []byte, targetNs share.Namespace, height uint64) []extractedBlob {
	var result []extractedBlob

	// Attempt to unmarshal as a BlobTx
	blobTx, isBlobTx, err := tx.UnmarshalBlobTx(txBytes)
	if err != nil {
		// Not a valid blob transaction, skip
		return result
	}
	if !isBlobTx {
		// Not a blob transaction, skip
		return result
	}

	// Iterate through all blobs in the transaction
	for _, blob := range blobTx.Blobs {
		// Check if this blob matches the target namespace
		if !blob.Namespace().Equals(targetNs) {
			continue
		}

		// Create commitment for the blob
		// Use merkle.HashFromByteSlices as the hasher function
		commitment, err := inclusion.CreateCommitment(blob, merkle.HashFromByteSlices, subtreeRootThreshold)
		if err != nil {
			c.logger.Debug().
				Err(err).
				Str("namespace", blob.Namespace().String()).
				Msg("failed to create commitment for blob")
			continue
		}

		// Create ID with height and commitment
		id := make([]byte, 8+len(commitment))
		binary.LittleEndian.PutUint64(id, height)
		copy(id[8:], commitment)

		result = append(result, extractedBlob{
			Data: blob.Data(),
			ID:   id,
		})
	}

	return result
}

// Get retrieves blobs by their IDs.
// Note: This implementation fetches the block and extracts the specific blobs.
func (c *Client) Get(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	// Group IDs by height for efficient fetching
	blobsByHeight := make(map[uint64][]datypes.ID)
	for _, id := range ids {
		height, _, err := datypes.SplitID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid blob id: %w", err)
		}
		blobsByHeight[height] = append(blobsByHeight[height], id)
	}

	var result []datypes.Blob
	for height, heightIDs := range blobsByHeight {
		// Fetch block at height
		retrieveResult := c.Retrieve(ctx, height, ns.Bytes())
		if retrieveResult.Code != datypes.StatusSuccess {
			continue
		}

		// Match retrieved blobs with requested IDs
		for i, blobID := range retrieveResult.IDs {
			for _, requestedID := range heightIDs {
				if bytes.Equal(blobID, requestedID) && i < len(retrieveResult.Data) {
					result = append(result, retrieveResult.Data[i])
					break
				}
			}
		}
	}

	return result, nil
}

// GetLatestDAHeight returns the latest height available on the DA layer.
func (c *Client) GetLatestDAHeight(ctx context.Context) (uint64, error) {
	status, err := c.getStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get status: %w", err)
	}
	var height uint64
	_, err = fmt.Sscanf(status.SyncInfo.LatestBlockHeight, "%d", &height)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block height: %w", err)
	}
	return height, nil
}

// GetProofs returns inclusion proofs for the provided IDs.
// Note: celestia-app doesn't provide proofs directly - they need to be computed
// from the block data or obtained from celestia-node.
func (c *Client) GetProofs(ctx context.Context, ids []datypes.ID, namespace []byte) ([]datypes.Proof, error) {
	return nil, errors.New("GetProofs not supported: celestia-app client does not support proof generation.")
}

// Validate validates commitments against the corresponding proofs.
// Note: This requires proof generation which is not implemented.
func (c *Client) Validate(ctx context.Context, ids []datypes.ID, proofs []datypes.Proof, namespace []byte) ([]bool, error) {
	return nil, errors.New("Validate not supported: celestia-app client does not support proof validation.")
}

// RPC Types for CometBFT JSON-RPC responses

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("RPC error %d: %s: %s", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

type blockResult struct {
	Block struct {
		Header struct {
			Height string `json:"height"`
			Time   string `json:"time"`
		} `json:"header"`
		Data struct {
			Txs []string `json:"txs"`
		} `json:"data"`
	} `json:"block"`
}

type statusResult struct {
	SyncInfo struct {
		LatestBlockHeight string `json:"latest_block_height"`
	} `json:"sync_info"`
}

// getBlock fetches a block at the specified height via CometBFT RPC.
func (c *Client) getBlock(ctx context.Context, height uint64) (*blockResult, error) {
	params := map[string]interface{}{
		"height": fmt.Sprintf("%d", height),
	}

	resp, err := c.rpcCall(ctx, "block", params)
	if err != nil {
		return nil, err
	}

	var result blockResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block result: %w", err)
	}

	return &result, nil
}

// getStatus fetches the node status via CometBFT RPC.
func (c *Client) getStatus(ctx context.Context) (*statusResult, error) {
	resp, err := c.rpcCall(ctx, "status", nil)
	if err != nil {
		return nil, err
	}

	var result statusResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status result: %w", err)
	}

	return &result, nil
}

// rpcCall makes a JSON-RPC call to the celestia-app CometBFT endpoint.
func (c *Client) rpcCall(ctx context.Context, method string, params interface{}) (*rpcResponse, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		reqBody["params"] = params
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcAddress, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return &rpcResp, nil
}
