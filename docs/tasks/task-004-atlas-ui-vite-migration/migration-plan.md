# atlas-ui Vite Migration — Migration Plan

Status: Draft
Created: 2026-04-17
Companion to: `prd.md`
---

## Approach: Big-bang on a dedicated branch

One branch, one end-to-end rewrite, one PR (or at most one PR per phase below). No incremental dual-router state, no temporary adapters.

Sequencing below is **preparation → scaffold → port → tests → infra → cleanup**. Each phase is a natural commit boundary.

## Pre-migration audit (half a day)

Before touching anything:

1. `grep -rn "use client" services/atlas-ui/` — record exact count (expected ~140).
2. `grep -rn "next/" services/atlas-ui/` — record every Next import, grouped by module:
   - `next/link`
   - `next/image`
   - `next/navigation` (broken down by hook: `useRouter`, `useParams`, `useSearchParams`, `usePathname`, `redirect`, `notFound`)
   - `next-themes` (**swap** to home-hub's `ThemeProvider`. Although framework-agnostic, keeping the dep leaves a `next*` entry in `package.json` and violates the acceptance criteria.)
   - `next/font` — if present, plan Geist or similar via `@fontsource-variable/*`
   - `next/headers`, `next/server`, `next/cache`, `next/dynamic` — flag each usage individually
3. Count test files (`find services/atlas-ui -name '*.test.ts*' | wc -l`) — this is the Vitest acceptance baseline.
4. Enumerate `apiClient` methods used across `services/api/*.service.ts` — anything not used in a hook or page can be dropped from the shrunk client.
5. Read `services/api/base.service.ts` end to end. List which of its primitives (query-param building, pagination, JSON:API envelope unwrapping, etc.) are used by which service modules. Decide for each: inline into service modules, move to a ~50-LOC `src/lib/api/json-api.ts` helper, or drop.
6. Read `next.config.ts` and identify every env var read (expect `DOCKER_ENV`, `KUBERNETES_SERVICE_HOST`, `NODE_ENV`). These disappear.
7. `grep -rn "NEXT_PUBLIC_" services/atlas-ui/` — list every public env var. Each gets a `VITE_*` rename.
8. Check `deploy/` for atlas-ui service definitions; note port, command, health check, and any env vars passed.
9. Capture current Docker image size (`docker images atlas-ui`) as the regression baseline.

Commit the audit results as `task-004-atlas-ui-vite-migration/audit.md` if helpful, or fold into this plan during execution.

## Phase 1 — Scaffold Vite project (side-by-side, half a day)

Branch: `atlas-ui-vite-migration`

1. In `services/atlas-ui/`, add new root files alongside Next.js config (do not delete Next yet):
   - `vite.config.ts` — copy from home-hub. Adjust the `@` alias target once `src/` exists. Add a `server.proxy` entry forwarding `/api` to `http://localhost:${VITE_INGRESS_PORT:-8080}` (matches the compose `atlas-ingress` published port).
   - `index.html` — copy from home-hub, set `<title>AtlasMS</title>`, link `/src/main.tsx`.
   - `tsconfig.app.json`, `tsconfig.node.json` — copy from home-hub, adjust `paths` to `{"@/*": ["./src/*"]}`.
   - `eslint.config.js` — copy from home-hub verbatim.
2. Update `package.json`:
   - Add `"type": "module"`.
   - Swap scripts to Vite equivalents.
   - Add new deps (`vite@^8`, `@vitejs/plugin-react@^6`, `@tailwindcss/vite@^4.2`, `react-router-dom@^7.13`, `vitest@^4.1`, `jsdom@^29`, `eslint-plugin-react-refresh`, `@testing-library/jest-dom@^6.9`, `@eslint/js@^10`, `eslint-plugin-react-hooks@^7`, `typescript-eslint@^8.57`, `eslint@^10`).
   - Remove `@tailwindcss/postcss` (will be replaced by `@tailwindcss/vite` above). Delete `postcss.config.mjs` in Phase 6.
   - React/TypeScript/core Tailwind versions do not change — atlas-ui is already on React 19.2.4, TS 6.x, Tailwind 4.2.x. Both repos use npm (`package-lock.json`); no package-manager switch.
   - Keep Next deps installed for now — remove in Phase 6.
   - Run `npm install`.
3. Create `src/` directory. Move `app/globals.css` → `src/index.css` (adjust any Next-specific `@layer` directives if needed; Tailwind 4 syntax is the same).
4. Write `src/main.tsx` identical to home-hub's.
5. Write `src/App.tsx` with the provider stack from home-hub, but routes commented out initially — just route to a "Hello" placeholder to verify `npm run dev` starts.
6. **Verify**: `npm run dev` serves a blank "Hello" at `http://localhost:5173`. Next.js dev server still works via `npm run next:dev` temporary alias if you want to A/B compare.

**Commit**: `chore(atlas-ui): scaffold Vite + react-router alongside Next.js`.

## Phase 2 — Port shared infrastructure (1 day)

Goal: get providers, router shell, and API client working before touching pages.

1. **Copy shared tree** into `src/`:
   - `components/` → `src/components/`
   - `context/` → `src/context/`
   - `hooks/` → `src/hooks/` (note: the few `hooks/` at the top level of atlas-ui; most are under `lib/hooks/`)
   - `lib/` → `src/lib/`
   - `services/` → `src/services/`
   - `types/` → `src/types/`
2. **Strip `"use client"`** from every file in `src/` — sed/rg replace:
   ```
   grep -rln '"use client"' services/atlas-ui/src/ | xargs sed -i '/^"use client";\?$/d'
   ```
   Handle both `"use client"` and `"use client";` variants.
3. **Replace `next/link`** — global find-and-replace:
   - `from "next/link"` → `from "react-router-dom"`
   - `<Link href={` → `<Link to={`
   - `href="` on `<Link>` → `to="`
4. **Replace `next/navigation`** — catalogue each hook and rewrite:
   - `useRouter` → `useNavigate` (and convert `router.push(x)` to `navigate(x)`, etc.)
   - `useParams` — signature compatible, but atlas-ui code may have `Array.isArray` guards that can be simplified; leave if it works.
   - `useSearchParams` — **API mismatch**. Next returns a `ReadonlyURLSearchParams`; RR v7 returns `[URLSearchParams, setSearchParams]`. Each call site needs adjustment. Expect ~5-10 sites based on usage patterns in list pages with filters.
   - `usePathname` → `useLocation().pathname`.
   - `redirect(x)` → throw + effect + `navigate(x, { replace: true })` or render `<Navigate to={x} replace />` from a component.
   - `notFound()` → render the 404 component or `navigate("/404")`.
5. **Replace `next/image`**:
   - `import Image from "next/image"` → remove import, use `<img>`.
   - `<Image src={x} width={y} height={z} alt={a} priority />` → `<img src={x} width={y} height={z} alt={a} />`.
   - `placeholder="blur"` and `blurDataURL` props dropped.
   - `sizes` prop dropped (only meaningful for Next optimizer).
   - `unoptimized` prop dropped.
   - For the `maplestory.io` sprite renderer: plain `<img loading="lazy">` at each call site. No `<MaplestoryImage>` wrapper — decision locked (see PRD §4.4).
6. **Delete Next-specific config files from src tree** if any were copied by accident:
   - `next-env.d.ts` reference in tsconfig (remove include).
   - Any `__next*` folders.
7. **Port error boundaries**:
   - `app/error.tsx` / `app/global-error.tsx` / `app/not-found.tsx` → `src/components/common/error-boundary.tsx` and `src/components/common/not-found-page.tsx`.
   - Wire the error boundary above `<Routes>` in `App.tsx`.
8. **Environment variables**:
   - `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL` (the only public env var in use today; re-audit at execution time in case more were added).
   - `process.env.X` → `import.meta.env.VITE_X`.
   - Add a `src/vite-env.d.ts` declaring module types (home-hub has this).
   - Update `.env`, `.env.example`, `.env.local`, and Docker build args.
9. **Shrink `lib/api/client.ts`**:
   - Start from home-hub's `src/lib/api/client.ts` (572 LOC).
   - Delete the auth/refresh/redirect flow entirely. Audit confirmed atlas-ui has no auth surface — no session, no login endpoint, no `next-auth`. Replace `handleUnauthorized` with a logger; still surface 401 as `ApiRequestError`.
   - Delete household logic (atlas-ui has no household concept — only tenant).
   - Delete `upload`, `download`, `uploadWithProgress`, and `onProgress` if the pre-migration audit confirmed no callers.
   - **Preserve the four-header tenant contract verbatim**: `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION` — all SCREAMING_SNAKE_CASE. Source of truth today is `services/atlas-ui/lib/headers.tsx`. Port this helper to `src/lib/headers.ts` (or inline into the client). Do not rename to `X-Tenant-Id` — Go services reject requests without the exact names.
   - Result: expect 350-500 LOC.
10. **Delete `services/api/base.service.ts`**:
    - For each primitive, inline the logic into callers OR move to a ~50-LOC helper file.
    - Typical content: `buildQuery(params)` helper, JSON:API envelope unwrap. Both are small.
11. **Providers**:
    - `QueryProvider` → `src/components/providers/query-provider.tsx`, copy home-hub's defaults.
    - `ThemeProvider` → swap to home-hub's `src/components/providers/theme-provider.tsx` to drop the `next-themes` dep.
    - `TenantProvider` → port from atlas-ui, but:
      - Add an effect that calls `apiClient.setTenant(activeTenant)` whenever `activeTenant` changes. Today, every service method calls `api.setTenant(tenant)` before each request — this centralises it.
      - Add `queryClient.clear()` in the same effect. Today this does not happen on tenant switch, which is a latent bug.
      - Remove the per-call `api.setTenant(tenant)` invocations from every service module (affects ~30+ call sites across `bans.service.ts`, `items.service.ts`, `inventory.service.ts`, `reactors.service.ts`, `merchants.service.ts`, `quests.service.ts`, `gachapons.service.ts`, `portal-scripts.service.ts`, and more). Service signatures can drop the `tenant` parameter accordingly.
      - Update any `next/navigation` imports inside the provider (there are none today, but audit to confirm).

**Commit**: `refactor(atlas-ui): port shared infra to Vite, strip Next.js imports`.

## Phase 3 — Port pages (2-3 days)

Every Next.js page becomes a React Router route + `src/pages/<Name>Page.tsx`.

1. **Layout route**:
   - Create `src/components/features/navigation/app-shell.tsx` — copy the outer markup from `app/layout.tsx` (sidebar, header, main frame), replace `{children}` with `<Outlet />` from `react-router-dom`.
   - Providers (`TenantProvider`, `QueryProvider`, `ThemeProvider`, `SidebarProvider`) move to `App.tsx`; `AppSidebar` and breadcrumb chrome stay inside `AppShell`.
2. **Convert each `app/**/page.tsx`**:
   - Default-export `function PageComponent()` → named export `function <Name>Page()`.
   - Move the file to `src/pages/<Name>Page.tsx` (flatten directory structure; see route table in §4.2 of PRD).
   - Supporting files co-located in App Router folders (`columns.tsx`, `loading.tsx`, `conversation-flow.tsx`, `conversation-page.tsx`) move to a sibling folder like `src/pages/<domain>/` or `src/components/features/<domain>/`.
   - Replace `useParams` destructuring if needed (`const { id } = useParams<{ id: string }>()`).
   - Replace any page-level `useEffect` + service-call fetches with the corresponding React Query hook.
3. **Route list wiring**:
   - In `App.tsx`, expand the `<Routes>` block with every route from PRD §4.2.
   - Sort routes by URL path, nested appropriately under `AppShell`.
4. **Dynamic routes gotcha**:
   - Next uses `[id]` and `[banId]` and `[portalId]` — in RR these are `:id`, `:banId`, `:portalId`. Ensure the param *name* matches what the page code reads via `useParams`.
5. **Delete the `app/` directory** once every page has been ported and renders.

**Commit(s)**: `feat(atlas-ui): port pages to react-router` — can be one big commit or split by domain (characters, items, maps, npcs, etc.) if the diff is too large.

## Phase 4 — Consolidate data fetching (1-2 days)

For every page:

1. Identify all reads it performs.
2. For each read:
   - If a React Query hook already exists in `src/lib/hooks/api/`, use it.
   - If not, create one: `src/lib/hooks/api/use-<resource>.ts` returning `useQuery({ queryKey, queryFn, staleTime })`.
   - Add the query key to `src/lib/hooks/api/query-keys.ts`.
3. For each write:
   - Create `useCreate<Resource>`, `useUpdate<Resource>`, `useDelete<Resource>` hooks using `useMutation` with `onSuccess: queryClient.invalidateQueries`.
4. Remove `useState(data)` + `useEffect(fetch)` patterns from every page.
5. Delete any now-unused service module functions or wrapper hooks from `src/lib/hooks/` (the non-`api/` parent folder — likely contains a few redundant wrappers like `useNpcData`, `useItemData`, `useMobData`, `useSkillData` that overlap with the `api/` hooks).

**Commit**: `refactor(atlas-ui): consolidate data fetching behind React Query hooks`.

## Phase 5 — Tests & lint (1 day)

1. **Vitest setup**:
   - Create `src/test/setup.ts` with `import "@testing-library/jest-dom/vitest"` and any global mocks (e.g., `window.matchMedia`).
   - `vite.config.ts` already references it from Phase 1.
   - Delete `jest.config.js`, `jest.setup.js`, `jest-dom.d.ts`.
2. **Port each test file**:
   - Rename imports: `jest.fn` → `vi.fn`, etc. (Vitest exposes `vi` globally with `globals: true`.)
   - Replace `jest.mock('module', factory)` → `vi.mock('module', factory)`. Mock hoisting rules are the same.
   - `vi.hoisted(() => ...)` replaces the Jest-specific top-of-file hoisting patterns if any exist.
   - Update import paths for anything that moved (shared tree now under `src/`).
3. **Next.js test-specific shims**:
   - If any test mocks `next/navigation` or `next/link`, swap to mocking `react-router-dom`.
   - If any test uses `next/jest` transformer — remove; Vite's transformer handles TypeScript and JSX natively.
4. Run `npm run test` until all tests pass. Count matches pre-migration baseline.
5. Run `npm run lint`. Fix any errors surfaced by the new ESLint flat config.

**Commit**: `test(atlas-ui): migrate Jest suite to Vitest`.

## Phase 6 — Infra cleanup (half a day)

1. **Remove Next.js**:
   - `npm uninstall next eslint-config-next next-themes` (and any other Next-specific deps).
   - Delete `next.config.ts`, `next-env.d.ts`, `eslint.config.mjs`, `eslint.config.mts`.
   - Delete `lib/image-loader.ts`.
   - Delete `postcss.config.mjs` if swapping to `@tailwindcss/vite` (home-hub has no PostCSS config). If keeping `@tailwindcss/postcss`, keep it but audit.
2. **Dockerfile**:
   - Replace contents with home-hub's Dockerfile verbatim, adjust `WORKDIR` and image name as needed.
   - Add `services/atlas-ui/nginx.conf` (copy from home-hub).
3. **Deploy manifests** (concrete inventory from Phase 0 audit):
   - `deploy/k8s/atlas-ui.yaml` — change Deployment `containerPort: 3000` → `80`; Service `port: 3000` → `80`; rename env var `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL`. No probes defined today, so no probe paths to update.
   - `deploy/k8s/ingress.yaml` — **delete** the `location /_next/webpack-hmr { ... }` block (lines ~324-331; Next HMR only). Update the catch-all `location /` to `proxy_pass http://atlas-ui:80;`.
   - `deploy/shared/routes.conf` — same surgery: delete the HMR block (lines ~347-355); update `set $u "atlas-ui:3000"` in the catch-all to `"atlas-ui:80"`.
   - `deploy/compose/docker-compose.core.yml` — change `ports: - "3000:3000"` → `ports: - "3000:80"` (host port 3000 stays so local tooling is unaffected; container port moves to nginx's 80). Rename `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL`.
4. **Dev scripts**:
   - Update any shell scripts under `scripts/` that reference `next dev` / `next build`.

**Commit**: `chore(atlas-ui): remove Next.js, switch to nginx runtime`.

## Phase 7 — Docs & verification (half a day)

1. Rewrite `services/atlas-ui/CLAUDE.md` from scratch. Use home-hub's frontend CLAUDE.md (if any) as a model or the new architecture as-implemented. Kill all references to App Router, `(auth)/`, `(dashboard)/`, server sessions, `next/image`, route handlers.
2. Update `services/atlas-ui/README.md`:
   - Dev: `npm run dev` → `http://localhost:5173`
   - Build: `npm run build` → `dist/`
   - Preview: `npm run preview`
   - Test: `npm run test`
   - Docker: `docker build . -t atlas-ui && docker run -p 8080:80 atlas-ui`
3. Verification checklist (manual):
   - [ ] Every route in PRD §4.2 loads without console errors.
   - [ ] Tenant switcher updates data on at least characters, items, maps pages.
   - [ ] Forms (`react-hook-form` + Zod) submit successfully on at least one create page.
   - [ ] NPC conversation React Flow graph renders.
   - [ ] Character sprite renderer shows sprites for at least one character.
   - [ ] Dark/light theme toggle works.
   - [ ] Breadcrumbs update on route changes.
   - [ ] `npm run build` produces `dist/` under a reasonable size.
   - [ ] `docker build` succeeds; the resulting container serves the app on port 80.
4. Update `docs/TODO.md` with any deferred follow-ups (route-level `React.lazy`, etc.). `docs/TODO.md` already exists and uses a per-area Markdown-checklist format (`### Area Name` then `- [ ] item (file:line)`). Add a new `### atlas-ui Frontend` section at the end.

**Commit**: `docs(atlas-ui): update guides for Vite + React Router`.

## Rollback plan

- Keep `atlas-ui-vite-migration` branch separate until verification passes.
- The old Next.js implementation remains on `main` until the PR is merged. Rollback = revert the merge commit.
- No database changes, no API changes — rollback has no data implications.
- Deploy pipeline should be able to pin to the pre-merge image tag indefinitely.

## Estimated duration

- Phase 0 (audit): 0.5 day
- Phase 1 (scaffold): 0.5 day
- Phase 2 (infra): 1 day
- Phase 3 (pages): 2-3 days
- Phase 4 (data fetching): 1-2 days
- Phase 5 (tests): 1 day
- Phase 6 (infra cleanup): 0.5 day
- Phase 7 (docs + verify): 0.5 day

**Total: 7-9 developer-days**, executed big-bang on a single branch.
