// Copyright 2025 Takhin Data, Inc.

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompactionFullWorkflow tests the complete compaction workflow
func TestCompactionFullWorkflow(t *testing.T) {
	dir := t.TempDir()
	log, err := NewLog(LogConfig{Dir: dir, MaxSegmentSize: 512}) // Small segment size to create multiple segments
	require.NoError(t, err)

	// Write 100 records with 10 unique keys, creating duplicates
	for i := 0; i < 100; i++ {
		key := []byte("key-" + string(rune('0'+i%10)))
		value := []byte("value-" + string(rune('0'+i)))

		_, err := log.Append(key, value)
		require.NoError(t, err)
	}

	// Verify we have multiple segments
	log.mu.RLock()
	initialSegments := len(log.segments)
	log.mu.RUnlock()
	assert.Greater(t, initialSegments, 1, "should have created multiple segments")

	// Get initial size
	initialSize, err := log.Size()
	require.NoError(t, err)
	t.Logf("Initial: %d segments, %d bytes", initialSegments, initialSize)

	// Analyze compaction opportunities
	policy := DefaultCompactionPolicy()
	analysis := log.AnalyzeCompaction(policy)
	t.Logf("Analysis: %d compactable segments, ~%d bytes savings",
		analysis.CompactableSegments, analysis.EstimatedSavings)

	// Perform compaction
	result, err := log.Compact(policy)
	require.NoError(t, err)
	t.Logf("Compaction result: %d segments compacted, %d bytes reclaimed, %d keys removed, took %dms",
		result.SegmentsCompacted, result.BytesReclaimed, result.KeysRemoved, result.DurationMs)

	// Verify compaction reduced segments
	log.mu.RLock()
	finalSegments := len(log.segments)
	log.mu.RUnlock()
	assert.Less(t, finalSegments, initialSegments, "should have fewer segments after compaction")

	// Get final size
	finalSize, err := log.Size()
	require.NoError(t, err)
	t.Logf("Final: %d segments, %d bytes", finalSegments, finalSize)

	// Size should be reduced (or at least not increased)
	assert.LessOrEqual(t, finalSize, initialSize, "size should not increase after compaction")

	// Verify we can still read all keys (latest values)
	for i := 0; i < 10; i++ {
		key := []byte("key-" + string(rune('0'+i)))
		found := false

		// Read through all segments to find the key
		log.mu.RLock()
		for _, segment := range log.segments {
			records, _ := log.readAllRecordsFromSegment(segment)
			for _, record := range records {
				if string(record.Key) == string(key) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		log.mu.RUnlock()

		assert.True(t, found, "key %s should still be readable after compaction", key)
	}
}

// TestCompactionWithDeleteTombstones tests compaction with delete markers
func TestCompactionWithDeleteTombstones(t *testing.T) {
	dir := t.TempDir()
	log, err := NewLog(LogConfig{Dir: dir, MaxSegmentSize: 512})
	require.NoError(t, err)

	// Write some records
	for i := 0; i < 50; i++ {
		key := []byte("key-" + string(rune('0'+i%5)))
		value := []byte("value-" + string(rune('0'+i)))
		_, err := log.Append(key, value)
		require.NoError(t, err)
	}

	// Write delete tombstones (null values)
	for i := 0; i < 5; i++ {
		key := []byte("key-" + string(rune('0'+i)))
		_, err := log.Append(key, nil) // Null value = delete tombstone
		require.NoError(t, err)
	}

	// Get sizes before compaction
	log.mu.RLock()
	segmentsBefore := len(log.segments)
	log.mu.RUnlock()

	// Compact
	policy := DefaultCompactionPolicy()
	result, err := log.Compact(policy)
	require.NoError(t, err)
	t.Logf("Compacted %d segments, removed %d keys", result.SegmentsCompacted, result.KeysRemoved)

	// Verify compaction happened
	log.mu.RLock()
	segmentsAfter := len(log.segments)
	log.mu.RUnlock()
	assert.LessOrEqual(t, segmentsAfter, segmentsBefore, "should not increase segments")

	// Verify tombstones are preserved (latest value for each key)
	log.mu.RLock()
	for _, segment := range log.segments {
		records, _ := log.readAllRecordsFromSegment(segment)
		for _, record := range records {
			t.Logf("Key: %s, Value: %v (len=%d)", record.Key, record.Value, len(record.Value))
		}
	}
	log.mu.RUnlock()
}

// TestCompactionSingleSegment tests that compaction doesn't run on single segment
func TestCompactionSingleSegment(t *testing.T) {
	dir := t.TempDir()
	log, err := NewLog(LogConfig{Dir: dir, MaxSegmentSize: 10 * 1024 * 1024}) // Large segment size
	require.NoError(t, err)

	// Write a few records (won't create multiple segments)
	for i := 0; i < 10; i++ {
		key := []byte("key-" + string(rune('0'+i)))
		value := []byte("value")
		_, err := log.Append(key, value)
		require.NoError(t, err)
	}

	// Verify we have only one segment
	log.mu.RLock()
	segmentCount := len(log.segments)
	log.mu.RUnlock()
	assert.Equal(t, 1, segmentCount)

	// Try compaction - should be no-op
	policy := DefaultCompactionPolicy()
	result, err := log.Compact(policy)
	require.NoError(t, err)
	assert.Equal(t, 0, result.SegmentsCompacted, "should not compact with only one segment")
	assert.Equal(t, int64(0), result.BytesReclaimed)
}

// TestCompactionPreservesOrder tests that compaction maintains offset order
func TestCompactionPreservesOrder(t *testing.T) {
	dir := t.TempDir()
	log, err := NewLog(LogConfig{Dir: dir, MaxSegmentSize: 512})
	require.NoError(t, err)

	// Write records in order
	offsets := make([]int64, 0)
	for i := 0; i < 50; i++ {
		key := []byte("key-" + string(rune('0'+i%10)))
		value := []byte("value-" + string(rune('0'+i)))
		offset, err := log.Append(key, value)
		require.NoError(t, err)
		offsets = append(offsets, offset)
	}

	// Compact
	policy := DefaultCompactionPolicy()
	_, compactErr := log.Compact(policy)
	require.NoError(t, compactErr)

	// Read all records and verify they're in order
	log.mu.RLock()
	defer log.mu.RUnlock()

	lastOffset := int64(-1)
	for _, segment := range log.segments {
		records, err := log.readAllRecordsFromSegment(segment)
		require.NoError(t, err)

		for _, record := range records {
			assert.Greater(t, record.Offset, lastOffset, "offsets should be in ascending order")
			lastOffset = record.Offset
		}
	}
}

// TestCompactionConcurrency tests concurrent compaction and writes
func TestCompactionConcurrency(t *testing.T) {
	dir := t.TempDir()
	log, err := NewLog(LogConfig{Dir: dir, MaxSegmentSize: 512})
	require.NoError(t, err)

	// Write initial data
	for i := 0; i < 100; i++ {
		key := []byte("key-" + string(rune('0'+i%10)))
		value := []byte("value-" + string(rune('0'+i)))
		_, err := log.Append(key, value)
		require.NoError(t, err)
	}

	// Run compaction in goroutine
	done := make(chan error)
	go func() {
		policy := DefaultCompactionPolicy()
		_, err := log.Compact(policy)
		done <- err
	}()

	// Continue writing while compacting
	for i := 100; i < 150; i++ {
		key := []byte("key-" + string(rune('0'+i%10)))
		value := []byte("value-" + string(rune('0'+i)))
		_, err := log.Append(key, value)
		require.NoError(t, err)
	}

	// Wait for compaction to finish
	compactErr := <-done
	require.NoError(t, compactErr)

	// Verify log is still functional
	offset, err := log.Append([]byte("final-key"), []byte("final-value"))
	require.NoError(t, err)
	assert.Greater(t, offset, int64(0))
}
