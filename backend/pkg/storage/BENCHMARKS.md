# Storage Layer Performance Benchmarks

## Quick Start

```bash
# Run full benchmark suite (30-60 minutes)
task backend:bench

# Run quick benchmarks (2-3 minutes)
task backend:bench:quick

# Run specific benchmark category
cd backend
go test -bench=BenchmarkProduce -benchtime=5s ./pkg/storage/log
go test -bench=BenchmarkFetch -benchtime=5s ./pkg/storage/log
go test -bench=BenchmarkCompaction -benchtime=3s ./pkg/storage/log
go test -bench=BenchmarkTopicManager -benchtime=5s ./pkg/storage/topic
```

## Benchmark Suite Overview

### Coverage

| Category | Tests | Metrics | Purpose |
|----------|-------|---------|---------|
| **Produce** | 4 | Throughput, Latency | Write performance |
| **Fetch** | 4 | Throughput, Latency | Read performance |
| **Compaction** | 1 | Space, Duration | Dedup efficiency |
| **Concurrent** | 4 | Throughput, Scaling | Multi-threaded |
| **Topic Mgr** | 6 | Throughput, Balance | Partition/Topic level |

### Test Matrix

**Produce Tests:**
- Single writes: 100/1K/10K messages × 100B/1KB/10KB sizes
- Batch writes: 10/100/1000 batch sizes × 100B/1KB/10KB sizes
- Produce latency: 100B/1KB/10KB message sizes
- Concurrent producers: 1/2/4/8 producers

**Fetch Tests:**
- Sequential: 100/1K/10K messages × 1/10/100 fetch sizes
- Random: 1K/10K/100K messages × 100B/1KB/10KB sizes
- Fetch latency: 100B/1KB/10KB message sizes
- Concurrent consumers: 1/2/4/8 consumers

**Compaction Tests:**
- Segments: 10/50/100
- Dedup ratios: 30%/50%/70%
- Message sizes: 100B/1KB

**Topic Manager Tests:**
- Partitions: 1/4/16
- Topics: 1/5/10
- Load balancing validation

## Results Location

All benchmark results are saved to `benchmark_results/`:
- `benchmark_YYYYMMDD_HHMMSS.txt` - Full detailed results
- `summary_YYYYMMDD_HHMMSS.md` - Executive summary report

## Performance Targets

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Produce Throughput (1KB) | 500-800 MB/s | TBD | ⏳ |
| Fetch Throughput (seq) | 1000+ MB/s | TBD | ⏳ |
| Produce Latency (p99) | < 2ms | TBD | ⏳ |
| Fetch Latency (p99) | < 1ms | TBD | ⏳ |
| Concurrent Scaling (4x) | 0.7x | TBD | ⏳ |

*Update this table after running initial baseline benchmarks*

## Analyzing Results

### Understanding Output

```
BenchmarkWriteThroughput/messages=1000/size=1024B-8
    1000    1234567 ns/op    812.50 MB/s    2345 allocs/op    123456 B/op
```

- `1000`: Iterations run
- `1234567 ns/op`: Average time per iteration (nanoseconds)
- `812.50 MB/s`: Custom throughput metric
- `2345 allocs/op`: Memory allocations
- `123456 B/op`: Bytes allocated

### Performance Profiling

```bash
cd backend

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof -benchtime=10s ./pkg/storage/log
go tool pprof -http=:8080 cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof -benchtime=10s ./pkg/storage/log
go tool pprof -http=:8080 mem.prof

# Execution trace
go test -bench=BenchmarkWriteThroughput -trace=trace.out ./pkg/storage/log
go tool trace trace.out
```

### Comparing Runs

```bash
# Save baseline
go test -bench=. -benchmem ./pkg/storage/log > baseline.txt

# After optimization
go test -bench=. -benchmem ./pkg/storage/log > optimized.txt

# Compare (requires benchstat: go install golang.org/x/perf/cmd/benchstat@latest)
benchstat baseline.txt optimized.txt
```

## Common Bottlenecks

### 1. Lock Contention
**Symptoms**: Poor concurrent scaling, CPU high but throughput low
**Check**: CPU profile shows time in `sync.Mutex.Lock`
**Solutions**:
- Reduce critical section size
- Use RWMutex for read-heavy workloads
- Consider sharding

### 2. Disk I/O
**Symptoms**: Low MB/s compared to disk specs
**Check**: iostat shows low utilization
**Solutions**:
- Increase buffer sizes
- Batch writes
- Use direct I/O for sequential

### 3. Memory Allocations
**Symptoms**: High allocs/op, GC pressure
**Check**: Memory profile, trace shows GC pauses
**Solutions**:
- Use sync.Pool for buffers
- Preallocate slices
- Reduce copying

### 4. CPU Bound
**Symptoms**: CPU at 100%, doesn't scale
**Check**: CPU profile shows encoding/compression hotspots
**Solutions**:
- Optimize hot loops
- Reduce data copying
- Parallelize where possible

## Continuous Integration

The benchmark suite should be run:
- **Weekly**: Full suite on main branch
- **On PR**: Quick suite for storage changes
- **On Release**: Full suite + profiling

Alert if regression > 10% in key metrics.

## Next Steps

1. **Run Baseline**: `task backend:bench`
2. **Review Results**: Check `benchmark_results/summary_*.md`
3. **Identify Bottlenecks**: Top 3 performance issues
4. **Profile**: Use pprof on specific bottlenecks
5. **Optimize**: Implement targeted fixes
6. **Validate**: Re-run benchmarks to confirm improvements

## References

- [Full Documentation](../../docs/performance/benchmarks.md)
- [Go Benchmarking Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Kafka Performance](https://kafka.apache.org/documentation/#performance)

---
**Maintainer**: Performance Team  
**Last Updated**: 2025-01-02
