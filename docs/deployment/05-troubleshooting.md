# Takhin Troubleshooting Guide

Comprehensive guide for diagnosing and resolving common Takhin issues.

## Quick Diagnostic Commands

```bash
# Check if Takhin is running
systemctl status takhin
ps aux | grep takhin

# Check port binding
netstat -tlnp | grep 9092
ss -tlnp | grep 9092

# View recent logs
journalctl -u takhin -n 100 --no-pager
tail -f /var/log/takhin/takhin.log

# Check disk space
df -h /var/lib/takhin/data

# Check memory usage
free -h
top -bn1 | grep takhin

# Test connectivity
telnet localhost 9092
nc -zv localhost 9092

# Check metrics
curl http://localhost:9090/metrics

# Test with Kafka tools
kafka-broker-api-versions.sh --bootstrap-server localhost:9092
```

## Common Issues and Solutions

### 1. Broker Won't Start

#### Symptom
```
systemctl start takhin
Job for takhin.service failed because the control process exited with error code.
```

#### Diagnostic Steps

```bash
# Check detailed status
systemctl status takhin -l

# View logs
journalctl -u takhin -n 50

# Check configuration
takhin -config /etc/takhin/takhin.yaml -validate

# Check file permissions
ls -la /var/lib/takhin/data
ls -la /etc/takhin/takhin.yaml
```

#### Common Causes & Solutions

**A. Port Already in Use**

```bash
# Check what's using the port
sudo lsof -i :9092
sudo netstat -tlnp | grep 9092

# Solution: Kill conflicting process or change port
sudo kill <PID>
# Or edit config: TAKHIN_SERVER_PORT=9093
```

**B. Permission Denied**

```bash
# Check ownership
ls -la /var/lib/takhin/data

# Fix permissions
sudo chown -R takhin:takhin /var/lib/takhin
sudo chown -R takhin:takhin /var/log/takhin
sudo chmod 755 /var/lib/takhin/data
```

**C. Invalid Configuration**

```bash
# Common config errors:
# - Invalid YAML syntax
# - Missing broker.id
# - Invalid cluster.brokers format

# Validate YAML
yamllint /etc/takhin/takhin.yaml

# Check for required fields
grep "broker:" /etc/takhin/takhin.yaml
grep "id:" /etc/takhin/takhin.yaml
```

**D. Insufficient Disk Space**

```bash
# Check disk usage
df -h /var/lib/takhin/data

# Solution: Free up space or change data directory
sudo rm -rf /var/lib/takhin/data/old-logs/*
# Or: TAKHIN_STORAGE_DATA_DIR=/new/path/with/space
```

---

### 2. Cannot Connect to Broker

#### Symptom
```
Connection to node -1 could not be established. Broker may not be available.
Error: Network timeout
```

#### Diagnostic Steps

```bash
# Test network connectivity
ping <broker-host>
telnet <broker-host> 9092
nc -zv <broker-host> 9092

# Check if broker is listening
netstat -tlnp | grep 9092

# Check firewall rules
sudo iptables -L -n
sudo ufw status

# Check advertised host configuration
grep "advertised" /etc/takhin/takhin.yaml

# Test DNS resolution
nslookup <broker-host>
dig <broker-host>
```

#### Solutions

**A. Firewall Blocking Connection**

```bash
# Allow port 9092
sudo ufw allow 9092/tcp

# Or with iptables
sudo iptables -A INPUT -p tcp --dport 9092 -j ACCEPT
sudo iptables-save
```

**B. Wrong Advertised Host**

```yaml
# Edit /etc/takhin/takhin.yaml
kafka:
  advertised:
    host: "correct-hostname-or-ip"  # NOT localhost for remote clients!
    port: 9092

# Restart
sudo systemctl restart takhin
```

**C. Network Routing Issues**

```bash
# Check routing table
ip route
route -n

# Trace route to broker
traceroute <broker-host>

# Check if listening on correct interface
sudo netstat -tlnp | grep 9092
# Should show 0.0.0.0:9092, not 127.0.0.1:9092
```

**D. Load Balancer Misconfiguration**

```bash
# Test direct connection to broker
kafka-broker-api-versions.sh --bootstrap-server broker1:9092

# Test via load balancer
kafka-broker-api-versions.sh --bootstrap-server lb.example.com:9092

# Check HAProxy/nginx logs
tail -f /var/log/haproxy.log
tail -f /var/log/nginx/error.log
```

---

### 3. High CPU Usage

#### Symptom
```
top shows takhin process using >80% CPU consistently
```

#### Diagnostic Steps

```bash
# Check CPU usage
top -bn1 | grep takhin

# Check for high request rate
curl http://localhost:9090/metrics | grep requests_total

# Check thread count
ps -eLf | grep takhin | wc -l

# Profile CPU usage
# (If pprof is enabled)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
```

#### Common Causes & Solutions

**A. High Request Rate**

```bash
# Check metrics
curl http://localhost:9090/metrics | grep takhin_kafka_requests_total

# Solution: Scale horizontally (add brokers)
# Or increase connection pool on clients
```

**B. Log Compaction Running**

```bash
# Check compaction metrics
curl http://localhost:9090/metrics | grep compaction

# Solution: Adjust compaction interval
# Edit config:
storage:
  compaction:
    interval:
      ms: 1800000  # Increase to 30 minutes
```

**C. Many Small Flushes**

```yaml
# Reduce flush frequency
storage:
  log:
    flush:
      interval:
        ms: 5000      # Increase from 1000
      messages: 50000  # Increase from 10000
```

---

### 4. High Memory Usage / OOM Errors

#### Symptom
```
Out of memory: Kill process <pid> (takhin)
takhin.service: Main process exited, code=killed, status=9/KILL
```

#### Diagnostic Steps

```bash
# Check memory usage
free -h
ps aux | grep takhin
top -p $(pgrep takhin)

# Check for memory leaks
curl http://localhost:9090/metrics | grep go_memstats

# Review systemd limits
systemctl show takhin | grep Memory
```

#### Solutions

**A. Too Many Connections**

```yaml
# Reduce max connections
kafka:
  max:
    connections: 500  # Reduce from 1000+
```

**B. Large Message Size**

```yaml
# If not needed, reduce max message size
kafka:
  max:
    message:
      bytes: 1048576  # 1MB instead of 10MB
```

**C. Increase System Memory**

```bash
# Add swap space (temporary solution)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Permanent: Add to /etc/fstab
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

**D. Set Memory Limits**

```ini
# Edit /etc/systemd/system/takhin.service
[Service]
MemoryMax=4G
MemoryHigh=3.5G

# Reload and restart
sudo systemctl daemon-reload
sudo systemctl restart takhin
```

---

### 5. Data Loss / Missing Messages

#### Symptom
```
Produced messages not appearing in topic
Consumer can't find expected offsets
```

#### Diagnostic Steps

```bash
# Check broker logs for errors
journalctl -u takhin | grep -i error

# Verify topic exists
kafka-topics.sh --list --bootstrap-server localhost:9092

# Check topic details
kafka-topics.sh --describe --topic my-topic --bootstrap-server localhost:9092

# Check segment files
ls -lh /var/lib/takhin/data/my-topic-0/

# Check retention settings
grep retention /etc/takhin/takhin.yaml

# Check disk space
df -h /var/lib/takhin/data
```

#### Common Causes & Solutions

**A. Aggressive Retention Policy**

```yaml
# Check retention
storage:
  log:
    retention:
      hours: 168  # Not too low (e.g., not 1 hour)
      bytes: 0    # 0 = unlimited

# Restart after change
sudo systemctl restart takhin
```

**B. Producer Not Acknowledging**

```bash
# Check producer configuration
# Ensure acks=all or acks=1, not acks=0

# Check for producer errors in application logs
```

**C. Replication Issues (Cluster Mode)**

```bash
# Check ISR (In-Sync Replicas)
kafka-topics.sh --describe --topic my-topic --bootstrap-server localhost:9092

# Look for: Replicas: 1,2,3  Isr: 1,2,3
# If ISR < Replicas, followers are lagging

# Check replication lag
curl http://localhost:9090/metrics | grep replica_lag
```

**D. Disk Full**

```bash
# Check disk space
df -h /var/lib/takhin/data

# Free up space
sudo du -sh /var/lib/takhin/data/*
sudo rm -rf /var/lib/takhin/data/old-topic-*

# Or increase retention cleanup frequency
storage:
  log:
    cleanup:
      interval:
        ms: 60000  # More frequent (1 minute)
```

---

### 6. Slow Performance / High Latency

#### Symptom
```
kafka-console-consumer slow to receive messages
High request latency in metrics
```

#### Diagnostic Steps

```bash
# Check request latency
curl http://localhost:9090/metrics | grep request_duration

# Check disk I/O
iostat -x 1 10
iotop

# Check network latency
ping <broker-host>

# Check for slow disk
time dd if=/dev/zero of=/var/lib/takhin/data/testfile bs=1M count=1024
```

#### Solutions

**A. Slow Disk I/O**

```bash
# Check if using HDD instead of SSD
lsblk -o NAME,ROTA
# ROTA=1 means HDD (rotational), ROTA=0 means SSD

# Solution: Migrate to SSD storage
# Or adjust flush settings for HDD:
storage:
  log:
    flush:
      interval:
        ms: 10000  # Less frequent flushes
      messages: 100000
```

**B. Network Latency**

```bash
# Test network latency between brokers
ping broker2.example.com
mtr broker2.example.com

# Ensure brokers are in same region/datacenter
```

**C. Log Compaction Load**

```yaml
# Reduce compaction frequency
storage:
  compaction:
    interval:
      ms: 3600000  # Once per hour
    min:
      cleanable:
        ratio: 0.7  # Compact less frequently
```

**D. Too Many Small Segments**

```yaml
# Increase segment size
storage:
  log:
    segment:
      size: 2147483648  # 2GB instead of 1GB
```

---

### 7. Cluster Issues

#### Symptom
```
Broker not joining cluster
Raft leader election fails
Partition replicas not syncing
```

#### Diagnostic Steps

```bash
# Check cluster state
curl http://localhost:9090/metrics | grep raft_state
curl http://localhost:9090/metrics | grep cluster_size

# Check broker connectivity
ping broker2.example.com
ping broker3.example.com

# Check Raft port (7946)
nc -zv broker2.example.com 7946

# Check cluster configuration
grep cluster /etc/takhin/takhin.yaml

# View Raft logs
journalctl -u takhin | grep -i raft
```

#### Solutions

**A. Mismatched Cluster Configuration**

```yaml
# Ensure ALL brokers have same cluster.brokers list
kafka:
  cluster:
    brokers: [1, 2, 3]  # Must match across all nodes

# Restart all brokers one by one
sudo systemctl restart takhin
```

**B. Raft Port Blocked**

```bash
# Allow Raft port 7946
sudo ufw allow from broker1.example.com to any port 7946
sudo ufw allow from broker2.example.com to any port 7946
sudo ufw allow from broker3.example.com to any port 7946
```

**C. Split Brain**

```bash
# Check leader on each node
curl http://broker1:9090/metrics | grep raft_state
curl http://broker2:9090/metrics | grep raft_state
curl http://broker3:9090/metrics | grep raft_state

# If multiple leaders: network partition issue
# Verify network connectivity between ALL brokers
```

**D. Insufficient Quorum**

```bash
# For 3-node cluster, need 2 nodes minimum
# Check how many brokers are running
systemctl status takhin  # On each node

# Start failed brokers
sudo systemctl start takhin
```

---

### 8. Console API Issues

#### Symptom
```
curl http://localhost:8080/health
curl: (7) Failed to connect to localhost port 8080: Connection refused
```

#### Diagnostic Steps

```bash
# Check if console is running
ps aux | grep takhin-console
systemctl status takhin-console

# Check port
netstat -tlnp | grep 8080

# Check logs
journalctl -u takhin-console -n 50

# Test health endpoint
curl -v http://localhost:8080/health
```

#### Solutions

**A. Console Not Started**

```bash
# Start console service
sudo systemctl start takhin-console

# Enable autostart
sudo systemctl enable takhin-console
```

**B. Authentication Failing**

```bash
# Test without auth
curl http://localhost:8080/health  # Should work

# Test with auth
curl -H "Authorization: your-api-key" http://localhost:8080/api/v1/topics

# Check configured API keys
systemctl cat takhin-console | grep api-keys

# Update API keys
sudo systemctl edit takhin-console
# Add: -api-keys "new-key-1,new-key-2"
```

**C. Wrong Data Directory**

```bash
# Console must point to same data directory as broker
# Check broker data dir
grep data.dir /etc/takhin/takhin.yaml

# Update console
sudo systemctl edit takhin-console
# Add: -data-dir /var/lib/takhin/data

sudo systemctl restart takhin-console
```

---

## Debugging Tools

### Enable Debug Logging

```yaml
# Edit config
logging:
  level: "debug"
  format: "json"

# Or with environment variable
export TAKHIN_LOGGING_LEVEL=debug

# Restart
sudo systemctl restart takhin
```

### Capture Network Traffic

```bash
# Capture Kafka protocol traffic
sudo tcpdump -i any -w kafka-traffic.pcap port 9092

# Analyze with Wireshark or tshark
tshark -r kafka-traffic.pcap -V
```

### Golang pprof Profiling

```bash
# If pprof is enabled (development build)
# CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profile
curl http://localhost:6060/debug/pprof/heap > mem.prof
go tool pprof mem.prof

# Goroutine profile
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

### Stress Testing

```bash
# Use kafka-producer-perf-test
kafka-producer-perf-test.sh \
  --topic test-perf \
  --num-records 1000000 \
  --record-size 1024 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092

# Monitor during test
watch -n 1 'curl -s http://localhost:9090/metrics | grep takhin_kafka_requests_total'
```

---

## Log Analysis

### Important Log Messages

```bash
# Fatal errors
journalctl -u takhin | grep -i "fatal"

# Connection issues
journalctl -u takhin | grep -i "connection"

# Replication errors
journalctl -u takhin | grep -i "replication"

# Raft issues
journalctl -u takhin | grep -i "raft"

# Storage errors
journalctl -u takhin | grep -i "storage\|disk"
```

### Log Patterns to Watch

| Pattern | Meaning | Action |
|---------|---------|--------|
| `connection refused` | Cannot reach broker | Check network/firewall |
| `permission denied` | File permissions | Fix with `chown`/`chmod` |
| `out of memory` | Memory exhausted | Add RAM or reduce limits |
| `no space left on device` | Disk full | Free up space |
| `bind: address already in use` | Port conflict | Change port or kill process |
| `raft: no leader` | Raft quorum lost | Check cluster connectivity |
| `replica lag` | Follower behind | Check network/disk I/O |

---

## Getting Help

### Collect Diagnostic Information

```bash
# Run diagnostic script
cat > /tmp/takhin-diag.sh <<'EOF'
#!/bin/bash
echo "=== Takhin Diagnostics ==="
echo "Date: $(date)"
echo ""
echo "=== System Info ==="
uname -a
free -h
df -h
echo ""
echo "=== Takhin Status ==="
systemctl status takhin
echo ""
echo "=== Recent Logs ==="
journalctl -u takhin -n 100 --no-pager
echo ""
echo "=== Configuration ==="
cat /etc/takhin/takhin.yaml
echo ""
echo "=== Network ==="
netstat -tlnp | grep -E '9092|9090|7946'
echo ""
echo "=== Metrics ==="
curl -s http://localhost:9090/metrics
EOF

chmod +x /tmp/takhin-diag.sh
/tmp/takhin-diag.sh > /tmp/takhin-diag.txt 2>&1

# Share takhin-diag.txt when reporting issues
```

### Report Issues

When reporting issues, include:
1. Takhin version (`takhin -version`)
2. Operating system and version
3. Deployment type (standalone/cluster, bare metal/Docker/K8s)
4. Configuration file (redact sensitive info)
5. Recent logs (last 100 lines)
6. Steps to reproduce
7. Expected vs actual behavior

### Community Support

- GitHub Issues: https://github.com/takhin-data/takhin/issues
- Documentation: https://github.com/takhin-data/takhin/docs

---

## Preventive Measures

### Monitoring Checklist

- [ ] Set up Prometheus metrics scraping
- [ ] Monitor disk space alerts (< 20% free)
- [ ] Monitor memory usage alerts (> 80%)
- [ ] Monitor CPU usage alerts (> 80%)
- [ ] Monitor replication lag (cluster mode)
- [ ] Set up log aggregation (ELK, Loki, etc.)
- [ ] Configure health check alerts

### Maintenance Checklist

- [ ] Regularly review logs for warnings
- [ ] Monitor disk usage growth
- [ ] Review retention policies
- [ ] Test backup and restore procedures
- [ ] Update to latest stable version
- [ ] Review and optimize configuration
- [ ] Test failover procedures (cluster)

## Next Steps

- [Configuration Reference](./04-configuration-reference.md)
- [Monitoring Best Practices](../operations/monitoring.md)
- [Performance Tuning](../operations/performance-tuning.md)
