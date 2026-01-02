# Takhin API Documentation

## Overview

Takhin provides two complementary APIs for interacting with the streaming platform:

1. **Kafka Protocol API**: Binary protocol compatible with Apache Kafka clients
2. **Console REST API**: HTTP/JSON API for management and monitoring

## API Comparison

| Feature | Kafka Protocol API | Console REST API |
|---------|-------------------|------------------|
| **Protocol** | Binary (TCP) | HTTP/JSON |
| **Port** | 9092 | 8080 |
| **Use Case** | High-performance streaming | Management, monitoring, debugging |
| **Authentication** | SASL | API Key |
| **Client Libraries** | kafka-go, kafka-python, KafkaJS | Any HTTP client |
| **Performance** | High throughput | Moderate |
| **Batch Operations** | Yes (record batches) | Single message |
| **Consumer Groups** | Full support | Read-only monitoring |
| **Transactions** | Yes | No |

## When to Use Each API

### Use Kafka Protocol API When:

- Building production applications that need high throughput
- Using existing Kafka client libraries
- Need consumer group coordination
- Require transactional semantics
- Want to use standard Kafka tools (kafka-console-producer, etc.)
- Need efficient binary protocol

**Example Use Cases:**
- Event streaming applications
- Real-time data pipelines
- Microservices communication
- Log aggregation

### Use Console REST API When:

- Building management/admin tools
- Creating web dashboards
- Debugging and testing
- Quick prototyping
- Need simple HTTP/JSON interface
- Monitoring consumer groups
- Exploring topics and messages

**Example Use Cases:**
- Web-based management UI
- Monitoring dashboards
- Testing and development tools
- CI/CD scripts
- Administrative tasks

## Quick Start

### Kafka Protocol API

```bash
# Install kafka-go
go get github.com/segmentio/kafka-go

# Or kafka-python
pip install kafka-python
```

```go
// Produce message
writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "my-topic",
})
writer.WriteMessages(context.Background(),
    kafka.Message{Value: []byte("hello")},
)
```

```python
# Consume messages
consumer = KafkaConsumer('my-topic',
    bootstrap_servers=['localhost:9092'])
for message in consumer:
    print(message.value)
```

### Console REST API

```bash
# Create topic
curl -X POST http://localhost:8080/api/topics \
  -H "Content-Type: application/json" \
  -d '{"name":"my-topic","partitions":3}'

# Produce message
curl -X POST http://localhost:8080/api/topics/my-topic/messages \
  -H "Content-Type: application/json" \
  -d '{"partition":0,"key":"key1","value":"hello"}'

# Read messages
curl "http://localhost:8080/api/topics/my-topic/messages?partition=0&offset=0&limit=10"
```

## Documentation Structure

```
docs/api/
├── README.md                    # This file
├── kafka-protocol-api.md        # Complete Kafka protocol reference
├── console-rest-api.md          # Complete REST API reference
└── examples/                    # Code examples
    ├── go/
    │   ├── kafka_client_example.go
    │   └── console_client_example.go
    ├── python/
    │   ├── kafka_client_example.py
    │   └── console_client_example.py
    └── javascript/
        └── console_client_example.ts
```

## API Features Matrix

### Kafka Protocol API

| Feature | Status | API Key |
|---------|--------|---------|
| Produce | ✅ Full | 0 |
| Fetch | ✅ Full | 1 |
| ListOffsets | ✅ Full | 2 |
| Metadata | ✅ Full | 3 |
| OffsetCommit | ✅ Full | 8 |
| OffsetFetch | ✅ Full | 9 |
| FindCoordinator | ✅ Full | 10 |
| JoinGroup | ✅ Full | 11 |
| Heartbeat | ✅ Full | 12 |
| LeaveGroup | ✅ Full | 13 |
| SyncGroup | ✅ Full | 14 |
| DescribeGroups | ✅ Full | 15 |
| ListGroups | ✅ Full | 16 |
| ApiVersions | ✅ Full | 18 |
| CreateTopics | ✅ Full | 19 |
| DeleteTopics | ✅ Full | 20 |
| DeleteRecords | ✅ Full | 21 |
| InitProducerID | ✅ Full | 22 |
| AddPartitionsToTxn | ✅ Full | 24 |
| AddOffsetsToTxn | ✅ Full | 25 |
| EndTxn | ✅ Full | 26 |
| WriteTxnMarkers | ✅ Full | 27 |
| TxnOffsetCommit | ✅ Full | 28 |
| DescribeConfigs | ✅ Full | 32 |
| AlterConfigs | ✅ Full | 33 |
| DescribeLogDirs | ✅ Full | 35 |
| SaslHandshake | ✅ Full | 36 |
| SaslAuthenticate | ✅ Full | 37 |

### Console REST API

| Endpoint | Method | Status |
|----------|--------|--------|
| Health Check | GET /api/health | ✅ |
| Readiness | GET /api/health/ready | ✅ |
| Liveness | GET /api/health/live | ✅ |
| List Topics | GET /api/topics | ✅ |
| Get Topic | GET /api/topics/{topic} | ✅ |
| Create Topic | POST /api/topics | ✅ |
| Delete Topic | DELETE /api/topics/{topic} | ✅ |
| Get Messages | GET /api/topics/{topic}/messages | ✅ |
| Produce Message | POST /api/topics/{topic}/messages | ✅ |
| List Consumer Groups | GET /api/consumer-groups | ✅ |
| Get Consumer Group | GET /api/consumer-groups/{group} | ✅ |

## Authentication

### Kafka Protocol (SASL)

```yaml
# takhin.yaml
kafka:
  sasl:
    enabled: true
    mechanisms: [PLAIN, SCRAM-SHA-256]
```

### Console REST API (API Key)

```bash
# Start with authentication
./console \
  -enable-auth \
  -api-keys="key1,key2,key3"

# Use API key
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8080/api/topics
```

## Client Libraries

### Kafka Protocol

**Go:**
- [kafka-go](https://github.com/segmentio/kafka-go) - Pure Go client
- [sarama](https://github.com/Shopify/sarama) - Shopify's Go client
- [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go) - Confluent's client

**Python:**
- [kafka-python](https://github.com/dpkp/kafka-python) - Pure Python client
- [confluent-kafka-python](https://github.com/confluentinc/confluent-kafka-python) - Confluent's client

**JavaScript:**
- [KafkaJS](https://kafka.js.org/) - Modern Node.js client
- [node-rdkafka](https://github.com/Blizzard/node-rdkafka) - librdkafka wrapper

**Java:**
- [Apache Kafka Java Client](https://kafka.apache.org/documentation/#api) - Official client

### Console REST API

Any HTTP client:
- **Go**: `net/http`, `resty`
- **Python**: `requests`, `httpx`
- **JavaScript**: `fetch`, `axios`
- **Java**: `HttpClient`, `OkHttp`
- **Rust**: `reqwest`

## Performance Guidelines

### Kafka Protocol API

**Producer:**
- Use batching (`batch.size`, `linger.ms`)
- Enable compression (gzip, snappy, lz4, zstd)
- Set appropriate `acks` level
- Use async sends for high throughput

**Consumer:**
- Increase `fetch.min.bytes` for better batching
- Use appropriate `max.poll.records`
- Enable consumer groups for parallel processing
- Commit offsets periodically, not per message

### Console REST API

**Best Practices:**
- Use higher `limit` values when reading messages
- Cache topic metadata
- Use HTTP keep-alive
- Read from multiple partitions in parallel
- Avoid polling; use appropriate offsets

## Error Handling

### Kafka Protocol

Common error codes:
- `0`: Success
- `1`: OffsetOutOfRange
- `3`: UnknownTopicOrPartition
- `6`: NotLeaderForPartition
- `27`: NotCoordinator
- `36`: TopicAlreadyExists

See [Kafka Protocol API docs](kafka-protocol-api.md#error-codes) for complete list.

### Console REST API

HTTP status codes:
- `200`: OK
- `201`: Created
- `204`: No Content
- `400`: Bad Request
- `401`: Unauthorized
- `404`: Not Found
- `500`: Internal Server Error
- `503`: Service Unavailable

Error response format:
```json
{
  "error": "descriptive error message"
}
```

## Interactive Documentation

### Swagger UI

When the Console server is running, access interactive API documentation:

**URL**: http://localhost:8080/swagger/index.html

Features:
- Browse all endpoints
- View request/response schemas
- Test API calls directly
- Download OpenAPI spec

### OpenAPI Specification

**URL**: http://localhost:8080/swagger/doc.json

Use for:
- Generating client SDKs
- Importing into Postman/Insomnia
- API testing tools
- Integration with API gateways

## Monitoring

### Metrics

Future versions will expose Prometheus metrics:
- Request rate and latency
- Message throughput
- Consumer lag
- Partition sizes
- Error rates

### Health Checks

```bash
# Basic health
curl http://localhost:8080/api/health

# Readiness (for K8s)
curl http://localhost:8080/api/health/ready

# Liveness (for K8s)
curl http://localhost:8080/api/health/live
```

## Migration from Apache Kafka

Takhin is protocol-compatible with Apache Kafka. To migrate:

1. **Update bootstrap servers** in your client configuration
2. **No code changes required** - use existing Kafka clients
3. **Test with subset of traffic** first
4. **Monitor performance** and adjust configuration
5. **Gradually migrate** all applications

## Limitations

### Current Limitations

- **Max message size**: 1MB (configurable)
- **Console API**: Single message operations only
- **No ACLs**: Authentication only (authorization planned)
- **No quotas**: Rate limiting planned for future

### Planned Features

- ⏳ Enhanced ACL support
- ⏳ Rate limiting and quotas
- ⏳ Prometheus metrics endpoint
- ⏳ WebSocket support for real-time updates
- ⏳ Batch operations in Console API
- ⏳ Schema registry integration

## Support

### Documentation

- [Kafka Protocol API Reference](kafka-protocol-api.md)
- [Console REST API Reference](console-rest-api.md)
- [Code Examples](examples/)
- [Architecture Documentation](../architecture/)

### Resources

- GitHub Repository: [takhin-data/takhin](https://github.com/takhin-data/takhin)
- Issue Tracker: [GitHub Issues](https://github.com/takhin-data/takhin/issues)
- Discussions: [GitHub Discussions](https://github.com/takhin-data/takhin/discussions)

### Community

- Questions: Use GitHub Discussions
- Bug Reports: Use GitHub Issues
- Feature Requests: Use GitHub Issues with enhancement label

## Version History

### v1.0 (Current)

- ✅ Full Kafka protocol compatibility
- ✅ 27 Kafka APIs implemented
- ✅ Console REST API (11 endpoints)
- ✅ Consumer group support
- ✅ Transaction support
- ✅ SASL authentication
- ✅ API key authentication
- ✅ Swagger/OpenAPI documentation

## Contributing

Contributions are welcome! See the main repository for contribution guidelines.

---

**Last Updated**: 2026-01-02  
**API Version**: 1.0  
**Protocol Version**: Kafka 2.8+
