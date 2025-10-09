package raft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"
)

type clusterClient interface {
	AddPeer(ctx context.Context, id, addr string) error
	RemovePeer(ctx context.Context, id string) error
}

// Node represents a raft consensus node
type Node struct {
	raft          *raft.Raft
	fsm           *FSM
	config        *Config
	clusterClient clusterClient
	logger        zerolog.Logger
}

// Config holds raft node configuration
type Config struct {
	NodeID      string
	RaftAddr    string
	RaftDir     string
	Bootstrap   bool
	Peers       []string
	SnapCount   uint64
	SendTimeout time.Duration
}

// FSM implements raft.FSM for block state
type FSM struct {
	logger  zerolog.Logger
	state   *block.RaftBlockState
	applyCh chan<- block.RaftApplyMsg
}

// NewNode creates a new raft node
func NewNode(cfg *Config, clusterClient clusterClient, logger zerolog.Logger) (*Node, error) {
	if err := os.MkdirAll(cfg.RaftDir, 0755); err != nil {
		return nil, fmt.Errorf("create raft dir: %w", err)
	}

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.NodeID)
	raftConfig.LogLevel = "INFO"

	fsm := &FSM{
		logger: logger.With().Str("component", "raft-fsm").Logger(),
		state:  &block.RaftBlockState{},
	}

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.RaftDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("create log store: %w", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.RaftDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("create stable store: %w", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(cfg.RaftDir, int(cfg.SnapCount), os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("create snapshot store: %w", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", cfg.RaftAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve raft addr: %w", err)
	}

	transport, err := raft.NewTCPTransport(cfg.RaftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("create transport: %w", err)
	}

	r, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("create raft: %w", err)
	}

	node := &Node{
		raft:          r,
		fsm:           fsm,
		config:        cfg,
		clusterClient: clusterClient,
		logger:        logger.With().Str("component", "raft-node").Logger(),
	}

	return node, nil
}

func (n *Node) Start(ctx context.Context) error {
	if !n.config.Bootstrap {
		n.logger.Info().Msg("Join raft cluster")
		return n.clusterClient.AddPeer(ctx, n.config.NodeID, n.config.RaftAddr)
	}

	n.logger.Info().Msg("Boostrap raft cluster")
	cfg := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(n.config.NodeID),
				Address: raft.ServerAddress(n.config.RaftAddr),
			},
		},
	}
	for _, peer := range n.config.Peers {
		addr, err := splitPeerAddr(peer)
		if err != nil {
			return err
		}
		cfg.Servers = append(cfg.Servers, addr)
	}
	cfg.Servers = deduplicateServers(cfg.Servers)

	if err := n.raft.BootstrapCluster(cfg).Error(); err != nil {
		return fmt.Errorf("bootstrap cluster: %w", err)
	}
	n.logger.Info().Msg("bootstrapped raft cluster")
	return nil

}

func deduplicateServers(servers []raft.Server) []raft.Server {
	seen := make(map[raft.ServerID]struct{})
	unique := make([]raft.Server, 0, len(servers))
	for _, server := range servers {
		key := server.ID
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			unique = append(unique, server)
		}
	}
	return unique
}

func (n *Node) Stop() error {
	return n.raft.Shutdown().Error()
}

// IsLeader returns true if this node is the raft leader
func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}
func (n *Node) NodeID() string {
	return n.config.NodeID
}

// ProposeBlock proposes a block state to be replicated via raft
func (n *Node) Broadcast(ctx context.Context, state *block.RaftBlockState) error {
	if !n.IsLeader() {
		return fmt.Errorf("not leader")
	}

	data, err := json.Marshal(state) // todo:use protobuf
	if err != nil {
		return fmt.Errorf("marshal block state: %w", err)
	}

	future := n.raft.Apply(data, n.config.SendTimeout)
	if err := future.Error(); err != nil {
		return fmt.Errorf("apply log: %w", err)
	}

	return nil
}

// GetState returns the current replicated state
func (n *Node) GetState() *block.RaftBlockState {
	return n.fsm.state
}

// AddPeer adds a peer to the raft cluster
func (n *Node) AddPeer(nodeID, addr string) error {
	n.logger.Debug().Msgf("received join request for remote node %s at %s", nodeID, addr)

	configFuture := n.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	// remove first when node is already a member of the cluster
	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				n.logger.Debug().Msgf("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}
			future := n.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("removing existing node %s at %s: %w", nodeID, addr, err)
			}
		}
	}

	f := n.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	n.logger.Debug().Msgf("node %s at %s joined successfully", nodeID, addr)
	return nil
}

// RemovePeer removes a peer from the raft cluster
func (n *Node) RemovePeer(nodeID string) error {
	future := n.raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
	return future.Error()
}

// Shutdown stops the raft node
func (n *Node) Shutdown() error {
	return n.raft.Shutdown().Error()
}

// SetApplyCallback sets a callback to be called when log entries are applied
func (n *Node) SetApplyCallback(ch chan<- block.RaftApplyMsg) {
	n.fsm.applyCh = ch
}

// Apply implements raft.FSM
func (f *FSM) Apply(log *raft.Log) interface{} {
	var state block.RaftBlockState
	if err := json.Unmarshal(log.Data, &state); err != nil {
		f.logger.Error().Err(err).Msg("unmarshal block state")
		return err
	}

	f.state = &state
	f.logger.Debug().Uint64("height", state.Height).Msg("received block state")

	if f.applyCh != nil {
		select {
		case f.applyCh <- block.RaftApplyMsg{Index: log.Index, State: &state}:
		default:
			f.logger.Warn().Msg("apply channel full, dropping message")
		}
	}

	return nil
}

// Snapshot implements raft.FSM
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{state: f.state}, nil
}

// Restore implements raft.FSM
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var state block.RaftBlockState
	if err := json.NewDecoder(rc).Decode(&state); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}

	f.state = &state
	f.logger.Info().Uint64("height", state.Height).Msg("restored from snapshot")
	return nil
}

type fsmSnapshot struct {
	state *block.RaftBlockState
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		if err := json.NewEncoder(sink).Encode(s.state); err != nil {
			return err
		}
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
		return err
	}

	return nil
}

func (s *fsmSnapshot) Release() {}

func splitPeerAddr(peer string) (raft.Server, error) {
	parts := strings.Split(peer, "@")
	if len(parts) != 2 {
		return raft.Server{}, errors.New("expecting nodeID@address for peer")
	}
	return raft.Server{
		ID:      raft.ServerID(parts[0]),
		Address: raft.ServerAddress(parts[1]),
	}, nil
}
