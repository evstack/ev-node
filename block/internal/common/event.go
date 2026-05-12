package common

import (
	"context"

	"github.com/evstack/ev-node/types"
)

type EventSource string

const (
	SourceDA   EventSource = "da"
	SourceRaft EventSource = "raft"
)

func AllEventSources() []EventSource {
	return []EventSource{SourceDA, SourceRaft}
}

type DAHeightEvent struct {
	Header   *types.SignedHeader
	Data     *types.Data
	DaHeight uint64
	Source   EventSource
}

type EventSink interface {
	PipeEvent(ctx context.Context, event DAHeightEvent) error
}

type EventSinkFunc func(ctx context.Context, event DAHeightEvent) error

func (f EventSinkFunc) PipeEvent(ctx context.Context, event DAHeightEvent) error {
	return f(ctx, event)
}
