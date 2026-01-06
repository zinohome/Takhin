# Takhin - High-Performance Kafka-Compatible Streaming Platform

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Backend CI](https://github.com/takhin-data/takhin/workflows/Backend%20CI/badge.svg)](https://github.com/takhin-data/takhin/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/takhin-data/takhin)](https://goreportcard.com/report/github.com/takhin-data/takhin)

Takhin 是一个使用 Go 和 React 重写的高性能、Kafka 兼容的流式数据平台。

## 📋 项目简介

Takhin 包含两个核心组件：

- **Takhin Core** - 使用 Go 实现的 Kafka 兼容流式引擎
- **Takhin Console** - 使用 React 实现的 Web 管理界面

## ✨ 主要特性

### Takhin Core
- ✅ Kafka 协议兼容 (0.11.x+)
- 🚀 高性能存储引擎（零拷贝 I/O）
- 🔄 Raft 共识算法（无需 ZooKeeper）
- 📊 内置 Prometheus 指标
- 🔐 TLS 加密和身份验证
- 🎯 低延迟（P99 < 10ms）
- ⚡ 高吞吐量（>100K msg/s）

### Takhin Console
- 🎨 现代化 Web UI
- 📈 实时监控和指标
- 🔍 消息查看和过滤
- 👥 主题和消费组管理
- ⚙️ 集群配置管理
- 📱 响应式设计

## 🚀 快速开始

### 前置要求

**Backend:**
- Go 1.23 或更高版本
- Task（任务运行器）

**Frontend:**
- Node.js >= 18.0.0
- npm >= 9.0.0

### 安装 Task

```bash
# macOS
brew install go-task/tap/go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

### 构建和运行

**Backend:**

```bash
# 设置开发环境
task dev:setup

# 构建
task backend:build

# 运行测试
task backend:test

# 运行服务
task backend:run
```

**Frontend:**

```bash
# 安装依赖
task frontend:deps

# 开发模式（http://localhost:3000）
task frontend:dev

# 生产构建
task frontend:build

# 预览生产构建
task frontend:preview
```

**同时运行前后端:**

```bash
task dev:all
```

### 使用 Docker

```bash
# 构建 Docker 镜像
task docker:build

# 运行容器
task docker:run
```

## 📖 文档

完整文档请查看 [docs/](docs/) 目录：

- [架构设计](docs/architecture/)
- [实施计划](docs/implementation/)
- [测试策略](docs/testing/)
- [质量控制](docs/quality/)

## 🛠️ 开发

### 项目结构

```
Takhin/
├── backend/              # Takhin Core (Go)
│   ├── cmd/             # 主程序入口
│   ├── pkg/             # 可复用的包
│   ├── internal/        # 私有代码
│   └── configs/         # 配置文件
├── frontend/            # Takhin Console (React + TypeScript)
│   ├── src/             # 源代码
│   │   ├── api/        # API 客户端
│   │   ├── components/ # React 组件
│   │   ├── layouts/    # 布局组件
│   │   ├── pages/      # 页面组件
│   │   ├── types/      # TypeScript 类型
│   │   └── utils/      # 工具函数
│   ├── public/          # 静态资源
│   └── README.md        # 前端文档
├── docs/                # 文档
├── projects/            # 参考项目
└── Taskfile.yaml        # 任务定义
```

### 运行测试

```bash
# 运行所有测试
task backend:test

# 仅运行单元测试
task backend:test:unit

# 查看测试覆盖率
task backend:coverage
```

### 代码检查

**Backend:**

```bash
# 格式化代码
task backend:fmt

# 运行 linter
task backend:lint

# 运行所有检查
task dev:check
```

**Frontend:**

```bash
# 格式化代码
task frontend:format

# 运行 linter
task frontend:lint

# 修复 lint 问题
task frontend:lint:fix

# TypeScript 类型检查
task frontend:type-check
```
task backend:fmt

# 运行 linter
task backend:lint

# 运行所有检查
task dev:check
```

## 🤝 贡献

欢迎为 Takhin 做出贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详细的贡献指南。

### 快速开始

```bash
# 1. Fork 并克隆仓库
git clone https://github.com/YOUR_USERNAME/takhin.git
cd takhin

# 2. 创建功能分支
git checkout -b feature/your-feature-name

# 3. 进行修改并测试
task dev:check

# 4. 提交代码
git commit -m "feat(scope): your change description"

# 5. 推送并创建 PR
git push origin feature/your-feature-name
```

### 开发规范

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 测试覆盖率 ≥ 80%（核心模块 ≥ 90%）
- 所有代码必须通过 golangci-lint 检查
- 使用 Conventional Commits 提交消息
- 查看 [快速参考](TASK_7.3_QUICK_REFERENCE.md) 了解常用命令

## 📊 开发进度

当前阶段：**Phase 2 - Sprint 9-10 完成**

### 已完成 ✅
- [x] 项目结构和配置
- [x] 配置管理模块  
- [x] 日志系统
- [x] CI/CD 流水线
- [x] 基础 Kafka 协议实现 (Produce, Fetch, Metadata, ApiVersions)
- [x] 存储引擎 (Log Segment, Partition, Topic Manager)
- [x] Raft 共识算法 (FSM, Leader 选举, 日志复制, 3 节点集群测试)
- [x] Consumer Group 完整实现 (7 个 API, Coordinator, Rebalance)
- [x] 压缩支持 (5 种压缩类型: None, GZIP, Snappy, LZ4, ZSTD)
- [x] Admin API (CreateTopics, DeleteTopics, DescribeConfigs)

### 进行中 🚧
- [ ] Console 后端开发 (gRPC API, REST API)
- [ ] Console 前端开发 (React + TypeScript)

### 计划中 📋
- [ ] Transactions 支持 (Exactly-Once Semantics)
- [ ] 更多 Admin API (AlterConfigs, ListGroups)
- [ ] ACL 和安全认证
- [ ] Schema Registry 集成

查看完整的 [开发计划](docs/implementation/project-plan.md)。

### 核心功能状态

| 功能 | 状态 | 覆盖率 | 文档 |
|------|------|--------|------|
| Kafka Protocol | ✅ | 95% | [Handler](backend/pkg/kafka/handler) |
| 存储引擎 | ✅ | 90% | [Storage](backend/pkg/storage) |
| Raft 共识 | ✅ | 85% | [Raft Summary](docs/raft-cluster-test-summary.md) |
| Consumer Group | ✅ | 100% | [Consumer Group Summary](docs/consumer-group-summary.md) |
| 压缩 | ✅ | 95% | [Compression](docs/implementation/compression.md) |
| Admin API | ✅ | 100% | [Admin API](docs/admin-api.md) |
| Transactions | 📋 | - | [Design Doc](docs/transactions-design.md) |

## 📄 许可证

本项目采用 Apache License 2.0 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

本项目参考了以下优秀项目：

- [Redpanda](https://github.com/redpanda-data/redpanda) - 高性能 Kafka 兼容平台
- [Apache Kafka](https://kafka.apache.org/) - 分布式流式平台

## 📧 联系方式

- 项目主页：https://github.com/takhin-data/takhin
- 问题反馈：https://github.com/takhin-data/takhin/issues

---

**注意：** 本项目目前处于早期开发阶段，不建议用于生产环境。
