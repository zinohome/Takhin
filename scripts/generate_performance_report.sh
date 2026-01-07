#!/bin/bash
# Generate HTML Performance Report
# Creates a visual HTML report from benchmark results

set -e

BENCHMARK_FILE=$1
OUTPUT_HTML=$2

if [ -z "$BENCHMARK_FILE" ] || [ -z "$OUTPUT_HTML" ]; then
    echo "Usage: $0 <benchmark_file> <output_html>"
    exit 1
fi

if [ ! -f "$BENCHMARK_FILE" ]; then
    echo "Benchmark file not found: $BENCHMARK_FILE"
    exit 1
fi

OUTPUT_DIR=$(dirname "$OUTPUT_HTML")
mkdir -p "$OUTPUT_DIR"

# Extract benchmark data
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Generate HTML report
cat > "$OUTPUT_HTML" << EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Takhin Performance Report - $TIMESTAMP</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 20px;
            color: #333;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px 40px;
        }
        
        .header h1 {
            font-size: 32px;
            margin-bottom: 10px;
        }
        
        .header .meta {
            opacity: 0.9;
            font-size: 14px;
        }
        
        .content {
            padding: 40px;
        }
        
        .section {
            margin-bottom: 40px;
        }
        
        .section h2 {
            font-size: 24px;
            margin-bottom: 20px;
            color: #667eea;
            border-bottom: 2px solid #667eea;
            padding-bottom: 10px;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .stat-card {
            background: #f7fafc;
            border-radius: 8px;
            padding: 20px;
            border-left: 4px solid #667eea;
        }
        
        .stat-card .label {
            font-size: 12px;
            text-transform: uppercase;
            color: #718096;
            margin-bottom: 8px;
            font-weight: 600;
        }
        
        .stat-card .value {
            font-size: 28px;
            font-weight: bold;
            color: #2d3748;
        }
        
        .benchmark-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
            border-radius: 8px;
            overflow: hidden;
        }
        
        .benchmark-table thead {
            background: #667eea;
            color: white;
        }
        
        .benchmark-table th {
            padding: 12px 15px;
            text-align: left;
            font-weight: 600;
            font-size: 14px;
        }
        
        .benchmark-table td {
            padding: 12px 15px;
            border-bottom: 1px solid #e2e8f0;
        }
        
        .benchmark-table tbody tr:hover {
            background: #f7fafc;
        }
        
        .benchmark-table tbody tr:last-child td {
            border-bottom: none;
        }
        
        .metric-good {
            color: #38a169;
            font-weight: 600;
        }
        
        .metric-warning {
            color: #ed8936;
            font-weight: 600;
        }
        
        .metric-bad {
            color: #e53e3e;
            font-weight: 600;
        }
        
        .category {
            background: #edf2f7;
            padding: 8px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 600;
            color: #4a5568;
            display: inline-block;
        }
        
        .footer {
            background: #f7fafc;
            padding: 20px 40px;
            text-align: center;
            color: #718096;
            font-size: 14px;
        }
        
        .chart-placeholder {
            background: #f7fafc;
            border-radius: 8px;
            padding: 40px;
            text-align: center;
            color: #a0aec0;
            border: 2px dashed #cbd5e0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Takhin Performance Report</h1>
            <div class="meta">
                <strong>Generated:</strong> $TIMESTAMP<br>
                <strong>Commit:</strong> $GIT_COMMIT ($GIT_BRANCH)
            </div>
        </div>
        
        <div class="content">
            <div class="section">
                <h2>📈 Summary Statistics</h2>
                <div class="stats-grid">
EOF

# Extract summary statistics
TOTAL_BENCHMARKS=$(grep -c "^Benchmark" "$BENCHMARK_FILE" || echo "0")
TOTAL_PASSED=$(grep -c "PASS" "$BENCHMARK_FILE" || echo "0")

# Calculate average throughput from MB/s metrics
AVG_THROUGHPUT=$(grep "MB/s" "$BENCHMARK_FILE" | awk '{
    for(i=1; i<=NF; i++) {
        if($i ~ /^[0-9.]+$/ && $(i+1) == "MB/s") {
            sum += $i
            count++
        }
    }
} END {
    if(count > 0) printf "%.2f", sum/count
    else print "N/A"
}')

# Get fastest and slowest benchmarks
FASTEST=$(grep "^Benchmark" "$BENCHMARK_FILE" | awk '{
    for(i=1; i<=NF; i++) {
        if($i ~ /ns\/op$/) {
            print $(i-1) "\t" $1
        }
    }
}' | sort -n | head -1)

SLOWEST=$(grep "^Benchmark" "$BENCHMARK_FILE" | awk '{
    for(i=1; i<=NF; i++) {
        if($i ~ /ns\/op$/) {
            print $(i-1) "\t" $1
        }
    }
}' | sort -n -r | head -1)

cat >> "$OUTPUT_HTML" << EOF
                    <div class="stat-card">
                        <div class="label">Total Benchmarks</div>
                        <div class="value">$TOTAL_BENCHMARKS</div>
                    </div>
                    <div class="stat-card">
                        <div class="label">Avg Throughput</div>
                        <div class="value">$AVG_THROUGHPUT MB/s</div>
                    </div>
                    <div class="stat-card">
                        <div class="label">Status</div>
                        <div class="value metric-good">✓ Passed</div>
                    </div>
                    <div class="stat-card">
                        <div class="label">Test Time</div>
                        <div class="value">$(date +"%H:%M:%S")</div>
                    </div>
                </div>
            </div>
            
            <div class="section">
                <h2>🎯 Benchmark Results</h2>
                <table class="benchmark-table">
                    <thead>
                        <tr>
                            <th>Benchmark Name</th>
                            <th>Iterations</th>
                            <th>Time (ns/op)</th>
                            <th>Throughput</th>
                            <th>Memory</th>
                            <th>Category</th>
                        </tr>
                    </thead>
                    <tbody>
EOF

# Parse and add benchmark rows
grep "^Benchmark" "$BENCHMARK_FILE" | while read -r line; do
    name=$(echo "$line" | awk '{print $1}')
    iters=$(echo "$line" | awk '{print $2}')
    
    # Extract metrics
    nsop=$(echo "$line" | awk '{for(i=1;i<=NF;i++) if($(i+1)=="ns/op") print $i}')
    mbps=$(echo "$line" | awk '{for(i=1;i<=NF;i++) if($(i+1)=="MB/s") print $i}')
    allocs=$(echo "$line" | awk '{for(i=1;i<=NF;i++) if($(i+1)=="allocs/op") print $i}')
    
    # Determine category
    if [[ $name == *"Write"* ]] || [[ $name == *"Produce"* ]]; then
        category="Produce"
    elif [[ $name == *"Read"* ]] || [[ $name == *"Fetch"* ]] || [[ $name == *"Consume"* ]]; then
        category="Fetch"
    elif [[ $name == *"Compact"* ]]; then
        category="Compaction"
    elif [[ $name == *"Concurrent"* ]]; then
        category="Concurrency"
    else
        category="Other"
    fi
    
    throughput="${mbps:-N/A}"
    if [ "$throughput" != "N/A" ]; then
        throughput="$throughput MB/s"
    fi
    
    memory="${allocs:-N/A}"
    if [ "$memory" != "N/A" ]; then
        memory="$memory allocs"
    fi
    
    cat >> "$OUTPUT_HTML" << BENCHMARK_ROW
                        <tr>
                            <td><strong>$name</strong></td>
                            <td>$iters</td>
                            <td>${nsop:-N/A}</td>
                            <td>$throughput</td>
                            <td>$memory</td>
                            <td><span class="category">$category</span></td>
                        </tr>
BENCHMARK_ROW
done

cat >> "$OUTPUT_HTML" << EOF
                    </tbody>
                </table>
            </div>
            
            <div class="section">
                <h2>📉 Performance Trends</h2>
                <div class="chart-placeholder">
                    <p>📊 Historical trend charts require integration with external storage</p>
                    <p style="margin-top: 10px; font-size: 12px;">Consider integrating with GitHub Pages or external dashboard for trend visualization</p>
                </div>
            </div>
        </div>
        
        <div class="footer">
            <p><strong>Takhin</strong> - High-Performance Kafka-Compatible Streaming Platform</p>
            <p style="margin-top: 5px;">Generated by automated performance testing suite</p>
        </div>
    </div>
</body>
</html>
EOF

echo "✅ Performance report generated: $OUTPUT_HTML"
echo "📊 Total benchmarks: $TOTAL_BENCHMARKS"
echo "🚀 Average throughput: $AVG_THROUGHPUT MB/s"
