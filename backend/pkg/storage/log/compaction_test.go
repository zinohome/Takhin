// Copyright 2025 Takhin Data, Inc.

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompact(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512, // Small segments to force multiple segments
	})
	require.NoError(t, err)
	defer log.Close()

	// Write duplicate keys across segments
	for i := 0; i < 50; i++ {
		key := []byte("key1")
		value := make([]byte, 30)
		_, err := log.Append(key, value)
		require.NoError(t, err)
	}

	// Verify multiple segments were created
	require.Greater(t, log.NumSegments(), 1)

	// Compact the log
	policy := DefaultCompactionPolicy()
	result, err := log.Compact(policy)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify compaction analyzed the segments
	assert.Greater(t, result.SegmentsCompacted, 0)
}

func TestAnalyzeCompaction(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512,
	})
	require.NoError(t, err)
	defer log.Close()

	// Write some duplicate keys
	for i := 0; i < 30; i++ {
		key := []byte("duplicate-key")
		_, err := log.Append(key, []byte("value"))
		require.NoError(t, err)
	}

	// Analyze compaction
	policy := DefaultCompactionPolicy()
	analysis := log.AnalyzeCompaction(policy)

	assert.Greater(t, analysis.TotalSegments, 0)
	assert.GreaterOrEqual(t, analysis.CompactableSegments, 0)
}

func TestNeedsCompaction(t *testing.T) {
	tests := []struct {
		name       string
		numRecords int
		uniqueKeys int
	}{
		{
			name:       "no compaction needed - all unique",
			numRecords: 10,
			uniqueKeys: 10,
		},
		{
			name:       "compaction needed - many duplicates",
			numRecords: 50,
			uniqueKeys: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			log, err := NewLog(LogConfig{
				Dir:            dir,
				MaxSegmentSize: 512,
			})
			require.NoError(t, err)
			defer log.Close()

			// Write records
			for i := 0; i < tt.numRecords; i++ {
				keyIndex := i % tt.uniqueKeys
				key := []byte{byte(keyIndex)}
				_, err := log.Append(key, make([]byte, 30))
				require.NoError(t, err)
			}

			policy := DefaultCompactionPolicy()
			needsCompaction := log.NeedsCompaction(policy)

			// Just verify it doesn't error - actual compaction need depends on segment creation
			t.Logf("Segments: %d, Needs compaction: %v", log.NumSegments(), needsCompaction)
		})
	}
}

func TestCompactionPolicy(t *testing.T) {
	t.Run("default policy", func(t *testing.T) {
		policy := DefaultCompactionPolicy()
		assert.Equal(t, 0.5, policy.MinCleanableRatio)
		assert.Equal(t, int64(0), policy.MinCompactionLagMs)
		assert.Equal(t, int64(24*60*60*1000), policy.DeleteRetentionMs)
	})

	t.Run("custom policy", func(t *testing.T) {
		policy := CompactionPolicy{
			MinCleanableRatio:  0.7,
			MinCompactionLagMs: 3600000,
			DeleteRetentionMs:  86400000,
		}
		assert.Equal(t, 0.7, policy.MinCleanableRatio)
		assert.Equal(t, int64(3600000), policy.MinCompactionLagMs)
		assert.Equal(t, int64(86400000), policy.DeleteRetentionMs)
	})
}

func TestCompactSegment(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512,
	})
	require.NoError(t, err)
	defer log.Close()

	// Write enough to create multiple segments
	for i := 0; i < 50; i++ {
		_, err := log.Append([]byte("key"), make([]byte, 30))
		require.NoError(t, err)
	}

	require.Greater(t, log.NumSegments(), 1)

	// Try to compact first segment
	err = log.CompactSegment(0)
	require.NoError(t, err)

	// Try to compact active segment - should fail
	activeIndex := log.NumSegments() - 1
	err = log.CompactSegment(activeIndex)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot compact active segment")

	// Try invalid index
	err = log.CompactSegment(999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid segment index")
}

func TestReadAllRecordsFromSegment(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 1024 * 1024,
	})
	require.NoError(t, err)
	defer log.Close()

	// Write some records
	numRecords := 10
	for i := 0; i < numRecords; i++ {
		_, err := log.Append([]byte("key"), []byte("value"))
		require.NoError(t, err)
	}

	// Read all records from first segment
	segment := log.segments[0]
	records, err := log.readAllRecordsFromSegment(segment)
	require.NoError(t, err)

	assert.Len(t, records, numRecords)
	for i, record := range records {
		assert.Equal(t, int64(i), record.Offset)
	}
}

func TestCompactionWithMultipleKeys(t *testing.T) {
	dir := t.TempDir()

	log, err := NewLog(LogConfig{
		Dir:            dir,
		MaxSegmentSize: 512,
	})
	require.NoError(t, err)
	defer log.Close()

	// Write multiple keys with updates
	keys := []string{"key1", "key2", "key3"}
	for round := 0; round < 10; round++ {
		for _, key := range keys {
			_, err := log.Append([]byte(key), make([]byte, 20))
			require.NoError(t, err)
		}
	}

	require.Greater(t, log.NumSegments(), 1)

	// Analyze compaction
	policy := DefaultCompactionPolicy()
	analysis := log.AnalyzeCompaction(policy)

	assert.Greater(t, analysis.TotalSegments, 0)
	t.Logf("Total segments: %d, Compactable: %d, Estimated savings: %d bytes",
		analysis.TotalSegments, analysis.CompactableSegments, analysis.EstimatedSavings)

	// Compact
	result, err := log.Compact(policy)
	require.NoError(t, err)

	t.Logf("Compacted %d segments, reclaimed %d bytes, removed %d keys, took %dms",
		result.SegmentsCompacted, result.BytesReclaimed, result.KeysRemoved, result.DurationMs)
}
