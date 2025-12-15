// Copyright 2025 Takhin Data, Inc.

package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/handler"
	raftpkg "github.com/takhin-data/takhin/pkg/raft"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestRaftBackendIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create Raft node
	raftDir := t.TempDir()
	raftCfg := &raftpkg.Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
	}

	node, err := raftpkg.NewNode(raftCfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Wait for leader election
	time.Sleep(2 * time.Second)
	require.True(t, node.IsLeader(), "node should become leader")

	// Create handler with Raft backend
	cfg := &config.Config{}
	backend := handler.NewRaftBackend(node, 5*time.Second)
	_ = handler.NewWithBackend(cfg, topicMgr, backend)

	// Test creating a topic through Raft backend
	err = backend.CreateTopic("test-topic", 1)
	require.NoError(t, err, "should create topic through Raft")

	// Verify topic exists
	tp, exists := backend.GetTopic("test-topic")
	require.True(t, exists, "topic should exist")
	require.NotNil(t, tp)

	// Test appending message through Raft backend
	offset, err := backend.Append("test-topic", 0, []byte("key"), []byte("test message"))
	require.NoError(t, err, "should append message through Raft")
	assert.Equal(t, int64(0), offset, "first message should have offset 0")

	// Verify message was written
	record, err := tp.Read(0, 0)
	require.NoError(t, err, "should read written message")
	assert.Equal(t, []byte("test message"), record.Value)

	// Test appending multiple messages
	for i := 0; i < 10; i++ {
		offset, err := backend.Append("test-topic", 0, nil, []byte("message"))
		require.NoError(t, err)
		assert.Equal(t, int64(i+1), offset)
	}

	// Verify high water mark
	hwm, err := tp.HighWaterMark(0)
	require.NoError(t, err)
	assert.Equal(t, int64(11), hwm, "should have 11 messages")
}

func TestDirectVsRaftBackend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test direct backend
	dir1 := t.TempDir()
	topicMgr1 := topic.NewManager(dir1, 1024*1024)
	defer topicMgr1.Close()

	directBackend := handler.NewDirectBackend(topicMgr1)

	err := directBackend.CreateTopic("direct-topic", 1)
	require.NoError(t, err)

	offset, err := directBackend.Append("direct-topic", 0, nil, []byte("direct"))
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset)

	// Test Raft backend
	dir2 := t.TempDir()
	topicMgr2 := topic.NewManager(dir2, 1024*1024)
	defer topicMgr2.Close()

	raftDir := t.TempDir()
	raftCfg := &raftpkg.Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
	}

	node, err := raftpkg.NewNode(raftCfg, topicMgr2)
	require.NoError(t, err)
	defer node.Shutdown()

	time.Sleep(2 * time.Second)
	require.True(t, node.IsLeader())

	raftBackend := handler.NewRaftBackend(node, 5*time.Second)

	err = raftBackend.CreateTopic("raft-topic", 1)
	require.NoError(t, err)

	offset, err = raftBackend.Append("raft-topic", 0, nil, []byte("raft"))
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset)

	// Both backends should work identically from the API perspective
	t.Log("Both direct and Raft backends work correctly")
}
