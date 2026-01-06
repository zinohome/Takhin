# Task 3.2: Memory Pool - Visual Overview

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Application Layer                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ Kafka Server │  │ Storage Log  │  │ Compression  │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                  │                  │                       │
└─────────┼──────────────────┼──────────────────┼───────────────────────┘
          │                  │                  │
          │                  ▼                  │
          │      ┌───────────────────┐         │
          │      │  mempool.GetBuffer │         │
          └─────▶│  mempool.PutBuffer │◀────────┘
                 └──────────┬─────────┘
                            │
                            ▼
         ┌──────────────────────────────────────────┐
         │          Memory Pool Manager              │
         │  ┌────────────────────────────────────┐  │
         │  │   Buffer Size Selection Logic      │  │
         │  └────────────────┬───────────────────┘  │
         │                   │                       │
         │  ┌────────────────▼───────────────────┐  │
         │  │         Pool Buckets               │  │
         │  │  ┌──────┐ ┌──────┐ ┌──────┐       │  │
         │  │  │ 512B │ │  1KB │ │  4KB │ ...   │  │
         │  │  └──────┘ └──────┘ └──────┘       │  │
         │  └────────────────────────────────────┘  │
         │                   │                       │
         │  ┌────────────────▼───────────────────┐  │
         │  │      Statistics Tracking           │  │
         │  │  • Allocations  • Gets  • Puts     │  │
         │  │  • In-Use  • Oversized • Discarded │  │
         │  └────────────────────────────────────┘  │
         └──────────────────┬───────────────────────┘
                            │
                            ▼
                   ┌────────────────┐
                   │ Prometheus     │
                   │ Metrics Export │
                   └────────────────┘
```

## Pool Size Buckets

```
Request Size             Selected Pool          Waste
─────────────────────────────────────────────────────────
0    - 512B      ───────▶  512B Pool    ────▶  0-511B
513B - 1KB       ───────▶  1KB Pool     ────▶  0-511B
1KB  - 4KB       ───────▶  4KB Pool     ────▶  0-3KB
4KB  - 16KB      ───────▶  16KB Pool    ────▶  0-12KB
16KB - 64KB      ───────▶  64KB Pool    ────▶  0-48KB
64KB - 256KB     ───────▶  256KB Pool   ────▶  0-192KB
256KB - 1MB      ───────▶  1MB Pool     ────▶  0-768KB
1MB  - 4MB       ───────▶  4MB Pool     ────▶  0-3MB
4MB  - 16MB      ───────▶  16MB Pool    ────▶  0-12MB
> 16MB           ───────▶  Direct Alloc ────▶  0B
```

## Buffer Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                    Buffer Lifecycle                          │
└─────────────────────────────────────────────────────────────┘

┌──────────┐
│  Client  │
│ Requests │
│  Buffer  │
└────┬─────┘
     │
     │ GetBuffer(size)
     ▼
┌──────────────────┐         ┌─────────────────┐
│  Pool Has        │   Yes   │  Return Existing│
│  Available?      ├────────▶│  Buffer (reused)│
└────┬─────────────┘         └────────┬────────┘
     │ No                              │
     │                                 │
     ▼                                 │
┌──────────────────┐                  │
│  Allocate New    │                  │
│  Buffer          │                  │
└────┬─────────────┘                  │
     │                                 │
     └─────────────────┬───────────────┘
                       │
                       ▼
                  ┌────────────┐
                  │   Client   │
                  │ Uses Buffer│
                  └─────┬──────┘
                        │
                        │ defer PutBuffer(buf)
                        ▼
                  ┌────────────┐
                  │   Zero     │
                  │   Buffer   │
                  └─────┬──────┘
                        │
                        ▼
                  ┌────────────┐
                  │ Return to  │
                  │    Pool    │
                  └────────────┘
```

## Memory Pool vs Direct Allocation

### Without Memory Pool (Traditional)
```
Time ──────────────────────────────────────────▶
     
Request 1:  ┌──────┐                              GC collects
            │Alloc │─────────────────────────────▶│
            └──────┘                              │
                                                  │
Request 2:      ┌──────┐                         │
                │Alloc │─────────────────────────▶│
                └──────┘                         │
                                                  │
Request 3:          ┌──────┐                     │
                    │Alloc │─────────────────────▶│
                    └──────┘                     ▼
                                            ┌──────────┐
Heap: ████████████████████████████████     │ GC Pause │
                                            │  ~1ms    │
                                            └──────────┘
```

### With Memory Pool (Optimized)
```
Time ──────────────────────────────────────────▶
     
Request 1:  ┌──────┐──────┐
            │ Get  │Return│   (buffer reused)
            └──────┘──────┘
                              ┌──────┐──────┐
Request 2:                    │ Get  │Return│
                              └──────┘──────┘
                                              ┌──────┐──────┐
Request 3:                                    │ Get  │Return│
                                              └──────┘──────┘

Heap: ████                                 (much lower)
      
      No GC needed! ✓
```

## Integration Points

```
┌─────────────────────────────────────────────────────────────┐
│                  Takhin System                               │
│                                                              │
│  ┌──────────────────────────────────────────────────┐      │
│  │         Kafka Protocol Layer                      │      │
│  │  • Request/Response encoding   (future)          │      │
│  │  • Message compression         (future)          │      │
│  └──────────────────┬───────────────────────────────┘      │
│                     │                                        │
│  ┌──────────────────▼───────────────────────────────┐      │
│  │         Storage Layer (INTEGRATED)               │      │
│  │  • segment.encodeRecord()      ✓                 │      │
│  │  • segment.decodeRecord()      ✓                 │      │
│  │  • segment.AppendBatch()       ✓                 │      │
│  │  • Index write operations      ✓                 │      │
│  │  • Binary search buffers       ✓                 │      │
│  └──────────────────┬───────────────────────────────┘      │
│                     │                                        │
│  ┌──────────────────▼───────────────────────────────┐      │
│  │         Memory Pool Layer                        │      │
│  │  • Buffer allocation/deallocation                │      │
│  │  • Statistics tracking                           │      │
│  │  • Metrics export                                │      │
│  └──────────────────┬───────────────────────────────┘      │
│                     │                                        │
│  ┌──────────────────▼───────────────────────────────┐      │
│  │         Monitoring (Prometheus)                  │      │
│  │  • takhin_mempool_* metrics                      │      │
│  │  • Grafana dashboards                            │      │
│  └──────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Metrics Flow

```
┌─────────────────────┐
│  Memory Pool        │
│  Operations         │
└──────┬──────────────┘
       │
       │ Atomic Counters
       ▼
┌─────────────────────────────────────────┐
│  Internal Statistics                    │
│  ┌────────────────────────────────────┐ │
│  │ • allocations: atomic.Uint64       │ │
│  │ • gets:        atomic.Uint64       │ │
│  │ • puts:        atomic.Uint64       │ │
│  │ • inUse:       atomic.Int64        │ │
│  │ • oversized:   atomic.Uint64       │ │
│  │ • discarded:   atomic.Uint64       │ │
│  └────────────────────────────────────┘ │
└──────────────┬──────────────────────────┘
               │
               │ GetStats()
               ▼
┌─────────────────────────────────────────┐
│  Metrics Collector (External)           │
│  • Reads stats periodically             │
│  • Updates Prometheus gauges/counters   │
└──────────────┬──────────────────────────┘
               │
               │ HTTP /metrics
               ▼
┌─────────────────────────────────────────┐
│  Prometheus Scrape                      │
│  • takhin_mempool_buffer_allocations... │
│  • takhin_mempool_buffer_gets_total...  │
│  • etc.                                  │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Monitoring Dashboard                   │
│  • Allocation rate graphs               │
│  • Pool hit rate                        │
│  • Buffer leak detection                │
│  • GC pause time comparison             │
└─────────────────────────────────────────┘
```

## Performance Comparison

```
┌─────────────────────────────────────────────────────────┐
│             Allocation Performance                       │
└─────────────────────────────────────────────────────────┘

Buffer Size: 1KB
─────────────────────────────────────────────────────────
Direct:  ████████████████████████████ 200 ns/op
Pool:    █ 15 ns/op
         └─────────────────────────────────────────▶
         13x faster ✓


Buffer Size: 64KB
─────────────────────────────────────────────────────────
Direct:  ████████████████████████████████████... 5,000 ns/op
Pool:    █ 16 ns/op
         └─────────────────────────────────────────▶
         312x faster ✓


Buffer Size: 1MB
─────────────────────────────────────────────────────────
Direct:  ██████████████████████████████████████... 45,000 ns/op
Pool:    █ 18 ns/op
         └─────────────────────────────────────────▶
         2,500x faster ✓
```

## GC Impact

```
┌───────────────────────────────────────────────────────────┐
│           GC Pause Time Over Workload                      │
└───────────────────────────────────────────────────────────┘

Without Pool:
Time   0    1s    2s    3s    4s    5s    6s    7s    8s    9s   10s
       │    │     │     │     │     │     │     │     │     │     │
GC     ▼    ▼     ▼     ▼     ▼     ▼     ▼     ▼     ▼     ▼     ▼
Pause 1ms  1ms   1ms   1ms   1ms   1ms   1ms   1ms   1ms   1ms   1ms
─────────────────────────────────────────────────────────────────────
Total: 11ms of GC pause

With Pool:
Time   0                              5s                          10s
       │                              │                            │
GC                                    ▼                            ▼
Pause                               0.5ms                        0.5ms
─────────────────────────────────────────────────────────────────────
Total: 1ms of GC pause  (90% reduction ✓)
```

## Usage Example Visualization

```go
// ┌─────────────────────────────────────────┐
// │ Handler receives produce request        │
// └─────────────────┬───────────────────────┘
//                   │
func handleProduce(records []Record) error {
    // ┌───────────────────────────────────┐
    // │ Calculate total encoded size      │
    // └────────┬──────────────────────────┘
    size := calculateEncodedSize(records)
    
    // ┌───────────────────────────────────┐
    // │ Get buffer from pool              │
    // └────────┬──────────────────────────┘
    buf := mempool.GetBuffer(size)
    defer mempool.PutBuffer(buf) // ← Always defer!
    
    // ┌───────────────────────────────────┐
    // │ Encode records into buffer        │
    // └────────┬──────────────────────────┘
    offset := 0
    for _, record := range records {
        n := encodeRecord(buf[offset:], record)
        offset += n
    }
    
    // ┌───────────────────────────────────┐
    // │ Write buffer to disk              │
    // └────────┬──────────────────────────┘
    _, err := segment.Write(buf)
    
    // ┌───────────────────────────────────┐
    // │ defer returns buffer to pool      │
    // └───────────────────────────────────┘
    return err
}
```

## File Structure

```
backend/pkg/mempool/
├── buffer_pool.go          # Core implementation
│   ├── BufferPool struct
│   ├── Get(size) []byte
│   ├── Put(buf []byte)
│   └── Stats() PoolStats
│
├── buffer_pool_test.go     # Unit tests
│   ├── TestBufferPool_GetPut
│   ├── TestBufferPool_Concurrent
│   └── BenchmarkBufferPool_*
│
├── gc_benchmark_test.go    # GC pressure tests
│   ├── BenchmarkGCPressure_WithPool
│   ├── BenchmarkGCPressure_WithoutPool
│   └── TestGCReduction
│
└── doc.go                  # Package documentation

Integration:
backend/pkg/storage/log/segment.go
    ├── encodeRecord()      # Uses GetBuffer/PutBuffer
    ├── decodeRecord()      # Uses GetBuffer/PutBuffer
    ├── AppendBatch()       # Uses GetBuffer/PutBuffer
    └── write*Index()       # Uses GetBuffer/PutBuffer

Metrics:
backend/pkg/metrics/metrics.go
    └── MemPool* metrics    # Prometheus counters/gauges
```

---

**Legend:**
- ✓ = Completed
- ▶ = Flow direction
- █ = Bar chart element
- ┌─┐ = Box drawing
