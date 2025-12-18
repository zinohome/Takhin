# Console API 认证功能

## 概述

Console API 支持基于 API Key 的认证机制，用于保护 API 端点的安全访问。

## 功能特性

- ✅ API Key 认证中间件
- ✅ 可选启用/禁用认证
- ✅ 支持多个 API Key
- ✅ 支持标准 Bearer token 格式
- ✅ 路径豁免机制（健康检查和文档无需认证）
- ✅ 统一的错误响应
- ✅ 完整的测试覆盖（84.8%）

## 快速开始

### 1. 启动服务器（启用认证）

```bash
./console \
  -data-dir=/tmp/takhin-data \
  -api-addr=:8080 \
  -enable-auth \
  -api-keys="secret-key-1,secret-key-2,secret-key-3"
```

**参数说明**:
- `-enable-auth`: 启用 API Key 认证（默认：false）
- `-api-keys`: 逗号分隔的有效 API Key 列表

### 2. 发送认证请求

#### 方式 1: 直接格式

```bash
curl -H "Authorization: secret-key-1" http://localhost:8080/api/topics
```

#### 方式 2: Bearer Token 格式

```bash
curl -H "Authorization: Bearer secret-key-1" http://localhost:8080/api/topics
```

### 3. 无需认证的端点

以下端点始终可以公开访问（无需 API Key）：

- `/api/health` - 健康检查
- `/swagger/*` - API 文档和 Swagger UI

## 命令行参数详解

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-enable-auth` | bool | false | 是否启用 API Key 认证 |
| `-api-keys` | string | "" | 有效的 API Key 列表（逗号分隔） |

## 使用示例

### 场景 1: 开发环境（禁用认证）

```bash
./console -data-dir=/tmp/dev-data -api-addr=:8080
```

所有端点都可以直接访问，无需 API Key。

### 场景 2: 测试环境（启用单个 Key）

```bash
./console \
  -data-dir=/tmp/test-data \
  -api-addr=:8080 \
  -enable-auth \
  -api-keys="test-key-123"
```

使用 API Key 访问：

```bash
# 创建 Topic
curl -X POST \
  -H "Authorization: test-key-123" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-topic","partitions":3}' \
  http://localhost:8080/api/topics

# 列出 Topics
curl -H "Authorization: test-key-123" \
  http://localhost:8080/api/topics

# 健康检查（无需 key）
curl http://localhost:8080/api/health
```

### 场景 3: 生产环境（启用多个 Key）

```bash
./console \
  -data-dir=/var/lib/takhin \
  -api-addr=:8080 \
  -enable-auth \
  -api-keys="prod-key-app1,prod-key-app2,prod-key-admin"
```

不同的应用程序可以使用各自的 API Key 访问：

```bash
# App1 使用自己的 key
curl -H "Authorization: prod-key-app1" http://localhost:8080/api/topics

# App2 使用自己的 key
curl -H "Authorization: Bearer prod-key-app2" http://localhost:8080/api/topics
```

## 错误处理

### 401 Unauthorized - 缺少认证头

```bash
$ curl http://localhost:8080/api/topics
{"error":"missing authorization header"}
```

### 401 Unauthorized - 无效 API Key

```bash
$ curl -H "Authorization: wrong-key" http://localhost:8080/api/topics
{"error":"invalid API key"}
```

### 200 OK - 认证成功

```bash
$ curl -H "Authorization: secret-key-1" http://localhost:8080/api/topics
[{"name":"my-topic","partitions":3}]
```

## 安全最佳实践

### 1. 使用强密钥

生成安全的 API Key：

```bash
# 使用 openssl 生成随机密钥
openssl rand -hex 32

# 使用 uuidgen
uuidgen

# 示例输出
# a3f8e9d2c1b4567890abcdef12345678901234567890abcdef1234567890ab
```

### 2. 密钥轮换

定期更换 API Key：

```bash
# 停止服务器
kill <pid>

# 使用新的 keys 启动
./console -enable-auth -api-keys="new-key-1,new-key-2"
```

### 3. 使用环境变量

避免在命令行中暴露密钥：

```bash
# 设置环境变量
export TAKHIN_API_KEYS="secret-key-1,secret-key-2"

# 在脚本中使用
./console -enable-auth -api-keys="$TAKHIN_API_KEYS"
```

### 4. HTTPS 部署

在生产环境中，始终使用 HTTPS：

```bash
# 使用反向代理（Nginx/Caddy）添加 TLS
# 或使用 go-chi 的 TLS 支持
```

## 自动化测试

运行认证功能的集成测试：

```bash
# 启动服务器（测试模式）
./console -data-dir=/tmp/test -api-addr=:8080 -enable-auth -api-keys="test-key-1" &

# 运行测试脚本
./test-auth.sh

# 或手动设置测试参数
API_URL=http://localhost:8080 VALID_KEY=test-key-1 ./test-auth.sh
```

## 单元测试

查看认证中间件的单元测试：

```bash
# 运行认证测试
go test ./pkg/console/ -v -run TestAuth

# 输出示例
=== RUN   TestAuthMiddleware
=== RUN   TestAuthMiddleware/authentication_disabled_-_should_pass
=== RUN   TestAuthMiddleware/valid_API_key_-_should_pass
=== RUN   TestAuthMiddleware/valid_API_key_with_Bearer_prefix_-_should_pass
=== RUN   TestAuthMiddleware/invalid_API_key_-_should_fail
=== RUN   TestAuthMiddleware/missing_authorization_header_-_should_fail
=== RUN   TestAuthMiddleware/health_check_path_-_should_skip_auth
=== RUN   TestAuthMiddleware/swagger_path_-_should_skip_auth
--- PASS: TestAuthMiddleware (0.00s)
PASS
```

## 代码示例

### Go 客户端

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

func main() {
    apiKey := "your-api-key"
    apiURL := "http://localhost:8080"

    // 创建 HTTP 客户端
    client := &http.Client{}

    // 创建 Topic
    topicData := map[string]interface{}{
        "name":       "go-client-topic",
        "partitions": 3,
    }
    body, _ := json.Marshal(topicData)

    req, _ := http.NewRequest("POST", apiURL+"/api/topics", bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, _ := client.Do(req)
    defer resp.Body.Close()

    fmt.Println("Status:", resp.Status)
}
```

### Python 客户端

```python
import requests

api_key = "your-api-key"
api_url = "http://localhost:8080"

# 设置认证头
headers = {
    "Authorization": f"Bearer {api_key}",
    "Content-Type": "application/json"
}

# 列出 Topics
response = requests.get(f"{api_url}/api/topics", headers=headers)
print(f"Topics: {response.json()}")

# 创建 Topic
topic_data = {
    "name": "python-client-topic",
    "partitions": 3
}
response = requests.post(f"{api_url}/api/topics", json=topic_data, headers=headers)
print(f"Created: {response.status_code}")
```

### JavaScript/TypeScript 客户端

```typescript
const apiKey = "your-api-key";
const apiURL = "http://localhost:8080";

// 使用 Fetch API
async function listTopics() {
  const response = await fetch(`${apiURL}/api/topics`, {
    headers: {
      "Authorization": `Bearer ${apiKey}`
    }
  });
  return response.json();
}

// 创建 Topic
async function createTopic(name: string, partitions: number) {
  const response = await fetch(`${apiURL}/api/topics`, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${apiKey}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ name, partitions })
  });
  return response.json();
}

// 使用
listTopics().then(topics => console.log("Topics:", topics));
createTopic("js-client-topic", 3).then(result => console.log("Created:", result));
```

## 限制和未来改进

### 当前限制

- ✅ 仅支持 API Key 认证（不支持 JWT）
- ✅ API Key 在内存中明文存储
- ✅ 不支持 Key 的细粒度权限控制
- ✅ 不支持 Key 过期时间

### 未来改进方向

1. **JWT 认证**: 支持 JSON Web Tokens
2. **密钥加密**: 使用加密存储 API Keys
3. **权限系统**: 基于角色的访问控制（RBAC）
4. **审计日志**: 记录所有认证和授权事件
5. **速率限制**: 按 API Key 限制请求频率
6. **密钥管理**: 提供 API 端点管理 Keys

## 技术实现

### AuthConfig 结构

```go
type AuthConfig struct {
    Enabled bool     // 是否启用认证
    APIKeys []string // 有效的 API Key 列表
}
```

### AuthMiddleware

```go
func AuthMiddleware(config AuthConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. 检查是否启用认证
            if !config.Enabled {
                next.ServeHTTP(w, r)
                return
            }

            // 2. 跳过豁免路径
            if strings.HasPrefix(r.URL.Path, "/swagger") || 
               r.URL.Path == "/api/health" {
                next.ServeHTTP(w, r)
                return
            }

            // 3. 验证 API Key
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                respondError(w, http.StatusUnauthorized, 
                    "missing authorization header")
                return
            }

            key := strings.TrimPrefix(authHeader, "Bearer ")
            if !isValidAPIKey(key, config.APIKeys) {
                respondError(w, http.StatusUnauthorized, 
                    "invalid API key")
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## 支持和反馈

如有问题或建议，请：
- 查看 [Console API 实现文档](../../docs/console-api-implementation.md)
- 提交 Issue 或 Pull Request

---

**版本**: v1.0  
**最后更新**: 2025-12-18
