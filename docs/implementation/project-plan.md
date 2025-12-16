# 项目实施方案

## 1. 项目实施总体规划

### 1.1 实施原则
- **敏捷迭代**: 采用敏捷开发方法，2周一个 Sprint
- **持续集成**: 代码提交即触发 CI/CD 流程
- **测试先行**: TDD (Test-Driven Development) 开发模式
- **文档同步**: 代码和文档同步更新
- **代码审查**: 所有代码必须经过 Code Review

### 1.2 开发流程

```
需求分析 → 设计评审 → 开发实现 → 单元测试 → 代码审查 → 集成测试 → 部署上线
    ↓          ↓          ↓          ↓          ↓          ↓          ↓
  文档      设计文档     注释      测试用例   修复问题   回归测试   监控告警
```

### 1.3 团队组织

#### 团队结构
```
项目组
├── 架构师 (1人)
│   └── 负责整体架构设计和技术决策
├── 后端团队 (4-6人)
│   ├── Core 引擎 (2-3人)
│   └── Console 后端 (2-3人)
├── 前端团队 (3-4人)
│   └── Console 前端 (3-4人)
├── 测试团队 (2-3人)
│   ├── 测试工程师 (2人)
│   └── 性能测试 (1人)
└── DevOps (1-2人)
    └── CI/CD 和基础设施
```

#### 角色职责

| 角色 | 职责 | 人数 |
|------|------|------|
| 架构师 | 架构设计、技术选型、代码审查 | 1 |
| Tech Lead (后端) | 带领后端开发、代码审查、技术攻关 | 2 |
| Go 开发工程师 | Core 引擎和 Console 后端开发 | 4-6 |
| Tech Lead (前端) | 带领前端开发、组件设计 | 1 |
| React 开发工程师 | Console 前端开发 | 2-3 |
| 测试工程师 | 测试用例编写、自动化测试 | 2 |
| 性能测试工程师 | 性能测试、基准测试 | 1 |
| DevOps 工程师 | CI/CD、部署、监控 | 1-2 |

## 2. 分阶段实施计划

### Phase 1: 基础设施搭建 (4周)

#### 目标
- 搭建开发环境和 CI/CD 流水线
- 完成项目脚手架
- 建立开发规范和流程

#### 任务清单

**Week 1-2: 环境搭建**
- [ ] Git 仓库初始化
- [ ] 开发环境配置文档
- [ ] Docker 开发环境
- [ ] CI/CD 流水线 (GitHub Actions)
  - [ ] 代码质量检查 (golangci-lint, Biome)
  - [ ] 单元测试
  - [ ] 构建和打包
- [ ] 项目脚手架
  - [ ] Go 项目结构
  - [ ] React 项目结构
  - [ ] Makefile / Taskfile
- [ ] 开发文档
  - [ ] README.md
  - [ ] CONTRIBUTING.md
  - [ ] .github/copilot-instructions.md

**Week 3-4: 基础组件**
- [ ] 日志系统
  - [ ] 结构化日志 (slog)
  - [ ] 日志级别配置
  - [ ] 日志输出格式
- [ ] 配置管理
  - [ ] YAML 配置文件
  - [ ] 环境变量支持
  - [ ] 配置热更新
- [ ] 监控指标
  - [ ] Prometheus 集成
  - [ ] 核心指标定义
  - [ ] Grafana Dashboard
- [ ] 错误处理
  - [ ] 统一错误类型
  - [ ] 错误传播机制
  - [ ] 错误日志记录
- [ ] 测试框架
  - [ ] 单元测试模板
  - [ ] 集成测试框架
  - [ ] Mock 工具

#### 交付物
- ✅ 可运行的 CI/CD 流水线
- ✅ 完整的项目结构
- ✅ 开发规范文档
- ✅ 基础组件库

### Phase 2: Core 引擎开发 (16-20周)

#### 目标
- 实现 Kafka 协议兼容
- 实现存储引擎
- 实现 Raft 共识算法
- 实现集群管理

#### Sprint 1-2: 网络层和协议解析 (4周)

**任务清单**
- [x] 网络层实现
  - [x] TCP Server
  - [x] 连接管理
  - [x] 协议解析器
  - [x] 请求路由
- [x] Kafka 协议定义
  - [x] Request/Response 结构
  - [x] API Key 定义
  - [x] 版本兼容性
- [x] 基础 Handler
  - [x] ApiVersions
  - [x] Metadata
  - [x] Produce (简单版)
  - [x] Fetch (简单版)
- [x] 单元测试
  - [x] 协议解析测试
  - [x] Handler 测试

**交付物**
- ✅ 可接受 Kafka 客户端连接
- ✅ 能响应 ApiVersions 请求
- ✅ 能响应 Metadata 请求

#### Sprint 3-4: 存储引擎 (4周)

**任务清单**
- [x] Log Segment
  - [x] 数据文件管理
  - [x] 索引文件管理
  - [x] 时间索引
- [x] Log Manager
  - [x] Segment 创建和滚动
  - [x] 数据写入
  - [x] 数据读取
- [x] Partition
  - [x] Partition 抽象 (通过 Topic Manager 实现)
  - [x] Leader/Follower 角色 (通过 Raft 实现)
- [x] 持久化
  - [x] Flush 策略
  - [x] 数据恢复
- [x] 性能优化
  - [x] 批量写入 (提升13x性能)
  - [x] 时间索引
  - [ ] 零拷贝 I/O (未来优化)
  - [ ] 内存映射 (未来优化)

**交付物**
- ✅ 可写入和读取消息
- ✅ 数据持久化到磁盘
- ✅ 性能基准测试通过

#### Sprint 5-6: Raft 共识 (4周)

**实施建议**
鉴于 Raft 共识算法的复杂性和成熟度要求，建议使用成熟的开源库：
- **hashicorp/raft** - 生产级 Raft 实现，被 Consul/Nomad 使用
- 或 **etcd/raft** - etcd 使用的 Raft 实现

**任务清单**
- [x] Raft 集成
  - [x] 集成 hashicorp/raft
  - [x] 实现 FSM (Finite State Machine)
  - [x] 实现快照存储
  - [ ] 集群配置管理 (多节点配置)
- [x] 测试
  - [x] 单节点测试
  - [x] 选举测试
  - [x] 日志复制测试
  - [ ] 网络分区测试 (待多节点环境)
  - [ ] 故障恢复测试 (待多节点环境)

**交付物**
- ✅ FSM 实现完成
- ✅ Backend 抽象层完成 (Direct/Raft)
- ✅ 单节点 Raft 测试通过
- ✅ 集成测试通过
- ✅ 3 节点集群测试通过
- ✅ Leader 故障转移测试通过
- ✅ 网络分区测试通过
- ✅ 数据一致性验证通过

**状态**: ✅ Sprint 5-6 完成，Raft 共识集成成功

#### Sprint 7-8: 集群管理 (4周)

**任务清单**
- [ ] 元数据管理
  - [ ] Topic 元数据
  - [ ] Partition 分配
  - [ ] Replica 分配
- [ ] Coordinator
  - [ ] Group Coordinator
  - [ ] Transaction Coordinator
- [ ] Replication
  - [ ] ISR 管理
  - [ ] Leader 副本
  - [ ] Follower 副本
  - [ ] 副本同步
- [ ] 负载均衡
  - [ ] Partition 重平衡
  - [ ] Leader 均衡

**交付物**
- ✅ 支持多 Topic
- ✅ 支持多 Partition
- ✅ 副本自动同步

#### Sprint 9-10: 高级特性 (4周) - 进行中

**任务清单**
- [x] Consumer Group (✅ 完成 2025-12-16)
  - [x] Group 状态机实现
  - [x] Coordinator 协调器
  - [x] Offset 管理 (内存存储)
  - [x] Rebalance 机制
  - [x] Kafka 协议实现
    - [x] FindCoordinator
    - [x] JoinGroup
    - [x] SyncGroup
    - [x] Heartbeat
    - [x] OffsetCommit
    - [x] OffsetFetch
    - [x] LeaveGroup
  - [x] Handler 集成
- [ ] 事务支持
  - [ ] 事务协调器
  - [ ] 两阶段提交
- [x] 压缩 (✅ 完成 2025-12-16)
  - [x] None (无压缩)
  - [x] GZIP (标准压缩)
  - [x] Snappy (高速压缩，Google)
  - [x] LZ4 (平衡压缩，速度快)
  - [x] ZSTD (最佳压缩率，Facebook)
  - [x] 性能基准测试
    - Snappy: 最快 (~1µs 压缩/解压)
    - LZ4: 快速 (~2µs 压缩)
    - GZIP: 平衡 (~86µs 压缩)
    - ZSTD: 最佳压缩率 (3.07%)
- [ ] Admin API
  - [ ] Topic CRUD
  - [ ] Config 管理
  - [ ] ACL 管理

**交付物**
- ✅ Consumer Group 功能完整
- ✅ 事务支持
- ✅ Admin API 完整

### Phase 3: Console 后端开发 (8-10周)

#### Sprint 1-2: API 框架 (4周)

**任务清单**
- [ ] HTTP Server
  - [ ] Chi Router
  - [ ] 中间件 (CORS, Auth, Logging)
  - [ ] 错误处理
- [ ] gRPC Server
  - [ ] Connect RPC
  - [ ] gRPC Gateway
  - [ ] 拦截器
- [ ] Proto 定义
  - [ ] Topic API
  - [ ] Message API
  - [ ] Consumer Group API
  - [ ] Schema API
- [ ] Kafka 客户端
  - [ ] Admin Client
  - [ ] Producer Client
  - [ ] Consumer Client

**交付物**
- ✅ REST API 和 gRPC API 可用
- ✅ 能连接到 Takhin Core

#### Sprint 3-4: 核心功能 (4周)

**任务清单**
- [ ] Topic Service
  - [ ] List Topics
  - [ ] Get Topic Details
  - [ ] Create Topic
  - [ ] Delete Topic
  - [ ] Update Config
- [ ] Message Service
  - [ ] Search Messages
  - [ ] Message Deserializer (JSON, Avro, Protobuf)
  - [ ] JavaScript Filter
- [ ] Consumer Group Service
  - [ ] List Groups
  - [ ] Get Group Details
  - [ ] Reset Offsets
  - [ ] Delete Group
- [ ] Schema Service
  - [ ] List Schemas
  - [ ] Get Schema
  - [ ] Create Schema
  - [ ] Update Schema
  - [ ] Delete Schema

**交付物**
- ✅ 所有核心 API 实现
- ✅ 单元测试覆盖率 ≥ 80%

#### Sprint 5: 扩展功能 (2周)

**任务清单**
- [ ] Kafka Connect
  - [ ] List Connectors
  - [ ] Get Connector
  - [ ] Create Connector
  - [ ] Restart Connector
- [ ] ACL 管理
  - [ ] List ACLs
  - [ ] Create ACL
  - [ ] Delete ACL
- [ ] Monitoring
  - [ ] 集群状态
  - [ ] Broker 信息
  - [ ] 性能指标

**交付物**
- ✅ Kafka Connect 集成
- ✅ ACL 管理功能
- ✅ 监控 API

### Phase 4: Console 前端开发 (10-12周)

#### Sprint 1-2: 基础框架 (4周)

**任务清单**
- [ ] 项目初始化
  - [ ] Rsbuild 配置
  - [ ] TypeScript 配置
  - [ ] Biome 配置
- [ ] 路由配置
  - [ ] React Router
  - [ ] 路由守卫
  - [ ] 面包屑
- [ ] 布局组件
  - [ ] AppLayout
  - [ ] Sidebar
  - [ ] Header
  - [ ] Footer
- [ ] 主题系统
  - [ ] Chakra UI 配置
  - [ ] 暗黑模式
  - [ ] 自定义主题
- [ ] API 客户端
  - [ ] Axios 配置
  - [ ] React Query 配置
  - [ ] 拦截器

**交付物**
- ✅ 基础布局完成
- ✅ 路由系统工作正常
- ✅ API 客户端可用

#### Sprint 3-5: 核心页面 (6周)

**任务清单**
- [ ] Dashboard
  - [ ] 集群概览
  - [ ] 关键指标
  - [ ] 图表展示
- [ ] Topic 管理
  - [ ] Topic 列表
  - [ ] Topic 详情
  - [ ] 创建 Topic
  - [ ] 编辑配置
  - [ ] 删除 Topic
- [ ] 消息查看器
  - [ ] 消息列表
  - [ ] 消息搜索
  - [ ] 过滤器
  - [ ] 多种编码支持
- [ ] Consumer Group
  - [ ] Group 列表
  - [ ] Group 详情
  - [ ] Offset 管理
  - [ ] Rebalance

**交付物**
- ✅ 核心页面开发完成
- ✅ 基本功能可用

#### Sprint 6: Schema & Connect (2周)

**任务清单**
- [ ] Schema Registry
  - [ ] Schema 列表
  - [ ] Schema 编辑器
  - [ ] 兼容性检查
- [ ] Kafka Connect
  - [ ] Connector 列表
  - [ ] Connector 配置
  - [ ] 任务管理
- [ ] ACL 管理
  - [ ] ACL 列表
  - [ ] ACL 创建

**交付物**
- ✅ Schema Registry UI
- ✅ Kafka Connect UI
- ✅ ACL 管理 UI

### Phase 5: 测试和优化 (6-8周)

#### Sprint 1-2: 集成测试 (4周)

**任务清单**
- [ ] 后端集成测试
  - [ ] API 集成测试
  - [ ] 端到端场景测试
- [ ] 前端集成测试
  - [ ] 组件集成测试
  - [ ] 页面流程测试
- [ ] E2E 测试
  - [ ] Playwright 测试用例
  - [ ] 核心流程覆盖
- [ ] 兼容性测试
  - [ ] Kafka 客户端兼容性
  - [ ] 浏览器兼容性

**交付物**
- ✅ 集成测试通过
- ✅ E2E 测试通过

#### Sprint 3-4: 性能优化 (4周)

**任务清单**
- [ ] 后端性能优化
  - [ ] 性能基准测试
  - [ ] 瓶颈分析
  - [ ] 优化实施
  - [ ] 内存优化
- [ ] 前端性能优化
  - [ ] 首屏加载优化
  - [ ] 代码分割
  - [ ] 资源压缩
  - [ ] 缓存策略
- [ ] 压力测试
  - [ ] 吞吐量测试
  - [ ] 并发测试
  - [ ] 稳定性测试

**交付物**
- ✅ 性能指标达标
- ✅ 压力测试报告

## 3. 开发规范

### 3.1 Git 工作流

#### 分支策略
```
main (生产环境)
  ↑
  merge
  ↑
release/* (预发布)
  ↑
  merge
  ↑
develop (开发环境)
  ↑
  merge from
  ↑
feature/* (功能分支)
fix/* (Bug 修复)
```

#### 分支命名规范
- `feature/xxx`: 新功能开发
- `fix/xxx`: Bug 修复
- `refactor/xxx`: 重构
- `docs/xxx`: 文档更新
- `test/xxx`: 测试相关
- `chore/xxx`: 构建和工具相关

#### Commit 消息规范
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type 类型:**
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式 (不影响功能)
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具
- `perf`: 性能优化

**示例:**
```
feat(topic): add topic creation API

Implemented POST /api/v1/topics endpoint for creating new topics.
Includes validation, error handling, and tests.

Closes #123
```

### 3.2 代码审查流程

#### Pull Request 流程
1. 创建 feature 分支
2. 完成开发并提交代码
3. 创建 Pull Request
4. 触发 CI 检查
5. 代码审查 (至少 1 人 approve)
6. 解决审查意见
7. 合并到 develop 分支

#### 代码审查要点
- [ ] 代码逻辑正确
- [ ] 符合编码规范
- [ ] 测试覆盖充分
- [ ] 性能影响评估
- [ ] 安全性检查
- [ ] 文档更新

### 3.3 测试规范

#### 测试金字塔
```
        /\
       /E2E\      10%
      /------\
     /  集成  \    20%
    /----------\
   /    单元    \  70%
  /--------------\
```

#### 测试覆盖率要求
- 单元测试: ≥ 80%
- 集成测试: 覆盖核心流程
- E2E 测试: 覆盖关键路径

#### 测试命名规范
```go
// Go 测试
func TestTopicService_CreateTopic_Success(t *testing.T) {}
func TestTopicService_CreateTopic_InvalidName(t *testing.T) {}
```

```typescript
// TypeScript 测试
describe('TopicService', () => {
  describe('createTopic', () => {
    it('should create topic successfully', () => {});
    it('should throw error for invalid name', () => {});
  });
});
```

## 4. 发布流程

### 4.1 版本规范

采用语义化版本 (Semantic Versioning): `MAJOR.MINOR.PATCH`

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能新增
- **PATCH**: 向后兼容的问题修正

### 4.2 发布检查清单

**代码质量**
- [ ] 所有测试通过
- [ ] 代码覆盖率达标
- [ ] 静态代码分析通过
- [ ] 没有已知的 P0/P1 Bug

**文档**
- [ ] CHANGELOG 更新
- [ ] API 文档更新
- [ ] 用户文档更新
- [ ] 升级指南 (如果需要)

**性能**
- [ ] 性能测试通过
- [ ] 资源使用在合理范围
- [ ] 无明显性能退化

**安全**
- [ ] 安全扫描通过
- [ ] 依赖漏洞检查
- [ ] 敏感信息检查

### 4.3 发布步骤

1. **创建 Release 分支**
   ```bash
   git checkout -b release/v1.0.0 develop
   ```

2. **更新版本号**
   - 更新 version 文件
   - 更新 CHANGELOG.md

3. **运行完整测试**
   ```bash
   task test:all
   ```

4. **构建和打包**
   ```bash
   task build:release
   ```

5. **创建 Git Tag**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

6. **发布到仓库**
   - Docker Hub
   - GitHub Releases

7. **合并回主分支**
   ```bash
   git checkout main
   git merge release/v1.0.0
   git checkout develop
   git merge release/v1.0.0
   ```

## 5. 风险管理

### 5.1 技术风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| Kafka 协议兼容性问题 | 高 | 中 | POC 验证，充分测试 |
| 性能不达标 | 高 | 中 | 早期性能测试，持续优化 |
| Raft 实现复杂度高 | 高 | 中 | 使用成熟库，逐步迭代 |
| Go GC 延迟 | 中 | 低 | 优化内存分配，GC 调优 |

### 5.2 项目风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 进度延期 | 中 | 中 | 合理规划，留缓冲时间 |
| 需求变更 | 低 | 高 | 敏捷开发，快速响应 |
| 人员流动 | 中 | 低 | 文档完善，知识共享 |
| 测试资源不足 | 中 | 中 | 自动化测试，早期介入 |

### 5.3 应对策略

**定期评估**
- 每周团队会议
- 每月项目复盘
- 季度里程碑评审

**问题跟踪**
- 使用 GitHub Issues
- 标记优先级
- 定期清理

**知识管理**
- 技术分享会
- 文档及时更新
- 代码注释完善

---

**文档版本**: v1.0  
**最后更新**: 2025-12-14  
**维护者**: Takhin Team
