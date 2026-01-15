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

func NewTimeBasedStrategy(maxDelay time.Duration, minItems uint64) *TimeBasedStrategy {
	if maxDelay == 0 {
		maxDelay = 6 * time.Second // default to DA block time
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

func NewAdaptiveStrategy(sizeThreshold float64, maxDelay time.Duration, minItems uint64) *AdaptiveStrategy {
	if sizeThreshold <= 0 || sizeThreshold > 1.0 {
		sizeThreshold = 0.8 // default to 80%
	}
	if maxDelay == 0 {
		maxDelay = 6 * time.Second // default to DA block time
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

// NewBatchingStrategy creates a batching strategy based on configuration
func NewBatchingStrategy(cfg config.DAConfig) (BatchingStrategy, error) {
	switch cfg.BatchingStrategy {
	case "immediate":
		return &ImmediateStrategy{}, nil
	case "size":
		return NewSizeBasedStrategy(cfg.BatchSizeThreshold, cfg.BatchMinItems), nil
	case "time":
		return NewTimeBasedStrategy(cfg.BatchMaxDelay.Duration, cfg.BatchMinItems), nil
	case "adaptive":
		return NewAdaptiveStrategy(cfg.BatchSizeThreshold, cfg.BatchMaxDelay.Duration, cfg.BatchMinItems), nil
	default:
		return nil, fmt.Errorf("unknown batching strategy: %s", cfg.BatchingStrategy)
	}
}

// estimateBatchSize estimates the total size of pending items
// This is a helper function that can be used by the submitter
func estimateBatchSize(marshaled [][]byte) int {
	totalSize := 0
	for _, data := range marshaled {
		totalSize += len(data)
	}
	return totalSize
}

// optimizeBatchSize returns the optimal number of items to include in a batch
// to maximize blob utilization while staying under the size limit
func optimizeBatchSize(marshaled [][]byte, maxBlobSize int, targetUtilization float64) int {
	if targetUtilization <= 0 || targetUtilization > 1.0 {
		targetUtilization = 0.9 // default to 90% utilization
	}

	targetSize := int(float64(maxBlobSize) * targetUtilization)
	totalSize := 0
	count := 0

	for i, data := range marshaled {
		itemSize := len(data)

		// If adding this item would exceed max blob size, stop
		if totalSize+itemSize > maxBlobSize {
			break
		}

		totalSize += itemSize
		count = i + 1

		// If we've reached our target utilization, we can stop
		// This helps create more predictably-sized batches
		if totalSize >= targetSize {
			break
		}
	}

	return count
}

// BatchMetrics provides information about batch efficiency
type BatchMetrics struct {
	ItemCount     int
	TotalBytes    int
	MaxBlobBytes  int
	Utilization   float64 // percentage of max blob size used
	EstimatedCost float64 // estimated cost relative to single full blob
}

// calculateBatchMetrics computes metrics for a batch
func calculateBatchMetrics(itemCount int, totalBytes int, maxBlobBytes int) BatchMetrics {
	utilization := 0.0
	if maxBlobBytes > 0 {
		utilization = float64(totalBytes) / float64(maxBlobBytes)
	}

	// Rough cost estimate: each blob submission has a fixed cost
	// Higher utilization = better cost efficiency
	estimatedCost := 1.0
	if utilization > 0 {
		// If we're only using 50% of the blob, we're paying 2x per byte effectively
		estimatedCost = 1.0 / utilization
	}

	return BatchMetrics{
		ItemCount:     itemCount,
		TotalBytes:    totalBytes,
		MaxBlobBytes:  maxBlobBytes,
		Utilization:   utilization,
		EstimatedCost: estimatedCost,
	}
}

// ShouldWaitForMoreItems determines if we should wait for more items
// to improve batch efficiency
func ShouldWaitForMoreItems(
	currentCount uint64,
	currentSize int,
	maxBlobSize int,
	minUtilization float64,
	hasMoreExpected bool,
) bool {
	// Don't wait if we're already at or near capacity
	if currentSize >= int(float64(maxBlobSize)*0.95) {
		return false
	}

	// Don't wait if we don't expect more items soon
	if !hasMoreExpected {
		return false
	}

	// Wait if current utilization is below minimum threshold
	// Use epsilon for floating point comparison
	const epsilon = 0.001
	currentUtilization := float64(currentSize) / float64(maxBlobSize)

	return currentUtilization < minUtilization-epsilon
}

// BatchingConfig holds configuration for batch optimization
type BatchingConfig struct {
	MaxBlobSize       int
	Strategy          BatchingStrategy
	TargetUtilization float64
}
