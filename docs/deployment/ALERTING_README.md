# Takhin AlertManager Integration

This directory contains AlertManager and Prometheus configurations for comprehensive monitoring and alerting of the Takhin streaming platform.

## Quick Start

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env with your SMTP password and Slack webhook URL

# 2. Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# 3. Verify installation
./scripts/test-alerts.sh

# 4. Access web interfaces
open http://localhost:9091  # Prometheus
open http://localhost:9093  # AlertManager
open http://localhost:3000  # Grafana
```

## Files Overview

### Configuration Files
- **`alertmanager.yml`** - AlertManager routing and notification configuration
- **`prometheus.yml`** - Prometheus scrape configuration and AlertManager integration
- **`prometheus-alerts.yml`** - 29 alert rules for comprehensive monitoring
- **`alert-templates.tmpl`** - Custom notification templates (Slack & Email)

### Deployment
- **`docker-compose.monitoring.yml`** - Complete monitoring stack (7 services)
- **`.env.example`** - Environment variable template

### Testing & Scripts
- **`scripts/test-alerts.sh`** - Automated testing suite

### Documentation
- **`TASK_5.5_ALERTING_COMPLETION.md`** - Complete implementation guide
- **`TASK_5.5_QUICK_REFERENCE.md`** - Quick reference for daily operations
- **`TASK_5.5_VISUAL_OVERVIEW.md`** - Architecture and flow diagrams
- **`TASK_5.5_ACCEPTANCE_CHECKLIST.md`** - Acceptance criteria verification

## Alert Categories

### Critical (6 alerts) - Immediate Action
- Service down
- High error rate (>10/sec)
- Out of memory (>95% heap)
- Too many goroutines (>10K)
- Disk space critical (>95%)
- Replication/Consumer lag critical

### High (10 alerts) - Urgent Attention
- Disk space high (>85%)
- Replication lag high (>1K offsets)
- Consumer lag high (>10K offsets)
- High latency (P99 >1s)
- ISR shrunk
- High I/O errors

### Warning (13 alerts) - Monitor Closely
- Too many log segments
- Frequent rebalances
- Throughput dropped
- High connections
- System resources

**Total: 29 comprehensive alert rules**

## Notification Channels

### Slack
- `#takhin-alerts` - All alerts
- `#takhin-critical` - Critical only (red)
- `#takhin-high-priority` - High severity (orange)
- `#takhin-warnings` - Warnings (yellow)

### Email
- team@takhin.io - All alerts
- oncall@takhin.io - Critical only
- Specialized team emails for categories

## Configuration

### Required Environment Variables
```bash
SMTP_PASSWORD=your-gmail-app-password
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T00/B00/XXX
GRAFANA_PASSWORD=secure-password
```

### Gmail Setup
1. Enable 2FA on Gmail account
2. Generate App Password: https://myaccount.google.com/apppasswords
3. Set `SMTP_PASSWORD` in `.env`

### Slack Setup
1. Create Slack app: https://api.slack.com/apps
2. Enable Incoming Webhooks
3. Create webhook for channels
4. Set `SLACK_WEBHOOK_URL` in `.env`

## Service Ports

| Service | Port | Purpose |
|---------|------|---------|
| Takhin | 9092 | Kafka protocol |
| Takhin Metrics | 9090 | Prometheus metrics |
| Console API | 8080 | REST API |
| Prometheus | 9091 | Metrics & queries |
| AlertManager | 9093 | Alert management |
| Grafana | 3000 | Visualization |
| Node Exporter | 9100 | System metrics |

## Common Commands

### Start/Stop Services
```bash
# Start all services
docker-compose -f docker-compose.monitoring.yml up -d

# Stop all services
docker-compose -f docker-compose.monitoring.yml down

# View logs
docker-compose -f docker-compose.monitoring.yml logs -f alertmanager

# Restart specific service
docker-compose -f docker-compose.monitoring.yml restart prometheus
```

### Alert Management
```bash
# List current alerts
curl http://localhost:9091/api/v1/alerts | jq

# Send test alert
curl -X POST http://localhost:9093/api/v2/alerts \
  -H "Content-Type: application/json" \
  -d '[{"labels":{"alertname":"Test","severity":"warning"}}]'

# Reload configurations
curl -X POST http://localhost:9091/-/reload
curl -X POST http://localhost:9093/-/reload
```

### Monitoring
```bash
# Check Prometheus targets
curl http://localhost:9091/api/v1/targets | jq

# Check AlertManager status
curl http://localhost:9093/api/v2/status | jq

# Query metrics
curl 'http://localhost:9091/api/v1/query?query=up' | jq
```

## Troubleshooting

### Alerts Not Firing
1. Check Prometheus is scraping: `curl http://localhost:9091/api/v1/targets`
2. Verify alert rules: `curl http://localhost:9091/api/v1/rules`
3. Check metric values: `curl 'http://localhost:9091/api/v1/query?query=up{job="takhin"}'`

### Notifications Not Received
1. Verify environment variables: `echo $SLACK_WEBHOOK_URL`
2. Check AlertManager status: `curl http://localhost:9093/api/v2/status`
3. View AlertManager logs: `docker-compose -f docker-compose.monitoring.yml logs alertmanager`
4. Send test alert: `./scripts/test-alerts.sh`

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

## Customization

### Adjust Alert Thresholds
Edit `docs/deployment/prometheus-alerts.yml`:
```yaml
- alert: TakhinReplicationLagHigh
  expr: takhin_replication_lag_offsets > 5000  # Adjust threshold
  for: 10m  # Adjust duration
```

### Add Custom Alert
Create new rule in `prometheus-alerts.yml`:
```yaml
- alert: CustomAlert
  expr: your_metric > threshold
  for: 5m
  labels:
    severity: warning
    category: custom
  annotations:
    summary: "Custom alert description"
    description: "Detailed description"
```

### Change Notification Frequency
Edit `docs/deployment/alertmanager.yml`:
```yaml
route:
  repeat_interval: 24h  # Reduce frequency
  group_interval: 10m   # Group longer
```

## Architecture

```
Takhin → Prometheus → AlertManager → Notifications
  ↓         ↓             ↓              ↓
Metrics   Rules       Routing      Slack/Email
```

## Performance

- **Prometheus**: ~100MB memory (30d retention, 50GB max)
- **AlertManager**: <50MB memory
- **Alert evaluation**: 15s interval
- **Scrape interval**: 15s (Takhin), 30s (system metrics)
- **Network overhead**: ~10KB/sec per target

## Support

For detailed information, see:
- Complete Guide: `TASK_5.5_ALERTING_COMPLETION.md`
- Quick Reference: `TASK_5.5_QUICK_REFERENCE.md`
- Visual Overview: `TASK_5.5_VISUAL_OVERVIEW.md`
- Acceptance Tests: `TASK_5.5_ACCEPTANCE_CHECKLIST.md`

For Takhin metrics documentation, see: `docs/metrics.md`

## License

Copyright 2025 Takhin Data, Inc.
