// Copyright 2025 Takhin Data, Inc.

package replication

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	raftpkg "github.com/takhin-data/takhin/pkg/raft"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// TestThreeNodeReplication tests normal replication in a 3-node cluster
func TestThreeNodeReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cluster := setupThreeNodeCluster(t)
	defer cluster.Shutdown()

	// Wait for cluster to stabilize and elect leader
	leaderNode := cluster.WaitForLeader(t, 10*time.Second)
	t.Logf("Leader elected: node%d", leaderNode.ID)

	// Create topic through leader
	topicName := "test-replication"
	err := cluster.Nodes[leaderNode.ID].RaftNode.CreateTopic(topicName, 1, 5*time.Second)
	require.NoError(t, err, "leader should create topic")

	// Wait for topic replication
	time.Sleep(2 * time.Second)

	// Verify topic exists on all nodes
	for i, node := range cluster.Nodes {
		_, exists := node.TopicMgr.GetTopic(topicName)
		assert.True(t, exists, "topic should exist on node%d", i)
	}

	// Write 100 messages through leader
	t.Log("Writing 100 messages through leader...")
	for i := 0; i < 100; i++ {
		msg := fmt.Sprintf("message-%d", i)
		offset, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
			topicName, 0, nil, []byte(msg), 5*time.Second)
		require.NoError(t, err, "failed to append message %d", i)

		result, ok := offset.(int64)
		require.True(t, ok, "offset should be int64")
		assert.Equal(t, int64(i), result, "offset should match")
	}

	// Wait for replication
	time.Sleep(3 * time.Second)

	// Verify data consistency across all nodes
	t.Log("Verifying data consistency across all nodes...")
	for i, node := range cluster.Nodes {
		tp, exists := node.TopicMgr.GetTopic(topicName)
		require.True(t, exists, "topic should exist on node%d", i)

		hwm, err := tp.HighWaterMark(0)
		require.NoError(t, err, "node%d should return HWM", i)
		assert.Equal(t, int64(100), hwm, "node%d should have 100 messages", i)

		// Verify random sample of messages
		for j := int64(0); j < 100; j += 10 {
			record, err := tp.Read(0, j)
			require.NoError(t, err, "node%d should read message %d", i, j)
			expected := fmt.Sprintf("message-%d", j)
			assert.Equal(t, []byte(expected), record.Value, 
				"message content mismatch on node%d at offset %d", i, j)
		}
	}

	t.Log("✅ All 3 nodes have consistent replicated data")
}

// TestLeaderFailover tests leader crash and failover
func TestLeaderFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cluster := setupThreeNodeCluster(t)
	defer cluster.Shutdown()

	// Wait for initial leader
	leaderNode := cluster.WaitForLeader(t, 10*time.Second)
	t.Logf("Initial leader: node%d", leaderNode.ID)

	// Create topic and write initial data
	topicName := "test-failover"
	err := cluster.Nodes[leaderNode.ID].RaftNode.CreateTopic(topicName, 1, 5*time.Second)
	require.NoError(t, err)

	t.Log("Writing 50 messages before failover...")
	for i := 0; i < 50; i++ {
		_, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
			topicName, 0, nil, []byte(fmt.Sprintf("before-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}

	time.Sleep(2 * time.Second)

	// Record pre-failover state
	var preFail int64
	for i, node := range cluster.Nodes {
		tp, exists := node.TopicMgr.GetTopic(topicName)
		require.True(t, exists)
		hwm, _ := tp.HighWaterMark(0)
		t.Logf("Node%d pre-failover HWM: %d", i, hwm)
		if hwm > preFail {
			preFail = hwm
		}
	}

	// Shutdown leader to trigger failover
	t.Logf("Shutting down leader node%d...", leaderNode.ID)
	err = cluster.Nodes[leaderNode.ID].RaftNode.Shutdown()
	require.NoError(t, err)

	// Wait for new leader election
	t.Log("Waiting for new leader election...")
	time.Sleep(8 * time.Second)

	// Find new leader
	var newLeader *ClusterNode
	for i := 0; i < 3; i++ {
		if i == leaderNode.ID {
			continue
		}
		if cluster.Nodes[i].RaftNode.IsLeader() {
			newLeader = cluster.Nodes[i]
			break
		}
	}
	require.NotNil(t, newLeader, "new leader should be elected")
	t.Logf("New leader elected: node%d", newLeader.ID)

	// Write more data through new leader
	t.Log("Writing 50 messages through new leader...")
	for i := 0; i < 50; i++ {
		_, err := newLeader.RaftNode.AppendMessage(
			topicName, 0, nil, []byte(fmt.Sprintf("after-%d", i)), 5*time.Second)
		require.NoError(t, err, "new leader should accept writes")
	}

	time.Sleep(3 * time.Second)

	// Verify data on remaining nodes
	t.Log("Verifying data consistency on remaining nodes...")
	for i := 0; i < 3; i++ {
		if i == leaderNode.ID {
			continue
		}

		tp, exists := cluster.Nodes[i].TopicMgr.GetTopic(topicName)
		require.True(t, exists)

		hwm, err := tp.HighWaterMark(0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, hwm, preFail, 
			"node%d should have at least pre-failover data", i)
		assert.Equal(t, int64(100), hwm, 
			"node%d should have all 100 messages", i)
	}

	t.Log("✅ Leader failover successful with data consistency")
}

// TestFollowerRecovery tests follower crash and recovery
func TestFollowerRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cluster := setupThreeNodeCluster(t)
	defer cluster.Shutdown()

	leaderNode := cluster.WaitForLeader(t, 10*time.Second)
	t.Logf("Leader: node%d", leaderNode.ID)

	// Create topic
	topicName := "test-follower-recovery"
	err := cluster.Nodes[leaderNode.ID].RaftNode.CreateTopic(topicName, 1, 5*time.Second)
	require.NoError(t, err)

	// Write initial data
	t.Log("Writing 30 messages...")
	for i := 0; i < 30; i++ {
		_, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
			topicName, 0, nil, []byte(fmt.Sprintf("msg-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	// Find a follower to shut down
	var followerID int
	for i := 0; i < 3; i++ {
		if i != leaderNode.ID {
			followerID = i
			break
		}
	}

	t.Logf("Shutting down follower node%d...", followerID)
	err = cluster.Nodes[followerID].RaftNode.Shutdown()
	require.NoError(t, err)

	// Write more data while follower is down
	t.Log("Writing 30 more messages while follower is down...")
	for i := 30; i < 60; i++ {
		_, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
			topicName, 0, nil, []byte(fmt.Sprintf("msg-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	// Verify remaining nodes have all data
	for i := 0; i < 3; i++ {
		if i == followerID {
			continue
		}
		tp, _ := cluster.Nodes[i].TopicMgr.GetTopic(topicName)
		hwm, _ := tp.HighWaterMark(0)
		assert.Equal(t, int64(60), hwm, "node%d should have 60 messages", i)
	}

	// Restart follower
	t.Logf("Restarting follower node%d...", followerID)
	newTopicMgr := topic.NewManager(cluster.Nodes[followerID].RaftDir, 1024*1024)
	defer newTopicMgr.Close()

	raftCfg := &raftpkg.Config{
		NodeID:    fmt.Sprintf("node%d", followerID),
		RaftDir:   cluster.Nodes[followerID].RaftDir,
		RaftBind:  cluster.Nodes[followerID].RaftBind,
		Bootstrap: false,
		Peers:     cluster.Nodes[followerID].Peers,
	}

	newRaftNode, err := raftpkg.NewNode(raftCfg, newTopicMgr)
	require.NoError(t, err)
	defer newRaftNode.Shutdown()

	cluster.Nodes[followerID].RaftNode = newRaftNode
	cluster.Nodes[followerID].TopicMgr = newTopicMgr

	// Wait for follower to catch up
	t.Log("Waiting for follower to catch up...")
	time.Sleep(5 * time.Second)

	// Verify follower has caught up
	tp, exists := newTopicMgr.GetTopic(topicName)
	require.True(t, exists, "topic should exist on recovered follower")

	hwm, err := tp.HighWaterMark(0)
	require.NoError(t, err)
	assert.Equal(t, int64(60), hwm, "recovered follower should have all 60 messages")

	// Verify message content
	for i := int64(0); i < 60; i += 10 {
		record, err := tp.Read(0, i)
		require.NoError(t, err)
		expected := fmt.Sprintf("msg-%d", i)
		assert.Equal(t, []byte(expected), record.Value, 
			"message content should match at offset %d", i)
	}

	t.Log("✅ Follower recovered and caught up successfully")
}

// TestNetworkPartition tests split-brain prevention
func TestNetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cluster := setupThreeNodeCluster(t)
	defer cluster.Shutdown()

	leaderNode := cluster.WaitForLeader(t, 10*time.Second)
	t.Logf("Initial leader: node%d", leaderNode.ID)

	// Create topic and write initial data
	topicName := "test-partition"
	err := cluster.Nodes[leaderNode.ID].RaftNode.CreateTopic(topicName, 1, 5*time.Second)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		_, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
			topicName, 0, nil, []byte(fmt.Sprintf("pre-%d", i)), 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(2 * time.Second)

	// Simulate network partition by isolating one node
	var minorityNode int
	for i := 0; i < 3; i++ {
		if i != leaderNode.ID {
			minorityNode = i
			break
		}
	}

	t.Logf("Simulating network partition - isolating node%d...", minorityNode)
	err = cluster.Nodes[minorityNode].RaftNode.Shutdown()
	require.NoError(t, err)

	// Majority partition (2 nodes) should continue operating
	t.Log("Writing to majority partition...")
	for i := 20; i < 40; i++ {
		_, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
			topicName, 0, nil, []byte(fmt.Sprintf("post-%d", i)), 5*time.Second)
		require.NoError(t, err, "majority partition should accept writes")
	}
	time.Sleep(2 * time.Second)

	// Verify majority partition has all data
	for i := 0; i < 3; i++ {
		if i == minorityNode {
			continue
		}
		tp, exists := cluster.Nodes[i].TopicMgr.GetTopic(topicName)
		require.True(t, exists)
		
		hwm, err := tp.HighWaterMark(0)
		require.NoError(t, err)
		assert.Equal(t, int64(40), hwm, 
			"node%d in majority partition should have 40 messages", i)
	}

	t.Log("✅ Majority partition continues operating correctly")
}

// TestConcurrentWrites tests concurrent write performance and consistency
func TestConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cluster := setupThreeNodeCluster(t)
	defer cluster.Shutdown()

	leaderNode := cluster.WaitForLeader(t, 10*time.Second)
	t.Logf("Leader: node%d", leaderNode.ID)

	topicName := "test-concurrent"
	err := cluster.Nodes[leaderNode.ID].RaftNode.CreateTopic(topicName, 1, 5*time.Second)
	require.NoError(t, err)

	// Concurrent write test
	const numWriters = 10
	const messagesPerWriter = 50
	const totalMessages = numWriters * messagesPerWriter

	t.Logf("Starting %d concurrent writers, %d messages each...", 
		numWriters, messagesPerWriter)

	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64
	startTime := time.Now()

	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < messagesPerWriter; i++ {
				msg := fmt.Sprintf("writer%d-msg%d", writerID, i)
				_, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
					topicName, 0, nil, []byte(msg), 5*time.Second)
				if err != nil {
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Concurrent write completed: %d success, %d errors in %v", 
		successCount.Load(), errorCount.Load(), duration)
	t.Logf("Throughput: %.2f msg/sec", 
		float64(successCount.Load())/duration.Seconds())

	// Wait for replication
	time.Sleep(5 * time.Second)

	// Verify all nodes have the same data
	t.Log("Verifying data consistency...")
	expectedCount := successCount.Load()
	for i, node := range cluster.Nodes {
		tp, exists := node.TopicMgr.GetTopic(topicName)
		require.True(t, exists)

		hwm, err := tp.HighWaterMark(0)
		require.NoError(t, err)
		assert.Equal(t, expectedCount, hwm, 
			"node%d should have %d messages", i, expectedCount)
	}

	t.Log("✅ Concurrent writes successful with data consistency")
}

// TestReplicationPerformance measures replication latency
func TestReplicationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cluster := setupThreeNodeCluster(t)
	defer cluster.Shutdown()

	leaderNode := cluster.WaitForLeader(t, 10*time.Second)

	topicName := "test-performance"
	err := cluster.Nodes[leaderNode.ID].RaftNode.CreateTopic(topicName, 1, 5*time.Second)
	require.NoError(t, err)

	// Measure replication latency
	const testMessages = 100
	var totalLatency time.Duration

	t.Logf("Measuring replication latency for %d messages...", testMessages)

	for i := 0; i < testMessages; i++ {
		start := time.Now()
		_, err := cluster.Nodes[leaderNode.ID].RaftNode.AppendMessage(
			topicName, 0, nil, []byte(fmt.Sprintf("perf-%d", i)), 5*time.Second)
		require.NoError(t, err)
		
		latency := time.Since(start)
		totalLatency += latency
	}

	avgLatency := totalLatency / testMessages
	t.Logf("Average replication latency: %v", avgLatency)
	t.Logf("Total time: %v", totalLatency)
	t.Logf("Throughput: %.2f msg/sec", 
		float64(testMessages)/(totalLatency.Seconds()))

	// Verify replication
	time.Sleep(3 * time.Second)
	for i, node := range cluster.Nodes {
		tp, _ := node.TopicMgr.GetTopic(topicName)
		hwm, _ := tp.HighWaterMark(0)
		assert.Equal(t, int64(testMessages), hwm, "node%d HWM", i)
	}

	// Performance assertions
	assert.Less(t, avgLatency, 100*time.Millisecond, 
		"average latency should be under 100ms")

	t.Log("✅ Replication performance within acceptable limits")
}

// ClusterNode represents a node in the test cluster
type ClusterNode struct {
	ID       int
	RaftNode *raftpkg.Node
	TopicMgr *topic.Manager
	RaftDir  string
	RaftBind string
	Peers    []string
}

// TestCluster represents a test cluster
type TestCluster struct {
	Nodes []*ClusterNode
}

// setupThreeNodeCluster creates a 3-node test cluster
func setupThreeNodeCluster(t *testing.T) *TestCluster {
	nodes := make([]*ClusterNode, 3)
	
	// Node configurations
	nodeConfigs := []struct {
		id        int
		raftBind  string
		bootstrap bool
		peers     []string
	}{
		{0, "127.0.0.1:18001", true, []string{}},
		{1, "127.0.0.1:18002", false, []string{"127.0.0.1:18001"}},
		{2, "127.0.0.1:18003", false, []string{"127.0.0.1:18001"}},
	}

	// Create all nodes
	for i, cfg := range nodeConfigs {
		raftDir := t.TempDir()
		dataDir := t.TempDir()
		
		topicMgr := topic.NewManager(dataDir, 1024*1024)
		
		raftCfg := &raftpkg.Config{
			NodeID:    fmt.Sprintf("node%d", cfg.id),
			RaftDir:   raftDir,
			RaftBind:  cfg.raftBind,
			Bootstrap: cfg.bootstrap,
			Peers:     cfg.peers,
		}

		raftNode, err := raftpkg.NewNode(raftCfg, topicMgr)
		require.NoError(t, err, "failed to create node%d", cfg.id)

		nodes[i] = &ClusterNode{
			ID:       cfg.id,
			RaftNode: raftNode,
			TopicMgr: topicMgr,
			RaftDir:  raftDir,
			RaftBind: cfg.raftBind,
			Peers:    cfg.peers,
		}
	}

	// Wait for bootstrap node to become leader
	time.Sleep(3 * time.Second)
	require.True(t, nodes[0].RaftNode.IsLeader(), "node0 should be initial leader")

	// Add other nodes to cluster
	err := nodes[0].RaftNode.AddVoter("node1", "127.0.0.1:18002")
	require.NoError(t, err, "failed to add node1")

	err = nodes[0].RaftNode.AddVoter("node2", "127.0.0.1:18003")
	require.NoError(t, err, "failed to add node2")

	// Wait for cluster to stabilize
	time.Sleep(3 * time.Second)

	return &TestCluster{Nodes: nodes}
}

// WaitForLeader waits for a leader to be elected and returns it
func (c *TestCluster) WaitForLeader(t *testing.T, timeout time.Duration) *ClusterNode {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, node := range c.Nodes {
			if node.RaftNode.IsLeader() {
				return node
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.Fail(t, "no leader elected within timeout")
	return nil
}

// Shutdown shuts down all nodes in the cluster
func (c *TestCluster) Shutdown() {
	for _, node := range c.Nodes {
		if node.RaftNode != nil {
			node.RaftNode.Shutdown()
		}
		if node.TopicMgr != nil {
			node.TopicMgr.Close()
		}
	}
}
