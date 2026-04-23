package celestianodefiber

import (
	"context"
	"errors"
	"fmt"

	appfibre "github.com/celestiaorg/celestia-app/v8/fibre"
	libshare "github.com/celestiaorg/go-square/v4/share"

	"github.com/celestiaorg/celestia-node/blob"

	"github.com/evstack/ev-node/block"
)

// Listen implements fiber.DA.Listen. It subscribes to blob.Subscribe on the
// bridge node starting at fromHeight and forwards only share-version-2
// (Fibre) blobs as BlobEvents. PFB blobs (v0/v1) sharing the namespace are
// dropped so consumers see a pure Fibre event stream.
//
// fromHeight == 0 starts the stream at the chain head (live follow).
// fromHeight > 0 replays from that block forward via the node's
// WaitForHeight loop so a subscriber can resume after a restart without
// missing blobs.
//
// DataSize on emitted events is the original payload byte length — matching
// the fibermock contract ev-node consumers code against. The v2 share only
// carries (fibre_blob_version + commitment), so the real size isn't derivable
// from the subscription alone; Listen therefore performs a Download per event
// to recover the size before forwarding. This adds one FSP round-trip per
// blob. If that cost becomes material we can expose an opt-out mode, but for
// now correctness over latency.
func (a *Adapter) Listen(ctx context.Context, namespace []byte, fromHeight uint64) (<-chan block.FiberBlobEvent, error) {
	ns, err := toV0Namespace(namespace)
	if err != nil {
		return nil, fmt.Errorf("namespace: %w", err)
	}
	sub, err := a.blob.Subscribe(ctx, ns, fromHeight)
	if err != nil {
		return nil, fmt.Errorf("subscribing to blob stream: %w", err)
	}
	out := make(chan block.FiberBlobEvent, a.listenChannelSz)
	go a.forwardFibreBlobs(ctx, sub, out)
	return out, nil
}

// forwardFibreBlobs drains a blob.SubscriptionResponse stream and emits a
// BlobEvent per share-version-2 blob. The output channel is closed when the
// subscription closes or ctx is cancelled.
func (a *Adapter) forwardFibreBlobs(
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
				event, err := a.fibreBlobToEvent(ctx, b.Blob, height)
				if err != nil {
					// Skip a malformed or un-fetchable v2 blob rather than
					// kill the subscription. Most likely causes: the v2
					// payload was garbage-collected from FSPs, or the
					// download was cancelled. Either way the consumer has
					// no actionable signal for this single blob.
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
// commitment) from a share-version-2 libshare.Blob, downloads the blob to
// determine the original payload size, and wraps everything as a BlobEvent.
//
// The Download is what makes DataSize accurate. Without it we would have to
// either report the v2 share size (wrong — misleads consumers) or zero
// (lossy). See the Listen doc for the cost / correctness rationale.
func (a *Adapter) fibreBlobToEvent(
	ctx context.Context,
	b *libshare.Blob,
	height uint64,
) (block.FiberBlobEvent, error) {
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

	res, err := a.fibre.Download(ctx, id)
	if err != nil {
		return block.FiberBlobEvent{}, fmt.Errorf("resolving payload size via Download: %w", err)
	}
	if res == nil {
		return block.FiberBlobEvent{}, errors.New("fibre.Download returned nil result while resolving payload size")
	}

	return block.FiberBlobEvent{
		BlobID:   block.FiberBlobID(id),
		Height:   height,
		DataSize: uint64(len(res.Data)),
	}, nil
}
