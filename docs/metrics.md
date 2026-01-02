# Takhin Prometheus Metrics

This document describes all Prometheus metrics exposed by Takhin.

## Metrics Endpoint

By default, metrics are exposed at `http://localhost:9090/metrics`

Configure in `configs/takhin.yaml`:
```yaml
metrics:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  path: "/metrics"
```

## Metric Categories

### Connection Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `takhin_connections_active` | Gauge | Number of active connections |
| `takhin_connections_total` | Counter | Total number of connections since startup |
| `takhin_bytes_sent_total` | Counter | Total bytes sent to clients |
| `takhin_bytes_received_total` | Counter | Total bytes received from clients |

### Kafka API Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `takhin_kafka_requests_total` | Counter | `api_key`, `version` | Total Kafka API requests by API key and version |
| `takhin_kafka_request_duration_seconds` | Histogram | `api_key` | Request processing duration distribution |
| `takhin_kafka_request_errors_total` | Counter | `api_key`, `error_code` | Total API errors by type |

#### API Keys Reference
- `0` = Produce
- `1` = Fetch
- `2` = ListOffsets
- `3` = Metadata
- `8` = OffsetCommit
- `9` = OffsetFetch
- `10` = FindCoordinator
- `11` = JoinGroup
- `12` = Heartbeat
- `13` = LeaveGroup
- `14` = SyncGroup
- `18` = ApiVersions
- `32` = DescribeConfigs
- `33` = AlterConfigs

### Producer Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `takhin_produce_requests_total` | Counter | `topic` | Total produce requests per topic |
| `takhin_produce_messages_total` | Counter | `topic`, `partition` | Total messages produced |
| `takhin_produce_bytes_total` | Counter | `topic` | Total bytes produced per topic |
| `takhin_produce_latency_seconds` | Histogram | `topic` | Produce request latency distribution |

### Consumer Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `takhin_fetch_requests_total` | Counter | `topic` | Total fetch requests per topic |
| `takhin_fetch_messages_total` | Counter | `topic`, `partition` | Total messages fetched |
| `takhin_fetch_bytes_total` | Counter | `topic` | Total bytes fetched per topic |
| `takhin_fetch_latency_seconds` | Histogram | `topic` | Fetch request latency distribution |

### Storage Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `takhin_storage_disk_usage_bytes` | Gauge | `topic`, `partition` | Disk usage per partition |
| `takhin_storage_log_segments` | Gauge | `topic`, `partition` | Number of log segments |
| `takhin_storage_log_end_offset` | Gauge | `topic`, `partition` | Log end offset (high water mark) |
| `takhin_storage_active_segment_bytes` | Gauge | `topic`, `partition` | Active segment size |
| `takhin_storage_io_reads_total` | Counter | `topic` | Total read operations |
| `takhin_storage_io_writes_total` | Counter | `topic` | Total write operations |
| `takhin_storage_io_errors_total` | Counter | `topic`, `operation` | I/O errors by operation type |

### Replication Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `takhin_replication_lag_offsets` | Gauge | `topic`, `partition`, `follower_id` | Replication lag in offsets |
| `takhin_replication_isr_size` | Gauge | `topic`, `partition` | Number of in-sync replicas (ISR) |
| `takhin_replication_replicas_total` | Gauge | `topic`, `partition` | Total number of replicas |
| `takhin_replication_fetch_requests_total` | Counter | `follower_id` | Replication fetch requests |
| `takhin_replication_fetch_latency_seconds` | Histogram | `follower_id` | Replication fetch latency |

### Consumer Group Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `takhin_consumer_group_members` | Gauge | `group_id` | Number of members in group |
| `takhin_consumer_group_state` | Gauge | `group_id`, `state` | Consumer group state (1=active, 0=inactive) |
| `takhin_consumer_group_rebalances_total` | Counter | `group_id` | Total rebalance events |
| `takhin_consumer_group_lag_offsets` | Gauge | `group_id`, `topic`, `partition` | Consumer group lag |
| `takhin_consumer_group_commits_total` | Counter | `group_id`, `topic` | Total offset commits |

#### Consumer Group States
- `Dead` = Group is being deleted
- `Empty` = No members
- `PreparingRebalance` = Rebalance initiated
- `CompletingRebalance` = Waiting for assignments
- `Stable` = All members have assignments

### Go Runtime Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `takhin_go_goroutines` | Gauge | Number of goroutines |
| `takhin_go_threads` | Gauge | Number of OS threads |
| `takhin_go_mem_alloc_bytes` | Gauge | Bytes of allocated heap objects |
| `takhin_go_mem_total_alloc_bytes` | Counter | Cumulative bytes allocated |
| `takhin_go_mem_sys_bytes` | Gauge | Total memory obtained from OS |
| `takhin_go_mem_heap_alloc_bytes` | Gauge | Bytes in allocated heap spans |
| `takhin_go_mem_heap_idle_bytes` | Gauge | Bytes in idle heap spans |
| `takhin_go_mem_heap_inuse_bytes` | Gauge | Bytes in in-use heap spans |
| `takhin_go_gc_pause_seconds` | Histogram | GC pause duration |
| `takhin_go_gc_total` | Counter | Total number of GC cycles |

## Using Metrics

### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'takhin'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
```

### Example Queries

#### Request Rate by API
```promql
rate(takhin_kafka_requests_total[5m])
```

#### P99 Produce Latency
```promql
histogram_quantile(0.99, rate(takhin_produce_latency_seconds_bucket[5m]))
```

#### Storage Usage by Topic
```promql
sum by (topic) (takhin_storage_disk_usage_bytes)
```

#### Replication Lag
```promql
takhin_replication_lag_offsets > 100
```

#### Consumer Group Lag
```promql
sum by (group_id, topic) (takhin_consumer_group_lag_offsets)
```

#### Memory Usage Trend
```promql
takhin_go_mem_heap_inuse_bytes
```

### Grafana Dashboards

Example dashboard panels:

1. **Request Rate**: `rate(takhin_kafka_requests_total[5m])`
2. **Error Rate**: `rate(takhin_kafka_request_errors_total[5m])`
3. **Produce Throughput**: `rate(takhin_produce_bytes_total[5m])`
4. **Fetch Throughput**: `rate(takhin_fetch_bytes_total[5m])`
5. **Replication Lag**: `max by (topic) (takhin_replication_lag_offsets)`
6. **Consumer Lag**: `sum by (group_id) (takhin_consumer_group_lag_offsets)`
7. **Memory Usage**: `takhin_go_mem_heap_inuse_bytes`
8. **GC Pauses**: `rate(takhin_go_gc_pause_seconds_sum[5m])`

## Metric Collection

Metrics are collected from multiple sources:

1. **Real-time metrics**: Updated immediately when events occur
   - Connection metrics
   - Request/response metrics
   - Error counters

2. **Periodic metrics**: Collected every 30 seconds (configurable)
   - Storage metrics (disk usage, segments)
   - Replication metrics (lag, ISR)
   - Consumer group metrics (lag, state)

3. **Runtime metrics**: Collected every 15 seconds
   - Go runtime stats
   - Memory usage
   - GC statistics

## Performance Impact

- Metric collection is lightweight and async
- Storage/replication metrics collection: ~30s interval
- Runtime metrics collection: ~15s interval
- No blocking of main request path
- Minimal memory overhead (~50MB for 1000s of time series)

## Alerting Examples

### High Replication Lag
```yaml
- alert: HighReplicationLag
  expr: takhin_replication_lag_offsets > 1000
  for: 5m
  annotations:
    summary: "Replication lag is high"
```

### Consumer Group Lag
```yaml
- alert: ConsumerGroupLag
  expr: takhin_consumer_group_lag_offsets > 10000
  for: 10m
  annotations:
    summary: "Consumer group {{ $labels.group_id }} is lagging"
```

### High Error Rate
```yaml
- alert: HighErrorRate
  expr: rate(takhin_kafka_request_errors_total[5m]) > 10
  for: 5m
  annotations:
    summary: "High error rate detected"
```

### Memory Usage
```yaml
- alert: HighMemoryUsage
  expr: takhin_go_mem_heap_inuse_bytes > 2e9
  for: 10m
  annotations:
    summary: "High memory usage detected"
```

## Troubleshooting

### Metrics not appearing
1. Check metrics server is enabled in config
2. Verify port is not blocked by firewall
3. Check logs for metrics server errors

### High cardinality warnings
- Consumer group metrics can create many time series
- Use recording rules for high-cardinality aggregations
- Consider label filtering in Prometheus config

### Missing metrics
- Storage metrics require topic creation
- Replication metrics require replica configuration
- Consumer group metrics require active consumers
