# Task 5.5: AlertManager Integration - Quick Reference

## Quick Start

```bash
# 1. Configure environment variables
export SMTP_PASSWORD="your-smtp-app-password"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T00/B00/XXX"

# 2. Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# 3. Run tests
./scripts/test-alerts.sh

# 4. Access UIs
# Prometheus: http://localhost:9091
# AlertManager: http://localhost:9093
# Grafana: http://localhost:3000
```

## Alert Rules Summary

### Critical Alerts (Immediate Action)
| Alert | Condition | Duration | Action |
|-------|-----------|----------|--------|
| TakhinDown | up == 0 | 1m | Check service status |
| TakhinHighErrorRate | >10 errors/sec | 5m | Check logs |
| TakhinOutOfMemory | >95% heap | 5m | Increase memory |
| TakhinDiskSpaceCritical | >95% disk | 2m | Clean old data |
| TakhinReplicationLagCritical | >10K offsets | 2m | Check replica |
| TakhinConsumerLagCritical | >100K offsets | 5m | Check consumer |

### High Severity Alerts
| Alert | Condition | Duration |
|-------|-----------|----------|
| TakhinDiskSpaceHigh | >85% disk | 5m |
| TakhinReplicationLagHigh | >1K offsets | 5m |
| TakhinConsumerLagHigh | >10K offsets | 10m |
| TakhinProduceLatencyHigh | P99 >1s | 10m |
| TakhinFetchLatencyHigh | P99 >1s | 10m |
| TakhinISRShrunk | ISR < replicas | 5m |
| TakhinHighIOErrorRate | >1 error/sec | 5m |

### Warning Alerts
| Alert | Condition | Duration |
|-------|-----------|----------|
| TakhinLogSegmentsTooMany | >1000 segments | 10m |
| TakhinConsumerGroupRebalancing | >5 in 10m | 5m |
| TakhinConsumerGroupNoMembers | 0 members | 10m |
| TakhinRequestLatencyHigh | P99 >5s | 10m |
| TakhinThroughputDropped | <50% of 1h ago | 10m |
| TakhinHighConnectionCount | >1000 | 10m |
| TakhinHighCPUUsage | >90% | 10m |
| TakhinHighGCPause | P99 >100ms | 5m |

## Notification Channels

### Slack Channels
- **#takhin-alerts**: Default (all alerts)
- **#takhin-critical**: Critical only (🚨 red)
- **#takhin-high-priority**: High severity (⚠️  yellow)
- **#takhin-warnings**: Warnings (ℹ️  blue)

### Email Recipients
- **team@takhin.io**: All alerts
- **oncall@takhin.io**: Critical alerts only
- **storage-team@takhin.io**: Storage alerts
- **replication-team@takhin.io**: Replication alerts
- **consumer-team@takhin.io**: Consumer alerts
- **performance-team@takhin.io**: Performance alerts

## Alert Routing Logic

```
Alert arrives → Group by [alertname, cluster, service]
              ↓
         Check severity
              ↓
    ┌─────────┴─────────┐
    ↓                   ↓
Critical            High/Warning
(0s wait)          (5-10s wait)
    ↓                   ↓
Multiple           Single
channels          channel
    ↓                   ↓
Repeat 4h         Repeat 6-24h
```

## Common Commands

### Check Alert Status
```bash
# List all alerts
curl -s http://localhost:9091/api/v1/alerts | jq '.data.alerts[] | {name:.labels.alertname, state:.state}'

# List firing alerts only
curl -s http://localhost:9091/api/v1/alerts | jq '.data.alerts[] | select(.state=="firing")'

# Check specific alert
curl -s http://localhost:9091/api/v1/alerts | jq '.data.alerts[] | select(.labels.alertname=="TakhinDown")'
```

### Send Test Alert
```bash
curl -X POST http://localhost:9093/api/v2/alerts \
  -H "Content-Type: application/json" \
  -d '[{
    "labels": {"alertname":"TestAlert","severity":"warning"},
    "annotations": {"summary":"Test alert"}
  }]'
```

### Check AlertManager Status
```bash
# Configuration status
curl -s http://localhost:9093/api/v2/status | jq .

# List receivers
curl -s http://localhost:9093/api/v2/status | jq '.config.receivers[].name'

# Check alerts in AlertManager
curl -s http://localhost:9093/api/v2/alerts | jq .
```

### Reload Configurations
```bash
# Reload Prometheus config
curl -X POST http://localhost:9091/-/reload

# Reload AlertManager config
curl -X POST http://localhost:9093/-/reload
```

### Query Alert Metrics
```bash
# Current error rate
curl -s 'http://localhost:9091/api/v1/query?query=rate(takhin_kafka_request_errors_total[5m])' | jq .

# Max replication lag
curl -s 'http://localhost:9091/api/v1/query?query=max(takhin_replication_lag_offsets)' | jq .

# Consumer lag
curl -s 'http://localhost:9091/api/v1/query?query=takhin_consumer_group_lag_offsets' | jq .
```

## Troubleshooting

### Alerts Not Firing
```bash
# 1. Check if Prometheus is scraping
curl http://localhost:9091/api/v1/targets

# 2. Check alert rules
curl http://localhost:9091/api/v1/rules

# 3. Check metric values
curl 'http://localhost:9091/api/v1/query?query=up{job="takhin"}'

# 4. Check Prometheus logs
docker-compose -f docker-compose.monitoring.yml logs prometheus
```

### Notifications Not Received
```bash
# 1. Check AlertManager status
curl http://localhost:9093/api/v2/status

# 2. Check environment variables
echo $SLACK_WEBHOOK_URL
echo $SMTP_PASSWORD

# 3. Check AlertManager logs
docker-compose -f docker-compose.monitoring.yml logs alertmanager

# 4. Send test alert
./scripts/test-alerts.sh
```

### Configuration Errors
```bash
# Validate Prometheus config
docker run --rm -v $(pwd)/docs/deployment:/config prom/prometheus:v2.48.0 \
  promtool check config /config/prometheus.yml

# Validate alert rules
docker run --rm -v $(pwd)/docs/deployment:/config prom/prometheus:v2.48.0 \
  promtool check rules /config/prometheus-alerts.yml

# Validate AlertManager config
docker run --rm -v $(pwd)/docs/deployment:/config prom/alertmanager:v0.26.0 \
  amtool check-config /config/alertmanager.yml
```

## Alert Tuning

### Adjust Thresholds
Edit `docs/deployment/prometheus-alerts.yml`:
```yaml
# Example: Increase lag threshold
- alert: TakhinReplicationLagHigh
  expr: takhin_replication_lag_offsets > 5000  # Was 1000
  for: 10m  # Increase wait time
```

### Change Notification Frequency
Edit `docs/deployment/alertmanager.yml`:
```yaml
route:
  repeat_interval: 24h  # Reduce notification frequency
  group_interval: 10m   # Group alerts longer
```

### Silence Alerts
```bash
# Silence alert for 2 hours
curl -X POST http://localhost:9093/api/v2/silences \
  -H "Content-Type: application/json" \
  -d '{
    "matchers": [{"name":"alertname","value":"TakhinDown"}],
    "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'",
    "endsAt": "'$(date -u -d '+2 hours' +%Y-%m-%dT%H:%M:%S.000Z)'",
    "comment": "Maintenance window"
  }'

# List silences
curl http://localhost:9093/api/v2/silences

# Delete silence by ID
curl -X DELETE http://localhost:9093/api/v2/silence/{id}
```

## Integration Examples

### Slack Webhook Setup
```bash
# 1. Create Slack app: https://api.slack.com/apps
# 2. Enable Incoming Webhooks
# 3. Create webhook
# 4. Set environment variable
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### Gmail SMTP Setup
```bash
# 1. Enable 2FA on Gmail account
# 2. Generate App Password: https://myaccount.google.com/apppasswords
# 3. Set environment variable
export SMTP_PASSWORD="your-16-char-app-password"
```

### PagerDuty Integration
Add to `alertmanager.yml`:
```yaml
receivers:
  - name: 'critical-alerts'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}'
```

## File Locations

### Configuration Files
- AlertManager: `docs/deployment/alertmanager.yml`
- Alert Rules: `docs/deployment/prometheus-alerts.yml`
- Prometheus: `docs/deployment/prometheus.yml`
- Docker Compose: `docker-compose.monitoring.yml`

### Test Scripts
- Alert Testing: `scripts/test-alerts.sh`

### Documentation
- Complete Guide: `TASK_5.5_ALERTING_COMPLETION.md`
- Quick Reference: `TASK_5.5_QUICK_REFERENCE.md`

## Metrics Reference

All alerts use metrics from Task 5.1:
- `takhin_kafka_*` - Kafka API metrics
- `takhin_storage_*` - Storage metrics
- `takhin_replication_*` - Replication metrics
- `takhin_consumer_group_*` - Consumer metrics
- `takhin_produce_*` / `takhin_fetch_*` - Performance metrics
- `takhin_go_*` - Go runtime metrics

See `docs/metrics.md` for complete metric documentation.

## Support

For issues or questions:
1. Check logs: `docker-compose -f docker-compose.monitoring.yml logs`
2. Run tests: `./scripts/test-alerts.sh`
3. Validate configs with promtool/amtool
4. Review documentation: `TASK_5.5_ALERTING_COMPLETION.md`
