// Copyright 2025 Takhin Data, Inc.

package log

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// BenchmarkWriteThroughput measures write throughput
func BenchmarkWriteThroughput(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	messageSizes := []int{100, 1024, 10240} // 100B, 1KB, 10KB

	for _, size := range sizes {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("messages=%d/size=%dB", size, msgSize)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				config := LogConfig{
					Dir:            dir,
					MaxSegmentSize: 100 * 1024 * 1024, // 100MB
				}

				log, err := NewLog(config)
				require.NoError(b, err)
				defer log.Close()

				value := make([]byte, msgSize)
				key := []byte("benchmark-key")

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					for j := 0; j < size; j++ {
						_, err := log.Append(key, value)
						if err != nil {
							b.Fatal(err)
						}
					}
				}

				b.StopTimer()
				totalBytes := int64(b.N) * int64(size) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
			})
		}
	}
}

// BenchmarkBatchWriteThroughput measures batch write throughput
func BenchmarkBatchWriteThroughput(b *testing.B) {
	batchSizes := []int{10, 100, 1000}
	messageSizes := []int{100, 1024, 10240} // 100B, 1KB, 10KB

	for _, batchSize := range batchSizes {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("batch=%d/size=%dB", batchSize, msgSize)
			b.Run(name, func(b *testing.B) {
				value := make([]byte, msgSize)
				batch := make([]struct{ Key, Value []byte }, batchSize)
				for i := 0; i < batchSize; i++ {
					batch[i].Key = []byte("benchmark-key")
					batch[i].Value = value
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					b.StopTimer()
					dir := b.TempDir()
					config := LogConfig{
						Dir:            dir,
						MaxSegmentSize: 1024 * 1024 * 1024, // 1GB
					}

					log, err := NewLog(config)
					require.NoError(b, err)
					b.StartTimer()

					_, err = log.AppendBatch(batch)
					if err != nil {
						b.Fatal(err)
					}

					b.StopTimer()
					log.Close()
					b.StartTimer()
				}

				b.StopTimer()
				totalBytes := int64(b.N) * int64(batchSize) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
			})
		}
	}
}

// BenchmarkReadThroughput measures read throughput
func BenchmarkReadThroughput(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	messageSizes := []int{100, 1024, 10240}

	for _, size := range sizes {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("messages=%d/size=%dB", size, msgSize)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				config := LogConfig{
					Dir:            dir,
					MaxSegmentSize: 100 * 1024 * 1024,
				}

				log, err := NewLog(config)
				require.NoError(b, err)
				defer log.Close()

				// Pre-populate with data
				value := make([]byte, msgSize)
				key := []byte("benchmark-key")
				for i := 0; i < size; i++ {
					_, err := log.Append(key, value)
					require.NoError(b, err)
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					for j := 0; j < size; j++ {
						_, err := log.Read(int64(j))
						if err != nil {
							b.Fatal(err)
						}
					}
				}

				b.StopTimer()
				totalBytes := int64(b.N) * int64(size) * int64(msgSize)
				b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
			})
		}
	}
}

// BenchmarkMixedWorkload simulates a mixed read/write workload
func BenchmarkMixedWorkload(b *testing.B) {
	dir := b.TempDir()
	config := LogConfig{
		Dir:            dir,
		MaxSegmentSize: 100 * 1024 * 1024,
	}

	log, err := NewLog(config)
	require.NoError(b, err)
	defer log.Close()

	// Pre-populate
	msgSize := 1024
	value := make([]byte, msgSize)
	key := []byte("benchmark-key")
	for i := 0; i < 1000; i++ {
		_, err := log.Append(key, value)
		require.NoError(b, err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	offset := int64(0)
	for i := 0; i < b.N; i++ {
		// Write
		_, err := log.Append(key, value)
		if err != nil {
			b.Fatal(err)
		}

		// Read
		_, err = log.Read(offset % 1000)
		if err != nil {
			b.Fatal(err)
		}
		offset++
	}
}

// BenchmarkSequentialFetch measures sequential read performance (Kafka fetch pattern)
func BenchmarkSequentialFetch(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	messageSizes := []int{100, 1024, 10240}
	fetchSizes := []int{1, 10, 100} // Number of messages to fetch at once

	for _, size := range sizes {
		for _, msgSize := range messageSizes {
			for _, fetchSize := range fetchSizes {
				name := fmt.Sprintf("messages=%d/msgSize=%dB/fetch=%d", size, msgSize, fetchSize)
				b.Run(name, func(b *testing.B) {
					dir := b.TempDir()
					config := LogConfig{
						Dir:            dir,
						MaxSegmentSize: 100 * 1024 * 1024,
					}

					log, err := NewLog(config)
					require.NoError(b, err)
					defer log.Close()

					// Pre-populate with data
					value := make([]byte, msgSize)
					key := []byte("benchmark-key")
					for i := 0; i < size; i++ {
						_, err := log.Append(key, value)
						require.NoError(b, err)
					}

					b.ResetTimer()
					b.ReportAllocs()

					totalFetched := int64(0)
					for i := 0; i < b.N; i++ {
						offset := int64(i*fetchSize) % int64(size)
						for j := 0; j < fetchSize && offset+int64(j) < int64(size); j++ {
							_, err := log.Read(offset + int64(j))
							if err != nil {
								b.Fatal(err)
							}
							totalFetched++
						}
					}

					b.StopTimer()
					totalBytes := totalFetched * int64(msgSize)
					b.ReportMetric(float64(totalBytes)/b.Elapsed().Seconds()/1024/1024, "MB/s")
					b.ReportMetric(float64(totalFetched)/b.Elapsed().Seconds(), "msg/s")
				})
			}
		}
	}
}

// BenchmarkRandomFetch measures random read performance
func BenchmarkRandomFetch(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	messageSizes := []int{100, 1024, 10240}

	for _, size := range sizes {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("messages=%d/size=%dB", size, msgSize)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				config := LogConfig{
					Dir:            dir,
					MaxSegmentSize: 100 * 1024 * 1024,
				}

				log, err := NewLog(config)
				require.NoError(b, err)
				defer log.Close()

				// Pre-populate with data
				value := make([]byte, msgSize)
				key := []byte("benchmark-key")
				for i := 0; i < size; i++ {
					_, err := log.Append(key, value)
					require.NoError(b, err)
				}

				// Pre-generate random offsets
				rng := rand.New(rand.NewSource(42))
				offsets := make([]int64, b.N)
				for i := 0; i < b.N; i++ {
					offsets[i] = int64(rng.Intn(size))
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					_, err := log.Read(offsets[i])
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

// BenchmarkProduceLatency measures single message produce latency
func BenchmarkProduceLatency(b *testing.B) {
	messageSizes := []int{100, 1024, 10240}

	for _, msgSize := range messageSizes {
		name := fmt.Sprintf("size=%dB", msgSize)
		b.Run(name, func(b *testing.B) {
			dir := b.TempDir()
			config := LogConfig{
				Dir:            dir,
				MaxSegmentSize: 100 * 1024 * 1024,
			}

			log, err := NewLog(config)
			require.NoError(b, err)
			defer log.Close()

			value := make([]byte, msgSize)
			key := []byte("benchmark-key")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := log.Append(key, value)
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			avgLatencyMs := float64(b.Elapsed().Nanoseconds()) / float64(b.N) / 1e6
			b.ReportMetric(avgLatencyMs, "ms/op")
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/s")
		})
	}
}

// BenchmarkFetchLatency measures single message fetch latency
func BenchmarkFetchLatency(b *testing.B) {
	messageSizes := []int{100, 1024, 10240}

	for _, msgSize := range messageSizes {
		name := fmt.Sprintf("size=%dB", msgSize)
		b.Run(name, func(b *testing.B) {
			dir := b.TempDir()
			config := LogConfig{
				Dir:            dir,
				MaxSegmentSize: 100 * 1024 * 1024,
			}

			log, err := NewLog(config)
			require.NoError(b, err)
			defer log.Close()

			// Pre-populate
			value := make([]byte, msgSize)
			key := []byte("benchmark-key")
			numMessages := 10000
			for i := 0; i < numMessages; i++ {
				_, err := log.Append(key, value)
				require.NoError(b, err)
			}

			// Pre-generate random offsets
			rng := rand.New(rand.NewSource(42))
			offsets := make([]int64, b.N)
			for i := 0; i < b.N; i++ {
				offsets[i] = int64(rng.Intn(numMessages))
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := log.Read(offsets[i])
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			avgLatencyMs := float64(b.Elapsed().Nanoseconds()) / float64(b.N) / 1e6
			b.ReportMetric(avgLatencyMs, "ms/op")
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops/s")
		})
	}
}

// BenchmarkCompaction measures compaction performance
func BenchmarkCompaction(b *testing.B) {
	segmentCounts := []int{10, 50, 100}
	messageSizes := []int{100, 1024}
	deduplicationRatios := []float64{0.3, 0.5, 0.7} // % of duplicate keys

	for _, segCount := range segmentCounts {
		for _, msgSize := range messageSizes {
			for _, dedupRatio := range deduplicationRatios {
				name := fmt.Sprintf("segments=%d/msgSize=%dB/dedup=%.0f%%", segCount, msgSize, dedupRatio*100)
				b.Run(name, func(b *testing.B) {
					dir := b.TempDir()
					config := LogConfig{
						Dir:            dir,
						MaxSegmentSize: 10 * 1024, // Small segments for compaction testing
					}

					log, err := NewLog(config)
					require.NoError(b, err)
					defer log.Close()

					// Pre-populate with data including duplicates
					value := make([]byte, msgSize)
					numUniqueKeys := int(100 * (1 - dedupRatio))
					messagesPerSegment := 20

					for i := 0; i < segCount*messagesPerSegment; i++ {
						keyID := i % numUniqueKeys
						key := []byte(fmt.Sprintf("key-%d", keyID))
						_, err := log.Append(key, value)
						require.NoError(b, err)
					}

					policy := DefaultCompactionPolicy()
					policy.MinCleanableRatio = 0.1 // Compact aggressively

					b.ResetTimer()
					b.ReportAllocs()

					for i := 0; i < b.N; i++ {
						result, err := log.Compact(policy)
						if err != nil {
							b.Fatal(err)
						}
						b.ReportMetric(float64(result.BytesReclaimed)/1024/1024, "MB_reclaimed")
						b.ReportMetric(float64(result.KeysRemoved), "keys_removed")
						b.ReportMetric(float64(result.DurationMs), "duration_ms")
					}
				})
			}
		}
	}
}

// BenchmarkConcurrentProducers measures multi-producer throughput
func BenchmarkConcurrentProducers(b *testing.B) {
	producerCounts := []int{1, 2, 4, 8}
	messageSizes := []int{100, 1024}

	for _, producerCount := range producerCounts {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("producers=%d/msgSize=%dB", producerCount, msgSize)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				config := LogConfig{
					Dir:            dir,
					MaxSegmentSize: 100 * 1024 * 1024,
				}

				log, err := NewLog(config)
				require.NoError(b, err)
				defer log.Close()

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
						for i := 0; i < opsPerProducer; i++ {
							_, err := log.Append(key, value)
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

// BenchmarkConcurrentConsumers measures multi-consumer fetch throughput
func BenchmarkConcurrentConsumers(b *testing.B) {
	consumerCounts := []int{1, 2, 4, 8}
	messageSizes := []int{100, 1024}

	for _, consumerCount := range consumerCounts {
		for _, msgSize := range messageSizes {
			name := fmt.Sprintf("consumers=%d/msgSize=%dB", consumerCount, msgSize)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				config := LogConfig{
					Dir:            dir,
					MaxSegmentSize: 100 * 1024 * 1024,
				}

				log, err := NewLog(config)
				require.NoError(b, err)
				defer log.Close()

				// Pre-populate
				value := make([]byte, msgSize)
				key := []byte("benchmark-key")
				numMessages := 10000
				for i := 0; i < numMessages; i++ {
					_, err := log.Append(key, value)
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
						for i := 0; i < opsPerConsumer; i++ {
							offset := int64(rng.Intn(numMessages))
							_, err := log.Read(offset)
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

// BenchmarkSegmentRollover measures performance impact of segment rollover
func BenchmarkSegmentRollover(b *testing.B) {
	segmentSizes := []int64{1024 * 1024, 10 * 1024 * 1024, 100 * 1024 * 1024}

	for _, segSize := range segmentSizes {
		name := fmt.Sprintf("segmentSize=%dMB", segSize/1024/1024)
		b.Run(name, func(b *testing.B) {
			dir := b.TempDir()
			config := LogConfig{
				Dir:            dir,
				MaxSegmentSize: segSize,
			}

			log, err := NewLog(config)
			require.NoError(b, err)
			defer log.Close()

			msgSize := 1024
			value := make([]byte, msgSize)
			key := []byte("benchmark-key")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := log.Append(key, value)
				if err != nil {
					b.Fatal(err)
				}
			}

			b.StopTimer()
			numSegments := len(log.segments)
			b.ReportMetric(float64(numSegments), "segments_created")
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "msg/s")
		})
	}
}
