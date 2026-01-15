package submitting

import (
	"fmt"
	"time"

	"github.com/evstack/ev-node/pkg/config"
)

// BatchingStrategy defines the interface for different batching strategies
type BatchingStrategy interface {
	// ShouldSubmit determines if a batch should be submitted based on the strategy
	// Returns true if submission should happen now
	ShouldSubmit(pendingCount uint64, totalSize int, maxBlobSize int, timeSinceLastSubmit time.Duration) bool
}

// NewBatchingStrategy creates a batching strategy based on configuration
func NewBatchingStrategy(cfg config.DAConfig) (BatchingStrategy, error) {
	switch cfg.BatchingStrategy {
	case "immediate":
		return &ImmediateStrategy{}, nil
	case "size":
		return NewSizeBasedStrategy(cfg.BatchSizeThreshold, cfg.BatchMinItems), nil
	case "time":
		return NewTimeBasedStrategy(cfg.BlockTime.Duration, cfg.BatchMaxDelay.Duration, cfg.BatchMinItems), nil
	case "adaptive":
		return NewAdaptiveStrategy(cfg.BlockTime.Duration, cfg.BatchSizeThreshold, cfg.BatchMaxDelay.Duration, cfg.BatchMinItems), nil
	default:
		return nil, fmt.Errorf("unknown batching strategy: %s", cfg.BatchingStrategy)
	}
}

// ImmediateStrategy submits as soon as any items are available
type ImmediateStrategy struct{}

func (s *ImmediateStrategy) ShouldSubmit(pendingCount uint64, totalSize int, maxBlobSize int, timeSinceLastSubmit time.Duration) bool {
	return pendingCount > 0
}

// SizeBasedStrategy waits until the batch reaches a certain size threshold
type SizeBasedStrategy struct {
	sizeThreshold float64 // fraction of max blob size (0.0 to 1.0)
	minItems      uint64
}

func NewSizeBasedStrategy(sizeThreshold float64, minItems uint64) *SizeBasedStrategy {
	if sizeThreshold <= 0 || sizeThreshold > 1.0 {
		sizeThreshold = 0.8 // default to 80%
	}
	if minItems == 0 {
		minItems = 1
	}
	return &SizeBasedStrategy{
		sizeThreshold: sizeThreshold,
		minItems:      minItems,
	}
}

func (s *SizeBasedStrategy) ShouldSubmit(pendingCount uint64, totalSize int, maxBlobSize int, timeSinceLastSubmit time.Duration) bool {
	if pendingCount < s.minItems {
		return false
	}

	threshold := int(float64(maxBlobSize) * s.sizeThreshold)
	return totalSize >= threshold
}

// TimeBasedStrategy submits after a certain time interval
type TimeBasedStrategy struct {
	maxDelay time.Duration
	minItems uint64
}

func NewTimeBasedStrategy(daBlockTime time.Duration, maxDelay time.Duration, minItems uint64) *TimeBasedStrategy {
	if maxDelay == 0 {
		maxDelay = daBlockTime
	}
	if minItems == 0 {
		minItems = 1
	}
	return &TimeBasedStrategy{
		maxDelay: maxDelay,
		minItems: minItems,
	}
}

func (s *TimeBasedStrategy) ShouldSubmit(pendingCount uint64, totalSize int, maxBlobSize int, timeSinceLastSubmit time.Duration) bool {
	if pendingCount < s.minItems {
		return false
	}

	return timeSinceLastSubmit >= s.maxDelay
}

// AdaptiveStrategy balances between size and time constraints
// It submits when either:
// - The batch reaches the size threshold, OR
// - The max delay is reached and we have at least min items
type AdaptiveStrategy struct {
	sizeThreshold float64
	maxDelay      time.Duration
	minItems      uint64
}

func NewAdaptiveStrategy(daBlockTime time.Duration, sizeThreshold float64, maxDelay time.Duration, minItems uint64) *AdaptiveStrategy {
	if sizeThreshold <= 0 || sizeThreshold > 1.0 {
		sizeThreshold = 0.8 // default to 80%
	}
	if maxDelay == 0 {
		maxDelay = daBlockTime
	}
	if minItems == 0 {
		minItems = 1
	}
	return &AdaptiveStrategy{
		sizeThreshold: sizeThreshold,
		maxDelay:      maxDelay,
		minItems:      minItems,
	}
}

func (s *AdaptiveStrategy) ShouldSubmit(pendingCount uint64, totalSize int, maxBlobSize int, timeSinceLastSubmit time.Duration) bool {
	if pendingCount < s.minItems {
		return false
	}

	// Submit if we've reached the size threshold
	threshold := int(float64(maxBlobSize) * s.sizeThreshold)
	if totalSize >= threshold {
		return true
	}

	// Submit if max delay has been reached
	if timeSinceLastSubmit >= s.maxDelay {
		return true
	}

	return false
}
