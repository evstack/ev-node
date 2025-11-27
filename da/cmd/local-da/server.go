package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/da"
)

// Blob represents a Celestia-compatible blob for the blob API
type Blob struct {
	Namespace  []byte `json:"namespace"`
	Data       []byte `json:"data"`
	ShareVer   uint32 `json:"share_version"`
	Commitment []byte `json:"commitment"`
	Index      int    `json:"index"`
}

// Proof represents a Celestia-compatible inclusion proof
type Proof struct {
	Data []byte `json:"data"`
}

// SubmitOptions contains options for blob submission
type SubmitOptions struct {
	Fee           float64 `json:"fee,omitempty"`
	GasLimit      uint64  `json:"gas_limit,omitempty"`
	SignerAddress string  `json:"signer_address,omitempty"`
}

// Server is a jsonrpc service that serves the LocalDA implementation
type Server struct {
	logger   zerolog.Logger
	srv      *http.Server
	rpc      *jsonrpc.RPCServer
	listener net.Listener
	daImpl   da.DA
	localDA  *LocalDA // For blob API access to internal data

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
	result := s.daImpl.Submit(ctx, blobs, gasPrice, ns)
	if result.Code != da.StatusSuccess {
		return result.IDs, da.StatusCodeToError(result.Code, result.Message)
	}
	return result.IDs, nil
}

// SubmitWithOptions implements the RPC method.
func (s *serverInternalAPI) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, ns []byte, options []byte) ([]da.ID, error) {
	s.logger.Debug().Int("num_blobs", len(blobs)).Float64("gas_price", gasPrice).Str("namespace", string(ns)).Str("options", string(options)).Msg("RPC server: SubmitWithOptions called")
	result := s.daImpl.SubmitWithOptions(ctx, blobs, gasPrice, ns, options)
	if result.Code != da.StatusSuccess {
		return result.IDs, da.StatusCodeToError(result.Code, result.Message)
	}
	return result.IDs, nil
}

// blobAPI provides Celestia-compatible Blob API methods
type blobAPI struct {
	logger  zerolog.Logger
	localDA *LocalDA
}

// Submit submits blobs and returns the DA height (Celestia blob API compatible)
func (b *blobAPI) Submit(ctx context.Context, blobs []*Blob, opts *SubmitOptions) (uint64, error) {
	b.logger.Debug().Int("num_blobs", len(blobs)).Msg("blob.Submit called")

	if len(blobs) == 0 {
		return 0, nil
	}

	ns := blobs[0].Namespace

	rawBlobs := make([][]byte, len(blobs))
	for i, blob := range blobs {
		rawBlobs[i] = blob.Data
	}

	var gasPrice float64
	if opts != nil {
		gasPrice = opts.Fee
	}

	result := b.localDA.Submit(ctx, rawBlobs, gasPrice, ns)
	if result.Code != da.StatusSuccess {
		return 0, da.StatusCodeToError(result.Code, result.Message)
	}

	b.logger.Info().Uint64("height", result.Height).Int("num_blobs", len(blobs)).Msg("blob.Submit successful")
	return result.Height, nil
}

// Get retrieves a single blob by commitment at a given height (Celestia blob API compatible)
func (b *blobAPI) Get(ctx context.Context, height uint64, ns []byte, commitment []byte) (*Blob, error) {
	b.logger.Debug().Uint64("height", height).Msg("blob.Get called")

	blobs, err := b.GetAll(ctx, height, [][]byte{ns})
	if err != nil {
		return nil, err
	}

	for _, blob := range blobs {
		if len(commitment) == 0 || bytesEqual(blob.Commitment, commitment) {
			return blob, nil
		}
	}

	return nil, nil
}

// GetAll retrieves all blobs at a given height for the specified namespaces (Celestia blob API compatible)
func (b *blobAPI) GetAll(ctx context.Context, height uint64, namespaces [][]byte) ([]*Blob, error) {
	b.logger.Debug().Uint64("height", height).Int("num_namespaces", len(namespaces)).Msg("blob.GetAll called")

	if len(namespaces) == 0 {
		return []*Blob{}, nil
	}

	ns := namespaces[0]

	b.localDA.mu.Lock()
	defer b.localDA.mu.Unlock()

	if height > b.localDA.height {
		b.logger.Debug().Uint64("requested", height).Uint64("current", b.localDA.height).Msg("blob.GetAll: height in future")
		return nil, fmt.Errorf("height %d from future, current height is %d", height, b.localDA.height)
	}

	kvps, ok := b.localDA.data[height]
	if !ok {
		b.logger.Debug().Uint64("height", height).Msg("blob.GetAll: no data for height")
		return []*Blob{}, nil
	}

	blobs := make([]*Blob, 0, len(kvps))
	for i, kv := range kvps {
		var commitment []byte
		if len(kv.key) > 8 {
			commitment = kv.key[8:]
		} else {
			hash := sha256.Sum256(kv.value)
			commitment = hash[:]
		}

		blobs = append(blobs, &Blob{
			Namespace:  ns,
			Data:       kv.value,
			ShareVer:   0,
			Commitment: commitment,
			Index:      i,
		})
	}

	b.logger.Debug().Uint64("height", height).Int("num_blobs", len(blobs)).Msg("blob.GetAll successful")
	return blobs, nil
}

// GetProof retrieves the inclusion proof for a blob (Celestia blob API compatible)
func (b *blobAPI) GetProof(ctx context.Context, height uint64, ns []byte, commitment []byte) (*Proof, error) {
	b.logger.Debug().Uint64("height", height).Msg("blob.GetProof called")

	b.localDA.mu.Lock()
	defer b.localDA.mu.Unlock()

	kvps, ok := b.localDA.data[height]
	if !ok {
		return nil, nil
	}

	for _, kv := range kvps {
		var blobCommitment []byte
		if len(kv.key) > 8 {
			blobCommitment = kv.key[8:]
		}

		if len(commitment) == 0 || bytesEqual(blobCommitment, commitment) {
			proof := b.localDA.getProof(kv.key, kv.value)
			return &Proof{Data: proof}, nil
		}
	}

	return nil, nil
}

// Included checks whether a blob is included in the DA layer (Celestia blob API compatible)
func (b *blobAPI) Included(ctx context.Context, height uint64, ns []byte, proof *Proof, commitment []byte) (bool, error) {
	b.logger.Debug().Uint64("height", height).Msg("blob.Included called")

	b.localDA.mu.Lock()
	defer b.localDA.mu.Unlock()

	kvps, ok := b.localDA.data[height]
	if !ok {
		return false, nil
	}

	for _, kv := range kvps {
		var blobCommitment []byte
		if len(kv.key) > 8 {
			blobCommitment = kv.key[8:]
		}

		if bytesEqual(blobCommitment, commitment) {
			return true, nil
		}
	}

	return false, nil
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// makeID creates an ID from height and commitment
func makeID(height uint64, commitment []byte) []byte {
	id := make([]byte, 8+len(commitment))
	binary.LittleEndian.PutUint64(id, height)
	copy(id[8:], commitment)
	return id
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
// It registers both the legacy "da" namespace and the Celestia-compatible "blob" namespace
func NewServer(logger zerolog.Logger, address, port string, localDA *LocalDA) *Server {
	rpc := jsonrpc.NewServer(jsonrpc.WithServerErrors(getKnownErrorsMapping()))
	srv := &Server{
		rpc:     rpc,
		logger:  logger,
		daImpl:  localDA,
		localDA: localDA,
		srv: &http.Server{
			Addr:              address + ":" + port,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}
	srv.srv.Handler = http.HandlerFunc(rpc.ServeHTTP)

	// Register legacy "da" namespace API
	daAPIHandler := &serverInternalAPI{
		logger: logger,
		daImpl: localDA,
	}
	srv.rpc.Register("da", daAPIHandler)

	// Register Celestia-compatible "blob" namespace API
	blobAPIHandler := &blobAPI{
		logger:  logger,
		localDA: localDA,
	}
	srv.rpc.Register("blob", blobAPIHandler)

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
