package jsonrpc

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/evstack/ev-node/core/da"
)

// TestSubmitWithOptions_SizeValidation tests the corrected behavior of SubmitWithOptions
// where it validates the entire batch before submission and returns ErrBlobSizeOverLimit
// if the batch is too large, instead of silently dropping blobs.
func TestSubmitWithOptions_SizeValidation(t *testing.T) {
	logger := zerolog.Nop()

	testCases := []struct {
		name          string
		maxBlobSize   uint64
		inputBlobs    []da.Blob
		expectError   bool
		expectedError error
		description   string
	}{
		{
			name:        "Empty input",
			maxBlobSize: 1000,
			inputBlobs:  []da.Blob{},
			expectError: false,
			description: "Empty input should return empty result without error",
		},
		{
			name:        "Single blob within limit",
			maxBlobSize: 1000,
			inputBlobs:  []da.Blob{make([]byte, 500)},
			expectError: false,
			description: "Single blob smaller than limit should succeed",
		},
		{
			name:          "Single blob exceeds limit",
			maxBlobSize:   1000,
			inputBlobs:    []da.Blob{make([]byte, 1500)},
			expectError:   true,
			expectedError: da.ErrBlobSizeOverLimit,
			description:   "Single blob larger than limit should fail",
		},
		{
			name:        "Multiple blobs within limit",
			maxBlobSize: 1000,
			inputBlobs:  []da.Blob{make([]byte, 300), make([]byte, 400), make([]byte, 200)},
			expectError: false,
			description: "Multiple blobs totaling less than limit should succeed",
		},
		{
			name:          "Multiple blobs exceed total limit",
			maxBlobSize:   1000,
			inputBlobs:    []da.Blob{make([]byte, 400), make([]byte, 400), make([]byte, 400)},
			expectError:   true,
			expectedError: da.ErrBlobSizeOverLimit,
			description:   "Multiple blobs totaling more than limit should fail completely",
		},
		{
			name:          "Mixed: some blobs fit, total exceeds limit",
			maxBlobSize:   1000,
			inputBlobs:    []da.Blob{make([]byte, 100), make([]byte, 200), make([]byte, 800)},
			expectError:   true,
			expectedError: da.ErrBlobSizeOverLimit,
			description:   "Should fail completely, not partially submit blobs that fit",
		},
		{
			name:          "One blob exceeds limit individually",
			maxBlobSize:   1000,
			inputBlobs:    []da.Blob{make([]byte, 300), make([]byte, 1500), make([]byte, 200)},
			expectError:   true,
			expectedError: da.ErrBlobSizeOverLimit,
			description:   "Should fail if any individual blob exceeds limit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create API with test configuration
			api := &API{
				Logger:      logger,
				MaxBlobSize: tc.maxBlobSize,
			}

			// Mock the Internal.SubmitWithOptions to always succeed if called
			// This tests that our validation logic works before reaching the actual RPC call
			mockCalled := false
			api.Internal.SubmitWithOptions = func(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
				mockCalled = true
				// Return mock IDs for successful submissions
				ids := make([]da.ID, len(blobs))
				for i := range blobs {
					ids[i] = []byte{byte(i)}
				}
				return ids, nil
			}

			// Call SubmitWithOptions
			ctx := context.Background()
			result, err := api.SubmitWithOptions(ctx, tc.inputBlobs, 1.0, []byte("test"), nil)

			// Verify expectations
			if tc.expectError {
				assert.Error(t, err, tc.description)
				if tc.expectedError != nil {
					assert.ErrorIs(t, err, tc.expectedError, tc.description)
				}
				assert.Nil(t, result, "Result should be nil on error")
				assert.False(t, mockCalled, "Internal RPC should not be called when validation fails")
			} else {
				assert.NoError(t, err, tc.description)
				assert.NotNil(t, result, "Result should not be nil on success")
				if len(tc.inputBlobs) > 0 {
					assert.True(t, mockCalled, "Internal RPC should be called for valid submissions")
					assert.Len(t, result, len(tc.inputBlobs), "Should return IDs for all submitted blobs")
				}
			}
		})
	}
}
