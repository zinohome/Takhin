// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// BenchmarkProduceThroughput_Single measures single record produce throughput
func BenchmarkProduceThroughput_Single(t *testing.B) {
	sizes := []int{100, 1024, 10240} // 100B, 1KB, 10KB

	for _, msgSize := range sizes {
		name := fmt.Sprintf("size=%dB", msgSize)
		t.Run(name, func(b *testing.B) {
			cfg := &config.Config{
				Storage: config.StorageConfig{
					DataDir:        b.TempDir(),
					LogSegmentSize: 1024 * 1024 * 1024, // 1GB
				},
			}

			mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
			require.NoError(b, mgr.CreateTopic("test-topic", 1))

			handler := New(cfg, mgr)
			defer handler.Close()

			key := []byte("benchmark-key")
			value := make([]byte, msgSize)
			topicObj, _ := handler.backend.GetTopic("test-topic")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := topicObj.Append(0, key, value)
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			totalBytes := int64(b.N) * int64(msgSize)
			b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
		})
	}
}

// BenchmarkProduceThroughput_Batch measures batch produce throughput
func BenchmarkProduceThroughput_Batch(b *testing.B) {
	batchSizes := []int{10, 100, 1000}
	msgSizes := []int{100, 1024, 10240}

	for _, batchSize := range batchSizes {
		for _, msgSize := range msgSizes {
			name := fmt.Sprintf("batch=%d/size=%dB", batchSize, msgSize)
			b.Run(name, func(b *testing.B) {
				cfg := &config.Config{
					Storage: config.StorageConfig{
						DataDir:        b.TempDir(),
						LogSegmentSize: 10 * 1024 * 1024 * 1024, // 10GB
					},
				}

				mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
				require.NoError(b, mgr.CreateTopic("test-topic", 1))

				handler := New(cfg, mgr)
				defer handler.Close()

				value := make([]byte, msgSize)
				records := make([]BatchRecord, batchSize)
				for i := 0; i < batchSize; i++ {
					records[i] = BatchRecord{
						Key:   []byte(fmt.Sprintf("key-%d", i)),
						Value: value,
					}
				}

				topicObj, _ := handler.backend.GetTopic("test-topic")

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					logRecords := make([]struct{ Key, Value []byte }, len(records))
					for i, rec := range records {
						logRecords[i].Key = rec.Key
						logRecords[i].Value = rec.Value
					}
					_, err := topicObj.AppendBatch(0, logRecords)
					if err != nil {
						b.Fatal(err)
					}
				}

				b.StopTimer()
				totalBytes := int64(b.N) * int64(batchSize) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
			})
		}
	}
}

// BenchmarkBatchAggregator_Throughput measures aggregator throughput
func BenchmarkBatchAggregator_Throughput(b *testing.B) {
	scenarios := []struct {
		name     string
		maxSize  int
		maxBytes int
		adaptive bool
	}{
		{"fixed-small", 10, 10240, false},
		{"fixed-medium", 100, 102400, false},
		{"fixed-large", 1000, 1048576, false},
		{"adaptive", 1000, 1048576, true},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			cfg := &config.BatchConfig{
				MaxSize:         sc.maxSize,
				MaxBytes:        sc.maxBytes,
				LingerMs:        0,
				AdaptiveEnabled: sc.adaptive,
				AdaptiveMinSize: 10,
				AdaptiveMaxSize: 1000,
			}

			ba := NewBatchAggregator(cfg)
			defer ba.Close()

			key := []byte("benchmark-key")
			value := make([]byte, 1024)

			b.ResetTimer()
			b.ReportAllocs()

			totalFlushed := 0
			for i := 0; i < b.N; i++ {
				batch, shouldFlush := ba.Add("test-topic", 0, key, value)
				if shouldFlush {
					totalFlushed++
					// Simulate processing
					if sc.adaptive {
						throughput := float64(batch.TotalSize) / (1024 * 1024) * 1000 // MB/s estimate
						ba.UpdateMetrics(len(batch.Records), throughput)
					}
				}
			}

			b.StopTimer()
			totalBytes := int64(b.N) * int64(len(value))
			b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
			b.ReportMetric(float64(totalFlushed), "batches_flushed")
		})
	}
}

// BenchmarkBatchVsSingle compares batch vs single append performance
func BenchmarkBatchVsSingle(b *testing.B) {
	msgCount := 1000
	msgSize := 1024

	b.Run("single", func(b *testing.B) {
		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir:        b.TempDir(),
				LogSegmentSize: 10 * 1024 * 1024 * 1024,
			},
		}

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
		require.NoError(b, mgr.CreateTopic("test-topic", 1))

		topicObj, _ := mgr.GetTopic("test-topic")
		key := []byte("key")
		value := make([]byte, msgSize)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			for j := 0; j < msgCount; j++ {
				_, err := topicObj.Append(0, key, value)
				if err != nil {
					b.Fatal(err)
				}
			}
		}

		b.StopTimer()
		totalBytes := int64(b.N) * int64(msgCount) * int64(msgSize)
		b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
	})

	b.Run("batch", func(b *testing.B) {
		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir:        b.TempDir(),
				LogSegmentSize: 10 * 1024 * 1024 * 1024,
			},
		}

		mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
		require.NoError(b, mgr.CreateTopic("test-topic", 1))

		topicObj, _ := mgr.GetTopic("test-topic")
		value := make([]byte, msgSize)

		records := make([]BatchRecord, msgCount)
		for i := 0; i < msgCount; i++ {
			records[i] = BatchRecord{
				Key:   []byte(fmt.Sprintf("key-%d", i)),
				Value: value,
			}
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			logRecords := make([]struct{ Key, Value []byte }, len(records))
			for i, rec := range records {
				logRecords[i].Key = rec.Key
				logRecords[i].Value = rec.Value
			}
			_, err := topicObj.AppendBatch(0, logRecords)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.StopTimer()
		totalBytes := int64(b.N) * int64(msgCount) * int64(msgSize)
		b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
	})
}

// BenchmarkConcurrentBatchProduce measures concurrent batch produce performance
func BenchmarkConcurrentBatchProduce(b *testing.B) {
	concurrency := []int{1, 2, 4, 8}

	for _, c := range concurrency {
		name := fmt.Sprintf("producers=%d", c)
		b.Run(name, func(b *testing.B) {
			cfg := &config.Config{
				Storage: config.StorageConfig{
					DataDir:        b.TempDir(),
					LogSegmentSize: 10 * 1024 * 1024 * 1024,
				},
			}

			mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
			require.NoError(b, mgr.CreateTopic("test-topic", int32(c)))

			handler := New(cfg, mgr)
			defer handler.Close()

			value := make([]byte, 1024)
			batchSize := 100
			records := make([]BatchRecord, batchSize)
			for i := 0; i < batchSize; i++ {
				records[i] = BatchRecord{
					Key:   []byte(fmt.Sprintf("key-%d", i)),
					Value: value,
				}
			}

			topicObj, _ := handler.backend.GetTopic("test-topic")

			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				partition := int32(0)
				for pb.Next() {
					logRecords := make([]struct{ Key, Value []byte }, len(records))
					for i, rec := range records {
						logRecords[i].Key = rec.Key
						logRecords[i].Value = rec.Value
					}
					_, err := topicObj.AppendBatch(partition, logRecords)
					if err != nil {
						b.Fatal(err)
					}
					partition = (partition + 1) % int32(c)
				}
			})

			b.StopTimer()
			totalBytes := int64(b.N) * int64(batchSize) * int64(len(value))
			b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
		})
	}
}

// BenchmarkAdaptiveBatching measures adaptive batch sizing performance
func BenchmarkAdaptiveBatching(b *testing.B) {
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir:        b.TempDir(),
			LogSegmentSize: 10 * 1024 * 1024 * 1024,
		},
		Kafka: config.KafkaConfig{
			Batch: config.BatchConfig{
				MaxSize:         10000,
				MaxBytes:        10485760, // 10MB
				LingerMs:        0,
				AdaptiveEnabled: true,
				AdaptiveMinSize: 10,
				AdaptiveMaxSize: 5000,
			},
		},
	}

	mgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	require.NoError(b, mgr.CreateTopic("test-topic", 1))

	handler := New(cfg, mgr)
	defer handler.Close()

	ba := NewBatchAggregator(&cfg.Kafka.Batch)
	defer ba.Close()

	topicObj, _ := handler.backend.GetTopic("test-topic")
	key := []byte("key")
	value := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	ctx := context.Background()
	totalRecords := 0

	for i := 0; i < b.N; i++ {
		batch, shouldFlush := ba.Add("test-topic", 0, key, value)
		if shouldFlush && batch != nil {
			totalRecords += len(batch.Records)

			// Process batch
			err := ba.ProcessBatch(ctx, batch, func(pb *PartitionBatch) error {
				logRecords := make([]struct{ Key, Value []byte }, len(pb.Records))
				for i, rec := range pb.Records {
					logRecords[i].Key = rec.Key
					logRecords[i].Value = rec.Value
				}
				_, err := topicObj.AppendBatch(pb.Partition, logRecords)
				return err
			})

			if err != nil {
				b.Fatal(err)
			}
		}
	}

	b.StopTimer()
	totalBytes := int64(totalRecords) * int64(len(value))
	b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
	b.ReportMetric(float64(totalRecords)/float64(b.N), "avg_batch_size")
}
