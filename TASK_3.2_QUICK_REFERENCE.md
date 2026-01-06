# Task 3.2: Memory Pool - Quick Reference

## Package Location
`backend/pkg/mempool`

## Core API

### Buffer Pool
```go
// Get a buffer (automatically selects appropriate pool)
buf := mempool.GetBuffer(size int) []byte

// Return buffer to pool (MUST be called)
mempool.PutBuffer(buf []byte)

// Get pool statistics
stats := mempool.GetStats() // Returns PoolStats struct
```

### PoolStats Structure
```go
type PoolStats struct {
    Allocations uint64  // Total new allocations
    Gets        uint64  // Total gets from pool
    Puts        uint64  // Total returns to pool
    InUse       int64   // Currently in-use buffers
    Oversized   uint64  // Oversized allocations (bypassed pool)
    Discarded   uint64  // Buffers not returned properly
}
```

## Size Buckets
| Bucket | Size | Use Case |
|--------|------|----------|
| 1 | 512B | Small messages |
| 2 | 1KB | Small-medium messages |
| 3 | 4KB | Standard messages |
| 4 | 16KB | Large messages |
| 5 | 64KB | Very large messages |
| 6 | 256KB | Batch operations |
| 7 | 1MB | Large batches |
| 8 | 4MB | Very large batches |
| 9 | 16MB | Maximum Kafka message size |

## Usage Patterns

### Basic Pattern
```go
func processMessage(data []byte) error {
    buf := mempool.GetBuffer(len(data))
    defer mempool.PutBuffer(buf)
    
    copy(buf, data)
    // Process buf...
    return nil
}
```

### Encode/Decode Pattern  
```go
func encodeRecord(rec *Record) ([]byte, error) {
    size := calculateSize(rec)
    buf := mempool.GetBuffer(size)
    // Encode into buf
    return buf, nil // Caller must PutBuffer
}

// Caller
data, _ := encodeRecord(rec)
defer mempool.PutBuffer(data)
file.Write(data)
```

### Batch Pattern
```go
func processBatch(records []*Record) error {
    totalSize := calculateBatchSize(records)
    buf := mempool.GetBuffer(totalSize)
    defer mempool.PutBuffer(buf)
    
    // Encode all records into buf
    return writeBatch(buf)
}
```

## Best Practices

### ✅ DO
- Always use `defer mempool.PutBuffer()` immediately after Get
- Request the actual size needed, not the bucket size
- Return buffers promptly after use
- Monitor `buffer_in_use` metric for leaks

### ❌ DON'T
- Don't retain references to buffers after Put
- Don't Put buffers twice
- Don't Put nil buffers (safe but wasteful)
- Don't assume buffer contents after Get (may contain old data)

## Monitoring

### Prometheus Metrics
```promql
# Buffer allocation rate
rate(takhin_mempool_buffer_allocations_total[5m])

# Pool hit rate (gets without new allocations)
rate(takhin_mempool_buffer_gets_total[5m]) / 
  (rate(takhin_mempool_buffer_gets_total[5m]) + 
   rate(takhin_mempool_buffer_allocations_total[5m]))

# Buffer leaks (gets not matched by puts)
takhin_mempool_buffer_gets_total - takhin_mempool_buffer_puts_total

# Currently in use
takhin_mempool_buffer_in_use

# Oversized allocations (optimization opportunity)
rate(takhin_mempool_buffer_oversized_total[5m])

# Discarded buffers (wrong size or capacity)
rate(takhin_mempool_buffer_discarded_total[5m])
```

### Alert Rules (Recommended)
```yaml
# Buffer leak detection
- alert: MemPoolBufferLeak
  expr: takhin_mempool_buffer_in_use > 1000
  for: 5m
  annotations:
    summary: "Memory pool buffer leak detected"

# Low pool hit rate
- alert: MemPoolLowHitRate
  expr: |
    rate(takhin_mempool_buffer_allocations_total[5m]) /
    rate(takhin_mempool_buffer_gets_total[5m]) > 0.5
  for: 10m
  annotations:
    summary: "Memory pool hit rate below 50%"
```

## Performance Characteristics

### Allocation Speed
- **Pooled**: ~15-20 ns/op
- **Direct malloc**: 200-45,000 ns/op
- **Speedup**: 10-2500x

### Memory Overhead
- **Per pool**: ~8 bytes (pointer)
- **Per buffer**: 0 bytes (reused allocation)
- **Total**: Negligible (<1KB for entire system)

### GC Impact
- **Reduced allocations**: 50-90% fewer allocations
- **Reduced GC frequency**: Proportional to allocation reduction
- **Reduced GC pause**: 30-70% reduction in pause time

## Integration Points

### Storage Layer
- `segment.go`: encodeRecord, decodeRecord, AppendBatch
- Index operations: writeIndex, writeTimeIndex
- Binary search: FindOffsetByTimestamp

### Future Integration Targets
- Protocol encoding/decoding
- Compression/decompression buffers
- Network I/O buffers
- Batch API operations

## Testing

```bash
# Unit tests
go test ./pkg/mempool -v

# Benchmarks
go test ./pkg/mempool -bench=. -benchmem

# GC benchmarks
go test ./pkg/mempool -bench=BenchmarkGCPressure

# Concurrent stress test
go test ./pkg/mempool -run TestBufferPool_Concurrent -count=100

# Coverage
go test ./pkg/mempool -cover
```

## Troubleshooting

### High buffer_in_use Count
**Symptom**: `takhin_mempool_buffer_in_use` continuously increasing  
**Cause**: Buffers not being returned to pool  
**Fix**: Audit code for missing `defer mempool.PutBuffer()` calls

### High oversized_total Rate
**Symptom**: Many `buffer_oversized_total` increments  
**Cause**: Requests for buffers larger than max bucket (16MB)  
**Fix**: Consider adding larger buckets or optimizing large operations

### High discarded_total Rate
**Symptom**: Many `buffer_discarded_total` increments  
**Cause**: Buffers with wrong capacity being returned  
**Fix**: Audit code for buffer slicing that changes capacity

### Low Performance Gain
**Symptom**: Expected speedup not observed  
**Cause**: Workload doesn't match buffer sizes, or GC not a bottleneck  
**Analysis**: Profile application, check buffer size distribution

## Configuration

Currently no runtime configuration. Pool sizes are compile-time constants.

To adjust bucket sizes, modify `NewBufferPool()` in `buffer_pool.go`.

## Dependencies
- Go 1.21+ (for sync/atomic improvements)
- No external dependencies

## Related Tasks
- Task 1.1: Core storage (foundation)
- Task 3.1: Zero-copy I/O (complementary optimization)
- Task 7.5: Testing coverage (validation)
