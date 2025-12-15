# 后端架构设计

## 1. Takhin Core 后端架构

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        Takhin Core                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                     API Layer                               │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │  Kafka API  │  Admin API  │  HTTP Proxy  │  Schema API    │ │
│  │  (9092)     │  (9644)     │  (8082)      │  (8081)        │ │
│  └─────┬────────────┬──────────────┬──────────────┬───────────┘ │
│        │            │              │              │              │
│  ┌─────▼────────────▼──────────────▼──────────────▼───────────┐ │
│  │                  Protocol Handler Layer                     │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ - Request Parsing                                           │ │
│  │ - Protocol Validation                                       │ │
│  │ - Request Routing                                           │ │
│  │ - Response Encoding                                         │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                          │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                    Service Layer                            │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ Topic Service │ Partition Service │ Consumer Group Service │ │
│  │ ACL Service   │ Config Service    │ Schema Service         │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                          │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                  Cluster Coordination                       │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ Raft Consensus │ Leader Election │ Partition Assignment   │ │
│  │ Metadata Sync  │ Membership      │ Health Check           │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                          │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                    Storage Layer                            │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ Log Manager │ Index Manager │ Compaction │ Tiered Storage │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                          │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                   Infrastructure                            │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ Network I/O │ Memory Pool │ Monitoring │ Configuration    │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 目录结构

```
projects/core/
├── cmd/
│   └── takhin/                 # 主程序入口
│       └── main.go
├── pkg/                        # 公共包
│   ├── kafka/                  # Kafka 协议实现
│   │   ├── protocol/           # 协议定义
│   │   ├── handler/            # 请求处理器
│   │   ├── client/             # 客户端抽象
│   │   └── server/             # 服务器实现
│   ├── admin/                  # Admin API
│   │   ├── handler/
│   │   └── service/
│   ├── storage/                # 存储引擎
│   │   ├── log/                # 日志段管理
│   │   ├── index/              # 索引管理
│   │   ├── compaction/         # 日志压缩
│   │   └── tiered/             # 分层存储
│   ├── raft/                   # Raft 共识
│   │   ├── core/               # Raft 核心算法
│   │   ├── transport/          # RPC 传输
│   │   └── snapshot/           # 快照管理
│   ├── cluster/                # 集群管理
│   │   ├── coordinator/        # 协调器
│   │   ├── metadata/           # 元数据
│   │   ├── partition/          # 分区管理
│   │   └── replication/        # 副本管理
│   ├── schema/                 # Schema Registry
│   │   ├── avro/
│   │   ├── protobuf/
│   │   └── json/
│   ├── security/               # 安全模块
│   │   ├── auth/               # 认证
│   │   ├── acl/                # 授权
│   │   └── tls/                # TLS
│   ├── config/                 # 配置管理
│   ├── metrics/                # 监控指标
│   └── utils/                  # 工具函数
├── internal/                   # 内部包
│   ├── codec/                  # 编解码器
│   ├── compression/            # 压缩算法
│   ├── network/                # 网络层
│   └── pool/                   # 对象池
├── api/                        # API 定义
│   └── proto/                  # Protocol Buffers
├── configs/                    # 配置文件
│   ├── default.yaml
│   └── production.yaml
└── tests/                      # 测试
    ├── unit/
    ├── integration/
    └── benchmark/
```

### 1.3 核心模块设计

#### 1.3.1 Kafka Protocol Handler

```go
// pkg/kafka/handler/handler.go
package handler

import (
    "context"
    "github.com/takhin/core/pkg/kafka/protocol"
)

// Handler 处理 Kafka 协议请求
type Handler interface {
    // Handle 处理请求并返回响应
    Handle(ctx context.Context, req protocol.Request) (protocol.Response, error)
    
    // APIKey 返回处理的 API Key
    APIKey() protocol.APIKey
    
    // MinVersion 返回支持的最小版本
    MinVersion() int16
    
    // MaxVersion 返回支持的最大版本
    MaxVersion() int16
}

// ProduceHandler 处理生产消息请求
type ProduceHandler struct {
    partitionMgr PartitionManager
    replicaMgr   ReplicationManager
}

func (h *ProduceHandler) Handle(ctx context.Context, req protocol.Request) (protocol.Response, error) {
    produceReq := req.(*protocol.ProduceRequest)
    
    // 1. 验证请求
    if err := h.validateRequest(produceReq); err != nil {
        return nil, err
    }
    
    // 2. 写入分区
    results, err := h.writeToPartitions(ctx, produceReq)
    if err != nil {
        return nil, err
    }
    
    // 3. 等待副本同步 (根据 acks 配置)
    if produceReq.Acks != 0 {
        if err := h.waitForReplicas(ctx, results); err != nil {
            return nil, err
        }
    }
    
    // 4. 构造响应
    return h.buildResponse(results), nil
}
```

#### 1.3.2 Storage Engine

```go
// pkg/storage/log/segment.go
package log

import (
    "io"
    "os"
    "sync"
)

// Segment 表示一个日志段
type Segment struct {
    mu            sync.RWMutex
    baseOffset    int64
    file          *os.File
    index         *Index
    timeIndex     *TimeIndex
    maxSegmentSize int64
}

// Append 追加消息到日志段
func (s *Segment) Append(offset int64, data []byte) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // 1. 写入数据文件
    position, err := s.file.Seek(0, io.SeekEnd)
    if err != nil {
        return err
    }
    
    if _, err := s.file.Write(data); err != nil {
        return err
    }
    
    // 2. 更新索引
    if err := s.index.Append(offset, position); err != nil {
        return err
    }
    
    // 3. 同步到磁盘 (根据配置)
    if err := s.file.Sync(); err != nil {
        return err
    }
    
    return nil
}

// Read 从日志段读取消息
func (s *Segment) Read(offset int64, maxBytes int) ([]byte, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // 1. 查找索引
    position, err := s.index.Lookup(offset)
    if err != nil {
        return nil, err
    }
    
    // 2. 读取数据
    data := make([]byte, maxBytes)
    n, err := s.file.ReadAt(data, position)
    if err != nil && err != io.EOF {
        return nil, err
    }
    
    return data[:n], nil
}
```

#### 1.3.3 Raft Consensus

```go
// pkg/raft/core/raft.go
package core

import (
    "context"
    "time"
)

// Raft 实现 Raft 共识算法
type Raft struct {
    // 持久化状态
    currentTerm  int64
    votedFor     int64
    log          []LogEntry
    
    // 易失状态
    commitIndex  int64
    lastApplied  int64
    
    // Leader 状态
    nextIndex    map[int64]int64  // 下一个发送给每个 follower 的索引
    matchIndex   map[int64]int64  // 已知已复制到每个 follower 的索引
    
    // 配置
    id           int64
    peers        []Peer
    state        State
    
    // 通道
    applyCh      chan ApplyMsg
    heartbeatCh  chan struct{}
    voteCh       chan struct{}
}

// Start 启动 Raft 节点
func (r *Raft) Start() {
    go r.electionLoop()
    go r.heartbeatLoop()
    go r.applyLoop()
}

// RequestVote RPC 处理
func (r *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // 1. 检查任期
    if args.Term < r.currentTerm {
        reply.Term = r.currentTerm
        reply.VoteGranted = false
        return nil
    }
    
    // 2. 更新任期
    if args.Term > r.currentTerm {
        r.currentTerm = args.Term
        r.votedFor = -1
        r.state = Follower
    }
    
    // 3. 投票逻辑
    if (r.votedFor == -1 || r.votedFor == args.CandidateId) &&
       r.isLogUpToDate(args.LastLogIndex, args.LastLogTerm) {
        r.votedFor = args.CandidateId
        reply.VoteGranted = true
        r.voteCh <- struct{}{}
    } else {
        reply.VoteGranted = false
    }
    
    reply.Term = r.currentTerm
    return nil
}

// AppendEntries RPC 处理
func (r *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // 1. 检查任期
    if args.Term < r.currentTerm {
        reply.Term = r.currentTerm
        reply.Success = false
        return nil
    }
    
    // 2. 重置选举超时
    r.heartbeatCh <- struct{}{}
    
    // 3. 日志一致性检查
    if args.PrevLogIndex >= 0 {
        if args.PrevLogIndex >= int64(len(r.log)) ||
           r.log[args.PrevLogIndex].Term != args.PrevLogTerm {
            reply.Success = false
            reply.Term = r.currentTerm
            return nil
        }
    }
    
    // 4. 追加日志
    r.log = r.log[:args.PrevLogIndex+1]
    r.log = append(r.log, args.Entries...)
    
    // 5. 更新 commit index
    if args.LeaderCommit > r.commitIndex {
        r.commitIndex = min(args.LeaderCommit, int64(len(r.log)-1))
    }
    
    reply.Success = true
    reply.Term = r.currentTerm
    return nil
}
```

## 2. Takhin Console 后端架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    Takhin Console Backend                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                      API Layer                              │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │  REST API  │  gRPC API   │  WebSocket  │  Static Assets   │ │
│  │  (HTTP/1.1)│  (HTTP/2)   │             │                  │ │
│  └─────┬────────────┬──────────────┬──────────────┬──────────┘ │
│        │            │              │              │             │
│  ┌─────▼────────────▼──────────────▼──────────────▼──────────┐ │
│  │                  Middleware Layer                          │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │ Auth │ CORS │ Logging │ Metrics │ Rate Limit │ Recovery │ │
│  └─────┬──────────────────────────────────────────────────────┘ │
│        │                                                         │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                   Service Layer                             │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ Topic Service      │ Consumer Group Service                │ │
│  │ Message Service    │ ACL Service                           │ │
│  │ Schema Service     │ Connect Service                       │ │
│  │ Cluster Service    │ Transform Service                     │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                         │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                   Client Layer                              │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ Kafka Client       │ Admin Client                          │ │
│  │ Schema Registry    │ Kafka Connect Client                  │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                         │
│        ▼                                                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              External Dependencies                        │  │
│  ├──────────────────────────────────────────────────────────┤  │
│  │ Takhin Core │ Schema Registry │ Kafka Connect Cluster   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 目录结构

```
projects/console/backend/
├── cmd/
│   └── api/                    # API 服务入口
│       └── main.go
├── pkg/                        # 公共包
│   ├── api/                    # API 层
│   │   ├── handler/            # HTTP 处理器
│   │   ├── middleware/         # 中间件
│   │   ├── router/             # 路由
│   │   └── validator/          # 请求验证
│   ├── service/                # 服务层
│   │   ├── topic/              # Topic 服务
│   │   ├── message/            # 消息服务
│   │   ├── consumer/           # Consumer Group 服务
│   │   ├── schema/             # Schema 服务
│   │   ├── connect/            # Kafka Connect 服务
│   │   ├── acl/                # ACL 服务
│   │   └── cluster/            # 集群服务
│   ├── client/                 # 客户端层
│   │   ├── kafka/              # Kafka 客户端
│   │   ├── admin/              # Admin 客户端
│   │   └── schema/             # Schema Registry 客户端
│   ├── model/                  # 数据模型
│   │   ├── topic.go
│   │   ├── message.go
│   │   ├── consumer.go
│   │   └── schema.go
│   ├── config/                 # 配置管理
│   ├── logger/                 # 日志
│   ├── metrics/                # 监控
│   └── utils/                  # 工具函数
├── internal/                   # 内部包
│   ├── serde/                  # 序列化/反序列化
│   │   ├── avro/
│   │   ├── protobuf/
│   │   ├── json/
│   │   └── msgpack/
│   └── interpreter/            # JavaScript 解释器 (消息过滤)
├── api/                        # API 定义
│   └── proto/                  # Protocol Buffers
│       ├── topic/
│       ├── message/
│       ├── consumer/
│       └── schema/
└── tests/                      # 测试
    ├── unit/
    ├── integration/
    └── e2e/
```

### 2.3 核心模块设计

#### 2.3.1 API Handler

```go
// pkg/api/handler/topic.go
package handler

import (
    "encoding/json"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "github.com/takhin/console/backend/pkg/service"
)

// TopicHandler 处理 Topic 相关请求
type TopicHandler struct {
    topicSvc service.TopicService
    logger   *slog.Logger
}

// ListTopics 列出所有 Topics
func (h *TopicHandler) ListTopics(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 1. 获取 Topics
    topics, err := h.topicSvc.ListTopics(ctx)
    if err != nil {
        h.handleError(w, err)
        return
    }
    
    // 2. 返回响应
    h.sendJSON(w, http.StatusOK, topics)
}

// GetTopicDetails 获取 Topic 详情
func (h *TopicHandler) GetTopicDetails(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    topicName := chi.URLParam(r, "topicName")
    
    // 1. 验证参数
    if topicName == "" {
        h.handleError(w, ErrInvalidTopicName)
        return
    }
    
    // 2. 获取详情
    details, err := h.topicSvc.GetTopicDetails(ctx, topicName)
    if err != nil {
        h.handleError(w, err)
        return
    }
    
    // 3. 返回响应
    h.sendJSON(w, http.StatusOK, details)
}

// CreateTopic 创建 Topic
func (h *TopicHandler) CreateTopic(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // 1. 解析请求
    var req CreateTopicRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.handleError(w, ErrInvalidRequest)
        return
    }
    
    // 2. 验证请求
    if err := h.validateCreateTopicRequest(&req); err != nil {
        h.handleError(w, err)
        return
    }
    
    // 3. 创建 Topic
    if err := h.topicSvc.CreateTopic(ctx, &req); err != nil {
        h.handleError(w, err)
        return
    }
    
    // 4. 返回响应
    h.sendJSON(w, http.StatusCreated, map[string]string{
        "message": "Topic created successfully",
    })
}
```

#### 2.3.2 Service Layer

```go
// pkg/service/topic/service.go
package topic

import (
    "context"
    "fmt"
    
    "github.com/takhin/console/backend/pkg/client/kafka"
    "github.com/takhin/console/backend/pkg/model"
)

// Service 提供 Topic 相关业务逻辑
type Service struct {
    kafkaClient kafka.Client
    adminClient kafka.AdminClient
    logger      *slog.Logger
}

// ListTopics 列出所有 Topics
func (s *Service) ListTopics(ctx context.Context) ([]*model.Topic, error) {
    // 1. 获取 Topic 列表
    metadata, err := s.kafkaClient.GetMetadata(ctx)
    if err != nil {
        return nil, fmt.Errorf("get metadata: %w", err)
    }
    
    // 2. 并发获取每个 Topic 的详细信息
    topics := make([]*model.Topic, 0, len(metadata.Topics))
    for _, topicMeta := range metadata.Topics {
        // 获取配置
        config, err := s.adminClient.DescribeConfig(ctx, topicMeta.Name)
        if err != nil {
            s.logger.Warn("failed to get topic config", 
                "topic", topicMeta.Name, 
                "error", err)
            config = nil
        }
        
        // 构造 Topic 对象
        topic := &model.Topic{
            Name:           topicMeta.Name,
            Partitions:     len(topicMeta.Partitions),
            ReplicationFactor: s.getReplicationFactor(topicMeta),
            Config:         config,
        }
        
        topics = append(topics, topic)
    }
    
    return topics, nil
}

// GetTopicDetails 获取 Topic 详情
func (s *Service) GetTopicDetails(ctx context.Context, topicName string) (*model.TopicDetails, error) {
    // 1. 获取基本信息
    metadata, err := s.kafkaClient.GetTopicMetadata(ctx, topicName)
    if err != nil {
        return nil, fmt.Errorf("get topic metadata: %w", err)
    }
    
    // 2. 获取配置
    config, err := s.adminClient.DescribeConfig(ctx, topicName)
    if err != nil {
        return nil, fmt.Errorf("get topic config: %w", err)
    }
    
    // 3. 获取分区详情
    partitions := make([]*model.PartitionInfo, 0, len(metadata.Partitions))
    for _, p := range metadata.Partitions {
        // 获取 low/high watermark
        lowWatermark, err := s.kafkaClient.GetWatermark(ctx, topicName, p.ID, kafka.OffsetOldest)
        if err != nil {
            s.logger.Warn("failed to get low watermark",
                "topic", topicName,
                "partition", p.ID,
                "error", err)
            lowWatermark = -1
        }
        
        highWatermark, err := s.kafkaClient.GetWatermark(ctx, topicName, p.ID, kafka.OffsetNewest)
        if err != nil {
            s.logger.Warn("failed to get high watermark",
                "topic", topicName,
                "partition", p.ID,
                "error", err)
            highWatermark = -1
        }
        
        partition := &model.PartitionInfo{
            ID:            p.ID,
            Leader:        p.Leader,
            Replicas:      p.Replicas,
            ISR:           p.ISR,
            LowWatermark:  lowWatermark,
            HighWatermark: highWatermark,
            MessageCount:  highWatermark - lowWatermark,
        }
        
        partitions = append(partitions, partition)
    }
    
    // 4. 构造详情对象
    details := &model.TopicDetails{
        Name:       topicName,
        Partitions: partitions,
        Config:     config,
    }
    
    return details, nil
}
```

#### 2.3.3 Message Service (消息查看器)

```go
// pkg/service/message/service.go
package message

import (
    "context"
    "fmt"
    
    "github.com/dop251/goja"
    "github.com/takhin/console/backend/pkg/client/kafka"
    "github.com/takhin/console/backend/pkg/internal/serde"
    "github.com/takhin/console/backend/pkg/model"
)

// Service 提供消息查看和搜索功能
type Service struct {
    kafkaClient kafka.Client
    serdeManager serde.Manager
    jsRuntime    *goja.Runtime
    logger       *slog.Logger
}

// SearchMessages 搜索消息
func (s *Service) SearchMessages(ctx context.Context, req *model.SearchRequest) (*model.SearchResponse, error) {
    // 1. 验证请求
    if err := s.validateSearchRequest(req); err != nil {
        return nil, err
    }
    
    // 2. 编译过滤器脚本
    var filterFunc func(msg *model.Message) bool
    if req.FilterCode != "" {
        var err error
        filterFunc, err = s.compileFilter(req.FilterCode)
        if err != nil {
            return nil, fmt.Errorf("compile filter: %w", err)
        }
    }
    
    // 3. 读取消息
    messages := make([]*model.Message, 0, req.MaxResults)
    
    for partition := req.StartPartition; partition <= req.EndPartition; partition++ {
        // 创建消费者
        consumer, err := s.kafkaClient.NewConsumer(ctx, &kafka.ConsumerConfig{
            GroupID: fmt.Sprintf("console-search-%s", uuid.New().String()),
        })
        if err != nil {
            return nil, fmt.Errorf("create consumer: %w", err)
        }
        defer consumer.Close()
        
        // 定位到起始 offset
        if err := consumer.Seek(ctx, req.Topic, partition, req.StartOffset); err != nil {
            return nil, fmt.Errorf("seek: %w", err)
        }
        
        // 读取消息
        for len(messages) < req.MaxResults {
            rawMsg, err := consumer.ReadMessage(ctx, req.Timeout)
            if err != nil {
                if err == kafka.ErrTimeout {
                    break
                }
                return nil, fmt.Errorf("read message: %w", err)
            }
            
            // 反序列化消息
            msg, err := s.deserializeMessage(rawMsg, req.Encoding)
            if err != nil {
                s.logger.Warn("failed to deserialize message",
                    "topic", req.Topic,
                    "partition", partition,
                    "offset", rawMsg.Offset,
                    "error", err)
                continue
            }
            
            // 应用过滤器
            if filterFunc != nil && !filterFunc(msg) {
                continue
            }
            
            messages = append(messages, msg)
            
            // 检查是否达到结束 offset
            if rawMsg.Offset >= req.EndOffset {
                break
            }
        }
        
        if len(messages) >= req.MaxResults {
            break
        }
    }
    
    // 4. 构造响应
    resp := &model.SearchResponse{
        Messages:    messages,
        TotalCount:  len(messages),
        HasMore:     len(messages) >= req.MaxResults,
    }
    
    return resp, nil
}

// compileFilter 编译 JavaScript 过滤器
func (s *Service) compileFilter(code string) (func(*model.Message) bool, error) {
    // 编译脚本
    program, err := goja.Compile("filter", code, false)
    if err != nil {
        return nil, fmt.Errorf("compile script: %w", err)
    }
    
    return func(msg *model.Message) bool {
        // 创建新的运行时
        vm := goja.New()
        
        // 注入消息对象
        vm.Set("message", msg)
        
        // 执行脚本
        result, err := vm.RunProgram(program)
        if err != nil {
            s.logger.Warn("filter execution error", "error", err)
            return false
        }
        
        // 返回布尔结果
        return result.ToBoolean()
    }, nil
}

// deserializeMessage 反序列化消息
func (s *Service) deserializeMessage(raw *kafka.Message, encoding string) (*model.Message, error) {
    deserializer := s.serdeManager.GetDeserializer(encoding)
    if deserializer == nil {
        return nil, fmt.Errorf("unsupported encoding: %s", encoding)
    }
    
    // 反序列化 key
    var key interface{}
    if len(raw.Key) > 0 {
        var err error
        key, err = deserializer.Deserialize(raw.Key)
        if err != nil {
            return nil, fmt.Errorf("deserialize key: %w", err)
        }
    }
    
    // 反序列化 value
    value, err := deserializer.Deserialize(raw.Value)
    if err != nil {
        return nil, fmt.Errorf("deserialize value: %w", err)
    }
    
    msg := &model.Message{
        Topic:     raw.Topic,
        Partition: raw.Partition,
        Offset:    raw.Offset,
        Timestamp: raw.Timestamp,
        Key:       key,
        Value:     value,
        Headers:   raw.Headers,
    }
    
    return msg, nil
}
```

### 2.4 API 设计原则

#### 2.4.1 RESTful API 规范
- **资源命名**: 使用复数名词，小写，单词用连字符分隔
- **HTTP 方法**: GET (查询), POST (创建), PUT (更新), DELETE (删除)
- **状态码**: 正确使用 HTTP 状态码
- **版本控制**: URL 路径中包含版本 `/api/v1/`
- **分页**: 使用 `limit` 和 `offset` 参数
- **过滤**: 使用查询参数过滤结果
- **排序**: 使用 `sort` 参数指定排序字段

#### 2.4.2 gRPC API 规范
- **服务定义**: Protocol Buffers 定义
- **命名规范**: PascalCase 命名服务和方法
- **错误处理**: 使用 gRPC 状态码
- **流式传输**: 支持 Server Streaming
- **认证**: gRPC 拦截器

### 2.5 性能优化策略

#### 2.5.1 并发处理
- **Goroutine Pool**: 限制并发数量
- **Context 传播**: 超时和取消控制
- **批量操作**: 减少 RPC 调用次数
- **连接复用**: 保持长连接

#### 2.5.2 缓存策略
- **内存缓存**: 热点数据缓存
- **TTL 控制**: 合理设置过期时间
- **缓存预热**: 启动时加载热点数据
- **缓存更新**: 事件驱动的缓存失效

#### 2.5.3 数据库优化
- **连接池**: 复用数据库连接
- **索引优化**: 关键字段建立索引
- **批量写入**: 减少写入次数
- **读写分离**: 分离读写流量

---

**文档版本**: v1.0  
**最后更新**: 2025-12-14  
**维护者**: Takhin Team
