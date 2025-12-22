package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// TestFollowerLEOTracking tests follower LEO tracking
func TestFollowerLEOTracking(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

	// Create topic
	testTopic := "test-topic"
	err := topicMgr.CreateTopic(testTopic, 1)
	assert.NoError(t, err)

	// Set up replication metadata
	topicObj, exists := topicMgr.GetTopic(testTopic)
	assert.True(t, exists)
	assert.NotNil(t, topicObj)
	topicObj.SetReplicationFactor(3)
	topicObj.SetReplicas(0, []int32{1, 2, 3})
	topicObj.SetISR(0, []int32{1})

	// Test UpdateFollowerLEO
	topicObj.UpdateFollowerLEO(0, 2, 100)

	// Test GetFollowerLEO
	leo, exists := topicObj.GetFollowerLEO(0, 2)
	assert.True(t, exists)
	assert.Equal(t, int64(100), leo)

	// Test follower not found
	_, exists = topicObj.GetFollowerLEO(0, 999)
	assert.False(t, exists)
}

// TestISRManagement tests ISR expansion and shrinking
func TestISRManagement(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

	testTopic := "test-topic"
	err := topicMgr.CreateTopic(testTopic, 1)
	assert.NoError(t, err)

	topicObj, exists := topicMgr.GetTopic(testTopic)
	assert.True(t, exists)
	assert.NotNil(t, topicObj)
	topicObj.SetReplicationFactor(3)
	topicObj.SetReplicas(0, []int32{1, 2, 3})
	topicObj.SetISR(0, []int32{1})

	// Leader LEO is 100
	leaderLEO := int64(100)

	// Follower 2 catches up (LEO = 100)
	topicObj.UpdateFollowerLEO(0, 2, 100)

	// Update ISR - follower 2 should be added
	topicObj.UpdateISR(0, leaderLEO)

	isr := topicObj.GetISR(0)
	assert.Contains(t, isr, int32(2), "ISR should contain follower 2 after catching up")

	// Follower 3 is lagging (LEO = 50)
	topicObj.UpdateFollowerLEO(0, 3, 50)

	// Update ISR - follower 3 should not be added (lagging)
	topicObj.UpdateISR(0, leaderLEO)

	isr = topicObj.GetISR(0)
	assert.NotContains(t, isr, int32(3), "ISR should not contain lagging follower 3")
}

// TestISRShrinkOnTimeout tests ISR removal based on fetch timeout
func TestISRShrinkOnTimeout(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

	testTopic := "test-topic"
	err := topicMgr.CreateTopic(testTopic, 1)
	assert.NoError(t, err)

	topicObj, exists := topicMgr.GetTopic(testTopic)
	assert.True(t, exists)
	assert.NotNil(t, topicObj)

	// Set short lag timeout for testing (100ms)
	topicObj.ReplicaLagMaxMs = 100
	topicObj.SetReplicationFactor(3)
	topicObj.SetReplicas(0, []int32{1, 2, 3})

	// Start with ISR containing leader and follower 2
	topicObj.SetISR(0, []int32{1, 2})

	// Follower 2 is caught up
	topicObj.UpdateFollowerLEO(0, 2, 100)

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Update ISR - follower 2 should be removed due to no recent fetch
	leaderLEO := int64(100)
	topicObj.UpdateISR(0, leaderLEO)

	isr := topicObj.GetISR(0)
	assert.NotContains(t, isr, int32(2), "ISR should remove follower 2 after timeout")
	assert.Contains(t, isr, int32(1), "ISR should still contain leader")
}

// TestGetLeaderForPartition tests leader retrieval
func TestGetLeaderForPartition(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

	testTopic := "test-topic"
	err := topicMgr.CreateTopic(testTopic, 1)
	assert.NoError(t, err)

	topicObj, exists := topicMgr.GetTopic(testTopic)
	assert.True(t, exists)
	assert.NotNil(t, topicObj)
	topicObj.SetReplicationFactor(3)
	topicObj.SetReplicas(0, []int32{2, 3, 4}) // Leader is first replica (2)

	leader, exists := topicObj.GetLeaderForPartition(0)
	assert.True(t, exists)
	assert.Equal(t, int32(2), leader, "Leader should be first replica")

	// Test partition not found
	_, exists = topicObj.GetLeaderForPartition(999)
	assert.False(t, exists)
}

// TestMultipleFollowers tests tracking multiple followers
func TestMultipleFollowers(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{
			DataDir:        t.TempDir(),
			LogSegmentSize: 1024 * 1024,
		},
		Kafka: config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)

	testTopic := "test-topic"
	err := topicMgr.CreateTopic(testTopic, 1)
	assert.NoError(t, err)

	topicObj, exists := topicMgr.GetTopic(testTopic)
	assert.True(t, exists)
	assert.NotNil(t, topicObj)
	topicObj.SetReplicationFactor(5)
	topicObj.SetReplicas(0, []int32{1, 2, 3, 4, 5})
	topicObj.SetISR(0, []int32{1})

	leaderLEO := int64(200)

	// Multiple followers with different LEOs
	followers := map[int32]int64{
		2: 200, // Fully caught up
		3: 199, // 1 offset behind (should be in ISR)
		4: 150, // Lagging
		5: 50,  // Very lagging
	}

	for followerID, leo := range followers {
		topicObj.UpdateFollowerLEO(0, followerID, leo)
	}

	// Update ISR
	topicObj.UpdateISR(0, leaderLEO)

	isr := topicObj.GetISR(0)

	// Verify ISR contains only leader and caught-up followers
	assert.Contains(t, isr, int32(1), "ISR should contain leader")
	assert.Contains(t, isr, int32(2), "ISR should contain fully caught-up follower 2")
	assert.Contains(t, isr, int32(3), "ISR should contain nearly caught-up follower 3")
	assert.NotContains(t, isr, int32(4), "ISR should not contain lagging follower 4")
	assert.NotContains(t, isr, int32(5), "ISR should not contain very lagging follower 5")
}
