# gRPC API Implementation

## Overview

The Takhin gRPC API provides high-performance streaming interfaces for interacting with the Takhin messaging platform. It offers better performance than REST for high-throughput scenarios and supports bidirectional streaming.

## Features

- **Topic Management**: Create, list, describe, and delete topics
- **Producer API**: Synchronous and streaming message production
- **Consumer API**: Stream-based message consumption
- **Consumer Groups**: List, describe, and manage consumer groups
- **Partition Operations**: Query partition offsets and metadata
- **Health Checks**: Built-in health and readiness probes

## Proto Definition

The API is defined in `api/proto/takhin.proto` using Protocol Buffers v3. Key service methods:

### Topic Operations
- `CreateTopic`: Create a new topic with specified partitions
- `DeleteTopic`: Delete an existing topic
- `ListTopics`: List all topics
- `GetTopic`: Get detailed topic metadata
- `DescribeTopics`: Batch describe multiple topics

### Producer Operations
- `ProduceMessage`: Produce a single message (unary RPC)
- `ProduceMessageStream`: Streaming message production (bidirectional)

### Consumer Operations
- `ConsumeMessages`: Stream messages from a topic partition (server streaming)
- `CommitOffset`: Commit consumer group offsets

### Consumer Group Operations
- `ListConsumerGroups`: List all consumer groups
- `DescribeConsumerGroup`: Get consumer group details
- `DeleteConsumerGroup`: Delete a consumer group

### Partition Operations
- `GetPartitionOffsets`: Get beginning and end offsets for a partition

### Health Check
- `HealthCheck`: Check service health and uptime

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ gRPC
       ▼
┌─────────────────────┐
│  GRPCServer         │
│  (grpc_server.go)   │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  Server             │
│  (server.go)        │
│  - CreateTopic      │
│  - ProduceMessage   │
│  - ConsumeMessages  │
│  - etc.             │
└─────────┬───────────┘
          │
    ┌─────┴─────┐
    ▼           ▼
┌─────────┐ ┌──────────────┐
│ Topic   │ │ Coordinator  │
│ Manager │ │              │
└─────────┘ └──────────────┘
```

## Usage

### Starting the gRPC Server

```go
import (
    "github.com/takhin-data/takhin/pkg/grpcapi"
    "github.com/takhin-data/takhin/pkg/storage/topic"
    "github.com/takhin-data/takhin/pkg/coordinator"
)

// Create dependencies
topicManager := topic.NewManager("/data", 1024*1024*1024)
coord := coordinator.NewCoordinator(topicManager, 1)
coord.Start()

// Create and start gRPC server
grpcServer, err := grpcapi.NewGRPCServer(":9092", topicManager, coord, "1.0.0")
if err != nil {
    log.Fatal(err)
}

go func() {
    if err := grpcServer.Start(); err != nil {
        log.Fatal(err)
    }
}()

// Graceful shutdown
defer grpcServer.Stop()
```

### Client Example (Go)

```go
import (
    "context"
    "google.golang.org/grpc"
    pb "github.com/takhin-data/takhin/api/proto"
)

// Connect to server
conn, err := grpc.Dial("localhost:9092", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewTakhinServiceClient(conn)

// Create topic
resp, err := client.CreateTopic(context.Background(), &pb.CreateTopicRequest{
    Name:          "my-topic",
    NumPartitions: 3,
})

// Produce message
produceResp, err := client.ProduceMessage(context.Background(), &pb.ProduceMessageRequest{
    Topic:     "my-topic",
    Partition: 0,
    Record: &pb.Record{
        Key:   []byte("key1"),
        Value: []byte("Hello, Takhin!"),
    },
})

// Consume messages (streaming)
stream, err := client.ConsumeMessages(context.Background(), &pb.ConsumeMessagesRequest{
    Topic:     "my-topic",
    Partition: 0,
    Offset:    0,
    MaxBytes:  1024 * 1024,
})

for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    for _, record := range resp.Records {
        fmt.Printf("Offset: %d, Value: %s\n", record.Offset, string(record.Value))
    }
}
```

## Performance Configuration

The gRPC server is configured with optimized settings:

- **Max Message Size**: 10MB (configurable)
- **Keepalive**: 5 minute intervals
- **Connection Limits**: 
  - Max idle: 15 minutes
  - Max age: 30 minutes
  - Grace period: 5 minutes

### Server Options

```go
grpc.MaxRecvMsgSize(10 * 1024 * 1024)  // 10MB
grpc.MaxSendMsgSize(10 * 1024 * 1024)  // 10MB
grpc.KeepaliveParams(keepalive.ServerParameters{
    MaxConnectionIdle:     15 * time.Minute,
    MaxConnectionAge:      30 * time.Minute,
    MaxConnectionAgeGrace: 5 * time.Minute,
    Time:                  5 * time.Minute,
    Timeout:               1 * time.Minute,
})
```

## Streaming APIs

### Producer Streaming

Bidirectional streaming for high-throughput message production:

```go
stream, err := client.ProduceMessageStream(context.Background())

// Send messages
for i := 0; i < 1000; i++ {
    err := stream.Send(&pb.ProduceMessageRequest{
        Topic:     "my-topic",
        Partition: int32(i % 3),
        Record: &pb.Record{
            Value: []byte(fmt.Sprintf("message-%d", i)),
        },
    })
}

// Receive responses
for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break
    }
    fmt.Printf("Produced to offset: %d\n", resp.Offset)
}
```

### Consumer Streaming

Server-side streaming for continuous message consumption:

```go
stream, err := client.ConsumeMessages(context.Background(), &pb.ConsumeMessagesRequest{
    Topic:     "my-topic",
    Partition: 0,
    Offset:    -1, // Start from earliest
    MaxBytes:  1024 * 1024,
})

for {
    batch, err := stream.Recv()
    if err != nil {
        break
    }
    
    for _, record := range batch.Records {
        processMessage(record)
    }
}
```

## Testing

### Unit Tests

Run the test suite:

```bash
cd backend
go test ./pkg/grpcapi/...
```

### Benchmarks

Run performance benchmarks:

```bash
cd backend
go test -bench=. -benchmem ./pkg/grpcapi/
```

Expected results:
- **Produce throughput**: 50,000-100,000 msgs/sec (1KB messages)
- **Average latency**: <1ms for local produce
- **Concurrent performance**: Linear scaling up to CPU cores

Example benchmark output:
```
BenchmarkProduceMessage-8                50000    25000 ns/op    2048 B/op    15 allocs/op
BenchmarkProduceMessageParallel-8       200000     6000 ns/op    2048 B/op    15 allocs/op
BenchmarkThroughput/size_1024-8                   120 MB/s      80000 msgs/s
```

## Code Generation

To regenerate proto files after modifying `takhin.proto`:

### Prerequisites

```bash
# Install protoc compiler
# macOS
brew install protobuf

# Linux
sudo apt-get install protobuf-compiler

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate Code

```bash
cd backend
make -f Makefile.proto proto
```

This generates:
- `api/proto/takhin.pb.go` - Message types
- `api/proto/takhin_grpc.pb.go` - Service interfaces

## Health Checks

The gRPC server implements the standard gRPC health checking protocol:

```bash
# Using grpcurl
grpcurl -plaintext localhost:9092 grpc.health.v1.Health/Check

# Response
{
  "status": "SERVING"
}
```

Custom health check endpoint:

```go
resp, err := client.HealthCheck(context.Background(), &pb.HealthCheckRequest{})
// Returns: status, version, uptime_seconds
```

## Reflection

The server enables gRPC reflection for debugging and testing:

```bash
# List services
grpcurl -plaintext localhost:9092 list

# List methods
grpcurl -plaintext localhost:9092 list takhin.v1.TakhinService

# Describe a method
grpcurl -plaintext localhost:9092 describe takhin.v1.TakhinService.CreateTopic
```

## Error Handling

The API uses standard gRPC status codes:

- `OK`: Success
- `INVALID_ARGUMENT`: Invalid request parameters
- `NOT_FOUND`: Topic/partition not found
- `ALREADY_EXISTS`: Topic already exists
- `INTERNAL`: Internal server error
- `UNAVAILABLE`: Service temporarily unavailable

Response messages include an `error` field with details.

## Integration

### With Takhin Server

Add gRPC server alongside Kafka protocol server:

```go
// In cmd/takhin/main.go
grpcServer, err := grpcapi.NewGRPCServer(
    config.GRPCAddr,
    topicManager,
    coordinator,
    version,
)

go grpcServer.Start()
defer grpcServer.Stop()
```

### Configuration

Add to `configs/takhin.yaml`:

```yaml
grpc:
  addr: ":9092"
  max_message_size: 10485760  # 10MB
  keepalive_interval: 300     # 5 minutes
```

## Monitoring

Integrate with Prometheus metrics:

- `grpc_server_handled_total`: Total requests by method
- `grpc_server_handling_seconds`: Request duration histogram
- `grpc_server_msg_received_total`: Messages received
- `grpc_server_msg_sent_total`: Messages sent

## Security

### TLS Support

Enable TLS for encrypted communication:

```go
creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
grpcServer := grpc.NewServer(grpc.Creds(creds))
```

### Authentication

Implement auth interceptor:

```go
func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // Validate token from metadata
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
    }
    
    // Check API key
    if !validateAPIKey(md["api-key"]) {
        return nil, status.Errorf(codes.PermissionDenied, "invalid API key")
    }
    
    return handler(ctx, req)
}
```

## Comparison with REST API

| Feature | gRPC | REST |
|---------|------|------|
| Protocol | HTTP/2 | HTTP/1.1 |
| Payload | Protobuf | JSON |
| Streaming | Bidirectional | Limited |
| Throughput | High | Medium |
| Latency | Low | Medium |
| Browser support | Limited | Full |
| Code generation | Required | Optional |

**Use gRPC when:**
- High throughput is required
- Low latency is critical
- Streaming data flows
- Strong typing is desired

**Use REST when:**
- Browser clients
- Simple CRUD operations
- Human-readable payloads
- Widespread tooling

## Future Enhancements

- [ ] TLS/mTLS support
- [ ] Authentication/authorization interceptors
- [ ] Rate limiting
- [ ] Request tracing (OpenTelemetry)
- [ ] Advanced compression (gzip, snappy)
- [ ] Load balancing support
- [ ] Circuit breaker pattern
- [ ] Metrics exporters

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
- [gRPC Go Quickstart](https://grpc.io/docs/languages/go/quickstart/)
- [gRPC Performance Best Practices](https://grpc.io/docs/guides/performance/)
