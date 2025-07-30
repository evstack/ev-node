package jsonrpc

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/filecoin-project/go-jsonrpc"

	"go.uber.org/zap"

	coreda "github.com/evstack/ev-node/core/da"
)

// Server is a jsonrpc service that can serve the DA interface
type Server struct {
	logger   *zap.Logger
	srv      *http.Server
	rpc      *jsonrpc.RPCServer
	listener net.Listener
	daImpl   coreda.DA

	started atomic.Bool
}

// serverInternalAPI provides the actual RPC methods.
type serverInternalAPI struct {
	logger *zap.Logger
	daImpl coreda.DA
}

// Get implements the RPC method.
func (s *serverInternalAPI) Get(ctx context.Context, ids []coreda.ID, ns []byte) ([]coreda.Blob, error) {
	s.logger.Debug("RPC server: Get called", zap.Int("num_ids", len(ids)), zap.String("namespace", string(ns)))
	return s.daImpl.Get(ctx, ids, ns)
}

// GetIDs implements the RPC method.
func (s *serverInternalAPI) GetIDs(ctx context.Context, height uint64, ns []byte) (*coreda.GetIDsResult, error) {
	s.logger.Debug("RPC server: GetIDs called", zap.Uint64("height", height), zap.String("namespace", string(ns)))
	return s.daImpl.GetIDs(ctx, height, ns)
}

// GetProofs implements the RPC method.
func (s *serverInternalAPI) GetProofs(ctx context.Context, ids []coreda.ID, ns []byte) ([]coreda.Proof, error) {
	s.logger.Debug("RPC server: GetProofs called", zap.Int("num_ids", len(ids)), zap.String("namespace", string(ns)))
	return s.daImpl.GetProofs(ctx, ids, ns)
}

// Commit implements the RPC method.
func (s *serverInternalAPI) Commit(ctx context.Context, blobs []coreda.Blob, ns []byte) ([]coreda.Commitment, error) {
	s.logger.Debug("RPC server: Commit called", zap.Int("num_blobs", len(blobs)), zap.String("namespace", string(ns)))
	return s.daImpl.Commit(ctx, blobs, ns)
}

// Validate implements the RPC method.
func (s *serverInternalAPI) Validate(ctx context.Context, ids []coreda.ID, proofs []coreda.Proof, ns []byte) ([]bool, error) {
	s.logger.Debug("RPC server: Validate called", zap.Int("num_ids", len(ids)), zap.Int("num_proofs", len(proofs)), zap.String("namespace", string(ns)))
	return s.daImpl.Validate(ctx, ids, proofs, ns)
}

// Submit implements the RPC method. This is the primary submit method which includes options.
func (s *serverInternalAPI) Submit(ctx context.Context, blobs []coreda.Blob, gasPrice float64, ns []byte) ([]coreda.ID, error) {
	s.logger.Debug("RPC server: Submit called", zap.Int("num_blobs", len(blobs)), zap.Float64("gas_price", gasPrice), zap.String("namespace", string(ns)))
	return s.daImpl.Submit(ctx, blobs, gasPrice, ns)
}

// SubmitWithOptions implements the RPC method.
func (s *serverInternalAPI) SubmitWithOptions(ctx context.Context, blobs []coreda.Blob, gasPrice float64, ns []byte, options []byte) ([]coreda.ID, error) {
	s.logger.Debug("RPC server: SubmitWithOptions called", zap.Int("num_blobs", len(blobs)), zap.Float64("gas_price", gasPrice), zap.String("namespace", string(ns)), zap.String("options", string(options)))
	return s.daImpl.SubmitWithOptions(ctx, blobs, gasPrice, ns, options)
}

// GasPrice implements the RPC method.
func (s *serverInternalAPI) GasPrice(ctx context.Context) (float64, error) {
	s.logger.Debug("RPC server: GasPrice called")
	return s.daImpl.GasPrice(ctx)
}

// GasMultiplier implements the RPC method.
func (s *serverInternalAPI) GasMultiplier(ctx context.Context) (float64, error) {
	s.logger.Debug("RPC server: GasMultiplier called")
	return s.daImpl.GasMultiplier(ctx)
}

// NewServer accepts the host address port and the DA implementation to serve as a jsonrpc service
func NewServer(logger *zap.Logger, address, port string, daImplementation coreda.DA) *Server {
	rpc := jsonrpc.NewServer(jsonrpc.WithServerErrors(getKnownErrorsMapping()))
	srv := &Server{
		rpc:    rpc,
		logger: logger,
		daImpl: daImplementation,
		srv: &http.Server{
			Addr:              address + ":" + port,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}
	srv.srv.Handler = http.HandlerFunc(rpc.ServeHTTP)

	apiHandler := &serverInternalAPI{
		logger: logger,
		daImpl: daImplementation,
	}

	srv.rpc.Register("da", apiHandler)
	return srv
}

// Start starts the RPC Server.
// This function can be called multiple times concurrently
// Once started, subsequent calls are a no-op
func (s *Server) Start(context.Context) error {
	couldStart := s.started.CompareAndSwap(false, true)

	if !couldStart {
		s.logger.Warn("cannot start server: already started")
		return nil
	}
	listener, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}
	s.listener = listener
	s.logger.Info("server started", zap.String("address", s.srv.Addr))
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
		s.logger.Warn("cannot stop server: already stopped")
		return nil
	}
	err := s.srv.Shutdown(ctx)
	if err != nil {
		return err
	}
	s.listener = nil
	s.logger.Info("server stopped")
	return nil
}
