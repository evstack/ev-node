package execution_test

import (
	"context"
	"slices"
	"testing"

	"github.com/evstack/ev-node/core/execution"
)

// TestWithForceIncludedMask_ContextRoundtrip verifies that the mask
// can be stored in and retrieved from context correctly
func TestWithForceIncludedMask_ContextRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mask []bool
	}{
		{
			name: "nil mask",
			mask: nil,
		},
		{
			name: "empty mask",
			mask: []bool{},
		},
		{
			name: "single element",
			mask: []bool{true},
		},
		{
			name: "multiple elements",
			mask: []bool{true, false, true, false, false},
		},
		{
			name: "all true",
			mask: []bool{true, true, true},
		},
		{
			name: "all false",
			mask: []bool{false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Add mask to context
			ctxWithMask := execution.WithForceIncludedMask(ctx, tt.mask)

			// Retrieve mask from context
			retrieved := execution.GetForceIncludedMask(ctxWithMask)

			// Verify it matches
			if !slices.Equal(tt.mask, retrieved) {
				t.Errorf("expected mask %v, got %v", tt.mask, retrieved)
			}
		})
	}
}

// TestGetForceIncludedMask_NoMask verifies that getting a mask from
// a context without one returns nil
func TestGetForceIncludedMask_NoMask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mask := execution.GetForceIncludedMask(ctx)
	if mask != nil {
		t.Errorf("expected nil mask from context without mask, got %v", mask)
	}
}

// TestGetForceIncludedMask_WrongType verifies that wrong types in context
// are handled gracefully
func TestGetForceIncludedMask_WrongType(t *testing.T) {
	t.Parallel()

	// Create context with wrong type
	ctx := context.WithValue(context.Background(), "force_included_mask", "wrong type")

	mask := execution.GetForceIncludedMask(ctx)
	if mask != nil {
		t.Errorf("expected nil mask when context has wrong type, got %v", mask)
	}
}
