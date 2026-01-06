// Copyright 2025 Takhin Data, Inc.

package mempool

import (
	"sync"
	"sync/atomic"
)

// BufferPool manages byte slices to reduce GC pressure
type BufferPool struct {
	pools map[int]*sync.Pool
	sizes []int
	stats poolStats
}

type poolStats struct {
	allocations  atomic.Uint64
	gets         atomic.Uint64
	puts         atomic.Uint64
	inUse        atomic.Int64
	oversized    atomic.Uint64
	discarded    atomic.Uint64
}

// NewBufferPool creates a new buffer pool with predefined size buckets
func NewBufferPool() *BufferPool {
	sizes := []int{
		512,        // Small messages
		1024,       // 1KB
		4096,       // 4KB
		16384,      // 16KB
		65536,      // 64KB
		262144,     // 256KB
		1048576,    // 1MB
		4194304,    // 4MB
		16777216,   // 16MB
	}

	bp := &BufferPool{
		pools: make(map[int]*sync.Pool),
		sizes: sizes,
	}

	for _, size := range sizes {
		size := size // capture loop variable
		bp.pools[size] = &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, size)
				bp.stats.allocations.Add(1)
				return &buf
			},
		}
	}

	return bp
}

// Get returns a buffer of at least the requested size
func (bp *BufferPool) Get(size int) []byte {
	// Find the appropriate pool
	poolSize := bp.findPoolSize(size)
	if poolSize == 0 {
		// Size too large, allocate directly
		bp.stats.allocations.Add(1)
		bp.stats.oversized.Add(1)
		return make([]byte, size)
	}

	// Get from pool
	pool := bp.pools[poolSize]
	bufPtr := pool.Get().(*[]byte)
	buf := (*bufPtr)[:size] // Slice to requested size
	
	bp.stats.gets.Add(1)
	bp.stats.inUse.Add(1)
	
	return buf
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	// Find the original pool size
	capacity := cap(buf)
	poolSize := bp.findPoolSize(capacity)
	if poolSize == 0 || poolSize != capacity {
		// Not from pool or wrong size, let GC handle it
		bp.stats.discarded.Add(1)
		return
	}

	// Reset buffer
	buf = buf[:capacity]
	for i := range buf {
		buf[i] = 0
	}

	// Return to pool
	pool := bp.pools[poolSize]
	pool.Put(&buf)
	
	bp.stats.puts.Add(1)
	bp.stats.inUse.Add(-1)
}

// findPoolSize returns the smallest pool size that can accommodate the requested size
func (bp *BufferPool) findPoolSize(size int) int {
	for _, poolSize := range bp.sizes {
		if size <= poolSize {
			return poolSize
		}
	}
	return 0 // Too large
}

// Stats returns current pool statistics
func (bp *BufferPool) Stats() PoolStats {
	return PoolStats{
		Allocations: bp.stats.allocations.Load(),
		Gets:        bp.stats.gets.Load(),
		Puts:        bp.stats.puts.Load(),
		InUse:       bp.stats.inUse.Load(),
		Oversized:   bp.stats.oversized.Load(),
		Discarded:   bp.stats.discarded.Load(),
	}
}

// PoolStats contains memory pool statistics
type PoolStats struct {
	Allocations uint64
	Gets        uint64
	Puts        uint64
	InUse       int64
	Oversized   uint64
	Discarded   uint64
}

// DefaultBufferPool is the global buffer pool instance
var DefaultBufferPool = NewBufferPool()

// GetBuffer is a convenience function to get a buffer from the default pool
func GetBuffer(size int) []byte {
	return DefaultBufferPool.Get(size)
}

// PutBuffer is a convenience function to return a buffer to the default pool
func PutBuffer(buf []byte) {
	DefaultBufferPool.Put(buf)
}

// GetStats returns statistics from the default buffer pool
func GetStats() PoolStats {
	return DefaultBufferPool.Stats()
}
