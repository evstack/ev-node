package sync

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/celestiaorg/go-header"
	goheaderp2p "github.com/celestiaorg/go-header/p2p"
	goheaderstore "github.com/celestiaorg/go-header/store"
	goheadersync "github.com/celestiaorg/go-header/sync"
	ds "github.com/ipfs/go-datastore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/types"
)

type syncType string

const (
	headerSync syncType = "headerSync"
	dataSync   syncType = "dataSync"
)

// TODO: when we add pruning we can remove this
const ninetyNineYears = 99 * 365 * 24 * time.Hour

// SyncService is the P2P Sync Service for blocks and headers.
//
// Uses the go-header library for handling all P2P logic.
type SyncService[H header.Header[H]] struct {
	conf     config.Config
	logger   zerolog.Logger
	syncType syncType

	genesis genesis.Genesis

	p2p *p2p.Client

	ex                *goheaderp2p.Exchange[H]
	sub               *goheaderp2p.Subscriber[H]
	p2pServer         *goheaderp2p.ExchangeServer[H]
	store             *goheaderstore.Store[H]
	syncer            *goheadersync.Syncer[H]
	syncerStatus      *SyncerStatus
	topicSubscription header.Subscription[H]
}

// DataSyncService is the P2P Sync Service for blocks.
type DataSyncService = SyncService[*types.Data]

// HeaderSyncService is the P2P Sync Service for headers.
type HeaderSyncService = SyncService[*types.SignedHeader]

// NewDataSyncService returns a new DataSyncService.
func NewDataSyncService(
	store ds.Batching,
	conf config.Config,
	genesis genesis.Genesis,
	p2p *p2p.Client,
	logger zerolog.Logger,
) (*DataSyncService, error) {
	return newSyncService[*types.Data](store, dataSync, conf, genesis, p2p, logger)
}

// NewHeaderSyncService returns a new HeaderSyncService.
func NewHeaderSyncService(
	store ds.Batching,
	conf config.Config,
	genesis genesis.Genesis,
	p2p *p2p.Client,
	logger zerolog.Logger,
) (*HeaderSyncService, error) {
	return newSyncService[*types.SignedHeader](store, headerSync, conf, genesis, p2p, logger)
}

func newSyncService[H header.Header[H]](
	store ds.Batching,
	syncType syncType,
	conf config.Config,
	genesis genesis.Genesis,
	p2p *p2p.Client,
	logger zerolog.Logger,
) (*SyncService[H], error) {
	if p2p == nil {
		return nil, errors.New("p2p client cannot be nil")
	}
	ss, err := goheaderstore.NewStore[H](
		store,
		goheaderstore.WithStorePrefix(string(syncType)),
		goheaderstore.WithMetrics(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the %s store: %w", syncType, err)
	}

	return &SyncService[H]{
		conf:         conf,
		genesis:      genesis,
		p2p:          p2p,
		store:        ss,
		syncType:     syncType,
		logger:       logger,
		syncerStatus: new(SyncerStatus),
	}, nil
}

// Store returns the store of the SyncService
func (syncService *SyncService[H]) Store() header.Store[H] {
	return syncService.store
}

// WriteToStoreAndBroadcast initializes store if needed and broadcasts provided header or block.
// Note: Only returns an error in case store can't be initialized. Logs error if there's one while broadcasting.
func (syncService *SyncService[H]) WriteToStoreAndBroadcast(ctx context.Context, headerOrData H) error {
	if syncService.genesis.InitialHeight == 0 {
		return fmt.Errorf("invalid initial height; cannot be zero")
	}

	if headerOrData.IsZero() {
		return fmt.Errorf("empty header/data cannot write to store or broadcast")
	}

	isGenesis := headerOrData.Height() == syncService.genesis.InitialHeight
	if isGenesis { // when starting the syncer for the first time, we have no blocks, so initFromP2P didn't initialize the genesis block.
		if err := syncService.initStore(ctx, headerOrData); err != nil {
			return fmt.Errorf("failed to initialize the store: %w", err)
		}
	}

	firstStart := false
	if !syncService.syncerStatus.started.Load() {
		firstStart = true
		if err := syncService.startSyncer(ctx); err != nil {
			return fmt.Errorf("failed to start syncer after initializing the store: %w", err)
		}
	}

	// Broadcast for subscribers
	if err := syncService.sub.Broadcast(ctx, headerOrData); err != nil {
		// for the first block when starting the app, broadcast error is expected
		// as we have already initialized the store for starting the syncer.
		// Hence, we ignore the error. Exact reason: validation ignored
		if (firstStart && errors.Is(err, pubsub.ValidationError{Reason: pubsub.RejectValidationIgnored})) ||
			// for the genesis header, broadcast error is expected as we have already initialized the store
			// for starting the syncer. Hence, we ignore the error.
			// exact reason: validation failed, err header verification failed: known header: '1' <= current '1'
			(isGenesis && errors.Is(err, pubsub.ValidationError{Reason: pubsub.RejectValidationFailed})) {

			return nil
		}

		syncService.logger.Error().Err(err).Msg("failed to broadcast")
	}

	return nil
}

// Start is a part of Service interface.
func (syncService *SyncService[H]) Start(ctx context.Context) error {
	// setup P2P infrastructure, but don't start Subscriber yet.
	peerIDs, err := syncService.setupP2PInfrastructure(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup syncer P2P infrastructure: %w", err)
	}

	// create syncer, must be before initFromP2PWithRetry which calls startSyncer.
	if syncService.syncer, err = newSyncer(
		syncService.ex,
		syncService.store,
		syncService.sub,
		[]goheadersync.Option{goheadersync.WithBlockTime(syncService.conf.Node.BlockTime.Duration)},
	); err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}

	// initialize stores from P2P (blocking until genesis is fetched for followers)
	// Aggregators (no peers configured) return immediately and initialize on first produced block.
	if err := syncService.initFromP2PWithRetry(ctx, peerIDs); err != nil {
		return fmt.Errorf("failed to initialize stores from P2P: %w", err)
	}

	// start the subscriber, stores are guaranteed to have genesis for followers.
	//
	// NOTE: we must start the subscriber after the syncer is initialized in initFromP2PWithRetry to ensure p2p syncing
	// works correctly.
	if err := syncService.startSubscriber(ctx); err != nil {
		return fmt.Errorf("failed to start subscriber: %w", err)
	}

	return nil
}

// startSyncer starts the SyncService's syncer
func (syncService *SyncService[H]) startSyncer(ctx context.Context) error {
	if syncService.syncerStatus.isStarted() {
		return nil
	}

	if err := syncService.syncer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start syncer: %w", err)
	}

	syncService.syncerStatus.started.Store(true)
	return nil
}

// initStore initializes the store with the given initial header.
// it is a no-op if the store is already initialized.
func (syncService *SyncService[H]) initStore(ctx context.Context, initial H) error {
	if initial.IsZero() {
		return errors.New("failed to initialize the store")
	}

	if _, err := syncService.store.Head(ctx); errors.Is(err, header.ErrNotFound) || errors.Is(err, header.ErrEmptyStore) {
		if err := syncService.store.Append(ctx, initial); err != nil {
			return err
		}

		if err := syncService.store.Sync(ctx); err != nil {
			return err
		}
	}

	return nil
}

// setupP2PInfrastructure sets up the P2P infrastructure (Exchange, ExchangeServer, Store)
// but does not start the Subscriber. Returns peer IDs for later use.
func (syncService *SyncService[H]) setupP2PInfrastructure(ctx context.Context) ([]peer.ID, error) {
	ps := syncService.p2p.PubSub()
	var err error

	// Create subscriber but DON'T start it yet
	syncService.sub, err = goheaderp2p.NewSubscriber[H](
		ps,
		pubsub.DefaultMsgIdFn,
		goheaderp2p.WithSubscriberNetworkID(syncService.getChainID()),
		goheaderp2p.WithSubscriberMetrics(),
	)
	if err != nil {
		return nil, err
	}

	if err := syncService.store.Start(ctx); err != nil {
		return nil, fmt.Errorf("error while starting store: %w", err)
	}

	_, _, network, err := syncService.p2p.Info()
	if err != nil {
		return nil, fmt.Errorf("error while fetching the network: %w", err)
	}
	networkID := syncService.getNetworkID(network)

	if syncService.p2pServer, err = newP2PServer(syncService.p2p.Host(), syncService.store, networkID); err != nil {
		return nil, fmt.Errorf("error while creating p2p server: %w", err)
	}
	if err := syncService.p2pServer.Start(ctx); err != nil {
		return nil, fmt.Errorf("error while starting p2p server: %w", err)
	}

	peerIDs := syncService.getPeerIDs()

	if syncService.ex, err = newP2PExchange[H](syncService.p2p.Host(), peerIDs, networkID, syncService.genesis.ChainID, syncService.p2p.ConnectionGater()); err != nil {
		return nil, fmt.Errorf("error while creating exchange: %w", err)
	}
	if err := syncService.ex.Start(ctx); err != nil {
		return nil, fmt.Errorf("error while starting exchange: %w", err)
	}
	return peerIDs, nil
}

// startSubscriber starts the Subscriber and subscribes to the P2P topic.
// This should be called AFTER stores are initialized to ensure proper validation.
func (syncService *SyncService[H]) startSubscriber(ctx context.Context) error {
	if err := syncService.sub.Start(ctx); err != nil {
		return fmt.Errorf("error while starting subscriber: %w", err)
	}

	var err error
	if syncService.topicSubscription, err = syncService.sub.Subscribe(); err != nil {
		return fmt.Errorf("error while subscribing: %w", err)
	}

	return nil
}

// initFromP2PWithRetry initializes the syncer from P2P with a retry mechanism.
// If trusted hash is available, it fetches the trusted header/block (by hash) from peers.
// Otherwise, it tries to fetch the genesis header/block by height.
func (syncService *SyncService[H]) initFromP2PWithRetry(ctx context.Context, peerIDs []peer.ID) error {
	if len(peerIDs) == 0 {
		return nil
	}

	tryInit := func(ctx context.Context) (bool, error) {
		var (
			trusted H
			err     error
		)

		if syncService.conf.Node.TrustedHash != "" {
			trustedHashBytes, err := hex.DecodeString(syncService.conf.Node.TrustedHash)
			if err != nil {
				return false, fmt.Errorf("failed to parse the trusted hash for initializing the store: %w", err)
			}
			if trusted, err = syncService.ex.Get(ctx, trustedHashBytes); err != nil {
				return false, fmt.Errorf("failed to fetch the trusted header/block for initializing the store: %w", err)
			}
		} else {
			if trusted, err = syncService.ex.GetByHeight(ctx, syncService.genesis.InitialHeight); err != nil {
				return false, fmt.Errorf("failed to fetch the genesis: %w", err)
			}
		}

		if err := syncService.initStore(ctx, trusted); err != nil {
			return false, fmt.Errorf("failed to initialize the store: %w", err)
		}
		if err := syncService.startSyncer(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	// block with exponential backoff until initialization succeeds or context is canceled.
	backoff := 1 * time.Second
	maxBackoff := 10 * time.Second

	timeoutTimer := time.NewTimer(time.Minute * 10)
	defer timeoutTimer.Stop()

	for {
		ok, err := tryInit(ctx)
		if ok {
			return nil
		}

		syncService.logger.Info().Err(err).Dur("retry_in", backoff).Msg("headers not yet available from peers, waiting to initialize header sync")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout reached while trying to initialize the store after 10 minutes: %w", err)
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// Stop is a part of Service interface.
//
// `store` is closed last because it's used by other services.
func (syncService *SyncService[H]) Stop(ctx context.Context) error {
	// unsubscribe from topic first so that sub.Stop() does not fail
	syncService.topicSubscription.Cancel()
	err := errors.Join(
		syncService.p2pServer.Stop(ctx),
		syncService.ex.Stop(ctx),
		syncService.sub.Stop(ctx),
	)
	if syncService.syncerStatus.isStarted() {
		err = errors.Join(err, syncService.syncer.Stop(ctx))
	}
	err = errors.Join(err, syncService.store.Stop(ctx))
	return err
}

// newP2PServer constructs a new ExchangeServer using the given Network as a protocolID suffix.
func newP2PServer[H header.Header[H]](
	host host.Host,
	store *goheaderstore.Store[H],
	network string,
	opts ...goheaderp2p.Option[goheaderp2p.ServerParameters],
) (*goheaderp2p.ExchangeServer[H], error) {
	opts = append(opts,
		goheaderp2p.WithNetworkID[goheaderp2p.ServerParameters](network),
		goheaderp2p.WithMetrics[goheaderp2p.ServerParameters](),
	)
	return goheaderp2p.NewExchangeServer(host, store, opts...)
}

func newP2PExchange[H header.Header[H]](
	host host.Host,
	peers []peer.ID,
	network, chainID string,
	conngater *conngater.BasicConnectionGater,
	opts ...goheaderp2p.Option[goheaderp2p.ClientParameters],
) (*goheaderp2p.Exchange[H], error) {
	opts = append(opts,
		goheaderp2p.WithNetworkID[goheaderp2p.ClientParameters](network),
		goheaderp2p.WithChainID(chainID),
		goheaderp2p.WithMetrics[goheaderp2p.ClientParameters](),
	)
	return goheaderp2p.NewExchange[H](host, peers, conngater, opts...)
}

// newSyncer constructs new Syncer for headers/blocks.
func newSyncer[H header.Header[H]](
	ex header.Exchange[H],
	store header.Store[H],
	sub header.Subscriber[H],
	opts []goheadersync.Option,
) (*goheadersync.Syncer[H], error) {
	opts = append(opts,
		goheadersync.WithMetrics(),
		goheadersync.WithPruningWindow(ninetyNineYears),
		goheadersync.WithTrustingPeriod(ninetyNineYears),
	)
	return goheadersync.NewSyncer(ex, store, sub, opts...)
}

func (syncService *SyncService[H]) getNetworkID(network string) string {
	return network + "-" + string(syncService.syncType)
}

func (syncService *SyncService[H]) getChainID() string {
	return syncService.genesis.ChainID + "-" + string(syncService.syncType)
}

func (syncService *SyncService[H]) getPeerIDs() []peer.ID {
	peerIDs := syncService.p2p.PeerIDs()
	if !syncService.conf.Node.Aggregator {
		peerIDs = append(peerIDs, getPeers(syncService.conf.P2P.Peers, syncService.logger)...)
	}
	return peerIDs
}

func getPeers(seeds string, logger zerolog.Logger) []peer.ID {
	var peerIDs []peer.ID
	if seeds == "" {
		return peerIDs
	}
	sl := strings.SplitSeq(seeds, ",")

	for seed := range sl {
		maddr, err := multiaddr.NewMultiaddr(seed)
		if err != nil {
			logger.Error().Str("address", seed).Err(err).Msg("failed to parse peer")
			continue
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			logger.Error().Str("address", maddr.String()).Err(err).Msg("failed to create addr info for peer")
			continue
		}
		peerIDs = append(peerIDs, addrInfo.ID)
	}
	return peerIDs
}
