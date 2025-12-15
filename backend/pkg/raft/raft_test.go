// Copyright 2025 Takhin Data, Inc.

package raft

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestFSMApplyCreateTopic(t *testing.T) {
	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create FSM
	fsm := NewFSM(topicMgr)

	// Create command
	cmd := Command{
		Type:      CommandCreateTopic,
		TopicName: "test-topic",
		NumParts:  3,
	}

	// Apply command (simulate Raft log application)
	result := fsm.applyCreateTopic(cmd)
	assert.Nil(t, result)

	// Verify topic was created
	_, exists := topicMgr.GetTopic("test-topic")
	assert.True(t, exists)
}

func TestFSMApplyAppend(t *testing.T) {
	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create topic first
	err := topicMgr.CreateTopic("test-topic", 1)
	require.NoError(t, err)

	// Create FSM
	fsm := NewFSM(topicMgr)

	// Create append command
	cmd := Command{
		Type:      CommandAppend,
		TopicName: "test-topic",
		Partition: 0,
		Key:       []byte("key1"),
		Value:     []byte("value1"),
	}

	// Apply command
	result := fsm.applyAppend(cmd)

	// Should return offset
	offset, ok := result.(int64)
	assert.True(t, ok)
	assert.Equal(t, int64(0), offset)
}

func TestRaftNodeCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft integration test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create Raft node
	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0", // Random port
		Bootstrap: true,
		Peers:     []string{},
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Wait for leader election
	time.Sleep(2 * time.Second)

	// Check if node is leader
	assert.True(t, node.IsLeader())
}

func TestRaftCreateTopic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft integration test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create Raft node
	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Wait for leader election
	time.Sleep(2 * time.Second)

	// Create topic through Raft
	err = node.CreateTopic("test-topic", 3, 5*time.Second)
	require.NoError(t, err)

	// Verify topic exists
	_, exists := topicMgr.GetTopic("test-topic")
	assert.True(t, exists)
}

func TestRaftAppendMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft integration test in short mode")
	}

	// Create topic manager
	dir := t.TempDir()
	topicMgr := topic.NewManager(dir, 1024*1024)
	defer topicMgr.Close()

	// Create Raft node
	raftDir := t.TempDir()
	cfg := &Config{
		NodeID:    "node1",
		RaftDir:   raftDir,
		RaftBind:  "127.0.0.1:0",
		Bootstrap: true,
		Peers:     []string{},
	}

	node, err := NewNode(cfg, topicMgr)
	require.NoError(t, err)
	defer node.Shutdown()

	// Wait for leader election
	time.Sleep(2 * time.Second)

	// Create topic first
	err = node.CreateTopic("test-topic", 1, 5*time.Second)
	require.NoError(t, err)

	// Append message through Raft
	result, err := node.AppendMessage("test-topic", 0, []byte("key"), []byte("value"), 5*time.Second)
	require.NoError(t, err)

	offset, ok := result.(int64)
	require.True(t, ok, "result should be int64")
	assert.Equal(t, int64(0), offset)

	// Verify message exists
	tp, exists := topicMgr.GetTopic("test-topic")
	require.True(t, exists)

	record, err := tp.Read(0, 0)
	require.NoError(t, err)
	assert.Equal(t, []byte("key"), record.Key)
	assert.Equal(t, []byte("value"), record.Value)
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
