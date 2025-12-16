# Kafka Admin API 实现

## 概述

Takhin 现已实现完整的 Kafka Admin API 支持，包括主题创建、删除和配置查询功能。这些 API 使得可以通过标准 Kafka 客户端工具（如 `kafka-topics.sh` 和 `kafka-configs.sh`）管理 Takhin 集群。

## 已实现的 API

### 1. CreateTopics (API Key 19)

**功能**: 创建一个或多个主题

**特性**:
- 支持配置分区数和副本因子
- 支持自定义分区副本分配
- 支持主题配置参数
- 支持验证模式（validate-only），只验证不实际创建

**请求结构**:
```go
type CreateTopicsRequest struct {
    Header       *RequestHeader
    Topics       []CreatableTopic
    TimeoutMs    int32
    ValidateOnly bool
}

type CreatableTopic struct {
    Name              string
    NumPartitions     int32
    ReplicationFactor int16
    Assignments       []CreatableReplicaAssignment
    Configs           []CreatableTopicConfig
}
```

**响应结构**:
```go
type CreateTopicsResponse struct {
    ThrottleTimeMs int32
    TopicResults   []CreatableTopicResult
}

type CreatableTopicResult struct {
    Name              string
    ErrorCode         ErrorCode
    ErrorMessage      *string
    NumPartitions     int32
    ReplicationFactor int16
    Configs           []CreatableTopicConfig
}
```

**错误码**:
- `0` (None): 成功
- `36` (TopicAlreadyExists): 主题已存在
- `42` (InvalidRequest): 请求参数无效

**使用示例**:
```bash
# 使用 kafka-topics.sh 创建主题
kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic my-topic \
  --partitions 3 \
  --replication-factor 1

# 使用验证模式
kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic test-topic \
  --partitions 2 \
  --dry-run
```

### 2. DeleteTopics (API Key 20)

**功能**: 删除一个或多个主题

**特性**:
- 批量删除主题
- 支持超时设置
- 删除主题数据和元数据
- 支持 Raft 共识（分布式模式）

**请求结构**:
```go
type DeleteTopicsRequest struct {
    Header     *RequestHeader
    TopicNames []string
    TimeoutMs  int32
}
```

**响应结构**:
```go
type DeleteTopicsResponse struct {
    ThrottleTimeMs int32
    Results        []DeletableTopicResult
}

type DeletableTopicResult struct {
    Name         string
    ErrorCode    ErrorCode
    ErrorMessage *string
}
```

**错误码**:
- `0` (None): 成功
- `3` (UnknownTopicOrPartition): 主题不存在
- `42` (InvalidRequest): 请求参数无效

**实现细节**:
- DirectBackend: 直接删除主题目录和元数据
- RaftBackend: 通过 Raft 共识协议确保分布式一致性
- 文件系统: 使用 `os.RemoveAll()` 清理主题目录

**使用示例**:
```bash
# 使用 kafka-topics.sh 删除主题
kafka-topics.sh --bootstrap-server localhost:9092 \
  --delete \
  --topic old-topic

# 批量删除
kafka-topics.sh --bootstrap-server localhost:9092 \
  --delete \
  --topic 'pattern.*'
```

### 3. DescribeConfigs (API Key 32)

**功能**: 查询主题或代理的配置

**特性**:
- 支持查询主题配置
- 支持查询代理配置
- 支持按配置名称过滤
- 返回配置属性（只读、默认值、敏感）

**请求结构**:
```go
type DescribeConfigsRequest struct {
    Header    *RequestHeader
    Resources []DescribeConfigsResource
}

type DescribeConfigsResource struct {
    ResourceType int8     // 2=Topic, 4=Broker
    ResourceName string
    ConfigNames  []string // nil = 返回所有配置
}
```

**响应结构**:
```go
type DescribeConfigsResponse struct {
    ThrottleTimeMs int32
    Results        []DescribeConfigsResult
}

type DescribeConfigsResult struct {
    ErrorCode    ErrorCode
    ErrorMessage *string
    ResourceType int8
    ResourceName string
    Entries      []DescribeConfigsEntry
}

type DescribeConfigsEntry struct {
    Name      string
    Value     *string
    ReadOnly  bool
    IsDefault bool
    Sensitive bool
}
```

**默认配置**:
- `compression.type`: `"none"` - 压缩类型
- `cleanup.policy`: `"delete"` - 清理策略
- `retention.ms`: `"604800000"` (7天) - 保留时间
- `segment.ms`: `"86400000"` (24小时) - 段滚动时间

**错误码**:
- `0` (None): 成功
- `3` (UnknownTopicOrPartition): 主题不存在
- `42` (InvalidRequest): 资源类型无效

**使用示例**:
```bash
# 查询主题配置
kafka-configs.sh --bootstrap-server localhost:9092 \
  --describe \
  --topic my-topic

# 查询特定配置
kafka-configs.sh --bootstrap-server localhost:9092 \
  --describe \
  --topic my-topic \
  --configs compression.type,retention.ms
```

## 架构设计

### Backend 抽象层

Admin API 通过 Backend 接口实现，支持两种后端模式：

```go
type Backend interface {
    CreateTopic(name string, numPartitions int32) error
    DeleteTopic(name string) error
    // ... 其他方法
}
```

#### DirectBackend (单节点)
- 直接操作本地存储
- 立即执行，无需共识
- 适合开发和测试环境

#### RaftBackend (分布式)
- 通过 Raft 共识协议
- 保证多节点一致性
- 适合生产环境

### Raft 集成

删除操作支持 Raft 共识：

```go
// FSM 命令类型
const (
    CommandCreateTopic = iota
    CommandDeleteTopic
    // ...
)

// FSM 应用删除命令
func (f *FSM) applyDeleteTopic(data []byte) error {
    var name string
    // 解码主题名称
    // 调用 topicManager.DeleteTopic()
}
```

### 协议实现

所有 Admin API 遵循 Kafka 协议规范：
- 使用标准 Kafka 编码格式
- 兼容 Kafka 客户端库
- 支持版本协商（通过 ApiVersions）

## 测试覆盖

### 单元测试

实现了 8 个全面的测试用例：

1. **TestCreateTopics**: 基本主题创建（3个分区）
2. **TestCreateTopicsValidateOnly**: 验证模式测试
3. **TestCreateTopicsAlreadyExists**: 重复创建错误处理
4. **TestDeleteTopics**: 成功删除主题
5. **TestDeleteTopicsNotFound**: 删除不存在的主题
6. **TestDescribeConfigs**: 查询主题配置
7. **TestDescribeConfigsTopicNotFound**: 查询不存在的主题
8. **TestAdminAPIEndToEnd**: 端到端工作流（创建→查询→删除）

### 测试策略

- 使用临时目录隔离测试环境
- 手动编码协议消息（验证序列化）
- 验证响应格式和错误码
- 检查副作用（主题创建/删除）

## 兼容性

### Kafka 客户端兼容性

Takhin Admin API 与以下 Kafka 工具完全兼容：

- `kafka-topics.sh` - 主题管理
- `kafka-configs.sh` - 配置管理
- Kafka Admin Client (Java/Python/Go)

### API 版本

当前实现支持：
- CreateTopics: 版本 0
- DeleteTopics: 版本 0
- DescribeConfigs: 版本 0

## 性能考虑

### 批量操作

- CreateTopics 和 DeleteTopics 支持批量操作
- 单个请求可以操作多个主题
- 减少网络往返次数

### 超时处理

- 所有操作支持超时参数
- 防止长时间阻塞
- Raft 操作使用 5 秒超时

### 错误处理

- 每个主题独立处理
- 部分失败不影响其他主题
- 详细的错误码和消息

## 未来扩展

### 计划实现的 Admin API

1. **AlterConfigs** (API Key 33) - 修改配置
2. **ListGroups** (API Key 16) - 列出消费者组
3. **DescribeGroups** (API Key 15) - 查询消费者组详情
4. **ListTopics** - 列出所有主题
5. **DescribeTopics** - 查询主题详情

### 增强功能

- 主题级别的 ACL
- 配额管理
- 分区重新分配
- 副本迁移

## 示例代码

### Go 客户端示例

```go
package main

import (
    "context"
    "github.com/segmentio/kafka-go"
)

func main() {
    // 创建主题
    conn, _ := kafka.Dial("tcp", "localhost:9092")
    controller, _ := conn.Controller()
    controllerConn, _ := kafka.Dial("tcp", controller.Host)
    
    topicConfigs := []kafka.TopicConfig{
        {
            Topic:             "my-topic",
            NumPartitions:     3,
            ReplicationFactor: 1,
        },
    }
    
    controllerConn.CreateTopics(topicConfigs...)
    
    // 删除主题
    controllerConn.DeleteTopics("my-topic")
}
```

### Python 客户端示例

```python
from kafka.admin import KafkaAdminClient, NewTopic

# 创建管理客户端
admin = KafkaAdminClient(bootstrap_servers='localhost:9092')

# 创建主题
topic = NewTopic(
    name='my-topic',
    num_partitions=3,
    replication_factor=1
)
admin.create_topics([topic])

# 删除主题
admin.delete_topics(['my-topic'])

# 查询配置
configs = admin.describe_configs(
    config_resources=[
        ('topic', 'my-topic')
    ]
)
```

## 故障排查

### 常见问题

1. **TopicAlreadyExists 错误**
   - 原因: 尝试创建已存在的主题
   - 解决: 使用不同的主题名或先删除旧主题

2. **UnknownTopicOrPartition 错误**
   - 原因: 操作不存在的主题
   - 解决: 确认主题名称正确，使用 `kafka-topics.sh --list` 查看

3. **InvalidRequest 错误**
   - 原因: 请求参数无效（如分区数 ≤ 0）
   - 解决: 检查请求参数是否符合要求

### 调试技巧

- 启用 DEBUG 日志查看详细请求信息
- 使用 `validate-only` 模式验证参数
- 检查文件系统权限（数据目录）
- 确认 Raft 集群健康状态（分布式模式）

## 参考资料

- [Kafka Protocol Documentation](https://kafka.apache.org/protocol)
- [Admin API Specification](https://kafka.apache.org/35/javadoc/org/apache/kafka/clients/admin/Admin.html)
- Takhin 代码实现:
  - `pkg/kafka/protocol/create_topics.go`
  - `pkg/kafka/protocol/delete_topics.go`
  - `pkg/kafka/protocol/describe_configs.go`
  - `pkg/kafka/handler/handler.go`

## 更新日志

### 2025-12-17
- ✅ 实现 CreateTopics API (API Key 19)
- ✅ 实现 DeleteTopics API (API Key 20)
- ✅ 实现 DescribeConfigs API (API Key 32)
- ✅ 添加 Backend 抽象层支持
- ✅ 集成 Raft 共识协议
- ✅ 创建完整测试套件（8个测试用例）
- ✅ 所有测试通过
