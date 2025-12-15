package log

import (
"testing"
"time"
)

func TestSegmentAppend(t *testing.T) {
	dir := t.TempDir()
	config := SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024,
		Dir:        dir,
	}

	segment, err := NewSegment(config)
	if err != nil {
		t.Fatalf("NewSegment failed: %v", err)
	}
	defer segment.Close()

	record := &Record{
		Timestamp: time.Now().UnixMilli(),
		Key:       []byte("key1"),
		Value:     []byte("value1"),
	}

	offset, err := segment.Append(record)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	if offset != 0 {
		t.Errorf("Expected offset 0, got %d", offset)
	}

	read, err := segment.Read(offset)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if string(read.Key) != "key1" {
		t.Errorf("Expected key key1, got %s", string(read.Key))
	}
	if string(read.Value) != "value1" {
		t.Errorf("Expected value value1, got %s", string(read.Value))
	}
}

func TestSegmentMultiple(t *testing.T) {
	dir := t.TempDir()
	config := SegmentConfig{
		BaseOffset: 0,
		MaxBytes:   1024 * 1024,
		Dir:        dir,
	}

	segment, err := NewSegment(config)
	if err != nil {
		t.Fatalf("NewSegment failed: %v", err)
	}
	defer segment.Close()

	for i := 0; i < 10; i++ {
		record := &Record{
			Timestamp: time.Now().UnixMilli(),
			Key:       []byte("key"),
			Value:     []byte("value"),
		}

		offset, err := segment.Append(record)
		if err != nil {
			t.Fatalf("Append failed: %v", err)
		}

		if offset != int64(i) {
			t.Errorf("Expected offset %d, got %d", i, offset)
		}
	}

	for i := 0; i < 10; i++ {
		record, err := segment.Read(int64(i))
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}

		if string(record.Key) != "key" {
			t.Errorf("Expected key key, got %s", string(record.Key))
		}
	}
}
