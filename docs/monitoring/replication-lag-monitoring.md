# Replication Lag Monitoring Guide

## Overview

Takhin provides comprehensive replication lag monitoring through Prometheus metrics. This guide covers the metrics available, how to interpret them, and recommended alerting rules.

## Available Metrics

### Follower Lag Metrics

#### `takhin_replication_lag_offsets`
**Type:** Gauge  
**Labels:** `topic`, `partition`, `follower_id`  
**Description:** Replication lag in offsets between leader and follower

This metric shows how many offsets behind the leader each follower is. A value of 0 indicates the follower is fully caught up.

**Example:**
```promql
takhin_replication_lag_offsets{topic="events",partition="0",follower_id="2"} 150
```

#### `takhin_replication_lag_time_ms`
**Type:** Gauge  
**Labels:** `topic`, `partition`, `follower_id`  
**Description:** Time in milliseconds since last follower fetch

This metric tracks how long ago the follower last fetched from the leader. Values exceeding `replica.lag.time.max.ms` (default 10000ms) indicate the follower may be removed from ISR.

**Example:**
```promql
takhin_replication_lag_time_ms{topic="events",partition="0",follower_id="2"} 2500
```

### ISR (In-Sync Replicas) Metrics

#### `takhin_replication_isr_size`
**Type:** Gauge  
**Labels:** `topic`, `partition`  
**Description:** Number of in-sync replicas for a partition

Tracks the current ISR size. Should equal `takhin_replication_replicas_total` in healthy state.

**Example:**
```promql
takhin_replication_isr_size{topic="events",partition="0"} 3
```

#### `takhin_replication_isr_shrinks_total`
**Type:** Counter  
**Labels:** `topic`, `partition`  
**Description:** Total number of ISR shrink events

Increments when a follower is removed from ISR. Frequent shrinks indicate replication issues.

**Example:**
```promql
rate(takhin_replication_isr_shrinks_total{topic="events"}[5m]) > 0.1
```

#### `takhin_replication_isr_expands_total`
**Type:** Counter  
**Labels:** `topic`, `partition`  
**Description:** Total number of ISR expand events

Increments when a follower rejoins ISR after catching up. Normal after maintenance or temporary network issues.

**Example:**
```promql
rate(takhin_replication_isr_expands_total{topic="events"}[5m])
```

### Replica Health Metrics

#### `takhin_replication_replicas_total`
**Type:** Gauge  
**Labels:** `topic`, `partition`  
**Description:** Total number of replicas configured for a partition

The target replication factor. Should be constant unless explicitly changed.

**Example:**
```promql
takhin_replication_replicas_total{topic="events",partition="0"} 3
```

#### `takhin_replication_under_replicated`
**Type:** Gauge  
**Labels:** `topic`, `partition`  
**Description:** Indicates if partition is under-replicated (1=yes, 0=no)

Set to 1 when `isr_size < replicas_total`, indicating data is at risk.

**Example:**
```promql
takhin_replication_under_replicated{topic="events",partition="0"} 1
```

### Replication Traffic Metrics

#### `takhin_replication_fetch_requests_total`
**Type:** Counter  
**Labels:** `follower_id`  
**Description:** Total number of replication fetch requests from follower

Tracks replication fetch activity per follower.

**Example:**
```promql
rate(takhin_replication_fetch_requests_total{follower_id="2"}[5m])
```

#### `takhin_replication_fetch_latency_seconds`
**Type:** Histogram  
**Labels:** `follower_id`  
**Description:** Replication fetch latency in seconds

Measures how long replication fetch requests take. High latency indicates network or disk issues.

**Example:**
```promql
histogram_quantile(0.99, rate(takhin_replication_fetch_latency_seconds_bucket{follower_id="2"}[5m]))
```

#### `takhin_replication_bytes_in_total`
**Type:** Counter  
**Labels:** `topic`, `partition`  
**Description:** Total bytes received from leader for replication

Tracks inbound replication traffic (from follower perspective).

**Example:**
```promql
rate(takhin_replication_bytes_in_total{topic="events"}[5m])
```

#### `takhin_replication_bytes_out_total`
**Type:** Counter  
**Labels:** `topic`, `partition`  
**Description:** Total bytes sent to followers for replication

Tracks outbound replication traffic (from leader perspective).

**Example:**
```promql
rate(takhin_replication_bytes_out_total{topic="events"}[5m])
```

## Common Queries

### Check Maximum Replication Lag
```promql
max by (topic, partition) (takhin_replication_lag_offsets)
```

### Under-Replicated Partitions
```promql
count by (topic) (takhin_replication_under_replicated == 1)
```

### ISR Churn Rate
```promql
sum by (topic) (
  rate(takhin_replication_isr_shrinks_total[5m]) + 
  rate(takhin_replication_isr_expands_total[5m])
)
```

### Replication Lag Time Exceeding Threshold
```promql
takhin_replication_lag_time_ms > 10000
```

### P99 Replication Fetch Latency
```promql
histogram_quantile(0.99, 
  sum by (follower_id, le) (
    rate(takhin_replication_fetch_latency_seconds_bucket[5m])
  )
)
```

### Replication Throughput by Topic
```promql
sum by (topic) (rate(takhin_replication_bytes_out_total[5m])) / 1024 / 1024
```

## Recommended Alert Rules

### Critical: Under-Replicated Partitions

```yaml
- alert: UnderReplicatedPartitions
  expr: takhin_replication_under_replicated == 1
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Partition {{ $labels.topic }}/{{ $labels.partition }} is under-replicated"
    description: "ISR size is less than replica count. Data is at risk."
```

### Critical: High Replication Lag

```yaml
- alert: HighReplicationLag
  expr: takhin_replication_lag_offsets > 10000
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "High replication lag on {{ $labels.topic }}/{{ $labels.partition }}"
    description: "Follower {{ $labels.follower_id }} is {{ $value }} offsets behind"
```

### Warning: ISR Shrinking Frequently

```yaml
- alert: FrequentISRShrinks
  expr: rate(takhin_replication_isr_shrinks_total[30m]) > 0.1
  for: 15m
  labels:
    severity: warning
  annotations:
    summary: "Frequent ISR shrinks on {{ $labels.topic }}/{{ $labels.partition }}"
    description: "ISR is shrinking at rate {{ $value }}/sec, indicating instability"
```

### Warning: Stale Replication Fetches

```yaml
- alert: StaleReplicationFetch
  expr: takhin_replication_lag_time_ms > 15000
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Stale replication fetch on {{ $labels.topic }}/{{ $labels.partition }}"
    description: "Follower {{ $labels.follower_id }} hasn't fetched in {{ $value }}ms"
```

### Warning: High Replication Fetch Latency

```yaml
- alert: HighReplicationFetchLatency
  expr: |
    histogram_quantile(0.99,
      sum by (follower_id, le) (
        rate(takhin_replication_fetch_latency_seconds_bucket[5m])
      )
    ) > 1.0
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "High replication fetch latency for follower {{ $labels.follower_id }}"
    description: "P99 latency is {{ $value }}s, may impact replication"
```

### Warning: No Replication Activity

```yaml
- alert: NoReplicationActivity
  expr: |
    rate(takhin_replication_fetch_requests_total[5m]) == 0
    and takhin_replication_replicas_total > 1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "No replication activity for follower {{ $labels.follower_id }}"
    description: "Follower is not fetching, may be down or partitioned"
```

## Grafana Dashboard

### Replication Overview Panel

```json
{
  "title": "Replication Lag by Topic",
  "targets": [
    {
      "expr": "max by (topic, partition, follower_id) (takhin_replication_lag_offsets)",
      "legendFormat": "{{topic}}/{{partition}} (follower {{follower_id}})"
    }
  ],
  "yAxis": {
    "label": "Offset Lag",
    "logBase": 10
  }
}
```

### ISR Health Panel

```json
{
  "title": "In-Sync Replicas",
  "targets": [
    {
      "expr": "takhin_replication_isr_size",
      "legendFormat": "{{topic}}/{{partition}} ISR"
    },
    {
      "expr": "takhin_replication_replicas_total",
      "legendFormat": "{{topic}}/{{partition}} Total"
    }
  ]
}
```

### ISR Changes Panel

```json
{
  "title": "ISR Changes Rate",
  "targets": [
    {
      "expr": "sum by (topic) (rate(takhin_replication_isr_shrinks_total[5m]))",
      "legendFormat": "{{topic}} Shrinks"
    },
    {
      "expr": "sum by (topic) (rate(takhin_replication_isr_expands_total[5m]))",
      "legendFormat": "{{topic}} Expands"
    }
  ]
}
```

### Replication Throughput Panel

```json
{
  "title": "Replication Throughput (MB/s)",
  "targets": [
    {
      "expr": "sum by (topic) (rate(takhin_replication_bytes_out_total[5m])) / 1024 / 1024",
      "legendFormat": "{{topic}}"
    }
  ]
}
```

## Troubleshooting

### High Replication Lag

**Symptoms:**
- `takhin_replication_lag_offsets` > 10000
- ISR shrinking

**Possible Causes:**
1. **Network congestion:** Check replication fetch latency
2. **Slow follower disk:** Check disk I/O metrics
3. **High producer throughput:** Compare with produce rate
4. **Under-provisioned followers:** Check CPU/memory

**Resolution:**
- Increase network bandwidth
- Upgrade follower disk (use SSD)
- Scale out (add more brokers)
- Tune `replica.fetch.max.bytes`

### Frequent ISR Changes

**Symptoms:**
- High `rate(takhin_replication_isr_shrinks_total)`
- High `rate(takhin_replication_isr_expands_total)`

**Possible Causes:**
1. **Flaky network:** Intermittent connectivity
2. **GC pauses:** Long pauses on followers
3. **Disk latency spikes:** Inconsistent disk performance
4. **Too aggressive timeout:** `replica.lag.time.max.ms` too low

**Resolution:**
- Investigate network stability
- Tune JVM GC settings (for Go: check `takhin_go_gc_pause_seconds`)
- Check disk health and consistency
- Increase `replica.lag.time.max.ms` (default 10s)

### Under-Replicated Partitions

**Symptoms:**
- `takhin_replication_under_replicated` == 1
- `takhin_replication_isr_size` < `takhin_replication_replicas_total`

**Possible Causes:**
1. **Follower down:** Broker crashed or unreachable
2. **Follower lagging:** Can't catch up to ISR threshold
3. **Network partition:** Follower isolated

**Resolution:**
- Check follower broker health
- Restart crashed brokers
- Investigate and resolve replication lag
- Verify network connectivity

## Configuration

### Replica Lag Time Max

Controls when followers are removed from ISR:

```yaml
# configs/takhin.yaml
replication:
  replica_lag_time_max_ms: 10000  # Default: 10 seconds
```

**Environment Variable:**
```bash
export TAKHIN_REPLICATION_REPLICA_LAG_TIME_MAX_MS=10000
```

### Metrics Collection Interval

Configure how often replication metrics are collected:

```yaml
# configs/takhin.yaml
metrics:
  enabled: true
  port: 9090
  path: /metrics
  collection_interval: 30s  # Default: 30 seconds
```

## Best Practices

1. **Monitor ISR size:** Alert when `isr_size < replicas_total` for more than 5 minutes
2. **Track lag trends:** Use rate() functions to identify growing lag
3. **Set appropriate timeouts:** Balance between false positives and data safety
4. **Correlate metrics:** Look at lag time AND offset lag together
5. **Dashboard visibility:** Include replication metrics on primary monitoring dashboard
6. **Regular testing:** Simulate follower failures to verify alerts work
7. **Capacity planning:** Monitor replication throughput trends for scaling decisions

## Related Metrics

- `takhin_storage_log_end_offset`: Leader LEO per partition
- `takhin_produce_messages_total`: Incoming write rate
- `takhin_storage_io_reads_total`: Disk read operations
- `takhin_go_goroutines`: Overall system health indicator

## References

- [Kafka Replication Design](https://kafka.apache.org/documentation/#replication)
- [Prometheus Metrics Best Practices](https://prometheus.io/docs/practices/naming/)
- [Grafana Dashboard Examples](https://grafana.com/grafana/dashboards/)
