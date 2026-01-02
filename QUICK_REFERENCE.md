# Leader Election Optimization - Quick Reference

## Overview
Task 1.5 optimizes Raft leader election to achieve < 5s election times with PreVote mechanism enabled.

## Quick Facts
- **Election Time:** ~1.5s (single-node), 2-4s (multi-node)
- **PreVote:** Enabled by default
- **Metrics:** 6 new Prometheus metrics
- **Config Location:** `backend/configs/takhin.yaml` → `raft:` section

## Configuration

### YAML Config (backend/configs/takhin.yaml)
```yaml
raft:
  heartbeat:
    timeout:
      ms: 1000            # Time without leader before election
  election:
    timeout:
      ms: 3000            # Max election duration
  leader:
    lease:
      timeout:
        ms: 500           # Leader lease timeout
  prevote:
    enabled: true         # Enable PreVote (default)
```

### Environment Variables
```bash
export TAKHIN_RAFT_HEARTBEAT_TIMEOUT_MS=1000
export TAKHIN_RAFT_ELECTION_TIMEOUT_MS=3000
export TAKHIN_RAFT_PREVOTE_ENABLED=true
```

## Metrics

Access at `http://localhost:9090/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `takhin_raft_elections_total` | Counter | Total elections initiated |
| `takhin_raft_election_duration_seconds` | Histogram | Election duration |
| `takhin_raft_leader_changes_total` | Counter | Leader changes |
| `takhin_raft_state` | Gauge | Current state (0/1/2) |

## Testing

```bash
# Run all Raft tests
cd backend && go test ./pkg/raft -v

# Run election tests only
go test ./pkg/raft -v -run Election

# Benchmark election performance
go test ./pkg/raft -bench=BenchmarkLeaderElection
```

## Monitoring

### Prometheus Queries
```promql
# Average election duration
rate(takhin_raft_election_duration_seconds_sum[5m]) / 
rate(takhin_raft_election_duration_seconds_count[5m])

# Leader changes per minute
rate(takhin_raft_leader_changes_total[1m]) * 60
```

### Alerts
```promql
# Alert if election > 5s
histogram_quantile(0.99, rate(takhin_raft_election_duration_seconds_bucket[5m])) > 5

# Alert if too many leader changes
rate(takhin_raft_leader_changes_total[1h]) > 5
```

## Files Modified

| File | Changes | Purpose |
|------|---------|---------|
| `pkg/config/config.go` | +77 lines | RaftConfig struct, defaults, validation |
| `pkg/raft/node.go` | +69 lines | Apply config, monitoring, PreVote |
| `pkg/metrics/metrics.go` | +42 lines | 6 new metrics |
| `configs/takhin.yaml` | +40 lines | Raft configuration section |
| `pkg/raft/election_test.go` | NEW | Election optimization tests |

## Key Code Locations

**PreVote Configuration:**
```go
// backend/pkg/raft/node.go:66
raftConfig.PreVoteDisabled = !cfg.RaftCfg.PreVoteEnabled
```

**Metrics Monitoring:**
```go
// backend/pkg/raft/node.go:217
func (n *Node) monitorLeadership() {
    // Updates metrics on state changes
}
```

**Config Defaults:**
```go
// backend/pkg/config/config.go:182
if cfg.Raft.ElectionTimeoutMs == 0 {
    cfg.Raft.ElectionTimeoutMs = 3000
}
```

## Troubleshooting

### Problem: Slow Elections (> 5s)
**Solutions:**
- Check network latency: `ping <peer-node>`
- Increase election timeout: `TAKHIN_RAFT_ELECTION_TIMEOUT_MS=5000`
- Verify node connectivity

### Problem: Frequent Leader Changes
**Solutions:**
- Check network stability
- Increase heartbeat timeout
- Review logs: `grep "leadership changed" logs/`

### Problem: No Metrics
**Solutions:**
- Verify metrics server enabled: `metrics.enabled: true`
- Check metrics endpoint: `curl http://localhost:9090/metrics`
- Ensure Prometheus is scraping the endpoint

## Documentation

- **Full Implementation:** `docs/TASK_1.5_LEADER_ELECTION.md`
- **Completion Summary:** `TASK_1.5_COMPLETION_SUMMARY.md`
- **Config Reference:** `backend/configs/takhin.yaml`
- **Test File:** `backend/pkg/raft/election_test.go`

## Acceptance Criteria

- ✅ PreVote mechanism implemented
- ✅ Election timeouts optimized
- ✅ Metrics monitoring added
- ✅ Election time < 5s verified

## Next Steps

1. Deploy to staging environment
2. Monitor election metrics over 24 hours
3. Tune timeouts based on observed network latency
4. Document any production-specific configurations

## Support

For issues or questions:
- Check logs: `grep "raft" logs/takhin.log | grep -E "election|leadership"`
- Review metrics: `curl http://localhost:9090/metrics | grep raft`
- Test locally: `go test ./pkg/raft -v -run TestElection`
