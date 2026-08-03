package sync

import (
	"encoding/binary"
	"testing"

	p2ppb "github.com/celestiaorg/go-header/p2p/pb"
	"github.com/celestiaorg/go-libp2p-messenger/serde"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/pkg/blobsize"
	"github.com/evstack/ev-node/types"
)

func TestP2PMessageSizeSupportsMaxBlob(t *testing.T) {
	data := &types.P2PData{
		Data: &types.Data{
			Metadata: &types.Metadata{
				ChainID: "test-chain",
				Height:  1,
				Time:    1,
			},
			Txs: types.Txs{make([]byte, int(blobsize.DefaultMaxBlobSize))},
		},
	}

	body, err := data.MarshalBinary()
	require.NoError(t, err)

	response := &p2ppb.HeaderResponse{Body: body}
	buf := make([]byte, response.Size()+binary.MaxVarintLen64)
	_, err = serde.Marshal(response, buf)
	require.NoError(t, err)
}
