# Task 2.8 - Configuration Management Quick Reference

## Quick Links
- **Frontend**: `frontend/src/pages/Configuration.tsx`
- **Backend Handlers**: `backend/pkg/console/config_handlers.go`
- **API Types**: `frontend/src/api/types.ts`
- **API Client**: `frontend/src/api/takhinApi.ts`
- **Menu**: `frontend/src/layouts/MainLayout.tsx`
- **Routes**: `frontend/src/App.tsx`

## API Endpoints

### Cluster Configuration
```
GET    /api/configs/cluster           - Get cluster config
PUT    /api/configs/cluster           - Update cluster config
```

### Topic Configuration
```
GET    /api/configs/topics/{topic}    - Get topic config
PUT    /api/configs/topics/{topic}    - Update topic config
PUT    /api/configs/topics            - Batch update topics
```

## Usage Examples

### Frontend API Calls

```typescript
// Get cluster configuration
const config = await takhinApi.getClusterConfig()

// Update cluster configuration
const updated = await takhinApi.updateClusterConfig({
  maxMessageBytes: 2097152,
  maxConnections: 200,
})

// Get topic configuration
const topicConfig = await takhinApi.getTopicConfig('my-topic')

// Update single topic
const updatedTopic = await takhinApi.updateTopicConfig('my-topic', {
  compressionType: 'gzip',
  retentionMs: 86400000, // 1 day
})

// Batch update topics
const result = await takhinApi.batchUpdateTopicConfigs({
  topics: ['topic1', 'topic2', 'topic3'],
  config: {
    compressionType: 'lz4',
    cleanupPolicy: 'delete',
    retentionMs: 604800000, // 7 days
  }
})
```

### Backend Handler Testing

```bash
# Get cluster config
curl http://localhost:8080/api/configs/cluster \
  -H "Authorization: your-api-key"

# Update cluster config
curl -X PUT http://localhost:8080/api/configs/cluster \
  -H "Authorization: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"maxMessageBytes": 2097152}'

# Get topic config
curl http://localhost:8080/api/configs/topics/my-topic \
  -H "Authorization: your-api-key"

# Update topic config
curl -X PUT http://localhost:8080/api/configs/topics/my-topic \
  -H "Authorization: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"compressionType": "gzip", "retentionMs": 86400000}'

# Batch update
curl -X PUT http://localhost:8080/api/configs/topics \
  -H "Authorization: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "topics": ["topic1", "topic2"],
    "config": {"compressionType": "lz4"}
  }'
```

## Configuration Options Reference

### Cluster Configuration Fields

| Field | Type | Description | Editable | Validation |
|-------|------|-------------|----------|------------|
| brokerId | int | Broker identifier | No | Read-only |
| listeners | []string | Listener addresses | No | Read-only |
| advertisedHost | string | Advertised hostname | No | Read-only |
| advertisedPort | int | Advertised port | No | Read-only |
| maxMessageBytes | int | Max message size | Yes | >= 1024 |
| maxConnections | int | Max connections | Yes | >= 1 |
| requestTimeoutMs | int | Request timeout | Yes | >= 1000 |
| connectionTimeoutMs | int | Connection timeout | Yes | >= 1000 |
| dataDir | string | Data directory | No | Read-only |
| logSegmentSize | int64 | Segment size | No | Read-only |
| logRetentionHours | int | Retention hours | Yes | >= 1 |
| logRetentionBytes | int64 | Retention bytes | No | Read-only |
| metricsEnabled | bool | Metrics enabled | No | Read-only |
| metricsPort | int | Metrics port | No | Read-only |

### Topic Configuration Fields

| Field | Type | Description | Options | Validation |
|-------|------|-------------|---------|------------|
| name | string | Topic name | - | Required |
| compressionType | string | Compression codec | none, gzip, snappy, lz4, zstd, producer | Valid type |
| cleanupPolicy | string | Cleanup policy | delete, compact | Valid policy |
| retentionMs | int64 | Retention time | - | > 0 |
| segmentMs | int64 | Segment time | - | > 0 |
| maxMessageBytes | int | Max message size | - | >= 1024 |
| minInSyncReplicas | int | Min ISR | - | >= 1 |

## UI Navigation

1. **Access Configuration Page**: Click "Configuration" in left sidebar
2. **Switch Tabs**: Click "Cluster Configuration" or "Topic Configuration"
3. **Edit Cluster**: Click "Edit Configuration" → Modify → "Save Changes"
4. **Select Topics**: Check boxes next to topic names or "Select All"
5. **Batch Update**: Fill batch form → "Apply to Selected Topics"
6. **View Topic Config**: Click "View Config" button for individual topic

## Component Structure

### Configuration.tsx State
```typescript
- activeTab: 'cluster' | 'topics'
- clusterConfig: ClusterConfig | null
- topics: TopicSummary[]
- selectedTopics: string[]
- topicConfigs: Map<string, TopicConfig>
- loading, error, editMode, saving, successMessage
- clusterForm: UpdateClusterConfigRequest
- topicForm: UpdateTopicConfigRequest
```

### Key Functions
- `loadData()` - Load config based on active tab
- `loadTopicConfig(topicName)` - Load specific topic config
- `handleSaveClusterConfig()` - Save cluster changes
- `handleBatchUpdate()` - Apply batch topic updates
- `toggleTopicSelection(topic)` - Select/deselect topic
- `selectAllTopics()` / `deselectAllTopics()` - Bulk selection

## Styling Classes

### Container Classes
- `.config-container` - Main page container
- `.config-section` - Content section with card style
- `.config-grid` - Responsive grid layout
- `.config-group` - Grouped config items

### Component Classes
- `.config-tabs` - Tab navigation
- `.tab-button` - Individual tab
- `.config-item` - Single config field
- `.config-value` - Display value
- `.batch-update-panel` - Batch operations panel
- `.topics-table` - Topic list table

### Button Classes
- `.btn` - Base button
- `.btn-primary` - Primary action
- `.btn-secondary` - Secondary action
- `.btn-sm` - Small button

### Alert Classes
- `.alert` - Base alert
- `.alert-error` - Error message
- `.alert-success` - Success message

## Build & Deploy

### Backend
```bash
cd backend
go build ./pkg/console/...
go test ./pkg/console/...
```

### Frontend
```bash
cd frontend
npm install
npm run build
# Output in dist/
```

### Swagger Documentation
```bash
# Regenerate if needed
cd backend
swag init -g cmd/console/main.go -o docs/swagger
```

## Troubleshooting

### Common Issues

**Issue**: Configuration changes don't persist
- **Cause**: Not yet wired to storage layer
- **Solution**: Changes are logged but need storage integration

**Issue**: Some cluster fields can't be edited
- **Cause**: Fields not exposed by topic manager
- **Solution**: These are read-only for now

**Issue**: Batch update affects wrong topics
- **Cause**: Selection not cleared after operation
- **Solution**: Click "Deselect All" before next operation

**Issue**: TypeScript errors in Configuration.tsx
- **Cause**: Missing type imports
- **Solution**: Ensure all types imported from `api/types.ts`

## Testing Checklist

- [ ] Load configuration page
- [ ] Switch between tabs
- [ ] Edit cluster configuration
- [ ] Save cluster changes
- [ ] Cancel cluster edits
- [ ] Load topic list
- [ ] Select individual topics
- [ ] Select all topics
- [ ] Apply batch updates
- [ ] View individual topic config
- [ ] Test validation errors
- [ ] Verify success messages
- [ ] Check error handling

## Performance Tips

- Topic configs loaded on-demand (not all at once)
- Batch updates processed as single API call
- Form state managed locally for fast UI updates
- Debounce validation for better UX (future enhancement)

## Security Notes

- All endpoints require authentication (API key)
- Input validation on client and server
- No sensitive data in logs
- Consider adding authorization layer
- Rate limiting recommended for batch operations

## Integration Points

### With Task 2.2 (API Client)
- Uses `TakhinApiClient` class
- Follows error handling patterns
- Type-safe API calls

### With Task 2.3 (Topics)
- Shares topic data structures
- Uses same topic list endpoint
- Compatible with topic operations

### Future Integration
- Task 2.6 (Message Browser): Config affects message display
- ACL System: Permission-based config access
- Monitoring: Config change events
- Audit Log: Track configuration history

## Quick Fixes

### Add new cluster config field
1. Add field to `ClusterConfig` in `config_handlers.go`
2. Add field to `ClusterConfig` in `types.ts`
3. Add UI field in `Configuration.tsx`
4. Update validation if needed

### Add new topic config option
1. Add field to `TopicConfig` types (both backend/frontend)
2. Add form field in batch update panel
3. Add validation in `handleUpdateTopicConfig`
4. Update server handler validation

### Change validation rules
1. Update validation in `handleUpdateClusterConfig` (backend)
2. Update validation in `handleSaveClusterConfig` (frontend)
3. Update error messages for clarity

## Related Files

### Backend
- `backend/pkg/console/config_handlers.go` - Handlers
- `backend/pkg/console/server.go` - Routes
- `backend/pkg/console/types.go` - Type definitions
- `backend/pkg/config/config.go` - System config

### Frontend
- `frontend/src/pages/Configuration.tsx` - Main component
- `frontend/src/api/types.ts` - Type definitions
- `frontend/src/api/takhinApi.ts` - API client
- `frontend/src/layouts/MainLayout.tsx` - Navigation
- `frontend/src/App.tsx` - Routing

## Support

For issues or questions:
1. Check this quick reference
2. Review TASK_2.8_COMPLETION_SUMMARY.md
3. Check Swagger docs at `/swagger/index.html`
4. Review backend logs for API errors
5. Check browser console for frontend errors
