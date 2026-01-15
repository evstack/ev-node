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
			strategy := NewTimeBasedStrategy(6*time.Second, maxDelay, tt.minItems)
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
			strategy := NewAdaptiveStrategy(6*time.Second, sizeThreshold, maxDelay, tt.minItems)
			result := strategy.ShouldSubmit(tt.pendingCount, tt.totalSize, maxBlobSize, tt.timeSinceLastSubmit)
			assert.Equal(t, tt.expectedSubmit, result, "reason: %s", tt.reason)
		})
	}

	// Test defaults
	strategy := NewAdaptiveStrategy(6*time.Second, 0, 0, 0)
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

func TestBatchingStrategiesComparison(t *testing.T) {
	// This test demonstrates how different strategies behave with the same input
	maxBlobSize := 8 * 1024 * 1024
	pendingCount := uint64(10)
	totalSize := 4 * 1024 * 1024 // 50% full
	timeSinceLastSubmit := 3 * time.Second

	immediate := &ImmediateStrategy{}
	size := NewSizeBasedStrategy(0.8, 1)
	timeBased := NewTimeBasedStrategy(6*time.Second, 6*time.Second, 1)
	adaptive := NewAdaptiveStrategy(6*time.Second, 0.8, 6*time.Second, 1)

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
