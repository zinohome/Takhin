package topic

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/storage/log"
)

// TestSaveAndLoadMetadata tests metadata persistence
func TestSaveAndLoadMetadata(t *testing.T) {
	tempDir := t.TempDir()

	// Create a topic with metadata
	topic := &Topic{
		Name:              "test-topic",
		ReplicationFactor: 3,
		Replicas: map[int32][]int32{
			0: {1, 2, 3},
			1: {2, 3, 1},
			2: {3, 1, 2},
		},
		ISR: map[int32][]int32{
			0: {1, 2},
			1: {2, 3},
			2: {3, 1},
		},
		Partitions: make(map[int32]*log.Log),
	}
	// Create dummy partitions
	topic.Partitions[0] = nil
	topic.Partitions[1] = nil
	topic.Partitions[2] = nil

	// Save metadata
	err := topic.SaveMetadata(tempDir)
	require.NoError(t, err)

	// Load metadata
	loaded, err := LoadMetadata(tempDir, "test-topic")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify metadata
	assert.Equal(t, "test-topic", loaded.Name)
	assert.Equal(t, int16(3), loaded.ReplicationFactor)
	assert.Equal(t, int32(1), loaded.Version)
	assert.Len(t, loaded.Partitions, 3)

	// Check partition 0
	var part0 *PartitionMetadata
	for i, p := range loaded.Partitions {
		if p.PartitionID == 0 {
			part0 = &loaded.Partitions[i]
			break
		}
	}
	require.NotNil(t, part0)
	assert.Equal(t, []int32{1, 2, 3}, part0.Replicas)
	assert.Equal(t, []int32{1, 2}, part0.ISR)
	assert.Equal(t, int32(1), part0.Leader) // First replica
}

// TestApplyMetadata tests applying loaded metadata to a topic
func TestApplyMetadata(t *testing.T) {
	metadata := &TopicMetadata{
		Name:              "test-topic",
		ReplicationFactor: 3,
		Partitions: []PartitionMetadata{
			{
				PartitionID: 0,
				Replicas:    []int32{1, 2, 3},
				ISR:         []int32{1, 2},
				Leader:      1,
			},
			{
				PartitionID: 1,
				Replicas:    []int32{2, 3, 1},
				ISR:         []int32{2, 3},
				Leader:      2,
			},
		},
	}

	topic := &Topic{
		Name:       "test-topic",
		Partitions: make(map[int32]*log.Log),
	}

	err := topic.ApplyMetadata(metadata)
	require.NoError(t, err)

	// Verify applied metadata
	assert.Equal(t, int16(3), topic.ReplicationFactor)
	assert.Equal(t, []int32{1, 2, 3}, topic.Replicas[0])
	assert.Equal(t, []int32{1, 2}, topic.ISR[0])
	assert.Equal(t, []int32{2, 3, 1}, topic.Replicas[1])
	assert.Equal(t, []int32{2, 3}, topic.ISR[1])
	assert.NotNil(t, topic.FollowerLEO)
	assert.NotNil(t, topic.LastFetchTime)
}

// TestLoadNonexistentMetadata tests loading metadata for a nonexistent topic
func TestLoadNonexistentMetadata(t *testing.T) {
	tempDir := t.TempDir()

	metadata, err := LoadMetadata(tempDir, "nonexistent-topic")
	require.NoError(t, err)
	assert.Nil(t, metadata)
}

// TestDeleteMetadata tests deleting metadata
func TestDeleteMetadata(t *testing.T) {
	tempDir := t.TempDir()

	topic := &Topic{
		Name:              "test-topic",
		ReplicationFactor: 1,
		Partitions:        make(map[int32]*log.Log),
		Replicas:          map[int32][]int32{0: {1}},
		ISR:               map[int32][]int32{0: {1}},
	}
	topic.Partitions[0] = nil

	// Save metadata
	err := topic.SaveMetadata(tempDir)
	require.NoError(t, err)

	// Verify file exists
	_, err = LoadMetadata(tempDir, "test-topic")
	require.NoError(t, err)

	// Delete metadata
	err = DeleteMetadata(tempDir, "test-topic")
	require.NoError(t, err)

	// Verify file is deleted
	metadata, err := LoadMetadata(tempDir, "test-topic")
	require.NoError(t, err)
	assert.Nil(t, metadata)
}

// TestManagerPersistence tests end-to-end persistence with Manager
func TestManagerPersistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create manager and topic
	mgr := NewManager(tempDir, 1024*1024)
	err := mgr.CreateTopic("persistent-topic", 2)
	require.NoError(t, err)

	// Set replicas
	topic, exists := mgr.GetTopic("persistent-topic")
	require.True(t, exists)
	topic.SetReplicas(0, []int32{1, 2, 3})
	topic.SetReplicas(1, []int32{2, 3, 1})

	// Verify metadata was saved
	metadata, err := LoadMetadata(tempDir, "persistent-topic")
	require.NoError(t, err)
	require.NotNil(t, metadata)
	assert.Len(t, metadata.Partitions, 2)

	// Create new manager (simulating restart)
	mgr2 := NewManager(tempDir, 1024*1024)

	// Verify topic was loaded
	topic2, exists := mgr2.GetTopic("persistent-topic")
	require.True(t, exists)
	assert.Equal(t, int16(1), topic2.ReplicationFactor)
	assert.Equal(t, []int32{1, 2, 3}, topic2.Replicas[0])
	assert.Equal(t, []int32{2, 3, 1}, topic2.Replicas[1])
}

// TestTopicDeleteRemovesMetadata tests that deleting a topic removes metadata
func TestTopicDeleteRemovesMetadata(t *testing.T) {
	tempDir := t.TempDir()

	// Create manager and topic
	mgr := NewManager(tempDir, 1024*1024)
	err := mgr.CreateTopic("temp-topic", 1)
	require.NoError(t, err)

	// Verify metadata exists
	metadata, err := LoadMetadata(tempDir, "temp-topic")
	require.NoError(t, err)
	require.NotNil(t, metadata)

	// Delete topic
	err = mgr.DeleteTopic("temp-topic")
	require.NoError(t, err)

	// Verify metadata is gone
	_, err = os.Stat(tempDir + "/temp-topic")
	assert.True(t, os.IsNotExist(err))
}

// TestAtomicWrite tests that metadata writes are atomic
func TestAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()

	topic := &Topic{
		Name:              "atomic-topic",
		ReplicationFactor: 2,
		Partitions:        make(map[int32]*log.Log),
		Replicas:          map[int32][]int32{0: {1, 2}},
		ISR:               map[int32][]int32{0: {1}},
	}
	topic.Partitions[0] = nil

	// Multiple writes should not corrupt data
	for i := 0; i < 10; i++ {
		err := topic.SaveMetadata(tempDir)
		require.NoError(t, err)
	}

	// Should be able to load clean metadata
	metadata, err := LoadMetadata(tempDir, "atomic-topic")
	require.NoError(t, err)
	require.NotNil(t, metadata)
	assert.Equal(t, int16(2), metadata.ReplicationFactor)
}

// TestCorruptedMetadata tests handling of corrupted metadata
func TestCorruptedMetadata(t *testing.T) {
	tempDir := t.TempDir()

	// Write corrupted metadata
	topicDir := tempDir + "/corrupted-topic"
	require.NoError(t, os.MkdirAll(topicDir, 0755))
	require.NoError(t, os.WriteFile(topicDir+"/topic-metadata.json", []byte("invalid json"), 0644))

	// Should return error
	_, err := LoadMetadata(tempDir, "corrupted-topic")
	assert.Error(t, err)
}
