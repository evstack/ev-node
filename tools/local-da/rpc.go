package main

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	libshare "github.com/celestiaorg/go-square/v3/share"
	fjrpc "github.com/filecoin-project/go-jsonrpc"
	"github.com/rs/zerolog"

	jsonrpc "github.com/evstack/ev-node/pkg/da/jsonrpc"
	datypes "github.com/evstack/ev-node/pkg/da/types"
)

// blobServer exposes a minimal Celestia-like blob RPC surface backed by LocalDA.
type blobServer struct {
	da     *LocalDA
	logger zerolog.Logger
}

// Submit stores blobs and returns the height they were included at.
func (s *blobServer) Submit(_ context.Context, blobs []*jsonrpc.Blob, _ *jsonrpc.SubmitOptions) (uint64, error) {
	s.da.mu.Lock()
	defer s.da.mu.Unlock()

	s.da.height++
	height := s.da.height

	if len(blobs) == 0 {
		s.da.timestamps[height] = time.Now()
		return height, nil
	}

	for i, b := range blobs {
		if uint64(len(b.Data())) > s.da.maxBlobSize {
			return 0, datypes.ErrBlobSizeOverLimit
		}
		if b.Commitment == nil {
			// ensure commitment exists, compute from blob
			if rebuilt, err := jsonrpc.NewBlob(b.ShareVersion(), b.Namespace(), b.Data(), b.Signer(), nil); err == nil {
				blobs[i] = rebuilt
				b = rebuilt
			}
		}
		s.da.blobData[height] = append(s.da.blobData[height], b)
	}
	s.da.timestamps[height] = time.Now()

	return height, nil
}

// Get returns a blob by height/namespace/commitment.
func (s *blobServer) Get(_ context.Context, height uint64, namespace libshare.Namespace, commitment jsonrpc.Commitment) (*jsonrpc.Blob, error) {
	s.da.mu.Lock()
	defer s.da.mu.Unlock()

	if height > s.da.height {
		return nil, datypes.ErrHeightFromFuture
	}

	blobs, ok := s.da.blobData[height]
	if !ok {
		return nil, datypes.ErrBlobNotFound
	}
	for _, b := range blobs {
		if b.Namespace().Equals(namespace) && b.EqualCommitment(commitment) {
			return b, nil
		}
	}
	return nil, datypes.ErrBlobNotFound
}

// GetAll returns blobs matching any of the provided namespaces at the given height.
func (s *blobServer) GetAll(_ context.Context, height uint64, namespaces []libshare.Namespace) ([]*jsonrpc.Blob, error) {
	s.da.mu.Lock()
	defer s.da.mu.Unlock()

	if height > s.da.height {
		return nil, datypes.ErrHeightFromFuture
	}

	blobs, ok := s.da.blobData[height]
	if !ok {
		return nil, datypes.ErrBlobNotFound
	}

	// If no namespaces specified, return everything at height.
	if len(namespaces) == 0 {
		return blobs, nil
	}

	out := make([]*jsonrpc.Blob, 0, len(blobs))
	for _, b := range blobs {
		if b != nil && slices.ContainsFunc(namespaces, b.Namespace().Equals) {
			out = append(out, b)
		}
	}

	if len(out) == 0 {
		return nil, datypes.ErrBlobNotFound
	}
	return out, nil
}

// GetProof returns a placeholder proof; LocalDA does not generate real proofs.
func (s *blobServer) GetProof(_ context.Context, _ uint64, _ libshare.Namespace, _ jsonrpc.Commitment) (*jsonrpc.Proof, error) {
	return &jsonrpc.Proof{}, nil
}

// Included reports whether a commitment is present at a given height/namespace.
func (s *blobServer) Included(_ context.Context, height uint64, namespace libshare.Namespace, _ *jsonrpc.Proof, commitment jsonrpc.Commitment) (bool, error) {
	_, err := s.Get(context.Background(), height, namespace, commitment)
	if err != nil {
		if errors.Is(err, datypes.ErrBlobNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetCommitmentProof returns a placeholder commitment proof; LocalDA does not generate real proofs.
func (s *blobServer) GetCommitmentProof(_ context.Context, _ uint64, _ libshare.Namespace, _ []byte) (*jsonrpc.CommitmentProof, error) {
	return &jsonrpc.CommitmentProof{}, nil
}

// Subscribe returns a closed channel; LocalDA does not push live updates.
func (s *blobServer) Subscribe(_ context.Context, _ libshare.Namespace) (<-chan *jsonrpc.SubscriptionResponse, error) {
	ch := make(chan *jsonrpc.SubscriptionResponse)
	close(ch)
	return ch, nil
}

// startBlobServer starts an HTTP JSON-RPC server on addr serving the blob namespace.
// headerServer exposes a minimal Celestia-like header RPC surface backed by LocalDA.
type headerServer struct {
	da     *LocalDA
	logger zerolog.Logger
}

// GetByHeight returns a mock header for the given height.
func (s *headerServer) GetByHeight(_ context.Context, height uint64) (*jsonrpc.Header, error) {
	s.da.mu.Lock()
	defer s.da.mu.Unlock()

	if height > s.da.height {
		return nil, datypes.ErrHeightFromFuture
	}

	ts, ok := s.da.timestamps[height]
	if !ok {
		// Fallback to current time if not found (should not happen if submitted)
		ts = time.Now()
	}

	return &jsonrpc.Header{
		Height:    height,
		BlockTime: ts,
		ChainID:   "local-da",
	}, nil
}

// LocalHead returns the latest local header.
func (s *headerServer) LocalHead(ctx context.Context) (*jsonrpc.Header, error) {
	s.da.mu.Lock()
	headHeight := s.da.height
	s.da.mu.Unlock()

	return s.GetByHeight(ctx, headHeight)
}

// NetworkHead returns the latest network header.
func (s *headerServer) NetworkHead(ctx context.Context) (*jsonrpc.Header, error) {
	return s.LocalHead(ctx)
}

// startBlobServer starts an HTTP JSON-RPC server on addr serving the blob namespace.
func startBlobServer(logger zerolog.Logger, addr string, da *LocalDA) (*http.Server, error) {
	rpc := fjrpc.NewServer()
	rpc.Register("blob", &blobServer{da: da, logger: logger})
	rpc.Register("header", &headerServer{da: da, logger: logger})

	srv := &http.Server{
		Addr:              addr,
		Handler:           http.HandlerFunc(rpc.ServeHTTP),
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Msg("blob RPC server failed")
		}
	}()

	return srv, nil
}
