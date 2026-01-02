# Kafka Protocol API Documentation

## Overview

Takhin implements the Apache Kafka wire protocol, providing full compatibility with standard Kafka clients and tools. This document describes all supported Kafka protocol APIs, their request/response formats, and usage examples.

**Protocol Version**: Kafka 2.8+  
**Binary Format**: Big-endian encoding  
**Transport**: TCP (default port: 9092)

## API Version Support

| API Key | API Name | Versions | Status |
|---------|----------|----------|--------|
| 0 | Produce | 0-9 | ✅ Full |
| 1 | Fetch | 0-13 | ✅ Full |
| 2 | ListOffsets | 0-7 | ✅ Full |
| 3 | Metadata | 0-12 | ✅ Full |
| 8 | OffsetCommit | 0-8 | ✅ Full |
| 9 | OffsetFetch | 0-8 | ✅ Full |
| 10 | FindCoordinator | 0-4 | ✅ Full |
| 11 | JoinGroup | 0-9 | ✅ Full |
| 12 | Heartbeat | 0-4 | ✅ Full |
| 13 | LeaveGroup | 0-5 | ✅ Full |
| 14 | SyncGroup | 0-5 | ✅ Full |
| 15 | DescribeGroups | 0-5 | ✅ Full |
| 16 | ListGroups | 0-4 | ✅ Full |
| 18 | ApiVersions | 0-3 | ✅ Full |
| 19 | CreateTopics | 0-7 | ✅ Full |
| 20 | DeleteTopics | 0-6 | ✅ Full |
| 21 | DeleteRecords | 0-2 | ✅ Full |
| 22 | InitProducerID | 0-4 | ✅ Full |
| 24 | AddPartitionsToTxn | 0-3 | ✅ Full |
| 25 | AddOffsetsToTxn | 0-3 | ✅ Full |
| 26 | EndTxn | 0-3 | ✅ Full |
| 27 | WriteTxnMarkers | 0-1 | ✅ Full |
| 28 | TxnOffsetCommit | 0-3 | ✅ Full |
| 32 | DescribeConfigs | 0-4 | ✅ Full |
| 33 | AlterConfigs | 0-2 | ✅ Full |
| 35 | DescribeLogDirs | 0-4 | ✅ Full |
| 36 | SaslHandshake | 0-1 | ✅ Full |
| 37 | SaslAuthenticate | 0-2 | ✅ Full |

## Core APIs

### 1. Produce (API Key 0)

Write messages to topic partitions.

**Request Format:**
```go
type ProduceRequest struct {
    TransactionalID *string              // Optional transaction ID
    Acks            int16                // -1 (all), 0 (none), 1 (leader)
    TimeoutMs       int32                // Request timeout
    TopicData       []TopicProduceData
}

type TopicProduceData struct {
    Topic          string
    PartitionData  []PartitionProduceData
}

type PartitionProduceData struct {
    Partition int32
    Records   []byte  // Record batch in binary format
}
```

**Response Format:**
```go
type ProduceResponse struct {
    Responses      []TopicProduceResponse
    ThrottleTimeMs int32
}

type TopicProduceResponse struct {
    Topic              string
    PartitionResponses []PartitionProduceResponse
}

type PartitionProduceResponse struct {
    Partition      int32
    ErrorCode      int16
    BaseOffset     int64
    LogAppendTimeMs int64
    LogStartOffset int64
}
```

**Acknowledgment Modes:**
- **acks=0**: No acknowledgment (fire-and-forget)
- **acks=1**: Leader acknowledgment only
- **acks=-1/all**: Wait for full ISR acknowledgment

**Example (kafka-go):**
```go
writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers:  []string{"localhost:9092"},
    Topic:    "my-topic",
    Balancer: &kafka.LeastBytes{},
    RequiredAcks: -1, // Wait for all ISR
})

err := writer.WriteMessages(context.Background(),
    kafka.Message{
        Key:   []byte("key"),
        Value: []byte("value"),
    },
)
```

**Error Codes:**
- `0` (None): Success
- `1` (OffsetOutOfRange): Invalid offset
- `3` (UnknownTopicOrPartition): Topic/partition doesn't exist
- `6` (NotLeaderForPartition): Request to non-leader broker
- `10` (MessageTooLarge): Message exceeds max size

### 2. Fetch (API Key 1)

Read messages from topic partitions.

**Request Format:**
```go
type FetchRequest struct {
    ReplicaID      int32  // -1 for consumer clients
    MaxWaitMs      int32  // Max time to wait for min bytes
    MinBytes       int32  // Min bytes to accumulate
    MaxBytes       int32  // Max response size
    IsolationLevel int8   // 0=read_uncommitted, 1=read_committed
    Topics         []FetchTopic
}

type FetchTopic struct {
    Topic      string
    Partitions []FetchPartition
}

type FetchPartition struct {
    Partition          int32
    FetchOffset        int64  // Offset to start reading from
    PartitionMaxBytes  int32  // Max bytes for this partition
    LogStartOffset     int64  // For truncation detection
}
```

**Response Format:**
```go
type FetchResponse struct {
    ThrottleTimeMs int32
    Responses      []FetchTopicResponse
}

type FetchTopicResponse struct {
    Topic      string
    Partitions []FetchPartitionResponse
}

type FetchPartitionResponse struct {
    Partition         int32
    ErrorCode         int16
    HighWatermark     int64  // Last committed offset
    LastStableOffset  int64  // For transactions
    Records           []byte // Record batch
}
```

**Example (kafka-go):**
```go
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers:  []string{"localhost:9092"},
    Topic:    "my-topic",
    Partition: 0,
    MinBytes: 1,
    MaxBytes: 10e6, // 10MB
    MaxWait:  500 * time.Millisecond,
})

msg, err := reader.ReadMessage(context.Background())
```

### 3. Metadata (API Key 3)

Query cluster and topic metadata.

**Request Format:**
```go
type MetadataRequest struct {
    Topics                         []string  // nil = all topics
    AllowAutoTopicCreation         bool
    IncludeClusterAuthorizedOps    bool
    IncludeTopicAuthorizedOps      bool
}
```

**Response Format:**
```go
type MetadataResponse struct {
    ThrottleTimeMs int32
    Brokers        []Broker
    ClusterID      *string
    ControllerID   int32
    TopicMetadata  []TopicMetadata
}

type Broker struct {
    NodeID int32
    Host   string
    Port   int32
    Rack   *string
}

type TopicMetadata struct {
    ErrorCode         int16
    TopicName         string
    IsInternal        bool
    PartitionMetadata []PartitionMetadata
}

type PartitionMetadata struct {
    ErrorCode       int16
    PartitionID     int32
    Leader          int32    // Leader broker ID
    Replicas        []int32  // All replicas
    ISR             []int32  // In-sync replicas
    OfflineReplicas []int32
}
```

**Example (kafka-topics.sh):**
```bash
kafka-topics.sh --bootstrap-server localhost:9092 --list
```

### 4. ListOffsets (API Key 2)

Query offset information for partitions.

**Special Offsets:**
- `-2` (earliest): First available offset
- `-1` (latest): Next offset to be written (high watermark)

**Request Format:**
```go
type ListOffsetsRequest struct {
    ReplicaID int32
    Topics    []ListOffsetsTopic
}

type ListOffsetsTopic struct {
    Topic      string
    Partitions []ListOffsetsPartition
}

type ListOffsetsPartition struct {
    Partition int32
    Timestamp int64  // -2 (earliest), -1 (latest), or Unix ms
}
```

**Example:**
```bash
# Get earliest offset
kafka-run-class.sh kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic my-topic \
  --time -2

# Get latest offset
kafka-run-class.sh kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic my-topic \
  --time -1
```

## Consumer Group APIs

### 5. FindCoordinator (API Key 10)

Locate the coordinator for a consumer group or transaction.

**Request Format:**
```go
type FindCoordinatorRequest struct {
    Key     string  // Group ID or Transactional ID
    KeyType int8    // 0=Group, 1=Transaction
}
```

**Response Format:**
```go
type FindCoordinatorResponse struct {
    ErrorCode   int16
    ErrorMsg    *string
    NodeID      int32   // Coordinator broker
    Host        string
    Port        int32
}
```

### 6. JoinGroup (API Key 11)

Join a consumer group and participate in rebalancing.

**Request Format:**
```go
type JoinGroupRequest struct {
    GroupID          string
    SessionTimeoutMs int32
    RebalanceTimeoutMs int32
    MemberID         string   // Empty on first join
    ProtocolType     string   // "consumer"
    Protocols        []GroupProtocol
}

type GroupProtocol struct {
    Name     string  // "range", "roundrobin", etc.
    Metadata []byte  // Subscribed topics
}
```

**Response Format:**
```go
type JoinGroupResponse struct {
    ErrorCode    int16
    GenerationID int32
    ProtocolName string
    Leader       string  // Leader member ID
    MemberID     string  // Assigned member ID
    Members      []GroupMember
}
```

### 7. SyncGroup (API Key 14)

Receive partition assignments from group leader.

### 8. Heartbeat (API Key 12)

Maintain group membership.

### 9. OffsetCommit (API Key 8)

Commit consumer offsets.

**Request Format:**
```go
type OffsetCommitRequest struct {
    GroupID      string
    GenerationID int32
    MemberID     string
    Topics       []OffsetCommitTopic
}

type OffsetCommitTopic struct {
    Topic      string
    Partitions []OffsetCommitPartition
}

type OffsetCommitPartition struct {
    Partition int32
    Offset    int64
    Metadata  *string
}
```

### 10. OffsetFetch (API Key 9)

Retrieve committed offsets.

### 11. LeaveGroup (API Key 13)

Leave a consumer group.

## Admin APIs

### 12. CreateTopics (API Key 19)

Create one or more topics.

**Request Format:**
```go
type CreateTopicsRequest struct {
    Topics       []CreatableTopic
    TimeoutMs    int32
    ValidateOnly bool  // Dry-run mode
}

type CreatableTopic struct {
    Name              string
    NumPartitions     int32
    ReplicationFactor int16
    Assignments       []CreatableReplicaAssignment
    Configs           []CreatableTopicConfig
}
```

**Example:**
```bash
kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic my-topic \
  --partitions 3 \
  --replication-factor 1
```

### 13. DeleteTopics (API Key 20)

Delete one or more topics.

**Example:**
```bash
kafka-topics.sh --bootstrap-server localhost:9092 \
  --delete \
  --topic old-topic
```

### 14. DescribeConfigs (API Key 32)

Query topic or broker configurations.

**Resource Types:**
- `2`: Topic
- `4`: Broker

**Example:**
```bash
kafka-configs.sh --bootstrap-server localhost:9092 \
  --describe \
  --topic my-topic
```

### 15. AlterConfigs (API Key 33)

Modify topic or broker configurations.

### 16. DeleteRecords (API Key 21)

Delete records up to a specified offset.

### 17. DescribeLogDirs (API Key 35)

Query log directory information.

## Transaction APIs

### 18. InitProducerID (API Key 22)

Initialize a transactional or idempotent producer.

**Request Format:**
```go
type InitProducerIDRequest struct {
    TransactionalID      *string
    TransactionTimeoutMs int32
}
```

**Response Format:**
```go
type InitProducerIDResponse struct {
    ErrorCode       int16
    ProducerID      int64   // Assigned producer ID
    ProducerEpoch   int16   // Epoch for fencing
}
```

### 19. AddPartitionsToTxn (API Key 24)

Add partitions to a transaction.

### 20. AddOffsetsToTxn (API Key 25)

Add consumer group offsets to a transaction.

### 21. EndTxn (API Key 26)

Commit or abort a transaction.

### 22. WriteTxnMarkers (API Key 27)

Write transaction markers (internal).

### 23. TxnOffsetCommit (API Key 28)

Commit offsets within a transaction.

## Authentication APIs

### 24. SaslHandshake (API Key 36)

Negotiate SASL mechanism.

**Supported Mechanisms:**
- PLAIN
- SCRAM-SHA-256
- SCRAM-SHA-512

### 25. SaslAuthenticate (API Key 37)

Exchange SASL authentication data.

## Error Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | None | Success |
| 1 | OffsetOutOfRange | Requested offset is out of range |
| 3 | UnknownTopicOrPartition | Topic or partition doesn't exist |
| 6 | NotLeaderForPartition | Request sent to non-leader |
| 10 | MessageTooLarge | Message exceeds max size |
| 14 | OffsetMetadataTooLarge | Offset metadata too large |
| 16 | GroupCoordinatorNotAvailable | Coordinator unavailable |
| 25 | InvalidCommitOffsetSize | Invalid offset size |
| 27 | NotCoordinator | Not the coordinator for this group |
| 36 | TopicAlreadyExists | Topic already exists |
| 42 | InvalidRequest | Invalid request parameters |

## Client Compatibility

### Tested Clients

| Client | Language | Version | Status |
|--------|----------|---------|--------|
| kafka-go | Go | v0.4.x | ✅ Full |
| sarama | Go | v1.38+ | ✅ Full |
| kafka-python | Python | v2.0+ | ✅ Full |
| KafkaJS | JavaScript | v2.2+ | ✅ Full |
| confluent-kafka-go | Go | v2.0+ | ✅ Full |

### Command-Line Tools

All standard Kafka command-line tools are compatible:
- `kafka-console-producer.sh`
- `kafka-console-consumer.sh`
- `kafka-topics.sh`
- `kafka-consumer-groups.sh`
- `kafka-configs.sh`

## Connection Flow

1. **Establish TCP Connection** to port 9092
2. **Send ApiVersions Request** to negotiate protocol versions
3. **Authenticate** (if SASL enabled) via SaslHandshake + SaslAuthenticate
4. **Send Application Requests** (Produce, Fetch, etc.)
5. **Close Connection** gracefully

## Performance Considerations

### Batching

- **Producer**: Batch messages to reduce request overhead
- **Consumer**: Use high `MaxWaitMs` and `MinBytes` for better throughput

### Compression

Supported codecs:
- None (0)
- GZIP (1)
- Snappy (2)
- LZ4 (3)
- ZSTD (4)

### Pipelining

Takhin supports request pipelining for improved throughput.

## Limitations

- **Max Message Size**: 1MB (default), configurable via `message.max.bytes`
- **Max Fetch Size**: 50MB (default)
- **Max Request Size**: 100MB (default)

## References

- [Apache Kafka Protocol Documentation](https://kafka.apache.org/protocol.html)
- [Kafka Wire Protocol](https://cwiki.apache.org/confluence/display/KAFKA/A+Guide+To+The+Kafka+Protocol)
- [Protocol Implementation](../../backend/pkg/kafka/protocol/)
- [Handler Implementation](../../backend/pkg/kafka/handler/)
