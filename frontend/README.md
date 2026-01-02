# Takhin Console Frontend

React + TypeScript frontend for Takhin Kafka-compatible streaming platform.

## Technology Stack

- **Framework**: React 19.2 with TypeScript
- **Build Tool**: Vite 7.2
- **UI Library**: Ant Design 6.1
- **Routing**: React Router 7.11
- **HTTP Client**: Axios 1.13
- **Code Quality**: ESLint + Prettier

## Project Structure

```
frontend/
├── src/
│   ├── api/              # API client and service modules
│   │   └── client.ts     # Axios instance with interceptors
│   ├── assets/           # Static assets (images, fonts)
│   ├── components/       # Reusable React components
│   ├── hooks/            # Custom React hooks
│   ├── layouts/          # Layout components
│   │   └── MainLayout.tsx
│   ├── pages/            # Page components (routed)
│   │   ├── Dashboard.tsx
│   │   ├── Topics.tsx
│   │   └── Brokers.tsx
│   ├── types/            # TypeScript type definitions
│   │   └── index.ts
│   ├── utils/            # Utility functions
│   ├── App.tsx           # Root component with routing
│   ├── main.tsx          # Application entry point
│   └── index.css         # Global styles
├── public/               # Public static files
├── .prettierrc           # Prettier configuration
├── .prettierignore       # Prettier ignore patterns
├── eslint.config.js      # ESLint configuration
├── vite.config.ts        # Vite configuration
├── tsconfig.json         # TypeScript configuration
└── package.json          # Dependencies and scripts
```

## Getting Started

### Prerequisites

- Node.js >= 18.0.0
- npm >= 9.0.0

### Installation

```bash
cd frontend
npm install
```

### Development

Start the development server on http://localhost:3000:

```bash
npm run dev
```

The dev server includes:
- Hot Module Replacement (HMR)
- API proxy to backend (http://localhost:8080)
- Fast refresh for instant updates

### Build

Build for production:

```bash
npm run build
```

Output will be in `dist/` directory.

Preview production build:

```bash
npm run preview
```

## Available Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start development server |
| `npm run build` | Build for production |
| `npm run preview` | Preview production build |
| `npm run lint` | Run ESLint |
| `npm run lint:fix` | Run ESLint and auto-fix issues |
| `npm run format` | Format code with Prettier |
| `npm run format:check` | Check code formatting |
| `npm run type-check` | Run TypeScript compiler checks |

## Configuration

### Vite Configuration

Path aliases and API proxy are configured in `vite.config.ts`:

```typescript
{
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'), // Use @/ for src imports
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080', // Backend API
        changeOrigin: true,
      },
    },
  },
}
```

### ESLint + Prettier

Code quality tools are pre-configured:

- **ESLint**: Enforces code standards with React and TypeScript rules
- **Prettier**: Enforces consistent code formatting
- **Integration**: ESLint + Prettier work together via `eslint-config-prettier`

Run before committing:

```bash
npm run lint:fix && npm run format
```

### TypeScript Configuration

TypeScript is configured with:
- Strict mode enabled
- Path aliases (`@/*` maps to `src/*`)
- React JSX support
- Modern ES2022 target

## Routing

React Router is configured in `src/App.tsx`:

```typescript
<Routes>
  <Route path="/" element={<MainLayout />}>
    <Route index element={<Navigate to="/dashboard" replace />} />
    <Route path="dashboard" element={<Dashboard />} />
    <Route path="topics" element={<Topics />} />
    <Route path="brokers" element={<Brokers />} />
  </Route>
</Routes>
```

### Adding New Routes

1. Create page component in `src/pages/`
2. Add route in `src/App.tsx`
3. Add menu item in `src/layouts/MainLayout.tsx`

## API Integration

### API Client

Configured Axios instance in `src/api/client.ts`:

```typescript
import apiClient from '@/api/client'

// GET request
const response = await apiClient.get('/topics')

// POST request
const response = await apiClient.post('/topics', { name: 'test' })
```

Features:
- Base URL: `/api` (proxied to backend)
- Authentication: Bearer token from localStorage
- Error handling: Auto-redirect on 401
- Timeout: 10 seconds

### Creating API Services

```typescript
// src/api/topics.ts
import apiClient from './client'
import type { Topic } from '@/types'

export const topicService = {
  list: () => apiClient.get<Topic[]>('/topics'),
  create: (data: Partial<Topic>) => apiClient.post('/topics', data),
  delete: (name: string) => apiClient.delete(`/topics/${name}`),
}
```

## Styling

### Ant Design Theming

Theme configuration in `src/main.tsx`:

```typescript
<ConfigProvider
  theme={{
    token: {
      colorPrimary: '#1890ff', // Customize primary color
    },
  }}
>
  <App />
</ConfigProvider>
```

### Global Styles

- `src/index.css`: Global CSS styles
- Component-specific styles: Use CSS Modules or styled-components

## Type Definitions

TypeScript types are defined in `src/types/`:

```typescript
export interface Topic {
  name: string
  partitions: number
  replicationFactor: number
  configs?: Record<string, string>
}
```

## Development Guidelines

### Code Style

- Use functional components with hooks
- Use TypeScript for all files
- Follow ESLint rules (no warnings/errors)
- Format with Prettier before commit
- Use path aliases (`@/`) for imports

### Component Structure

```typescript
import { useState } from 'react'
import { Typography } from 'antd'
import type { FC } from 'react'

interface Props {
  title: string
}

const MyComponent: FC<Props> = ({ title }) => {
  const [state, setState] = useState('')

  return <div>{title}</div>
}

export default MyComponent
```

### Best Practices

1. **Separation of Concerns**: Keep business logic in hooks/utils
2. **Type Safety**: Define interfaces for all data structures
3. **Error Handling**: Use try/catch with user-friendly messages
4. **Performance**: Use React.memo, useMemo, useCallback when needed
5. **Accessibility**: Follow WCAG guidelines

## Troubleshooting

### Port Already in Use

Change port in `vite.config.ts`:

```typescript
server: {
  port: 3001, // Change port
}
```

### API Connection Issues

1. Verify backend is running on http://localhost:8080
2. Check proxy configuration in `vite.config.ts`
3. Check browser network tab for CORS errors

### Build Errors

```bash
# Clear cache
rm -rf node_modules dist
npm install
npm run build
```

### Type Errors

```bash
# Run type checker
npm run type-check
```

## Next Steps

1. Implement authentication flow
2. Create topic management pages
3. Add broker monitoring dashboard
4. Implement consumer group views
5. Add real-time metrics with WebSocket
6. Write unit tests with Vitest
7. Add E2E tests with Playwright

## Resources

- [React Documentation](https://react.dev)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Vite Guide](https://vitejs.dev/guide/)
- [Ant Design Components](https://ant.design/components/overview/)
- [React Router](https://reactrouter.com/)
- [Axios Documentation](https://axios-http.com/docs/intro)

## License

Part of the Takhin project. See root LICENSE file.
