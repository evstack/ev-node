package syncing

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/evstack/ev-node/block/internal/common"
	"github.com/evstack/ev-node/types"
)

func TestCalculateBlockFullness_HalfFull(t *testing.T) {
	s := &Syncer{}

	// Create 5000 transactions of 100 bytes each = 500KB
	txs := make([]types.Tx, 5000)
	for i := range txs {
		txs[i] = make([]byte, 100)
	}

	data := &types.Data{
		Txs: txs,
	}

	fullness := s.calculateBlockFullness(data)
	// Size fullness: 500000/2097152 ≈ 0.238
	assert.InDelta(t, 0.238, fullness, 0.05)
}

func TestCalculateBlockFullness_Full(t *testing.T) {
	s := &Syncer{}

	// Create 10000 transactions of 210 bytes each = ~2MB
	txs := make([]types.Tx, 10000)
	for i := range txs {
		txs[i] = make([]byte, 210)
	}

	data := &types.Data{
		Txs: txs,
	}

	fullness := s.calculateBlockFullness(data)
	// Both metrics at or near 1.0
	assert.Greater(t, fullness, 0.95)
}

func TestCalculateBlockFullness_VerySmall(t *testing.T) {
	s := &Syncer{}

	data := &types.Data{
		Txs: []types.Tx{[]byte("tx1"), []byte("tx2")},
	}

	fullness := s.calculateBlockFullness(data)
	// Very small relative to heuristic limits
	assert.Less(t, fullness, 0.001)
}

func TestUpdateDynamicGracePeriod_NoChangeWhenBelowThreshold(t *testing.T) {
	initialMultiplier := 1.0
	initialEMA := 0.1 // Well below threshold

	config := forcedInclusionGracePeriodConfig{
		dynamicMinMultiplier:     0.5,
		dynamicMaxMultiplier:     3.0,
		dynamicFullnessThreshold: 0.8,
		dynamicAdjustmentRate:    0.01, // Low adjustment rate
	}

	s := &Syncer{
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		gracePeriodConfig:     config,
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update with low fullness - multiplier should stay at 1.0 initially
	s.updateDynamicGracePeriod(0.2)

	// With low adjustment rate and starting EMA below threshold,
	// multiplier should not change significantly on first call
	newMultiplier := *s.gracePeriodMultiplier.Load()
	assert.InDelta(t, 1.0, newMultiplier, 0.05)
}

func TestUpdateDynamicGracePeriod_IncreaseOnHighFullness(t *testing.T) {
	initialMultiplier := 1.0
	initialEMA := 0.5

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update multiple times with very high fullness to build up the effect
	for i := 0; i < 20; i++ {
		s.updateDynamicGracePeriod(0.95)
	}

	// EMA should increase
	newEMA := *s.blockFullnessEMA.Load()
	assert.Greater(t, newEMA, initialEMA)

	// Multiplier should increase because EMA is now above threshold
	newMultiplier := *s.gracePeriodMultiplier.Load()
	assert.Greater(t, newMultiplier, initialMultiplier)
}

func TestUpdateDynamicGracePeriod_DecreaseOnLowFullness(t *testing.T) {
	initialMultiplier := 2.0
	initialEMA := 0.9

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update multiple times with low fullness to build up the effect
	for i := 0; i < 20; i++ {
		s.updateDynamicGracePeriod(0.2)
	}

	// EMA should decrease significantly
	newEMA := *s.blockFullnessEMA.Load()
	assert.Less(t, newEMA, initialEMA)

	// Multiplier should decrease
	newMultiplier := *s.gracePeriodMultiplier.Load()
	assert.Less(t, newMultiplier, initialMultiplier)
}

func TestUpdateDynamicGracePeriod_ClampToMin(t *testing.T) {
	initialMultiplier := 0.6
	initialEMA := 0.1

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.5, // High rate to force clamping
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update many times with very low fullness - should eventually clamp to min
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(0.0)
	}

	newMultiplier := *s.gracePeriodMultiplier.Load()
	assert.Equal(t, 0.5, newMultiplier)
}

func TestUpdateDynamicGracePeriod_ClampToMax(t *testing.T) {
	initialMultiplier := 2.5
	initialEMA := 0.9

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.5, // High rate to force clamping
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Update many times with very high fullness - should eventually clamp to max
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(1.0)
	}

	newMultiplier := *s.gracePeriodMultiplier.Load()
	assert.Equal(t, 3.0, newMultiplier)
}

func TestGetEffectiveGracePeriod_WithMultiplier(t *testing.T) {
	multiplier := 2.5

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.05,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
	}
	s.gracePeriodMultiplier.Store(&multiplier)

	effective := s.getEffectiveGracePeriod()
	// 2 * 2.5 = 5
	assert.Equal(t, uint64(5), effective)
}

func TestGetEffectiveGracePeriod_RoundingUp(t *testing.T) {
	multiplier := 2.6

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.05,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
	}
	s.gracePeriodMultiplier.Store(&multiplier)

	effective := s.getEffectiveGracePeriod()
	// 2 * 2.6 = 5.2, rounds to 5
	assert.Equal(t, uint64(5), effective)
}

func TestGetEffectiveGracePeriod_EnsuresMinimum(t *testing.T) {
	multiplier := 0.3

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               4,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.05,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
	}
	s.gracePeriodMultiplier.Store(&multiplier)

	effective := s.getEffectiveGracePeriod()
	// 4 * 0.3 = 1.2, but minimum is 4 * 0.5 = 2
	assert.Equal(t, uint64(2), effective)
}

func TestDynamicGracePeriod_Integration_HighCongestion(t *testing.T) {
	initialMultiplier := 1.0
	initialEMA := 0.3

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Simulate processing many blocks with very high fullness (above threshold)
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(0.95)
	}

	// Multiplier should have increased due to sustained high fullness
	finalMultiplier := *s.gracePeriodMultiplier.Load()
	assert.Greater(t, finalMultiplier, initialMultiplier, "multiplier should increase with sustained congestion")

	// Effective grace period should be higher than base
	effectiveGracePeriod := s.getEffectiveGracePeriod()
	assert.Greater(t, effectiveGracePeriod, s.gracePeriodConfig.basePeriod, "effective grace period should be higher than base")
}

func TestDynamicGracePeriod_Integration_LowCongestion(t *testing.T) {
	initialMultiplier := 2.0
	initialEMA := 0.85

	s := &Syncer{
		gracePeriodConfig: forcedInclusionGracePeriodConfig{
			basePeriod:               2,
			dynamicMinMultiplier:     0.5,
			dynamicMaxMultiplier:     3.0,
			dynamicFullnessThreshold: 0.8,
			dynamicAdjustmentRate:    0.1,
		},
		gracePeriodMultiplier: &atomic.Pointer[float64]{},
		blockFullnessEMA:      &atomic.Pointer[float64]{},
		metrics:               common.NopMetrics(),
	}
	s.gracePeriodMultiplier.Store(&initialMultiplier)
	s.blockFullnessEMA.Store(&initialEMA)

	// Simulate processing many blocks with very low fullness (below threshold)
	for i := 0; i < 50; i++ {
		s.updateDynamicGracePeriod(0.1)
	}

	// Multiplier should have decreased
	finalMultiplier := *s.gracePeriodMultiplier.Load()
	assert.Less(t, finalMultiplier, initialMultiplier, "multiplier should decrease with low congestion")
}
