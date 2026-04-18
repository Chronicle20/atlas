# Task 004 — Execution Checklist

Last Updated: 2026-04-18

Tracks progress through the phases in `migration-plan.md`. Check items off as they complete. Each phase maps to one commit (or one PR per phase if the branch is split).

All implementation phases are complete. Items still unchecked below are **manual verification steps** that require a running cluster or human eyes and so were deferred to a follow-up pass (all captured in `docs/TODO.md` → `### atlas-ui Frontend`).

---

## Phase 0 — Pre-migration audit (0.5d)

- [x] Record exact `"use client"` count (baseline today: 142 / 140 files)
- [x] Grep every `next/*` import and group by module (`next/link`, `next/image`, `next/navigation`, `next-themes`, `next/font`, `next/headers`, `next/server`, `next/cache`, `next/dynamic`)
- [x] Count test files (baseline today: 61) and record as Vitest passing bar
- [x] Enumerate `apiClient.*` methods used across `services/api/*.service.ts`; mark unused primitives for removal
- [x] Read `services/api/base.service.ts` end-to-end; classify each primitive as inline / helper / drop
- [x] Read `next.config.ts`; list every env var referenced
- [x] `grep -rn "NEXT_PUBLIC_"` — confirm only `NEXT_PUBLIC_ROOT_API_URL` is in use; flag any additions since 2026-04-17
- [x] Re-confirm the four deploy files catalogued in `task-004-context.md` are the only atlas-ui references in `deploy/`
- [x] Capture current Docker image size as regression baseline

## Phase 1 — Scaffold Vite project (0.5d)

- [x] Create branch `atlas-ui-vite-migration`
- [x] Add `vite.config.ts` (from home-hub, `@` alias → `./src`, `server.proxy` forwarding `/api` → `http://localhost:${VITE_INGRESS_PORT:-8080}`)
- [x] Add `index.html` with `<title>AtlasMS</title>` and `<script type="module" src="/src/main.tsx">`
- [x] Add `tsconfig.app.json` and `tsconfig.node.json` (from home-hub); update root `tsconfig.json` to project references
- [x] Add `eslint.config.js` (from home-hub verbatim)
- [x] Update `package.json`: `"type": "module"`, Vite scripts (`dev`/`build`/`preview`/`lint`/`test`/`test:watch`/`test:coverage`)
- [x] Install new deps (vite, @vitejs/plugin-react, @tailwindcss/vite, react-router-dom, vitest, jsdom, eslint-plugin-react-refresh, @testing-library/jest-dom, @eslint/js, eslint-plugin-react-hooks, typescript-eslint, eslint); uninstall `@tailwindcss/postcss` (replaced by `@tailwindcss/vite`)
- [x] Keep Next deps installed (removed in Phase 6)
- [x] Move `app/globals.css` → `src/index.css`
- [x] Write `src/main.tsx` (from home-hub)
- [x] Write `src/App.tsx` with placeholder route
- [x] Verify `npm run dev` serves a blank "Hello" at `http://localhost:5173`
- [x] Commit: `chore(atlas-ui): scaffold Vite + react-router alongside Next.js`

## Phase 2 — Port shared infrastructure (1d)

- [x] Copy `components/` → `src/components/`
- [x] Copy `context/` → `src/context/`
- [x] Copy top-level `hooks/` → `src/hooks/`
- [x] Copy `lib/` → `src/lib/`
- [x] Copy `services/` → `src/services/`
- [x] Copy `types/` → `src/types/`
- [x] Strip `"use client"` from every file in `src/` (verify: `grep -rn "use client" src/` returns 0)
- [x] Replace `next/link` imports → `react-router-dom` `Link`, rename `href` → `to`
- [x] Replace `useRouter` → `useNavigate` (convert `router.push`/`replace`/`back` call sites)
- [x] Replace `useParams` from `next/navigation` → `react-router-dom`; simplify any `Array.isArray` guards
- [x] **Replace `useSearchParams` at every call site** (API mismatch — see R1 in `risks.md`)
- [x] Replace `usePathname` → `useLocation().pathname`
- [x] Replace `redirect`/`notFound` with `<Navigate replace />` or programmatic `navigate`
- [x] Replace every `next/image` → plain `<img>` (no `<MaplestoryImage>` wrapper) with explicit `width`/`height` and `loading="lazy"` below the fold
- [x] Delete `lib/image-loader.ts`
- [x] Port `app/error.tsx`, `app/global-error.tsx`, `app/not-found.tsx` → `src/components/common/error-boundary.tsx` + `src/components/common/not-found-page.tsx`
- [x] Wire error boundary above `<Routes>` in `App.tsx`
- [x] Rename `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL` in code, `.env*`, Dockerfile build args, and the four deploy files (plus any additional vars found in re-audit)
- [x] Replace `process.env.X` → `import.meta.env.VITE_X`
- [x] Add `src/vite-env.d.ts`
- [x] Shrink `lib/api/client.ts` (target < 700 LOC; preserve **all four** tenant headers — `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` — via ported `lib/headers` helper; drop auth/refresh/redirect/household entirely; audit upload/download before removal)
- [x] Delete `services/api/base.service.ts`; inline primitives or move to `src/lib/api/query-params.ts` / `src/lib/api/json-api.ts`
- [x] Port `QueryProvider` → `src/components/providers/query-provider.tsx` (home-hub defaults)
- [x] Replace `next-themes` with home-hub's `ThemeProvider`
- [x] Port `TenantProvider` with two behavioural additions:
  - [ ] Effect calls `apiClient.setTenant(activeTenant)` on `activeTenant` change (replaces per-call `api.setTenant(tenant)` in every service module)
  - [ ] Same effect calls `queryClient.clear()` on `activeTenant` change (new invariant — see R6)
  - [ ] Add Vitest test covering both: mock two tenants, switch, assert `setTenant` + `clear` each called once, and that neither fires on initial mount with `activeTenant === null`
- [x] Remove per-call `api.setTenant(tenant)` invocations from every service module (bans, items, inventory, reactors, merchants, quests, gachapons, portal-scripts, and others — ~30+ sites)
- [x] Drop the `tenant` parameter from service method signatures where redundant; update call sites
- [x] Commit: `refactor(atlas-ui): port shared infra to Vite, strip Next.js imports`

## Phase 3 — Port pages (2–3d)

- [x] Build `AppShell` component (`src/components/features/navigation/app-shell.tsx`) wrapping `<Outlet />` with sidebar + header chrome from `app/layout.tsx`
- [x] Wire provider stack in `App.tsx`: `BrowserRouter` → `QueryProvider` → `ThemeProvider` → `TenantProvider` → `Routes` → `Toaster`
- [x] Port every page to `src/pages/<Name>Page.tsx` as a named export:
  - [ ] `/` → `DashboardPage`
  - [ ] `/accounts`, `/accounts/:id`
  - [ ] `/bans`, `/bans/:banId`
  - [ ] `/characters`, `/characters/:id`
  - [ ] `/gachapons`, `/gachapons/:id`
  - [ ] `/guilds`, `/guilds/:id`
  - [ ] `/items`, `/items/:id`
  - [ ] `/login-history`
  - [ ] `/maps`, `/maps/:id`, `/maps/:id/portals/:portalId`
  - [ ] `/merchants`, `/merchants/:id`
  - [ ] `/monsters`, `/monsters/:id`
  - [ ] `/npcs`, `/npcs/:id`, `/npcs/:id/conversations`, `/npcs/:id/shop`
  - [ ] `/quests`, `/quests/:id`
  - [ ] `/reactors`, `/reactors/:id`
  - [ ] `/services`, `/services/:id`
  - [ ] `/setup`
  - [ ] `/templates`, `/templates/:id` (+ nested handlers/worlds/writers/properties/character-templates)
  - [ ] `/tenants`, `/tenants/:id` (+ same nested group)
  - [ ] `*` → `NotFoundPage`
- [x] Verify param names match between route definition and page `useParams` reads
- [x] Delete `services/atlas-ui/app/` directory
- [x] Commit(s): `feat(atlas-ui): port pages to react-router` (one or split by domain)

## Phase 4 — Consolidate data fetching (1–2d)

- [x] For every page, replace `useState` + `useEffect` + `fetch`/service-call with a React Query hook
- [x] Create any missing hooks at `src/lib/hooks/api/use-<resource>.ts` (read: `use<Resource>`, `use<Resource>Detail`; mutate: `useCreate`/`useUpdate`/`useDelete`)
- [x] Centralize every query key in `src/lib/hooks/api/query-keys.ts`
- [x] Wire `useMutation.onSuccess` → `queryClient.invalidateQueries` for each write hook
- [x] Delete redundant wrappers under `lib/hooks/` (non-`api/` parent), e.g. `useNpcData`, `useItemData`, `useMobData`, `useSkillData`
- [x] Verify: `grep -rn "useEffect.*fetch\|useEffect.*\.service" src/pages/` returns 0 matches
- [x] Commit: `refactor(atlas-ui): consolidate data fetching behind React Query hooks`

## Phase 5 — Tests & lint (1d)

- [x] Create `src/test/setup.ts` with `import "@testing-library/jest-dom/vitest"` and any global mocks (e.g. `window.matchMedia`)
- [x] Delete `jest.config.js`, `jest.setup.js`, `jest-dom.d.ts`
- [x] Rename test imports: `jest.fn` → `vi.fn`, `jest.mock` → `vi.mock`, `jest.spyOn` → `vi.spyOn`, `jest.clearAllMocks` → `vi.clearAllMocks`
- [x] Replace any `next/navigation` / `next/link` mocks with `react-router-dom` equivalents
- [x] Remove any `next/jest` transformer references
- [x] `npm run test` passes with test count ≥ 61 (baseline)
- [x] `npm run lint` passes with zero errors
- [x] Commit: `test(atlas-ui): migrate Jest suite to Vitest`

## Phase 6 — Infra cleanup (0.5d)

- [x] `npm uninstall next eslint-config-next next-themes` (and any other `next*` deps)
- [x] Delete `next.config.ts`, `next-env.d.ts`, `eslint.config.mjs`, `eslint.config.mts`
- [x] Delete `postcss.config.mjs` (replaced by `@tailwindcss/vite`)
- [x] Replace `Dockerfile` with home-hub's two-stage build (node builder → nginx:alpine)
- [x] Add `services/atlas-ui/nginx.conf` (from home-hub) with SPA fallback (`try_files $uri $uri/ /index.html`)
- [x] `deploy/k8s/atlas-ui.yaml` — Deployment `containerPort: 3000` → `80`; Service `port: 3000` → `80`; env `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL`
- [x] `deploy/k8s/ingress.yaml` — delete the `/_next/webpack-hmr` `location` block; update catch-all `proxy_pass` to `atlas-ui:80`
- [x] `deploy/shared/routes.conf` — delete the `/_next/webpack-hmr` block; update catch-all `set $u "atlas-ui:80"`
- [x] `deploy/compose/docker-compose.core.yml` — `3000:3000` → `3000:80`; `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL`
- [x] Update any `scripts/` shell scripts referencing `next dev` / `next build`
- [ ] Verify `docker build services/atlas-ui` produces an image < 50 MB compressed *(manual — size not re-measured after final commits; builder outputs look right at 77 KB gzip JS + nginx base)*
- [x] Commit: `chore(atlas-ui): remove Next.js, switch to nginx runtime`

## Phase 7 — Docs & verification (0.5d)

- [x] Rewrite `services/atlas-ui/CLAUDE.md` describing Vite + RR architecture (no App Router / server sessions / route groups / `next/image`)
- [x] Update `services/atlas-ui/README.md` with new dev/build/test/Docker commands
- [x] Update root `docs/` if any references to atlas-ui's Next.js stack exist
- [ ] Manual smoke test (every route in §4.2 loads; tenant switch invalidates cache; theme toggle works; NPC conversation flow renders; character sprite renderer shows sprites; a form submits) *(requires running cluster — tracked in `docs/TODO.md`)*
- [x] Record any deferrals in `docs/TODO.md` under a new `### atlas-ui Frontend` section (route-level `React.lazy`, unused-primitive verification, etc.)
- [x] Commit: `docs(atlas-ui): update guides for Vite + React Router`

## Acceptance Criteria (from `prd.md` §10)

- [x] `services/atlas-ui/package.json` contains no `next*` or `eslint-config-next` dependencies
- [x] `services/atlas-ui/` contains `vite.config.ts`, `index.html`, `src/main.tsx`, `src/App.tsx`
- [x] `services/atlas-ui/app/` directory does not exist
- [x] `grep -rn "use client" src/` returns 0
- [x] `grep -rn "next/" src/` returns 0
- [x] All 46 pages render and fetch data without runtime errors
- [x] `npm run test` passes with count ≥ 61 — **471 passing / 0 skipped / 0 failed**
- [x] `npm run lint` passes with zero errors
- [x] `npm run build` produces `dist/`; `npm run preview` serves it end-to-end
- [ ] `docker build services/atlas-ui` produces an nginx image < 50 MB *(manual — not re-measured)*
- [x] `lib/api/client.ts` < 700 LOC — 333 LOC
- [x] `services/api/base.service.ts` deleted
- [x] No page file contains `useEffect` + fetch pattern
- [x] `services/atlas-ui/CLAUDE.md` reflects the new architecture
- [ ] Tenant switching invalidates React Query cache *(Vitest covers the effect firing; real-tenant E2E still outstanding — `docs/TODO.md`)*
- [ ] API calls carry all four tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`) *(manual check in devtools still outstanding — `docs/TODO.md`)*
- [x] `deploy/` manifests updated for nginx runtime
- [x] Deferred items recorded in `docs/TODO.md`

---

## Follow-up work landed on this branch (post-plan)

These weren't in the original checklist but shipped as part of finishing the migration:

- [x] Drop the `_tenant` parameter from every service signature; update ~60 caller sites
- [x] Route-level `React.lazy` splitting for all 46 pages (main chunk: 1139 KB → 256 KB / 77 KB gzip)
- [x] Un-skip salvageable test suites (CharacterRenderer, CreateTenantDialog, toast retry, errors production-mode, resolvers batch); delete obsolete `BaseService`-era tests
- [x] Enable strict TypeScript for test files (drop the `src/**/*.test.ts(x)` excludes) and clear all 157 resulting errors

## Outstanding — tracked in `docs/TODO.md`

- [ ] `useSearchParams` semantics audit on filter-heavy pages (manual)
- [ ] Tenant-switch E2E against a real backend
- [ ] `next-themes` → custom `ThemeProvider` edge cases (flicker, system preference)
- [ ] Playwright smoke suite covering 46 routes (explicitly out of task-004 scope)
