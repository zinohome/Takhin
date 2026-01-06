// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"context"
	"sync"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/logger"
)

// BatchRecord represents a single record in a batch
type BatchRecord struct {
	Key   []byte
	Value []byte
}

// PartitionBatch represents a batch of records for a specific partition
type PartitionBatch struct {
	TopicName string
	Partition int32
	Records   []BatchRecord
	TotalSize int
}

// BatchAggregator aggregates produce requests into batches for better throughput
type BatchAggregator struct {
	config  *config.BatchConfig
	logger  *logger.Logger
	mu      sync.RWMutex
	batches map[string]map[int32]*PartitionBatch // topic -> partition -> batch
	flushCh chan flushRequest
	closeCh chan struct{}
	wg      sync.WaitGroup

	// Adaptive batching state
	avgBatchSize  int
	avgThroughput float64
	lastAdjust    time.Time
}

type flushRequest struct {
	topic      string
	partition  int32
	responseCh chan<- error
}

// NewBatchAggregator creates a new batch aggregator
func NewBatchAggregator(cfg *config.BatchConfig) *BatchAggregator {
	ba := &BatchAggregator{
		config:       cfg,
		logger:       logger.Default().WithComponent("batch-aggregator"),
		batches:      make(map[string]map[int32]*PartitionBatch),
		flushCh:      make(chan flushRequest, 1000),
		closeCh:      make(chan struct{}),
		avgBatchSize: cfg.AdaptiveMinSize,
		lastAdjust:   time.Now(),
	}

	if cfg.LingerMs > 0 {
		ba.wg.Add(1)
		go ba.flushLoop()
	}

	return ba
}

// Add adds a record to the batch for a specific topic-partition
func (ba *BatchAggregator) Add(topic string, partition int32, key, value []byte) (*PartitionBatch, bool) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	if _, exists := ba.batches[topic]; !exists {
		ba.batches[topic] = make(map[int32]*PartitionBatch)
	}

	batch := ba.batches[topic][partition]
	if batch == nil {
		batch = &PartitionBatch{
			TopicName: topic,
			Partition: partition,
			Records:   make([]BatchRecord, 0, ba.getTargetBatchSize()),
		}
		ba.batches[topic][partition] = batch
	}

	recordSize := len(key) + len(value)
	batch.Records = append(batch.Records, BatchRecord{Key: key, Value: value})
	batch.TotalSize += recordSize

	// Check if batch should be flushed
	shouldFlush := ba.shouldFlush(batch)

	if shouldFlush {
		// Remove from pending batches
		delete(ba.batches[topic], partition)
		if len(ba.batches[topic]) == 0 {
			delete(ba.batches, topic)
		}
		return batch, true
	}

	return nil, false
}

// shouldFlush determines if a batch should be flushed based on configured limits
func (ba *BatchAggregator) shouldFlush(batch *PartitionBatch) bool {
	// Check max size limit
	if ba.config.MaxSize > 0 && len(batch.Records) >= ba.config.MaxSize {
		return true
	}

	// Check max bytes limit
	if ba.config.MaxBytes > 0 && batch.TotalSize >= ba.config.MaxBytes {
		return true
	}

	// Check adaptive batch size
	if ba.config.AdaptiveEnabled && len(batch.Records) >= ba.getTargetBatchSize() {
		return true
	}

	return false
}

// getTargetBatchSize returns the current target batch size for adaptive batching
func (ba *BatchAggregator) getTargetBatchSize() int {
	if !ba.config.AdaptiveEnabled {
		if ba.config.MaxSize > 0 {
			return ba.config.MaxSize
		}
		return 1000 // default
	}

	size := ba.avgBatchSize
	if size < ba.config.AdaptiveMinSize {
		size = ba.config.AdaptiveMinSize
	}
	if size > ba.config.AdaptiveMaxSize {
		size = ba.config.AdaptiveMaxSize
	}
	return size
}

// FlushAll flushes all pending batches
func (ba *BatchAggregator) FlushAll() []*PartitionBatch {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	result := make([]*PartitionBatch, 0)

	for topic, partitions := range ba.batches {
		for partition, batch := range partitions {
			if len(batch.Records) > 0 {
				result = append(result, batch)
			}
			delete(partitions, partition)
		}
		delete(ba.batches, topic)
	}

	return result
}

// FlushPartition flushes a specific partition's batch
func (ba *BatchAggregator) FlushPartition(topic string, partition int32) *PartitionBatch {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	if partitions, exists := ba.batches[topic]; exists {
		if batch, exists := partitions[partition]; exists {
			delete(partitions, partition)
			if len(partitions) == 0 {
				delete(ba.batches, topic)
			}
			return batch
		}
	}

	return nil
}

// flushLoop periodically flushes batches based on linger time
func (ba *BatchAggregator) flushLoop() {
	defer ba.wg.Done()

	ticker := time.NewTicker(time.Duration(ba.config.LingerMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			batches := ba.FlushAll()
			if len(batches) > 0 {
				ba.logger.Debug("periodic flush", "batches", len(batches))
			}
		case <-ba.closeCh:
			return
		}
	}
}

// UpdateMetrics updates adaptive batching metrics
func (ba *BatchAggregator) UpdateMetrics(batchSize int, throughput float64) {
	if !ba.config.AdaptiveEnabled {
		return
	}

	// Update moving average
	alpha := 0.2 // smoothing factor
	ba.avgBatchSize = int(alpha*float64(batchSize) + (1-alpha)*float64(ba.avgBatchSize))
	ba.avgThroughput = alpha*throughput + (1-alpha)*ba.avgThroughput

	// Adjust batch size every 5 seconds
	if time.Since(ba.lastAdjust) > 5*time.Second {
		ba.adjustBatchSize()
		ba.lastAdjust = time.Now()
	}
}

// adjustBatchSize adjusts the target batch size based on performance metrics
func (ba *BatchAggregator) adjustBatchSize() {
	// Increase batch size if throughput is improving
	if ba.avgThroughput > 0 {
		newSize := ba.avgBatchSize + ba.avgBatchSize/10 // increase by 10%
		if newSize <= ba.config.AdaptiveMaxSize {
			ba.avgBatchSize = newSize
			ba.logger.Debug("increased batch size",
				"new_size", ba.avgBatchSize,
				"throughput_mbps", ba.avgThroughput,
			)
		}
	}
}

// GetStats returns aggregator statistics
func (ba *BatchAggregator) GetStats() map[string]interface{} {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	totalBatches := 0
	totalRecords := 0

	for _, partitions := range ba.batches {
		for _, batch := range partitions {
			totalBatches++
			totalRecords += len(batch.Records)
		}
	}

	return map[string]interface{}{
		"pending_batches":   totalBatches,
		"pending_records":   totalRecords,
		"target_batch_size": ba.getTargetBatchSize(),
		"avg_throughput":    ba.avgThroughput,
	}
}

// Close stops the aggregator and flushes remaining batches
func (ba *BatchAggregator) Close() {
	close(ba.closeCh)
	ba.wg.Wait()

	// Flush remaining batches
	batches := ba.FlushAll()
	if len(batches) > 0 {
		ba.logger.Info("flushed remaining batches on close", "count", len(batches))
	}
}

// ProcessBatch processes a batch synchronously using the provided handler function
func (ba *BatchAggregator) ProcessBatch(ctx context.Context, batch *PartitionBatch, handler func(*PartitionBatch) error) error {
	startTime := time.Now()
	err := handler(batch)
	duration := time.Since(startTime)

	if err == nil && ba.config.AdaptiveEnabled {
		// Calculate throughput
		throughput := float64(batch.TotalSize) / duration.Seconds() / (1024 * 1024) // MB/s
		ba.UpdateMetrics(len(batch.Records), throughput)
	}

	return err
}
