package common

import "strconv"

// defaultMaxBlobSizeStr holds the string representation of the default blob
// size limit. Override at link time via:
//
//	go build -ldflags "-X github.com/evstack/ev-node/block/internal/common.defaultMaxBlobSizeStr=125829120"
var defaultMaxBlobSizeStr = "5242880" // 5 MB

// DefaultMaxBlobSize is the max blob size limit used for blob submission.
var DefaultMaxBlobSize uint64 = 5 * 1024 * 1024

func init() {
	v, err := strconv.ParseUint(defaultMaxBlobSizeStr, 10, 64)
	if err != nil || v == 0 {
		DefaultMaxBlobSize = 5 * 1024 * 1024
		return
	}
	DefaultMaxBlobSize = v
}
