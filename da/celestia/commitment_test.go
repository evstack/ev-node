package celestia

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCommitment(t *testing.T) {
	// Create a valid 29-byte namespace (version 0 + 28 bytes ID)
	namespace := make([]byte, 29)
	namespace[0] = 0 // version 0

	tests := []struct {
		name        string
		data        []byte
		namespace   []byte
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid small blob",
			data:      []byte("hello world"),
			namespace: namespace,
			wantErr:   false,
		},
		{
			name:      "valid larger blob",
			data:      make([]byte, 1024),
			namespace: namespace,
			wantErr:   false,
		},
		{
			name:        "empty blob not allowed",
			data:        []byte{},
			namespace:   namespace,
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "invalid namespace too short",
			data:        []byte("test"),
			namespace:   make([]byte, 10),
			wantErr:     true,
			errContains: "namespace",
		},
		{
			name:        "invalid namespace too long",
			data:        []byte("test"),
			namespace:   make([]byte, 30),
			wantErr:     true,
			errContains: "namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitment, err := CreateCommitment(tt.data, tt.namespace)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, commitment)
			} else {
				require.NoError(t, err)
				require.NotNil(t, commitment)
				// Commitment should be non-empty
				assert.Greater(t, len(commitment), 0)
			}
		})
	}
}

func TestCreateCommitment_Deterministic(t *testing.T) {
	// Create a valid namespace
	namespace := make([]byte, 29)
	namespace[0] = 0

	data := []byte("test data for deterministic commitment")

	// Create commitment twice
	commitment1, err := CreateCommitment(data, namespace)
	require.NoError(t, err)

	commitment2, err := CreateCommitment(data, namespace)
	require.NoError(t, err)

	// Should be identical
	assert.Equal(t, commitment1, commitment2)
}

func TestCreateCommitment_DifferentData(t *testing.T) {
	// Create a valid namespace
	namespace := make([]byte, 29)
	namespace[0] = 0

	data1 := []byte("data one")
	data2 := []byte("data two")

	commitment1, err := CreateCommitment(data1, namespace)
	require.NoError(t, err)

	commitment2, err := CreateCommitment(data2, namespace)
	require.NoError(t, err)

	// Should be different
	assert.NotEqual(t, commitment1, commitment2)
}

func TestCreateCommitment_DifferentNamespace(t *testing.T) {
	// Create two different valid namespaces
	namespace1 := make([]byte, 29)
	namespace1[0] = 0
	namespace1[28] = 1

	namespace2 := make([]byte, 29)
	namespace2[0] = 0
	namespace2[28] = 2

	data := []byte("same data")

	commitment1, err := CreateCommitment(data, namespace1)
	require.NoError(t, err)

	commitment2, err := CreateCommitment(data, namespace2)
	require.NoError(t, err)

	// Should be different due to different namespaces
	assert.NotEqual(t, commitment1, commitment2)
}

func TestCreateCommitments(t *testing.T) {
	// Create a valid namespace
	namespace := make([]byte, 29)
	namespace[0] = 0

	blobs := [][]byte{
		[]byte("blob one"),
		[]byte("blob two"),
		[]byte("blob three"),
	}

	commitments, err := CreateCommitments(blobs, namespace)
	require.NoError(t, err)
	require.Len(t, commitments, 3)

	// All commitments should be non-empty and different
	for i, c := range commitments {
		assert.Greater(t, len(c), 0, "commitment %d should not be empty", i)
	}

	assert.NotEqual(t, commitments[0], commitments[1])
	assert.NotEqual(t, commitments[1], commitments[2])
	assert.NotEqual(t, commitments[0], commitments[2])
}

func TestCreateCommitments_Empty(t *testing.T) {
	namespace := make([]byte, 29)

	commitments, err := CreateCommitments([][]byte{}, namespace)
	require.NoError(t, err)
	assert.Len(t, commitments, 0)
}
