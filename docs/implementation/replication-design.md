# 副本复制机制设计文档

## 1. 概述

本文档描述 Takhin 副本复制机制的设计和实现，这是 Kafka 兼容性和生产环境高可用的核心功能。

## 2. 架构设计

### 2.1 核心概念

```
Topic
  └── Partition (多个)
        ├── Leader Replica (1个) - 处理读写请求
        └── Follower Replicas (N-1个) - 从 Leader 同步数据
```

**关键组件**：
- **Replica**: 分区的副本，包含完整的 Log 数据
- **Leader**: 处理所有读写请求的副本
- **Follower**: 从 Leader 复制数据的副本
- **ISR (In-Sync Replicas)**: 与 Leader 保持同步的副本集合
- **HWM (High Water Mark)**: 所有 ISR 副本都已复制的最高偏移量

### 2.2 数据结构设计

```go
// Partition 表示一个分区及其所有副本
type Partition struct {
    TopicName     string
    PartitionID   int32
    Leader        int32              // Leader broker ID
    Replicas      []int32            // 所有副本的 broker IDs
    ISR           []int32            // In-Sync Replicas
    Log           *log.Log           // 本地 Log（如果本节点是副本之一）
    HWM           int64              // High Water Mark
    LEO           int64              // Log End Offset
    mu            sync.RWMutex
}

// ReplicaManager 管理本节点的所有副本
type ReplicaManager struct {
    BrokerID      int32
    Partitions    map[string]*Partition  // "topic-partition" -> Partition
    mu            sync.RWMutex
}
```

### 2.3 副本分配策略

**Round-Robin 分配算法**（初始实现）：
```
Topic: test-topic, Partitions: 3, Replication Factor: 3, Brokers: [1, 2, 3]

Partition 0: Leader=1, Replicas=[1, 2, 3]
Partition 1: Leader=2, Replicas=[2, 3, 1]
Partition 2: Leader=3, Replicas=[3, 1, 2]
```

## 3. 副本同步协议

### 3.1 Follower 同步流程

```
1. Follower 定期发送 Fetch 请求到 Leader
2. Fetch 请求包含：
   - replica_id: Follower 的 broker ID
   - max_wait_ms: 最大等待时间
   - fetch_offset: Follower 的下一个期望偏移量
3. Leader 返回数据（如果有）
4. Follower 写入本地 Log
5. Follower 更新 LEO（Log End Offset）
6. Leader 根据 Follower LEO 更新 ISR 和 HWM
```

### 3.2 ISR 管理

**加入 ISR 条件**：
- 副本的 LEO >= HWM
- 副本在 `replica.lag.time.max.ms` 内保持同步（默认 10 秒）

**移出 ISR 条件**：
- 副本的 LEO 落后 HWM 超过 `replica.lag.time.max.ms`
- 副本失联或故障

### 3.3 High Water Mark (HWM)

HWM 是所有 ISR 副本都已复制的最高偏移量：

```go
func (p *Partition) UpdateHWM() {
    minLEO := p.LEO  // Leader's LEO
    
    for _, replicaID := range p.ISR {
        if replicaID == p.Leader {
            continue
        }
        followerLEO := p.getFollowerLEO(replicaID)
        if followerLEO < minLEO {
            minLEO = followerLEO
        }
    }
    
    p.HWM = minLEO
}
```

**HWM 的作用**：
- 消费者只能读取 HWM 之前的消息（已被所有 ISR 副本确认）
- 保证数据一致性

## 4. 生产者 ACKs 语义

```go
// acks = 0: 不等待任何确认（最快，可能丢失数据）
// acks = 1: 等待 Leader 写入确认（默认）
// acks = -1 (all): 等待所有 ISR 副本写入确认（最安全）
```

**实现流程**：
```
1. Producer 发送消息到 Leader
2. Leader 写入本地 Log，更新 LEO
3. 根据 acks 配置：
   - acks = 0: 立即返回
   - acks = 1: Leader 写入后返回
   - acks = -1: 等待所有 ISR 副本同步后返回（或超时）
```

## 5. 实现计划

### Phase 1: 基础副本结构（2天）
- [ ] 创建 Partition 和 ReplicaManager 数据结构
- [ ] 实现 Round-Robin 副本分配算法
- [ ] 集成到 CreateTopics 处理器

### Phase 2: Leader/Follower 管理（2天）
- [ ] 实现 Leader 选举逻辑（简化版：选择第一个副本）
- [ ] 实现 Follower 识别和角色管理
- [ ] 添加副本元数据到 Metadata 响应

### Phase 3: ISR 管理（2天）
- [ ] 实现 ISR 集合管理
- [ ] 实现 ISR 更新逻辑（添加/移除副本）
- [ ] 实现副本滞后检测

### Phase 4: 副本同步（3天）
- [ ] 实现 Follower Fetch 请求处理
- [ ] 实现 Follower 后台同步任务
- [ ] 实现 Leader 副本状态跟踪

### Phase 5: HWM 和 ACKs（2天）
- [ ] 实现 HWM 计算和更新
- [ ] 实现 Producer ACKs 语义（acks=1 和 acks=-1）
- [ ] 修改 Produce 处理器支持等待 ISR 同步

### Phase 6: 测试和优化（2天）
- [ ] 单元测试：副本分配、ISR 管理、HWM 计算
- [ ] 集成测试：副本同步、故障恢复
- [ ] 性能测试和优化

## 6. 配置参数

```yaml
replication:
  # 默认副本因子
  default_replication_factor: 1
  
  # 副本滞后阈值（毫秒）
  replica_lag_time_max_ms: 10000
  
  # Follower 同步间隔（毫秒）
  replica_fetch_wait_max_ms: 500
  
  # Follower Fetch 批量大小
  replica_fetch_max_bytes: 1048576  # 1MB
```

## 7. 与 Raft 的关系

**Raft vs Replication**：
- **Raft**: 管理集群元数据（topic配置、分区分配）的一致性
- **Replication**: 管理分区数据的副本复制

两者配合工作：
1. Raft 存储副本分配信息（哪些节点有哪些副本）
2. Replication 根据 Raft 提供的信息进行数据复制
3. Leader 变更通过 Raft 通知所有节点

## 8. 限制和未来改进

**当前限制**：
- 不支持动态副本迁移
- Leader 选举简化（选择第一个副本，未来集成 Raft）
- 不支持优先副本（preferred replica）
- 不支持副本限流（replica throttling）

**未来改进**：
- 自动 Leader 均衡
- 副本迁移工具
- 跨机架感知（rack awareness）
- 智能副本分配策略

## 9. 参考资料

- [Kafka Replication Design](https://kafka.apache.org/documentation/#replication)
- [Kafka Protocol - Fetch Request](https://kafka.apache.org/protocol.html#The_Messages_Fetch)
- [Kafka ISR Management](https://kafka.apache.org/documentation/#design_replicatedevents)
