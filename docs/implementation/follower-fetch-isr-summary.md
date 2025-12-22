# Follower Fetch and ISR Management Implementation Summary

## 概述
实现了 Takhin 副本系统的核心功能：Follower Fetch 机制和 ISR（In-Sync Replicas）动态管理。

## 已完成的工作

### 1. 协议层增强
**文件**: `backend/pkg/kafka/protocol/fetch.go`
- 在 `FetchRequest` 结构体中添加 `ReplicaID` 字段
  - ReplicaID = -1：消费者 fetch 请求
  - ReplicaID >= 0：Follower 副本 fetch 请求（broker ID）
- 更新 `DecodeFetchRequest` 以正确读取 ReplicaID 字段
- 符合 Kafka 协议规范

### 2. Topic Metadata 扩展
**文件**: `backend/pkg/storage/topic/manager.go`

#### 新增字段
```go
type Topic struct {
    // 跟踪每个 Follower 的 Log End Offset
    FollowerLEO map[int32]map[int32]int64  // partition -> broker -> LEO
    
    // 跟踪每个 Follower 的最后 fetch 时间
    LastFetchTime map[int32]map[int32]time.Time  // partition -> broker -> time
    
    // ISR 超时阈值（默认 10 秒）
    ReplicaLagMaxMs int64
}
```

#### 新增方法
- `UpdateFollowerLEO(partitionID, followerID, leo)`: 更新 follower LEO 和 fetch 时间
- `GetFollowerLEO(partitionID, followerID)`: 获取 follower LEO
- `UpdateISR(partitionID, leaderLEO)`: 基于 lag 动态更新 ISR
- `SetISR(partitionID, isr)`: 手动设置 ISR（用于测试）
- `GetLeaderForPartition(partitionID)`: 获取分区 leader（第一个副本）

#### ISR 管理逻辑
Follower 加入 ISR 的条件（**同时满足**）：
1. **LEO 同步**: `leaderLEO - followerLEO <= 1`
2. **Fetch 活跃**: `now - lastFetchTime <= ReplicaLagMaxMs`

### 3. Handler 集成
**文件**: `backend/pkg/kafka/handler/handler.go`

#### handleFetch 增强
```go
// 检测副本 fetch 请求
if req.ReplicaID >= 0 {
    // 更新 follower LEO
    topic.UpdateFollowerLEO(partition, req.ReplicaID, fetchOffset)
    
    // 触发 ISR 更新
    topic.UpdateISR(partition, hwm)
}
```

#### handleProduce 增强（acks=-1 支持）
```go
if produceReq.Acks == -1 {
    // 获取当前 ISR
    isr := topic.GetISR(partition)
    
    // 记录 ISR 状态（用于监控）
    logger.Info("produce with acks=-1", 
        "isr_size", len(isr), 
        "replication_factor", topic.ReplicationFactor)
}
```

### 4. 测试覆盖
**文件**: `backend/pkg/kafka/handler/replication_test.go`

#### 测试场景
1. **TestFollowerLEOTracking**: 基础 LEO 跟踪功能
   - 更新 follower LEO
   - 查询 follower LEO
   - 处理不存在的 follower

2. **TestISRManagement**: ISR 扩展和收缩
   - Follower 追上 leader → 加入 ISR
   - Follower 落后 → 不加入 ISR

3. **TestISRShrinkOnTimeout**: 基于超时的 ISR 收缩
   - Follower 长时间未 fetch → 从 ISR 移除
   - 100ms 超时测试

4. **TestGetLeaderForPartition**: Leader 查询
   - 正确返回第一个副本作为 leader
   - 处理不存在的分区

5. **TestMultipleFollowers**: 多 follower 场景
   - 同时跟踪 5 个 follower
   - 不同 lag 状态的 follower 正确分类

#### 测试结果
```
PASS: TestFollowerLEOTracking (0.00s)
PASS: TestISRManagement (0.00s)
PASS: TestISRShrinkOnTimeout (0.15s)
PASS: TestGetLeaderForPartition (0.00s)
PASS: TestMultipleFollowers (0.00s)
ok   github.com/takhin-data/takhin/pkg/kafka/handler 0.165s
```

## 技术细节

### ISR 更新算法
```go
for each follower in replicas:
    hasLEO = (leaderLEO - followerLEO <= 1)
    hasFetch = (now - lastFetchTime <= ReplicaLagMaxMs)
    
    if hasLEO AND hasFetch:
        add to ISR
```

### Follower Fetch 流程
```
1. Follower 发送 FetchRequest (ReplicaID = broker_id)
2. Leader 处理请求，读取日志
3. Leader 调用 UpdateFollowerLEO(partition, broker_id, fetch_offset)
4. Leader 调用 UpdateISR(partition, leader_leo) 更新 ISR
5. Leader 返回 FetchResponse 携带数据和 HWM
```

### HWM（High Water Mark）计算
```
HWM = min(LEO of all replicas in ISR)
```
消费者只能读取到 HWM 之前的数据，确保数据已被足够多的副本确认。

## 与现有系统的集成

### 1. Metadata API
- `handleMetadata` 已经从 `Topic.GetISR()` 读取 ISR
- ISR 变化会自动反映在 Metadata 响应中

### 2. Produce API
- acks=1: 只等待 leader 确认（现有行为）
- acks=-1: 等待所有 ISR 副本确认（已添加日志，实际等待逻辑待后续实现）

### 3. Replication Metadata
- 与 `backend/pkg/replication/assigner.go` 的副本分配兼容
- Topic 创建时初始化 FollowerLEO/LastFetchTime map

## 下一步工作（按优先级）

### 优先级 2: 副本分配持久化
- [ ] 将 Replicas、ISR、FollowerLEO、LastFetchTime 序列化到磁盘
- [ ] Broker 重启后恢复副本状态
- [ ] 实现元数据文件格式（JSON 或 Protobuf）

### 优先级 3: Metadata 动态更新（已部分完成）
- [x] GetISR 集成到 handleMetadata
- [ ] 测试 Metadata ISR 反映实时变化

### 优先级 4: 多 Broker 副本分配
- [ ] ReplicaAssigner 支持从配置读取多个 broker ID
- [ ] CreateTopics 时使用多 broker 分配
- [ ] Leader 选举机制（controller 角色）

### acks=-1 完整实现
- [ ] Produce 时等待所有 ISR 副本确认
- [ ] 超时处理（request.timeout.ms）
- [ ] 错误码：NotEnoughReplicas, NotEnoughReplicasAfterAppend

### 监控和可观测性
- [ ] 添加 Prometheus metrics: isr_size, replica_lag_ms
- [ ] ISR 变化事件日志
- [ ] Under-replicated partition 告警

## 验证和测试

### 单元测试
- ✅ Follower LEO 跟踪
- ✅ ISR 动态管理
- ✅ 超时驱动的 ISR 收缩
- ✅ Leader 查询
- ✅ 多 follower 场景

### 集成测试（待添加）
- [ ] 真实 Produce + Follower Fetch 端到端
- [ ] acks=-1 等待 ISR 确认
- [ ] Follower 追赶和 ISR 扩展
- [ ] 网络分区场景

### 性能测试（待添加）
- [ ] Follower Fetch 吞吐量
- [ ] ISR 更新开销
- [ ] 大量副本场景（100+ follower）

## 相关文件
- `backend/pkg/kafka/protocol/fetch.go` - 协议定义
- `backend/pkg/storage/topic/manager.go` - Topic 和 ISR 管理
- `backend/pkg/kafka/handler/handler.go` - Fetch 和 Produce 处理
- `backend/pkg/kafka/handler/replication_test.go` - 测试
- `docs/implementation/project-plan.md` - 项目计划

## 变更日志
- 2025-12-22: 初始实现 Follower Fetch 和 ISR 管理
  - 添加 ReplicaID 到 FetchRequest
  - 实现 UpdateFollowerLEO 和 UpdateISR
  - 完整测试覆盖
