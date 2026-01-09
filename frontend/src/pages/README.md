# Takhin Console Frontend Pages

This directory contains the main pages of the Takhin Console web application.

## Pages Overview

### 📊 Dashboard (`Dashboard.tsx`)
**Route**: `/dashboard`

The main landing page displaying real-time cluster metrics and health information.

**Features**:
- Real-time WebSocket connection for live metrics updates
- Cluster health statistics (Topics, Partitions, Consumer Groups, Connections)
- Throughput charts (Produce/Fetch rates)
- Latency charts (P50, P95, P99 percentiles)
- Top topics by message count and size
- Consumer group lag monitoring
- System resource usage (Memory, Disk, Goroutines)

**Data Sources**:
- WebSocket: `/api/monitoring/ws` (real-time metrics)
- REST API: `/api/health` (health status)

**Auto-refresh**: Every 2 seconds via WebSocket

---

### 📁 Topics (`Topics.tsx`)
**Route**: `/topics`

Manage Kafka topics - create, view, and delete topics.

**Features**:
- List all topics with partition count and message statistics
- Search/filter topics by name
- Create new topics with configurable partition count
- Delete topics with confirmation
- View topic details (partition distribution, high water marks)
- Navigate to message browser for each topic

**Actions**:
- Create Topic: Validates name format and partition count
- Delete Topic: Requires confirmation, cascade deletes all messages
- View Messages: Navigate to message browser

**Validation**:
- Topic name: `^[a-zA-Z0-9._-]+$`
- Partitions: Min 1, Max 100

---

### 🖥️ Brokers (`Brokers.tsx`)
**Route**: `/brokers`

View broker information and cluster statistics.

**Features**:
- List all brokers in the cluster
- Show broker status (Online/Offline)
- Display controller broker
- Cluster-wide statistics (Total messages, size, topics, partitions)
- Broker details (Host, Port, Topic/Partition counts)

**Data**:
- Currently supports single-broker mode
- Multi-broker support planned for future releases

---

### 👥 Consumer Groups (`Consumers.tsx`)
**Route**: `/consumers`, `/consumers/:groupId`

Monitor consumer groups and their lag metrics.

**Features**:
- List all consumer groups with state and member count
- Real-time lag monitoring
- Auto-refresh every 5 seconds (toggle on/off)
- Detailed group view:
  - Member list with client IDs and hosts
  - Partition assignments
  - Offset commits per topic/partition
  - Lag visualization

**Lag Warning**:
- Displays warning when lag > 1000 messages
- Progress bars show consumption progress

---

### 🔍 Messages (`Messages.tsx`)
**Route**: `/topics/:topicName/messages`

Browse and search messages within a topic.

**Features**:
- Select topic and partition
- Filter by offset range
- Filter by timestamp (date/time picker)
- Search messages by key or value
- Message detail view (JSON pretty-print, headers, metadata)
- Export messages (planned)
- Pagination and scrolling

**Query Modes**:
1. **Offset Mode**: Specify start offset and limit
2. **Timestamp Mode**: Fetch messages from a specific time

**Filters**:
- Client-side filtering by key/value search
- Timestamp range filtering
- Partition selection

---

### ⚙️ Configuration (`Configuration.tsx`)
**Route**: `/configuration`

Manage cluster and topic configurations.

**Features**:
- View and update cluster-wide settings
- Per-topic configuration management
- Batch update multiple topic configs
- Configuration change history
- Validate configuration values

**Settings**:
- Cluster: Max message bytes, connections, timeouts, retention
- Topics: Compression, cleanup policy, retention, segment size

---

## Component Dependencies

All pages use:
- **Ant Design** components for UI
- **React Router** for navigation
- **takhinApi** client for backend communication
- **TypeScript** for type safety

### Common Patterns

#### API Error Handling
```typescript
try {
  const data = await takhinApi.someMethod()
  setData(data)
} catch (error) {
  message.error('Operation failed')
  console.error(error)
}
```

#### Loading States
All pages implement loading skeletons or spinners during data fetch:
```typescript
const [loading, setLoading] = useState(false)
setLoading(true)
// ... fetch data
setLoading(false)
```

#### Responsive Design
Pages use Ant Design's responsive grid system:
- `xs`: Mobile (< 576px)
- `sm`: Tablet (≥ 576px)
- `lg`: Desktop (≥ 992px)

---

## Development Guidelines

### Adding a New Page

1. **Create the page component** in `frontend/src/pages/`
2. **Add route** in `frontend/src/App.tsx`:
   ```typescript
   <Route path="/my-page" element={<MyPage />} />
   ```
3. **Update navigation** in `frontend/src/layouts/MainLayout.tsx`
4. **Define TypeScript types** in `frontend/src/api/types.ts`
5. **Add API methods** in `frontend/src/api/takhinApi.ts`
6. **Update this README** with page documentation

### Code Style
- Use functional components with hooks
- Destructure props and state
- Use TypeScript interfaces for all props
- Follow Ant Design naming conventions
- Keep components under 300 lines (split into smaller components if needed)

### Testing
- Manual testing in Chrome, Firefox, Edge
- Test responsive layouts at 1366x768 and 1920x1080
- Verify error states and loading states
- Test with slow network (throttling)

---

## API Endpoints Reference

| Page | Endpoints Used |
|------|---------------|
| Dashboard | `/api/health`, `/api/monitoring/ws`, `/api/monitoring/metrics` |
| Topics | `/api/topics`, `/api/topics/{topic}` |
| Brokers | `/api/brokers`, `/api/cluster/stats` |
| Consumers | `/api/consumer-groups`, `/api/consumer-groups/{group}` |
| Messages | `/api/topics/{topic}/messages` |
| Configuration | `/api/configs/cluster`, `/api/configs/topics/{topic}` |

---

## Troubleshooting

### WebSocket Connection Issues
If real-time metrics don't update:
1. Check browser console for WebSocket errors
2. Verify backend is running and accessible
3. Check CORS configuration
4. Ensure `/api/monitoring/ws` endpoint is reachable

### API Authentication
If you see 401 errors:
1. Check if API key authentication is enabled in backend
2. Set API key in localStorage: `localStorage.setItem('takhin_api_key', 'your-key')`
3. Verify `Authorization` header is included in requests

### Slow Performance
If UI is sluggish:
1. Reduce auto-refresh intervals
2. Limit WebSocket data payloads
3. Use pagination for large data sets
4. Consider disabling real-time updates on slower devices

---

## Future Enhancements

- [ ] Message producer UI (send messages from browser)
- [ ] ACL management page
- [ ] Audit log viewer
- [ ] Performance metrics and benchmarks
- [ ] Schema registry integration
- [ ] Dark mode theme
- [ ] Multi-language support
- [ ] Export data to CSV/JSON
- [ ] Advanced search with filters
- [ ] Saved queries and bookmarks

---

For backend API documentation, see `docs/api/console-rest-api.md`.
