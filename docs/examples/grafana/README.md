# Takhin Grafana Dashboards

This directory contains pre-built Grafana dashboards for monitoring Takhin streaming platform.

## Available Dashboards

### 1. Cluster Overview Dashboard
**File:** `cluster-overview-dashboard.json`  
**Description:** High-level cluster health and performance metrics

**Panels:**
- Active Connections
- Request Rate
- Error Rate
- Memory Usage
- Goroutines
- Total Storage
- Request Rate by API
- Request Latency (Percentiles)
- Network Throughput
- Memory Usage Details
- Goroutines and Threads
- Garbage Collection

**Use Cases:**
- Quick health check of the cluster
- Identify performance bottlenecks
- Monitor resource utilization
- Track overall system health

---

### 2. Topic Monitoring Dashboard
**File:** `topic-monitoring-dashboard.json`  
**Description:** Topic-specific metrics and performance monitoring

**Panels:**
- Produce Request Rate by Topic
- Fetch Request Rate by Topic
- Produce Throughput (Bytes/sec)
- Fetch Throughput (Bytes/sec)
- Produce Latency (Percentiles)
- Fetch Latency (Percentiles)
- Disk Usage by Topic
- Log Segments by Topic
- Log End Offset (High Water Mark)
- Storage I/O Operations
- Storage I/O Errors

**Variables:**
- `$topic` - Filter by topic name (multi-select, includes All)

**Use Cases:**
- Monitor topic-specific throughput
- Track disk usage per topic
- Identify slow topics
- Analyze I/O patterns

---

### 3. Consumer Group Dashboard
**File:** `consumer-group-dashboard.json`  
**Description:** Consumer group health and lag monitoring

**Panels:**
- Total Consumer Lag
- Total Group Members
- Commit Rate
- Rebalances (Last Hour)
- Consumer Group Lag by Topic
- Consumer Group Members
- Rebalance Events (5min windows)
- Offset Commit Rate
- Consumer Group State
- Consumer Lag Details (Table)

**Variables:**
- `$group` - Filter by consumer group ID (multi-select, includes All)

**Use Cases:**
- Monitor consumer lag
- Track rebalance events
- Identify slow consumers
- Ensure consumers are keeping up

---

### 4. Performance Analysis Dashboard
**File:** `performance-analysis-dashboard.json`  
**Description:** Deep performance analysis and optimization metrics

**Panels:**
- Kafka API Request Latency (All Percentiles)
- Request Rate by API Key
- Throughput (Bytes/sec)
- Message Rate (Messages/sec)
- Memory Usage
- Concurrency Metrics
- Garbage Collection Pause Time
- Garbage Collection Frequency
- Storage I/O Operations
- Replication Lag
- Error Rate by API and Error Code

**Use Cases:**
- Performance tuning
- Latency analysis
- Identify memory issues
- Track GC impact
- Diagnose errors

---

## Installation

### Prerequisites
- Grafana 9.0+ installed
- Prometheus data source configured
- Takhin metrics endpoint accessible at `http://localhost:9090/metrics`

### Step 1: Configure Prometheus Data Source

1. Open Grafana UI
2. Navigate to **Configuration** → **Data Sources**
3. Click **Add data source**
4. Select **Prometheus**
5. Configure:
   - **Name:** `Prometheus`
   - **URL:** `http://localhost:9090` (or your Prometheus server)
   - **Access:** `Server` (default)
6. Click **Save & Test**

### Step 2: Import Dashboards

#### Option A: Via Grafana UI

1. Navigate to **Dashboards** → **Import**
2. Click **Upload JSON file**
3. Select one of the dashboard JSON files
4. Select **Prometheus** as the data source
5. Click **Import**

Repeat for all four dashboards.

#### Option B: Via API

```bash
# Set your Grafana URL and API key
GRAFANA_URL="http://localhost:3000"
GRAFANA_API_KEY="your-api-key"

# Import all dashboards
for dashboard in cluster-overview topic-monitoring consumer-group performance-analysis; do
  curl -X POST "${GRAFANA_URL}/api/dashboards/db" \
    -H "Authorization: Bearer ${GRAFANA_API_KEY}" \
    -H "Content-Type: application/json" \
    -d @"${dashboard}-dashboard.json"
done
```

#### Option C: Provisioning (Recommended for Production)

Create a provisioning file at `/etc/grafana/provisioning/dashboards/takhin.yaml`:

```yaml
apiVersion: 1

providers:
  - name: 'Takhin Dashboards'
    orgId: 1
    folder: 'Takhin'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards/takhin
```

Copy dashboard files:

```bash
sudo mkdir -p /var/lib/grafana/dashboards/takhin
sudo cp *.json /var/lib/grafana/dashboards/takhin/
sudo chown -R grafana:grafana /var/lib/grafana/dashboards/takhin
sudo systemctl restart grafana-server
```

---

## Configuration

### Prometheus Scrape Configuration

Ensure Prometheus is scraping Takhin metrics:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'takhin'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
    scrape_timeout: 10s
```

### Takhin Metrics Configuration

Verify metrics are enabled in `configs/takhin.yaml`:

```yaml
metrics:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  path: "/metrics"
```

---

## Dashboard Features

### Auto-Refresh
All dashboards refresh every **10 seconds** by default. You can change this:
1. Click the refresh dropdown in the top-right
2. Select your preferred interval (5s, 30s, 1m, etc.)

### Time Range
Default time range is **Last 1 hour**. Change via:
1. Click the time range picker in the top-right
2. Select preset range or custom range

### Variables
Topic and Consumer Group dashboards support filtering:
- **Topic Dashboard:** Filter by topic name (multi-select)
- **Consumer Group Dashboard:** Filter by group ID (multi-select)
- Select "All" to see all entities

### Legends
Most panels show legends with:
- **Last value:** Current value
- **Max value:** Maximum in time range
- **Mean value:** Average over time range

Click legend items to hide/show specific series.

---

## Common Queries

### High Latency Detection
```promql
# Find APIs with p99 latency > 100ms
histogram_quantile(0.99, sum(rate(takhin_kafka_request_duration_seconds_bucket[5m])) by (api_key, le)) > 0.1
```

### Consumer Lag Alert
```promql
# Consumer lag > 10,000 offsets
sum(takhin_consumer_group_lag_offsets) by (group_id, topic) > 10000
```

### High Error Rate
```promql
# Error rate > 1 error/sec
sum(rate(takhin_kafka_request_errors_total[5m])) > 1
```

### Memory Pressure
```promql
# Heap usage > 80% of system memory
takhin_go_mem_heap_inuse_bytes / takhin_go_mem_sys_bytes > 0.8
```

---

## Alerting

### Example Alert Rules

Create `/etc/prometheus/rules/takhin.yml`:

```yaml
groups:
  - name: takhin_alerts
    interval: 30s
    rules:
      - alert: HighConsumerLag
        expr: sum(takhin_consumer_group_lag_offsets) by (group_id, topic) > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High consumer lag detected"
          description: "Consumer group {{ $labels.group_id }} has {{ $value }} messages lag on topic {{ $labels.topic }}"

      - alert: HighReplicationLag
        expr: max(takhin_replication_lag_offsets) > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High replication lag detected"
          description: "Replication lag is {{ $value }} offsets"

      - alert: HighErrorRate
        expr: sum(rate(takhin_kafka_request_errors_total[5m])) > 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      - alert: HighMemoryUsage
        expr: takhin_go_mem_heap_inuse_bytes > 2e9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value | humanize }}B"

      - alert: FrequentRebalances
        expr: increase(takhin_consumer_group_rebalances_total[1h]) > 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Frequent consumer group rebalances"
          description: "Consumer group {{ $labels.group_id }} has rebalanced {{ $value }} times in the last hour"
```

Reload Prometheus configuration:
```bash
curl -X POST http://localhost:9090/-/reload
```

---

## Troubleshooting

### Dashboard Shows "No Data"

**Check Metrics Endpoint:**
```bash
curl http://localhost:9090/metrics | grep takhin
```

**Verify Prometheus Scraping:**
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets | jq
```

**Check Takhin Logs:**
```bash
# Look for metrics server startup
tail -f takhin.log | grep metrics
```

### Variables Not Populating

**Ensure metrics exist:**
```bash
# Check for topic metrics
curl -s http://localhost:9090/api/v1/label/topic/values

# Check for consumer group metrics
curl -s http://localhost:9090/api/v1/label/group_id/values
```

**Wait for metrics collection:** Some metrics are collected periodically (30s interval). Wait at least one collection cycle.

### High Cardinality Warnings

If you see high cardinality warnings in Prometheus:
- Limit the number of topics/partitions
- Use recording rules for frequently-queried aggregations
- Adjust Prometheus `--storage.tsdb.max-series-per-metric` if needed

---

## Customization

### Adding Custom Panels

1. Open dashboard in edit mode
2. Click **Add panel**
3. Select **Add a new panel**
4. Enter PromQL query
5. Configure visualization type
6. Save panel

### Creating Recording Rules

For expensive queries used in multiple panels, create recording rules:

```yaml
# /etc/prometheus/rules/takhin_recordings.yml
groups:
  - name: takhin_recordings
    interval: 30s
    rules:
      - record: takhin:api_request_rate:5m
        expr: sum(rate(takhin_kafka_requests_total[5m])) by (api_key)
      
      - record: takhin:consumer_lag:total
        expr: sum(takhin_consumer_group_lag_offsets) by (group_id)
      
      - record: takhin:produce_throughput:5m
        expr: sum(rate(takhin_produce_bytes_total[5m]))
```

Use in dashboards:
```promql
# Instead of: sum(rate(takhin_kafka_requests_total[5m])) by (api_key)
# Use: takhin:api_request_rate:5m
```

---

## Best Practices

### Performance
- Use recording rules for complex queries
- Limit time range for historical analysis
- Use dashboard variables to reduce cardinality
- Enable query caching in Grafana

### Monitoring
- Set up alerts for critical metrics
- Review dashboards regularly
- Adjust thresholds based on your workload
- Create custom dashboards for specific use cases

### Maintenance
- Keep dashboards in version control
- Document custom modifications
- Test dashboards before production deployment
- Update dashboards when metrics change

---

## Metrics Reference

For detailed information about available metrics, see:
- [Metrics Documentation](../../metrics.md)
- [Prometheus Metrics Reference](../../METRICS_QUICK_REF.md)

---

## Support

For issues or questions:
1. Check [Takhin Documentation](../../README.md)
2. Review [Troubleshooting Guide](../../docs/deployment/05-troubleshooting.md)
3. Open an issue on GitHub

---

## Version History

- **v1.0** (2026-01-06) - Initial release with 4 dashboards
  - Cluster Overview
  - Topic Monitoring
  - Consumer Group Monitoring
  - Performance Analysis
