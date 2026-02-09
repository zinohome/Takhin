# Task 4.4 Integration Example

This example demonstrates how to integrate encryption at rest into a Takhin deployment.

## Scenario: Production Deployment with Encryption

### 1. Configuration Setup

**File: `/etc/takhin/takhin.yaml`**
```yaml
server:
  host: "0.0.0.0"
  port: 9092
  tls:
    enabled: true
    cert:
      file: "/etc/takhin/certs/server.crt"
    key:
      file: "/etc/takhin/certs/server.key"
    ca:
      file: "/etc/takhin/certs/ca.crt"
    client:
      auth: "require"
    min:
      version: "TLS1.2"

storage:
  data:
    dir: "/var/lib/takhin/data"
  log:
    segment:
      size: 1073741824  # 1GB
    retention:
      hours: 720        # 30 days
  
  # Encryption at Rest
  encryption:
    enabled: true
    algorithm: "aes-256-gcm"
    key:
      dir: "/var/lib/takhin/keys"

kafka:
  broker:
    id: 1
  cluster:
    brokers: [1, 2, 3]
  advertised:
    host: "kafka-1.example.com"
    port: 9092
```

### 2. Directory Setup

```bash
# Create data directory
sudo mkdir -p /var/lib/takhin/data
sudo chown takhin:takhin /var/lib/takhin/data
sudo chmod 750 /var/lib/takhin/data

# Create key directory (restricted access)
sudo mkdir -p /var/lib/takhin/keys
sudo chown takhin:takhin /var/lib/takhin/keys
sudo chmod 700 /var/lib/takhin/keys

# Create config directory
sudo mkdir -p /etc/takhin
sudo chown root:takhin /etc/takhin
sudo chmod 750 /etc/takhin
```

### 3. Start Takhin Server

```bash
# Start with systemd
sudo systemctl start takhin

# Or start manually
takhin -config /etc/takhin/takhin.yaml
```

**Expected startup log**:
```
INFO  Loaded config from file path=/etc/takhin/takhin.yaml
INFO  Encryption enabled algorithm=aes-256-gcm keyDir=/var/lib/takhin/keys
INFO  Created encryption key manager
INFO  Generated initial encryption key keyID=key-1234567890
INFO  Starting Takhin broker brokerID=1 port=9092
INFO  TLS enabled with encryption at rest
```

### 4. Verify Encryption

**Check that keys were generated**:
```bash
ls -la /var/lib/takhin/keys/
# Expected output:
# drwx------ 2 takhin takhin 4096 Jan 6 08:00 .
# -rw------- 1 takhin takhin   44 Jan 6 08:00 key-1234567890.key
```

**Create a test topic**:
```bash
# Using kafka-topics command
kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic encrypted-test \
  --partitions 3 \
  --replication-factor 3
```

**Produce test messages**:
```bash
# Using kafka-console-producer
kafka-console-producer \
  --broker-list localhost:9092 \
  --topic encrypted-test <<EOF
{"user": "alice", "action": "login", "ip": "192.168.1.100"}
{"user": "bob", "action": "purchase", "amount": 99.99}
{"user": "charlie", "action": "logout"}
EOF
```

**Verify data is encrypted on disk**:
```bash
# Check segment file (should see encrypted data)
sudo hexdump -C /var/lib/takhin/data/encrypted-test-0/00000000000000000000.log | head -20

# Output shows encrypted bytes (not plaintext JSON)
# 00000000  00 0e 6b 65 79 2d 31 32  33 34 35 36 37 38 39 30  |..key-1234567890|
# 00000010  00 00 00 6c a3 b7 c4 2f  89 1e 5d 4a 9b 3c ...    |...l../...]J.<..|
# (encrypted data, not readable)
```

**Consume and verify decryption works**:
```bash
kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic encrypted-test \
  --from-beginning

# Output (decrypted automatically):
# {"user": "alice", "action": "login", "ip": "192.168.1.100"}
# {"user": "bob", "action": "purchase", "amount": 99.99}
# {"user": "charlie", "action": "logout"}
```

### 5. Key Rotation Procedure

**Automated rotation script** (`rotate-keys.sh`):
```bash
#!/bin/bash
# Script to rotate encryption keys

echo "🔄 Rotating encryption keys..."

# Call Takhin API to rotate keys
curl -X POST http://localhost:8080/api/v1/encryption/rotate-key \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json"

if [ $? -eq 0 ]; then
  echo "✅ Key rotation successful"
  
  # Backup old keys
  BACKUP_DIR="/var/backups/takhin-keys/$(date +%Y%m%d)"
  sudo mkdir -p $BACKUP_DIR
  sudo cp /var/lib/takhin/keys/*.key $BACKUP_DIR/
  echo "📦 Keys backed up to $BACKUP_DIR"
else
  echo "❌ Key rotation failed"
  exit 1
fi
```

**Schedule with cron** (rotate every 90 days):
```cron
# /etc/cron.d/takhin-key-rotation
0 0 1 */3 * takhin /opt/takhin/scripts/rotate-keys.sh
```

### 6. Monitoring and Alerts

**Prometheus metrics to monitor**:
```yaml
# Alert when encryption overhead is too high
- alert: EncryptionOverheadHigh
  expr: takhin_encryption_duration_seconds > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Encryption taking too long"

# Alert when key rotation is needed
- alert: EncryptionKeyAge
  expr: time() - takhin_encryption_key_created_timestamp > 7776000  # 90 days
  labels:
    severity: info
  annotations:
    summary: "Encryption key rotation recommended"
```

### 7. Backup and Disaster Recovery

**Backup script** (`backup.sh`):
```bash
#!/bin/bash
# Backup both data and keys

BACKUP_ROOT="/var/backups/takhin"
DATE=$(date +%Y%m%d-%H%M%S)

# Backup keys (critical!)
echo "📦 Backing up encryption keys..."
sudo tar -czf "$BACKUP_ROOT/keys-$DATE.tar.gz" \
  -C /var/lib/takhin keys/

# Backup data (encrypted segments)
echo "📦 Backing up data..."
sudo tar -czf "$BACKUP_ROOT/data-$DATE.tar.gz" \
  -C /var/lib/takhin data/

# Verify backups
if [ -f "$BACKUP_ROOT/keys-$DATE.tar.gz" ] && \
   [ -f "$BACKUP_ROOT/data-$DATE.tar.gz" ]; then
  echo "✅ Backup complete"
  echo "   Keys: $BACKUP_ROOT/keys-$DATE.tar.gz"
  echo "   Data: $BACKUP_ROOT/data-$DATE.tar.gz"
else
  echo "❌ Backup failed!"
  exit 1
fi

# Upload to S3 (optional)
# aws s3 cp "$BACKUP_ROOT/keys-$DATE.tar.gz" s3://my-backups/takhin/keys/
# aws s3 cp "$BACKUP_ROOT/data-$DATE.tar.gz" s3://my-backups/takhin/data/
```

**Restore procedure**:
```bash
#!/bin/bash
# Restore from backup

BACKUP_ROOT="/var/backups/takhin"
RESTORE_DATE="20240106-120000"  # Specify backup date

echo "🔄 Restoring from backup..."

# Stop Takhin
sudo systemctl stop takhin

# Restore keys first (must match)
sudo rm -rf /var/lib/takhin/keys
sudo tar -xzf "$BACKUP_ROOT/keys-$RESTORE_DATE.tar.gz" \
  -C /var/lib/takhin/

# Restore data
sudo rm -rf /var/lib/takhin/data
sudo tar -xzf "$BACKUP_ROOT/data-$RESTORE_DATE.tar.gz" \
  -C /var/lib/takhin/

# Fix permissions
sudo chown -R takhin:takhin /var/lib/takhin
sudo chmod 700 /var/lib/takhin/keys

# Start Takhin
sudo systemctl start takhin

echo "✅ Restore complete"
```

### 8. Performance Tuning

**Optimize for AES-NI (Intel/AMD)**:
```bash
# Check if AES-NI is available
grep -o aes /proc/cpuinfo | head -1

# If available, ensure CPU governor is set to performance
sudo cpupower frequency-set -g performance

# Monitor CPU usage
mpstat -P ALL 1 10
```

**Optimize for ARM (use ChaCha20)**:
```yaml
storage:
  encryption:
    algorithm: "chacha20-poly1305"  # Better for ARM
```

### 9. Security Hardening

**Key directory security**:
```bash
# Use encrypted filesystem for keys
sudo cryptsetup luksFormat /dev/sdb1
sudo cryptsetup luksOpen /dev/sdb1 takhin-keys
sudo mkfs.ext4 /dev/mapper/takhin-keys
sudo mount /dev/mapper/takhin-keys /var/lib/takhin/keys
```

**SELinux context** (if using SELinux):
```bash
sudo semanage fcontext -a -t takhin_key_t "/var/lib/takhin/keys(/.*)?"
sudo restorecon -Rv /var/lib/takhin/keys
```

**AppArmor profile** (if using AppArmor):
```
/var/lib/takhin/keys/** rw,
/var/lib/takhin/keys/ r,
```

### 10. Compliance Verification

**Generate compliance report**:
```bash
#!/bin/bash
# Compliance verification script

echo "=== Takhin Encryption Compliance Report ==="
echo "Date: $(date)"
echo

# Check encryption is enabled
echo "1. Encryption Status:"
curl -s http://localhost:8080/api/v1/status | jq '.encryption.enabled'

# Check algorithm
echo "2. Algorithm:"
curl -s http://localhost:8080/api/v1/status | jq '.encryption.algorithm'

# Check key permissions
echo "3. Key Directory Permissions:"
ls -ld /var/lib/takhin/keys

# Check data encryption
echo "4. Sample Encrypted Segment:"
sudo hexdump -C /var/lib/takhin/data/*/00*.log | head -5

# Check TLS
echo "5. TLS Status:"
openssl s_client -connect localhost:9092 -showcerts < /dev/null 2>&1 | grep "Protocol"

echo
echo "=== End of Report ==="
```

## Summary

This integration example demonstrates:

✅ **Production configuration** with encryption enabled
✅ **Secure directory setup** with proper permissions
✅ **Key generation and management** handled automatically
✅ **Data encryption verification** on disk
✅ **Transparent decryption** for consumers
✅ **Key rotation procedures** for long-term security
✅ **Backup and recovery** strategies
✅ **Performance tuning** for different CPU architectures
✅ **Security hardening** best practices
✅ **Compliance verification** procedures

The encryption at rest feature integrates seamlessly with existing Takhin deployments while providing strong data protection.
