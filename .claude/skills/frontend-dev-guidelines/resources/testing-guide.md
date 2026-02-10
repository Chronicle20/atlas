# Testing Guide

## Overview

Atlas UI uses **Jest 30** with **React Testing Library** and **jsdom** environment. Tests live alongside source files in `__tests__/` directories or as `*.test.{ts,tsx}` files.

## Test Configuration

```javascript
// jest.config.js
{
  testEnvironment: 'jsdom',
  testMatch: ['**/*.test.{js,jsx,ts,tsx}', '**/__tests__/**'],
  moduleNameMapper: { '@/*': '<rootDir>/*' },
  collectCoverageFrom: ['components/**', 'lib/**'],
}
```

## Test Structure

```
components/
├── common/
│   └── __tests__/
│       └── ErrorPage.test.tsx
├── features/
│   └── tenants/
│       └── __tests__/
│           └── CreateTenantDialog.test.tsx
lib/
├── __tests__/
│   └── query-client.test.ts
├── api/
│   └── __tests__/
│       ├── cache.test.ts
│       ├── errors.test.ts
│       └── retry.test.ts
└── hooks/
    └── __tests__/
        └── useCharacterImage.test.ts
```

## Unit Test Pattern (Pure Functions/Config)

```typescript
// lib/__tests__/query-client.test.ts
import { createQueryClient } from '../query-client';

describe('Query Client Configuration', () => {
  let queryClient: ReturnType<typeof createQueryClient>;

  beforeEach(() => {
    queryClient = createQueryClient();
  });

  afterEach(() => {
    queryClient.clear();
  });

  it('should have correct query default options', () => {
    const defaultOptions = queryClient.getDefaultOptions();
    expect(defaultOptions.queries?.staleTime).toBe(5 * 60 * 1000);
    expect(defaultOptions.queries?.retry).toBe(3);
    expect(defaultOptions.queries?.refetchOnWindowFocus).toBe(false);
  });

  it('should have exponential backoff retry delay', () => {
    const retryDelay = defaultOptions.queries?.retryDelay as (i: number) => number;
    expect(retryDelay(0)).toBe(1000);
    expect(retryDelay(1)).toBe(2000);
    expect(retryDelay(2)).toBe(4000);
    expect(retryDelay(10)).toBe(30000);  // Max cap
  });
});
```

## Component Test Pattern

```typescript
// components/common/__tests__/ErrorPage.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { ErrorPage } from '../ErrorPage';

// Mock Next.js modules
jest.mock('next/link', () => {
  const MockLink = ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  );
  MockLink.displayName = 'MockLink';
  return MockLink;
});

describe('ErrorPage', () => {
  it('renders with correct status code', () => {
    render(<ErrorPage statusCode={404} />);
    expect(screen.getByText('Page Not Found')).toBeInTheDocument();
    expect(screen.getByText('Error Code: 404')).toBeInTheDocument();
  });

  it('calls onRetry when retry button clicked', () => {
    const mockRetry = jest.fn();
    render(<ErrorPage statusCode={500} showRetryButton onRetry={mockRetry} />);
    fireEvent.click(screen.getByRole('button', { name: /try again/i }));
    expect(mockRetry).toHaveBeenCalledTimes(1);
  });

  it('renders home link', () => {
    render(<ErrorPage statusCode={404} />);
    const homeLink = screen.getByRole('link', { name: /go home/i });
    expect(homeLink).toHaveAttribute('href', '/');
  });
});
```

## Integration Test Pattern (With Service Mocks)

```typescript
// components/features/tenants/__tests__/CreateTenantDialog.test.tsx
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateTenantDialog } from '../CreateTenantDialog';

// Mock entire service module
jest.mock('@/services/api', () => ({
  templatesService: {
    getTemplateOptions: jest.fn(),
  },
  onboardingService: {
    onboardTenantByVersion: jest.fn(),
  },
}));

jest.mock('sonner', () => ({
  toast: { success: jest.fn(), error: jest.fn() },
}));

describe('CreateTenantDialog', () => {
  const mockOptions = [
    { id: 't1', attributes: { region: 'GMS', majorVersion: 83, minorVersion: 1 } },
    { id: 't2', attributes: { region: 'JMS', majorVersion: 185, minorVersion: 1 } },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    templatesService.getTemplateOptions.mockResolvedValue(mockOptions);
  });

  it('fetches options when dialog opens', async () => {
    render(<CreateTenantDialog open onOpenChange={jest.fn()} />);
    await waitFor(() => {
      expect(templatesService.getTemplateOptions).toHaveBeenCalled();
    });
  });

  it('shows error when options fail to load', async () => {
    templatesService.getTemplateOptions.mockRejectedValue(new Error('Network error'));
    render(<CreateTenantDialog open onOpenChange={jest.fn()} />);
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });

  it('allows typing in name field', async () => {
    render(<CreateTenantDialog open onOpenChange={jest.fn()} />);
    await waitFor(() => expect(templatesService.getTemplateOptions).toHaveBeenCalled());
    const input = screen.getByPlaceholderText(/enter tenant name/i);
    await userEvent.type(input, 'Test Tenant');
    expect(input).toHaveValue('Test Tenant');
  });

  it('does not render when closed', () => {
    render(<CreateTenantDialog open={false} onOpenChange={jest.fn()} />);
    expect(screen.queryByText('Create New Tenant')).not.toBeInTheDocument();
  });
});
```

## Common Mocks

### Next.js Link
```typescript
jest.mock('next/link', () => {
  const MockLink = ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  );
  MockLink.displayName = 'MockLink';
  return MockLink;
});
```

### Next.js Router
```typescript
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), back: jest.fn() }),
  useParams: () => ({ id: '123' }),
  usePathname: () => '/test',
}));
```

### Toast (sonner)
```typescript
jest.mock('sonner', () => ({
  toast: { success: jest.fn(), error: jest.fn(), warning: jest.fn() },
}));
```

### Service Modules
```typescript
jest.mock('@/services/api', () => ({
  bansService: {
    getAllBans: jest.fn(),
    createBan: jest.fn(),
    deleteBan: jest.fn(),
  },
}));
```

### React Query Wrapper
```typescript
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

// Usage
render(<Component />, { wrapper: createWrapper() });
```

## Testing Rules

1. **Always mock external dependencies** — services, toast, next/link, next/navigation
2. **Use `waitFor` for async operations** — service calls, state updates
3. **Query by role/text, not implementation** — `getByRole('button', { name: /submit/i })`
4. **Use `userEvent` for user interactions** — More realistic than `fireEvent`
5. **Clear mocks between tests** — `jest.clearAllMocks()` in `beforeEach`
6. **Test loading, error, and success states** — All three paths
7. **Test dialog open/close behavior** — Controlled components should respect `open` prop
8. **Verify accessibility** — Use `getByRole`, check `aria-label`s

## Running Tests

```bash
# Run all tests
npm test

# Watch mode (development)
npm run test:watch

# With coverage report
npm run test:coverage
```

## Pre-Commit Checklist

- [ ] All tests pass: `npm test`
- [ ] No new TypeScript errors: `npx tsc --noEmit`
- [ ] Lint passes: `npm run lint`
- [ ] Build succeeds: `npm run build`
