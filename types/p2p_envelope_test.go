package types

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PEnvelope_MarshalUnmarshal(t *testing.T) {
	// Create a P2PData envelope
	data := &Data{
		Metadata: &Metadata{
			ChainID:      "test-chain",
			Height:       10,
			Time:         uint64(time.Now().UnixNano()),
			LastDataHash: bytes.Repeat([]byte{0x1}, 32),
		},
		Txs: Txs{[]byte{0x1}, []byte{0x2}},
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
	assert.Equal(t, envelope.Message.LastDataHash, newEnvelope.Message.LastDataHash)
	assert.Equal(t, envelope.Message.Txs, newEnvelope.Message.Txs)
}

func TestP2PSignedHeader_MarshalUnmarshal(t *testing.T) {
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
			LastHeaderHash:  GetRandomBytes(32),
			DataHash:        GetRandomBytes(32),
			AppHash:         GetRandomBytes(32),
			ProposerAddress: GetRandomBytes(32),
			ValidatorHash:   GetRandomBytes(32),
		},
		// Signature and Signer are transient
	}

	envelope := &P2PSignedHeader{
		Message:      header,
		DAHeightHint: 200,
	}

	// Marshaling
	bz, err := envelope.MarshalBinary()
	require.NoError(t, err)
	assert.NotEmpty(t, bz)

	// Unmarshaling
	newEnvelope := (&P2PSignedHeader{}).New()
	err = newEnvelope.UnmarshalBinary(bz)
	require.NoError(t, err)
	assert.Equal(t, envelope.DAHeightHint, newEnvelope.DAHeightHint)
	assert.Equal(t, envelope, newEnvelope)
}

func TestSignedHeaderBinaryCompatibility(t *testing.T) {
	signedHeader, _, err := GetRandomSignedHeader("chain-id")
	require.NoError(t, err)
	bytes, err := signedHeader.MarshalBinary()
	require.NoError(t, err)

	p2pHeader := (&P2PSignedHeader{}).New()
	err = p2pHeader.UnmarshalBinary(bytes)
	require.NoError(t, err)

	assert.Equal(t, signedHeader.Header, p2pHeader.Message.Header)
	assert.Equal(t, signedHeader.Signature, p2pHeader.Message.Signature)
	assert.Equal(t, signedHeader.Signer, p2pHeader.Message.Signer)
	assert.Zero(t, p2pHeader.DAHeightHint)

	p2pHeader.DAHeightHint = 100
	p2pBytes, err := p2pHeader.MarshalBinary()
	require.NoError(t, err)

	var decodedSignedHeader SignedHeader
	err = decodedSignedHeader.UnmarshalBinary(p2pBytes)
	require.NoError(t, err)
	assert.Equal(t, signedHeader.Header, decodedSignedHeader.Header)
	assert.Equal(t, signedHeader.Signature, decodedSignedHeader.Signature)
	assert.Equal(t, signedHeader.Signer, decodedSignedHeader.Signer)
}

func TestDataBinaryCompatibility(t *testing.T) {
	data := &Data{
		Metadata: &Metadata{
			ChainID:      "chain-id",
			Height:       10,
			Time:         uint64(time.Now().UnixNano()),
			LastDataHash: []byte("last-hash"),
		},
		Txs: Txs{
			[]byte("tx1"),
			[]byte("tx2"),
		},
	}
	bytes, err := data.MarshalBinary()
	require.NoError(t, err)

	p2pData := (&P2PData{}).New()
	err = p2pData.UnmarshalBinary(bytes)
	require.NoError(t, err)

	assert.Equal(t, data.Metadata, p2pData.Message.Metadata)
	assert.Equal(t, data.Txs, p2pData.Message.Txs)
	assert.Zero(t, p2pData.DAHeightHint)

	p2pData.DAHeightHint = 200

	p2pBytes, err := p2pData.MarshalBinary()
	require.NoError(t, err)

	var decodedData Data
	err = decodedData.UnmarshalBinary(p2pBytes)
	require.NoError(t, err)
	assert.Equal(t, data.Metadata, decodedData.Metadata)
	assert.Equal(t, data.Txs, decodedData.Txs)
}
