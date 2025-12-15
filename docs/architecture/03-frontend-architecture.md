# 前端架构设计

## 1. Console 前端整体架构

### 1.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    Takhin Console Frontend                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                   Presentation Layer                        │ │
│  ├────────────────────────────────────────────────────────────┤ │
│  │  Pages  │  Layouts  │  Components  │  Theme  │  Assets    │ │
│  └─────┬────────────────────────────────────────────────────────┘ │
│        │                                                          │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                   State Management                          │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ React Query (Server State) │ Context (UI State)            │ │
│  │ Zustand (Global State)     │ Form State (React Hook Form) │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                          │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                   Business Logic                            │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ Custom Hooks │ Utils │ Helpers │ Transformers              │ │
│  └─────┬───────────────────────────────────────────────────────┘ │
│        │                                                          │
│  ┌─────▼───────────────────────────────────────────────────────┐ │
│  │                    API Layer                                │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │ REST Client │ gRPC Client │ WebSocket │ Auth Interceptor  │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 技术栈详细说明

#### 1.2.1 核心框架
- **React 18.3+**: 使用最新特性 (Concurrent Features, Suspense)
- **TypeScript 5.3+**: 严格类型检查
- **Rsbuild**: 基于 Rspack 的构建工具，比 Webpack 快 10x

#### 1.2.2 UI 框架
- **Chakra UI**: 组件库，提供完整的设计系统
- **React Icons**: 图标库
- **Framer Motion**: 动画库

#### 1.2.3 路由和导航
- **React Router v6**: 声明式路由
- **History API**: 浏览器历史管理

#### 1.2.4 状态管理
- **React Query (TanStack Query)**: 服务端状态管理
- **Zustand**: 轻量级全局状态管理
- **Context API**: UI 状态 (主题、语言等)
- **React Hook Form**: 表单状态管理

#### 1.2.5 数据验证
- **Zod**: TypeScript-first 的 schema 验证

#### 1.2.6 HTTP 客户端
- **Axios**: HTTP 请求库
- **Connect (gRPC-Web)**: gRPC 客户端

#### 1.2.7 代码质量
- **Biome**: 统一的 linter 和 formatter (替代 ESLint + Prettier)
- **TypeScript**: 静态类型检查

#### 1.2.8 测试
- **Vitest**: 单元测试和集成测试
- **React Testing Library**: 组件测试
- **Playwright**: E2E 测试
- **MSW (Mock Service Worker)**: API Mock

### 1.3 目录结构

```
projects/console/frontend/
├── public/                     # 静态资源
│   ├── index.html
│   ├── favicon.ico
│   └── assets/
├── src/
│   ├── app.tsx                 # 应用入口
│   ├── index.tsx               # 渲染入口
│   ├── routes.tsx              # 路由配置
│   │
│   ├── pages/                  # 页面组件
│   │   ├── Dashboard/
│   │   ├── Topics/
│   │   │   ├── TopicList.tsx
│   │   │   ├── TopicDetails.tsx
│   │   │   ├── TopicCreate.tsx
│   │   │   └── TopicEdit.tsx
│   │   ├── Messages/
│   │   │   ├── MessageViewer.tsx
│   │   │   └── MessageSearch.tsx
│   │   ├── ConsumerGroups/
│   │   ├── SchemaRegistry/
│   │   ├── KafkaConnect/
│   │   ├── ACLs/
│   │   └── Settings/
│   │
│   ├── components/             # 可复用组件
│   │   ├── common/             # 通用组件
│   │   │   ├── Button/
│   │   │   ├── Input/
│   │   │   ├── Table/
│   │   │   ├── Card/
│   │   │   ├── Modal/
│   │   │   ├── Drawer/
│   │   │   ├── Toast/
│   │   │   ├── Loading/
│   │   │   ├── Empty/
│   │   │   └── Error/
│   │   ├── layout/             # 布局组件
│   │   │   ├── AppLayout/
│   │   │   ├── Sidebar/
│   │   │   ├── Header/
│   │   │   ├── Footer/
│   │   │   └── Breadcrumb/
│   │   ├── features/           # 功能组件
│   │   │   ├── TopicCard/
│   │   │   ├── MessageList/
│   │   │   ├── ConsumerGroupTable/
│   │   │   ├── SchemaEditor/
│   │   │   ├── ConnectorCard/
│   │   │   └── ACLTable/
│   │   └── charts/             # 图表组件
│   │       ├── LineChart/
│   │       ├── BarChart/
│   │       ├── PieChart/
│   │       └── Gauge/
│   │
│   ├── hooks/                  # 自定义 Hooks
│   │   ├── api/                # API Hooks
│   │   │   ├── useTopic.ts
│   │   │   ├── useMessage.ts
│   │   │   ├── useConsumerGroup.ts
│   │   │   ├── useSchema.ts
│   │   │   └── useConnector.ts
│   │   ├── ui/                 # UI Hooks
│   │   │   ├── useTheme.ts
│   │   │   ├── useToast.ts
│   │   │   ├── useModal.ts
│   │   │   └── useBreakpoint.ts
│   │   └── utils/              # 工具 Hooks
│   │       ├── useDebounce.ts
│   │       ├── useLocalStorage.ts
│   │       └── useClipboard.ts
│   │
│   ├── services/               # API 服务
│   │   ├── api.ts              # API 基础配置
│   │   ├── topic.service.ts
│   │   ├── message.service.ts
│   │   ├── consumer.service.ts
│   │   ├── schema.service.ts
│   │   ├── connector.service.ts
│   │   └── acl.service.ts
│   │
│   ├── store/                  # 全局状态
│   │   ├── auth.store.ts
│   │   ├── user.store.ts
│   │   └── settings.store.ts
│   │
│   ├── types/                  # TypeScript 类型定义
│   │   ├── topic.types.ts
│   │   ├── message.types.ts
│   │   ├── consumer.types.ts
│   │   ├── schema.types.ts
│   │   ├── connector.types.ts
│   │   ├── acl.types.ts
│   │   └── common.types.ts
│   │
│   ├── utils/                  # 工具函数
│   │   ├── format.ts           # 格式化函数
│   │   ├── validation.ts       # 验证函数
│   │   ├── transform.ts        # 数据转换
│   │   ├── date.ts             # 日期处理
│   │   └── string.ts           # 字符串处理
│   │
│   ├── constants/              # 常量
│   │   ├── routes.ts
│   │   ├── api.ts
│   │   └── config.ts
│   │
│   ├── theme/                  # 主题配置
│   │   ├── index.ts
│   │   ├── colors.ts
│   │   ├── fonts.ts
│   │   └── components.ts
│   │
│   ├── contexts/               # React Context
│   │   ├── AuthContext.tsx
│   │   ├── ThemeContext.tsx
│   │   └── ConfigContext.tsx
│   │
│   └── assets/                 # 资源文件
│       ├── images/
│       ├── icons/
│       └── fonts/
│
├── tests/                      # 测试文件
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
├── rsbuild.config.ts           # Rsbuild 配置
├── tsconfig.json               # TypeScript 配置
├── biome.jsonc                 # Biome 配置
├── vitest.config.ts            # Vitest 配置
└── playwright.config.ts        # Playwright 配置
```

## 2. 核心功能设计

### 2.1 Topic 管理

#### 2.1.1 Topic 列表页面

```tsx
// src/pages/Topics/TopicList.tsx
import { useState } from 'react';
import { Box, Button, Input, Table, Thead, Tbody, Tr, Th, Td } from '@chakra-ui/react';
import { useTopics } from '@/hooks/api/useTopic';
import { TopicCard } from '@/components/features/TopicCard';

export function TopicList() {
  const [search, setSearch] = useState('');
  const { data: topics, isLoading, error } = useTopics();
  
  // 过滤 topics
  const filteredTopics = topics?.filter(topic => 
    topic.name.toLowerCase().includes(search.toLowerCase())
  );
  
  if (isLoading) return <Loading />;
  if (error) return <Error error={error} />;
  
  return (
    <Box>
      <Flex justify="space-between" mb={4}>
        <Input 
          placeholder="Search topics..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          maxW="400px"
        />
        <Button colorScheme="blue" onClick={handleCreateTopic}>
          Create Topic
        </Button>
      </Flex>
      
      <Table>
        <Thead>
          <Tr>
            <Th>Name</Th>
            <Th>Partitions</Th>
            <Th>Replication Factor</Th>
            <Th>Messages</Th>
            <Th>Size</Th>
            <Th>Actions</Th>
          </Tr>
        </Thead>
        <Tbody>
          {filteredTopics?.map(topic => (
            <Tr key={topic.name}>
              <Td>{topic.name}</Td>
              <Td>{topic.partitions}</Td>
              <Td>{topic.replicationFactor}</Td>
              <Td>{formatNumber(topic.messageCount)}</Td>
              <Td>{formatBytes(topic.size)}</Td>
              <Td>
                <Button size="sm" onClick={() => navigateToDetails(topic.name)}>
                  View
                </Button>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    </Box>
  );
}
```

#### 2.1.2 Custom Hook for Topics

```tsx
// src/hooks/api/useTopic.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { topicService } from '@/services/topic.service';
import { Topic, CreateTopicRequest } from '@/types/topic.types';

// 查询所有 topics
export function useTopics() {
  return useQuery({
    queryKey: ['topics'],
    queryFn: () => topicService.listTopics(),
    staleTime: 30000, // 30 秒
    refetchInterval: 60000, // 每分钟自动刷新
  });
}

// 查询单个 topic 详情
export function useTopic(topicName: string) {
  return useQuery({
    queryKey: ['topics', topicName],
    queryFn: () => topicService.getTopicDetails(topicName),
    enabled: !!topicName, // 只有 topicName 存在时才查询
  });
}

// 创建 topic
export function useCreateTopic() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (request: CreateTopicRequest) => 
      topicService.createTopic(request),
    onSuccess: () => {
      // 创建成功后，使缓存失效，触发重新查询
      queryClient.invalidateQueries({ queryKey: ['topics'] });
    },
  });
}

// 删除 topic
export function useDeleteTopic() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (topicName: string) => 
      topicService.deleteTopic(topicName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['topics'] });
    },
  });
}

// 更新 topic 配置
export function useUpdateTopicConfig() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ topicName, config }: { topicName: string; config: Record<string, string> }) =>
      topicService.updateConfig(topicName, config),
    onSuccess: (_, { topicName }) => {
      queryClient.invalidateQueries({ queryKey: ['topics', topicName] });
    },
  });
}
```

### 2.2 消息查看器

#### 2.2.1 消息查看器组件

```tsx
// src/pages/Messages/MessageViewer.tsx
import { useState } from 'react';
import { Box, Button, Select, Input, Code } from '@chakra-ui/react';
import { useMessages } from '@/hooks/api/useMessage';
import { MessageList } from '@/components/features/MessageList';

export function MessageViewer() {
  const [topic, setTopic] = useState('');
  const [partition, setPartition] = useState<number>(0);
  const [encoding, setEncoding] = useState<'json' | 'avro' | 'protobuf'>('json');
  const [filterCode, setFilterCode] = useState('');
  
  const { 
    data: messages, 
    isLoading, 
    fetchNextPage,
    hasNextPage 
  } = useMessages({
    topic,
    partition,
    encoding,
    filterCode,
  });
  
  return (
    <Box>
      {/* 搜索栏 */}
      <Flex gap={4} mb={4}>
        <Select value={topic} onChange={(e) => setTopic(e.target.value)}>
          <option value="">Select Topic</option>
          {topics?.map(t => (
            <option key={t.name} value={t.name}>{t.name}</option>
          ))}
        </Select>
        
        <Select value={partition} onChange={(e) => setPartition(Number(e.target.value))}>
          {Array.from({ length: partitionCount }, (_, i) => (
            <option key={i} value={i}>Partition {i}</option>
          ))}
        </Select>
        
        <Select value={encoding} onChange={(e) => setEncoding(e.target.value as any)}>
          <option value="json">JSON</option>
          <option value="avro">Avro</option>
          <option value="protobuf">Protocol Buffers</option>
        </Select>
      </Flex>
      
      {/* JavaScript 过滤器 */}
      <Box mb={4}>
        <Text mb={2}>JavaScript Filter (optional)</Text>
        <Code
          as="textarea"
          value={filterCode}
          onChange={(e) => setFilterCode(e.target.value)}
          placeholder="// Example: return message.value.userId === '123'"
          rows={5}
        />
      </Box>
      
      {/* 消息列表 */}
      {isLoading ? (
        <Loading />
      ) : (
        <>
          <MessageList messages={messages} />
          {hasNextPage && (
            <Button onClick={() => fetchNextPage()} mt={4}>
              Load More
            </Button>
          )}
        </>
      )}
    </Box>
  );
}
```

#### 2.2.2 消息列表组件

```tsx
// src/components/features/MessageList/MessageList.tsx
import { useState } from 'react';
import { Box, Text, Code, Collapse, IconButton } from '@chakra-ui/react';
import { ChevronDownIcon, ChevronRightIcon } from '@chakra-ui/icons';
import { Message } from '@/types/message.types';
import { JsonViewer } from '@/components/common/JsonViewer';

interface MessageListProps {
  messages: Message[];
}

export function MessageList({ messages }: MessageListProps) {
  const [expandedMessages, setExpandedMessages] = useState<Set<string>>(new Set());
  
  const toggleMessage = (messageId: string) => {
    setExpandedMessages(prev => {
      const next = new Set(prev);
      if (next.has(messageId)) {
        next.delete(messageId);
      } else {
        next.add(messageId);
      }
      return next;
    });
  };
  
  return (
    <Box>
      {messages.map(message => {
        const messageId = `${message.partition}-${message.offset}`;
        const isExpanded = expandedMessages.has(messageId);
        
        return (
          <Box 
            key={messageId} 
            borderWidth="1px" 
            borderRadius="md" 
            p={4} 
            mb={2}
          >
            {/* 消息头 */}
            <Flex justify="space-between" align="center" mb={2}>
              <Flex align="center" gap={2}>
                <IconButton
                  size="sm"
                  icon={isExpanded ? <ChevronDownIcon /> : <ChevronRightIcon />}
                  onClick={() => toggleMessage(messageId)}
                  aria-label="Toggle message"
                />
                <Text fontWeight="bold">
                  Partition {message.partition} | Offset {message.offset}
                </Text>
              </Flex>
              <Text fontSize="sm" color="gray.500">
                {formatTimestamp(message.timestamp)}
              </Text>
            </Flex>
            
            {/* Key (如果存在) */}
            {message.key && (
              <Box mb={2}>
                <Text fontSize="sm" fontWeight="semibold">Key:</Text>
                <Code>{JSON.stringify(message.key)}</Code>
              </Box>
            )}
            
            {/* Value 预览 */}
            <Box mb={2}>
              <Text fontSize="sm" fontWeight="semibold">Value Preview:</Text>
              <Code noOfLines={isExpanded ? undefined : 3}>
                {JSON.stringify(message.value, null, 2)}
              </Code>
            </Box>
            
            {/* 展开的详细内容 */}
            <Collapse in={isExpanded}>
              <Box mt={4}>
                <Text fontSize="sm" fontWeight="semibold" mb={2}>Full Value:</Text>
                <JsonViewer data={message.value} />
                
                {/* Headers */}
                {message.headers && message.headers.length > 0 && (
                  <Box mt={4}>
                    <Text fontSize="sm" fontWeight="semibold" mb={2}>Headers:</Text>
                    {message.headers.map((header, i) => (
                      <Box key={i} fontSize="sm">
                        <Text as="span" fontWeight="semibold">{header.key}:</Text>{' '}
                        <Text as="span">{header.value}</Text>
                      </Box>
                    ))}
                  </Box>
                )}
              </Box>
            </Collapse>
          </Box>
        );
      })}
    </Box>
  );
}
```

### 2.3 Schema Registry 管理

```tsx
// src/pages/SchemaRegistry/SchemaList.tsx
import { useState } from 'react';
import { Box, Button, Input, Table, Thead, Tbody, Tr, Th, Td, Badge } from '@chakra-ui/react';
import { useSchemas } from '@/hooks/api/useSchema';
import { SchemaEditor } from '@/components/features/SchemaEditor';

export function SchemaList() {
  const [search, setSearch] = useState('');
  const [selectedSchema, setSelectedSchema] = useState<string | null>(null);
  
  const { data: schemas, isLoading, error } = useSchemas();
  
  const filteredSchemas = schemas?.filter(schema =>
    schema.subject.toLowerCase().includes(search.toLowerCase())
  );
  
  if (isLoading) return <Loading />;
  if (error) return <Error error={error} />;
  
  return (
    <Box>
      <Flex justify="space-between" mb={4}>
        <Input
          placeholder="Search schemas..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          maxW="400px"
        />
        <Button colorScheme="blue" onClick={() => setSelectedSchema('new')}>
          Create Schema
        </Button>
      </Flex>
      
      <Table>
        <Thead>
          <Tr>
            <Th>Subject</Th>
            <Th>Version</Th>
            <Th>Type</Th>
            <Th>Compatibility</Th>
            <Th>Actions</Th>
          </Tr>
        </Thead>
        <Tbody>
          {filteredSchemas?.map(schema => (
            <Tr key={schema.id}>
              <Td>{schema.subject}</Td>
              <Td>{schema.version}</Td>
              <Td>
                <Badge colorScheme={getTypeColor(schema.type)}>
                  {schema.type}
                </Badge>
              </Td>
              <Td>{schema.compatibility}</Td>
              <Td>
                <Button size="sm" onClick={() => setSelectedSchema(schema.subject)}>
                  View
                </Button>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
      
      {/* Schema Editor Modal */}
      {selectedSchema && (
        <SchemaEditor
          subject={selectedSchema}
          onClose={() => setSelectedSchema(null)}
        />
      )}
    </Box>
  );
}
```

## 3. 性能优化策略

### 3.1 代码分割

```tsx
// src/routes.tsx
import { lazy, Suspense } from 'react';
import { Routes, Route } from 'react-router-dom';

// 懒加载页面组件
const Dashboard = lazy(() => import('@/pages/Dashboard'));
const TopicList = lazy(() => import('@/pages/Topics/TopicList'));
const MessageViewer = lazy(() => import('@/pages/Messages/MessageViewer'));
const SchemaRegistry = lazy(() => import('@/pages/SchemaRegistry'));

export function AppRoutes() {
  return (
    <Suspense fallback={<Loading />}>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/topics" element={<TopicList />} />
        <Route path="/messages" element={<MessageViewer />} />
        <Route path="/schemas" element={<SchemaRegistry />} />
      </Routes>
    </Suspense>
  );
}
```

### 3.2 虚拟滚动

```tsx
// src/components/features/MessageList/VirtualMessageList.tsx
import { useVirtualizer } from '@tanstack/react-virtual';
import { useRef } from 'react';

export function VirtualMessageList({ messages }: { messages: Message[] }) {
  const parentRef = useRef<HTMLDivElement>(null);
  
  const virtualizer = useVirtualizer({
    count: messages.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 100, // 预估每个消息的高度
    overscan: 5, // 额外渲染的数量
  });
  
  return (
    <div ref={parentRef} style={{ height: '600px', overflow: 'auto' }}>
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualizer.getVirtualItems().map(virtualRow => (
          <div
            key={virtualRow.index}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`,
            }}
          >
            <MessageCard message={messages[virtualRow.index]} />
          </div>
        ))}
      </div>
    </div>
  );
}
```

### 3.3 缓存策略

```tsx
// src/services/api.ts
import axios from 'axios';
import { setupCache } from 'axios-cache-interceptor';

const instance = axios.create({
  baseURL: '/api/v1',
});

// 设置缓存
const cachedAxios = setupCache(instance, {
  ttl: 5 * 60 * 1000, // 5 分钟
  methods: ['get'],
  cachePredicate: {
    statusCheck: (status) => status >= 200 && status < 400,
  },
});

export default cachedAxios;
```

### 3.4 防抖和节流

```tsx
// src/hooks/utils/useDebounce.ts
import { useEffect, useState } from 'react';

export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);
  
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);
    
    return () => {
      clearTimeout(timer);
    };
  }, [value, delay]);
  
  return debouncedValue;
}

// 使用示例
function SearchComponent() {
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 500);
  
  // 只有在用户停止输入 500ms 后才会触发查询
  const { data } = useTopics({ search: debouncedSearch });
  
  return <Input value={search} onChange={(e) => setSearch(e.target.value)} />;
}
```

## 4. 测试策略

### 4.1 单元测试

```tsx
// src/components/common/Button/Button.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from './Button';

describe('Button', () => {
  it('renders with text', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });
  
  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click me</Button>);
    
    fireEvent.click(screen.getByText('Click me'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
  
  it('is disabled when disabled prop is true', () => {
    render(<Button disabled>Click me</Button>);
    expect(screen.getByText('Click me')).toBeDisabled();
  });
});
```

### 4.2 集成测试

```tsx
// src/pages/Topics/TopicList.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TopicList } from './TopicList';
import { server } from '@/tests/mocks/server';
import { rest } from 'msw';

describe('TopicList', () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  
  const wrapper = ({ children }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
  
  it('displays topics', async () => {
    render(<TopicList />, { wrapper });
    
    await waitFor(() => {
      expect(screen.getByText('test-topic')).toBeInTheDocument();
    });
  });
  
  it('handles error state', async () => {
    server.use(
      rest.get('/api/v1/topics', (req, res, ctx) => {
        return res(ctx.status(500));
      })
    );
    
    render(<TopicList />, { wrapper });
    
    await waitFor(() => {
      expect(screen.getByText(/error/i)).toBeInTheDocument();
    });
  });
});
```

### 4.3 E2E 测试

```typescript
// tests/e2e/topics.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Topic Management', () => {
  test('should list all topics', async ({ page }) => {
    await page.goto('/topics');
    
    await expect(page.locator('h1')).toContainText('Topics');
    await expect(page.locator('table')).toBeVisible();
  });
  
  test('should create a new topic', async ({ page }) => {
    await page.goto('/topics');
    
    await page.click('text=Create Topic');
    await page.fill('[name="name"]', 'new-topic');
    await page.fill('[name="partitions"]', '3');
    await page.fill('[name="replicationFactor"]', '2');
    await page.click('text=Create');
    
    await expect(page.locator('text=new-topic')).toBeVisible();
  });
  
  test('should view topic details', async ({ page }) => {
    await page.goto('/topics');
    
    await page.click('text=test-topic');
    
    await expect(page.locator('h2')).toContainText('test-topic');
    await expect(page.locator('text=Partitions')).toBeVisible();
  });
});
```

---

**文档版本**: v1.0  
**最后更新**: 2025-12-14  
**维护者**: Takhin Team
