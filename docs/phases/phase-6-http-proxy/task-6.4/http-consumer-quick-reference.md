# HTTP Consumer API - Quick Reference

## 🚀 Quick Start

```bash
# 1. Subscribe
curl -X POST http://localhost:8080/api/consumers/subscribe \
  -d '{"group_id":"my-group","topics":["orders"]}'
# Returns: {"consumer_id": "uuid", "assignment": {...}}

# 2. Consume (long-polling)
curl -X POST http://localhost:8080/api/consumers/{consumer_id}/consume \
  -d '{"max_records":100,"timeout_ms":30000}'

# 3. Commit offsets
curl -X POST http://localhost:8080/api/consumers/{consumer_id}/commit \
  -d '{"offsets":{"orders":{"0":100}}}'

# 4. Close
curl -X DELETE http://localhost:8080/api/consumers/{consumer_id}
```

---

## 📡 API Endpoints

### Subscribe
```
POST /api/consumers/subscribe
```

**Request**:
```json
{
  "group_id": "my-group",
  "topics": ["orders", "events"],
  "auto_offset_reset": "earliest",  // "earliest" | "latest" (default)
  "session_timeout_ms": 30000       // default: 30000
}
```

**Response**:
```json
{
  "consumer_id": "550e8400-e29b-41d4-a716-446655440000",
  "group_id": "my-group",
  "topics": ["orders", "events"],
  "assignment": {
    "orders": [0, 1, 2],
    "events": [0, 1]
  }
}
```

---

### Consume (Long-Polling)
```
POST /api/consumers/{consumer_id}/consume
```

**Request**:
```json
{
  "max_records": 500,        // default: 500
  "timeout_ms": 30000,       // default: 30000
  "max_bytes_total": 1048576 // default: 1MB
}
```

**Response**:
```json
{
  "records": [
    {
      "topic": "orders",
      "partition": 0,
      "offset": 0,
      "timestamp": 1704537600000,
      "key": "b3JkZXItMTIz",      // base64
      "value": "eyJhbW91bnQiOjk5Ljk5fQ==", // base64
      "headers": {}
    }
  ],
  "timestamp": 1704537600500
}
```

---

### Commit Offsets
```
POST /api/consumers/{consumer_id}/commit
```

**Request**:
```json
{
  "offsets": {
    "orders": {
      "0": 100,
      "1": 150
    },
    "events": {
      "0": 50
    }
  }
}
```

**Response**:
```json
{
  "success": true,
  "message": "offsets committed"
}
```

---

### Seek to Offset
```
POST /api/consumers/{consumer_id}/seek
```

**Request**:
```json
{
  "topic": "orders",
  "partition": 0,
  "offset": 5
}
```

**Response**:
```json
{
  "success": true,
  "message": "offset updated"
}
```

---

### Get Position
```
GET /api/consumers/{consumer_id}/position
```

**Response**:
```json
{
  "offsets": {
    "orders": {
      "0": 100,
      "1": 150
    },
    "events": {
      "0": 50
    }
  }
}
```

---

### Manual Assignment
```
PUT /api/consumers/{consumer_id}/assignment
```

**Request**:
```json
{
  "topics": {
    "orders": [0, 1],
    "events": [0]
  }
}
```

**Response**:
```json
{
  "consumer_id": "550e8400-e29b-41d4-a716-446655440000",
  "group_id": "my-group",
  "topics": ["orders", "events"],
  "assignment": {
    "orders": [0, 1],
    "events": [0]
  }
}
```

---

### Unsubscribe
```
DELETE /api/consumers/{consumer_id}
```

**Response**:
```json
{
  "message": "consumer closed"
}
```

---

## 🐍 Python Client

```python
import requests

class TakhinConsumer:
    def __init__(self, base_url, group_id, topics):
        self.base_url = base_url
        self.consumer_id = None
        
        # Subscribe
        resp = requests.post(f"{base_url}/api/consumers/subscribe", json={
            "group_id": group_id,
            "topics": topics,
            "auto_offset_reset": "earliest"
        })
        self.consumer_id = resp.json()["consumer_id"]
    
    def poll(self, max_records=100, timeout_ms=30000):
        resp = requests.post(
            f"{self.base_url}/api/consumers/{self.consumer_id}/consume",
            json={"max_records": max_records, "timeout_ms": timeout_ms}
        )
        return resp.json()["records"]
    
    def commit(self, offsets):
        requests.post(
            f"{self.base_url}/api/consumers/{self.consumer_id}/commit",
            json={"offsets": offsets}
        )
    
    def close(self):
        requests.delete(f"{self.base_url}/api/consumers/{self.consumer_id}")

# Usage
consumer = TakhinConsumer("http://localhost:8080", "my-group", ["orders"])

while True:
    records = consumer.poll(max_records=50, timeout_ms=5000)
    
    offsets = {}
    for record in records:
        print(f"Offset {record['offset']}: {record['value']}")
        
        topic = record['topic']
        partition = record['partition']
        if topic not in offsets:
            offsets[topic] = {}
        offsets[topic][partition] = record['offset'] + 1
    
    if offsets:
        consumer.commit(offsets)

consumer.close()
```

---

## 📊 Configuration Defaults

| Parameter | Default | Description |
|-----------|---------|-------------|
| `session_timeout_ms` | 30000 | Consumer session timeout |
| `auto_offset_reset` | latest | earliest, latest |
| `max_records` | 500 | Max records per poll |
| `timeout_ms` | 30000 | Long poll timeout (30s) |
| `max_bytes_total` | 1048576 | Max bytes per poll (1MB) |

---

## 🎯 Best Practices

### 1. Commit Strategy
```python
# Commit after each batch (safe, more overhead)
records = consumer.poll()
process(records)
consumer.commit(offsets)

# Commit every N records (faster, some risk)
while True:
    records = consumer.poll()
    batch_count += len(records)
    if batch_count >= 1000:
        consumer.commit(offsets)
        batch_count = 0
```

### 2. Error Handling
```python
try:
    records = consumer.poll(timeout_ms=5000)
except requests.Timeout:
    # Expected on empty partitions
    continue
except requests.HTTPError as e:
    if e.response.status_code == 404:
        # Consumer expired, resubscribe
        consumer = TakhinConsumer(...)
    else:
        raise
```

### 3. Long-Polling
```python
# Good: Use long timeout
records = consumer.poll(timeout_ms=30000)  # Wait up to 30s

# Bad: Short polling
while True:
    records = consumer.poll(timeout_ms=100)  # Hammers server
    time.sleep(0.1)
```

### 4. Session Keepalive
```python
# Poll regularly to keep session alive
while True:
    records = consumer.poll(timeout_ms=25000)  # < session_timeout
    # Session timeout is 30s, poll every 25s
```

### 5. Graceful Shutdown
```python
import signal

def shutdown(signum, frame):
    consumer.close()
    sys.exit(0)

signal.signal(signal.SIGINT, shutdown)
signal.signal(signal.SIGTERM, shutdown)

# Consume loop
while True:
    records = consumer.poll()
    process(records)
```

---

## ⚡ Performance Tips

1. **Batch Size**: Increase `max_records` for throughput
   ```json
   {"max_records": 1000, "max_bytes_total": 10485760}
   ```

2. **Parallel Processing**: Use thread pool for record processing
   ```python
   from concurrent.futures import ThreadPoolExecutor
   
   with ThreadPoolExecutor(max_workers=10) as executor:
       executor.map(process_record, records)
   ```

3. **Commit Batching**: Commit every N records, not every poll
   ```python
   if total_processed % 1000 == 0:
       consumer.commit(offsets)
   ```

4. **Connection Pooling**: Reuse HTTP connections
   ```python
   session = requests.Session()
   session.post(...)  # Reuses connection
   ```

---

## 🔍 Monitoring

### Check Consumer Position
```bash
curl http://localhost:8080/api/consumers/$CONSUMER_ID/position | jq
```

### Check Consumer Group
```bash
curl http://localhost:8080/api/consumer-groups/my-group | jq
```

### Health Check
```bash
curl http://localhost:8080/api/health | jq
```

---

## 🐛 Troubleshooting

### Consumer Not Found (404)
- Session expired (no poll in >30s)
- Consumer ID incorrect
- **Fix**: Resubscribe

### No Records Returned
- No new messages in topic
- Already at end of log
- **Fix**: Check with producer, or seek to beginning

### Timeout on Poll
- Expected behavior with long-polling
- Returns empty records array
- **Fix**: Normal, continue polling

### Offset Commit Failed
- Consumer expired
- Partition not assigned
- **Fix**: Check assignment, resubscribe if needed

---

## 📚 See Also

- Full examples: `backend/pkg/console/CONSUMER_API_EXAMPLE.md`
- Completion summary: `TASK_6.4_HTTP_CONSUMER_COMPLETION.md`
- Producer API: `TASK_6.3_HTTP_PRODUCER_*.md`
- Consumer groups: `backend/pkg/coordinator/`

---

## 🎓 Cheat Sheet

```bash
# Subscribe
CONSUMER_ID=$(curl -s -X POST localhost:8080/api/consumers/subscribe \
  -d '{"group_id":"g","topics":["t"]}' | jq -r '.consumer_id')

# Consume loop
while true; do
  curl -s -X POST localhost:8080/api/consumers/$CONSUMER_ID/consume \
    -d '{"max_records":10}' | jq -c '.records[]'
done

# Commit
curl -X POST localhost:8080/api/consumers/$CONSUMER_ID/commit \
  -d '{"offsets":{"t":{"0":100}}}'

# Position
curl localhost:8080/api/consumers/$CONSUMER_ID/position | jq

# Seek
curl -X POST localhost:8080/api/consumers/$CONSUMER_ID/seek \
  -d '{"topic":"t","partition":0,"offset":0}'

# Close
curl -X DELETE localhost:8080/api/consumers/$CONSUMER_ID
```
