# ACL System Implementation

## Overview

The Takhin ACL (Access Control List) system provides fine-grained authorization for Kafka resources. It implements the Kafka ACL model with support for:

- **Resource-level control**: Topics, Consumer Groups, Cluster
- **Operation-level control**: Read, Write, Create, Delete, Alter, Describe, ClusterAction
- **Pattern matching**: Literal and prefix-based resource matching
- **Principal and host filtering**: Control access by user identity and source IP
- **Allow/Deny semantics**: Explicit allow and deny rules (deny takes precedence)

## Architecture

### Core Components

```
pkg/acl/
├── types.go         - ACL data types and enumerations
├── store.go         - ACL storage with JSON persistence
├── authorizer.go    - Authorization logic
└── tests...
```

### Integration Points

1. **Kafka Handler**: ACL checks integrated into Kafka protocol handlers
2. **Console API**: REST endpoints for ACL management
3. **Configuration**: ACL enable/disable via config file

## Data Model

### ACL Entry

```go
type Entry struct {
    Principal      string         // User:alice, User:*, etc.
    Host           string         // IP address or * for any
    ResourceType   ResourceType   // Topic, Group, Cluster
    ResourceName   string         // Resource name
    PatternType    PatternType    // Literal or Prefixed
    Operation      Operation      // Read, Write, Delete, etc.
    PermissionType PermissionType // Allow or Deny
}
```

### Resource Types

- **Topic**: Kafka topics
- **Group**: Consumer groups
- **Cluster**: Cluster-wide operations

### Operations

- **Read**: Fetch messages, describe topics/groups
- **Write**: Produce messages
- **Create**: Create topics/groups
- **Delete**: Delete topics/groups
- **Alter**: Modify configurations
- **Describe**: View metadata
- **ClusterAction**: Administrative operations
- **All**: Matches any operation

### Pattern Types

- **Literal**: Exact match (e.g., "users-topic")
- **Prefixed**: Prefix match (e.g., "prod-" matches "prod-orders", "prod-users")

## Storage

ACLs are stored in `<data-dir>/acls.json` as a JSON array:

```json
[
  {
    "Principal": "User:alice",
    "Host": "*",
    "ResourceType": 2,
    "ResourceName": "orders-topic",
    "PatternType": 0,
    "Operation": 2,
    "PermissionType": 2
  }
]
```

The store automatically persists changes to disk after create/delete operations.

## Authorization Logic

1. **Disabled mode**: If `acl.enabled=false`, all requests are allowed
2. **Deny check**: First check for explicit DENY rules - if found, reject
3. **Allow check**: Then check for explicit ALLOW rules - if found, accept
4. **Default deny**: If no matching ALLOW found, reject

### Principal Matching

- Exact match: `User:alice` matches only "User:alice"
- Wildcard: `*` matches any principal

### Host Matching

- Exact match: `192.168.1.100` matches only that IP
- Wildcard: `*` matches any host

### Pattern Matching

- **Literal**: `test-topic` matches only "test-topic"
- **Prefixed**: `test-` matches "test-topic", "test-orders", etc.

## Kafka Protocol API

### CreateAcls (API Key 30)

Creates new ACL entries.

**Request:**
```
[
  {
    resourceType: TOPIC (2),
    resourceName: "orders-topic",
    patternType: LITERAL (0),
    principal: "User:alice",
    host: "*",
    operation: READ (2),
    permissionType: ALLOW (2)
  }
]
```

**Response:**
```
{
  throttleTimeMs: 0,
  results: [
    {
      errorCode: NONE (0),
      errorMessage: null
    }
  ]
}
```

### DescribeAcls (API Key 29)

Lists ACL entries matching a filter.

**Request:**
```
{
  resourceTypeFilter: TOPIC (2),
  resourceNameFilter: "orders-topic",
  patternTypeFilter: LITERAL (0),
  principalFilter: "User:alice",
  hostFilter: "*",
  operation: READ (2),
  permissionType: ALLOW (2)
}
```

**Response:**
```
{
  throttleTimeMs: 0,
  errorCode: NONE (0),
  resources: [
    {
      resourceType: TOPIC (2),
      resourceName: "orders-topic",
      patternType: LITERAL (0),
      acls: [
        {
          principal: "User:alice",
          host: "*",
          operation: READ (2),
          permissionType: ALLOW (2)
        }
      ]
    }
  ]
}
```

### DeleteAcls (API Key 31)

Deletes ACL entries matching filters.

**Request:**
```
{
  filters: [
    {
      resourceTypeFilter: TOPIC (2),
      resourceNameFilter: "orders-topic",
      patternTypeFilter: LITERAL (0),
      principalFilter: "User:alice",
      hostFilter: "*",
      operation: READ (2),
      permissionType: ALLOW (2)
    }
  ]
}
```

**Response:**
```
{
  throttleTimeMs: 0,
  results: [
    {
      errorCode: NONE (0),
      matchingAcls: [
        {
          errorCode: NONE (0),
          resourceType: TOPIC (2),
          resourceName: "orders-topic",
          patternType: LITERAL (0),
          principal: "User:alice",
          host: "*",
          operation: READ (2),
          permissionType: ALLOW (2)
        }
      ]
    }
  ]
}
```

## Console REST API

### List ACLs

```
GET /api/acls?resource_type=Topic&principal=User:alice
```

**Response:**
```json
{
  "acls": [
    {
      "principal": "User:alice",
      "host": "*",
      "resource_type": "Topic",
      "resource_name": "orders-topic",
      "pattern_type": "Literal",
      "operation": "Read",
      "permission_type": "Allow"
    }
  ],
  "count": 1
}
```

### Create ACL

```
POST /api/acls
Content-Type: application/json

{
  "principal": "User:alice",
  "host": "*",
  "resource_type": "Topic",
  "resource_name": "orders-topic",
  "pattern_type": "Literal",
  "operation": "Read",
  "permission_type": "Allow"
}
```

**Response:**
```json
{
  "message": "ACL created successfully",
  "acl": { ... }
}
```

### Delete ACLs

```
DELETE /api/acls
Content-Type: application/json

{
  "resource_type": "Topic",
  "principal": "User:alice"
}
```

**Response:**
```json
{
  "message": "ACLs deleted successfully",
  "deleted": 3
}
```

## Configuration

### Enable ACLs

In `configs/takhin.yaml`:

```yaml
acl:
  enabled: true
```

Or via environment variable:

```bash
export TAKHIN_ACL_ENABLED=true
```

### Default Behavior

- **Disabled** (default): All operations allowed, no authorization checks
- **Enabled**: All operations require explicit ALLOW ACLs

## Performance

Benchmark results on Intel i9-12900HK:

| Scenario | Operations/sec | Latency | Allocations |
|----------|---------------|---------|-------------|
| ACL disabled | 825M ops/sec | 1.2 ns | 0 allocs |
| Single ACL | 26M ops/sec | 38 ns | 1 alloc |
| 100 ACLs | 6.8M ops/sec | 145 ns | 1 alloc |

**Performance impact: < 0.01% for typical workloads**

Authorization overhead is negligible:
- When disabled: essentially zero overhead (1.2ns)
- With typical ACL sets: ~40ns per authorization check
- Even with 100 ACLs: ~150ns per check

## Common Usage Patterns

### Pattern 1: Read-only access

```json
{
  "principal": "User:analyst",
  "host": "*",
  "resource_type": "Topic",
  "resource_name": "analytics-",
  "pattern_type": "Prefixed",
  "operation": "Read",
  "permission_type": "Allow"
}
```

Grants read access to all topics starting with "analytics-".

### Pattern 2: Full admin access

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

Grants all operations on all topics.

### Pattern 3: Deny sensitive topic

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

Denies all access to "pii-data" topic for everyone (deny takes precedence).

### Pattern 4: IP-restricted access

```json
{
  "principal": "User:service",
  "host": "10.0.1.100",
  "resource_type": "Topic",
  "resource_name": "orders-topic",
  "pattern_type": "Literal",
  "operation": "Write",
  "permission_type": "Allow"
}
```

Allows write access only from specific IP address.

## Testing

```bash
# Run unit tests
task backend:test

# Run ACL-specific tests
go test ./pkg/acl/...

# Run benchmarks
go test -bench=. ./pkg/acl/...
```

## Troubleshooting

### ACLs not being enforced

1. Check if ACLs are enabled: `acl.enabled: true` in config
2. Verify ACL file exists: `<data-dir>/acls.json`
3. Check logs for ACL load errors

### Authorization denied unexpectedly

1. List all ACLs: `GET /api/acls`
2. Check for DENY rules (they take precedence)
3. Verify principal and host match exactly
4. Check pattern type (Literal vs Prefixed)

### Performance concerns

1. Run benchmarks: `go test -bench=BenchmarkAuthorizer ./pkg/acl/...`
2. Check ACL count: `GET /api/acls`
3. Consider consolidating ACLs using prefix patterns
4. If needed, disable ACLs: `acl.enabled: false`

## Security Considerations

1. **Default deny**: When ACLs are enabled, all operations require explicit ALLOW
2. **Deny precedence**: DENY rules override ALLOW rules
3. **Wildcard caution**: Use `*` for principal/host carefully
4. **Audit logging**: ACL changes are logged at INFO level
5. **Persistence**: ACLs persist across restarts via `acls.json`

## Future Enhancements

Potential future improvements:

1. **SASL integration**: Link principals to SASL authenticated users
2. **Group support**: Organize principals into groups
3. **Audit trail**: Detailed audit log of authorization decisions
4. **Dynamic reload**: Hot reload of ACL changes
5. **Raft replication**: Replicate ACLs across cluster nodes
