# Takhin Console API Client

A fully-typed TypeScript client for the Takhin Console REST API.

## Installation

The API client is already included in the frontend project. Import from `@/api`:

```typescript
import { takhinApi, authService, TakhinApiError } from '@/api'
```

## Quick Start

### Authentication

```typescript
import { authService } from '@/api'

// Set API key (from user input or environment)
authService.setApiKey('your-api-key-here')

// Check if authenticated
if (authService.isAuthenticated()) {
  console.log('User is authenticated')
}

// Remove API key (logout)
authService.removeApiKey()
```

### Basic Usage

```typescript
import { takhinApi } from '@/api'

// List all topics
const topics = await takhinApi.listTopics()

// Get topic details
const topic = await takhinApi.getTopic('my-topic')

// Create a new topic
await takhinApi.createTopic({
  name: 'new-topic',
  partitions: 3
})
```

## API Reference

### Health Endpoints

#### Get Health Status
```typescript
const health = await takhinApi.getHealth()
// Returns: HealthCheck
// {
//   status: 'healthy' | 'degraded' | 'unhealthy',
//   version: '1.0.0',
//   timestamp: 1704240000,
//   components: {
//     storage: { status: 'healthy', message?: string },
//     coordinator: { status: 'healthy', message?: string }
//   }
// }
```

#### Readiness Check
```typescript
const readiness = await takhinApi.getReadiness()
// Returns: { ready: boolean }
```

#### Liveness Check
```typescript
const liveness = await takhinApi.getLiveness()
// Returns: { alive: boolean }
```

### Topic Endpoints

#### List Topics
```typescript
const topics = await takhinApi.listTopics()
// Returns: TopicSummary[]
// [{
//   name: 'my-topic',
//   partitionCount: 3,
//   partitions: [
//     { id: 0, highWaterMark: 1000 },
//     { id: 1, highWaterMark: 1500 },
//     { id: 2, highWaterMark: 800 }
//   ]
// }]
```

#### Get Topic Details
```typescript
const topic = await takhinApi.getTopic('my-topic')
// Returns: TopicDetail
// {
//   name: 'my-topic',
//   partitionCount: 3,
//   partitions: [...]
// }
```

#### Create Topic
```typescript
const result = await takhinApi.createTopic({
  name: 'new-topic',
  partitions: 5
})
// Returns: CreateTopicResponse
// { name: 'new-topic', partitions: '5' }
```

#### Delete Topic
```typescript
const result = await takhinApi.deleteTopic('old-topic')
// Returns: { message: 'topic deleted successfully' }
```

### Message Endpoints

#### Get Messages
```typescript
const messages = await takhinApi.getMessages('my-topic', {
  partition: 0,
  offset: 100,
  limit: 50  // Optional, default 100
})
// Returns: Message[]
// [{
//   partition: 0,
//   offset: 100,
//   key: 'message-key',
//   value: 'message-value',
//   timestamp: 1704240000
// }]
```

#### Produce Message
```typescript
const result = await takhinApi.produceMessage('my-topic', {
  partition: 0,
  key: 'my-key',
  value: 'my-value'
})
// Returns: ProduceMessageResponse
// { partition: 0, offset: 1234 }
```

### Consumer Group Endpoints

#### List Consumer Groups
```typescript
const groups = await takhinApi.listConsumerGroups()
// Returns: ConsumerGroupSummary[]
// [{
//   groupId: 'my-consumer-group',
//   state: 'Stable',
//   members: 3
// }]
```

#### Get Consumer Group Details
```typescript
const group = await takhinApi.getConsumerGroup('my-consumer-group')
// Returns: ConsumerGroupDetail
// {
//   groupId: 'my-consumer-group',
//   state: 'Stable',
//   protocolType: 'consumer',
//   protocol: 'range',
//   members: [
//     {
//       memberId: 'consumer-1',
//       clientId: 'my-client',
//       clientHost: '/192.168.1.100',
//       partitions: [0, 1, 2]
//     }
//   ],
//   offsetCommits: [
//     {
//       topic: 'my-topic',
//       partition: 0,
//       offset: 1000,
//       metadata: ''
//     }
//   ]
// }
```

## Error Handling

### Using Try-Catch

```typescript
import { takhinApi, TakhinApiError } from '@/api'

try {
  const topic = await takhinApi.getTopic('non-existent')
} catch (error) {
  if (error instanceof TakhinApiError) {
    console.error('Status:', error.statusCode)
    console.error('Message:', error.message)
    console.error('API Error:', error.apiError?.error)
    
    // Handle specific error codes
    if (error.statusCode === 404) {
      console.log('Topic not found')
    }
  }
}
```

### Error Types

- **401 Unauthorized**: Invalid or missing API key
- **404 Not Found**: Resource doesn't exist
- **400 Bad Request**: Invalid input parameters
- **500 Internal Server Error**: Server-side error
- **503 Service Unavailable**: Server not ready

### Unauthorized Event

The client emits a custom event when receiving 401 responses:

```typescript
window.addEventListener('auth:unauthorized', () => {
  console.log('Session expired, redirecting to login')
  // Handle unauthorized access
})
```

## React Integration Examples

### Using with React Query

```typescript
import { useQuery, useMutation } from '@tanstack/react-query'
import { takhinApi } from '@/api'

// Query topics
function useTopics() {
  return useQuery({
    queryKey: ['topics'],
    queryFn: () => takhinApi.listTopics()
  })
}

// Create topic mutation
function useCreateTopic() {
  return useMutation({
    mutationFn: (data) => takhinApi.createTopic(data)
  })
}

// Usage in component
function TopicList() {
  const { data, isLoading, error } = useTopics()
  const createTopic = useCreateTopic()
  
  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  
  return (
    <div>
      {data?.map(topic => (
        <div key={topic.name}>{topic.name}</div>
      ))}
      <button onClick={() => createTopic.mutate({ name: 'test', partitions: 3 })}>
        Create Topic
      </button>
    </div>
  )
}
```

### Using with useState/useEffect

```typescript
import { useState, useEffect } from 'react'
import { takhinApi, TopicSummary, TakhinApiError } from '@/api'

function TopicList() {
  const [topics, setTopics] = useState<TopicSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  useEffect(() => {
    const fetchTopics = async () => {
      try {
        setLoading(true)
        const data = await takhinApi.listTopics()
        setTopics(data)
      } catch (err) {
        if (err instanceof TakhinApiError) {
          setError(err.message)
        }
      } finally {
        setLoading(false)
      }
    }
    
    fetchTopics()
  }, [])
  
  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>
  
  return (
    <ul>
      {topics.map(topic => (
        <li key={topic.name}>{topic.name}</li>
      ))}
    </ul>
  )
}
```

## Advanced Usage

### Custom Base URL and Timeout

```typescript
import { TakhinApiClient } from '@/api'

const customApi = new TakhinApiClient('https://api.example.com/v1', 30000)
```

### Custom Request

For endpoints not yet wrapped:

```typescript
const data = await takhinApi.request({
  method: 'GET',
  url: '/custom-endpoint',
  params: { key: 'value' }
})
```

## TypeScript Support

All API methods are fully typed with TypeScript. Import types as needed:

```typescript
import type {
  TopicSummary,
  TopicDetail,
  Message,
  ConsumerGroupDetail,
  HealthCheck
} from '@/api'
```

## Configuration

### Environment Variables

Set the API base URL in your `.env` file:

```bash
VITE_API_BASE_URL=/api
```

Then use it in the client:

```typescript
const baseURL = import.meta.env.VITE_API_BASE_URL || '/api'
const api = new TakhinApiClient(baseURL)
```

## Best Practices

1. **Use the singleton instance** (`takhinApi`) for most cases
2. **Handle errors** with try-catch blocks
3. **Store API keys securely** using the `authService`
4. **Type your responses** with the provided TypeScript types
5. **Use React Query** for caching and state management
6. **Listen for auth events** to handle session expiration

## Testing

### Mock Example

```typescript
import { vi } from 'vitest'
import { takhinApi } from '@/api'

// Mock the API client
vi.mock('@/api', () => ({
  takhinApi: {
    listTopics: vi.fn().mockResolvedValue([
      { name: 'test-topic', partitionCount: 3 }
    ])
  }
}))

// Use in tests
test('renders topics', async () => {
  const topics = await takhinApi.listTopics()
  expect(topics).toHaveLength(1)
})
```
