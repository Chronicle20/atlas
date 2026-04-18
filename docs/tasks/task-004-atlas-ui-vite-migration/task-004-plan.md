# Task 004 — atlas-ui Vite + React Router Migration

Last Updated: 2026-04-18
Status: **Complete** — all seven phases merged; deferrals either closed or documented
Branch: `task-004` (ahead of `main`)
PR strategy: one PR, commits split per phase
Deferrals bucket: `docs/TODO.md` (no new task folders)

Final state (2026-04-18):
- 0 TypeScript errors across production and test files (strict flags on both)
- 471 Vitest tests passing / 0 skipped / 0 failed
- `vite build` green; 77 KB gzip main chunk
- Remaining open items are manual smoke tests + Playwright (explicitly out of scope)

## Index

This folder is the single source of truth for the migration. The planning work is split across four documents:

| File | Purpose |
|---|---|
| `prd.md` | Product requirements, scope, goals, functional requirements, acceptance criteria |
| `migration-plan.md` | Phased implementation plan with commit boundaries and time estimates |
| `risks.md` | Risk register (R1–R13) with probability, impact, and mitigation |
| `task-004-context.md` | Key files, decisions, dependencies, baseline measurements |
| `task-004-tasks.md` | Checklist for tracking execution progress across all phases |

Read `prd.md` first for the "what/why", then `migration-plan.md` for the "how", then `risks.md` for the failure modes. Use `task-004-tasks.md` to track progress during execution.

## Executive Summary

`services/atlas-ui` is a Next.js 16 App Router application that is effectively an SPA — 140+ files declare `"use client"`, no server components or route handlers exist, and the framework overhead is not earning its keep. This task rebuilds atlas-ui on the home-hub template (Vite 8 + `react-router-dom` v7 + Vitest + nginx static serving) in a single big-bang migration on a dedicated branch. Feature parity is the only correctness bar — no UX changes, no backend changes.

## Phases (see `migration-plan.md` for detail)

- **Phase 0 — Audit** (0.5d): grep baselines, test count, env var inventory, deploy manifest audit
- **Phase 1 — Scaffold** (0.5d): Vite config, `index.html`, `src/main.tsx`, `src/App.tsx` alongside Next.js
- **Phase 2 — Shared infra** (1d): port `components/`, `context/`, `hooks/`, `lib/`, `services/`, strip `"use client"`, replace `next/*` imports, shrink API client
- **Phase 3 — Pages** (2–3d): convert 46 pages to RR routes, wire `AppShell` + `<Outlet />`
- **Phase 4 — Data fetching** (1–2d): every page behind a React Query hook, delete `base.service.ts`
- **Phase 5 — Tests & lint** (1d): Jest → Vitest, ESLint flat config
- **Phase 6 — Infra cleanup** (0.5d): remove Next deps, nginx Dockerfile, deploy manifest updates
- **Phase 7 — Docs & verify** (0.5d): rewrite `CLAUDE.md`, smoke test every route

**Total: 7–9 developer-days.**

## Success Metrics

From `prd.md` §10 acceptance criteria. Highlights:

- Zero `next/*` imports and zero `"use client"` directives in `services/atlas-ui/src/`
- All 46 routes listed in `prd.md` §4.2 render and fetch data without errors
- `npm run test` passes with test count ≥ baseline (61 test files today)
- `npm run build` produces `dist/`; `docker build` yields an nginx image < 50 MB
- `lib/api/client.ts` < 700 LOC; `services/api/base.service.ts` deleted
- No `useEffect` + `fetch` patterns remain on any page

## Top Risks (see `risks.md` for full register)

1. **R1** — `useSearchParams` API mismatch between Next and RR v7 breaks filter/sort UI
2. **R3** — Shrinking `lib/api/client.ts` silently removes a primitive a page relies on
3. **R5** — Docker port/runtime swap (3000 → 80, Node → nginx) breaks deploy manifests *(scope now concrete — see `risks.md`)*
4. **R6** — Tenant-switch cache invalidation regression (correctness/security)

## Pre-execution audit results (2026-04-17)

- **Auth:** confirmed nil. No `next-auth`, no `getServerSession`, no route guards. Current `CLAUDE.md`'s auth sections are fiction.
- **Tenant header contract:** four headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`), SCREAMING_SNAKE_CASE. Preserve verbatim. Source: `services/atlas-ui/lib/headers.tsx`.
- **Tenant wiring:** today every service method calls `api.setTenant(tenant)` before each request (~30+ sites). Migration centralises this in a `TenantProvider` effect. Network behaviour unchanged, architecture simpler.
- **Tenant-switch cache invalidation:** **does not exist today.** Migration adds `queryClient.clear()` as a new invariant — R6 recast from "regression" to "new invariant".
- **Deploy scope:** four files (`k8s/atlas-ui.yaml`, `k8s/ingress.yaml`, `shared/routes.conf`, `compose/docker-compose.core.yml`). No probes, no PDBs/HPAs to update.
- **Public env vars:** only `NEXT_PUBLIC_ROOT_API_URL` → `VITE_ROOT_API_URL`.
- **Vite dev proxy target:** `http://localhost:${VITE_INGRESS_PORT:-8080}` (the compose `atlas-ingress` nginx).
- **Version alignment:** React 19.2.4, TS 6.x, Tailwind 4.2.x, npm — already matched between atlas-ui and home-hub. Only delta is `@tailwindcss/postcss` → `@tailwindcss/vite`. No version bumps in scope.
- **Image wrapper decision:** no `<MaplestoryImage>` — plain `<img>` at each call site.
- **Deferrals bucket:** `docs/TODO.md` exists; add a new `### atlas-ui Frontend` section at the end.
- **E2E scope:** no Playwright work in this task (no existing suite, not in scope).

## Out of Scope

See `prd.md` §2 non-goals. In particular: no auth work, no UX redesign, no state-management swap, no SSR/SSG, no new component library.
