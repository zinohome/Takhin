# Kafka Transactions 设计文档

## 概述

Kafka Transactions 提供了跨多个分区的原子写入能力，确保一组消息要么全部成功，要么全部失败。这对于"精确一次语义"（Exactly-Once Semantics, EOS）至关重要。

## 核心概念

### 1. Transactional ID (事务 ID)
- 每个事务生产者有一个唯一的 `transactional.id`
- 用于识别和恢复生产者状态
- 支持幂等性和崩溃恢复

### 2. Producer ID (PID) 和 Epoch
- **Producer ID**: 由 Transaction Coordinator 分配的唯一 64 位整数
- **Epoch**: 生产者的版本号（16 位），用于检测僵尸生产者
- 每次生产者初始化或恢复时，Epoch 递增

### 3. Sequence Number (序列号)
- 每个消息有一个单调递增的序列号
- 用于检测重复消息和顺序
- 按 `(topicPartition, pid, epoch)` 分组

### 4. Transaction Coordinator
- 管理事务状态和元数据
- 协调两阶段提交协议
- 存储事务日志到内部 `__transaction_state` 主题

### 5. Transaction Marker (事务标记)
- `COMMIT` 标记: 事务成功提交
- `ABORT` 标记: 事务中止
- 写入到所有参与事务的分区

## 事务状态机

```
Empty
  |
  v
Ongoing (添加分区)
  |
  v
PrepareCommit / PrepareAbort (准备阶段)
  |
  v
CompleteCommit / CompleteAbort (完成阶段)
  |
  v
Empty (清理)
```

### 状态说明

1. **Empty**: 初始状态，没有进行中的事务
2. **Ongoing**: 事务进行中，生产者正在发送消息
3. **PrepareCommit**: 准备提交，等待所有分区确认
4. **PrepareAbort**: 准备中止，清理事务
5. **CompleteCommit**: 提交完成，写入 COMMIT 标记
6. **CompleteAbort**: 中止完成，写入 ABORT 标记

## Kafka Transaction APIs

### 1. InitProducerId (API Key 22)

**功能**: 初始化或恢复生产者 ID 和 Epoch

**请求结构**:
```go
type InitProducerIdRequest struct {
    Header              *RequestHeader
    TransactionalID     *string // 事务 ID (null = 仅幂等性)
    TransactionTimeout  int32   // 事务超时 (ms)
    ProducerID          int64   // 当前 PID (恢复时使用)
    ProducerEpoch       int16   // 当前 Epoch (恢复时使用)
}
```

**响应结构**:
```go
type InitProducerIdResponse struct {
    ThrottleTimeMs int32
    ErrorCode      ErrorCode
    ProducerID     int64  // 分配的 PID
    ProducerEpoch  int16  // 分配的 Epoch
}
```

**使用场景**:
- 生产者启动时
- 生产者崩溃恢复
- 事务超时后重新初始化

### 2. AddPartitionsToTxn (API Key 24)

**功能**: 将分区添加到当前事务

**请求结构**:
```go
type AddPartitionsToTxnRequest struct {
    Header          *RequestHeader
    TransactionalID string
    ProducerID      int64
    ProducerEpoch   int16
    Topics          []AddPartitionsTopic
}

type AddPartitionsTopic struct {
    Name       string
    Partitions []int32
}
```

**响应结构**:
```go
type AddPartitionsToTxnResponse struct {
    ThrottleTimeMs int32
    Results        []AddPartitionsTopicResult
}

type AddPartitionsTopicResult struct {
    Name       string
    Results    []AddPartitionsPartitionResult
}

type AddPartitionsPartitionResult struct {
    PartitionIndex int32
    ErrorCode      ErrorCode
}
```

**使用场景**:
- 第一次向分区发送事务消息
- 事务中动态添加新分区

### 3. AddOffsetsToTxn (API Key 25)

**功能**: 将消费者组的 offset 提交纳入事务（用于 consume-process-produce 模式）

**请求结构**:
```go
type AddOffsetsToTxnRequest struct {
    Header          *RequestHeader
    TransactionalID string
    ProducerID      int64
    ProducerEpoch   int16
    GroupID         string  // 消费者组 ID
}
```

**响应结构**:
```go
type AddOffsetsToTxnResponse struct {
    ThrottleTimeMs int32
    ErrorCode      ErrorCode
}
```

**使用场景**:
- 实现 Exactly-Once 消费-处理-生产流程
- 确保 offset 提交和消息生产的原子性

### 4. EndTxn (API Key 26)

**功能**: 提交或中止事务

**请求结构**:
```go
type EndTxnRequest struct {
    Header          *RequestHeader
    TransactionalID string
    ProducerID      int64
    ProducerEpoch   int16
    Committed       bool    // true = COMMIT, false = ABORT
}
```

**响应结构**:
```go
type EndTxnResponse struct {
    ThrottleTimeMs int32
    ErrorCode      ErrorCode
}
```

**使用场景**:
- 事务成功完成，提交所有消息
- 事务失败或超时，中止所有消息

### 5. TxnOffsetCommit (API Key 28)

**功能**: 在事务中提交消费者 offset

**请求结构**:
```go
type TxnOffsetCommitRequest struct {
    Header          *RequestHeader
    TransactionalID string
    GroupID         string
    ProducerID      int64
    ProducerEpoch   int16
    Topics          []TxnOffsetCommitTopic
}

type TxnOffsetCommitTopic struct {
    Name       string
    Partitions []TxnOffsetCommitPartition
}

type TxnOffsetCommitPartition struct {
    PartitionIndex int32
    Offset         int64
    LeaderEpoch    int32
    Metadata       *string
}
```

**响应结构**:
```go
type TxnOffsetCommitResponse struct {
    ThrottleTimeMs int32
    Topics         []TxnOffsetCommitTopicResult
}

type TxnOffsetCommitTopicResult struct {
    Name       string
    Partitions []TxnOffsetCommitPartitionResult
}

type TxnOffsetCommitPartitionResult struct {
    PartitionIndex int32
    ErrorCode      ErrorCode
}
```

## 实现架构

### 组件设计

```
┌─────────────────────────────────────────────┐
│          Transaction Coordinator            │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  Transaction Manager                │   │
│  │  - PID 分配                         │   │
│  │  - 事务状态管理                     │   │
│  │  - 两阶段提交协调                   │   │
│  └─────────────────────────────────────┘   │
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  Transaction Log                    │   │
│  │  - __transaction_state 主题         │   │
│  │  - 持久化事务元数据                 │   │
│  │  - 快照和恢复                       │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────────┐
│          Partition Log                      │
│                                             │
│  - Control Records (事务标记)               │
│  - Sequence Number 验证                     │
│  - PID/Epoch 验证                           │
└─────────────────────────────────────────────┘
```

### 数据结构

#### Transaction Metadata
```go
type TransactionMetadata struct {
    TransactionalID string
    ProducerID      int64
    ProducerEpoch   int16
    State           TransactionState
    Timeout         time.Duration
    StartTime       time.Time
    Partitions      map[string][]int32  // topic -> partitions
}

type TransactionState int8

const (
    TransactionStateEmpty TransactionState = iota
    TransactionStateOngoing
    TransactionStatePrepareCommit
    TransactionStatePrepareAbort
    TransactionStateCompleteCommit
    TransactionStateCompleteAbort
)
```

#### Producer ID Metadata
```go
type ProducerIDMetadata struct {
    ProducerID    int64
    Epoch         int16
    LastTimestamp time.Time
    Sequences     map[TopicPartition]int32  // 每个分区的最后序列号
}

type TopicPartition struct {
    Topic     string
    Partition int32
}
```

#### Control Record
```go
type ControlRecord struct {
    Type          ControlRecordType  // COMMIT or ABORT
    Version       int16
    CoordinatorEpoch int32
}

type ControlRecordType int16

const (
    ControlRecordCommit ControlRecordType = 0
    ControlRecordAbort  ControlRecordType = 1
)
```

## 实现步骤

### Phase 1: 基础设施 (Week 1-2)

**任务**:
1. **Transaction Coordinator 框架**
   - 创建 `pkg/transaction/coordinator.go`
   - Transaction Manager 基本结构
   - PID 生成器（使用 Raft 保证唯一性）

2. **数据结构定义**
   - TransactionMetadata
   - ProducerIDMetadata
   - TransactionState 枚举

3. **存储层扩展**
   - Control Record 支持
   - Sequence Number 存储
   - PID/Epoch 验证

**交付物**:
- Transaction Coordinator 基本框架
- 数据结构定义完成
- 单元测试

### Phase 2: InitProducerId API (Week 2-3)

**任务**:
1. **协议实现**
   - `pkg/kafka/protocol/init_producer_id.go`
   - Request/Response 编解码

2. **Handler 实现**
   - handleInitProducerId()
   - PID 分配逻辑
   - Epoch 递增逻辑

3. **测试**
   - 新生产者初始化
   - 生产者恢复
   - 并发 PID 分配

**交付物**:
- InitProducerId API 完整实现
- 测试覆盖率 ≥ 80%

### Phase 3: AddPartitionsToTxn API (Week 3-4)

**任务**:
1. **协议实现**
   - `pkg/kafka/protocol/add_partitions_to_txn.go`
   - Request/Response 编解码

2. **Transaction Manager**
   - 事务状态转换 (Empty → Ongoing)
   - 分区跟踪
   - 超时检测

3. **测试**
   - 单分区事务
   - 多分区事务
   - 动态添加分区

**交付物**:
- AddPartitionsToTxn API 完整实现
- 事务状态管理

### Phase 4: EndTxn API 和两阶段提交 (Week 4-6)

**任务**:
1. **协议实现**
   - `pkg/kafka/protocol/end_txn.go`
   - Request/Response 编解码

2. **两阶段提交**
   - Prepare 阶段: 状态转换，通知所有分区
   - Commit/Abort 阶段: 写入 Control Record
   - 清理: 移除事务元数据

3. **Control Record**
   - Record Batch 标记
   - COMMIT/ABORT Marker
   - 消费者过滤逻辑

4. **测试**
   - 成功提交事务
   - 事务中止
   - 部分失败处理
   - 崩溃恢复

**交付物**:
- EndTxn API 完整实现
- 两阶段提交完成
- Control Record 支持

### Phase 5: AddOffsetsToTxn 和 TxnOffsetCommit (Week 6-7)

**任务**:
1. **协议实现**
   - `pkg/kafka/protocol/add_offsets_to_txn.go`
   - `pkg/kafka/protocol/txn_offset_commit.go`

2. **Coordinator 集成**
   - 将 Consumer Group Coordinator 集成到事务
   - Offset 提交原子性

3. **测试**
   - Exactly-Once 消费-处理-生产流程
   - Offset 提交和消息生产原子性

**交付物**:
- AddOffsetsToTxn API
- TxnOffsetCommit API
- E2E 事务测试

### Phase 6: Raft 集成和持久化 (Week 7-8)

**任务**:
1. **Transaction Log**
   - FSM 扩展支持事务命令
   - Transaction State 持久化到 Raft
   - 快照和恢复

2. **高可用性**
   - Leader 故障转移
   - 事务恢复
   - PID 一致性

3. **性能优化**
   - 批量提交
   - 异步写入
   - 缓存优化

**交付物**:
- Raft 集成完成
- 事务持久化
- 性能基准测试

## 错误处理

### 常见错误码

| 错误码 | 名称 | 说明 |
|-------|------|------|
| 45 | OutOfOrderSequenceNumber | 序列号不连续 |
| 46 | DuplicateSequenceNumber | 重复序列号 |
| 47 | InvalidProducerEpoch | Epoch 过期 |
| 48 | InvalidTxnState | 事务状态无效 |
| 49 | InvalidProducerIDMapping | PID 映射无效 |
| 50 | InvalidTransactionTimeout | 超时参数无效 |
| 51 | ConcurrentTransactions | 并发事务冲突 |
| 52 | TransactionCoordinatorFenced | Coordinator 被隔离 |

### 错误场景

1. **生产者崩溃**: 使用 transactional.id 恢复，Epoch 递增
2. **事务超时**: 自动中止，清理元数据
3. **网络分区**: 使用 Epoch 检测僵尸生产者
4. **序列号跳跃**: 拒绝消息，返回错误
5. **Coordinator 故障**: Leader 切换，从 Raft 恢复状态

## 性能考虑

### 优化策略

1. **批量操作**: 多个分区的标记批量写入
2. **异步提交**: Prepare 和 Complete 阶段异步化
3. **缓存**: Transaction Metadata 内存缓存
4. **索引**: PID → TransactionalID 映射索引
5. **并发**: 多事务并行处理

### 性能目标

- **吞吐量**: 支持 10K+ 事务/秒
- **延迟**: P99 < 50ms (提交延迟)
- **内存**: < 100MB (10K 活跃事务)
- **恢复时间**: < 5 秒 (Leader 切换)

## 测试策略

### 单元测试
- Transaction Coordinator 各组件
- 状态转换逻辑
- PID 分配和 Epoch 管理
- Sequence Number 验证

### 集成测试
- 完整事务流程 (InitProducerId → AddPartitions → Produce → EndTxn)
- 多分区事务
- Exactly-Once 消费-处理-生产
- 事务超时和中止

### 故障测试
- 生产者崩溃恢复
- Coordinator 故障转移
- 网络分区和僵尸生产者
- 并发事务冲突

### 性能测试
- 吞吐量基准测试
- 延迟分布测试
- 内存占用测试
- 长时间稳定性测试

## 兼容性

### Kafka 客户端
- Java Producer (transactions enabled)
- librdkafka (transactional producer)
- Go kafka-go (未来支持)

### 协议版本
- InitProducerId: v0+
- AddPartitionsToTxn: v0+
- EndTxn: v0+
- AddOffsetsToTxn: v0+
- TxnOffsetCommit: v0+

## 参考资料

- [Kafka Transactions Design](https://kafka.apache.org/documentation/#semantics)
- [KIP-98: Exactly Once Delivery and Transactional Messaging](https://cwiki.apache.org/confluence/display/KAFKA/KIP-98+-+Exactly+Once+Delivery+and+Transactional+Messaging)
- [KIP-129: Streams Exactly-Once Semantics](https://cwiki.apache.org/confluence/display/KAFKA/KIP-129%3A+Streams+Exactly-Once+Semantics)
- [Transactions in Apache Kafka](https://www.confluent.io/blog/transactions-apache-kafka/)

## 风险和挑战

### 技术风险

1. **复杂性**: 事务是 Kafka 最复杂的特性
2. **状态管理**: 大量事务元数据需要高效管理
3. **性能影响**: 两阶段提交增加延迟
4. **Raft 集成**: Transaction Log 需要强一致性

### 缓解策略

1. **分阶段实现**: 按 API 逐步实现和测试
2. **充分测试**: 覆盖各种故障场景
3. **性能优化**: 提前考虑性能优化策略
4. **参考实现**: 借鉴 Kafka 官方实现

## 里程碑

| 里程碑 | 时间 | 交付物 |
|--------|------|--------|
| M1: 基础设施 | Week 1-2 | Transaction Coordinator 框架 |
| M2: InitProducerId | Week 2-3 | InitProducerId API |
| M3: AddPartitions | Week 3-4 | AddPartitionsToTxn API |
| M4: EndTxn | Week 4-6 | 两阶段提交完成 |
| M5: Offsets | Week 6-7 | AddOffsetsToTxn + TxnOffsetCommit |
| M6: 持久化 | Week 7-8 | Raft 集成，生产就绪 |

## 结论

Kafka Transactions 是一个复杂但关键的特性，为 Takhin 提供 Exactly-Once 语义。建议：

1. **独立 Sprint**: 将 Transactions 作为 Sprint 11-12 的唯一目标
2. **充分准备**: 深入理解 Kafka 事务机制
3. **分阶段实施**: 按照上述 6 个 Phase 逐步实现
4. **充分测试**: 确保各种故障场景下的正确性

鉴于 Transactions 的复杂性，建议先完成 Phase 3（Console 开发）或其他较简单的特性，积累更多经验后再实施 Transactions。
