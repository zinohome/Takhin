# Swagger UI 认证使用指南

## 概述

Console API 的 Swagger UI 现在支持 API Key 认证测试。本指南介绍如何在 Swagger UI 中使用认证功能。

## 访问 Swagger UI

1. 启动 Console API（启用认证）：

```bash
./console \
  -data-dir=/tmp/takhin-data \
  -api-addr=:8080 \
  -enable-auth \
  -api-keys="demo-key-123,admin-key-456"
```

2. 在浏览器中打开 Swagger UI：

```
http://localhost:8080/swagger/index.html
```

## 在 Swagger UI 中配置认证

### 步骤 1: 找到 Authorize 按钮

在 Swagger UI 页面的右上角，你会看到一个 **Authorize** 按钮（通常带有锁图标 🔓）。

### 步骤 2: 点击 Authorize 按钮

点击后会弹出认证对话框，显示可用的认证方案：

```
ApiKeyAuth (apiKey)
  API Key authentication. Use 'your-api-key' or 'Bearer your-api-key' format.

  Value: [输入框]
  
  [Authorize] [Close]
```

### 步骤 3: 输入 API Key

在 **Value** 输入框中输入你的 API Key，支持两种格式：

**方式 1: 直接输入 Key**
```
demo-key-123
```

**方式 2: 使用 Bearer 前缀**
```
Bearer demo-key-123
```

### 步骤 4: 点击 Authorize

点击 **Authorize** 按钮完成认证。成功后：
- 锁图标会变为 🔒（已锁定状态）
- 对话框显示 "Authorized"
- 所有需要认证的端点会自动带上 Authorization header

### 步骤 5: 测试 API

现在你可以测试任何 API 端点：

1. 展开任意端点（如 `GET /api/topics`）
2. 点击 **Try it out** 按钮
3. 填写必需的参数（如果有）
4. 点击 **Execute** 按钮
5. 查看响应结果

Swagger UI 会自动在请求中添加 Authorization header。

## 认证状态识别

### 未认证状态
- 锁图标显示为 🔓（开锁）
- 需要认证的端点标记为灰色锁图标
- 测试这些端点会返回 401 Unauthorized

### 已认证状态
- 锁图标显示为 🔒（已锁）
- 需要认证的端点标记为绿色锁图标
- 可以正常测试所有端点

## 端点认证要求

### 🔓 无需认证（公开访问）

这些端点始终可以访问，无需 API Key：

- `GET /api/health` - 健康检查

### 🔒 需要认证

这些端点需要有效的 API Key：

**Topics**:
- `GET /api/topics` - 列出所有 Topics
- `GET /api/topics/{topic}` - 获取 Topic 详情
- `POST /api/topics` - 创建 Topic
- `DELETE /api/topics/{topic}` - 删除 Topic

**Messages**:
- `GET /api/topics/{topic}/messages` - 读取消息
- `POST /api/topics/{topic}/messages` - 生产消息

**Consumer Groups**:
- `GET /api/consumer-groups` - 列出 Consumer Groups
- `GET /api/consumer-groups/{group}` - 获取 Consumer Group 详情

## 示例场景

### 场景 1: 创建 Topic

1. 点击 **Authorize** 并输入 `demo-key-123`
2. 展开 `POST /api/topics`
3. 点击 **Try it out**
4. 输入请求体：
   ```json
   {
     "name": "test-topic",
     "partitions": 3
   }
   ```
5. 点击 **Execute**
6. 查看响应（应该返回 201 Created）

### 场景 2: 测试无效 API Key

1. 点击 **Authorize**
2. 输入错误的 key: `invalid-key`
3. 点击 **Authorize**
4. 尝试任何需要认证的端点
5. 应该返回 401 Unauthorized 和错误消息：
   ```json
   {
     "error": "invalid API key"
   }
   ```

### 场景 3: 切换 API Key

如果你有多个 API Key：

1. 点击 **Authorize** 按钮
2. 点击 **Logout** 注销当前认证
3. 输入新的 API Key
4. 点击 **Authorize**

## 查看请求详情

Swagger UI 会显示实际发送的请求信息：

**Request URL**:
```
http://localhost:8080/api/topics
```

**Request Headers**:
```
Authorization: demo-key-123
```
或
```
Authorization: Bearer demo-key-123
```

**cURL 命令**:
```bash
curl -X 'GET' \
  'http://localhost:8080/api/topics' \
  -H 'accept: application/json' \
  -H 'Authorization: demo-key-123'
```

你可以直接复制 cURL 命令在终端中测试。

## 常见问题

### Q: 为什么点击 Execute 后返回 401？

**A**: 检查以下几点：
1. 确认已点击 **Authorize** 按钮并输入正确的 API Key
2. 确认服务器启动时使用了 `-enable-auth` 参数
3. 确认输入的 Key 在 `-api-keys` 列表中
4. 检查 Key 是否有多余的空格或特殊字符

### Q: 如何知道哪些端点需要认证？

**A**: 在 Swagger UI 中：
- 每个端点的右侧有一个锁图标
- 灰色锁 🔓 = 需要认证但未认证
- 绿色锁 🔒 = 需要认证且已认证
- 无锁图标 = 无需认证（公开端点）

### Q: Bearer 前缀是必需的吗？

**A**: 不是。两种格式都支持：
- `demo-key-123` ✅
- `Bearer demo-key-123` ✅

Swagger UI 会自动处理。

### Q: 如何清除认证？

**A**: 
1. 点击右上角的 **Authorize** 按钮
2. 在对话框中点击 **Logout** 按钮
3. 锁图标会变回 🔓 状态

### Q: 可以保存认证状态吗？

**A**: Swagger UI 的认证状态不会持久化。刷新页面后需要重新认证。

## 开发技巧

### 使用浏览器开发者工具

打开浏览器的开发者工具（F12）查看网络请求：

1. 切换到 **Network** 标签
2. 在 Swagger UI 中执行 API 请求
3. 查看请求详情：
   - Headers 中应该包含 `Authorization: your-key`
   - 如果返回 401，检查 Response 中的错误消息

### 快速测试脚本

从 Swagger UI 复制 cURL 命令并保存为脚本：

```bash
#!/bin/bash
# test-api.sh

API_KEY="demo-key-123"
API_URL="http://localhost:8080"

# List topics
curl -X 'GET' \
  "$API_URL/api/topics" \
  -H "Authorization: $API_KEY"

# Create topic
curl -X 'POST' \
  "$API_URL/api/topics" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "script-topic",
    "partitions": 3
  }'
```

## 安全提示

### ⚠️ 开发环境

- 可以使用简单的 API Key（如 `demo-key-123`）
- Swagger UI 是公开访问的
- 所有用户都能看到 API 文档

### 🔒 生产环境

- 使用强随机生成的 API Key
- 考虑通过反向代理保护 Swagger UI
- 使用 HTTPS 加密传输
- 定期轮换 API Keys
- 不要在公共网络暴露 Swagger UI

生成安全的 API Key：
```bash
# 使用 openssl
openssl rand -hex 32

# 使用 uuidgen
uuidgen

# 输出示例
# a3f8e9d2c1b4567890abcdef12345678901234567890abcdef1234567890ab
```

## 相关资源

- **API 文档**: [console-api-implementation.md](../../docs/console-api-implementation.md)
- **认证指南**: [AUTH.md](AUTH.md)
- **测试脚本**: [test-auth.sh](../../backend/test-auth.sh)
- **Swagger 规范**: http://localhost:8080/swagger/doc.json

---

**版本**: v1.0  
**最后更新**: 2025-12-18
