# Takhin Console Usage Guide

The Takhin Console is a web-based management interface for the Takhin streaming platform. It provides real-time monitoring, topic management, consumer group tracking, and message browsing capabilities.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Dashboard Overview](#dashboard-overview)
3. [Managing Topics](#managing-topics)
4. [Browsing Messages](#browsing-messages)
5. [Monitoring Consumer Groups](#monitoring-consumer-groups)
6. [Broker Information](#broker-information)
7. [Configuration Management](#configuration-management)
8. [API Authentication](#api-authentication)
9. [Troubleshooting](#troubleshooting)

---

## Getting Started

### Accessing the Console

1. **Start the Takhin Console server**:
   ```bash
   cd backend
   go run ./cmd/console -data-dir /path/to/data -api-addr :8080
   ```

2. **Open your browser** and navigate to:
   ```
   http://localhost:8080
   ```

3. **Default landing page**: The Dashboard will load automatically.

### System Requirements

- **Modern browser**: Chrome 90+, Firefox 88+, Edge 90+, Safari 14+
- **Screen resolution**: Minimum 1366x768, optimized for 1920x1080
- **Network**: Stable connection for WebSocket real-time updates

---

## Dashboard Overview

The Dashboard provides a real-time view of your Takhin cluster's health and performance.

### Key Metrics

**Top Statistics Cards**:
- **Topics**: Total number of topics in the cluster
- **Partitions**: Total partition count across all topics
- **Consumer Groups**: Active consumer groups
- **Connections**: Current active client connections

**Charts**:
1. **Throughput Chart** (Messages/Second)
   - Blue line: Produce rate (messages written)
   - Green line: Fetch rate (messages read)
   - Updates every 2 seconds

2. **Latency Chart** (Milliseconds)
   - Shows P50, P95, P99 latency percentiles
   - Separate metrics for produce and fetch operations

**Tables**:
- **Topic Statistics**: Top 5 topics by message count
- **Consumer Lag**: Groups with highest lag

**System Resources**:
- Memory usage (current allocation)
- Disk usage (data directory size)
- Goroutine count (concurrent tasks)

### WebSocket Connection

The Dashboard uses WebSocket for real-time updates:
- Connection status shown in browser console
- Auto-reconnects on disconnect (3-second delay)
- Falls back to polling if WebSocket unavailable

---

## Managing Topics

Navigate to **Topics** from the sidebar to manage your Kafka topics.

### Creating a Topic

1. Click **"Create Topic"** button
2. Fill in the form:
   - **Topic Name**: Alphanumeric, dots, underscores, hyphens allowed
     - Valid: `my-topic`, `events.user.signup`
     - Invalid: `my topic`, `events/user`
   - **Partitions**: Number of partitions (1-100)
     - More partitions = higher parallelism
     - Consider your consumer group size
3. Click **"Create"**
4. Topic appears in the list immediately

### Viewing Topics

**Table Columns**:
- **Topic Name**: Click to view details
- **Partitions**: Number of partitions
- **Total Messages**: Aggregate high water mark
- **Status**: Health indicator (always "HEALTHY" in current version)

**Actions**:
- **Search**: Filter topics by name (top-right search box)
- **Sort**: Click column headers to sort
- **Refresh**: Reload topic list

### Deleting a Topic

1. Click **"Delete"** button for the topic
2. Confirm deletion in the modal
3. **Warning**: This action is irreversible and deletes all messages

### Viewing Topic Details

Click **"Messages"** button to navigate to the message browser for that topic.

---

## Browsing Messages

Access via **Topics → [Topic Name] → Messages** or directly from topic list.

### Selecting Data

**Partition Selection**:
- Dropdown shows all partitions for the topic
- Each partition is independent (separate message log)

**Query Modes**:

1. **Offset Mode** (default):
   - **Start Offset**: Beginning offset to fetch from
   - **Limit**: Maximum messages to return (default: 100)
   - Use for precise offset-based queries

2. **Timestamp Mode**:
   - Select date and time range
   - Fetches messages published within that window
   - Useful for time-based debugging

### Filtering Messages

**Available Filters**:
- **Key Search**: Filter messages by key content
- **Value Search**: Filter messages by value content
- **Offset Range**: Limit to specific offset range
- **Timestamp Range**: Date/time range picker

**Filter Behavior**:
- Filters are applied client-side after fetch
- Combine multiple filters for precise results
- Clear filters to reset

### Viewing Message Details

Click **"View"** on any message to see:
- **Key**: Message key (string representation)
- **Value**: Message value with JSON pretty-printing
- **Offset**: Exact position in partition
- **Partition**: Partition ID
- **Timestamp**: Unix timestamp (milliseconds)
- **Headers**: Message headers (if any)

### Best Practices

- **Large topics**: Use offset ranges to limit data transfer
- **Recent messages**: Sort by timestamp descending
- **Debugging**: Use timestamp mode to find messages around an incident
- **Performance**: Reduce limit if UI becomes sluggish

---

## Monitoring Consumer Groups

Navigate to **Consumer Groups** to track consumption lag and group health.

### Consumer Group List

**Columns**:
- **Group ID**: Unique consumer group identifier
- **State**: Current state (Stable, Empty, PreparingRebalance, etc.)
- **Members**: Number of active consumers
- **Total Lag**: Sum of lag across all partitions

**Features**:
- Auto-refresh every 5 seconds (toggle on/off)
- Click group to view details

### Consumer Group Details

**Members Section**:
- Member ID (UUID)
- Client ID (application name)
- Host (IP address)
- Assigned partitions

**Offset Commits**:
- Topic and partition
- Committed offset
- Log end offset
- **Lag**: Difference between committed and end offset
- Metadata (commit timestamp, etc.)

**Lag Visualization**:
- Progress bars show consumption progress
- Red warning when lag > 1000 messages

### Understanding Lag

**Healthy Lag**: 0-100 messages (consumer keeping up)
**Warning Lag**: 100-1000 messages (consumer slightly behind)
**Critical Lag**: >1000 messages (consumer falling behind)

**Common Causes**:
- Slow consumer processing
- Network issues
- Consumer restarts/rebalances
- Sudden spike in produce rate

### Troubleshooting Consumer Issues

1. **High Lag**:
   - Scale out consumers (add more instances)
   - Optimize consumer processing logic
   - Check for network bottlenecks

2. **Rebalancing**:
   - Consumers joining/leaving trigger rebalances
   - Minimize consumer restarts
   - Increase session timeout if network is unstable

3. **No Progress**:
   - Check consumer logs for errors
   - Verify consumer is running
   - Check broker connectivity

---

## Broker Information

Navigate to **Brokers** to view cluster topology and broker health.

### Cluster Statistics

**Top Cards**:
- **Brokers**: Number of brokers in cluster (1 in single-node mode)
- **Topics**: Total topics
- **Total Messages**: Aggregate message count
- **Total Size**: Disk space used by messages

### Broker Details

Each broker card shows:
- **Broker ID**: Unique broker identifier
- **Address**: Host and port (e.g., `localhost:9092`)
- **Status**: Online/Offline indicator
- **Controller**: Badge if this broker is the cluster controller
- **Topics**: Number of topics hosted
- **Partitions**: Number of partitions on this broker

**Current Limitation**: Single-broker mode only. Multi-broker clustering planned for future releases.

---

## Configuration Management

Navigate to **Configuration** to manage cluster and topic settings.

### Cluster Configuration

**Modifiable Settings**:
- **Max Message Bytes**: Maximum message size (default: 1MB)
- **Max Connections**: Maximum concurrent connections
- **Request Timeout**: Client request timeout (ms)
- **Connection Timeout**: TCP connection timeout (ms)
- **Log Retention Hours**: How long to keep messages

**Editing**:
1. Click **"Edit"** button
2. Modify values in the form
3. Click **"Save"** to apply
4. Changes take effect immediately

### Topic Configuration

**Per-Topic Settings**:
- **Compression Type**: none, gzip, snappy, lz4, zstd
- **Cleanup Policy**: delete, compact
- **Retention MS**: Message retention time (milliseconds)
- **Segment MS**: Time before rolling to new segment
- **Max Message Bytes**: Topic-specific message size limit
- **Min In-Sync Replicas**: Minimum replicas for ack=all

**Batch Update**:
1. Select multiple topics
2. Click **"Batch Update"**
3. Modify settings (applies to all selected topics)
4. Confirm to apply

### Configuration Best Practices

- **Retention**: Balance between storage cost and data availability
- **Compression**: Use for network-limited environments (slight CPU overhead)
- **Cleanup Policy**: 
  - `delete`: For time-series data
  - `compact`: For changelog/state data
- **Segment Size**: Larger segments = fewer files, slower recovery

---

## API Authentication

The Console supports optional API key authentication.

### Enabling Authentication

**Backend Configuration**:
```bash
./takhin-console \
  -enable-auth \
  -api-keys "key1,key2,key3"
```

### Setting API Key (Browser)

**Option 1: LocalStorage**
```javascript
localStorage.setItem('takhin_api_key', 'your-api-key-here')
```

**Option 2: URL Parameter**
```
http://localhost:8080?apiKey=your-api-key-here
```

**Option 3: Login Form** (if implemented)
- Enter API key in the login modal
- Key is stored in localStorage for subsequent requests

### Authentication Header

All API requests include:
```
Authorization: Bearer <api-key>
```

**Unauthenticated Endpoints**:
- `/api/health` (health checks always accessible)
- `/swagger/*` (API documentation)

---

## Troubleshooting

### WebSocket Not Connecting

**Symptoms**: Dashboard metrics not updating, console shows WebSocket errors

**Solutions**:
1. Check backend is running: `curl http://localhost:8080/api/health`
2. Verify WebSocket endpoint: `ws://localhost:8080/api/monitoring/ws`
3. Check browser console for CORS or connection errors
4. Ensure no proxy/firewall blocking WebSocket upgrades

### Page Loads Slowly

**Symptoms**: UI freezes, delayed responses

**Solutions**:
1. Reduce auto-refresh intervals
2. Limit message fetch count (use smaller limit in message browser)
3. Disable real-time updates on Dashboard (pause WebSocket)
4. Check network latency (use browser DevTools → Network tab)

### Messages Not Appearing

**Symptoms**: Empty message list despite messages in topic

**Solutions**:
1. Verify partition selection (check correct partition)
2. Check offset range (ensure start offset is valid)
3. Verify messages exist: `curl http://localhost:8080/api/topics/{topic}/messages?partition=0&offset=0&limit=10`
4. Clear browser cache

### 401 Unauthorized Errors

**Symptoms**: API calls fail with 401 status

**Solutions**:
1. Verify API key authentication is enabled in backend
2. Check API key is set in localStorage
3. Re-login or reset API key
4. Verify API key matches backend configuration

### Consumer Group Lag Not Updating

**Symptoms**: Lag metrics frozen or incorrect

**Solutions**:
1. Verify consumer group is active (check members count)
2. Check consumer is committing offsets
3. Refresh page to force reload
4. Verify backend coordinator is running

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl + K` | Focus search box (Topics page) |
| `Esc` | Close modal/drawer |
| `F5` | Refresh page |
| `Ctrl + R` | Reload data (context-dependent) |

---

## API Reference

For detailed API documentation, visit:
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **REST API Docs**: `docs/api/console-rest-api.md`

---

## Support

**Documentation**:
- Architecture: `docs/architecture/`
- API Reference: `docs/api/`
- Developer Guide: `docs/DEVELOPER_GUIDE.md`

**Community**:
- GitHub Issues: Report bugs and feature requests
- Discussions: Ask questions and share ideas

**Logs**:
- Backend logs: Check console server output
- Browser console: Check for JavaScript errors
- Network tab: Inspect API requests/responses

---

## Version Information

**Console Version**: 1.0.0
**Supported Kafka Protocol**: 2.8.0+
**Minimum Backend Version**: Takhin 1.0.0

For release notes and changelog, see `CHANGELOG.md` in the project root.
