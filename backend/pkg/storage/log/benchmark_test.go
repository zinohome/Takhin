// Copyright 2025 Takhin Data, Inc.

package log

import (
	"fmt"
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
