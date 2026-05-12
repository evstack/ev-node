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
	// SourceRaft indicates the event came from Raft consensus recovery
	SourceRaft EventSource = "raft"
)

// AllEventSources returns all possible event sources.
func AllEventSources() []EventSource {
	return []EventSource{SourceDA, SourceRaft}
}

// DAHeightEvent represents a DA event for caching
type DAHeightEvent struct {
	Header *types.SignedHeader
	Data   *types.Data
	// DaHeight corresponds to the highest DA included height between the Header and Data.
	DaHeight uint64
	// Source indicates where this event originated from (DA or Raft)
	Source EventSource
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
