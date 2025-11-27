package celestia

import (
	"encoding/json"
	"testing"

	"github.com/celestiaorg/nmt"
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
	proof := Proof{
		&nmt.Proof{},
	}

	// Marshal
	data, err := json.Marshal(proof)
	require.NoError(t, err)

	// Unmarshal
	var decoded Proof
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, len(proof), len(decoded))
}

func TestSubmitOptionsJSON(t *testing.T) {
	opts := &SubmitOptions{
		GasPrice:          0.002,
		IsGasPriceSet:     true,
		Gas:               100000,
		SignerAddress:     "celestia1abc123",
		FeeGranterAddress: "celestia1feegranter",
	}

	// Marshal
	data, err := json.Marshal(opts)
	require.NoError(t, err)

	// Unmarshal
	var decoded SubmitOptions
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, opts.GasPrice, decoded.GasPrice)
	assert.Equal(t, opts.IsGasPriceSet, decoded.IsGasPriceSet)
	assert.Equal(t, opts.Gas, decoded.Gas)
	assert.Equal(t, opts.SignerAddress, decoded.SignerAddress)
	assert.Equal(t, opts.FeeGranterAddress, decoded.FeeGranterAddress)
}
