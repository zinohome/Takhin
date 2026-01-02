# Takhin Configuration Reference

Complete reference for all Takhin configuration options.

## Configuration Loading Order

Takhin uses [Koanf](https://github.com/knadh/koanf) for layered configuration:

1. **Default values** (hardcoded in application)
2. **YAML configuration file** (`takhin.yaml`)
3. **Environment variables** (prefix: `TAKHIN_`)

Environment variables override YAML settings. Example:
```bash
# YAML: server.port: 9092
# Override with: TAKHIN_SERVER_PORT=9093
```

## Configuration File Format

Configuration is defined in YAML format (typically `/etc/takhin/takhin.yaml`):

```yaml
server:
  host: "0.0.0.0"
  port: 9092

kafka:
  broker:
    id: 1
  # ... more settings
```

## Complete Configuration Reference

### Server Configuration

Controls the main server binding and network settings.

```yaml
server:
  host: "0.0.0.0"              # Server bind address
  port: 9092                    # Server port
```

**Environment Variables:**
- `TAKHIN_SERVER_HOST` - Server host address
- `TAKHIN_SERVER_PORT` - Server port number

**Defaults:**
- `host`: `"0.0.0.0"` (listen on all interfaces)
- `port`: `9092` (Kafka default port)

**Notes:**
- Use `0.0.0.0` to listen on all interfaces
- Use `127.0.0.1` for localhost-only (development)
- Ensure port is not blocked by firewall

---

### Kafka Protocol Configuration

Controls Kafka protocol behavior and cluster setup.

```yaml
kafka:
  broker:
    id: 1                       # Unique broker ID (required, must be unique in cluster)
  
  cluster:
    brokers: [1, 2, 3]          # List of all broker IDs in cluster
  
  listeners:
    - "tcp://0.0.0.0:9092"      # Internal listener addresses
  
  advertised:
    host: "localhost"           # Advertised hostname for clients
    port: 9092                  # Advertised port for clients
  
  max:
    message:
      bytes: 1048576            # Max message size (1MB default)
    connections: 1000           # Max concurrent connections
  
  request:
    timeout:
      ms: 30000                 # Request timeout (30 seconds)
  
  connection:
    timeout:
      ms: 60000                 # Connection idle timeout (60 seconds)
```

**Environment Variables:**
- `TAKHIN_KAFKA_BROKER_ID` - Broker ID
- `TAKHIN_KAFKA_CLUSTER_BROKERS` - Cluster broker list (JSON array format)
- `TAKHIN_KAFKA_ADVERTISED_HOST` - Advertised hostname
- `TAKHIN_KAFKA_ADVERTISED_PORT` - Advertised port
- `TAKHIN_KAFKA_MAX_MESSAGE_BYTES` - Max message size
- `TAKHIN_KAFKA_MAX_CONNECTIONS` - Max connections

**Important Settings:**

#### `broker.id`
- **Required**: Yes
- **Type**: Integer (1-2147483647)
- **Default**: None (must be set)
- **Must be unique** across cluster
- Cannot be changed after initial setup

#### `cluster.brokers`
- **Required**: Yes
- **Type**: Array of integers
- **Default**: `[1]` (single broker)
- **Cluster mode**: List all broker IDs, e.g., `[1, 2, 3]`
- Must be consistent across all brokers in cluster

#### `advertised.host`
- **Required**: Yes
- **Type**: String (hostname or IP)
- **Default**: `"localhost"`
- **Critical**: Clients use this to connect
- Use external hostname/IP for remote access
- Use load balancer address for cluster deployments

#### `max.message.bytes`
- **Type**: Integer (bytes)
- **Default**: `1048576` (1MB)
- **Range**: 1024 - 104857600 (1KB - 100MB)
- Must be >= largest expected message
- Impacts memory usage

---

### Storage Configuration

Controls data persistence, log management, and cleanup.

```yaml
storage:
  data:
    dir: "/tmp/takhin-data"     # Data directory path
  
  log:
    segment:
      size: 1073741824          # Log segment size (1GB)
    
    retention:
      hours: 168                # Retention period (7 days)
      bytes: 0                  # Max retention bytes (0 = unlimited)
    
    cleanup:
      interval:
        ms: 300000              # Cleanup interval (5 minutes)
    
    flush:
      interval:
        ms: 1000                # Flush interval (1 second)
      messages: 10000           # Flush after N messages
  
  cleaner:
    enabled: true               # Enable background cleaner
  
  compaction:
    interval:
      ms: 600000                # Compaction interval (10 minutes)
    min:
      cleanable:
        ratio: 0.5              # Min dirty ratio for compaction (50%)
```

**Environment Variables:**
- `TAKHIN_STORAGE_DATA_DIR` - Data directory
- `TAKHIN_STORAGE_LOG_SEGMENT_SIZE` - Segment size
- `TAKHIN_STORAGE_LOG_RETENTION_HOURS` - Retention hours
- `TAKHIN_STORAGE_LOG_RETENTION_BYTES` - Retention bytes
- `TAKHIN_STORAGE_CLEANER_ENABLED` - Enable cleaner

**Important Settings:**

#### `data.dir`
- **Required**: Yes
- **Type**: String (file path)
- **Default**: `"/tmp/takhin-data"`
- Must have write permissions
- **Production**: Use dedicated disk/mount (SSD recommended)
- Ensure sufficient disk space

#### `log.segment.size`
- **Type**: Integer (bytes)
- **Default**: `1073741824` (1GB)
- **Range**: 1048576 - 2147483648 (1MB - 2GB)
- Larger segments = fewer files, slower compaction
- Smaller segments = more files, faster compaction

#### `log.retention.hours`
- **Type**: Integer (hours)
- **Default**: `168` (7 days)
- **Range**: 1 - 8760 (1 hour - 1 year)
- `0` = keep forever
- Data older than this is deleted

#### `log.retention.bytes`
- **Type**: Integer (bytes)
- **Default**: `0` (unlimited)
- Max total size per partition
- `0` = unlimited
- When exceeded, oldest segments deleted

#### `log.flush.interval.ms`
- **Type**: Integer (milliseconds)
- **Default**: `1000` (1 second)
- How often to flush to disk
- **Lower** = more durability, higher I/O
- **Higher** = less durability, better performance

#### `log.flush.messages`
- **Type**: Integer (message count)
- **Default**: `10000`
- Flush after N messages accumulated
- Works with `flush.interval.ms`

#### `cleaner.enabled`
- **Type**: Boolean
- **Default**: `true`
- Enables background log cleanup and compaction
- Disable for read-only replicas

#### `compaction.min.cleanable.ratio`
- **Type**: Float (0.0 - 1.0)
- **Default**: `0.5` (50%)
- Min ratio of "dirty" records to trigger compaction
- Lower = more frequent compaction

---

### Replication Configuration

Controls data replication across cluster brokers.

```yaml
replication:
  default:
    replication:
      factor: 1                 # Default replication factor
  
  replica:
    lag:
      time:
        max:
          ms: 10000             # Max replica lag (10 seconds)
    
    fetch:
      wait:
        max:
          ms: 500               # Follower fetch wait time
      max:
        bytes: 1048576          # Max bytes per follower fetch (1MB)
```

**Environment Variables:**
- `TAKHIN_REPLICATION_DEFAULT_REPLICATION_FACTOR` - Default replication factor
- `TAKHIN_REPLICATION_REPLICA_LAG_TIME_MAX_MS` - Max replica lag
- `TAKHIN_REPLICATION_REPLICA_FETCH_MAX_BYTES` - Max fetch bytes

**Important Settings:**

#### `default.replication.factor`
- **Type**: Integer (1-3)
- **Default**: `1` (no replication)
- **Standalone**: `1`
- **Production cluster**: `2` or `3`
- Cannot exceed number of brokers
- Higher = more durability, more storage

#### `replica.lag.time.max.ms`
- **Type**: Integer (milliseconds)
- **Default**: `10000` (10 seconds)
- Max time replica can lag before removed from ISR
- Lower = stricter consistency, more failures
- Higher = more tolerance, potential data loss

#### `replica.fetch.max.bytes`
- **Type**: Integer (bytes)
- **Default**: `1048576` (1MB)
- Max data fetched per replication request
- Higher = fewer requests, more memory
- Should be >= `max.message.bytes`

---

### Logging Configuration

Controls application logging behavior.

```yaml
logging:
  level: "info"                 # Log level: debug, info, warn, error
  format: "json"                # Log format: json, text
```

**Environment Variables:**
- `TAKHIN_LOGGING_LEVEL` - Log level
- `TAKHIN_LOGGING_FORMAT` - Log format

**Log Levels:**
- `debug` - Verbose debugging information
- `info` - General informational messages (default)
- `warn` - Warning messages
- `error` - Error messages only

**Log Formats:**
- `json` - Structured JSON logs (recommended for production)
- `text` - Human-readable text logs (better for development)

---

### Metrics Configuration

Controls Prometheus metrics exposure.

```yaml
metrics:
  enabled: true                 # Enable metrics endpoint
  host: "0.0.0.0"              # Metrics server host
  port: 9090                    # Metrics server port
  path: "/metrics"              # Metrics endpoint path
```

**Environment Variables:**
- `TAKHIN_METRICS_ENABLED` - Enable/disable metrics
- `TAKHIN_METRICS_HOST` - Metrics host
- `TAKHIN_METRICS_PORT` - Metrics port
- `TAKHIN_METRICS_PATH` - Metrics path

**Metrics Endpoint:**
- Access at: `http://<host>:<port>/metrics`
- Default: `http://localhost:9090/metrics`
- Returns Prometheus-format metrics

**Key Metrics:**
- `takhin_kafka_requests_total` - Total Kafka requests by API
- `takhin_kafka_request_duration_seconds` - Request latency histogram
- `takhin_storage_bytes_total` - Total storage used
- `takhin_active_connections` - Current active connections
- `takhin_raft_state` - Raft cluster state (leader/follower)

---

## Configuration Examples

### Development (Single Node)

```yaml
server:
  host: "127.0.0.1"
  port: 9092

kafka:
  broker:
    id: 1
  cluster:
    brokers: [1]
  advertised:
    host: "localhost"
    port: 9092

storage:
  data:
    dir: "/tmp/takhin-dev"
  log:
    retention:
      hours: 24  # Short retention for dev

replication:
  default:
    replication:
      factor: 1

logging:
  level: "debug"
  format: "text"
```

### Production (3-Node Cluster)

**Broker 1:**
```yaml
server:
  host: "0.0.0.0"
  port: 9092

kafka:
  broker:
    id: 1
  cluster:
    brokers: [1, 2, 3]
  advertised:
    host: "broker1.prod.example.com"
    port: 9092
  max:
    message:
      bytes: 10485760  # 10MB
    connections: 5000

storage:
  data:
    dir: "/data/takhin"
  log:
    segment:
      size: 2147483648  # 2GB
    retention:
      hours: 168  # 7 days
    flush:
      interval:
        ms: 5000
      messages: 100000

replication:
  default:
    replication:
      factor: 3

logging:
  level: "info"
  format: "json"

metrics:
  enabled: true
  port: 9090
```

**Broker 2 & 3:** Same config, change `broker.id` and `advertised.host`

### High Throughput

```yaml
kafka:
  max:
    message:
      bytes: 10485760
    connections: 10000

storage:
  log:
    segment:
      size: 2147483648
    flush:
      interval:
        ms: 10000  # Less frequent flushes
      messages: 100000
    cleanup:
      interval:
        ms: 600000

metrics:
  enabled: false  # Disable if not needed
```

### High Durability

```yaml
storage:
  log:
    flush:
      interval:
        ms: 100  # Frequent flushes
      messages: 1000

replication:
  default:
    replication:
      factor: 3
  replica:
    lag:
      time:
        max:
          ms: 5000  # Strict ISR requirements
```

## Console Server Configuration

The Console server (separate binary) has different configuration:

### Command-Line Flags

```bash
takhin-console \
  -data-dir /var/lib/takhin/data \
  -api-addr :8080 \
  -enable-auth \
  -api-keys "key1,key2,key3"
```

**Flags:**
- `-data-dir string` - Data directory for topics (default: `/tmp/takhin-console-data`)
- `-api-addr string` - API server address (default: `:8080`)
- `-enable-auth` - Enable API key authentication (default: `false`)
- `-api-keys string` - Comma-separated list of valid API keys

### Environment Variables

```bash
export DATA_DIR=/var/lib/takhin/data
export API_ADDR=:8080
export ENABLE_AUTH=true
export API_KEYS=secret-key-1,secret-key-2
```

## Validation

### Check Configuration

```bash
# Test configuration file syntax
takhin -config /etc/takhin/takhin.yaml -validate

# Dry-run with config
takhin -config /etc/takhin/takhin.yaml -dry-run
```

### Common Validation Errors

1. **Invalid broker ID**: Must be positive integer
2. **Missing data directory**: Path must exist or be creatable
3. **Port already in use**: Change port or stop conflicting service
4. **Invalid cluster brokers**: Must be valid JSON array

## Performance Tuning Guide

### High Throughput

- Increase `max.message.bytes` to 10MB+
- Increase `log.segment.size` to 2GB
- Increase `log.flush.interval.ms` to 5000-10000ms
- Increase `log.flush.messages` to 100000+
- Disable metrics if not needed

### Low Latency

- Decrease `log.flush.interval.ms` to 100-500ms
- Decrease `replica.fetch.wait.max.ms` to 100-200ms
- Use SSD storage
- Ensure low network latency between brokers

### High Availability

- Set `replication.factor` to 3
- Deploy across multiple availability zones
- Use dedicated storage volumes
- Enable log compaction

### Large Messages

- Increase `max.message.bytes` to 50MB+
- Increase `replica.fetch.max.bytes` accordingly
- Increase segment size to accommodate large messages
- Monitor memory usage

## Troubleshooting Configuration Issues

### Issue: Broker won't start

**Check:**
1. Configuration file syntax (valid YAML)
2. Data directory permissions
3. Port availability
4. Logs for specific errors

### Issue: Clients can't connect

**Check:**
1. `advertised.host` is correct hostname/IP
2. Firewall allows port 9092
3. Network connectivity between client and broker
4. Load balancer configuration (if applicable)

### Issue: High memory usage

**Reduce:**
1. `max.connections`
2. `max.message.bytes`
3. `replica.fetch.max.bytes`

### Issue: Data not being retained

**Check:**
1. `log.retention.hours` not too low
2. `log.retention.bytes` not exceeded
3. Cleaner is enabled
4. Sufficient disk space

## Next Steps

- [Troubleshooting Guide](./05-troubleshooting.md)
- [Monitoring Best Practices](../operations/monitoring.md)
- [Performance Tuning](../operations/performance-tuning.md)
