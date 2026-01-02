package topic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetReplicationFactor(t *testing.T) {
	tests := []struct {
		name     string
		input    int16
		expected int16
	}{
		{"valid factor", 3, 3},
		{"zero becomes one", 0, 1},
		{"negative becomes one", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic := &Topic{Name: "test"}
			topic.SetReplicationFactor(tt.input)
			assert.Equal(t, tt.expected, topic.ReplicationFactor)
		})
	}
}

func TestGetSetReplicas(t *testing.T) {
	tmpDir := t.TempDir()
	topic := &Topic{
		Name:    "test",
		baseDir: tmpDir,
	}

	// Initially nil
	replicas := topic.GetReplicas(0)
	assert.Nil(t, replicas)

	// Set replicas
	expected := []int32{1, 2, 3}
	topic.SetReplicas(0, expected)

	// Verify replicas set
	got := topic.GetReplicas(0)
	assert.Equal(t, expected, got)

	// Verify ISR initialized
	isr := topic.GetISR(0)
	assert.Equal(t, expected, isr)
}

func TestGetSetISR(t *testing.T) {
	tmpDir := t.TempDir()
	topic := &Topic{
		Name:    "test",
		baseDir: tmpDir,
	}

	// Initially nil
	isr := topic.GetISR(0)
	assert.Nil(t, isr)

	// Set ISR
	expected := []int32{1, 2}
	topic.SetISR(0, expected)

	// Verify ISR set
	got := topic.GetISR(0)
	assert.Equal(t, expected, got)
}

func TestUpdateFollowerLEO(t *testing.T) {
	topic := &Topic{Name: "test"}

	// Update LEO
	topic.UpdateFollowerLEO(0, 2, 100)
	topic.UpdateFollowerLEO(0, 3, 95)

	// Verify LEO values
	leo, exists := topic.GetFollowerLEO(0, 2)
	assert.True(t, exists)
	assert.Equal(t, int64(100), leo)

	leo, exists = topic.GetFollowerLEO(0, 3)
	assert.True(t, exists)
	assert.Equal(t, int64(95), leo)

	// Non-existent follower
	leo, exists = topic.GetFollowerLEO(0, 99)
	assert.False(t, exists)
	assert.Equal(t, int64(0), leo)
}

func TestUpdateISR(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Topic)
		leaderLEO   int64
		expectedISR []int32
	}{
		{
			name: "all replicas in sync",
			setup: func(top *Topic) {
				top.Replicas = map[int32][]int32{0: {1, 2, 3}}
				top.UpdateFollowerLEO(0, 2, 100)
				top.UpdateFollowerLEO(0, 3, 100)
			},
			leaderLEO:   100,
			expectedISR: []int32{1, 2, 3},
		},
		{
			name: "one follower lagging",
			setup: func(top *Topic) {
				top.Replicas = map[int32][]int32{0: {1, 2, 3}}
				top.UpdateFollowerLEO(0, 2, 100)
				top.UpdateFollowerLEO(0, 3, 50)
			},
			leaderLEO:   100,
			expectedISR: []int32{1, 2},
		},
		{
			name: "follower caught up but stale fetch",
			setup: func(top *Topic) {
				top.Replicas = map[int32][]int32{0: {1, 2}}
				top.ReplicaLagMaxMs = 1000
				top.FollowerLEO = map[int32]map[int32]int64{0: {2: 100}}
				top.LastFetchTime = map[int32]map[int32]time.Time{
					0: {2: time.Now().Add(-2 * time.Second)},
				}
			},
			leaderLEO:   100,
			expectedISR: []int32{1},
		},
		{
			name: "no replicas",
			setup: func(top *Topic) {
				top.Replicas = map[int32][]int32{0: nil}
			},
			leaderLEO:   100,
			expectedISR: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic := &Topic{Name: "test"}
			tt.setup(topic)
			isr := topic.UpdateISR(0, tt.leaderLEO)
			assert.Equal(t, tt.expectedISR, isr)
		})
	}
}

func TestGetLeaderForPartition(t *testing.T) {
	topic := &Topic{
		Name: "test",
		Replicas: map[int32][]int32{
			0: {1, 2, 3},
			1: {2, 3, 1},
		},
	}

	// Partition 0 leader
	leader, exists := topic.GetLeaderForPartition(0)
	assert.True(t, exists)
	assert.Equal(t, int32(1), leader)

	// Partition 1 leader
	leader, exists = topic.GetLeaderForPartition(1)
	assert.True(t, exists)
	assert.Equal(t, int32(2), leader)

	// Non-existent partition
	leader, exists = topic.GetLeaderForPartition(99)
	assert.False(t, exists)
	assert.Equal(t, int32(-1), leader)
}

func TestManagerListTopics(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	// Empty initially
	topics := mgr.ListTopics()
	assert.Empty(t, topics)

	// Create topics
	err := mgr.CreateTopic("topic1", 2)
	require.NoError(t, err)
	err = mgr.CreateTopic("topic2", 1)
	require.NoError(t, err)

	// List topics
	topics = mgr.ListTopics()
	assert.Len(t, topics, 2)
	assert.Contains(t, topics, "topic1")
	assert.Contains(t, topics, "topic2")
}

func TestTopicHighWaterMark(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	err := mgr.CreateTopic("test", 1)
	require.NoError(t, err)

	mgr.mu.RLock()
	topic := mgr.topics["test"]
	mgr.mu.RUnlock()
	require.NotNil(t, topic)

	// Append data
	offset, err := topic.Append(0, []byte("key"), []byte("test message"))
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset)

	// Check HWM
	hwm, err := topic.HighWaterMark(0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), hwm)

	// Non-existent partition
	_, err = topic.HighWaterMark(99)
	assert.Error(t, err)
}

func TestTopicSize(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	err := mgr.CreateTopic("test", 2)
	require.NoError(t, err)

	mgr.mu.RLock()
	topic := mgr.topics["test"]
	mgr.mu.RUnlock()
	require.NotNil(t, topic)

	// Append to partition 0
	_, err = topic.Append(0, []byte("key"), []byte("message1"))
	require.NoError(t, err)

	// Append to partition 1
	_, err = topic.Append(1, []byte("key"), []byte("message2"))
	require.NoError(t, err)

	// Total size
	size, err := topic.Size()
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))

	// Partition size
	pSize, err := topic.PartitionSize(0)
	require.NoError(t, err)
	assert.Greater(t, pSize, int64(0))

	// Non-existent partition
	_, err = topic.PartitionSize(99)
	assert.Error(t, err)
}

func TestTopicNumPartitions(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	err := mgr.CreateTopic("test", 3)
	require.NoError(t, err)

	mgr.mu.RLock()
	topic := mgr.topics["test"]
	mgr.mu.RUnlock()
	require.NotNil(t, topic)

	count := len(topic.Partitions)
	assert.Equal(t, 3, count)
}

func TestTopicReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	err := mgr.CreateTopic("test", 1)
	require.NoError(t, err)

	mgr.mu.RLock()
	topic := mgr.topics["test"]
	mgr.mu.RUnlock()
	require.NotNil(t, topic)

	// Append some data
	offset1, err := topic.Append(0, []byte("key1"), []byte("msg1"))
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset1)

	offset2, err := topic.Append(0, []byte("key2"), []byte("msg2"))
	require.NoError(t, err)
	assert.Equal(t, int64(1), offset2)

	// Read data back
	record, err := topic.Read(0, 0)
	require.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, []byte("key1"), record.Key)
	assert.Equal(t, []byte("msg1"), record.Value)

	// Read non-existent partition
	_, err = topic.Read(99, 0)
	assert.Error(t, err)
}

func TestTopicAppendErrors(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	err := mgr.CreateTopic("test", 1)
	require.NoError(t, err)

	mgr.mu.RLock()
	topic := mgr.topics["test"]
	mgr.mu.RUnlock()
	require.NotNil(t, topic)

	// Try to append to non-existent partition
	_, err = topic.Append(99, []byte("key"), []byte("value"))
	assert.Error(t, err)
}

func TestManagerClose(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	err := mgr.CreateTopic("test", 2)
	require.NoError(t, err)

	mgr.mu.RLock()
	topic := mgr.topics["test"]
	mgr.mu.RUnlock()

	// Append data
	_, err = topic.Append(0, []byte("key"), []byte("msg"))
	require.NoError(t, err)

	// Close manager
	err = mgr.Close()
	assert.NoError(t, err)
}

func TestTopicClose(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, 1024*1024)

	// Create a topic
	err := mgr.CreateTopic("test", 1)
	require.NoError(t, err)

	// Get topic
	mgr.mu.RLock()
	topic := mgr.topics["test"]
	mgr.mu.RUnlock()
	require.NotNil(t, topic)

	// Append data
	_, err = topic.Append(0, []byte("key"), []byte("value"))
	require.NoError(t, err)

	// Close manager (which closes topics)
	err = mgr.Close()
	assert.NoError(t, err)
}

func TestManagerReloadTopics(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial manager
	mgr1 := NewManager(tmpDir, 1024*1024)
	err := mgr1.CreateTopic("test", 2)
	require.NoError(t, err)

	mgr1.mu.RLock()
	topic1 := mgr1.topics["test"]
	mgr1.mu.RUnlock()

	_, err = topic1.Append(0, []byte("key"), []byte("message"))
	require.NoError(t, err)

	err = mgr1.Close()
	require.NoError(t, err)

	// Create new manager - should reload existing topics
	mgr2 := NewManager(tmpDir, 1024*1024)
	topics := mgr2.ListTopics()
	assert.Contains(t, topics, "test")

	mgr2.mu.RLock()
	topic2 := mgr2.topics["test"]
	mgr2.mu.RUnlock()

	// Verify data persisted
	hwm, err := topic2.HighWaterMark(0)
	require.NoError(t, err)
	assert.Greater(t, hwm, int64(0))

	err = mgr2.Close()
	require.NoError(t, err)
}
