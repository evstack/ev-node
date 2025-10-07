package raft

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/etcd/server/v3/etcdserver/api/rafthttp"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	stats "go.etcd.io/etcd/server/v3/etcdserver/api/v2stats"
	"go.etcd.io/etcd/server/v3/storage/wal"
	"go.etcd.io/etcd/server/v3/storage/wal/walpb"
	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
)

var _ rafthttp.Raft = &Node{}

// Node represents a raft node with leader election capabilities
type Node struct {
	id     uint64
	peers  []string
	logger zerolog.Logger

	proposeC    chan []byte
	confChangeC chan raftpb.ConfChange

	commitC   chan *Commit
	errorC    chan error
	snapshotC chan struct{}

	node        raft.Node
	raftStorage *raft.MemoryStorage
	wal         *wal.WAL
	snapshotter *snap.Snapshotter

	snapCount   uint64
	transport   *rafthttp.Transport
	stopc       chan struct{}
	httpstopc   chan struct{}
	httpdonec   chan struct{}
	appliedIdx  uint64
	confState   raftpb.ConfState
	snapshotIdx uint64

	stateMachine StateMachine

	mu     sync.RWMutex
	leader uint64
}

// StateMachine defines the interface for raft state machine
type StateMachine interface {
	Apply(data []byte) error
	Snapshot() ([]byte, error)
	Restore(snapshot []byte) error
}

// Commit represents a committed raft entry
type Commit struct {
	Data  []byte
	Index uint64
}

// Config contains raft node configuration
type Config struct {
	ID              uint64
	Peers           []string
	RaftDir         string
	RaftAddr        string
	SnapCount       uint64
	Logger          zerolog.Logger
	StateMachine    StateMachine
	ExistingCluster bool
}

// NewNode creates a new raft node
func NewNode(cfg Config) (*Node, error) {
	if cfg.SnapCount == 0 {
		cfg.SnapCount = 10000
	}

	commitC := make(chan *Commit)
	errorC := make(chan error)

	rn := &Node{
		id:           cfg.ID,
		peers:        cfg.Peers,
		logger:       cfg.Logger.With().Str("component", "raft").Uint64("node_id", cfg.ID).Logger(),
		proposeC:     make(chan []byte),
		confChangeC:  make(chan raftpb.ConfChange),
		commitC:      commitC,
		errorC:       errorC,
		snapshotC:    make(chan struct{}),
		snapCount:    cfg.SnapCount,
		stopc:        make(chan struct{}),
		httpstopc:    make(chan struct{}),
		httpdonec:    make(chan struct{}),
		stateMachine: cfg.StateMachine,
	}

	snapdir := filepath.Join(cfg.RaftDir, "snap")
	if err := os.MkdirAll(snapdir, 0750); err != nil {
		return nil, fmt.Errorf("create snap dir: %w", err)
	}
	rn.snapshotter = snap.New(cfg.Logger, snapdir)

	waldir := filepath.Join(cfg.RaftDir, "wal")
	if !wal.Exist(waldir) {
		if err := os.MkdirAll(waldir, 0750); err != nil {
			return nil, fmt.Errorf("create wal dir: %w", err)
		}

		w, err := wal.Create(cfg.Logger, waldir, nil)
		if err != nil {
			return nil, fmt.Errorf("create wal: %w", err)
		}
		w.Close()
	}

	walsnap := walpb.Snapshot{}
	w, err := wal.Open(cfg.Logger, waldir, walsnap)
	if err != nil {
		return nil, fmt.Errorf("open wal: %w", err)
	}
	rn.wal = w

	rn.raftStorage = raft.NewMemoryStorage()

	if err := rn.replayWAL(); err != nil {
		return nil, fmt.Errorf("replay wal: %w", err)
	}

	c := &raft.Config{
		ID:              cfg.ID,
		ElectionTick:    10,
		HeartbeatTick:   1,
		Storage:         rn.raftStorage,
		MaxSizePerMsg:   1024 * 1024,
		MaxInflightMsgs: 256,
		Logger:          &raftLogger{logger: cfg.Logger},
	}

	var startPeers []raft.Peer
	if !cfg.ExistingCluster {
		for i, peer := range cfg.Peers {
			startPeers = append(startPeers, raft.Peer{ID: uint64(i + 1)})
		}
		rn.node = raft.StartNode(c, startPeers)
	} else {
		rn.node = raft.RestartNode(c)
	}

	rn.transport = &rafthttp.Transport{
		Logger:      cfg.Logger,
		ID:          types.ID(cfg.ID),
		ClusterID:   0x1000,
		Raft:        rn,
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(cfg.Logger, fmt.Sprintf("%d", cfg.ID)),
		ErrorC:      errorC,
	}

	rn.transport.Start()

	for i, peer := range cfg.Peers {
		if uint64(i+1) != cfg.ID {
			rn.transport.AddPeer(raftpb.PeerID(i+1), []string{peer})
		}
	}

	go rn.serveRaft(cfg.RaftAddr)
	go rn.serveChannels()

	return rn, nil
}

// Start starts the raft node
func (rn *Node) Start(ctx context.Context) error {
	rn.logger.Info().Msg("raft node started")
	return nil
}

// Stop stops the raft node
func (rn *Node) Stop() error {
	rn.logger.Info().Msg("stopping raft node")
	close(rn.stopc)

	rn.transport.Stop()
	close(rn.httpstopc)
	<-rn.httpdonec

	if rn.node != nil {
		rn.node.Stop()
	}

	if rn.wal != nil {
		rn.wal.Close()
	}

	rn.logger.Info().Msg("raft node stopped")
	return nil
}

// IsLeader returns true if this node is the current leader
func (rn *Node) IsLeader() bool {
	rn.mu.RLock()
	defer rn.mu.RUnlock()
	return rn.leader == rn.id
}

// LeaderID returns the current leader ID
func (rn *Node) LeaderID() uint64 {
	rn.mu.RLock()
	defer rn.mu.RUnlock()
	return rn.leader
}

// Propose proposes data to be committed via raft
func (rn *Node) Propose(ctx context.Context, data []byte) error {
	select {
	case rn.proposeC <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-rn.stopc:
		return fmt.Errorf("raft node stopped")
	}
}

// Process implements rafthttp.Raft interface
func (rn *Node) Process(ctx context.Context, m raftpb.Message) error {
	return rn.node.Step(ctx, m)
}

// IsIDRemoved implements rafthttp.Raft interface
func (rn *Node) IsIDRemoved(id uint64) bool {
	return false
}

// ReportUnreachable implements rafthttp.Raft interface
func (rn *Node) ReportUnreachable(id uint64) {
	rn.node.ReportUnreachable(id)
}

// ReportSnapshot implements rafthttp.Raft interface
func (rn *Node) ReportSnapshot(id uint64, status raft.SnapshotStatus) {
	rn.node.ReportSnapshot(id, status)
}

func (rn *Node) serveRaft(addr string) {
	url, err := url.Parse(addr)
	if err != nil {
		rn.logger.Fatal().Err(err).Msg("parse raft address")
	}

	ln, err := net.Listen("tcp", url.Host)
	if err != nil {
		rn.logger.Fatal().Err(err).Msg("listen raft address")
	}

	rn.logger.Info().Str("addr", addr).Msg("raft transport listening")

	err = (&http.Server{Handler: rn.transport.Handler()}).Serve(ln)
	select {
	case <-rn.httpstopc:
	default:
		rn.logger.Fatal().Err(err).Msg("serve raft transport")
	}
	close(rn.httpdonec)
}

func (rn *Node) serveChannels() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rn.node.Tick()

		case rd := <-rn.node.Ready():
			if !raft.IsEmptySnap(rd.Snapshot) {
				rn.saveSnap(rd.Snapshot)
				rn.raftStorage.ApplySnapshot(rd.Snapshot)
				rn.publishSnapshot(rd.Snapshot)
			}

			rn.wal.Save(rd.HardState, rd.Entries)

			if !raft.IsEmptySnap(rd.Snapshot) {
				rn.raftStorage.ApplySnapshot(rd.Snapshot)
			}
			rn.raftStorage.Append(rd.Entries)

			rn.transport.Send(rd.Messages)

			if ok := rn.publishEntries(rn.entriesToApply(rd.CommittedEntries)); !ok {
				rn.Stop()
				return
			}

			rn.maybeTriggerSnapshot()

			// Update leader
			if rd.SoftState != nil {
				rn.mu.Lock()
				oldLeader := rn.leader
				rn.leader = rd.SoftState.Lead
				if oldLeader != rn.leader {
					if rn.leader == rn.id {
						rn.logger.Info().Msg("became leader")
					} else {
						rn.logger.Info().Uint64("leader", rn.leader).Msg("new leader elected")
					}
				}
				rn.mu.Unlock()
			}

			rn.node.Advance()

		case err := <-rn.transport.ErrorC:
			rn.logger.Error().Err(err).Msg("raft transport error")
			return

		case <-rn.stopc:
			return
		}
	}
}

func (rn *Node) entriesToApply(ents []raftpb.Entry) []raftpb.Entry {
	if len(ents) == 0 {
		return nil
	}
	firstIdx := ents[0].Index
	if firstIdx > rn.appliedIdx+1 {
		rn.logger.Fatal().
			Uint64("first_index", firstIdx).
			Uint64("applied_index", rn.appliedIdx).
			Msg("first index of committed entry > applied index + 1")
	}
	if rn.appliedIdx-firstIdx+1 < uint64(len(ents)) {
		return ents[rn.appliedIdx-firstIdx+1:]
	}
	return nil
}

func (rn *Node) publishEntries(ents []raftpb.Entry) bool {
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				break
			}

			if err := rn.stateMachine.Apply(ents[i].Data); err != nil {
				rn.logger.Error().Err(err).Msg("apply to state machine")
				return false
			}

			select {
			case rn.commitC <- &Commit{Data: ents[i].Data, Index: ents[i].Index}:
			case <-rn.stopc:
				return false
			}

		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			cc.Unmarshal(ents[i].Data)
			rn.confState = *rn.node.ApplyConfChange(cc)
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				if len(cc.Context) > 0 {
					rn.transport.AddPeer(raftpb.PeerID(cc.NodeID), []string{string(cc.Context)})
				}
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == rn.id {
					rn.logger.Info().Msg("removed from cluster")
					return false
				}
				rn.transport.RemovePeer(raftpb.PeerID(cc.NodeID))
			}
		}

		rn.appliedIdx = ents[i].Index
	}
	return true
}

func (rn *Node) maybeTriggerSnapshot() {
	if rn.appliedIdx-rn.snapshotIdx <= rn.snapCount {
		return
	}

	rn.logger.Info().Uint64("applied", rn.appliedIdx).Uint64("last_snap", rn.snapshotIdx).Msg("triggering snapshot")

	data, err := rn.stateMachine.Snapshot()
	if err != nil {
		rn.logger.Error().Err(err).Msg("snapshot state machine")
		return
	}

	snap, err := rn.raftStorage.CreateSnapshot(rn.appliedIdx, &rn.confState, data)
	if err != nil {
		panic(err)
	}
	if err := rn.saveSnap(snap); err != nil {
		panic(err)
	}

	compactIdx := uint64(1)
	if rn.appliedIdx > rn.snapCount {
		compactIdx = rn.appliedIdx - rn.snapCount
	}
	if err := rn.raftStorage.Compact(compactIdx); err != nil {
		panic(err)
	}

	rn.logger.Info().Uint64("index", compactIdx).Msg("compacted log")
	rn.snapshotIdx = rn.appliedIdx
}

func (rn *Node) saveSnap(snap raftpb.Snapshot) error {
	walSnap := walpb.Snapshot{
		Index: snap.Metadata.Index,
		Term:  snap.Metadata.Term,
	}
	if err := rn.wal.SaveSnapshot(walSnap); err != nil {
		return err
	}
	if err := rn.snapshotter.SaveSnap(snap); err != nil {
		return err
	}
	return rn.wal.ReleaseLockTo(snap.Metadata.Index)
}

func (rn *Node) publishSnapshot(snap raftpb.Snapshot) {
	if raft.IsEmptySnap(snap) {
		return
	}

	rn.logger.Info().
		Uint64("index", snap.Metadata.Index).
		Uint64("term", snap.Metadata.Term).
		Msg("publishing snapshot")

	if snap.Metadata.Index <= rn.appliedIdx {
		rn.logger.Fatal().
			Uint64("snap_index", snap.Metadata.Index).
			Uint64("applied_index", rn.appliedIdx).
			Msg("snapshot index <= applied index")
	}

	if err := rn.stateMachine.Restore(snap.Data); err != nil {
		rn.logger.Fatal().Err(err).Msg("restore snapshot")
	}

	rn.confState = snap.Metadata.ConfState
	rn.snapshotIdx = snap.Metadata.Index
	rn.appliedIdx = snap.Metadata.Index
}

func (rn *Node) replayWAL() error {
	snapshot, err := rn.snapshotter.Load()
	if err != nil && err != snap.ErrNoSnapshot {
		return fmt.Errorf("load snapshot: %w", err)
	}

	if snapshot != nil {
		if err := rn.raftStorage.ApplySnapshot(*snapshot); err != nil {
			return fmt.Errorf("apply snapshot: %w", err)
		}
	}

	walsnap := walpb.Snapshot{}
	if snapshot != nil {
		walsnap.Index = snapshot.Metadata.Index
		walsnap.Term = snapshot.Metadata.Term
		rn.snapshotIdx = snapshot.Metadata.Index
		rn.appliedIdx = snapshot.Metadata.Index
	}

	w, err := wal.Open(rn.logger, filepath.Join(filepath.Dir(rn.wal.Dir()), "wal"), walsnap)
	if err != nil {
		return fmt.Errorf("open wal for replay: %w", err)
	}
	rn.wal = w

	_, st, ents, err := rn.wal.ReadAll()
	if err != nil {
		return fmt.Errorf("read wal: %w", err)
	}

	if snapshot != nil {
		if err := rn.stateMachine.Restore(snapshot.Data); err != nil {
			return fmt.Errorf("restore snapshot: %w", err)
		}
	}

	rn.raftStorage.SetHardState(st)
	rn.raftStorage.Append(ents)

	return nil
}

// GetStateMachine returns the underlying state machine
func (rn *Node) GetStateMachine() interface{} {
	return rn.stateMachine
}

// raftLogger adapts zerolog to raft.Logger
type raftLogger struct {
	logger zerolog.Logger
}

func (l *raftLogger) Debug(v ...interface{}) {
	l.logger.Debug().Msg(fmt.Sprint(v...))
}

func (l *raftLogger) Debugf(format string, v ...interface{}) {
	l.logger.Debug().Msgf(format, v...)
}

func (l *raftLogger) Error(v ...interface{}) {
	l.logger.Error().Msg(fmt.Sprint(v...))
}

func (l *raftLogger) Errorf(format string, v ...interface{}) {
	l.logger.Error().Msgf(format, v...)
}

func (l *raftLogger) Info(v ...interface{}) {
	l.logger.Info().Msg(fmt.Sprint(v...))
}

func (l *raftLogger) Infof(format string, v ...interface{}) {
	l.logger.Info().Msgf(format, v...)
}

func (l *raftLogger) Warning(v ...interface{}) {
	l.logger.Warn().Msg(fmt.Sprint(v...))
}

func (l *raftLogger) Warningf(format string, v ...interface{}) {
	l.logger.Warn().Msgf(format, v...)
}

func (l *raftLogger) Fatal(v ...interface{}) {
	l.logger.Fatal().Msg(fmt.Sprint(v...))
}

func (l *raftLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatal().Msgf(format, v...)
}

func (l *raftLogger) Panic(v ...interface{}) {
	l.logger.Panic().Msg(fmt.Sprint(v...))
}

func (l *raftLogger) Panicf(format string, v ...interface{}) {
	l.logger.Panic().Msgf(format, v...)
}
