package celestianodefiber

import (
	"context"
	"fmt"

	appfibre "github.com/celestiaorg/celestia-app/v8/fibre"
	libshare "github.com/celestiaorg/go-square/v4/share"

	"github.com/celestiaorg/celestia-node/blob"

	"github.com/evstack/ev-node/block"
)

// Listen implements fiber.DA.Listen. It subscribes to blob.Subscribe on the
// bridge node for the given namespace and forwards only share-version-2
// (Fibre) blobs as BlobEvents. PFB blobs (v0/v1) sharing the namespace are
// dropped so consumers see a pure Fibre event stream.
func (a *Adapter) Listen(ctx context.Context, namespace []byte) (<-chan block.FiberBlobEvent, error) {
	ns, err := toV0Namespace(namespace)
	if err != nil {
		return nil, fmt.Errorf("namespace: %w", err)
	}
	sub, err := a.blob.Subscribe(ctx, ns)
	if err != nil {
		return nil, fmt.Errorf("subscribing to blob stream: %w", err)
	}
	out := make(chan block.FiberBlobEvent, a.listenChannelSz)
	go forwardFibreBlobs(ctx, sub, out)
	return out, nil
}

// forwardFibreBlobs drains a blob.SubscriptionResponse stream and emits a
// BlobEvent per share-version-2 blob. The output channel is closed when the
// subscription closes or ctx is cancelled.
func forwardFibreBlobs(
	ctx context.Context,
	sub <-chan *blob.SubscriptionResponse,
	out chan<- block.FiberBlobEvent,
) {
	defer close(out)
	for {
		select {
		case resp, ok := <-sub:
			if !ok {
				return
			}
			if resp == nil {
				continue
			}
			height := resolveHeight(resp)
			for _, b := range resp.Blobs {
				if b == nil || !b.IsFibreBlob() {
					continue
				}
				event, err := fibreBlobToEvent(b.Blob, height)
				if err != nil {
					// Skip a malformed v2 blob rather than kill the
					// subscription. The server should not produce these.
					continue
				}
				select {
				case out <- event:
				case <-ctx.Done():
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// resolveHeight picks the authoritative height from the subscription
// response. celestia-node flagged resp.Height as deprecated in favour of
// resp.Header.Height(); use the header when present, fall back otherwise.
func resolveHeight(resp *blob.SubscriptionResponse) uint64 {
	if resp.Header != nil {
		return uint64(resp.Header.Height)
	}
	return resp.Height
}

// fibreBlobToEvent reconstructs the Fibre BlobID (version byte + 32-byte
// commitment) from a share-version-2 libshare.Blob and wraps it as a
// BlobEvent.
//
// DataSize caveat: a v2 share carries only (fibre_blob_version + commitment),
// not the original blob payload, so b.DataLen() is the on-chain share size
// (a fixed constant), not the user-facing "how big is this blob" number
// that ev-node's fibermock and its consumers typically expect. Reporting
// the true payload size requires an on-chain query against x/fibre's
// PaymentPromise keyed by commitment. Tracked as a follow-up; for now we
// report the share size so the field is non-zero.
func fibreBlobToEvent(b *libshare.Blob, height uint64) (block.FiberBlobEvent, error) {
	version, err := b.FibreBlobVersion()
	if err != nil {
		return block.FiberBlobEvent{}, err
	}
	commit, err := b.FibreCommitment()
	if err != nil {
		return block.FiberBlobEvent{}, err
	}
	if len(commit) != appfibre.CommitmentSize {
		return block.FiberBlobEvent{}, fmt.Errorf(
			"fibre commitment must be %d bytes, got %d",
			appfibre.CommitmentSize, len(commit),
		)
	}
	var c appfibre.Commitment
	copy(c[:], commit)
	id := appfibre.NewBlobID(uint8(version), c)
	return block.FiberBlobEvent{
		BlobID:   block.FiberBlobID(id),
		Height:   height,
		DataSize: uint64(b.DataLen()),
	}, nil
}
