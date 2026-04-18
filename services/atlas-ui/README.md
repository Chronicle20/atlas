# Atlas UI

Administrative SPA for the Atlas MapleStory server. Built on Vite 8 + React 19 + `react-router-dom` v7 + TanStack React Query + shadcn/ui + Tailwind 4. Served in production by nginx.

## Stack

- **Build**: Vite 8 + `@vitejs/plugin-react` + `@tailwindcss/vite`
- **Routing**: `react-router-dom` v7, one `<BrowserRouter>` wrapping a shared `AppShell`
- **Data**: TanStack React Query 5
- **Forms**: `react-hook-form` + `zod`
- **Testing**: Vitest + `@testing-library/react` + `jsdom`
- **Runtime**: nginx:alpine serving the `dist/` output with a SPA fallback

## Prerequisites

- Node.js 24 (matches the Docker builder)
- npm 10+

## Commands

```bash
npm install          # Install dependencies
npm run dev          # Vite dev server on http://localhost:5173
npm run build        # tsc -b && vite build → dist/
npm run preview      # Serve dist/ locally (smoke test the prod bundle)
npm run lint         # ESLint flat config
npm run test         # Vitest (run mode)
npm run test:watch   # Vitest watch mode
npm run test:coverage
```

Dev server proxies `/api` to `http://localhost:${VITE_INGRESS_PORT:-8080}` (the compose-local `atlas-ingress` nginx). Override `VITE_INGRESS_PORT` if you're pointed at a remote stack.

## Docker

Two-stage build. The repository root is the build context (matches
`deploy/compose/docker-compose.core.yml`):

```bash
# From the atlas repo root:
docker build -f services/atlas-ui/Dockerfile -t atlas-ui:local .
docker run --rm -p 3000:80 atlas-ui:local
```

Container listens on port 80; the compose mapping preserves host port 3000 for local-dev compatibility (`3000:80`).

## Environment variables

All env vars use the `VITE_` prefix and are read via `import.meta.env.VITE_*`:

| Variable | Purpose |
|---|---|
| `VITE_ROOT_API_URL` | Base URL for API requests (falls back to `window.location.origin`) |
| `VITE_BUILD_VERSION` | Surfaced in error reports |
| `VITE_ERROR_ENDPOINT` | Remote error logging endpoint |
| `VITE_ERROR_API_KEY` | Auth key for remote error logging |
| `VITE_ASSET_BASE_URL` | Override for `/api/assets` prefix |
| `VITE_INGRESS_PORT` | Dev-only — port the Vite proxy targets |

## Tenant contract

Every API request injects four headers (SCREAMING_SNAKE_CASE, consumed verbatim by Go services):

- `TENANT_ID`
- `REGION`
- `MAJOR_VERSION`
- `MINOR_VERSION`

Tenant selection lives in `src/context/tenant-context.tsx`. A `useEffect` there calls `api.setTenant(activeTenant)` and `queryClient.clear()` on every tenant change (skipped on the initial null mount).

## Architecture

See [`CLAUDE.md`](./CLAUDE.md) for the full architectural guide — directory layout, provider stack, data-fetching patterns, image policy, testing conventions, and deploy surface.

Historical Next.js-era notes under `docs/service-layer.md` and `docs/error-handling.md` still reference `next/image`, `NEXT_PUBLIC_*`, and the App Router. They're scheduled for rewrite in `docs/TODO.md` → atlas-ui Frontend → Phase 7 deferrals.

## Deferrals

The Vite + React Router migration (task-004) merged with a focused feature-parity scope. A full list of held-back work lives in the repo root at `docs/TODO.md` under `### atlas-ui Frontend`:

- Phase 2: shrink `lib/api/client.ts` below 700 LOC, delete `services/api/base.service.ts`, remove per-call `api.setTenant` duplicates.
- Phase 3: route-level `React.lazy` splitting, `useSearchParams` semantics audit on filter-heavy pages.
- Phase 4: replace remaining `useState` + `useEffect` + service-call patterns with React Query hooks.
- Phase 5: migrate 61 Jest test files to Vitest; re-enable stricter tsconfig flags once tests compile.
- Phase 7: rewrite `docs/service-layer.md` and `docs/error-handling.md`.
