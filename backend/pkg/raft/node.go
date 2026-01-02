// Copyright 2025 Takhin Data, Inc.

package raft

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/metrics"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// Node represents a Raft node
type Node struct {
	raft          *raft.Raft
	fsm           *FSM
	config        *Config
	transport     *raft.NetworkTransport
	logStore      *raftboltdb.BoltStore
	stableStore   *raftboltdb.BoltStore
	snapshotStore raft.SnapshotStore
	logger        *logger.Logger
	notifyCh      chan bool
	lastState     raft.RaftState
	electionStart time.Time
}

// Config holds the configuration for a Raft node
type Config struct {
	NodeID    string
	RaftDir   string
	RaftBind  string
	Bootstrap bool
	Peers     []string
	RaftCfg   *config.RaftConfig
}

// NewNode creates a new Raft node
func NewNode(cfg *Config, topicManager *topic.Manager) (*Node, error) {
	// Create raft directory
	if err := os.MkdirAll(cfg.RaftDir, 0755); err != nil {
		return nil, fmt.Errorf("create raft directory: %w", err)
	}

	// Create FSM
	fsm := NewFSM(topicManager)

	// Setup Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.NodeID)

	// Apply optimized election settings from config
	if cfg.RaftCfg != nil {
		raftConfig.HeartbeatTimeout = time.Duration(cfg.RaftCfg.HeartbeatTimeoutMs) * time.Millisecond
		raftConfig.ElectionTimeout = time.Duration(cfg.RaftCfg.ElectionTimeoutMs) * time.Millisecond
		raftConfig.LeaderLeaseTimeout = time.Duration(cfg.RaftCfg.LeaderLeaseTimeoutMs) * time.Millisecond
		raftConfig.CommitTimeout = time.Duration(cfg.RaftCfg.CommitTimeoutMs) * time.Millisecond
		raftConfig.SnapshotInterval = time.Duration(cfg.RaftCfg.SnapshotIntervalMs) * time.Millisecond
		raftConfig.SnapshotThreshold = uint64(cfg.RaftCfg.SnapshotThreshold)
		raftConfig.MaxAppendEntries = cfg.RaftCfg.MaxAppendEntries

		// Enable PreVote to reduce unnecessary elections
		// PreVoteDisabled=false means PreVote is enabled
		raftConfig.PreVoteDisabled = !cfg.RaftCfg.PreVoteEnabled
	}

	// Create notification channel for leadership changes
	notifyCh := make(chan bool, 10)
	raftConfig.NotifyCh = notifyCh

	// Create logger
	log := logger.Default().WithComponent("raft")

	// Setup log store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.RaftDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("create log store: %w", err)
	}

	// Setup stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.RaftDir, "raft-stable.db"))
	if err != nil {
		logStore.Close()
		return nil, fmt.Errorf("create stable store: %w", err)
	}

	// Create snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(cfg.RaftDir, 3, os.Stderr)
	if err != nil {
		logStore.Close()
		stableStore.Close()
		return nil, fmt.Errorf("create snapshot store: %w", err)
	}

	// Setup transport
	addr, err := net.ResolveTCPAddr("tcp", cfg.RaftBind)
	if err != nil {
		logStore.Close()
		stableStore.Close()
		return nil, fmt.Errorf("resolve raft bind address: %w", err)
	}

	transport, err := raft.NewTCPTransport(cfg.RaftBind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		logStore.Close()
		stableStore.Close()
		return nil, fmt.Errorf("create transport: %w", err)
	}

	// Create Raft instance
	r, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		transport.Close()
		logStore.Close()
		stableStore.Close()
		return nil, fmt.Errorf("create raft: %w", err)
	}

	node := &Node{
		raft:          r,
		fsm:           fsm,
		config:        cfg,
		transport:     transport,
		logStore:      logStore,
		stableStore:   stableStore,
		snapshotStore: snapshotStore,
		logger:        log,
		notifyCh:      notifyCh,
		lastState:     raft.Follower,
	}

	// Start monitoring leadership changes in background
	go node.monitorLeadership()

	// Bootstrap cluster if needed
	if cfg.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(cfg.NodeID),
					Address: transport.LocalAddr(),
				},
			},
		}
		future := r.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			node.logger.Error("failed to bootstrap cluster", "error", err)
		}
	}

	return node, nil
}

// GetFSM returns the FSM instance
func (n *Node) GetFSM() *FSM {
	return n.fsm
}

// IsLeader returns whether this node is the leader
func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}

// Leader returns the current leader address
func (n *Node) Leader() string {
	addr, _ := n.raft.LeaderWithID()
	return string(addr)
}

// Apply applies a command to the Raft log
func (n *Node) Apply(cmd Command, timeout time.Duration) (interface{}, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshal command: %w", err)
	}

	future := n.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		return nil, err
	}

	return future.Response(), nil
}

// CreateTopic creates a new topic through Raft
func (n *Node) CreateTopic(name string, numPartitions int32, timeout time.Duration) error {
	cmd := Command{
		Type:      CommandCreateTopic,
		TopicName: name,
		NumParts:  numPartitions,
	}

	_, err := n.Apply(cmd, timeout)
	return err
}

// DeleteTopic deletes a topic through Raft
func (n *Node) DeleteTopic(name string, timeout time.Duration) error {
	cmd := Command{
		Type:      CommandDeleteTopic,
		TopicName: name,
	}

	_, err := n.Apply(cmd, timeout)
	return err
}

// AppendMessage appends a message through Raft
func (n *Node) AppendMessage(topic string, partition int32, key, value []byte, timeout time.Duration) (interface{}, error) {
	cmd := Command{
		Type:      CommandAppend,
		TopicName: topic,
		Partition: partition,
		Key:       key,
		Value:     value,
	}

	return n.Apply(cmd, timeout)
}

// AddVoter adds a new voting member to the cluster
func (n *Node) AddVoter(id, address string) error {
	future := n.raft.AddVoter(raft.ServerID(id), raft.ServerAddress(address), 0, 0)
	return future.Error()
}

// RemoveServer removes a server from the cluster
func (n *Node) RemoveServer(id string) error {
	future := n.raft.RemoveServer(raft.ServerID(id), 0, 0)
	return future.Error()
}

// Stats returns Raft stats
func (n *Node) Stats() map[string]string {
	return n.raft.Stats()
}

// monitorLeadership monitors leadership changes and updates metrics
func (n *Node) monitorLeadership() {
	for isLeader := range n.notifyCh {
		currentState := n.raft.State()

		// Track state transitions
		if currentState != n.lastState {
			// Update state gauge
			switch currentState {
			case raft.Follower:
				metrics.RaftState.Set(0)
			case raft.Candidate:
				metrics.RaftState.Set(1)
				// Track election start
				n.electionStart = time.Now()
				metrics.RaftElectionsTotal.Inc()
				n.logger.Info("starting leader election")
			case raft.Leader:
				metrics.RaftState.Set(2)
				// Track election duration if we were candidate
				if n.lastState == raft.Candidate && !n.electionStart.IsZero() {
					duration := time.Since(n.electionStart).Seconds()
					metrics.RaftElectionDuration.Observe(duration)
					n.logger.Info("leader election completed", "duration_seconds", duration)
				}
			}

			// Track leader changes
			if (n.lastState == raft.Leader && currentState != raft.Leader) ||
				(n.lastState != raft.Leader && currentState == raft.Leader) {
				metrics.RaftLeaderChanges.Inc()
				n.logger.Info("leadership changed",
					"from", n.lastState.String(),
					"to", currentState.String(),
					"is_leader", isLeader)
			}

			n.lastState = currentState
		}
	}
}

// GetElectionTimeout returns the configured election timeout
func (n *Node) GetElectionTimeout() time.Duration {
	if n.config.RaftCfg != nil {
		return time.Duration(n.config.RaftCfg.ElectionTimeoutMs) * time.Millisecond
	}
	return 3 * time.Second // default
}

// GetHeartbeatTimeout returns the configured heartbeat timeout
func (n *Node) GetHeartbeatTimeout() time.Duration {
	if n.config.RaftCfg != nil {
		return time.Duration(n.config.RaftCfg.HeartbeatTimeoutMs) * time.Millisecond
	}
	return 1 * time.Second // default
}

// Shutdown closes the Raft node
func (n *Node) Shutdown() error {
	n.logger.Info("shutting down raft node")

	if err := n.raft.Shutdown().Error(); err != nil {
		n.logger.Error("failed to shutdown raft", "error", err)
	}

	if err := n.transport.Close(); err != nil {
		n.logger.Error("failed to close transport", "error", err)
	}

	if err := n.logStore.Close(); err != nil {
		n.logger.Error("failed to close log store", "error", err)
	}

	if err := n.stableStore.Close(); err != nil {
		n.logger.Error("failed to close stable store", "error", err)
	}

	return nil
}
