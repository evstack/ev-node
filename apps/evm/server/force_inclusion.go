package server

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/core/da"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

const (
	defaultForceInclusionAddr = "127.0.0.1:8547"
	defaultReadTimeout        = 30 * time.Second
	defaultWriteTimeout       = 30 * time.Second
)

// ForceInclusionServer provides an Ethereum-compatible JSON-RPC endpoint
// that accepts transactions and submits them directly to the DA layer for force inclusion
type ForceInclusionServer struct {
	server          *http.Server
	daClient        da.DA
	config          config.Config
	genesis         genesis.Genesis
	logger          zerolog.Logger
	namespace       []byte
	executionRPCURL string
	httpClient      *http.Client
}

// NewForceInclusionServer creates a new force inclusion server
func NewForceInclusionServer(
	addr string,
	daClient da.DA,
	cfg config.Config,
	gen genesis.Genesis,
	logger zerolog.Logger,
	executionRPCURL string,
) (*ForceInclusionServer, error) {
	if addr == "" {
		addr = defaultForceInclusionAddr
	}

	if len(cfg.DA.GetForcedInclusionNamespace()) == 0 {
		return nil, errors.New("forced inclusion namespace is empty")
	}

	namespace := da.NamespaceFromString(cfg.DA.GetForcedInclusionNamespace()).Bytes()

	s := &ForceInclusionServer{
		daClient:        daClient,
		config:          cfg,
		genesis:         gen,
		logger:          logger.With().Str("module", "force_inclusion_api").Logger(),
		namespace:       namespace,
		executionRPCURL: executionRPCURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleJSONRPC)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}

	return s, nil
}

// Start starts the HTTP server
func (s *ForceInclusionServer) Start(ctx context.Context) error {
	s.logger.Info().
		Str("address", s.server.Addr).
		Msg("starting force inclusion API server")

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("failed to start force inclusion server: %w", err)
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// Stop gracefully stops the HTTP server
func (s *ForceInclusionServer) Stop(ctx context.Context) error {
	s.logger.Info().Msg("stopping force inclusion API server")
	return s.server.Shutdown(ctx)
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// handleJSONRPC handles JSON-RPC requests
func (s *ForceInclusionServer) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, nil, MethodNotFound, "method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, nil, ParseError, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, nil, ParseError, "invalid JSON")
		return
	}

	s.logger.Debug().
		Str("method", req.Method).
		Interface("id", req.ID).
		Msg("received JSON-RPC request")

	switch req.Method {
	case "eth_sendRawTransaction":
		s.handleSendRawTransaction(w, &req)
	default:
		s.proxyToExecutionRPC(w, &req, body)
	}
}

// handleChainID handles eth_chainId requests
func (s *ForceInclusionServer) handleChainID(w http.ResponseWriter, req *JSONRPCRequest) {
	// Convert chain ID string to integer
	chainIDInt, err := strconv.ParseUint(s.genesis.ChainID, 10, 64)
	if err != nil {
		s.writeError(w, req.ID, InternalError, fmt.Sprintf("invalid chain ID: %v", err))
		return
	}
	// Return the chain ID as a hex string prefixed with 0x
	chainID := fmt.Sprintf("0x%x", chainIDInt)
	s.writeSuccess(w, req.ID, chainID)
}

// proxyToExecutionRPC forwards unknown RPC methods to the execution RPC endpoint
func (s *ForceInclusionServer) proxyToExecutionRPC(w http.ResponseWriter, req *JSONRPCRequest, body []byte) {
	if s.executionRPCURL == "" {
		s.writeError(w, req.ID, MethodNotFound, fmt.Sprintf("method %s not found", req.Method))
		return
	}

	s.logger.Debug().
		Str("method", req.Method).
		Str("execution_rpc_url", s.executionRPCURL).
		Msg("proxying request to execution RPC")

	proxyReq, err := http.NewRequest(http.MethodPost, s.executionRPCURL, strings.NewReader(string(body)))
	if err != nil {
		s.writeError(w, req.ID, InternalError, fmt.Sprintf("failed to create proxy request: %v", err))
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(proxyReq)
	if err != nil {
		s.writeError(w, req.ID, InternalError, fmt.Sprintf("failed to proxy request: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.writeError(w, req.ID, InternalError, fmt.Sprintf("failed to read proxy response: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(respBody); err != nil {
		s.logger.Error().Err(err).Msg("failed to write proxy response")
	}
}

// handleSendRawTransaction handles eth_sendRawTransaction requests
func (s *ForceInclusionServer) handleSendRawTransaction(w http.ResponseWriter, req *JSONRPCRequest) {
	var params []string
	if err := json.Unmarshal(req.Params, &params); err != nil || len(params) != 1 {
		s.writeError(w, req.ID, InvalidParams, "invalid params: expected [\"0x...\"]")
		return
	}

	txHex := params[0]
	txData, err := s.decodeHexTx(txHex)
	if err != nil {
		s.writeError(w, req.ID, InvalidParams, fmt.Sprintf("invalid transaction hex: %v", err))
		return
	}

	if len(txData) == 0 {
		s.writeError(w, req.ID, InvalidParams, "transaction data cannot be empty")
		return
	}

	s.logger.Info().
		Int("tx_size", len(txData)).
		Msg("submitting transaction to DA for force inclusion")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	blobs := [][]byte{txData}
	options := []byte(s.config.DA.SubmitOptions)
	gasPrice := -1.0 // auto gas price

	ids, err := s.daClient.SubmitWithOptions(ctx, blobs, gasPrice, s.namespace, options)
	if err != nil {
		s.writeError(w, req.ID, InternalError, fmt.Sprintf("failed to submit to DA: %v", err))
		return
	}

	if len(ids) == 0 {
		s.writeError(w, req.ID, InternalError, "no DA IDs returned")
		return
	}

	// Extract height from the first ID
	// IDs are structured with height in the first 8 bytes (little-endian uint64)
	if len(ids[0]) < 8 {
		s.writeError(w, req.ID, InternalError, "invalid DA ID format")
		return
	}
	daHeight := binary.LittleEndian.Uint64(ids[0][:8])

	s.logger.Info().
		Uint64("da_height", daHeight).
		Msg("transaction successfully submitted to DA layer")

	_, epochEnd, _ := types.CalculateEpochBoundaries(
		daHeight,
		s.genesis.DAStartHeight,
		s.genesis.DAEpochForcedInclusion,
	)
	blocksUntilInclusion := epochEnd - (daHeight + 1)

	s.logger.Info().
		Uint64("blocks_until_inclusion", blocksUntilInclusion).
		Uint64("inclusion_at_height", epochEnd+1).
		Msg("transaction will be force included")

	// Return a transaction hash-like response
	// We use the DA height as part of the response since we don't have the actual tx hash yet
	txHash := fmt.Sprintf("0x%064x", daHeight)
	s.writeSuccess(w, req.ID, txHash)
}

// decodeHexTx decodes a hex-encoded Ethereum transaction
func (s *ForceInclusionServer) decodeHexTx(hexStr string) ([]byte, error) {
	hexStr = strings.TrimSpace(hexStr)

	if !strings.HasPrefix(hexStr, "0x") && !strings.HasPrefix(hexStr, "0X") {
		return nil, fmt.Errorf("hex string must start with 0x")
	}

	hexStr = hexStr[2:]

	txBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}

	return txBytes, nil
}

// writeSuccess writes a successful JSON-RPC response
func (s *ForceInclusionServer) writeSuccess(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeResponse(w, resp)
}

// writeError writes an error JSON-RPC response
func (s *ForceInclusionServer) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	s.writeResponse(w, resp)
}

// writeResponse writes a JSON-RPC response
func (s *ForceInclusionServer) writeResponse(w http.ResponseWriter, resp JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error().Err(err).Msg("failed to write response")
	}
}
