package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/celestiaorg/go-header"
	goheaderp2p "github.com/celestiaorg/go-header/p2p"
	goheadersync "github.com/celestiaorg/go-header/sync"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
)

type syncType string

const (
	headerSync syncType = "headerSync"
	dataSync   syncType = "dataSync"
)

// HeaderSyncService is the P2P Sync Service for headers.
type HeaderSyncService = SyncService[*types.P2PSignedHeader]

// DataSyncService is the P2P Sync Service for blocks.
type DataSyncService = SyncService[*types.P2PData]

// SyncService is the P2P Sync Service for blocks and headers.
//
// Uses the go-header library for handling all P2P logic.
type SyncService[H store.EntityWithDAHint[H]] struct {
	conf     config.Config
	logger   zerolog.Logger
	syncType syncType

	genesis genesis.Genesis

	p2p *p2p.Client

	ex                *goheaderp2p.Exchange[H]
	sub               *goheaderp2p.Subscriber[H]
	p2pServer         *goheaderp2p.ExchangeServer[H]
	store             header.Store[H]
	syncer            *goheadersync.Syncer[H]
	syncerStatus      *SyncerStatus
	topicSubscription header.Subscription[H]

	storeInitialized atomic.Bool

	// trustedHeight tracks the configured trusted height for sync initialization
	trustedHeight uint64
	// trustedHeaderHash, trustedDataHash is the expected hash of the trusted header
	trustedHeaderHash, trustedDataHash string
}

// NewDataSyncService returns a new DataSyncService.
func NewDataSyncService(
	evStore store.Store,
	conf config.Config,
	genesis genesis.Genesis,
	p2p *p2p.Client,
	logger zerolog.Logger,
) (*DataSyncService, error) {
	storeAdapter := store.NewDataStoreAdapter(evStore, genesis)
	return newSyncService(storeAdapter, dataSync, conf, genesis, p2p, logger)
}

// NewHeaderSyncService returns a new HeaderSyncService.
func NewHeaderSyncService(
	evStore store.Store,
	conf config.Config,
	genesis genesis.Genesis,
	p2p *p2p.Client,
	logger zerolog.Logger,
) (*HeaderSyncService, error) {
	storeAdapter := store.NewHeaderStoreAdapter(evStore, genesis)
	return newSyncService(storeAdapter, headerSync, conf, genesis, p2p, logger)
}

func newSyncService[H store.EntityWithDAHint[H]](
	storeAdapter header.Store[H],
	syncType syncType,
	conf config.Config,
	genesis genesis.Genesis,
	p2p *p2p.Client,
	logger zerolog.Logger,
) (*SyncService[H], error) {
	if p2p == nil {
		return nil, errors.New("p2p client cannot be nil")
	}

	svc := &SyncService[H]{
		conf:         conf,
		genesis:      genesis,
		p2p:          p2p,
		store:        storeAdapter,
		syncType:     syncType,
		logger:       logger,
		syncerStatus: new(SyncerStatus),
	}

	return svc, nil
}

// Store returns the store of the SyncService
func (syncService *SyncService[H]) Store() header.Store[H] {
	return syncService.store
}

// WriteToStoreAndBroadcast broadcasts provided header or block to P2P network.
func (syncService *SyncService[H]) WriteToStoreAndBroadcast(ctx context.Context, headerOrData H, opts ...pubsub.PubOpt) error {
	if syncService.genesis.InitialHeight == 0 {
		return fmt.Errorf("invalid initial height; cannot be zero")
	}

	if headerOrData.IsZero() {
		return fmt.Errorf("empty header/data cannot write to store or broadcast")
	}

	storeInitialized := false
	if syncService.storeInitialized.CompareAndSwap(false, true) {
		var err error
		storeInitialized, err = syncService.initStore(ctx, headerOrData)
		if err != nil {
			syncService.storeInitialized.Store(false)
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
	if err := syncService.sub.Broadcast(ctx, headerOrData, opts...); err != nil {
		// for the first block when starting the app, broadcast error is expected
		// as we have already initialized the store for starting the syncer.
		// Hence, we ignore the error. Exact reason: validation ignored
		if (firstStart && errors.Is(err, pubsub.ValidationError{Reason: pubsub.RejectValidationIgnored})) ||
			// for the genesis header (or any first header used to bootstrap the store), broadcast error is expected as we have already initialized the store
			// for starting the syncer. Hence, we ignore the error.
			// exact reason: validation failed, err header verification failed: known header: '1' <= current '1'
			((storeInitialized) && errors.Is(err, pubsub.ValidationError{Reason: pubsub.RejectValidationFailed})) {

			return nil
		}

		syncService.logger.Error().Err(err).Msg("failed to broadcast")
	}

	return nil
}

func (s *SyncService[H]) AppendDAHint(ctx context.Context, daHeight uint64, heights ...uint64) error {
	entries := make([]H, 0, len(heights))
	for _, height := range heights {
		v, err := s.store.GetByHeight(ctx, height)
		if err != nil {
			if errors.Is(err, header.ErrNotFound) {
				s.logger.Debug().Uint64("height", height).Msg("cannot append DA height hint; header/data not found in store")
				continue
			}
			return err
		}
		v.SetDAHint(daHeight)
		entries = append(entries, v)
	}
	return s.store.Append(ctx, entries...)
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

	// Initialize trusted height configuration
	syncService.trustedHeight = syncService.conf.P2P.TrustedHeight
	syncService.trustedHeaderHash = syncService.conf.P2P.TrustedHeaderHash
	syncService.trustedDataHash = syncService.conf.P2P.TrustedDataHash

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
// Returns true when the store was initialized by this call.
func (syncService *SyncService[H]) initStore(ctx context.Context, initial H) (bool, error) {
	if initial.IsZero() {
		return false, errors.New("failed to initialize the store")
	}

	if _, err := syncService.store.Head(ctx); errors.Is(err, header.ErrNotFound) || errors.Is(err, header.ErrEmptyStore) {
		if err := syncService.store.Append(ctx, initial); err != nil {
			return false, err
		}

		// Sync is optional for adapters - they may not need explicit syncing
		if syncer, ok := syncService.store.(interface{ Sync(context.Context) error }); ok {
			if err := syncer.Sync(ctx); err != nil {
				return false, err
			}
		}

		return true, nil
	} else if err != nil {
		return false, err
	}

	return false, nil
}

// setupP2PInfrastructure sets up the P2P infrastructure (Exchange, ExchangeServer, Store)
// but does not start the Subscriber. Returns peer IDs for later use.
func (syncService *SyncService[H]) setupP2PInfrastructure(ctx context.Context) ([]peer.ID, error) {
	ps := syncService.p2p.PubSub()

	_, _, chainID, err := syncService.p2p.Info()
	if err != nil {
		return nil, fmt.Errorf("error while fetching the network: %w", err)
	}
	networkID := syncService.getNetworkID(chainID)

	// Create subscriber but DON'T start it yet
	syncService.sub, err = goheaderp2p.NewSubscriber[H](
		ps,
		pubsub.DefaultMsgIdFn,
		goheaderp2p.WithSubscriberNetworkID(networkID),
		goheaderp2p.WithSubscriberMetrics(),
	)
	if err != nil {
		return nil, err
	}

	// Start the store adapter if it has a Start method
	if starter, ok := syncService.store.(interface{ Start(context.Context) error }); ok {
		if err := starter.Start(ctx); err != nil {
			return nil, fmt.Errorf("error while starting store: %w", err)
		}
	}

	if syncService.p2pServer, err = newP2PServer(syncService.p2p.Host(), syncService.store, networkID); err != nil {
		return nil, fmt.Errorf("error while creating p2p server: %w", err)
	}
	if err := syncService.p2pServer.Start(ctx); err != nil {
		return nil, fmt.Errorf("error while starting p2p server: %w", err)
	}

	peerIDs := syncService.getPeerIDs()

	syncService.ex, err = newP2PExchange[H](syncService.p2p.Host(), peerIDs, networkID, syncService.genesis.ChainID, syncService.p2p.ConnectionGater())
	if err != nil {
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

// Height returns the current height storeda
func (s *SyncService[H]) Height() uint64 {
	return s.store.Height()
}

// initFromP2PWithRetry initializes the syncer from P2P with a retry mechanism.
// It inspects the local store to determine the first height to request:
//   - when the store already contains items, it reuses the latest height as the starting point;
//   - otherwise, it falls back to the configured genesis height.
//   - if trusted height is configured, it fetches from that height first and verifies the hash.
func (syncService *SyncService[H]) initFromP2PWithRetry(ctx context.Context, peerIDs []peer.ID) error {
	if len(peerIDs) == 0 {
		return nil
	}

	// If trusted height is configured, fetch from that height first
	if syncService.trustedHeight > 0 {
		if err := syncService.fetchAndVerifyTrustedHeader(ctx, peerIDs); err != nil {
			return fmt.Errorf("failed to fetch trusted header at height %d: %w", syncService.trustedHeight, err)
		}
	}

	tryInit := func(ctx context.Context) (bool, error) {
		var (
			trusted       H
			err           error
			heightToQuery uint64
		)

		head, headErr := syncService.store.Head(ctx)
		switch {
		case errors.Is(headErr, header.ErrNotFound), errors.Is(headErr, header.ErrEmptyStore):
			// If we have a trusted header, use its height as the starting point
			if syncService.trustedHeight > 0 {
				heightToQuery = syncService.trustedHeight
			} else {
				heightToQuery = syncService.genesis.InitialHeight
			}
		case headErr != nil:
			return false, fmt.Errorf("failed to inspect local store head: %w", headErr)
		default:
			heightToQuery = head.Height()
		}

		if trusted, err = syncService.ex.GetByHeight(ctx, heightToQuery); err != nil {
			return false, fmt.Errorf("failed to fetch height %d from peers: %w", heightToQuery, err)
		}

		if syncService.storeInitialized.CompareAndSwap(false, true) {
			if _, err := syncService.initStore(ctx, trusted); err != nil {
				syncService.storeInitialized.Store(false)
				return false, fmt.Errorf("failed to initialize the store: %w", err)
			}
		}
		if err := syncService.startSyncer(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	// block with exponential backoff until initialization succeeds, context is canceled, or timeout.
	// If timeout is reached, we return nil to allow startup to continue - DA sync will
	// provide headers and WriteToStoreAndBroadcast will lazily initialize the store/syncer.
	backoff := 1 * time.Second
	maxBackoff := 10 * time.Second

	p2pInitTimeout := 30 * time.Second
	timeoutTimer := time.NewTimer(p2pInitTimeout)
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
			syncService.logger.Warn().
				Dur("timeout", p2pInitTimeout).
				Msg("P2P header sync initialization timed out, deferring to DA sync")
			return nil
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// fetchAndVerifyTrustedHeader fetches the header at the trusted height from P2P
// and verifies it matches the trusted hash. If verification passes, it stores the header.
func (syncService *SyncService[H]) fetchAndVerifyTrustedHeader(ctx context.Context, peerIDs []peer.ID) error {
	syncService.logger.Info().Uint64("height", syncService.trustedHeight).Msg("fetching trusted header from P2P")

	// Fetch the header from trusted height
	trusted, err := syncService.ex.GetByHeight(ctx, syncService.trustedHeight)
	if err != nil {
		return fmt.Errorf("failed to fetch trusted header at height %d: %w", syncService.trustedHeight, err)
	}

	// Verify the hash matches
	actualHash := trusted.Hash().String()
	if actualHash != syncService.trustedHeaderHash && actualHash != syncService.trustedDataHash {
		return fmt.Errorf("trusted header hash mismatch at height %d: expected %s or %s, got %s",
			syncService.trustedHeight, syncService.trustedHeaderHash, syncService.trustedDataHash, actualHash)
	}

	syncService.logger.Info().Uint64("height", syncService.trustedHeight).
		Str("hash", actualHash).
		Msg("trusted header verified and stored")

	if err := syncService.store.Append(ctx, trusted); err != nil {
		return fmt.Errorf("failed to store trusted header: %w", err)
	}

	syncService.storeInitialized.Store(true)

	return nil
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
	// Stop the store adapter if it has a Stop method
	if stopper, ok := syncService.store.(interface{ Stop(context.Context) error }); ok {
		err = errors.Join(err, stopper.Stop(ctx))
	}
	return err
}

// newP2PServer constructs a new ExchangeServer using the given Network as a protocolID suffix.
func newP2PServer[H header.Header[H]](
	host host.Host,
	store header.Store[H],
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
	// using a very long duration for effectively disabling pruning and trusting period checks.
	const ninetyNineYears = 99 * 365 * 24 * time.Hour

	opts = append(opts,
		goheadersync.WithMetrics(),
		goheadersync.WithPruningWindow(ninetyNineYears), // pruning window not relevant, because of the store wrapper.
		goheadersync.WithTrustingPeriod(ninetyNineYears),
	)
	return goheadersync.NewSyncer(ex, store, sub, opts...)
}

func (syncService *SyncService[H]) getNetworkID(network string) string {
	return network + "-" + string(syncService.syncType)
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
