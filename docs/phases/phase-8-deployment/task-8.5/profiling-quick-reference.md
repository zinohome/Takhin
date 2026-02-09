# Task 8.5: Performance Profiling - Quick Reference

## Quick Start

### Enable Profiler Server
```bash
# Via environment variable
export TAKHIN_PROFILER_ENABLED=true
./takhin -config configs/takhin.yaml

# Server runs on http://localhost:6060/debug/pprof/
```

### Collect Profiles

```bash
# CPU profile (30 seconds)
takhin-profiler -type cpu -duration 30s -output cpu.prof

# Memory heap profile
takhin-profiler -type heap -output heap.prof

# All profiles at once
takhin-profiler -type all -duration 30s -output-dir /tmp/profiles

# From remote server
takhin-profiler -type cpu -remote localhost:6060 -duration 30s
```

### Analyze Profiles

```bash
# Interactive analysis
go tool pprof cpu.prof

# Top 10 functions
go tool pprof -top cpu.prof

# Web UI with flame graph
go tool pprof -http=:8080 cpu.prof

# Compare two profiles
go tool pprof -base old.prof new.prof
```

## Profile Types

| Type | Purpose | Duration | Use When |
|------|---------|----------|----------|
| `cpu` | CPU hotspots | 30-60s | High CPU usage |
| `heap` | Memory usage | Instant | Memory investigation |
| `allocs` | Allocation patterns | Instant | GC pressure |
| `goroutine` | Goroutine leaks | Instant | Goroutine count high |
| `block` | Blocking operations | 30s+ | Slow requests |
| `mutex` | Lock contention | 30s+ | Concurrency issues |
| `trace` | Execution trace | 5-10s | Scheduler analysis |

## pprof Endpoints

```bash
# CPU profile (30 seconds default)
curl http://localhost:6060/debug/pprof/profile?seconds=30 -o cpu.prof

# Heap snapshot
curl http://localhost:6060/debug/pprof/heap -o heap.prof

# Goroutine stacks
curl http://localhost:6060/debug/pprof/goroutine -o goroutine.prof

# All available profiles
curl http://localhost:6060/debug/pprof/
```

## Common Tasks

### Find CPU Hotspots
```bash
takhin-profiler -type cpu -duration 30s -remote localhost:6060
go tool pprof -top cpu.prof | head -20
```

### Detect Memory Leaks
```bash
# Take two heap snapshots 5 minutes apart
takhin-profiler -type heap -remote localhost:6060 -output heap1.prof
# ... wait 5 minutes ...
takhin-profiler -type heap -remote localhost:6060 -output heap2.prof

# Compare
go tool pprof -base heap1.prof heap2.prof
```

### Find Goroutine Leaks
```bash
# Take snapshots before and after
takhin-profiler -type goroutine -remote localhost:6060 -output g1.prof
# ... reproduce leak ...
takhin-profiler -type goroutine -remote localhost:6060 -output g2.prof

# Compare
go tool pprof -base g1.prof g2.prof
```

### Analyze Request Latency
```bash
# CPU profile during load test
takhin-profiler -type cpu -duration 60s -remote localhost:6060

# Find slow paths
go tool pprof -http=:8080 cpu.prof
# Look for kafka.handler, storage.log functions
```

### Check Allocation Rate
```bash
takhin-profiler -type allocs -remote localhost:6060
go tool pprof -alloc_space allocs.prof
go tool pprof -sample_index=alloc_objects allocs.prof
```

## Flame Graphs

### Generate Flame Graph
```bash
# Collect profile
takhin-profiler -type cpu -duration 30s -output cpu.prof

# Open in browser
go tool pprof -http=:8080 cpu.prof
# Navigate to: http://localhost:8080/ui/flamegraph
```

### Read Flame Graph
- **Width**: Time spent (wider = more time)
- **Height**: Call stack depth
- **Color**: Random (for differentiation)
- **Click**: Zoom into function
- **Search**: Find specific functions

## Configuration

### Default (Production Safe)
```yaml
profiler:
  enabled: false      # Disabled
  host: "127.0.0.1"  # Localhost only
  port: 6060
```

### Development
```yaml
profiler:
  enabled: true
  host: "0.0.0.0"
  port: 6060
```

### Environment Override
```bash
export TAKHIN_PROFILER_ENABLED=true
export TAKHIN_PROFILER_HOST=0.0.0.0
export TAKHIN_PROFILER_PORT=6060
```

## CLI Tool Options

```
takhin-profiler [options]

Profile Types:
  -type string        cpu, heap, allocs, goroutine, block, mutex, trace, all

Collection:
  -duration duration  Profile duration (default 30s)
  -output string      Output file path
  -output-dir string  Directory for batch collection
  -remote string      Remote server (e.g., localhost:6060)

Analysis:
  -analyze           Analyze after collection
  -flamegraph        Show flame graph command

Tuning:
  -sample-rate int   Block/mutex sample rate (default 1)
  -mem-rate int      Memory profile rate (0=default)
```

## go tool pprof Commands

```bash
# Top functions by CPU
go tool pprof -top cpu.prof

# List functions matching pattern
go tool pprof -list=kafka cpu.prof

# Show call graph
go tool pprof -web cpu.prof

# Interactive mode
go tool pprof cpu.prof
(pprof) top
(pprof) list main.handle
(pprof) web

# Remote profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

## Troubleshooting

### High CPU Usage
1. Collect CPU profile: `takhin-profiler -type cpu -duration 60s`
2. Find hotspots: `go tool pprof -top cpu.prof`
3. Analyze: `go tool pprof -http=:8080 cpu.prof`

### Memory Growing
1. Take heap snapshot: `takhin-profiler -type heap`
2. Check top allocations: `go tool pprof -top heap.prof`
3. Compare over time: `go tool pprof -base old.prof new.prof`

### Too Many Goroutines
1. Get goroutine dump: `takhin-profiler -type goroutine`
2. Analyze stacks: `go tool pprof -top goroutine.prof`
3. Find leak pattern in stack traces

### Slow Requests
1. CPU profile during load: `takhin-profiler -type cpu -duration 60s`
2. Block profile: `takhin-profiler -type block -duration 60s`
3. Check metrics: `curl http://localhost:9090/metrics | grep latency`

## Performance Impact

| Profile | Overhead | Production? | Notes |
|---------|----------|------------|-------|
| CPU | ~5% | ✅ Yes | Safe for prod |
| Heap | <1% | ✅ Yes | Minimal impact |
| Allocs | <1% | ✅ Yes | Safe |
| Goroutine | ~0% | ✅ Yes | Instant snapshot |
| Block | Variable | ⚠️ Dev Only | Can slow down |
| Mutex | Variable | ⚠️ Dev Only | Can slow down |
| Trace | 10-30% | ❌ Dev Only | High overhead |

## Best Practices

1. **Profile in Production**
   - CPU and heap are safe
   - Enable profiler temporarily
   - Use localhost-only binding

2. **Baseline First**
   - Profile normal operation
   - Compare against baseline
   - Understand normal patterns

3. **Representative Load**
   - Profile during realistic traffic
   - 30-60 seconds for CPU
   - Multiple snapshots for memory

4. **Correlate with Metrics**
   - Check Prometheus metrics
   - Look at logs
   - Reproduce conditions

5. **Security**
   - Disable in production by default
   - Use firewall rules
   - Localhost binding only
   - Consider authentication

## Integration Examples

### Kubernetes
```yaml
apiVersion: v1
kind: Service
metadata:
  name: takhin-profiler
spec:
  ports:
  - port: 6060
    name: pprof
  selector:
    app: takhin
  type: ClusterIP  # Internal only
```

### Docker Compose
```yaml
services:
  takhin:
    ports:
      - "9092:9092"
      - "127.0.0.1:6060:6060"  # Localhost only
    environment:
      - TAKHIN_PROFILER_ENABLED=true
```

### Automated Collection
```bash
#!/bin/bash
# Collect profiles during incident

DATE=$(date +%Y%m%d_%H%M%S)
DIR=/var/log/takhin/profiles/$DATE

mkdir -p $DIR
takhin-profiler -type all -duration 30s -output-dir $DIR -remote localhost:6060

echo "Profiles saved to $DIR"
```

## Resources

- **Go Profiling Blog**: https://go.dev/blog/pprof
- **pprof Documentation**: https://pkg.go.dev/net/http/pprof
- **Flame Graphs**: https://www.brendangregg.com/flamegraphs.html
- **Go Execution Tracer**: https://go.dev/doc/diagnostics#execution-tracer

## Quick Commands Cheatsheet

```bash
# Enable profiler
export TAKHIN_PROFILER_ENABLED=true

# CPU profile (most common)
takhin-profiler -type cpu -duration 30s -remote localhost:6060 -analyze

# Memory check
takhin-profiler -type heap -remote localhost:6060 -analyze

# Goroutine count
curl http://localhost:6060/debug/pprof/goroutine?debug=1

# All profiles
takhin-profiler -type all -remote localhost:6060 -output-dir ./profiles

# Interactive analysis
go tool pprof -http=:8080 cpu.prof

# Top 20 functions
go tool pprof -top20 cpu.prof
```
