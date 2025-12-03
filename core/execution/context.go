package execution

import "context"

type forceInclusionMaskContextKey struct{}

// WithForceIncludedMask adds the force-included mask to the context
// The mask indicates which transactions are force-included from DA (true) vs mempool (false)
func WithForceIncludedMask(ctx context.Context, mask []bool) context.Context {
	return context.WithValue(ctx, forceInclusionMaskContextKey{}, mask)
}

// GetForceIncludedMask retrieves the force-included mask from the context
// Returns nil if no mask is present in the context
func GetForceIncludedMask(ctx context.Context) []bool {
	if mask, ok := ctx.Value(forceInclusionMaskContextKey{}).([]bool); ok {
		return mask
	}
	return nil
}
