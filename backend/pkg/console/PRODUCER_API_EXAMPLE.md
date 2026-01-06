# HTTP Proxy Producer API - Usage Examples

This document provides examples of using the REST Producer API to send messages to Takhin topics.

## Table of Contents
- [Basic Usage](#basic-usage)
- [Data Formats](#data-formats)
- [Batch Operations](#batch-operations)
- [Compression](#compression)
- [Async Production](#async-production)
- [Advanced Features](#advanced-features)

## Basic Usage

### Produce a Single Message (JSON)

```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "key": "user-123",
        "value": {
          "userId": "123",
          "event": "login",
          "timestamp": 1704545600
        }
      }
    ]
  }'
```

**Response:**
```json
{
  "offsets": [
    {
      "partition": 0,
      "offset": 42,
      "timestamp": 1704545600123
    }
  ]
}
```

### Produce to Specific Partition

```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "partition": 2,
        "key": "partition-key",
        "value": {"data": "goes to partition 2"}
      }
    ]
  }'
```

## Data Formats

### JSON Format (Default)

JSON is the default format for both keys and values:

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?key.format=json&value.format=json" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "key": {"userId": "123"},
        "value": {"eventType": "purchase", "amount": 99.99}
      }
    ]
  }'
```

### String Format

Send plain strings without JSON wrapping:

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?key.format=string&value.format=string" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "key": "simple-key",
        "value": "simple-value"
      }
    ]
  }'
```

### Binary Format

Send base64-encoded binary data:

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?value.format=binary" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "key": "binary-key",
        "value": "aGVsbG8gd29ybGQ="
      }
    ]
  }'
```

### Avro Format (with Schema Registry)

> **Note:** Requires schema registry to be enabled and schemas registered

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?key.format=avro&value.format=avro&key.schema=user-key&value.schema=user-event" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "key": {"userId": "123"},
        "value": {
          "eventId": "evt-456",
          "eventType": "purchase",
          "timestamp": 1704545600
        }
      }
    ]
  }'
```

## Batch Operations

### Produce Multiple Messages

```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "key": "key1",
        "value": {"event": "event1"}
      },
      {
        "key": "key2",
        "value": {"event": "event2"}
      },
      {
        "key": "key3",
        "value": {"event": "event3"}
      }
    ]
  }'
```

**Response:**
```json
{
  "offsets": [
    {"partition": 0, "offset": 100, "timestamp": 1704545600123},
    {"partition": 0, "offset": 101, "timestamp": 1704545600124},
    {"partition": 0, "offset": 102, "timestamp": 1704545600125}
  ]
}
```

### Distribute Across Partitions

```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "partition": 0,
        "value": {"data": "partition-0"}
      },
      {
        "partition": 1,
        "value": {"data": "partition-1"}
      },
      {
        "partition": 2,
        "value": {"data": "partition-2"}
      }
    ]
  }'
```

## Compression

### GZIP Compression

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?compression=gzip" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "value": {"largeData": "...very large payload..."}
      }
    ]
  }'
```

### Snappy Compression (Fast)

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?compression=snappy" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "value": "data to compress"
      }
    ]
  }'
```

### LZ4 Compression

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?compression=lz4" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "value": {"data": "compressed with lz4"}
      }
    ]
  }'
```

### ZSTD Compression (Best Ratio)

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?compression=zstd" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "value": {"data": "compressed with zstd"}
      }
    ]
  }'
```

## Async Production

### Submit Async Request

For large batches, use async mode to return immediately:

```bash
curl -X POST "http://localhost:8080/api/topics/my-topic/produce?async=true" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {"value": "message1"},
      {"value": "message2"},
      {"value": "message3"}
    ]
  }'
```

**Response (202 Accepted):**
```json
{
  "requestId": "req_1704545600123456789",
  "status": "pending"
}
```

### Check Async Request Status

```bash
curl -X GET http://localhost:8080/api/produce/status/req_1704545600123456789 \
  -H "Authorization: your-api-key"
```

**Response (pending):**
```json
{
  "requestId": "req_1704545600123456789",
  "status": "pending"
}
```

**Response (completed):**
```json
{
  "requestId": "req_1704545600123456789",
  "status": "completed",
  "offsets": [
    {"partition": 0, "offset": 100, "timestamp": 1704545600123},
    {"partition": 0, "offset": 101, "timestamp": 1704545600124},
    {"partition": 0, "offset": 102, "timestamp": 1704545600125}
  ]
}
```

**Response (failed):**
```json
{
  "requestId": "req_1704545600123456789",
  "status": "failed",
  "error": "topic not found"
}
```

## Advanced Features

### Messages with Headers

```bash
curl -X POST http://localhost:8080/api/topics/my-topic/produce \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "key": "user-123",
        "value": {"event": "login"},
        "headers": [
          {"key": "source", "value": "web-app"},
          {"key": "version", "value": "1.0"},
          {"key": "trace-id", "value": "abc-123"}
        ]
      }
    ]
  }'
```

### Complex Example: Batch with Mixed Formats

```bash
curl -X POST "http://localhost:8080/api/topics/events/produce?compression=snappy&async=true" \
  -H "Content-Type: application/json" \
  -H "Authorization: your-api-key" \
  -d '{
    "records": [
      {
        "partition": 0,
        "key": "user-1",
        "value": {"eventType": "login", "timestamp": 1704545600},
        "headers": [
          {"key": "source", "value": "mobile-app"}
        ]
      },
      {
        "partition": 1,
        "key": "user-2",
        "value": {"eventType": "purchase", "amount": 99.99},
        "headers": [
          {"key": "source", "value": "web-app"}
        ]
      }
    ]
  }'
```

## Error Handling

### Topic Not Found
```json
{
  "error": "topic not found"
}
```
HTTP Status: 404

### Invalid Request
```json
{
  "error": "no records to produce"
}
```
HTTP Status: 400

### Serialization Error
```json
{
  "offsets": [
    {
      "error": "record 0: serialize value: invalid JSON"
    }
  ]
}
```
HTTP Status: 200 (with partial success)

## Client Libraries

### Python Example

```python
import requests
import json

url = "http://localhost:8080/api/topics/my-topic/produce"
headers = {
    "Content-Type": "application/json",
    "Authorization": "your-api-key"
}
data = {
    "records": [
        {
            "key": "user-123",
            "value": {"event": "login", "timestamp": 1704545600}
        }
    ]
}

response = requests.post(url, headers=headers, json=data)
print(response.json())
```

### JavaScript Example

```javascript
const response = await fetch('http://localhost:8080/api/topics/my-topic/produce', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'your-api-key'
  },
  body: JSON.stringify({
    records: [
      {
        key: 'user-123',
        value: { event: 'login', timestamp: 1704545600 }
      }
    ]
  })
});

const result = await response.json();
console.log(result);
```

### Go Example

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ProduceRequest struct {
	Records []ProducerRecord `json:"records"`
}

type ProducerRecord struct {
	Key   string      `json:"key,omitempty"`
	Value interface{} `json:"value"`
}

func main() {
	req := ProduceRequest{
		Records: []ProducerRecord{
			{
				Key:   "user-123",
				Value: map[string]interface{}{"event": "login"},
			},
		},
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "http://localhost:8080/api/topics/my-topic/produce", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "your-api-key")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println(result)
}
```

## Performance Tips

1. **Use Batch Produce**: Send multiple records in one request for better throughput
2. **Choose Compression Wisely**:
   - `snappy`: Best for latency-sensitive applications
   - `lz4`: Good balance of speed and compression
   - `zstd`: Best compression ratio for large payloads
   - `gzip`: Standard compression, widely supported
3. **Use Async for Large Batches**: Enable async mode for batches > 1000 records
4. **Partition Distribution**: Manually specify partitions for even distribution
5. **Connection Pooling**: Reuse HTTP connections in client applications

## Comparison with Kafka REST Proxy

| Feature | Takhin HTTP Producer | Kafka REST Proxy |
|---------|---------------------|------------------|
| Batch Produce | ✅ | ✅ |
| JSON Format | ✅ | ✅ |
| Avro Format | ✅ | ✅ |
| Binary Format | ✅ | ✅ |
| Compression | ✅ (gzip, snappy, lz4, zstd) | ✅ (gzip, snappy, lz4) |
| Async Mode | ✅ | ❌ |
| Custom Headers | ✅ | ✅ |
| Schema Registry | ✅ (when enabled) | ✅ |

## See Also

- [HTTP Consumer API](TASK_6.4_HTTP_CONSUMER_QUICK_REFERENCE.md)
- [Schema Registry](../backend/pkg/schema/README.md)
- [Compression](../backend/pkg/compression/README.md)
