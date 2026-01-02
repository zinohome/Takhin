# Task 2.6: Message Browser - Quick Reference

## Quick Start

### Accessing Message Browser
1. Navigate to Topics page
2. Click "Messages" button on any topic row
3. Loads at: `/topics/{topicName}/messages`

### Basic Usage
```
1. Page loads → Shows topic info card
2. Click "Filter" → Open filter modal
3. Select partition → Required
4. Set offset range → Start at specific offset
5. Click "Apply & Load" → Fetches and displays messages
6. Click "View" on any row → Opens detail drawer
7. Click "Export" → Downloads all filtered messages as JSON
```

## Component API

### Messages Component
**Location**: `frontend/src/pages/Messages.tsx`

**Route Parameters**:
- `topicName`: string (from URL path)

**State**:
```typescript
messages: Message[]           // Filtered messages
topicDetail: TopicDetail      // Topic metadata
filter: MessageFilter         // Active filter criteria
selectedMessage: Message      // For detail drawer
```

**Key Functions**:
```typescript
loadTopicDetail()            // Fetch topic metadata
loadMessages()               // Fetch messages from API
applyFilters(msgs)          // Client-side filtering
handleViewMessage(msg)       // Open detail drawer
handleExport()              // Export to JSON
```

## Filter Options

### MessageFilter Interface
```typescript
interface MessageFilter {
  partition?: number       // Which partition to read
  startOffset?: number    // Starting offset (inclusive)
  endOffset?: number      // Ending offset (inclusive)
  startTime?: number      // Start timestamp (Unix ms)
  endTime?: number        // End timestamp (Unix ms)
  keySearch?: string      // Key substring search
  valueSearch?: string    // Value substring search
}
```

### Filter Behavior
- **Partition**: Required, selects which partition to query
- **Offset Range**: Optional, filters messages by offset
- **Time Range**: Optional, filters by timestamp
- **Key/Value Search**: Optional, case-insensitive substring match
- **Combination**: All filters applied together (AND logic)

## API Integration

### Get Messages
```typescript
// API Call
takhinApi.getMessages(topicName, {
  partition: 0,
  offset: 100,
  limit: 100
})

// Backend Endpoint
GET /api/topics/{topic}/messages?partition=0&offset=100&limit=100

// Response
Message[] = [
  {
    partition: 0,
    offset: 100,
    key: "user123",
    value: '{"name":"Alice"}',
    timestamp: 1735819200000
  }
]
```

### Get Topic Details
```typescript
// API Call
takhinApi.getTopic(topicName)

// Response
TopicDetail = {
  name: "orders",
  partitionCount: 3,
  partitions: [
    { id: 0, highWaterMark: 1500 },
    { id: 1, highWaterMark: 1200 },
    { id: 2, highWaterMark: 1800 }
  ]
}
```

## UI Components

### Table Columns
| Column | Width | Features |
|--------|-------|----------|
| Partition | 100px | Blue tag |
| Offset | 120px | Sortable |
| Timestamp | 180px | Formatted date, sortable |
| Key | 200px | Ellipsis, code style |
| Value | Auto | Ellipsis, JSON badge |
| Actions | 100px | View button |

### Filter Modal Fields
```
┌─────────────────────────────────────┐
│ Filter Messages                     │
├─────────────────────────────────────┤
│ Partition: [Select v]              │
│                                     │
│ Offset Range:                       │
│   [Start] [End]                     │
│                                     │
│ Time Range:                         │
│   [Date Picker Range]               │
│                                     │
│ Search by Key:                      │
│   [🔍 Search input]                 │
│                                     │
│ Search by Value:                    │
│   [🔍 Search input]                 │
│                                     │
│ [Reset] [Cancel] [Apply & Load]    │
└─────────────────────────────────────┘
```

### Detail Drawer Layout
```
┌──────────────────────────────────────┐
│ Message Details                   [×]│
├──────────────────────────────────────┤
│ Partition: [0]                       │
│                                      │
│ Offset: 12345 [📋]                   │
│                                      │
│ Timestamp:                           │
│   2026-01-02 10:00:00.000           │
│   1735819200000                      │
│                                      │
│ Key:                                 │
│   user123                            │
│                                      │
│ Value: [JSON]                        │
│   {                                  │
│     "name": "Alice",                 │
│     "email": "alice@example.com"     │
│   }                                  │
│                                      │
│ [📥 Export Message]                  │
└──────────────────────────────────────┘
```

## JSON Detection

### Automatic Formatting
```typescript
// Detection
isJSON(str: string): boolean {
  try {
    JSON.parse(str)
    return true
  } catch {
    return false
  }
}

// Formatting
formatValue(value: string): string {
  try {
    const parsed = JSON.parse(value)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return value
  }
}
```

### Visual Indicators
- **Table**: Green "JSON" badge next to value
- **Drawer**: Pretty-printed with indentation
- **Styling**: Code block background for JSON

## Export Functionality

### Bulk Export
```typescript
// Exports all filtered messages
handleExport() {
  const jsonData = JSON.stringify(messages, null, 2)
  const blob = new Blob([jsonData], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.download = `${topicName}_messages_${Date.now()}.json`
  link.href = url
  link.click()
  URL.revokeObjectURL(url)
}
```

### Filename Format
```
{topicName}_messages_{timestamp}.json
Example: orders_messages_1735819200000.json
```

### Single Message Export
- Available in detail drawer
- Filename: `message_{partition}_{offset}.json`
- Exports full message object

## Common Tasks

### View Recent Messages
1. Click "Filter" button
2. Select partition
3. Leave start offset at 0 (or set to HWM - 100)
4. Click "Apply & Load"

### Search for Specific Key
1. Load messages for partition
2. Click "Filter"
3. Enter key substring in "Search by Key"
4. Click "Apply & Load"

### Export Time Range
1. Click "Filter"
2. Select partition and offset
3. Set time range with date picker
4. Click "Apply & Load"
5. Click "Export" button

### View JSON Messages
1. JSON automatically detected
2. Green "JSON" badge in table
3. Click "View" for pretty-printed version
4. Drawer shows formatted JSON

## Keyboard Shortcuts

### Table Navigation
- **Arrow Keys**: Navigate table rows
- **Page Up/Down**: Change pages
- **Home/End**: First/last page

### Modal/Drawer
- **Esc**: Close modal or drawer
- **Enter**: Submit form (in filter modal)

## Error Handling

### Common Errors
| Error | Cause | Solution |
|-------|-------|----------|
| "Failed to load topic details" | Topic doesn't exist | Check topic name |
| "Failed to load messages" | Invalid partition/offset | Verify partition exists |
| "No messages to export" | Empty result set | Adjust filters |

### Network Errors
- Auto-retry not implemented
- Manual refresh required
- Error notifications display in top-right

## Performance Tips

### Optimal Batch Sizes
- **Small topics**: 100 messages default is fine
- **Large topics**: Consider smaller batches (50)
- **Slow network**: Reduce page size in table

### Filter Strategy
1. **Start with partition**: Narrows dataset
2. **Use offset range**: Reduces fetch size
3. **Apply client filters**: Fast, no API call
4. **Export selectively**: Only what you need

### Avoid Performance Issues
- ❌ Don't fetch entire partition (millions of messages)
- ❌ Don't export huge datasets (>10K messages)
- ✅ Use offset ranges to limit scope
- ✅ Export in smaller batches if needed

## Integration with Topics Page

### Navigation Flow
```
Topics Page
  ↓ Click "Messages" button
Messages Page (for topic)
  ↓ Click "Back" button
Topics Page
```

### Passing Data
- Topic name via URL parameter
- No state passing required
- Fresh data loaded on each visit

## Troubleshooting

### Messages Not Loading
1. Check browser console for errors
2. Verify backend is running
3. Check API endpoint availability
4. Confirm topic exists

### Filter Not Working
1. Check filter criteria are valid
2. Verify partition has messages
3. Try resetting filters
4. Reload page

### Export Not Working
1. Check browser download permissions
2. Verify messages are loaded
3. Check browser console
4. Try smaller export

### JSON Not Pretty-Printing
1. Verify value is valid JSON
2. Check browser console for parse errors
3. JSON badge should appear if valid

## Code Examples

### Custom Filter Logic
```typescript
// Add to applyFilters() function
if (filter.customField) {
  filtered = filtered.filter(m => 
    m.value.includes(filter.customField)
  )
}
```

### Additional Table Column
```typescript
{
  title: 'Size',
  dataIndex: 'value',
  key: 'size',
  width: 100,
  render: (value: string) => `${value.length} bytes`,
}
```

### Custom Export Format
```typescript
// CSV export
const csv = messages.map(m => 
  `${m.partition},${m.offset},${m.key},${m.value},${m.timestamp}`
).join('\n')
const blob = new Blob([csv], { type: 'text/csv' })
```

## Dependencies

### Required Packages
- `antd`: ^5.x - UI components
- `dayjs`: ^1.x - Date formatting
- `react-router-dom`: ^6.x - Navigation
- `@ant-design/icons`: ^5.x - Icons

### Optional Enhancements
- `react-json-view`: Better JSON visualization
- `papaparse`: CSV export
- `file-saver`: Enhanced download

## Related Documentation

- **API Client**: `frontend/src/api/README.md`
- **Topics Component**: `frontend/src/pages/Topics.tsx`
- **Type Definitions**: `frontend/src/api/types.ts`
- **Backend API**: `backend/pkg/console/server.go`

## Quick Reference Card

```
╔══════════════════════════════════════════════════════════╗
║                MESSAGE BROWSER QUICK REF                  ║
╠══════════════════════════════════════════════════════════╣
║ Load Messages:   Filter → Select Partition → Apply       ║
║ View Details:    Click "View" on message row             ║
║ Export All:      Click "Export" button (toolbar)         ║
║ Export One:      Click "Export" in detail drawer         ║
║ Search Key:      Filter → "Search by Key" field          ║
║ Search Value:    Filter → "Search by Value" field        ║
║ Time Range:      Filter → Date Picker Range              ║
║ Offset Range:    Filter → Start/End Offset fields        ║
║ Go Back:         Click "Back" button (top-left)          ║
║ Refresh:         Click "Refresh" button (toolbar)        ║
╚══════════════════════════════════════════════════════════╝
```

---

**Version**: 1.0  
**Last Updated**: 2026-01-02  
**Component**: Message Browser  
