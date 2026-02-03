package p2p

import (
	"testing"

	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/stretchr/testify/require"
)

func TestAllowlistGaterOverridesPeerBlock(t *testing.T) {
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	base, err := conngater.NewBasicConnectionGater(ds)
	require.NoError(t, err)

	allowedID, err := peer.Decode("12D3KooWBY6qMG2KAcDdgH7ED2Pg4UzhuwnSy8CSZdCKDG911iyP")
	require.NoError(t, err)
	blockedID, err := peer.Decode("12D3KooWM1NFkZozoatQi3JvFE57eBaX56mNgBA68Lk5MTPxBE4U")
	require.NoError(t, err)

	require.NoError(t, base.BlockPeer(allowedID))
	require.NoError(t, base.BlockPeer(blockedID))

	gater := newAllowlistGater(base, map[peer.ID]struct{}{allowedID: {}})
	require.True(t, gater.InterceptPeerDial(allowedID))
	require.False(t, gater.InterceptPeerDial(blockedID))
	require.True(t, gater.InterceptSecured(network.DirInbound, allowedID, nil))
	require.False(t, gater.InterceptSecured(network.DirInbound, blockedID, nil))
}
