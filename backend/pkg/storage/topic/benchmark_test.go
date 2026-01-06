// Copyright 2025 Takhin Data, Inc.

package topic

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/storage/log"
)

// BenchmarkTopicManagerProduceThroughput measures produce throughput across multiple partitions
func BenchmarkTopicManagerProduceThroughput(b *testing.B) {
	partitionCounts := []int{1, 4, 16}
	messageSizes := []int{100, 1024, 10240}

	for _, partCount := range partitionCounts {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("partitions=%d/msgSize=%dB", partCount, msgSize)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				mgr := NewManager(dir, 100*1024*1024)

				// Create topic with partitions
				err := mgr.CreateTopic("benchmark-topic", int32(partCount))
				require.NoError(b, err)

				value := make([]byte, msgSize)
				key := []byte("benchmark-key")

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					partitionID := int32(i % partCount)
					topic, exists := mgr.GetTopic("benchmark-topic")
					require.True(b, exists)
					require.NotNil(b, topic)

					partition := topic.Partitions[partitionID]
					_, err := partition.Append(key, value)
					if err != nil {
						b.Fatal(err)
					}
				}

				b.StopTimer()
				totalBytes := int64(b.N) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msg/s")
			})
		}
	}
}

// BenchmarkTopicManagerFetchThroughput measures fetch throughput across multiple partitions
func BenchmarkTopicManagerFetchThroughput(b *testing.B) {
	partitionCounts := []int{1, 4, 16}
	messageSizes := []int{100, 1024, 10240}

	for _, partCount := range partitionCounts {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("partitions=%d/msgSize=%dB", partCount, msgSize)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				mgr := NewManager(dir, 100*1024*1024)

				// Create topic with partitions
				err := mgr.CreateTopic("benchmark-topic", int32(partCount))
				require.NoError(b, err)

				topic, exists := mgr.GetTopic("benchmark-topic")
				require.True(b, exists)
				require.NotNil(b, topic)

				// Pre-populate
				value := make([]byte, msgSize)
				key := []byte("benchmark-key")
				numMessages := 10000
				for i := 0; i < numMessages; i++ {
					partitionID := int32(i % partCount)
					partition := topic.Partitions[partitionID]
					_, err := partition.Append(key, value)
					require.NoError(b, err)
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					partitionID := int32(i % partCount)
					partition := topic.Partitions[partitionID]
					offset := int64(i/partCount) % int64(numMessages/partCount)
					_, err := partition.Read(offset)
					if err != nil {
						b.Fatal(err)
					}
				}

				b.StopTimer()
				totalBytes := int64(b.N) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msg/s")
			})
		}
	}
}

// BenchmarkTopicManagerConcurrentProducers measures multi-producer throughput to multiple partitions
func BenchmarkTopicManagerConcurrentProducers(b *testing.B) {
	producerCounts := []int{1, 4, 8, 16}
	partitionCounts := []int{4, 16}

	for _, producerCount := range producerCounts {
		for _, partCount := range partitionCounts {
			name := fmt.Sprintf("producers=%d/partitions=%d", producerCount, partCount)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				mgr := NewManager(dir, 100*1024*1024)

				// Create topic with partitions
				err := mgr.CreateTopic("benchmark-topic", int32(partCount))
				require.NoError(b, err)

				msgSize := 1024
				value := make([]byte, msgSize)

				b.ResetTimer()
				b.ReportAllocs()

				var wg sync.WaitGroup
				opsPerProducer := b.N / producerCount

				for p := 0; p < producerCount; p++ {
					wg.Add(1)
					go func(producerID int) {
						defer wg.Done()
						key := []byte(fmt.Sprintf("producer-%d", producerID))
						topic, exists := mgr.GetTopic("benchmark-topic")

						require.True(b, exists)

						require.NotNil(b, topic)

						for i := 0; i < opsPerProducer; i++ {
							partitionID := int32((producerID + i) % partCount)
							partition := topic.Partitions[partitionID]
							_, err := partition.Append(key, value)
							if err != nil {
								b.Error(err)
								return
							}
						}
					}(p)
				}

				wg.Wait()
				b.StopTimer()

				totalBytes := int64(b.N) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msg/s")
			})
		}
	}
}

// BenchmarkTopicManagerConcurrentConsumers measures multi-consumer fetch throughput from multiple partitions
func BenchmarkTopicManagerConcurrentConsumers(b *testing.B) {
	consumerCounts := []int{1, 4, 8, 16}
	partitionCounts := []int{4, 16}

	for _, consumerCount := range consumerCounts {
		for _, partCount := range partitionCounts {
			name := fmt.Sprintf("consumers=%d/partitions=%d", consumerCount, partCount)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				mgr := NewManager(dir, 100*1024*1024)

				// Create topic with partitions
				err := mgr.CreateTopic("benchmark-topic", int32(partCount))
				require.NoError(b, err)

				topic, exists := mgr.GetTopic("benchmark-topic")
				require.True(b, exists)
				require.NotNil(b, topic)

				// Pre-populate
				msgSize := 1024
				value := make([]byte, msgSize)
				key := []byte("benchmark-key")
				numMessages := 10000
				for i := 0; i < numMessages; i++ {
					partitionID := int32(i % partCount)
					partition := topic.Partitions[partitionID]
					_, err := partition.Append(key, value)
					require.NoError(b, err)
				}

				b.ResetTimer()
				b.ReportAllocs()

				var wg sync.WaitGroup
				opsPerConsumer := b.N / consumerCount

				for c := 0; c < consumerCount; c++ {
					wg.Add(1)
					go func(consumerID int) {
						defer wg.Done()
						rng := rand.New(rand.NewSource(int64(consumerID)))
						topic, exists := mgr.GetTopic("benchmark-topic")

						require.True(b, exists)

						require.NotNil(b, topic)

						for i := 0; i < opsPerConsumer; i++ {
							partitionID := int32(rng.Intn(partCount))
							partition := topic.Partitions[partitionID]
							offset := int64(rng.Intn(numMessages / partCount))
							_, err := partition.Read(offset)
							if err != nil {
								b.Error(err)
								return
							}
						}
					}(c)
				}

				wg.Wait()
				b.StopTimer()

				totalBytes := int64(b.N) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msg/s")
			})
		}
	}
}

// BenchmarkTopicManagerPartitionBalance measures load distribution across partitions
func BenchmarkTopicManagerPartitionBalance(b *testing.B) {
	partitionCounts := []int{4, 8, 16, 32}

	for _, partCount := range partitionCounts {
		name := fmt.Sprintf("partitions=%d", partCount)
		b.Run(name, func(b *testing.B) {
			dir := b.TempDir()
			mgr := NewManager(dir, 100*1024*1024)

			// Create topic with partitions
			err := mgr.CreateTopic("benchmark-topic", int32(partCount))
			require.NoError(b, err)

			msgSize := 1024
			value := make([]byte, msgSize)

			// Track writes per partition
			partitionCounts := make(map[int32]int)

			b.ResetTimer()
			b.ReportAllocs()

			topic, exists := mgr.GetTopic("benchmark-topic")

			require.True(b, exists)

			require.NotNil(b, topic)
			for i := 0; i < b.N; i++ {
				// Use key-based partitioning (hash key % partitions)
				key := []byte(fmt.Sprintf("key-%d", i))
				// Simple hash: sum of bytes
				var hash int32
				for _, b := range key {
					hash += int32(b)
				}
				partitionID := hash % int32(partCount)
				if partitionID < 0 {
					partitionID = -partitionID
				}

				partition := topic.Partitions[partitionID]
				_, err := partition.Append(key, value)
				if err != nil {
					b.Fatal(err)
				}
				partitionCounts[partitionID]++
			}

			b.StopTimer()

			// Calculate balance metrics
			var maxCount, minCount int
			for _, count := range partitionCounts {
				if maxCount == 0 || count > maxCount {
					maxCount = count
				}
				if minCount == 0 || count < minCount {
					minCount = count
				}
			}

			imbalance := float64(maxCount-minCount) / float64(b.N) * 100
			b.ReportMetric(imbalance, "imbalance_%")
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msg/s")
		})
	}
}

// BenchmarkTopicManagerMultiTopic measures performance with multiple topics
func BenchmarkTopicManagerMultiTopic(b *testing.B) {
	topicCounts := []int{1, 5, 10}
	partitionsPerTopic := 4

	for _, topicCount := range topicCounts {
		name := fmt.Sprintf("topics=%d/partitions=%d", topicCount, partitionsPerTopic)
		b.Run(name, func(b *testing.B) {
			dir := b.TempDir()
			mgr := NewManager(dir, 100*1024*1024)

			// Create multiple topics
			for i := 0; i < topicCount; i++ {
				topicName := fmt.Sprintf("topic-%d", i)
				err := mgr.CreateTopic(topicName, int32(partitionsPerTopic))
				require.NoError(b, err)
			}

			msgSize := 1024
			value := make([]byte, msgSize)
			key := []byte("benchmark-key")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				topicID := i % topicCount
				topicName := fmt.Sprintf("topic-%d", topicID)
				topic, exists := mgr.GetTopic(topicName)
				require.True(b, exists)
				require.NotNil(b, topic)

				partitionID := int32((i / topicCount) % partitionsPerTopic)
				partition := topic.Partitions[partitionID]
				_, err := partition.Append(key, value)
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			totalBytes := int64(b.N) * int64(msgSize)
			b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msg/s")
		})
	}
}

// BenchmarkTopicManagerCompaction measures compaction performance across partitions
func BenchmarkTopicManagerCompaction(b *testing.B) {
	partitionCounts := []int{1, 4, 8}

	for _, partCount := range partitionCounts {
		name := fmt.Sprintf("partitions=%d", partCount)
		b.Run(name, func(b *testing.B) {
			dir := b.TempDir()
			mgr := NewManager(dir, 10*1024) // Small segments for compaction

			// Create topic with partitions
			err := mgr.CreateTopic("benchmark-topic", int32(partCount))
			require.NoError(b, err)

			topic, exists := mgr.GetTopic("benchmark-topic")
			require.True(b, exists)
			require.NotNil(b, topic)

			// Pre-populate with duplicates
			msgSize := 1024
			value := make([]byte, msgSize)
			numMessages := 1000
			numUniqueKeys := 50

			for i := 0; i < numMessages; i++ {
				keyID := i % numUniqueKeys
				key := []byte(fmt.Sprintf("key-%d", keyID))
				partitionID := int32(i % partCount)
				partition := topic.Partitions[partitionID]
				_, err := partition.Append(key, value)
				require.NoError(b, err)
			}

			policy := log.DefaultCompactionPolicy()
			policy.MinCleanableRatio = 0.1

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var totalBytesReclaimed int64
				var totalDurationMs int64

				for partID := int32(0); partID < int32(partCount); partID++ {
					partition := topic.Partitions[partID]
					result, err := partition.Compact(policy)
					if err != nil {
						b.Fatal(err)
					}
					totalBytesReclaimed += result.BytesReclaimed
					totalDurationMs += result.DurationMs
				}

				b.ReportMetric(float64(totalBytesReclaimed)/1024/1024, "MB_reclaimed")
				b.ReportMetric(float64(totalDurationMs), "duration_ms")
			}
		})
	}
}
