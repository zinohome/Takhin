# Takhin Cluster Deployment Guide

This guide covers deploying Takhin in a distributed cluster configuration with Raft consensus for high availability and fault tolerance.

## Architecture Overview

Takhin uses Raft consensus for cluster coordination (no ZooKeeper dependency). A typical production cluster consists of:

- **3+ Broker Nodes**: Minimum 3 nodes for Raft quorum
- **Replication Factor**: 2-3 for data durability
- **Leader Election**: Automatic via Raft consensus

### Cluster Topology Example

```
┌─────────────────────────────────────────────┐
│              Load Balancer                  │
│         (HAProxy/nginx/L4 LB)              │
└──────────┬──────────┬──────────┬───────────┘
           │          │          │
    ┌──────▼───┐ ┌────▼────┐ ┌──▼──────┐
    │ Broker 1 │ │Broker 2 │ │Broker 3 │
    │ ID: 1    │ │ ID: 2   │ │ ID: 3   │
    │ :9092    │ │ :9092   │ │ :9092   │
    │ Raft     │ │ Raft    │ │ Raft    │
    └──────────┘ └─────────┘ └─────────┘
         │            │            │
    ┌────┴────────────┴────────────┴────┐
    │     Shared Raft Consensus          │
    │  (Leader Election & Metadata)      │
    └────────────────────────────────────┘
```

## Prerequisites

- **Nodes**: 3+ servers (5 recommended for production)
- **OS**: Linux (Ubuntu 20.04+ or CentOS 7+)
- **Memory**: 8GB+ RAM per node
- **Disk**: 100GB+ SSD storage per node
- **Network**: Low latency (<10ms RTT between nodes)
- **Ports**: 
  - 9092: Kafka protocol
  - 9090: Metrics
  - 7946: Raft peer communication
  - 8080: Console API

## Installation

### On Each Node

```bash
# Download and install Takhin binary
curl -LO https://github.com/takhin-data/takhin/releases/latest/download/takhin-linux-amd64
sudo mv takhin-linux-amd64 /usr/local/bin/takhin
sudo chmod +x /usr/local/bin/takhin

# Create directories
sudo mkdir -p /etc/takhin
sudo mkdir -p /var/lib/takhin/data
sudo mkdir -p /var/log/takhin

# Create service user
sudo useradd -r -s /bin/false takhin
sudo chown -R takhin:takhin /var/lib/takhin
sudo chown -R takhin:takhin /var/log/takhin
```

## Configuration

### Node 1 Configuration (`/etc/takhin/takhin.yaml`)

```yaml
# Server Configuration
server:
  host: "0.0.0.0"
  port: 9092

# Kafka Protocol Configuration
kafka:
  broker:
    id: 1  # UNIQUE for each broker
  
  # Cluster Configuration - All broker IDs
  cluster:
    brokers: [1, 2, 3]  # List all brokers in cluster
  
  listeners:
    - "tcp://0.0.0.0:9092"
  
  advertised:
    host: "broker1.example.com"  # External hostname or IP
    port: 9092
  
  max:
    message:
      bytes: 10485760      # 10MB
    connections: 5000
  
  request:
    timeout:
      ms: 30000
  connection:
    timeout:
      ms: 60000

# Storage Configuration
storage:
  data:
    dir: "/var/lib/takhin/data"
  
  log:
    segment:
      size: 2147483648    # 2GB
    retention:
      hours: 168          # 7 days
      bytes: 0            # Unlimited
    cleanup:
      interval:
        ms: 300000        # 5 minutes
    flush:
      interval:
        ms: 1000
      messages: 10000
  
  cleaner:
    enabled: true
  
  compaction:
    interval:
      ms: 600000
    min:
      cleanable:
        ratio: 0.5

# Replication Configuration - KEY FOR CLUSTERS
replication:
  default:
    replication:
      factor: 3           # 3 replicas for each partition
  
  replica:
    lag:
      time:
        max:
          ms: 10000       # 10 seconds
    fetch:
      wait:
        max:
          ms: 500
      max:
        bytes: 1048576

# Raft Configuration (internal, auto-configured)
raft:
  bind:
    addr: "0.0.0.0:7946"
  advertise:
    addr: "broker1.example.com:7946"
  peers:
    - "broker1.example.com:7946"
    - "broker2.example.com:7946"
    - "broker3.example.com:7946"

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

### Node 2 Configuration

Copy Node 1 config and modify:

```yaml
kafka:
  broker:
    id: 2  # Change to 2
  advertised:
    host: "broker2.example.com"  # Change hostname

raft:
  advertise:
    addr: "broker2.example.com:7946"  # Change hostname
```

### Node 3 Configuration

Copy Node 1 config and modify:

```yaml
kafka:
  broker:
    id: 3  # Change to 3
  advertised:
    host: "broker3.example.com"  # Change hostname

raft:
  advertise:
    addr: "broker3.example.com:7946"  # Change hostname
```

## systemd Service Setup

On each node, create `/etc/systemd/system/takhin.service`:

```ini
[Unit]
Description=Takhin Kafka-Compatible Streaming Platform
Documentation=https://github.com/takhin-data/takhin
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=takhin
Group=takhin
ExecStart=/usr/local/bin/takhin -config /etc/takhin/takhin.yaml
Restart=on-failure
RestartSec=10s
TimeoutStopSec=30s

# Logging
StandardOutput=append:/var/log/takhin/takhin.log
StandardError=append:/var/log/takhin/takhin-error.log

# Resource limits
LimitNOFILE=100000
LimitNPROC=8192
LimitMEMLOCK=infinity

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/takhin /var/log/takhin

[Install]
WantedBy=multi-user.target
```

## Starting the Cluster

### Step 1: Start First Node (Bootstrap)

```bash
# On Node 1
sudo systemctl daemon-reload
sudo systemctl start takhin
sudo systemctl status takhin

# Check logs for Raft initialization
sudo journalctl -u takhin -f
```

### Step 2: Start Remaining Nodes

```bash
# On Node 2
sudo systemctl start takhin

# On Node 3
sudo systemctl start takhin
```

### Step 3: Verify Cluster Formation

```bash
# Check Raft cluster status (on any node)
curl http://localhost:9090/metrics | grep raft_state

# Expected output: raft_state{role="leader"} or raft_state{role="follower"}

# Check broker list
kafka-broker-api-versions.sh --bootstrap-server \
  broker1.example.com:9092,broker2.example.com:9092,broker3.example.com:9092
```

## Load Balancer Configuration

### HAProxy Setup

Create `/etc/haproxy/haproxy.cfg`:

```haproxy
global
    log /dev/log local0
    maxconn 10000
    user haproxy
    group haproxy
    daemon

defaults
    mode tcp
    timeout connect 10s
    timeout client 300s
    timeout server 300s
    log global

# Kafka protocol load balancing
frontend kafka_frontend
    bind *:9092
    mode tcp
    default_backend kafka_backend

backend kafka_backend
    mode tcp
    balance roundrobin
    option tcp-check
    
    server broker1 broker1.example.com:9092 check
    server broker2 broker2.example.com:9092 check
    server broker3 broker3.example.com:9092 check

# Console API load balancing
frontend console_frontend
    bind *:8080
    mode http
    default_backend console_backend

backend console_backend
    mode http
    balance roundrobin
    option httpchk GET /health
    
    server console1 broker1.example.com:8080 check
    server console2 broker2.example.com:8080 check
    server console3 broker3.example.com:8080 check

# Metrics (read-only, any node)
frontend metrics_frontend
    bind *:9090
    mode http
    default_backend metrics_backend

backend metrics_backend
    mode http
    balance roundrobin
    
    server metrics1 broker1.example.com:9090 check
    server metrics2 broker2.example.com:9090 check
    server metrics3 broker3.example.com:9090 check
```

Start HAProxy:

```bash
sudo systemctl restart haproxy
sudo systemctl enable haproxy
```

### nginx TCP Load Balancing

Add to `/etc/nginx/nginx.conf`:

```nginx
stream {
    upstream kafka_cluster {
        least_conn;
        server broker1.example.com:9092 max_fails=3 fail_timeout=30s;
        server broker2.example.com:9092 max_fails=3 fail_timeout=30s;
        server broker3.example.com:9092 max_fails=3 fail_timeout=30s;
    }

    server {
        listen 9092;
        proxy_pass kafka_cluster;
        proxy_connect_timeout 10s;
    }
}
```

## Testing the Cluster

### Create Replicated Topic

```bash
# Using Kafka CLI tools
kafka-topics.sh --create \
  --bootstrap-server broker1.example.com:9092,broker2.example.com:9092,broker3.example.com:9092 \
  --topic test-replicated \
  --partitions 6 \
  --replication-factor 3

# Describe topic to verify replication
kafka-topics.sh --describe \
  --bootstrap-server broker1.example.com:9092 \
  --topic test-replicated
```

Expected output shows replicas distributed across brokers:

```
Topic: test-replicated  Partition: 0  Leader: 1  Replicas: 1,2,3  Isr: 1,2,3
Topic: test-replicated  Partition: 1  Leader: 2  Replicas: 2,3,1  Isr: 2,3,1
...
```

### Test Failover

```bash
# Stop one broker
sudo systemctl stop takhin  # On Node 2

# Verify cluster still operational
kafka-console-producer.sh --broker-list broker1.example.com:9092 --topic test-replicated
> Test message during failover

# Check ISR (In-Sync Replicas) - should exclude failed broker
kafka-topics.sh --describe --topic test-replicated --bootstrap-server broker1.example.com:9092

# Restart broker
sudo systemctl start takhin  # On Node 2

# Verify broker rejoins cluster
```

## Monitoring Cluster Health

### Cluster Metrics

```bash
# Raft leader status
curl http://broker1.example.com:9090/metrics | grep raft_state

# Cluster size
curl http://broker1.example.com:9090/metrics | grep cluster_size

# Replication lag
curl http://broker1.example.com:9090/metrics | grep replica_lag
```

### Console API Cluster Status

```bash
# Get cluster info
curl -H "Authorization: your-api-key" \
  http://localhost:8080/api/v1/cluster/info

# Get broker list
curl -H "Authorization: your-api-key" \
  http://localhost:8080/api/v1/brokers
```

## Scaling the Cluster

### Adding a New Broker

1. **Provision new server** with same setup as existing nodes

2. **Configure new broker** (`/etc/takhin/takhin.yaml` on new node):

```yaml
kafka:
  broker:
    id: 4  # New unique ID
  cluster:
    brokers: [1, 2, 3, 4]  # Add new ID to list
  advertised:
    host: "broker4.example.com"
    port: 9092
```

3. **Update existing brokers**: Add broker ID 4 to `kafka.cluster.brokers` list in all existing nodes' configs

4. **Restart existing brokers** (one at a time):

```bash
# On each existing node
sudo systemctl restart takhin
sleep 30  # Wait for stabilization
```

5. **Start new broker**:

```bash
# On new node
sudo systemctl start takhin
```

6. **Rebalance partitions** using reassignment tools

### Removing a Broker

1. **Reassign partitions** from broker to be removed
2. **Update cluster configuration** (remove broker ID from all configs)
3. **Restart remaining brokers**
4. **Decommission removed broker**

## Backup and Disaster Recovery

### Cluster Backup Strategy

```bash
# On each node, backup configuration and data
sudo tar -czf takhin-node$(hostname)-$(date +%Y%m%d).tar.gz \
  /etc/takhin \
  /var/lib/takhin/data

# Upload to remote storage
aws s3 cp takhin-node*.tar.gz s3://backups/takhin/
```

### Disaster Recovery

For complete cluster failure:

1. **Restore configurations** on all nodes
2. **Restore data** on at least quorum (2 of 3) nodes
3. **Start nodes sequentially** (bootstrap first, then others)
4. **Verify Raft quorum** established
5. **Sync missing data** from replicas

## Performance Tuning

### OS-Level Tuning (All Nodes)

```bash
# Network tuning
sudo tee -a /etc/sysctl.conf <<EOF
net.core.rmem_max=134217728
net.core.wmem_max=134217728
net.ipv4.tcp_rmem=4096 87380 67108864
net.ipv4.tcp_wmem=4096 65536 67108864
net.ipv4.tcp_max_syn_backlog=8192
net.core.netdev_max_backlog=5000
EOF

sudo sysctl -p

# File descriptor limits
sudo tee -a /etc/security/limits.conf <<EOF
takhin soft nofile 100000
takhin hard nofile 100000
takhin soft nproc 8192
takhin hard nproc 8192
EOF
```

### Takhin Configuration Tuning

```yaml
kafka:
  max:
    message:
      bytes: 10485760
    connections: 10000

storage:
  log:
    segment:
      size: 2147483648     # 2GB segments
    flush:
      interval:
        ms: 5000           # Batch flushes
      messages: 100000

replication:
  replica:
    fetch:
      wait:
        max:
          ms: 500
      max:
        bytes: 10485760    # 10MB
```

## Security Hardening

### Firewall Rules

```bash
# Allow only cluster internal communication
sudo ufw allow from broker1.example.com to any port 7946
sudo ufw allow from broker2.example.com to any port 7946
sudo ufw allow from broker3.example.com to any port 7946

# Allow Kafka from application subnet
sudo ufw allow from 10.0.1.0/24 to any port 9092

# Deny all other traffic
sudo ufw default deny incoming
sudo ufw enable
```

## Next Steps

- [Docker Deployment](./03-docker-deployment.md)
- [Kubernetes Deployment](./03-docker-deployment.md#kubernetes-deployment)
- [Configuration Reference](./04-configuration-reference.md)
- [Troubleshooting Guide](./05-troubleshooting.md)
