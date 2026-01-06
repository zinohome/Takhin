# Network Throttling - Quick Reference

## Overview
Network throttling prevents server overload through rate limiting for producer and consumer traffic.

## Quick Start

### Configuration (configs/takhin.yaml)
```yaml
throttle:
  producer:
    bytes:
      per:
        second: 10485760    # 10 MB/s
    burst: 20971520         # 20 MB burst
  consumer:
    bytes:
      per:
        second: 10485760    # 10 MB/s
    burst: 20971520         # 20 MB burst
  dynamic:
    enabled: false          # Dynamic adjustment
```

### Environment Variables
```bash
TAKHIN_THROTTLE_PRODUCER_BYTES_PER_SECOND=10485760
TAKHIN_THROTTLE_CONSUMER_BYTES_PER_SECOND=10485760
TAKHIN_THROTTLE_DYNAMIC_ENABLED=true
```

## Usage

### Basic Throttling
```go
import "github.com/takhin-data/takhin/pkg/throttle"

// Create throttler
cfg := throttle.DefaultConfig()
throttler := throttle.New(cfg)
defer throttler.Close()

// Check producer quota
ctx := context.Background()
err := throttler.AllowProducer(ctx, messageSize)

// Check consumer quota
err = throttler.AllowConsumer(ctx, responseSize)
```

### Dynamic Rate Updates
```go
// Update producer rate at runtime
throttler.UpdateProducerRate(20*1024*1024, 40*1024*1024)

// Update consumer rate
throttler.UpdateConsumerRate(15*1024*1024, 0)

// Get statistics
stats := throttler.GetStats()
```

## Monitoring

### Prometheus Metrics
```prometheus
# Request counters
takhin_throttle_requests_total{type,status}

# Byte counters
takhin_throttle_bytes_total{type,status}

# Current rates
takhin_throttle_rate_bytes_per_second{type}
```

### Example Queries
```promql
# Throttle rate
rate(takhin_throttle_requests_total{status="throttled"}[5m])

# Throughput
rate(takhin_throttle_bytes_total{status="allowed"}[1m])
```

## Common Configurations

### High Throughput
```yaml
producer:
  bytes_per_second: 100000000  # 100 MB/s
  burst: 200000000
```

### Low Latency
```yaml
producer:
  bytes_per_second: 10000000   # 10 MB/s
  burst: 50000000              # Generous burst
dynamic:
  target_util_pct: 0.70        # Leave headroom
```

### Resource Constrained
```yaml
producer:
  bytes_per_second: 1000000    # 1 MB/s
  burst: 2000000
dynamic:
  enabled: true
  min_rate: 524288             # 512 KB/s
  max_rate: 5242880            # 5 MB/s
```

## Troubleshooting

### Producer/Consumer Throttled
**Symptom**: Requests timing out or delayed  
**Solution**: Increase rate limit or burst capacity

### High Memory Usage
**Symptom**: Memory increasing over time  
**Solution**: Disable throttling or increase rate limits

### Uneven Distribution
**Symptom**: Some clients throttled, others not  
**Solution**: Enable dynamic adjustment

## Testing

```bash
# Run tests
go test ./pkg/throttle/... -v

# Run benchmarks
go test ./pkg/throttle/... -bench=.

# Check metrics
curl http://localhost:9090/metrics | grep throttle
```

## API Reference

### Types
- `Throttler` - Main throttler instance
- `Config` - Configuration struct
- `Stats` - Statistics struct

### Methods
- `New(cfg *Config) *Throttler` - Create throttler
- `AllowProducer(ctx, bytes) error` - Check producer quota
- `AllowConsumer(ctx, bytes) error` - Check consumer quota
- `UpdateProducerRate(rate, burst)` - Update producer rate
- `UpdateConsumerRate(rate, burst)` - Update consumer rate
- `GetStats() Stats` - Get statistics
- `Close() error` - Cleanup resources

### Configuration Fields
- `ProducerBytesPerSecond` (int64) - Producer rate limit
- `ProducerBurst` (int) - Producer burst capacity
- `ConsumerBytesPerSecond` (int64) - Consumer rate limit
- `ConsumerBurst` (int) - Consumer burst capacity
- `DynamicEnabled` (bool) - Enable dynamic adjustment
- `DynamicCheckInterval` (int) - Check interval (ms)
- `DynamicMinRate` (int64) - Minimum rate
- `DynamicMaxRate` (int64) - Maximum rate
- `DynamicTargetUtilPct` (float64) - Target utilization
- `DynamicAdjustmentStep` (float64) - Adjustment step

## Performance

- **Disabled**: ~134 ns/op
- **Enabled**: ~2.8 µs/op
- **Concurrent**: ~1.2 µs/op
- **Memory**: 320 B/op

## Related Documentation

- Full Implementation: `TASK_3.4_NETWORK_THROTTLING.md`
- Configuration: `backend/configs/takhin.yaml`
- Metrics: `backend/pkg/metrics/metrics.go`
- Handler Integration: `backend/pkg/kafka/handler/handler.go`
