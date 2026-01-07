# Task 8.3: Performance Regression Testing - Quick Reference

## 🚀 Quick Start

### Run Performance Tests Locally

```bash
# Quick benchmark (1-2 minutes)
task backend:bench:quick

# Full benchmark suite (10-30 minutes)
task backend:bench

# Compare with baseline
task backend:bench:regression

# Generate HTML report
task backend:bench:report
```

### First Time Setup

```bash
# 1. Save initial baseline
task backend:bench:save-baseline

# 2. Make performance changes
# ... edit code ...

# 3. Check for regressions
task backend:bench:regression

# 4. View detailed report
task backend:bench:report
open benchmark_results/report.html
```

## 📊 CI Workflow

### Automatic Triggers

**Pull Requests** (on changes to):
- `backend/pkg/storage/**`
- `backend/pkg/kafka/handler/**`
- `backend/pkg/grpcapi/**`
- `backend/pkg/throttle/**`
- `backend/pkg/mempool/**`

**Scheduled**: Every Sunday at 2 AM UTC

**Manual Trigger**:
```bash
gh workflow run performance-regression.yml \
  -f baseline_ref=main \
  -f benchmark_time=5s
```

### Workflow Jobs

1. **benchmark-current**: Run benchmarks on current code
2. **benchmark-baseline**: Run benchmarks on baseline (main branch)
3. **compare-and-report**: Compare results, generate reports, post PR comment
4. **store-baseline**: Archive results (main branch only)
5. **alert-on-regression**: Send alerts if regressions detected

## 🎯 Key Files

### Workflow
- `.github/workflows/performance-regression.yml` - Main CI workflow

### Scripts
- `scripts/analyze_benchmark_regression.sh` - Compare baseline vs current
- `scripts/generate_performance_report.sh` - Generate HTML report
- `scripts/run_benchmarks.sh` - Comprehensive benchmark runner

### Tools
- `backend/cmd/benchcmp/main.go` - Go-based comparison tool

### Results
- `benchmark_results/current.txt` - Latest benchmark results
- `benchmark_results/baseline.txt` - Baseline for comparison
- `benchmark_results/regression_report.md` - Regression analysis
- `benchmark_results/report.html` - Visual report

## 🔧 Configuration

### Regression Thresholds

```yaml
# In workflow or script parameters
THROUGHPUT_THRESHOLD: '10'   # % decrease = regression
LATENCY_THRESHOLD: '20'      # % increase = regression
MEMORY_THRESHOLD: '15'       # % increase in allocs = regression
```

### Benchmark Coverage

```go
// Packages tested
./pkg/storage/log         // Storage layer
./pkg/storage/topic       // Topic manager
./pkg/kafka/handler       // Kafka handlers
./pkg/grpcapi            // gRPC API
./pkg/throttle           // Network throttling
./pkg/mempool            // Memory pool
```

### Benchmark Duration

```bash
# Quick: 1s per test
go test -bench=. -benchtime=1s

# Default: 5s per test
go test -bench=. -benchtime=5s

# Comprehensive: 10s per test
go test -bench=. -benchtime=10s
```

## 📈 Metrics Tracked

| Metric | Unit | Better | Example |
|--------|------|--------|---------|
| Throughput | MB/s | Higher | 850 MB/s |
| Latency | ns/op | Lower | 1234 ns/op |
| Allocations | allocs/op | Lower | 5 allocs/op |
| Memory | B/op | Lower | 1024 B/op |

## 🚨 Alert Channels

### PR Comments
- Posted automatically on PR
- Shows comparison table
- Indicates regressions with 🔴
- Fails CI if regression detected

### Slack Notifications
- Requires `SLACK_WEBHOOK_URL` secret
- Sent on scheduled run failures
- Links to workflow run

### GitHub Issues
- Auto-created on scheduled failures
- Tags: `performance`, `regression`, `P1`
- Assigns: `@performance-team`

## 🛠️ Common Tasks

### Save New Baseline
```bash
# After verifying improvements
task backend:bench:save-baseline
git add benchmark_results/baseline.txt
git commit -m "perf: update performance baseline"
```

### Compare with Specific Commit
```bash
# Using benchcmp tool
cd backend
go run ./cmd/benchcmp \
  -baseline=../benchmark_results/baseline.txt \
  -current=../benchmark_results/current.txt
```

### Profile Regressed Benchmark
```bash
cd backend
go test -bench=BenchmarkName \
  -cpuprofile=cpu.prof \
  -memprofile=mem.prof \
  -benchtime=10s \
  ./pkg/storage/log

# View CPU profile
go tool pprof -http=:8080 cpu.prof

# View memory profile
go tool pprof -http=:8080 mem.prof
```

### Run Specific Benchmark
```bash
cd backend
go test -bench=BenchmarkWriteThroughput \
  -benchtime=5s \
  -benchmem \
  ./pkg/storage/log
```

### Generate Report Only
```bash
# If you already have benchmark results
./scripts/generate_performance_report.sh \
  benchmark_results/current.txt \
  benchmark_results/report.html
```

## 📖 Reading Results

### Status Indicators

- 🔴 **Regression**: Exceeds threshold, needs attention
- 🟢 **Improvement**: >10% improvement, good job!
- ✅ **OK**: Within acceptable range
- ⚠️ **Missing**: Benchmark not found in current run

### Example Output

```markdown
| Benchmark | Metric | Baseline | Current | Change | Status |
|-----------|--------|----------|---------|--------|--------|
| BenchmarkWriteThroughput | Throughput (MB/s) | 850 | 780 | -8.2% | ✅ OK |
| BenchmarkProduceLatency | Latency (ns/op) | 1234 | 1567 | +27% | 🔴 Regression |
| BenchmarkReadThroughput | Throughput (MB/s) | 1200 | 1350 | +12.5% | 🟢 Improvement |
```

### Interpreting Changes

**Throughput (MB/s)**:
- Decrease > 10% → 🔴 Regression
- Increase > 10% → 🟢 Improvement

**Latency (ns/op)**:
- Increase > 20% → 🔴 Regression
- Decrease > 10% → 🟢 Improvement

**Memory (allocs/op)**:
- Increase > 15% → 🔴 Regression
- Decrease > 10% → 🟢 Improvement

## 🔍 Troubleshooting

### CI Fails with Regression

1. **Review PR comment** for details
2. **Download artifacts** from workflow
3. **Profile locally**:
   ```bash
   task backend:bench:regression
   ```
4. **Fix or discuss** with team

### Inconsistent Results

1. **Run multiple times** for average
2. **Check system load** during benchmark
3. **Use dedicated runners** in CI
4. **Increase benchmark time**:
   ```bash
   go test -bench=. -benchtime=10s
   ```

### Missing Baseline

```bash
# CI will auto-create on first run
# Or manually save:
task backend:bench:save-baseline
```

### Script Permissions

```bash
chmod +x scripts/analyze_benchmark_regression.sh
chmod +x scripts/generate_performance_report.sh
chmod +x scripts/run_benchmarks.sh
```

## 💡 Tips

### Before Submitting PR
1. Run `task backend:bench:regression` locally
2. Review any regressions
3. Profile and optimize if needed
4. Document trade-offs in PR

### After Performance Optimization
1. Save baseline before changes
2. Make optimizations
3. Run regression test
4. Show before/after in PR
5. Update baseline after merge

### Baseline Management
- Update after verified improvements
- Archive for major releases
- Document significant changes
- Keep history for trends

## 🔗 Related Commands

```bash
# Development
task backend:test           # Run all tests
task backend:test:unit      # Unit tests only
task backend:lint           # Linting

# Benchmarking
task backend:bench          # Full suite
task backend:bench:quick    # Quick test
task backend:bench:regression  # Compare with baseline

# Building
task backend:build          # Build all binaries
task cli:build             # Build CLI only

# Profiling (from TASK_8.5)
task backend:profile       # Run profiler
```

## 📚 Documentation

- **Full Guide**: `TASK_8.3_COMPLETION_SUMMARY.md`
- **Benchmark Details**: `docs/performance/benchmarks.md`
- **Architecture**: `TASK_1.3_ARCHITECTURE.md`
- **Profiling**: `TASK_8.5_PROFILING_COMPLETION.md`

---

**Quick Help**: `task --list | grep bench`  
**CI Workflow**: `.github/workflows/performance-regression.yml`  
**Issues**: Tag with `performance` label
