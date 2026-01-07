package raft

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/zerolog"
)

// Node represents a raft consensus node
type Node struct {
	raft   *raft.Raft
	fsm    *FSM
	config *Config
	logger zerolog.Logger
}

// Config holds raft node configuration
type Config struct {
	NodeID           string
	RaftAddr         string
	RaftDir          string
	Bootstrap        bool
	Peers            []string
	SnapCount        uint64
	SendTimeout      time.Duration
	HeartbeatTimeout time.Duration
}

// FSM implements raft.FSM for block state
type FSM struct {
	logger  zerolog.Logger
	state   *atomic.Pointer[RaftBlockState]
	applyCh chan<- RaftApplyMsg
}

// NewNode creates a new raft node
func NewNode(cfg *Config, logger zerolog.Logger) (*Node, error) {
	if err := os.MkdirAll(cfg.RaftDir, 0755); err != nil {
		return nil, fmt.Errorf("create raft dir: %w", err)
	}

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.NodeID)
	raftConfig.LogLevel = "INFO"
	raftConfig.HeartbeatTimeout = cfg.HeartbeatTimeout
	raftConfig.LeaderLeaseTimeout = cfg.HeartbeatTimeout / 2

	startPointer := new(atomic.Pointer[RaftBlockState])
	startPointer.Store(&RaftBlockState{})
	fsm := &FSM{
		logger: logger.With().Str("component", "raft-fsm").Logger(),
		state:  startPointer,
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
		raft:   r,
		fsm:    fsm,
		config: cfg,
		logger: logger.With().Str("component", "raft-node").Logger(),
	}

	return node, nil
}

func (n *Node) Start(_ context.Context) error {
	if n == nil {
		return nil
	}
	if !n.config.Bootstrap {
		// it is intended to fail fast here. at this stage only bootstrap mode is supported.
		return fmt.Errorf("raft cluster requires bootstrap mode")
	}

	if future := n.raft.GetConfiguration(); future.Error() == nil && len(future.Configuration().Servers) > 0 {
		n.logger.Info().Msg("cluster already bootstrapped, skipping")
		return nil
	}

	n.logger.Info().Msg("Boostrap raft cluster")
	thisNode := raft.Server{ID: raft.ServerID(n.config.NodeID), Address: raft.ServerAddress(n.config.RaftAddr)}
	cfg := raft.Configuration{
		Servers: []raft.Server{
			thisNode,
		},
	}
	for _, peer := range n.config.Peers {
		addr, err := splitPeerAddr(peer)
		if err != nil {
			return fmt.Errorf("peer %q : %w", peer, err)
		}
		if addr != thisNode {
			cfg.Servers = append(cfg.Servers, addr)
		}
	}

	if svrs := deduplicateServers(cfg.Servers); len(svrs) != len(cfg.Servers) {
		return fmt.Errorf("duplicate peers found in config: %v", cfg.Servers)
	}

	if err := n.raft.BootstrapCluster(cfg).Error(); err != nil {
		return fmt.Errorf("bootstrap cluster: %w", err)
	}
	n.logger.Info().Msg("bootstrapped raft cluster")
	return nil
}

func (n *Node) waitForMsgsLanded(timeout time.Duration) error {
	if n == nil {
		return nil
	}
	timeoutTicker := time.NewTicker(timeout)
	defer timeoutTicker.Stop()
	ticker := time.NewTicker(min(n.config.SendTimeout, timeout) / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if n.raft.AppliedIndex() >= n.raft.LastIndex() {
				return nil
			}
		case <-timeoutTicker.C:
			return errors.New("max wait time reached")
		}
	}
}

func (n *Node) Stop() error {
	if n == nil {
		return nil
	}
	return n.raft.Shutdown().Error()
}

// IsLeader returns true if this node is the raft leader
func (n *Node) IsLeader() bool {
	if n == nil || n.raft == nil {
		return false
	}
	return n.raft.State() == raft.Leader
}

func (n *Node) NodeID() string {
	return n.config.NodeID
}

func (n *Node) leaderID() string {
	_, id := n.raft.LeaderWithID()
	return string(id)
}

func (n *Node) leaderCh() <-chan bool {
	return n.raft.LeaderCh()
}

func (n *Node) leadershipTransfer() error {
	return n.raft.LeadershipTransfer().Error()
}

func (n *Node) Config() Config {
	return *n.config
}

// Broadcast proposes a block state to be replicated via raft
func (n *Node) Broadcast(_ context.Context, state *RaftBlockState) error {
	if !n.IsLeader() {
		return fmt.Errorf("not leader")
	}

	data, err := proto.Marshal(state)
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
func (n *Node) GetState() *RaftBlockState {
	return proto.Clone(n.fsm.state.Load()).(*RaftBlockState)
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

// SetApplyCallback sets a callback channel to receive notifications when a new block state is replicated.
// The channel must have sufficient buffer space since updates are published only once without blocking.
// If the channel is full, state updates will be skipped to prevent blocking the raft cluster.
func (n *Node) SetApplyCallback(ch chan<- RaftApplyMsg) {
	n.fsm.applyCh = ch
}

// Apply implements raft.FSM
func (f *FSM) Apply(log *raft.Log) interface{} {
	var state RaftBlockState
	if err := proto.Unmarshal(log.Data, &state); err != nil {
		f.logger.Error().Err(err).Msg("unmarshal block state")
		return err
	}
	if err := assertValid(f.state.Load(), &state); err != nil {
		return err
	}
	f.state.Store(&state)
	f.logger.Debug().Uint64("height", state.Height).Msg("received block state")

	if f.applyCh != nil {
		select {
		case f.applyCh <- RaftApplyMsg{Index: log.Index, State: &state}:
		default:
			// on a slow consumer, the raft cluster should not be blocked. Followers can sync from DA or other peers, too.
			f.logger.Warn().Msg("apply channel full, dropping message")
		}
	}

	return nil
}

// Snapshot implements raft.FSM
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{state: *f.state.Load()}, nil
}

// Restore implements raft.FSM
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}

	var state RaftBlockState
	if err := proto.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}

	f.state.Store(&state)
	f.logger.Info().Uint64("height", state.Height).Msg("restored from snapshot")
	return nil
}

type fsmSnapshot struct {
	state RaftBlockState
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		data, err := proto.Marshal(&s.state)
		if err != nil {
			return err
		}
		if _, err := sink.Write(data); err != nil {
			return err
		}
		return sink.Close()
	}()

	if err != nil {
		_ = sink.Cancel()
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

	nodeID, address := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	if nodeID == "" {
		return raft.Server{}, errors.New("nodeID cannot be empty")
	}
	if address == "" {
		return raft.Server{}, errors.New("address cannot be empty")
	}
	// we can skip address validation as they come from a local configuration

	return raft.Server{
		ID:      raft.ServerID(nodeID),
		Address: raft.ServerAddress(address),
	}, nil
}

func deduplicateServers(servers []raft.Server) []raft.Server {
	if len(servers) == 0 {
		return []raft.Server{}
	}
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
