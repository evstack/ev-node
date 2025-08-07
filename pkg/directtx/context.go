package directtx

import "context"

type contextKey struct{}

var fallbackModeKey = contextKey{}

func IsInFallbackMode(ctx context.Context) bool {
	h, ok := ctx.Value(fallbackModeKey).(bool)
	return ok && h
}
func WithFallbackMode(ctx context.Context) context.Context {
	return context.WithValue(ctx, fallbackModeKey, true)
}
