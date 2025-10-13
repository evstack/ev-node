package common

import "github.com/evstack/ev-node/types"

// EventSource represents the origin of a block event
type EventSource string

const (
	// SourceDA indicates the event came from the DA layer
	SourceDA EventSource = "DA"
	// SourceP2P indicates the event came from P2P network
	SourceP2P EventSource = "P2P"
)

// DAHeightEvent represents a DA event for caching
type DAHeightEvent struct {
	Header *types.SignedHeader
	Data   *types.Data
	// DaHeight corresponds to the highest DA included height between the Header and Data.
	DaHeight uint64
	// Source indicates where this event originated from (DA or P2P)
	Source EventSource
}
