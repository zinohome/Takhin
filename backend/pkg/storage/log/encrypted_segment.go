// Copyright 2025 Takhin Data, Inc.

package log

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/encryption"
	"github.com/takhin-data/takhin/pkg/mempool"
)

// EncryptedSegment wraps a segment with encryption
type EncryptedSegment struct {
	*Segment
	encryptor  encryption.Encryptor
	keyManager encryption.KeyManager
	keyID      string
}

// EncryptedSegmentConfig extends SegmentConfig with encryption
type EncryptedSegmentConfig struct {
	SegmentConfig
	Encryptor  encryption.Encryptor
	KeyManager encryption.KeyManager
}

// NewEncryptedSegment creates an encrypted segment
func NewEncryptedSegment(config EncryptedSegmentConfig) (*EncryptedSegment, error) {
	segment, err := NewSegment(config.SegmentConfig)
	if err != nil {
		return nil, fmt.Errorf("create base segment: %w", err)
	}

	keyID, _, err := config.KeyManager.GetCurrentKey()
	if err != nil {
		segment.Close()
		return nil, fmt.Errorf("get current key: %w", err)
	}

	return &EncryptedSegment{
		Segment:    segment,
		encryptor:  config.Encryptor,
		keyManager: config.KeyManager,
		keyID:      keyID,
	}, nil
}

// Append encrypts and appends a record
func (es *EncryptedSegment) Append(record *Record) (int64, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	stat, err := es.dataFile.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}
	if stat.Size() >= es.maxBytes {
		return 0, fmt.Errorf("segment is full")
	}

	position, err := es.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("seek to end: %w", err)
	}

	offset := es.nextOffset
	record.Offset = offset

	// Encode record
	plaintext, err := encodeRecord(record)
	if err != nil {
		return 0, fmt.Errorf("encode record: %w", err)
	}

	// Encrypt
	ciphertext, err := es.encryptor.Encrypt(plaintext)
	mempool.PutBuffer(plaintext)
	if err != nil {
		return 0, fmt.Errorf("encrypt record: %w", err)
	}

	// Write encrypted data with keyID header
	if err := es.writeEncryptedData(ciphertext); err != nil {
		mempool.PutBuffer(ciphertext)
		return 0, fmt.Errorf("write encrypted data: %w", err)
	}
	mempool.PutBuffer(ciphertext)

	if err := es.writeIndex(offset, position); err != nil {
		return 0, fmt.Errorf("write index: %w", err)
	}

	if err := es.writeTimeIndex(record.Timestamp, offset); err != nil {
		return 0, fmt.Errorf("write time index: %w", err)
	}

	es.nextOffset++
	return offset, nil
}

// AppendBatch encrypts and appends multiple records
func (es *EncryptedSegment) AppendBatch(records []*Record) ([]int64, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if len(records) == 0 {
		return nil, nil
	}

	stat, err := es.dataFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Calculate total size with encryption overhead
	totalSize := int64(0)
	encryptionOverhead := es.encryptor.Overhead()
	keyIDSize := int64(2 + len(es.keyID)) // 2 bytes for length + keyID

	for _, record := range records {
		keyLen := len(record.Key)
		valueLen := len(record.Value)
		recordSize := 4 + 8 + 8 + 4 + keyLen + 4 + valueLen
		encryptedSize := int64(4 + recordSize + encryptionOverhead)
		totalSize += keyIDSize + 4 + encryptedSize // keyID + size prefix + encrypted data
	}

	if stat.Size()+totalSize > es.maxBytes {
		return nil, fmt.Errorf("batch too large for segment")
	}

	position, err := es.dataFile.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("seek to end: %w", err)
	}

	offsets := make([]int64, len(records))
	currentPosition := position

	for i, record := range records {
		offset := es.nextOffset + int64(i)
		record.Offset = offset
		offsets[i] = offset

		// Encode record
		plaintext, err := encodeRecord(record)
		if err != nil {
			return nil, fmt.Errorf("encode record %d: %w", i, err)
		}

		// Encrypt
		ciphertext, err := es.encryptor.Encrypt(plaintext)
		mempool.PutBuffer(plaintext)
		if err != nil {
			return nil, fmt.Errorf("encrypt record %d: %w", i, err)
		}

		// Write encrypted data
		if err := es.writeEncryptedData(ciphertext); err != nil {
			mempool.PutBuffer(ciphertext)
			return nil, fmt.Errorf("write encrypted record %d: %w", i, err)
		}

		writeSize := int64(2 + len(es.keyID) + 4 + len(ciphertext))
		mempool.PutBuffer(ciphertext)

		// Write index entries
		if err := es.writeIndex(offset, currentPosition); err != nil {
			return nil, fmt.Errorf("write index for record %d: %w", i, err)
		}

		if err := es.writeTimeIndex(record.Timestamp, offset); err != nil {
			return nil, fmt.Errorf("write time index for record %d: %w", i, err)
		}

		currentPosition += writeSize
	}

	es.nextOffset += int64(len(records))
	return offsets, nil
}

// Read decrypts and reads a record
func (es *EncryptedSegment) Read(offset int64) (*Record, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if offset < es.baseOffset || offset >= es.nextOffset {
		return nil, fmt.Errorf("offset out of range: %d", offset)
	}

	position, err := es.findPosition(offset)
	if err != nil {
		return nil, fmt.Errorf("find position: %w", err)
	}

	if _, err := es.dataFile.Seek(position, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to position: %w", err)
	}

	// Read encrypted data
	ciphertext, keyID, err := es.readEncryptedData()
	if err != nil {
		return nil, fmt.Errorf("read encrypted data: %w", err)
	}

	// Get decryption key
	key, err := es.keyManager.GetKey(keyID)
	if err != nil {
		mempool.PutBuffer(ciphertext)
		return nil, fmt.Errorf("get decryption key: %w", err)
	}

	// Create decryptor (in case key was rotated)
	decryptor, err := encryption.NewEncryptor(es.encryptor.Algorithm(), key)
	if err != nil {
		mempool.PutBuffer(ciphertext)
		return nil, fmt.Errorf("create decryptor: %w", err)
	}

	// Decrypt
	plaintext, err := decryptor.Decrypt(ciphertext)
	mempool.PutBuffer(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt record: %w", err)
	}

	// Decode record
	record, err := decodeRecordFromBuffer(plaintext)
	mempool.PutBuffer(plaintext)
	if err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}

	return record, nil
}

// writeEncryptedData writes encrypted data with keyID header
// Format: [keyID_len(2)][keyID][encrypted_data_len(4)][encrypted_data]
func (es *EncryptedSegment) writeEncryptedData(ciphertext []byte) error {
	keyIDBytes := []byte(es.keyID)
	keyIDLen := uint16(len(keyIDBytes))

	// Write key ID length
	header := mempool.GetBuffer(2)
	binary.BigEndian.PutUint16(header, keyIDLen)
	if _, err := es.dataFile.Write(header); err != nil {
		mempool.PutBuffer(header)
		return fmt.Errorf("write key ID length: %w", err)
	}
	mempool.PutBuffer(header)

	// Write key ID
	if _, err := es.dataFile.Write(keyIDBytes); err != nil {
		return fmt.Errorf("write key ID: %w", err)
	}

	// Write encrypted data length
	sizeHeader := mempool.GetBuffer(4)
	binary.BigEndian.PutUint32(sizeHeader, uint32(len(ciphertext)))
	if _, err := es.dataFile.Write(sizeHeader); err != nil {
		mempool.PutBuffer(sizeHeader)
		return fmt.Errorf("write data length: %w", err)
	}
	mempool.PutBuffer(sizeHeader)

	// Write encrypted data
	if _, err := es.dataFile.Write(ciphertext); err != nil {
		return fmt.Errorf("write encrypted data: %w", err)
	}

	return nil
}

// readEncryptedData reads encrypted data with keyID header
func (es *EncryptedSegment) readEncryptedData() ([]byte, string, error) {
	// Read key ID length
	keyIDLenBuf := mempool.GetBuffer(2)
	if _, err := io.ReadFull(es.dataFile, keyIDLenBuf); err != nil {
		mempool.PutBuffer(keyIDLenBuf)
		return nil, "", fmt.Errorf("read key ID length: %w", err)
	}
	keyIDLen := binary.BigEndian.Uint16(keyIDLenBuf)
	mempool.PutBuffer(keyIDLenBuf)

	// Read key ID
	keyIDBuf := mempool.GetBuffer(int(keyIDLen))
	if _, err := io.ReadFull(es.dataFile, keyIDBuf); err != nil {
		mempool.PutBuffer(keyIDBuf)
		return nil, "", fmt.Errorf("read key ID: %w", err)
	}
	keyID := string(keyIDBuf)
	mempool.PutBuffer(keyIDBuf)

	// Read encrypted data length
	sizeBuf := mempool.GetBuffer(4)
	if _, err := io.ReadFull(es.dataFile, sizeBuf); err != nil {
		mempool.PutBuffer(sizeBuf)
		return nil, "", fmt.Errorf("read data length: %w", err)
	}
	dataLen := binary.BigEndian.Uint32(sizeBuf)
	mempool.PutBuffer(sizeBuf)

	// Read encrypted data
	ciphertext := mempool.GetBuffer(int(dataLen))
	if _, err := io.ReadFull(es.dataFile, ciphertext); err != nil {
		mempool.PutBuffer(ciphertext)
		return nil, "", fmt.Errorf("read encrypted data: %w", err)
	}

	return ciphertext, keyID, nil
}

// decodeRecordFromBuffer decodes a record from a buffer
func decodeRecordFromBuffer(data []byte) (*Record, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("buffer too short")
	}

	size := binary.BigEndian.Uint32(data[0:4])
	if int(size)+4 > len(data) {
		return nil, fmt.Errorf("invalid record size")
	}

	recordData := data[4 : 4+size]
	
	record := &Record{}
	record.Offset = int64(binary.BigEndian.Uint64(recordData[0:8]))
	record.Timestamp = int64(binary.BigEndian.Uint64(recordData[8:16]))

	keyLen := binary.BigEndian.Uint32(recordData[16:20])
	record.Key = make([]byte, keyLen)
	copy(record.Key, recordData[20:20+keyLen])

	valueLen := binary.BigEndian.Uint32(recordData[20+keyLen : 24+keyLen])
	record.Value = make([]byte, valueLen)
	copy(record.Value, recordData[24+keyLen:])

	return record, nil
}

// scanSegment scans and recovers an encrypted segment
func (es *EncryptedSegment) scanSegment() error {
	if _, err := es.dataFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	count := int64(0)
	for {
		// Try to read encrypted record
		ciphertext, keyID, err := es.readEncryptedData()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Get decryption key
		key, err := es.keyManager.GetKey(keyID)
		if err != nil {
			mempool.PutBuffer(ciphertext)
			return fmt.Errorf("get decryption key during scan: %w", err)
		}

		// Verify we can decrypt (don't need the actual data)
		decryptor, err := encryption.NewEncryptor(es.encryptor.Algorithm(), key)
		if err != nil {
			mempool.PutBuffer(ciphertext)
			return fmt.Errorf("create decryptor during scan: %w", err)
		}

		plaintext, err := decryptor.Decrypt(ciphertext)
		mempool.PutBuffer(ciphertext)
		if err != nil {
			return fmt.Errorf("decrypt during scan: %w", err)
		}
		mempool.PutBuffer(plaintext)

		count++
	}

	es.nextOffset = es.baseOffset + count
	return nil
}
