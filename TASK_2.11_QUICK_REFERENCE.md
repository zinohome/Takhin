# Task 2.11: Batch Operations API - Quick Reference

## API Endpoints

### Batch Create Topics
```
POST /api/topics/batch
Authorization: Bearer <api-key>
Content-Type: application/json
```

**Request:**
```json
{
  "topics": [
    {"name": "topic-1", "partitions": 5},
    {"name": "topic-2", "partitions": 3}
  ]
}
```

**Response (200 OK):**
```json
{
  "totalRequested": 2,
  "successful": 2,
  "failed": 0,
  "results": [
    {"resource": "topic-1", "success": true, "partitions": 5},
    {"resource": "topic-2", "success": true, "partitions": 3}
  ],
  "errors": []
}
```

### Batch Delete Topics
```
DELETE /api/topics/batch
Authorization: Bearer <api-key>
Content-Type: application/json
```

**Request:**
```json
{
  "topics": ["topic-1", "topic-2"]
}
```

**Response (200 OK):**
```json
{
  "totalRequested": 2,
  "successful": 2,
  "failed": 0,
  "results": [
    {"resource": "topic-1", "success": true},
    {"resource": "topic-2", "success": true}
  ],
  "errors": []
}
```

### Batch Update Configs
```
PUT /api/configs/topics
Authorization: Bearer <api-key>
Content-Type: application/json
```

**Request:**
```json
{
  "topics": ["topic-1", "topic-2"],
  "config": {
    "retentionMs": 86400000,
    "compressionType": "lz4",
    "cleanupPolicy": "delete"
  }
}
```

## Transaction Behavior

### Batch Create
- ✅ **All-or-nothing:** If any topic creation fails, ALL created topics are deleted
- ✅ **Fail-fast:** Validates all inputs before creating any topics
- ✅ **Rollback:** Automatic cleanup on failure

### Batch Delete
- ✅ **Abort-on-missing:** If any topic doesn't exist, NO topics are deleted
- ✅ **Pre-validation:** Checks all topics exist before deleting any

### Batch Config Update
- ✅ **Abort-on-missing:** If any topic doesn't exist, NO configs are updated
- ✅ **Atomic:** All updates succeed or none do

## Error Codes

| Status | Condition | Behavior |
|--------|-----------|----------|
| 200 | All operations successful | Full success |
| 400 | Validation error | No operations executed |
| 400 | Topic exists (create) | Rollback all created topics |
| 400 | Topic missing (delete) | No deletions performed |
| 400 | Topic missing (config) | No configs updated |

## Validation Rules

### Batch Create
- ❌ Empty topic name
- ❌ Partitions ≤ 0
- ❌ Duplicate names in request
- ❌ Empty request array
- ❌ Topic already exists

### Batch Delete
- ❌ Empty topic name
- ❌ Duplicate names in request
- ❌ Empty request array
- ❌ Topic doesn't exist

## cURL Examples

**Create:**
```bash
curl -X POST http://localhost:8080/api/topics/batch \
  -H "Authorization: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"topics":[{"name":"events","partitions":5},{"name":"logs","partitions":3}]}'
```

**Delete:**
```bash
curl -X DELETE http://localhost:8080/api/topics/batch \
  -H "Authorization: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"topics":["events","logs"]}'
```

**Update Configs:**
```bash
curl -X PUT http://localhost:8080/api/configs/topics \
  -H "Authorization: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"topics":["events","logs"],"config":{"retentionMs":86400000}}'
```

## Go Client Example

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

client := &http.Client{}

// Batch create
req := BatchCreateTopicsRequest{
    Topics: []CreateTopicRequest{
        {Name: "events", Partitions: 5},
        {Name: "logs", Partitions: 3},
    },
}

body, _ := json.Marshal(req)
httpReq, _ := http.NewRequest("POST", 
    "http://localhost:8080/api/topics/batch",
    bytes.NewReader(body))
httpReq.Header.Set("Authorization", "your-api-key")
httpReq.Header.Set("Content-Type", "application/json")

resp, err := client.Do(httpReq)

var result BatchOperationResult
json.NewDecoder(resp.Body).Decode(&result)
```

## Best Practices

1. **Batch Size:** Keep batches under 50 topics for optimal performance
2. **Idempotency:** Check existing topics before batch create
3. **Error Handling:** Always check `failed` count in response
4. **Retries:** Entire batch is safe to retry on failure
5. **Monitoring:** Watch for batch operation events via WebSocket

## WebSocket Events

Batch operations trigger these events:

**Topic Created:**
```json
{
  "type": "topic.created",
  "data": {
    "topic": "events",
    "partitions": 5
  }
}
```

**Topic Deleted:**
```json
{
  "type": "topic.deleted",
  "data": {
    "topic": "events"
  }
}
```

## Testing

Run batch operation tests:
```bash
cd backend
go test -v -run TestBatch ./pkg/console/...
```

## Files

- `backend/pkg/console/batch_handlers.go` - Implementation
- `backend/pkg/console/batch_handlers_test.go` - Tests
- `backend/pkg/console/server.go` - Route registration
- `backend/docs/swagger/*` - OpenAPI documentation

## Swagger Documentation

Access interactive API docs:
```
http://localhost:8080/swagger/index.html
```

Navigate to **Topics** section to see batch endpoints.
