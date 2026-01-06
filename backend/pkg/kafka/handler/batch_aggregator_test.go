// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
)

func TestBatchAggregator_Basic(t *testing.T) {
	cfg := &config.BatchConfig{
		MaxSize:  100,
		MaxBytes: 10240,
		LingerMs: 0, // disabled for testing
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	// Add records to the same partition
	topic := "test-topic"
	partition := int32(0)

	// Add records that don't trigger flush
	batch, shouldFlush := ba.Add(topic, partition, []byte("key1"), []byte("value1"))
	assert.Nil(t, batch)
	assert.False(t, shouldFlush)

	batch, shouldFlush = ba.Add(topic, partition, []byte("key2"), []byte("value2"))
	assert.Nil(t, batch)
	assert.False(t, shouldFlush)

	// Manually flush
	flushed := ba.FlushPartition(topic, partition)
	require.NotNil(t, flushed)
	assert.Equal(t, 2, len(flushed.Records))
	assert.Equal(t, topic, flushed.TopicName)
	assert.Equal(t, partition, flushed.Partition)
}

func TestBatchAggregator_MaxSizeFlush(t *testing.T) {
	cfg := &config.BatchConfig{
		MaxSize:  3,
		MaxBytes: 100000,
		LingerMs: 0,
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	topic := "test-topic"
	partition := int32(0)

	// Add records up to max size
	ba.Add(topic, partition, []byte("key1"), []byte("value1"))
	ba.Add(topic, partition, []byte("key2"), []byte("value2"))

	// Third add should trigger flush
	batch, shouldFlush := ba.Add(topic, partition, []byte("key3"), []byte("value3"))
	require.True(t, shouldFlush)
	require.NotNil(t, batch)
	assert.Equal(t, 3, len(batch.Records))
}

func TestBatchAggregator_MaxBytesFlush(t *testing.T) {
	cfg := &config.BatchConfig{
		MaxSize:  1000,
		MaxBytes: 50, // very small to trigger quickly
		LingerMs: 0,
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	topic := "test-topic"
	partition := int32(0)

	largeValue := make([]byte, 30)

	ba.Add(topic, partition, []byte("key"), largeValue)

	// Second add should trigger flush due to byte limit
	batch, shouldFlush := ba.Add(topic, partition, []byte("key"), largeValue)
	require.True(t, shouldFlush)
	require.NotNil(t, batch)
	assert.True(t, batch.TotalSize >= cfg.MaxBytes)
}

func TestBatchAggregator_FlushAll(t *testing.T) {
	cfg := &config.BatchConfig{
		MaxSize:  100,
		MaxBytes: 10240,
		LingerMs: 0,
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	// Add records to multiple partitions
	ba.Add("topic1", 0, []byte("key1"), []byte("value1"))
	ba.Add("topic1", 1, []byte("key2"), []byte("value2"))
	ba.Add("topic2", 0, []byte("key3"), []byte("value3"))

	batches := ba.FlushAll()
	assert.Equal(t, 3, len(batches))

	// Verify batches are now empty
	stats := ba.GetStats()
	assert.Equal(t, 0, stats["pending_batches"])
}

func TestBatchAggregator_AdaptiveBatching(t *testing.T) {
	cfg := &config.BatchConfig{
		MaxSize:         1000,
		MaxBytes:        100000,
		LingerMs:        0,
		AdaptiveEnabled: true,
		AdaptiveMinSize: 10,
		AdaptiveMaxSize: 500,
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	initialSize := ba.getTargetBatchSize()
	assert.Equal(t, cfg.AdaptiveMinSize, initialSize)

	// Simulate successful batches with good throughput
	ba.UpdateMetrics(50, 100.0) // 100 MB/s throughput
	ba.UpdateMetrics(50, 100.0)
	ba.lastAdjust = time.Now().Add(-10 * time.Second) // force adjustment
	ba.UpdateMetrics(50, 100.0)

	newSize := ba.getTargetBatchSize()
	assert.True(t, newSize >= initialSize, "batch size should increase with good throughput")
}

func TestBatchAggregator_Stats(t *testing.T) {
	cfg := &config.BatchConfig{
		MaxSize:  100,
		MaxBytes: 10240,
		LingerMs: 0,
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	ba.Add("topic1", 0, []byte("key1"), []byte("value1"))
	ba.Add("topic1", 0, []byte("key2"), []byte("value2"))
	ba.Add("topic1", 1, []byte("key3"), []byte("value3"))

	stats := ba.GetStats()
	assert.Equal(t, 2, stats["pending_batches"])
	assert.Equal(t, 3, stats["pending_records"])
}

func TestBatchAggregator_PeriodicFlush(t *testing.T) {
	cfg := &config.BatchConfig{
		MaxSize:  100,
		MaxBytes: 10240,
		LingerMs: 50, // 50ms linger time
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	topic := "test-topic"
	partition := int32(0)

	// Add records that won't trigger immediate flush
	ba.Add(topic, partition, []byte("key1"), []byte("value1"))
	ba.Add(topic, partition, []byte("key2"), []byte("value2"))

	// Verify batches are pending
	stats := ba.GetStats()
	assert.Equal(t, 1, stats["pending_batches"])

	// Wait for periodic flush (linger time + buffer)
	time.Sleep(100 * time.Millisecond)

	// Batches should be flushed by periodic timer
	stats = ba.GetStats()
	assert.Equal(t, 0, stats["pending_batches"])
}

func BenchmarkBatchAggregator_Add(b *testing.B) {
	cfg := &config.BatchConfig{
		MaxSize:  10000,
		MaxBytes: 1048576,
		LingerMs: 0,
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	key := []byte("benchmark-key")
	value := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		batch, shouldFlush := ba.Add("test-topic", 0, key, value)
		if shouldFlush && batch != nil {
			// Simulate processing flushed batch
			_ = batch
		}
	}
}

func BenchmarkBatchAggregator_AddMultiPartition(b *testing.B) {
	cfg := &config.BatchConfig{
		MaxSize:  10000,
		MaxBytes: 1048576,
		LingerMs: 0,
	}

	ba := NewBatchAggregator(cfg)
	defer ba.Close()

	key := []byte("benchmark-key")
	value := make([]byte, 1024)
	numPartitions := 16

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		partition := int32(i % numPartitions)
		batch, shouldFlush := ba.Add("test-topic", partition, key, value)
		if shouldFlush && batch != nil {
			_ = batch
		}
	}
}

func BenchmarkBatchAggregator_FlushAll(b *testing.B) {
	cfg := &config.BatchConfig{
		MaxSize:  10000,
		MaxBytes: 1048576,
		LingerMs: 0,
	}

	key := []byte("benchmark-key")
	value := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ba := NewBatchAggregator(cfg)

		// Add 1000 records across 10 partitions
		for j := 0; j < 1000; j++ {
			ba.Add("test-topic", int32(j%10), key, value)
		}

		batches := ba.FlushAll()
		_ = batches

		ba.Close()
	}
}
