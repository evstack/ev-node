package jsonrpc

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/core/da"
)

// Server is a jsonrpc service that can serve the DA interface
type Server struct {
	logger   zerolog.Logger
	srv      *http.Server
	rpc      *jsonrpc.RPCServer
	listener net.Listener
	daImpl   da.DA

	started atomic.Bool
}

// serverInternalAPI provides the actual RPC methods.
type serverInternalAPI struct {
	logger zerolog.Logger
	daImpl da.DA
}

// Get implements the RPC method.
func (s *serverInternalAPI) Get(ctx context.Context, ids []da.ID, ns []byte) ([]da.Blob, error) {
	s.logger.Debug().Int("num_ids", len(ids)).Str("namespace", string(ns)).Msg("RPC server: Get called")
	return s.daImpl.Get(ctx, ids, ns)
}

// GetIDs implements the RPC method.
func (s *serverInternalAPI) GetIDs(ctx context.Context, height uint64, ns []byte) (*da.GetIDsResult, error) {
	s.logger.Debug().Uint64("height", height).Str("namespace", string(ns)).Msg("RPC server: GetIDs called")
	return s.daImpl.GetIDs(ctx, height, ns)
}

// GetProofs implements the RPC method.
func (s *serverInternalAPI) GetProofs(ctx context.Context, ids []da.ID, ns []byte) ([]da.Proof, error) {
	s.logger.Debug().Int("num_ids", len(ids)).Str("namespace", string(ns)).Msg("RPC server: GetProofs called")
	return s.daImpl.GetProofs(ctx, ids, ns)
}

// Commit implements the RPC method.
func (s *serverInternalAPI) Commit(ctx context.Context, blobs []da.Blob, ns []byte) ([]da.Commitment, error) {
	s.logger.Debug().Int("num_blobs", len(blobs)).Str("namespace", string(ns)).Msg("RPC server: Commit called")
	return s.daImpl.Commit(ctx, blobs, ns)
}

// Validate implements the RPC method.
func (s *serverInternalAPI) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, ns []byte) ([]bool, error) {
	s.logger.Debug().Int("num_ids", len(ids)).Int("num_proofs", len(proofs)).Str("namespace", string(ns)).Msg("RPC server: Validate called")
	return s.daImpl.Validate(ctx, ids, proofs, ns)
}

// Submit implements the RPC method. This is the primary submit method which includes options.
func (s *serverInternalAPI) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte) ([]da.ID, error) {
	s.logger.Debug().Int("num_blobs", len(blobs)).Float64("gas_price", gasPrice).Str("namespace", string(ns)).Msg("RPC server: Submit called")
	return s.daImpl.Submit(ctx, blobs, gasPrice, ns)
}

// SubmitWithOptions implements the RPC method.
func (s *serverInternalAPI) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte, options []byte) ([]da.ID, error) {
	s.logger.Debug().Int("num_blobs", len(blobs)).Float64("gas_price", gasPrice).Str("namespace", string(ns)).Str("options", string(options)).Msg("RPC server: SubmitWithOptions called")
	return s.daImpl.SubmitWithOptions(ctx, blobs, gasPrice, ns, options)
}

// GasPrice implements the RPC method.
func (s *serverInternalAPI) GasPrice(ctx context.Context) (float64, error) {
	s.logger.Debug().Msg("RPC server: GasPrice called")
	return s.daImpl.GasPrice(ctx)
}

// GasMultiplier implements the RPC method.
func (s *serverInternalAPI) GasMultiplier(ctx context.Context) (float64, error) {
	s.logger.Debug().Msg("RPC server: GasMultiplier called")
	return s.daImpl.GasMultiplier(ctx)
}

// NewServer accepts the host address port and the DA implementation to serve as a jsonrpc service
func NewServer(logger zerolog.Logger, address, port string, daImplementation da.DA) *Server {
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
