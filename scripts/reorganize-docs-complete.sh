#!/bin/bash

# Takhin 文档完全重组脚本
# 用途：整理 docs/ 目录下所有松散的文档

set -e

PROJECT_ROOT="/Users/zhangjun/CursorProjects/Takhin"
cd "$PROJECT_ROOT/docs"

echo "🚀 开始 docs 目录完全重组..."
echo "================================================"

# 统计当前状态
ROOT_DOCS=$(ls -1 *.md 2>/dev/null | wc -l | tr -d ' ')
echo "📊 当前docs根目录有 $ROOT_DOCS 个文档需要整理"
echo ""

# 创建必要的目录结构
echo "📁 创建目录结构..."
mkdir -p guides/user
mkdir -p guides/development
mkdir -p guides/operations
mkdir -p api/rest
mkdir -p api/grpc
mkdir -p api/kafka
mkdir -p architecture/design
mkdir -p reference
mkdir -p archive/old-summaries
mkdir -p archive/old-reports

echo "✅ 目录结构创建完成"
echo ""

# 移动计数器
MOVED_COUNT=0

echo "🔄 开始整理文档..."
echo ""

# ===== 1. Phase任务相关文档 =====
echo "📂 Phase 1 - 稳定性任务文档"
if [ -f "TASK_1.5_LEADER_ELECTION.md" ]; then
    mv "TASK_1.5_LEADER_ELECTION.md" "phases/phase-1-stability/task-1.5/"
    echo "  ✓ TASK_1.5 → phase-1-stability/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

echo ""
echo "📂 Phase 2 - Console任务文档"
if [ -f "TASK_2.2_ARCHITECTURE.md" ]; then
    mv "TASK_2.2_ARCHITECTURE.md" "phases/phase-2-console/task-2.2/"
    echo "  ✓ TASK_2.2_ARCHITECTURE → phase-2-console/task-2.2/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

if [ -f "TASK_2.2_QUICK_REFERENCE.md" ]; then
    mv "TASK_2.2_QUICK_REFERENCE.md" "phases/phase-2-console/task-2.2/"
    echo "  ✓ TASK_2.2_QUICK_REFERENCE → phase-2-console/task-2.2/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 2. API 文档 =====
echo ""
echo "📂 API 文档整理"

if [ -f "admin-api.md" ]; then
    mv "admin-api.md" "api/rest/admin-api.md"
    echo "  ✓ admin-api → api/rest/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

if [ -f "websocket-api.md" ]; then
    mv "websocket-api.md" "api/rest/websocket-api.md"
    echo "  ✓ websocket-api → api/rest/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 3. 用户指南 =====
echo ""
echo "📂 用户指南整理"

if [ -f "USER_MANUAL.md" ]; then
    mv "USER_MANUAL.md" "guides/user/user-manual.md"
    echo "  ✓ USER_MANUAL → guides/user/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

if [ -f "console-usage-guide.md" ]; then
    mv "console-usage-guide.md" "guides/user/console-guide.md"
    echo "  ✓ console-usage-guide → guides/user/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 4. Console 实现文档 =====
echo ""
echo "📂 Console 实现文档"

if [ -f "console-api-auth-report.md" ]; then
    mv "console-api-auth-report.md" "phases/phase-2-console/auth-implementation-report.md"
    echo "  ✓ console-api-auth-report → phase-2-console/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

if [ -f "console-api-implementation.md" ]; then
    mv "console-api-implementation.md" "phases/phase-2-console/api-implementation-guide.md"
    echo "  ✓ console-api-implementation → phase-2-console/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

if [ -f "consumer-group-summary.md" ]; then
    mv "consumer-group-summary.md" "phases/phase-2-console/task-2.5/consumer-group-summary.md"
    echo "  ✓ consumer-group-summary → phase-2-console/task-2.5/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 5. 部署文档 =====
echo ""
echo "📂 部署文档整理"

if [ -f "deployment-documentation-summary.md" ]; then
    mv "deployment-documentation-summary.md" "deployment/documentation-summary.md"
    echo "  ✓ deployment-documentation-summary → deployment/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 6. 监控和指标 =====
echo ""
echo "📂 监控文档整理"

if [ -f "metrics.md" ]; then
    mv "metrics.md" "monitoring/metrics-overview.md"
    echo "  ✓ metrics → monitoring/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

if [ -f "METRICS_QUICK_REF.md" ]; then
    mv "METRICS_QUICK_REF.md" "reference/metrics-quick-reference.md"
    echo "  ✓ METRICS_QUICK_REF → reference/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 7. 测试文档 =====
echo ""
echo "📂 测试文档整理"

if [ -f "raft-cluster-test-summary.md" ]; then
    mv "raft-cluster-test-summary.md" "testing/raft-cluster-test-summary.md"
    echo "  ✓ raft-cluster-test-summary → testing/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

if [ -f "raft-integration-summary.md" ]; then
    mv "raft-integration-summary.md" "testing/raft-integration-summary.md"
    echo "  ✓ raft-integration-summary → testing/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 8. 架构设计文档 =====
echo ""
echo "📂 架构设计文档"

if [ -f "transactions-design.md" ]; then
    mv "transactions-design.md" "architecture/design/transactions-design.md"
    echo "  ✓ transactions-design → architecture/design/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 9. Sprint 和实现总结 =====
echo ""
echo "📂 实现总结归档"

if [ -f "sprint-9-10-summary.md" ]; then
    mv "sprint-9-10-summary.md" "implementation/sprint-9-10-summary.md"
    echo "  ✓ sprint-9-10-summary → implementation/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 10. 评估和分析报告（归档） =====
echo ""
echo "📂 分析报告归档"

if [ -f "takhin-assessment-2026.md" ]; then
    mv "takhin-assessment-2026.md" "archive/old-reports/takhin-assessment-2026.md"
    echo "  ✓ takhin-assessment-2026 → archive/old-reports/"
    MOVED_COUNT=$((MOVED_COUNT + 1))
fi

# ===== 11. TODO-Kanban（保留在根目录或归档） =====
echo ""
echo "📂 项目管理文档"

if [ -f "TODO-Kanban.md" ]; then
    # TODO-Kanban保留在docs根目录，因为是活跃文档
    echo "  → TODO-Kanban.md 保留在 docs/ 根目录"
fi

# ===== 12. 更新已有目录的README =====
echo ""
echo "📄 更新目录索引文件..."

# 更新 guides/user/README.md
cat > guides/user/README.md << 'EOF'
# Takhin 用户指南

本目录包含 Takhin 的用户使用手册和指南。

## 📚 文档列表

- **[用户手册](user-manual.md)** - 完整的用户使用手册
- **[Console 使用指南](console-guide.md)** - Web 控制台使用指南

## 快速开始

如果你是新用户，建议按以下顺序阅读：

1. 先阅读[用户手册](user-manual.md)了解基本概念
2. 然后查看[Console 使用指南](console-guide.md)学习如何使用Web界面

## 其他资源

- [API 文档](../../api/) - REST API 和 Kafka 协议文档
- [部署指南](../../deployment/) - 部署和配置参考
- [开发者指南](../development/) - 如何参与开发
EOF

# 更新 guides/development/README.md
cat > guides/development/README.md << 'EOF'
# Takhin 开发者指南

本目录包含 Takhin 的开发文档和指南。

## 📚 文档列表

开发者文档正在整理中，请参考：

- [架构文档](../../architecture/) - 系统架构设计
- [API 文档](../../api/) - API 接口文档
- [测试指南](../../testing/) - 测试策略和工具

## 贡献指南

请查看项目根目录的 [CONTRIBUTING.md](../../../CONTRIBUTING.md)
EOF

# 更新 guides/operations/README.md
cat > guides/operations/README.md << 'EOF'
# Takhin 运维指南

本目录包含 Takhin 的运维相关文档。

## 📚 文档列表

运维文档正在整理中，请参考：

- [部署文档](../../deployment/) - 部署和配置
- [监控文档](../../monitoring/) - 监控和告警
- [性能文档](../../performance/) - 性能优化

## 快速链接

- [健康检查](../../phases/phase-5-monitoring/task-5.2/)
- [Debug Bundle](../../phases/phase-5-monitoring/task-5.3/)
- [Grafana 集成](../../phases/phase-5-monitoring/task-5.4/)
EOF

# 更新 api/rest/README.md
cat > api/rest/README.md << 'EOF'
# Takhin REST API 文档

本目录包含 Takhin 的 REST API 文档。

## 📚 API 文档

- **[Console REST API](console-rest-api.md)** - Console API 完整文档
- **[Admin API](admin-api.md)** - 管理 API 文档
- **[WebSocket API](websocket-api.md)** - WebSocket 实时推送 API

## 使用示例

查看 [examples/](../examples/) 目录获取更多示例代码。

## 认证

Console API 支持 API Key 认证，详见各 API 文档的认证章节。
EOF

# 更新 api/kafka/README.md
cat > api/kafka/README.md << 'EOF'
# Takhin Kafka Protocol API

本目录包含 Kafka 协议相关文档。

## 📚 文档列表

- **[Kafka Protocol API](kafka-protocol-api.md)** - Kafka 协议实现文档

## 兼容性

Takhin 兼容 Apache Kafka 0.11.x+ 协议。
EOF

# 更新 api/grpc/README.md
cat > api/grpc/README.md << 'EOF'
# Takhin gRPC API

本目录包含 gRPC API 相关文档。

## 📚 文档列表

gRPC API 详细文档请参考：
- [Phase 2 - Task 2.10 gRPC 实现](../../phases/phase-2-console/task-2.10/)

## Protocol Buffers

Proto 文件位置: `backend/api/proto/takhin.proto`
EOF

# 更新 architecture/design/README.md
cat > architecture/design/README.md << 'EOF'
# Takhin 设计文档

本目录包含 Takhin 的详细设计文档。

## 📚 设计文档列表

- **[事务设计](transactions-design.md)** - 事务处理设计文档

## 架构文档

更多架构文档请参考上级目录的架构概览文档。
EOF

# 更新 reference/README.md
cat > reference/README.md << 'EOF'
# Takhin 快速参考

本目录包含各种快速参考文档。

## 📚 参考文档

- **[Metrics 快速参考](metrics-quick-reference.md)** - 监控指标速查表

## 其他参考

- [API 文档](../api/)
- [配置参考](../deployment/04-configuration-reference.md)
EOF

echo "✅ 索引文件更新完成"

# ===== 13. 移动API子目录中的文档 =====
echo ""
echo "📂 整理 api/ 子目录"

# 移动kafka协议文档
if [ -f "api/kafka-protocol-api.md" ]; then
    mv "api/kafka-protocol-api.md" "api/kafka/"
    echo "  ✓ kafka-protocol-api → api/kafka/"
fi

# 移动console rest api
if [ -f "api/console-rest-api.md" ]; then
    mv "api/console-rest-api.md" "api/rest/"
    echo "  ✓ console-rest-api → api/rest/"
fi

# 清理空的旧README
if [ -f "api/README.md" ]; then
    # 检查是否只是简单的README，如果是则替换
    cat > api/README.md << 'EOF'
# Takhin API 文档

Takhin 提供多种 API 接口方式。

## 📚 API 类型

### [REST API](rest/)
Web Console 的 REST API，提供完整的集群管理功能。

- [Console REST API](rest/console-rest-api.md)
- [Admin API](rest/admin-api.md)
- [WebSocket API](rest/websocket-api.md)

### [Kafka Protocol](kafka/)
兼容 Apache Kafka 0.11.x+ 协议。

- [Kafka Protocol API](kafka/kafka-protocol-api.md)

### [gRPC API](grpc/)
高性能 gRPC 接口。

- [gRPC 实现文档](../phases/phase-2-console/task-2.10/)

## 使用示例

查看 [examples/](examples/) 目录获取各种 API 的使用示例。

## 认证

不同 API 使用不同的认证方式：
- **REST API**: API Key 认证
- **Kafka Protocol**: SASL 认证
- **gRPC**: TLS + Token 认证

详见各 API 文档的认证章节。
EOF
    echo "  ✓ 更新 api/README.md"
fi

# ===== 完成 =====
echo ""
echo "================================================"
echo "🎉 docs 目录重组完成！"
echo ""
echo "📊 统计:"
echo "  - 移动文件: $MOVED_COUNT"
echo "  - 创建索引: 多个 README.md"
echo ""
echo "📁 新的 docs 结构:"
echo "  docs/"
echo "  ├── README.md (主导航)"
echo "  ├── 分析报告 (4份)"
echo "  ├── phases/ (8个阶段)"
echo "  ├── api/"
echo "  │   ├── rest/ (REST API)"
echo "  │   ├── grpc/ (gRPC API)"
echo "  │   └── kafka/ (Kafka 协议)"
echo "  ├── guides/"
echo "  │   ├── user/ (用户指南)"
echo "  │   ├── development/ (开发指南)"
echo "  │   └── operations/ (运维指南)"
echo "  ├── architecture/"
echo "  │   └── design/ (设计文档)"
echo "  ├── deployment/ (部署文档)"
echo "  ├── monitoring/ (监控文档)"
echo "  ├── testing/ (测试文档)"
echo "  ├── implementation/ (实现总结)"
echo "  ├── reference/ (快速参考)"
echo "  └── archive/ (归档)"
echo ""
echo "✨ 建议下一步:"
echo "  1. 查看 docs/README.md 确认新结构"
echo "  2. 检查各子目录的README索引"
echo "  3. 提交更改: git add docs/ && git commit -m 'docs: 完全重组docs目录结构'"
echo ""
