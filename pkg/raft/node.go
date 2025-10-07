package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/zerolog"

	"github.com/evstack/ev-node/block"
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
	NodeID      string
	RaftAddr    string
	RaftDir     string
	Bootstrap   bool
	Peers       []string
	SnapCount   uint64
	Logger      zerolog.Logger
	SendTimeout time.Duration
}

// FSM implements raft.FSM for block state
type FSM struct {
	logger  zerolog.Logger
	state   *block.RaftBlockState
	applyCh chan<- block.RaftApplyMsg
}

// NewNode creates a new raft node
func NewNode(ctx context.Context, cfg *Config) (*Node, error) {
	if err := os.MkdirAll(cfg.RaftDir, 0755); err != nil {
		return nil, fmt.Errorf("create raft dir: %w", err)
	}

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.NodeID)
	raftConfig.LogLevel = "INFO"

	fsm := &FSM{
		logger: cfg.Logger.With().Str("component", "raft-fsm").Logger(),
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
		raft:   r,
		fsm:    fsm,
		config: cfg,
		logger: cfg.Logger.With().Str("component", "raft-node").Logger(),
	}

	if cfg.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(cfg.NodeID),
					Address: raft.ServerAddress(cfg.RaftAddr),
				},
			},
		}
		r.BootstrapCluster(configuration)
		node.logger.Info().Msg("bootstrapped raft cluster")
	}
	// todo: start listening for leader changes
	return node, nil
}

// IsLeader returns true if this node is the raft leader
func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}

// ProposeBlock proposes a block state to be replicated via raft
func (n *Node) ProposeBlock(ctx context.Context, state *block.RaftBlockState) error {
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
func (n *Node) AddPeer(id, addr string) error {
	future := n.raft.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), 0, 0)
	return future.Error()
}

// RemovePeer removes a peer from the raft cluster
func (n *Node) RemovePeer(id string) error {
	future := n.raft.RemoveServer(raft.ServerID(id), 0, 0)
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
	f.logger.Debug().Uint64("height", state.Height).Msg("applied block state")

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
