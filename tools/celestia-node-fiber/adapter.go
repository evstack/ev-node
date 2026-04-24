package celestianodefiber

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	appfibre "github.com/celestiaorg/celestia-app/v8/fibre"
	libshare "github.com/celestiaorg/go-square/v4/share"

	"github.com/celestiaorg/celestia-node/api/client"
	blobapi "github.com/celestiaorg/celestia-node/nodebuilder/blob"
	fibreapi "github.com/celestiaorg/celestia-node/nodebuilder/fibre"

	"github.com/evstack/ev-node/block"
)

// blobExpiration is the advisory ExpiresAt returned from Upload. celestia-node
// does not yet expose per-blob retention via UploadResult; 24h mirrors the
// placeholder used by tools/local-fiber so ev-node callers do not treat
// freshly-uploaded blobs as already-expired.
//
// TODO: surface actual retention from the x/fibre protocol params once
// celestia-node exposes it.
const blobExpiration = 24 * time.Hour

// defaultListenChannelSize matches the buffer used by celestia-node's
// blob.Subscribe so the forwarder does not become a tighter bottleneck than
// the upstream stream.
const defaultListenChannelSize = 16

// Adapter implements the ev-node fiber.DA interface on top of a
// celestia-node api/client.Client. Upload and Download run locally against
// consensus gRPC + FSPs; Listen forwards share-version-2 blobs from the
// bridge node's subscription stream.
type Adapter struct {
	fibre           fibreapi.Module
	blob            blobapi.Module
	listenChannelSz int

	// closer, if non-nil, is invoked by Close. Set only when the Adapter
	// owns the underlying api/client.Client (via New).
	closer func() error
}

// Compile-time assertion that Adapter satisfies the ev-node Fiber DA contract.
var _ block.FiberClient = (*Adapter)(nil)

// New constructs a celestia-node api/client.Client from cfg and wraps it as
// an Adapter. The returned Adapter owns the client and must be closed via
// Close to release its gRPC connections.
func New(ctx context.Context, cfg Config, kr keyring.Keyring) (*Adapter, error) {
	c, err := client.New(ctx, cfg.Client, kr)
	if err != nil {
		return nil, fmt.Errorf("constructing celestia-node client: %w", err)
	}
	return &Adapter{
		fibre:           c.Fibre,
		blob:            c.Blob,
		listenChannelSz: resolveListenChannelSize(cfg.ListenChannelSize),
		closer:          c.Close,
	}, nil
}

// FromModules wraps existing Fibre and Blob module implementations. It is
// intended for tests and for callers that already own a *client.Client and
// want to pass its Fibre + Blob fields directly. The caller retains
// responsibility for the underlying client's lifecycle; Close is a no-op.
func FromModules(fibre fibreapi.Module, blob blobapi.Module, listenChannelSize int) *Adapter {
	return &Adapter{
		fibre:           fibre,
		blob:            blob,
		listenChannelSz: resolveListenChannelSize(listenChannelSize),
	}
}

// Close tears down the underlying client, if the Adapter owns one.
func (a *Adapter) Close() error {
	if a.closer == nil {
		return nil
	}
	return a.closer()
}

// Upload implements fiber.DA.Upload. client.Fibre.Upload does off-chain row
// upload plus validator-sig aggregation and spawns a background
// MsgPayForFibre broadcast; this call returns as soon as the off-chain
// stages finish.
func (a *Adapter) Upload(
	ctx context.Context,
	namespace []byte,
	data []byte,
) (block.FiberUploadResult, error) {
	ns, err := toV0Namespace(namespace)
	if err != nil {
		return block.FiberUploadResult{}, fmt.Errorf("namespace: %w", err)
	}
	up, err := a.fibre.Upload(ctx, ns, data, nil)
	if err != nil {
		return block.FiberUploadResult{}, fmt.Errorf("fibre upload: %w", err)
	}
	if up == nil {
		return block.FiberUploadResult{}, errors.New("fibre upload returned nil result")
	}
	// Copy the returned BlobID to decouple the caller from any internal
	// reuse of the upstream slice.
	id := make(block.FiberBlobID, len(up.BlobID))
	copy(id, up.BlobID)
	return block.FiberUploadResult{
		BlobID:    id,
		ExpiresAt: time.Now().Add(blobExpiration),
	}, nil
}

// Download implements fiber.DA.Download. Reads go directly to FSPs via the
// appfibre client embedded in client.Fibre — no bridge hop.
func (a *Adapter) Download(ctx context.Context, blobID block.FiberBlobID) ([]byte, error) {
	res, err := a.fibre.Download(ctx, appfibre.BlobID(blobID))
	if err != nil {
		return nil, fmt.Errorf("fibre download: %w", err)
	}
	if res == nil {
		return nil, errors.New("fibre download returned nil result")
	}
	return res.Data, nil
}

// toV0Namespace converts an ev-node raw namespace ([]byte) into a libshare
// Namespace. The adapter contract is that callers pass the 10-byte v0
// namespace ID (matching ev-node's datypes.NamespaceFromString output); we
// surface a clear error on length mismatch rather than silently padding.
func toV0Namespace(id []byte) (libshare.Namespace, error) {
	if len(id) != libshare.NamespaceVersionZeroIDSize {
		return libshare.Namespace{}, fmt.Errorf(
			"expected %d bytes, got %d",
			libshare.NamespaceVersionZeroIDSize, len(id),
		)
	}
	return libshare.NewV0Namespace(id)
}

func resolveListenChannelSize(size int) int {
	if size <= 0 {
		return defaultListenChannelSize
	}
	return size
}
