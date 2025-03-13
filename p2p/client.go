package p2p

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cometbft/cometbft/p2p"
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

	"github.com/rollkit/rollkit/config"
	"github.com/rollkit/rollkit/p2p/key"
	rollhash "github.com/rollkit/rollkit/pkg/hash"
	"github.com/rollkit/rollkit/third_party/log"
)

// TODO(tzdybal): refactor to configuration parameters
const (
	// reAdvertisePeriod defines a period after which P2P client re-attempt advertising namespace in DHT.
	reAdvertisePeriod = 1 * time.Hour

	// peerLimit defines limit of number of peers returned during active peer discovery.
	peerLimit = 60
)

// Client is a P2P client, implemented with libp2p.
//
// Initially, client connects to predefined seed nodes (aka bootnodes, bootstrap nodes).
// Those seed nodes serve Kademlia DHT protocol, and are agnostic to ORU chain. Using DHT
// peer routing and discovery clients find other peers within ORU network.
type Client struct {
	logger log.Logger

	conf    config.P2PConfig
	chainID string
	privKey crypto.PrivKey

	host  host.Host
	dht   *dht.IpfsDHT
	disc  *discovery.RoutingDiscovery
	gater *conngater.BasicConnectionGater
	ps    *pubsub.PubSub

	metrics *Metrics
}

// NewClient creates new Client object.
//
// Basic checks on parameters are done, and default parameters are provided for unset-configuration
// TODO(tzdybal): consider passing entire config, not just P2P config, to reduce number of arguments
func NewClient(conf config.Config, chainID string, ds datastore.Datastore, logger log.Logger, metrics *Metrics) (*Client, error) {
	if conf.RootDir == "" {
		return nil, fmt.Errorf("rootDir is required")
	}

	if conf.P2P.ListenAddress == "" {
		conf.P2P.ListenAddress = config.DefaultListenAddress
	}

	gater, err := conngater.NewBasicConnectionGater(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection gater: %w", err)
	}

	nodeKeyFile := filepath.Join(conf.RootDir, "config", "node_key.json")
	nodeKey, err := key.LoadOrGenNodeKey(nodeKeyFile)
	if err != nil {
		return nil, err
	}

	return &Client{
		conf:    conf.P2P,
		gater:   gater,
		privKey: nodeKey.PrivKey,
		chainID: chainID,
		logger:  logger,
		metrics: metrics,
	}, nil
}

// Start establish Client's P2P connectivity.
//
// Following steps are taken:
// 1. Setup libp2p host, start listening for incoming connections.
// 2. Setup gossibsub.
// 3. Setup DHT, establish connection to seed nodes and initialize peer discovery.
// 4. Use active peer discovery to look for peers from same ORU network.
func (c *Client) Start(ctx context.Context) error {
	c.logger.Debug("starting P2P client")
	host, err := c.listen()
	if err != nil {
		return err
	}
	return c.startWithHost(ctx, host)
}

func (c *Client) startWithHost(ctx context.Context, h host.Host) error {
	c.host = h
	for _, a := range c.host.Addrs() {
		c.logger.Info("listening on", "address", fmt.Sprintf("%s/p2p/%s", a, c.host.ID()))
	}

	c.logger.Debug("blocking blacklisted peers", "blacklist", c.conf.BlockedPeers)
	if err := c.setupBlockedPeers(c.parseAddrInfoList(c.conf.BlockedPeers)); err != nil {
		return err
	}

	c.logger.Debug("allowing whitelisted peers", "whitelist", c.conf.AllowedPeers)
	if err := c.setupAllowedPeers(c.parseAddrInfoList(c.conf.AllowedPeers)); err != nil {
		return err
	}

	c.logger.Debug("setting up gossiping")
	if err := c.setupGossiping(ctx); err != nil {
		return err
	}

	c.logger.Debug("setting up DHT")
	if err := c.setupDHT(ctx); err != nil {
		return err
	}

	c.logger.Debug("setting up active peer discovery")
	if err := c.peerDiscovery(ctx); err != nil {
		return err
	}

	return nil
}

// Close gently stops Client.
func (c *Client) Close() error {

	return errors.Join(
		c.dht.Close(),
		c.host.Close(),
	)
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
		peerIDs = append(peerIDs, conn.RemotePeer())
	}
	return peerIDs
}

// Peers returns list of peers connected to Client.
func (c *Client) Peers() []PeerConnection {
	conns := c.host.Network().Conns()
	res := make([]PeerConnection, 0, len(conns))
	for _, conn := range conns {
		pc := PeerConnection{
			NodeInfo: p2p.DefaultNodeInfo{
				ListenAddr:    c.conf.ListenAddress,
				Network:       c.chainID,
				DefaultNodeID: p2p.ID(conn.RemotePeer().String()),
				// TODO(tzdybal): fill more fields
			},
			IsOutbound: conn.Stat().Direction == network.DirOutbound,
			ConnectionStatus: p2p.ConnectionStatus{
				Duration: time.Since(conn.Stat().Opened),
				// TODO(tzdybal): fill more fields
			},
			RemoteIP: conn.RemoteMultiaddr().String(),
		}
		res = append(res, pc)
	}
	return res
}

func (c *Client) listen() (host.Host, error) {
	maddr, err := multiaddr.NewMultiaddr(c.conf.ListenAddress)
	if err != nil {
		return nil, err
	}

	return libp2p.New(libp2p.ListenAddrs(maddr), libp2p.Identity(c.privKey), libp2p.ConnectionGater(c.gater))
}

func (c *Client) setupDHT(ctx context.Context) error {
	seedNodes := c.parseAddrInfoList(c.conf.Seeds)
	if len(seedNodes) == 0 {
		c.logger.Info("no seed nodes - only listening for connections")
	}

	for _, sa := range seedNodes {
		c.logger.Debug("seed node", "addr", sa)
	}

	var err error
	c.dht, err = dht.New(ctx, c.host, dht.Mode(dht.ModeServer), dht.BootstrapPeers(seedNodes...))
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
	err := c.host.Connect(ctx, peer)
	if err != nil && ctx.Err() == nil {
		c.logger.Error("failed to connect to peer", "peer", peer, "error", err)
	}
}

func (c *Client) setupGossiping(ctx context.Context) error {
	var err error
	c.ps, err = pubsub.NewGossipSub(ctx, c.host)
	if err != nil {
		return err
	}
	return nil
}

// parseAddrInfoList parses a comma separated string of multiaddrs into a list of peer.AddrInfo structs
func (c *Client) parseAddrInfoList(addrInfoStr string) []peer.AddrInfo {
	if len(addrInfoStr) == 0 {
		return []peer.AddrInfo{}
	}
	peers := strings.Split(addrInfoStr, ",")
	addrs := make([]peer.AddrInfo, 0, len(peers))
	for _, p := range peers {
		maddr, err := multiaddr.NewMultiaddr(p)
		if err != nil {
			c.logger.Error("failed to parse peer", "address", p, "error", err)
			continue
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			c.logger.Error("failed to create addr info for peer", "address", maddr, "error", err)
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
