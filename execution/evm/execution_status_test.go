package evm

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePayloadStatus tests the payload status validation logic
func TestValidatePayloadStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		status      engine.PayloadStatusV1
		expectError bool
		errorType   error
	}{
		{
			name: "valid status",
			status: engine.PayloadStatusV1{
				Status:          engine.VALID,
				LatestValidHash: &common.Hash{},
			},
			expectError: false,
		},
		{
			name: "invalid status",
			status: engine.PayloadStatusV1{
				Status:          engine.INVALID,
				LatestValidHash: &common.Hash{},
			},
			expectError: true,
			errorType:   ErrInvalidPayloadStatus,
		},
		{
			name: "syncing status should retry",
			status: engine.PayloadStatusV1{
				Status:          engine.SYNCING,
				LatestValidHash: &common.Hash{},
			},
			expectError: true,
			errorType:   ErrPayloadSyncing,
		},
		{
			name: "accepted status treated as syncing",
			status: engine.PayloadStatusV1{
				Status:          engine.ACCEPTED,
				LatestValidHash: &common.Hash{},
			},
			expectError: true,
			errorType:   ErrPayloadSyncing,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validatePayloadStatus(tt.status)

			if tt.expectError {
				require.Error(t, err, "expected error but got nil")
				assert.ErrorIs(t, err, tt.errorType, "expected error type %v, got %v", tt.errorType, err)
			} else {
				require.NoError(t, err, "expected no error but got %v", err)
			}
		})
	}
}

// TestRetryWithBackoffOnPayloadStatus tests the retry logic with exponential backoff
func TestRetryWithBackoffOnPayloadStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		maxRetries     int
		shouldSucceed  bool
		failuresCount  int
		syncingCount   int
		expectAttempts int
	}{
		{
			name:           "succeeds on first attempt",
			maxRetries:     3,
			shouldSucceed:  true,
			failuresCount:  0,
			syncingCount:   0,
			expectAttempts: 1,
		},
		{
			name:           "succeeds after syncing",
			maxRetries:     3,
			shouldSucceed:  true,
			failuresCount:  0,
			syncingCount:   2,
			expectAttempts: 3,
		},
		{
			name:           "fails after max retries",
			maxRetries:     3,
			shouldSucceed:  false,
			failuresCount:  0,
			syncingCount:   5,
			expectAttempts: 3,
		},
		{
			name:           "immediate failure on invalid",
			maxRetries:     3,
			shouldSucceed:  false,
			failuresCount:  1,
			syncingCount:   0,
			expectAttempts: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			attempts := 0
			syncingAttempts := 0
			failureAttempts := 0

			retryFn := func() error {
				attempts++

				// Return INVALID status for configured failures
				if failureAttempts < tt.failuresCount {
					failureAttempts++
					return ErrInvalidPayloadStatus
				}

				// Return SYNCING status for configured syncing attempts
				if syncingAttempts < tt.syncingCount {
					syncingAttempts++
					return ErrPayloadSyncing
				}

				// Success
				if tt.shouldSucceed {
					return nil
				}

				// Keep returning syncing to exhaust retries
				return ErrPayloadSyncing
			}

			ctx := context.Background()
			err := retryWithBackoffOnPayloadStatus(ctx, retryFn, tt.maxRetries, 1*time.Millisecond, "test_operation")

			if tt.shouldSucceed {
				require.NoError(t, err, "expected success but got error")
			} else {
				require.Error(t, err, "expected error but got nil")
			}

			assert.Equal(t, tt.expectAttempts, attempts, "expected %d attempts, got %d", tt.expectAttempts, attempts)
		})
	}
}

// TestRetryWithBackoffOnPayloadStatus_ContextCancellation tests that retry respects context cancellation
func TestRetryWithBackoffOnPayloadStatus_ContextCancellation(t *testing.T) {
	t.Parallel()

	attempts := 0
	retryFn := func() error {
		attempts++
		return ErrPayloadSyncing
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := retryWithBackoffOnPayloadStatus(ctx, retryFn, 5, 100*time.Millisecond, "test_operation")

	require.Error(t, err, "expected error due to context cancellation")
	assert.ErrorIs(t, err, context.Canceled, "expected context.Canceled error")
	// Should fail fast on context cancellation without retries
	assert.LessOrEqual(t, attempts, 1, "expected at most 1 attempt, got %d", attempts)
}

// TestRetryWithBackoffOnPayloadStatus_ContextTimeout tests that retry respects context timeout
func TestRetryWithBackoffOnPayloadStatus_ContextTimeout(t *testing.T) {
	t.Parallel()

	attempts := 0
	retryFn := func() error {
		attempts++
		time.Sleep(50 * time.Millisecond) // Simulate some work
		return ErrPayloadSyncing
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := retryWithBackoffOnPayloadStatus(ctx, retryFn, 10, 1*time.Second, "test_operation")

	require.Error(t, err, "expected error due to context timeout")
	assert.ErrorIs(t, err, context.DeadlineExceeded, "expected context.DeadlineExceeded error")
	// Should stop on timeout, not exhaust all retries
	assert.Less(t, attempts, 10, "expected fewer than 10 attempts due to timeout, got %d", attempts)
}

// TestRetryWithBackoffOnPayloadStatus_RPCErrors tests that RPC errors are not retried
func TestRetryWithBackoffOnPayloadStatus_RPCErrors(t *testing.T) {
	t.Parallel()

	rpcError := errors.New("RPC connection failed")
	attempts := 0
	retryFn := func() error {
		attempts++
		return rpcError
	}

	ctx := context.Background()
	err := retryWithBackoffOnPayloadStatus(ctx, retryFn, 5, 1*time.Millisecond, "test_operation")

	require.Error(t, err, "expected error from RPC failure")
	assert.Equal(t, rpcError, err, "expected original RPC error to be returned")
	// Should fail immediately without retries on non-syncing errors
	assert.Equal(t, 1, attempts, "expected exactly 1 attempt, got %d", attempts)
}

// TestRetryWithBackoffOnPayloadStatus_WrappedRPCErrors tests that wrapped RPC errors are not retried
func TestRetryWithBackoffOnPayloadStatus_WrappedRPCErrors(t *testing.T) {
	t.Parallel()

	rpcError := errors.New("connection refused")
	attempts := 0
	retryFn := func() error {
		attempts++
		return fmt.Errorf("forkchoice update failed: %w", rpcError)
	}

	ctx := context.Background()
	err := retryWithBackoffOnPayloadStatus(ctx, retryFn, 5, 1*time.Millisecond, "test_operation")

	require.Error(t, err, "expected error from RPC failure")
	// Should fail immediately without retries on non-syncing errors
	assert.Equal(t, 1, attempts, "expected exactly 1 attempt, got %d", attempts)
}
