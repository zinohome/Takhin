# Storage Layer Performance Benchmarks

## Overview

This document describes the comprehensive storage layer performance benchmark suite for Takhin. The benchmarks cover produce, fetch, and compaction operations at both the log and topic manager levels, identifying performance bottlenecks and optimization opportunities.

## Benchmark Coverage

### 1. Produce Performance

#### Single Message Writes (`BenchmarkWriteThroughput`)
- **Purpose**: Measure single message write throughput
- **Parameters**:
  - Message counts: 100, 1,000, 10,000
  - Message sizes: 100B, 1KB, 10KB
- **Metrics**: MB/s, allocations
- **Bottleneck Detection**: Lock contention, disk I/O

#### Batch Writes (`BenchmarkBatchWriteThroughput`)
- **Purpose**: Measure batch write efficiency
- **Parameters**:
  - Batch sizes: 10, 100, 1,000
  - Message sizes: 100B, 1KB, 10KB
- **Metrics**: MB/s, allocations
- **Expected Improvement**: 2-5x over single writes

#### Produce Latency (`BenchmarkProduceLatency`)
- **Purpose**: Measure per-operation latency
- **Parameters**: Message sizes: 100B, 1KB, 10KB
- **Metrics**: ms/op, ops/s
- **Target**: < 1ms for 1KB messages

### 2. Fetch Performance

#### Sequential Fetch (`BenchmarkSequentialFetch`)
- **Purpose**: Simulate Kafka consumer pattern
- **Parameters**:
  - Message counts: 100, 1,000, 10,000
  - Message sizes: 100B, 1KB, 10KB
  - Fetch sizes: 1, 10, 100 messages
- **Metrics**: MB/s, msg/s, allocations
- **Bottleneck Detection**: Sequential I/O, buffering

#### Random Fetch (`BenchmarkRandomFetch`)
- **Purpose**: Test random access performance
- **Parameters**:
  - Message counts: 1,000, 10,000, 100,000
  - Message sizes: 100B, 1KB, 10KB
- **Metrics**: MB/s, msg/s
- **Bottleneck Detection**: Disk seeks, cache misses

#### Fetch Latency (`BenchmarkFetchLatency`)
- **Purpose**: Measure per-read latency
- **Parameters**: Message sizes: 100B, 1KB, 10KB
- **Metrics**: ms/op, ops/s
- **Target**: < 0.5ms for cached reads

### 3. Compaction Performance

#### Log Compaction (`BenchmarkCompaction`)
- **Purpose**: Measure compaction efficiency
- **Parameters**:
  - Segment counts: 10, 50, 100
  - Message sizes: 100B, 1KB
  - Deduplication ratios: 30%, 50%, 70%
- **Metrics**: MB reclaimed, keys removed, duration (ms)
- **Bottleneck Detection**: Key map overhead, disk I/O

### 4. Concurrent Access

#### Concurrent Producers (`BenchmarkConcurrentProducers`)
- **Purpose**: Test multi-producer scalability
- **Parameters**:
  - Producer counts: 1, 2, 4, 8
  - Message sizes: 100B, 1KB
- **Metrics**: MB/s, msg/s
- **Bottleneck Detection**: Lock contention, CPU saturation

#### Concurrent Consumers (`BenchmarkConcurrentConsumers`)
- **Purpose**: Test multi-consumer scalability
- **Parameters**:
  - Consumer counts: 1, 2, 4, 8
  - Message sizes: 100B, 1KB
- **Metrics**: MB/s, msg/s
- **Expected**: Near-linear scaling with read locks

### 5. Topic Manager Performance

#### Multi-Partition (`BenchmarkTopicManagerProduceThroughput`, `BenchmarkTopicManagerFetchThroughput`)
- **Purpose**: Test partition-level parallelism
- **Parameters**:
  - Partition counts: 1, 4, 16
  - Message sizes: 100B, 1KB, 10KB
- **Metrics**: MB/s, msg/s
- **Expected**: Linear scaling up to CPU count

#### Partition Balance (`BenchmarkTopicManagerPartitionBalance`)
- **Purpose**: Verify even load distribution
- **Parameters**: Partition counts: 4, 8, 16, 32
- **Metrics**: Imbalance %, msg/s
- **Target**: < 5% imbalance

#### Multi-Topic (`BenchmarkTopicManagerMultiTopic`)
- **Purpose**: Test resource sharing across topics
- **Parameters**:
  - Topic counts: 1, 5, 10
  - Partitions per topic: 4
- **Metrics**: MB/s, msg/s
- **Bottleneck Detection**: File descriptor limits

## Running Benchmarks

### Full Suite (30-60 minutes)
```bash
# Using Task
task backend:bench

# Or directly
./scripts/run_benchmarks.sh
```

### Quick Test (2-3 minutes)
```bash
task backend:bench:quick
```

### Individual Benchmarks
```bash
cd backend

# Produce benchmarks
go test -bench=BenchmarkWriteThroughput -benchtime=5s ./pkg/storage/log

# Fetch benchmarks
go test -bench=BenchmarkSequentialFetch -benchtime=5s ./pkg/storage/log

# Compaction benchmarks
go test -bench=BenchmarkCompaction -benchtime=3s ./pkg/storage/log

# Topic manager benchmarks
go test -bench=BenchmarkTopicManager -benchtime=5s ./pkg/storage/topic
```

### With CPU/Memory Profiling
```bash
cd backend

# CPU profile
go test -bench=. -cpuprofile=cpu.prof -benchtime=10s ./pkg/storage/log
go tool pprof -http=:8080 cpu.prof

# Memory profile
go test -bench=. -memprofile=mem.prof -benchtime=10s ./pkg/storage/log
go tool pprof -http=:8080 mem.prof

# Trace
go test -bench=BenchmarkWriteThroughput -trace=trace.out ./pkg/storage/log
go tool trace trace.out
```

## Interpreting Results

### Throughput Metrics

```
BenchmarkWriteThroughput/messages=1000/size=1024B-8    1000    1234567 ns/op    812.50 MB/s    2345 allocs/op
```

- **1000**: Number of iterations
- **1234567 ns/op**: Average time per iteration (nanoseconds)
- **812.50 MB/s**: Custom throughput metric
- **2345 allocs/op**: Allocations per iteration

**Analysis**:
- Compare MB/s across message sizes (should scale linearly)
- Check allocs/op (lower is better, < 10 for writes)
- Compare with Kafka benchmarks (target: 600-800 MB/s single partition)

### Latency Metrics

```
BenchmarkProduceLatency/size=1024B-8    100000    1.234 ms/op    809876 ops/s
```

**Analysis**:
- Target latencies (1KB messages):
  - Produce: < 1ms
  - Fetch: < 0.5ms (cached)
- p99 latency critical for tail latency

### Identifying Bottlenecks

#### 1. Lock Contention
**Symptoms**:
- Poor scaling with concurrent producers/consumers
- High CPU usage but low throughput
- CPU profile shows time in `sync.Mutex.Lock`

**Solutions**:
- Reduce critical section size
- Use read/write locks appropriately
- Consider lock-free structures for hot paths

#### 2. Disk I/O
**Symptoms**:
- Low MB/s compared to disk bandwidth
- High write latency variance
- iostat shows low disk utilization

**Solutions**:
- Increase buffer sizes
- Batch writes more aggressively
- Use `O_DIRECT` for large sequential writes
- Consider mmap for reads

#### 3. Memory Allocations
**Symptoms**:
- High allocs/op count
- Memory profile shows allocation hotspots
- GC pressure in trace output

**Solutions**:
- Reuse buffers with sync.Pool
- Preallocate slices with capacity
- Avoid unnecessary copying

#### 4. CPU Bound
**Symptoms**:
- CPU at 100% utilization
- CPU profile shows hotspots in encoding/compression
- Does not scale with more cores

**Solutions**:
- Optimize hot loops
- Use SIMD where applicable
- Reduce data copying

## Performance Targets

### Baseline Targets (Single Partition)

| Operation | Target | Notes |
|-----------|--------|-------|
| Produce Throughput | 500-800 MB/s | 1KB messages, sync writes |
| Fetch Throughput | 1000+ MB/s | Sequential reads |
| Produce Latency (p99) | < 2ms | 1KB messages |
| Fetch Latency (p99) | < 1ms | Cached reads |
| Compaction Duration | < 100ms | Per 100MB segment |

### Scalability Targets

| Scenario | Target | Notes |
|----------|--------|-------|
| Concurrent Producers (4) | 0.7x linear | Expected lock overhead |
| Concurrent Consumers (4) | 0.9x linear | Read-only, less contention |
| Partitions (16) | 0.95x linear | Up to CPU count |

## Continuous Monitoring

### CI/CD Integration

Add to `.github/workflows/benchmark.yml`:
```yaml
name: Performance Benchmarks

on:
  pull_request:
    paths:
      - 'backend/pkg/storage/**'
  schedule:
    - cron: '0 2 * * 0'  # Weekly on Sunday

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run benchmarks
        run: |
          cd backend
          go test -bench=. -benchtime=5s -benchmem ./pkg/storage/... > bench.txt
      
      - name: Compare with baseline
        run: |
          # Compare with main branch results
          # Alert if > 10% regression
```

### Alerting Thresholds

- **Regression**: > 10% decrease in MB/s or ops/s
- **Memory leak**: > 20% increase in allocs/op
- **Latency spike**: > 50% increase in p99

## Next Steps

1. **Run initial baseline** with current implementation
2. **Identify top 3 bottlenecks** from results
3. **Create optimization tickets** with priority
4. **Implement fixes** incrementally
5. **Re-benchmark** to validate improvements
6. **Update targets** based on achievements

## References

- [Go Benchmark Guidelines](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Kafka Performance Testing](https://kafka.apache.org/documentation/#performance)
- [pprof User Guide](https://github.com/google/pprof/blob/master/doc/README.md)
- [Takhin Architecture Docs](../architecture/)

---
**Last Updated**: 2025-01-02  
**Maintainer**: Performance Team
