# Takhin 开发者快速参考

## ⚡ 快速开始

### 环境要求
```bash
# Backend
Go >= 1.23
Task >= 3.0
golangci-lint

# Frontend
Node.js >= 18.0
npm >= 9.0
```

### 首次设置
```bash
# 1. 克隆仓库
git clone https://github.com/takhin-data/takhin.git
cd takhin

# 2. 安装依赖
task dev:setup

# 3. 运行测试
task backend:test
task frontend:lint

# 4. 启动开发
task backend:run    # 终端 1
task frontend:dev   # 终端 2
```

## 📝 常用命令

### Backend 开发
```bash
# 构建
task backend:build

# 测试
task backend:test           # 完整测试（带竞态检测）
task backend:test:unit      # 仅单元测试
task backend:coverage       # 查看覆盖率报告

# 代码质量
task backend:fmt            # 格式化代码
task backend:lint           # 运行 linter
task dev:check              # 运行所有检查

# 运行
task backend:run            # 使用默认配置运行
cd backend && go run ./cmd/takhin -config configs/takhin-dev.yaml

# 调试
task backend:debug          # 生成调试信息包
```

### Frontend 开发
```bash
# 开发
task frontend:dev           # 启动开发服务器（:3000）

# 构建
task frontend:build         # 生产构建
task frontend:preview       # 预览生产构建

# 代码质量
task frontend:lint          # ESLint 检查
task frontend:lint:fix      # 自动修复 lint 问题
task frontend:format        # Prettier 格式化
task frontend:type-check    # TypeScript 类型检查
```

### Docker 开发
```bash
task docker:build           # 构建镜像
task docker:run             # 运行容器
```

## 🔀 Git 工作流

### 创建功能分支
```bash
git checkout develop
git pull upstream develop
git checkout -b <type>/<name>

# 分支类型
# feature/  - 新功能
# fix/      - Bug 修复
# refactor/ - 重构
# docs/     - 文档
# test/     - 测试
```

### 提交代码
```bash
# 1. 运行检查
task dev:check

# 2. 暂存修改
git add .

# 3. 提交（遵循 Conventional Commits）
git commit -m "<type>(<scope>): <description>"

# 常用 type
# feat, fix, docs, style, refactor, perf, test, chore, ci
```

### 提交示例
```bash
# 新功能
git commit -m "feat(kafka): add support for Kafka protocol v3.0"
git commit -m "feat(console): add topic creation UI"

# Bug 修复
git commit -m "fix(storage): prevent data corruption on crash"
git commit -m "fix(raft): handle leader election timeout"

# 文档
git commit -m "docs(api): update REST API documentation"

# 测试
git commit -m "test(storage): add integration tests for compaction"

# 性能优化
git commit -m "perf(storage): optimize zero-copy read path"

# 重大变更
git commit -m "feat(api)!: change REST API endpoint structure

BREAKING CHANGE: REST API endpoints now use /api/v2 prefix"
```

### 推送和创建 PR
```bash
# 推送到你的 fork
git push origin <branch-name>

# 在 GitHub 上创建 PR
# - 标题格式同 commit message
# - 填写 PR 描述模板
# - 添加相关 issue 链接
```

## 🧪 测试指南

### 表驱动测试模板
```go
func TestXxx(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid case",
            input:   "test",
            want:    "expected",
            wantErr: false,
        },
        {
            name:    "invalid case",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            tmpDir := t.TempDir()
            
            // Execute
            got, err := Function(tt.input)
            
            // Assert
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 覆盖率要求
- 新代码: ≥ 80%
- 核心模块: ≥ 90% (storage, kafka/handler, raft)
- 工具函数: ≥ 70%

### 运行特定测试
```bash
# 运行特定包
cd backend
go test -v ./pkg/storage/...

# 运行特定测试
go test -v -run TestSegment_Append ./pkg/storage/log

# 运行基准测试
go test -bench=. -benchmem ./pkg/storage/log
```

## 📐 代码规范速查

### 错误处理
```go
// ✅ Good: 包装错误
if err != nil {
    return fmt.Errorf("failed to X: %w", err)
}

// ✅ Good: 哨兵错误
var ErrNotFound = errors.New("not found")

// ❌ Bad: 忽略错误
_ = f.Close()

// ❌ Bad: 不包装错误
if err != nil {
    return err
}
```

### 并发安全
```go
// ✅ Good: 使用互斥锁
type Manager struct {
    mu   sync.RWMutex
    data map[string]string
}

func (m *Manager) Get(key string) (string, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    val, ok := m.data[key]
    return val, ok
}
```

### 日志记录
```go
// ✅ Good: 结构化日志
logger.Info("topic created",
    "topic", name,
    "partitions", count)

logger.Error("operation failed",
    "operation", "append",
    "error", err)
```

## 🔍 故障排查

### Backend 问题

**测试失败**
```bash
# 清理并重新运行
cd backend
rm -rf coverage.out
go clean -testcache
go test -v ./...
```

**Linter 错误**
```bash
# 查看详细信息
golangci-lint run --verbose

# 自动修复
task backend:fmt
```

**编译错误**
```bash
# 清理并重新构建
go clean -cache
go mod tidy
task backend:build
```

### Frontend 问题

**依赖问题**
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

**类型错误**
```bash
# 检查类型
npm run type-check

# 更新类型定义
npm update @types/react @types/node
```

## 📂 项目结构速查

```
backend/
├── cmd/
│   ├── takhin/          # Kafka 服务器入口
│   ├── console/         # Console 服务器入口
│   └── takhin-debug/    # 调试工具
├── pkg/
│   ├── kafka/           # Kafka 协议实现
│   │   ├── protocol/    # 二进制协议
│   │   ├── handler/     # 请求处理器
│   │   └── server/      # TCP 服务器
│   ├── storage/         # 存储引擎
│   │   ├── log/         # Log segment
│   │   └── topic/       # Topic 管理
│   ├── coordinator/     # Consumer group
│   ├── raft/            # Raft 共识
│   ├── console/         # Console REST API
│   ├── grpcapi/         # gRPC API
│   ├── config/          # 配置管理
│   ├── logger/          # 日志系统
│   └── metrics/         # Prometheus 指标
└── configs/
    └── takhin.yaml      # 配置文件

frontend/
├── src/
│   ├── api/             # API 客户端
│   ├── components/      # React 组件
│   ├── pages/           # 页面组件
│   ├── types/           # TypeScript 类型
│   └── utils/           # 工具函数
└── public/              # 静态资源
```

## 🔗 重要链接

### 文档
- [完整贡献指南](CONTRIBUTING.md)
- [架构设计](docs/architecture/)
- [测试策略](docs/testing/)
- [API 文档](docs/api/)

### 外部资源
- [Effective Go](https://go.dev/doc/effective_go)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Task](https://taskfile.dev/)
- [golangci-lint](https://golangci-lint.run/)

### 社区
- [GitHub Issues](https://github.com/takhin-data/takhin/issues)
- [Discussions](https://github.com/takhin-data/takhin/discussions)

## ✅ PR 检查清单

提交 PR 前确保：
- [ ] `task backend:test` 通过
- [ ] `task backend:lint` 无警告
- [ ] `task backend:fmt` 已执行
- [ ] 测试覆盖率 ≥ 80%
- [ ] 更新了相关文档
- [ ] Commit message 遵循规范
- [ ] 解决了所有合并冲突

## 💡 开发技巧

### 高效开发
1. 使用 `task --list` 查看所有可用命令
2. 运行 `task dev:check` 在提交前检查所有问题
3. 使用 `t.TempDir()` 创建测试临时目录（自动清理）
4. 使用 `go test -short` 跳过慢速测试

### 调试技巧
```bash
# 1. 使用详细日志
TAKHIN_LOGGING_LEVEL=debug task backend:run

# 2. 使用 delve 调试器
cd backend
dlv debug ./cmd/takhin -- -config configs/takhin-dev.yaml

# 3. 查看 Prometheus 指标
curl http://localhost:9090/metrics

# 4. 使用 pprof 性能分析
go tool pprof http://localhost:6060/debug/pprof/profile
```

### 性能优化
- 使用 `go test -bench` 运行基准测试
- 使用 `pprof` 分析性能瓶颈
- 注意零拷贝 I/O 的使用场景
- 使用 goroutine pool 限制并发

---

📚 **详细文档**: [CONTRIBUTING.md](CONTRIBUTING.md)  
❓ **问题反馈**: [GitHub Issues](https://github.com/takhin-data/takhin/issues)  
💬 **社区讨论**: [Discussions](https://github.com/takhin-data/takhin/discussions)
