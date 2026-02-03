package p2p

import (
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	ma "github.com/multiformats/go-multiaddr"
)

// allowlistGater wraps a BasicConnectionGater and permits specific peers even if blocked.
type allowlistGater struct {
	allowlist map[peer.ID]struct{}
	base      *conngater.BasicConnectionGater
}

func newAllowlistGater(base *conngater.BasicConnectionGater, allowlist map[peer.ID]struct{}) *allowlistGater {
	return &allowlistGater{
		allowlist: allowlist,
		base:      base,
	}
}

func (g *allowlistGater) isAllowed(id peer.ID) bool {
	_, ok := g.allowlist[id]
	return ok
}

func (g *allowlistGater) InterceptPeerDial(p peer.ID) (allow bool) {
	if g.isAllowed(p) {
		return true
	}
	return g.base.InterceptPeerDial(p)
}

func (g *allowlistGater) InterceptAddrDial(p peer.ID, a ma.Multiaddr) (allow bool) {
	return g.base.InterceptAddrDial(p, a)
}

func (g *allowlistGater) InterceptAccept(cma network.ConnMultiaddrs) (allow bool) {
	return g.base.InterceptAccept(cma)
}

func (g *allowlistGater) InterceptSecured(dir network.Direction, p peer.ID, cma network.ConnMultiaddrs) (allow bool) {
	if g.isAllowed(p) {
		return true
	}
	return g.base.InterceptSecured(dir, p, cma)
}

func (g *allowlistGater) InterceptUpgraded(conn network.Conn) (allow bool, reason control.DisconnectReason) {
	return g.base.InterceptUpgraded(conn)
}
