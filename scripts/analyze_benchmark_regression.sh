#!/bin/bash
# Analyze Benchmark Regression
# Compares baseline and current benchmark results and detects regressions

set -e

BASELINE_FILE=$1
CURRENT_FILE=$2
REPORT_FILE=$3
THROUGHPUT_THRESHOLD=${4:-10}  # % decrease allowed
LATENCY_THRESHOLD=${5:-20}     # % increase allowed  
MEMORY_THRESHOLD=${6:-15}      # % increase in allocs/op allowed

if [ -z "$BASELINE_FILE" ] || [ -z "$CURRENT_FILE" ] || [ -z "$REPORT_FILE" ]; then
    echo "Usage: $0 <baseline_file> <current_file> <report_file> [throughput_threshold] [latency_threshold] [memory_threshold]"
    exit 1
fi

if [ ! -f "$BASELINE_FILE" ]; then
    echo "Baseline file not found: $BASELINE_FILE"
    exit 1
fi

if [ ! -f "$CURRENT_FILE" ]; then
    echo "Current file not found: $CURRENT_FILE"
    exit 1
fi

REPORT_DIR=$(dirname "$REPORT_FILE")
mkdir -p "$REPORT_DIR"

REGRESSION_FOUND=false

# Initialize report
cat > "$REPORT_FILE" << 'EOF'
### Performance Analysis Summary

EOF

# Extract benchmark results using awk
extract_benchmarks() {
    local file=$1
    # Extract lines like: BenchmarkName-8    1000    1234 ns/op    100 MB/s    5 allocs/op
    grep "^Benchmark" "$file" | awk '{
        name = $1
        iters = $2
        
        # Find ns/op
        for (i=3; i<=NF; i++) {
            if ($i ~ /ns\/op$/) {
                nsop = $(i-1)
            }
            if ($i ~ /MB\/s$/) {
                mbps = $(i-1)
            }
            if ($i ~ /allocs\/op$/) {
                allocs = $(i-1)
            }
            if ($i ~ /B\/op$/) {
                bytes = $(i-1)
            }
        }
        
        if (nsop != "") {
            print name "\t" nsop "\t" mbps "\t" allocs "\t" bytes
        }
    }'
}

echo "Extracting baseline benchmarks..."
extract_benchmarks "$BASELINE_FILE" > /tmp/baseline_parsed.txt

echo "Extracting current benchmarks..."
extract_benchmarks "$CURRENT_FILE" > /tmp/current_parsed.txt

# Compare benchmarks
echo "Comparing benchmarks..."

cat >> "$REPORT_FILE" << EOF
| Benchmark | Metric | Baseline | Current | Change | Status |
|-----------|--------|----------|---------|--------|--------|
EOF

while IFS=$'\t' read -r name base_nsop base_mbps base_allocs base_bytes; do
    # Find matching current benchmark
    current_line=$(grep "^${name}" /tmp/current_parsed.txt || echo "")
    
    if [ -z "$current_line" ]; then
        echo "| $name | - | - | Not found | - | ⚠️ Missing |" >> "$REPORT_FILE"
        continue
    fi
    
    curr_nsop=$(echo "$current_line" | cut -f2)
    curr_mbps=$(echo "$current_line" | cut -f3)
    curr_allocs=$(echo "$current_line" | cut -f4)
    
    # Calculate latency change (ns/op - lower is better)
    if [ -n "$base_nsop" ] && [ -n "$curr_nsop" ]; then
        latency_change=$(awk "BEGIN {printf \"%.2f\", (($curr_nsop - $base_nsop) / $base_nsop) * 100}")
        
        if (( $(echo "$latency_change > $LATENCY_THRESHOLD" | bc -l) )); then
            status="🔴 Regression"
            REGRESSION_FOUND=true
        elif (( $(echo "$latency_change < -10" | bc -l) )); then
            status="🟢 Improvement"
        else
            status="✅ OK"
        fi
        
        echo "| $name | Latency (ns/op) | $base_nsop | $curr_nsop | ${latency_change}% | $status |" >> "$REPORT_FILE"
    fi
    
    # Calculate throughput change (MB/s - higher is better)
    if [ -n "$base_mbps" ] && [ -n "$curr_mbps" ] && [ "$base_mbps" != "" ]; then
        throughput_change=$(awk "BEGIN {printf \"%.2f\", (($curr_mbps - $base_mbps) / $base_mbps) * 100}")
        
        if (( $(echo "$throughput_change < -$THROUGHPUT_THRESHOLD" | bc -l) )); then
            status="🔴 Regression"
            REGRESSION_FOUND=true
        elif (( $(echo "$throughput_change > 10" | bc -l) )); then
            status="🟢 Improvement"
        else
            status="✅ OK"
        fi
        
        echo "| $name | Throughput (MB/s) | $base_mbps | $curr_mbps | ${throughput_change}% | $status |" >> "$REPORT_FILE"
    fi
    
    # Calculate memory change (allocs/op - lower is better)
    if [ -n "$base_allocs" ] && [ -n "$curr_allocs" ] && [ "$base_allocs" != "" ]; then
        allocs_change=$(awk "BEGIN {printf \"%.2f\", (($curr_allocs - $base_allocs) / $base_allocs) * 100}")
        
        if (( $(echo "$allocs_change > $MEMORY_THRESHOLD" | bc -l) )); then
            status="🔴 Regression"
            REGRESSION_FOUND=true
        elif (( $(echo "$allocs_change < -10" | bc -l) )); then
            status="🟢 Improvement"
        else
            status="✅ OK"
        fi
        
        echo "| $name | Allocs/op | $base_allocs | $curr_allocs | ${allocs_change}% | $status |" >> "$REPORT_FILE"
    fi
    
done < /tmp/baseline_parsed.txt

# Add summary
cat >> "$REPORT_FILE" << EOF

### Thresholds Used

- **Throughput Regression**: > ${THROUGHPUT_THRESHOLD}% decrease
- **Latency Regression**: > ${LATENCY_THRESHOLD}% increase
- **Memory Regression**: > ${MEMORY_THRESHOLD}% increase in allocations

### Legend

- 🔴 **Regression**: Performance degraded beyond threshold
- 🟢 **Improvement**: Significant performance improvement (>10%)
- ✅ **OK**: Within acceptable performance variation
- ⚠️ **Missing**: Benchmark not found in current run

EOF

if [ "$REGRESSION_FOUND" = true ]; then
    cat >> "$REPORT_FILE" << EOF

### ⚠️ Action Required

Performance regressions have been detected. Please:

1. Review the affected benchmarks above
2. Profile the code to identify bottlenecks
3. Consider reverting or optimizing the changes
4. Re-run benchmarks after fixes

EOF
    
    # Create marker file for CI
    touch "$REPORT_DIR/regression_detected"
    echo "❌ Regressions detected!"
    exit_code=1
else
    cat >> "$REPORT_FILE" << EOF

### ✅ All Clear

No significant performance regressions detected. All metrics are within acceptable thresholds.

EOF
    echo "✅ No regressions found"
    exit_code=0
fi

# Cleanup
rm -f /tmp/baseline_parsed.txt /tmp/current_parsed.txt

echo "Report written to: $REPORT_FILE"
exit $exit_code
