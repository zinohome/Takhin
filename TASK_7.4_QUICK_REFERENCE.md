# Task 7.4: User Manual - Quick Reference

## 📖 Document Locations

- **Main User Manual**: [docs/USER_MANUAL.md](docs/USER_MANUAL.md)
- **Task Index**: [TASK_7.4_USER_MANUAL_INDEX.md](TASK_7.4_USER_MANUAL_INDEX.md)
- **Completion Summary**: [TASK_7.4_COMPLETION_SUMMARY.md](TASK_7.4_COMPLETION_SUMMARY.md)

---

## 🚀 Quick Start (5 Minutes)

### Install Takhin
```bash
# Download binary
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-linux-amd64
chmod +x takhin-linux-amd64
sudo mv takhin-linux-amd64 /usr/local/bin/takhin

# Start server
takhin -config configs/takhin.yaml
```

### Create Your First Topic
```bash
# Create topic
kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --partitions 3

# Produce message
echo "Hello Takhin!" | kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic my-topic

# Consume message
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --from-beginning
```

---

## 📚 Manual Structure

### 1. Quick Start Guide
- What is Takhin?
- Five-minute setup
- Your first topic
- System requirements

### 2. Installation
- Pre-built binary
- Build from source
- Docker
- Takhin Console

### 3. Configuration
- Basic configuration
- Security (TLS, SASL, ACL, audit)
- Performance tuning
- Tiered storage (S3)
- Cluster setup

### 4. Feature Usage
- Topic management
- Producing messages
- Consuming messages
- Consumer groups
- Message browser
- Monitoring
- Transactions
- Compression

### 5. Best Practices
- Topic design
- Producer patterns
- Consumer patterns
- Performance optimization
- Security hardening
- Backup & DR
- Monitoring & alerting

### 6. FAQ (40+ Questions)
- General
- Configuration
- Topics
- Consumers
- Security
- Operations
- Troubleshooting

### 7. Troubleshooting
- Common issues
- Performance issues
- Cluster issues
- Getting help

---

## 🎯 Find What You Need

### I want to...

#### Get Started
→ **Section 1**: Quick Start Guide  
→ **Section 2**: Installation

#### Configure Takhin
→ **Section 3.2**: Basic Configuration  
→ **Section 3.3**: Security Configuration  
→ **Section 3.4**: Performance Tuning

#### Use Features
→ **Section 4.1**: Topic Management  
→ **Section 4.2**: Producing Messages  
→ **Section 4.3**: Consuming Messages  
→ **Section 4.4**: Consumer Groups  
→ **Section 4.5**: Message Browser

#### Monitor System
→ **Section 4.6**: Monitoring & Metrics  
→ **Section 5.7**: Monitoring Best Practices  
→ **Appendix C**: Grafana Dashboard

#### Optimize Performance
→ **Section 3.4**: Performance Tuning  
→ **Section 5.4**: Performance Optimization  
→ **Section 7.2**: Performance Issues

#### Secure Takhin
→ **Section 3.3**: Security Configuration  
→ **Section 5.5**: Security Best Practices  
→ **FAQ**: Security Questions

#### Deploy to Production
→ **Section 2**: Installation  
→ **Section 3.6**: Cluster Configuration  
→ **Section 5.6**: Backup & DR  
→ **External**: [Deployment Guide](docs/deployment/README.md)

#### Troubleshoot Issues
→ **Section 6**: FAQ  
→ **Section 7**: Troubleshooting  
→ **External**: [Troubleshooting Guide](docs/deployment/05-troubleshooting.md)

---

## 💡 Common Commands

### Server Management
```bash
# Start Takhin
takhin -config /etc/takhin/takhin.yaml

# Check version
takhin -version

# Check health
curl http://localhost:9091/health

# View metrics
curl http://localhost:9090/metrics
```

### Topic Operations
```bash
# List topics
kafka-topics.sh --list --bootstrap-server localhost:9092

# Create topic
kafka-topics.sh --create \
  --topic my-topic \
  --partitions 3 \
  --replication-factor 1 \
  --bootstrap-server localhost:9092

# Describe topic
kafka-topics.sh --describe \
  --topic my-topic \
  --bootstrap-server localhost:9092

# Delete topic
kafka-topics.sh --delete \
  --topic my-topic \
  --bootstrap-server localhost:9092
```

### Producer/Consumer
```bash
# Produce messages
kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic my-topic

# Consume messages
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --from-beginning

# Consume with group
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --group my-group
```

### Consumer Groups
```bash
# List consumer groups
kafka-consumer-groups.sh --list \
  --bootstrap-server localhost:9092

# Describe consumer group
kafka-consumer-groups.sh --describe \
  --group my-group \
  --bootstrap-server localhost:9092

# Reset offsets
kafka-consumer-groups.sh --reset-offsets \
  --group my-group \
  --topic my-topic \
  --to-earliest \
  --execute \
  --bootstrap-server localhost:9092
```

### REST API (Console)
```bash
# List topics
curl http://localhost:8080/api/v1/topics

# Create topic
curl -X POST http://localhost:8080/api/v1/topics \
  -H "Content-Type: application/json" \
  -d '{"name":"my-topic","partitions":3}'

# Produce message
curl -X POST http://localhost:8080/api/v1/topics/my-topic/messages \
  -H "Content-Type: application/json" \
  -d '{"key":"key1","value":"value1"}'

# Get messages
curl "http://localhost:8080/api/v1/topics/my-topic/messages?partition=0&offset=0&limit=10"

# List consumer groups
curl http://localhost:8080/api/v1/consumer-groups
```

---

## ⚙️ Configuration Shortcuts

### Environment Variables
```bash
# Override any config with TAKHIN_ prefix
export TAKHIN_SERVER_PORT=9093
export TAKHIN_STORAGE_DATA_DIR=/custom/path
export TAKHIN_LOGGING_LEVEL=debug
export TAKHIN_METRICS_ENABLED=true
```

### Quick Configs

**Development:**
```yaml
logging:
  level: "debug"
metrics:
  enabled: true
storage:
  data:
    dir: "/tmp/takhin-data"
```

**Production:**
```yaml
server:
  tls:
    enabled: true
sasl:
  enabled: true
  mechanisms: [SCRAM-SHA-256]
acl:
  enabled: true
storage:
  data:
    dir: "/var/lib/takhin/data"
  tiered:
    enabled: true
replication:
  default:
    replication:
      factor: 3
```

---

## 🔍 Troubleshooting Quick Fixes

### Port Already in Use
```bash
# Find process
sudo lsof -i :9092
# Kill it
sudo kill -9 <PID>
```

### Permission Denied
```bash
sudo chown -R takhin:takhin /var/lib/takhin/data
sudo chmod -R 755 /var/lib/takhin/data
```

### Consumer Lag
```bash
# Check lag
kafka-consumer-groups.sh --describe --group my-group

# Scale consumers (start more instances)
# Or reset offset:
kafka-consumer-groups.sh --reset-offsets --to-latest
```

### Enable Debug Logging
```bash
export TAKHIN_LOGGING_LEVEL=debug
sudo systemctl restart takhin
# View logs
sudo journalctl -u takhin -f
```

---

## 📊 Monitoring Quick Setup

### Prometheus Scrape Config
```yaml
scrape_configs:
  - job_name: 'takhin'
    static_configs:
      - targets: ['localhost:9090']
```

### Key Metrics to Watch
- `takhin_kafka_requests_total` - Request rate
- `takhin_kafka_request_duration_seconds` - Latency
- `takhin_consumer_lag_messages` - Consumer lag
- `takhin_storage_bytes_total` - Disk usage
- `takhin_active_connections` - Connection count

### Alert Rules
```yaml
- alert: HighConsumerLag
  expr: takhin_consumer_lag_messages > 1000
  
- alert: HighDiskUsage
  expr: takhin_storage_bytes_total / disk_total > 0.8
```

---

## 🎓 Learning Paths

### Path 1: Beginner (30 minutes)
1. Read Section 1.1 (What is Takhin)
2. Follow Section 1.2 (Five-minute setup)
3. Complete Section 1.3 (Your first topic)
4. Try Console UI at http://localhost:3000

### Path 2: Developer (2 hours)
1. Section 4.1 (Topics)
2. Section 4.2 (Producers)
3. Section 4.3 (Consumers)
4. Section 5.2-5.3 (Best practices)
5. Practice with code examples

### Path 3: Operator (4 hours)
1. Section 2 (Installation)
2. Section 3 (Configuration)
3. Section 4.6 (Monitoring)
4. Section 5.6 (Backup & DR)
5. Section 7 (Troubleshooting)

### Path 4: Advanced (1 day)
1. Section 3.4-3.6 (Advanced config)
2. Section 5.4 (Performance)
3. Section 5.5 (Security)
4. Section 5.7 (Alerting)
5. External: [Architecture docs](docs/architecture/)

---

## 📞 Get Help

### Documentation
- **Main Manual**: [docs/USER_MANUAL.md](docs/USER_MANUAL.md)
- **API Docs**: [docs/api/README.md](docs/api/README.md)
- **Deployment**: [docs/deployment/README.md](docs/deployment/README.md)
- **Architecture**: [docs/architecture/README.md](docs/architecture/README.md)

### Support Channels
- **Issues**: [GitHub Issues](https://github.com/takhin-data/takhin/issues)
- **Discussions**: [GitHub Discussions](https://github.com/takhin-data/takhin/discussions)
- **FAQ**: Section 6 of User Manual

### Reporting Issues
1. Check FAQ (Section 6)
2. Review Troubleshooting (Section 7)
3. Enable debug logging
4. Collect logs and config
5. Create GitHub issue with details

---

## 📈 Statistics

- **Total Lines**: 1,585 (main manual)
- **Sections**: 7 main + 3 appendices
- **Code Examples**: 50+
- **FAQ Entries**: 40+
- **Configuration Examples**: 20+
- **Commands**: 100+

---

## ✅ Task Status

**Status**: ✅ COMPLETE  
**Priority**: P1 - Medium  
**Deliverables**: 3 files (2,473 total lines)  
**Quality**: Production Ready

### Files Created
1. ✅ `docs/USER_MANUAL.md` (1,585 lines)
2. ✅ `TASK_7.4_USER_MANUAL_INDEX.md` (360 lines)
3. ✅ `TASK_7.4_COMPLETION_SUMMARY.md` (528 lines)
4. ✅ `TASK_7.4_QUICK_REFERENCE.md` (this file)

---

**Last Updated**: 2026-01-06  
**Version**: 1.0  
**For**: Takhin v1.0
