package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignedHeaderBinaryCompatibility(t *testing.T) {
	signedHeader, _, err := GetRandomSignedHeader("chain-id")
	require.NoError(t, err)
	bytes, err := signedHeader.MarshalBinary()
	require.NoError(t, err)

	var p2pHeader P2PSignedHeader
	err = p2pHeader.UnmarshalBinary(bytes)
	require.NoError(t, err)

	assert.Equal(t, signedHeader.Header, p2pHeader.Header)
	assert.Equal(t, signedHeader.Signature, p2pHeader.Signature)
	assert.Equal(t, signedHeader.Signer, p2pHeader.Signer)
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

	var p2pData P2PData
	err = p2pData.UnmarshalBinary(bytes)
	require.NoError(t, err)

	assert.Equal(t, data.Metadata, p2pData.Metadata)
	assert.Equal(t, data.Txs, p2pData.Txs)
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
