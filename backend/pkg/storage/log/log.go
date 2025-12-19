// Copyright 2025 Takhin Data, Inc.

package log

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Log struct {
	dir            string
	segments       []*Segment
	activeSegment  *Segment
	maxSegmentSize int64
	mu             sync.RWMutex
}

type LogConfig struct {
	Dir            string
	MaxSegmentSize int64
}

func NewLog(config LogConfig) (*Log, error) {
	log := &Log{
		dir:            config.Dir,
		maxSegmentSize: config.MaxSegmentSize,
		segments:       make([]*Segment, 0),
	}

	if len(log.segments) == 0 {
		if err := log.newSegment(0); err != nil {
			return nil, fmt.Errorf("create initial segment: %w", err)
		}
	}

	return log, nil
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
	defer l.mu.RUnlock()

	segment := l.findSegment(offset)
	if segment == nil {
		return nil, fmt.Errorf("offset not found: %d", offset)
	}

	return segment.Read(offset)
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
