# Atlas UI Frontend Architecture Guide

## Overview

atlas-ui is a Vite + React Router single-page application built on the home-hub template. The stack is:

- **Build**: Vite 8 + `@vitejs/plugin-react` + `@tailwindcss/vite` (no PostCSS config)
- **Routing**: `react-router-dom` v7 with a top-level `<BrowserRouter>` + nested `<Routes>`
- **Data**: TanStack React Query 5 (one `QueryProvider` at the provider root)
- **State**: Context-based only — `TenantProvider` (tenant selection + cache invalidation) and `ThemeProvider` (light/dark)
- **Forms**: `react-hook-form` + `zod`
- **UI**: shadcn/ui components under `src/components/ui/`, Tailwind 4
- **Tests**: Vitest + `@testing-library/react` + `jsdom`, setup in `src/test/setup.ts`
- **Runtime**: production is nginx:alpine serving the Vite `dist/` output with a `try_files $uri $uri/ /index.html` SPA fallback

There is no authentication, no SSR, no App Router, no route groups, no server actions, no `next/image` pipeline. Previous iterations of this guide documented features that never existed; if you see references to `next-auth`, `getServerSession`, or route guards in older notes, they're wrong.

## Project structure

```
services/atlas-ui/
├── index.html                      # Vite entry, loads /src/main.tsx
├── vite.config.ts                  # @/ alias, dev proxy, Vitest config
├── tsconfig.json                   # Project references to app + node
├── tsconfig.app.json               # App compilation (src/, excludes __tests__)
├── tsconfig.node.json              # vite.config.ts only
├── eslint.config.js                # Flat config (js + typescript-eslint + react-hooks + react-refresh)
├── Dockerfile                      # Two-stage: node:24-alpine builder → nginx:alpine runtime
├── nginx.conf                      # SPA fallback — included at /etc/nginx/conf.d/default.conf
├── package.json                    # "type": "module", scripts: dev/build/preview/lint/test
├── public/                         # Static assets (logo.png, favicon.ico, sw-character-cache.js)
└── src/
    ├── main.tsx                    # createRoot + <App />
    ├── App.tsx                     # Provider stack + <Routes> (all 46 routes listed here)
    ├── index.css                   # Tailwind 4 + theme tokens
    ├── vite-env.d.ts               # import.meta.env.VITE_* declarations
    ├── components/
    │   ├── providers/              # QueryProvider, ThemeProvider
    │   ├── common/                 # error-boundary, not-found-page, skeletons, form helpers
    │   ├── ui/                     # shadcn base components
    │   └── features/               # Domain components (bans, characters, npc, quests, ...)
    ├── context/                    # TenantProvider + useTenant
    ├── hooks/                      # Project-wide hooks (use-mobile)
    ├── lib/
    │   ├── api/                    # client.ts + errors + headers
    │   ├── breadcrumbs/
    │   ├── hooks/                  # React Query hooks under api/, plus data-derivation hooks
    │   ├── schemas/                # Zod schemas
    │   └── utils/
    ├── pages/                      # 46 route components as named exports, colocated columns/forms
    ├── services/
    │   ├── api/                    # *.service.ts modules (thin adapters over lib/api/client)
    │   └── errorLogger.ts
    ├── test/setup.ts               # Vitest global setup (jest-dom matchers + matchMedia stub)
    └── types/                      # models/, api/, components/
```

Pages follow PascalCase: `DashboardPage.tsx`, `AccountDetailPage.tsx`, `NpcConversationPage.tsx`. Colocated columns/forms use kebab-case: `accounts-columns.tsx`, `tenants-handlers-form.tsx`.

## Provider stack (App.tsx)

Order matters. `TenantProvider` uses `useQueryClient`, so it must live inside `QueryProvider`.

```tsx
<BrowserRouter>
  <QueryProvider>
    <ThemeProvider>
      <TenantProvider>
        <Toaster />
        <RouteErrorBoundary>
          <Routes>
            <Route element={<AppShell />}>
              <Route index element={<DashboardPage />} />
              …44 more routes…
            </Route>
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </RouteErrorBoundary>
      </TenantProvider>
    </ThemeProvider>
  </QueryProvider>
</BrowserRouter>
```

`AppShell` wraps the shared sidebar + header + breadcrumb chrome around `<Outlet />`. Non-shell routes (`*` 404) render outside the shell.

## Tenant contract

Four headers, SCREAMING_SNAKE_CASE, set by `src/lib/headers.tsx#tenantHeaders` and consumed verbatim by Go services — do **not** rename these:

- `TENANT_ID` — tenant UUID
- `REGION` — tenant region string
- `MAJOR_VERSION` — integer as string
- `MINOR_VERSION` — integer as string

Tenant wiring is centralised in `TenantProvider`. A single effect fires whenever the active tenant changes:

1. `api.setTenant(activeTenant)` — pushes the tenant into the API client so every subsequent request carries the four headers.
2. `queryClient.clear()` — invalidates all React Query caches so one tenant can't see another tenant's data.

The effect is skipped on initial mount while `activeTenant === null`. Covered by `src/context/__tests__/tenant-context.test.tsx`.

There is still redundant per-call `api.setTenant(tenant)` code in several service modules (legacy from the Next.js era). They're harmless duplicates; a follow-up PR will remove them (see `docs/TODO.md` → atlas-ui Frontend → Phase 2 deferrals).

## Data fetching

React Query is the one source of truth for server state. Hooks live at `src/lib/hooks/api/use<Resource>.ts`:

```tsx
export function useAccounts(options?: QueryOptions) {
  return useQuery({
    queryKey: accountsKeys.list(options),
    queryFn: () => accountsService.getAll(options),
    ...sharedOptions,
  });
}
```

Write operations invalidate related queries via `onSuccess`:

```tsx
export function useCreateAccount() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: accountsService.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: accountsKeys.all }),
  });
}
```

Not every page is migrated yet — a number still use `useState` + `useEffect` + direct service calls. See `docs/TODO.md` → Phase 4 deferral for the consolidation backlog.

## API client

`src/lib/api/client.ts` exports a singleton `apiClient` (class) and a thin `api` object with `get`/`getList`/`getOne`/`post`/`put`/`patch`/`delete`/`upload`/`download`/`setTenant`. Service modules under `src/services/api/` compose these primitives — they don't reach into `apiClient` internals.

Environment variables are resolved via `import.meta.env.VITE_*`:

- `VITE_ROOT_API_URL` — base URL for API requests. Default in local dev proxies `/api/**` to `http://localhost:${VITE_INGRESS_PORT:-8080}` via `vite.config.ts#server.proxy`.
- `VITE_BUILD_VERSION`, `VITE_ERROR_ENDPOINT`, `VITE_ERROR_API_KEY` — error logger config.
- `VITE_ASSET_BASE_URL` — override for `/api/assets` prefix used by `getAssetIconUrl`.

## Images

Plain `<img>` at each call site with explicit `width`/`height` and `loading="lazy"` (below the fold). No `next/image`, no wrapper component. For sprites from `maplestory.io`, the browser handles caching. If sprite-version pinning or CDN layering becomes a real need, add a wrapper then — don't add one speculatively.

## Theme

`ThemeProvider` at `src/components/providers/theme-provider.tsx` stores `"light" | "dark"` in localStorage and toggles the `<html>` class. The "system" option from `next-themes` was intentionally dropped — revisit only if users miss it.

## Running locally

```bash
cd services/atlas-ui
npm install
npm run dev          # Vite dev server on :5173, /api → localhost:8080 (atlas-ingress)
npm run test         # Vitest (run mode)
npm run test:watch   # Vitest watch mode
npm run lint
npm run build        # tsc -b + vite build → dist/
npm run preview      # Serve dist/ for smoke testing the production build
```

The compose runtime publishes host port 3000 mapped to the container's nginx port 80 (`3000:80` in `deploy/compose/docker-compose.core.yml`) so local `http://localhost:3000` still hits atlas-ui the way it used to.

## Production runtime

`Dockerfile` builds in two stages:

1. `node:24-alpine` installs deps with `npm ci`, copies source, runs `npm run build` → `/build/dist`.
2. `nginx:alpine` copies `dist/` into `/usr/share/nginx/html` and `nginx.conf` into `/etc/nginx/conf.d/default.conf`. Container listens on port 80. SPA fallback sends unknown paths to `index.html`.

Deploy surface (four files):

- `deploy/k8s/atlas-ui.yaml` — Deployment `containerPort: 80`, Service `port: 80`, env `VITE_ROOT_API_URL`
- `deploy/k8s/ingress.yaml` — catch-all `location /` proxies to `atlas-ui:80` (Next.js HMR block removed)
- `deploy/shared/routes.conf` — same catch-all → `atlas-ui:80`
- `deploy/compose/docker-compose.core.yml` — port mapping `3000:80`, env `VITE_ROOT_API_URL`

## Testing

Vitest with `jsdom`, `@testing-library/react`, and global test APIs (`describe`/`it`/`expect`/`vi`). Setup lives in `src/test/setup.ts`. Tests colocated under `__tests__/` directories next to the code they cover.

Most existing test files (61 files) are still Jest-era — they use `jest.fn` / `jest.mock` rather than `vi.*`. They're excluded from `tsc -b` for now; Phase 5 in `docs/TODO.md` → atlas-ui Frontend has the migration backlog.

## Conventions

- **Named exports on pages** — App.tsx imports them by name, no default exports.
- **No `"use client"`, no `next/*` imports** — both were purged during the migration. `grep -rn "use client" services/atlas-ui/src/` and `grep -rn "from ['\"]next/" services/atlas-ui/src/` both return zero.
- **`@/` alias** — resolves to `./src` in both Vite and TypeScript. Use it consistently; don't mix with relative paths past one level.
- **`import.meta.env.VITE_*`** — never `process.env.*`. The Vite build evaluates env at bundle time.
- **Keep strict TS flags where they're on** — `strict` + `noFallthroughCasesInSwitch`. Several of the home-hub stricter flags (`verbatimModuleSyntax`, `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noUnusedLocals`, `noUnusedParameters`, `erasableSyntaxOnly`) are off pending a follow-up; don't reintroduce patterns that exploit that.
