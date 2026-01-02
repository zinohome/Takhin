// Copyright 2025 Takhin Data, Inc.

package log

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrCorruptedSegment   = errors.New("segment data is corrupted")
	ErrCorruptedIndex     = errors.New("index is corrupted")
	ErrCorruptedTimeIndex = errors.New("time index is corrupted")
	ErrIncompleteRecord   = errors.New("incomplete record found")
	ErrIndexSizeMismatch  = errors.New("index size does not match data")
)

// SegmentRecovery handles recovery operations for segments
type SegmentRecovery struct {
	segment *Segment
}

// NewSegmentRecovery creates a new recovery handler for a segment
func NewSegmentRecovery(segment *Segment) *SegmentRecovery {
	return &SegmentRecovery{
		segment: segment,
	}
}

// RecoveryResult contains the results of a recovery operation
type RecoveryResult struct {
	RecordsRecovered   int64
	RecordsTruncated   int64
	IndexRebuilt       bool
	TimeIndexRebuilt   bool
	CorruptionDetected bool
	Errors             []error
}

// Recover performs full recovery on the segment
func (sr *SegmentRecovery) Recover() (*RecoveryResult, error) {
	result := &RecoveryResult{}

	// Step 1: Validate segment data integrity
	validRecords, err := sr.ValidateData()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("validate data: %w", err))
		result.CorruptionDetected = true
	}
	result.RecordsRecovered = validRecords

	// Step 2: Rebuild index from data
	if err := sr.RebuildIndex(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("rebuild index: %w", err))
	} else {
		result.IndexRebuilt = true
	}

	// Step 3: Rebuild time index from data
	if err := sr.RebuildTimeIndex(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("rebuild time index: %w", err))
	} else {
		result.TimeIndexRebuilt = true
	}

	// Step 4: Verify consistency
	if err := sr.VerifyConsistency(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("verify consistency: %w", err))
		result.CorruptionDetected = true
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("recovery completed with errors: %d error(s)", len(result.Errors))
	}

	return result, nil
}

// ValidateData validates the segment data file and truncates at corruption
func (sr *SegmentRecovery) ValidateData() (int64, error) {
	sr.segment.mu.Lock()
	defer sr.segment.mu.Unlock()

	if _, err := sr.segment.dataFile.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek to start: %w", err)
	}

	validBytes := int64(0)
	recordCount := int64(0)
	lastValidOffset := int64(0)

	for {
		currentPos, err := sr.segment.dataFile.Seek(0, io.SeekCurrent)
		if err != nil {
			return recordCount, fmt.Errorf("get current position: %w", err)
		}

		record, err := decodeRecord(sr.segment.dataFile)
		if err == io.EOF {
			// Reached end of file normally
			break
		}
		if err != nil {
			// Corruption detected, truncate at last valid position
			if truncErr := sr.segment.dataFile.Truncate(validBytes); truncErr != nil {
				return recordCount, fmt.Errorf("truncate corrupted data: %w", truncErr)
			}
			if _, seekErr := sr.segment.dataFile.Seek(validBytes, io.SeekStart); seekErr != nil {
				return recordCount, fmt.Errorf("seek after truncate: %w", seekErr)
			}
			return recordCount, fmt.Errorf("%w at position %d: %v", ErrCorruptedSegment, currentPos, err)
		}

		// Validate record integrity
		if record.Offset < lastValidOffset {
			// Offset went backwards - corruption
			if err := sr.segment.dataFile.Truncate(validBytes); err != nil {
				return recordCount, fmt.Errorf("truncate at invalid offset: %w", err)
			}
			return recordCount, fmt.Errorf("%w: offset decreased from %d to %d", ErrCorruptedSegment, lastValidOffset, record.Offset)
		}

		// Record is valid
		newPos, err := sr.segment.dataFile.Seek(0, io.SeekCurrent)
		if err != nil {
			return recordCount, fmt.Errorf("get position after record: %w", err)
		}
		validBytes = newPos
		recordCount++
		lastValidOffset = record.Offset
	}

	// Update segment's nextOffset
	sr.segment.nextOffset = sr.segment.baseOffset + recordCount

	return recordCount, nil
}

// RebuildIndex rebuilds the offset index from the data file
func (sr *SegmentRecovery) RebuildIndex() error {
	sr.segment.mu.Lock()
	defer sr.segment.mu.Unlock()

	// Truncate existing index
	if err := sr.segment.indexFile.Truncate(0); err != nil {
		return fmt.Errorf("truncate index file: %w", err)
	}

	if _, err := sr.segment.indexFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek index to start: %w", err)
	}

	if _, err := sr.segment.dataFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek data to start: %w", err)
	}

	position := int64(0)
	for {
		currentPos := position
		record, err := decodeRecord(sr.segment.dataFile)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("decode record at position %d: %w", position, err)
		}

		// Write index entry
		if err := sr.segment.writeIndex(record.Offset, currentPos); err != nil {
			return fmt.Errorf("write index entry: %w", err)
		}

		newPos, err := sr.segment.dataFile.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("get current position: %w", err)
		}
		position = newPos
	}

	// Sync index to disk
	if err := sr.segment.indexFile.Sync(); err != nil {
		return fmt.Errorf("sync index file: %w", err)
	}

	return nil
}

// RebuildTimeIndex rebuilds the time index from the data file
func (sr *SegmentRecovery) RebuildTimeIndex() error {
	sr.segment.mu.Lock()
	defer sr.segment.mu.Unlock()

	// Truncate existing time index
	if err := sr.segment.timeIndexFile.Truncate(0); err != nil {
		return fmt.Errorf("truncate time index file: %w", err)
	}

	if _, err := sr.segment.timeIndexFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek time index to start: %w", err)
	}

	if _, err := sr.segment.dataFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek data to start: %w", err)
	}

	for {
		record, err := decodeRecord(sr.segment.dataFile)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("decode record: %w", err)
		}

		// Write time index entry
		if err := sr.segment.writeTimeIndex(record.Timestamp, record.Offset); err != nil {
			return fmt.Errorf("write time index entry: %w", err)
		}
	}

	// Sync time index to disk
	if err := sr.segment.timeIndexFile.Sync(); err != nil {
		return fmt.Errorf("sync time index file: %w", err)
	}

	return nil
}

// VerifyConsistency verifies that indexes match the data file
func (sr *SegmentRecovery) VerifyConsistency() error {
	sr.segment.mu.RLock()
	defer sr.segment.mu.RUnlock()

	// Count records in data file
	dataRecordCount, err := sr.countRecordsInData()
	if err != nil {
		return fmt.Errorf("count data records: %w", err)
	}

	// Count entries in index
	indexEntryCount, err := sr.countIndexEntries()
	if err != nil {
		return fmt.Errorf("count index entries: %w", err)
	}

	// Count entries in time index
	timeIndexEntryCount, err := sr.countTimeIndexEntries()
	if err != nil {
		return fmt.Errorf("count time index entries: %w", err)
	}

	if dataRecordCount != indexEntryCount {
		return fmt.Errorf("%w: data has %d records but index has %d entries",
			ErrIndexSizeMismatch, dataRecordCount, indexEntryCount)
	}

	if dataRecordCount != timeIndexEntryCount {
		return fmt.Errorf("%w: data has %d records but time index has %d entries",
			ErrIndexSizeMismatch, dataRecordCount, timeIndexEntryCount)
	}

	return nil
}

func (sr *SegmentRecovery) countRecordsInData() (int64, error) {
	if _, err := sr.segment.dataFile.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	count := int64(0)
	for {
		_, err := decodeRecord(sr.segment.dataFile)
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

func (sr *SegmentRecovery) countIndexEntries() (int64, error) {
	size, err := sr.segment.indexFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	return size / 16, nil // Each index entry is 16 bytes
}

func (sr *SegmentRecovery) countTimeIndexEntries() (int64, error) {
	size, err := sr.segment.timeIndexFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	return size / 16, nil // Each time index entry is 16 bytes
}

// ChecksumRecord calculates CRC32 checksum for a record
func ChecksumRecord(record *Record) uint32 {
	checksummer := crc32.NewIEEE()
	checksummer.Write([]byte(fmt.Sprintf("%d", record.Offset)))
	checksummer.Write([]byte(fmt.Sprintf("%d", record.Timestamp)))
	checksummer.Write(record.Key)
	checksummer.Write(record.Value)
	return checksummer.Sum32()
}

// LogRecovery handles recovery operations for an entire log
type LogRecovery struct {
	log *Log
}

// NewLogRecovery creates a new recovery handler for a log
func NewLogRecovery(log *Log) *LogRecovery {
	return &LogRecovery{
		log: log,
	}
}

// RecoverLog performs recovery on all segments in the log
func (lr *LogRecovery) RecoverLog() (*RecoveryResult, error) {
	lr.log.mu.Lock()
	defer lr.log.mu.Unlock()

	aggregateResult := &RecoveryResult{}

	for i, segment := range lr.log.segments {
		recovery := NewSegmentRecovery(segment)
		result, err := recovery.Recover()

		if result != nil {
			aggregateResult.RecordsRecovered += result.RecordsRecovered
			aggregateResult.RecordsTruncated += result.RecordsTruncated
			if result.IndexRebuilt {
				aggregateResult.IndexRebuilt = true
			}
			if result.TimeIndexRebuilt {
				aggregateResult.TimeIndexRebuilt = true
			}
			if result.CorruptionDetected {
				aggregateResult.CorruptionDetected = true
			}
			aggregateResult.Errors = append(aggregateResult.Errors, result.Errors...)
		}

		if err != nil {
			aggregateResult.Errors = append(aggregateResult.Errors,
				fmt.Errorf("segment %d (base offset %d): %w", i, segment.BaseOffset(), err))
		}
	}

	if len(aggregateResult.Errors) > 0 {
		return aggregateResult, fmt.Errorf("log recovery completed with %d error(s)", len(aggregateResult.Errors))
	}

	return aggregateResult, nil
}

// RecoverFromDirectory attempts to recover all segments found in a directory
func RecoverFromDirectory(dir string, maxSegmentSize int64) (*Log, error) {
	// Find all .log files in directory
	pattern := filepath.Join(dir, "*.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob log files: %w", err)
	}

	if len(matches) == 0 {
		// No segments found, create new log
		return NewLog(LogConfig{
			Dir:            dir,
			MaxSegmentSize: maxSegmentSize,
		})
	}

	// Create log structure
	log := &Log{
		dir:            dir,
		maxSegmentSize: maxSegmentSize,
		segments:       make([]*Segment, 0),
	}

	// Load each segment and attempt recovery
	for _, logFile := range matches {
		baseOffset, err := parseBaseOffsetFromFilename(filepath.Base(logFile))
		if err != nil {
			return nil, fmt.Errorf("parse base offset from %s: %w", logFile, err)
		}

		segment, err := NewSegment(SegmentConfig{
			BaseOffset: baseOffset,
			MaxBytes:   maxSegmentSize,
			Dir:        dir,
		})
		if err != nil {
			return nil, fmt.Errorf("load segment %s: %w", logFile, err)
		}

		// Attempt recovery on this segment
		recovery := NewSegmentRecovery(segment)
		result, err := recovery.Recover()
		if err != nil {
			// Log error but continue with recovered data
			fmt.Fprintf(os.Stderr, "Warning: segment %s recovery had errors: %v (recovered %d records)\n",
				logFile, err, result.RecordsRecovered)
		}

		log.segments = append(log.segments, segment)
	}

	// Set active segment to the last one
	if len(log.segments) > 0 {
		log.activeSegment = log.segments[len(log.segments)-1]
	}

	return log, nil
}

func parseBaseOffsetFromFilename(filename string) (int64, error) {
	var baseOffset int64
	_, err := fmt.Sscanf(filename, "%020d.log", &baseOffset)
	if err != nil {
		return 0, fmt.Errorf("invalid filename format: %w", err)
	}
	return baseOffset, nil
}
