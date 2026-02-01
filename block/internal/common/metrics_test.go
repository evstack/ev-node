package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	t.Run("PrometheusMetrics", func(t *testing.T) {
		em := PrometheusMetrics("test_prometheus", "chain_id", "test_chain")

		// Test that base metrics are initialized
		assert.NotNil(t, em.Height)
		assert.NotNil(t, em.NumTxs)
		assert.NotNil(t, em.BlockSizeBytes)
		assert.NotNil(t, em.TotalTxs)
		assert.NotNil(t, em.CommittedHeight)
		assert.NotNil(t, em.TxsPerBlock)

		// Test performance metrics initialization
		operations := []string{"block_production", "da_submission", "block_retrieval", "block_validation", "state_update"}
		assert.Len(t, em.OperationDuration, len(operations))
		for _, op := range operations {
			assert.NotNil(t, em.OperationDuration[op])
		}

		// Test DA metrics initialization
		assert.NotNil(t, em.DARetrievalAttempts)
		assert.NotNil(t, em.DARetrievalSuccesses)
		assert.NotNil(t, em.DARetrievalFailures)
		assert.NotNil(t, em.DAInclusionHeight)
	})

	t.Run("NopMetrics", func(t *testing.T) {
		em := NopMetrics()

		// Test that all base metrics are initialized with no-op implementations
		assert.NotNil(t, em.Height)
		assert.NotNil(t, em.NumTxs)
		assert.NotNil(t, em.BlockSizeBytes)
		assert.NotNil(t, em.TotalTxs)
		assert.NotNil(t, em.CommittedHeight)
		assert.NotNil(t, em.TxsPerBlock)

		operations := []string{"block_production", "da_submission", "block_retrieval", "block_validation", "state_update"}
		assert.Len(t, em.OperationDuration, len(operations))
		for _, op := range operations {
			assert.NotNil(t, em.OperationDuration[op])
		}

		// Test DA metrics
		assert.NotNil(t, em.DARetrievalAttempts)
		assert.NotNil(t, em.DARetrievalSuccesses)
		assert.NotNil(t, em.DARetrievalFailures)
		assert.NotNil(t, em.DAInclusionHeight)

		// Verify no-op metrics don't panic when used
		em.Height.Set(100)
		em.OperationDuration["block_production"].Observe(0.1)
	})
}

func TestMetricsWithLabels(t *testing.T) {
	t.Run("PrometheusMetricsWithLabels", func(t *testing.T) {
		// Test with multiple labels
		em := PrometheusMetrics("test_with_labels", "chain_id", "test_chain", "node_id", "test_node")

		// All metrics should be properly initialized
		assert.NotNil(t, em.Height)
		assert.NotNil(t, em.NumTxs)
		assert.Len(t, em.OperationDuration, 5)
	})

	t.Run("PrometheusMetricsNoLabels", func(t *testing.T) {
		// Test with just namespace, no labels
		em := PrometheusMetrics("test_no_labels")

		// All metrics should still be properly initialized
		assert.NotNil(t, em.Height)
		assert.NotNil(t, em.NumTxs)
		assert.Len(t, em.OperationDuration, 5)
	})
}

// Test integration with actual metrics recording
func TestMetricsIntegration(t *testing.T) {
	// This test verifies that metrics can be recorded without panics
	// when using Prometheus metrics
	em := PrometheusMetrics("test_integration")

	// Test base metrics operations
	em.Height.Set(1000)
	em.NumTxs.Set(50)
	em.BlockSizeBytes.Set(2048)
	em.TotalTxs.Set(50000)
	em.CommittedHeight.Set(999)
	em.TxsPerBlock.Observe(100)

	// Test operation duration
	em.OperationDuration["block_production"].Observe(0.05)
	em.OperationDuration["da_submission"].Observe(0.1)
	em.OperationDuration["block_retrieval"].Observe(0.02)
	em.OperationDuration["block_validation"].Observe(0.03)
	em.OperationDuration["state_update"].Observe(0.01)

	// Test DA metrics
	em.DARetrievalAttempts.Add(10)
	em.DARetrievalSuccesses.Add(9)
	em.DARetrievalFailures.Add(1)
	em.DAInclusionHeight.Set(995)
}

func TestMetricsSubsystem(t *testing.T) {
	// Test that the metrics subsystem constant is properly defined
	assert.Equal(t, "sequencer", MetricsSubsystem)
}
