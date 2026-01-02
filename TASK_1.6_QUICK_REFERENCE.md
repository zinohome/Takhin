# Task 1.6 Quick Reference - Replication Lag Monitoring

## Metrics Summary

### Key Metrics

| Metric | Type | Description | Critical Threshold |
|--------|------|-------------|-------------------|
| `takhin_replication_lag_offsets` | Gauge | Offset lag per follower | > 10000 offsets |
| `takhin_replication_lag_time_ms` | Gauge | Time since last fetch | > 10000 ms |
| `takhin_replication_isr_size` | Gauge | In-sync replica count | < replicas_total |
| `takhin_replication_under_replicated` | Gauge | Under-replicated flag | == 1 |
| `takhin_replication_isr_shrinks_total` | Counter | ISR shrink events | rate > 0.1/s |
| `takhin_replication_isr_expands_total` | Counter | ISR expand events | - |

## Essential Queries

### Check Health Status
```promql
# Under-replicated partitions
takhin_replication_under_replicated == 1

# Max lag across all partitions
max(takhin_replication_lag_offsets)

# ISR health
takhin_replication_isr_size < takhin_replication_replicas_total
```

### Monitor Trends
```promql
# Replication lag growth rate
rate(takhin_replication_lag_offsets[5m])

# ISR churn
sum(rate(takhin_replication_isr_shrinks_total[5m]) + rate(takhin_replication_isr_expands_total[5m]))

# Replication throughput
rate(takhin_replication_bytes_out_total[5m]) / 1024 / 1024
```

## Critical Alerts

### Alert: Under-Replicated Partitions
```yaml
expr: takhin_replication_under_replicated == 1
for: 5m
severity: critical
```

### Alert: High Replication Lag
```yaml
expr: takhin_replication_lag_offsets > 10000
for: 10m
severity: critical
```

### Alert: Frequent ISR Shrinks
```yaml
expr: rate(takhin_replication_isr_shrinks_total[30m]) > 0.1
for: 15m
severity: warning
```

## Troubleshooting Checklist

### High Replication Lag
- [ ] Check network latency between brokers
- [ ] Verify follower disk I/O performance
- [ ] Compare with producer throughput
- [ ] Check follower CPU/memory usage
- [ ] Review `takhin_replication_fetch_latency_seconds`

### Frequent ISR Changes
- [ ] Check network stability
- [ ] Review GC pause times (`takhin_go_gc_pause_seconds`)
- [ ] Verify disk consistency
- [ ] Consider increasing `replica_lag_time_max_ms`
- [ ] Check broker logs for ISR change events

### Under-Replicated Partitions
- [ ] Verify all brokers are running
- [ ] Check follower connectivity
- [ ] Review follower replication lag
- [ ] Check disk space on followers
- [ ] Verify firewall rules

## Configuration

### Replica Lag Threshold
```yaml
# configs/takhin.yaml
replication:
  replica_lag_time_max_ms: 10000
```

```bash
export TAKHIN_REPLICATION_REPLICA_LAG_TIME_MAX_MS=10000
```

### Metrics Collection
```yaml
metrics:
  enabled: true
  port: 9090
  path: /metrics
  collection_interval: 30s
```

## Testing

### Run Tests
```bash
cd backend
go test -v ./pkg/metrics -run TestReplicationLag
go test -v ./pkg/metrics -run TestISR
go test -v ./pkg/metrics -run TestCollector
```

### Manual Testing
```bash
# Start Takhin with metrics enabled
./takhin -config configs/takhin.yaml

# Check metrics endpoint
curl http://localhost:9090/metrics | grep replication

# Query specific metric
curl -s http://localhost:9090/metrics | grep takhin_replication_lag_offsets
```

## Integration

### Helper Functions
```go
import "github.com/takhin-data/takhin/pkg/metrics"

// Update follower lag
metrics.UpdateReplicationMetrics(topic, partition, followerID, lag, isrSize, replicasTotal)

// Update lag time
metrics.UpdateReplicationLagTime(topic, partition, followerID, lagMs)

// Record ISR changes
metrics.RecordISRShrink(topic, partition)
metrics.RecordISRExpand(topic, partition)

// Track replication traffic
metrics.RecordReplicationBytesOut(topic, partition, bytes)
```

### Automatic Collection
Metrics are automatically collected by `metrics.Collector`:
```go
collector := metrics.NewCollector(topicManager, coordinator, 30*time.Second)
collector.Start()
defer collector.Stop()
```

## Files Modified/Created

### Modified
- `backend/pkg/metrics/metrics.go` - 7 new metrics
- `backend/pkg/metrics/helpers.go` - 5 new helpers
- `backend/pkg/metrics/collector.go` - ISR tracking
- `backend/pkg/storage/topic/manager.go` - GetLastFetchTime()

### Created
- `backend/pkg/metrics/replication_lag_test.go` - Test suite
- `docs/monitoring/replication-lag-monitoring.md` - Full guide

## Key Features

✅ **Follower Lag Tracking**
- Offset-based lag per follower
- Time-based lag (last fetch time)
- Per-partition granularity

✅ **ISR Change Monitoring**
- Shrink/expand event counters
- ISR size tracking
- Under-replicated detection

✅ **Traffic Metrics**
- Inbound replication bytes
- Outbound replication bytes
- Fetch request latency

✅ **Production Ready**
- Comprehensive tests (23 test cases)
- Thread-safe implementation
- Low performance overhead
- Clear documentation

## Quick Commands

```bash
# Build
cd backend && go build ./cmd/takhin

# Test
cd backend && go test ./pkg/metrics

# Run with metrics
./takhin -config configs/takhin.yaml &

# Check metrics
curl localhost:9090/metrics | grep replication

# Stop
pkill takhin
```

## Documentation

Full guide: `docs/monitoring/replication-lag-monitoring.md`

Sections:
- Metrics reference with examples
- Common PromQL queries
- 6 alert rules (2 critical, 4 warning)
- 4 Grafana dashboard panels
- Troubleshooting guide
- Configuration reference
- Best practices

## Status

✅ Implementation complete
✅ All tests passing
✅ Documentation complete
✅ Production ready
