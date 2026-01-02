# Prometheus Metrics Quick Reference

## 📊 Metrics Overview

| Category | Metrics Count | Key Metrics |
|----------|--------------|-------------|
| **Kafka API** | 3 | requests_total, request_duration_seconds, request_errors_total |
| **Producer** | 4 | produce_requests_total, produce_messages_total, produce_bytes_total, produce_latency_seconds |
| **Consumer** | 4 | fetch_requests_total, fetch_messages_total, fetch_bytes_total, fetch_latency_seconds |
| **Storage** | 7 | disk_usage_bytes, log_segments, log_end_offset, io_reads_total, io_writes_total |
| **Replication** | 5 | lag_offsets, isr_size, replicas_total, fetch_requests_total |
| **Consumer Groups** | 5 | members, state, rebalances_total, lag_offsets, commits_total |
| **Runtime** | 10 | goroutines, threads, mem_alloc_bytes, gc_pause_seconds, gc_total |
| **Connection** | 4 | connections_active, connections_total, bytes_sent_total, bytes_received_total |

**Total: 42 unique metrics**

## 🚀 Quick Start

### 1. Enable Metrics
```yaml
# configs/takhin.yaml
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

### 2. Start Collector
```go
collector := metrics.NewCollector(topicManager, coordinator, 30*time.Second)
collector.Start()
defer collector.Stop()
```

### 3. Record Metrics
```go
// Kafka API
metrics.RecordKafkaRequest(apiKey, version, duration, errorCode)

// Producer
metrics.RecordProduceRequest(topic, partition, msgCount, bytes, duration)

// Consumer
metrics.RecordFetchRequest(topic, partition, msgCount, bytes, duration)

// Consumer Group
metrics.UpdateConsumerGroupLag(groupID, topic, partition, lag)
```

## 📈 Top Queries

```promql
# Request Rate (req/sec)
rate(takhin_kafka_requests_total[5m])

# P99 Produce Latency
histogram_quantile(0.99, rate(takhin_produce_latency_seconds_bucket[5m]))

# Replication Lag
max by (topic, partition) (takhin_replication_lag_offsets)

# Consumer Lag
sum by (group_id) (takhin_consumer_group_lag_offsets)

# Error Rate
rate(takhin_kafka_request_errors_total[5m])

# Memory Usage
takhin_go_mem_heap_inuse_bytes / 1024 / 1024

# Top Topics by Traffic
topk(10, rate(takhin_produce_bytes_total[5m]))
```

## 🔔 Alert Templates

```yaml
# High Lag
- alert: HighReplicationLag
  expr: takhin_replication_lag_offsets > 1000
  for: 5m

- alert: HighConsumerLag
  expr: takhin_consumer_group_lag_offsets > 10000
  for: 10m

# Errors
- alert: HighErrorRate
  expr: rate(takhin_kafka_request_errors_total[5m]) > 10
  for: 5m

# Resources
- alert: HighMemory
  expr: takhin_go_mem_heap_inuse_bytes > 2e9
  for: 10m
```

## 🛠️ Troubleshooting

| Issue | Solution |
|-------|----------|
| Metrics not appearing | Check `metrics.enabled: true` in config |
| High cardinality | Use recording rules, filter labels |
| Missing storage metrics | Ensure topics are created |
| Missing replication metrics | Configure replicas in topic metadata |
| Missing consumer metrics | Start consumer groups |

## 📝 Files

- `backend/pkg/metrics/metrics.go` - Metric definitions
- `backend/pkg/metrics/helpers.go` - Recording functions
- `backend/pkg/metrics/collector.go` - Periodic collector
- `docs/metrics.md` - Full documentation

## 🎯 Key Features

✅ **42 comprehensive metrics** across all components  
✅ **Zero-copy counters** for minimal overhead  
✅ **Periodic collection** (30s storage, 15s runtime)  
✅ **Thread-safe** with proper locking  
✅ **Optimized buckets** for latency histograms  
✅ **Low cardinality** labels to prevent explosion  
✅ **Backward compatible** with legacy metrics  

## 📊 Coverage

| Component | Status | Metrics |
|-----------|--------|---------|
| Kafka API | ✅ 100% | Requests, latency, errors |
| Storage | ✅ 100% | Disk, I/O, offsets |
| Replication | ✅ 100% | Lag, ISR, fetches |
| Consumer Groups | ✅ 100% | Members, lag, commits |
| Runtime | ✅ 100% | Memory, GC, goroutines |
