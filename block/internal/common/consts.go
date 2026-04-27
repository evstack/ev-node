package common

import "strconv"

// fibreMaxPayload is the maximum payload Fibre will accept in a single
// blob: protocol max blob size (1 << 27 = 128 MiB) minus the 5-byte
// Fibre blob header (1 byte version + 4 byte data size). See
// celestia-app/v9/fibre/blob.go (blobHeaderLen) and fibre/protocol_params.go
// (MaxBlobSize). Sized to fit a full Fibre blob and nothing more.
const fibreMaxPayload = (1 << 27) - 5

// defaultMaxBlobSizeStr holds the string representation of the default blob
// size limit. Override at link time via:
//
//	go build -ldflags "-X github.com/evstack/ev-node/block/internal/common.defaultMaxBlobSizeStr=125829120"
var defaultMaxBlobSizeStr = strconv.FormatUint(fibreMaxPayload, 10)

// DefaultMaxBlobSize is the max blob size limit used for blob submission.
var DefaultMaxBlobSize uint64

func init() {
	v, err := strconv.ParseUint(defaultMaxBlobSizeStr, 10, 64)
	if err != nil || v == 0 {
		DefaultMaxBlobSize = fibreMaxPayload
		return
	}
	DefaultMaxBlobSize = v
}
