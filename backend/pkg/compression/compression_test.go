// Copyright 2025 Takhin Data, Inc.

package compression

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoneCompression(t *testing.T) {
	data := []byte("hello world")

	compressed, err := Compress(None, data)
	require.NoError(t, err)
	assert.Equal(t, data, compressed)

	decompressed, err := Decompress(None, compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestGZIPCompression(t *testing.T) {
	data := []byte(strings.Repeat("test data ", 100))

	compressed, err := Compress(GZIP, data)
	require.NoError(t, err)
	assert.Less(t, len(compressed), len(data))

	decompressed, err := Decompress(GZIP, compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestSnappyCompression(t *testing.T) {
	data := []byte(strings.Repeat("test data ", 100))

	compressed, err := Compress(Snappy, data)
	require.NoError(t, err)
	assert.Less(t, len(compressed), len(data))

	decompressed, err := Decompress(Snappy, compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestLZ4Compression(t *testing.T) {
	data := []byte(strings.Repeat("test data ", 100))

	compressed, err := Compress(LZ4, data)
	require.NoError(t, err)
	assert.Less(t, len(compressed), len(data))

	decompressed, err := Decompress(LZ4, compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestZSTDCompression(t *testing.T) {
	data := []byte(strings.Repeat("test data ", 100))

	compressed, err := Compress(ZSTD, data)
	require.NoError(t, err)
	assert.Less(t, len(compressed), len(data))

	decompressed, err := Decompress(ZSTD, compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestInvalidType(t *testing.T) {
	data := []byte("test")

	_, err := Compress(Type(99), data)
	assert.Error(t, err)

	_, err = Decompress(Type(99), data)
	assert.Error(t, err)
}

func TestAllCompressionTypes(t *testing.T) {
	testData := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 50))

	types := []Type{None, GZIP, Snappy, LZ4, ZSTD}

	for _, compType := range types {
		t.Run(compType.String(), func(t *testing.T) {
			compressed, err := Compress(compType, testData)
			require.NoError(t, err)

			decompressed, err := Decompress(compType, compressed)
			require.NoError(t, err)

			assert.Equal(t, testData, decompressed)

			if compType != None {
				ratio := float64(len(compressed)) / float64(len(testData))
				t.Logf("Type: %s, Original: %d, Compressed: %d, Ratio: %.2f%%",
					compType.String(), len(testData), len(compressed), ratio*100)
			}
		})
	}
}

func (t Type) String() string {
	switch t {
	case None:
		return "None"
	case GZIP:
		return "GZIP"
	case Snappy:
		return "Snappy"
	case LZ4:
		return "LZ4"
	case ZSTD:
		return "ZSTD"
	default:
		return "Unknown"
	}
}

func BenchmarkCompression(b *testing.B) {
	data := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100))

	types := []Type{GZIP, Snappy, LZ4, ZSTD}

	for _, compType := range types {
		b.Run("Compress_"+compType.String(), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = Compress(compType, data)
			}
		})

		compressed, _ := Compress(compType, data)
		b.Run("Decompress_"+compType.String(), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = Decompress(compType, compressed)
			}
		})
	}
}
