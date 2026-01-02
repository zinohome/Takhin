// Copyright 2025 Takhin Data, Inc.

package raft

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestElectionTimeoutOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft election test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create optimized Raft config
	raftCfg := &config.RaftConfig{
		HeartbeatTimeoutMs:   1000,
		ElectionTimeoutMs:    3000,
		LeaderLeaseTimeoutMs: 500,
		CommitTimeoutMs:      50,
		SnapshotIntervalMs:   120000,
		SnapshotThreshold:    8192,
		PreVoteEnabled:       true,
		MaxAppendEntries:     64,
	}

	// Create Raft node with optimized config
	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
		RaftCfg:   raftCfg,
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Measure election time
	startTime := time.Now()

	// Wait for leader election with timeout
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	elected := false
	var electionTime time.Duration

	for !elected {
		select {
		case <-timeout:
			t.Fatal("leader election took longer than 5 seconds")
		case <-ticker.C:
			if node.IsLeader() {
				electionTime = time.Since(startTime)
				elected = true
			}
		}
	}

	// Verify election time is less than 5 seconds
	assert.True(t, electionTime < 5*time.Second,
		"election time %v should be less than 5s", electionTime)

	t.Logf("Leader elected in %v (target: < 5s)", electionTime)

	// Verify configuration is applied
	assert.Equal(t, time.Duration(3000)*time.Millisecond, node.GetElectionTimeout())
	assert.Equal(t, time.Duration(1000)*time.Millisecond, node.GetHeartbeatTimeout())
}

func TestPreVoteEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft prevote test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Test with PreVote enabled
	raftCfg := &config.RaftConfig{
		HeartbeatTimeoutMs:   1000,
		ElectionTimeoutMs:    3000,
		LeaderLeaseTimeoutMs: 500,
		CommitTimeoutMs:      50,
		SnapshotIntervalMs:   120000,
		SnapshotThreshold:    8192,
		PreVoteEnabled:       true,
		MaxAppendEntries:     64,
	}

	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
		RaftCfg:   raftCfg,
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Wait for leader election
	time.Sleep(3 * time.Second)

	// Node should become leader
	assert.True(t, node.IsLeader(), "node should be leader with PreVote enabled")
}

func TestPreVoteDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft prevote test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Test with PreVote disabled
	raftCfg := &config.RaftConfig{
		HeartbeatTimeoutMs:   1000,
		ElectionTimeoutMs:    3000,
		LeaderLeaseTimeoutMs: 500,
		CommitTimeoutMs:      50,
		SnapshotIntervalMs:   120000,
		SnapshotThreshold:    8192,
		PreVoteEnabled:       false,
		MaxAppendEntries:     64,
	}

	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
		RaftCfg:   raftCfg,
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Wait for leader election
	time.Sleep(3 * time.Second)

	// Node should still become leader (single node cluster)
	assert.True(t, node.IsLeader(), "node should be leader even with PreVote disabled")
}

func TestDefaultRaftConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft config test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Test with nil RaftCfg (should use defaults)
	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
		RaftCfg:   nil,
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Should use default timeouts
	assert.Equal(t, 3*time.Second, node.GetElectionTimeout())
	assert.Equal(t, 1*time.Second, node.GetHeartbeatTimeout())
}

func TestElectionMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft metrics test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create Raft node
	raftCfg := &config.RaftConfig{
		HeartbeatTimeoutMs:   1000,
		ElectionTimeoutMs:    3000,
		LeaderLeaseTimeoutMs: 500,
		CommitTimeoutMs:      50,
		SnapshotIntervalMs:   120000,
		SnapshotThreshold:    8192,
		PreVoteEnabled:       true,
		MaxAppendEntries:     64,
	}

	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
		RaftCfg:   raftCfg,
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Wait for leader election
	time.Sleep(3 * time.Second)

	// Verify node is leader
	assert.True(t, node.IsLeader())

	// Note: Metrics are updated asynchronously via the monitoring goroutine
	// In a real test environment, you would query the metrics registry
	// For now, we just verify the node functions correctly
}

func BenchmarkLeaderElection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Create topic manager
		dir := b.TempDir()
		topicMgr := topic.NewManager(dir, 1024*1024)

		// Create optimized Raft config
		raftCfg := &config.RaftConfig{
			HeartbeatTimeoutMs:   1000,
			ElectionTimeoutMs:    3000,
			LeaderLeaseTimeoutMs: 500,
			CommitTimeoutMs:      50,
			SnapshotIntervalMs:   120000,
			SnapshotThreshold:    8192,
			PreVoteEnabled:       true,
			MaxAppendEntries:     64,
		}

		raftDir := b.TempDir()
		cfg := &Config{
			NodeID:    "node1",
			RaftDir:   raftDir,
			RaftBind:  "127.0.0.1:0",
			Bootstrap: true,
			Peers:     []string{},
			RaftCfg:   raftCfg,
		}

		node, err := NewNode(cfg, topicMgr)
		if err != nil {
			b.Fatal(err)
		}

		// Wait for leader election
		timeout := time.After(5 * time.Second)
		ticker := time.NewTicker(50 * time.Millisecond)

	waitLoop:
		for {
			select {
			case <-timeout:
				b.Fatal("election timeout")
			case <-ticker.C:
				if node.IsLeader() {
					break waitLoop
				}
			}
		}

		ticker.Stop()
		node.Shutdown()
		topicMgr.Close()
	}
}
