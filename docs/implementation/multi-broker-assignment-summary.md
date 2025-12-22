# 多 Broker 副本分配实现总结

## 概述
完成了 Priority 4：扩展 ReplicaAssigner 支持多 broker 环境，实现真实集群的副本分布策略。

## 实施时间
- 开始时间：2025-12-22
- 完成时间：2025-12-22
- 用时：约 1 小时

## 已实现功能

### 1. 配置系统扩展
#### 文件：`backend/pkg/config/config.go`
- **新增字段**：`KafkaConfig.ClusterBrokers []int` - 集群中所有 broker ID 列表
- **配置验证**：确保当前 broker ID 在集群列表中
- **默认行为**：未配置时自动回退到单 broker 模式

#### 配置示例（takhin.yaml）
```yaml
kafka:
  broker:
    id: 1
  cluster:
    brokers: [1, 2, 3, 4, 5]  # 多 broker 集群
```

### 2. Handler 架构改进
#### 文件：`backend/pkg/kafka/handler/handler.go`
- **ReplicaAssigner 字段**：Handler 结构新增 `replicaAssigner` 字段，避免重复创建
- **buildBrokerList() 函数**：
  - 优先使用 `ClusterBrokers` 配置
  - 未配置时回退到当前 broker ID（单 broker 模式）
  - 在 `New()` 和 `NewWithBackend()` 中调用
- **CreateTopics 简化**：移除每次创建 assigner 的逻辑，直接使用 `h.replicaAssigner`

#### 核心逻辑
```go
// buildBrokerList constructs the broker list for replica assignment
func buildBrokerList(cfg *config.Config) []int32 {
    if len(cfg.Kafka.ClusterBrokers) > 0 {
        brokers := make([]int32, len(cfg.Kafka.ClusterBrokers))
        for i, brokerID := range cfg.Kafka.ClusterBrokers {
            brokers[i] = int32(brokerID)
        }
        return brokers
    }
    return []int32{int32(cfg.Kafka.BrokerID)}
}
```

### 3. ReplicaAssigner 已有功能
#### 文件：`backend/pkg/replication/assigner.go`（无需修改）
已支持多 broker 的核心功能：
- **Round-Robin 算法**：Leader 均衡分布到所有 broker
- **副本分散**：确保副本不重复，分布在不同 broker
- **参数验证**：
  - 检查 replication factor 不超过 broker 数量
  - 验证无重复副本
  - 确保所有分区都有分配

### 4. 测试覆盖
#### 文件：`backend/pkg/kafka/handler/multi_broker_test.go`（新增）
共 4 个测试用例，全部通过：

**TestMultiBrokerReplicaAssignment**
- 测试多种集群配置：
  - 3 brokers, 3 partitions, RF=3
  - 3 brokers, 6 partitions, RF=2
  - 5 brokers, 10 partitions, RF=3
  - 单 broker, 3 partitions, RF=1
- 验证 leader 分布符合 round-robin
- 验证无重复副本
- 验证所有副本都在集群中

**TestReplicaDistribution**
- 大规模测试：5 brokers, 50 partitions, RF=3
- 验证 leader 分布均衡：每个 broker 10 个 leader（50/5）
- 验证副本分布均衡：每个 broker 30 个副本（150/5）
- 容忍度：leader ±20%，replica ±10%

**TestBuildBrokerList**
- 测试配置解析逻辑
- 验证多 broker 模式
- 验证单 broker 回退

**TestReplicationFactorExceedsBrokers**
- 测试错误处理：RF > broker 数量
- 验证优雅失败（记录错误日志但不崩溃）

#### 文件：`backend/pkg/config/cluster_brokers_test.go`（新增）
共 2 个测试用例，全部通过：

**TestClusterBrokersValidation**
- 验证当前 broker 必须在集群列表中
- 验证空列表允许（单 broker 模式）
- 验证配置错误时返回清晰错误信息

**TestClusterBrokersDefault**
- 验证默认值处理（不修改用户配置）

### 5. 测试结果
```bash
# 多 broker 副本分配测试
$ go test -v -run TestMultiBrokerReplicaAssignment ./pkg/kafka/handler/
✓ 3 brokers, 3 partitions, RF=3 (0.00s)
✓ 3 brokers, 6 partitions, RF=2 (0.00s)
✓ 5 brokers, 10 partitions, RF=3 (0.00s)
✓ single broker, 3 partitions, RF=1 (0.00s)

# 副本分布均衡性测试
$ go test -v -run TestReplicaDistribution ./pkg/kafka/handler/
Leader distribution: map[1:10 2:10 3:10 4:10 5:10]    ← 完全均衡！
Replica distribution: map[1:30 2:30 3:30 4:30 5:30]   ← 完全均衡！
✓ PASS (0.04s)

# 配置验证测试
$ go test -v ./pkg/config/...
✓ TestClusterBrokersValidation (0.00s)
✓ TestClusterBrokersDefault (0.00s)

# 完整回归测试
$ go test ./pkg/kafka/handler/...
ok  github.com/takhin-data/takhin/pkg/kafka/handler  0.285s
```

## 技术细节

### 1. Round-Robin 分配算法
```go
// Example: 3 brokers [1,2,3], 3 partitions, RF=3
// Partition 0: [1, 2, 3] (leader=1)
// Partition 1: [2, 3, 1] (leader=2)
// Partition 2: [3, 1, 2] (leader=3)

for partitionID := 0; partitionID < numPartitions; partitionID++ {
    startIndex := partitionID % len(brokers)
    replicas := []int32{}
    for i := 0; i < replicationFactor; i++ {
        brokerIndex := (startIndex + i) % len(brokers)
        replicas = append(replicas, brokers[brokerIndex])
    }
    assignments[partitionID] = replicas
}
```

### 2. 副本分布特性
- **Leader 均衡**：每个 broker 承担接近的 leader 数量
- **副本分散**：避免单点故障，副本分布在不同 broker
- **确定性**：相同参数总是产生相同分配（便于测试和调试）
- **无状态**：不依赖集群运行时状态，配置驱动

### 3. 配置优先级
1. **ClusterBrokers 配置**：显式指定集群成员
2. **单 broker 回退**：未配置时使用当前 broker ID
3. **未来扩展**：可从 Raft 集群动态获取

### 4. 错误处理
- **RF > brokers**：记录 ERROR 日志但不阻止 topic 创建
- **配置验证失败**：启动时立即失败，防止错误配置
- **空 broker 列表**：返回友好错误信息

## 与已实现功能的集成

### 1. Metadata Persistence Integration
- 副本分配自动持久化到 `metadata.json`
- `Replicas` 字段包含完整 broker 列表
- 重启后从元数据恢复副本分配

### 2. ISR Management Integration
- 初始 ISR = Replicas（所有副本同步）
- Follower Fetch 更新 ISR 时自动持久化
- Metadata API 反映多 broker 的 ISR 状态

### 3. Follower Fetch Integration
- Follower 可以是不同 broker 上的副本
- Leader 追踪多个 Follower 的 LEO
- HWM 计算考虑所有 broker 的副本

## 配置示例

### 单 Broker 模式（默认）
```yaml
kafka:
  broker:
    id: 1
  # cluster.brokers 未配置，自动使用单 broker 模式
```

### 3-Broker 集群
```yaml
# Broker 1 配置
kafka:
  broker:
    id: 1
  cluster:
    brokers: [1, 2, 3]

# Broker 2 配置
kafka:
  broker:
    id: 2
  cluster:
    brokers: [1, 2, 3]

# Broker 3 配置
kafka:
  broker:
    id: 3
  cluster:
    brokers: [1, 2, 3]
```

### 环境变量覆盖
```bash
# 设置 broker ID
export TAKHIN_KAFKA_BROKER_ID=2

# 设置集群成员（逗号分隔）
export TAKHIN_KAFKA_CLUSTER_BROKERS="1,2,3,4,5"
```

## 已知限制与未来优化

### 1. 动态 Broker 发现（待实现）
当前实现需要手动配置 broker 列表：
- 优化方向：从 Raft 集群动态获取成员
- 收益：自动适应 broker 加入/退出
- 实施：集成 Raft membership API

### 2. 机架感知（Rack Awareness）
当前算法不考虑机架拓扑：
- 优化方向：优先分配副本到不同机架
- 收益：提升跨机架故障容忍度
- 实施：扩展配置增加 `rack` 字段

### 3. 自定义分配策略
当前仅支持 round-robin：
- 优化方向：支持 range、sticky 等策略
- 收益：灵活适配不同业务需求
- 实施：ReplicaAssigner 接口化

### 4. 副本迁移（Replica Reassignment）
当前副本分配是静态的：
- 优化方向：AlterConfigs 支持动态修改副本分配
- 收益：负载均衡、broker 下线支持
- 实施：实现副本同步和切换逻辑

## 项目进度更新

### 完成的优先级任务
| 优先级 | 任务 | 状态 | 完成时间 |
|--------|------|------|----------|
| P1 | Follower Fetch & ISR 更新 | ✅ 完成 | 2025-12-XX |
| P2 | 副本元数据持久化 | ✅ 完成 | 2025-12-XX |
| P3 | Metadata ISR 动态反映 | ✅ 完成 | 2025-12-XX |
| **P4** | **多 Broker 副本分配** | **✅ 完成** | **2025-12-22** |

### Sprint 16 完整成果
- ✅ 副本复制基础架构
- ✅ ISR 管理和 HWM 计算
- ✅ Follower Fetch 机制
- ✅ 副本元数据持久化
- ✅ Metadata 动态 ISR 反映
- ✅ **多 Broker 副本分配**
- ✅ Round-Robin 副本分配算法
- ✅ Replication 配置系统扩展

### 下一步计划：Producer acks=-1 实现
**目标**：实现 Producer 等待 ISR 所有副本确认的机制

**任务分解**：
1. Produce Handler 扩展：
   - 检查 acks 参数（-1/all, 0, 1）
   - acks=-1 时等待 ISR 所有副本确认
   - 超时处理（request.timeout.ms）

2. 副本确认追踪：
   - Leader 追踪每个副本的确认状态
   - Follower Fetch 时更新确认位置
   - HWM 更新后通知 Producer

3. 错误处理：
   - NotEnoughReplicas：ISR 小于 min.isr
   - NotEnoughReplicasAfterAppend：写入后 ISR 不足
   - Timeout：等待确认超时

4. 测试：
   - 单副本 acks=-1（立即返回）
   - 多副本 acks=-1（等待 ISR）
   - 超时场景
   - ISR 收缩场景

## 测试命令

### 运行多 broker 副本分配测试
```bash
cd backend
go test -v -run TestMultiBroker ./pkg/kafka/handler/
```

### 运行配置验证测试
```bash
cd backend
go test -v ./pkg/config/...
```

### 运行完整测试套件
```bash
cd backend
go test ./pkg/...
```

## 参考文档
- [ReplicaAssigner 实现](../backend/pkg/replication/assigner.go)
- [Handler 集成](../backend/pkg/kafka/handler/handler.go#L70-L85)
- [配置系统](../backend/pkg/config/config.go#L32-L42)
- [多 broker 测试](../backend/pkg/kafka/handler/multi_broker_test.go)
- [Kafka Replication Protocol](https://kafka.apache.org/documentation/#replication)
