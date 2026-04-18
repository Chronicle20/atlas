# atlas-ui Vite + React Router Migration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-17
---

## 1. Overview

`services/atlas-ui` is currently a Next.js 16 App Router application that ships with `"use client"` on every one of its 46 pages, uses zero server components, zero server actions, zero route handlers, and no middleware. It is, in effect, a single-page application wrapped in the Next.js framework — paying the complexity cost (Turbopack config, `next.config.ts` image-loader fallback for container environments, App Router file conventions, dual `next/navigation` + `next/link` APIs, a 1801-line bespoke `lib/api/client.ts`) without collecting any of the framework's benefits.

The companion project `home-hub` at `/home/tumidanski/source/home-hub/frontend/` is a comparable internal admin dashboard (React 19, TypeScript, shadcn/ui, Tailwind 4, React Query, react-hook-form + Zod, multi-tenant context) built on **Vite 8 + react-router-dom 7 + Vitest**. It is simpler, smaller, and proves the template is sufficient for this class of application.

This task rebuilds atlas-ui on the home-hub template in a single big-bang migration. Feature parity is the sole correctness bar. No backend services are touched. No UX changes. The goal is to end with an atlas-ui that is behaviourally identical to today's build but shorter, simpler, and using its tools idiomatically.

## 2. Goals

Primary goals:
- Replace Next.js 16 with Vite 8 as the build tool, dev server, and test runner host.
- Replace the Next.js App Router with `react-router-dom` v7, using a single nested layout route (`AppShell` + `<Outlet />`) mirroring home-hub.
- Remove all `"use client"` directives (140+ across pages and components).
- Replace `next/link`, `next/navigation` (`useRouter`, `useParams`, `useSearchParams`, `usePathname`), and `next/image` with `react-router-dom` and plain HTML equivalents.
- Consolidate the dual data-fetch layer (`lib/hooks/api/` + `services/api/` + pages doing raw `useEffect` fetches) into a single React Query hook-per-resource pattern used uniformly by every page.
- Shrink `lib/api/client.ts` (currently 1801 LOC) and delete `services/api/base.service.ts` (499 LOC), porting or adapting home-hub's ~572-line client. Keep only primitives actually used by atlas-ui pages.
- Migrate Jest tests to Vitest.
- Replace `eslint-config-next` with home-hub's ESLint config (`@eslint/js` + `typescript-eslint` + `eslint-plugin-react-hooks` + `eslint-plugin-react-refresh`).
- Replace the Next.js container runtime with nginx + static `dist/`, mirroring home-hub's Dockerfile.
- Update `services/atlas-ui/CLAUDE.md` to describe the new architecture accurately (today's version documents patterns that were never implemented).

Non-goals:
- Any backend / Go service changes.
- Adding auth, session guards, or authorization logic (the current CLAUDE.md describes `getServerSession` and `(auth)/` / `(dashboard)/` route groups that **do not exist** — this migration does not add them).
- Redesigning pages, components, or UX.
- Introducing new state management (no Zustand / Redux / Jotai); keep React Query + Context.
- Adding image optimization for `maplestory.io` sprites — drop Next's `next/image` optimizer and use plain `<img loading="lazy">` with browser caching.
- Code-splitting strategy work beyond Vite's defaults (route-level `React.lazy` can be considered in a follow-up).
- Introducing SSR, SSG, or streaming.
- Migrating the underlying UI library, theme tokens, shadcn component set, Tailwind config, or React Flow graph.

## 3. User Stories

- As an atlas-ui developer, I want one data-fetching pattern across the app so that I don't have to choose between `useEffect` + service call vs. React Query hook every time I build a page.
- As an atlas-ui developer, I want the `dev` command to start in under a second using Vite so that iteration feels the same as home-hub.
- As an atlas-ui developer, I want the tests and ESLint config to match home-hub so I can move between the two repos without mental context switching.
- As an operator deploying atlas-ui, I want a static nginx container (like home-hub) so that there is no Node runtime in production and image size shrinks accordingly.
- As a new contributor, I want the architecture guide (`services/atlas-ui/CLAUDE.md`) to match what's actually in the codebase so I can learn from it without being misled.

## 4. Functional Requirements

### 4.1 Build & tooling
- `package.json` must declare `"type": "module"` and scripts: `dev` (`vite`), `build` (`tsc -b && vite build`), `preview` (`vite preview`), `lint` (`eslint .`), `test` (`vitest run`), `test:watch` (`vitest`), `test:coverage` (`vitest run --coverage`).
- Remove dependencies: `next`, `eslint-config-next`, `next-themes`, `next-env.d.ts`, `tailwind-nord` (if unused elsewhere — verify), Jest (`jest`, `jest-environment-jsdom`, `@types/jest`).
- Add dependencies: `vite`, `@vitejs/plugin-react`, `@tailwindcss/vite`, `react-router-dom`, `vitest`, `jsdom`, `eslint-plugin-react-refresh`. Remove `@tailwindcss/postcss` and delete `postcss.config.mjs` — home-hub uses the Vite plugin and no PostCSS config. Tailwind version itself (`tailwindcss@^4.2.x`) is already in place and does not change.
- React (19.2.4), TypeScript (6.x), and core React Query / react-hook-form / Zod / shadcn / Tailwind versions are already aligned with home-hub. No version bumps in scope.
- Replace `next-themes` with home-hub's `ThemeProvider` (a thin `next-themes`-compatible wrapper exists at `frontend/src/components/providers/theme-provider.tsx`).
- `vite.config.ts` must configure `react()`, `tailwindcss()`, `"@"` → `./src` alias, and Vitest with `jsdom`, globals, a `./src/test/setup.ts` setup file, and `include: ["src/**/*.test.{ts,tsx}"]`.
- `index.html` at the repo root loads `/src/main.tsx` as a module.
- `tsconfig.json` + `tsconfig.app.json` + `tsconfig.node.json` split mirrors home-hub (`tsc -b` builds both).

### 4.2 Routing
- Single `BrowserRouter` at `src/App.tsx` with providers in this order (inside→out): `Toaster`, `Routes`, `TenantProvider`, `ThemeProvider`, `QueryProvider`, `BrowserRouter`.
- One top-level layout route using `AppShell` with `<Outlet />` that renders the existing sidebar (`AppSidebar`), header (`SidebarToggle`, `Separator`, `BreadcrumbBar`, `ThemeToggle`), and main content frame currently in `app/layout.tsx`.
- Every current page becomes a child route. Full route list:
  - `/` → `DashboardPage` (current `app/page.tsx`)
  - `/accounts`, `/accounts/:id`
  - `/bans`, `/bans/:banId`
  - `/characters`, `/characters/:id`
  - `/gachapons`, `/gachapons/:id`
  - `/guilds`, `/guilds/:id`
  - `/items`, `/items/:id`
  - `/login-history`
  - `/maps`, `/maps/:id`, `/maps/:id/portals/:portalId`
  - `/merchants`, `/merchants/:id`
  - `/monsters`, `/monsters/:id`
  - `/npcs`, `/npcs/:id`, `/npcs/:id/conversations`, `/npcs/:id/shop`
  - `/quests`, `/quests/:id`
  - `/reactors`, `/reactors/:id`
  - `/services`, `/services/:id`
  - `/setup`
  - `/templates`, `/templates/:id`
  - `/tenants`, `/tenants/:id`
  - `*` → 404 page (port `app/not-found.tsx`)
- Error boundaries: port `app/error.tsx` and `app/global-error.tsx` to a React error boundary component mounted above `<Routes>` (pattern: `react-error-boundary` or a bespoke class component). Route-level `loading.tsx` (currently only `app/bans/loading.tsx`) becomes a `<Suspense>` wrapper or inline skeleton inside the page.

### 4.3 Navigation API replacements
Every occurrence must be migrated:
- `import Link from "next/link"` → `import { Link } from "react-router-dom"`. Replace `href` prop with `to`.
- `import { useRouter } from "next/navigation"` → `import { useNavigate } from "react-router-dom"`. Replace `router.push(path)` → `navigate(path)`, `router.replace(path)` → `navigate(path, { replace: true })`, `router.back()` → `navigate(-1)`.
- `import { useParams } from "next/navigation"` → `import { useParams } from "react-router-dom"`. Note that `useParams()` returns typed strings in RR v7; no `Array.isArray` check needed.
- `import { useSearchParams } from "next/navigation"` → `import { useSearchParams } from "react-router-dom"`. API: returns `[URLSearchParams, setSearchParams]` (not a plain `URLSearchParams` as in Next) — update call sites accordingly.
- `import { usePathname } from "next/navigation"` → `import { useLocation } from "react-router-dom"` and read `location.pathname`. Update `BreadcrumbBar` consumer.
- `import { redirect, notFound } from "next/navigation"` → navigate programmatically in an effect, or render `<Navigate to="..." replace />` / the 404 component.

### 4.4 Image replacements
- Replace `import Image from "next/image"` with plain `<img>` elements. Remove `width`, `height`, `priority`, `placeholder`, `blurDataURL` props (keep native `width`/`height` attributes where dimensions are known).
- Add `loading="lazy"` to below-the-fold images by default.
- Delete `lib/image-loader.ts`.
- For `maplestory.io` character sprite usage (currently the main `next/image` consumer), use plain `<img>` at each call site — no wrapper component. The existing character renderer already composes sprites through its own components; an additional `<MaplestoryImage>` abstraction would add no behaviour beyond what the browser provides. Revisit only if centralised sprite-version pinning or fallback handling becomes a concrete need.
- Remove `next/image` `remotePatterns`, `formats`, `deviceSizes`, `imageSizes`, `minimumCacheTTL`, `dangerouslyAllowSVG`, and `contentSecurityPolicy` config — all deleted with `next.config.ts`.

### 4.5 Data-fetch consolidation
- Every page reads data exclusively via a React Query hook. No page uses `useState` + `useEffect` + `fetch` / service call. 35 atlas-ui pages currently violate this — all must be converted.
- Each resource has one hook file under `src/lib/hooks/api/use-<resource>.ts` named `use<Resource>`, `use<Resource>Detail`, `use<Resource>Create`, `use<Resource>Update`, `use<Resource>Delete` as applicable. Naming is kebab-case filename + camelCase export (home-hub pattern, e.g. `use-recipes.ts` exports `useRecipes`).
- Service modules under `src/services/api/<resource>.service.ts` contain only thin REST wrappers calling `apiClient.get/post/put/patch/delete`. They return typed domain models. No caching, retry, or tenant-header logic — those live in the client.
- Delete `services/api/base.service.ts` (499 LOC). Any query-building helpers still needed go into a small `src/lib/api/query-params.ts` utility or inline in individual service files.
- Shrink `lib/api/client.ts` toward home-hub's 572-LOC shape. Retain: base-URL config, tenant header injection (four headers — see §4.6), JSON body handling, `ApiRequestError` with status, 401 surfacing as an error (atlas-ui has **no auth endpoint, no session, no `next-auth` — confirmed via audit**; all of home-hub's refresh/redirect/household logic is removed, not adapted), request deduplication MAY stay if already used by existing hooks (audit before keeping), retry on 5xx/429 with exponential backoff MAY stay.
- Drop from the client (these are unused by atlas-ui pages today — verify in the migration plan): upload with progress (`uploadWithProgress`, XHR + progress events), `onProgress` callback, `cache` / TTL / stale-while-revalidate (React Query already handles caching), `download` blob fetch unless a page actually uses it, `skipDeduplication` option, `staleWhileRevalidate` flag.
- Query keys live in `src/lib/hooks/api/query-keys.ts` as a single object tree, mirroring home-hub.
- `QueryProvider` (`src/components/providers/query-provider.tsx`) sets sensible defaults (`staleTime`, `retry`, error handling) identical to home-hub.

### 4.6 Multi-tenant context
- `context/tenant-context.tsx` moves to `src/context/tenant-context.tsx`. LocalStorage persistence, tenant list fetch, and `activeTenant` selection semantics are preserved verbatim.
- **Behavioural change — tenant header wiring:** today every service method explicitly calls `apiClient.setTenant(tenant)` immediately before each request, with the active tenant passed in as a parameter. This is ~30+ call sites of redundant plumbing. The migration moves to the home-hub pattern: `TenantProvider` calls `apiClient.setTenant(activeTenant)` once in an effect whenever `activeTenant` changes, and service methods no longer receive `tenant` as a parameter. Network behaviour (all four headers on every request) is preserved; only the wiring is centralised.
- **Behavioural addition — cache invalidation on tenant switch:** today no `queryClient.clear()` runs when the active tenant changes. This is a latent bug (stale tenant-A data can persist when switching to tenant B for any query whose key doesn't include the tenant). The migration adds `queryClient.clear()` in the same `TenantProvider` effect that calls `setTenant`. This is a **new invariant**, not a preservation — R6 is recast accordingly.
- The tenant-scoping contract is **four** HTTP headers, injected by `apiClient` via the helper currently at `services/atlas-ui/lib/headers.tsx`. Names and shapes must be preserved verbatim (Go services depend on them):
  - `TENANT_ID` — tenant UUID
  - `REGION` — tenant region string
  - `MAJOR_VERSION` — integer as string
  - `MINOR_VERSION` — integer as string
- Header names are SCREAMING_SNAKE_CASE (unusual for HTTP, but this is the contract the Go services expect — do not normalise to `X-Tenant-Id` or similar).
- `TenantProvider` wraps `<Routes>` so every page and every query has tenant context available.

### 4.7 Preserved features
- shadcn/ui component set under `src/components/ui/` — copy verbatim, update any `"use client"` directives to simply be removed.
- Tailwind config, `globals.css`, theme tokens, dark/light mode toggle.
- `BreadcrumbBar`, `AppSidebar`, `SidebarToggle`, `ThemeToggle` components.
- React Query Devtools in development.
- `react-hook-form` + Zod forms — no changes.
- React Flow NPC conversation graph (`app/npcs/[id]/conversations/conversation-flow.tsx`) — copy to `src/pages/npcs/npc-conversation-flow.tsx` or equivalent. `reactflow` package stays.
- Character renderer components (maplestory.io sprite composition) — copy verbatim, swap `next/image` for `<img>`.
- `sonner` toaster.

### 4.8 Testing
- Migrate all Jest tests to Vitest. Replacements:
  - `jest.fn()` → `vi.fn()`
  - `jest.mock()` → `vi.mock()`
  - `jest.spyOn()` → `vi.spyOn()`
  - `jest.clearAllMocks()` → `vi.clearAllMocks()`
  - `@testing-library/jest-dom` continues to work under Vitest via `import "@testing-library/jest-dom/vitest"` in `src/test/setup.ts`.
- `jest.setup.js` → `src/test/setup.ts`, imported by `vite.config.ts` `test.setupFiles`.
- `jest.config.js`, `jest-dom.d.ts`, Jest devDeps removed.
- All existing test files (`lib/hooks/api/__tests__/*`, `services/api/__tests__/*`, component tests) run under Vitest and pass. If a test relies on a Jest-specific API with no Vitest equivalent (e.g., `jest.useFakeTimers('modern')` nuances), port it; do not skip.
- Minimum passing bar: `npm run test` completes green with the same number of test cases as before migration (counted before work starts and tracked through the migration).

### 4.9 ESLint
- Replace `eslint.config.mjs` / `eslint.config.mts` with a single flat config based on home-hub's `eslint.config.js`: `@eslint/js` recommended + `typescript-eslint` recommended + `react-hooks` recommended-latest + `react-refresh/recommended`.
- Remove `eslint-config-next` rules. Any atlas-ui rule overrides currently in place must be reviewed and either dropped (Next-specific) or ported.
- `npm run lint` produces zero errors on the migrated tree. Warnings are acceptable if they existed pre-migration, but no new warnings introduced by the migration.

### 4.10 Docker & deployment
- New `services/atlas-ui/Dockerfile` is a two-stage build: Node builder (`node:24-alpine`) runs `npm ci` + `npm run build`, output copied to `nginx:alpine` at `/usr/share/nginx/html`. Port 80 exposed.
- New `services/atlas-ui/nginx.conf` configures SPA fallback (`try_files $uri $uri/ /index.html`) — copy home-hub's verbatim.
- Remove Next.js server runtime from the production image. Image should be substantially smaller (target: < 50 MB compressed, matching home-hub's nginx image class).
- Update any `deploy/` / Compose / Kubernetes manifests referencing atlas-ui's Next server (port 3000, standalone build output, `node server.js` command) to the nginx runtime. Verify `deploy/` manifests are updated consistently.
- `NEXT_PUBLIC_*` env vars → `VITE_*` replacements throughout `.env` files, Dockerfile build args, deploy manifests, and code references. Currently only one public env var is in use: `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL`. Audit again at execution time in case more are added.

### 4.11 Documentation
- Rewrite `services/atlas-ui/CLAUDE.md` to describe the Vite + React Router architecture, data-fetch pattern, component conventions, and testing approach. Remove sections describing Next.js features (App Router, server components, route groups, `getServerSession`, `next/image`).
- Update `services/atlas-ui/README.md` with new dev / build / test / Docker commands.
- Update root `docs/` if any references to atlas-ui's Next.js stack exist there.

## 5. API Surface

No backend API changes. The Go services remain the source of truth. Every endpoint currently called by atlas-ui continues to be called with the same URL, method, body shape, headers (including the four tenant-scoping headers defined in §4.6), and response shape.

No new atlas-ui-owned API routes are created. The Next.js `app/api/**/route.ts` directory does not exist today and is not added (the frontend proxies directly to Go services via relative URLs and Vite's dev proxy if needed; revisit if CORS issues appear in dev).

Dev-time CORS / proxy: `vite.config.ts` gets a `server.proxy` entry forwarding `/api` to the local atlas-ingress nginx. Default target: `http://localhost:${VITE_INGRESS_PORT:-8080}` (the compose stack exposes `atlas-ingress` as `${INGRESS_HOST_PORT:-8080}:80`). Developers running the full compose stack will hit `localhost:8080`; those running against a remote stack override via env. Production runs behind the same nginx/ingress as today — routing is unchanged.

## 6. Data Model

No data-model changes. No database migrations. No new tenant-scoped tables.

## 7. Service Impact

- **atlas-ui** (only affected service): full rewrite of the build tool, router, data-fetch layer, tests, lint, and Dockerfile. No code is preserved unchanged; every file is at minimum re-imported, stripped of `"use client"`, and its Next.js imports replaced.
- **Go services**: no impact. They continue to serve the same endpoints to atlas-ui.
- **atlas-tenants**: no impact. atlas-ui continues to fetch tenant config via the same endpoints.
- **deploy/**: four files reference atlas-ui, all are in scope:
  - `deploy/k8s/atlas-ui.yaml` — Deployment (containerPort 3000 → 80) and Service (port 3000 → 80). No probes defined today, so no probe paths to update. `NEXT_PUBLIC_ROOT_API_URL` env → `VITE_ROOT_API_URL`.
  - `deploy/k8s/ingress.yaml` — two nginx `location` blocks proxy to `atlas-ui:3000`. The `/_next/webpack-hmr` block is Next-specific and **deleted entirely**. The catch-all `location /` → `atlas-ui:80`.
  - `deploy/shared/routes.conf` — same pattern (HMR block deleted; catch-all updated to port 80).
  - `deploy/compose/docker-compose.core.yml` — `3000:3000` mapping → `3000:80` (host port can stay 3000 so local dev tooling doesn't change; container port moves to 80). `NEXT_PUBLIC_ROOT_API_URL` env → `VITE_ROOT_API_URL`.

## 8. Non-Functional Requirements

### 8.1 Performance
- `npm run dev` cold start in under 2 seconds (Vite HMR baseline).
- `npm run build` produces a `dist/` folder with route-level bundles (Vite default chunking is sufficient; no custom code-splitting required for v1).
- Runtime performance parity: page loads, data fetches, and interactions must feel the same as before. No regressions in time-to-interactive on main list pages (characters, items, maps, monsters).
- Production Docker image under 50 MB compressed.

### 8.2 Security
- SPA fallback in nginx (`try_files`) does not expose directory listings.
- Tenant header injection (the four headers in §4.6) remains the only tenant-scoping mechanism on the client side; this is unchanged from today.
- Drop `dangerouslyAllowSVG` and the related CSP from `next.config.ts` — these were Next's image optimizer pass-through, not application security boundaries. If SVG rendering is needed, it continues via `<img src="*.svg">` from trusted origins.
- **No auth surface exists today.** Audit confirmed zero `next-auth`, no `getServerSession`, no route guards, no session middleware. The existing `CLAUDE.md`'s references to `(auth)/` route groups and session logic are aspirational fiction, not present in the codebase. The migration removes home-hub's auth/refresh/redirect logic from the ported client rather than adapting it. No new auth surface is introduced.

### 8.3 Observability
- `services/api/errorLogger.ts` is ported verbatim (or replaced with a minimal equivalent under `src/services/errorLogger.ts`) so client-side error reporting continues to work.
- React Query error handling defaults (in `QueryProvider`) log or toast errors consistently with today's behaviour. Match home-hub's defaults.

### 8.4 Multi-tenancy
- Tenant context is established by `TenantProvider` and persisted to `localStorage` — unchanged.
- Every API call carries all four tenant headers (see §4.6). **Wiring changes** from per-call `setTenant(tenant)` at every service method to a single `TenantProvider` effect that calls `apiClient.setTenant(activeTenant)` on change. Network-observable behaviour is unchanged.
- Switching tenants invalidates React Query cache (`queryClient.clear()`) — **new invariant added by this migration**, not preserved from today. See §4.6 for context.

### 8.5 Accessibility
- No regressions. shadcn/ui accessibility properties are preserved.
- Replacement of `next/link` with `react-router-dom` `<Link>` does not change semantic markup (both render `<a>`).

## 9. Open Questions

1. Are there any atlas-ui pages or components that import `next/font`, `next/headers`, or other Next-only APIs beyond the ones catalogued in section 4.3? Migration plan must confirm via `grep` audit before execution.
2. Does `services/api/base.service.ts` contain any primitives (beyond retry/dedup/caching) that actual service modules rely on? Confirm by auditing every `.service.ts` file's imports.
3. Does any React component rely on Next.js implicit CSS handling (e.g., CSS Modules auto-imports, `globals.css` load order)? Vite's CSS handling is compatible but import paths need to be explicit. Audit during execution.
4. Does any existing test rely on the Next.js Jest transformer (via `next/jest`)? If so, configure the Vitest equivalent or port the transformations.
5. What is the current production deploy topology for atlas-ui? Confirm whether port 3000 → 80 is safe or whether the ingress config expects port 3000 regardless of the backing server.
6. Is there a `deploy/` directory that currently references atlas-ui's Next runtime? If so, its manifests are in scope for this task.

## 10. Acceptance Criteria

- [ ] `services/atlas-ui/` contains no `next*` or `eslint-config-next` dependencies in `package.json`.
- [ ] `services/atlas-ui/` contains `vite.config.ts`, `index.html`, `src/main.tsx`, `src/App.tsx`.
- [ ] `services/atlas-ui/app/` directory no longer exists.
- [ ] `grep -r "use client"` in `services/atlas-ui/src/` returns zero matches.
- [ ] `grep -r "next/" services/atlas-ui/src/` returns zero matches.
- [ ] Every one of the 46 pages listed in §4.2 resolves, renders, and fetches data without runtime errors (manual smoke test).
- [ ] All existing Jest tests have been ported to Vitest. `npm run test` passes with test count ≥ pre-migration baseline.
- [ ] `npm run lint` passes with zero errors.
- [ ] `npm run build` produces a `dist/` folder. `npm run preview` serves the built app and it loads end-to-end.
- [ ] Docker build succeeds: `docker build services/atlas-ui` produces an nginx-based image under 50 MB.
- [ ] `lib/api/client.ts` is under 700 LOC; `services/api/base.service.ts` is deleted.
- [ ] No page file contains `useEffect` + `fetch` / service-call pattern. Every page uses a React Query hook from `src/lib/hooks/api/`.
- [ ] `services/atlas-ui/CLAUDE.md` accurately describes the new architecture and contains no references to Next.js features.
- [ ] Tenant switching invalidates the React Query cache (manual verification: switch tenants → character list refetches).
- [ ] Multi-tenant API calls carry all four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`) in the network inspector for every request.
- [ ] `deploy/` manifests referencing atlas-ui have been updated to match the new runtime (port, command, health check).
- [ ] Follow-up PR list documented for anything explicitly deferred (e.g., route-level `React.lazy` splitting).
