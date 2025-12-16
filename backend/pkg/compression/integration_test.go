// Copyright 2025 Takhin Data, Inc.

package compression

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompressionIntegration tests end-to-end compression workflow
// simulating what would happen in Kafka protocol handling
func TestCompressionIntegration(t *testing.T) {
	// 模拟 Kafka 消息记录
	type Message struct {
		Key   []byte
		Value []byte
	}

	messages := []Message{
		{Key: []byte("user-1"), Value: []byte("Hello, World!")},
		{Key: []byte("user-2"), Value: []byte("This is a test message")},
		{Key: []byte("user-3"), Value: []byte("Kafka compression integration test")},
	}

	// 序列化消息批次（简化版）
	var batch bytes.Buffer
	for _, msg := range messages {
		batch.Write(msg.Key)
		batch.WriteByte(0) // separator
		batch.Write(msg.Value)
		batch.WriteByte(0) // separator
	}

	originalData := batch.Bytes()
	t.Logf("Original batch size: %d bytes", len(originalData))

	// 测试所有压缩类型
	types := []Type{None, GZIP, Snappy, LZ4, ZSTD}
	for _, compType := range types {
		t.Run(compType.String(), func(t *testing.T) {
			// 步骤 1: 压缩（模拟客户端）
			compressed, err := Compress(compType, originalData)
			require.NoError(t, err, "compress should succeed")

			if compType != None {
				t.Logf("Compressed size: %d bytes (%.2f%% of original)",
					len(compressed),
					float64(len(compressed))/float64(len(originalData))*100)
			}

			// 步骤 2: 存储（模拟服务端存储压缩数据）
			// 在实际系统中，这里会写入磁盘
			stored := make([]byte, len(compressed))
			copy(stored, compressed)

			// 步骤 3: 读取和解压（模拟客户端读取）
			decompressed, err := Decompress(compType, stored)
			require.NoError(t, err, "decompress should succeed")

			// 步骤 4: 验证数据完整性
			assert.Equal(t, originalData, decompressed,
				"decompressed data should match original")
		})
	}
}

// TestCompressionRoundTripLargeData tests compression with larger data sets
func TestCompressionRoundTripLargeData(t *testing.T) {
	// 生成大量重复数据（模拟日志消息）
	logEntry := []byte(`{"timestamp":"2025-12-17T10:00:00Z","level":"INFO","service":"api","message":"Request processed successfully","duration_ms":125}`)

	var batch bytes.Buffer
	for i := 0; i < 1000; i++ {
		batch.Write(logEntry)
		batch.WriteByte('\n')
	}

	originalData := batch.Bytes()
	t.Logf("Original data size: %d bytes", len(originalData))

	types := []Type{None, GZIP, Snappy, LZ4, ZSTD}
	for _, compType := range types {
		t.Run(compType.String(), func(t *testing.T) {
			// 压缩
			compressed, err := Compress(compType, originalData)
			require.NoError(t, err)

			// 计算压缩率
			ratio := float64(len(compressed)) / float64(len(originalData)) * 100
			t.Logf("Compression ratio: %.2f%% (original: %d -> compressed: %d)",
				ratio, len(originalData), len(compressed))

			// 解压
			decompressed, err := Decompress(compType, compressed)
			require.NoError(t, err)

			// 验证
			assert.Equal(t, originalData, decompressed)

			// 对于有压缩的类型，验证确实压缩了
			if compType != None {
				assert.Less(t, len(compressed), len(originalData),
					"compressed data should be smaller than original")
			}
		})
	}
}

// TestCompressionWithRandomData tests compression with non-compressible data
func TestCompressionWithRandomData(t *testing.T) {
	// 生成随机数据（不可压缩）
	randomData := make([]byte, 10000)
	for i := range randomData {
		randomData[i] = byte(i % 256)
	}

	types := []Type{None, GZIP, Snappy, LZ4, ZSTD}
	for _, compType := range types {
		t.Run(compType.String(), func(t *testing.T) {
			compressed, err := Compress(compType, randomData)
			require.NoError(t, err)

			t.Logf("Original: %d bytes, Compressed: %d bytes (%.2f%%)",
				len(randomData),
				len(compressed),
				float64(len(compressed))/float64(len(randomData))*100)

			// 对于随机数据，压缩可能不会减小大小（甚至可能增大）
			// 但仍应该能够正确往返
			decompressed, err := Decompress(compType, compressed)
			require.NoError(t, err)
			assert.Equal(t, randomData, decompressed)
		})
	}
}

// TestCompressionEmptyData tests compression with empty data
func TestCompressionEmptyData(t *testing.T) {
	emptyData := []byte{}

	types := []Type{None, GZIP, Snappy, LZ4, ZSTD}
	for _, compType := range types {
		t.Run(compType.String(), func(t *testing.T) {
			compressed, err := Compress(compType, emptyData)
			require.NoError(t, err)

			decompressed, err := Decompress(compType, compressed)
			require.NoError(t, err)

			// Snappy may return nil for empty data, which is functionally equivalent
			if len(decompressed) == 0 && len(emptyData) == 0 {
				return // both empty, test passes
			}
			assert.Equal(t, emptyData, decompressed)
		})
	}
}

// TestCompressionConcurrent tests compression from multiple goroutines
func TestCompressionConcurrent(t *testing.T) {
	data := []byte("test data for concurrent compression " + string(make([]byte, 1000)))

	const numGoroutines = 10
	const numIterations = 100

	type result struct {
		err error
	}

	results := make(chan result, numGoroutines*numIterations)

	// 启动多个 goroutine 并发压缩
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIterations; j++ {
				// 测试所有压缩类型
				for _, compType := range []Type{None, GZIP, Snappy, LZ4, ZSTD} {
					compressed, err := Compress(compType, data)
					if err != nil {
						results <- result{err: err}
						continue
					}

					decompressed, err := Decompress(compType, compressed)
					if err != nil {
						results <- result{err: err}
						continue
					}

					if !bytes.Equal(data, decompressed) {
						results <- result{err: fmt.Errorf("data mismatch")}
						continue
					}

					results <- result{err: nil}
				}
			}
		}()
	}

	// 收集结果
	for i := 0; i < numGoroutines*numIterations*5; i++ {
		res := <-results
		require.NoError(t, res.err)
	}
}
