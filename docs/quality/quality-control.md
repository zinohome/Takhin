# 质量管控计划

## 1. 质量管理体系

### 1.1 质量目标

| 维度 | 目标 | 度量标准 |
|------|------|----------|
| **功能正确性** | 无 P0/P1 Bug | 生产环境 Bug 数量 |
| **代码质量** | 测试覆盖率 ≥ 80% | 单元测试覆盖率 |
| **性能指标** | P99 延迟 < 10ms | 性能监控数据 |
| **可用性** | 服务可用性 99.9% | SLA 监控 |
| **安全性** | 无已知高危漏洞 | 安全扫描结果 |
| **文档完整性** | API 文档覆盖 100% | 文档审查 |

### 1.2 质量保证流程

```
开发阶段          测试阶段          发布阶段          运维阶段
    │                │                │                │
    ├─ 代码规范      ├─ 单元测试      ├─ 集成测试      ├─ 监控告警
    ├─ 静态分析      ├─ 代码审查      ├─ E2E 测试      ├─ 性能监控
    ├─ 单元测试      ├─ 集成测试      ├─ 性能测试      ├─ 错误追踪
    └─ 提交检查      └─ 安全扫描      └─ 发布审批      └─ 日志分析
```

## 2. 代码质量管控

### 2.1 代码规范

#### Go 代码规范
- **格式化**: `gofmt` 或 `goimports`
- **Linting**: `golangci-lint` (集成多个 linter)
  ```yaml
  # .golangci.yaml
  linters:
    enable:
      - errcheck      # 检查错误处理
      - gosimple      # 简化代码
      - govet         # 静态分析
      - ineffassign   # 检查无效赋值
      - staticcheck   # 高级静态分析
      - unused        # 检查未使用代码
      - gocyclo       # 检查圈复杂度
      - gofmt         # 格式检查
      - misspell      # 拼写检查
  
  linters-settings:
    gocyclo:
      min-complexity: 15  # 圈复杂度阈值
    errcheck:
      check-blank: true   # 检查 _ 忽略的错误
  ```

#### TypeScript 代码规范
- **格式化和 Linting**: Biome (统一工具)
  ```jsonc
  // biome.jsonc
  {
    "linter": {
      "enabled": true,
      "rules": {
        "recommended": true,
        "complexity": {
          "noExtraBooleanCast": "error",
          "noMultipleSpacesInRegularExpressionLiterals": "error"
        },
        "correctness": {
          "noUnusedVariables": "error",
          "useExhaustiveDependencies": "warn"
        },
        "style": {
          "noNegationElse": "off",
          "useConst": "error"
        }
      }
    },
    "formatter": {
      "enabled": true,
      "indentStyle": "space",
      "indentWidth": 2,
      "lineWidth": 100
    }
  }
  ```

### 2.2 代码复杂度控制

#### 复杂度指标
- **圈复杂度**: 单个函数 < 15
- **函数长度**: < 100 行
- **文件长度**: < 500 行
- **嵌套深度**: < 4 层

#### 检测工具
```bash
# Go: gocyclo
gocyclo -over 15 .

# TypeScript: Biome
biome check src/
```

### 2.3 代码审查

#### 审查流程
1. 开发者提交 Pull Request
2. CI 自动检查 (格式、测试、构建)
3. 至少 1 名审查者 approve
4. 所有讨论解决
5. 合并到目标分支

#### 审查清单
- [ ] **功能实现**
  - [ ] 需求是否完整实现
  - [ ] 边界条件是否处理
  - [ ] 错误处理是否完善
  
- [ ] **代码质量**
  - [ ] 命名是否清晰
  - [ ] 逻辑是否简洁
  - [ ] 是否有重复代码
  - [ ] 是否遵循设计模式
  
- [ ] **性能考虑**
  - [ ] 是否有性能瓶颈
  - [ ] 内存使用是否合理
  - [ ] 是否有不必要的计算
  
- [ ] **安全性**
  - [ ] 是否有安全漏洞
  - [ ] 输入验证是否充分
  - [ ] 敏感信息是否保护
  
- [ ] **测试覆盖**
  - [ ] 单元测试是否充分
  - [ ] 测试用例是否合理
  - [ ] 边界条件是否测试
  
- [ ] **文档更新**
  - [ ] API 文档是否更新
  - [ ] 代码注释是否清晰
  - [ ] README 是否更新

### 2.4 静态代码分析

#### SonarQube 集成
```yaml
# sonar-project.properties
sonar.projectKey=takhin
sonar.projectName=Takhin
sonar.sources=.
sonar.exclusions=**/node_modules/**,**/vendor/**,**/*_test.go

# Go
sonar.go.coverage.reportPaths=coverage.out
sonar.go.tests.reportPaths=test-report.json

# TypeScript
sonar.typescript.lcov.reportPaths=coverage/lcov.info
sonar.javascript.lcov.reportPaths=coverage/lcov.info

# Quality Gates
sonar.qualitygate.wait=true
```

#### 质量门禁标准
| 指标 | 阈值 | 级别 |
|------|------|------|
| 代码覆盖率 | ≥ 80% | 错误 |
| 重复代码率 | ≤ 3% | 警告 |
| 代码异味 | ≤ 10 | 警告 |
| 漏洞数量 | 0 | 错误 |
| Bug 数量 | 0 | 错误 |
| 安全热点 | 已审查 | 警告 |

## 3. 测试质量管控

### 3.1 测试覆盖率

#### 覆盖率要求
```yaml
# 后端
Core 引擎: ≥ 85%
Console 后端: ≥ 80%
工具函数: ≥ 90%

# 前端
核心组件: ≥ 75%
业务页面: ≥ 70%
工具函数: ≥ 85%
```

#### 覆盖率监控
```bash
# Go
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# TypeScript
npm run test:coverage
```

#### CI 集成
```yaml
# .github/workflows/coverage.yml
- name: Check coverage
  run: |
    go test -coverprofile=coverage.out ./...
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 80%"
      exit 1
    fi
```

### 3.2 测试用例质量

#### 测试用例要求
- **独立性**: 测试之间互不影响
- **可重复性**: 多次运行结果一致
- **可读性**: 测试意图清晰
- **完整性**: 覆盖正常和异常情况
- **断言明确**: 每个测试有明确的断言

#### 测试命名规范
```go
// Go: TestFunctionName_Scenario_ExpectedBehavior
func TestTopicService_CreateTopic_Success(t *testing.T) {}
func TestTopicService_CreateTopic_InvalidName(t *testing.T) {}
```

```typescript
// TypeScript: describe + it
describe('TopicService', () => {
  describe('createTopic', () => {
    it('should create topic successfully', () => {});
    it('should throw error for invalid name', () => {});
  });
});
```

### 3.3 测试数据管理

#### 测试数据原则
- **隔离性**: 每个测试使用独立数据
- **清理性**: 测试后清理数据
- **真实性**: 数据接近生产环境
- **可维护性**: 易于创建和管理

#### 测试数据工厂
```go
// testutil/factory/topic.go
package factory

func CreateTopic(opts ...TopicOption) *model.Topic {
    topic := &model.Topic{
        Name:              "test-topic",
        Partitions:        3,
        ReplicationFactor: 2,
        Config:            make(map[string]string),
    }
    
    for _, opt := range opts {
        opt(topic)
    }
    
    return topic
}

type TopicOption func(*model.Topic)

func WithName(name string) TopicOption {
    return func(t *model.Topic) {
        t.Name = name
    }
}

func WithPartitions(partitions int) TopicOption {
    return func(t *model.Topic) {
        t.Partitions = partitions
    }
}
```

## 4. 性能质量管控

### 4.1 性能基准

#### 关键指标
| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| 吞吐量 | >100K msg/s | k6 压测 |
| P50 延迟 | <5ms | k6 压测 |
| P99 延迟 | <10ms | k6 压测 |
| P999 延迟 | <50ms | k6 压测 |
| CPU 使用率 | <80% | 压测监控 |
| 内存使用 | <8GB | 压测监控 |
| 首屏加载时间 | <2s | Lighthouse |
| TTI (可交互时间) | <3s | Lighthouse |

### 4.2 性能监控

#### 监控指标
```yaml
# Prometheus 指标
# 吞吐量
rate(kafka_requests_total[5m])

# 延迟
histogram_quantile(0.99, kafka_request_duration_seconds_bucket)

# 错误率
rate(kafka_request_errors_total[5m]) / rate(kafka_requests_total[5m])

# 资源使用
process_cpu_seconds_total
process_resident_memory_bytes
```

#### 告警规则
```yaml
# alerts.yml
groups:
  - name: performance
    rules:
      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High request latency"
      
      - alert: LowThroughput
        expr: rate(kafka_messages_total[5m]) < 10000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Low message throughput"
      
      - alert: HighErrorRate
        expr: rate(kafka_request_errors_total[5m]) / rate(kafka_requests_total[5m]) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate"
```

### 4.3 性能测试

#### 压力测试场景
```javascript
// tests/performance/stress-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    // 恒定负载测试
    constant_load: {
      executor: 'constant-vus',
      vus: 100,
      duration: '10m',
    },
    
    // 阶梯负载测试
    ramping_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5m', target: 100 },
        { duration: '10m', target: 100 },
        { duration: '5m', target: 200 },
        { duration: '10m', target: 200 },
        { duration: '5m', target: 0 },
      ],
    },
    
    // 峰值负载测试
    spike_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 500 },
        { duration: '30s', target: 500 },
        { duration: '1m', target: 0 },
      ],
    },
  },
  
  thresholds: {
    http_req_duration: ['p(99)<500'],
    http_req_failed: ['rate<0.01'],
  },
};
```

## 5. 安全质量管控

### 5.1 安全扫描

#### 依赖扫描
```yaml
# .github/workflows/security.yml
- name: Run Snyk to check for vulnerabilities
  uses: snyk/actions/golang@master
  env:
    SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
  with:
    args: --severity-threshold=high

- name: Run npm audit
  run: npm audit --audit-level=high
```

#### 代码扫描
```yaml
- name: Initialize CodeQL
  uses: github/codeql-action/init@v2
  with:
    languages: go, javascript

- name: Perform CodeQL Analysis
  uses: github/codeql-action/analyze@v2
```

#### 容器扫描
```bash
# Trivy 扫描
trivy image takhin-core:latest --severity HIGH,CRITICAL
trivy image takhin-console:latest --severity HIGH,CRITICAL
```

### 5.2 安全编码规范

#### 输入验证
```go
// 所有用户输入必须验证
func CreateTopic(req *CreateTopicRequest) error {
    // 验证 Topic 名称
    if !isValidTopicName(req.Name) {
        return ErrInvalidTopicName
    }
    
    // 验证分区数
    if req.Partitions < 1 || req.Partitions > MaxPartitions {
        return ErrInvalidPartitionCount
    }
    
    // 验证副本因子
    if req.ReplicationFactor < 1 || req.ReplicationFactor > MaxReplicationFactor {
        return ErrInvalidReplicationFactor
    }
    
    return nil
}
```

#### 敏感信息处理
```go
// 配置中的敏感信息加密
type Config struct {
    KafkaBrokers []string
    Username     string
    Password     SecretString  // 特殊类型，不会被日志输出
}

// SecretString 不会被 JSON 序列化
type SecretString string

func (s SecretString) MarshalJSON() ([]byte, error) {
    return []byte(`"[REDACTED]"`), nil
}
```

### 5.3 权限控制

#### RBAC 实现
```go
// 基于角色的访问控制
type Permission string

const (
    PermTopicRead   Permission = "topic:read"
    PermTopicWrite  Permission = "topic:write"
    PermTopicDelete Permission = "topic:delete"
)

func (api *API) RequirePermission(perm Permission) middleware.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := GetUserFromContext(r.Context())
            if !user.HasPermission(perm) {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## 6. 文档质量管控

### 6.1 文档要求

#### 代码注释
```go
// CreateTopic 创建新的 Topic
//
// 参数:
//   - name: Topic 名称，必须符合 Kafka 命名规范
//   - partitions: 分区数量，必须 > 0
//   - replicationFactor: 副本因子，必须 ≤ 集群节点数
//
// 返回:
//   - error: 创建失败时返回错误
//
// 示例:
//   err := CreateTopic("my-topic", 3, 2)
//   if err != nil {
//       log.Fatal(err)
//   }
func CreateTopic(name string, partitions, replicationFactor int) error {
    // ...
}
```

#### API 文档
- 使用 OpenAPI/Swagger 规范
- 自动从代码生成
- 包含请求/响应示例
- 包含错误码说明

#### README 文档
- 项目介绍
- 快速开始
- 安装说明
- 配置说明
- API 文档链接
- 贡献指南

### 6.2 文档审查

#### 文档检查清单
- [ ] 文档格式正确 (Markdown)
- [ ] 代码示例可运行
- [ ] 链接有效
- [ ] 图片清晰
- [ ] 无拼写错误
- [ ] 结构清晰
- [ ] 内容完整

## 7. 持续改进

### 7.1 质量度量

#### 关键指标
```yaml
# 每周质量报告
质量指标:
  - 代码覆盖率: 82.5% (↑2.3%)
  - Bug 修复率: 95% (→)
  - 代码审查通过率: 88% (↑5%)
  - 平均修复时间: 2.3 天 (↓0.5天)
  
技术债务:
  - 代码异味: 15 个 (↓3)
  - 技术债务时间: 8 小时 (↓2小时)
  
性能指标:
  - P99 延迟: 8.5ms (↓1.5ms)
  - 吞吐量: 125K msg/s (↑15K)
```

### 7.2 改进措施

#### 定期回顾
- **每日站会**: 同步进度和问题
- **每周复盘**: 回顾本周质量问题
- **每月总结**: 分析质量趋势
- **季度规划**: 制定改进目标

#### 问题跟踪
```yaml
# 质量问题模板
标题: [组件] 简短描述
类型: Bug / 技术债务 / 性能问题
优先级: P0 / P1 / P2 / P3
影响范围: 核心功能 / 次要功能 / 边缘情况
根因分析: 问题原因
解决方案: 修复方案
预防措施: 如何避免再次发生
```

#### 知识分享
- 技术分享会 (每两周)
- 代码审查最佳实践分享
- 测试案例分享
- 性能优化经验分享

---

**文档版本**: v1.0  
**最后更新**: 2025-12-14  
**维护者**: Takhin Team
