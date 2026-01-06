# E2E Test Suite - README

## Overview

This directory contains the comprehensive end-to-end (E2E) test suite for Takhin. The tests verify the complete system behavior across all components.

## Test Categories

### 1. Producer/Consumer Tests (`producer_consumer/`)
- **Basic produce/consume**: Single partition produce and consume
- **Multi-partition**: Producing to and consuming from multiple partitions
- **Large messages**: Handling of messages up to 1MB
- **Batch operations**: High-volume batch produce operations
- **Offset management**: Consuming from specific offsets
- **Acknowledgments**: Testing different ack modes

### 2. Consumer Group Tests (`consumer_group/`)
- **Join/Leave**: Consumer group membership management
- **Rebalancing**: Partition rebalancing across consumers
- **Offset commit**: Consumer offset commit and fetch
- **Multiple consumers**: Concurrent consumers in same group
- **Failover**: Consumer failure and recovery
- **Session timeout**: Handling of session timeouts

### 3. Admin API Tests (`admin_api/`)
- **Create topics**: Topic creation via API
- **List topics**: Listing all topics
- **Delete topics**: Topic deletion
- **Describe topics**: Getting topic metadata
- **Alter configs**: Modifying topic configurations
- **Cluster metadata**: Fetching cluster information
- **Consumer groups**: Managing consumer groups
- **Error handling**: API error scenarios

### 4. Fault Injection Tests (`fault_injection/`)
- **Server restart**: Data persistence across restarts
- **Network partition**: Handling network splits
- **Leader failover**: Leader election and failover
- **Disk failure**: Handling disk full scenarios
- **Slow consumer**: Backpressure and flow control
- **Message corruption**: Handling corrupted data
- **Connection churn**: Rapid connection open/close
- **Memory pressure**: High memory usage scenarios

### 5. Performance Tests (`performance/`)
- **Produce throughput**: Maximum produce rate
- **Consume throughput**: Maximum consume rate
- **Concurrent producers**: Multiple concurrent producers
- **Concurrent consumers**: Multiple concurrent consumers
- **Latency**: End-to-end message latency
- **Backpressure**: System behavior under load
- **Long-running**: Stability over extended periods

## Running Tests

### Prerequisites

1. Go 1.21 or later
2. No external Kafka cluster needed (tests start embedded servers)
3. Sufficient disk space for test data

### Run All Tests

```bash
# From repository root
./scripts/run_e2e_tests.sh all

# Or use task
task e2e:test
```

### Run Specific Test Suite

```bash
# Producer/Consumer tests only
./scripts/run_e2e_tests.sh producer_consumer

# Consumer Group tests
./scripts/run_e2e_tests.sh consumer_group

# Admin API tests
./scripts/run_e2e_tests.sh admin_api

# Fault injection tests
./scripts/run_e2e_tests.sh fault_injection

# Performance tests
./scripts/run_e2e_tests.sh performance
```

### Run Individual Test

```bash
cd backend
go test -v -tags=e2e ./tests/e2e/producer_consumer -run TestBasicProduceConsume
```

### Skip E2E Tests

E2E tests are skipped by default in short mode:

```bash
go test -short ./...  # Skips E2E tests
```

## Test Infrastructure

### Test Utilities (`testutil/`)

#### TestServer
- Starts an embedded Takhin server for testing
- Automatically finds available ports
- Cleans up resources on test completion
- Provides helper methods for common operations

Example:
```go
srv := testutil.NewTestServer(t)
defer srv.Close()

err := srv.CreateTopic("test-topic", 3)
```

#### TestCluster
- Manages multiple servers in a cluster
- Simulates multi-broker scenarios
- Supports failover testing

Example:
```go
cluster := testutil.NewTestCluster(t, 3)
defer cluster.Close()

leader := cluster.Leader()
followers := cluster.Followers()
```

#### KafkaClient
- Simple Kafka protocol client
- Supports produce, fetch, metadata operations
- Uses binary Kafka protocol

Example:
```go
client, _ := testutil.NewKafkaClient(srv.Address())
defer client.Close()

err := client.Produce("topic", 0, []byte("key"), []byte("value"))
records, _ := client.Fetch("topic", 0, 0, 1024*1024)
```

## Test Patterns

### Table-Driven Tests

Most tests follow table-driven patterns for multiple scenarios:

```go
tests := []struct {
    name     string
    setup    func()
    expected interface{}
}{
    {"scenario1", setup1, expected1},
    {"scenario2", setup2, expected2},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic
    })
}
```

### Cleanup

All tests use `t.Cleanup()` or `defer` for proper resource cleanup:

```go
srv := testutil.NewTestServer(t)
t.Cleanup(func() {
    srv.Close()
})
```

### Assertions

Tests use `testify/assert` and `testify/require`:

```go
require.NoError(t, err)        // Stop test on error
assert.Equal(t, expected, actual)  // Continue on failure
assert.Greater(t, value, threshold)
```

## Performance Benchmarks

Performance tests log metrics for tracking:

```
Produce Throughput: 45.23 MB/s, 47234.56 msg/s
Consume Throughput: 67.89 MB/s, 70856.12 msg/s
Latency Statistics:
  Average: 12.3ms
  Min: 5.2ms
  Max: 45.6ms
```

## Continuous Integration

E2E tests run in CI pipeline:

```yaml
- name: E2E Tests
  run: ./scripts/run_e2e_tests.sh all
  timeout-minutes: 30
```

## Troubleshooting

### Port Already in Use
Tests automatically find available ports. If issues persist:
```bash
lsof -i :9092  # Check for processes using default port
```

### Test Timeouts
Increase timeout for slow systems:
```bash
go test -timeout=60m -tags=e2e ./tests/e2e/...
```

### Debug Mode
Run with verbose output:
```bash
./scripts/run_e2e_tests.sh all true
```

### View Server Logs
Tests log server output to test output:
```bash
go test -v -tags=e2e ./tests/e2e/producer_consumer
```

## Adding New Tests

1. Choose appropriate test category
2. Create test file with `// +build e2e` tag
3. Use `testutil.NewTestServer(t)` for setup
4. Follow existing test patterns
5. Add cleanup with `defer` or `t.Cleanup()`
6. Update this README with test description

Example:
```go
// +build e2e

package producer_consumer

func TestNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test in short mode")
    }
    
    srv := testutil.NewTestServer(t)
    defer srv.Close()
    
    // Test implementation
}
```

## Test Coverage

Current coverage by category:

| Category | Tests | Coverage |
|----------|-------|----------|
| Producer/Consumer | 7 | Core functionality |
| Consumer Group | 6 | Group coordination |
| Admin API | 10 | Admin operations |
| Fault Injection | 9 | Fault tolerance |
| Performance | 8 | Throughput & latency |

## Known Limitations

1. **Kafka Protocol**: Test client uses simplified protocol encoding
2. **Cluster Tests**: Limited multi-broker testing (Raft not fully integrated)
3. **Auth/TLS**: Security tests not included (covered in unit tests)
4. **Schema Registry**: Not included in E2E suite

## Future Enhancements

- [ ] Full Kafka protocol client implementation
- [ ] Multi-datacenter scenarios
- [ ] Cross-version compatibility tests
- [ ] Security E2E tests (SASL, TLS, ACL)
- [ ] Tiered storage E2E tests
- [ ] Exactly-once semantics tests
- [ ] Transactions E2E tests

## Related Documentation

- [Testing Strategy](../../docs/testing/)
- [Architecture Overview](../../docs/architecture/)
- [Development Guide](../../CONTRIBUTING.md)
