# Frontend Quick Start Guide

## Setup (First Time)

```bash
cd frontend
npm install
```

## Daily Development

### Start Development Server

```bash
npm run dev
```

Opens on http://localhost:3000 with:
- Hot reload enabled
- API proxy to backend (localhost:8080)

### Code Quality Checks

Run before committing:

```bash
npm run lint:fix
npm run format
npm run type-check
```

### Production Build

```bash
npm run build
npm run preview  # Test production build locally
```

## Common Tasks

### Add New Page

1. Create `src/pages/MyPage.tsx`:
```typescript
import { Typography } from 'antd'

export default function MyPage() {
  return <Typography.Title level={2}>My Page</Typography.Title>
}
```

2. Add route in `src/App.tsx`:
```typescript
<Route path="mypage" element={<MyPage />} />
```

3. Add menu item in `src/layouts/MainLayout.tsx`:
```typescript
{
  key: '/mypage',
  icon: <SomeIcon />,
  label: <Link to="/mypage">My Page</Link>,
}
```

### Add API Service

Create `src/api/myservice.ts`:
```typescript
import apiClient from './client'

export const myService = {
  getData: () => apiClient.get('/endpoint'),
  createData: (data: any) => apiClient.post('/endpoint', data),
}
```

### Add Component

Create `src/components/MyComponent.tsx`:
```typescript
import type { FC } from 'react'

interface Props {
  title: string
}

const MyComponent: FC<Props> = ({ title }) => {
  return <div>{title}</div>
}

export default MyComponent
```

## Troubleshooting

### Reset Everything

```bash
rm -rf node_modules dist package-lock.json
npm install
```

### Port Conflict

Edit `vite.config.ts` and change port:
```typescript
server: { port: 3001 }
```

### Backend Connection Failed

1. Check backend is running: `curl http://localhost:8080/health`
2. Check proxy config in `vite.config.ts`
3. Check browser console for errors

## Useful Commands

```bash
npm run dev          # Development server
npm run build        # Production build
npm run lint         # Check code quality
npm run lint:fix     # Fix code issues
npm run format       # Format code
npm run type-check   # Check TypeScript
```

## VS Code Setup

Recommended extensions:
- ESLint
- Prettier
- TypeScript Vue Plugin (Volar)

Add to `.vscode/settings.json`:
```json
{
  "editor.formatOnSave": true,
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  }
}
```
