# WebSocket API - Quick Reference

## Connection
```javascript
const ws = new WebSocket('ws://localhost:8080/api/monitoring/ws');
```

## Message Types

### Server → Client

**Metrics** (every 2s)
```json
{"type": "metrics", "data": {...}, "timestamp": 1704225600}
```

**Topic Created**
```json
{"type": "topic_created", "data": {"name": "my-topic", "partitions": 5}}
```

**Topic Deleted**
```json
{"type": "topic_deleted", "data": {"name": "my-topic"}}
```

**Group Updated**
```json
{"type": "group_updated", "data": {"groupId": "my-group", "state": "Stable", "members": 3}}
```

### Client → Server

**Subscribe**
```json
{"type": "subscribe", "data": "topic:my-topic", "timestamp": 1704225600}
```

**Ping**
```json
{"type": "ping", "timestamp": 1704225600}
```

## JavaScript Client

```javascript
class TakhinWS {
  constructor(url) {
    this.ws = new WebSocket(url);
    this.ws.onmessage = (e) => this.handleMessage(JSON.parse(e.data));
    this.ws.onclose = () => this.reconnect();
  }

  handleMessage(msg) {
    switch(msg.type) {
      case 'metrics': this.onMetrics(msg.data); break;
      case 'topic_created': this.onTopicCreated(msg.data); break;
      case 'topic_deleted': this.onTopicDeleted(msg.data); break;
      case 'group_updated': this.onGroupUpdated(msg.data); break;
    }
  }

  send(type, data) {
    this.ws.send(JSON.stringify({type, data, timestamp: Date.now()/1000}));
  }

  reconnect() {
    setTimeout(() => new TakhinWS(this.url), 1000);
  }
}
```

## React Hook

```typescript
function useWebSocket(url: string) {
  const [metrics, setMetrics] = useState(null);
  
  useEffect(() => {
    const ws = new WebSocket(url);
    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === 'metrics') setMetrics(msg.data);
    };
    return () => ws.close();
  }, [url]);
  
  return metrics;
}

// Usage
function Dashboard() {
  const metrics = useWebSocket('ws://localhost:8080/api/monitoring/ws');
  return <div>Topics: {metrics?.clusterHealth.totalTopics}</div>;
}
```

## Backend Events

```go
// Broadcast topic creation
server.BroadcastTopicCreated("my-topic", 5)

// Broadcast topic deletion
server.BroadcastTopicDeleted("my-topic")

// Broadcast group update
server.BroadcastGroupUpdated("my-group", "Stable", 3)
```

## Configuration

```go
const (
    writeWait      = 10 * time.Second   // Write timeout
    pongWait       = 60 * time.Second   // Pong timeout
    pingPeriod     = 54 * time.Second   // Ping interval
    maxMessageSize = 512 * 1024         // 512KB limit
)
```

## Testing

```bash
# Run WebSocket tests
go test ./pkg/console -v -run TestWebSocket

# All tests
go test ./pkg/console -v
```

## Metrics Data Structure

```typescript
interface MonitoringMetrics {
  throughput: {
    produceRate: number;      // msgs/sec
    fetchRate: number;        // msgs/sec
    produceBytes: number;     // bytes/sec
    fetchBytes: number;       // bytes/sec
  };
  latency: {
    produceP50: number;       // ms
    produceP95: number;       // ms
    produceP99: number;       // ms
    fetchP50: number;         // ms
    fetchP95: number;         // ms
    fetchP99: number;         // ms
  };
  topicStats: Array<{
    name: string;
    partitions: number;
    totalMessages: number;
    totalBytes: number;
    produceRate: number;
    fetchRate: number;
  }>;
  consumerLags: Array<{
    groupId: string;
    totalLag: number;
    topicLags: Array<{
      topic: string;
      totalLag: number;
      partitionLags: Array<{
        partition: number;
        currentOffset: number;
        logEndOffset: number;
        lag: number;
      }>;
    }>;
  }>;
  clusterHealth: {
    activeConnections: number;
    totalTopics: number;
    totalPartitions: number;
    totalConsumers: number;
    diskUsageBytes: number;
    memoryUsageBytes: number;
    goroutineCount: number;
  };
  timestamp: number;
}
```

## Files

```
backend/pkg/console/
├── websocket.go           # Implementation
├── websocket_test.go      # Tests
└── server.go              # Integration

docs/
└── websocket-api.md       # Full documentation
```

## Key Features

✅ Real-time metrics (2s interval)  
✅ Topic/Group event notifications  
✅ Automatic reconnection  
✅ Connection management  
✅ Ping/pong heartbeat  
✅ Client subscriptions  
✅ Multi-client broadcast  
✅ Authentication support  
✅ Comprehensive tests  

## See Also

- Full API Documentation: `docs/websocket-api.md`
- Implementation: `backend/pkg/console/websocket.go`
- Tests: `backend/pkg/console/websocket_test.go`
