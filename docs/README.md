# Takhin 文档中心

欢迎来到 Takhin 项目文档中心！这里是所有项目文档的入口。

## 📚 快速导航

### 项目概览
- [项目状态报告](PROJECT_STATUS_REPORT.md) - 最新的项目状态和分析
- [项目架构](architecture/) - 系统架构文档
- [贡献指南](../CONTRIBUTING.md) - 如何参与项目

### 开发指南
- [开发环境搭建](guides/development/setup.md)
- [测试指南](guides/development/testing.md)
- [调试技巧](guides/development/debugging.md)

### 部署指南
- [Docker 部署](guides/deployment/docker.md)
- [Kubernetes 部署](guides/deployment/kubernetes.md)
- [监控配置](guides/deployment/monitoring.md)

### 用户手册
- [快速开始](guides/user/quickstart.md)
- [Console 使用指南](guides/user/console-guide.md)
- [CLI 工具参考](guides/user/cli-reference.md)

### API 文档
- [Kafka 协议](api/kafka-protocol.md)
- [REST API](api/rest-api.md)
- [gRPC API](api/grpc-api.md)
- [Swagger 文档](swagger/)

## 🎯 开发阶段文档

项目按以下8个阶段完成开发：

1. [Phase 1: 稳定性与基础补全](phases/phase-1-stability/) - ✅ 已完成
2. [Phase 2: Console 前端开发](phases/phase-2-console/) - ✅ 已完成
3. [Phase 3: 性能优化](phases/phase-3-performance/) - ✅ 已完成
4. [Phase 4: 安全特性](phases/phase-4-security/) - ✅ 已完成
5. [Phase 5: 监控运维](phases/phase-5-monitoring/) - ✅ 已完成
6. [Phase 6: HTTP Proxy](phases/phase-6-http-proxy/) - ✅ 已完成
7. [Phase 7: 文档与测试](phases/phase-7-documentation/) - ✅ 已完成
8. [Phase 8: 部署运维](phases/phase-8-deployment/) - ✅ 已完成

每个阶段目录包含该阶段所有任务的详细文档。

## 🔍 搜索文档

使用以下命令在文档中搜索关键词：

```bash
# 在所有文档中搜索
grep -r "关键词" docs/

# 在特定阶段中搜索
grep -r "关键词" docs/phases/phase-2-console/
```

## 📝 文档贡献

发现文档问题或想要改进？请参考 [贡献指南](../CONTRIBUTING.md)。

## 🗂️ 文档结构

```
docs/
├── README.md                    # 本文件
├── PROJECT_STATUS_REPORT.md     # 项目状态报告
├── phases/                      # 开发阶段文档
│   ├── phase-1-stability/
│   ├── phase-2-console/
│   ├── phase-3-performance/
│   ├── phase-4-security/
│   ├── phase-5-monitoring/
│   ├── phase-6-http-proxy/
│   ├── phase-7-documentation/
│   └── phase-8-deployment/
├── api/                         # API 文档
├── guides/                      # 使用指南
│   ├── development/            # 开发指南
│   ├── deployment/             # 部署指南
│   └── user/                   # 用户手册
├── architecture/               # 架构文档
└── archive/                    # 归档文档
```

---

**最后更新**: 2026年2月9日
