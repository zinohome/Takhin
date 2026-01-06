// Copyright 2025 Takhin Data, Inc.

package mempool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferPool_GetPut(t *testing.T) {
	pool := NewBufferPool()

	tests := []struct {
		name string
		size int
	}{
		{"Small buffer", 256},
		{"1KB buffer", 1024},
		{"4KB buffer", 4096},
		{"16KB buffer", 16384},
		{"64KB buffer", 65536},
		{"256KB buffer", 262144},
		{"1MB buffer", 1048576},
		{"4MB buffer", 4194304},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := pool.Get(tt.size)
			assert.NotNil(t, buf)
			assert.GreaterOrEqual(t, len(buf), tt.size)
			assert.GreaterOrEqual(t, cap(buf), tt.size)

			// Write some data
			for i := 0; i < len(buf); i++ {
				buf[i] = byte(i % 256)
			}

			// Return to pool
			pool.Put(buf)

			// Get again and verify it's zeroed
			buf2 := pool.Get(tt.size)
			assert.NotNil(t, buf2)
			for i := 0; i < len(buf2); i++ {
				assert.Equal(t, byte(0), buf2[i], "Buffer should be zeroed at index %d", i)
			}
			pool.Put(buf2)
		})
	}
}

func TestBufferPool_OversizedBuffer(t *testing.T) {
	pool := NewBufferPool()

	// Request very large buffer (larger than any pool)
	size := 32 * 1024 * 1024 // 32MB
	buf := pool.Get(size)
	assert.NotNil(t, buf)
	assert.Equal(t, size, len(buf))

	// Put should discard it
	pool.Put(buf)
}

func TestBufferPool_NilBuffer(t *testing.T) {
	pool := NewBufferPool()
	
	// Should not panic
	pool.Put(nil)
}

func TestBufferPool_DefaultPool(t *testing.T) {
	buf := GetBuffer(4096)
	assert.NotNil(t, buf)
	assert.GreaterOrEqual(t, len(buf), 4096)
	
	PutBuffer(buf)
}

func TestBufferPool_Concurrent(t *testing.T) {
	pool := NewBufferPool()
	const goroutines = 100
	const iterations = 1000

	done := make(chan bool, goroutines)
	
	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < iterations; i++ {
				size := 1024 * (1 + i%10)
				buf := pool.Get(size)
				assert.NotNil(t, buf)
				
				// Write some data
				for j := 0; j < len(buf); j++ {
					buf[j] = byte(j % 256)
				}
				
				pool.Put(buf)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for g := 0; g < goroutines; g++ {
		<-done
	}
}

func BenchmarkBufferPool_Get(b *testing.B) {
	pool := NewBufferPool()
	
	b.Run("1KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := pool.Get(1024)
			pool.Put(buf)
		}
	})
	
	b.Run("64KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := pool.Get(65536)
			pool.Put(buf)
		}
	})
	
	b.Run("1MB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := pool.Get(1048576)
			pool.Put(buf)
		}
	})
}

func BenchmarkBufferPool_GetNoPool(b *testing.B) {
	b.Run("1KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1024)
			_ = buf
		}
	})
	
	b.Run("64KB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 65536)
			_ = buf
		}
	})
	
	b.Run("1MB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1048576)
			_ = buf
		}
	})
}
