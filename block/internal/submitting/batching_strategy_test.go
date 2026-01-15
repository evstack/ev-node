package submitting

import (
	"testing"
	"time"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImmediateStrategy(t *testing.T) {
	strategy := &ImmediateStrategy{}

	tests := []struct {
		name         string
		pendingCount uint64
		totalSize    int
		expected     bool
	}{
		{
			name:         "no pending items",
			pendingCount: 0,
			totalSize:    0,
			expected:     false,
		},
		{
			name:         "one pending item",
			pendingCount: 1,
			totalSize:    1000,
			expected:     true,
		},
		{
			name:         "multiple pending items",
			pendingCount: 10,
			totalSize:    10000,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.ShouldSubmit(tt.pendingCount, tt.totalSize, common.DefaultMaxBlobSize, 0)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSizeBasedStrategy(t *testing.T) {
	maxBlobSize := 8 * 1024 * 1024 // 8MB

	tests := []struct {
		name           string
		sizeThreshold  float64
		minItems       uint64
		pendingCount   uint64
		totalSize      int
		expectedSubmit bool
	}{
		{
			name:           "below threshold and min items",
			sizeThreshold:  0.8,
			minItems:       2,
			pendingCount:   1,
			totalSize:      1 * 1024 * 1024, // 1MB
			expectedSubmit: false,
		},
		{
			name:           "below threshold but has min items",
			sizeThreshold:  0.8,
			minItems:       1,
			pendingCount:   5,
			totalSize:      4 * 1024 * 1024, // 4MB (50% of 8MB)
			expectedSubmit: false,
		},
		{
			name:           "at threshold with min items",
			sizeThreshold:  0.8,
			minItems:       1,
			pendingCount:   10,
			totalSize:      int(float64(maxBlobSize) * 0.8), // 80% of max
			expectedSubmit: true,
		},
		{
			name:           "above threshold",
			sizeThreshold:  0.8,
			minItems:       1,
			pendingCount:   20,
			totalSize:      7 * 1024 * 1024, // 7MB (87.5% of 8MB)
			expectedSubmit: true,
		},
		{
			name:           "full blob",
			sizeThreshold:  0.8,
			minItems:       1,
			pendingCount:   100,
			totalSize:      maxBlobSize,
			expectedSubmit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewSizeBasedStrategy(tt.sizeThreshold, tt.minItems)
			result := strategy.ShouldSubmit(tt.pendingCount, tt.totalSize, maxBlobSize, 0)
			assert.Equal(t, tt.expectedSubmit, result)
		})
	}

	// Test invalid threshold defaults to 0.8
	strategy := NewSizeBasedStrategy(1.5, 1)
	assert.Equal(t, 0.8, strategy.sizeThreshold)

	strategy = NewSizeBasedStrategy(0, 1)
	assert.Equal(t, 0.8, strategy.sizeThreshold)
}

func TestTimeBasedStrategy(t *testing.T) {
	maxDelay := 6 * time.Second
	maxBlobSize := 8 * 1024 * 1024

	tests := []struct {
		name                string
		minItems            uint64
		pendingCount        uint64
		totalSize           int
		timeSinceLastSubmit time.Duration
		expectedSubmit      bool
	}{
		{
			name:                "below min items",
			minItems:            2,
			pendingCount:        1,
			totalSize:           1 * 1024 * 1024,
			timeSinceLastSubmit: 10 * time.Second,
			expectedSubmit:      false,
		},
		{
			name:                "before max delay",
			minItems:            1,
			pendingCount:        5,
			totalSize:           4 * 1024 * 1024,
			timeSinceLastSubmit: 3 * time.Second,
			expectedSubmit:      false,
		},
		{
			name:                "at max delay",
			minItems:            1,
			pendingCount:        3,
			totalSize:           2 * 1024 * 1024,
			timeSinceLastSubmit: 6 * time.Second,
			expectedSubmit:      true,
		},
		{
			name:                "after max delay",
			minItems:            1,
			pendingCount:        2,
			totalSize:           1 * 1024 * 1024,
			timeSinceLastSubmit: 10 * time.Second,
			expectedSubmit:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewTimeBasedStrategy(maxDelay, tt.minItems)
			result := strategy.ShouldSubmit(tt.pendingCount, tt.totalSize, maxBlobSize, tt.timeSinceLastSubmit)
			assert.Equal(t, tt.expectedSubmit, result)
		})
	}
}

func TestAdaptiveStrategy(t *testing.T) {
	maxBlobSize := 8 * 1024 * 1024 // 8MB
	sizeThreshold := 0.8
	maxDelay := 6 * time.Second

	tests := []struct {
		name                string
		minItems            uint64
		pendingCount        uint64
		totalSize           int
		timeSinceLastSubmit time.Duration
		expectedSubmit      bool
		reason              string
	}{
		{
			name:                "below min items",
			minItems:            3,
			pendingCount:        2,
			totalSize:           7 * 1024 * 1024,
			timeSinceLastSubmit: 10 * time.Second,
			expectedSubmit:      false,
			reason:              "not enough items",
		},
		{
			name:                "size threshold reached",
			minItems:            1,
			pendingCount:        10,
			totalSize:           int(float64(maxBlobSize) * 0.85), // 85%
			timeSinceLastSubmit: 1 * time.Second,
			expectedSubmit:      true,
			reason:              "size threshold met",
		},
		{
			name:                "time threshold reached",
			minItems:            1,
			pendingCount:        2,
			totalSize:           1 * 1024 * 1024, // Only 12.5%
			timeSinceLastSubmit: 7 * time.Second,
			expectedSubmit:      true,
			reason:              "time threshold met",
		},
		{
			name:                "neither threshold reached",
			minItems:            1,
			pendingCount:        5,
			totalSize:           4 * 1024 * 1024, // 50%
			timeSinceLastSubmit: 3 * time.Second,
			expectedSubmit:      false,
			reason:              "waiting for threshold",
		},
		{
			name:                "both thresholds reached",
			minItems:            1,
			pendingCount:        20,
			totalSize:           7 * 1024 * 1024, // 87.5%
			timeSinceLastSubmit: 10 * time.Second,
			expectedSubmit:      true,
			reason:              "both thresholds met",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewAdaptiveStrategy(sizeThreshold, maxDelay, tt.minItems)
			result := strategy.ShouldSubmit(tt.pendingCount, tt.totalSize, maxBlobSize, tt.timeSinceLastSubmit)
			assert.Equal(t, tt.expectedSubmit, result, "reason: %s", tt.reason)
		})
	}

	// Test defaults
	strategy := NewAdaptiveStrategy(0, 0, 0)
	assert.Equal(t, 0.8, strategy.sizeThreshold)
	assert.Equal(t, 6*time.Second, strategy.maxDelay)
	assert.Equal(t, uint64(1), strategy.minItems)
}

func TestNewBatchingStrategy(t *testing.T) {
	tests := []struct {
		name         string
		strategyName string
		expectedType string
		expectError  bool
	}{
		{
			name:         "immediate strategy",
			strategyName: "immediate",
			expectError:  false,
		},
		{
			name:         "size strategy",
			strategyName: "size",
			expectError:  false,
		},
		{
			name:         "time strategy",
			strategyName: "time",
			expectError:  false,
		},
		{
			name:         "adaptive strategy",
			strategyName: "adaptive",
			expectError:  false,
		},
		{
			name:         "unknown strategy",
			strategyName: "unknown",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DAConfig{
				BatchingStrategy:   tt.strategyName,
				BatchSizeThreshold: 0.8,
				BatchMaxDelay:      config.DurationWrapper{Duration: 6 * time.Second},
				BatchMinItems:      1,
			}

			strategy, err := NewBatchingStrategy(cfg)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, strategy)
			} else {
				require.NoError(t, err)
				require.NotNil(t, strategy)
			}
		})
	}
}

func TestEstimateBatchSize(t *testing.T) {
	tests := []struct {
		name         string
		marshaled    [][]byte
		expectedSize int
	}{
		{
			name:         "empty batch",
			marshaled:    [][]byte{},
			expectedSize: 0,
		},
		{
			name: "single item",
			marshaled: [][]byte{
				make([]byte, 1024),
			},
			expectedSize: 1024,
		},
		{
			name: "multiple items",
			marshaled: [][]byte{
				make([]byte, 1024),
				make([]byte, 2048),
				make([]byte, 512),
			},
			expectedSize: 3584,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := estimateBatchSize(tt.marshaled)
			assert.Equal(t, tt.expectedSize, size)
		})
	}
}

func TestOptimizeBatchSize(t *testing.T) {
	maxBlobSize := 8 * 1024 * 1024 // 8MB

	tests := []struct {
		name              string
		itemSizes         []int
		targetUtilization float64
		expectedCount     int
		expectedTotalSize int
	}{
		{
			name:              "empty batch",
			itemSizes:         []int{},
			targetUtilization: 0.9,
			expectedCount:     0,
			expectedTotalSize: 0,
		},
		{
			name:              "single small item",
			itemSizes:         []int{1024},
			targetUtilization: 0.9,
			expectedCount:     1,
			expectedTotalSize: 1024,
		},
		{
			name:              "reach target utilization",
			itemSizes:         []int{1024 * 1024, 2 * 1024 * 1024, 3 * 1024 * 1024, 1024 * 1024},
			targetUtilization: 0.8,
			expectedCount:     4, // 1+2+3+1 = 7MB (87.5% of 8MB, exceeds 80% target so stops)
			expectedTotalSize: 7 * 1024 * 1024,
		},
		{
			name:              "stop at max blob size",
			itemSizes:         []int{7 * 1024 * 1024, 2 * 1024 * 1024},
			targetUtilization: 0.9,
			expectedCount:     1, // Second item would exceed max
			expectedTotalSize: 7 * 1024 * 1024,
		},
		{
			name:              "all items fit below target",
			itemSizes:         []int{1024 * 1024, 1024 * 1024, 1024 * 1024},
			targetUtilization: 0.9,
			expectedCount:     3,
			expectedTotalSize: 3 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create marshaled data
			marshaled := make([][]byte, len(tt.itemSizes))
			for i, size := range tt.itemSizes {
				marshaled[i] = make([]byte, size)
			}

			count := optimizeBatchSize(marshaled, maxBlobSize, tt.targetUtilization)
			assert.Equal(t, tt.expectedCount, count)

			if count > 0 {
				totalSize := estimateBatchSize(marshaled[:count])
				assert.Equal(t, tt.expectedTotalSize, totalSize)
			}
		})
	}
}

func TestCalculateBatchMetrics(t *testing.T) {
	maxBlobSize := 8 * 1024 * 1024

	tests := []struct {
		name              string
		itemCount         int
		totalBytes        int
		expectedUtil      float64
		expectedCostRange [2]float64 // min, max
	}{
		{
			name:              "empty batch",
			itemCount:         0,
			totalBytes:        0,
			expectedUtil:      0.0,
			expectedCostRange: [2]float64{0, 999999}, // cost is undefined for empty
		},
		{
			name:              "half full",
			itemCount:         10,
			totalBytes:        4 * 1024 * 1024,
			expectedUtil:      0.5,
			expectedCostRange: [2]float64{2.0, 2.0}, // 1/0.5 = 2.0x cost
		},
		{
			name:              "80% full",
			itemCount:         20,
			totalBytes:        int(float64(maxBlobSize) * 0.8),
			expectedUtil:      0.8,
			expectedCostRange: [2]float64{1.25, 1.25}, // 1/0.8 = 1.25x cost
		},
		{
			name:              "nearly full",
			itemCount:         50,
			totalBytes:        int(float64(maxBlobSize) * 0.95),
			expectedUtil:      0.95,
			expectedCostRange: [2]float64{1.05, 1.06}, // ~1.05x cost
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := calculateBatchMetrics(tt.itemCount, tt.totalBytes, maxBlobSize)

			assert.Equal(t, tt.itemCount, metrics.ItemCount)
			assert.Equal(t, tt.totalBytes, metrics.TotalBytes)
			assert.Equal(t, maxBlobSize, metrics.MaxBlobBytes)
			assert.InDelta(t, tt.expectedUtil, metrics.Utilization, 0.01)

			if tt.totalBytes > 0 {
				assert.InEpsilon(t, (tt.expectedCostRange[0]+tt.expectedCostRange[1])/2,
					metrics.EstimatedCost, 0.01, "cost should be within range")
			}
		})
	}
}

func TestShouldWaitForMoreItems(t *testing.T) {
	maxBlobSize := 8 * 1024 * 1024

	tests := []struct {
		name            string
		currentCount    uint64
		currentSize     int
		minUtilization  float64
		hasMoreExpected bool
		expectedWait    bool
	}{
		{
			name:            "near capacity",
			currentCount:    50,
			currentSize:     int(float64(maxBlobSize) * 0.96),
			minUtilization:  0.8,
			hasMoreExpected: true,
			expectedWait:    false,
		},
		{
			name:            "below threshold but no more expected",
			currentCount:    10,
			currentSize:     4 * 1024 * 1024,
			minUtilization:  0.8,
			hasMoreExpected: false,
			expectedWait:    false,
		},
		{
			name:            "below threshold with more expected",
			currentCount:    10,
			currentSize:     4 * 1024 * 1024, // 50%
			minUtilization:  0.8,
			hasMoreExpected: true,
			expectedWait:    true,
		},
		{
			name:            "at threshold",
			currentCount:    20,
			currentSize:     int(float64(maxBlobSize) * 0.8),
			minUtilization:  0.8,
			hasMoreExpected: true,
			expectedWait:    false, // At threshold, no need to wait
		},
		{
			name:            "above threshold",
			currentCount:    30,
			currentSize:     int(float64(maxBlobSize) * 0.85),
			minUtilization:  0.8,
			hasMoreExpected: true,
			expectedWait:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldWaitForMoreItems(
				tt.currentCount,
				tt.currentSize,
				maxBlobSize,
				tt.minUtilization,
				tt.hasMoreExpected,
			)
			assert.Equal(t, tt.expectedWait, result)
		})
	}
}

func TestBatchingStrategiesComparison(t *testing.T) {
	// This test demonstrates how different strategies behave with the same input
	maxBlobSize := 8 * 1024 * 1024
	pendingCount := uint64(10)
	totalSize := 4 * 1024 * 1024 // 50% full
	timeSinceLastSubmit := 3 * time.Second

	immediate := &ImmediateStrategy{}
	size := NewSizeBasedStrategy(0.8, 1)
	timeBased := NewTimeBasedStrategy(6*time.Second, 1)
	adaptive := NewAdaptiveStrategy(0.8, 6*time.Second, 1)

	// Immediate should always submit if there are items
	assert.True(t, immediate.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))

	// Size-based should not submit at 50% when threshold is 80%
	assert.False(t, size.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))

	// Time-based should not submit at 3s when max delay is 6s
	assert.False(t, timeBased.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))

	// Adaptive should not submit (neither threshold met)
	assert.False(t, adaptive.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))

	// Now test with time threshold exceeded
	timeSinceLastSubmit = 7 * time.Second
	assert.True(t, immediate.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))
	assert.False(t, size.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))
	assert.True(t, timeBased.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))
	assert.True(t, adaptive.ShouldSubmit(pendingCount, totalSize, maxBlobSize, timeSinceLastSubmit))
}
