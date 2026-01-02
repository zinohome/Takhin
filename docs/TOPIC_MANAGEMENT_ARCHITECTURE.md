# Topic Management Page - Component Architecture

## Component Hierarchy

```
Topics.tsx (Main Page)
├── Statistics Cards (Row)
│   ├── Total Topics Card
│   ├── Total Partitions Card
│   └── Total Messages Card
│
├── Action Buttons
│   ├── Refresh Button
│   └── Create Topic Button → CreateTopicModal
│
├── TopicList Component
│   ├── Search Input (Filter)
│   ├── Table
│   │   ├── Topic Name Column (sortable)
│   │   ├── Partitions Column (sortable)
│   │   ├── Total Messages Column (calculated)
│   │   └── Actions Column
│   │       ├── View Button → TopicDetailDrawer
│   │       └── Delete Button → DeleteTopicModal
│   └── Pagination
│
├── CreateTopicModal (Dialog)
│   ├── Topic Name Input (validated)
│   ├── Partitions Input (1-1000)
│   └── Submit/Cancel Buttons
│
├── TopicDetailDrawer (Side Panel)
│   ├── Topic Summary
│   │   ├── Name (copyable)
│   │   ├── Partition Count
│   │   └── Total Messages
│   └── Partition Table
│       ├── Partition ID
│       └── High Water Mark
│
└── DeleteTopicModal (Dialog)
    ├── Warning Alert
    ├── Topic Details Preview
    │   ├── Name
    │   ├── Partition Count
    │   └── Total Messages
    └── Delete/Cancel Buttons
```

## Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                         Topics.tsx                          │
│                     (Page Controller)                       │
│                                                             │
│  State:                                                     │
│  - topics: TopicSummary[]                                   │
│  - loading, error                                           │
│  - modal/drawer visibility flags                           │
│  - selected topic/topic name                               │
└─────────────────────────────────────────────────────────────┘
                                │
                                │ Props
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Child Components                         │
├─────────────────────────────────────────────────────────────┤
│ TopicList          - Displays topics                        │
│                    - Emits: onView, onDelete                │
│                                                             │
│ CreateTopicModal   - Form validation                        │
│                    - Calls: topicApi.create()               │
│                    - Emits: onSuccess                       │
│                                                             │
│ TopicDetailDrawer  - Fetches: topicApi.get(name)            │
│                    - Displays partition details             │
│                                                             │
│ DeleteTopicModal   - Calls: topicApi.delete(name)           │
│                    - Emits: onSuccess                       │
└─────────────────────────────────────────────────────────────┘
                                │
                                │ API Calls
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                      API Layer (topics.ts)                  │
├─────────────────────────────────────────────────────────────┤
│ topicApi.list()    → GET  /api/topics                       │
│ topicApi.get(name) → GET  /api/topics/{topic}               │
│ topicApi.create()  → POST /api/topics                       │
│ topicApi.delete()  → DEL  /api/topics/{topic}               │
└─────────────────────────────────────────────────────────────┘
                                │
                                │ HTTP
                                ▼
┌─────────────────────────────────────────────────────────────┐
│              Backend API (pkg/console/server.go)            │
└─────────────────────────────────────────────────────────────┘
```

## State Management Pattern

```typescript
// Main page manages all state
const Topics = () => {
  // Data state
  const [topics, setTopics] = useState<TopicSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  // UI state
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [detailDrawerOpen, setDetailDrawerOpen] = useState(false)
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  
  // Selection state
  const [selectedTopicName, setSelectedTopicName] = useState<string | null>(null)
  const [selectedTopic, setSelectedTopic] = useState<TopicSummary | null>(null)
  
  // Actions
  const loadTopics = async () => { /* fetch and update */ }
  const handleViewTopic = (topic) => { /* open drawer */ }
  const handleDeleteTopic = (topic) => { /* open modal */ }
  
  // Child components receive callbacks and data via props
}
```

## Event Flow Examples

### 1. Create Topic Flow
```
User clicks "Create Topic"
  → setCreateModalOpen(true)
  → CreateTopicModal renders
  → User fills form
  → User clicks "Create"
  → topicApi.create(request)
  → onSuccess callback
  → loadTopics() refreshes list
  → setCreateModalOpen(false)
  → Success message displayed
```

### 2. View Details Flow
```
User clicks eye icon on topic row
  → handleViewTopic(topic)
  → setSelectedTopicName(topic.name)
  → setDetailDrawerOpen(true)
  → TopicDetailDrawer renders
  → useEffect triggers loadTopicDetails()
  → topicApi.get(topicName)
  → Partition data displayed
```

### 3. Delete Topic Flow
```
User clicks delete icon
  → handleDeleteTopic(topic)
  → setSelectedTopic(topic)
  → setDeleteModalOpen(true)
  → DeleteTopicModal shows warning + preview
  → User confirms
  → topicApi.delete(topic.name)
  → onSuccess callback
  → loadTopics() refreshes list
  → setDeleteModalOpen(false)
  → Success message displayed
```

### 4. Search/Filter Flow
```
User types in search box
  → setSearchText(value) in TopicList
  → useMemo recomputes filteredTopics
  → Table re-renders with filtered data
  → No API calls (client-side filtering)
```

## Component Responsibilities

| Component | Responsibility | State | Side Effects |
|-----------|---------------|-------|--------------|
| **Topics.tsx** | Orchestration, data fetching | All app state | API calls for list |
| **TopicList** | Display & interactions | Search text only | None |
| **CreateTopicModal** | Form UI & validation | Form values, loading | API call to create |
| **TopicDetailDrawer** | Detail display | Detail data, loading | API call to get details |
| **DeleteTopicModal** | Confirmation UI | Loading state | API call to delete |

## API Client Pattern

```typescript
// api/topics.ts - Clean separation of concerns
export const topicApi = {
  list: async (): Promise<TopicSummary[]> => {
    const response = await apiClient.get<TopicSummary[]>('/topics')
    return response.data
  },
  // ... other methods
}

// Usage in components
import { topicApi } from '../api/topics'

const data = await topicApi.list()  // Type-safe, clean
```

## Styling Approach

- **Ant Design Components**: Primary UI library
- **Inline Styles**: Used for layout-specific adjustments
- **Theme**: Uses Ant Design default theme
- **Responsive**: Grid system (Row/Col) for statistics cards
- **Icons**: @ant-design/icons for consistency

## Performance Considerations

1. **Memoization**: Search filter uses `useMemo` to avoid unnecessary recalculations
2. **Lazy Loading**: Detail drawer fetches data only when opened
3. **Pagination**: Table shows 10 items by default, configurable
4. **Efficient Re-renders**: Proper key props on list items
5. **DestroyOnClose**: Modals/drawers destroy content when closed to free memory

## Error Handling Strategy

1. **API Level**: Try-catch in all async functions
2. **User Feedback**: Ant Design `message` component for notifications
3. **Error State**: Error alert shown at page level for list failures
4. **Graceful Degradation**: Loading states prevent broken UI

## Accessibility Features

- Semantic HTML through Ant Design components
- Keyboard navigation support (via Ant Design)
- ARIA labels on interactive elements
- Focus management in modals/drawers
- Clear visual feedback for actions
