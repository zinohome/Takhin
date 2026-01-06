// Copyright 2025 Takhin Data, Inc.

package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type TierManager interface {
	RecordAccess(segmentPath string, readBytes int64)
	DetermineTier(segmentPath string, segmentAge time.Duration) string
	PromoteSegment(ctx context.Context, segmentPath string, fromTier, toTier string) error
}

type Log struct {
	dir            string
	segments       []*Segment
	activeSegment  *Segment
	maxSegmentSize int64
	tierManager    TierManager
	mu             sync.RWMutex
}

type LogConfig struct {
	Dir            string
	MaxSegmentSize int64
	TierManager    TierManager
}

func NewLog(config LogConfig) (*Log, error) {
	log := &Log{
		dir:            config.Dir,
		maxSegmentSize: config.MaxSegmentSize,
		tierManager:    config.TierManager,
		segments:       make([]*Segment, 0),
	}

	// Try to load existing segments from disk
	if err := log.loadExistingSegments(); err != nil {
		return nil, fmt.Errorf("load existing segments: %w", err)
	}

	// If no segments exist, create a new one
	if len(log.segments) == 0 {
		if err := log.newSegment(0); err != nil {
			return nil, fmt.Errorf("create initial segment: %w", err)
		}
	}

	return log, nil
}

// loadExistingSegments loads segments from disk
func (l *Log) loadExistingSegments() error {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read directory: %w", err)
	}

	// Find all .log files and extract base offsets
	segmentOffsets := make([]int64, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) >= 24 && name[len(name)-4:] == ".log" {
			var offset int64
			if _, err := fmt.Sscanf(name, "%020d.log", &offset); err == nil {
				segmentOffsets = append(segmentOffsets, offset)
			}
		}
	}

	// Sort offsets
	sort.Slice(segmentOffsets, func(i, j int) bool {
		return segmentOffsets[i] < segmentOffsets[j]
	})

	// Load each segment
	for _, offset := range segmentOffsets {
		segment, err := NewSegment(SegmentConfig{
			BaseOffset: offset,
			MaxBytes:   l.maxSegmentSize,
			Dir:        l.dir,
		})
		if err != nil {
			return fmt.Errorf("load segment at offset %d: %w", offset, err)
		}
		l.segments = append(l.segments, segment)
	}

	// Set active segment to the last one
	if len(l.segments) > 0 {
		l.activeSegment = l.segments[len(l.segments)-1]
	}

	return nil
}

func (l *Log) Append(key, value []byte) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.activeSegment.IsFull() {
		nextOffset := l.activeSegment.NextOffset()
		if err := l.newSegment(nextOffset); err != nil {
			return 0, fmt.Errorf("create new segment: %w", err)
		}
	}

	record := &Record{
		Timestamp: time.Now().UnixMilli(),
		Key:       key,
		Value:     value,
	}

	offset, err := l.activeSegment.Append(record)
	if err != nil {
		return 0, fmt.Errorf("append to segment: %w", err)
	}

	return offset, nil
}

// AppendBatch appends multiple records in a single batch for better performance
func (l *Log) AppendBatch(records []struct{ Key, Value []byte }) ([]int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(records) == 0 {
		return nil, nil
	}

	// Prepare records with timestamps
	batch := make([]*Record, len(records))
	now := time.Now().UnixMilli()
	for i, r := range records {
		batch[i] = &Record{
			Timestamp: now,
			Key:       r.Key,
			Value:     r.Value,
		}
	}

	// Check if we need to split the batch across segments
	var allOffsets []int64
	remaining := batch

	for len(remaining) > 0 {
		if l.activeSegment.IsFull() {
			nextOffset := l.activeSegment.NextOffset()
			if err := l.newSegment(nextOffset); err != nil {
				return nil, fmt.Errorf("create new segment: %w", err)
			}
		}

		// Try to append as many records as possible to current segment
		offsets, err := l.activeSegment.AppendBatch(remaining)
		if err != nil {
			// If batch is too large, try with single record
			if len(remaining) == 1 {
				return nil, fmt.Errorf("append batch: %w", err)
			}
			// Split batch in half and retry
			mid := len(remaining) / 2
			offsets, err = l.activeSegment.AppendBatch(remaining[:mid])
			if err != nil {
				return nil, fmt.Errorf("append batch: %w", err)
			}
			allOffsets = append(allOffsets, offsets...)
			remaining = remaining[mid:]
		} else {
			allOffsets = append(allOffsets, offsets...)
			remaining = nil
		}
	}

	return allOffsets, nil
}

func (l *Log) Read(offset int64) (*Record, error) {
	l.mu.RLock()
	segment := l.findSegment(offset)
	if segment == nil {
		l.mu.RUnlock()
		return nil, fmt.Errorf("offset not found: %d", offset)
	}
	
	// Track access for tier management
	if l.tierManager != nil {
		segmentPath, _ := filepath.Rel(filepath.Dir(l.dir), segment.Path())
		l.tierManager.RecordAccess(segmentPath, 0)
	}
	l.mu.RUnlock()

	return segment.Read(offset)
}

// ReadRange reads multiple records from startOffset up to maxBytes.
// Returns the segment and position/size for zero-copy transfer.
func (l *Log) ReadRange(offset int64, maxBytes int64) (*Segment, int64, int64, error) {
	l.mu.RLock()
	segment := l.findSegment(offset)
	if segment == nil {
		l.mu.RUnlock()
		return nil, 0, 0, fmt.Errorf("offset not found: %d", offset)
	}
	
	// Track access for tier management
	if l.tierManager != nil {
		segmentPath, _ := filepath.Rel(filepath.Dir(l.dir), segment.Path())
		l.tierManager.RecordAccess(segmentPath, maxBytes)
	}
	l.mu.RUnlock()

	position, size, err := segment.ReadRange(offset, maxBytes)
	if err != nil {
		return nil, 0, 0, err
	}

	return segment, position, size, nil
}

func (l *Log) HighWaterMark() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.activeSegment != nil {
		return l.activeSegment.NextOffset()
	}
	return 0
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	for _, segment := range l.segments {
		if err := segment.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close segments: %v", errs)
	}
	return nil
}

func (l *Log) newSegment(baseOffset int64) error {
	config := SegmentConfig{
		BaseOffset: baseOffset,
		MaxBytes:   l.maxSegmentSize,
		Dir:        l.dir,
	}

	segment, err := NewSegment(config)
	if err != nil {
		return err
	}

	l.segments = append(l.segments, segment)
	l.activeSegment = segment

	return nil
}

func (l *Log) findSegment(offset int64) *Segment {
	idx := l.findSegmentIndex(offset)
	if idx == -1 {
		return nil
	}
	return l.segments[idx]
}

func (l *Log) findSegmentIndex(offset int64) int {
	idx := sort.Search(len(l.segments), func(i int) bool {
		return l.segments[i].BaseOffset() > offset
	})

	if idx == 0 {
		return -1
	}
	return idx - 1
}

// SearchByTimestamp searches for the first offset whose timestamp >= the given timestamp
func (l *Log) SearchByTimestamp(timestamp int64) (int64, int64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.segments) == 0 {
		return 0, 0, fmt.Errorf("no segments available")
	}

	// Search through segments to find the one containing the timestamp
	for _, segment := range l.segments {
		// Try to find in this segment's time index
		offset, actualTimestamp, err := segment.SearchByTimestamp(timestamp)
		if err == nil {
			return offset, actualTimestamp, nil
		}
	}

	// If not found, return the next available offset (HWM)
	return l.HighWaterMark(), timestamp, nil
}

// Size returns the total size in bytes of all segments
func (l *Log) Size() (int64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	totalSize := int64(0)
	for _, segment := range l.segments {
		size, err := segment.Size()
		if err != nil {
			return 0, fmt.Errorf("get segment size: %w", err)
		}
		totalSize += size
	}

	return totalSize, nil
}

// NumSegments returns the number of segments in this log
func (l *Log) NumSegments() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.segments)
}

// GetSegments returns information about all segments
func (l *Log) GetSegments() []SegmentInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()

	infos := make([]SegmentInfo, len(l.segments))
	for i, segment := range l.segments {
		size, _ := segment.Size()
		infos[i] = SegmentInfo{
			BaseOffset: segment.BaseOffset(),
			NextOffset: segment.NextOffset(),
			Size:       size,
		}
	}

	return infos
}

// SegmentInfo contains information about a segment
type SegmentInfo struct {
	BaseOffset int64
	NextOffset int64
	Size       int64
}
