# Audit Logging - Quick Reference

## Configuration

### YAML Configuration
```yaml
audit:
  enabled: true
  output:
    path: "/var/log/takhin/audit.log"
  max:
    file:
      size: 104857600           # 100MB
    backups: 10
    age: 30
  compress: true
  store:
    enabled: true
    retention:
      ms: 604800000             # 7 days
```

### Environment Variables
```bash
TAKHIN_AUDIT_ENABLED=true
TAKHIN_AUDIT_OUTPUT_PATH=/var/log/takhin/audit.log
TAKHIN_AUDIT_MAX_FILE_SIZE=104857600
TAKHIN_AUDIT_MAX_BACKUPS=10
TAKHIN_AUDIT_MAX_AGE=30
TAKHIN_AUDIT_COMPRESS=true
TAKHIN_AUDIT_STORE_ENABLED=true
TAKHIN_AUDIT_STORE_RETENTION_MS=604800000
```

### Command Line (Console)
```bash
./console -enable-audit -audit-path /var/log/takhin/audit.log
```

## Event Types

| Event Type | Description | Severity |
|------------|-------------|----------|
| `auth.success` | Successful authentication | info |
| `auth.failure` | Failed authentication | warning |
| `acl.create` | ACL entry created | info |
| `acl.deny` | Access denied by ACL | warning |
| `topic.create` | Topic created | info |
| `topic.delete` | Topic deleted | info |
| `data.produce` | Messages produced | info |
| `data.consume` | Messages consumed | info |
| `config.change` | Configuration changed | warning |
| `system.error` | System error | error |

## API Endpoints

### Query Audit Logs
```http
POST /api/audit/logs
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json

{
  "event_types": ["auth.failure", "acl.deny"],
  "start_time": "2026-01-01T00:00:00Z",
  "end_time": "2026-01-06T23:59:59Z",
  "principals": ["user1"],
  "resource_type": "topic",
  "resource_name": "orders",
  "severity": "warning",
  "limit": 100,
  "offset": 0
}
```

### Get Audit Statistics
```http
GET /api/audit/stats?start_time=2026-01-01T00:00:00Z&end_time=2026-01-06T23:59:59Z
Authorization: Bearer YOUR_API_KEY
```

Response:
```json
{
  "total_events": 1234,
  "by_type": {
    "auth.success": 800,
    "auth.failure": 45,
    "topic.create": 20
  },
  "by_severity": {
    "info": 1000,
    "warning": 200,
    "error": 34
  },
  "by_principal": {
    "admin": 500,
    "user1": 400
  },
  "by_result": {
    "success": 1100,
    "failure": 134
  }
}
```

### Get Specific Event
```http
GET /api/audit/events/{event_id}
Authorization: Bearer YOUR_API_KEY
```

### Export Audit Logs
```bash
# Export as JSON
curl "http://localhost:8080/api/audit/export?format=json&limit=1000" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -o audit.json

# Export as CSV
curl "http://localhost:8080/api/audit/export?format=csv&start_time=2026-01-01T00:00:00Z" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -o audit.csv
```

## Programmatic Usage

### Initialize Audit Logger
```go
import "github.com/takhin-data/takhin/pkg/audit"

auditLogger, err := audit.NewLogger(audit.Config{
    Enabled:          true,
    OutputPath:       "/var/log/takhin/audit.log",
    MaxFileSize:      100 * 1024 * 1024, // 100MB
    MaxBackups:       10,
    MaxAge:           30,
    Compress:         true,
    StoreEnabled:     true,
    StoreRetentionMs: 7 * 24 * 60 * 60 * 1000, // 7 days
})
if err != nil {
    log.Fatal(err)
}
defer auditLogger.Close()
```

### Log Events

```go
// Authentication
auditLogger.LogAuth("user1", "192.168.1.100", "success", "api-key-123", nil)
auditLogger.LogAuth("user2", "192.168.1.101", "failure", "bad-key", errors.New("invalid credentials"))

// ACL Operations
auditLogger.LogACL("create", "admin", "localhost", "topic", "orders", "success", nil)
auditLogger.LogACL("deny", "user1", "192.168.1.100", "topic", "secret", "denied", 
    errors.New("insufficient permissions"))

// Topic Operations
auditLogger.LogTopic("create", "admin", "localhost", "orders", 3, "success", nil)
auditLogger.LogTopic("delete", "admin", "localhost", "old-topic", 0, "success", nil)

// Data Access
auditLogger.LogDataAccess("produce", "producer1", "192.168.1.100", "orders", 0, 1000, 2048)
auditLogger.LogDataAccess("consume", "consumer1", "192.168.1.101", "orders", 0, 1000, 2048)

// Custom Events
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
        "key":   "max.connections",
        "value": 2000,
    },
})
```

### Query Events

```go
// Query by event type
events, err := auditLogger.Query(audit.Filter{
    EventTypes: []audit.EventType{
        audit.EventTypeAuthFailure,
        audit.EventTypeACLDeny,
    },
    Limit: 100,
})

// Query by time range
startTime := time.Now().Add(-24 * time.Hour)
endTime := time.Now()
events, err := auditLogger.Query(audit.Filter{
    StartTime: &startTime,
    EndTime:   &endTime,
})

// Query by principal
events, err := auditLogger.Query(audit.Filter{
    Principals: []string{"user1", "user2"},
    Limit:      50,
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

## Log File Structure

### Log Directory Layout
```
/var/log/takhin/
├── audit.log                    # Current log file
├── audit.log.2026-01-06T10-30-00
├── audit.log.2026-01-05T08-15-00.gz
└── audit.log.2026-01-04T12-45-00.gz
```

### Log Entry Format
```json
{
  "timestamp": "2026-01-06T08:18:38.954Z",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "topic.create",
  "severity": "info",
  "principal": "admin",
  "host": "192.168.1.100",
  "user_agent": "takhin-cli/1.0",
  "resource_type": "topic",
  "resource_name": "orders",
  "operation": "create",
  "result": "success",
  "metadata": {
    "partitions": 3
  },
  "request_id": "req-123",
  "duration_ms": 45
}
```

## Common Queries

### Find Failed Authentication Attempts
```bash
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "event_types": ["auth.failure"],
    "limit": 100
  }'
```

### Find Recent ACL Denials
```bash
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "event_types": ["acl.deny"],
    "start_time": "'$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)'"
  }'
```

### Audit User Activity
```bash
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "principals": ["user1"],
    "start_time": "'$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ)'"
  }'
```

### Find Operations on Specific Topic
```bash
curl -X POST http://localhost:8080/api/audit/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "resource_type": "topic",
    "resource_name": "orders"
  }'
```

## Performance Tips

1. **Enable Store**: Set `store.enabled: true` for fast queries
2. **Adjust Retention**: Reduce `store.retention.ms` for memory savings
3. **Use Filters**: Leverage indexed fields (principal, resource)
4. **Limit Results**: Always use `limit` parameter in queries
5. **Compress Old Logs**: Enable `compress: true` to save disk space
6. **Monitor Size**: Watch log file size and rotation frequency

## Troubleshooting

### Audit Logs Not Written
- Check `audit.enabled` is `true`
- Verify output path is writable
- Check disk space
- Review application logs for errors

### Query Returns No Results
- Verify `store.enabled: true`
- Check retention period hasn't expired
- Ensure events were logged after logger started
- Try query without filters first

### High Memory Usage
- Reduce `store.retention.ms`
- Disable store if queries not needed: `store.enabled: false`
- Increase cleanup frequency (modify cleanup loop)

### Log Files Too Large
- Reduce `max.file.size`
- Enable `compress: true`
- Reduce `max.backups`
- Implement external log shipping

## Security Considerations

- API keys are masked (first 4 chars only)
- Log files should have restricted permissions (0640)
- Consider encrypting log files at rest
- Ship logs to external secure storage
- Implement log integrity checking
- Regular log review and analysis
- Alert on suspicious patterns

## Related Files

- Implementation: `backend/pkg/audit/`
- Configuration: `backend/pkg/config/config.go`
- Console Integration: `backend/pkg/console/audit_handlers.go`
- Tests: `backend/pkg/audit/logger_test.go`
