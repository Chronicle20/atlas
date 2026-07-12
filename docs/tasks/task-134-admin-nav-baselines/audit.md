# Plan Audit — task-134-admin-nav-baselines

**Plan Path:** docs/tasks/task-134-admin-nav-baselines/plan.md
**Audit Date:** 2026-07-04
**Branch:** task-134-admin-nav-baselines
**Base Branch:** main (plan commit `6260e82cfa` → HEAD `207b35a12d`)

## Plan Adherence

### Executive Summary

All 17 plan tasks were faithfully implemented; there are no skipped, partial, or
deferred tasks. The 16 implementation commits map one-to-one onto the plan's
task/commit list, and the produced files match the plan's declared file lists
and interface contracts. Both affected components are green:
atlas-data (`go build`/`go vet`/`go test -race ./...`) is clean and atlas-ui
(`npx tsc -b --noEmit` + `vitest run`, 867 tests across 100 files) passes.

### Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | `minio.Client.List` + `ObjectInfo` | DONE | `storage/minio/client.go:93` (`ObjectInfo`), `:103` (`List`); commit `5c6847e708` |
| 2 | `parseDumpKey` canonical dump-key parsing | DONE | `baseline/list.go:107` (named returns per style commit `29151bd2f6`); tests `baseline/list_test.go`; commit `5d10c41697` |
| 3 | `Lister` + `ListItemModel` | DONE | `baseline/list.go:29` (`Lister`), `:40` (`List`), `:84` (`readSha`); `baseline/rest.go:63` (`ListItemModel`), `:73` (`GetName`→`"baselines"`); commit `6d41645655` |
| 4 | `GET /data/baselines` handler + route | DONE | `baseline/handler.go:28` (route), `:68` (`listInner`), `:72` 503-gate, `:75` 403 operator-gate, `:88` `MarshalResponse[[]ListItemModel]`; commit `42ec9d1528` |
| 5 | atlas-data verification gates | DONE (verification-only) | Re-run this audit: build/vet/test all clean (see Build & Test Results) |
| 6 | `isDeploymentRoute` predicate | DONE | `src/lib/deployment-routes.ts` (`DEPLOYMENT_ROUTE_PREFIXES` = templates/tenants/services/baselines, prefix-boundary guard); commit `ffa151c4b4` |
| 7 | `canonicalHeaders` + shared `formatBytes` | DONE | `src/lib/headers.tsx:19` (`CANONICAL_TENANT_ID`), `:21` (`CanonicalSelection`), `:32` (`canonicalHeaders`, bakes `X-Atlas-Operator:1`); `src/lib/format.ts`; commit `be93e0f187` |
| 8 | canonical service fns + `listBaselines` | DONE | `seed.service.ts:203-216` (four `*Canonical*` fns, `scope=shared`); `baseline.service.ts:89` (`listBaselines` → `/api/data/baselines`); commit `283033c53f` |
| 9 | migrate `publish` off Tenant | DONE | `baseline.service.ts:66` (`publish(sel: CanonicalSelection)`); `useBaseline.ts` — `usePublishBaseline` deleted, `useRestoreBaseline` kept; SetupPage publish row removed; commit `eec913979d` |
| 10 | de-scope tenant path; delete ScopeToggle | DONE | `seed.service.ts:184-197` (tenant fns hard-code `scope=tenant`, no operator header); `ScopeToggle.tsx` + test deleted (dir now only `SetupRow.tsx`); `SetupPage.tsx:257` titled "Setup", no scope toggle/publish; commit `59cca88e29` |
| 11 | canonical React Query hooks | DONE | `src/lib/hooks/api/useCanonicalData.ts` — all 6 hooks + `baselinesKey` exported (lines 17,19,29,39,56,72,79); commit `e030529072` |
| 12 | sidebar regroup + Deployment treatment | DONE | `app-sidebar.tsx:43` `sidebarItems` ordered Operations/Security/Setup/Deployment; Deployment `separated:true` (`:84`), `caption:"Applies to all tenants"` (`:85`), children Templates/Tenants/Services/Baselines; breadcrumb Setup label; commit `a81ec61299` |
| 13 | scope-aware tenant switcher | DONE | `app-tenant-switcher.tsx:27` inert branch on `isDeploymentRoute`, renders "Deployment-wide"/"tenant selection inactive" (`:39-40`), no write path; commit `8b5759bc16` |
| 14 | deployment scope banner in shell | DONE | `common/deployment-scope-banner.tsx` (self-conditions, exact copy "Changes on this page affect all tenants."); mounted once `app-shell.tsx:28`; commit `6093059973` |
| 15 | `BaselineTargetPicker` | DONE | `features/baselines/BaselineTargetPicker.tsx` exports `dedupeSelections`/`parseCustomSelection`/`selectionKey`; commit `8a250d2960` |
| 16 | BaselinesPage + route + breadcrumb | DONE | `pages/BaselinesPage.tsx` (upload→process→publish workflow, empty state, re-publish confirm copy `:304`, "Replace Baseline" `:314`); `App.tsx:20,81` lazy import + route; `routes.ts:42` + `:458` (`BASELINES`); commit `207b35a12d` |
| 17 | full verification sweep | DONE (verification-only) | Re-run this audit: atlas-ui + atlas-data green; residue greps (`ScopeToggle`, `usePublishBaseline`, TODO) return no matches |

**Completion Rate:** 17/17 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Skipped / Deferred Tasks

None.

### Build & Test Results

| Component | Build/Typecheck | Tests | Notes |
|-----------|-----------------|-------|-------|
| atlas-data | `go build ./...` PASS, `go vet ./...` PASS | `go test -race ./...` PASS | all packages ok/cached, incl. `atlas-data/baseline` |
| atlas-ui | `npx tsc -b --noEmit` PASS | `vitest run` PASS | 867 tests / 100 files, 0 failures |

Note: `docker buildx bake atlas-data` and `tools/redis-key-guard.sh` (plan Task 5/17
gates) were not re-run in this audit; no shared lib or `go.mod` was added
(the change is confined to existing `atlas-data` packages) and no new raw
go-redis usage was introduced, so neither gate is at risk, but the executor's
Task 17 run remains the authority.

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

### Action Items

None. All 17 tasks implemented as specified with matching interfaces and passing
gates.

---

# Backend Audit — atlas-data (task-134 Go changes)

- **Service Path:** services/atlas-data/atlas.com/data
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-04
- **Scope:** Go changes only (Tasks 1–5) — `storage/minio/client.go`, `baseline/list.go`, `baseline/list_test.go`, `baseline/rest.go`, `baseline/handler.go`, `baseline/handler_test.go`. Range `6260e82cfa` → `207b35a12d`.
- **Build:** PASS (`go build ./...`)
- **Vet:** PASS (`go vet ./...`)
- **Tests:** PASS (`go test -race ./...` and `go test ./... -count=1`, all packages ok; `atlas-data/baseline` green)
- **Overall:** PASS

## Package Classification

`baseline/` is a **REST-exposed support/operations package**, not a standard DOM
domain package: it has no `model.go`, `processor.go`, or `administrator.go`, and
predates this change. Its operation structs (`Publisher`/`Restorer`/`Lister`)
play the processor role and the REST models in `rest.go` are marshal-only outputs
plus flat JSON:API inputs. The DOM checklist items that presuppose an immutable
domain model + provider/administrator layering (DOM-01/02/03/04/05/10/11/16) are
**N/A**. The applicable REST/JSON:API, logger, handler-hygiene, and testing items
were verified and all PASS.

## Checklist Results — baseline (Go changes)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Operation accepts `FieldLogger` (not `*logrus.Logger`) | PASS | `baseline/list.go:33` `L logrus.FieldLogger`; `objectStore` iface `:23` |
| DOM-07 | Handlers pass `d.Logger()`, no `StandardLogger()` | PASS | `baseline/handler.go:79` (`L: d.Logger()`); no `logrus.StandardLogger` in package |
| DOM-08 | POST→`RegisterInputHandler[T]`, GET→`RegisterHandler` | PASS | `handler.go:24-25` (publish/restore `RegisterInputHandler[T]` + `Methods(POST)`); `:28` GET `RegisterHandler` |
| DOM-09 | No dropped errors on transform/marshal path | PASS | `list.go:41-42,88-97` all errors checked; no `_, _ :=`/`_ =` in changed files |
| DOM-12 | No `os.Getenv()` in handlers | PASS | grep `os.Getenv` in `baseline/` → 0 matches |
| DOM-14 | Handler doesn't call providers directly | PASS | `listInner` (`handler.go:79`) delegates to `Lister.List`; all enumeration/parse/sha logic in `list.go`, handler is thin |
| DOM-15 | No `db.Create/Save/Delete` in handler | PASS | `listInner` never touches `db`; GET path is read-only against MinIO |
| DOM-17 | Domain error → HTTP status mapping | PASS | `handler.go:71-83`: nil-mc→503, non-operator→403, list error→500. No 400/404/409 semantics exist for an enumerate op (empty bucket → `data:[]`, not 404) |
| DOM-18 | JSON:API interface on REST models | PASS | `rest.go:73-75` `ListItemModel` `GetName()=="baselines"`, `GetID()`, `SetID()` |
| DOM-19 | Request/response models flat (no nested Data/Type/Attributes) | PASS | `rest.go:13-18,44-50,63-71` all flat; `Id` tagged `json:"-"` |
| DOM-20 | Table-driven tests | PASS | `list_test.go:18` (`cases := []struct`), `:40` (slice), `:183` (map-driven); map-backed fake store `:64` |
| DOM-21 | No duplication of atlas-constants types | PASS (N/A) | `Region string` + `MajorVersion/MinorVersion int` are tenant region/version primitives (mirror existing `PublishInputModel`/`RestoreInputModel`); no atlas-constants equivalent for tenant region/version. No item/inventory/world/channel id reinvention introduced |
| SUB-04 | No manual JSON parsing of request body | PASS | No `json.NewDecoder`/`json.Unmarshal` of body; `io.ReadAll` at `list.go:88` reads the MinIO sha256 **sidecar object**, not the HTTP body |
| EXT-01 | JSON:API marshal-only model needs no relationship setters | PASS (N/A) | `ListItemModel` is response-only (never api2go-unmarshaled); `SetToOne/ManyReferenceID` not required |

## Requirement-Specific Verification

| Requirement | Status | Evidence |
|-------------|--------|----------|
| `PublishedAt` from MinIO `LastModified`, never the epoch-zero tar header | PASS | `list.go:61` `o.LastModified.UTC().Format(time.RFC3339)`; handler never reads tar internal `publishedAt` |
| Handler must NOT read tenant id for authz; gates on `X-Atlas-Operator: 1` only | PASS | `handler.go:75` reads only `X-Atlas-Operator` header; nil-UUID synthetic tenant accepted and ignored (`:64-67` comment; no `tenant.*FromContext` in `listInner`) |
| Gate order nil-mc→503, non-operator→403, then list | PASS | `handler.go:71-78` in that order; tests `handler_test.go:118` (503), `:133` (403) |
| Empty bucket marshals as `"data": []` (non-nil slice) | PASS | `list.go:45` `make([]ListItemModel, 0)`; test `list_test.go:99` asserts non-nil empty |
| `SizeBytes` accurately reflects baseline footprint | PASS | A baseline = one `documents.dump` **tar** containing all tables (`publish.go:56,106`); `SizeBytes` = that tar's object size (`list.go:62`), so it is the full payload, not an undercount |
| One bad key never fails the listing (skip-and-warn) | PASS | `list.go:50-54,81-98` unparseable keys / missing-or-malformed sidecar degrade to skip/`Sha256:""`; tests `list_test.go:160,181` |

## Security Notes (non-blocking)

- atlas-data is not an auth/token service, so SEC-01/02/03 do not apply.
- The `X-Atlas-Operator: 1` header is the authorization gate. `listInner`
  mirrors the pre-existing `publishInner`/`restoreInner` gate exactly; it is a
  read-only enumerate and strictly less sensitive than the sibling
  publish/restore endpoints it copies. The header must be set/stripped at the
  ingress boundary — that is a pre-existing platform assumption for this whole
  package, not a task-134 regression. No hardcoded secrets introduced.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.

**Backend verdict:** PASS. Build/vet/test green; every applicable DOM/SUB/EXT
check and every task-specific requirement is satisfied with file:line evidence.

---

# Frontend Audit — task-134-admin-nav-baselines

- **Audit Scope:** atlas-ui TS/React changes (Tasks 6–16), commit range `6260e82cfa` → `207b35a12d`
- **Guidelines Source:** frontend-dev-guidelines skill (FE-* checklist)
- **Date:** 2026-07-04
- **Toolchain:** node v22.22.2 (nvm), run from `services/atlas-ui/`
- **Typecheck (`npx tsc -b --noEmit`):** PASS (exit 0)
- **Tests (`npm run test` → `vitest run`):** 867 passed / 867 (100 test files), 0 failed
- **Overall:** NEEDS-WORK (1 non-blocking FE-06 finding; typecheck + suite green)

> Note: atlas-ui is a Vite + react-router SPA (not Next.js). The guidelines are
> written against a Next.js layout (`app/`, default-export `page.tsx`); the FE-08
> "named export" rule is satisfied here by named page exports wrapped in
> `React.lazy`, and the "no default export" rule applies cleanly.

## File Inventory

- `src/lib/deployment-routes.ts` — **Other** (route predicate, single source of truth)
- `src/lib/headers.tsx` — **Other** (header builders; `CANONICAL_TENANT_ID`, `canonicalHeaders`)
- `src/lib/format.ts` — **Other** (`formatBytes`)
- `src/lib/breadcrumbs/routes.ts` — **Other** (breadcrumb config; added `/baselines`)
- `src/services/api/seed.service.ts` — **Service**
- `src/services/api/baseline.service.ts` — **Service**
- `src/lib/hooks/api/useSeed.ts` — **Hook**
- `src/lib/hooks/api/useCanonicalData.ts` — **Hook** (new)
- `src/lib/hooks/api/useBaseline.ts` — **Hook** (`usePublishBaseline` removed)
- `src/components/app-sidebar.tsx` — **Component**
- `src/components/app-tenant-switcher.tsx` — **Component**
- `src/components/common/deployment-scope-banner.tsx` — **Component** (new)
- `src/components/features/navigation/app-shell.tsx` — **Component**
- `src/components/features/baselines/BaselineTargetPicker.tsx` — **Component** (new)
- `src/pages/BaselinesPage.tsx` — **Page** (new)
- `src/pages/SetupPage.tsx` — **Page**
- `src/App.tsx` — route registration
- Deleted: `src/components/features/setup/ScopeToggle.tsx` (+ test)
- Tests: 10 new/updated `__tests__` files (sidebar, switcher, banner, picker, BaselinesPage, SetupPage, seed.service, baseline.service, useSeed, useBaseline)

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | Grep `: any`/`as any` over all 16 in-scope source files → zero matches |
| FE-02 | No manual class concatenation | PASS | No `className={"…" + …}` in changed files; `cn()` used where merging needed (e.g. app-tenant-switcher uses `pointer-events-none opacity-70` static string); app-sidebar.tsx:127 uses a single ternary value, not concatenation |
| FE-03 | No direct API client in components | PASS | BaselinesPage/SetupPage consume hooks only; no `@/lib/api/client` import in any changed component/page |
| FE-04 | No inline Zod in components | PASS | No `z.*` in BaselineTargetPicker or any changed component; custom-selection validation is regex (`parseCustomSelection`, BaselineTargetPicker.tsx:54-63) |
| FE-05 | No spinners for content loading | PASS | `animate-spin` only inside action buttons (BaselinesPage.tsx:236,262,287; SetupPage.tsx:290,317,345,377) — all submit/action affordances |
| FE-06 | No hardcoded colors | FAIL | `components/common/deployment-scope-banner.tsx:16` — `border-amber-500/50 bg-amber-500/10 text-amber-900 dark:text-amber-200 [&>svg]:text-amber-600` uses raw Tailwind palette instead of semantic tokens. Non-blocking (see finding). |
| FE-07 | No state mutation | PASS | Immutable throughout; `dedupeSelections` (BaselineTargetPicker.tsx:30-48) builds a fresh Map and sorts a spread copy `[...map.values()]`; no in-place state mutation |
| FE-08 | No default exports for components | PASS | All changed components/pages use named exports (e.g. BaselinesPage.tsx:37, deployment-scope-banner.tsx:12); App.tsx:20 adapts via `.then(m => ({default: m.BaselinesPage}))` for `lazy` |
| FE-09 | Tenant guard in hooks | PASS | Tenant-scoped hooks in useSeed.ts use `useTenant()` + `enabled: !!activeTenant` (e.g. lines 178,189,202); canonical (deployment-scoped, tenant-agnostic) hooks guard on the selection: `enabled: !!sel` (useCanonicalData.ts:23,35) and throw when `sel` is null in mutation fns (lines 43-46,60-63,83-86) |
| FE-10 | Tenant ID in query keys | PASS | Tenant keys embed `tenantId` (useSeed.ts:37-46) with `['…','none']` fallback; canonical keys embed region/major/minor (useCanonicalData.ts:13-16); `baselines` list is deployment-wide, correctly tenant-free (useCanonicalData.ts:17) |
| FE-11 | Error handling surfaces to user | PASS (with note) | Mutation errors are typed `Error` via RQ generics and surfaced via `toast.error(error.message)` (BaselinesPage.tsx:86,97; SetupPage.tsx:123,142); WZ-upload errors routed through shared `showWzUploadErrorToast` (useSeed.ts:26-35). No raw `catch(unknown)` left unhandled. `createErrorFromUnknown` is not used, but it is unnecessary here since errors arrive pre-typed — deviation noted, not a defect. |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape / decoding | PASS | `listBaselines` decodes `body.data[].attributes` from the JSON:API envelope into flat `Baseline` DTOs (baseline.service.ts:97-98); `fetchJsonApi` reads `body.data.attributes` (seed.service.ts:103-104). Seeder status endpoints emit plain JSON (documented, seed.service.ts:70-78) and are read directly — correct. |
| FE-13 | Service pattern | PASS | `SeedService`/`BaselineService` follow the documented direct-client/`fetch` pattern (needed for multipart + scope query-param + operator header); singletons exported (seed.service.ts:295, baseline.service.ts:102) |
| FE-14 | Query keys use `as const` | PASS | All factories `as const` (useSeed.ts:37-46, useCanonicalData.ts:13-17) |
| FE-15 | Forms use RHF + zodResolver | N/A | No react-hook-form form introduced. BaselineTargetPicker is a controlled selection widget (no submit/mutation of a form object), not a create/update form. |
| FE-16 | Schema in `lib/schemas/` + inferred type | N/A | No Zod schema introduced (see FE-15). |

## Task-Specific Requirements

| Requirement | Status | Evidence |
|-------------|--------|----------|
| POST to input-handler endpoints wraps `{ data: { type, attributes } }`, publish type `baselinePublishes` | PASS | baseline.service.ts:72-82; verified by baseline.service.test.ts:114-121 |
| Canonical requests carry `X-Atlas-Operator: 1` + nil-UUID tenant | PASS | headers.tsx:32-39 (baked into `canonicalHeaders`, cannot be assembled without it); verified baseline.service.test.ts:108-113 |
| React Query: enabled guards on null selection | PASS | useCanonicalData.ts:23,35 (`enabled: !!sel`); mutation fns throw on null sel |
| React Query: invalidation | PASS | publish invalidates `baselinesKey` + canonical data status (useCanonicalData.ts:88-92); upload/process invalidate their scoped keys |
| Rules of hooks: inert switcher early-return AFTER all hooks | PASS | app-tenant-switcher.tsx — `useSidebar` (19), `useTenant` (20), `useState` (21), `useLocation` (22) all precede the `isDeploymentRoute` return at line 27 |
| Inert switcher never writes TenantContext (render-only) | PASS | Inert branch (app-tenant-switcher.tsx:27-47) contains no `setActiveTenant`/localStorage; test asserts `setActiveTenant` not called (app-tenant-switcher.test.tsx:75-78) |
| `isDeploymentRoute` single source of truth (banner + switcher agree) | PASS | Both consume the same predicate (deployment-scope-banner.tsx:14, app-tenant-switcher.tsx:27); sidebar sync test enforces it (app-sidebar.test.tsx:50-60) |
| publish de-scoped to `CanonicalSelection` (no tenant path to scope=shared) | PASS | `publish(sel: CanonicalSelection)` (baseline.service.ts:66); tenant `uploadWzFiles`/`getWzInputStatus` hardcode `scope=tenant` (seed.service.ts:184-197) — structurally incapable of shared |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | New/updated tests for sidebar, switcher, banner, picker, BaselinesPage, SetupPage, both services, both hooks (10 files) |
| FE-18 | Mocks updated when services changed | PASS | baseline.service.test.ts asserts new `publish(sel)`/`listBaselines` signatures + envelope + operator header; BaselinesPage.test.tsx mocks the canonical hooks and asserts real gating/confirm-dialog behavior, not tautologies |

## Findings

### FE-06 (non-blocking): hardcoded palette colors in the deployment scope banner
- **File:** `src/components/common/deployment-scope-banner.tsx:16`
- **Detail:** The warning `Alert` is styled with raw Tailwind palette classes (`border-amber-500/50`, `bg-amber-500/10`, `text-amber-900`, `dark:text-amber-200`, `[&>svg]:text-amber-600`) rather than semantic theme tokens — the documented anti-pattern #8.
- **Severity:** Minor / non-blocking. The shadcn default theme ships no `warning` token, and the codebase already uses `amber-*` for warning callouts in dozens of pre-existing components (ItemCashShopWidget, MonsterMesoWidget, ConversationEditorPanel, etc.), all with explicit `dark:` variants — this instance follows that convention and handles dark mode. Recommend either (a) introduce a semantic `--warning` token and reuse it, or (b) accept as an established codebase convention and waive.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- FE-06 — `deployment-scope-banner.tsx:16` uses raw amber palette classes instead of semantic tokens (consistent with existing codebase convention; owner may waive).

**Frontend verdict:** NEEDS-WORK (strictly: one FE-06 FAIL prevents overall PASS),
but the sole finding is a low-severity styling item that matches established
codebase convention. Typecheck (`tsc -b`) is clean and the full suite (867/867)
passes. Every multi-tenancy, JSON:API-envelope, React-Query, and rules-of-hooks
requirement is satisfied with file:line evidence.
