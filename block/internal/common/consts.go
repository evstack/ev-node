package common

// DefaultMaxBlobSize is the fallback blob size limit used when the DA layer
// does not report one. Override at build time with:
//
//	go build -ldflags "-X github.com/evstack/ev-node/block/internal/common.DefaultMaxBlobSize=10485760"
var DefaultMaxBlobSize uint64 = 5 * 1024 * 1024 // 5MB
