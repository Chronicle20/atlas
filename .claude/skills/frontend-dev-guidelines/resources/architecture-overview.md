# Architecture Overview

## Tech Stack

| Technology | Version | Purpose |
|-----------|---------|---------|
| Next.js | 16 | App Router, SSR, routing |
| React | 19 | UI framework |
| TypeScript | strict | Type safety |
| Tailwind CSS | 4 | Utility-first styling |
| shadcn/ui | 3.8 | Radix-based component library |
| TanStack React Query | 5 | Server state management |
| TanStack React Table | 8 | Data table rendering |
| react-hook-form | 7 | Form state management |
| Zod | 4 | Schema validation |
| sonner | 2 | Toast notifications |
| Lucide React | - | Icon library |
| next-themes | - | Dark/light mode |
| tailwind-nord | - | Nordic color palette |

## Project Structure

```
services/atlas-ui/
├── app/                    # Next.js App Router (pages + layouts)
├── components/
│   ├── ui/                 # shadcn/ui base primitives
│   ├── common/             # Shared presentational components
│   ├── features/           # Feature-specific containers
│   └── providers/          # React context provider wrappers
├── lib/
│   ├── api/                # API client + error utilities
│   ├── hooks/api/          # React Query hooks (per-resource)
│   ├── schemas/            # Zod validation schemas
│   ├── breadcrumbs/        # Breadcrumb navigation system
│   └── utils.ts            # cn() classname helper
├── services/api/           # Service classes (BaseService + concrete)
├── types/
│   ├── models/             # Domain model interfaces
│   └── api/                # API response/error types
├── context/                # React context definitions
└── tests/                  # Test files
```

## Architectural Layers

```
┌─────────────────────────────────────────────┐
│  App Router Pages (app/*/page.tsx)           │  ← Data fetching, composition
├─────────────────────────────────────────────┤
│  Feature Components (components/features/)   │  ← Business UI logic
├─────────────────────────────────────────────┤
│  React Query Hooks (lib/hooks/api/)          │  ← Server state, cache
├─────────────────────────────────────────────┤
│  Service Layer (services/api/)               │  ← API abstraction
├─────────────────────────────────────────────┤
│  API Client (lib/api/client.ts)              │  ← HTTP, retry, dedup
├─────────────────────────────────────────────┤
│  Backend Microservices                       │  ← Go services via ingress
└─────────────────────────────────────────────┘
```

**Data flows top-down.** Pages compose feature components, which use hooks, which call services, which use the API client. Never skip layers (e.g., components should not call `api.get()` directly).

## Provider Hierarchy (Root Layout)

```tsx
// app/layout.tsx
<TenantProvider>
  <QueryProvider>
    <ThemeProvider>
      <SidebarProvider>
        <AppSidebar />
        <main>
          <BreadcrumbBar />
          {children}
        </main>
      </SidebarProvider>
    </ThemeProvider>
  </QueryProvider>
</TenantProvider>
```

**Order matters:**
1. `TenantProvider` — outermost, so all children can access tenant context
2. `QueryProvider` — wraps React Query client for data fetching
3. `ThemeProvider` — dark/light mode via `next-themes`
4. `SidebarProvider` — shadcn/ui sidebar state

## Key Configuration

### TypeScript (tsconfig.json)
- `strict: true` — all strict checks enabled
- `noUncheckedIndexedAccess: true` — safe index access
- `exactOptionalPropertyTypes: true` — precise optional types
- `noImplicitOverride: true` — explicit override keyword
- Path alias: `@/*` maps to project root

### React Query (lib/query-client.ts)
- Stale time: 5 minutes default
- GC time: 10 minutes default
- Retry: 3 attempts with exponential backoff (max 30s)
- `refetchOnWindowFocus: false`
- Mutations: 1 retry, 1s delay

### Next.js (next.config.ts)
- Image optimization from `maplestory.io`
- Container-aware configuration (Docker/K8s)
- Turbopack bundler
- Image formats: WebP, AVIF
