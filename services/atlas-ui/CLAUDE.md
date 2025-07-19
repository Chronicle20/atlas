# Atlas UI Frontend Architecture Guide

## Overview

This guide establishes architectural conventions for the Atlas UI project, a modern single-page application built with Next.js, TypeScript, Tailwind CSS, and shadcn/ui. All contributors must follow these standards to ensure consistency, maintainability, and scalability.

## 1. Project Structure

```
atlas-ui/
├── app/                      # Next.js app directory
│   ├── (auth)/              # Route groups for authentication
│   ├── (dashboard)/         # Main application routes
│   ├── api/                 # API routes
│   ├── layout.tsx           # Root layout
│   └── globals.css          # Global styles
├── components/              # Reusable UI components
│   ├── ui/                  # shadcn/ui base components
│   ├── common/              # Shared components
│   └── features/            # Feature-specific components
├── lib/                     # Utility functions and libraries
│   ├── api/                 # API client and utilities
│   ├── hooks/               # Custom React hooks
│   ├── utils/               # Helper functions
│   └── constants/           # App-wide constants
├── services/                # External service integrations
│   ├── api/                 # API service layer
│   └── auth/                # Authentication services
├── store/                   # Global state management
│   ├── slices/              # State slices (if using Redux/Zustand)
│   └── providers/           # React context providers
├── types/                   # TypeScript type definitions
│   ├── api/                 # API response types
│   ├── models/              # Domain models
│   └── components/          # Component prop types
├── styles/                  # Additional style files
├── tests/                   # Test files
│   ├── unit/                # Unit tests
│   ├── integration/         # Integration tests
│   └── e2e/                 # End-to-end tests
└── public/                  # Static assets
```

### Directory Purposes

- **app/**: Next.js App Router pages and layouts
- **components/**: All React components, organized by scope
- **lib/**: Pure utility functions, hooks, and helpers
- **services/**: Business logic and external integrations
- **store/**: Global state management
- **types/**: TypeScript interfaces and types
- **tests/**: All test files mirroring source structure

## 2. Component Conventions

### Component Types

#### Presentational Components
Located in `components/ui/` and `components/common/`
```tsx
// components/ui/Button.tsx
interface ButtonProps {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
}

export function Button({ variant = 'primary', size = 'md', children, ...props }: ButtonProps) {
  return <button className={cn(buttonVariants({ variant, size }))} {...props}>{children}</button>;
}
```

#### Container Components
Located in `components/features/`
```tsx
// components/features/UserProfile/UserProfileContainer.tsx
export function UserProfileContainer() {
  const { data, isLoading } = useUserProfile();
  
  if (isLoading) return <UserProfileSkeleton />;
  
  return <UserProfile user={data} />;
}
```

### Naming Conventions

- **Components**: PascalCase (e.g., `UserProfile.tsx`)
- **Hooks**: camelCase with 'use' prefix (e.g., `useUserProfile.ts`)
- **Utils**: camelCase (e.g., `formatDate.ts`)
- **Types**: PascalCase with descriptive suffixes (e.g., `UserProfileProps`, `ApiResponse`)

### Component Structure

```tsx
// 1. Imports
import { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';

// 2. Type definitions
interface ComponentProps {
  required: string;
  optional?: number;
  children?: React.ReactNode;
}

// 3. Component definition
export function Component({ required, optional = 0, children }: ComponentProps) {
  // 4. Hooks
  const [state, setState] = useState(false);
  
  // 5. Effects
  useEffect(() => {
    // Effect logic
  }, []);
  
  // 6. Event handlers
  const handleClick = () => {
    setState(!state);
  };
  
  // 7. Render
  return (
    <div className="component">
      {children}
    </div>
  );
}

// 8. Display name (if needed)
Component.displayName = 'Component';
```

## 3. State Management

### State Architecture

```
Local Component State → React Context → Global Store
     (useState)         (useContext)    (Zustand)
```

### Local State
Use for component-specific state that doesn't need sharing
```tsx
const [isOpen, setIsOpen] = useState(false);
```

### Context State
Use for feature-specific state shared across a component tree
```tsx
// store/providers/ThemeProvider.tsx
const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  
  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) throw new Error('useTheme must be used within ThemeProvider');
  return context;
};
```

### Global Store (Zustand)
Use for application-wide state
```tsx
// store/slices/userStore.ts
interface UserStore {
  user: User | null;
  setUser: (user: User | null) => void;
  logout: () => void;
}

export const useUserStore = create<UserStore>((set) => ({
  user: null,
  setUser: (user) => set({ user }),
  logout: () => set({ user: null }),
}));
```

## 4. Data Fetching & Caching

### API Client Setup
```tsx
// lib/api/client.ts
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL;

export const apiClient = {
  get: <T>(url: string, options?: RequestInit): Promise<T> => 
    fetch(`${API_BASE_URL}${url}`, { ...options, method: 'GET' }).then(handleResponse),
  post: <T>(url: string, data: unknown, options?: RequestInit): Promise<T> =>
    fetch(`${API_BASE_URL}${url}`, {
      ...options,
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...options?.headers },
      body: JSON.stringify(data),
    }).then(handleResponse),
};
```

### React Query Integration
```tsx
// lib/hooks/useUserProfile.ts
export function useUserProfile(userId: string) {
  return useQuery({
    queryKey: ['user', userId],
    queryFn: () => apiClient.get<User>(`/users/${userId}`),
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 3,
  });
}

// lib/hooks/useUpdateUser.ts
export function useUpdateUser() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: UpdateUserData) => 
      apiClient.put<User>(`/users/${data.id}`, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['user', data.id] });
    },
  });
}
```

### Error Handling
```tsx
// components/features/ErrorBoundary.tsx
export function QueryErrorBoundary({ children }: { children: React.ReactNode }) {
  return (
    <ErrorBoundary
      fallbackRender={({ error, resetErrorBoundary }) => (
        <ErrorDisplay error={error} onReset={resetErrorBoundary} />
      )}
    >
      {children}
    </ErrorBoundary>
  );
}
```

## 5. Styling

### Tailwind CSS Guidelines

```tsx
// ✅ Good - Semantic class grouping
<div className="flex items-center justify-between p-4 bg-white rounded-lg shadow-sm">

// ❌ Bad - Random order
<div className="shadow-sm p-4 flex bg-white justify-between rounded-lg items-center">

// Class order:
// 1. Layout (flex, grid)
// 2. Positioning (relative, absolute)
// 3. Box model (p, m, w, h)
// 4. Typography (text-, font-)
// 5. Visual (bg-, border-, shadow-)
// 6. Effects (transition, transform)
// 7. States (hover:, focus:)
```

### shadcn/ui Customization

```tsx
// components/ui/button.tsx - Extend shadcn/ui components
import { cva, type VariantProps } from 'class-variance-authority';

const buttonVariants = cva(
  'inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
        outline: 'border border-input bg-background hover:bg-accent',
        // Custom variants
        brand: 'bg-brand text-white hover:bg-brand/90',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-md px-3',
        lg: 'h-11 rounded-md px-8',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);
```

### Theme Management

```css
/* app/globals.css */
@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --brand: 217 91% 60%;
    /* ... other tokens */
  }
  
  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --brand: 217 91% 70%;
    /* ... other tokens */
  }
}
```

## 6. Routing

### Route Structure

```
app/
├── (auth)/
│   ├── login/page.tsx
│   ├── register/page.tsx
│   └── layout.tsx
├── (dashboard)/
│   ├── page.tsx              # Dashboard home
│   ├── projects/
│   │   ├── page.tsx          # Projects list
│   │   └── [id]/
│   │       ├── page.tsx      # Project detail
│   │       └── settings/page.tsx
│   └── layout.tsx
└── api/
    └── [...]/route.ts
```

### Route Guards

```tsx
// app/(dashboard)/layout.tsx
export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await getServerSession();
  
  if (!session) {
    redirect('/login');
  }
  
  return <DashboardShell>{children}</DashboardShell>;
}
```

### Dynamic Routes

```tsx
// app/(dashboard)/projects/[id]/page.tsx
interface PageProps {
  params: { id: string };
  searchParams: { [key: string]: string | string[] | undefined };
}

export default function ProjectPage({ params, searchParams }: PageProps) {
  // Use params.id for data fetching
  return <ProjectDetail projectId={params.id} />;
}
```

## 7. Testing

### Test Structure

```
tests/
├── unit/
│   ├── components/
│   │   └── Button.test.tsx
│   └── utils/
│       └── formatDate.test.ts
├── integration/
│   └── UserFlow.test.tsx
└── e2e/
    └── auth.spec.ts
```

### Unit Testing

```tsx
// tests/unit/components/Button.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from '@/components/ui/Button';

describe('Button', () => {
  it('renders with correct text', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });
  
  it('calls onClick when clicked', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click me</Button>);
    
    fireEvent.click(screen.getByText('Click me'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
});
```

### Integration Testing

```tsx
// tests/integration/UserProfile.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { UserProfile } from '@/components/features/UserProfile';

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('UserProfile Integration', () => {
  it('loads and displays user data', async () => {
    render(<UserProfile userId="123" />, { wrapper: createWrapper() });
    
    await waitFor(() => {
      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });
  });
});
```

## 8. Accessibility & Performance

### Accessibility Practices

```tsx
// ✅ Good
<button
  aria-label="Delete item"
  aria-pressed={isSelected}
  onClick={handleDelete}
>
  <TrashIcon aria-hidden="true" />
</button>

// Components should include:
// - Semantic HTML elements
// - ARIA labels for interactive elements
// - Keyboard navigation support
// - Focus management
// - Screen reader announcements
```

### Performance Optimization

```tsx
// Dynamic imports
const HeavyComponent = dynamic(() => import('./HeavyComponent'), {
  loading: () => <Skeleton />,
  ssr: false,
});

// Memoization
const ExpensiveComponent = memo(({ data }: Props) => {
  const processedData = useMemo(() => processData(data), [data]);
  
  const handleClick = useCallback((id: string) => {
    // Handle click
  }, []);
  
  return <div>{/* Render */}</div>;
});

// Image optimization
import Image from 'next/image';

<Image
  src="/hero.jpg"
  alt="Hero image"
  width={1200}
  height={600}
  priority
  placeholder="blur"
/>
```

## 9. Code Quality

### ESLint Configuration

```json
{
  "extends": [
    "next/core-web-vitals",
    "plugin:@typescript-eslint/recommended",
    "plugin:tailwindcss/recommended"
  ],
  "rules": {
    "@typescript-eslint/no-unused-vars": ["error", { "argsIgnorePattern": "^_" }],
    "tailwindcss/classnames-order": "warn",
    "tailwindcss/no-custom-classname": "warn"
  }
}
```

### Prettier Configuration

```json
{
  "semi": true,
  "singleQuote": true,
  "tabWidth": 2,
  "trailingComma": "es5",
  "printWidth": 100,
  "plugins": ["prettier-plugin-tailwindcss"]
}
```

### Commit Convention

```
feat: add user profile component
fix: resolve navigation issue in mobile view
docs: update component documentation
style: format code with prettier
refactor: extract shared logic to custom hook
test: add unit tests for Button component
chore: update dependencies
```

### Pre-commit Hooks

```json
// .husky/pre-commit
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

npm run lint-staged
```

## 10. Extensibility

### Component Design Principles

```tsx
// 1. Composition over configuration
<Card>
  <CardHeader>
    <CardTitle>Title</CardTitle>
    <CardDescription>Description</CardDescription>
  </CardHeader>
  <CardContent>Content</CardContent>
</Card>

// 2. Polymorphic components
interface ButtonProps<T extends React.ElementType> {
  as?: T;
  children: React.ReactNode;
}

function Button<T extends React.ElementType = 'button'>({
  as,
  children,
  ...props
}: ButtonProps<T> & React.ComponentPropsWithoutRef<T>) {
  const Component = as || 'button';
  return <Component {...props}>{children}</Component>;
}

// 3. Render props for flexibility
<DataTable
  data={users}
  columns={columns}
  renderCell={(item, column) => <CustomCell item={item} column={column} />}
/>
```

### Feature Module Pattern

```tsx
// features/analytics/
├── components/
│   ├── AnalyticsDashboard.tsx
│   └── MetricCard.tsx
├── hooks/
│   └── useAnalytics.ts
├── services/
│   └── analytics.service.ts
├── types/
│   └── analytics.types.ts
└── index.ts  // Public API
```

### Plugin System

```tsx
// lib/plugins/registry.ts
interface Plugin {
  name: string;
  routes?: RouteConfig[];
  components?: ComponentConfig[];
  hooks?: HookConfig[];
}

export class PluginRegistry {
  private plugins: Map<string, Plugin> = new Map();
  
  register(plugin: Plugin) {
    this.plugins.set(plugin.name, plugin);
  }
  
  getRoutes(): RouteConfig[] {
    return Array.from(this.plugins.values())
      .flatMap(p => p.routes || []);
  }
}
```

## Best Practices Summary

1. **Keep components small and focused** - Single responsibility principle
2. **Use TypeScript strictly** - Enable strict mode and avoid `any`
3. **Optimize bundle size** - Use dynamic imports and tree shaking
4. **Test critical paths** - Focus on user flows and business logic
5. **Document complex logic** - Use JSDoc for non-obvious implementations
6. **Maintain consistent patterns** - Follow established conventions
7. **Consider accessibility first** - Build inclusive interfaces
8. **Monitor performance** - Use Next.js analytics and Web Vitals
9. **Review dependencies** - Audit and update regularly
10. **Iterate on architecture** - Refactor as requirements evolve

This architecture is designed to scale with your team and product. Regular reviews and updates to these guidelines ensure they remain relevant and helpful.