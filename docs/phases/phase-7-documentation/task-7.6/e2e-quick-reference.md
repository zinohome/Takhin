# E2E Test Suite Quick Reference

## Quick Commands

```bash
# Run all E2E tests
./scripts/run_e2e_tests.sh all
task backend:test:e2e

# Run specific suite
./scripts/run_e2e_tests.sh producer_consumer
./scripts/run_e2e_tests.sh consumer_group
./scripts/run_e2e_tests.sh admin_api
./scripts/run_e2e_tests.sh fault_injection
./scripts/run_e2e_tests.sh performance

# Quick smoke tests
task backend:test:e2e:quick

# Individual test
go test -v -tags=e2e ./backend/tests/e2e/producer_consumer -run TestBasicProduceConsume
```

## Test Categories (40 Tests Total)

### Producer/Consumer (7 tests)
- Basic produce/consume
- Multi-partition operations
- Large messages (1MB+)
- Batch operations (1000+ msgs)
- Offset-based consumption
- Acknowledgment verification

### Consumer Group (6 tests)
- Group join/leave
- Partition rebalancing
- Offset commit/fetch
- Multiple concurrent consumers
- Failover scenarios
- Session timeout handling

### Admin API (10 tests)
- Topic CRUD operations
- Configuration management
- Cluster metadata
- Consumer group management
- Error handling

### Fault Injection (9 tests)
- Server restart & recovery
- Network partitions
- Leader failover
- Disk failure
- Slow consumer backpressure
- Message corruption
- Connection churn
- Memory pressure

### Performance (8 tests)
- Produce throughput (MB/s, msg/s)
- Consume throughput (MB/s, msg/s)
- Concurrent producers
- Concurrent consumers
- Latency (avg/min/max)
- Backpressure handling
- Long-running stability

## Test Infrastructure

### TestServer
```go
srv := testutil.NewTestServer(t)
defer srv.Close()

srv.CreateTopic("test-topic", 3)
addr := srv.Address()  // "localhost:PORT"
```

### TestCluster
```go
cluster := testutil.NewTestCluster(t, 3)
defer cluster.Close()

leader := cluster.Leader()
followers := cluster.Followers()
```

### KafkaClient
```go
client, _ := testutil.NewKafkaClient(srv.Address())
defer client.Close()

// Produce
client.Produce("topic", 0, []byte("key"), []byte("value"))

// Consume
records, _ := client.Fetch("topic", 0, 0, 1024*1024)

// Admin
client.CreateTopics([]string{"topic1", "topic2"}, 3, 1)
metadata, _ := client.Metadata([]string{"topic"})
```

## Common Patterns

### Basic Test Template
```go
// +build e2e

package mytest

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/takhin-data/takhin/tests/e2e/testutil"
)

func TestMyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test in short mode")
    }

    srv := testutil.NewTestServer(t)
    defer srv.Close()

    // Setup
    err := srv.CreateTopic("test-topic", 1)
    require.NoError(t, err)

    // Test
    client, err := testutil.NewKafkaClient(srv.Address())
    require.NoError(t, err)
    defer client.Close()

    // Verify
    assert.NoError(t, err)
}
```

### Performance Benchmark Template
```go
func TestThroughput(t *testing.T) {
    srv := testutil.NewTestServer(t)
    defer srv.Close()

    startTime := time.Now()
    // ... operations ...
    duration := time.Since(startTime)

    throughputMsgSec := float64(count) / duration.Seconds()
    t.Logf("Throughput: %.2f msg/s", throughputMsgSec)

    assert.Greater(t, throughputMsgSec, 100.0)
}
```

## Directory Structure

```
backend/tests/e2e/
├── testutil/
│   ├── server.go           # TestServer, TestCluster
│   └── kafka_client.go     # KafkaClient
├── producer_consumer/      # 7 tests
├── consumer_group/         # 6 tests
├── admin_api/              # 10 tests
├── fault_injection/        # 9 tests
├── performance/            # 8 tests
└── README.md
```

## Performance Baselines

| Metric | Baseline | Test |
|--------|----------|------|
| Produce Throughput | >100 msg/s | TestProduceThroughput |
| Consume Throughput | >100 msg/s | TestConsumeThroughput |
| Average Latency | <100ms | TestLatency |
| Concurrent Success | >80% | TestConcurrentProducers |

## Troubleshooting

### Port Conflicts
```bash
lsof -i :9092  # Check usage
# Tests auto-allocate ports
```

### Test Timeouts
```bash
go test -timeout=60m -tags=e2e ./backend/tests/e2e/...
```

### Debug Output
```bash
go test -v -tags=e2e ./backend/tests/e2e/producer_consumer
```

### Memory Issues
```bash
go test -parallel=4 -tags=e2e ./backend/tests/e2e/...
```

## CI Integration

```yaml
- name: E2E Tests
  run: ./scripts/run_e2e_tests.sh all
  timeout-minutes: 30
```

## Test Flags

```bash
-tags=e2e          # Required for E2E tests
-v                 # Verbose output
-race              # Race detector
-timeout=30m       # Timeout
-parallel=N        # Concurrency
-run=TestName      # Specific test
-short             # Skip E2E tests
```

## Related Files

- [Test Implementation](./backend/tests/e2e/)
- [Completion Doc](./TASK_7.6_E2E_COMPLETION.md)
- [Test Runner](./scripts/run_e2e_tests.sh)
- [Taskfile](./Taskfile.yaml)
