# Takhin User Manual

**Version**: 1.0  
**Last Updated**: 2026-01-06  
**Document Status**: Production Ready

Welcome to Takhin, a high-performance Kafka-compatible streaming platform built with Go. This manual guides you through installing, configuring, and using Takhin for your streaming data needs.

---

## Table of Contents

1. [Quick Start Guide](#1-quick-start-guide)
2. [Installation](#2-installation)
3. [Configuration](#3-configuration)
4. [Feature Usage](#4-feature-usage)
5. [Best Practices](#5-best-practices)
6. [FAQ](#6-faq)
7. [Troubleshooting](#7-troubleshooting)

---

## 1. Quick Start Guide

### 1.1 What is Takhin?

Takhin is a modern streaming platform that provides:
- **Kafka Protocol Compatibility**: Drop-in replacement for Apache Kafka
- **High Performance**: 100K+ messages/second with P99 latency < 10ms
- **Zero Dependencies**: No ZooKeeper needed (uses Raft consensus)
- **Easy Management**: Built-in web console for administration
- **Enterprise Features**: Encryption, authentication, audit logging, and tiered storage

### 1.2 Five-Minute Setup

Get Takhin running in 5 minutes:

```bash
# 1. Download and install
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-linux-amd64
chmod +x takhin-linux-amd64
sudo mv takhin-linux-amd64 /usr/local/bin/takhin

# 2. Create data directory
mkdir -p /tmp/takhin-data

# 3. Start server
takhin -config configs/takhin.yaml

# 4. Verify installation
curl http://localhost:9090/metrics
```

### 1.3 Your First Topic

Create your first topic and send messages:

```bash
# Using kafka-console-producer (Apache Kafka tools)
echo "Hello Takhin!" | kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic quickstart-topic

# Consume messages
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic quickstart-topic \
  --from-beginning
```

### 1.4 System Requirements

**Minimum:**
- OS: Linux (Ubuntu 20.04+), macOS, or Windows (WSL2)
- CPU: 2 cores
- RAM: 2GB
- Disk: 10GB

**Recommended for Production:**
- OS: Linux (Ubuntu 22.04+ or RHEL 8+)
- CPU: 8+ cores
- RAM: 16GB+
- Disk: 500GB+ SSD
- Network: 1Gbps+

---

## 2. Installation

### 2.1 Installation Methods

#### Method 1: Pre-built Binary (Recommended)

Download from GitHub releases:

```bash
# Linux AMD64
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-linux-amd64
chmod +x takhin-linux-amd64
sudo mv takhin-linux-amd64 /usr/local/bin/takhin

# Linux ARM64
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-linux-arm64
chmod +x takhin-linux-arm64
sudo mv takhin-linux-arm64 /usr/local/bin/takhin

# macOS
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-darwin-amd64
chmod +x takhin-darwin-amd64
sudo mv takhin-darwin-amd64 /usr/local/bin/takhin
```

#### Method 2: Build from Source

```bash
# Prerequisites: Go 1.23+, Task
git clone https://github.com/takhin-data/takhin.git
cd takhin

# Install dependencies
task backend:deps

# Build
task backend:build

# Binary created at: build/takhin
sudo mv build/takhin /usr/local/bin/
```

#### Method 3: Docker

```bash
# Pull image
docker pull ghcr.io/takhin-data/takhin:latest

# Run container
docker run -d \
  --name takhin \
  -p 9092:9092 \
  -p 9090:9090 \
  -v /var/lib/takhin:/data \
  ghcr.io/takhin-data/takhin:latest
```

### 2.2 Verify Installation

```bash
# Check version
takhin -version

# Expected output:
# Takhin version v1.0.0 (commit: abc1234)
```

### 2.3 Installing Takhin Console

The console provides a web UI for management:

```bash
# Build console binary
cd backend
go build -o /usr/local/bin/takhin-console ./cmd/console

# Or download from releases
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-console-linux-amd64
chmod +x takhin-console-linux-amd64
sudo mv takhin-console-linux-amd64 /usr/local/bin/takhin-console
```

---

## 3. Configuration

### 3.1 Configuration Overview

Takhin uses a layered configuration system:

1. **YAML file**: Base configuration (`configs/takhin.yaml`)
2. **Environment variables**: Override YAML settings (prefix: `TAKHIN_`)
3. **Command-line flags**: Override both YAML and env vars

Example:
```bash
# Override data directory
export TAKHIN_STORAGE_DATA_DIR=/custom/path

# Or use command-line flag
takhin -config configs/takhin.yaml -data-dir /custom/path
```

### 3.2 Basic Configuration

Create `/etc/takhin/takhin.yaml`:

```yaml
# Server Configuration
server:
  host: "0.0.0.0"
  port: 9092

# Kafka Protocol Configuration
kafka:
  broker:
    id: 1
  cluster:
    brokers: [1]  # Single broker mode
  advertised:
    host: "localhost"  # Change to your server hostname
    port: 9092

# Storage Configuration
storage:
  data:
    dir: "/var/lib/takhin/data"
  log:
    segment:
      size: 1073741824    # 1GB
    retention:
      hours: 168          # 7 days

# Logging Configuration
logging:
  level: "info"
  format: "json"

# Metrics Configuration
metrics:
  enabled: true
  port: 9090
```

### 3.3 Security Configuration

#### Enable TLS Encryption

```yaml
server:
  tls:
    enabled: true
    cert:
      file: "/etc/takhin/certs/server.crt"
    key:
      file: "/etc/takhin/certs/server.key"
    client:
      auth: "require"  # Require client certificates
    verify:
      client:
        cert: true
```

#### Enable SASL Authentication

```yaml
sasl:
  enabled: true
  mechanisms:
    - PLAIN
    - SCRAM-SHA-256
    - SCRAM-SHA-512
  plain:
    users: "/etc/takhin/users.properties"
  cache:
    enabled: true
    ttl:
      seconds: 3600
```

#### Enable ACL Authorization

```yaml
acl:
  enabled: true
```

#### Enable Audit Logging

```yaml
audit:
  enabled: true
  output:
    path: "/var/log/takhin/audit.log"
  max:
    file:
      size: 104857600  # 100MB
    backups: 10
    age: 30
  compress: true
```

### 3.4 Performance Tuning

#### High-Throughput Configuration

```yaml
kafka:
  max:
    message:
      bytes: 10485760      # 10MB
    connections: 5000

storage:
  log:
    segment:
      size: 2147483648     # 2GB segments
    flush:
      interval:
        ms: 5000           # Less frequent flushes
      messages: 50000

# Disable if not needed
metrics:
  enabled: false
```

#### Low-Latency Configuration

```yaml
storage:
  log:
    flush:
      interval:
        ms: 100            # Flush every 100ms
      messages: 1000       # Flush after 1K messages

raft:
  heartbeat:
    timeout:
      ms: 500              # Faster heartbeats
  commit:
    timeout:
      ms: 25               # Faster commits
```

### 3.5 Tiered Storage (S3)

Enable tiered storage for cost-effective archiving:

```yaml
storage:
  tiered:
    enabled: true
    s3:
      bucket: "my-takhin-bucket"
      region: "us-east-1"
      prefix: "takhin-segments"
      endpoint: ""  # Leave empty for AWS S3
    cold:
      age:
        hours: 168  # Archive segments older than 7 days
    local:
      cache:
        size:
          mb: 10240  # 10GB local cache
```

### 3.6 Cluster Configuration

For multi-broker clusters:

```yaml
kafka:
  broker:
    id: 1  # Unique per broker
  cluster:
    brokers: [1, 2, 3]  # All broker IDs
  advertised:
    host: "broker1.example.com"
    port: 9092

replication:
  default:
    replication:
      factor: 3  # Replicate to 3 brokers
```

---

## 4. Feature Usage

### 4.1 Working with Topics

#### Create a Topic

**Using Kafka CLI:**
```bash
kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --partitions 3 \
  --replication-factor 1
```

**Using Console REST API:**
```bash
curl -X POST http://localhost:8080/api/v1/topics \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-topic",
    "partitions": 3,
    "replicationFactor": 1
  }'
```

**Using Takhin Console UI:**
1. Navigate to http://localhost:3000/topics
2. Click "Create Topic" button
3. Fill in topic name and partition count
4. Click "Create"

#### List Topics

```bash
# Kafka CLI
kafka-topics.sh --list \
  --bootstrap-server localhost:9092

# REST API
curl http://localhost:8080/api/v1/topics

# Console UI: Navigate to /topics
```

#### Delete a Topic

```bash
# Kafka CLI
kafka-topics.sh --delete \
  --bootstrap-server localhost:9092 \
  --topic my-topic

# REST API
curl -X DELETE http://localhost:8080/api/v1/topics/my-topic

# Console UI: Click trash icon on topic row
```

#### View Topic Details

```bash
# Kafka CLI
kafka-topics.sh --describe \
  --bootstrap-server localhost:9092 \
  --topic my-topic

# REST API
curl http://localhost:8080/api/v1/topics/my-topic

# Console UI: Click topic name in table
```

### 4.2 Producing Messages

#### Simple Producer

```bash
# Interactive console producer
kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic my-topic

# Then type messages (one per line)
```

#### Producer with Keys

```bash
kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic my-topic \
  --property parse.key=true \
  --property key.separator=:

# Format: key:value
user123:{"name":"Alice","age":30}
```

#### Producer with Compression

**Go Client:**
```go
writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "my-topic",
    CompressionCodec: &kafka.GzipCodec,
    BatchSize: 100,
    BatchTimeout: 10 * time.Millisecond,
})

writer.WriteMessages(context.Background(),
    kafka.Message{
        Key:   []byte("key1"),
        Value: []byte("value1"),
    },
)
```

#### REST API Producer

```bash
curl -X POST http://localhost:8080/api/v1/topics/my-topic/messages \
  -H "Content-Type: application/json" \
  -d '{
    "partition": 0,
    "key": "user123",
    "value": "{\"name\":\"Alice\"}"
  }'
```

### 4.3 Consuming Messages

#### Simple Consumer

```bash
# Consume from beginning
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --from-beginning

# Consume only new messages
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic my-topic
```

#### Consumer with Consumer Group

```bash
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --group my-consumer-group \
  --from-beginning
```

#### Consumer with Key Display

```bash
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic my-topic \
  --from-beginning \
  --property print.key=true \
  --property key.separator=" : "
```

#### Go Consumer Example

```go
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers:  []string{"localhost:9092"},
    Topic:    "my-topic",
    GroupID:  "my-consumer-group",
    MinBytes: 10e3, // 10KB
    MaxBytes: 10e6, // 10MB
})

for {
    msg, err := reader.ReadMessage(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Key: %s, Value: %s\n", 
        string(msg.Key), string(msg.Value))
}
```

### 4.4 Consumer Group Management

#### List Consumer Groups

```bash
# Kafka CLI
kafka-consumer-groups.sh --list \
  --bootstrap-server localhost:9092

# REST API
curl http://localhost:8080/api/v1/consumer-groups

# Console UI: Navigate to /consumers
```

#### Describe Consumer Group

```bash
# Kafka CLI
kafka-consumer-groups.sh --describe \
  --bootstrap-server localhost:9092 \
  --group my-consumer-group

# REST API
curl http://localhost:8080/api/v1/consumer-groups/my-consumer-group

# Console UI: Click group ID in consumer list
```

#### Monitor Consumer Lag

**Console UI (Recommended):**
1. Navigate to http://localhost:3000/consumers
2. Enable auto-refresh toggle (updates every 5 seconds)
3. View lag in "Total Lag" column
4. Click group ID for detailed lag per partition

**Color Coding:**
- 🟢 Green: lag < 100 (Healthy)
- 🟠 Orange: 100 ≤ lag < 1000 (Warning)
- 🔴 Red: lag ≥ 1000 (Critical)

### 4.5 Message Browser

The message browser allows you to explore topic messages through the web UI.

#### Access Message Browser

1. Navigate to http://localhost:3000/topics
2. Click "Messages" button on any topic
3. Opens at: `/topics/{topicName}/messages`

#### Browse Messages

**Step-by-step:**
1. Click "Filter" button
2. Select partition (required)
3. Optionally set offset range (start/end)
4. Optionally set time range
5. Click "Apply & Load"
6. Messages display in table

#### Search Messages

**By Key:**
1. Click "Filter"
2. Enter search term in "Search by Key"
3. Click "Apply & Load"
4. Displays messages with matching keys (case-insensitive)

**By Value:**
1. Click "Filter"
2. Enter search term in "Search by Value"
3. Click "Apply & Load"
4. Displays messages with matching values (case-insensitive)

#### View Message Details

1. Click "View" button on any message row
2. Detail drawer opens showing:
   - Partition and offset
   - Timestamp (formatted and Unix ms)
   - Key
   - Value (pretty-printed JSON if applicable)

#### Export Messages

**Export All:**
1. Load messages with desired filter
2. Click "Export" button in toolbar
3. Downloads: `{topicName}_messages_{timestamp}.json`

**Export Single Message:**
1. Click "View" on message row
2. In detail drawer, click "Export Message"
3. Downloads: `message_{partition}_{offset}.json`

### 4.6 Monitoring and Metrics

#### Prometheus Metrics

Takhin exposes Prometheus metrics at `http://localhost:9090/metrics`:

```bash
# Scrape metrics
curl http://localhost:9090/metrics
```

**Key Metrics:**
- `takhin_kafka_requests_total` - Total requests by API
- `takhin_kafka_request_duration_seconds` - Request latency
- `takhin_storage_bytes_total` - Storage usage
- `takhin_active_connections` - Active client connections
- `takhin_consumer_lag_messages` - Consumer lag per group/topic

#### Health Checks

```bash
# Basic health check
curl http://localhost:9091/health

# Response: {"status":"healthy","timestamp":"2026-01-06T10:00:00Z"}

# Readiness check (for Kubernetes)
curl http://localhost:9091/health/ready

# Liveness check
curl http://localhost:9091/health/live
```

#### Grafana Dashboard

Import pre-built Grafana dashboards:

1. Install Grafana and Prometheus
2. Configure Prometheus to scrape Takhin metrics
3. Import dashboard from `docs/examples/grafana/takhin-dashboard.json`
4. View real-time metrics

See [Monitoring Guide](./monitoring/README.md) for details.

### 4.7 Transactions (Exactly-Once Semantics)

Enable transactional producers:

```go
// Initialize transactional producer
writer := &kafka.Writer{
    Addr:                   kafka.TCP("localhost:9092"),
    Topic:                  "my-topic",
    RequiredAcks:           kafka.RequireAll,
    MaxAttempts:            3,
    Idempotent:             true,
    TransactionalID:        "my-transaction-id",
}

// Begin transaction
err := writer.BeginTransaction()
if err != nil {
    log.Fatal(err)
}

// Write messages
err = writer.WriteMessages(context.Background(),
    kafka.Message{Value: []byte("msg1")},
    kafka.Message{Value: []byte("msg2")},
)

// Commit transaction
err = writer.CommitTransaction()
if err != nil {
    // Rollback on error
    writer.AbortTransaction()
}
```

### 4.8 Compression

Takhin supports 5 compression algorithms:

**Supported Codecs:**
- None (no compression)
- GZIP (balanced)
- Snappy (fast)
- LZ4 (fastest)
- ZSTD (best compression)

**Configuration:**
```go
writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "my-topic",
    CompressionCodec: &kafka.ZstdCodec,  // Use ZSTD
})
```

**When to Use:**
- **GZIP**: General purpose, good compression
- **Snappy**: Network-bound scenarios
- **LZ4**: CPU-bound scenarios, lowest latency
- **ZSTD**: Storage-bound scenarios, best compression

---

## 5. Best Practices

### 5.1 Topic Design

#### Partition Count

**Guidelines:**
- Start with: `num_consumers` or `num_cores`
- Max recommended: 100 partitions per topic
- Consider: Consumer parallelism needs

**Example:**
```bash
# For 10 consumers processing in parallel
kafka-topics.sh --create \
  --topic orders \
  --partitions 10 \
  --replication-factor 3
```

#### Naming Conventions

**Recommended format:**
```
{environment}.{team}.{domain}.{entity}.{version}

Examples:
- prod.payments.transactions.completed.v1
- staging.analytics.events.user-clicked.v2
- dev.logging.application.errors.v1
```

**Avoid:**
- Special characters (except `.`, `-`, `_`)
- Spaces
- Very long names (>249 characters)

#### Retention Policy

```yaml
# Time-based retention
storage:
  log:
    retention:
      hours: 168  # 7 days

# Size-based retention
storage:
  log:
    retention:
      bytes: 107374182400  # 100GB per partition
```

**Guidelines:**
- **Transactional data**: 7-30 days
- **Event logs**: 30-90 days
- **Audit logs**: 365+ days (with tiered storage)
- **CDC streams**: Match source database retention

### 5.2 Producer Best Practices

#### Batching

```go
writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers:      []string{"localhost:9092"},
    Topic:        "my-topic",
    BatchSize:    100,              // Messages per batch
    BatchTimeout: 10 * time.Millisecond,  // Max wait time
})
```

**Guidelines:**
- Increase `BatchSize` for higher throughput
- Decrease `BatchTimeout` for lower latency
- Balance based on your use case

#### Error Handling

```go
err := writer.WriteMessages(ctx, messages...)
if err != nil {
    if errors.Is(err, kafka.LeaderNotAvailable) {
        // Retry with backoff
        time.Sleep(500 * time.Millisecond)
        err = writer.WriteMessages(ctx, messages...)
    } else {
        // Log and handle unrecoverable errors
        log.Printf("Failed to write: %v", err)
    }
}
```

#### Idempotence

Always enable idempotent producers:

```go
writer := kafka.NewWriter(kafka.WriterConfig{
    Idempotent:   true,
    RequiredAcks: kafka.RequireAll,
    MaxAttempts:  5,
})
```

### 5.3 Consumer Best Practices

#### Consumer Group Sizing

**Rule of thumb:**
```
num_consumers_in_group ≤ num_partitions
```

**Why:** Each partition assigned to at most one consumer in a group.

**Example:**
- Topic has 12 partitions
- Optimal: 12 consumers (1:1 mapping)
- Acceptable: 6 consumers (2 partitions each)
- Wasteful: 24 consumers (12 idle)

#### Offset Management

```go
reader := kafka.NewReader(kafka.ReaderConfig{
    CommitInterval: 5 * time.Second,  // Commit every 5 seconds
    StartOffset:    kafka.LastOffset, // Or: kafka.FirstOffset
})

// Manual commit for critical data
msg, _ := reader.FetchMessage(ctx)
processMessage(msg)
reader.CommitMessages(ctx, msg)  // Commit after processing
```

#### Rebalance Handling

```go
reader := kafka.NewReader(kafka.ReaderConfig{
    SessionTimeout:  10 * time.Second,
    HeartbeatInterval: 3 * time.Second,
    RebalanceTimeout: 30 * time.Second,
})
```

**During rebalancing:**
- Consumers stop fetching
- Partitions reassigned
- Consumers resume with new assignment

**Minimize impact:**
- Keep processing fast
- Send heartbeats regularly
- Use shorter session timeouts (with care)

### 5.4 Performance Optimization

#### Operating System Tuning

```bash
# Increase file descriptors
echo "* soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "* hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# Network buffer sizes
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.wmem_max=134217728
sudo sysctl -w net.ipv4.tcp_rmem="4096 87380 67108864"
sudo sysctl -w net.ipv4.tcp_wmem="4096 65536 67108864"

# Disable swap
sudo swapoff -a

# Use deadline I/O scheduler for SSDs
echo deadline | sudo tee /sys/block/sda/queue/scheduler
```

#### Storage Optimization

**Use SSDs:**
- 10x faster than HDDs for Takhin workloads
- Lower latency for writes and reads

**Separate data and logs:**
```yaml
storage:
  data:
    dir: "/mnt/ssd/takhin/data"

logging:
  output:
    path: "/var/log/takhin/takhin.log"
```

**Monitor disk usage:**
```bash
# Check disk space
df -h /var/lib/takhin/data

# Check inode usage
df -i /var/lib/takhin/data
```

#### Memory Configuration

**JVM-like tuning not needed** (Go's GC handles memory automatically)

**Monitor memory usage:**
```bash
# Check Takhin memory usage
ps aux | grep takhin

# Expected: 100MB-2GB depending on workload
```

### 5.5 Security Best Practices

#### Enable TLS

Always use TLS in production:

```yaml
server:
  tls:
    enabled: true
    cert:
      file: "/etc/takhin/certs/server.crt"
    key:
      file: "/etc/takhin/certs/server.key"
    min:
      version: "TLS1.2"  # Minimum TLS 1.2
```

#### Enable Authentication

```yaml
sasl:
  enabled: true
  mechanisms: [SCRAM-SHA-256]  # Prefer SCRAM over PLAIN
```

#### Enable Authorization (ACL)

```yaml
acl:
  enabled: true
```

**Create ACLs:**
```bash
# Grant read access
kafka-acls.sh --authorizer-properties \
  --add \
  --allow-principal User:alice \
  --operation Read \
  --topic my-topic

# Grant write access
kafka-acls.sh --authorizer-properties \
  --add \
  --allow-principal User:bob \
  --operation Write \
  --topic my-topic
```

#### Network Security

**Firewall rules:**
```bash
# Allow only necessary ports
sudo ufw allow 9092/tcp  # Kafka protocol
sudo ufw allow 9090/tcp  # Metrics (internal only)
sudo ufw allow 8080/tcp  # Console API (internal only)
```

**Use private networks** for broker-to-broker communication.

### 5.6 Backup and Disaster Recovery

#### Data Backup Strategy

**1. Regular snapshots:**
```bash
#!/bin/bash
# Backup script
DATE=$(date +%Y%m%d_%H%M%S)
tar -czf takhin-backup-$DATE.tar.gz /var/lib/takhin/data
aws s3 cp takhin-backup-$DATE.tar.gz s3://backups/takhin/
```

**2. Replication:**
- Use `replication.factor >= 3` for critical topics
- Spread replicas across availability zones

**3. Tiered storage:**
- Enable S3 archiving for long-term retention
- Automatic failover to S3 if local disk fails

#### Recovery Procedures

**Restore from backup:**
```bash
# Stop Takhin
sudo systemctl stop takhin

# Restore data
sudo rm -rf /var/lib/takhin/data/*
sudo tar -xzf takhin-backup-20260106.tar.gz -C /

# Fix permissions
sudo chown -R takhin:takhin /var/lib/takhin/data

# Start Takhin
sudo systemctl start takhin
```

**Recover from replica:**
- If one broker fails, data preserved on other brokers
- Replace failed broker, it automatically syncs from leader

### 5.7 Monitoring and Alerting

#### Critical Metrics to Monitor

| Metric | Threshold | Action |
|--------|-----------|--------|
| Consumer Lag | > 1000 | Scale consumers |
| Disk Usage | > 80% | Add storage or reduce retention |
| Under-replicated Partitions | > 0 | Check broker health |
| Request Latency (P99) | > 100ms | Investigate performance |
| Error Rate | > 1% | Check logs |

#### Set Up Alerts

**Prometheus AlertManager:**
```yaml
# prometheus-alerts.yml
groups:
  - name: takhin
    rules:
      - alert: HighConsumerLag
        expr: takhin_consumer_lag_messages > 1000
        for: 5m
        annotations:
          summary: "High consumer lag detected"
```

See [Alerting Guide](./deployment/ALERTING_README.md) for complete setup.

---

## 6. FAQ

### General Questions

**Q: Is Takhin compatible with Apache Kafka?**  
A: Yes, Takhin implements the Kafka wire protocol and is compatible with all standard Kafka clients (kafka-go, kafka-python, KafkaJS, etc.).

**Q: Can I migrate from Apache Kafka to Takhin?**  
A: Yes, simply update the bootstrap server address in your clients. No code changes required.

**Q: Does Takhin require ZooKeeper?**  
A: No, Takhin uses Raft consensus for coordination, eliminating the need for ZooKeeper.

**Q: What is the performance comparison to Kafka?**  
A: Takhin achieves 100K+ msg/s with P99 latency < 10ms, comparable to or better than Kafka in many scenarios.

**Q: Is Takhin production-ready?**  
A: Yes, Takhin v1.0 is production-ready with comprehensive testing and documentation.

### Configuration Questions

**Q: How do I change the data directory?**  
A: Set `storage.data.dir` in YAML or use environment variable:
```bash
export TAKHIN_STORAGE_DATA_DIR=/custom/path
```

**Q: Can I use environment variables for all config?**  
A: Yes, prefix any YAML key with `TAKHIN_` and replace dots with underscores:
```bash
TAKHIN_KAFKA_BROKER_ID=1
TAKHIN_SERVER_PORT=9093
```

**Q: How do I enable debug logging?**  
A: Set in config:
```yaml
logging:
  level: "debug"
```
Or via environment:
```bash
export TAKHIN_LOGGING_LEVEL=debug
```

### Topic Questions

**Q: What's the maximum message size?**  
A: Default is 1MB, configurable up to 10MB:
```yaml
kafka:
  max:
    message:
      bytes: 10485760  # 10MB
```

**Q: How many partitions should I create?**  
A: Start with number of consumers or CPU cores, max 100 per topic.

**Q: Can I add partitions to existing topic?**  
A: No, partition count is immutable. Create new topic with more partitions.

**Q: How do I delete old messages?**  
A: Configure retention:
```yaml
storage:
  log:
    retention:
      hours: 168  # 7 days
```

### Consumer Questions

**Q: Why is my consumer lagging?**  
A: Common causes:
1. Slow processing - optimize your consumer code
2. Too few consumers - scale up consumer group
3. Network issues - check connectivity

**Q: How do I reset consumer offset?**  
A: Use kafka-consumer-groups tool:
```bash
kafka-consumer-groups.sh --reset-offsets \
  --bootstrap-server localhost:9092 \
  --group my-group \
  --topic my-topic \
  --to-earliest
```

**Q: What happens during consumer rebalance?**  
A: Consumers stop fetching, partitions reassign, then resume. Typically takes 5-10 seconds.

### Security Questions

**Q: How do I enable TLS?**  
A: See [Security Configuration](#33-security-configuration) section.

**Q: Does Takhin support SASL?**  
A: Yes, PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, and GSSAPI (Kerberos).

**Q: Can I use mTLS (mutual TLS)?**  
A: Yes:
```yaml
server:
  tls:
    enabled: true
    client:
      auth: "require"
    verify:
      client:
        cert: true
```

**Q: How do I rotate TLS certificates?**  
A: Replace cert files and restart Takhin. Zero downtime rotation not yet supported.

### Operations Questions

**Q: How do I upgrade Takhin?**  
A: Replace binary and restart. For clusters, rolling upgrade supported.

**Q: How do I backup Takhin data?**  
A: See [Backup and Disaster Recovery](#56-backup-and-disaster-recovery) section.

**Q: What's the disk space calculation?**  
A: Formula:
```
disk_space = (messages_per_day × avg_message_size × retention_days × partitions) / replication_factor
```

**Q: How do I monitor Takhin?**  
A: Use Prometheus + Grafana. See [Monitoring Guide](./monitoring/README.md).

### Troubleshooting Questions

**Q: Takhin won't start, what should I check?**  
A:
1. Check port not already in use: `netstat -tlnp | grep 9092`
2. Check data directory permissions
3. Check logs: `journalctl -u takhin -n 50`

**Q: Messages not appearing in topic?**  
A:
1. Verify topic exists: `kafka-topics.sh --list`
2. Check producer errors
3. Verify no firewall blocking port 9092

**Q: High latency, how to debug?**  
A:
1. Check Prometheus metrics for bottlenecks
2. Monitor disk I/O: `iostat -x 1`
3. Check network latency: `ping localhost`

See full [Troubleshooting Guide](./deployment/05-troubleshooting.md).

---

## 7. Troubleshooting

### 7.1 Common Issues

#### Issue: Port Already in Use

**Symptoms:**
```
Error: bind: address already in use
```

**Solution:**
```bash
# Find process using port 9092
sudo lsof -i :9092

# Kill the process
sudo kill -9 <PID>

# Or change Takhin port
export TAKHIN_SERVER_PORT=9093
```

#### Issue: Permission Denied on Data Directory

**Symptoms:**
```
Error: permission denied: /var/lib/takhin/data
```

**Solution:**
```bash
# Fix ownership
sudo chown -R takhin:takhin /var/lib/takhin/data

# Fix permissions
sudo chmod -R 755 /var/lib/takhin/data
```

#### Issue: Out of Disk Space

**Symptoms:**
```
Error: no space left on device
```

**Solution:**
```bash
# Check disk usage
df -h /var/lib/takhin/data

# Clean up old segments
# Option 1: Reduce retention
# Option 2: Add more disk space
# Option 3: Enable tiered storage
```

#### Issue: High Consumer Lag

**Symptoms:**
- Messages not processing fast enough
- Lag increasing over time

**Diagnosis:**
```bash
# Check consumer lag
kafka-consumer-groups.sh --describe \
  --bootstrap-server localhost:9092 \
  --group my-group
```

**Solutions:**
1. **Scale consumers:**
   ```bash
   # Start more consumer instances
   # Up to number of partitions
   ```

2. **Optimize consumer code:**
   - Reduce processing time per message
   - Use batching
   - Parallel processing within consumer

3. **Increase partitions:**
   ```bash
   # Create new topic with more partitions
   # Migrate consumers
   ```

#### Issue: Connection Refused

**Symptoms:**
```
Error: dial tcp localhost:9092: connect: connection refused
```

**Solution:**
```bash
# Check Takhin is running
sudo systemctl status takhin

# Check firewall
sudo ufw status

# Verify port binding
sudo netstat -tlnp | grep 9092
```

### 7.2 Performance Issues

#### Slow Writes

**Check:**
1. Disk I/O: `iostat -x 1`
2. Flush settings: Increase `flush.interval.ms`
3. Compression: Try different codec

**Optimize:**
```yaml
storage:
  log:
    flush:
      interval:
        ms: 5000
      messages: 50000
```

#### Slow Reads

**Check:**
1. Disk I/O: Sequential reads should be fast
2. Fetch size: Increase `fetch.min.bytes`
3. Network latency

**Optimize:**
```go
reader := kafka.NewReader(kafka.ReaderConfig{
    MinBytes: 10e3,  // 10KB
    MaxBytes: 10e6,  // 10MB
    MaxWait:  500 * time.Millisecond,
})
```

### 7.3 Cluster Issues

#### Under-Replicated Partitions

**Diagnosis:**
```bash
# Check cluster health via metrics
curl http://localhost:9090/metrics | grep under_replicated
```

**Solution:**
1. Check broker health
2. Verify network connectivity
3. Check replication lag settings

#### Split Brain (Multiple Leaders)

**Should not happen** with Raft consensus. If it does:

1. Check Raft logs
2. Verify cluster configuration
3. Restart affected brokers

### 7.4 Getting Help

#### Check Logs

```bash
# systemd logs
sudo journalctl -u takhin -f

# File logs
sudo tail -f /var/log/takhin/takhin.log
```

#### Enable Debug Logging

```bash
export TAKHIN_LOGGING_LEVEL=debug
sudo systemctl restart takhin
```

#### Generate Debug Bundle

```bash
# Collect diagnostic info
./takhin debug-bundle \
  --output takhin-debug-$(date +%Y%m%d).tar.gz

# Share with support team
```

#### Report Issues

1. Check [existing issues](https://github.com/takhin-data/takhin/issues)
2. Create new issue with:
   - Takhin version
   - Configuration (redact secrets)
   - Error logs
   - Steps to reproduce

#### Community Support

- **GitHub Discussions**: Q&A and general discussion
- **GitHub Issues**: Bug reports and feature requests
- **Documentation**: Complete docs at `/docs`

---

## Appendix A: Glossary

- **Broker**: Takhin server instance
- **Topic**: Named message stream
- **Partition**: Ordered, immutable sequence of messages
- **Offset**: Unique message ID within partition
- **Producer**: Application that writes messages
- **Consumer**: Application that reads messages
- **Consumer Group**: Set of consumers sharing workload
- **Lag**: Difference between latest offset and consumer offset
- **ISR**: In-Sync Replicas, replicas caught up with leader
- **HWM**: High Water Mark, highest committed offset
- **Segment**: Physical log file on disk
- **Raft**: Consensus algorithm for leader election

## Appendix B: Quick Reference Commands

### Essential Commands

```bash
# Start Takhin
takhin -config /etc/takhin/takhin.yaml

# Check version
takhin -version

# List topics
kafka-topics.sh --list --bootstrap-server localhost:9092

# Create topic
kafka-topics.sh --create --topic my-topic --partitions 3

# Produce message
echo "hello" | kafka-console-producer.sh --topic my-topic

# Consume messages
kafka-console-consumer.sh --topic my-topic --from-beginning

# Describe consumer group
kafka-consumer-groups.sh --describe --group my-group

# Check metrics
curl http://localhost:9090/metrics

# Health check
curl http://localhost:9091/health
```

### Configuration Overrides

```bash
# Via environment variables
export TAKHIN_SERVER_PORT=9093
export TAKHIN_STORAGE_DATA_DIR=/custom/path
export TAKHIN_LOGGING_LEVEL=debug

# Via command-line flags
takhin -config takhin.yaml -port 9093 -data-dir /custom/path
```

---

## Appendix C: Additional Resources

### Documentation

- [Architecture Guide](./architecture/README.md)
- [API Reference](./api/README.md)
- [Deployment Guide](./deployment/README.md)
- [Monitoring Guide](./monitoring/README.md)
- [Developer Guide](../TASK_7.3_DEVELOPER_GUIDE_SUMMARY.md)

### External Links

- [GitHub Repository](https://github.com/takhin-data/takhin)
- [Release Notes](https://github.com/takhin-data/takhin/releases)
- [Issue Tracker](https://github.com/takhin-data/takhin/issues)

### Related Projects

- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [kafka-go Client](https://github.com/segmentio/kafka-go)
- [Prometheus](https://prometheus.io/)
- [Grafana](https://grafana.com/)

---

**Document Version**: 1.0  
**Takhin Version**: 1.0  
**Last Updated**: 2026-01-06  
**Maintained By**: Takhin Team

For the latest version of this document, visit: [docs/USER_MANUAL.md](https://github.com/takhin-data/takhin/blob/main/docs/USER_MANUAL.md)
