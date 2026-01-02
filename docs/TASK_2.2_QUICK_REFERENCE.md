# Quick Reference: Takhin API Client

## Installation
Already installed! Just import and use.

## Basic Import
```typescript
import { takhinApi, authService, TakhinApiError } from '@/api'
```

## Authentication

### Set API Key
```typescript
authService.setApiKey('your-api-key-here')
```

### Check Auth Status
```typescript
const isLoggedIn = authService.isAuthenticated()
```

### Logout
```typescript
authService.removeApiKey()
```

## Common Operations

### Topics

#### List All Topics
```typescript
const topics = await takhinApi.listTopics()
// Returns: TopicSummary[]
```

#### Get Topic Details
```typescript
const topic = await takhinApi.getTopic('my-topic')
// Returns: TopicDetail
```

#### Create Topic
```typescript
await takhinApi.createTopic({
  name: 'new-topic',
  partitions: 3
})
```

#### Delete Topic
```typescript
await takhinApi.deleteTopic('old-topic')
```

### Messages

#### Fetch Messages
```typescript
const messages = await takhinApi.getMessages('my-topic', {
  partition: 0,
  offset: 0,
  limit: 100
})
// Returns: Message[]
```

#### Produce Message
```typescript
const result = await takhinApi.produceMessage('my-topic', {
  partition: 0,
  key: 'my-key',
  value: 'my-value'
})
// Returns: { partition: number, offset: number }
```

### Consumer Groups

#### List Consumer Groups
```typescript
const groups = await takhinApi.listConsumerGroups()
// Returns: ConsumerGroupSummary[]
```

#### Get Consumer Group Details
```typescript
const group = await takhinApi.getConsumerGroup('my-group')
// Returns: ConsumerGroupDetail
```

### Health Checks

#### Full Health Check
```typescript
const health = await takhinApi.getHealth()
// Returns: HealthCheck
```

#### Readiness Check
```typescript
const ready = await takhinApi.getReadiness()
// Returns: { ready: boolean }
```

#### Liveness Check
```typescript
const alive = await takhinApi.getLiveness()
// Returns: { alive: boolean }
```

## Error Handling

### Try-Catch Pattern
```typescript
try {
  const topic = await takhinApi.getTopic('non-existent')
} catch (error) {
  if (error instanceof TakhinApiError) {
    console.error('Status:', error.statusCode)
    console.error('Message:', error.message)
    
    if (error.statusCode === 404) {
      console.log('Topic not found')
    }
  }
}
```

### Listen for Auth Errors
```typescript
window.addEventListener('auth:unauthorized', () => {
  console.log('Session expired')
  // Redirect to login
})
```

## React Hooks

### Import
```typescript
import { useTopics, useMessages, useHealth } from '@/examples/hooks'
```

### Use Topics Hook
```typescript
function MyComponent() {
  const { topics, loading, error, createTopic, deleteTopic } = useTopics()
  
  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>
  
  return (
    <div>
      {topics.map(t => <div key={t.name}>{t.name}</div>)}
      <button onClick={() => createTopic('new', 3)}>Create</button>
    </div>
  )
}
```

### Use Messages Hook
```typescript
function MessageViewer() {
  const { messages, loading, produceMessage } = useMessages(
    'my-topic',
    0,     // partition
    0,     // offset
    100    // limit
  )
  
  return (
    <div>
      {messages.map(m => <div key={m.offset}>{m.value}</div>)}
      <button onClick={() => produceMessage('key', 'value')}>
        Send Message
      </button>
    </div>
  )
}
```

### Use Health Hook with Polling
```typescript
function HealthMonitor() {
  const { health, loading } = useHealth(30000) // Poll every 30s
  
  return (
    <div>Status: {health?.status}</div>
  )
}
```

## TypeScript Types

### Import Types
```typescript
import type {
  TopicSummary,
  TopicDetail,
  Message,
  ConsumerGroupDetail,
  HealthCheck,
  GetMessagesParams,
  CreateTopicRequest,
  ProduceMessageRequest
} from '@/api'
```

### Use in Components
```typescript
function TopicCard({ topic }: { topic: TopicSummary }) {
  return <div>{topic.name} ({topic.partitionCount} partitions)</div>
}
```

## Configuration

### Custom Base URL
```typescript
import { TakhinApiClient } from '@/api'

const customApi = new TakhinApiClient('https://api.example.com/v1', 30000)
```

### Environment Variable
```bash
# .env
VITE_API_BASE_URL=/api
```

```typescript
const api = new TakhinApiClient(import.meta.env.VITE_API_BASE_URL)
```

## Best Practices

1. **Always handle errors** with try-catch
2. **Use the singleton** `takhinApi` for most cases
3. **Store API keys** using `authService`
4. **Type your data** with imported types
5. **Use React hooks** for components
6. **Listen for auth events** to handle logout

## Cheat Sheet

| Operation | Method | Returns |
|-----------|--------|---------|
| List topics | `listTopics()` | `TopicSummary[]` |
| Get topic | `getTopic(name)` | `TopicDetail` |
| Create topic | `createTopic({name, partitions})` | `CreateTopicResponse` |
| Delete topic | `deleteTopic(name)` | `{message: string}` |
| Get messages | `getMessages(topic, params)` | `Message[]` |
| Produce message | `produceMessage(topic, msg)` | `{partition, offset}` |
| List groups | `listConsumerGroups()` | `ConsumerGroupSummary[]` |
| Get group | `getConsumerGroup(id)` | `ConsumerGroupDetail` |
| Health check | `getHealth()` | `HealthCheck` |
| Readiness | `getReadiness()` | `{ready: boolean}` |
| Liveness | `getLiveness()` | `{alive: boolean}` |

## Common HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| 200 | Success | Continue |
| 201 | Created | Resource created |
| 204 | No Content | Delete successful |
| 400 | Bad Request | Check input |
| 401 | Unauthorized | Login again |
| 404 | Not Found | Resource doesn't exist |
| 500 | Server Error | Check server logs |
| 503 | Unavailable | Service not ready |

## Files Location

- **API Client**: `frontend/src/api/`
- **Examples**: `frontend/src/examples/`
- **Documentation**: `frontend/src/api/README.md`

## Get Help

1. Read `frontend/src/api/README.md` for detailed docs
2. Check `frontend/src/examples/apiExamples.ts` for patterns
3. Use `frontend/src/examples/hooks.ts` for React integration
4. TypeScript will guide you with autocomplete!
