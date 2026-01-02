# Takhin Deployment & Operations Documentation

Complete guide for deploying and operating Takhin in production and development environments.

## Documentation Overview

This directory contains comprehensive deployment and operations documentation for Takhin, covering everything from single-node development setups to production-grade cluster deployments.

## Quick Start

- **New to Takhin?** Start with [Standalone Deployment](./01-standalone-deployment.md)
- **Production cluster?** See [Cluster Deployment](./02-cluster-deployment.md)
- **Using containers?** Check [Docker & Kubernetes](./03-docker-deployment.md)
- **Need configuration help?** Review [Configuration Reference](./04-configuration-reference.md)
- **Having issues?** Consult [Troubleshooting Guide](./05-troubleshooting.md)

## Documentation Structure

### 1. [Standalone Deployment Guide](./01-standalone-deployment.md)

**For:** Development, testing, small-scale production

**Covers:**
- Installation methods (source build, pre-built binary)
- Configuration setup
- systemd service configuration
- Running Takhin Console
- Verification and testing
- Performance tuning
- Backup and recovery
- Upgrading procedures

**Time to deploy:** ~15-30 minutes

---

### 2. [Cluster Deployment Guide](./02-cluster-deployment.md)

**For:** Production environments requiring high availability

**Covers:**
- Multi-node cluster architecture with Raft consensus
- 3+ node deployment (5 recommended for production)
- Replication configuration
- Load balancer setup (HAProxy, nginx)
- Cluster testing and failover
- Scaling operations (adding/removing brokers)
- Disaster recovery procedures
- Security hardening

**Time to deploy:** ~2-4 hours

---

### 3. [Docker & Kubernetes Deployment](./03-docker-deployment.md)

**For:** Container-based deployments and cloud-native environments

**Covers:**
- Dockerfile creation and image building
- Docker Compose for standalone and cluster modes
- Kubernetes StatefulSet deployment
- ConfigMaps and Secrets management
- Persistent volume configuration
- Console deployment in K8s
- Helm chart deployment
- Monitoring with Prometheus ServiceMonitor
- Scaling and backup operations

**Time to deploy:** ~1-3 hours (depending on K8s experience)

---

### 4. [Configuration Reference](./04-configuration-reference.md)

**Complete reference for all configuration options**

**Sections:**
- Configuration loading order (YAML + environment variables)
- Server configuration
- Kafka protocol settings
- Storage and retention policies
- Replication configuration
- Logging and metrics
- Console server configuration
- Configuration examples (dev, production, high-throughput)
- Performance tuning guide
- Validation and troubleshooting

**Use as:** Reference guide and configuration template source

---

### 5. [Troubleshooting Guide](./05-troubleshooting.md)

**Comprehensive problem-solving guide**

**Covers:**
- Quick diagnostic commands
- 8 most common issues with solutions:
  1. Broker won't start
  2. Cannot connect to broker
  3. High CPU usage
  4. High memory usage / OOM errors
  5. Data loss / missing messages
  6. Slow performance / high latency
  7. Cluster issues
  8. Console API issues
- Debugging tools (pprof, tcpdump, stress testing)
- Log analysis patterns
- Diagnostic information collection
- Preventive measures and monitoring checklist

**Use when:** Encountering issues or performing maintenance

---

## Deployment Decision Matrix

| Scenario | Recommended Deployment | Documentation |
|----------|----------------------|---------------|
| Local development | Standalone (Task runner) | [01-standalone](./01-standalone-deployment.md#development-mode) |
| CI/CD testing | Docker single container | [03-docker](./03-docker-deployment.md#running-single-container) |
| Small production (<1K msg/s) | Standalone with systemd | [01-standalone](./01-standalone-deployment.md#production-mode---systemd-service) |
| Medium production (1K-10K msg/s) | 3-node cluster | [02-cluster](./02-cluster-deployment.md) |
| Large production (>10K msg/s) | 5+ node cluster + tuning | [02-cluster](./02-cluster-deployment.md) + [04-config](./04-configuration-reference.md#high-throughput) |
| Cloud-native | Kubernetes StatefulSet | [03-docker](./03-docker-deployment.md#kubernetes-deployment) |
| Multi-cloud | K8s + Helm | [03-docker](./03-docker-deployment.md#helm-chart-deployment) |

## Prerequisites by Deployment Type

### Standalone Deployment
- Linux/macOS operating system
- 2GB+ RAM
- 10GB+ disk space
- Go 1.21+ (if building from source)

### Cluster Deployment
- 3+ Linux servers
- 8GB+ RAM per server
- 100GB+ SSD per server
- Low-latency network (<10ms RTT)

### Docker Deployment
- Docker 20.10+
- Docker Compose 2.0+ (for multi-container)
- 4GB+ RAM available to Docker
- 20GB+ disk space

### Kubernetes Deployment
- Kubernetes 1.24+
- kubectl configured
- StorageClass with dynamic provisioning
- 3+ worker nodes (for cluster mode)
- Helm 3.x (optional)

## Common Configuration Patterns

### Development Environment

```yaml
# Minimal config for local development
server:
  host: "127.0.0.1"
  port: 9092

kafka:
  broker:
    id: 1
  cluster:
    brokers: [1]

storage:
  data:
    dir: "/tmp/takhin-dev"

logging:
  level: "debug"
  format: "text"
```

**Start with:** `task backend:run`

### Production Cluster (3 Nodes)

```yaml
# High-availability configuration
kafka:
  broker:
    id: 1  # Change per node: 1, 2, 3
  cluster:
    brokers: [1, 2, 3]
  advertised:
    host: "broker1.prod.example.com"  # Change per node

replication:
  default:
    replication:
      factor: 3

storage:
  data:
    dir: "/data/takhin"
  log:
    segment:
      size: 2147483648  # 2GB

logging:
  level: "info"
  format: "json"
```

**Deploy with:** systemd service + load balancer

### Container Environment (Docker/K8s)

```yaml
# Container-optimized configuration
server:
  host: "0.0.0.0"

kafka:
  advertised:
    host: "${POD_NAME}.takhin-headless.takhin.svc.cluster.local"

storage:
  data:
    dir: "/var/lib/takhin/data"

logging:
  format: "json"  # For log aggregation
```

**Deploy with:** StatefulSet + PersistentVolumeClaim

## Port Reference

| Port | Service | Protocol | Purpose |
|------|---------|----------|---------|
| 9092 | Kafka API | TCP | Kafka protocol communication |
| 9090 | Metrics | HTTP | Prometheus metrics endpoint |
| 8080 | Console API | HTTP | REST API for management |
| 7946 | Raft | TCP | Cluster coordination (internal) |

## Environment Variables Quick Reference

All configuration can be overridden with `TAKHIN_` prefixed environment variables:

```bash
# Common overrides
export TAKHIN_SERVER_PORT=9093
export TAKHIN_STORAGE_DATA_DIR=/custom/path
export TAKHIN_LOGGING_LEVEL=debug
export TAKHIN_KAFKA_BROKER_ID=2
export TAKHIN_KAFKA_CLUSTER_BROKERS='[1,2,3]'
export TAKHIN_KAFKA_ADVERTISED_HOST=broker2.example.com
```

**Format:** `TAKHIN_<SECTION>_<SUBSECTION>_<KEY>`
- Dots (`.`) become underscores (`_`)
- Example: `kafka.broker.id` → `TAKHIN_KAFKA_BROKER_ID`

## Health Check Endpoints

### Broker Health

```bash
# Metrics endpoint (always available)
curl http://localhost:9090/metrics

# Key metrics to check
curl http://localhost:9090/metrics | grep -E 'takhin_active_connections|takhin_kafka_requests_total'
```

### Console Health

```bash
# Health check endpoint
curl http://localhost:8080/health

# Expected response
{"status":"ok","timestamp":"2026-01-02T04:37:30Z"}
```

## Monitoring Integration

Takhin exposes Prometheus-compatible metrics at `/metrics` endpoint:

```yaml
# Prometheus scrape configuration
scrape_configs:
  - job_name: 'takhin'
    static_configs:
      - targets: ['broker1:9090', 'broker2:9090', 'broker3:9090']
    scrape_interval: 30s
```

**Key metrics to monitor:**
- `takhin_kafka_requests_total` - Request count by API
- `takhin_kafka_request_duration_seconds` - Latency distribution
- `takhin_storage_bytes_total` - Storage usage
- `takhin_active_connections` - Active connections
- `takhin_raft_state` - Cluster state (leader/follower)

## Backup Best Practices

### Standalone
1. Stop Takhin service
2. Backup `/var/lib/takhin/data`
3. Backup `/etc/takhin/takhin.yaml`
4. Restart service

### Cluster
1. Ensure all brokers healthy (no need to stop)
2. Backup data directory on each node
3. Backup configurations
4. Test restore on non-production cluster

### Kubernetes
1. Use VolumeSnapshot CRD
2. Snapshot PersistentVolumes
3. Backup ConfigMaps and Secrets
4. Test restore procedure

## Security Considerations

### Network Security
- Restrict port 9092 to application subnet
- Restrict port 7946 to cluster nodes only
- Use TLS/SSL for production (future feature)
- Deploy behind VPC/firewall

### Authentication
- Console API supports API key authentication
- Enable with `-enable-auth` flag
- Rotate API keys regularly
- Store keys in secrets management (Vault, K8s Secrets)

### File System Security
- Run as non-root user (`takhin` user)
- Restrict data directory permissions (0700)
- Use encrypted volumes for data at rest
- Regular security patches for OS

## Performance Optimization

### For High Throughput
1. Increase `max.message.bytes` to 10MB
2. Increase `log.segment.size` to 2GB
3. Use SSD storage
4. Tune `log.flush.interval.ms` to 5000+
5. Scale horizontally (add brokers)

### For Low Latency
1. Decrease `log.flush.interval.ms` to 100-500ms
2. Use NVMe SSD storage
3. Deploy in same availability zone
4. Optimize network (10Gbps+)
5. Reduce replication factor to 2

### For High Availability
1. Deploy 5-node cluster (tolerates 2 failures)
2. Set replication factor to 3
3. Use multiple availability zones
4. Deploy load balancer with health checks
5. Configure aggressive monitoring/alerting

## Common Operational Tasks

### Add Broker to Cluster
1. Provision new server
2. Configure with unique broker ID
3. Update all broker configs with new ID
4. Restart existing brokers
5. Start new broker
6. Rebalance partitions

### Upgrade Takhin Version
1. Download new binary
2. Backup current binary
3. Replace binary
4. Restart service (one broker at a time for clusters)
5. Verify metrics/logs

### Change Configuration
1. Edit `/etc/takhin/takhin.yaml`
2. Validate configuration
3. Restart service
4. Verify change in metrics/logs

### Rotate Logs
1. Configure logrotate (example in [01-standalone](./01-standalone-deployment.md#log-management))
2. Or use systemd's built-in rotation
3. Monitor disk usage

## Getting Help

### Documentation
- **Architecture:** [../architecture/](../architecture/)
- **Implementation:** [../implementation/](../implementation/)
- **Testing:** [../testing/](../testing/)
- **Project README:** [../../README.md](../../README.md)

### Support Channels
- **GitHub Issues:** https://github.com/takhin-data/takhin/issues
- **GitHub Discussions:** https://github.com/takhin-data/takhin/discussions

### Reporting Bugs
When reporting issues, include:
1. Takhin version
2. Deployment type (standalone/cluster/docker/k8s)
3. Configuration file (redacted)
4. Recent logs (last 100 lines)
5. Steps to reproduce
6. Expected vs actual behavior

Use the diagnostic script from [Troubleshooting Guide](./05-troubleshooting.md#collect-diagnostic-information)

## Contributing

Found an issue with documentation? Want to add deployment examples?

1. Fork the repository
2. Create feature branch
3. Make changes
4. Submit pull request

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

Takhin is open-source software. See [LICENSE](../../LICENSE) for details.

---

**Last Updated:** 2026-01-02

**Documentation Version:** 1.0.0

**Compatible Takhin Versions:** 1.0.0+
