// Copyright 2025 Takhin Data, Inc.

package raft

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// TestThreeNodeCluster tests a 3-node Raft cluster
func TestThreeNodeCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cluster test in short mode")
	}

	// Create 3 nodes
	nodes := make([]*Node, 3)
	topicMgrs := make([]*topic.Manager, 3)

	// Node configurations
	configs := []*Config{
		{
			NodeID:    "node1",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17001",
			Bootstrap: true,
			Peers:     []string{},
		},
		{
			NodeID:    "node2",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17002",
			Bootstrap: false,
			Peers:     []string{"127.0.0.1:17001"},
		},
		{
			NodeID:    "node3",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17003",
			Bootstrap: false,
			Peers:     []string{"127.0.0.1:17001"},
		},
	}

	// Create and start all nodes
	for i := 0; i < 3; i++ {
		topicMgrs[i] = topic.NewManager(t.TempDir(), 1024*1024)
		defer topicMgrs[i].Close()

		node, err := NewNode(configs[i], topicMgrs[i])
		require.NoError(t, err, "failed to create node%d", i+1)
		defer node.Shutdown()
		nodes[i] = node
	}

	// Wait for bootstrap node to become leader
	t.Log("Waiting for node1 to become leader...")
	time.Sleep(3 * time.Second)
	require.True(t, nodes[0].IsLeader(), "node1 should be leader")

	// Add node2 and node3 as voters
	t.Log("Adding node2 and node3 to cluster...")
	err := nodes[0].AddVoter("node2", "127.0.0.1:17002")
	require.NoError(t, err, "failed to add node2")

	err = nodes[0].AddVoter("node3", "127.0.0.1:17003")
	require.NoError(t, err, "failed to add node3")

	// Wait for cluster to stabilize
	time.Sleep(2 * time.Second)

	// Test 1: Create topic through leader
	t.Log("Test 1: Creating topic through leader...")
	err = nodes[0].CreateTopic("test-topic", 3, 5*time.Second)
	require.NoError(t, err, "leader should create topic")

	// Wait for replication
	time.Sleep(1 * time.Second)

	// Verify topic exists on all nodes
	for i, mgr := range topicMgrs {
		_, exists := mgr.GetTopic("test-topic")
		assert.True(t, exists, "topic should exist on node%d", i+1)
	}

	// Test 2: Write messages through leader
	t.Log("Test 2: Writing messages through leader...")
	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("message-%d", i)
		result, err := nodes[0].AppendMessage("test-topic", 0, nil, []byte(msg), 5*time.Second)
		require.NoError(t, err, "leader should append message %d", i)

		offset, ok := result.(int64)
		require.True(t, ok, "result should be int64")
		assert.Equal(t, int64(i), offset, "offset should match")
	}

	// Wait for replication
	time.Sleep(2 * time.Second)

	// Test 3: Verify data consistency across all nodes
	t.Log("Test 3: Verifying data consistency...")
	for i, mgr := range topicMgrs {
		tp, exists := mgr.GetTopic("test-topic")
		require.True(t, exists, "topic should exist on node%d", i+1)

		hwm, err := tp.HighWaterMark(0)
		require.NoError(t, err)
		assert.Equal(t, int64(10), hwm, "node%d should have 10 messages", i+1)

		// Verify message content
		for j := int64(0); j < 10; j++ {
			record, err := tp.Read(0, j)
			require.NoError(t, err, "node%d should read message %d", i+1, j)
			expected := fmt.Sprintf("message-%d", j)
			assert.Equal(t, []byte(expected), record.Value, "message content should match on node%d", i+1)
		}
	}

	t.Log("✅ All 3 nodes have consistent data")
}

// TestLeaderFailover tests leader failover
func TestLeaderFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cluster test in short mode")
	}

	// Create 3 nodes
	nodes := make([]*Node, 3)
	topicMgrs := make([]*topic.Manager, 3)

	configs := []*Config{
		{
			NodeID:    "node1",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17011",
			Bootstrap: true,
			Peers:     []string{},
		},
		{
			NodeID:    "node2",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17012",
			Bootstrap: false,
			Peers:     []string{"127.0.0.1:17011"},
		},
		{
			NodeID:    "node3",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17013",
			Bootstrap: false,
			Peers:     []string{"127.0.0.1:17011"},
		},
	}

	// Create and start all nodes
	for i := 0; i < 3; i++ {
		topicMgrs[i] = topic.NewManager(t.TempDir(), 1024*1024)
		defer topicMgrs[i].Close()

		node, err := NewNode(configs[i], topicMgrs[i])
		require.NoError(t, err)
		defer node.Shutdown()
		nodes[i] = node
	}

	// Setup cluster
	time.Sleep(3 * time.Second)
	require.True(t, nodes[0].IsLeader())

	err := nodes[0].AddVoter("node2", "127.0.0.1:17012")
	require.NoError(t, err)
	err = nodes[0].AddVoter("node3", "127.0.0.1:17013")
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Write some data
	t.Log("Writing initial data...")
	err = nodes[0].CreateTopic("test-failover", 1, 5*time.Second)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := nodes[0].AppendMessage("test-failover", 0, nil, []byte(fmt.Sprintf("before-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}

	time.Sleep(2 * time.Second)

	// Shutdown leader (node1)
	t.Log("Shutting down leader (node1)...")
	err = nodes[0].Shutdown()
	require.NoError(t, err)

	// Wait for new leader election
	t.Log("Waiting for new leader election...")
	time.Sleep(5 * time.Second)

	// Find new leader
	var newLeader *Node
	var newLeaderIdx int
	for i := 1; i < 3; i++ {
		if nodes[i].IsLeader() {
			newLeader = nodes[i]
			newLeaderIdx = i
			break
		}
	}
	require.NotNil(t, newLeader, "a new leader should be elected")
	t.Logf("New leader is node%d", newLeaderIdx+1)

	// Write data through new leader
	t.Log("Writing data through new leader...")
	for i := 0; i < 5; i++ {
		_, err := newLeader.AppendMessage("test-failover", 0, nil, []byte(fmt.Sprintf("after-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}

	time.Sleep(2 * time.Second)

	// Verify data on remaining nodes
	t.Log("Verifying data consistency on remaining nodes...")
	for i := 1; i < 3; i++ {
		tp, exists := topicMgrs[i].GetTopic("test-failover")
		require.True(t, exists, "topic should exist on node%d", i+1)

		hwm, err := tp.HighWaterMark(0)
		require.NoError(t, err)
		assert.Equal(t, int64(10), hwm, "node%d should have 10 messages", i+1)
	}

	t.Log("✅ Leader failover successful, data consistent")
}

// TestNetworkPartition tests behavior during network partition
func TestNetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cluster test in short mode")
	}

	// Create 3 nodes
	nodes := make([]*Node, 3)
	topicMgrs := make([]*topic.Manager, 3)

	configs := []*Config{
		{
			NodeID:    "node1",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17021",
			Bootstrap: true,
			Peers:     []string{},
		},
		{
			NodeID:    "node2",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17022",
			Bootstrap: false,
			Peers:     []string{"127.0.0.1:17021"},
		},
		{
			NodeID:    "node3",
			RaftDir:   t.TempDir(),
			RaftBind:  "127.0.0.1:17023",
			Bootstrap: false,
			Peers:     []string{"127.0.0.1:17021"},
		},
	}

	for i := 0; i < 3; i++ {
		topicMgrs[i] = topic.NewManager(t.TempDir(), 1024*1024)
		defer topicMgrs[i].Close()

		node, err := NewNode(configs[i], topicMgrs[i])
		require.NoError(t, err)
		defer node.Shutdown()
		nodes[i] = node
	}

	// Setup cluster
	time.Sleep(3 * time.Second)
	err := nodes[0].AddVoter("node2", "127.0.0.1:17022")
	require.NoError(t, err)
	err = nodes[0].AddVoter("node3", "127.0.0.1:17023")
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// Write initial data
	t.Log("Writing initial data...")
	err = nodes[0].CreateTopic("test-partition", 1, 5*time.Second)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err := nodes[0].AppendMessage("test-partition", 0, nil, []byte(fmt.Sprintf("msg-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	// Simulate partition by shutting down node3
	t.Log("Simulating network partition (shutting down node3)...")
	err = nodes[2].Shutdown()
	require.NoError(t, err)

	// Continue writing to majority partition (node1, node2)
	t.Log("Writing to majority partition...")
	for i := 5; i < 10; i++ {
		_, err := nodes[0].AppendMessage("test-partition", 0, nil, []byte(fmt.Sprintf("msg-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	// Verify majority partition has all messages
	t.Log("Verifying majority partition...")
	for i := 0; i < 2; i++ {
		tp, exists := topicMgrs[i].GetTopic("test-partition")
		require.True(t, exists)

		hwm, err := tp.HighWaterMark(0)
		require.NoError(t, err)
		assert.Equal(t, int64(10), hwm, "node%d should have 10 messages", i+1)
	}

	t.Log("✅ Cluster continues operating with majority")
}
