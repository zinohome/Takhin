# Takhin Grafana Dashboards - Quick Reference

## 🎯 Quick Start

### Import Dashboards
```bash
# Navigate to Grafana → Dashboards → Import → Upload JSON file
# Import all 4 dashboard files from docs/examples/grafana/
```

### Access Dashboards
- **Cluster Overview:** `http://localhost:3000/d/takhin-cluster-overview`
- **Topic Monitoring:** `http://localhost:3000/d/takhin-topic-monitoring`
- **Consumer Group:** `http://localhost:3000/d/takhin-consumer-group`
- **Performance Analysis:** `http://localhost:3000/d/takhin-performance-analysis`

---

## 📊 Dashboard Summary

| Dashboard | Panels | Variables | Primary Use |
|-----------|--------|-----------|-------------|
| **Cluster Overview** | 12 | None | System health, overall performance |
| **Topic Monitoring** | 11 | $topic | Topic-specific metrics |
| **Consumer Group** | 10 | $group | Consumer lag, rebalances |
| **Performance Analysis** | 11 | None | Latency, GC, memory analysis |

---

## 🔍 Key Metrics at a Glance

### Cluster Overview
```
Active Connections | Request Rate | Error Rate | Memory Usage
-----------------------------------------------------------
Request Rate by API | Request Latency (p99, p95, p50)
Network Throughput | Memory Details | Goroutines | GC Stats
```

### Topic Monitoring
```
Produce Rate | Fetch Rate | Produce Throughput | Fetch Throughput
-----------------------------------------------------------------
Produce Latency | Fetch Latency | Disk Usage | Log Segments
Log End Offset | Storage I/O Ops | Storage I/O Errors
```

### Consumer Group
```
Total Lag | Total Members | Commit Rate | Rebalances (1h)
-----------------------------------------------------------
Lag by Topic | Group Members | Rebalance Events | Commit Rate
Group State | Lag Details Table (sortable)
```

### Performance Analysis
```
API Latency (all percentiles) | Request Rate by API
Throughput | Message Rate | Memory Usage | Concurrency
GC Pause | GC Frequency | Storage I/O | Replication Lag
Error Rate by API and Code
```

---

## 🚀 Common Operations

### Check Cluster Health
1. Open **Cluster Overview** dashboard
2. Check top row: Connections, Request Rate, Error Rate, Memory
3. Green = healthy, Yellow = warning, Red = critical

### Investigate Slow Topics
1. Open **Topic Monitoring** dashboard
2. Select topic from `$topic` dropdown (or "All")
3. Check **Latency Percentiles** panel
4. p99 > 100ms = investigate

### Monitor Consumer Lag
1. Open **Consumer Group** dashboard
2. Select group from `$group` dropdown
3. Check **Total Consumer Lag** stat panel
4. >10,000 offsets = warning

### Analyze Performance Issues
1. Open **Performance Analysis** dashboard
2. Check **API Latency** for slow APIs
3. Check **Memory Usage** for leaks
4. Check **GC Pause Time** for GC pressure
5. Check **Error Rate** for failures

---

## ⚠️ Alert Thresholds

| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| **Consumer Lag** | >1,000 | >10,000 | Scale consumers, check processing |
| **Error Rate** | >1/sec | >10/sec | Check logs, investigate errors |
| **Memory Usage** | >1GB | >2GB | Investigate leaks, restart if needed |
| **Replication Lag** | >100 | >1,000 | Check network, disk I/O |
| **Rebalances** | >3/hour | >5/hour | Investigate group stability |
| **GC Pause** | >10ms | >50ms | Tune GC, check memory |
| **p99 Latency** | >100ms | >500ms | Investigate slow operations |

---

## 📈 PromQL Quick Reference

### Request Rate
```promql
rate(takhin_kafka_requests_total[5m])
```

### p99 Latency
```promql
histogram_quantile(0.99, rate(takhin_kafka_request_duration_seconds_bucket[5m]))
```

### Consumer Lag
```promql
sum(takhin_consumer_group_lag_offsets) by (group_id, topic)
```

### Error Rate
```promql
rate(takhin_kafka_request_errors_total[5m])
```

### Memory Usage
```promql
takhin_go_mem_heap_inuse_bytes
```

### Throughput
```promql
rate(takhin_produce_bytes_total[5m])
```

---

## 🛠️ Troubleshooting

### "No Data" in Dashboard
```bash
# Check Prometheus is scraping
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.job=="takhin")'

# Check Takhin metrics endpoint
curl http://localhost:9090/metrics | grep takhin | head -20

# Verify Takhin metrics enabled
grep -A 3 "^metrics:" configs/takhin.yaml
```

### Variables Not Populating
```bash
# Check label values exist
curl -s http://localhost:9090/api/v1/label/topic/values | jq
curl -s http://localhost:9090/api/v1/label/group_id/values | jq

# Wait 30+ seconds for metric collection cycle
```

### Slow Dashboard Loading
- Reduce time range (e.g., 1h → 15m)
- Disable auto-refresh temporarily
- Use recording rules for expensive queries

---

## 🔧 Configuration

### Prometheus Scrape
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'takhin'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
```

### Takhin Metrics
```yaml
# configs/takhin.yaml
metrics:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  path: "/metrics"
```

### Grafana Data Source
```
Name: Prometheus
Type: Prometheus
URL: http://localhost:9090
Access: Server (default)
```

---

## 📦 Installation Methods

### Method 1: Manual Import (Fastest)
1. Grafana UI → Dashboards → Import
2. Upload JSON file
3. Select Prometheus data source
4. Import

### Method 2: API Import (Automated)
```bash
GRAFANA_URL="http://localhost:3000"
GRAFANA_API_KEY="your-api-key"

curl -X POST "${GRAFANA_URL}/api/dashboards/db" \
  -H "Authorization: Bearer ${GRAFANA_API_KEY}" \
  -H "Content-Type: application/json" \
  -d @"cluster-overview-dashboard.json"
```

### Method 3: Provisioning (Production)
```yaml
# /etc/grafana/provisioning/dashboards/takhin.yaml
apiVersion: 1
providers:
  - name: 'Takhin Dashboards'
    folder: 'Takhin'
    type: file
    options:
      path: /var/lib/grafana/dashboards/takhin
```

---

## 🎨 Customization Tips

### Add Custom Panel
1. Edit dashboard
2. Add panel → Add a new panel
3. Enter PromQL query
4. Select visualization type
5. Save

### Adjust Thresholds
1. Edit panel
2. Field → Thresholds
3. Add/modify threshold values
4. Set colors (green/yellow/red)

### Create Alert
1. Edit panel
2. Alert tab → Create alert rule
3. Set condition (e.g., `value > 1000`)
4. Configure notification channel
5. Save

---

## 📚 Documentation Links

- **Full Documentation:** [docs/examples/grafana/README.md](README.md)
- **Metrics Reference:** [docs/metrics.md](../../metrics.md)
- **Task Completion:** [TASK_5.4_GRAFANA_COMPLETION.md](../../TASK_5.4_GRAFANA_COMPLETION.md)
- **Prometheus Guide:** https://prometheus.io/docs/
- **Grafana Guide:** https://grafana.com/docs/

---

## 🔗 Related Tasks

- **Task 5.1:** Prometheus Metrics Implementation
- **Task 5.2:** Health Check API
- **Task 5.3:** Logging and Tracing (if implemented)

---

## 💡 Pro Tips

1. **Use Variables:** Filter by topic/group to reduce noise
2. **Set Alerts:** Don't rely on manual dashboard checks
3. **Recording Rules:** Speed up expensive queries
4. **Time Range:** Adjust based on investigation needs
5. **Legend Sorting:** Click column headers to sort
6. **Panel Linking:** Use drill-down links between dashboards
7. **Refresh Interval:** Balance between freshness and load
8. **Export Data:** Use "Inspect" → "Data" for CSV export

---

**Version:** 1.0  
**Last Updated:** 2026-01-06  
**Dashboard Count:** 4  
**Panel Count:** 44
