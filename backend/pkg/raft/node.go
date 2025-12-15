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
	"github.com/takhin-data/takhin/pkg/logger"
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
}

// Config holds the configuration for a Raft node
type Config struct {
	NodeID    string
	RaftDir   string
	RaftBind  string
	Bootstrap bool
	Peers     []string
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
	}

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
