package localfiber

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-app/v9/fibre"
	"github.com/celestiaorg/go-square/v4/share"
	"github.com/evstack/ev-node/block"
)

type Adapter struct {
	client *fibre.Client
}

var _ block.FiberClient = (*Adapter)(nil)

func NewAdapter(c *fibre.Client) *Adapter {
	return &Adapter{client: c}
}

func (a *Adapter) Upload(ctx context.Context, namespace []byte, data []byte) (block.FiberUploadResult, error) {
	ns, err := share.NewNamespaceFromBytes(namespace)
	if err != nil {
		return block.FiberUploadResult{}, fmt.Errorf("invalid namespace: %w", err)
	}

	blob, err := fibre.NewBlob(data, fibre.DefaultBlobConfigV0())
	if err != nil {
		return block.FiberUploadResult{}, fmt.Errorf("creating fibre blob: %w", err)
	}

	_, err = a.client.Upload(ctx, ns, blob)
	if err != nil {
		return block.FiberUploadResult{}, fmt.Errorf("fibre upload: %w", err)
	}

	id := blob.ID()
	blobIDCopy := make([]byte, len(id))
	copy(blobIDCopy, id)

	return block.FiberUploadResult{
		BlobID:    blobIDCopy,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

func (a *Adapter) Download(ctx context.Context, blobID block.FiberBlobID) ([]byte, error) {
	var id fibre.BlobID
	if err := id.UnmarshalBinary(blobID); err != nil {
		return nil, fmt.Errorf("unmarshalling blob id: %w", err)
	}

	blob, err := a.client.Download(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fibre download: %w", err)
	}

	return blob.Data(), nil
}

func (a *Adapter) Listen(_ context.Context, _ []byte) (<-chan block.FiberBlobEvent, error) {
	ch := make(chan block.FiberBlobEvent)
	close(ch)
	return ch, nil
}
