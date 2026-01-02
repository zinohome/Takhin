# Takhin Standalone Deployment Guide

This guide covers deploying Takhin in standalone (single-node) mode for development, testing, or small-scale production environments.

## Prerequisites

- **Operating System**: Linux (Ubuntu 20.04+, CentOS 7+) or macOS
- **Go**: 1.21+ (for building from source)
- **Memory**: Minimum 2GB RAM, recommended 4GB+
- **Disk**: 10GB+ available space for data storage
- **Network**: Port 9092 (Kafka), 9090 (metrics), 8080 (Console API)

## Installation Methods

### Method 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/takhin-data/takhin.git
cd takhin

# Install dependencies and build
task backend:deps
task backend:build

# Binary will be created at: build/takhin
```

### Method 2: Pre-built Binary (Recommended)

```bash
# Download the latest release
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-linux-amd64
chmod +x takhin-linux-amd64
sudo mv takhin-linux-amd64 /usr/local/bin/takhin

# Verify installation
takhin -version
```

## Configuration

### 1. Create Configuration Directory

```bash
sudo mkdir -p /etc/takhin
sudo mkdir -p /var/lib/takhin/data
sudo mkdir -p /var/log/takhin
```

### 2. Create Configuration File

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
  listeners:
    - "tcp://0.0.0.0:9092"
  advertised:
    host: "localhost"  # Change to your server's hostname or IP
    port: 9092
  max:
    message:
      bytes: 1048576      # 1MB
    connections: 1000
  request:
    timeout:
      ms: 30000           # 30 seconds
  connection:
    timeout:
      ms: 60000           # 60 seconds

# Storage Configuration
storage:
  data:
    dir: "/var/lib/takhin/data"
  log:
    segment:
      size: 1073741824    # 1GB
    retention:
      hours: 168          # 7 days
      bytes: 0            # 0 = unlimited
    cleanup:
      interval:
        ms: 300000        # 5 minutes
    flush:
      interval:
        ms: 1000          # 1 second
      messages: 10000
  cleaner:
    enabled: true
  compaction:
    interval:
      ms: 600000          # 10 minutes
    min:
      cleanable:
        ratio: 0.5

# Replication Configuration
replication:
  default:
    replication:
      factor: 1           # No replication in standalone mode

# Logging Configuration
logging:
  level: "info"
  format: "json"

# Metrics Configuration
metrics:
  enabled: true
  host: "0.0.0.0"
  port: 9090
  path: "/metrics"
```

### 3. Environment Variables (Optional)

Configuration can be overridden using environment variables with `TAKHIN_` prefix:

```bash
# Example: Override data directory
export TAKHIN_STORAGE_DATA_DIR=/custom/data/path

# Example: Change log level
export TAKHIN_LOGGING_LEVEL=debug

# Example: Change server port
export TAKHIN_SERVER_PORT=9093
```

## Running Takhin

### Development Mode

```bash
# Run directly with config file
cd takhin
task backend:run
```

### Production Mode - systemd Service

#### 1. Create Service User

```bash
sudo useradd -r -s /bin/false takhin
sudo chown -R takhin:takhin /var/lib/takhin
sudo chown -R takhin:takhin /var/log/takhin
```

#### 2. Create systemd Service File

Create `/etc/systemd/system/takhin.service`:

```ini
[Unit]
Description=Takhin Kafka-Compatible Streaming Platform
Documentation=https://github.com/takhin-data/takhin
After=network.target

[Service]
Type=simple
User=takhin
Group=takhin
ExecStart=/usr/local/bin/takhin -config /etc/takhin/takhin.yaml
Restart=on-failure
RestartSec=5s

# Logging
StandardOutput=append:/var/log/takhin/takhin.log
StandardError=append:/var/log/takhin/takhin-error.log

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/takhin /var/log/takhin

[Install]
WantedBy=multi-user.target
```

#### 3. Start and Enable Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Start Takhin
sudo systemctl start takhin

# Check status
sudo systemctl status takhin

# Enable autostart on boot
sudo systemctl enable takhin

# View logs
sudo journalctl -u takhin -f
```

## Running Takhin Console (Management UI)

### Console Server Deployment

```bash
# Build console binary
cd backend
go build -o /usr/local/bin/takhin-console ./cmd/console

# Create systemd service
sudo nano /etc/systemd/system/takhin-console.service
```

Console service file:

```ini
[Unit]
Description=Takhin Console Management API
After=takhin.service

[Service]
Type=simple
User=takhin
Group=takhin
ExecStart=/usr/local/bin/takhin-console \
  -data-dir /var/lib/takhin/data \
  -api-addr :8080 \
  -enable-auth \
  -api-keys "your-secret-api-key-here"
Restart=on-failure
RestartSec=5s
StandardOutput=append:/var/log/takhin/console.log
StandardError=append:/var/log/takhin/console-error.log

[Install]
WantedBy=multi-user.target
```

Start console:

```bash
sudo systemctl daemon-reload
sudo systemctl start takhin-console
sudo systemctl enable takhin-console
```

## Verification

### 1. Check Server Status

```bash
# Check if Takhin is listening
sudo netstat -tlnp | grep 9092

# Check metrics endpoint
curl http://localhost:9090/metrics
```

### 2. Test with Kafka Client

```bash
# Using kafka-console-producer (from Apache Kafka distribution)
echo "test message" | kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic test-topic

# Using kafka-console-consumer
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic test-topic \
  --from-beginning
```

### 3. Check Console API

```bash
# Health check
curl http://localhost:8080/health

# List topics (with authentication)
curl -H "Authorization: your-secret-api-key-here" \
  http://localhost:8080/api/v1/topics
```

## Performance Tuning

### Operating System

```bash
# Increase file descriptor limits
echo "* soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "* hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# Increase network buffer sizes
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.wmem_max=134217728
sudo sysctl -w net.ipv4.tcp_rmem="4096 87380 67108864"
sudo sysctl -w net.ipv4.tcp_wmem="4096 65536 67108864"
```

### Takhin Configuration

For high-throughput scenarios:

```yaml
kafka:
  max:
    message:
      bytes: 10485760      # 10MB
    connections: 5000

storage:
  log:
    segment:
      size: 2147483648     # 2GB
    flush:
      interval:
        ms: 5000           # Less frequent flushes
      messages: 50000

metrics:
  enabled: false           # Disable if not needed
```

## Log Management

### Log Rotation

Create `/etc/logrotate.d/takhin`:

```
/var/log/takhin/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 takhin takhin
    postrotate
        systemctl reload takhin > /dev/null 2>&1 || true
    endscript
}
```

## Monitoring

### Prometheus Metrics

Takhin exposes Prometheus metrics at `http://localhost:9090/metrics`.

Example metrics to monitor:
- `takhin_kafka_requests_total` - Total requests by API type
- `takhin_kafka_request_duration_seconds` - Request latency
- `takhin_storage_bytes_total` - Storage usage
- `takhin_active_connections` - Active client connections

### Health Check Endpoint

```bash
# Basic health check
curl http://localhost:8080/health

# Expected response
{"status":"ok","timestamp":"2026-01-02T04:37:30Z"}
```

## Backup and Recovery

### Data Backup

```bash
# Stop Takhin service
sudo systemctl stop takhin

# Backup data directory
sudo tar -czf takhin-backup-$(date +%Y%m%d).tar.gz \
  /var/lib/takhin/data

# Start Takhin service
sudo systemctl start takhin
```

### Data Recovery

```bash
# Stop Takhin
sudo systemctl stop takhin

# Restore backup
sudo tar -xzf takhin-backup-20260102.tar.gz -C /

# Set permissions
sudo chown -R takhin:takhin /var/lib/takhin/data

# Start Takhin
sudo systemctl start takhin
```

## Upgrading

### Rolling Upgrade (Zero Downtime)

```bash
# Download new binary
curl -LO https://github.com/takhin-data/takhin/releases/download/vX.Y.Z/takhin-linux-amd64

# Backup current binary
sudo cp /usr/local/bin/takhin /usr/local/bin/takhin.backup

# Replace binary
sudo mv takhin-linux-amd64 /usr/local/bin/takhin
sudo chmod +x /usr/local/bin/takhin

# Restart service
sudo systemctl restart takhin

# Verify
sudo systemctl status takhin
```

## Uninstallation

```bash
# Stop and disable services
sudo systemctl stop takhin takhin-console
sudo systemctl disable takhin takhin-console

# Remove binaries
sudo rm /usr/local/bin/takhin
sudo rm /usr/local/bin/takhin-console

# Remove configuration and data (optional)
sudo rm -rf /etc/takhin
sudo rm -rf /var/lib/takhin
sudo rm -rf /var/log/takhin

# Remove service files
sudo rm /etc/systemd/system/takhin.service
sudo rm /etc/systemd/system/takhin-console.service
sudo systemctl daemon-reload
```

## Next Steps

- [Cluster Deployment Guide](./02-cluster-deployment.md)
- [Docker Deployment](./03-docker-deployment.md)
- [Configuration Reference](./04-configuration-reference.md)
- [Troubleshooting Guide](./05-troubleshooting.md)
