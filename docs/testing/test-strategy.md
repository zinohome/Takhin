# 测试方案

## 1. 测试策略

### 1.1 测试目标
- 确保功能正确性
- 保证系统稳定性
- 验证性能指标
- 检查安全性
- 确认兼容性

### 1.2 测试原则
- **测试先行**: TDD 开发模式
- **自动化优先**: 最大化自动化测试覆盖
- **持续测试**: CI/CD 流程中集成测试
- **分层测试**: 单元、集成、E2E 全覆盖
- **左移测试**: 尽早发现问题

### 1.3 测试金字塔

```
           /\
          /E2E\           10% - 用户场景测试
         /------\
        /  集成  \        20% - 模块间交互测试
       /----------\
      /    单元    \      70% - 功能单元测试
     /--------------\
```

## 2. 单元测试

### 2.1 Go 单元测试

#### 测试框架
- **测试库**: Go 标准库 `testing`
- **断言库**: `testify/assert`
- **Mock 库**: `testify/mock` 或 `gomock`
- **覆盖率**: `go test -cover`

#### 测试示例

```go
// pkg/storage/log/segment_test.go
package log

import (
    "os"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSegment_Append(t *testing.T) {
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
            name:    "append empty message",
            offset:  1,
            data:    []byte(""),
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
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
            }
        })
    }
}

func TestSegment_Read(t *testing.T) {
    // Setup
    tmpDir := t.TempDir()
    segment, err := NewSegment(tmpDir, 0, 1024*1024)
    require.NoError(t, err)
    defer segment.Close()
    
    // Write test data
    testData := []byte("test message")
    err = segment.Append(0, testData)
    require.NoError(t, err)
    
    // Execute
    data, err := segment.Read(0, 1024)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, testData, data)
}

func BenchmarkSegment_Append(b *testing.B) {
    tmpDir := b.TempDir()
    segment, _ := NewSegment(tmpDir, 0, 1024*1024*100)
    defer segment.Close()
    
    data := make([]byte, 1024)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = segment.Append(int64(i), data)
    }
}
```

#### 表驱动测试

```go
func TestKafkaProtocol_ParseRequest(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        wantType APIKey
        wantErr  bool
    }{
        {
            name:     "produce request",
            input:    []byte{...},
            wantType: ProduceKey,
            wantErr:  false,
        },
        {
            name:     "fetch request",
            input:    []byte{...},
            wantType: FetchKey,
            wantErr:  false,
        },
        {
            name:     "invalid request",
            input:    []byte{0x00},
            wantType: 0,
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req, err := ParseRequest(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.wantType, req.APIKey)
        })
    }
}
```

### 2.2 TypeScript 单元测试

#### 测试框架
- **测试库**: Vitest
- **渲染库**: React Testing Library
- **Mock 库**: Vitest Mock
- **覆盖率**: Vitest Coverage

#### 测试示例

```typescript
// src/hooks/api/useTopic.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useTopic } from './useTopic';
import { topicService } from '@/services/topic.service';

vi.mock('@/services/topic.service');

describe('useTopic', () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
  
  beforeEach(() => {
    vi.clearAllMocks();
  });
  
  it('should fetch topic details', async () => {
    const mockTopic = {
      name: 'test-topic',
      partitions: 3,
      replicationFactor: 2,
    };
    
    vi.mocked(topicService.getTopicDetails).mockResolvedValue(mockTopic);
    
    const { result } = renderHook(
      () => useTopic('test-topic'),
      { wrapper }
    );
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    
    expect(result.current.data).toEqual(mockTopic);
    expect(topicService.getTopicDetails).toHaveBeenCalledWith('test-topic');
  });
  
  it('should handle error', async () => {
    const error = new Error('Failed to fetch topic');
    vi.mocked(topicService.getTopicDetails).mockRejectedValue(error);
    
    const { result } = renderHook(
      () => useTopic('test-topic'),
      { wrapper }
    );
    
    await waitFor(() => expect(result.current.isError).toBe(true));
    
    expect(result.current.error).toEqual(error);
  });
});
```

#### 组件测试

```typescript
// src/components/features/TopicCard/TopicCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { TopicCard } from './TopicCard';

describe('TopicCard', () => {
  const mockTopic = {
    name: 'test-topic',
    partitions: 3,
    replicationFactor: 2,
    messageCount: 1000,
    size: 1024 * 1024,
  };
  
  it('should render topic information', () => {
    render(<TopicCard topic={mockTopic} />);
    
    expect(screen.getByText('test-topic')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
  });
  
  it('should call onClick when clicked', () => {
    const handleClick = vi.fn();
    render(<TopicCard topic={mockTopic} onClick={handleClick} />);
    
    fireEvent.click(screen.getByText('test-topic'));
    
    expect(handleClick).toHaveBeenCalledWith(mockTopic);
  });
  
  it('should display formatted size', () => {
    render(<TopicCard topic={mockTopic} />);
    
    expect(screen.getByText('1.00 MB')).toBeInTheDocument();
  });
});
```

### 2.3 覆盖率要求

| 项目 | 最低覆盖率 | 目标覆盖率 |
|------|-----------|-----------|
| 核心模块 | 80% | 90% |
| 业务逻辑 | 80% | 85% |
| 工具函数 | 90% | 95% |
| UI 组件 | 70% | 80% |

## 3. 集成测试

### 3.1 后端集成测试

#### 测试环境
- **容器化**: 使用 testcontainers-go
- **依赖服务**: Kafka, Schema Registry
- **数据隔离**: 每个测试使用独立数据

#### 测试示例

```go
// tests/integration/api_test.go
package integration

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
)

func TestTopicAPI_CreateAndList(t *testing.T) {
    // Setup test environment
    ctx := context.Background()
    
    // Start Takhin Core container
    coreContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "takhin-core:test",
            ExposedPorts: []string{"9092/tcp", "9644/tcp"},
            WaitingFor:   wait.ForLog("Server started"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer coreContainer.Terminate(ctx)
    
    // Start Console Backend container
    consoleContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "takhin-console-backend:test",
            ExposedPorts: []string{"8080/tcp"},
            Env: map[string]string{
                "KAFKA_BROKERS": coreContainer.GetIPAddress(ctx) + ":9092",
            },
            WaitingFor: wait.ForLog("Server started"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer consoleContainer.Terminate(ctx)
    
    // Get API endpoint
    apiURL, err := consoleContainer.Endpoint(ctx, "http")
    require.NoError(t, err)
    
    client := NewAPIClient(apiURL)
    
    // Test: Create topic
    createReq := &CreateTopicRequest{
        Name:              "test-topic",
        Partitions:        3,
        ReplicationFactor: 1,
    }
    err = client.CreateTopic(ctx, createReq)
    assert.NoError(t, err)
    
    // Test: List topics
    topics, err := client.ListTopics(ctx)
    assert.NoError(t, err)
    assert.Len(t, topics, 1)
    assert.Equal(t, "test-topic", topics[0].Name)
    assert.Equal(t, 3, topics[0].Partitions)
}
```

### 3.2 前端集成测试

#### 测试框架
- **测试库**: Vitest
- **API Mock**: MSW (Mock Service Worker)

#### 测试示例

```typescript
// tests/integration/topic-flow.test.tsx
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { rest } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TopicList } from '@/pages/Topics/TopicList';

const server = setupServer(
  rest.get('/api/v1/topics', (req, res, ctx) => {
    return res(ctx.json([
      { name: 'topic-1', partitions: 3, replicationFactor: 2 },
      { name: 'topic-2', partitions: 5, replicationFactor: 3 },
    ]));
  }),
  
  rest.post('/api/v1/topics', async (req, res, ctx) => {
    const body = await req.json();
    return res(ctx.json({ message: 'Topic created' }));
  })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('Topic Management Flow', () => {
  it('should display topics and create new topic', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    
    render(
      <QueryClientProvider client={queryClient}>
        <TopicList />
      </QueryClientProvider>
    );
    
    // Wait for topics to load
    await waitFor(() => {
      expect(screen.getByText('topic-1')).toBeInTheDocument();
      expect(screen.getByText('topic-2')).toBeInTheDocument();
    });
    
    // Click create button
    fireEvent.click(screen.getByText('Create Topic'));
    
    // Fill form
    fireEvent.change(screen.getByLabelText('Topic Name'), {
      target: { value: 'new-topic' },
    });
    fireEvent.change(screen.getByLabelText('Partitions'), {
      target: { value: '3' },
    });
    
    // Submit form
    fireEvent.click(screen.getByText('Create'));
    
    // Verify success message
    await waitFor(() => {
      expect(screen.getByText('Topic created')).toBeInTheDocument();
    });
  });
});
```

## 4. E2E 测试

### 4.1 测试框架
- **框架**: Playwright
- **浏览器**: Chromium, Firefox, WebKit
- **并行执行**: 多浏览器并行测试

### 4.2 测试场景

#### 核心业务流程

```typescript
// tests/e2e/topic-lifecycle.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Topic Lifecycle', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3000');
  });
  
  test('complete topic lifecycle', async ({ page }) => {
    // Navigate to topics page
    await page.click('text=Topics');
    await expect(page).toHaveURL('/topics');
    
    // Create new topic
    await page.click('text=Create Topic');
    await page.fill('[name="name"]', 'e2e-test-topic');
    await page.fill('[name="partitions"]', '3');
    await page.fill('[name="replicationFactor"]', '2');
    await page.click('button:has-text("Create")');
    
    // Verify topic appears in list
    await expect(page.locator('text=e2e-test-topic')).toBeVisible();
    
    // View topic details
    await page.click('text=e2e-test-topic');
    await expect(page.locator('h1')).toContainText('e2e-test-topic');
    await expect(page.locator('text=Partitions: 3')).toBeVisible();
    
    // Edit topic configuration
    await page.click('text=Edit Config');
    await page.fill('[name="retention.ms"]', '86400000');
    await page.click('button:has-text("Save")');
    await expect(page.locator('text=Config updated')).toBeVisible();
    
    // Delete topic
    await page.click('text=Delete Topic');
    await page.click('button:has-text("Confirm")');
    await expect(page).toHaveURL('/topics');
    await expect(page.locator('text=e2e-test-topic')).not.toBeVisible();
  });
});
```

#### 消息查看器流程

```typescript
// tests/e2e/message-viewer.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Message Viewer', () => {
  test('search and filter messages', async ({ page }) => {
    await page.goto('http://localhost:3000/messages');
    
    // Select topic
    await page.selectOption('[name="topic"]', 'test-topic');
    
    // Select partition
    await page.selectOption('[name="partition"]', '0');
    
    // Select encoding
    await page.selectOption('[name="encoding"]', 'json');
    
    // Enter filter code
    await page.fill('[name="filterCode"]', `
      return message.value.userId === '123';
    `);
    
    // Click search
    await page.click('button:has-text("Search")');
    
    // Wait for results
    await expect(page.locator('.message-list')).toBeVisible();
    
    // Verify messages are displayed
    const messages = page.locator('.message-item');
    await expect(messages).toHaveCount(5);
    
    // Expand first message
    await messages.first().click();
    await expect(page.locator('.message-details')).toBeVisible();
    
    // Verify message content
    await expect(page.locator('.message-value')).toContainText('userId');
  });
});
```

### 4.3 视觉回归测试

```typescript
// tests/e2e/visual-regression.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Visual Regression', () => {
  test('dashboard snapshot', async ({ page }) => {
    await page.goto('http://localhost:3000/dashboard');
    await page.waitForLoadState('networkidle');
    
    // Take screenshot
    await expect(page).toHaveScreenshot('dashboard.png');
  });
  
  test('topic list snapshot', async ({ page }) => {
    await page.goto('http://localhost:3000/topics');
    await page.waitForLoadState('networkidle');
    
    await expect(page).toHaveScreenshot('topic-list.png');
  });
});
```

## 5. 性能测试

### 5.1 基准测试

#### Go 基准测试

```go
// pkg/storage/log/segment_bench_test.go
func BenchmarkSegment_Append_1KB(b *testing.B) {
    benchmarkAppend(b, 1024)
}

func BenchmarkSegment_Append_10KB(b *testing.B) {
    benchmarkAppend(b, 10*1024)
}

func BenchmarkSegment_Append_100KB(b *testing.B) {
    benchmarkAppend(b, 100*1024)
}

func benchmarkAppend(b *testing.B, size int) {
    tmpDir := b.TempDir()
    segment, _ := NewSegment(tmpDir, 0, 1024*1024*100)
    defer segment.Close()
    
    data := make([]byte, size)
    
    b.SetBytes(int64(size))
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _ = segment.Append(int64(i), data)
    }
}
```

### 5.2 压力测试

#### 工具
- **后端**: k6, Apache JMeter
- **前端**: Lighthouse, WebPageTest

#### 测试脚本

```javascript
// tests/performance/produce-test.js
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 200 },  // Ramp up
    { duration: '5m', target: 200 },  // Stay at 200 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% requests < 500ms
    http_req_failed: ['rate<0.01'],     // Error rate < 1%
  },
};

export default function () {
  const url = 'http://localhost:8080/api/v1/topics/test-topic/produce';
  const payload = JSON.stringify({
    messages: [
      {
        key: 'test-key',
        value: { data: 'test message' },
      },
    ],
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  const res = http.post(url, payload, params);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
}
```

### 5.3 性能指标

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| 吞吐量 | >100K msg/s | k6 压测 |
| P99 延迟 | <10ms | k6 压测 |
| 首屏加载 | <2s | Lighthouse |
| 内存使用 | <8GB | 压测监控 |
| CPU 使用 | <80% | 压测监控 |

## 6. 兼容性测试

### 6.1 Kafka 客户端兼容性

测试与主流 Kafka 客户端的兼容性:

- kafka-python
- confluent-kafka-go
- sarama (Go)
- KafkaJS (Node.js)
- Java Kafka Client

### 6.2 浏览器兼容性

| 浏览器 | 版本 | 测试状态 |
|--------|------|---------|
| Chrome | 最新 2 个版本 | ✅ |
| Firefox | 最新 2 个版本 | ✅ |
| Safari | 最新 2 个版本 | ✅ |
| Edge | 最新 2 个版本 | ✅ |

## 7. 安全测试

### 7.1 安全扫描
- **依赖扫描**: Snyk, npm audit
- **代码扫描**: SonarQube, CodeQL
- **容器扫描**: Trivy, Clair

### 7.2 渗透测试
- SQL 注入
- XSS 攻击
- CSRF 攻击
- 权限绕过
- 敏感信息泄露

## 8. 测试自动化

### 8.1 CI/CD 集成

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  unit-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - name: Run unit tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out -o coverage.html
      - name: Upload coverage
        uses: codecov/codecov-action@v3
  
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start services
        run: docker-compose -f docker-compose.test.yml up -d
      - name: Run integration tests
        run: go test -v -tags=integration ./tests/integration/...
      - name: Stop services
        run: docker-compose -f docker-compose.test.yml down
  
  e2e-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '22'
      - name: Install dependencies
        run: npm ci
      - name: Install Playwright
        run: npx playwright install --with-deps
      - name: Run E2E tests
        run: npm run test:e2e
```

### 8.2 测试报告

- **单元测试**: Go test coverage, Vitest coverage
- **集成测试**: JUnit XML 格式
- **E2E 测试**: Playwright HTML 报告
- **性能测试**: k6 HTML 报告

---

**文档版本**: v1.0  
**最后更新**: 2025-12-14  
**维护者**: Takhin Team
