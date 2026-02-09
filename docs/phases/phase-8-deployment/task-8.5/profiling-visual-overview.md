# Task 8.5: Performance Profiling Tools - Visual Overview

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Takhin Server                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Kafka       │  │  Health      │  │  Metrics     │              │
│  │  Server      │  │  Check       │  │  Server      │              │
│  │  :9092       │  │  :9091       │  │  :9090       │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                       │
│  ┌──────────────────────────────────────────────────────┐           │
│  │           Profiler Server (NEW)                       │           │
│  │           :6060/debug/pprof/                          │           │
│  │  ┌─────────────────────────────────────────────┐     │           │
│  │  │  /debug/pprof/profile    (CPU)              │     │           │
│  │  │  /debug/pprof/heap       (Memory)           │     │           │
│  │  │  /debug/pprof/goroutine  (Goroutines)       │     │           │
│  │  │  /debug/pprof/allocs     (Allocations)      │     │           │
│  │  │  /debug/pprof/block      (Blocking)         │     │           │
│  │  │  /debug/pprof/mutex      (Mutex)            │     │           │
│  │  │  /debug/pprof/trace      (Execution Trace)  │     │           │
│  │  └─────────────────────────────────────────────┘     │           │
│  └──────────────────────────────────────────────────────┘           │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
                              │ HTTP
                              │
        ┌─────────────────────┴──────────────────────────┐
        │                                                  │
   ┌────▼─────────┐                            ┌─────────▼──────────┐
   │  takhin-     │                            │   go tool pprof    │
   │  profiler    │                            │   (Built-in)       │
   │  CLI Tool    │                            └────────────────────┘
   │              │                                     │
   │  - Collect   │                            ┌────────▼────────────┐
   │  - Analyze   │                            │  Flame Graphs       │
   │  - Remote    │                            │  Interactive UI     │
   └──────────────┘                            └─────────────────────┘
```

## Profiling Workflow

```
┌────────────────────┐
│  Performance       │
│  Issue Detected    │
└────────┬───────────┘
         │
         ▼
┌────────────────────┐      ┌──────────────────────────────┐
│  Enable Profiler   │──────▶  TAKHIN_PROFILER_ENABLED=true│
│  (if disabled)     │      └──────────────────────────────┘
└────────┬───────────┘
         │
         ▼
┌────────────────────┐      ┌──────────────────────────────┐
│  Collect Profile   │──────▶  takhin-profiler -type cpu   │
│                    │      │  -remote localhost:6060      │
└────────┬───────────┘      └──────────────────────────────┘
         │
         ▼
┌────────────────────┐      ┌──────────────────────────────┐
│  Analyze Results   │──────▶  go tool pprof -http=:8080   │
│                    │      │  cpu.prof                     │
└────────┬───────────┘      └──────────────────────────────┘
         │
         ▼
┌────────────────────┐      ┌──────────────────────────────┐
│  Identify Hotspot  │──────▶  View Flame Graph            │
│                    │      │  Check Top Functions         │
└────────┬───────────┘      └──────────────────────────────┘
         │
         ▼
┌────────────────────┐
│  Optimize Code     │
└────────┬───────────┘
         │
         ▼
┌────────────────────┐
│  Validate Fix      │
│  (Re-profile)      │
└────────────────────┘
```

## Profile Types Matrix

```
┌──────────────┬─────────────────┬──────────┬─────────────┬──────────────┐
│ Profile Type │    Use Case     │ Overhead │ Production? │   Duration   │
├──────────────┼─────────────────┼──────────┼─────────────┼──────────────┤
│ CPU          │ Hot paths       │   ~5%    │     ✅      │   30-60s     │
│ Heap         │ Memory usage    │   <1%    │     ✅      │   Instant    │
│ Allocs       │ Allocations     │   <1%    │     ✅      │   Instant    │
│ Goroutine    │ Goroutine leaks │   ~0%    │     ✅      │   Instant    │
│ Block        │ Blocking ops    │ Variable │     ⚠️      │   30s+       │
│ Mutex        │ Lock contention │ Variable │     ⚠️      │   30s+       │
│ Trace        │ Scheduler       │  10-30%  │     ❌      │   5-10s      │
└──────────────┴─────────────────┴──────────┴─────────────┴──────────────┘

Legend: ✅ Safe  ⚠️ Caution  ❌ Development Only
```

## Component Interaction

```
┌──────────────────────────────────────────────────────────────┐
│                     Profiler Package                          │
├──────────────────────────────────────────────────────────────┤
│                                                                │
│  ┌────────────────────────────────────────────────────┐      │
│  │               Profiler (profiler.go)               │      │
│  │  ┌──────────────────────────────────────────────┐ │      │
│  │  │  Profile(ctx, opts) → profile file           │ │      │
│  │  │  - CPU profiling (pprof)                     │ │      │
│  │  │  - Heap profiling                            │ │      │
│  │  │  - Goroutine snapshots                       │ │      │
│  │  │  - Execution traces                          │ │      │
│  │  └──────────────────────────────────────────────┘ │      │
│  └────────────────────────────────────────────────────┘      │
│                           │                                   │
│                           ▼                                   │
│  ┌────────────────────────────────────────────────────┐      │
│  │              Analyzer (analyzer.go)                │      │
│  │  ┌──────────────────────────────────────────────┐ │      │
│  │  │  Analyze(path, type) → ProfileStats          │ │      │
│  │  │  - Parse profile using google/pprof          │ │      │
│  │  │  - Extract function statistics               │ │      │
│  │  │  - Generate reports                          │ │      │
│  │  └──────────────────────────────────────────────┘ │      │
│  └────────────────────────────────────────────────────┘      │
│                           │                                   │
│                           ▼                                   │
│  ┌────────────────────────────────────────────────────┐      │
│  │               Server (server.go)                   │      │
│  │  ┌──────────────────────────────────────────────┐ │      │
│  │  │  HTTP Server exposing pprof endpoints        │ │      │
│  │  │  - Configurable host/port                    │ │      │
│  │  │  - Disabled by default                       │ │      │
│  │  │  - Standard pprof handlers                   │ │      │
│  │  └──────────────────────────────────────────────┘ │      │
│  └────────────────────────────────────────────────────┘      │
└──────────────────────────────────────────────────────────────┘
```

## CLI Tool Flow

```
┌─────────────────────┐
│  takhin-profiler    │
│  Command Invoked    │
└──────────┬──────────┘
           │
           ▼
    ┌──────────────┐
    │ Parse Flags  │
    └──────┬───────┘
           │
           ├─────────┐
           │         │
           ▼         ▼
    ┌──────────┐  ┌──────────┐
    │  Local   │  │  Remote  │
    │  Profile │  │  Profile │
    └──────┬───┘  └────┬─────┘
           │           │
           │           ▼
           │     ┌──────────────────────┐
           │     │ HTTP GET to          │
           │     │ remote pprof server  │
           │     └──────────┬───────────┘
           │                │
           ├────────────────┘
           │
           ▼
    ┌──────────────────┐
    │  Save Profile    │
    │  to File         │
    └──────┬───────────┘
           │
           ▼
    ┌──────────────────┐     No
    │  Analyze Flag?   ├──────────┐
    └──────┬───────────┘          │
           │ Yes                  │
           ▼                      │
    ┌──────────────────┐          │
    │  Parse Profile   │          │
    │  Generate Report │          │
    └──────┬───────────┘          │
           │                      │
           ├──────────────────────┘
           │
           ▼
    ┌──────────────────┐
    │  Display Results │
    └──────────────────┘
```

## Flame Graph Visualization

```
Example CPU Profile Flame Graph:

█████████████████████████████████████████████ main.main (100%)
█████████████████████████ server.Start (60%)
███████████████ handler.Handle (40%)
██████████ produce.Handle (25%)
████ storage.Append (10%)
██ log.Write (5%)
█ syscall.Write (2%)
```

Reading the Flame Graph:
- **X-axis (width)**: Proportion of samples (time spent)
- **Y-axis (height)**: Call stack depth (who called whom)
- **Wider = More Time**: Focus optimization here
- **Colors**: Random, for visual separation only

## Data Flow

```
┌──────────────┐
│  Application │
│  Runtime     │
└──────┬───────┘
       │ Performance Data
       ▼
┌──────────────────┐
│  runtime/pprof   │  ← Go Built-in
│  Collects:       │
│  - CPU samples   │
│  - Heap state    │
│  - Goroutines    │
└──────┬───────────┘
       │ Binary Profile
       ▼
┌──────────────────┐
│  Profile File    │
│  (.prof)         │
└──────┬───────────┘
       │
       ├─────────────────┐
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────────┐
│  Analyzer    │  │  go tool pprof   │
│  (Built-in)  │  │  (Standard Tool) │
└──────┬───────┘  └──────┬───────────┘
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────────┐
│  Text        │  │  Flame Graph     │
│  Report      │  │  Interactive UI  │
└──────────────┘  └──────────────────┘
```

## Configuration Hierarchy

```
┌─────────────────────────────────────────────────┐
│               Configuration                      │
├─────────────────────────────────────────────────┤
│                                                  │
│  1. Default Values (in code)                    │
│     profiler:                                   │
│       enabled: false                            │
│       host: "0.0.0.0"                          │
│       port: 6060                               │
│                 │                               │
│                 ▼                               │
│  2. YAML Config File                           │
│     backend/configs/takhin.yaml                │
│     [Override defaults]                        │
│                 │                               │
│                 ▼                               │
│  3. Environment Variables                      │
│     TAKHIN_PROFILER_ENABLED=true              │
│     TAKHIN_PROFILER_PORT=6060                 │
│     [Override YAML]                            │
│                 │                               │
│                 ▼                               │
│  4. Final Configuration                        │
│     Used by Server                             │
│                                                  │
└─────────────────────────────────────────────────┘
```

## Security Model

```
┌───────────────────────────────────────────────────┐
│              Security Layers                       │
├───────────────────────────────────────────────────┤
│                                                    │
│  1. Default Disabled                              │
│     ┌──────────────────────────────────┐         │
│     │  enabled: false                  │         │
│     │  (No exposure by default)        │         │
│     └──────────────────────────────────┘         │
│                    │                              │
│                    ▼                              │
│  2. Localhost Binding (Production)               │
│     ┌──────────────────────────────────┐         │
│     │  host: "127.0.0.1"               │         │
│     │  (Only local access)             │         │
│     └──────────────────────────────────┘         │
│                    │                              │
│                    ▼                              │
│  3. Separate Port                                │
│     ┌──────────────────────────────────┐         │
│     │  port: 6060                      │         │
│     │  (Isolated from main traffic)    │         │
│     └──────────────────────────────────┘         │
│                    │                              │
│                    ▼                              │
│  4. Firewall Rules (Infrastructure)              │
│     ┌──────────────────────────────────┐         │
│     │  Block external access           │         │
│     │  Allow internal monitoring       │         │
│     └──────────────────────────────────┘         │
│                                                    │
└───────────────────────────────────────────────────┘
```

## Performance Impact Visualization

```
Normal Operation (Profiler Disabled)
CPU:  ████████████████████ 100%

With CPU Profiling (30s)
CPU:  █████████████████████ 105%  (+5% overhead)

With Heap Profiling
CPU:  ████████████████████ 100.5%  (<1% overhead)

With Execution Trace (5s)
CPU:  ████████████████████████████ 130%  (+30% overhead)
                                           ⚠️ Dev only!
```

## Integration Points

```
┌────────────────────────────────────────────────────────┐
│                Monitoring Ecosystem                     │
├────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐        ┌──────────────┐             │
│  │  Prometheus  │        │  Grafana     │             │
│  │  Metrics     │───────▶│  Dashboards  │             │
│  │  :9090       │        │              │             │
│  └──────────────┘        └──────────────┘             │
│         │                                              │
│         │ Alert on High CPU/Memory                    │
│         ▼                                              │
│  ┌──────────────┐        ┌──────────────┐             │
│  │  Alert       │        │  takhin-     │             │
│  │  Triggered   │───────▶│  profiler    │             │
│  │              │        │  Collect     │             │
│  └──────────────┘        └──────┬───────┘             │
│                                 │                      │
│                                 ▼                      │
│                          ┌──────────────┐             │
│                          │  Profile     │             │
│                          │  Storage     │             │
│                          │  (S3/NFS)    │             │
│                          └──────────────┘             │
│                                                         │
└────────────────────────────────────────────────────────┘
```

## File Structure

```
Takhin/
├── backend/
│   ├── pkg/
│   │   └── profiler/              ← New Package
│   │       ├── profiler.go        (Core profiling)
│   │       ├── analyzer.go        (Analysis)
│   │       ├── server.go          (HTTP server)
│   │       └── profiler_test.go   (Tests)
│   │
│   ├── cmd/
│   │   └── takhin-profiler/       ← New CLI Tool
│   │       └── main.go
│   │
│   ├── configs/
│   │   └── takhin.yaml            (+ profiler config)
│   │
│   └── pkg/config/
│       └── config.go              (+ ProfilerConfig)
│
├── build/
│   └── takhin-profiler            ← Built binary
│
└── TASK_8.5_*.md                  ← Documentation
```

## Summary

The profiling system provides:
- ✅ **Multiple profile types** for different use cases
- ✅ **Minimal overhead** for production use
- ✅ **Easy CLI tool** for operators
- ✅ **Standard integration** with go tool pprof
- ✅ **Security-first** design (disabled by default)
- ✅ **Comprehensive testing** with full coverage

Perfect for:
- 🎯 Performance optimization
- 🐛 Memory leak detection  
- 🔍 Goroutine debugging
- 📊 Latency analysis
- 🔥 Flame graph generation
