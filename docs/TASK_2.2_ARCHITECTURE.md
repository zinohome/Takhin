# Takhin API Client Architecture

## File Structure
```
frontend/src/
├── api/
│   ├── auth.ts           # Authentication service (API key management)
│   ├── client.ts         # Backwards compatible exports
│   ├── errors.ts         # Error handling utilities
│   ├── index.ts          # Main exports
│   ├── takhinApi.ts      # Core API client implementation
│   ├── types.ts          # TypeScript type definitions
│   └── README.md         # API documentation
│
└── examples/
    ├── apiExamples.ts    # Usage examples (10 scenarios)
    └── hooks.ts          # React hooks for API integration
```

## Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     React Application                        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              React Components                         │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │  │
│  │  │ TopicList    │  │ MessageView  │  │ Dashboard  │  │  │
│  │  └──────┬───────┘  └──────┬───────┘  └─────┬──────┘  │  │
│  └─────────┼──────────────────┼─────────────────┼─────────┘  │
│            │                  │                 │            │
│  ┌─────────▼──────────────────▼─────────────────▼─────────┐  │
│  │              Custom React Hooks                        │  │
│  │   useTopics()  useMessages()  useHealth()             │  │
│  └─────────┬──────────────────┬─────────────────┬─────────┘  │
│            │                  │                 │            │
└────────────┼──────────────────┼─────────────────┼────────────┘
             │                  │                 │
┌────────────▼──────────────────▼─────────────────▼────────────┐
│                    TakhinApiClient                            │
│  ┌───────────────────────────────────────────────────────┐   │
│  │  Public API Methods                                   │   │
│  │  • listTopics()    • getMessages()                   │   │
│  │  • getTopic()      • produceMessage()                │   │
│  │  • createTopic()   • listConsumerGroups()            │   │
│  │  • deleteTopic()   • getConsumerGroup()              │   │
│  │  • getHealth()     • getReadiness()                  │   │
│  └───────────────────────────────────────────────────────┘   │
│                            │                                  │
│  ┌────────────────────────▼──────────────────────────────┐   │
│  │         Axios HTTP Client (Interceptors)              │   │
│  │  ┌──────────────┐              ┌──────────────┐      │   │
│  │  │   Request    │              │   Response   │      │   │
│  │  │ Interceptor  │              │ Interceptor  │      │   │
│  │  │ (Add Auth)   │              │ (Handle 401) │      │   │
│  │  └──────┬───────┘              └──────┬───────┘      │   │
│  └─────────┼──────────────────────────────┼─────────────┘   │
└────────────┼──────────────────────────────┼─────────────────┘
             │                              │
┌────────────▼──────────────────────────────▼─────────────────┐
│               Authentication Service                          │
│  • setApiKey()      • getApiKey()       • removeApiKey()    │
│  • isAuthenticated() • getAuthHeader()                      │
│  (localStorage: 'takhin_api_key')                           │
└───────────────────────────────────────────────────────────────┘
             │                              │
┌────────────▼──────────────────────────────▼─────────────────┐
│                 Error Handling Layer                          │
│  • TakhinApiError class                                      │
│  • HTTP status code mapping (401, 404, 400, 500, 503)      │
│  • User-friendly error messages                             │
└───────────────────────────────────────────────────────────────┘
             │
┌────────────▼──────────────────────────────────────────────────┐
│              Takhin Console REST API Server                   │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  Endpoints:                                             │  │
│  │  • GET    /api/health                                   │  │
│  │  • GET    /api/health/ready                             │  │
│  │  • GET    /api/health/live                              │  │
│  │  • GET    /api/topics                                   │  │
│  │  • GET    /api/topics/:topic                            │  │
│  │  • POST   /api/topics                                   │  │
│  │  • DELETE /api/topics/:topic                            │  │
│  │  • GET    /api/topics/:topic/messages                   │  │
│  │  • POST   /api/topics/:topic/messages                   │  │
│  │  • GET    /api/consumer-groups                          │  │
│  │  • GET    /api/consumer-groups/:group                   │  │
│  └─────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

## Data Flow

### Request Flow
```
Component → Hook → TakhinApiClient → Request Interceptor → Auth Service → HTTP Request → API Server
```

### Response Flow (Success)
```
API Server → HTTP Response → Response Interceptor → TakhinApiClient → Hook → Component
```

### Response Flow (Error)
```
API Server → Error Response → Response Interceptor → Error Handler → TakhinApiError → Hook → Component
```

### Authentication Flow
```
1. User Login
   └─> authService.setApiKey(key)
       └─> localStorage.setItem('takhin_api_key', key)

2. API Request
   └─> Request Interceptor
       └─> authService.getAuthHeader()
           └─> "Bearer {key}" added to headers

3. 401 Response
   └─> Response Interceptor
       └─> authService.removeApiKey()
           └─> localStorage.removeItem('takhin_api_key')
           └─> window.dispatchEvent('auth:unauthorized')
               └─> Application redirects to login
```

## Type Safety

```
┌──────────────────────────────────────────────────────────┐
│                 TypeScript Type Flow                      │
├──────────────────────────────────────────────────────────┤
│                                                           │
│  Backend (Go)           Frontend (TypeScript)            │
│  ═══════════            ════════════════════             │
│                                                           │
│  types.go               types.ts                         │
│  ─────────              ────────                         │
│  TopicSummary    ───►   TopicSummary                     │
│  {                      {                                │
│    Name string            name: string                   │
│    PartitionCount int     partitionCount: number         │
│    Partitions []...       partitions?: PartitionInfo[]   │
│  }                      }                                │
│                                                           │
│  Message         ───►   Message                          │
│  {                      {                                │
│    Partition int32        partition: number              │
│    Offset int64           offset: number                 │
│    Key string             key: string                    │
│    Value string           value: string                  │
│    Timestamp int64        timestamp: number              │
│  }                      }                                │
│                                                           │
└──────────────────────────────────────────────────────────┘
```

## Error Handling Strategy

```
┌────────────────────────────────────────────────────────────┐
│                  Error Handling Flow                        │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  API Server Error                                          │
│        │                                                    │
│        ▼                                                    │
│  HTTP Status Code                                          │
│  ┌───────┬─────────┬─────────┬─────────┬─────────┐       │
│  │  401  │   404   │   400   │   500   │   503   │       │
│  └───┬───┴────┬────┴────┬────┴────┬────┴────┬────┘       │
│      │        │         │         │         │             │
│      ▼        ▼         ▼         ▼         ▼             │
│  Unauthorized NotFound BadRequest ServerErr Unavailable   │
│      │        │         │         │         │             │
│      └────────┴─────────┴─────────┴─────────┘             │
│                       │                                    │
│                       ▼                                    │
│              TakhinApiError                                │
│              {                                             │
│                message: string                             │
│                statusCode?: number                         │
│                apiError?: { error: string }                │
│              }                                             │
│                       │                                    │
│                       ▼                                    │
│              Component Error State                         │
│              (Display to user)                             │
│                                                             │
└────────────────────────────────────────────────────────────┘
```

## Key Design Principles

1. **Singleton Pattern**: Default `takhinApi` instance for convenience
2. **Type Safety**: Full TypeScript coverage, no `any` types
3. **Error Handling**: Consistent error transformation and messaging
4. **Authentication**: Centralized auth service with auto-logout
5. **Interceptors**: Cross-cutting concerns (auth, error handling)
6. **Modularity**: Separated concerns (auth, errors, types, client)
7. **Extensibility**: Class-based design allows custom instances
8. **Developer Experience**: Comprehensive docs and examples

## Performance Considerations

- **Connection Pooling**: Axios reuses connections
- **Timeout**: 10 second default timeout prevents hanging
- **Error Recovery**: Clear error messages for debugging
- **Type Inference**: TypeScript provides compile-time checks
- **Tree Shaking**: Modular design allows unused code elimination

## Security Features

- **API Key Storage**: localStorage (client-side only)
- **Auth Header**: Bearer token format
- **Auto-logout**: 401 responses clear credentials
- **HTTPS**: Use in production for encrypted transport
- **No Credential Exposure**: API keys not logged or exposed

## Browser Compatibility

- Modern browsers with localStorage support
- ES6+ features (can be transpiled)
- Axios browser compatibility
- React 19.2.0+ required for hooks
