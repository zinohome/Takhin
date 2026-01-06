# Audit Logging - Usage Examples

## Quick Start

### 1. Enable Audit Logging in Console

```bash
# Basic usage
./console -enable-audit -data-dir /var/lib/takhin

# With custom audit path
./console -enable-audit -audit-path /var/log/takhin/audit.log -data-dir /var/lib/takhin

# With authentication
./console -enable-audit -enable-auth -api-keys "key1,key2" -data-dir /var/lib/takhin
```

### 2. Configure via YAML

```yaml
# configs/takhin.yaml
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
  store:
    enabled: true
    retention:
      ms: 604800000  # 7 days
```

Then start console:
```bash
./console -data-dir /var/lib/takhin
```

### 3. Programmatic Usage

```go
package main

import (
    "github.com/takhin-data/takhin/pkg/audit"
    "log"
)

func main() {
    // Initialize audit logger
    auditLogger, err := audit.NewLogger(audit.Config{
        Enabled:          true,
        OutputPath:       "/var/log/takhin/audit.log",
        MaxFileSize:      100 * 1024 * 1024,
        MaxBackups:       10,
        MaxAge:           30,
        Compress:         true,
        StoreEnabled:     true,
        StoreRetentionMs: 7 * 24 * 60 * 60 * 1000,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer auditLogger.Close()

    // Use the logger
    auditLogger.LogAuth("admin", "192.168.1.100", "success", "api-key-123", nil)
}
```

## Common Scenarios

### Scenario 1: Track Failed Login Attempts

```bash
# Query failed authentication attempts from the last hour
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "event_types": ["auth.failure"],
    "start_time": "'$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)'",
    "limit": 50
  }'
```

Response:
```json
{
  "events": [
    {
      "timestamp": "2026-01-06T08:15:32Z",
      "event_id": "550e8400-e29b-41d4-a716-446655440001",
      "event_type": "auth.failure",
      "severity": "warning",
      "principal": "unknown",
      "host": "192.168.1.55",
      "operation": "authenticate",
      "result": "failure",
      "error": "invalid API key",
      "metadata": {
        "api_key_prefix": "abcd****"
      }
    }
  ],
  "total_count": 1,
  "limit": 50,
  "offset": 0
}
```

### Scenario 2: Audit User Activity

```bash
# Get all activity for a specific user
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "principals": ["user1"],
    "start_time": "'$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ)'",
    "limit": 100
  }'
```

### Scenario 3: Monitor Topic Operations

```bash
# Track all operations on a specific topic
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_type": "topic",
    "resource_name": "orders",
    "limit": 100
  }'
```

### Scenario 4: Find ACL Violations

```bash
# Query ACL denial events
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "event_types": ["acl.deny"],
    "severity": "warning",
    "limit": 50
  }'
```

### Scenario 5: Export Compliance Report

```bash
# Export last 30 days as CSV
START_TIME=$(date -u -d '30 days ago' +%Y-%m-%dT%H:%M:%SZ)
END_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

curl "http://localhost:8080/api/audit/export?format=csv&start_time=${START_TIME}&end_time=${END_TIME}&limit=10000" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -o compliance-report.csv
```

### Scenario 6: Get Audit Statistics

```bash
# Get statistics for the last 7 days
curl "http://localhost:8080/api/audit/stats?start_time=$(date -u -d '7 days ago' +%Y-%m-%dT%H:%M:%SZ)" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

Response:
```json
{
  "total_events": 1532,
  "by_type": {
    "auth.success": 850,
    "auth.failure": 23,
    "topic.create": 45,
    "topic.delete": 12,
    "data.produce": 500,
    "data.consume": 102
  },
  "by_severity": {
    "info": 1450,
    "warning": 75,
    "error": 7
  },
  "by_principal": {
    "admin": 600,
    "user1": 500,
    "user2": 432
  },
  "by_result": {
    "success": 1509,
    "failure": 23
  }
}
```

## Advanced Queries

### Multi-criteria Query

```bash
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "event_types": ["topic.create", "topic.delete"],
    "principals": ["admin", "operator"],
    "start_time": "'$(date -u -d '7 days ago' +%Y-%m-%dT%H:%M:%SZ)'",
    "severity": "info",
    "result": "success",
    "limit": 100,
    "offset": 0
  }'
```

### Pagination Example

```bash
# First page
curl -X POST http://localhost:8080/api/audit/logs \
  -d '{"limit": 10, "offset": 0}' -H "Authorization: Bearer KEY"

# Second page
curl -X POST http://localhost:8080/api/audit/logs \
  -d '{"limit": 10, "offset": 10}' -H "Authorization: Bearer KEY"

# Third page
curl -X POST http://localhost:8080/api/audit/logs \
  -d '{"limit": 10, "offset": 20}' -H "Authorization: Bearer KEY"
```

## Programmatic Examples

### Log Authentication Events

```go
// Successful authentication
auditLogger.LogAuth("user1", "192.168.1.100", "success", "api-key-123", nil)

// Failed authentication
err := errors.New("invalid credentials")
auditLogger.LogAuth("user1", "192.168.1.100", "failure", "wrong-key", err)
```

### Log ACL Operations

```go
// ACL entry created
auditLogger.LogACL("create", "admin", "localhost", "topic", "orders", "success", nil)

// Access denied
err := errors.New("insufficient permissions")
auditLogger.LogACL("deny", "user1", "192.168.1.100", "topic", "secret-data", "denied", err)
```

### Log Topic Operations

```go
// Topic created
auditLogger.LogTopic("create", "admin", "localhost", "orders", 3, "success", nil)

// Topic deleted
auditLogger.LogTopic("delete", "admin", "localhost", "old-topic", 0, "success", nil)

// Topic update failed
err := errors.New("invalid partition count")
auditLogger.LogTopic("update", "user1", "localhost", "orders", 10, "failure", err)
```

### Log Data Access

```go
// Message produced
auditLogger.LogDataAccess("produce", "producer1", "192.168.1.100", "orders", 0, 1000, 2048)

// Message consumed
auditLogger.LogDataAccess("consume", "consumer1", "192.168.1.101", "orders", 0, 1000, 2048)
```

### Custom Events

```go
auditLogger.Log(&audit.Event{
    EventType:    audit.EventTypeConfigChange,
    Severity:     audit.SeverityWarning,
    Principal:    "admin",
    Host:         "localhost",
    ResourceType: "cluster",
    ResourceName: "config",
    Operation:    "update",
    Result:       "success",
    Metadata: map[string]interface{}{
        "setting": "max.connections",
        "old_value": 1000,
        "new_value": 2000,
    },
    RequestID: "req-123",
    Duration:  45,
})
```

### Query Events in Code

```go
// Query by time range
startTime := time.Now().Add(-24 * time.Hour)
endTime := time.Now()
events, err := auditLogger.Query(audit.Filter{
    StartTime: &startTime,
    EndTime:   &endTime,
})

// Query by event type
events, err := auditLogger.Query(audit.Filter{
    EventTypes: []audit.EventType{
        audit.EventTypeAuthFailure,
        audit.EventTypeACLDeny,
    },
    Limit: 100,
})

// Query by principal
events, err := auditLogger.Query(audit.Filter{
    Principals: []string{"user1", "user2"},
})

// Query by resource
events, err := auditLogger.Query(audit.Filter{
    ResourceType: "topic",
    ResourceName: "orders",
})

// Complex query
events, err := auditLogger.Query(audit.Filter{
    StartTime:    &startTime,
    EventTypes:   []audit.EventType{audit.EventTypeDataProduce},
    ResourceType: "topic",
    Severity:     audit.SeverityWarning,
    Limit:        100,
    Offset:       0,
})
```

## Integration with Monitoring

### Export to Elasticsearch

```bash
#!/bin/bash
# export-to-elasticsearch.sh

# Get audit logs
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"limit": 1000}' | \
jq -c '.events[]' | \
while read event; do
  # Index each event in Elasticsearch
  curl -X POST "http://elasticsearch:9200/takhin-audit/_doc" \
    -H "Content-Type: application/json" \
    -d "$event"
done
```

### Create Alerts

```bash
#!/bin/bash
# alert-on-failures.sh

# Check for recent failures
FAILURES=$(curl -s -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "event_types": ["auth.failure"],
    "start_time": "'$(date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%SZ)'"
  }' | jq '.total_count')

if [ "$FAILURES" -gt 5 ]; then
  echo "ALERT: $FAILURES failed login attempts in last 5 minutes"
  # Send alert (email, Slack, PagerDuty, etc.)
fi
```

## Log Analysis with jq

```bash
# Count events by type
cat /var/log/takhin/audit.log | jq -r '.event_type' | sort | uniq -c

# Find all failed operations
cat /var/log/takhin/audit.log | jq 'select(.result == "failure")'

# Extract IP addresses with failed auth
cat /var/log/takhin/audit.log | jq -r 'select(.event_type == "auth.failure") | .host'

# Group by principal
cat /var/log/takhin/audit.log | jq -r '.principal' | sort | uniq -c | sort -rn

# Find operations on specific topic
cat /var/log/takhin/audit.log | jq 'select(.resource_name == "orders")'
```

## Best Practices

1. **Regular Review**: Review audit logs regularly for suspicious activity
2. **Retention Policy**: Set appropriate retention based on compliance requirements
3. **Alerting**: Set up alerts for critical events (repeated failures, ACL denials)
4. **Export**: Regularly export logs to external storage for long-term retention
5. **Monitoring**: Monitor audit log file size and rotation
6. **Access Control**: Restrict access to audit logs (file permissions, API authentication)
7. **Correlation**: Use request IDs to correlate audit events with application logs

## Troubleshooting

### No audit logs appearing
```bash
# Check if audit logging is enabled
grep "audit:" /path/to/takhin.yaml

# Check log file permissions
ls -la /var/log/takhin/audit.log

# Check disk space
df -h /var/log/takhin/

# Check application logs for errors
grep "audit" /var/log/takhin/takhin.log
```

### Query returns empty results
```bash
# Verify store is enabled
grep "store.enabled" /path/to/takhin.yaml

# Check retention period
# Events older than retention are automatically cleaned up

# Try query without filters first
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"limit": 10}'
```
