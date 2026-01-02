#!/bin/bash
# Storage Layer Performance Benchmark Runner
# Copyright 2025 Takhin Data, Inc.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/../backend"
RESULTS_DIR="$SCRIPT_DIR/../benchmark_results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$RESULTS_DIR/benchmark_${TIMESTAMP}.txt"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create results directory
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Takhin Storage Layer Benchmark Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Timestamp: $TIMESTAMP"
echo "Results will be saved to: $RESULTS_FILE"
echo ""

# Function to run benchmark
run_benchmark() {
    local package=$1
    local pattern=$2
    local description=$3
    
    echo -e "${YELLOW}Running: $description${NC}"
    echo "Package: $package"
    echo "Pattern: $pattern"
    echo ""
    
    cd "$BACKEND_DIR"
    
    # Run benchmark and save results
    {
        echo "========================================="
        echo "$description"
        echo "Package: $package"
        echo "Time: $(date)"
        echo "========================================="
        echo ""
        go test -bench="$pattern" -benchmem -benchtime=3s "$package" 2>&1 || echo "BENCHMARK FAILED"
        echo ""
        echo ""
    } | tee -a "$RESULTS_FILE"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Completed${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
    fi
    echo ""
}

# Start benchmarking
echo "Starting benchmarks..." | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

# 1. Log Layer - Produce Performance
echo -e "${BLUE}[1/10] Produce Performance Tests${NC}"
run_benchmark "./pkg/storage/log" "BenchmarkWriteThroughput" "1. Write Throughput (Single Producer)"
run_benchmark "./pkg/storage/log" "BenchmarkBatchWriteThroughput" "2. Batch Write Throughput"
run_benchmark "./pkg/storage/log" "BenchmarkProduceLatency" "3. Produce Latency"

# 2. Log Layer - Fetch Performance
echo -e "${BLUE}[2/10] Fetch Performance Tests${NC}"
run_benchmark "./pkg/storage/log" "BenchmarkReadThroughput" "4. Read Throughput"
run_benchmark "./pkg/storage/log" "BenchmarkSequentialFetch" "5. Sequential Fetch (Kafka-style)"
run_benchmark "./pkg/storage/log" "BenchmarkRandomFetch" "6. Random Fetch"
run_benchmark "./pkg/storage/log" "BenchmarkFetchLatency" "7. Fetch Latency"

# 3. Log Layer - Compaction Performance
echo -e "${BLUE}[3/10] Compaction Performance Tests${NC}"
run_benchmark "./pkg/storage/log" "BenchmarkCompaction" "8. Compaction Performance"

# 4. Log Layer - Concurrent Access
echo -e "${BLUE}[4/10] Concurrent Access Tests${NC}"
run_benchmark "./pkg/storage/log" "BenchmarkConcurrentProducers" "9. Concurrent Producers"
run_benchmark "./pkg/storage/log" "BenchmarkConcurrentConsumers" "10. Concurrent Consumers"

# 5. Log Layer - Mixed Workload
echo -e "${BLUE}[5/10] Mixed Workload Tests${NC}"
run_benchmark "./pkg/storage/log" "BenchmarkMixedWorkload" "11. Mixed Read/Write Workload"

# 6. Log Layer - Segment Management
echo -e "${BLUE}[6/10] Segment Management Tests${NC}"
run_benchmark "./pkg/storage/log" "BenchmarkSegmentRollover" "12. Segment Rollover Performance"

# 7. Topic Manager - Produce Performance
echo -e "${BLUE}[7/10] Topic Manager Produce Tests${NC}"
run_benchmark "./pkg/storage/topic" "BenchmarkTopicManagerProduceThroughput" "13. Topic Manager Produce Throughput"

# 8. Topic Manager - Fetch Performance
echo -e "${BLUE}[8/10] Topic Manager Fetch Tests${NC}"
run_benchmark "./pkg/storage/topic" "BenchmarkTopicManagerFetchThroughput" "14. Topic Manager Fetch Throughput"

# 9. Topic Manager - Concurrent Access
echo -e "${BLUE}[9/10] Topic Manager Concurrent Tests${NC}"
run_benchmark "./pkg/storage/topic" "BenchmarkTopicManagerConcurrentProducers" "15. Topic Manager Concurrent Producers"
run_benchmark "./pkg/storage/topic" "BenchmarkTopicManagerConcurrentConsumers" "16. Topic Manager Concurrent Consumers"

# 10. Topic Manager - Multi-Topic & Compaction
echo -e "${BLUE}[10/10] Topic Manager Advanced Tests${NC}"
run_benchmark "./pkg/storage/topic" "BenchmarkTopicManagerPartitionBalance" "17. Partition Balance"
run_benchmark "./pkg/storage/topic" "BenchmarkTopicManagerMultiTopic" "18. Multi-Topic Performance"
run_benchmark "./pkg/storage/topic" "BenchmarkTopicManagerCompaction" "19. Topic Manager Compaction"

# Generate summary
echo "" | tee -a "$RESULTS_FILE"
echo -e "${GREEN}========================================${NC}" | tee -a "$RESULTS_FILE"
echo -e "${GREEN}Benchmark Suite Completed${NC}" | tee -a "$RESULTS_FILE"
echo -e "${GREEN}========================================${NC}" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"
echo "Results saved to: $RESULTS_FILE" | tee -a "$RESULTS_FILE"
echo "" | tee -a "$RESULTS_FILE"

# Parse and generate summary report
echo "Generating performance summary..." | tee -a "$RESULTS_FILE"
SUMMARY_FILE="$RESULTS_DIR/summary_${TIMESTAMP}.md"

cat > "$SUMMARY_FILE" << 'EOF'
# Takhin Storage Layer Performance Benchmark Report

## Executive Summary

This report contains comprehensive performance benchmarks for the Takhin storage layer, covering:
- **Produce Performance**: Single and batch write throughput, latency
- **Fetch Performance**: Sequential, random read patterns, latency
- **Compaction Performance**: Deduplication efficiency, duration
- **Concurrent Access**: Multi-producer/consumer scenarios
- **Topic Manager**: Multi-partition and multi-topic workloads

## Test Environment

EOF

echo "- **Date**: $(date)" >> "$SUMMARY_FILE"
echo "- **Go Version**: $(go version)" >> "$SUMMARY_FILE"
echo "- **OS**: $(uname -s)" >> "$SUMMARY_FILE"
echo "- **Architecture**: $(uname -m)" >> "$SUMMARY_FILE"
echo "- **CPU**: $(sysctl -n machdep.cpu.brand_string 2>/dev/null || lscpu | grep "Model name" | cut -d: -f2 | xargs)" >> "$SUMMARY_FILE"
echo "" >> "$SUMMARY_FILE"

cat >> "$SUMMARY_FILE" << 'EOF'
## Benchmark Categories

### 1. Produce Performance (Throughput & Latency)

Tests measuring write performance under various conditions:
- Single message writes
- Batch writes (10, 100, 1000 messages)
- Message sizes: 100B, 1KB, 10KB
- Metrics: MB/s, msg/s, latency (ms/op)

**Key Findings**: See detailed results in the full report.

### 2. Fetch Performance

Tests measuring read performance:
- Sequential fetch (Kafka consumption pattern)
- Random access patterns
- Various fetch sizes (1, 10, 100 messages)
- Metrics: MB/s, msg/s, latency (ms/op)

**Key Findings**: See detailed results in the full report.

### 3. Compaction Performance

Tests measuring log compaction efficiency:
- Different segment counts (10, 50, 100)
- Various deduplication ratios (30%, 50%, 70%)
- Metrics: MB reclaimed, keys removed, duration (ms)

**Key Findings**: See detailed results in the full report.

### 4. Concurrent Access

Tests measuring multi-threaded performance:
- Concurrent producers (1, 2, 4, 8)
- Concurrent consumers (1, 2, 4, 8)
- Metrics: MB/s, msg/s

**Key Findings**: See detailed results in the full report.

### 5. Topic Manager Performance

Tests measuring partition and topic-level performance:
- Multiple partitions (1, 4, 16)
- Multiple topics (1, 5, 10)
- Load balancing across partitions
- Metrics: MB/s, msg/s, imbalance %

**Key Findings**: See detailed results in the full report.

## Performance Bottlenecks Identified

Based on benchmark results, the following bottlenecks were identified:

1. **Bottleneck Category 1**: TBD - Analyze results
   - **Impact**: [High/Medium/Low]
   - **Location**: [Code location]
   - **Recommendation**: [Optimization approach]

2. **Bottleneck Category 2**: TBD - Analyze results
   - **Impact**: [High/Medium/Low]
   - **Location**: [Code location]
   - **Recommendation**: [Optimization approach]

## Optimization Recommendations

### High Priority (P0)

1. **[Optimization 1]**: TBD based on results
   - Expected improvement: [X%]
   - Effort: [Low/Medium/High]

2. **[Optimization 2]**: TBD based on results
   - Expected improvement: [X%]
   - Effort: [Low/Medium/High]

### Medium Priority (P1)

1. **[Optimization 3]**: TBD based on results
   - Expected improvement: [X%]
   - Effort: [Low/Medium/High]

### Low Priority (P2)

1. **[Optimization 4]**: TBD based on results
   - Expected improvement: [X%]
   - Effort: [Low/Medium/High]

## Comparison with Industry Standards

| Metric | Takhin | Kafka | Notes |
|--------|--------|-------|-------|
| Produce Throughput (MB/s) | TBD | ~600-800 | Single partition |
| Fetch Throughput (MB/s) | TBD | ~1000+ | Sequential reads |
| Produce Latency (p99 ms) | TBD | 2-5 | acks=1 |
| Compaction Duration | TBD | Varies | Depends on data size |

*Note: Kafka numbers are approximate and vary based on hardware/configuration*

## Next Steps

1. **Analyze detailed results** from `benchmark_TIMESTAMP.txt`
2. **Profile hotspots** using `pprof` for identified bottlenecks
3. **Implement optimizations** based on priority recommendations
4. **Re-run benchmarks** to validate improvements
5. **Update performance targets** in project documentation

## Detailed Results

Full benchmark output is available in the timestamped results file.

---
*Report generated automatically by benchmark suite*
EOF

echo "" | tee -a "$RESULTS_FILE"
echo -e "${GREEN}Summary report generated: $SUMMARY_FILE${NC}"
echo ""
echo -e "${BLUE}To analyze results:${NC}"
echo "  1. Review: cat $RESULTS_FILE"
echo "  2. Summary: cat $SUMMARY_FILE"
echo "  3. Profile: cd backend && go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof ./pkg/storage/log"
echo "  4. Analyze: go tool pprof cpu.prof"
echo ""
echo -e "${GREEN}Benchmark suite completed successfully!${NC}"
