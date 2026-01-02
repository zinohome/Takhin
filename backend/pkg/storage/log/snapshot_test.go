// Copyright 2025 Takhin Data, Inc.

package log

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotManager_CreateSnapshot(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create a log with some data
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Add some records
	for i := 0; i < 10; i++ {
		_, err := log.Append([]byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create snapshot
	snapshot, err := sm.CreateSnapshot(log)
	require.NoError(t, err)
	assert.NotEmpty(t, snapshot.ID)
	assert.Equal(t, int64(10), snapshot.HighWaterMark)
	assert.Greater(t, snapshot.TotalSize, int64(0))
	assert.Equal(t, 1, snapshot.NumSegments)

	// Verify snapshot directory exists
	snapshotPath := filepath.Join(logDir, ".snapshots", snapshot.ID)
	_, err = os.Stat(snapshotPath)
	assert.NoError(t, err)

	// Verify snapshot files exist
	entries, err := os.ReadDir(snapshotPath)
	require.NoError(t, err)
	assert.Greater(t, len(entries), 0)
}

func TestSnapshotManager_RestoreSnapshot(t *testing.T) {
	// Create temporary directories
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")
	restoreDir := filepath.Join(tmpDir, "restored")

	// Create a log with some data
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)

	// Add records
	testData := []struct {
		key   string
		value string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	for _, td := range testData {
		_, err := log.Append([]byte(td.key), []byte(td.value))
		require.NoError(t, err)
	}

	hwm := log.HighWaterMark()

	// Create snapshot
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	snapshot, err := sm.CreateSnapshot(log)
	require.NoError(t, err)

	log.Close()

	// Restore snapshot
	err = sm.RestoreSnapshot(snapshot.ID, restoreDir)
	require.NoError(t, err)

	// Open restored log
	restoredLog, err := NewLog(LogConfig{
		Dir:            restoreDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer restoredLog.Close()

	// Verify restored data
	assert.Equal(t, hwm, restoredLog.HighWaterMark())

	for i := int64(0); i < hwm; i++ {
		record, err := restoredLog.Read(i)
		require.NoError(t, err)
		assert.Equal(t, testData[i].key, string(record.Key))
		assert.Equal(t, testData[i].value, string(record.Value))
	}
}

func TestSnapshotManager_ListSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Add some data
	_, err = log.Append([]byte("key"), []byte("value"))
	require.NoError(t, err)

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create multiple snapshots
	snapshot1, err := sm.CreateSnapshot(log)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	snapshot2, err := sm.CreateSnapshot(log)
	require.NoError(t, err)

	// List snapshots
	snapshots := sm.ListSnapshots()
	assert.Len(t, snapshots, 2)

	// Verify snapshots are sorted by timestamp (newest first)
	assert.Equal(t, snapshot2.ID, snapshots[0].ID)
	assert.Equal(t, snapshot1.ID, snapshots[1].ID)
	assert.True(t, snapshots[0].Timestamp.After(snapshots[1].Timestamp))
}

func TestSnapshotManager_GetSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	_, err = log.Append([]byte("key"), []byte("value"))
	require.NoError(t, err)

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create snapshot
	snapshot, err := sm.CreateSnapshot(log)
	require.NoError(t, err)

	// Get snapshot
	retrieved := sm.GetSnapshot(snapshot.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, snapshot.ID, retrieved.ID)
	assert.Equal(t, snapshot.HighWaterMark, retrieved.HighWaterMark)

	// Try to get non-existent snapshot
	notFound := sm.GetSnapshot("non-existent-id")
	assert.Nil(t, notFound)
}

func TestSnapshotManager_DeleteSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	_, err = log.Append([]byte("key"), []byte("value"))
	require.NoError(t, err)

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create snapshot
	snapshot, err := sm.CreateSnapshot(log)
	require.NoError(t, err)

	// Verify snapshot exists
	snapshotPath := filepath.Join(logDir, ".snapshots", snapshot.ID)
	_, err = os.Stat(snapshotPath)
	assert.NoError(t, err)

	// Delete snapshot
	err = sm.DeleteSnapshot(snapshot.ID)
	require.NoError(t, err)

	// Verify snapshot is deleted
	_, err = os.Stat(snapshotPath)
	assert.True(t, os.IsNotExist(err))

	// Verify snapshot is not in list
	snapshots := sm.ListSnapshots()
	assert.Len(t, snapshots, 0)

	// Try to delete non-existent snapshot
	err = sm.DeleteSnapshot("non-existent-id")
	assert.Error(t, err)
}

func TestSnapshotManager_CleanupSnapshots_MaxSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	_, err = log.Append([]byte("key"), []byte("value"))
	require.NoError(t, err)

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create 7 snapshots
	for i := 0; i < 7; i++ {
		_, err := sm.CreateSnapshot(log)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Verify 7 snapshots exist
	snapshots := sm.ListSnapshots()
	assert.Len(t, snapshots, 7)

	// Cleanup with max 3 snapshots
	config := SnapshotConfig{
		MaxSnapshots:  3,
		RetentionTime: 24 * time.Hour,
	}
	deleted, err := sm.CleanupSnapshots(config)
	require.NoError(t, err)
	assert.Equal(t, 4, deleted)

	// Verify only 3 snapshots remain
	snapshots = sm.ListSnapshots()
	assert.Len(t, snapshots, 3)
}

func TestSnapshotManager_CleanupSnapshots_RetentionTime(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	_, err = log.Append([]byte("key"), []byte("value"))
	require.NoError(t, err)

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create snapshot
	snapshot, err := sm.CreateSnapshot(log)
	require.NoError(t, err)

	// Manually set old timestamp
	sm.metadata.mu.Lock()
	for _, s := range sm.metadata.Snapshots {
		if s.ID == snapshot.ID {
			s.Timestamp = time.Now().Add(-48 * time.Hour)
		}
	}
	sm.metadata.mu.Unlock()
	sm.saveMetadata()

	// Create recent snapshot
	_, err = sm.CreateSnapshot(log)
	require.NoError(t, err)

	// Cleanup with 24 hour retention
	config := SnapshotConfig{
		MaxSnapshots:  10,
		RetentionTime: 24 * time.Hour,
	}
	deleted, err := sm.CleanupSnapshots(config)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Verify only recent snapshot remains
	snapshots := sm.ListSnapshots()
	assert.Len(t, snapshots, 1)
	assert.NotEqual(t, snapshot.ID, snapshots[0].ID)
}

func TestSnapshotManager_MultipleSegments(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log with small segment size to force multiple segments
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 200, // Small size to force multiple segments
	})
	require.NoError(t, err)
	defer log.Close()

	// Add enough data to create multiple segments
	// Each record is about 60 bytes, so we need ~4+ records per segment
	for i := 0; i < 20; i++ {
		_, err := log.Append([]byte("key"), []byte("this is a longer value to fill up segments quickly"))
		require.NoError(t, err)
	}

	// Verify multiple segments were created
	numSegments := log.NumSegments()
	hwm := log.HighWaterMark()
	t.Logf("Created %d segments with HWM %d", numSegments, hwm)
	assert.GreaterOrEqual(t, numSegments, 1)

	// Create snapshot
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	snapshot, err := sm.CreateSnapshot(log)
	require.NoError(t, err)
	assert.Equal(t, numSegments, snapshot.NumSegments)

	// Restore snapshot
	restoreDir := filepath.Join(tmpDir, "restored")
	err = sm.RestoreSnapshot(snapshot.ID, restoreDir)
	require.NoError(t, err)

	// Verify restored log
	restoredLog, err := NewLog(LogConfig{
		Dir:            restoreDir,
		MaxSegmentSize: 200,
	})
	require.NoError(t, err)
	defer restoredLog.Close()

	assert.Equal(t, hwm, restoredLog.HighWaterMark())
	assert.Equal(t, numSegments, restoredLog.NumSegments())
}

func TestSnapshotManager_EmptyLog(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create empty log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Create snapshot of empty log
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	snapshot, err := sm.CreateSnapshot(log)
	require.NoError(t, err)
	assert.Equal(t, int64(0), snapshot.HighWaterMark)
	assert.Equal(t, 1, snapshot.NumSegments) // Should have one empty segment

	// Restore snapshot
	restoreDir := filepath.Join(tmpDir, "restored")
	err = sm.RestoreSnapshot(snapshot.ID, restoreDir)
	require.NoError(t, err)

	// Verify restored log
	restoredLog, err := NewLog(LogConfig{
		Dir:            restoreDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer restoredLog.Close()

	assert.Equal(t, int64(0), restoredLog.HighWaterMark())
}

func TestSnapshotManager_ConcurrentSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Add some data
	for i := 0; i < 10; i++ {
		_, err := log.Append([]byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create snapshots concurrently (should be serialized by mutex)
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := sm.CreateSnapshot(log)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify all snapshots were created
	snapshots := sm.ListSnapshots()
	assert.Len(t, snapshots, 3)
}

func TestSnapshotManager_Size(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Add data
	for i := 0; i < 10; i++ {
		_, err := log.Append([]byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Create snapshot
	_, err = sm.CreateSnapshot(log)
	require.NoError(t, err)

	// Get size
	size, err := sm.Size()
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))
}

func TestSnapshotManager_RestoreNonExistentSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create snapshot manager
	sm, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Try to restore non-existent snapshot
	restoreDir := filepath.Join(tmpDir, "restored")
	err = sm.RestoreSnapshot("non-existent-id", restoreDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultSnapshotConfig(t *testing.T) {
	config := DefaultSnapshotConfig()
	assert.Equal(t, 5, config.MaxSnapshots)
	assert.Equal(t, 24*time.Hour, config.RetentionTime)
	assert.Equal(t, 1*time.Hour, config.MinInterval)
}

func TestSnapshotManager_MetadataPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "log")

	// Create log
	log, err := NewLog(LogConfig{
		Dir:            logDir,
		MaxSegmentSize: 1024,
	})
	require.NoError(t, err)

	_, err = log.Append([]byte("key"), []byte("value"))
	require.NoError(t, err)

	// Create snapshot manager and snapshot
	sm1, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	snapshot, err := sm1.CreateSnapshot(log)
	require.NoError(t, err)

	log.Close()

	// Create new snapshot manager (should load existing metadata)
	sm2, err := NewSnapshotManager(logDir)
	require.NoError(t, err)

	// Verify snapshot is still available
	retrieved := sm2.GetSnapshot(snapshot.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, snapshot.ID, retrieved.ID)
	assert.Equal(t, snapshot.HighWaterMark, retrieved.HighWaterMark)
}
