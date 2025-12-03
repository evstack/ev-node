package jsonrpc

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/celestiaorg/go-square/v3/share"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/blob"
)

// BlobAPI captures the blob RPC surface exposed over JSON-RPC.
type BlobAPI interface {
	Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error)
	GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error)
	GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error)
	Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error)
}

// Server is a jsonrpc service that serves the blob API surface
type Server struct {
	logger   zerolog.Logger
	srv      *http.Server
	rpc      *jsonrpc.RPCServer
	listener net.Listener
	blobAPI  BlobAPI

	started atomic.Bool
}

// serverInternalAPI provides the actual RPC methods.
type serverInternalAPI struct {
	logger  zerolog.Logger
	blobAPI BlobAPI
}

// Submit implements the RPC submit method.
func (s *serverInternalAPI) Submit(ctx context.Context, blobs []*blob.Blob, opts *blob.SubmitOptions) (uint64, error) {
	s.logger.Debug().Int("num_blobs", len(blobs)).Msg("RPC server: Submit called")
	return s.blobAPI.Submit(ctx, blobs, opts)
}

// GetAll implements the RPC method to fetch blobs by height/namespace.
func (s *serverInternalAPI) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	s.logger.Debug().Uint64("height", height).Int("namespaces", len(namespaces)).Msg("RPC server: GetAll called")
	return s.blobAPI.GetAll(ctx, height, namespaces)
}

// GetProof implements the RPC method to fetch a proof for a commitment.
func (s *serverInternalAPI) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	s.logger.Debug().Uint64("height", height).Msg("RPC server: GetProof called")
	return s.blobAPI.GetProof(ctx, height, namespace, commitment)
}

// Included implements the RPC method to verify inclusion.
func (s *serverInternalAPI) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	s.logger.Debug().Uint64("height", height).Msg("RPC server: Included called")
	return s.blobAPI.Included(ctx, height, namespace, proof, commitment)
}

// NewServer accepts the host address port and the DA implementation to serve as a jsonrpc service
func NewServer(logger zerolog.Logger, address, port string, blobAPI BlobAPI) *Server {
	rpc := jsonrpc.NewServer()
	srv := &Server{
		rpc:     rpc,
		logger:  logger,
		blobAPI: blobAPI,
		srv: &http.Server{
			Addr:              address + ":" + port,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}
	srv.srv.Handler = http.HandlerFunc(rpc.ServeHTTP)

	apiHandler := &serverInternalAPI{
		logger:  logger,
		blobAPI: blobAPI,
	}

	srv.rpc.Register("blob", apiHandler)
	return srv
}

// Start starts the RPC Server.
// This function can be called multiple times concurrently
// Once started, subsequent calls are a no-op
func (s *Server) Start(context.Context) error {
	couldStart := s.started.CompareAndSwap(false, true)

	if !couldStart {
		s.logger.Warn().Msg("cannot start server: already started")
		return nil
	}
	listener, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}
	s.listener = listener
	s.logger.Info().Str("listening_on", s.srv.Addr).Msg("server started")
	//nolint:errcheck
	go s.srv.Serve(listener)
	return nil
}

// Stop stops the RPC Server.
// This function can be called multiple times concurrently
// Once stopped, subsequent calls are a no-op
func (s *Server) Stop(ctx context.Context) error {
	couldStop := s.started.CompareAndSwap(true, false)
	if !couldStop {
		s.logger.Warn().Msg("cannot stop server: already stopped")
		return nil
	}
	err := s.srv.Shutdown(ctx)
	if err != nil {
		return err
	}
	s.listener = nil
	s.logger.Info().Msg("server stopped")
	return nil
}
