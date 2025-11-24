package main

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/da"
)

// Server is a jsonrpc service that serves the LocalDA implementation
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

// Submit implements the RPC method.
func (s *serverInternalAPI) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte) ([]da.ID, error) {
	s.logger.Debug().Int("num_blobs", len(blobs)).Float64("gas_price", gasPrice).Str("namespace", string(ns)).Msg("RPC server: Submit called")
	return s.daImpl.Submit(ctx, blobs, gasPrice, ns)
}

// SubmitWithOptions implements the RPC method.
func (s *serverInternalAPI) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte, options []byte) ([]da.ID, error) {
	s.logger.Debug().Int("num_blobs", len(blobs)).Float64("gas_price", gasPrice).Str("namespace", string(ns)).Str("options", string(options)).Msg("RPC server: SubmitWithOptions called")
	return s.daImpl.SubmitWithOptions(ctx, blobs, gasPrice, ns, options)
}

func getKnownErrorsMapping() jsonrpc.Errors {
	errs := jsonrpc.NewErrors()
	errs.Register(jsonrpc.ErrorCode(da.StatusNotFound), &da.ErrBlobNotFound)
	errs.Register(jsonrpc.ErrorCode(da.StatusTooBig), &da.ErrBlobSizeOverLimit)
	errs.Register(jsonrpc.ErrorCode(da.StatusContextDeadline), &da.ErrTxTimedOut)
	errs.Register(jsonrpc.ErrorCode(da.StatusAlreadyInMempool), &da.ErrTxAlreadyInMempool)
	errs.Register(jsonrpc.ErrorCode(da.StatusIncorrectAccountSequence), &da.ErrTxIncorrectAccountSequence)
	errs.Register(jsonrpc.ErrorCode(da.StatusContextDeadline), &da.ErrContextDeadline)
	errs.Register(jsonrpc.ErrorCode(da.StatusContextCanceled), &da.ErrContextCanceled)
	errs.Register(jsonrpc.ErrorCode(da.StatusHeightFromFuture), &da.ErrHeightFromFuture)
	return errs
}

// NewServer creates a new JSON-RPC server for the LocalDA implementation
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
