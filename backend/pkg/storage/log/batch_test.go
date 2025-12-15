// Copyright 2025 Takhin Data, Inc.

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentAppendBatch(t *testing.T) {
	dir := t.TempDir()

	config := SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024,
		Dir:        dir,
	}

	segment, err := NewSegment(config)
	require.NoError(t, err)
	defer segment.Close()

	// Prepare batch of records
	batchSize := 100
	records := make([]*Record, batchSize)
	for i := 0; i < batchSize; i++ {
		records[i] = &Record{
			Timestamp: int64(i),
			Key:       []byte("key"),
			Value:     []byte("value"),
		}
	}

	// Append batch
	offsets, err := segment.AppendBatch(records)
	require.NoError(t, err)
	assert.Equal(t, batchSize, len(offsets))

	// Verify offsets are sequential
	for i, offset := range offsets {
		assert.Equal(t, int64(i), offset)
	}

	// Verify we can read back the records
	for i := 0; i < batchSize; i++ {
		record, err := segment.Read(int64(i))
		require.NoError(t, err)
		assert.Equal(t, int64(i), record.Offset)
		assert.Equal(t, []byte("key"), record.Key)
		assert.Equal(t, []byte("value"), record.Value)
	}
}

func TestLogAppendBatch(t *testing.T) {
	dir := t.TempDir()

	config := LogConfig{
		Dir:            dir,
		MaxSegmentSize: 1024 * 1024,
	}

	log, err := NewLog(config)
	require.NoError(t, err)
	defer log.Close()

	// Prepare batch
	batchSize := 100
	batch := make([]struct{ Key, Value []byte }, batchSize)
	for i := 0; i < batchSize; i++ {
		batch[i].Key = []byte("key")
		batch[i].Value = []byte("value")
	}

	// Append batch
	offsets, err := log.AppendBatch(batch)
	require.NoError(t, err)
	assert.Equal(t, batchSize, len(offsets))

	// Verify we can read back
	for i := 0; i < batchSize; i++ {
		record, err := log.Read(int64(i))
		require.NoError(t, err)
		assert.Equal(t, int64(i), record.Offset)
		assert.Equal(t, []byte("key"), record.Key)
		assert.Equal(t, []byte("value"), record.Value)
	}
}

func BenchmarkSegmentAppend(b *testing.B) {
	dir := b.TempDir()

	config := SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024 * 1024, // 1GB
		Dir:        dir,
	}

	segment, err := NewSegment(config)
	require.NoError(b, err)
	defer segment.Close()

	record := &Record{
		Timestamp: 1234567890,
		Key:       []byte("benchmark-key"),
		Value:     []byte("benchmark-value-with-some-data"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := segment.Append(record)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSegmentAppendBatch(b *testing.B) {
	dir := b.TempDir()

	config := SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024 * 1024, // 1GB
		Dir:        dir,
	}

	batchSize := 100
	records := make([]*Record, batchSize)
	for i := 0; i < batchSize; i++ {
		records[i] = &Record{
			Timestamp: 1234567890,
			Key:       []byte("benchmark-key"),
			Value:     []byte("benchmark-value-with-some-data"),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		segment, err := NewSegment(config)
		require.NoError(b, err)

		_, err = segment.AppendBatch(records)
		if err != nil {
			b.Fatal(err)
		}
		segment.Close()
	}
}
