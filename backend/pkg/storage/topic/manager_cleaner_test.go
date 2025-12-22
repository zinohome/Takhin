// Copyright 2025 Takhin Data, Inc.

package topic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/storage/log"
)

func TestManagerCleanerIntegration(t *testing.T) {
	dir := t.TempDir()

	// Create manager
	manager := NewManager(dir, 512)

	// Create cleaner
	cleanerConfig := log.CleanerConfig{
		CleanupIntervalSeconds:    1,
		CompactionIntervalSeconds: 2,
		RetentionPolicy: log.RetentionPolicy{
			RetentionBytes: 2048, // Small retention to trigger cleanup
			RetentionMs:    -1,
		},
		CompactionPolicy: log.DefaultCompactionPolicy(),
		Enabled:          false, // Manual control for testing
	}
	cleaner := log.NewCleaner(cleanerConfig)
	manager.SetCleaner(cleaner)

	// Create topic with partitions
	err := manager.CreateTopic("test-topic", 2)
	require.NoError(t, err)

	// Write data to partitions
	topic, exists := manager.GetTopic("test-topic")
	require.True(t, exists)

	for i := 0; i < 50; i++ {
		_, err := topic.Append(0, []byte("key"), make([]byte, 30))
		require.NoError(t, err)
		_, err = topic.Append(1, []byte("key"), make([]byte, 30))
		require.NoError(t, err)
	}

	// Verify logs are registered with cleaner
	status := cleaner.GetStatus()
	assert.Equal(t, 2, status.NumRegisteredLogs)

	// Trigger cleanup manually
	err = cleaner.ForceCleanup()
	require.NoError(t, err)

	// Check that cleanup ran
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalCleanupRuns, int64(0))

	// Delete topic should unregister logs
	err = manager.DeleteTopic("test-topic")
	require.NoError(t, err)

	// Verify logs are unregistered
	status = cleaner.GetStatus()
	assert.Equal(t, 0, status.NumRegisteredLogs)
}

func TestManagerCleanerAutoCleanup(t *testing.T) {
	dir := t.TempDir()

	// Create manager
	manager := NewManager(dir, 512)

	// Create cleaner with enabled background tasks
	cleanerConfig := log.CleanerConfig{
		CleanupIntervalSeconds:    1, // Run every second
		CompactionIntervalSeconds: 2,
		RetentionPolicy: log.RetentionPolicy{
			RetentionBytes: 2048,
			RetentionMs:    -1,
		},
		CompactionPolicy: log.DefaultCompactionPolicy(),
		Enabled:          true,
	}
	cleaner := log.NewCleaner(cleanerConfig)
	manager.SetCleaner(cleaner)

	// Start cleaner
	err := cleaner.Start()
	require.NoError(t, err)
	defer cleaner.Stop()

	// Create topic
	err = manager.CreateTopic("auto-cleanup-topic", 1)
	require.NoError(t, err)

	// Write enough data to exceed retention
	topic, _ := manager.GetTopic("auto-cleanup-topic")
	for i := 0; i < 100; i++ {
		_, err := topic.Append(0, []byte("key"), make([]byte, 30))
		require.NoError(t, err)
	}

	// Wait for cleanup to run
	time.Sleep(2 * time.Second)

	// Verify cleanup happened
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalCleanupRuns, int64(0))

	// Cleanup
	err = manager.DeleteTopic("auto-cleanup-topic")
	require.NoError(t, err)
}

func TestManagerWithoutCleaner(t *testing.T) {
	dir := t.TempDir()

	// Create manager without cleaner
	manager := NewManager(dir, 1024*1024)

	// Should work normally without cleaner
	err := manager.CreateTopic("no-cleaner-topic", 2)
	require.NoError(t, err)

	topic, exists := manager.GetTopic("no-cleaner-topic")
	require.True(t, exists)

	// Write data
	_, err = topic.Append(0, []byte("key"), []byte("value"))
	require.NoError(t, err)

	// Read data
	record, err := topic.Read(0, 0)
	require.NoError(t, err)
	assert.Equal(t, []byte("key"), record.Key)
	assert.Equal(t, []byte("value"), record.Value)

	// Cleanup
	err = manager.DeleteTopic("no-cleaner-topic")
	require.NoError(t, err)
}
