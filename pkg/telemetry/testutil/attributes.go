// Package testutil provides test utilities for OpenTelemetry tracing tests.
package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

// RequireAttribute asserts that an attribute with the given key exists and has the expected value.
func RequireAttribute(t *testing.T, attrs []attribute.KeyValue, key string, expected interface{}) {
	t.Helper()
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == key {
			found = true
			switch v := expected.(type) {
			case string:
				require.Equal(t, v, attr.Value.AsString())
			case int64:
				require.Equal(t, v, attr.Value.AsInt64())
			case int:
				require.Equal(t, int64(v), attr.Value.AsInt64())
			case bool:
				require.Equal(t, v, attr.Value.AsBool())
			default:
				t.Fatalf("unsupported attribute type: %T", expected)
			}
			break
		}
	}
	require.True(t, found, "attribute %s not found", key)
}
