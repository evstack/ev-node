package celestia

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace Namespace
		wantErr   bool
	}{
		{
			name:      "valid namespace (29 bytes)",
			namespace: make([]byte, 29),
			wantErr:   false,
		},
		{
			name:      "invalid namespace too short",
			namespace: make([]byte, 10),
			wantErr:   true,
		},
		{
			name:      "invalid namespace too long",
			namespace: make([]byte, 30),
			wantErr:   true,
		},
		{
			name:      "invalid namespace empty",
			namespace: []byte{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNamespace(tt.namespace)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProofJSONMarshaling(t *testing.T) {
	proof := &Proof{
		Data: []byte{1, 2, 3, 4, 5},
	}

	// Marshal
	data, err := json.Marshal(proof)
	require.NoError(t, err)

	// Unmarshal
	var decoded Proof
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, proof.Data, decoded.Data)
}

func TestSubmitOptionsJSON(t *testing.T) {
	opts := &SubmitOptions{
		Fee:           0.002,
		GasLimit:      100000,
		SignerAddress: "celestia1abc123",
	}

	// Marshal
	data, err := json.Marshal(opts)
	require.NoError(t, err)

	// Unmarshal
	var decoded SubmitOptions
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, opts.Fee, decoded.Fee)
	assert.Equal(t, opts.GasLimit, decoded.GasLimit)
	assert.Equal(t, opts.SignerAddress, decoded.SignerAddress)
}
