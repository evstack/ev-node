package types

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PEnvelope_MarshalUnmarshal(t *testing.T) {
	// Create a P2PData envelope
	data := &Data{
		Metadata: &Metadata{
			ChainID: "test-chain",
			Height:  10,
			Time:    uint64(time.Now().UnixNano()),
		},
		Txs: nil,
	}
	envelope := &P2PData{
		Message:      data,
		DAHeightHint: 100,
	}

	// Marshaling
	bytes, err := envelope.MarshalBinary()
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	// Unmarshaling
	newEnvelope := (&P2PData{}).New()
	err = newEnvelope.UnmarshalBinary(bytes)
	require.NoError(t, err)
	assert.Equal(t, envelope.DAHeightHint, newEnvelope.DAHeightHint)
	assert.Equal(t, envelope.Message.Height(), newEnvelope.Message.Height())
	assert.Equal(t, envelope.Message.ChainID(), newEnvelope.Message.ChainID())
}

func TestP2PSignedHeader_MarshalUnmarshal(t *testing.T) {
	// Create a SignedHeader
	// Minimal valid SignedHeader
	header := &SignedHeader{
		Header: Header{
			BaseHeader: BaseHeader{
				ChainID: "test-chain",
				Height:  5,
				Time:    uint64(time.Now().UnixNano()),
			},
			Version: Version{
				Block: 1,
				App:   2,
			},
			DataHash: make([]byte, 32),
		},
		Signature: make([]byte, 64),
		Signer: Signer{
			// PubKey can be nil for basic marshal check
			Address: make([]byte, 20),
		},
	}
	_, _ = rand.Read(header.DataHash)
	_, _ = rand.Read(header.Signature)
	_, _ = rand.Read(header.Signer.Address)

	envelope := &P2PSignedHeader{
		Message:      header,
		DAHeightHint: 200,
	}

	// Marshaling
	bytes, err := envelope.MarshalBinary()
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	// Unmarshaling
	newEnvelope := (&P2PSignedHeader{}).New()
	err = newEnvelope.UnmarshalBinary(bytes)
	require.NoError(t, err)
	assert.Equal(t, envelope.DAHeightHint, newEnvelope.DAHeightHint)
	assert.Equal(t, envelope.Message.Height(), newEnvelope.Message.Height())
	assert.Equal(t, envelope.Message.ChainID(), newEnvelope.Message.ChainID())
	// Deep comparison of structs if needed
}
