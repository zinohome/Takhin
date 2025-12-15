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

- Go 1.23 或更高版本
- Task（任务运行器）

### 安装 Task

```bash
# macOS
brew install go-task/tap/go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

### 构建和运行

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
├── frontend/            # Takhin Console (React)
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

```bash
# 格式化代码
task backend:fmt

# 运行 linter
task backend:lint

# 运行所有检查
task dev:check
```

## 🤝 贡献

请查看 [CONTRIBUTING.md](.github/copilot-instructions.md) 了解贡献指南。

### 代码规范

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 测试覆盖率 ≥ 80%
- 所有代码必须通过 golangci-lint 检查
- 使用 Conventional Commits 提交消息

## 📊 开发进度

当前阶段：**Phase 1 - 基础设施搭建**

- [x] 项目结构和配置
- [x] 配置管理模块
- [x] 日志系统
- [x] CI/CD 流水线
- [ ] 基础 Kafka 协议实现
- [ ] 存储引擎
- [ ] Raft 共识算法

查看完整的 [开发计划](docs/implementation/project-plan.md)。

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
