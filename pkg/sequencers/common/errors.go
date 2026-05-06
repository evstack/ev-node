package common

import "errors"

var (
	// ErrInvalidId is returned when the chain id is invalid
	ErrInvalidID = errors.New("invalid chain id")
	// ErrQueueFull is returned when the batch queue has reached its maximum size
	ErrQueueFull = errors.New("sequencer queue full")
)
