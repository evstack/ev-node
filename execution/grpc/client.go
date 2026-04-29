package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/evstack/ev-node/core/execution"
	pb "github.com/evstack/ev-node/execution/grpc/types/pb/evnode/v1"
	"github.com/evstack/ev-node/execution/grpc/types/pb/evnode/v1/v1connect"
)

// Ensure Client implements the execution.Executor interface
var _ execution.Executor = (*Client)(nil)

// Client is a gRPC client that implements the execution.Executor interface.
// It communicates with a remote execution service via gRPC using Connect-RPC.
type Client struct {
	client v1connect.ExecutorServiceClient
}

const (
	unixURLPrefix   = "unix://"
	unixHTTPBaseURL = "http://unix"
)

// newHTTP2Client creates an HTTP/2 client that supports cleartext (h2c) connections.
// This is required to connect to native gRPC servers without TLS.
func newHTTP2Client() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
}

// newUnixHTTP2Client creates an HTTP/2 client that speaks h2c over a Unix domain socket.
func newUnixHTTP2Client(socketPath string) (*http.Client, error) {
	if socketPath == "" {
		return nil, errors.New("unix socket path is required")
	}
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, _, _ string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
	}, nil
}

func clientTransportForTarget(target string) (*http.Client, string, error) {
	socketPath, ok := unixSocketPath(target)
	if ok {
		httpClient, err := newUnixHTTP2Client(socketPath)
		if err != nil {
			return nil, "", err
		}
		return httpClient, unixHTTPBaseURL, nil
	}
	return newHTTP2Client(), target, nil
}

func unixSocketPath(target string) (string, bool) {
	if !strings.HasPrefix(target, unixURLPrefix) {
		return "", false
	}
	return strings.TrimPrefix(target, unixURLPrefix), true
}

// NewClient creates a new gRPC execution client.
//
// Parameters:
// - url: The URL of the gRPC server (e.g., "http://localhost:50051" or "unix:///tmp/executor.sock")
// - opts: Optional Connect client options for configuring the connection
//
// Returns:
// - *Client: The initialized gRPC client
// - error: Any client construction error
func NewClient(url string, opts ...connect.ClientOption) (*Client, error) {
	// Prepend WithGRPC to use the native gRPC protocol (required for tonic/gRPC servers)
	opts = append([]connect.ClientOption{connect.WithGRPC()}, opts...)
	httpClient, targetURL, err := clientTransportForTarget(url)
	if err != nil {
		return nil, err
	}
	opts = append([]connect.ClientOption{connect.WithInterceptors(outboundPropagationInterceptor())}, opts...)
	return &Client{
		client: v1connect.NewExecutorServiceClient(
			httpClient,
			targetURL,
			opts...,
		),
	}, nil
}

// InitChain initializes a new blockchain instance with genesis parameters.
//
// This method sends an InitChain request to the remote execution service and
// returns the initial state root.
func (c *Client) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) (stateRoot []byte, err error) {
	req := connect.NewRequest(&pb.InitChainRequest{
		GenesisTime:   timestamppb.New(genesisTime),
		InitialHeight: initialHeight,
		ChainId:       chainID,
	})

	resp, err := c.client.InitChain(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc client: failed to init chain: %w", err)
	}

	return resp.Msg.StateRoot, nil
}

// GetTxs fetches available transactions from the execution layer's mempool.
//
// This method retrieves transactions that are ready to be included in a block.
// The execution service may perform validation and filtering before returning transactions.
func (c *Client) GetTxs(ctx context.Context) ([][]byte, error) {
	req := connect.NewRequest(&pb.GetTxsRequest{})

	resp, err := c.client.GetTxs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc client: failed to get txs: %w", err)
	}

	txs, err := decodeTxBatch(resp.Msg.TxBatch)
	if err != nil {
		return nil, fmt.Errorf("grpc client: invalid get txs response: %w", err)
	}

	return txs, nil
}

// ExecuteTxs processes transactions to produce a new block state.
//
// This method sends transactions to the execution service for processing and
// returns the updated state root after execution. The execution service ensures
// deterministic execution and validates the state transition.
func (c *Client) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) (updatedStateRoot []byte, err error) {
	txBatch, err := encodeTxBatch(txs)
	if err != nil {
		return nil, fmt.Errorf("grpc client: failed to encode tx batch: %w", err)
	}

	req := connect.NewRequest(&pb.ExecuteTxsRequest{
		TxBatch:       txBatch,
		BlockHeight:   blockHeight,
		Timestamp:     timestamppb.New(timestamp),
		PrevStateRoot: prevStateRoot,
	})

	resp, err := c.client.ExecuteTxs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc client: failed to execute txs: %w", err)
	}

	return resp.Msg.UpdatedStateRoot, nil
}

// SetFinal marks a block as finalized at the specified height.
//
// This method notifies the execution service that a block has been finalized,
// allowing it to perform cleanup operations and commit the state permanently.
func (c *Client) SetFinal(ctx context.Context, blockHeight uint64) error {
	req := connect.NewRequest(&pb.SetFinalRequest{
		BlockHeight: blockHeight,
	})

	_, err := c.client.SetFinal(ctx, req)
	if err != nil {
		return fmt.Errorf("grpc client: failed to set final: %w", err)
	}

	return nil
}

// GetExecutionInfo returns current execution layer parameters.
//
// This method retrieves execution parameters such as the block gas limit
// from the remote execution service.
func (c *Client) GetExecutionInfo(ctx context.Context) (execution.ExecutionInfo, error) {
	req := connect.NewRequest(&pb.GetExecutionInfoRequest{})

	resp, err := c.client.GetExecutionInfo(ctx, req)
	if err != nil {
		return execution.ExecutionInfo{}, fmt.Errorf("grpc client: failed to get execution info: %w", err)
	}

	return execution.ExecutionInfo{
		MaxGas: resp.Msg.MaxGas,
	}, nil
}

// FilterTxs validates force-included transactions and applies gas and size filtering.
//
// This method sends transactions to the remote execution service for validation.
// Returns a slice of FilterStatus for each transaction.
func (c *Client) FilterTxs(ctx context.Context, txs [][]byte, maxBytes, maxGas uint64, hasForceIncludedTransaction bool) ([]execution.FilterStatus, error) {
	txBatch, err := encodeTxBatch(txs)
	if err != nil {
		return nil, fmt.Errorf("grpc client: failed to encode tx batch: %w", err)
	}

	req := connect.NewRequest(&pb.FilterTxsRequest{
		TxBatch:                     txBatch,
		MaxBytes:                    maxBytes,
		MaxGas:                      maxGas,
		HasForceIncludedTransaction: hasForceIncludedTransaction,
	})

	resp, err := c.client.FilterTxs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc client: failed to filter transactions: %w", err)
	}

	// Convert protobuf FilterStatus to execution.FilterStatus
	result := make([]execution.FilterStatus, len(resp.Msg.Statuses))
	for i, status := range resp.Msg.Statuses {
		result[i] = execution.FilterStatus(status)
	}
	return result, nil
}
