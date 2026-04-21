package p2p

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ipfs/go-datastore"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	cdiscovery "github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	discutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	rollhash "github.com/evstack/ev-node/pkg/hash"
)

// TODO(tzdybal): refactor to configuration parameters
const (
	// reAdvertisePeriod defines a period after which P2P client re-attempt advertising namespace in DHT.
	reAdvertisePeriod = 1 * time.Hour

	// peerLimit defines limit of number of peers returned during active peer discovery.
	peerLimit = 60

	// seedReconnectBackoff is the initial backoff when reconnecting to a disconnected seed peer.
	seedReconnectBackoff = 1 * time.Second

	// seedReconnectMaxBackoff is the maximum backoff for seed peer reconnection attempts.
	seedReconnectMaxBackoff = 30 * time.Second
)

// Client is a P2P client, implemented with libp2p.
//
// Initially, client connects to predefined seed nodes (aka bootnodes, bootstrap nodes).
// Those seed nodes serve Kademlia DHT protocol, and are agnostic to ORU chain. Using DHT
// peer routing and discovery clients find other peers within ORU network.
type Client struct {
	logger zerolog.Logger

	conf    config.P2PConfig
	chainID string
	privKey crypto.PrivKey

	host    host.Host
	dht     *dht.IpfsDHT
	disc    *discovery.RoutingDiscovery
	gater   *conngater.BasicConnectionGater
	ps      *pubsub.PubSub
	started bool

	ctx    context.Context
	cancel context.CancelFunc

	seedPeers []peer.AddrInfo

	metrics *Metrics
}

// NewClient creates new Client object.
//
// Basic checks on parameters are done, and default parameters are provided for unset-configuration
func NewClient(
	conf config.P2PConfig,
	privKey crypto.PrivKey,
	ds datastore.Datastore,
	chainID string,
	logger zerolog.Logger,
	metrics *Metrics,
) (*Client, error) {

	// When DisableConnectionGater is true (default) the gater is a no-op: it
	// uses an ephemeral in-memory store, is not registered with the libp2p host,
	// and never blocks any peer. The instance is kept only because go-header's
	// Exchange requires a *conngater.BasicConnectionGater parameter.
	// Set DisableConnectionGater=false to activate peer filtering (e.g. when
	// experiencing P2P flooding).
	var gaterDS datastore.Datastore
	if !conf.DisableConnectionGater {
		gaterDS = datastore.NewMapDatastore()
	}
	gater, err := conngater.NewBasicConnectionGater(gaterDS)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection gater: %w", err)
	}

	return &Client{
		conf:    conf,
		gater:   gater,
		privKey: privKey,
		chainID: chainID,
		logger:  logger,
		metrics: metrics,
	}, nil
}

func NewClientWithHost(
	conf config.P2PConfig,
	privKey crypto.PrivKey,
	ds datastore.Datastore,
	chainID string,
	logger zerolog.Logger,
	metrics *Metrics,
	h host.Host, // injected host (mocknet or custom)
) (*Client, error) {
	c, err := NewClient(conf, privKey, ds, chainID, logger, metrics)
	if err != nil {
		return nil, err
	}

	// Reject hosts whose identity does not match the supplied node key
	expectedID, _ := peer.IDFromPrivateKey(privKey)
	if h.ID() != expectedID {
		return nil, fmt.Errorf(
			"injected host ID %s does not match node key ID %s",
			h.ID(),
			expectedID,
		)
	}

	c.host = h
	return c, nil
}

// Start establish Client's P2P connectivity.
//
// Following steps are taken:
// 1. Setup libp2p host, start listening for incoming connections.
// 2. Setup gossibsub.
// 3. Setup DHT, establish connection to seed nodes and initialize peer discovery.
// 4. Use active peer discovery to look for peers from same ORU network.
func (c *Client) Start(ctx context.Context) error {
	if c.started {
		return nil
	}
	c.logger.Debug().Msg("starting P2P client")

	if c.host != nil {
		return c.startWithHost(ctx, c.host)
	}

	h, err := c.listen()
	if err != nil {
		return err
	}
	return c.startWithHost(ctx, h)
}

func (c *Client) startWithHost(ctx context.Context, h host.Host) error {
	c.host = h
	c.ctx, c.cancel = context.WithCancel(ctx)
	for _, a := range c.host.Addrs() {
		c.logger.Info().Str("address", fmt.Sprintf("%s/p2p/%s", a, c.host.ID())).Msg("listening on address")
	}

	if !c.conf.DisableConnectionGater {
		c.logger.Debug().Str("blacklist", c.conf.BlockedPeers).Msg("blocking blacklisted peers")
		if err := c.setupBlockedPeers(c.parseAddrInfoList(c.conf.BlockedPeers)); err != nil {
			return err
		}
		c.logger.Debug().Str("whitelist", c.conf.AllowedPeers).Msg("allowing whitelisted peers")
		if err := c.setupAllowedPeers(c.parseAddrInfoList(c.conf.AllowedPeers)); err != nil {
			return err
		}
	}

	c.logger.Debug().Msg("setting up gossiping")
	if err := c.setupGossiping(ctx); err != nil {
		return err
	}

	c.logger.Debug().Msg("setting up DHT")
	if err := c.setupDHT(ctx); err != nil {
		return err
	}

	c.logger.Debug().Msg("setting up active peer discovery")
	if err := c.peerDiscovery(ctx); err != nil {
		return err
	}

	c.started = true

	c.host.Network().Notify(c.newDisconnectNotifee())

	return nil
}

// Close gently stops Client.
func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	var err error
	if c.dht != nil {
		err = errors.Join(err, c.dht.Close())
	}
	if c.host != nil {
		err = errors.Join(err, c.host.Close())
	}
	c.started = false
	return err
}

// Addrs returns listen addresses of Client.
func (c *Client) Addrs() []multiaddr.Multiaddr {
	return c.host.Addrs()
}

// Host returns the libp2p node in a peer-to-peer network
func (c *Client) Host() host.Host {
	return c.host
}

// PubSub returns the libp2p node pubsub for adding future subscriptions
func (c *Client) PubSub() *pubsub.PubSub {
	return c.ps
}

// ConnectionGater returns the client's connection gater
func (c *Client) ConnectionGater() *conngater.BasicConnectionGater {
	return c.gater
}

// Info returns client ID, ListenAddr, and Network info
func (c *Client) Info() (string, string, string, error) {
	rawKey, err := c.privKey.GetPublic().Raw()
	if err != nil {
		return "", "", "", err
	}
	return hex.EncodeToString(rollhash.SumTruncated(rawKey)), c.conf.ListenAddress, c.chainID, nil
}

// PeerIDs returns list of peer IDs of connected peers excluding self and inactive
func (c *Client) PeerIDs() []peer.ID {
	peerIDs := make([]peer.ID, 0)
	for _, conn := range c.host.Network().Conns() {
		if conn.RemotePeer() != c.host.ID() {
			peerIDs = append(peerIDs, conn.RemotePeer())
		}
	}
	return peerIDs
}

// Peers returns list of peers connected to Client.
func (c *Client) Peers() []PeerConnection {
	conns := c.host.Network().Conns()
	res := make([]PeerConnection, 0, len(conns))
	for _, conn := range conns {
		pc := PeerConnection{
			NodeInfo: NodeInfo{
				ListenAddr: c.conf.ListenAddress,
				Network:    c.chainID,
				NodeID:     conn.RemotePeer().String(),
			},
			IsOutbound: conn.Stat().Direction == network.DirOutbound,
			RemoteIP:   conn.RemoteMultiaddr().String(),
		}
		res = append(res, pc)
	}
	return res
}

type disconnectNotifee struct {
	c *Client
}

func (n disconnectNotifee) Connected(_ network.Network, conn network.Conn) {
	p := conn.RemotePeer()
	for _, sp := range n.c.seedPeers {
		if sp.ID == p {
			n.c.logger.Info().Str("peer", p.String()).Msg("connected to seed peer")
			return
		}
	}
}
func (n disconnectNotifee) OpenedStream(_ network.Network, _ network.Stream)     {}
func (n disconnectNotifee) ClosedStream(_ network.Network, _ network.Stream)     {}
func (n disconnectNotifee) Listen(_ network.Network, _ multiaddr.Multiaddr)      {}
func (n disconnectNotifee) ListenClose(_ network.Network, _ multiaddr.Multiaddr) {}

func (n disconnectNotifee) Disconnected(_ network.Network, conn network.Conn) {
	p := conn.RemotePeer()
	for _, sp := range n.c.seedPeers {
		if sp.ID == p {
			n.c.logger.Warn().Str("peer", p.String()).Msg("disconnected from seed peer, scheduling reconnect")
			go n.c.reconnectSeedPeer(sp)
			return
		}
	}
}

func (c *Client) reconnectSeedPeer(sp peer.AddrInfo) {
	backoff := seedReconnectBackoff
	for {
		if c.ctx.Err() != nil {
			return
		}
		if c.isConnected(sp.ID) {
			return
		}

		err := c.host.Connect(c.ctx, sp)
		if err == nil {
			c.logger.Info().Str("peer", sp.ID.String()).Msg("reconnected to seed peer")
			return
		}
		if c.ctx.Err() != nil {
			return
		}

		c.logger.Debug().Str("peer", sp.ID.String()).Dur("backoff", backoff).Err(err).Msg("failed to reconnect to seed peer, retrying")
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > seedReconnectMaxBackoff {
			backoff = seedReconnectMaxBackoff
		}
	}
}

func (c *Client) newDisconnectNotifee() disconnectNotifee {
	return disconnectNotifee{c: c}
}

// isConnected returns true if there is an active connection to the given peer.
func (c *Client) isConnected(id peer.ID) bool {
	return c.host.Network().Connectedness(id) == network.Connected
}

func (c *Client) listen() (host.Host, error) {
	maddr, err := multiaddr.NewMultiaddr(c.conf.ListenAddress)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{libp2p.ListenAddrs(maddr), libp2p.Identity(c.privKey)}
	if !c.conf.DisableConnectionGater {
		opts = append(opts, libp2p.ConnectionGater(c.gater))
	}
	return libp2p.New(opts...)
}

func (c *Client) setupDHT(ctx context.Context) error {
	peers := c.parseAddrInfoList(c.conf.Peers)
	c.seedPeers = peers
	if len(peers) == 0 {
		c.logger.Info().Msg("no peers - only listening for connections")
	}

	for _, sa := range peers {
		c.logger.Debug().Str("addr", sa.String()).Msg("peer")
	}

	var err error
	c.dht, err = dht.New(ctx, c.host, dht.Mode(dht.ModeServer), dht.BootstrapPeers(peers...))
	if err != nil {
		return fmt.Errorf("failed to create DHT: %w", err)
	}

	err = c.dht.Bootstrap(ctx)
	if err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	c.host = routedhost.Wrap(c.host, c.dht)

	return nil
}

func (c *Client) peerDiscovery(ctx context.Context) error {
	err := c.setupPeerDiscovery(ctx)
	if err != nil {
		return err
	}

	err = c.advertise(ctx)
	if err != nil {
		return err
	}

	err = c.findPeers(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) setupPeerDiscovery(ctx context.Context) error {
	// wait for DHT
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.dht.RefreshRoutingTable():
	}
	c.disc = discovery.NewRoutingDiscovery(c.dht)
	return nil
}

func (c *Client) setupBlockedPeers(peers []peer.AddrInfo) error {
	for _, p := range peers {
		if err := c.gater.BlockPeer(p.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) setupAllowedPeers(peers []peer.AddrInfo) error {
	for _, p := range peers {
		if err := c.gater.UnblockPeer(p.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) advertise(ctx context.Context) error {
	discutil.Advertise(ctx, c.disc, c.getNamespace(), cdiscovery.TTL(reAdvertisePeriod))
	return nil
}

func (c *Client) findPeers(ctx context.Context) error {
	peerCh, err := c.disc.FindPeers(ctx, c.getNamespace(), cdiscovery.Limit(peerLimit))
	if err != nil {
		return err
	}

	for peer := range peerCh {
		go c.tryConnect(ctx, peer)
	}

	return nil
}

// tryConnect attempts to connect to a peer and logs error if necessary
func (c *Client) tryConnect(ctx context.Context, peer peer.AddrInfo) {
	if peer.ID == c.host.ID() {
		return
	}

	err := c.host.Connect(ctx, peer)
	if err != nil && ctx.Err() == nil {
		c.logger.Error().Str("peer", peer.String()).Err(err).Msg("failed to connect to peer")
	}
}

func (c *Client) setupGossiping(ctx context.Context) error {
	var err error
	c.ps, err = pubsub.NewFloodSub(ctx, c.host)
	if err != nil {
		return err
	}
	return nil
}

// parseAddrInfoList parses a comma-separated string of multiaddrs into a list of peer.AddrInfo structs
func (c *Client) parseAddrInfoList(addrInfoStr string) []peer.AddrInfo {
	if len(addrInfoStr) == 0 {
		return []peer.AddrInfo{}
	}
	peers := strings.Split(addrInfoStr, ",")
	addrs := make([]peer.AddrInfo, 0, len(peers))
	for _, p := range peers {
		maddr, err := multiaddr.NewMultiaddr(p)
		if err != nil {
			c.logger.Error().Str("address", p).Err(err).Msg("failed to parse peer")
			continue
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			c.logger.Error().Str("address", maddr.String()).Err(err).Msg("failed to create addr info for peer")
			continue
		}
		addrs = append(addrs, *addrInfo)
	}
	return addrs
}

// getNamespace returns unique string identifying ORU network.
//
// It is used to advertise/find peers in libp2p DHT.
// For now, chainID is used.
func (c *Client) getNamespace() string {
	return c.chainID
}

func (c *Client) GetPeers() ([]peer.AddrInfo, error) {
	peerCh, err := c.disc.FindPeers(context.Background(), c.getNamespace(), cdiscovery.Limit(peerLimit))
	if err != nil {
		return nil, err
	}

	var peers []peer.AddrInfo
	for p := range peerCh {
		if p.ID == c.host.ID() {
			continue
		}
		peers = append(peers, p)
	}
	return peers, nil
}

func (c *Client) GetNetworkInfo() (NetworkInfo, error) {
	hostAddrs := c.host.Addrs()
	addrs := make([]string, 0, len(hostAddrs))
	peerIDSuffix := "/p2p/" + c.host.ID().String()
	for _, a := range hostAddrs {
		addr := a.String()
		// Only append peer ID if not already present
		if !strings.HasSuffix(addr, peerIDSuffix) {
			addr = addr + peerIDSuffix
		}
		addrs = append(addrs, addr)
	}

	return NetworkInfo{
		ID:             c.host.ID().String(),
		ListenAddress:  addrs,
		ConnectedPeers: c.PeerIDs(),
	}, nil
}
