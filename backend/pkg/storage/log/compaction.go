// Copyright 2025 Takhin Data, Inc.

package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CompactionPolicy defines the compaction policy for log segments
type CompactionPolicy struct {
	// MinCleanableRatio is the minimum ratio of dirty (uncompacted) data
	// before a segment is eligible for compaction (0.0 to 1.0)
	MinCleanableRatio float64

	// MinCompactionLagMs is the minimum time a message must exist before compaction
	MinCompactionLagMs int64

	// DeleteRetentionMs is how long to retain delete tombstones
	DeleteRetentionMs int64
}

// DefaultCompactionPolicy returns the default compaction policy
func DefaultCompactionPolicy() CompactionPolicy {
	return CompactionPolicy{
		MinCleanableRatio:  0.5,                 // Compact when 50% is dirty
		MinCompactionLagMs: 0,                   // No lag by default
		DeleteRetentionMs:  24 * 60 * 60 * 1000, // 24 hours
	}
}

// CompactionResult contains the results of a compaction operation
type CompactionResult struct {
	SegmentsCompacted int
	BytesReclaimed    int64
	KeysRemoved       int
	DurationMs        int64
}

// Compact performs log compaction on eligible segments
// This removes duplicate keys, keeping only the latest value for each key
func (l *Log) Compact(policy CompactionPolicy) (*CompactionResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	startTime := time.Now()
	result := &CompactionResult{}

	if len(l.segments) <= 1 {
		// Need at least 2 segments (keep active segment uncompacted)
		return result, nil
	}

	// Don't compact the active segment
	eligibleSegments := l.segments[:len(l.segments)-1]

	// Build key map from all eligible segments
	// Map key -> latest record
	keyMap := make(map[string]*Record)
	totalBytes := int64(0)

	for _, segment := range eligibleSegments {
		records, err := l.readAllRecordsFromSegment(segment)
		if err != nil {
			return nil, fmt.Errorf("read segment: %w", err)
		}

		for _, record := range records {
			if len(record.Key) == 0 {
				continue // Skip records without keys
			}

			key := string(record.Key)

			// Check compaction lag
			if policy.MinCompactionLagMs > 0 {
				age := time.Now().UnixMilli() - record.Timestamp
				if age < policy.MinCompactionLagMs {
					continue // Too recent to compact
				}
			}

			// Keep only the latest value for each key
			if existing, exists := keyMap[key]; !exists || record.Offset > existing.Offset {
				keyMap[key] = record
			}
		}

		size, _ := segment.Size()
		totalBytes += size
	}

	// Calculate bytes saved (original size - unique keys)
	uniqueBytes := int64(0)
	for _, record := range keyMap {
		uniqueBytes += int64(len(record.Key) + len(record.Value) + 24) // Approximate
	}

	// Get unique records in offset order
	uniqueRecords := make([]*Record, 0, len(keyMap))
	for _, record := range keyMap {
		uniqueRecords = append(uniqueRecords, record)
	}

	sort.Slice(uniqueRecords, func(i, j int) bool {
		return uniqueRecords[i].Offset < uniqueRecords[j].Offset
	})

	if len(uniqueRecords) == 0 {
		result.DurationMs = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// Create a new compacted segment starting from the first offset
	firstOffset := uniqueRecords[0].Offset
	compactedSegment, err := NewSegment(SegmentConfig{
		BaseOffset: firstOffset,
		MaxBytes:   l.maxSegmentSize,
		Dir:        l.dir,
	})
	if err != nil {
		return nil, fmt.Errorf("create compacted segment: %w", err)
	}

	// Write unique records to the new segment
	keysWritten := 0
	for _, record := range uniqueRecords {
		_, err := compactedSegment.AppendBatch([]*Record{record})
		if err != nil {
			compactedSegment.Close()
			return nil, fmt.Errorf("write record: %w", err)
		}
		keysWritten++
	}

	// Sync the new segment to disk
	if err := compactedSegment.dataFile.Sync(); err != nil {
		compactedSegment.Close()
		return nil, fmt.Errorf("sync compacted segment: %w", err)
	}

	// Close old segments and delete their files
	oldSegmentPaths := make([]string, 0, len(eligibleSegments))
	for _, segment := range eligibleSegments {
		oldSegmentPaths = append(oldSegmentPaths, segment.dataFile.Name())
		segment.Close()
	}

	// Replace old segments with the compacted one
	newSegments := make([]*Segment, 0, len(l.segments)-len(eligibleSegments)+1)
	newSegments = append(newSegments, compactedSegment)
	// Keep the active segment
	newSegments = append(newSegments, l.segments[len(l.segments)-1])
	l.segments = newSegments

	// Delete old segment files
	for _, path := range oldSegmentPaths {
		if err := deleteSegmentFiles(path); err != nil {
			// Log error but continue
			fmt.Printf("failed to delete old segment files: %v\n", err)
		}
	}

	result.BytesReclaimed = totalBytes - uniqueBytes
	result.SegmentsCompacted = len(eligibleSegments)
	result.KeysRemoved = len(eligibleSegments)*len(uniqueRecords) - keysWritten
	result.DurationMs = time.Since(startTime).Milliseconds()

	return result, nil
}

// readAllRecordsFromSegment reads all records from a segment
func (l *Log) readAllRecordsFromSegment(segment *Segment) ([]*Record, error) {
	segment.mu.RLock()
	defer segment.mu.RUnlock()

	records := make([]*Record, 0)

	// Reset file position
	if _, err := segment.dataFile.Seek(0, 0); err != nil {
		return nil, err
	}

	// Read all records
	for {
		record, err := decodeRecord(segment.dataFile)
		if err != nil {
			break // EOF or error
		}
		records = append(records, record)
	}

	return records, nil
}

// AnalyzeCompaction analyzes segments and returns compaction recommendations
func (l *Log) AnalyzeCompaction(policy CompactionPolicy) *CompactionAnalysis {
	l.mu.RLock()
	defer l.mu.RUnlock()

	analysis := &CompactionAnalysis{
		TotalSegments:       len(l.segments),
		CompactableSegments: 0,
		EstimatedSavings:    0,
	}

	if len(l.segments) <= 1 {
		return analysis
	}

	// Analyze each segment (except active one)
	for i := 0; i < len(l.segments)-1; i++ {
		segment := l.segments[i]
		records, err := l.readAllRecordsFromSegment(segment)
		if err != nil {
			continue
		}

		// Count unique keys
		keyMap := make(map[string]bool)
		for _, record := range records {
			if len(record.Key) > 0 {
				keyMap[string(record.Key)] = true
			}
		}

		totalRecords := len(records)
		uniqueKeys := len(keyMap)

		if totalRecords > 0 {
			dirtyRatio := float64(totalRecords-uniqueKeys) / float64(totalRecords)
			if dirtyRatio >= policy.MinCleanableRatio {
				analysis.CompactableSegments++
				size, _ := segment.Size()
				analysis.EstimatedSavings += int64(float64(size) * dirtyRatio)
			}
		}
	}

	return analysis
}

// CompactionAnalysis contains analysis of compaction opportunities
type CompactionAnalysis struct {
	TotalSegments       int
	CompactableSegments int
	EstimatedSavings    int64
}

// NeedsCompaction returns true if log needs compaction based on policy
func (l *Log) NeedsCompaction(policy CompactionPolicy) bool {
	analysis := l.AnalyzeCompaction(policy)
	return analysis.CompactableSegments > 0
}

// CompactSegment compacts a single segment by removing duplicate keys
func (l *Log) CompactSegment(segmentIndex int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if segmentIndex < 0 || segmentIndex >= len(l.segments) {
		return fmt.Errorf("invalid segment index: %d", segmentIndex)
	}

	if segmentIndex == len(l.segments)-1 {
		return fmt.Errorf("cannot compact active segment")
	}

	segment := l.segments[segmentIndex]

	// Read all records
	records, err := l.readAllRecordsFromSegment(segment)
	if err != nil {
		return fmt.Errorf("read records: %w", err)
	}

	// Build map of latest value for each key
	keyMap := make(map[string]*Record)
	for _, record := range records {
		if len(record.Key) == 0 {
			continue
		}
		key := string(record.Key)
		if existing, exists := keyMap[key]; !exists || record.Offset > existing.Offset {
			keyMap[key] = record
		}
	}

	// Get unique records in offset order
	uniqueRecords := make([]*Record, 0, len(keyMap))
	for _, record := range keyMap {
		uniqueRecords = append(uniqueRecords, record)
	}

	sort.Slice(uniqueRecords, func(i, j int) bool {
		return uniqueRecords[i].Offset < uniqueRecords[j].Offset
	})

	if len(uniqueRecords) == 0 {
		return nil // Nothing to write
	}

	// Create a new temp segment
	firstOffset := uniqueRecords[0].Offset
	tempPath := fmt.Sprintf("%s.%d.compacting", segment.dataFile.Name(), time.Now().Unix())
	tempSegment, err := createSegmentAtPath(tempPath, firstOffset)
	if err != nil {
		return fmt.Errorf("create temp segment: %w", err)
	}
	defer func() {
		if tempSegment != nil {
			tempSegment.Close()
		}
	}()

	// Write unique records to temp segment
	for _, record := range uniqueRecords {
		_, err := tempSegment.AppendBatch([]*Record{record})
		if err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	// Sync temp segment
	if err := tempSegment.dataFile.Sync(); err != nil {
		return fmt.Errorf("sync temp segment: %w", err)
	}

	// Close both segments
	oldPath := segment.dataFile.Name()
	tempSegment.Close()
	segment.Close()
	tempSegment = nil

	// Replace old segment files with temp segment
	if err := replaceSegmentFiles(oldPath, tempPath); err != nil {
		return fmt.Errorf("replace segment files: %w", err)
	}

	// Reopen the compacted segment
	newSegment, err := openSegment(oldPath)
	if err != nil {
		return fmt.Errorf("reopen segment: %w", err)
	}

	// Replace in segments list
	l.segments[segmentIndex] = newSegment

	return nil
}

// createSegmentAtPath creates a new segment at the specified path
func createSegmentAtPath(path string, baseOffset int64) (*Segment, error) {
	dir := filepath.Dir(path)
	baseName := filepath.Base(path)

	// Create directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	// Create data file
	dataFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("create data file: %w", err)
	}

	// Create index files
	indexPath := path[:len(path)-4] + ".index"
	timeIndexPath := path[:len(path)-4] + ".timeindex"

	indexFile, err := os.OpenFile(indexPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		dataFile.Close()
		return nil, fmt.Errorf("create index file: %w", err)
	}

	timeIndexFile, err := os.OpenFile(timeIndexPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		dataFile.Close()
		indexFile.Close()
		return nil, fmt.Errorf("create time index file: %w", err)
	}

	_ = baseName // Avoid unused warning

	return &Segment{
		baseOffset:    baseOffset,
		nextOffset:    baseOffset,
		dataFile:      dataFile,
		indexFile:     indexFile,
		timeIndexFile: timeIndexFile,
		maxBytes:      1024 * 1024 * 1024, // 1GB default
	}, nil
}

// openSegment opens an existing segment from disk
func openSegment(dataPath string) (*Segment, error) {
	// Extract base offset from filename
	baseName := filepath.Base(dataPath)
	offsetStr := baseName[:strings.Index(baseName, ".")]
	baseOffset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse base offset: %w", err)
	}

	// Open files
	dataFile, err := os.OpenFile(dataPath, os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open data file: %w", err)
	}

	indexPath := dataPath[:len(dataPath)-4] + ".index"
	indexFile, err := os.OpenFile(indexPath, os.O_RDWR, 0644)
	if err != nil {
		dataFile.Close()
		return nil, fmt.Errorf("open index file: %w", err)
	}

	timeIndexPath := dataPath[:len(dataPath)-4] + ".timeindex"
	timeIndexFile, err := os.OpenFile(timeIndexPath, os.O_RDWR, 0644)
	if err != nil {
		dataFile.Close()
		indexFile.Close()
		return nil, fmt.Errorf("open time index file: %w", err)
	}

	segment := &Segment{
		baseOffset:    baseOffset,
		nextOffset:    baseOffset,
		dataFile:      dataFile,
		indexFile:     indexFile,
		timeIndexFile: timeIndexFile,
		maxBytes:      1024 * 1024 * 1024,
	}

	// Scan to find next offset
	if err := segment.scanSegment(); err != nil {
		segment.Close()
		return nil, fmt.Errorf("scan segment: %w", err)
	}

	return segment, nil
}

// replaceSegmentFiles replaces old segment files with new ones
func replaceSegmentFiles(oldPath, newPath string) error {
	// Replace data file
	if err := os.Rename(newPath, oldPath); err != nil {
		return fmt.Errorf("rename data file: %w", err)
	}

	// Replace index file
	oldIndex := oldPath[:len(oldPath)-4] + ".index"
	newIndex := newPath[:len(newPath)-4] + ".index"
	if err := os.Rename(newIndex, oldIndex); err != nil {
		return fmt.Errorf("rename index file: %w", err)
	}

	// Replace time index file
	oldTimeIndex := oldPath[:len(oldPath)-4] + ".timeindex"
	newTimeIndex := newPath[:len(newPath)-4] + ".timeindex"
	if err := os.Rename(newTimeIndex, oldTimeIndex); err != nil {
		return fmt.Errorf("rename time index file: %w", err)
	}

	return nil
}

// deleteSegmentFiles deletes all files associated with a segment (data, index, timeindex)
func deleteSegmentFiles(dataPath string) error {
	// Delete data file
	if err := os.Remove(dataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove data file: %w", err)
	}

	// Delete index file
	indexPath := dataPath[:len(dataPath)-4] + ".index"
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove index file: %w", err)
	}

	// Delete time index file
	timeIndexPath := dataPath[:len(dataPath)-4] + ".timeindex"
	if err := os.Remove(timeIndexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove time index file: %w", err)
	}

	return nil
}
