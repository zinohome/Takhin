# HTTP Consumer API Example

This example demonstrates how to use the HTTP Consumer API for consuming messages from Takhin.

## Complete Consumer Workflow

```bash
# 1. Subscribe to topics
curl -X POST http://localhost:8080/api/consumers/subscribe \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "my-consumer-group",
    "topics": ["orders", "events"],
    "auto_offset_reset": "earliest",
    "session_timeout_ms": 30000
  }'

# Response:
# {
#   "consumer_id": "550e8400-e29b-41d4-a716-446655440000",
#   "group_id": "my-consumer-group",
#   "topics": ["orders", "events"],
#   "assignment": {
#     "orders": [0, 1, 2],
#     "events": [0, 1]
#   }
# }

# Save consumer_id for subsequent requests
CONSUMER_ID="550e8400-e29b-41d4-a716-446655440000"

# 2. Poll for messages (long polling)
curl -X POST http://localhost:8080/api/consumers/$CONSUMER_ID/consume \
  -H "Content-Type: application/json" \
  -d '{
    "max_records": 100,
    "timeout_ms": 30000,
    "max_bytes_total": 1048576
  }'

# Response:
# {
#   "records": [
#     {
#       "topic": "orders",
#       "partition": 0,
#       "offset": 0,
#       "timestamp": 1704537600000,
#       "key": "order-123",
#       "value": "{\"order_id\": \"123\", \"amount\": 99.99}",
#       "headers": {
#         "correlation_id": "abc-123"
#       }
#     }
#   ],
#   "timestamp": 1704537600500
# }

# 3. Commit offsets
curl -X POST http://localhost:8080/api/consumers/$CONSUMER_ID/commit \
  -H "Content-Type: application/json" \
  -d '{
    "offsets": {
      "orders": {
        "0": 10,
        "1": 15,
        "2": 8
      },
      "events": {
        "0": 5,
        "1": 12
      }
    }
  }'

# 4. Get current position
curl http://localhost:8080/api/consumers/$CONSUMER_ID/position

# Response:
# {
#   "offsets": {
#     "orders": {
#       "0": 10,
#       "1": 15,
#       "2": 8
#     },
#     "events": {
#       "0": 5,
#       "1": 12
#     }
#   }
# }

# 5. Seek to specific offset
curl -X POST http://localhost:8080/api/consumers/$CONSUMER_ID/seek \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "orders",
    "partition": 0,
    "offset": 5
  }'

# 6. Manual partition assignment (optional)
curl -X PUT http://localhost:8080/api/consumers/$CONSUMER_ID/assignment \
  -H "Content-Type: application/json" \
  -d '{
    "topics": {
      "orders": [0, 1],
      "events": [0]
    }
  }'

# 7. Unsubscribe and close consumer
curl -X DELETE http://localhost:8080/api/consumers/$CONSUMER_ID
```

## Python Consumer Example

```python
import requests
import json
import time

class TakhinHTTPConsumer:
    def __init__(self, base_url, group_id, topics, auto_offset_reset="latest"):
        self.base_url = base_url
        self.group_id = group_id
        self.topics = topics
        self.consumer_id = None
        self.auto_offset_reset = auto_offset_reset
        
    def subscribe(self):
        """Subscribe to topics"""
        response = requests.post(
            f"{self.base_url}/api/consumers/subscribe",
            json={
                "group_id": self.group_id,
                "topics": self.topics,
                "auto_offset_reset": self.auto_offset_reset,
                "session_timeout_ms": 30000
            }
        )
        response.raise_for_status()
        data = response.json()
        self.consumer_id = data["consumer_id"]
        print(f"Subscribed with consumer ID: {self.consumer_id}")
        print(f"Assigned partitions: {data['assignment']}")
        return data
        
    def poll(self, max_records=100, timeout_ms=30000, max_bytes=1048576):
        """Poll for messages"""
        if not self.consumer_id:
            raise Exception("Not subscribed")
            
        response = requests.post(
            f"{self.base_url}/api/consumers/{self.consumer_id}/consume",
            json={
                "max_records": max_records,
                "timeout_ms": timeout_ms,
                "max_bytes_total": max_bytes
            }
        )
        response.raise_for_status()
        return response.json()
        
    def commit(self, offsets):
        """Commit offsets"""
        if not self.consumer_id:
            raise Exception("Not subscribed")
            
        response = requests.post(
            f"{self.base_url}/api/consumers/{self.consumer_id}/commit",
            json={"offsets": offsets}
        )
        response.raise_for_status()
        return response.json()
        
    def seek(self, topic, partition, offset):
        """Seek to offset"""
        if not self.consumer_id:
            raise Exception("Not subscribed")
            
        response = requests.post(
            f"{self.base_url}/api/consumers/{self.consumer_id}/seek",
            json={
                "topic": topic,
                "partition": partition,
                "offset": offset
            }
        )
        response.raise_for_status()
        return response.json()
        
    def position(self):
        """Get current positions"""
        if not self.consumer_id:
            raise Exception("Not subscribed")
            
        response = requests.get(
            f"{self.base_url}/api/consumers/{self.consumer_id}/position"
        )
        response.raise_for_status()
        return response.json()
        
    def close(self):
        """Close consumer"""
        if self.consumer_id:
            response = requests.delete(
                f"{self.base_url}/api/consumers/{self.consumer_id}"
            )
            response.raise_for_status()
            print("Consumer closed")
            self.consumer_id = None

# Usage example
def main():
    consumer = TakhinHTTPConsumer(
        base_url="http://localhost:8080",
        group_id="my-python-group",
        topics=["orders", "events"],
        auto_offset_reset="earliest"
    )
    
    try:
        # Subscribe
        consumer.subscribe()
        
        # Consume loop
        while True:
            # Poll for messages
            result = consumer.poll(max_records=50, timeout_ms=5000)
            records = result["records"]
            
            if records:
                print(f"Received {len(records)} records")
                
                # Process messages
                offsets_to_commit = {}
                for record in records:
                    print(f"Topic: {record['topic']}, "
                          f"Partition: {record['partition']}, "
                          f"Offset: {record['offset']}, "
                          f"Value: {record['value']}")
                    
                    # Track offsets for commit
                    topic = record['topic']
                    partition = record['partition']
                    offset = record['offset'] + 1  # Commit next offset
                    
                    if topic not in offsets_to_commit:
                        offsets_to_commit[topic] = {}
                    offsets_to_commit[topic][partition] = offset
                
                # Commit offsets
                if offsets_to_commit:
                    consumer.commit(offsets_to_commit)
                    print("Committed offsets")
            else:
                print("No new messages")
                
            time.sleep(1)
            
    except KeyboardInterrupt:
        print("\nShutting down...")
    finally:
        consumer.close()

if __name__ == "__main__":
    main()
```

## Node.js Consumer Example

```javascript
const axios = require('axios');

class TakhinHTTPConsumer {
  constructor(baseUrl, groupId, topics, autoOffsetReset = 'latest') {
    this.baseUrl = baseUrl;
    this.groupId = groupId;
    this.topics = topics;
    this.autoOffsetReset = autoOffsetReset;
    this.consumerId = null;
  }

  async subscribe() {
    const response = await axios.post(`${this.baseUrl}/api/consumers/subscribe`, {
      group_id: this.groupId,
      topics: this.topics,
      auto_offset_reset: this.autoOffsetReset,
      session_timeout_ms: 30000
    });
    
    this.consumerId = response.data.consumer_id;
    console.log(`Subscribed with consumer ID: ${this.consumerId}`);
    console.log(`Assigned partitions:`, response.data.assignment);
    return response.data;
  }

  async poll(maxRecords = 100, timeoutMs = 30000, maxBytes = 1048576) {
    if (!this.consumerId) {
      throw new Error('Not subscribed');
    }

    const response = await axios.post(
      `${this.baseUrl}/api/consumers/${this.consumerId}/consume`,
      {
        max_records: maxRecords,
        timeout_ms: timeoutMs,
        max_bytes_total: maxBytes
      }
    );
    
    return response.data;
  }

  async commit(offsets) {
    if (!this.consumerId) {
      throw new Error('Not subscribed');
    }

    const response = await axios.post(
      `${this.baseUrl}/api/consumers/${this.consumerId}/commit`,
      { offsets }
    );
    
    return response.data;
  }

  async seek(topic, partition, offset) {
    if (!this.consumerId) {
      throw new Error('Not subscribed');
    }

    const response = await axios.post(
      `${this.baseUrl}/api/consumers/${this.consumerId}/seek`,
      { topic, partition, offset }
    );
    
    return response.data;
  }

  async position() {
    if (!this.consumerId) {
      throw new Error('Not subscribed');
    }

    const response = await axios.get(
      `${this.baseUrl}/api/consumers/${this.consumerId}/position`
    );
    
    return response.data;
  }

  async close() {
    if (this.consumerId) {
      await axios.delete(`${this.baseUrl}/api/consumers/${this.consumerId}`);
      console.log('Consumer closed');
      this.consumerId = null;
    }
  }
}

// Usage
async function main() {
  const consumer = new TakhinHTTPConsumer(
    'http://localhost:8080',
    'my-nodejs-group',
    ['orders', 'events'],
    'earliest'
  );

  try {
    await consumer.subscribe();

    while (true) {
      const result = await consumer.poll(50, 5000);
      const { records } = result;

      if (records.length > 0) {
        console.log(`Received ${records.length} records`);

        const offsetsToCommit = {};
        
        for (const record of records) {
          console.log(`Topic: ${record.topic}, Partition: ${record.partition}, ` +
                     `Offset: ${record.offset}, Value: ${record.value}`);

          if (!offsetsToCommit[record.topic]) {
            offsetsToCommit[record.topic] = {};
          }
          offsetsToCommit[record.topic][record.partition] = record.offset + 1;
        }

        if (Object.keys(offsetsToCommit).length > 0) {
          await consumer.commit(offsetsToCommit);
          console.log('Committed offsets');
        }
      } else {
        console.log('No new messages');
      }

      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  } catch (error) {
    console.error('Error:', error.message);
  } finally {
    await consumer.close();
  }
}

main();
```

## Key Features

### 1. **Long Polling**
- Consumers can wait up to `timeout_ms` for new messages
- Reduces polling overhead and latency
- Returns immediately when messages are available

### 2. **Offset Management**
- Auto-commit: Offsets updated automatically on poll
- Manual commit: Explicit offset commits via `/commit`
- Seek: Jump to specific offsets via `/seek`

### 3. **Consumer Groups**
- Automatic partition assignment
- Group coordination via Takhin coordinator
- Session timeout monitoring

### 4. **Flexible Consumption**
- Control batch size with `max_records`
- Limit memory with `max_bytes_total`
- Configurable poll timeout

## Best Practices

1. **Commit Frequency**: Balance between durability and performance
   - More frequent commits = less reprocessing on failure
   - Less frequent commits = better throughput

2. **Batch Processing**: Process records in batches for efficiency
   ```python
   records = consumer.poll(max_records=100)
   # Process all records
   # Then commit once
   consumer.commit(offsets)
   ```

3. **Error Handling**: Always close consumer on exit
   ```python
   try:
       # Consume loop
   finally:
       consumer.close()
   ```

4. **Heartbeat**: Keep polling regularly to maintain session
   - Session timeout default: 30 seconds
   - Poll at least every 25 seconds

5. **Rebalancing**: Handle assignment changes gracefully
   - New consumers joining group trigger rebalance
   - Check assignment after rebalance

## Configuration Options

| Parameter | Default | Description |
|-----------|---------|-------------|
| `session_timeout_ms` | 30000 | Consumer session timeout |
| `auto_offset_reset` | latest | earliest, latest |
| `max_records` | 500 | Max records per poll |
| `timeout_ms` | 30000 | Long poll timeout |
| `max_bytes_total` | 1048576 | Max bytes per poll (1MB) |
