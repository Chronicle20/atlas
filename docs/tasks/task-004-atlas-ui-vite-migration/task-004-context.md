# Task 004 — Context

Last Updated: 2026-04-17

Companion to `task-004-plan.md`. Captures the surrounding information needed to execute the migration without re-deriving it.

## Key Locations

### Source (being rewritten)
- `services/atlas-ui/` — the project being migrated
- `services/atlas-ui/app/` — current Next.js App Router tree (46 pages, deleted end of Phase 3)
- `services/atlas-ui/components/` — shared components (copied into `src/`)
- `services/atlas-ui/context/tenant-context.tsx` — multi-tenant provider
- `services/atlas-ui/lib/api/client.ts` — 1801-LOC API client (shrink to < 700 LOC)
- `services/atlas-ui/services/api/base.service.ts` — 499 LOC, to be deleted
- `services/atlas-ui/lib/hooks/api/` — existing React Query hooks (keep and extend)
- `services/atlas-ui/next.config.ts` — deleted in Phase 6
- `services/atlas-ui/CLAUDE.md` — rewritten in Phase 7 (currently describes features that do not exist)

### Reference template
- `/home/tumidanski/source/home-hub/frontend/` — Vite 8 + RR v7 reference app
- `frontend/vite.config.ts` — template config
- `frontend/src/App.tsx` — provider stack pattern (Toaster, Routes, TenantProvider, ThemeProvider, QueryProvider, BrowserRouter, inside → out)
- `frontend/src/lib/api/client.ts` — 572-LOC client to adapt
- `frontend/src/components/providers/theme-provider.tsx` — next-themes replacement
- `frontend/Dockerfile` + `frontend/nginx.conf` — production runtime template
- `frontend/eslint.config.js` — flat ESLint config template

### Deploy surface (audit complete)
Four files reference atlas-ui, all in scope for Phase 6:
- `deploy/k8s/atlas-ui.yaml` — Deployment (containerPort 3000) + Service (port 3000). Env: `NEXT_PUBLIC_ROOT_API_URL`. No liveness/readiness/startup probes defined.
- `deploy/k8s/ingress.yaml` — two `location` blocks proxy to `atlas-ui:3000`: `/_next/webpack-hmr` (delete entirely — Next HMR only) and the catch-all `location /`.
- `deploy/shared/routes.conf` — mirrors ingress.yaml (HMR block + catch-all). Delete HMR block.
- `deploy/compose/docker-compose.core.yml` — `3000:3000` port mapping, `NEXT_PUBLIC_ROOT_API_URL` env.

No PDBs, HPAs, NetworkPolicies, or TLS terminators reference port 3000 beyond these four files.

## Baseline Measurements (captured 2026-04-17)

| Metric | Value | Used as |
|---|---|---|
| `"use client"` occurrences | 142 across 140 files | Phase 2 completion bar (must reach 0) |
| `from "next/*"` imports | 93 occurrences across 66 files | Phase 2 completion bar (must reach 0) |
| Test files (`*.test.ts(x)`) | 61 | Phase 5 minimum passing bar |
| `lib/api/client.ts` LOC | 1801 | Shrink target: < 700 |
| `services/api/base.service.ts` LOC | 499 | Delete target: 0 |
| Pages under `app/` | 46 | Phase 3 coverage target |

Recapture at execution start in case the tree drifts.

## Key Decisions

1. **Big-bang on a single branch, single PR**, not incremental. Dual-router intermediate states are worse than a longer-lived branch. Multiple commits inside the PR are fine (one per phase is the intended split). Rollback = revert the merge.
2. **home-hub is the template**, not an abstract best-practice shape. Match its file layout, provider order, client shape, and lint config verbatim wherever possible to minimize invention.
3. **Feature parity only**. No UX, auth, or state-management work rides along. Every deferral goes into `docs/TODO.md`.
4. **Drop `next/image` optimization layer; no wrapper component**. `maplestory.io` sprites are the only meaningful image consumer; plain `<img>` at each call site with explicit `width`/`height` and `loading="lazy"` below the fold. A `<MaplestoryImage>` wrapper was considered and **rejected** — it would add nothing beyond what the browser does and counts as speculative abstraction. Revisit only if sprite-version pinning or CDN layering becomes a real need.
5. **Drop `next-themes`** even though it is framework-agnostic — keeping it leaves a `next*` entry in `package.json` and violates the acceptance criteria. Swap to home-hub's `ThemeProvider`.
6. **Shrink API client pragmatically**, not to an LOC target. < 700 LOC is a soft goal. If `upload`/`download`/`dedup` turn out to be used, they stay.
7. **Per-page React Query hooks**. No more `useEffect` + service call. Hooks live in `src/lib/hooks/api/use-<resource>.ts`; query keys in `src/lib/hooks/api/query-keys.ts`.
8. **Production runtime: nginx static**. No Node in the prod image. Container port 80 replaces 3000 — deploy manifests updated in the same PR. In `docker-compose.core.yml` the host port stays 3000 for local-dev compatibility (`3000:80`).
9. **Auth confirmed nil and stays nil.** No session, no route guards, no `next-auth` — audit in 2026-04-17 confirmed every match of `auth|session|login` in atlas-ui is a domain term (login-history page, `LoginTenantConfig`, game account sessions, 401 error mapping). Home-hub's auth/refresh/redirect code path is deleted from the ported client, not adapted.
10. **Tenant contract = four SCREAMING_SNAKE_CASE headers**, not one. See below.
11. **Tenant wiring is centralised, not per-call.** Today every service method calls `api.setTenant(tenant)` before each request (~30+ redundant call sites). The migration moves this to a single `TenantProvider` effect that calls `apiClient.setTenant(activeTenant)` on change. Service signatures drop the `tenant` parameter. Network behaviour is unchanged.
12. **Cache invalidation on tenant switch is new, not preserved.** Today no `queryClient.clear()` runs on tenant change — a latent bug. The migration adds it in the same `TenantProvider` effect. R6 is recast from "regression" to "new invariant".
13. **Vite dev proxy default target:** `http://localhost:${VITE_INGRESS_PORT:-8080}` for `/api/**`, matching the compose `atlas-ingress` published port. Developers running against remote stacks override via env.
14. **Version alignment confirmed:** atlas-ui and home-hub already match on React 19.2.4, TS 6.x, Tailwind 4.2.x, npm. Only delta is `@tailwindcss/postcss` + `postcss.config.mjs` → `@tailwindcss/vite` (no PostCSS config). No version bumps in scope.

## Dependencies & External Systems

- **No Go service changes.** Every endpoint called today continues to be called with the same URL/method/headers/shape.
- **Tenant header contract — four headers, SCREAMING_SNAKE_CASE, preserve verbatim.** Source: `services/atlas-ui/lib/headers.tsx`:
  - `TENANT_ID` — tenant UUID
  - `REGION` — tenant region string
  - `MAJOR_VERSION` — integer as string
  - `MINOR_VERSION` — integer as string
  Do **not** rename to `X-Tenant-Id` or similar; Go services match on the exact names.
- **atlas-tenants** integration unchanged: tenant list + per-tenant config fetches stay on the existing endpoints.
- **React Flow** (NPC conversation graph) stays on the same package version; only the import surrounding it changes.

## Open Questions

Resolved during pre-execution audit (2026-04-17):
- ~~What's currently in `deploy/` for atlas-ui?~~ → Inventoried above.
- ~~Is deploy ingress hard-coded to port 3000?~~ → Yes, four locations across four files. All in scope.
- ~~Does atlas-ui have an auth surface?~~ → No. Confirmed nil.
- ~~Tenant header name?~~ → Four headers (see above).

Still open, resolve during execution:
1. Any `next/font`, `next/headers`, `next/dynamic` usage beyond what's catalogued? — Confirm in Phase 0 re-audit.
2. Does any service module depend on `base.service.ts` primitives beyond query-building? — Resolve in Phase 0 audit.
3. Any Next-implicit CSS behaviour (CSS Modules, `globals.css` load order)? — Check in Phase 2.
4. Any test using `next/jest` transformer? — Resolve in Phase 5.

## docs/TODO.md format (for Phase 7 deferrals)

`docs/TODO.md` already exists at the repo root `docs/` folder. Format is Markdown checklists grouped by area:

```
### Area Name
- [ ] item description (optional file:line)
```

Phase 7 adds a new `### atlas-ui Frontend` section at the end of the file with deferrals like route-level `React.lazy`, `next-themes` wrapper edge-case review, and unused-client-primitive audit results.

## Rollback

- Branch `atlas-ui-vite-migration` stays off `main` until verification passes.
- No database or API changes → rollback is `git revert <merge-sha>`.
- Deploy pipeline must be able to pin to the pre-merge image tag indefinitely (verify this exists before merging).
