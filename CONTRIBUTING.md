# Takhin 贡献指南 / Contributing Guide

欢迎为 Takhin 项目做出贡献！本指南将帮助你了解如何为项目贡献代码、报告问题和提出建议。

Welcome to contribute to Takhin! This guide will help you understand how to contribute code, report issues, and make suggestions.

## 📋 目录 / Table of Contents

- [开发环境搭建](#开发环境搭建)
- [代码规范](#代码规范)
- [测试规范](#测试规范)
- [提交规范](#提交规范)
- [PR 流程](#pr-流程)
- [架构说明](#架构说明)
- [社区准则](#社区准则)

---

## 🛠️ 开发环境搭建

### 前置要求

**Backend (Go):**
- Go 1.23 或更高版本
- Task (任务运行器) - [安装指南](https://taskfile.dev/installation/)
- golangci-lint - [安装指南](https://golangci-lint.run/usage/install/)
- Git 2.0+

**Frontend (React/TypeScript):**
- Node.js >= 18.0.0
- npm >= 9.0.0

**可选工具:**
- Docker & Docker Compose (用于容器化开发和测试)
- kubectl (用于 Kubernetes 部署)
- Make (部分脚本使用)

### 安装步骤

#### 1. 克隆仓库

```bash
git clone https://github.com/takhin-data/takhin.git
cd takhin
```

#### 2. 安装 Task

**macOS:**
```bash
brew install go-task/tap/go-task
```

**Linux:**
```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

**验证安装:**
```bash
task --version
```

#### 3. 设置开发环境

```bash
# 安装所有依赖（前后端）
task dev:setup

# 或者分别安装
task backend:deps    # 安装 Go 依赖
task frontend:deps   # 安装 Node.js 依赖
```

#### 4. 验证环境

**Backend:**
```bash
cd backend
go version              # 应该是 1.23+
golangci-lint --version # 验证 linter 已安装
go test -short ./...    # 运行快速测试
```

**Frontend:**
```bash
cd frontend
node --version          # 应该是 18.0+
npm --version           # 应该是 9.0+
npm run type-check      # TypeScript 类型检查
```

### IDE 配置

#### VS Code (推荐)

安装以下扩展：
- Go (官方 Go 扩展)
- golangci-lint
- ESLint
- Prettier
- TypeScript Vue Plugin (Volar)

**settings.json 配置示例:**
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "goimports",
  "editor.formatOnSave": true,
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSave": true,
  "[typescript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "[typescriptreact]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  }
}
```

#### GoLand / IntelliJ IDEA

1. 打开项目后，IDE 会自动检测 Go 模块
2. 启用 golangci-lint：Preferences → Tools → File Watchers → 添加 golangci-lint
3. 配置 goimports 格式化：Preferences → Go → On Save → Run goimports

### 配置文件

复制示例配置文件：
```bash
cp backend/configs/takhin.yaml backend/configs/takhin-dev.yaml
```

根据需要修改开发配置，例如：
```yaml
server:
  host: "127.0.0.1"
  port: 9092

storage:
  data:
    dir: "/tmp/takhin-dev"

logging:
  level: "debug"
```

---

## 📐 代码规范

### Go 代码规范

#### 基本原则

1. **遵循官方指南**
   - [Effective Go](https://go.dev/doc/effective_go)
   - [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
   - [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

2. **代码风格**
   - 使用 `gofmt` 和 `goimports` 格式化代码
   - 所有导出的函数、类型必须有文档注释
   - 注释使用完整的英文句子，以大写字母开头，句号结尾
   - 保持函数简短（< 100 行），职责单一

3. **命名规范**
   ```go
   // ✅ Good: 清晰、简洁的命名
   type TopicManager struct {
       dataDir string
       topics  map[string]*Topic
   }
   
   func (m *TopicManager) CreateTopic(name string, partitions int) error {
       // ...
   }
   
   // ❌ Bad: 冗余的命名
   type TopicManagerStruct struct {
       topicDataDirectory string
       mapOfTopics        map[string]*Topic
   }
   
   func (m *TopicManagerStruct) CreateTopicWithName(topicName string, numberOfPartitions int) error {
       // ...
   }
   ```

4. **错误处理**
   ```go
   // ✅ Good: 包装错误，提供上下文
   if err := segment.Append(offset, data); err != nil {
       return fmt.Errorf("failed to append to segment at offset %d: %w", offset, err)
   }
   
   // ✅ Good: 定义哨兵错误
   var (
       ErrTopicExists    = errors.New("topic already exists")
       ErrTopicNotFound  = errors.New("topic not found")
       ErrInvalidOffset  = errors.New("invalid offset")
   )
   
   // ❌ Bad: 丢失错误上下文
   if err := segment.Append(offset, data); err != nil {
       return err
   }
   
   // ❌ Bad: 忽略错误
   _ = segment.Close()
   ```

5. **并发安全**
   ```go
   // ✅ Good: 使用互斥锁保护共享状态
   type TopicManager struct {
       mu     sync.RWMutex
       topics map[string]*Topic
   }
   
   func (m *TopicManager) GetTopic(name string) (*Topic, bool) {
       m.mu.RLock()
       defer m.mu.RUnlock()
       topic, ok := m.topics[name]
       return topic, ok
   }
   
   // ✅ Good: 使用通道进行通信
   func (p *Producer) Send(msg *Message) error {
       select {
       case p.msgCh <- msg:
           return nil
       case <-p.stopCh:
           return ErrProducerStopped
       case <-time.After(5 * time.Second):
           return ErrTimeout
       }
   }
   ```

#### 项目特定规范

1. **Kafka 协议处理**
   ```go
   // 所有 handler 函数必须遵循此签名
   func (h *Handler) HandleProduce(ctx context.Context, req *protocol.ProduceRequest) (*protocol.ProduceResponse, error) {
       // 1. 验证请求
       if err := req.Validate(); err != nil {
           return nil, fmt.Errorf("invalid request: %w", err)
       }
       
       // 2. 处理业务逻辑
       // ...
       
       // 3. 构造响应
       resp := &protocol.ProduceResponse{
           // ...
       }
       return resp, nil
   }
   ```

2. **存储层操作**
   ```go
   // 所有存储操作必须使用 defer Close()
   func (l *Log) Read(offset int64, maxBytes int) ([]byte, error) {
       l.mu.RLock()
       defer l.mu.RUnlock()
       
       segment := l.findSegment(offset)
       if segment == nil {
           return nil, ErrOffsetOutOfRange
       }
       
       return segment.Read(offset, maxBytes)
   }
   ```

3. **配置使用**
   ```go
   // 使用 Koanf 配置，支持 YAML + 环境变量
   type Config struct {
       Server   ServerConfig   `koanf:"server"`
       Storage  StorageConfig  `koanf:"storage"`
       Kafka    KafkaConfig    `koanf:"kafka"`
   }
   
   // 环境变量使用 TAKHIN_ 前缀
   // 例如: TAKHIN_SERVER_PORT=9093
   ```

4. **日志记录**
   ```go
   // 使用结构化日志 (slog)
   logger.Info("topic created",
       "topic", topicName,
       "partitions", numPartitions,
       "replication_factor", replFactor)
   
   logger.Error("failed to append message",
       "topic", topicName,
       "partition", partition,
       "error", err)
   ```

#### Linter 配置

项目使用 `.golangci.yml` 配置了以下 linters：

- **errcheck**: 检查未处理的错误
- **gosimple**: 简化代码
- **govet**: 静态分析
- **gocyclo**: 圈复杂度检查 (阈值 15)
- **gosec**: 安全漏洞检查
- **revive**: 代码风格检查
- 更多详见 `backend/.golangci.yml`

运行 linter：
```bash
task backend:lint
```

自动修复（部分问题）：
```bash
task backend:fmt
```

### TypeScript/React 代码规范

#### 基本原则

1. **TypeScript 严格模式**
   ```typescript
   // tsconfig.json
   {
     "compilerOptions": {
       "strict": true,
       "noImplicitAny": true,
       "strictNullChecks": true
     }
   }
   ```

2. **组件规范**
   ```typescript
   // ✅ Good: 函数组件 + TypeScript
   interface TopicListProps {
     topics: Topic[];
     onSelect: (topic: Topic) => void;
   }
   
   export const TopicList: React.FC<TopicListProps> = ({ topics, onSelect }) => {
     return (
       <div>
         {topics.map(topic => (
           <TopicCard key={topic.name} topic={topic} onClick={() => onSelect(topic)} />
         ))}
       </div>
     );
   };
   ```

3. **API 调用**
   ```typescript
   // 使用封装的 API 客户端
   import { api } from '@/api';
   
   const fetchTopics = async () => {
     try {
       const topics = await api.topics.list();
       setTopics(topics);
     } catch (error) {
       console.error('Failed to fetch topics:', error);
       toast.error('无法加载主题列表');
     }
   };
   ```

4. **样式规范**
   - 使用 Tailwind CSS 工具类
   - 组件样式保持一致
   - 响应式设计优先

运行检查：
```bash
task frontend:lint       # ESLint 检查
task frontend:format     # Prettier 格式化
task frontend:type-check # TypeScript 类型检查
```

---

## 🧪 测试规范

### Go 测试规范

#### 测试文件结构

```go
// pkg/storage/log/segment_test.go
package log

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSegment_Append(t *testing.T) {
    // 使用表驱动测试
    tests := []struct {
        name    string
        offset  int64
        data    []byte
        wantErr bool
    }{
        {
            name:    "append valid message",
            offset:  0,
            data:    []byte("test message"),
            wantErr: false,
        },
        {
            name:    "append with negative offset",
            offset:  -1,
            data:    []byte("test"),
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup: 使用 t.TempDir() 创建临时目录
            tmpDir := t.TempDir()
            segment, err := NewSegment(tmpDir, 0, 1024*1024)
            require.NoError(t, err)
            defer segment.Close()
            
            // Execute
            err = segment.Append(tt.offset, tt.data)
            
            // Assert
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                // 验证数据确实写入
                data, err := segment.Read(tt.offset, len(tt.data))
                assert.NoError(t, err)
                assert.Equal(t, tt.data, data)
            }
        })
    }
}
```

#### 测试覆盖率要求

- **新代码**: ≥ 80% 覆盖率
- **核心模块**: ≥ 90% 覆盖率（storage, kafka/handler, raft）
- **工具函数**: ≥ 70% 覆盖率

运行测试：
```bash
# 运行所有测试（带竞态检测）
task backend:test

# 仅运行单元测试（跳过集成测试）
task backend:test:unit

# 查看覆盖率报告
task backend:coverage

# 运行特定包的测试
cd backend
go test -v -race ./pkg/storage/...
```

#### 测试类型

1. **单元测试** (70%)
   - 测试单个函数/方法
   - 使用 mock 隔离依赖
   - 快速执行（< 1s）

2. **集成测试** (20%)
   ```go
   // 使用 build tag
   // +build integration
   
   package integration
   
   func TestKafkaProduceConsume(t *testing.T) {
       // 启动真实的 Takhin 实例
       // 使用真实的存储
       // 测试端到端流程
   }
   ```

3. **基准测试** (10%)
   ```go
   func BenchmarkSegment_Append(b *testing.B) {
       tmpDir := b.TempDir()
       segment, _ := NewSegment(tmpDir, 0, 1024*1024)
       defer segment.Close()
       
       data := make([]byte, 1024)
       
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           _ = segment.Append(int64(i), data)
       }
   }
   ```

   运行基准测试：
   ```bash
   task backend:bench        # 完整基准测试
   task backend:bench:quick  # 快速基准测试
   ```

#### Mock 使用

```go
// 使用 testify/mock
type MockTopicManager struct {
    mock.Mock
}

func (m *MockTopicManager) CreateTopic(name string, partitions int) error {
    args := m.Called(name, partitions)
    return args.Error(0)
}

func TestHandler_CreateTopics(t *testing.T) {
    mockMgr := new(MockTopicManager)
    mockMgr.On("CreateTopic", "test-topic", 3).Return(nil)
    
    handler := NewHandler(mockMgr)
    err := handler.CreateTopics([]string{"test-topic"}, 3)
    
    assert.NoError(t, err)
    mockMgr.AssertExpectations(t)
}
```

### Frontend 测试规范

```typescript
// 使用 Vitest + React Testing Library
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { TopicList } from './TopicList';

describe('TopicList', () => {
  it('renders topics correctly', () => {
    const topics = [
      { name: 'topic-1', partitions: 3 },
      { name: 'topic-2', partitions: 5 },
    ];
    
    render(<TopicList topics={topics} onSelect={vi.fn()} />);
    
    expect(screen.getByText('topic-1')).toBeInTheDocument();
    expect(screen.getByText('topic-2')).toBeInTheDocument();
  });
  
  it('calls onSelect when topic is clicked', () => {
    const onSelect = vi.fn();
    const topics = [{ name: 'topic-1', partitions: 3 }];
    
    render(<TopicList topics={topics} onSelect={onSelect} />);
    
    fireEvent.click(screen.getByText('topic-1'));
    
    expect(onSelect).toHaveBeenCalledWith(topics[0]);
  });
});
```

---

## 📝 提交规范

### Commit Message 格式

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <subject>

<body>

<footer>
```

#### Type 类型

- **feat**: 新功能
- **fix**: Bug 修复
- **docs**: 文档更新
- **style**: 代码格式（不影响功能）
- **refactor**: 重构（既不是新功能也不是修复）
- **perf**: 性能优化
- **test**: 测试相关
- **chore**: 构建/工具链相关
- **ci**: CI/CD 相关

#### Scope 范围

- **kafka**: Kafka 协议相关
- **storage**: 存储引擎
- **raft**: Raft 共识
- **console**: Console API/UI
- **config**: 配置管理
- **auth**: 认证/授权
- **metrics**: 监控指标
- **docs**: 文档

#### 示例

```bash
# 新功能
feat(kafka): add support for Kafka protocol v3.0
feat(console): add topic creation UI

# Bug 修复
fix(storage): prevent data corruption on crash
fix(raft): handle leader election timeout correctly

# 文档
docs(api): update REST API documentation
docs(readme): add deployment guide

# 性能优化
perf(storage): optimize zero-copy read path
perf(kafka): batch message processing

# 重构
refactor(handler): simplify error handling logic
refactor(console): extract API client to separate package

# 测试
test(storage): add integration tests for log compaction
test(kafka): improve handler test coverage

# Breaking Change (使用 !)
feat(api)!: change REST API endpoint structure

BREAKING CHANGE: REST API endpoints now use /api/v2 prefix
```

### Git 工作流

#### 1. Fork 并克隆仓库

```bash
# Fork 仓库到你的 GitHub 账号
# 然后克隆你的 fork
git clone https://github.com/YOUR_USERNAME/takhin.git
cd takhin

# 添加上游仓库
git remote add upstream https://github.com/takhin-data/takhin.git
```

#### 2. 创建功能分支

```bash
# 从 develop 分支创建新分支
git checkout develop
git pull upstream develop
git checkout -b feature/your-feature-name

# 分支命名规范
# feature/  - 新功能
# fix/      - Bug 修复
# refactor/ - 重构
# docs/     - 文档
# test/     - 测试

# 示例
git checkout -b feature/add-kafka-transactions
git checkout -b fix/storage-data-race
git checkout -b docs/contributing-guide
```

#### 3. 开发和提交

```bash
# 进行修改
# ...

# 运行测试和检查
task dev:check

# 暂存修改
git add .

# 提交（遵循 Conventional Commits）
git commit -m "feat(kafka): add transaction coordinator"

# 如果提交信息较长，可以使用编辑器
git commit
```

#### 4. 保持分支更新

```bash
# 定期同步上游更新
git fetch upstream
git rebase upstream/develop

# 如果有冲突，解决后继续
git rebase --continue

# 或放弃 rebase
git rebase --abort
```

#### 5. 推送到你的 Fork

```bash
git push origin feature/your-feature-name

# 如果进行了 rebase，需要强制推送
git push -f origin feature/your-feature-name
```

---

## 🔄 PR 流程

### 创建 Pull Request

#### 1. 提交 PR 前检查清单

- [ ] 代码已通过所有测试：`task backend:test`
- [ ] 代码已通过 linter：`task backend:lint`
- [ ] 代码已格式化：`task backend:fmt`
- [ ] 测试覆盖率达标（≥ 80%）
- [ ] 更新了相关文档
- [ ] Commit message 遵循规范
- [ ] 解决了所有合并冲突

#### 2. PR 标题和描述

**标题格式**（与 commit message 相同）：
```
<type>(<scope>): <description>
```

**描述模板**：
```markdown
## 📝 变更说明

简要描述本 PR 的目的和实现方式。

## 🎯 相关 Issue

Closes #123
Related to #456

## 🔨 变更内容

- [ ] 添加了 XXX 功能
- [ ] 修复了 YYY 问题
- [ ] 重构了 ZZZ 模块

## 🧪 测试

### 测试方法
描述如何测试这些变更。

### 测试结果
```bash
task backend:test
# 粘贴测试输出
```

## 📸 截图（如果有 UI 变更）

Before:
![before](url)

After:
![after](url)

## ✅ Checklist

- [ ] 代码已通过所有测试
- [ ] 代码已通过 linter 检查
- [ ] 更新了相关文档
- [ ] 添加了测试用例
- [ ] 更新了 CHANGELOG（如果需要）

## 💭 其他说明

补充任何其他相关信息。
```

#### 3. PR 标签

维护者会添加以下标签：

- **Type**: `feature`, `bugfix`, `documentation`, `enhancement`
- **Priority**: `P0-critical`, `P1-high`, `P2-medium`, `P3-low`
- **Status**: `in-review`, `needs-changes`, `approved`, `merged`
- **Component**: `backend`, `frontend`, `docs`, `ci/cd`

### Code Review 流程

#### 1. 自动检查

PR 提交后会自动运行 CI/CD 流程：
- ✅ Lint 检查
- ✅ 单元测试
- ✅ 集成测试
- ✅ 代码覆盖率检查
- ✅ 安全扫描

所有检查必须通过才能合并。

#### 2. 人工审查

至少需要 **2 名维护者** 批准 PR。

**审查重点**：
- 代码质量和可维护性
- 测试覆盖率和质量
- 性能影响
- 安全性考虑
- 文档完整性
- 向后兼容性

#### 3. 响应反馈

**收到审查意见后**：
```bash
# 1. 进行修改
# 2. 提交新的 commit（不要 squash）
git add .
git commit -m "fix: address review comments"
git push origin feature/your-feature-name

# 3. 在 PR 中回复审查意见
```

**常见审查意见示例**：
- "请添加错误处理"
- "这里可能有并发安全问题"
- "建议添加单元测试"
- "文档需要更新"

#### 4. 合并前准备

**维护者合并前会进行**：
- Squash commits（可选，保持历史清晰）
- 确认 CI/CD 全部通过
- 更新 CHANGELOG
- 标记版本号（如果需要）

### PR 合并策略

- **main 分支**: 仅接受来自 develop 的 merge
- **develop 分支**: 接受所有功能和修复的 PR
- **hotfix 分支**: 紧急修复可以直接合并到 main

```
main (生产)
  ↑
  └── develop (开发)
        ↑
        ├── feature/xxx
        ├── fix/yyy
        └── refactor/zzz
```

---

## 🏗️ 架构说明

### 项目结构

```
Takhin/
├── backend/              # Takhin Core (Go)
│   ├── cmd/             # 主程序入口
│   │   ├── takhin/      # Kafka 服务器
│   │   ├── console/     # Console 服务器
│   │   └── takhin-debug/# 调试工具
│   ├── pkg/             # 公共包
│   │   ├── kafka/       # Kafka 协议实现
│   │   │   ├── protocol/   # 二进制协议编解码
│   │   │   ├── handler/    # 请求处理器
│   │   │   └── server/     # TCP 服务器
│   │   ├── storage/     # 存储引擎
│   │   │   ├── log/        # Log segment 管理
│   │   │   └── topic/      # Topic 和 partition 管理
│   │   ├── coordinator/ # Consumer group 协调器
│   │   ├── raft/        # Raft 共识算法
│   │   ├── console/     # Console REST API
│   │   ├── grpcapi/     # gRPC API
│   │   ├── config/      # 配置管理 (Koanf)
│   │   ├── logger/      # 结构化日志 (slog)
│   │   └── metrics/     # Prometheus 指标
│   ├── configs/         # 配置文件
│   │   └── takhin.yaml
│   └── scripts/         # 脚本工具
├── frontend/            # Takhin Console (React)
│   ├── src/
│   │   ├── api/         # API 客户端
│   │   ├── components/  # React 组件
│   │   ├── pages/       # 页面组件
│   │   ├── types/       # TypeScript 类型
│   │   └── utils/       # 工具函数
│   └── public/          # 静态资源
├── docs/                # 文档
│   ├── architecture/    # 架构设计
│   ├── implementation/  # 实现细节
│   └── testing/         # 测试策略
├── scripts/             # 项目脚本
└── Taskfile.yaml        # 任务定义
```

### 核心组件交互

```
┌─────────────┐
│   Client    │
│  (Producer/ │
│  Consumer)  │
└──────┬──────┘
       │ Kafka Protocol
       ↓
┌─────────────────────────────────────┐
│         Takhin Core                 │
├─────────────────────────────────────┤
│                                     │
│  ┌───────────┐    ┌──────────────┐ │
│  │  Handler  │───→│    Backend   │ │
│  │  (Kafka)  │    │  (Interface) │ │
│  └───────────┘    └──────┬───────┘ │
│                          │          │
│         ┌────────────────┴────┐    │
│         ↓                     ↓    │
│  ┌─────────────┐      ┌──────────┐│
│  │Topic Manager│      │Coordinator││
│  │  (Storage)  │      │ (Groups) ││
│  └─────────────┘      └──────────┘│
│         │                          │
│         ↓                          │
│  ┌─────────────┐                  │
│  │ Log Segment │                  │
│  │  (Disk I/O) │                  │
│  └─────────────┘                  │
│                                     │
└─────────────────────────────────────┘
```

### 关键设计决策

#### 1. Kafka 协议处理

```go
// pkg/kafka/handler/handler.go
type Handler struct {
    config      *config.Config
    topicMgr    *topic.Manager
    coordinator *coordinator.Coordinator
}

// 所有 Kafka API 的入口
func (h *Handler) Handle(ctx context.Context, apiKey int16, req []byte) ([]byte, error) {
    switch apiKey {
    case protocol.APIKeyProduce:
        return h.handleProduce(ctx, req)
    case protocol.APIKeyFetch:
        return h.handleFetch(ctx, req)
    // ... 其他 API
    }
}
```

#### 2. 存储层设计

- **Topic**: 逻辑容器，包含多个 partition
- **Partition**: 物理存储单元，由多个 segment 组成
- **Segment**: 固定大小的日志文件（默认 1GB）

```
/data/
  ├── topic-1-0/           # Topic "topic-1", Partition 0
  │   ├── 00000000000000000000.log   # Segment
  │   ├── 00000000000000000000.index
  │   ├── 00000000000001000000.log
  │   └── 00000000000001000000.index
  └── topic-1-1/           # Topic "topic-1", Partition 1
      └── ...
```

#### 3. Consumer Group 协调

- 使用单独的 Coordinator 管理所有 consumer group
- 支持 Range 和 RoundRobin 分区策略
- 实现完整的 rebalance 协议

#### 4. 配置管理

使用 Koanf 支持多层配置：
1. YAML 配置文件
2. 环境变量（`TAKHIN_` 前缀）
3. 命令行参数

```go
// 配置优先级：命令行 > 环境变量 > 配置文件
cfg, err := config.Load("configs/takhin.yaml")
```

### 性能考虑

- **零拷贝 I/O**: 使用 `sendfile()` 系统调用
- **批量处理**: 批量写入和读取消息
- **并发控制**: 使用 goroutine pool 限制并发
- **内存池**: 复用缓冲区，减少 GC 压力

### 监控和可观测性

- **Metrics**: Prometheus 格式，暴露在 `/metrics`
- **Logging**: 结构化日志（slog）
- **Tracing**: 预留 OpenTelemetry 集成

---

## 🤝 社区准则

### 行为准则

我们致力于提供友好、安全和包容的环境。所有参与者都应：

- ✅ 保持尊重和专业
- ✅ 接受建设性批评
- ✅ 关注对社区最有利的事情
- ✅ 对其他社区成员表示同理心

请勿：
- ❌ 使用性化的语言或图像
- ❌ 进行人身攻击或侮辱
- ❌ 骚扰他人
- ❌ 发布他人的私人信息

### 报告问题

#### Bug 报告

使用 [Bug Report](https://github.com/takhin-data/takhin/issues/new?template=bug_report.md) 模板：

```markdown
**描述问题**
清晰简洁地描述 bug。

**复现步骤**
1. 执行 '...'
2. 访问 '...'
3. 看到错误

**期望行为**
描述你期望发生的行为。

**实际行为**
描述实际发生的行为。

**环境**
- OS: [e.g. Ubuntu 22.04]
- Go 版本: [e.g. 1.23.0]
- Takhin 版本: [e.g. v0.1.0]

**日志**
```
粘贴相关日志
```

**其他信息**
其他有助于理解问题的信息。
```

#### 功能请求

使用 [Feature Request](https://github.com/takhin-data/takhin/issues/new?template=feature_request.md) 模板：

```markdown
**功能描述**
简要描述你想要的功能。

**使用场景**
描述这个功能在什么情况下有用。

**建议方案**
如果有具体的实现想法，请描述。

**替代方案**
考虑过哪些替代方案。

**其他信息**
其他相关的信息或截图。
```

### 获取帮助

- 📖 **文档**: [docs/](docs/)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/takhin-data/takhin/discussions)
- 🐛 **Issues**: [GitHub Issues](https://github.com/takhin-data/takhin/issues)
- 📧 **邮件**: takhin-dev@example.com (邮件列表)

### 成为维护者

活跃贡献者可以被邀请成为维护者。要求：
- 持续贡献 3 个月以上
- 提交至少 10 个高质量 PR
- 参与 code review 和社区讨论
- 熟悉项目架构和代码规范

---

## 📚 参考资源

### 项目文档

- [架构设计](docs/architecture/)
- [实现细节](docs/implementation/)
- [测试策略](docs/testing/)
- [API 文档](docs/api/)

### 技术文档

- [Kafka Protocol](https://kafka.apache.org/protocol)
- [Raft Consensus](https://raft.github.io/)
- [Go Documentation](https://go.dev/doc/)
- [React Documentation](https://react.dev/)

### 开发工具

- [Task](https://taskfile.dev/) - 任务运行器
- [golangci-lint](https://golangci-lint.run/) - Go linter
- [Testify](https://github.com/stretchr/testify) - 测试框架
- [Koanf](https://github.com/knadh/koanf) - 配置管理

---

## 🙏 致谢

感谢所有为 Takhin 做出贡献的开发者！

查看完整的贡献者列表：[Contributors](https://github.com/takhin-data/takhin/graphs/contributors)

---

## 📄 许可证

本项目采用 Apache License 2.0 许可证。查看 [LICENSE](LICENSE) 文件了解详情。

贡献代码即表示你同意将你的贡献以 Apache License 2.0 许可证授权。

---

**Happy Contributing! 🎉**

如有任何问题，请随时在 [Discussions](https://github.com/takhin-data/takhin/discussions) 中提问。
