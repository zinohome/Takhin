// Copyright 2025 Takhin Data, Inc.

package log

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/encryption"
)

func TestEncryptedSegment_AppendAndRead(t *testing.T) {
	dir := t.TempDir()
	
	// Create key manager
	key := make([]byte, 32)
	rand.Read(key)
	km := encryption.NewStaticKeyManager("test-key", key)
	
	// Create encryptor
	encryptor, err := encryption.NewEncryptor(encryption.AlgorithmAES256GCM, key)
	require.NoError(t, err)

	// Create encrypted segment
	segment, err := NewEncryptedSegment(EncryptedSegmentConfig{
		SegmentConfig: SegmentConfig{
			BaseOffset: 0,
			MaxBytes:   1024 * 1024,
			Dir:        dir,
		},
		Encryptor:  encryptor,
		KeyManager: km,
	})
	require.NoError(t, err)
	defer segment.Close()

	// Append records
	record1 := &Record{
		Timestamp: 1000,
		Key:       []byte("key1"),
		Value:     []byte("value1"),
	}

	offset1, err := segment.Append(record1)
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset1)

	record2 := &Record{
		Timestamp: 2000,
		Key:       []byte("key2"),
		Value:     []byte("value2"),
	}

	offset2, err := segment.Append(record2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), offset2)

	// Read records back
	readRecord1, err := segment.Read(0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), readRecord1.Offset)
	assert.Equal(t, int64(1000), readRecord1.Timestamp)
	assert.Equal(t, []byte("key1"), readRecord1.Key)
	assert.Equal(t, []byte("value1"), readRecord1.Value)

	readRecord2, err := segment.Read(1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), readRecord2.Offset)
	assert.Equal(t, []byte("key2"), readRecord2.Key)
	assert.Equal(t, []byte("value2"), readRecord2.Value)
}

func TestEncryptedSegment_AppendBatch(t *testing.T) {
	dir := t.TempDir()
	
	key := make([]byte, 32)
	rand.Read(key)
	km := encryption.NewStaticKeyManager("test-key", key)
	encryptor, err := encryption.NewEncryptor(encryption.AlgorithmAES256GCM, key)
	require.NoError(t, err)

	segment, err := NewEncryptedSegment(EncryptedSegmentConfig{
		SegmentConfig: SegmentConfig{
			BaseOffset: 0,
			MaxBytes:   1024 * 1024,
			Dir:        dir,
		},
		Encryptor:  encryptor,
		KeyManager: km,
	})
	require.NoError(t, err)
	defer segment.Close()

	// Create batch
	records := []*Record{
		{Timestamp: 1000, Key: []byte("key1"), Value: []byte("value1")},
		{Timestamp: 2000, Key: []byte("key2"), Value: []byte("value2")},
		{Timestamp: 3000, Key: []byte("key3"), Value: []byte("value3")},
	}

	offsets, err := segment.AppendBatch(records)
	require.NoError(t, err)
	assert.Len(t, offsets, 3)
	assert.Equal(t, []int64{0, 1, 2}, offsets)

	// Read back
	for i, expectedRecord := range records {
		record, err := segment.Read(int64(i))
		require.NoError(t, err)
		assert.Equal(t, int64(i), record.Offset)
		assert.Equal(t, expectedRecord.Timestamp, record.Timestamp)
		assert.Equal(t, expectedRecord.Key, record.Key)
		assert.Equal(t, expectedRecord.Value, record.Value)
	}
}

func TestEncryptedSegment_DifferentAlgorithms(t *testing.T) {
	algorithms := []struct {
		name      string
		algorithm encryption.Algorithm
		keySize   int
	}{
		{"AES-128-GCM", encryption.AlgorithmAES128GCM, 16},
		{"AES-256-GCM", encryption.AlgorithmAES256GCM, 32},
		{"ChaCha20", encryption.AlgorithmChaCha20, 32},
	}

	for _, algo := range algorithms {
		t.Run(algo.name, func(t *testing.T) {
			dir := t.TempDir()
			
			key := make([]byte, algo.keySize)
			rand.Read(key)
			km := encryption.NewStaticKeyManager("test-key", key)
			encryptor, err := encryption.NewEncryptor(algo.algorithm, key)
			require.NoError(t, err)

			segment, err := NewEncryptedSegment(EncryptedSegmentConfig{
				SegmentConfig: SegmentConfig{
					BaseOffset: 0,
					MaxBytes:   1024 * 1024,
					Dir:        dir,
				},
				Encryptor:  encryptor,
				KeyManager: km,
			})
			require.NoError(t, err)
			defer segment.Close()

			record := &Record{
				Timestamp: 1000,
				Key:       []byte("test-key"),
				Value:     []byte("test-value"),
			}

			offset, err := segment.Append(record)
			require.NoError(t, err)

			readRecord, err := segment.Read(offset)
			require.NoError(t, err)
			assert.Equal(t, record.Key, readRecord.Key)
			assert.Equal(t, record.Value, readRecord.Value)
		})
	}
}

func TestEncryptedSegment_LargeRecord(t *testing.T) {
	dir := t.TempDir()
	
	key := make([]byte, 32)
	rand.Read(key)
	km := encryption.NewStaticKeyManager("test-key", key)
	encryptor, err := encryption.NewEncryptor(encryption.AlgorithmAES256GCM, key)
	require.NoError(t, err)

	segment, err := NewEncryptedSegment(EncryptedSegmentConfig{
		SegmentConfig: SegmentConfig{
			BaseOffset: 0,
			MaxBytes:   10 * 1024 * 1024, // 10MB
			Dir:        dir,
		},
		Encryptor:  encryptor,
		KeyManager: km,
	})
	require.NoError(t, err)
	defer segment.Close()

	// Create 1MB record
	largeValue := make([]byte, 1024*1024)
	rand.Read(largeValue)

	record := &Record{
		Timestamp: 1000,
		Key:       []byte("large-key"),
		Value:     largeValue,
	}

	offset, err := segment.Append(record)
	require.NoError(t, err)

	readRecord, err := segment.Read(offset)
	require.NoError(t, err)
	assert.Equal(t, record.Key, readRecord.Key)
	assert.Equal(t, record.Value, readRecord.Value)
}

func TestEncryptedSegment_KeyRotation(t *testing.T) {
	dir := t.TempDir()
	
	// Create file key manager
	keyDir := t.TempDir()
	km, err := encryption.NewFileKeyManager(encryption.FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	keyID1, key1, err := km.GetCurrentKey()
	require.NoError(t, err)
	
	encryptor1, err := encryption.NewEncryptor(encryption.AlgorithmAES256GCM, key1)
	require.NoError(t, err)

	// Create segment with first key
	segment, err := NewEncryptedSegment(EncryptedSegmentConfig{
		SegmentConfig: SegmentConfig{
			BaseOffset: 0,
			MaxBytes:   1024 * 1024,
			Dir:        dir,
		},
		Encryptor:  encryptor1,
		KeyManager: km,
	})
	require.NoError(t, err)

	// Write record with first key
	record1 := &Record{
		Timestamp: 1000,
		Key:       []byte("key1"),
		Value:     []byte("value1"),
	}
	offset1, err := segment.Append(record1)
	require.NoError(t, err)

	// Rotate key
	keyID2, key2, err := km.RotateKey()
	require.NoError(t, err)
	assert.NotEqual(t, keyID1, keyID2)

	// Update segment encryptor and keyID
	encryptor2, err := encryption.NewEncryptor(encryption.AlgorithmAES256GCM, key2)
	require.NoError(t, err)
	segment.encryptor = encryptor2
	segment.keyID = keyID2

	// Write record with new key
	record2 := &Record{
		Timestamp: 2000,
		Key:       []byte("key2"),
		Value:     []byte("value2"),
	}
	offset2, err := segment.Append(record2)
	require.NoError(t, err)

	// Should be able to read both records
	readRecord1, err := segment.Read(offset1)
	require.NoError(t, err)
	assert.Equal(t, record1.Key, readRecord1.Key)
	assert.Equal(t, record1.Value, readRecord1.Value)

	readRecord2, err := segment.Read(offset2)
	require.NoError(t, err)
	assert.Equal(t, record2.Key, readRecord2.Key)
	assert.Equal(t, record2.Value, readRecord2.Value)

	segment.Close()
}

func BenchmarkEncryptedSegment_Append_AES256(b *testing.B) {
	dir := b.TempDir()
	
	key := make([]byte, 32)
	rand.Read(key)
	km := encryption.NewStaticKeyManager("test-key", key)
	encryptor, _ := encryption.NewEncryptor(encryption.AlgorithmAES256GCM, key)

	segment, _ := NewEncryptedSegment(EncryptedSegmentConfig{
		SegmentConfig: SegmentConfig{
			BaseOffset: 0,
			MaxBytes:   1024 * 1024 * 1024, // 1GB
			Dir:        dir,
		},
		Encryptor:  encryptor,
		KeyManager: km,
	})
	defer segment.Close()

	record := &Record{
		Timestamp: 1000,
		Key:       []byte("benchmark-key"),
		Value:     make([]byte, 1024), // 1KB
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = segment.Append(record)
	}
}

func BenchmarkEncryptedSegment_Read_AES256(b *testing.B) {
	dir := b.TempDir()
	
	key := make([]byte, 32)
	rand.Read(key)
	km := encryption.NewStaticKeyManager("test-key", key)
	encryptor, _ := encryption.NewEncryptor(encryption.AlgorithmAES256GCM, key)

	segment, _ := NewEncryptedSegment(EncryptedSegmentConfig{
		SegmentConfig: SegmentConfig{
			BaseOffset: 0,
			MaxBytes:   1024 * 1024 * 1024,
			Dir:        dir,
		},
		Encryptor:  encryptor,
		KeyManager: km,
	})
	defer segment.Close()

	// Write a record
	record := &Record{
		Timestamp: 1000,
		Key:       []byte("benchmark-key"),
		Value:     make([]byte, 1024),
	}
	offset, _ := segment.Append(record)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = segment.Read(offset)
	}
}
