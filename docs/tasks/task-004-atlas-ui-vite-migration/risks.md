# atlas-ui Vite Migration — Risks

Status: Draft
Created: 2026-04-17
Companion to: `prd.md`, `migration-plan.md`
---

## High-risk items

### R1 — `useSearchParams` API mismatch breaks filter/sort UI

**Probability:** High  
**Impact:** Medium  
**Description:** Next.js `useSearchParams()` returns a `ReadonlyURLSearchParams`. `react-router-dom` v7 returns `[URLSearchParams, setSearchParams]`. Any list page with search, filter, or sort controls (characters, items, maps, monsters, NPCs likely) will compile but behave wrong if call sites aren't all updated.

**Mitigation:** Grep `useSearchParams` exhaustively during Phase 2. Update every call site in one sweep. Write a regression test for at least one filtered list page to verify URL-state persistence.

### R2 — Jest → Vitest mock incompatibilities

**Probability:** Medium  
**Impact:** Medium  
**Description:** `jest.mock('module', () => ({ ... }))` and `vi.mock(...)` differ on hoisting edge cases, especially when the factory closes over imports. Tests that rely on `jest.unstable_mockModule`, `jest.requireActual`, or dynamic mock factories may need restructuring via `vi.hoisted()`.

**Mitigation:** Migrate tests in small batches, starting with the simplest (pure-function util tests). Tackle hook and component tests last. Budget extra time for conversation/NPC tests which are the most complex.

### R3 — `lib/api/client.ts` shrink breaks a feature that isn't covered by tests

**Probability:** Medium  
**Impact:** High  
**Description:** Dropping primitives (`onProgress`, `upload`, `download`, dedup cache, stale-while-revalidate, retry with backoff) assumes they are unused. If any page silently relies on one — e.g., a file-upload form for imports — it will break with a runtime error that may not surface until production.

**Mitigation:** Before deleting, run `grep -rn "apiClient\.\(upload\|download\|onProgress\|cache\)" services/atlas-ui/` and confirm zero matches. For primitives that are used, keep them. The shrink target (<700 LOC) is a soft goal, not a hard requirement.

### R4 — `next/image` loss causes visible layout shift or broken sprites

**Probability:** Medium  
**Impact:** Low-Medium  
**Description:** Next's Image component enforces width/height to prevent CLS and uses a blur placeholder. Plain `<img>` without explicit dimensions can cause layout shift, especially on character sprite grids.

**Mitigation:** For every `<Image>` replacement, keep the `width` and `height` HTML attributes so the browser reserves space. Add `loading="lazy"` below the fold. For character sprites, reserve space via CSS (`aspect-ratio` or fixed `w-*`/`h-*` Tailwind classes).

### R5 — Production port mismatch after Docker swap

**Probability:** Medium (downgraded after audit)  
**Impact:** High  
**Description:** Going from a Node server on port 3000 to nginx on port 80 changes the container's network contract. The four deploy files that reference atlas-ui (see `task-004-context.md`) must all be updated atomically with the Dockerfile change, or the rollout fails.

**Audit findings (2026-04-17):** No liveness/readiness/startup probes are defined on the atlas-ui Deployment, so no probe paths need to be updated. No PDBs, HPAs, or NetworkPolicies reference port 3000. The surface is four files: `k8s/atlas-ui.yaml`, `k8s/ingress.yaml`, `shared/routes.conf`, `compose/docker-compose.core.yml`.

**Mitigation:** Update all four files in the same commit as the Dockerfile change (Phase 6). Delete the `/_next/webpack-hmr` `location` blocks (Next-specific, useless under nginx-static). Keep the host port at 3000 in docker-compose (`3000:80`) to avoid disrupting any local-dev tooling hitting `localhost:3000`.

## Medium-risk items

### R6 — Multi-tenant cache invalidation (new invariant added by this migration)

**Probability:** Low-Medium  
**Impact:** High  

**Description:** Switching tenants must clear React Query cache so that stale data from tenant A isn't shown to tenant B. **Audit finding (2026-04-17):** no such invalidation runs today — `TenantProvider` only updates state + localStorage; `queryClient.clear()` is not called on tenant change. The migration is *adding* this behaviour for the first time, not preserving it. That also means today's codebase has a latent bug here (stale tenant-A data can leak into tenant-B views for any query whose key doesn't include the tenant). Risk is that the new `useEffect(() => queryClient.clear(), [activeTenant])` is wired wrong (wrong deps, runs on mount with `null` tenant, etc.) and either fails to clear or clears too aggressively.

**Mitigation:** Add a Vitest test for tenant switching — mock two tenants, call `setActiveTenant`, assert `queryClient.clear()` is called once. Assert it is *not* called on the initial mount when `activeTenant` is `null`. Manual verification (Phase 7): switch tenants in the UI, confirm that a list page re-fetches and shows tenant-B data rather than a stale tenant-A payload.

### R7 — ESLint flat config surfaces new errors blocking merge

**Probability:** Medium  
**Impact:** Low  
**Description:** Home-hub's ESLint config is stricter in some areas than `eslint-config-next` and may flag pre-existing atlas-ui code (unused vars with non-underscore prefix, `any` types, etc.).

**Mitigation:** Accept that Phase 5 may produce a long list of lint errors. Either fix them inline during migration (preferred) or file a follow-up task. Do NOT disable rules to paper over issues.

### R8 — `next-themes` dependency keeps Next.js in the tree

**Probability:** Low  
**Impact:** Low  
**Description:** The package name is misleading — `next-themes` is framework-agnostic and ships a React provider. Keeping it means a `next*` entry lingers in `package.json`, which the acceptance criteria disallow.

**Mitigation:** Swap to home-hub's `ThemeProvider` (a ~30-LOC wrapper using `document.documentElement.classList`). Drops the dep cleanly.

### R9 — Dev server proxy missing breaks local dev CORS

**Probability:** Low-Medium  
**Impact:** Low  
**Description:** Next.js dev server serves the app on the same origin as its API routes. Vite dev server (port 5173) hitting Go services (different origin/port) triggers CORS unless proxied.

**Mitigation:** Configure `vite.config.ts` `server.proxy` forwarding `/api/**` to the dev Go service URL. Document the proxy target in README.

### R10 — `BreadcrumbBar` relies on `usePathname` + segment metadata that App Router provides implicitly

**Probability:** Medium  
**Impact:** Low  
**Description:** The current `BreadcrumbBar` likely walks `usePathname()` segments and maps them to labels. React Router's `useLocation` gives the same path string, but any reliance on App Router segment introspection (`useSelectedLayoutSegments`) will not port directly.

**Mitigation:** Read `BreadcrumbBar` implementation in Phase 0. If it uses App Router-specific hooks, rewrite it to split `location.pathname` by `/` and look up labels from a route metadata map (typed constant in `src/lib/breadcrumbs.ts`).

## Low-risk items

### R11 — React Flow bundle size regression without Next's Turbopack code-splitting

**Probability:** Low  
**Impact:** Low  
**Description:** Turbopack's default splitting may produce smaller initial bundles than Vite's defaults for the NPC conversation page.

**Mitigation:** Accept default Vite chunking for v1. If the NPC page's bundle is notably larger, wrap the React Flow component in `React.lazy` + `Suspense` in a follow-up.

### R12 — Test setup differs (`@testing-library/jest-dom` import path)

**Probability:** Low  
**Impact:** Low  
**Description:** Under Vitest, `@testing-library/jest-dom/vitest` is the correct import path (not `@testing-library/jest-dom/extend-expect`).

**Mitigation:** Use the correct import in `src/test/setup.ts`. Verify one test uses a matcher like `toBeInTheDocument` to confirm setup worked.

### R13 — Follow-up cleanup items are forgotten

**Probability:** Medium  
**Impact:** Low  
**Description:** Items explicitly deferred (route-level `React.lazy`, unused primitive verification) may never get done if not recorded.

**Mitigation:** Maintain a "Deferred" section in this task folder or in `docs/TODO.md`. Every deferral is a line item, not just a comment.

## Accepted risks (not mitigated in this task)

- **Manual smoke testing only.** Regressions will be caught by manual smoke tests during the verification phase — acceptable because this is an internal admin dashboard with a small user set. Unit + integration coverage (Vitest) remains unchanged from today.
- **No rollback automation.** Rollback is "revert the merge commit." Acceptable given single-PR big-bang and no data changes.
- **No feature-flag rollout.** The migration ships as a hard cutover. Acceptable for the same reasons — internal tool, small user set, trivial revert path.
