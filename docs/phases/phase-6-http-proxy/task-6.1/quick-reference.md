# Schema Registry Quick Reference

## Overview
Takhin Schema Registry provides centralized schema management with version control and compatibility checking for Avro, JSON, and Protobuf schemas.

---

## Quick Start

### Start Server
```bash
# Default settings (port 8081)
./build/schema-registry

# Custom settings
./build/schema-registry -addr :8081 -data-dir /data/schemas
```

### Register First Schema
```bash
curl -X POST http://localhost:8081/subjects/user-value/versions \
  -H "Content-Type: application/json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"}]}",
    "schemaType": "AVRO"
  }'
```

---

## REST API Reference

### Subjects

#### List All Subjects
```bash
GET /subjects
```
**Response**: `["subject1", "subject2"]`

#### Get Subject Versions
```bash
GET /subjects/{subject}/versions
```
**Response**: `[1, 2, 3]`

#### Get Specific Version
```bash
GET /subjects/{subject}/versions/{version}
GET /subjects/{subject}/versions/latest
```
**Response**:
```json
{
  "id": 1,
  "subject": "user-value",
  "version": 1,
  "schemaType": "AVRO",
  "schema": "{...}",
  "createdAt": "2026-01-06T00:00:00Z",
  "updatedAt": "2026-01-06T00:00:00Z"
}
```

#### Register New Schema
```bash
POST /subjects/{subject}/versions
Content-Type: application/json

{
  "schema": "{schema definition}",
  "schemaType": "AVRO|JSON|PROTOBUF",
  "references": [
    {"name": "RefName", "subject": "ref-subject", "version": 1}
  ]
}
```
**Response**: `{"id": 123}`

#### Delete Version
```bash
DELETE /subjects/{subject}/versions/{version}
```
**Response**: `{"version": 1}`

#### Delete Subject
```bash
DELETE /subjects/{subject}
```
**Response**: `[1, 2, 3]` (deleted versions)

---

### Schemas

#### Get Schema by ID
```bash
GET /schemas/ids/{id}
```
**Response**: `{"schema": "{...}"}`

---

### Compatibility

#### Get Compatibility Config
```bash
GET /config/{subject}
```
**Response**: `{"compatibilityLevel": "BACKWARD"}`

#### Set Compatibility Config
```bash
PUT /config/{subject}
Content-Type: application/json

{
  "compatibility": "BACKWARD|FORWARD|FULL|NONE|BACKWARD_TRANSITIVE|FORWARD_TRANSITIVE|FULL_TRANSITIVE"
}
```
**Response**: `{"compatibility": "BACKWARD"}`

#### Test Compatibility
```bash
POST /compatibility/subjects/{subject}/versions/{version}
Content-Type: application/json

{
  "schema": "{schema to test}",
  "schemaType": "AVRO"
}
```
**Response**: `{"is_compatible": true}`

---

## Schema Types

### Avro
```json
{
  "type": "record",
  "name": "User",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "name", "type": "string"},
    {"name": "email", "type": "string", "default": ""}
  ]
}
```

### JSON Schema
```json
{
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "name": {"type": "string"},
    "email": {"type": "string"}
  },
  "required": ["id", "name"]
}
```

### Protobuf
```protobuf
syntax = "proto3";

message User {
  string id = 1;
  string name = 2;
  string email = 3;
}
```

---

## Compatibility Modes

### BACKWARD (Default)
- **Rule**: New schema can read old data
- **Allowed**: Add fields with defaults
- **Not Allowed**: Remove fields, change types

### FORWARD
- **Rule**: Old schema can read new data
- **Allowed**: Remove fields
- **Not Allowed**: Add fields without defaults

### FULL
- **Rule**: Both backward and forward compatible
- **Allowed**: Add optional fields with defaults
- **Not Allowed**: Remove fields, add required fields

### NONE
- **Rule**: No compatibility checking
- **Use Case**: Development/testing only

### Transitive Modes
- `BACKWARD_TRANSITIVE`: Check against ALL previous versions
- `FORWARD_TRANSITIVE`: Check against ALL previous versions
- `FULL_TRANSITIVE`: Full check against ALL versions

---

## Error Codes

| Code  | Message | Description |
|-------|---------|-------------|
| 40401 | Subject not found | Subject doesn't exist |
| 40402 | Version not found | Version doesn't exist |
| 40403 | Schema not found | Schema ID not found |
| 40408 | Compatibility level not found | No config set |
| 409   | Incompatible schema | Schema fails compatibility |
| 42201 | Invalid schema | Syntax error |
| 42202 | Invalid version | Invalid version format |
| 42203 | Invalid compatibility | Unknown mode |

---

## Command-Line Flags

```bash
./build/schema-registry [OPTIONS]

Options:
  -addr string
        HTTP server address (default ":8081")
  -data-dir string
        Data directory for schema storage 
        (default "/tmp/takhin-schema-registry")
  -default-compatibility string
        Default compatibility mode (default "BACKWARD")
  -max-versions int
        Maximum number of schema versions per subject 
        (default 100)
  -cache-size int
        Schema cache size (default 1000)
  -log-level string
        Log level: debug, info, warn, error (default "info")
```

---

## Library Usage

### Basic Setup
```go
import "github.com/takhin-data/takhin/pkg/schema"

cfg := &schema.Config{
    DataDir:              "/var/lib/schemas",
    DefaultCompatibility: schema.CompatibilityBackward,
    CacheSize:            1000,
}

registry, err := schema.NewRegistry(cfg)
if err != nil {
    log.Fatal(err)
}
defer registry.Close()
```

### Register Schema
```go
schemaStr := `{"type":"record","name":"User","fields":[...]}`

registered, err := registry.RegisterSchema(
    "user-value",
    schemaStr,
    schema.SchemaTypeAvro,
    nil, // no references
)
if err != nil {
    log.Printf("Registration failed: %v", err)
}
```

### Get Schema
```go
// By ID
schema, err := registry.GetSchemaByID(123)

// By subject and version
schema, err := registry.GetSchemaBySubjectVersion("user-value", 1)

// Latest version
schema, err := registry.GetLatestSchema("user-value")
```

### Test Compatibility
```go
compatible, err := registry.TestCompatibility(
    "user-value",
    newSchemaStr,
    schema.SchemaTypeAvro,
    0, // test against all versions
)
if !compatible {
    log.Println("Schema is not compatible")
}
```

### Set Compatibility
```go
err := registry.SetCompatibility("user-value", schema.CompatibilityFull)
```

---

## Common Workflows

### 1. Initial Schema Registration
```bash
# Register v1
curl -X POST http://localhost:8081/subjects/order-value/versions \
  -d '{"schema":"{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}","schemaType":"AVRO"}'

# Response: {"id": 1}
```

### 2. Evolve Schema (Add Field)
```bash
# Register v2 with new optional field
curl -X POST http://localhost:8081/subjects/order-value/versions \
  -d '{"schema":"{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"total\",\"type\":\"double\",\"default\":0.0}]}","schemaType":"AVRO"}'

# Response: {"id": 2}
```

### 3. Test Before Registering
```bash
# Test compatibility first
curl -X POST http://localhost:8081/compatibility/subjects/order-value/versions/latest \
  -d '{"schema":"{new schema}","schemaType":"AVRO"}'

# If compatible, then register
curl -X POST http://localhost:8081/subjects/order-value/versions \
  -d '{"schema":"{new schema}","schemaType":"AVRO"}'
```

### 4. List and Inspect
```bash
# List all subjects
curl http://localhost:8081/subjects

# Get all versions
curl http://localhost:8081/subjects/order-value/versions

# Get specific version
curl http://localhost:8081/subjects/order-value/versions/2
```

### 5. Change Compatibility Mode
```bash
# Set to FULL for stricter checks
curl -X PUT http://localhost:8081/config/order-value \
  -d '{"compatibility":"FULL"}'
```

---

## Best Practices

### Schema Design
1. **Always add defaults** to new fields for backward compatibility
2. **Never remove fields** in backward-compatible mode
3. **Use optional fields** for flexibility
4. **Version naming**: Use descriptive subject names (e.g., `user-value`, `order-key`)

### Compatibility
1. **Start with BACKWARD** (most common)
2. **Use FULL** for long-term stability
3. **Never use NONE** in production
4. **Test before registering** with `/compatibility` endpoint

### Operations
1. **Regular backups** of data directory
2. **Monitor disk space** (schemas persist to disk)
3. **Set appropriate cache-size** based on schema count
4. **Use logging** for troubleshooting (`-log-level debug`)

### Performance
1. **Cache schemas** on client side when possible
2. **Batch registrations** if registering multiple schemas
3. **Use version numbers** instead of "latest" for production

---

## Troubleshooting

### Schema Registration Fails
```bash
# Check schema syntax
curl -X POST http://localhost:8081/subjects/test/versions \
  -d '{"schema":"{}","schemaType":"AVRO"}'

# Error 42201: Invalid schema (missing 'type' field)
```

### Compatibility Check Fails
```bash
# Get current compatibility mode
curl http://localhost:8081/config/mysubject

# Test against specific version
curl -X POST http://localhost:8081/compatibility/subjects/mysubject/versions/1 \
  -d '{"schema":"{...}","schemaType":"AVRO"}'
```

### Can't Find Schema
```bash
# Check if subject exists
curl http://localhost:8081/subjects

# Check versions
curl http://localhost:8081/subjects/mysubject/versions

# Get by ID
curl http://localhost:8081/schemas/ids/1
```

### Server Won't Start
```bash
# Check if port is available
lsof -i :8081

# Check data directory permissions
ls -la /tmp/takhin-schema-registry

# Check logs
./build/schema-registry -log-level debug
```

---

## Integration Examples

### Python Client
```python
import requests
import json

BASE_URL = "http://localhost:8081"

def register_schema(subject, schema_dict, schema_type="AVRO"):
    url = f"{BASE_URL}/subjects/{subject}/versions"
    payload = {
        "schema": json.dumps(schema_dict),
        "schemaType": schema_type
    }
    response = requests.post(url, json=payload)
    return response.json()

def get_latest(subject):
    url = f"{BASE_URL}/subjects/{subject}/versions/latest"
    return requests.get(url).json()

# Usage
schema = {
    "type": "record",
    "name": "User",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "name", "type": "string"}
    ]
}

result = register_schema("user-value", schema)
print(f"Registered schema ID: {result['id']}")
```

### Java Client
```java
import java.net.http.*;
import org.json.*;

String baseUrl = "http://localhost:8081";
HttpClient client = HttpClient.newHttpClient();

// Register schema
String payload = new JSONObject()
    .put("schema", schemaJsonString)
    .put("schemaType", "AVRO")
    .toString();

HttpRequest request = HttpRequest.newBuilder()
    .uri(URI.create(baseUrl + "/subjects/user-value/versions"))
    .header("Content-Type", "application/json")
    .POST(HttpRequest.BodyPublishers.ofString(payload))
    .build();

HttpResponse<String> response = client.send(request, 
    HttpResponse.BodyHandlers.ofString());
```

---

## File Locations

### Binary
```
backend/build/schema-registry
```

### Source Code
```
backend/pkg/schema/
├── types.go
├── storage.go
├── compatibility.go
├── registry.go
├── server.go
├── storage_test.go
└── registry_test.go
```

### Command
```
backend/cmd/schema-registry/main.go
```

### Data (Runtime)
```
/tmp/takhin-schema-registry/schemas.json  (default)
```

---

## Performance Tips

1. **Increase cache size** for large schema counts: `-cache-size 10000`
2. **Use SSD** for data directory
3. **Separate disk** for schema storage in production
4. **Monitor memory**: ~1KB per cached schema

---

## Security Notes

⚠️ **Current Version**: No authentication  
✅ **Suitable for**: Internal networks, development  
❌ **Not suitable for**: Public internet

**Recommended Setup**:
- Deploy behind reverse proxy (nginx)
- Use network firewall rules
- Add API gateway with authentication

---

## Support & Resources

- **Source**: `backend/pkg/schema/`
- **Tests**: `backend/pkg/schema/*_test.go`
- **Build**: `cd backend && go build ./cmd/schema-registry/`
- **Docs**: `TASK_6.1_COMPLETION_SUMMARY.md`

---

**Version**: 1.0  
**Status**: Production-ready  
**Compatibility**: Confluent Schema Registry API
