// Copyright 2025 Takhin Data, Inc.

package log

import (
	"fmt"
	"os"
	"time"
)

// RetentionPolicy defines the retention policy for log segments
type RetentionPolicy struct {
	// RetentionBytes is the maximum size in bytes for the log
	// Segments older than this will be deleted. -1 means no limit.
	RetentionBytes int64

	// RetentionMs is the maximum age in milliseconds for the log
	// Segments older than this will be deleted. -1 means no limit.
	RetentionMs int64

	// MinCompactionLagMs is the minimum time a message will remain uncompacted
	MinCompactionLagMs int64
}

// DefaultRetentionPolicy returns the default retention policy
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		RetentionBytes:     -1,                      // No size limit
		RetentionMs:        7 * 24 * 60 * 60 * 1000, // 7 days
		MinCompactionLagMs: 0,
	}
}

// DeleteSegmentsIfNeeded deletes old segments based on retention policy
func (l *Log) DeleteSegmentsIfNeeded(policy RetentionPolicy) (int, int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.segments) <= 1 {
		// Always keep at least one segment (the active one)
		return 0, 0, nil
	}

	deletedCount := 0
	deletedBytes := int64(0)
	now := time.Now().UnixMilli()

	// Calculate total log size
	totalSize := int64(0)
	for _, segment := range l.segments {
		size, err := segment.Size()
		if err != nil {
			return deletedCount, deletedBytes, fmt.Errorf("get segment size: %w", err)
		}
		totalSize += size
	}

	// Delete segments based on retention policy
	// Keep segments from the end (most recent), delete from the beginning (oldest)
	var toDelete []*Segment
	remainingSize := totalSize

	for i := 0; i < len(l.segments)-1; i++ { // Never delete the last (active) segment
		segment := l.segments[i]
		segmentSize, _ := segment.Size()

		shouldDelete := false

		// Check size-based retention
		if policy.RetentionBytes > 0 && remainingSize > policy.RetentionBytes {
			shouldDelete = true
		}

		// Check time-based retention
		if policy.RetentionMs > 0 {
			// Get the last record timestamp in this segment
			// For simplicity, we use the segment's base offset timestamp
			// In production, we'd check the actual last record timestamp
			segmentAge := now - segment.baseOffset
			if segmentAge > policy.RetentionMs {
				shouldDelete = true
			}
		}

		if shouldDelete {
			toDelete = append(toDelete, segment)
			remainingSize -= segmentSize
			deletedBytes += segmentSize
		} else {
			// Stop deleting once we're within limits
			break
		}
	}

	// Actually delete the segments
	for _, segment := range toDelete {
		if err := l.deleteSegment(segment); err != nil {
			return deletedCount, deletedBytes, fmt.Errorf("delete segment: %w", err)
		}
		deletedCount++
	}

	// Update segments slice
	if deletedCount > 0 {
		l.segments = l.segments[deletedCount:]
	}

	return deletedCount, deletedBytes, nil
}

// deleteSegment removes a segment and its associated files
func (l *Log) deleteSegment(segment *Segment) error {
	// Close the segment first
	if err := segment.Close(); err != nil {
		return fmt.Errorf("close segment: %w", err)
	}

	// Delete data file
	dataPath := segment.dataFile.Name()
	if err := os.Remove(dataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove data file: %w", err)
	}

	// Delete index file
	indexPath := segment.indexFile.Name()
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove index file: %w", err)
	}

	// Delete time index file
	timeIndexPath := segment.timeIndexFile.Name()
	if err := os.Remove(timeIndexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove time index file: %w", err)
	}

	return nil
}

// OldestSegmentAge returns the age of the oldest segment in milliseconds
func (l *Log) OldestSegmentAge() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.segments) == 0 {
		return 0
	}

	// Get the oldest segment and read its first record to get timestamp
	oldestSegment := l.segments[0]
	oldestSegment.mu.RLock()
	defer oldestSegment.mu.RUnlock()

	// Try to read first record's timestamp
	if stat, err := oldestSegment.dataFile.Stat(); err == nil && stat.Size() > 0 {
		oldestSegment.dataFile.Seek(0, 0)
		if record, err := decodeRecord(oldestSegment.dataFile); err == nil {
			now := time.Now().UnixMilli()
			return now - record.Timestamp
		}
	}

	// Fallback: use current time (segment just created)
	return 0
}

// TruncateTo truncates the log to the given offset
// This removes all records with offset >= the given offset
func (l *Log) TruncateTo(offset int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Find the segment containing the offset
	idx := l.findSegmentIndex(offset)
	if idx == -1 {
		// Offset is before all segments, nothing to truncate
		return nil
	}

	// Truncate the found segment
	segment := l.segments[idx]
	if err := segment.TruncateTo(offset); err != nil {
		return fmt.Errorf("truncate segment: %w", err)
	}

	// Delete all segments after this one
	for i := idx + 1; i < len(l.segments); i++ {
		if err := l.deleteSegment(l.segments[i]); err != nil {
			return fmt.Errorf("delete segment: %w", err)
		}
	}

	// Update segments slice
	l.segments = l.segments[:idx+1]
	l.activeSegment = segment

	return nil
}
