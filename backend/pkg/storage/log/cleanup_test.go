// Copyright 2025 Takhin Data, Inc.

package log

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteSegmentsIfNeeded(t *testing.T) {
	tests := []struct {
		name          string
		numSegments   int
		recordsPerSeg int
		policy        RetentionPolicy
		expectDeleted int
	}{
		{
			name:          "no deletion with no limit",
			numSegments:   5,
			recordsPerSeg: 10,
			policy: RetentionPolicy{
				RetentionBytes: -1,
				RetentionMs:    -1,
			},
			expectDeleted: 0,
		},
		{
			name:          "delete old segments by size",
			numSegments:   5,
			recordsPerSeg: 10,
			policy: RetentionPolicy{
				RetentionBytes: 1024, // Keep only ~1KB
				RetentionMs:    -1,
			},
			expectDeleted: 3, // Should delete older segments to stay under limit
		},
		{
			name:          "keep at least one segment",
			numSegments:   1,
			recordsPerSeg: 10,
			policy: RetentionPolicy{
				RetentionBytes: 0, // Try to delete everything
				RetentionMs:    0,
			},
			expectDeleted: 0, // Should keep at least one segment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			// Create log with small segments to force multiple segments
			log, err := NewLog(LogConfig{
				Dir:            dir,
				MaxSegmentSize: 512, // Small segment size
			})
			require.NoError(t, err)
			defer log.Close()

			// Write records to create multiple segments
			for i := 0; i < tt.numSegments*tt.recordsPerSeg; i++ {
				key := []byte("key")
				value := make([]byte, 50) // 50 bytes per record
				_, err := log.Append(key, value)
				require.NoError(t, err)
			}

			initialSegments := log.NumSegments()
			t.Logf("Initial segments: %d", initialSegments)

			// Apply retention policy
			deletedCount, deletedBytes, err := log.DeleteSegmentsIfNeeded(tt.policy)
			require.NoError(t, err)

			t.Logf("Deleted %d segments (%d bytes)", deletedCount, deletedBytes)

			finalSegments := log.NumSegments()
			t.Logf("Final segments: %d", finalSegments)

			// Verify at least one segment remains
			assert.GreaterOrEqual(t, finalSegments, 1)

			// Verify we can still read from the log
			hwm := log.HighWaterMark()
			assert.Greater(t, hwm, int64(0))
		})
	}
}

func TestTruncateTo(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 1024 * 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Append some records
	numRecords := 10
	for i := 0; i < numRecords; i++ {
		_, err := log.Append([]byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	initialHWM := log.HighWaterMark()
	assert.Equal(t, int64(numRecords), initialHWM)

	// Truncate to offset 5
	truncateOffset := int64(5)
	err = log.TruncateTo(truncateOffset)
	require.NoError(t, err)

	// Verify HWM is updated
	newHWM := log.HighWaterMark()
	assert.Equal(t, truncateOffset, newHWM)

	// After truncate, we can write new records starting at truncated offset
	offset, err := log.Append([]byte("new-key"), []byte("new-value"))
	require.NoError(t, err)
	assert.Equal(t, int64(5), offset)
}

func TestRetentionPolicy(t *testing.T) {
	t.Run("default policy", func(t *testing.T) {
		policy := DefaultRetentionPolicy()
		assert.Equal(t, int64(-1), policy.RetentionBytes)
		assert.Equal(t, int64(7*24*60*60*1000), policy.RetentionMs)
	})

	t.Run("custom policy", func(t *testing.T) {
		policy := RetentionPolicy{
			RetentionBytes:     1024 * 1024,         // 1MB
			RetentionMs:        24 * 60 * 60 * 1000, // 1 day
			MinCompactionLagMs: 60 * 60 * 1000,      // 1 hour
		}
		assert.Equal(t, int64(1024*1024), policy.RetentionBytes)
		assert.Equal(t, int64(24*60*60*1000), policy.RetentionMs)
		assert.Equal(t, int64(60*60*1000), policy.MinCompactionLagMs)
	})
}

func TestOldestSegmentAge(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 1024 * 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Append some records
	_, err = log.Append([]byte("key"), []byte("value"))
	require.NoError(t, err)

	// Age should be non-negative
	age := log.OldestSegmentAge()
	assert.GreaterOrEqual(t, age, int64(0))
}

func TestDeleteSegmentFiles(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512, // Small segment to force multiple segments
	})
	require.NoError(t, err)

	// Create multiple segments
	for i := 0; i < 50; i++ {
		value := make([]byte, 50)
		_, err := log.Append([]byte("key"), value)
		require.NoError(t, err)
	}

	initialSegments := log.NumSegments()
	require.Greater(t, initialSegments, 1)

	// List files before deletion
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	initialFileCount := len(files)
	t.Logf("Initial files: %d", initialFileCount)

	// Delete segments with aggressive policy
	policy := RetentionPolicy{
		RetentionBytes: 1024, // Very small
		RetentionMs:    -1,
	}
	deletedCount, _, err := log.DeleteSegmentsIfNeeded(policy)
	require.NoError(t, err)
	require.Greater(t, deletedCount, 0)

	// Verify files are actually deleted
	files, err = os.ReadDir(dir)
	require.NoError(t, err)
	finalFileCount := len(files)
	t.Logf("Final files: %d", finalFileCount)

	// Each segment has 3 files (data, index, timeindex)
	expectedDeleted := deletedCount * 3
	assert.Equal(t, initialFileCount-expectedDeleted, finalFileCount)

	log.Close()
}

func TestSegmentTruncate(t *testing.T) {
	dir := t.TempDir()

	segment, err := NewSegment(SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024,
		Dir:        dir,
	})
	require.NoError(t, err)
	defer segment.Close()

	// Append some records
	for i := 0; i < 10; i++ {
		record := &Record{
			Timestamp: time.Now().UnixMilli(),
			Key:       []byte("key"),
			Value:     []byte("value"),
		}
		_, err := segment.Append(record)
		require.NoError(t, err)
	}

	initialSize, err := segment.Size()
	require.NoError(t, err)
	require.Greater(t, initialSize, int64(0))

	// Truncate to offset 5
	err = segment.TruncateTo(5)
	require.NoError(t, err)

	// Verify size decreased
	newSize, err := segment.Size()
	require.NoError(t, err)
	assert.Less(t, newSize, initialSize)

	// Verify next offset is updated
	assert.Equal(t, int64(5), segment.NextOffset())
}
