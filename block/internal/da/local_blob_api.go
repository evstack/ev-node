package da

import (
	"context"
	"fmt"
	"sync"

	"github.com/celestiaorg/go-square/v3/share"

	celestia "github.com/evstack/ev-node/da/celestia"
	"github.com/evstack/ev-node/pkg/blob"
)

// LocalBlobAPI is a simple in-memory BlobAPI implementation for tests.
type LocalBlobAPI struct {
	mu       sync.Mutex
	height   uint64
	maxSize  uint64
	byHeight map[uint64][]*blob.Blob
}

// NewLocalBlobAPI creates an in-memory BlobAPI with a max blob size.
func NewLocalBlobAPI(maxSize uint64) *LocalBlobAPI {
	return &LocalBlobAPI{
		maxSize:  maxSize,
		byHeight: make(map[uint64][]*blob.Blob),
	}
}

func (l *LocalBlobAPI) Submit(ctx context.Context, blobs []*blob.Blob, _ *blob.SubmitOptions) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, b := range blobs {
		if uint64(len(b.Data())) > l.maxSize {
			return 0, fmt.Errorf("blob %d too big", i)
		}
	}

	l.height++
	// store clones to avoid external mutation
	stored := make([]*blob.Blob, len(blobs))
	copy(stored, blobs)
	l.byHeight[l.height] = append(l.byHeight[l.height], stored...)
	return l.height, nil
}

func (l *LocalBlobAPI) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	nsMap := make(map[string]struct{}, len(namespaces))
	for _, ns := range namespaces {
		nsMap[string(ns.Bytes())] = struct{}{}
	}

	blobs, ok := l.byHeight[height]
	if !ok {
		return []*blob.Blob{}, nil
	}
	var out []*blob.Blob
	for _, b := range blobs {
		if _, ok := nsMap[string(b.Namespace().Bytes())]; ok {
			out = append(out, b)
		}
	}
	return out, nil
}

func (l *LocalBlobAPI) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	return &blob.Proof{}, nil
}

func (l *LocalBlobAPI) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	return true, nil
}

func (l *LocalBlobAPI) GetCommitmentProof(ctx context.Context, height uint64, namespace share.Namespace, shareCommitment []byte) (*celestia.CommitmentProof, error) {
	return &celestia.CommitmentProof{}, nil
}

func (l *LocalBlobAPI) Subscribe(ctx context.Context, namespace share.Namespace) (<-chan *celestia.SubscriptionResponse, error) {
	ch := make(chan *celestia.SubscriptionResponse)
	close(ch)
	return ch, nil
}
