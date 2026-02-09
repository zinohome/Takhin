# HTTP Producer API - Quick Reference

## Endpoints

### Produce Messages (Batch)
```
POST /api/topics/{topic}/produce
```

**Query Parameters:**
- `key.format`: `json|string|binary|avro` (default: `json`)
- `value.format`: `json|string|binary|avro` (default: `json`)
- `key.schema`: Avro schema subject (required if `key.format=avro`)
- `value.schema`: Avro schema subject (required if `value.format=avro`)
- `compression`: `none|gzip|snappy|lz4|zstd` (default: `none`)
- `async`: `true|false` (default: `false`)

**Request Body:**
```json
{
  "records": [
    {
      "key": <any>,           // optional
      "value": <any>,         // required
      "partition": <int32>,   // optional
      "headers": [            // optional
        {"key": "string", "value": "string"}
      ]
    }
  ]
}
```

**Response (Sync - 200 OK):**
```json
{
  "offsets": [
    {
      "partition": 0,
      "offset": 42,
      "timestamp": 1704545600123,
      "error": "..."  // present if record failed
    }
  ]
}
```

**Response (Async - 202 Accepted):**
```json
{
  "requestId": "req_1704545600123456789",
  "status": "pending"
}
```

### Get Async Status
```
GET /api/produce/status/{requestId}
```

**Response (200 OK):**
```json
{
  "requestId": "req_1704545600123456789",
  "status": "pending|completed|failed",
  "offsets": [...],  // when completed
  "error": "..."     // when failed
}
```

## Quick Examples

### Basic JSON Produce
```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{"records":[{"key":"k1","value":"v1"}]}'
```

### Batch Produce
```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -d '{
    "records": [
      {"key":"k1", "value":"v1"},
      {"key":"k2", "value":"v2"},
      {"key":"k3", "value":"v3"}
    ]
  }'
```

### With Compression
```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?compression=snappy" \
  -H "Content-Type: application/json" \
  -d '{"records":[{"value":"data"}]}'
```

### Async Produce
```bash
# Submit
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?async=true" \
  -H "Content-Type: application/json" \
  -d '{"records":[...]}'

# Poll status
curl http://localhost:8080/api/produce/status/req_123456
```

### String Format
```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?value.format=string" \
  -H "Content-Type: application/json" \
  -d '{"records":[{"value":"plain text"}]}'
```

### Binary Format
```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?value.format=binary" \
  -H "Content-Type: application/json" \
  -d '{"records":[{"value":"aGVsbG8="}]}'  # base64
```

### With Headers
```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -d '{
    "records": [{
      "value": "data",
      "headers": [
        {"key": "source", "value": "api"},
        {"key": "version", "value": "1.0"}
      ]
    }]
  }'
```

### Specific Partition
```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -d '{
    "records": [
      {"partition": 0, "value": "goes to partition 0"},
      {"partition": 1, "value": "goes to partition 1"}
    ]
  }'
```

## Data Formats

| Format   | Use Case                  | Example                     |
|----------|---------------------------|----------------------------|
| `json`   | Objects, structured data  | `{"key": "value"}`         |
| `string` | Plain text                | `"hello world"`            |
| `binary` | Raw bytes (base64)        | `"aGVsbG8="`               |
| `avro`   | Schema-validated data     | Requires schema registry   |

## Compression Types

| Codec    | Speed  | Ratio | Best For                    |
|----------|--------|-------|-----------------------------|
| `none`   | -      | 1.0x  | Small messages, low latency |
| `snappy` | Fast   | 2.0x  | Real-time applications      |
| `lz4`    | Fast   | 2.5x  | Balanced use cases          |
| `gzip`   | Medium | 3.0x  | Standard compression        |
| `zstd`   | Slow   | 3.5x  | Large payloads, best ratio  |

## Error Codes

| Code | Meaning                    |
|------|----------------------------|
| 200  | Success (sync)             |
| 202  | Accepted (async)           |
| 400  | Bad request / validation   |
| 404  | Topic not found            |
| 500  | Internal server error      |

## Response Fields

### ProduceResponse
- `offsets`: Array of record metadata
  - `partition`: Partition ID
  - `offset`: Message offset
  - `timestamp`: Production timestamp (ms)
  - `error`: Error message (if failed)

### AsyncProduceResponse
- `requestId`: Unique request identifier
- `status`: Request status (pending)

### ProduceStatusResponse
- `requestId`: Request identifier
- `status`: Current status (pending/completed/failed)
- `offsets`: Record metadata (when completed)
- `error`: Error message (when failed)

## Performance Tips

1. **Batch Size**: 100-1000 records per request for best throughput
2. **Compression**: Use `snappy` for latency-sensitive, `zstd` for throughput
3. **Async Mode**: Enable for batches > 1000 records
4. **Connection Reuse**: Use HTTP keep-alive
5. **Partition Control**: Distribute across partitions for parallelism

## Authentication

All endpoints require authentication via `Authorization` header:
```bash
curl -H "Authorization: your-api-key" ...
# or
curl -H "Authorization: Bearer your-api-key" ...
```

## Client Libraries

### Python
```python
import requests

response = requests.post(
    'http://localhost:8080/api/topics/my-topic/produce',
    headers={'Authorization': 'your-api-key'},
    json={'records': [{'key': 'k1', 'value': 'v1'}]}
)
print(response.json())
```

### JavaScript
```javascript
const response = await fetch(
  'http://localhost:8080/api/topics/my-topic/produce',
  {
    method: 'POST',
    headers: {
      'Authorization': 'your-api-key',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      records: [{key: 'k1', value: 'v1'}]
    })
  }
);
const result = await response.json();
```

### Go
```go
req := ProduceRequest{
    Records: []ProducerRecord{{Key: "k1", Value: "v1"}},
}
body, _ := json.Marshal(req)
resp, _ := http.Post(
    "http://localhost:8080/api/topics/my-topic/produce",
    "application/json",
    bytes.NewBuffer(body),
)
```

## Limits & Constraints

- **Max Request Size**: Determined by topic configuration
- **Async Request TTL**: 30 minutes
- **Cleanup Interval**: Every 5 minutes
- **Supported Formats**: JSON, String, Binary, Avro (with schema registry)

## Monitoring

Track these metrics for producer health:
- Request rate (req/sec)
- Batch size distribution
- Compression ratio
- Latency percentiles (p50, p95, p99)
- Error rate

## See Also

- [Detailed Examples](PRODUCER_API_EXAMPLE.md)
- [HTTP Consumer API](TASK_6.4_HTTP_CONSUMER_QUICK_REFERENCE.md)
- [Schema Registry](../backend/pkg/schema/README.md)
