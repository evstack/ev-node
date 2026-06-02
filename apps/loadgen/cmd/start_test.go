package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateStartConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := validateStartConfig(startConfig{
			txPerDay:     1,
			interval:     time.Hour,
			burstTxCount: 1,
			burstPerDay:  0,
		})
		require.NoError(t, err)
	})

	t.Run("invalid interval", func(t *testing.T) {
		err := validateStartConfig(startConfig{interval: 0})
		require.ErrorContains(t, err, "interval must be > 0")
	})

	t.Run("invalid tx per day", func(t *testing.T) {
		err := validateStartConfig(startConfig{interval: time.Hour, txPerDay: -1})
		require.ErrorContains(t, err, "tx-per-day must be >= 0")
	})

	t.Run("invalid burst tx count", func(t *testing.T) {
		err := validateStartConfig(startConfig{interval: time.Hour, burstTxCount: -1})
		require.ErrorContains(t, err, "burst-tx-count must be >= 0")
	})

	t.Run("invalid burst per day", func(t *testing.T) {
		err := validateStartConfig(startConfig{interval: time.Hour, burstPerDay: -1})
		require.ErrorContains(t, err, "burst-per-day must be >= 0")
	})
}
