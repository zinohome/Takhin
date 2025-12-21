// Copyright 2025 Takhin Data, Inc.

package log

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanerBasic(t *testing.T) {
	config := CleanerConfig{
		CleanupIntervalSeconds:    1, // 1 second for testing
		CompactionIntervalSeconds: 1,
		RetentionPolicy: RetentionPolicy{
			RetentionBytes: 1024,
			RetentionMs:    -1,
		},
		CompactionPolicy: DefaultCompactionPolicy(),
		Enabled:          true,
	}

	cleaner := NewCleaner(config)
	require.NotNil(t, cleaner)

	// Create test log
	dir := t.TempDir()
	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512,
	})
	require.NoError(t, err)
	defer log.Close()

	// Register log
	cleaner.RegisterLog("test-topic-0", log)

	// Write enough data to create multiple segments
	for i := 0; i < 50; i++ {
		_, err := log.Append([]byte("key"), make([]byte, 30))
		require.NoError(t, err)
	}

	initialSegments := log.NumSegments()
	require.Greater(t, initialSegments, 1)

	// Start cleaner
	err = cleaner.Start()
	require.NoError(t, err)

	// Wait for at least one cleanup run
	time.Sleep(2 * time.Second)

	// Stop cleaner
	err = cleaner.Stop()
	require.NoError(t, err)

	// Check stats
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalCleanupRuns, int64(0))

	// Check status
	status := cleaner.GetStatus()
	assert.True(t, status.Enabled)
	assert.Equal(t, 1, status.NumRegisteredLogs)
}

func TestCleanerDisabled(t *testing.T) {
	config := DefaultCleanerConfig()
	config.Enabled = false

	cleaner := NewCleaner(config)

	err := cleaner.Start()
	require.NoError(t, err)

	// Should not run any tasks
	time.Sleep(100 * time.Millisecond)

	stats := cleaner.GetStats()
	assert.Equal(t, int64(0), stats.TotalCleanupRuns)

	err = cleaner.Stop()
	require.NoError(t, err)
}

func TestCleanerRegisterUnregister(t *testing.T) {
	config := DefaultCleanerConfig()
	cleaner := NewCleaner(config)

	dir1 := t.TempDir()
	log1, err := NewLog(LogConfig{Dir: dir1, MaxSegmentSize: 1024 * 1024})
	require.NoError(t, err)
	defer log1.Close()

	dir2 := t.TempDir()
	log2, err := NewLog(LogConfig{Dir: dir2, MaxSegmentSize: 1024 * 1024})
	require.NoError(t, err)
	defer log2.Close()

	// Register logs
	cleaner.RegisterLog("topic1-0", log1)
	cleaner.RegisterLog("topic2-0", log2)

	status := cleaner.GetStatus()
	assert.Equal(t, 2, status.NumRegisteredLogs)

	// Unregister one
	cleaner.UnregisterLog("topic1-0")

	status = cleaner.GetStatus()
	assert.Equal(t, 1, status.NumRegisteredLogs)

	// Unregister all
	cleaner.UnregisterLog("topic2-0")

	status = cleaner.GetStatus()
	assert.Equal(t, 0, status.NumRegisteredLogs)
}

func TestCleanerForceCleanup(t *testing.T) {
	config := DefaultCleanerConfig()
	config.Enabled = false // Disable automatic runs

	cleaner := NewCleaner(config)

	dir := t.TempDir()
	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512,
	})
	require.NoError(t, err)
	defer log.Close()

	// Write data
	for i := 0; i < 50; i++ {
		_, err := log.Append([]byte("key"), make([]byte, 30))
		require.NoError(t, err)
	}

	cleaner.RegisterLog("test-topic-0", log)

	// Force cleanup manually
	err = cleaner.ForceCleanup()
	require.NoError(t, err)

	// Check that cleanup ran
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalCleanupRuns, int64(0))
}

func TestCleanerForceCompactionAnalysis(t *testing.T) {
	config := DefaultCleanerConfig()
	config.Enabled = false

	cleaner := NewCleaner(config)

	dir := t.TempDir()
	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512,
	})
	require.NoError(t, err)
	defer log.Close()

	// Write duplicate keys
	for i := 0; i < 50; i++ {
		_, err := log.Append([]byte("same-key"), make([]byte, 30))
		require.NoError(t, err)
	}

	cleaner.RegisterLog("test-topic-0", log)

	// Force compaction analysis
	err = cleaner.ForceCompactionAnalysis()
	require.NoError(t, err)

	// Check that analysis ran
	stats := cleaner.GetStats()
	assert.Greater(t, stats.TotalCompactionRuns, int64(0))
}

func TestCleanerUpdateConfig(t *testing.T) {
	config := DefaultCleanerConfig()
	cleaner := NewCleaner(config)

	// Update config
	newConfig := config
	newConfig.CleanupIntervalSeconds = 600
	newConfig.Enabled = false

	err := cleaner.UpdateConfig(newConfig)
	require.NoError(t, err)

	// Verify config updated
	assert.Equal(t, 600, cleaner.config.CleanupIntervalSeconds)
	assert.False(t, cleaner.config.Enabled)
}

func TestCleanerStatus(t *testing.T) {
	config := DefaultCleanerConfig()
	config.Enabled = false

	cleaner := NewCleaner(config)

	dir := t.TempDir()
	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512,
	})
	require.NoError(t, err)
	defer log.Close()

	// Write data
	for i := 0; i < 50; i++ {
		_, err := log.Append([]byte("key"), make([]byte, 30))
		require.NoError(t, err)
	}

	cleaner.RegisterLog("test-topic-0", log)

	// Run cleanup
	err = cleaner.ForceCleanup()
	require.NoError(t, err)

	// Get status
	status := cleaner.GetStatus()
	assert.False(t, status.Enabled)
	assert.Equal(t, 1, status.NumRegisteredLogs)
	assert.Greater(t, status.TotalCleanupRuns, int64(0))
	assert.NotZero(t, status.LastCleanup)

	// Test String() method
	statusStr := status.String()
	assert.Contains(t, statusStr, "Cleaner")
	assert.Contains(t, statusStr, "logs=1")
}

func TestCleanerMultipleLogs(t *testing.T) {
	config := CleanerConfig{
		CleanupIntervalSeconds:    1,
		CompactionIntervalSeconds: 1,
		RetentionPolicy: RetentionPolicy{
			RetentionBytes: 2048,
			RetentionMs:    -1,
		},
		CompactionPolicy: DefaultCompactionPolicy(),
		Enabled:          false, // Manual control for testing
	}

	cleaner := NewCleaner(config)

	// Create multiple logs
	numLogs := 3
	logs := make([]*Log, numLogs)
	for i := 0; i < numLogs; i++ {
		dir := t.TempDir()
		log, err := NewLog(LogConfig{
			Dir:            dir,
			MaxSegmentSize: 512,
		})
		require.NoError(t, err)
		defer log.Close()

		// Write data
		for j := 0; j < 50; j++ {
			_, err := log.Append([]byte("key"), make([]byte, 30))
			require.NoError(t, err)
		}

		logs[i] = log
		cleaner.RegisterLog(fmt.Sprintf("topic-%d-0", i), log)
	}

	// Force cleanup for all logs
	err := cleaner.ForceCleanup()
	require.NoError(t, err)

	// Check stats
	stats := cleaner.GetStats()
	assert.Equal(t, int64(1), stats.TotalCleanupRuns)

	// Check status
	status := cleaner.GetStatus()
	assert.Equal(t, numLogs, status.NumRegisteredLogs)
}

func TestCleanerDefaultConfig(t *testing.T) {
	config := DefaultCleanerConfig()

	assert.Equal(t, 300, config.CleanupIntervalSeconds)
	assert.Equal(t, 600, config.CompactionIntervalSeconds)
	assert.True(t, config.Enabled)
	assert.NotNil(t, config.RetentionPolicy)
	assert.NotNil(t, config.CompactionPolicy)
}
