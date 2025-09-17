package common

import "github.com/evstack/ev-node/types"

// DAHeightEvent represents a DA event for caching
type DAHeightEvent struct {
	Header *types.SignedHeader
	Data   *types.Data
	// DaHeight corresponds to the highest DA included height between the Header and Data.
	DaHeight uint64
	// HeaderDaIncludedHeight corresponds to the DA height at which the Header was included.
	HeaderDaIncludedHeight uint64
}
