# WebSocket Real-Time Updates API

## Overview

The Takhin Console provides a WebSocket API for real-time updates on cluster metrics, topic changes, and consumer group events. This enables frontend applications to receive live updates without polling.

## Connection

### Endpoint

```
ws://localhost:8080/api/monitoring/ws
```

### Authentication

If authentication is enabled, pass the API key in the WebSocket upgrade request:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/monitoring/ws', {
  headers: {
    'Authorization': 'Bearer your-api-key-here'
  }
});
```

## Message Types

### Server → Client Messages

#### 1. Metrics Update (`metrics`)

Sent every 2 seconds with current cluster metrics.

```json
{
  "type": "metrics",
  "timestamp": 1704225600,
  "data": {
    "throughput": {
      "produceRate": 1500.5,
      "fetchRate": 1200.3,
      "produceBytes": 1048576,
      "fetchBytes": 819200
    },
    "latency": {
      "produceP50": 5.2,
      "produceP95": 12.8,
      "produceP99": 25.6,
      "fetchP50": 3.1,
      "fetchP95": 8.5,
      "fetchP99": 15.2
    },
    "topicStats": [
      {
        "name": "test-topic",
        "partitions": 3,
        "totalMessages": 10000,
        "totalBytes": 5242880,
        "produceRate": 500.0,
        "fetchRate": 400.0
      }
    ],
    "consumerLags": [
      {
        "groupId": "consumer-group-1",
        "totalLag": 150,
        "topicLags": [
          {
            "topic": "test-topic",
            "totalLag": 150,
            "partitionLags": [
              {
                "partition": 0,
                "currentOffset": 8500,
                "logEndOffset": 8550,
                "lag": 50
              }
            ]
          }
        ]
      }
    ],
    "clusterHealth": {
      "activeConnections": 5,
      "totalTopics": 10,
      "totalPartitions": 30,
      "totalConsumers": 3,
      "diskUsageBytes": 10485760,
      "memoryUsageBytes": 52428800,
      "goroutineCount": 125
    }
  }
}
```

#### 2. Topic Created (`topic_created`)

Sent when a new topic is created.

```json
{
  "type": "topic_created",
  "timestamp": 1704225600,
  "data": {
    "name": "new-topic",
    "partitions": 5
  }
}
```

#### 3. Topic Deleted (`topic_deleted`)

Sent when a topic is deleted.

```json
{
  "type": "topic_deleted",
  "timestamp": 1704225600,
  "data": {
    "name": "old-topic"
  }
}
```

#### 4. Group Updated (`group_updated`)

Sent when a consumer group state changes.

```json
{
  "type": "group_updated",
  "timestamp": 1704225600,
  "data": {
    "groupId": "consumer-group-1",
    "state": "Stable",
    "members": 3
  }
}
```

#### 5. Pong (`pong`)

Response to client ping.

```json
{
  "type": "pong",
  "timestamp": 1704225600
}
```

### Client → Server Messages

#### 1. Subscribe (`subscribe`)

Subscribe to specific topic or group updates.

```json
{
  "type": "subscribe",
  "data": "topic:my-topic",
  "timestamp": 1704225600
}
```

#### 2. Unsubscribe (`unsubscribe`)

Unsubscribe from updates.

```json
{
  "type": "unsubscribe",
  "data": "topic:my-topic",
  "timestamp": 1704225600
}
```

#### 3. Ping (`ping`)

Keep-alive message.

```json
{
  "type": "ping",
  "timestamp": 1704225600
}
```

## JavaScript/TypeScript Example

### Basic Connection

```typescript
class TakhinWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  connect(url: string, apiKey?: string) {
    const headers: any = {};
    if (apiKey) {
      headers.Authorization = `Bearer ${apiKey}`;
    }

    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.reconnect(url, apiKey);
    };
  }

  private handleMessage(message: any) {
    switch (message.type) {
      case 'metrics':
        this.onMetrics(message.data);
        break;
      case 'topic_created':
        this.onTopicCreated(message.data);
        break;
      case 'topic_deleted':
        this.onTopicDeleted(message.data);
        break;
      case 'group_updated':
        this.onGroupUpdated(message.data);
        break;
      case 'pong':
        console.log('Received pong');
        break;
    }
  }

  private reconnect(url: string, apiKey?: string) {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    
    setTimeout(() => {
      this.connect(url, apiKey);
    }, delay);
  }

  send(message: any) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        ...message,
        timestamp: Math.floor(Date.now() / 1000)
      }));
    }
  }

  subscribe(topic: string) {
    this.send({ type: 'subscribe', data: `topic:${topic}` });
  }

  unsubscribe(topic: string) {
    this.send({ type: 'unsubscribe', data: `topic:${topic}` });
  }

  ping() {
    this.send({ type: 'ping' });
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  // Override these methods to handle events
  onMetrics(data: any) {}
  onTopicCreated(data: any) {}
  onTopicDeleted(data: any) {}
  onGroupUpdated(data: any) {}
}
```

### React Hook Example

```typescript
import { useEffect, useState } from 'react';

interface MetricsData {
  throughput: any;
  latency: any;
  topicStats: any[];
  consumerLags: any[];
  clusterHealth: any;
}

export function useWebSocket(url: string, apiKey?: string) {
  const [metrics, setMetrics] = useState<MetricsData | null>(null);
  const [connected, setConnected] = useState(false);
  const [ws, setWs] = useState<WebSocket | null>(null);

  useEffect(() => {
    const socket = new WebSocket(url);

    socket.onopen = () => {
      console.log('WebSocket connected');
      setConnected(true);
    };

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      if (message.type === 'metrics') {
        setMetrics(message.data);
      }
    };

    socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    socket.onclose = () => {
      console.log('WebSocket disconnected');
      setConnected(false);
    };

    setWs(socket);

    return () => {
      socket.close();
    };
  }, [url, apiKey]);

  return { metrics, connected, ws };
}

// Usage in component
function Dashboard() {
  const { metrics, connected } = useWebSocket('ws://localhost:8080/api/monitoring/ws');

  if (!connected) {
    return <div>Connecting...</div>;
  }

  return (
    <div>
      <h1>Cluster Metrics</h1>
      {metrics && (
        <>
          <div>Active Connections: {metrics.clusterHealth.activeConnections}</div>
          <div>Total Topics: {metrics.clusterHealth.totalTopics}</div>
          <div>Produce Rate: {metrics.throughput.produceRate}/s</div>
        </>
      )}
    </div>
  );
}
```

### Vue.js Composition API Example

```typescript
import { ref, onMounted, onUnmounted } from 'vue';

export function useWebSocket(url: string, apiKey?: string) {
  const metrics = ref(null);
  const connected = ref(false);
  let ws: WebSocket | null = null;

  const connect = () => {
    ws = new WebSocket(url);

    ws.onopen = () => {
      console.log('WebSocket connected');
      connected.value = true;
    };

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      if (message.type === 'metrics') {
        metrics.value = message.data;
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      connected.value = false;
    };
  };

  const disconnect = () => {
    if (ws) {
      ws.close();
      ws = null;
    }
  };

  onMounted(() => {
    connect();
  });

  onUnmounted(() => {
    disconnect();
  });

  return { metrics, connected, disconnect };
}
```

## Connection Management

### Heartbeat/Ping

To keep the connection alive and detect disconnections:

```javascript
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'ping', timestamp: Date.now() / 1000 }));
  }
}, 30000); // Every 30 seconds
```

### Automatic Reconnection

Implement exponential backoff for reconnection:

```javascript
let reconnectAttempt = 0;
const maxReconnectAttempts = 5;

function reconnect() {
  if (reconnectAttempt >= maxReconnectAttempts) {
    console.error('Max reconnection attempts reached');
    return;
  }

  const delay = Math.min(1000 * Math.pow(2, reconnectAttempt), 30000);
  reconnectAttempt++;

  setTimeout(() => {
    connectWebSocket();
  }, delay);
}
```

## Performance Considerations

1. **Message Frequency**: Metrics are sent every 2 seconds by default
2. **Message Size**: Expect 1-10KB per metrics message depending on cluster size
3. **Connection Limit**: No hard limit, but consider server resources
4. **Buffering**: Client should buffer messages during temporary disconnections

## Error Handling

The server will close the connection in these cases:
- Client authentication fails
- Client sends malformed messages
- Server shutdown
- Network errors

Always implement reconnection logic in your client.

## Security

- Use WSS (WebSocket Secure) in production
- Pass API keys via query parameters or headers
- Validate all incoming messages
- Implement rate limiting if needed
