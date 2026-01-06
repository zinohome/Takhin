// Copyright 2025 Takhin Data, Inc.

package log

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/takhin-data/takhin/pkg/mempool"
)

type Record struct {
	Offset    int64
	Timestamp int64
	Key       []byte
	Value     []byte
}

type Segment struct {
	baseOffset    int64
	nextOffset    int64
	dataFile      *os.File
	indexFile     *os.File
	timeIndexFile *os.File
	maxBytes      int64
	mu            sync.RWMutex
}

type SegmentConfig struct {
	BaseOffset int64
	MaxBytes   int64
	Dir        string
}

func NewSegment(config SegmentConfig) (*Segment, error) {
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	dataPath := filepath.Join(config.Dir, fmt.Sprintf("%020d.log", config.BaseOffset))
	indexPath := filepath.Join(config.Dir, fmt.Sprintf("%020d.index", config.BaseOffset))
	timeIndexPath := filepath.Join(config.Dir, fmt.Sprintf("%020d.timeindex", config.BaseOffset))

	dataFile, err := os.OpenFile(dataPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open data file: %w", err)
	}

	indexFile, err := os.OpenFile(indexPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		dataFile.Close()
		return nil, fmt.Errorf("open index file: %w", err)
	}

	timeIndexFile, err := os.OpenFile(timeIndexPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		dataFile.Close()
		indexFile.Close()
		return nil, fmt.Errorf("open time index file: %w", err)
	}

	segment := &Segment{
		baseOffset:    config.BaseOffset,
		nextOffset:    config.BaseOffset,
		dataFile:      dataFile,
		indexFile:     indexFile,
		timeIndexFile: timeIndexFile,
		maxBytes:      config.MaxBytes,
	}

	stat, err := dataFile.Stat()
	if err != nil {
		segment.Close()
		return nil, fmt.Errorf("stat data file: %w", err)
	}

	if stat.Size() > 0 {
		if err := segment.scanSegment(); err != nil {
			segment.Close()
			return nil, fmt.Errorf("scan segment: %w", err)
		}
	}

	return segment, nil
}

func (s *Segment) Append(record *Record) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stat, err := s.dataFile.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}
	if stat.Size() >= s.maxBytes {
		return 0, fmt.Errorf("segment is full")
	}

	position, err := s.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("seek to end: %w", err)
	}

	offset := s.nextOffset
	record.Offset = offset

	data, err := encodeRecord(record)
	if err != nil {
		return 0, fmt.Errorf("encode record: %w", err)
	}

	if _, err := s.dataFile.Write(data); err != nil {
		mempool.PutBuffer(data)
		return 0, fmt.Errorf("write data: %w", err)
	}
	mempool.PutBuffer(data)

	if err := s.writeIndex(offset, position); err != nil {
		return 0, fmt.Errorf("write index: %w", err)
	}

	if err := s.writeTimeIndex(record.Timestamp, offset); err != nil {
		return 0, fmt.Errorf("write time index: %w", err)
	}

	s.nextOffset++
	return offset, nil
}

// AppendBatch appends multiple records in a single batch for better performance
func (s *Segment) AppendBatch(records []*Record) ([]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(records) == 0 {
		return nil, nil
	}

	stat, err := s.dataFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Calculate total size needed
	totalSize := int64(0)
	for _, record := range records {
		keyLen := len(record.Key)
		valueLen := len(record.Value)
		recordSize := 4 + 8 + 8 + 4 + keyLen + 4 + valueLen
		totalSize += int64(4 + recordSize) // 4 bytes for size prefix
	}

	if stat.Size()+totalSize > s.maxBytes {
		return nil, fmt.Errorf("batch too large for segment")
	}

	position, err := s.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("seek to end: %w", err)
	}

	// Pre-allocate buffer for all records using memory pool
	batchBuf := mempool.GetBuffer(int(totalSize))[:0]
	indexBuf := mempool.GetBuffer(len(records) * 16)[:0]
	timeIndexBuf := mempool.GetBuffer(len(records) * 16)[:0]
	offsets := make([]int64, len(records))
	currentPosition := position

	for i, record := range records {
		offset := s.nextOffset + int64(i)
		record.Offset = offset
		offsets[i] = offset

		// Encode record
		data, err := encodeRecord(record)
		if err != nil {
			mempool.PutBuffer(batchBuf)
			mempool.PutBuffer(indexBuf)
			mempool.PutBuffer(timeIndexBuf)
			return nil, fmt.Errorf("encode record %d: %w", i, err)
		}
		batchBuf = append(batchBuf, data...)
		mempool.PutBuffer(data) // Return immediately after use

		// Prepare index entry
		indexEntry := mempool.GetBuffer(16)
		binary.BigEndian.PutUint64(indexEntry[0:8], uint64(offset))
		binary.BigEndian.PutUint64(indexEntry[8:16], uint64(currentPosition))
		indexBuf = append(indexBuf, indexEntry...)
		mempool.PutBuffer(indexEntry)

		// Prepare time index entry
		timeIndexEntry := mempool.GetBuffer(16)
		binary.BigEndian.PutUint64(timeIndexEntry[0:8], uint64(record.Timestamp))
		binary.BigEndian.PutUint64(timeIndexEntry[8:16], uint64(offset))
		timeIndexBuf = append(timeIndexBuf, timeIndexEntry...)
		mempool.PutBuffer(timeIndexEntry)

		currentPosition += int64(len(data))
	}

	// Write all data at once
	if _, err := s.dataFile.Write(batchBuf); err != nil {
		mempool.PutBuffer(batchBuf)
		mempool.PutBuffer(indexBuf)
		mempool.PutBuffer(timeIndexBuf)
		return nil, fmt.Errorf("write batch data: %w", err)
	}

	// Write all index entries at once
	if _, err := s.indexFile.Write(indexBuf); err != nil {
		mempool.PutBuffer(batchBuf)
		mempool.PutBuffer(indexBuf)
		mempool.PutBuffer(timeIndexBuf)
		return nil, fmt.Errorf("write batch index: %w", err)
	}

	// Write all time index entries at once
	if _, err := s.timeIndexFile.Write(timeIndexBuf); err != nil {
		mempool.PutBuffer(batchBuf)
		mempool.PutBuffer(indexBuf)
		mempool.PutBuffer(timeIndexBuf)
		return nil, fmt.Errorf("write batch time index: %w", err)
	}

	// Return buffers to pool after successful write
	mempool.PutBuffer(batchBuf)
	mempool.PutBuffer(indexBuf)
	mempool.PutBuffer(timeIndexBuf)

	s.nextOffset += int64(len(records))
	return offsets, nil
}

func (s *Segment) Read(offset int64) (*Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if offset < s.baseOffset || offset >= s.nextOffset {
		return nil, fmt.Errorf("offset out of range: %d", offset)
	}

	position, err := s.findPosition(offset)
	if err != nil {
		return nil, fmt.Errorf("find position: %w", err)
	}

	if _, err := s.dataFile.Seek(position, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to position: %w", err)
	}

	record, err := decodeRecord(s.dataFile)
	if err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}

	return record, nil
}

// ReadRange reads multiple records from startOffset up to maxBytes using zero-copy I/O.
// Returns the position and size of the data in the file for zero-copy transfer.
func (s *Segment) ReadRange(startOffset int64, maxBytes int64) (position int64, size int64, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if startOffset < s.baseOffset || startOffset >= s.nextOffset {
		return 0, 0, fmt.Errorf("offset out of range: %d", startOffset)
	}

	position, err = s.findPosition(startOffset)
	if err != nil {
		return 0, 0, fmt.Errorf("find position: %w", err)
	}

	// Calculate the size of data we can read
	stat, err := s.dataFile.Stat()
	if err != nil {
		return 0, 0, fmt.Errorf("stat file: %w", err)
	}

	fileSize := stat.Size()
	remaining := fileSize - position

	if remaining <= 0 {
		return 0, 0, fmt.Errorf("no data available at offset %d", startOffset)
	}

	// Limit to maxBytes
	size = remaining
	if maxBytes > 0 && size > maxBytes {
		size = maxBytes
	}

	return position, size, nil
}

// DataFile returns the underlying data file for zero-copy operations.
// Caller must hold appropriate locks.
func (s *Segment) DataFile() *os.File {
	return s.dataFile
}

func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	if err := s.dataFile.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.indexFile.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.timeIndexFile.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

func (s *Segment) BaseOffset() int64 {
	return s.baseOffset
}

func (s *Segment) NextOffset() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextOffset
}

func (s *Segment) IsFull() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stat, err := s.dataFile.Stat()
	if err != nil {
		return false
	}
	return stat.Size() >= s.maxBytes
}

func (s *Segment) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.dataFile.Sync(); err != nil {
		return err
	}
	if err := s.indexFile.Sync(); err != nil {
		return err
	}
	if err := s.timeIndexFile.Sync(); err != nil {
		return err
	}
	return nil
}

func (s *Segment) writeIndex(offset int64, position int64) error {
	buf := mempool.GetBuffer(16)
	defer mempool.PutBuffer(buf)
	binary.BigEndian.PutUint64(buf[0:8], uint64(offset))
	binary.BigEndian.PutUint64(buf[8:16], uint64(position))
	_, err := s.indexFile.Write(buf)
	return err
}

func (s *Segment) writeTimeIndex(timestamp int64, offset int64) error {
	buf := mempool.GetBuffer(16)
	defer mempool.PutBuffer(buf)
	binary.BigEndian.PutUint64(buf[0:8], uint64(timestamp))
	binary.BigEndian.PutUint64(buf[8:16], uint64(offset))
	_, err := s.timeIndexFile.Write(buf)
	return err
}

// FindOffsetByTimestamp finds the offset of the first message with timestamp >= target
func (s *Segment) FindOffsetByTimestamp(timestamp int64) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	timeIndexSize, err := s.timeIndexFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	if timeIndexSize == 0 {
		return s.baseOffset, nil
	}

	entrySize := int64(16)
	numEntries := timeIndexSize / entrySize

	// Binary search for timestamp
	left, right := int64(0), numEntries-1
	result := s.baseOffset

	for left <= right {
		mid := (left + right) / 2
		pos := mid * entrySize

		if _, err := s.timeIndexFile.Seek(pos, io.SeekStart); err != nil {
			return 0, err
		}

		buf := mempool.GetBuffer(16)
		if _, err := io.ReadFull(s.timeIndexFile, buf); err != nil {
			mempool.PutBuffer(buf)
			return 0, err
		}

		midTimestamp := int64(binary.BigEndian.Uint64(buf[0:8]))
		midOffset := int64(binary.BigEndian.Uint64(buf[8:16]))
		mempool.PutBuffer(buf)

		if midTimestamp >= timestamp {
			result = midOffset
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	return result, nil
}

func (s *Segment) findPosition(offset int64) (int64, error) {
	indexSize, err := s.indexFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	if indexSize == 0 {
		// No index entries, start from beginning
		return 0, nil
	}

	entrySize := int64(16)
	numEntries := indexSize / entrySize

	left, right := int64(0), numEntries-1
	var lastValidPosition int64 = 0

	for left <= right {
		mid := (left + right) / 2
		pos := mid * entrySize

		if _, err := s.indexFile.Seek(pos, io.SeekStart); err != nil {
			return 0, err
		}

		buf := mempool.GetBuffer(16)
		if _, err := io.ReadFull(s.indexFile, buf); err != nil {
			mempool.PutBuffer(buf)
			return 0, err
		}

		midOffset := int64(binary.BigEndian.Uint64(buf[0:8]))
		position := int64(binary.BigEndian.Uint64(buf[8:16]))
		mempool.PutBuffer(buf)

		if midOffset == offset {
			return position, nil
		} else if midOffset < offset {
			lastValidPosition = position
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	// Return the last valid position we found (closest to but less than target)
	return lastValidPosition, nil
}

func (s *Segment) scanSegment() error {
	if _, err := s.dataFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	count := int64(0)
	for {
		_, err := decodeRecord(s.dataFile)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		count++
	}

	s.nextOffset = s.baseOffset + count
	return nil
}

func encodeRecord(record *Record) ([]byte, error) {
	keyLen := len(record.Key)
	valueLen := len(record.Value)
	size := 4 + 8 + 8 + 4 + keyLen + 4 + valueLen

	buf := mempool.GetBuffer(4 + size)
	binary.BigEndian.PutUint32(buf[0:4], uint32(size))
	binary.BigEndian.PutUint64(buf[4:12], uint64(record.Offset))
	binary.BigEndian.PutUint64(buf[12:20], uint64(record.Timestamp))
	binary.BigEndian.PutUint32(buf[20:24], uint32(keyLen))
	copy(buf[24:24+keyLen], record.Key)
	binary.BigEndian.PutUint32(buf[24+keyLen:28+keyLen], uint32(valueLen))
	copy(buf[28+keyLen:], record.Value)

	return buf, nil
}

func decodeRecord(r io.Reader) (*Record, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return nil, err
	}

	data := mempool.GetBuffer(int(size))
	defer mempool.PutBuffer(data)
	
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	record := &Record{}
	record.Offset = int64(binary.BigEndian.Uint64(data[0:8]))
	record.Timestamp = int64(binary.BigEndian.Uint64(data[8:16]))

	keyLen := binary.BigEndian.Uint32(data[16:20])
	record.Key = make([]byte, keyLen)
	copy(record.Key, data[20:20+keyLen])

	valueLen := binary.BigEndian.Uint32(data[20+keyLen : 24+keyLen])
	record.Value = make([]byte, valueLen)
	copy(record.Value, data[24+keyLen:])

	return record, nil
}

// SearchByTimestamp searches for the first offset whose timestamp >= the given timestamp
func (s *Segment) SearchByTimestamp(timestamp int64) (int64, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Use time index to find approximate position
	// For simplicity, we'll do a linear search through the segment
	// In a production system, you'd use the time index for binary search

	// Get segment size
	stat, err := s.dataFile.Stat()
	if err != nil {
		return 0, 0, err
	}

	if stat.Size() == 0 {
		return 0, 0, fmt.Errorf("empty segment")
	}

	// Read through records to find the first one >= timestamp
	_, err = s.dataFile.Seek(0, io.SeekStart)
	if err != nil {
		return 0, 0, err
	}

	for {
		record, err := decodeRecord(s.dataFile)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, err
		}

		if record.Timestamp >= timestamp {
			return record.Offset, record.Timestamp, nil
		}
	}

	// Not found in this segment
	return 0, 0, fmt.Errorf("timestamp not found in segment")
}

// Size returns the size of the segment's data file in bytes
func (s *Segment) Size() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stat, err := s.dataFile.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat data file: %w", err)
	}

	return stat.Size(), nil
}

// TruncateTo truncates the segment to the given offset
// This removes all records with offset >= the given offset
func (s *Segment) TruncateTo(offset int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if offset <= s.baseOffset {
		// Truncate everything, reset to base offset
		if err := s.dataFile.Truncate(0); err != nil {
			return fmt.Errorf("truncate data file: %w", err)
		}
		if err := s.indexFile.Truncate(0); err != nil {
			return fmt.Errorf("truncate index file: %w", err)
		}
		if err := s.timeIndexFile.Truncate(0); err != nil {
			return fmt.Errorf("truncate time index file: %w", err)
		}
		s.nextOffset = s.baseOffset
		return nil
	}

	if offset >= s.nextOffset {
		// Nothing to truncate
		return nil
	}

	// Find the position of the offset in the data file
	position, err := s.findPosition(offset)
	if err != nil {
		return fmt.Errorf("find position: %w", err)
	}

	// Truncate files
	if err := s.dataFile.Truncate(position); err != nil {
		return fmt.Errorf("truncate data file: %w", err)
	}

	// Rebuild indexes (simplified - in production you'd do a more efficient truncation)
	if err := s.indexFile.Truncate(0); err != nil {
		return fmt.Errorf("truncate index file: %w", err)
	}
	if err := s.timeIndexFile.Truncate(0); err != nil {
		return fmt.Errorf("truncate time index file: %w", err)
	}

	// Update next offset
	s.nextOffset = offset

	return nil
}
