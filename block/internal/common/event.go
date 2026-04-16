package common

import (
	"context"

	"github.com/evstack/ev-node/types"
)

// EventSource represents the origin of a block event
type EventSource string

const (
	// SourceDA indicates the event came from the DA layer
	SourceDA EventSource = "da"
	// SourceP2P indicates the event came from P2P network
	SourceP2P EventSource = "p2p"
	// SourceRaft indicates the event came from Raft consensus recovery
	SourceRaft EventSource = "raft"
)

// AllEventSources returns all possible event sources.
func AllEventSources() []EventSource {
	return []EventSource{SourceDA, SourceP2P, SourceRaft}
}

// DAHeightEvent represents a DA event for caching
type DAHeightEvent struct {
	Header *types.SignedHeader
	Data   *types.Data
	// DaHeight corresponds to the highest DA included height between the Header and Data.
	DaHeight uint64
	// Source indicates where this event originated from (DA or P2P)
	Source EventSource

	// Optional DA height hints from P2P. first is the DA height hint for the header, second is the DA height hint for the data
	DaHeightHints [2]uint64
}

// EventSink receives parsed DA events with backpressure support.
type EventSink interface {
	PipeEvent(ctx context.Context, event DAHeightEvent) error
}

// EventSinkFunc adapts a plain function to the EventSink interface.
// Useful in tests:
//
//	sink := common.EventSinkFunc(func(ctx context.Context, ev common.DAHeightEvent) error { return nil })
type EventSinkFunc func(ctx context.Context, event DAHeightEvent) error

func (f EventSinkFunc) PipeEvent(ctx context.Context, event DAHeightEvent) error {
	return f(ctx, event)
}
