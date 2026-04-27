package common

import "strconv"

// defaultMaxBlobSizeStr holds the string representation of the default blob
// size limit. Anchored to Fibre's actual cap: protocol MaxBlobSize
// (1 << 27 = 128 MiB) minus the 5-byte Fibre blob header (1 byte version +
// 4 byte data size). See celestia-app/v9/fibre/blob.go (blobHeaderLen)
// and fibre/protocol_params.go (MaxBlobSize).
//
// MUST be a string literal: Go's `-ldflags "-X ..."` only takes effect
// on variables initialized to a string constant, NOT a function call.
// A previous version used strconv.FormatUint here, which compiled but
// silently ignored ldflag overrides.
//
// Override at link time via:
//
//	go build -ldflags "-X github.com/evstack/ev-node/block/internal/common.defaultMaxBlobSizeStr=33554432"
var defaultMaxBlobSizeStr = "134217723" // 1 << 27 - 5 = 128 MiB - 5 B

// DefaultMaxBlobSize is the max blob size limit used for blob submission.
//
// TODO(throughput-cleanup): this single value is currently plugged in
// at two semantically different limits and the conflation has caused
// real bugs (a packed block marshals larger than its raw-tx total, so
// using MaxBlobSize as both input cap and output cap let blocks blow
// past the DA cap). Split into two:
//
//   MaxBlobSize       — chain-side ceiling on a marshaled DA blob
//   MaxBlockTxBytes() — derived raw-tx budget = MaxBlobSize - per-block
//                       marshal overhead. Used by RetrieveBatch /
//                       FilterTxs.
//
// Once that derivation exists, drop the ad-hoc 2% reservation in
// executing/executor.go::RetrieveBatch and the duplicate cap in
// submitting/da_submitter.go::defaultRetryPolicy.
var DefaultMaxBlobSize uint64

func init() {
	v, err := strconv.ParseUint(defaultMaxBlobSizeStr, 10, 64)
	if err != nil || v == 0 {
		DefaultMaxBlobSize = 134217723
		return
	}
	DefaultMaxBlobSize = v
}
