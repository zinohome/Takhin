# API Examples

This directory contains example code demonstrating how to interact with Takhin using both the Kafka protocol and Console REST API.

## Directory Structure

```
examples/
├── go/                          # Go examples
│   ├── kafka_client_example.go      # Kafka protocol with kafka-go
│   └── console_client_example.go    # Console REST API
├── python/                      # Python examples
│   ├── kafka_client_example.py      # Kafka protocol with kafka-python
│   └── console_client_example.py    # Console REST API with requests
└── javascript/                  # JavaScript/TypeScript examples
    └── console_client_example.ts    # Console REST API with Fetch API
```

## Prerequisites

### Go Examples

```bash
# Install kafka-go
go get github.com/segmentio/kafka-go
```

### Python Examples

```bash
# Install dependencies
pip install kafka-python requests
```

### JavaScript/TypeScript Examples

```bash
# For Node.js < 18, install node-fetch
npm install node-fetch

# For TypeScript
npm install -D @types/node
```

## Running Examples

### Go

```bash
# Kafka protocol example
cd examples/go
go run kafka_client_example.go

# Console REST API example
go run console_client_example.go
```

### Python

```bash
# Kafka protocol example
cd examples/python
python kafka_client_example.py

# Console REST API example
python console_client_example.py
```

### JavaScript/TypeScript

```bash
# TypeScript example (Node.js 18+)
cd examples/javascript
npx ts-node console_client_example.ts

# Or compile and run
tsc console_client_example.ts
node console_client_example.js
```

## Example Features

### Kafka Protocol Examples

All Kafka protocol examples demonstrate:
- Creating topics
- Producing messages
- Consuming messages
- Consumer groups
- Offset management
- Transactions (where supported)

### Console REST API Examples

All Console REST API examples demonstrate:
- Health checks
- Topic management (create, list, get, delete)
- Message production
- Message consumption
- Consumer group monitoring

## Authentication

If API authentication is enabled on the Console server, set the API key in the examples:

**Go:**
```go
const apiKey = "your-api-key"
```

**Python:**
```python
client = TakhinClient(api_key="your-api-key")
```

**JavaScript:**
```typescript
const client = new TakhinClient(BASE_URL, "your-api-key");
```

## Configuration

Update these constants in the examples to match your environment:

- **Kafka Bootstrap Server**: Default `localhost:9092`
- **Console API Base URL**: Default `http://localhost:8080/api`
- **Topic Names**: Default `demo-topic`, `events`, etc.

## Error Handling

All examples include error handling. Common errors:

- **Connection refused**: Ensure Takhin is running
- **Topic not found**: Create the topic first
- **Authentication failed**: Check API key if auth is enabled
- **Timeout**: Increase timeout values if needed

## Additional Resources

- [Kafka Protocol API Documentation](../kafka-protocol-api.md)
- [Console REST API Documentation](../console-rest-api.md)
- [Swagger UI](http://localhost:8080/swagger/index.html) (when Console is running)
