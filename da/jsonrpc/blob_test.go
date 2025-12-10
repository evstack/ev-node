package jsonrpc

import (
	"encoding/json"
	"testing"

	libshare "github.com/celestiaorg/go-square/v3/share"
	"github.com/stretchr/testify/require"
)

func TestMakeAndSplitID(t *testing.T) {
	id := MakeID(42, []byte{0x01, 0x02, 0x03})
	height, com := SplitID(id)
	require.Equal(t, uint64(42), height)
	require.Equal(t, []byte{0x01, 0x02, 0x03}, []byte(com))
}

func TestBlobJSONRoundTrip(t *testing.T) {
	ns := libshare.MustNewV0Namespace([]byte("test-ids"))

	blob, err := NewBlobV0(ns, []byte("hello"))
	require.NoError(t, err)
	require.NotEmpty(t, blob.Commitment)

	encoded, err := json.Marshal(blob)
	require.NoError(t, err)

	var decoded Blob
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	require.Equal(t, blob.Namespace().Bytes(), decoded.Namespace().Bytes())
	require.Equal(t, blob.Data(), decoded.Data())
	require.Equal(t, blob.Commitment, decoded.Commitment)
}
