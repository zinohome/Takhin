# ACL Quick Reference

## Enable ACLs

```yaml
# configs/takhin.yaml
acl:
  enabled: true
```

## REST API Endpoints

```bash
# List all ACLs
curl http://localhost:8080/api/acls

# Filter by resource type
curl http://localhost:8080/api/acls?resource_type=Topic&principal=User:alice

# Create ACL
curl -X POST http://localhost:8080/api/acls \
  -H "Content-Type: application/json" \
  -d '{
    "principal": "User:alice",
    "host": "*",
    "resource_type": "Topic",
    "resource_name": "test-topic",
    "pattern_type": "Literal",
    "operation": "Read",
    "permission_type": "Allow"
  }'

# Delete ACLs
curl -X DELETE http://localhost:8080/api/acls \
  -H "Content-Type: application/json" \
  -d '{
    "resource_type": "Topic",
    "principal": "User:alice"
  }'
```

## Common ACL Patterns

### Grant read access to topic
```json
{
  "principal": "User:consumer",
  "host": "*",
  "resource_type": "Topic",
  "resource_name": "orders-topic",
  "pattern_type": "Literal",
  "operation": "Read",
  "permission_type": "Allow"
}
```

### Grant write access with prefix
```json
{
  "principal": "User:producer",
  "host": "*",
  "resource_type": "Topic",
  "resource_name": "logs-",
  "pattern_type": "Prefixed",
  "operation": "Write",
  "permission_type": "Allow"
}
```

### Full admin access
```json
{
  "principal": "User:admin",
  "host": "*",
  "resource_type": "Topic",
  "resource_name": "*",
  "pattern_type": "Literal",
  "operation": "All",
  "permission_type": "Allow"
}
```

### Deny sensitive topic
```json
{
  "principal": "*",
  "host": "*",
  "resource_type": "Topic",
  "resource_name": "pii-data",
  "pattern_type": "Literal",
  "operation": "All",
  "permission_type": "Deny"
}
```

### IP-restricted access
```json
{
  "principal": "User:service",
  "host": "192.168.1.100",
  "resource_type": "Topic",
  "resource_name": "secure-topic",
  "pattern_type": "Literal",
  "operation": "Write",
  "permission_type": "Allow"
}
```

### Consumer group access
```json
{
  "principal": "User:consumer",
  "host": "*",
  "resource_type": "Group",
  "resource_name": "my-consumer-group",
  "pattern_type": "Literal",
  "operation": "Read",
  "permission_type": "Allow"
}
```

## Field Values

### Resource Types
- `Topic` - Kafka topics
- `Group` - Consumer groups
- `Cluster` - Cluster operations

### Pattern Types
- `Literal` - Exact match
- `Prefixed` - Prefix match

### Operations
- `Read` - Fetch/consume
- `Write` - Produce
- `Create` - Create resources
- `Delete` - Delete resources
- `Alter` - Modify configs
- `Describe` - View metadata
- `ClusterAction` - Admin operations
- `All` - Any operation

### Permission Types
- `Allow` - Grant access
- `Deny` - Deny access (takes precedence)

## Authorization Logic

1. If ACL disabled → **Allow**
2. Check for DENY → If found, **Deny**
3. Check for ALLOW → If found, **Allow**
4. No match → **Deny** (default deny)

## Testing

```bash
# Run ACL tests
go test ./pkg/acl/...

# Run benchmarks
go test -bench=. ./pkg/acl/...

# Expected performance
# - Disabled: ~1ns per check
# - Single ACL: ~38ns per check
# - 100 ACLs: ~145ns per check
```

## Files

- Config: `configs/takhin.yaml`
- Storage: `<data-dir>/acls.json`
- Code: `backend/pkg/acl/`
- Tests: `backend/pkg/acl/*_test.go`
- Handler: `backend/pkg/kafka/handler/acl.go`
- Console API: `backend/pkg/console/acl_handlers.go`

## Environment Variables

```bash
# Enable/disable ACLs
export TAKHIN_ACL_ENABLED=true
```
