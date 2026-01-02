# Replication Test Execution Guide

## Quick Start

### Run All Replication Tests
```bash
cd backend
go test -v ./pkg/replication/
```

### Run Only Integration Tests
```bash
go test -v -run "Integration|Failover|Recovery|Partition" ./pkg/replication/
```

### Run Unit Tests Only (Skip Integration)
```bash
go test -v -short ./pkg/replication/
```

## Test Execution Matrix

| Command | Duration | Use Case |
|---------|----------|----------|
| `go test ./pkg/replication/` | ~2s | Quick validation (unit tests only with -short) |
| `go test -v ./pkg/replication/` | ~91s | Full test suite with verbose output |
| `go test -race ./pkg/replication/` | ~120s | Race condition detection |
| `go test -cover ./pkg/replication/` | ~91s | Code coverage report |
| `go test -bench ./pkg/replication/` | Varies | Performance benchmarks |

## Individual Test Commands

### 1. Three-Node Replication
```bash
go test -v -run TestThreeNodeReplication ./pkg/replication/
# Expected: ~14s, 100 messages replicated across 3 nodes
```

**What it tests:**
- Normal replication in healthy cluster
- Data consistency verification
- Message ordering guarantees

**Expected output:**
```
=== RUN   TestThreeNodeReplication
    integration_test.go:29: Leader elected: node0
    integration_test.go:46: Writing 100 messages through leader...
    integration_test.go:62: Verifying data consistency across all nodes...
    integration_test.go:81: ✅ All 3 nodes have consistent replicated data
--- PASS: TestThreeNodeReplication (13.86s)
```

---

### 2. Leader Failover
```bash
go test -v -run TestLeaderFailover ./pkg/replication/
# Expected: ~22s, automatic leader election after crash
```

**What it tests:**
- Leader crash detection
- New leader election
- Data consistency during failover
- Zero data loss guarantee

**Expected output:**
```
=== RUN   TestLeaderFailover
    integration_test.go:119: Initial leader: node0
    integration_test.go:129: Writing initial data...
    integration_test.go:144: Shutting down leader (node0)...
    integration_test.go:151: Waiting for new leader election...
    integration_test.go:144: New leader elected: node2
    integration_test.go:147: Writing 50 messages through new leader...
    integration_test.go:174: ✅ Leader failover successful with data consistency
--- PASS: TestLeaderFailover (21.56s)
```

---

### 3. Follower Recovery
```bash
go test -v -run TestFollowerRecovery ./pkg/replication/
# Expected: ~17s, follower catches up after restart
```

**What it tests:**
- Follower node crash tolerance
- Automatic catch-up mechanism
- Data replication after recovery
- Cluster resilience (2/3 nodes operating)

**Expected output:**
```
=== RUN   TestFollowerRecovery
    integration_test.go:187: Leader: node0
    integration_test.go:195: Writing 30 messages...
    integration_test.go:212: Shutting down follower node1...
    integration_test.go:217: Writing 30 more messages while follower is down...
    integration_test.go:236: Restarting follower node1...
    integration_test.go:256: Waiting for follower to catch up...
    integration_test.go:276: ✅ Follower recovered and caught up successfully
--- PASS: TestFollowerRecovery (16.64s)
```

---

### 4. Network Partition
```bash
go test -v -run TestNetworkPartition ./pkg/replication/
# Expected: ~11s, majority partition continues operating
```

**What it tests:**
- Split-brain prevention
- Majority quorum enforcement
- Minority partition cannot form quorum
- Data consistency in majority partition

**Expected output:**
```
=== RUN   TestNetworkPartition
    integration_test.go:295: Initial leader: node0
    integration_test.go:303: Writing initial data...
    integration_test.go:315: Simulating network partition - isolating node2...
    integration_test.go:319: Writing to majority partition...
    integration_test.go:339: ✅ Majority partition continues operating correctly
--- PASS: TestNetworkPartition (11.11s)
```

---

### 5. Concurrent Writes
```bash
go test -v -run TestConcurrentWrites ./pkg/replication/
# Expected: ~17s, 500 messages from 10 concurrent writers
```

**What it tests:**
- Concurrent write handling
- Race condition safety
- Throughput under load
- Data consistency with concurrent access

**Expected output:**
```
=== RUN   TestConcurrentWrites
    integration_test.go:352: Leader: node0
    integration_test.go:363: Starting 10 concurrent writers, 50 messages each...
    integration_test.go:391: Concurrent write completed: 500 success, 0 errors in 5.39s
    integration_test.go:393: Throughput: 92.67 msg/sec
    integration_test.go:412: ✅ Concurrent writes successful with data consistency
--- PASS: TestConcurrentWrites (16.66s)
```

---

### 6. Replication Performance
```bash
go test -v -run TestReplicationPerformance ./pkg/replication/
# Expected: ~12s, latency measurement for 100 writes
```

**What it tests:**
- Per-message replication latency
- System throughput
- Performance baselines
- SLA validation (< 100ms target)

**Expected output:**
```
=== RUN   TestReplicationPerformance
    integration_test.go:434: Measuring replication latency for 100 messages...
    integration_test.go:447: Average replication latency: 25.76ms
    integration_test.go:448: Total time: 2.58 seconds
    integration_test.go:449: Throughput: 38.82 msg/sec
    integration_test.go:464: ✅ Replication performance within acceptable limits
--- PASS: TestReplicationPerformance (11.75s)
```

---

## Advanced Test Execution

### Race Detection (Critical for Concurrency)
```bash
go test -v -race ./pkg/replication/
```
**Purpose:** Detect data races in concurrent code  
**Overhead:** 5-10x slower, higher memory usage  
**When to use:** Before committing changes to replication logic

---

### Code Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./pkg/replication/

# View coverage in browser
go tool cover -html=coverage.out

# View coverage summary
go tool cover -func=coverage.out
```

**Target Coverage:**
- Overall: > 80%
- Critical paths (leader election, replication): > 90%

---

### Stress Testing
```bash
# Run tests multiple times to catch flaky behavior
go test -count=10 ./pkg/replication/

# Run with timeout
go test -timeout 5m ./pkg/replication/
```

---

### Parallel Execution
```bash
# Run tests in parallel (default)
go test -v -parallel 4 ./pkg/replication/

# Sequential execution (for debugging)
go test -v -p 1 ./pkg/replication/
```

---

### Filter by Pattern
```bash
# Run all failover-related tests
go test -v -run ".*Failover.*" ./pkg/replication/

# Run all recovery tests
go test -v -run ".*Recovery.*" ./pkg/replication/

# Run performance tests only
go test -v -run ".*Performance.*" ./pkg/replication/
```

---

## Continuous Integration

### Local Pre-Commit Checks
```bash
# Complete validation before commit
cd backend

# 1. Format code
go fmt ./pkg/replication/

# 2. Run linter
golangci-lint run ./pkg/replication/

# 3. Run tests with race detector
go test -v -race ./pkg/replication/

# 4. Check coverage
go test -coverprofile=coverage.out ./pkg/replication/
go tool cover -func=coverage.out | grep total
```

**Acceptance criteria:**
- ✅ All tests pass
- ✅ No race conditions detected
- ✅ Coverage > 80%
- ✅ No linter warnings

---

### Task Runner (Recommended)
```bash
# Use Taskfile for convenience
task backend:test         # Run all tests with race detector
task backend:test:unit    # Run only unit tests (skip integration)
task backend:lint         # Run linter
task backend:coverage     # Generate and view coverage report
```

---

## Interpreting Results

### Success Indicators
```
✅ All nodes have consistent replicated data
✅ Leader failover successful with data consistency
✅ Follower recovered and caught up successfully
✅ Majority partition continues operating correctly
✅ Concurrent writes successful with data consistency
✅ Replication performance within acceptable limits
```

### Failure Indicators

**Test Timeout**
```
panic: test timed out after 10m0s
```
**Cause:** Node not responding, network issue, or deadlock  
**Action:** Check logs for specific node failures, increase timeout

**HWM Mismatch**
```
Expected: 100, Actual: 95
```
**Cause:** Replication lag or node failure  
**Action:** Increase wait time, check node health

**Leader Election Failure**
```
no leader elected within timeout
```
**Cause:** Insufficient quorum or port conflicts  
**Action:** Check node connectivity, verify port availability

**Race Condition Detected**
```
WARNING: DATA RACE
Read at 0x00c0001a2c00
```
**Cause:** Concurrent access without proper synchronization  
**Action:** Fix race condition with mutexes/channels

---

## Performance Baselines

### Expected Metrics (Localhost)

| Metric | Target | Typical | Notes |
|--------|--------|---------|-------|
| Leader election time | < 10s | ~3-5s | After node failure |
| Follower catch-up time | < 10s | ~5s | For 30 missed messages |
| Avg replication latency | < 100ms | ~25ms | Per message |
| Concurrent write throughput | > 50 msg/s | ~90 msg/s | 10 writers |
| Sequential throughput | > 30 msg/s | ~40 msg/s | Single writer |

### Factors Affecting Performance
- **Disk I/O:** SSD vs HDD significantly impacts latency
- **Network latency:** Localhost has ~0.1ms, production may have 1-50ms
- **System load:** CPU/memory contention affects results
- **Test environment:** tmpfs vs physical disk

---

## Troubleshooting Guide

### Common Issues

#### 1. Port Already in Use
```
Error: bind: address already in use
```
**Solution:**
```bash
# Find process using port
lsof -i :18001

# Kill process
kill -9 <PID>

# Or wait for TIME_WAIT to expire (~60s)
```

#### 2. Test Hangs/Deadlock
**Symptoms:** Test runs forever without output  
**Debug:**
```bash
# Run with verbose logging
go test -v -run TestThreeNodeReplication ./pkg/replication/

# Check goroutine stacks
kill -QUIT <test-process-pid>
```

#### 3. Flaky Tests
**Symptoms:** Tests pass sometimes, fail others  
**Common causes:**
- Race conditions (run with `-race`)
- Timing issues (increase wait times)
- Resource contention (run with `-p 1`)

**Debug approach:**
```bash
# Run multiple times to reproduce
go test -count=20 -run TestLeaderFailover ./pkg/replication/
```

#### 4. Memory Issues
```
Error: cannot allocate memory
```
**Solution:**
- Close unused applications
- Increase system swap space
- Run fewer parallel tests: `-parallel 1`

---

## Debugging Tips

### Enable Detailed Logging
```bash
# Set Raft log level (in test setup)
export RAFT_LOG_LEVEL=DEBUG

# Run test with all logs
go test -v -run TestLeaderFailover ./pkg/replication/ 2>&1 | tee test.log
```

### Inspect Raft State
```go
// In test code
for i, node := range cluster.Nodes {
    stats := node.RaftNode.Stats()
    t.Logf("Node%d stats: %+v", i, stats)
}
```

### Add Breakpoints (Delve)
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug test
dlv test ./pkg/replication/ -- -test.run TestLeaderFailover
(dlv) break integration_test.go:144
(dlv) continue
```

---

## Best Practices

### 1. Run Tests Before Committing
```bash
task backend:test
```

### 2. Check Race Conditions Regularly
```bash
go test -race ./pkg/replication/
```

### 3. Monitor Performance Trends
Track metrics over time:
- Replication latency should remain < 100ms
- Throughput should not degrade
- Test duration should be stable

### 4. Clean Up After Tests
- Tests use `t.TempDir()` for automatic cleanup
- No manual cleanup needed
- Verify with: `du -sh /tmp/go-build*`

### 5. Document Changes
When modifying tests:
- Update comments
- Adjust timeouts if needed
- Update expected metrics
- Add new failure scenarios

---

## CI/CD Integration

### GitHub Actions Example
See `.github/workflows/replication-tests.yml`

### Manual CI Run
```bash
# Simulate CI environment
docker run --rm -v $(pwd):/workspace -w /workspace golang:1.21 \
  bash -c "cd backend && go test -v -race ./pkg/replication/"
```

---

## Support

**Documentation:** `docs/testing/replication-integration-tests.md`  
**Issues:** Create GitHub issue with test logs  
**Questions:** Contact #takhin-dev on Slack  

**Test Artifacts:**
- Logs: `backend/test.log`
- Coverage: `backend/coverage.out`
- Race reports: Console output

---

**Last Updated:** 2026-01-02  
**Maintainer:** Takhin Data Engineering Team
