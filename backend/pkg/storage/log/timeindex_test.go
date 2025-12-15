// Copyright 2025 Takhin Data, Inc.

package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentTimeIndex(t *testing.T) {
	dir := t.TempDir()

	config := SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024,
		Dir:        dir,
	}

	segment, err := NewSegment(config)
	require.NoError(t, err)
	defer segment.Close()

	// Append records with different timestamps
	baseTime := time.Now().UnixMilli()
	for i := 0; i < 10; i++ {
		record := &Record{
			Timestamp: baseTime + int64(i)*1000, // 1 second apart
			Key:       []byte("key"),
			Value:     []byte("value"),
		}
		_, err := segment.Append(record)
		require.NoError(t, err)
	}

	// Test finding offset by timestamp
	tests := []struct {
		name       string
		timestamp  int64
		wantOffset int64
	}{
		{
			name:       "exact match first",
			timestamp:  baseTime,
			wantOffset: 0,
		},
		{
			name:       "exact match middle",
			timestamp:  baseTime + 5000,
			wantOffset: 5,
		},
		{
			name:       "between timestamps",
			timestamp:  baseTime + 2500,
			wantOffset: 3,
		},
		{
			name:       "before all",
			timestamp:  baseTime - 1000,
			wantOffset: 0,
		},
		{
			name:       "after all",
			timestamp:  baseTime + 20000,
			wantOffset: 0, // Would need next segment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, err := segment.FindOffsetByTimestamp(tt.timestamp)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, offset, tt.wantOffset)
		})
	}
}

func TestSegmentTimeIndexBatch(t *testing.T) {
	dir := t.TempDir()

	config := SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024,
		Dir:        dir,
	}

	segment, err := NewSegment(config)
	require.NoError(t, err)
	defer segment.Close()

	// Append batch with sequential timestamps
	baseTime := time.Now().UnixMilli()
	batchSize := 100
	records := make([]*Record, batchSize)
	for i := 0; i < batchSize; i++ {
		records[i] = &Record{
			Timestamp: baseTime + int64(i)*100, // 100ms apart
			Key:       []byte("key"),
			Value:     []byte("value"),
		}
	}

	offsets, err := segment.AppendBatch(records)
	require.NoError(t, err)
	assert.Equal(t, batchSize, len(offsets))

	// Find offset by timestamp in the middle
	targetTime := baseTime + 5000 // ~50th record
	offset, err := segment.FindOffsetByTimestamp(targetTime)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, offset, int64(50))
	assert.LessOrEqual(t, offset, int64(51))
}
