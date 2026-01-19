package telemetry

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/config"
)

func TestInitTracing_Disabled(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	cfg := &config.InstrumentationConfig{
		Tracing: false,
	}

	shutdown, err := InitTracing(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// shutdown should be a no-op
	err = shutdown(ctx)
	require.NoError(t, err)
}

func TestInitTracing_EnabledWithoutEndpoint(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	cfg := &config.InstrumentationConfig{
		Tracing:         true,
		TracingEndpoint: "", // missing endpoint
	}

	shutdown, err := InitTracing(ctx, cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// should return no-op shutdown when disabled
	err = shutdown(ctx)
	require.NoError(t, err)
}
