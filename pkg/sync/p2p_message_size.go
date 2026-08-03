package sync

import (
	"github.com/celestiaorg/go-libp2p-messenger/serde"

	"github.com/evstack/ev-node/pkg/blobsize"
)

func init() {
	// go-header wraps block data in another protobuf message, so leave room for
	// framing overhead beyond the maximum block payload.
	serde.MaxMessageSize = 2 * blobsize.DefaultMaxBlobSize
}
