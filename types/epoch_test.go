package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateEpochNumber(t *testing.T) {
	tests := []struct {
		name          string
		daStartHeight uint64
		daEpochSize   uint64
		daHeight      uint64
		expectedEpoch uint64
	}{
		{
			name:          "first epoch - start height",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      100,
			expectedEpoch: 1,
		},
		{
			name:          "first epoch - middle",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      105,
			expectedEpoch: 1,
		},
		{
			name:          "first epoch - last height",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      109,
			expectedEpoch: 1,
		},
		{
			name:          "second epoch - start",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      110,
			expectedEpoch: 2,
		},
		{
			name:          "second epoch - middle",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      115,
			expectedEpoch: 2,
		},
		{
			name:          "tenth epoch",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      195,
			expectedEpoch: 10,
		},
		{
			name:          "before start height",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      50,
			expectedEpoch: 0,
		},
		{
			name:          "zero epoch size",
			daStartHeight: 100,
			daEpochSize:   0,
			daHeight:      200,
			expectedEpoch: 1,
		},
		{
			name:          "large epoch size",
			daStartHeight: 1000,
			daEpochSize:   1000,
			daHeight:      2500,
			expectedEpoch: 2,
		},
		{
			name:          "start height zero",
			daStartHeight: 0,
			daEpochSize:   5,
			daHeight:      10,
			expectedEpoch: 3,
		},
		{
			name:          "epoch size one",
			daStartHeight: 100,
			daEpochSize:   1,
			daHeight:      105,
			expectedEpoch: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epoch := CalculateEpochNumber(tt.daHeight, tt.daStartHeight, tt.daEpochSize)
			assert.Equal(t, tt.expectedEpoch, epoch)
		})
	}
}

func TestCalculateEpochBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		daStartHeight uint64
		daEpochSize   uint64
		daHeight      uint64
		expectedStart uint64
		expectedEnd   uint64
	}{
		{
			name:          "first epoch",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      105,
			expectedStart: 100,
			expectedEnd:   109,
		},
		{
			name:          "second epoch",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      110,
			expectedStart: 110,
			expectedEnd:   119,
		},
		{
			name:          "third epoch - last height",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      129,
			expectedStart: 120,
			expectedEnd:   129,
		},
		{
			name:          "before start height returns first epoch",
			daStartHeight: 100,
			daEpochSize:   10,
			daHeight:      50,
			expectedStart: 100,
			expectedEnd:   109,
		},
		{
			name:          "before start height with zero epoch size",
			daStartHeight: 2,
			daEpochSize:   0,
			daHeight:      1,
			expectedStart: 2,
			expectedEnd:   2,
		},
		{
			name:          "zero epoch size",
			daStartHeight: 100,
			daEpochSize:   0,
			daHeight:      200,
			expectedStart: 100,
			expectedEnd:   100,
		},
		{
			name:          "large epoch",
			daStartHeight: 1000,
			daEpochSize:   1000,
			daHeight:      1500,
			expectedStart: 1000,
			expectedEnd:   1999,
		},
		{
			name:          "epoch boundary exact start",
			daStartHeight: 100,
			daEpochSize:   50,
			daHeight:      100,
			expectedStart: 100,
			expectedEnd:   149,
		},
		{
			name:          "epoch boundary exact end of first epoch",
			daStartHeight: 100,
			daEpochSize:   50,
			daHeight:      149,
			expectedStart: 100,
			expectedEnd:   149,
		},
		{
			name:          "epoch boundary exact start of second epoch",
			daStartHeight: 100,
			daEpochSize:   50,
			daHeight:      150,
			expectedStart: 150,
			expectedEnd:   199,
		},
		{
			name:          "start height zero",
			daStartHeight: 0,
			daEpochSize:   5,
			daHeight:      10,
			expectedStart: 10,
			expectedEnd:   14,
		},
		{
			name:          "epoch size one",
			daStartHeight: 100,
			daEpochSize:   1,
			daHeight:      105,
			expectedStart: 105,
			expectedEnd:   105,
		},
		{
			name:          "very large numbers",
			daStartHeight: 1000000,
			daEpochSize:   100000,
			daHeight:      5500000,
			expectedStart: 5500000,
			expectedEnd:   5599999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, _ := CalculateEpochBoundaries(tt.daHeight, tt.daStartHeight, tt.daEpochSize)
			assert.Equal(t, tt.expectedStart, start, "start height mismatch")
			assert.Equal(t, tt.expectedEnd, end, "end height mismatch")
		})
	}
}

func TestEpochConsistency(t *testing.T) {
	tests := []struct {
		name          string
		daStartHeight uint64
		daEpochSize   uint64
	}{
		{
			name:          "standard epoch",
			daStartHeight: 100,
			daEpochSize:   10,
		},
		{
			name:          "large epoch",
			daStartHeight: 1000,
			daEpochSize:   1000,
		},
		{
			name:          "small epoch",
			daStartHeight: 0,
			daEpochSize:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that all heights in an epoch return the same epoch number
			// and boundaries
			for epoch := uint64(1); epoch <= 10; epoch++ {
				// Calculate expected boundaries for this epoch
				expectedStart := tt.daStartHeight + (epoch-1)*tt.daEpochSize
				expectedEnd := tt.daStartHeight + epoch*tt.daEpochSize - 1

				// Test every height in the epoch
				for h := expectedStart; h <= expectedEnd; h++ {
					epochNum := CalculateEpochNumber(h, tt.daStartHeight, tt.daEpochSize)
					assert.Equal(t, epoch, epochNum, "height %d should be in epoch %d", h, epoch)

					start, end, _ := CalculateEpochBoundaries(h, tt.daStartHeight, tt.daEpochSize)
					assert.Equal(t, expectedStart, start, "height %d should have start %d", h, expectedStart)
					assert.Equal(t, expectedEnd, end, "height %d should have end %d", h, expectedEnd)
				}
			}
		})
	}
}

func TestEpochBoundaryTransitions(t *testing.T) {
	daStartHeight := uint64(100)
	daEpochSize := uint64(10)

	// Test that epoch boundaries are correctly calculated at transitions
	transitions := []struct {
		height        uint64
		expectedEpoch uint64
		expectedStart uint64
		expectedEnd   uint64
	}{
		{100, 1, 100, 109}, // First height of epoch 1
		{109, 1, 100, 109}, // Last height of epoch 1
		{110, 2, 110, 119}, // First height of epoch 2
		{119, 2, 110, 119}, // Last height of epoch 2
		{120, 3, 120, 129}, // First height of epoch 3
	}

	for _, tr := range transitions {
		epoch := CalculateEpochNumber(tr.height, daStartHeight, daEpochSize)
		assert.Equal(t, tr.expectedEpoch, epoch, "height %d epoch mismatch", tr.height)

		start, end, _ := CalculateEpochBoundaries(tr.height, daStartHeight, daEpochSize)
		assert.Equal(t, tr.expectedStart, start, "height %d start mismatch", tr.height)
		assert.Equal(t, tr.expectedEnd, end, "height %d end mismatch", tr.height)
	}
}
