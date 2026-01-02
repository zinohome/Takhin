# Replication Integration Tests - Quick Reference

## 🚀 Quick Start

```bash
# Run all tests
cd backend
go test -v ./pkg/replication/

# Run specific test
go test -v -run TestLeaderFailover ./pkg/replication/

# With race detection
go test -v -race ./pkg/replication/
```

## 📋 Test Suite Overview

| Test | Duration | What It Tests | Status |
|------|----------|---------------|--------|
| **TestThreeNodeReplication** | 14s | Normal 3-node replication, data consistency | ✅ |
| **TestLeaderFailover** | 21s | Leader crash, automatic re-election, zero data loss | ✅ |
| **TestFollowerRecovery** | 17s | Follower crash, automatic catch-up | ✅ |
| **TestNetworkPartition** | 11s | Split-brain prevention, majority quorum | ✅ |
| **TestConcurrentWrites** | 17s | Concurrent write handling, throughput | ✅ |
| **TestReplicationPerformance** | 12s | Latency measurement, performance baseline | ✅ |

**Total:** 6 integration tests + 4 unit tests = 10 tests  
**Total Duration:** ~92 seconds  
**Pass Rate:** 100%

## 📊 Performance Metrics

```
Average Replication Latency:  25ms   (target: < 100ms) ✅
Concurrent Write Throughput:  92 msg/sec  (target: > 50) ✅
Sequential Write Throughput:  38 msg/sec  (target: > 30) ✅
Leader Election Time:         ~5 seconds  (target: < 10s) ✅
Follower Catch-up Time:       ~5 seconds  (target: < 10s) ✅
```

## 📖 Documentation

- **[Integration Test Details](./replication-integration-tests.md)** - Full test documentation
- **[Test Execution Guide](./replication-test-guide.md)** - How to run and debug tests
- **[Task Completion Report](./TASK-1.4-COMPLETION.md)** - Complete task summary

## 🔧 Common Commands

```bash
# Run with coverage
go test -coverprofile=coverage.out ./pkg/replication/
go tool cover -html=coverage.out

# Run specific tests
go test -run TestLeaderFailover ./pkg/replication/
go test -run TestConcurrent ./pkg/replication/

# Debug mode
go test -v -run TestLeaderFailover ./pkg/replication/ 2>&1 | tee test.log

# Skip integration tests (fast)
go test -short ./pkg/replication/
```

## 🎯 Acceptance Criteria ✅

- [x] 3-node cluster normal replication test
- [x] Leader crash failover test
- [x] Follower crash recovery test  
- [x] Network partition test (split-brain prevention)
- [x] Data consistency verification
- [x] Performance impact assessment

## 🏗️ Architecture

```
Client → Leader Node → Raft Consensus → Followers
                ↓
         Replicate to majority
                ↓
         Commit (update HWM)
                ↓
         Apply to FSM (Topic Manager)
                ↓
         Client Acknowledgment
```

## 🐛 Troubleshooting

**Port conflicts:**
```bash
lsof -i :18001  # Check what's using ports
```

**Slow tests:**
- Check system load
- Verify disk speed (avoid slow tmpfs)
- Run sequentially: `go test -p 1`

**Flaky tests:**
```bash
go test -count=10 -run TestLeaderFailover ./pkg/replication/
```

## 📦 CI/CD

GitHub Actions automatically runs tests on:
- Push to main/develop
- Pull requests
- Manual workflow dispatch

See `.github/workflows/replication-tests.yml`

## 🎓 Key Learnings

1. **Strong Consistency:** Raft provides linearizable reads/writes
2. **Fault Tolerance:** Cluster survives n/2 failures (3-node = 1 failure)
3. **Performance:** 25ms latency is acceptable for most use cases
4. **Testing:** Comprehensive tests catch issues before production

## 📞 Support

- **Questions:** See detailed docs above
- **Issues:** Create GitHub issue with test logs
- **Contributions:** Follow existing test patterns

---

**Status:** ✅ Production Ready  
**Last Updated:** 2026-01-02  
**Maintainer:** Takhin Data Engineering Team
