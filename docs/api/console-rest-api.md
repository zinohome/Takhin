# Console REST API Documentation

## Overview

Takhin Console provides a RESTful HTTP API for managing topics, messages, and consumer groups. This API powers the Web UI and can be used directly by applications or scripts.

**Base URL**: `http://localhost:8080/api`  
**Format**: JSON  
**Authentication**: API Key (optional)  
**CORS**: Enabled for localhost origins

## Authentication

API Key authentication can be optionally enabled when starting the Console server.

### Enabling Authentication

```bash
./console \
  -data-dir=/tmp/takhin-data \
  -api-addr=:8080 \
  -enable-auth \
  -api-keys="key1,key2,key3"
```

### Using API Keys

Include the API key in the `Authorization` header:

```bash
# Direct key format
curl -H "Authorization: your-api-key" http://localhost:8080/api/topics

# Bearer token format
curl -H "Authorization: Bearer your-api-key" http://localhost:8080/api/topics
```

### Public Endpoints (No Auth Required)

- `GET /api/health` - Health check
- `GET /api/health/ready` - Readiness probe
- `GET /api/health/live` - Liveness probe
- `GET /swagger/*` - API documentation

## Health Endpoints

### Health Check

Get basic health status.

**Endpoint:** `GET /api/health`

**Response:** `200 OK`
```json
{
  "status": "healthy"
}
```

**Example:**
```bash
curl http://localhost:8080/api/health
```

### Readiness Probe

Check if the service is ready to accept requests.

**Endpoint:** `GET /api/health/ready`

**Response:** `200 OK` (ready) or `503 Service Unavailable` (not ready)
```json
{
  "status": "ready",
  "version": "1.0.0",
  "components": {
    "topic_manager": "ok",
    "coordinator": "ok"
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/health/ready
```

### Liveness Probe

Check if the service is alive.

**Endpoint:** `GET /api/health/live`

**Response:** `200 OK`
```json
{
  "status": "alive"
}
```

## Topic Management

### List Topics

Get all topics.

**Endpoint:** `GET /api/topics`

**Response:** `200 OK`
```json
[
  {
    "name": "my-topic",
    "partitionCount": 3,
    "partitions": [
      {
        "id": 0,
        "highWaterMark": 1500
      },
      {
        "id": 1,
        "highWaterMark": 1502
      },
      {
        "id": 2,
        "highWaterMark": 1498
      }
    ]
  }
]
```

**Example:**
```bash
curl http://localhost:8080/api/topics
```

### Get Topic Details

Get detailed information about a specific topic.

**Endpoint:** `GET /api/topics/{topic}`

**Path Parameters:**
- `topic` (string, required): Topic name

**Response:** `200 OK`
```json
{
  "name": "my-topic",
  "partitionCount": 3,
  "partitions": [
    {
      "id": 0,
      "highWaterMark": 1500
    },
    {
      "id": 1,
      "highWaterMark": 1502
    },
    {
      "id": 2,
      "highWaterMark": 1498
    }
  ]
}
```

**Error Responses:**
- `404 Not Found`: Topic doesn't exist
```json
{
  "error": "topic not found: unknown-topic"
}
```

**Example:**
```bash
curl http://localhost:8080/api/topics/my-topic
```

### Create Topic

Create a new topic.

**Endpoint:** `POST /api/topics`

**Request Body:**
```json
{
  "name": "new-topic",
  "partitions": 3
}
```

**Request Fields:**
- `name` (string, required): Topic name (must not be empty)
- `partitions` (integer, required): Number of partitions (must be > 0)

**Response:** `201 Created`
```json
{
  "name": "new-topic",
  "partitions": "3"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid parameters
```json
{
  "error": "topic name cannot be empty"
}
```
- `500 Internal Server Error`: Creation failed
```json
{
  "error": "failed to create topic: ..."
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/topics \
  -H "Content-Type: application/json" \
  -d '{
    "name": "events",
    "partitions": 5
  }'
```

### Delete Topic

Delete an existing topic.

**Endpoint:** `DELETE /api/topics/{topic}`

**Path Parameters:**
- `topic` (string, required): Topic name

**Response:** `204 No Content`

**Error Responses:**
- `500 Internal Server Error`: Deletion failed
```json
{
  "error": "failed to delete topic: ..."
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/topics/old-topic
```

## Message Operations

### Get Messages

Read messages from a topic partition.

**Endpoint:** `GET /api/topics/{topic}/messages`

**Path Parameters:**
- `topic` (string, required): Topic name

**Query Parameters:**
- `partition` (integer, required): Partition ID (>= 0)
- `offset` (integer, required): Starting offset (>= 0)
- `limit` (integer, optional): Maximum number of messages to return (default: 100, must be > 0)

**Response:** `200 OK`
```json
[
  {
    "partition": 0,
    "offset": 0,
    "key": "user-123",
    "value": "{\"action\":\"login\",\"timestamp\":1704067200}",
    "timestamp": 1704067200000
  },
  {
    "partition": 0,
    "offset": 1,
    "key": "user-456",
    "value": "{\"action\":\"purchase\",\"amount\":99.99}",
    "timestamp": 1704067201000
  }
]
```

**Response Fields:**
- `partition` (integer): Partition ID
- `offset` (integer): Message offset
- `key` (string): Message key (may be empty)
- `value` (string): Message value
- `timestamp` (integer): Unix timestamp in milliseconds

**Error Responses:**
- `400 Bad Request`: Invalid parameters
```json
{
  "error": "partition must be >= 0"
}
```
- `404 Not Found`: Topic doesn't exist
```json
{
  "error": "topic not found: unknown-topic"
}
```
- `500 Internal Server Error`: Read failed
```json
{
  "error": "failed to read messages: ..."
}
```

**Example:**
```bash
# Read 10 messages starting from offset 0
curl "http://localhost:8080/api/topics/events/messages?partition=0&offset=0&limit=10"

# Read from offset 1000 with default limit
curl "http://localhost:8080/api/topics/events/messages?partition=1&offset=1000"
```

### Produce Message

Write a message to a topic partition.

**Endpoint:** `POST /api/topics/{topic}/messages`

**Path Parameters:**
- `topic` (string, required): Topic name

**Request Body:**
```json
{
  "partition": 0,
  "key": "user-123",
  "value": "{\"action\":\"login\"}"
}
```

**Request Fields:**
- `partition` (integer, required): Partition ID (>= 0)
- `key` (string, optional): Message key (can be empty)
- `value` (string, required): Message value

**Response:** `201 Created`
```json
{
  "offset": 1500,
  "partition": 0
}
```

**Response Fields:**
- `offset` (integer): Assigned offset
- `partition` (integer): Partition ID

**Error Responses:**
- `400 Bad Request`: Invalid parameters
```json
{
  "error": "partition must be >= 0"
}
```
- `404 Not Found`: Topic doesn't exist
```json
{
  "error": "topic not found: unknown-topic"
}
```
- `500 Internal Server Error`: Write failed
```json
{
  "error": "failed to append message: ..."
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/topics/events/messages \
  -H "Content-Type: application/json" \
  -d '{
    "partition": 0,
    "key": "order-789",
    "value": "{\"product\":\"laptop\",\"price\":1299.99}"
  }'
```

## Consumer Group Management

### List Consumer Groups

Get all consumer groups.

**Endpoint:** `GET /api/consumer-groups`

**Response:** `200 OK`
```json
[
  {
    "groupId": "analytics-service",
    "state": "Stable",
    "members": 3
  },
  {
    "groupId": "notification-workers",
    "state": "PreparingRebalance",
    "members": 5
  }
]
```

**Response Fields:**
- `groupId` (string): Consumer group ID
- `state` (string): Group state (Empty, PreparingRebalance, CompletingRebalance, Stable, Dead)
- `members` (integer): Number of active members

**Example:**
```bash
curl http://localhost:8080/api/consumer-groups
```

### Get Consumer Group Details

Get detailed information about a specific consumer group.

**Endpoint:** `GET /api/consumer-groups/{group}`

**Path Parameters:**
- `group` (string, required): Consumer group ID

**Response:** `200 OK`
```json
{
  "groupId": "analytics-service",
  "state": "Stable",
  "protocolType": "consumer",
  "protocol": "range",
  "members": [
    {
      "memberId": "consumer-1-a1b2c3d4",
      "clientId": "analytics-worker-1",
      "clientHost": "192.168.1.10",
      "partitions": [
        {"topic": "events", "partition": 0},
        {"topic": "events", "partition": 1}
      ]
    },
    {
      "memberId": "consumer-2-e5f6g7h8",
      "clientId": "analytics-worker-2",
      "clientHost": "192.168.1.11",
      "partitions": [
        {"topic": "events", "partition": 2}
      ]
    }
  ],
  "offsetCommits": [
    {
      "topic": "events",
      "partition": 0,
      "offset": 1234,
      "metadata": "committed by consumer-1"
    },
    {
      "topic": "events",
      "partition": 1,
      "offset": 1245,
      "metadata": "committed by consumer-1"
    },
    {
      "topic": "events",
      "partition": 2,
      "offset": 1230,
      "metadata": "committed by consumer-2"
    }
  ]
}
```

**Response Fields:**
- `groupId` (string): Consumer group ID
- `state` (string): Group state
- `protocolType` (string): Protocol type (typically "consumer")
- `protocol` (string): Partition assignment strategy (range, roundrobin, sticky)
- `members` (array): List of group members
  - `memberId` (string): Unique member ID
  - `clientId` (string): Client identifier
  - `clientHost` (string): Client IP address
  - `partitions` (array): Assigned partitions
- `offsetCommits` (array): Committed offsets
  - `topic` (string): Topic name
  - `partition` (integer): Partition ID
  - `offset` (integer): Committed offset
  - `metadata` (string): Optional commit metadata

**Error Responses:**
- `404 Not Found`: Consumer group doesn't exist
```json
{
  "error": "consumer group not found: unknown-group"
}
```

**Example:**
```bash
curl http://localhost:8080/api/consumer-groups/analytics-service
```

## Error Handling

All error responses follow a consistent format:

```json
{
  "error": "error message describing what went wrong"
}
```

### HTTP Status Codes

| Code | Meaning | When Used |
|------|---------|-----------|
| 200 | OK | Successful GET request |
| 201 | Created | Successful POST creating resource |
| 204 | No Content | Successful DELETE request |
| 400 | Bad Request | Invalid request parameters |
| 401 | Unauthorized | Missing or invalid API key |
| 404 | Not Found | Resource doesn't exist |
| 500 | Internal Server Error | Server-side error |
| 503 | Service Unavailable | Service not ready |

## Rate Limiting

Currently, no rate limiting is enforced. Future versions may implement rate limiting based on API keys.

## Pagination

Message retrieval supports pagination via the `offset` and `limit` query parameters:

```bash
# First page (offsets 0-99)
curl "http://localhost:8080/api/topics/events/messages?partition=0&offset=0&limit=100"

# Second page (offsets 100-199)
curl "http://localhost:8080/api/topics/events/messages?partition=0&offset=100&limit=100"
```

Topics and consumer groups do not support pagination in the current version.

## CORS Configuration

CORS is enabled for the following origins:
- `http://localhost:*`
- `http://127.0.0.1:*`

Allowed methods: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`  
Allowed headers: `Accept`, `Authorization`, `Content-Type`, `X-CSRF-Token`

## Swagger/OpenAPI Documentation

Interactive API documentation is available via Swagger UI:

**URL**: http://localhost:8080/swagger/index.html

The Swagger UI provides:
- Complete API reference
- Request/response schemas
- Interactive testing
- Code generation support

**OpenAPI Spec**: http://localhost:8080/swagger/doc.json

## SDK Examples

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

const baseURL = "http://localhost:8080/api"

type CreateTopicRequest struct {
    Name       string `json:"name"`
    Partitions int    `json:"partitions"`
}

type ProduceMessageRequest struct {
    Partition int    `json:"partition"`
    Key       string `json:"key"`
    Value     string `json:"value"`
}

type ProduceMessageResponse struct {
    Offset    int64 `json:"offset"`
    Partition int32 `json:"partition"`
}

func main() {
    // Create topic
    createReq := CreateTopicRequest{
        Name:       "orders",
        Partitions: 3,
    }
    if err := createTopic(createReq); err != nil {
        panic(err)
    }
    
    // Produce message
    produceReq := ProduceMessageRequest{
        Partition: 0,
        Key:       "order-123",
        Value:     `{"product":"laptop","price":1299.99}`,
    }
    resp, err := produceMessage("orders", produceReq)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Message produced at offset %d\n", resp.Offset)
}

func createTopic(req CreateTopicRequest) error {
    data, _ := json.Marshal(req)
    resp, err := http.Post(baseURL+"/topics", "application/json", bytes.NewReader(data))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 201 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to create topic: %s", body)
    }
    return nil
}

func produceMessage(topic string, req ProduceMessageRequest) (*ProduceMessageResponse, error) {
    data, _ := json.Marshal(req)
    url := fmt.Sprintf("%s/topics/%s/messages", baseURL, topic)
    resp, err := http.Post(url, "application/json", bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 201 {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to produce message: %s", body)
    }
    
    var result ProduceMessageResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}
```

### Python

```python
import requests
import json

BASE_URL = "http://localhost:8080/api"

class TakhinClient:
    def __init__(self, base_url=BASE_URL, api_key=None):
        self.base_url = base_url
        self.headers = {}
        if api_key:
            self.headers["Authorization"] = f"Bearer {api_key}"
    
    def create_topic(self, name, partitions):
        """Create a new topic"""
        url = f"{self.base_url}/topics"
        data = {"name": name, "partitions": partitions}
        resp = requests.post(url, json=data, headers=self.headers)
        resp.raise_for_status()
        return resp.json()
    
    def list_topics(self):
        """List all topics"""
        url = f"{self.base_url}/topics"
        resp = requests.get(url, headers=self.headers)
        resp.raise_for_status()
        return resp.json()
    
    def produce_message(self, topic, partition, key, value):
        """Produce a message to a topic"""
        url = f"{self.base_url}/topics/{topic}/messages"
        data = {
            "partition": partition,
            "key": key,
            "value": value
        }
        resp = requests.post(url, json=data, headers=self.headers)
        resp.raise_for_status()
        return resp.json()
    
    def get_messages(self, topic, partition, offset, limit=100):
        """Get messages from a topic partition"""
        url = f"{self.base_url}/topics/{topic}/messages"
        params = {
            "partition": partition,
            "offset": offset,
            "limit": limit
        }
        resp = requests.get(url, params=params, headers=self.headers)
        resp.raise_for_status()
        return resp.json()
    
    def list_consumer_groups(self):
        """List all consumer groups"""
        url = f"{self.base_url}/consumer-groups"
        resp = requests.get(url, headers=self.headers)
        resp.raise_for_status()
        return resp.json()

# Example usage
if __name__ == "__main__":
    client = TakhinClient(api_key="your-api-key")
    
    # Create topic
    client.create_topic("events", 3)
    
    # Produce message
    result = client.produce_message(
        topic="events",
        partition=0,
        key="user-123",
        value='{"action":"login","timestamp":1704067200}'
    )
    print(f"Produced message at offset {result['offset']}")
    
    # Read messages
    messages = client.get_messages("events", partition=0, offset=0, limit=10)
    for msg in messages:
        print(f"Offset {msg['offset']}: {msg['value']}")
```

### JavaScript/TypeScript

```typescript
interface CreateTopicRequest {
  name: string;
  partitions: number;
}

interface ProduceMessageRequest {
  partition: number;
  key: string;
  value: string;
}

interface ProduceMessageResponse {
  offset: number;
  partition: number;
}

interface Message {
  partition: number;
  offset: number;
  key: string;
  value: string;
  timestamp: number;
}

class TakhinClient {
  constructor(
    private baseUrl: string = "http://localhost:8080/api",
    private apiKey?: string
  ) {}

  private get headers() {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (this.apiKey) {
      headers["Authorization"] = `Bearer ${this.apiKey}`;
    }
    return headers;
  }

  async createTopic(name: string, partitions: number): Promise<void> {
    const resp = await fetch(`${this.baseUrl}/topics`, {
      method: "POST",
      headers: this.headers,
      body: JSON.stringify({ name, partitions }),
    });
    if (!resp.ok) {
      throw new Error(`Failed to create topic: ${await resp.text()}`);
    }
  }

  async listTopics(): Promise<any[]> {
    const resp = await fetch(`${this.baseUrl}/topics`, {
      headers: this.headers,
    });
    if (!resp.ok) {
      throw new Error(`Failed to list topics: ${await resp.text()}`);
    }
    return resp.json();
  }

  async produceMessage(
    topic: string,
    req: ProduceMessageRequest
  ): Promise<ProduceMessageResponse> {
    const resp = await fetch(`${this.baseUrl}/topics/${topic}/messages`, {
      method: "POST",
      headers: this.headers,
      body: JSON.stringify(req),
    });
    if (!resp.ok) {
      throw new Error(`Failed to produce message: ${await resp.text()}`);
    }
    return resp.json();
  }

  async getMessages(
    topic: string,
    partition: number,
    offset: number,
    limit: number = 100
  ): Promise<Message[]> {
    const params = new URLSearchParams({
      partition: partition.toString(),
      offset: offset.toString(),
      limit: limit.toString(),
    });
    const resp = await fetch(
      `${this.baseUrl}/topics/${topic}/messages?${params}`,
      { headers: this.headers }
    );
    if (!resp.ok) {
      throw new Error(`Failed to get messages: ${await resp.text()}`);
    }
    return resp.json();
  }
}

// Example usage
const client = new TakhinClient("http://localhost:8080/api", "your-api-key");

// Create topic and produce message
await client.createTopic("events", 3);
const result = await client.produceMessage("events", {
  partition: 0,
  key: "user-123",
  value: JSON.stringify({ action: "login" }),
});
console.log(`Produced at offset ${result.offset}`);
```

## Performance Best Practices

1. **Batch Message Reads**: Use higher `limit` values to reduce API calls
2. **Connection Reuse**: Use HTTP keep-alive for better performance
3. **Parallel Requests**: Read from multiple partitions concurrently
4. **Avoid Polling**: Use appropriate `offset` values instead of repeated queries

## Monitoring

Monitor API performance via standard HTTP metrics:
- Request rate
- Response time
- Error rate
- Status code distribution

Future versions will include Prometheus metrics endpoint.

## References

- [Swagger UI](http://localhost:8080/swagger/index.html)
- [OpenAPI Specification](http://localhost:8080/swagger/doc.json)
- [Source Code](../../backend/pkg/console/)
- [Authentication Documentation](../../backend/pkg/console/AUTH.md)
