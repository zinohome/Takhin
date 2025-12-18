#!/bin/bash
# Console API 认证功能测试脚本

set -e

API_URL="${API_URL:-http://localhost:8080}"
VALID_KEY="${VALID_KEY:-test-key-1}"
INVALID_KEY="invalid-key-xxx"

echo "======================================"
echo "Console API 认证功能测试"
echo "======================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试健康检查（无需认证）
echo -e "${YELLOW}1. 测试健康检查端点（无需认证）${NC}"
response=$(curl -s -w "\n%{http_code}" "$API_URL/api/health")
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | head -n -1)

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ 健康检查成功: $body${NC}"
else
    echo -e "${RED}✗ 健康检查失败: HTTP $http_code${NC}"
    exit 1
fi
echo ""

# 测试 Swagger 文档（无需认证）
echo -e "${YELLOW}2. 测试 Swagger 文档端点（无需认证）${NC}"
response=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/swagger/index.html")

if [ "$response" -eq 200 ]; then
    echo -e "${GREEN}✓ Swagger 文档访问成功${NC}"
else
    echo -e "${RED}✗ Swagger 文档访问失败: HTTP $response${NC}"
    exit 1
fi
echo ""

# 测试缺少认证头
echo -e "${YELLOW}3. 测试缺少认证头（应该返回 401）${NC}"
response=$(curl -s -w "\n%{http_code}" "$API_URL/api/topics")
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | head -n -1)

if [ "$http_code" -eq 401 ]; then
    echo -e "${GREEN}✓ 正确拒绝未认证请求: $body${NC}"
else
    echo -e "${RED}✗ 未正确拒绝: HTTP $http_code${NC}"
    exit 1
fi
echo ""

# 测试无效 API Key
echo -e "${YELLOW}4. 测试无效 API Key（应该返回 401）${NC}"
response=$(curl -s -H "Authorization: $INVALID_KEY" -w "\n%{http_code}" "$API_URL/api/topics")
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | head -n -1)

if [ "$http_code" -eq 401 ]; then
    echo -e "${GREEN}✓ 正确拒绝无效 API Key: $body${NC}"
else
    echo -e "${RED}✗ 未正确拒绝: HTTP $http_code${NC}"
    exit 1
fi
echo ""

# 测试有效 API Key（直接格式）
echo -e "${YELLOW}5. 测试有效 API Key - 直接格式（应该返回 200）${NC}"
response=$(curl -s -H "Authorization: $VALID_KEY" -w "\n%{http_code}" "$API_URL/api/topics")
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | head -n -1)

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ 直接格式 API Key 认证成功${NC}"
else
    echo -e "${RED}✗ 认证失败: HTTP $http_code - $body${NC}"
    exit 1
fi
echo ""

# 测试有效 API Key（Bearer 格式）
echo -e "${YELLOW}6. 测试有效 API Key - Bearer 格式（应该返回 200）${NC}"
response=$(curl -s -H "Authorization: Bearer $VALID_KEY" -w "\n%{http_code}" "$API_URL/api/topics")
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | head -n -1)

if [ "$http_code" -eq 200 ]; then
    echo -e "${GREEN}✓ Bearer 格式 API Key 认证成功${NC}"
else
    echo -e "${RED}✗ 认证失败: HTTP $http_code - $body${NC}"
    exit 1
fi
echo ""

# 测试创建 Topic（需要认证）
echo -e "${YELLOW}7. 测试创建 Topic（使用有效 API Key）${NC}"
response=$(curl -s -X POST \
    -H "Authorization: Bearer $VALID_KEY" \
    -H "Content-Type: application/json" \
    -d '{"name":"auth-test-topic","partitions":3}' \
    -w "\n%{http_code}" \
    "$API_URL/api/topics")
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | head -n -1)

if [ "$http_code" -eq 201 ]; then
    echo -e "${GREEN}✓ Topic 创建成功（认证通过）${NC}"
else
    echo -e "${YELLOW}! Topic 创建响应: HTTP $http_code - $body${NC}"
fi
echo ""

echo "======================================"
echo -e "${GREEN}所有认证测试通过！${NC}"
echo "======================================"
