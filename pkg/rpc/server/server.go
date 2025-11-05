package server

import (
	"context"
	"fmt"

	"net/http"
	"time"

	"encoding/binary"
	"errors"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	coreda "github.com/evstack/ev-node/core/da"
	ds "github.com/ipfs/go-datastore"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/evstack/ev-node/pkg/config"
	"github.com/evstack/ev-node/pkg/p2p"
	"github.com/evstack/ev-node/pkg/store"
	"github.com/evstack/ev-node/types"
	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
	rpc "github.com/evstack/ev-node/types/pb/evnode/v1/v1connect"
)

var _ rpc.StoreServiceHandler = (*StoreServer)(nil)

// StoreServer implements the StoreService defined in the proto file
type StoreServer struct {
	store  store.Store
	logger zerolog.Logger
}

// NewStoreServer creates a new StoreServer instance
func NewStoreServer(store store.Store, logger zerolog.Logger) *StoreServer {
	return &StoreServer{
		store:  store,
		logger: logger,
	}
}

// GetBlock implements the GetBlock RPC method
func (s *StoreServer) GetBlock(
	ctx context.Context,
	req *connect.Request[pb.GetBlockRequest],
) (*connect.Response[pb.GetBlockResponse], error) {
	var header *types.SignedHeader
	var data *types.Data
	var err error

	switch identifier := req.Msg.Identifier.(type) {
	case *pb.GetBlockRequest_Height:
		fetchHeight := identifier.Height
		if fetchHeight == 0 {
			// Subcase 2a: Height is 0 -> Fetch latest block
			fetchHeight, err = s.store.Height(ctx)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get latest height: %w", err))
			}
			if fetchHeight == 0 {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("store is empty, no latest block available"))
			}
		}
		// Fetch by the determined height (either specific or latest)
		header, data, err = s.store.GetBlockData(ctx, fetchHeight)

	case *pb.GetBlockRequest_Hash:
		hash := types.Hash(identifier.Hash)
		header, data, err = s.store.GetBlockByHash(ctx, hash)

	default:
		// This case handles potential future identifier types or invalid states
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or unsupported identifier type provided"))
	}

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to retrieve block data: %w", err))
	}

	// Convert retrieved types to protobuf types
	pbHeader, err := header.ToProto()
	if err != nil {
		// Error during conversion indicates an issue with the retrieved data or proto definition
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert block header to proto format: %w", err))
	}
	pbData := data.ToProto() // Assuming data.ToProto() exists and doesn't return an error

	// Return the successful response
	resp := &pb.GetBlockResponse{
		Block: &pb.Block{
			Header: pbHeader,
			Data:   pbData,
		},
	}

	// Fetch and set DA heights
	blockHeight := header.Height()
	if blockHeight > 0 { // DA heights are not stored for genesis/height 0 in the current impl
		headerDAHeightKey := fmt.Sprintf("%s/%d/h", store.HeightToDAHeightKey, blockHeight)
		headerDAHeightBytes, err := s.store.GetMetadata(ctx, headerDAHeightKey)
		if err == nil && len(headerDAHeightBytes) == 8 {
			resp.HeaderDaHeight = binary.LittleEndian.Uint64(headerDAHeightBytes)
		} else if err != nil && !errors.Is(err, ds.ErrNotFound) {
			s.logger.Error().Uint64("height", blockHeight).Err(err).Msg("Error fetching header DA height for block")
		}

		dataDAHeightKey := fmt.Sprintf("%s/%d/d", store.HeightToDAHeightKey, blockHeight)
		dataDAHeightBytes, err := s.store.GetMetadata(ctx, dataDAHeightKey)
		if err == nil && len(dataDAHeightBytes) == 8 {
			resp.DataDaHeight = binary.LittleEndian.Uint64(dataDAHeightBytes)
		} else if err != nil && !errors.Is(err, ds.ErrNotFound) {
			s.logger.Error().Uint64("height", blockHeight).Err(err).Msg("Error fetching data DA height for block")
		}
	}

	return connect.NewResponse(resp), nil
}

// GetState implements the GetState RPC method
func (s *StoreServer) GetState(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetStateResponse], error) {
	state, err := s.store.GetState(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Convert state to protobuf type
	pbState := &pb.State{
		AppHash:         state.AppHash,
		LastBlockHeight: state.LastBlockHeight,
		LastBlockTime:   timestamppb.New(state.LastBlockTime),
		DaHeight:        state.DAHeight,
		ChainId:         state.ChainID,
		Version: &pb.Version{
			Block: state.Version.Block,
			App:   state.Version.App,
		},
		InitialHeight: state.InitialHeight,
	}

	return connect.NewResponse(&pb.GetStateResponse{
		State: pbState,
	}), nil
}

// GetGenesisDaHeight implements the GetGenesisDaHeight RPC method
func (s *StoreServer) GetGenesisDaHeight(
	ctx context.Context,
	_ *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetGenesisDaHeightResponse], error) {
	resp, err := s.GetMetadata(ctx, connect.NewRequest(&pb.GetMetadataRequest{
		Key: store.GenesisDAHeightKey,
	}))
	if err != nil {
		return nil, err
	}

	if len(resp.Msg.GetValue()) != 8 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("invalid metadata value"))
	}

	return connect.NewResponse(&pb.GetGenesisDaHeightResponse{
		Height: binary.LittleEndian.Uint64(resp.Msg.GetValue()),
	}), nil

}

// GetMetadata implements the GetMetadata RPC method
func (s *StoreServer) GetMetadata(
	ctx context.Context,
	req *connect.Request[pb.GetMetadataRequest],
) (*connect.Response[pb.GetMetadataResponse], error) {
	value, err := s.store.GetMetadata(ctx, req.Msg.Key)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&pb.GetMetadataResponse{
		Value: value,
	}), nil
}

type ConfigServer struct {
	config config.Config
	signer []byte
	logger zerolog.Logger
}

func NewConfigServer(config config.Config, proposerAddress []byte, logger zerolog.Logger) *ConfigServer {
	return &ConfigServer{
		config: config,
		signer: proposerAddress,
		logger: logger,
	}
}

func (cs *ConfigServer) GetNamespace(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetNamespaceResponse], error) {

	hns := coreda.NamespaceFromString(cs.config.DA.GetNamespace())
	dns := coreda.NamespaceFromString(cs.config.DA.GetDataNamespace())

	return connect.NewResponse(&pb.GetNamespaceResponse{
		HeaderNamespace: hns.HexString(),
		DataNamespace:   dns.HexString(),
	}), nil
}

func (cs *ConfigServer) GetSignerInfo(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetSignerInfoResponse], error) {

	// If no signer is available, return an error
	if cs.signer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sequencer signer not available"))
	}

	return connect.NewResponse(&pb.GetSignerInfoResponse{
		Address: cs.signer,
	}), nil
}

// P2PServer implements the P2PService defined in the proto file
type P2PServer struct {
	// Add dependencies needed for P2P functionality
	peerManager p2p.P2PRPC
}

// NewP2PServer creates a new P2PServer instance
func NewP2PServer(peerManager p2p.P2PRPC) *P2PServer {
	return &P2PServer{
		peerManager: peerManager,
	}
}

// GetPeerInfo implements the GetPeerInfo RPC method
func (p *P2PServer) GetPeerInfo(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetPeerInfoResponse], error) {
	peers, err := p.peerManager.GetPeers()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get peer info: %w", err))
	}

	// Convert to protobuf format
	pbPeers := make([]*pb.PeerInfo, len(peers))
	for i, peer := range peers {
		pbPeers[i] = &pb.PeerInfo{
			Id:      peer.ID.String(),
			Address: peer.String(),
		}
	}

	return connect.NewResponse(&pb.GetPeerInfoResponse{
		Peers: pbPeers,
	}), nil
}

// GetNetInfo implements the GetNetInfo RPC method
func (p *P2PServer) GetNetInfo(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetNetInfoResponse], error) {
	netInfo, err := p.peerManager.GetNetworkInfo()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get network info: %w", err))
	}

	pbNetInfo := &pb.NetInfo{
		Id:              netInfo.ID,
		ListenAddresses: netInfo.ListenAddress,
	}

	return connect.NewResponse(&pb.GetNetInfoResponse{
		NetInfo: pbNetInfo,
	}), nil
}

// HealthServer implements the HealthService defined in the proto file
// DEPRECATED: This is a legacy compatibility shim for external frameworks.
// New code should use GET /health/live HTTP endpoint instead.
type HealthServer struct {
	store  store.Store
	config config.Config
	logger zerolog.Logger
}

// NewHealthServer creates a new HealthServer instance
func NewHealthServer(store store.Store, config config.Config, logger zerolog.Logger) *HealthServer {
	return &HealthServer{
		store:  store,
		config: config,
		logger: logger,
	}
}

// Livez implements the HealthService.Livez RPC
// DEPRECATED: Use GET /health/live HTTP endpoint instead. This endpoint exists only
// for backward compatibility with external testing frameworks.
func (h *HealthServer) Livez(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[pb.GetHealthResponse], error) {
	// Log deprecation warning
	h.logger.Warn().
		Str("deprecated_endpoint", "/evnode.v1.HealthService/Livez").
		Str("recommended_endpoint", "GET /health/live").
		Msg("DEPRECATED: gRPC health endpoint called. Please migrate to HTTP endpoint GET /health/live")

	status := pb.HealthStatus_PASS

	// For aggregator nodes, check if block production is healthy
	if h.config.Node.Aggregator {
		state, err := h.store.GetState(ctx)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get state for health check")
			return connect.NewResponse(&pb.GetHealthResponse{
				Status: pb.HealthStatus_FAIL,
			}), nil
		}

		// If we have blocks, check if the last block time is recent
		if state.LastBlockHeight > 0 {
			timeSinceLastBlock := time.Since(state.LastBlockTime)

			// Calculate the threshold based on block time
			blockTime := h.config.Node.BlockTime.Duration

			// For lazy mode, use the lazy block interval instead
			if h.config.Node.LazyMode {
				blockTime = h.config.Node.LazyBlockInterval.Duration
			}

			warnThreshold := blockTime * 3 // healthCheckWarnMultiplier
			failThreshold := blockTime * 5 // healthCheckFailMultiplier

			if timeSinceLastBlock > failThreshold {
				h.logger.Error().
					Dur("time_since_last_block", timeSinceLastBlock).
					Dur("fail_threshold", failThreshold).
					Uint64("last_block_height", state.LastBlockHeight).
					Time("last_block_time", state.LastBlockTime).
					Msg("Health check: node has stopped producing blocks (FAIL)")
				status = pb.HealthStatus_FAIL
			} else if timeSinceLastBlock > warnThreshold {
				h.logger.Warn().
					Dur("time_since_last_block", timeSinceLastBlock).
					Dur("warn_threshold", warnThreshold).
					Uint64("last_block_height", state.LastBlockHeight).
					Time("last_block_time", state.LastBlockTime).
					Msg("Health check: block production is slow (WARN)")
				status = pb.HealthStatus_WARN
			}
		}
	}

	// Add deprecation warning to response headers
	resp := connect.NewResponse(&pb.GetHealthResponse{
		Status: status,
	})
	resp.Header().Set("X-Deprecated", "true")
	resp.Header().Set("X-Deprecated-Message", "Use GET /health/live instead")
	resp.Header().Set("Warning", "299 - \"Deprecated endpoint. Use GET /health/live\"")

	return resp, nil
}

// NewServiceHandler creates a new HTTP handler for Store, P2P, Health and Config services
func NewServiceHandler(store store.Store, peerManager p2p.P2PRPC, proposerAddress []byte, logger zerolog.Logger, config config.Config, bestKnown BestKnownHeightProvider) (http.Handler, error) {
	storeServer := NewStoreServer(store, logger)
	p2pServer := NewP2PServer(peerManager)
	healthServer := NewHealthServer(store, config, logger) // Legacy gRPC endpoint
	configServer := NewConfigServer(config, proposerAddress, logger)

	mux := http.NewServeMux()

	compress1KB := connect.WithCompressMinBytes(1024)
	reflector := grpcreflect.NewStaticReflector(
		rpc.StoreServiceName,
		rpc.P2PServiceName,
		rpc.HealthServiceName, // Legacy gRPC endpoint
		rpc.ConfigServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector, compress1KB))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector, compress1KB))

	// Register StoreService
	storePath, storeHandler := rpc.NewStoreServiceHandler(storeServer)
	mux.Handle(storePath, storeHandler)

	// Register P2PService
	p2pPath, p2pHandler := rpc.NewP2PServiceHandler(p2pServer)
	mux.Handle(p2pPath, p2pHandler)

	// Register HealthService (legacy gRPC endpoint for backward compatibility)
	healthPath, healthHandler := rpc.NewHealthServiceHandler(healthServer)
	mux.Handle(healthPath, healthHandler)

	configPath, configHandler := rpc.NewConfigServiceHandler(configServer)
	mux.Handle(configPath, configHandler)

	// Register custom HTTP endpoints (including the preferred /health/live endpoint)
	RegisterCustomHTTPEndpoints(mux, store, peerManager, config, bestKnown, logger)

	// Use h2c to support HTTP/2 without TLS
	return h2c.NewHandler(mux, &http2.Server{
		IdleTimeout:          120 * time.Second,
		MaxReadFrameSize:     1 << 24,
		MaxConcurrentStreams: 100,
		ReadIdleTimeout:      30 * time.Second,
		PingTimeout:          15 * time.Second,
	}), nil
}
