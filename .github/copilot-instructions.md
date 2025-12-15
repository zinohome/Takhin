# Copilot Instructions for Takhin Project

## 项目概述

这是一个包含 Console 和 Redpanda 项目的 monorepo，使用 Go 语言（后端）和 React（前端）技术栈。

## Go 语言最佳实践

### 代码风格
- 严格遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化所有代码
- 使用 `golangci-lint` 进行代码检查
- 包名使用小写，不使用下划线或驼峰命名
- 接口名称应该以 `-er` 结尾（如 `Reader`, `Writer`）

### 项目结构
```
backend/
├── cmd/           # 主应用程序入口
├── pkg/           # 可复用的库代码
│   ├── api/       # API 定义
│   ├── config/    # 配置管理
│   ├── models/    # 数据模型
│   ├── service/   # 业务逻辑
│   └── utils/     # 工具函数
└── internal/      # 私有应用代码
```

### 错误处理
- 始终检查并处理错误，不要使用 `_` 忽略错误
- 使用 `fmt.Errorf` 包装错误以提供上下文：`fmt.Errorf("failed to process: %w", err)`
- 对于预期的错误，定义明确的错误变量或类型
- 在函数签名中明确返回错误类型

### 并发编程
- 使用 `context.Context` 进行取消和超时控制
- 始终使用 `defer` 清理资源（如关闭文件、数据库连接）
- 避免共享内存，优先使用 channel 进行通信
- 使用 `sync.WaitGroup` 或 `errgroup` 管理 goroutine

### 测试
- 每个包都应该有对应的测试文件 `*_test.go`
- 测试覆盖率应达到至少 80%
- 使用表驱动测试（table-driven tests）
- 对于集成测试，使用 build tags：`// +build integration`
- 使用 `testify` 库进行断言

### 性能优化
- 使用 `strings.Builder` 而非字符串拼接
- 预分配 slice 容量：`make([]Type, 0, expectedSize)`
- 使用指针接收器处理大型结构体
- 避免不必要的内存分配

## React 最佳实践

### 代码风格
- 使用 TypeScript，启用严格模式
- 遵循 Biome 配置的代码规范（参考 `biome.jsonc`）
- 组件文件使用 PascalCase 命名
- 工具函数文件使用 camelCase 命名

### 项目结构
```
frontend/src/
├── components/    # 可复用组件
│   ├── common/    # 通用组件
│   └── features/  # 功能特定组件
├── hooks/         # 自定义 Hooks
├── pages/         # 页面组件
├── services/      # API 服务
├── store/         # 状态管理
├── types/         # TypeScript 类型定义
├── utils/         # 工具函数
└── styles/        # 全局样式
```

### 组件开发
- 优先使用函数组件和 Hooks
- 组件应该单一职责，保持简洁
- 使用 `React.memo` 优化不必要的重渲染
- Props 类型必须明确定义（使用 TypeScript interface）
- 使用 `children` prop 实现组合模式

### Hooks 使用
- 使用 `useMemo` 和 `useCallback` 优化性能
- 自定义 Hook 必须以 `use` 开头
- Hook 依赖数组必须包含所有使用的外部变量
- 避免在 Hook 中使用过多的状态

### 状态管理
- 局部状态优先使用 `useState`
- 跨组件状态使用 Context API 或状态管理库
- 避免过度使用全局状态
- 异步状态使用 React Query 或 SWR

### 性能优化
- 使用代码分割（React.lazy 和 Suspense）
- 图片使用懒加载
- 避免内联函数和对象（在渲染函数中）
- 使用虚拟滚动处理长列表

### 测试
- 使用 Vitest 进行单元测试
- 使用 Playwright 进行端到端测试
- 测试覆盖率应达到至少 80%
- 测试文件命名：`*.test.ts` 或 `*.spec.ts`

## 文档管理规范

### 文档分类
所有文档必须放在 `docs/` 目录下，按以下结构组织：

```
docs/
├── README.md              # 文档目录索引
├── architecture/          # 架构设计文档
│   ├── overview.md
│   ├── backend.md
│   └── frontend.md
├── api/                   # API 文档
│   ├── rest-api.md
│   └── grpc-api.md
├── development/           # 开发指南
│   ├── setup.md          # 环境搭建
│   ├── workflow.md       # 开发流程
│   └── testing.md        # 测试指南
├── deployment/            # 部署文档
│   ├── local.md
│   ├── staging.md
│   └── production.md
├── features/              # 功能文档
├── rfcs/                  # RFC（请求评论）
└── troubleshooting/       # 故障排查
```

### 文档编写规范
- 使用 Markdown 格式
- 每个文档必须包含标题和目录
- 代码示例必须包含语言标识
- 保持文档与代码同步更新
- 使用相对链接引用其他文档

## 开发质量管控

### 代码审查（Code Review）
- 所有代码必须经过至少一名团队成员审查
- PR 必须通过所有 CI 检查才能合并
- 审查要点：
  - 代码逻辑正确性
  - 性能影响
  - 安全性考虑
  - 测试覆盖
  - 文档更新

### 持续集成（CI/CD）
- 每次提交必须通过以下检查：
  - 代码格式检查（gofmt, Biome）
  - 代码静态分析（golangci-lint, ESLint）
  - 单元测试（覆盖率 ≥ 80%）
  - 集成测试
  - 构建检查

### Git 工作流
- 使用 feature branch 工作流
- 分支命名规范：
  - `feature/xxx` - 新功能
  - `fix/xxx` - Bug 修复
  - `docs/xxx` - 文档更新
  - `refactor/xxx` - 重构
  - `test/xxx` - 测试相关
- Commit 消息规范（Conventional Commits）：
  ```
  <type>(<scope>): <subject>
  
  <body>
  
  <footer>
  ```
  - type: feat, fix, docs, style, refactor, test, chore
  - scope: 影响的模块/组件
  - subject: 简短描述（50 字符以内）

### 代码质量标准
- **Go 代码：**
  - `golangci-lint` 无错误和警告
  - 测试覆盖率 ≥ 80%
  - 循环复杂度 < 15
  - 函数长度 < 100 行

- **React 代码：**
  - Biome 检查通过
  - 测试覆盖率 ≥ 80%
  - 组件复杂度合理（< 200 行）
  - 无 console.log（生产代码）

### 性能监控
- 使用 profiling 工具定期检查性能
- Go: `pprof`
- React: React DevTools Profiler
- 监控关键指标：
  - API 响应时间
  - 页面加载时间
  - 内存使用
  - 错误率

### 安全规范
- 不在代码中硬编码敏感信息
- 使用环境变量管理配置
- 所有用户输入必须验证和净化
- 依赖包定期更新和安全扫描
- SQL 查询使用参数化防止注入
- API 实施适当的认证和授权

### 依赖管理
- Go: 使用 `go.mod` 管理依赖，定期执行 `go mod tidy`
- React: 使用 `package.json`，避免使用已废弃的包
- 定期更新依赖，检查安全漏洞
- 锁定关键依赖版本

## 编码原则

### SOLID 原则
- **S**ingle Responsibility - 单一职责
- **O**pen/Closed - 开闭原则
- **L**iskov Substitution - 里氏替换
- **I**nterface Segregation - 接口隔离
- **D**ependency Inversion - 依赖倒置

### DRY (Don't Repeat Yourself)
- 避免重复代码
- 提取公共逻辑到工具函数
- 使用组件组合而非复制粘贴

### KISS (Keep It Simple, Stupid)
- 优先选择简单的解决方案
- 避免过度设计
- 代码应该易于理解和维护

### YAGNI (You Aren't Gonna Need It)
- 不要添加当前不需要的功能
- 根据实际需求进行开发
- 避免预测性编程

## 开发工具

### 推荐工具
- **Go:**
  - IDE: VS Code with Go extension / GoLand
  - Linter: golangci-lint
  - Testing: go test, testify
  - Profiling: pprof

- **React:**
  - IDE: VS Code with extensions
  - Formatter: Biome
  - Testing: Vitest, Playwright
  - DevTools: React Developer Tools

### 任务运行
- 使用 Taskfile.yaml 管理项目任务
- 运行任务：`task <task-name>`
- 查看可用任务：`task --list`

## 贡献指南

1. Fork 项目并创建 feature branch
2. 遵循上述所有编码规范和最佳实践
3. 编写测试确保覆盖率
4. 更新相关文档
5. 提交 Pull Request
6. 等待代码审查和 CI 检查通过

## 许可证

- Console: Business Source License (BSL)
- 参考 `licenses/` 目录了解详细信息

---

**注意：** 在生成代码时，Copilot 应始终遵循以上指南，确保代码质量、可维护性和团队协作效率。
