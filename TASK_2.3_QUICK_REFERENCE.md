# Task 2.3: Quick Reference Guide

## 🚀 Development Commands

```bash
# Navigate to frontend
cd frontend

# Install dependencies
npm install

# Development server (http://localhost:3000)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Type checking
npm run type-check

# Linting
npm run lint
npm run lint:fix

# Code formatting
npm run format
npm run format:check
```

## 📁 File Structure

```
frontend/src/
├── layouts/
│   └── MainLayout.tsx          # Main app layout with sidebar/header
├── pages/
│   ├── Dashboard.tsx           # Dashboard page with statistics
│   ├── Topics.tsx              # Topics management page
│   ├── Brokers.tsx             # Brokers monitoring page
│   └── Consumers.tsx           # Consumer groups page
├── api/
│   └── client.ts               # API client (from Task 2.1)
├── types/
│   └── index.ts                # TypeScript type definitions
├── App.tsx                     # Main app component with routing
├── main.tsx                    # Entry point
├── index.css                   # Global styles
└── App.css                     # App-level styles
```

## 🎨 Component Usage Examples

### Adding a New Page

```typescript
// 1. Create page component
// src/pages/NewPage.tsx
import { Typography } from 'antd'

const { Title } = Typography

export default function NewPage() {
  return (
    <div>
      <Title level={2}>New Page</Title>
      {/* Page content */}
    </div>
  )
}

// 2. Add route in App.tsx
import NewPage from './pages/NewPage'

<Route path="new-page" element={<NewPage />} />

// 3. Add menu item in MainLayout.tsx
{
  key: '/new-page',
  icon: <YourIcon />,
  label: <Link to="/new-page">New Page</Link>,
}
```

### Using Ant Design Components

```typescript
// Import components
import { Button, Table, Card, Space } from 'antd'
import { PlusOutlined } from '@ant-design/icons'

// Use in component
<Space>
  <Button type="primary" icon={<PlusOutlined />}>
    Create
  </Button>
  <Button>Cancel</Button>
</Space>
```

### Table with TypeScript

```typescript
import type { TableColumnsType } from 'antd'

interface DataType {
  key: string
  name: string
  value: number
}

const columns: TableColumnsType<DataType> = [
  {
    title: 'Name',
    dataIndex: 'name',
    key: 'name',
    sorter: (a, b) => a.name.localeCompare(b.name),
  },
  {
    title: 'Value',
    dataIndex: 'value',
    key: 'value',
    sorter: (a, b) => a.value - b.value,
  },
]

<Table columns={columns} dataSource={data} />
```

## 🎯 Common Patterns

### Loading State

```typescript
const [loading, setLoading] = useState(false)

const handleRefresh = async () => {
  setLoading(true)
  try {
    // API call
  } finally {
    setLoading(false)
  }
}

<Button loading={loading}>Refresh</Button>
```

### Search Filter

```typescript
const [searchText, setSearchText] = useState('')

const filteredData = data.filter(item =>
  item.name.toLowerCase().includes(searchText.toLowerCase())
)

<Input
  placeholder="Search..."
  value={searchText}
  onChange={e => setSearchText(e.target.value)}
/>
```

### Responsive Grid

```typescript
import { Row, Col } from 'antd'

<Row gutter={[16, 16]}>
  <Col xs={24} sm={12} lg={6}>
    <Card>Content 1</Card>
  </Col>
  <Col xs={24} sm={12} lg={6}>
    <Card>Content 2</Card>
  </Col>
</Row>
```

## 🔧 Configuration

### Vite Config (`vite.config.ts`)
```typescript
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:8080', // Future backend proxy
    },
  },
})
```

### Theme Config (`main.tsx`)
```typescript
<ConfigProvider
  theme={{
    token: {
      colorPrimary: '#1890ff',
      // Add more theme tokens
    },
  }}
>
  <App />
</ConfigProvider>
```

## 🐛 Troubleshooting

### Issue: Types not found
```bash
# Reinstall dependencies
rm -rf node_modules package-lock.json
npm install
```

### Issue: Build fails
```bash
# Clear cache
rm -rf dist
npm run build
```

### Issue: ESLint errors
```bash
# Auto-fix
npm run lint:fix
```

### Issue: Port already in use
```bash
# Change port in vite.config.ts or use:
PORT=3001 npm run dev
```

## 📊 Current Implementation Status

### Completed ✅
- [x] Main layout structure
- [x] Sidebar navigation
- [x] Top header with breadcrumbs
- [x] User dropdown menu
- [x] Routing configuration
- [x] Dashboard page skeleton
- [x] Topics page skeleton
- [x] Brokers page skeleton
- [x] Consumers page skeleton
- [x] Responsive design
- [x] Loading states
- [x] Empty states

### Not Yet Implemented ❌
- [ ] API integration
- [ ] Real data fetching
- [ ] Forms (Create/Edit)
- [ ] Modals
- [ ] Detail pages
- [ ] Charts/Graphs
- [ ] Authentication
- [ ] Error handling
- [ ] Notifications
- [ ] WebSocket updates

## 🔗 Quick Links

### Documentation
- [Ant Design Components](https://ant.design/components/overview/)
- [React Router v6](https://reactrouter.com/en/main)
- [Vite Guide](https://vitejs.dev/guide/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)

### Project Files
- [Main Layout](./frontend/src/layouts/MainLayout.tsx)
- [App Routes](./frontend/src/App.tsx)
- [Dashboard](./frontend/src/pages/Dashboard.tsx)
- [Topics](./frontend/src/pages/Topics.tsx)

## 💡 Tips

1. **Use Ant Design Icons**: Import from `@ant-design/icons`
2. **Type Everything**: Enable TypeScript strict mode
3. **Responsive First**: Use Ant Design grid system
4. **Loading States**: Always show loading indicators
5. **Empty States**: Provide helpful messages
6. **Error Boundaries**: Will add in future tasks
7. **Code Splitting**: Use React.lazy() for large pages
8. **Memoization**: Use React.memo() for expensive renders

## 📝 Next Steps (Task 2.4)

1. Create API service layer
2. Implement data fetching hooks
3. Connect pages to backend
4. Add error handling
5. Implement refresh logic
6. Add notifications

## 🎓 Learning Resources

### For Layout
- Ant Design Layout: https://ant.design/components/layout
- CSS Flexbox: https://css-tricks.com/snippets/css/a-guide-to-flexbox/

### For Tables
- Ant Design Table: https://ant.design/components/table
- Sorting & Filtering: https://ant.design/components/table#components-table-demo-ajax

### For Forms
- Ant Design Form: https://ant.design/components/form
- Form Validation: https://ant.design/components/form#components-form-demo-advanced-search

### For Routing
- React Router Tutorial: https://reactrouter.com/en/main/start/tutorial
- Nested Routes: https://reactrouter.com/en/main/start/concepts#nested-routes

---

**Quick Reference Guide** - Task 2.3 Complete ✅

For questions or issues, refer to:
- TASK_2.3_SUMMARY.md (Detailed implementation)
- TASK_2.3_VISUAL_OVERVIEW.md (Visual layouts)
- This file (Quick reference)
