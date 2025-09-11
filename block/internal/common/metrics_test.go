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

		// Test channel metrics initialization
		channelNames := []string{"height_in", "header_store", "data_store", "retrieve", "da_includer", "tx_notify"}
		assert.Len(t, em.ChannelBufferUsage, len(channelNames))
		for _, name := range channelNames {
			assert.NotNil(t, em.ChannelBufferUsage[name])
		}
		assert.NotNil(t, em.DroppedSignals)

		// Test error metrics initialization
		errorTypes := []string{"block_production", "da_submission", "sync", "validation", "state_update"}
		assert.Len(t, em.ErrorsByType, len(errorTypes))
		for _, errType := range errorTypes {
			assert.NotNil(t, em.ErrorsByType[errType])
		}
		assert.NotNil(t, em.RecoverableErrors)
		assert.NotNil(t, em.NonRecoverableErrors)

		// Test performance metrics initialization
		operations := []string{"block_production", "da_submission", "block_retrieval", "block_validation", "state_update"}
		assert.Len(t, em.OperationDuration, len(operations))
		for _, op := range operations {
			assert.NotNil(t, em.OperationDuration[op])
		}
		assert.NotNil(t, em.GoroutineCount)

		// Test DA metrics initialization
		assert.NotNil(t, em.DASubmissionAttempts)
		assert.NotNil(t, em.DASubmissionSuccesses)
		assert.NotNil(t, em.DASubmissionFailures)
		assert.NotNil(t, em.DARetrievalAttempts)
		assert.NotNil(t, em.DARetrievalSuccesses)
		assert.NotNil(t, em.DARetrievalFailures)
		assert.NotNil(t, em.DAInclusionHeight)
		assert.NotNil(t, em.PendingHeadersCount)
		assert.NotNil(t, em.PendingDataCount)

		// Test sync metrics initialization
		assert.NotNil(t, em.SyncLag)
		assert.NotNil(t, em.HeadersSynced)
		assert.NotNil(t, em.DataSynced)
		assert.NotNil(t, em.BlocksApplied)
		assert.NotNil(t, em.InvalidHeadersCount)

		// Test block production metrics initialization
		assert.NotNil(t, em.BlockProductionTime)
		assert.NotNil(t, em.EmptyBlocksProduced)
		assert.NotNil(t, em.LazyBlocksProduced)
		assert.NotNil(t, em.NormalBlocksProduced)
		assert.NotNil(t, em.TxsPerBlock)

		// Test state transition metrics initialization
		transitions := []string{"pending_to_submitted", "submitted_to_included", "included_to_finalized"}
		assert.Len(t, em.StateTransitions, len(transitions))
		for _, transition := range transitions {
			assert.NotNil(t, em.StateTransitions[transition])
		}
		assert.NotNil(t, em.InvalidTransitions)
	})

	t.Run("NopMetrics", func(t *testing.T) {
		em := NopMetrics()

		// Test that all base metrics are initialized with no-op implementations
		assert.NotNil(t, em.Height)
		assert.NotNil(t, em.NumTxs)
		assert.NotNil(t, em.BlockSizeBytes)
		assert.NotNil(t, em.TotalTxs)
		assert.NotNil(t, em.CommittedHeight)

		// Test extended metrics
		assert.NotNil(t, em.DroppedSignals)
		assert.NotNil(t, em.RecoverableErrors)
		assert.NotNil(t, em.NonRecoverableErrors)
		assert.NotNil(t, em.GoroutineCount)

		// Test maps are initialized
		channelNames := []string{"height_in", "header_store", "data_store", "retrieve", "da_includer", "tx_notify"}
		assert.Len(t, em.ChannelBufferUsage, len(channelNames))
		for _, name := range channelNames {
			assert.NotNil(t, em.ChannelBufferUsage[name])
		}

		errorTypes := []string{"block_production", "da_submission", "sync", "validation", "state_update"}
		assert.Len(t, em.ErrorsByType, len(errorTypes))
		for _, errType := range errorTypes {
			assert.NotNil(t, em.ErrorsByType[errType])
		}

		operations := []string{"block_production", "da_submission", "block_retrieval", "block_validation", "state_update"}
		assert.Len(t, em.OperationDuration, len(operations))
		for _, op := range operations {
			assert.NotNil(t, em.OperationDuration[op])
		}

		transitions := []string{"pending_to_submitted", "submitted_to_included", "included_to_finalized"}
		assert.Len(t, em.StateTransitions, len(transitions))
		for _, transition := range transitions {
			assert.NotNil(t, em.StateTransitions[transition])
		}

		// Test DA metrics
		assert.NotNil(t, em.DASubmissionAttempts)
		assert.NotNil(t, em.DASubmissionSuccesses)
		assert.NotNil(t, em.DASubmissionFailures)
		assert.NotNil(t, em.DARetrievalAttempts)
		assert.NotNil(t, em.DARetrievalSuccesses)
		assert.NotNil(t, em.DARetrievalFailures)
		assert.NotNil(t, em.DAInclusionHeight)
		assert.NotNil(t, em.PendingHeadersCount)
		assert.NotNil(t, em.PendingDataCount)

		// Test sync metrics
		assert.NotNil(t, em.SyncLag)
		assert.NotNil(t, em.HeadersSynced)
		assert.NotNil(t, em.DataSynced)
		assert.NotNil(t, em.BlocksApplied)
		assert.NotNil(t, em.InvalidHeadersCount)

		// Test block production metrics
		assert.NotNil(t, em.BlockProductionTime)
		assert.NotNil(t, em.EmptyBlocksProduced)
		assert.NotNil(t, em.LazyBlocksProduced)
		assert.NotNil(t, em.NormalBlocksProduced)
		assert.NotNil(t, em.TxsPerBlock)
		assert.NotNil(t, em.InvalidTransitions)

		// Verify no-op metrics don't panic when used
		em.Height.Set(100)
		em.DroppedSignals.Add(1)
		em.RecoverableErrors.Add(1)
		em.GoroutineCount.Set(100)
		em.BlockProductionTime.Observe(0.5)
		em.ChannelBufferUsage["height_in"].Set(5)
		em.ErrorsByType["block_production"].Add(1)
		em.OperationDuration["block_production"].Observe(0.1)
		em.StateTransitions["pending_to_submitted"].Add(1)
	})
}

func TestMetricsWithLabels(t *testing.T) {
	t.Run("PrometheusMetricsWithLabels", func(t *testing.T) {
		// Test with multiple labels
		em := PrometheusMetrics("test_with_labels", "chain_id", "test_chain", "node_id", "test_node")

		// All metrics should be properly initialized
		assert.NotNil(t, em.Height)
		assert.NotNil(t, em.NumTxs)
		assert.Len(t, em.ChannelBufferUsage, 6)
		assert.Len(t, em.ErrorsByType, 5)
		assert.Len(t, em.OperationDuration, 5)
		assert.Len(t, em.StateTransitions, 3)
	})

	t.Run("PrometheusMetricsNoLabels", func(t *testing.T) {
		// Test with just namespace, no labels
		em := PrometheusMetrics("test_no_labels")

		// All metrics should still be properly initialized
		assert.NotNil(t, em.Height)
		assert.NotNil(t, em.NumTxs)
		assert.Len(t, em.ChannelBufferUsage, 6)
		assert.Len(t, em.ErrorsByType, 5)
		assert.Len(t, em.OperationDuration, 5)
		assert.Len(t, em.StateTransitions, 3)
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

	// Test extended metric operations
	em.DroppedSignals.Add(1)
	em.RecoverableErrors.Add(1)
	em.NonRecoverableErrors.Add(1)
	em.GoroutineCount.Set(50)

	// Test channel metrics
	em.ChannelBufferUsage["height_in"].Set(5)
	em.ChannelBufferUsage["header_store"].Set(2)
	em.ChannelBufferUsage["data_store"].Set(3)

	// Test error metrics
	em.ErrorsByType["block_production"].Add(1)
	em.ErrorsByType["da_submission"].Add(2)
	em.ErrorsByType["sync"].Add(0)
	em.ErrorsByType["validation"].Add(1)
	em.ErrorsByType["state_update"].Add(0)

	// Test operation duration
	em.OperationDuration["block_production"].Observe(0.05)
	em.OperationDuration["da_submission"].Observe(0.1)
	em.OperationDuration["block_retrieval"].Observe(0.02)
	em.OperationDuration["block_validation"].Observe(0.03)
	em.OperationDuration["state_update"].Observe(0.01)

	// Test DA metrics
	em.DASubmissionAttempts.Add(5)
	em.DASubmissionSuccesses.Add(4)
	em.DASubmissionFailures.Add(1)
	em.DARetrievalAttempts.Add(10)
	em.DARetrievalSuccesses.Add(9)
	em.DARetrievalFailures.Add(1)
	em.DAInclusionHeight.Set(995)
	em.PendingHeadersCount.Set(3)
	em.PendingDataCount.Set(2)

	// Test sync metrics
	em.SyncLag.Set(5)
	em.HeadersSynced.Add(10)
	em.DataSynced.Add(10)
	em.BlocksApplied.Add(10)
	em.InvalidHeadersCount.Add(0)

	// Test block production metrics
	em.BlockProductionTime.Observe(0.02)
	em.TxsPerBlock.Observe(100)
	em.EmptyBlocksProduced.Add(2)
	em.LazyBlocksProduced.Add(5)
	em.NormalBlocksProduced.Add(10)

	// Test state transitions
	em.StateTransitions["pending_to_submitted"].Add(15)
	em.StateTransitions["submitted_to_included"].Add(14)
	em.StateTransitions["included_to_finalized"].Add(13)
	em.InvalidTransitions.Add(0)
}

func TestMetricsSubsystem(t *testing.T) {
	// Test that the metrics subsystem constant is properly defined
	assert.Equal(t, "sequencer", MetricsSubsystem)
}
