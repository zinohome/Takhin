#!/bin/bash

# Takhin 文档重组脚本
# 用途：将根目录的 TASK_*.md 文件按阶段重新组织到 docs/phases/ 目录

set -e

PROJECT_ROOT="/Users/zhangjun/CursorProjects/Takhin"
cd "$PROJECT_ROOT"

echo "🚀 开始 Takhin 文档重组..."
echo "================================================"

# 创建新的目录结构
echo "📁 创建目录结构..."
mkdir -p docs/phases/phase-1-stability
mkdir -p docs/phases/phase-2-console
mkdir -p docs/phases/phase-3-performance
mkdir -p docs/phases/phase-4-security
mkdir -p docs/phases/phase-5-monitoring
mkdir -p docs/phases/phase-6-http-proxy
mkdir -p docs/phases/phase-7-documentation
mkdir -p docs/phases/phase-8-deployment
mkdir -p docs/project
mkdir -p docs/guides/development
mkdir -p docs/guides/deployment
mkdir -p docs/guides/user
mkdir -p docs/archive/legacy-tasks

echo "✅ 目录结构创建完成"

# 定义任务到阶段的映射函数
get_phase() {
    local task_num=$1
    case $task_num in
        1.2|1.3|1.5|1.6) echo "phase-1-stability" ;;
        2.2|2.3|2.5|2.6|2.7|2.8|2.9|2.10|2.11) echo "phase-2-console" ;;
        3.1|3.2|3.3|3.4) echo "phase-3-performance" ;;
        4.1|4.2|4.3|4.4|4.5) echo "phase-4-security" ;;
        5.1|5.2|5.3|5.4|5.5) echo "phase-5-monitoring" ;;
        6.1|6.2|6.3|6.4|6.5|6.6) echo "phase-6-http-proxy" ;;
        7.3|7.4|7.5|7.6) echo "phase-7-documentation" ;;
        8.1|8.3|8.4|8.5) echo "phase-8-deployment" ;;
        *) echo "" ;;
    esac
}

# 移动文件的计数器
MOVED_COUNT=0
TOTAL_FILES=$(ls -1 TASK_*.md 2>/dev/null | wc -l | tr -d ' ')

echo "📝 找到 $TOTAL_FILES 个 TASK 文档"
echo ""
echo "🔄 开始移动文件..."

# 遍历所有 TASK 文件
for file in TASK_*.md; do
    if [ ! -f "$file" ]; then
        continue
    fi
    
    # 提取任务编号 (例如: TASK_2.5_CONSUMER_GROUPS_COMPLETION.md -> 2.5)
    TASK_NUM=$(echo "$file" | sed -E 's/TASK_([0-9]+\.[0-9]+)_.*/\1/')
    
    if [ "$TASK_NUM" = "$file" ]; then
        # 无法识别的文件，移动到 archive
        echo "  ⚠️  无法解析: $file -> 移动到 archive"
        mv "$file" "docs/archive/legacy-tasks/" 2>/dev/null || true
        MOVED_COUNT=$((MOVED_COUNT + 1))
        continue
    fi
    
    # 获取对应的阶段
    PHASE=$(get_phase "$TASK_NUM")
    
    if [ -z "$PHASE" ]; then
        echo "⚠️  未知任务编号: $TASK_NUM (文件: $file)"
        # 移动到 archive
        mv "$file" "docs/archive/legacy-tasks/" 2>/dev/null || true
        MOVED_COUNT=$((MOVED_COUNT + 1))
        continue
    fi
    
    # 提取文档类型和任务名称
    # 例如: TASK_2.5_CONSUMER_GROUPS_COMPLETION.md
    BASENAME=$(basename "$file" .md)
    # 移除 TASK_X.X_ 前缀 (处理中文中的特殊字符)
    SUFFIX=$(echo "$BASENAME" | sed -E "s/TASK_${TASK_NUM//\./\\.}_//")
    
    # 创建任务子目录
    TASK_DIR="docs/phases/$PHASE/task-$TASK_NUM"
    mkdir -p "$TASK_DIR"
    
    # 确定新文件名
    # 将 CONSUMER_GROUPS_COMPLETION -> consumer-groups-completion.md
    NEW_NAME=$(echo "$SUFFIX" | tr '[:upper:]' '[:lower:]' | tr '_' '-').md
    
    # 移动文件
    if mv "$file" "$TASK_DIR/$NEW_NAME" 2>/dev/null; then
        echo "  ✓ $file -> task-$TASK_NUM/$NEW_NAME"
    else
        echo "  ⚠️  移动失败: $file"
    fi
    MOVED_COUNT=$((MOVED_COUNT + 1))
done

echo ""
echo "✅ 文件移动完成: $MOVED_COUNT/$TOTAL_FILES"

# 创建各阶段的 README.md
echo ""
echo "📄 创建阶段索引文件..."

# Phase 1
cat > docs/phases/phase-1-stability/README.md << 'EOF'
# Phase 1: 稳定性与基础补全

## 概述
本阶段专注于 Takhin 核心稳定性和基础功能的完善，包括存储层优化和复制系统增强。

## 任务列表

- [1.2 - 存储错误恢复](task-1.2/)
- [1.3 - Snapshot 支持](task-1.3/)
- [1.5 - Leader 选举优化](task-1.5/)
- [1.6 - 复制延迟监控](task-1.6/)

## 状态
✅ 已完成
EOF

# Phase 2
cat > docs/phases/phase-2-console/README.md << 'EOF'
# Phase 2: Console 前端开发

## 概述
本阶段开发 Takhin Console Web 管理界面，提供完整的集群管理和监控能力。

## 任务列表

- [2.2 - API 客户端](task-2.2/)
- [2.3 - 主题管理](task-2.3/)
- [2.5 - 消费者组管理](task-2.5/)
- [2.6 - 消息浏览器](task-2.6/)
- [2.7 - 配置管理](task-2.7/)
- [2.8 - 监控仪表盘](task-2.8/)
- [2.9 - WebSocket 实时更新](task-2.9/)
- [2.10 - gRPC API](task-2.10/)
- [2.11 - 批量操作 API](task-2.11/)

## 状态
✅ 已完成
EOF

# Phase 3
cat > docs/phases/phase-3-performance/README.md << 'EOF'
# Phase 3: 性能优化

## 概述
本阶段实施各项性能优化措施，提升 Takhin 的吞吐量和降低延迟。

## 任务列表

- [3.1 - 零拷贝实现](task-3.1/)
- [3.2 - 内存池](task-3.2/)
- [3.3 - 批处理优化](task-3.3/)
- [3.4 - 网络流量控制](task-3.4/)

## 性能目标
- P99 延迟 < 10ms
- 吞吐量 > 100K msg/s

## 状态
✅ 已完成
EOF

# Phase 4
cat > docs/phases/phase-4-security/README.md << 'EOF'
# Phase 4: 安全特性

## 概述
本阶段实现完整的安全特性，包括认证、授权、加密和审计。

## 任务列表

- [4.1 - ACL 权限控制](task-4.1/)
- [4.2 - TLS 加密](task-4.2/)
- [4.3 - SASL 认证](task-4.3/)
- [4.4 - 数据加密](task-4.4/)
- [4.5 - 审计日志](task-4.5/)

## 安全特性
- ✅ TLS 1.2/1.3 支持
- ✅ SASL/PLAIN, SASL/SCRAM-SHA-256/512
- ✅ ACL 细粒度权限控制
- ✅ 静态数据加密 (AES-256-GCM)
- ✅ 完整审计日志

## 状态
✅ 已完成
EOF

# Phase 5
cat > docs/phases/phase-5-monitoring/README.md << 'EOF'
# Phase 5: 监控运维

## 概述
本阶段建立完整的可观测性体系，包括指标收集、健康检查和告警。

## 任务列表

- [5.1 - 指标收集](task-5.1/)
- [5.2 - 健康检查](task-5.2/)
- [5.3 - Debug Bundle](task-5.3/)
- [5.4 - Grafana 集成](task-5.4/)
- [5.5 - 告警系统](task-5.5/)

## 技术栈
- Prometheus - 指标收集
- Grafana - 可视化
- Alertmanager - 告警

## 状态
✅ 已完成
EOF

# Phase 6
cat > docs/phases/phase-6-http-proxy/README.md << 'EOF'
# Phase 6: HTTP Proxy

## 概述
本阶段实现 HTTP REST API，使非 Kafka 协议的客户端也能访问 Takhin。

## 任务列表

- [6.1 - REST API 设计](task-6.1/)
- [6.2 - Producer API](task-6.2/)
- [6.3 - HTTP Producer](task-6.3/)
- [6.4 - HTTP Consumer](task-6.4/)
- [6.5 - S3 集成](task-6.5/)
- [6.6 - 冷热数据分离](task-6.6/)

## 特性
- RESTful API
- S3 对象存储集成
- 冷热数据自动分离

## 状态
✅ 已完成
EOF

# Phase 7
cat > docs/phases/phase-7-documentation/README.md << 'EOF'
# Phase 7: 文档与测试

## 概述
本阶段完善项目文档和测试覆盖，确保项目的可维护性。

## 任务列表

- [7.3 - 开发者指南](task-7.3/)
- [7.4 - 用户手册](task-7.4/)
- [7.5 - 测试覆盖率](task-7.5/)
- [7.6 - E2E 测试套件](task-7.6/)

## 文档类型
- 开发者指南
- 用户手册
- API 文档
- 部署指南

## 状态
✅ 已完成
EOF

# Phase 8
cat > docs/phases/phase-8-deployment/README.md << 'EOF'
# Phase 8: 部署运维

## 概述
本阶段完善构建、部署和运维工具，支持生产环境部署。

## 任务列表

- [8.1 - 多平台构建](task-8.1/)
- [8.3 - 性能回归测试](task-8.3/)
- [8.4 - CLI 管理工具](task-8.4/)
- [8.5 - 性能分析工具](task-8.5/)

## 支持平台
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## 部署方式
- Docker / Docker Compose
- Kubernetes
- 二进制直接部署

## 状态
✅ 已完成
EOF

echo "✅ 阶段索引文件创建完成"

# 创建总索引
cat > docs/README.md << 'EOF'
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
EOF

echo "✅ 总索引文件创建完成"

echo ""
echo "================================================"
echo "🎉 文档重组完成！"
echo ""
echo "📊 统计:"
echo "  - 移动文件: $MOVED_COUNT"
echo "  - 创建目录: 8个阶段 + 多个子目录"
echo "  - 创建索引: 9个 README.md"
echo ""
echo "📁 新的文档结构:"
echo "  docs/"
echo "  ├── README.md (总导航)"
echo "  ├── phases/ (8个阶段)"
echo "  ├── guides/ (开发/部署/用户指南)"
echo "  └── archive/ (旧文档归档)"
echo ""
echo "🔍 查看文档导航:"
echo "  cat docs/README.md"
echo ""
echo "✨ 建议下一步:"
echo "  1. 查看 docs/README.md 熟悉新结构"
echo "  2. 检查 docs/phases/ 中的任务分类"
echo "  3. 如有需要，更新文档内的链接引用"
echo "  4. 提交更改: git add docs/ && git commit -m 'docs: 重组项目文档结构'"
echo ""
