# Frontend Audit — task-189-tenant-config-seed-provisioning

- **Audit Scope:** `git diff --name-only 24c78aa85..HEAD -- 'services/atlas-ui/*'` (6 files)
- **Guidelines Source:** frontend-dev-guidelines skill + `services/atlas-ui/CLAUDE.md` (Vite/Vitest overrides — this project is Vite + Vitest, not Next.js + Jest; `pages/*.tsx` are named exports, not App Router)
- **Date:** 2026-08-03
- **Build:** PASS
- **Tests:** 1383 passed, 0 failed (192 test files, `npm test` → `vitest run`)
- **Overall:** NEEDS-WORK

Two items from the dispatch brief are treated as pre-adjudicated, not re-litigated here:
1. Three independent rows (not one grouped row) — PRD/context.md §4 decision.
2. New code deliberately mirrors the eight pre-existing seed rows' idioms. Where a mirrored idiom is itself weak, it is marked **(pre-existing/inherited)** below and does not count toward Blocking.

## Build & Test Results

```
$ npm run build
✓ built in 959ms
(chunk-size warning only, pre-existing — ConversationEditorPanel 1.65MB — unrelated to this diff)

$ npm test
 Test Files  192 passed (192)
      Tests  1383 passed (1383)
   Duration  20.27s
```

## File Inventory

- `services/atlas-ui/src/lib/hooks/api/useSeed.ts` — Hook (React Query hooks + query key factories)
- `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx` — Test
- `services/atlas-ui/src/services/api/seed.service.ts` — Service (direct API client pattern)
- `services/atlas-ui/src/services/api/__tests__/seed.service.test.ts` — Test
- `services/atlas-ui/src/pages/SetupPage.tsx` — Page
- `services/atlas-ui/src/pages/__tests__/SetupPage.test.tsx` — Test

## Anti-Pattern Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -n ': any\b\|as any\b'` over all 6 files — zero matches |
| FE-02 | No manual class concatenation | PASS | `grep -n 'className={"'` over all 6 files — zero matches; `SetupPage.tsx` uses plain string literals / template-literal badge strings, no conditional class concatenation was added |
| FE-03 | No direct API client calls in components | PASS | `pages/SetupPage.tsx` has no `import ... from "@/lib/api/client"`; all three new seed calls route through `seedService` (`SetupPage.tsx:78-80` → `useSeedTransportRoutes/Vessels/InstanceRoutes` → `seed.service.ts:224-234`) |
| FE-04 | No inline Zod schemas in components | PASS (N/A) | No `z.object(`/`from "zod"` in `SetupPage.tsx`; this change adds no forms |
| FE-05 | No spinners for content loading | PASS | `SetupPage.tsx:453` uses `<Loader2 className="h-4 w-4 animate-spin" />` only inside the per-row `Seed` submit `<Button>` (mirrors `SetupPage.tsx:361,391,419` for the pre-existing rows), never for page/content loading |
| FE-06 | No hardcoded colors | PASS | `grep -nE 'bg-(white\|black\|gray-[0-9]+\|red-[0-9]+\|green-[0-9]+\|blue-[0-9]+)'` over all 6 files — zero matches; only semantic classes (`text-muted-foreground`, `text-2xl font-bold tracking-tight`, etc.) |
| FE-07 | No state mutation | PASS | `grep -n '\.push(\|\.splice(\|\.sort('` over all 6 files — zero matches; `seedRows` in `SetupPage.tsx:198-320` is a freshly-built array literal each render, not mutated |
| FE-08 | No default exports for components | PASS | `SetupPage.tsx:66` — `export function SetupPage()`, named export; `grep -n 'export default function'` — zero matches |
| FE-09 | Tenant guard in hooks | PASS | Query hooks: `useSeed.ts:440-450` (`useTransportRoutesSeedStatus`), `:456-466` (`useTransportVesselsSeedStatus`), `:472-482` (`useInstanceRoutesSeedStatus`) all call `useTenant()` and set `enabled: !!activeTenant`. Mutation hooks: `useSeed.ts:201-213, 215-231, 233-245` all call `useTenant()` (used to guard the `onSuccess` invalidation). **(pre-existing/inherited)**: like all 8 sibling seed mutations, the mutation's `mutationFn` itself takes no tenant argument and relies on the API client's globally-set tenant (via `TenantProvider`'s `api.setTenant`) rather than an explicit parameter — this is a real deviation from `patterns-service-layer.md`'s stated rule ("every tenant-scoped method takes tenant as first parameter") but it is the file's established pattern (`seedDrops()`, `seedGachapons()`, etc. at `seed.service.ts:188-218` are identical), faithfully mirrored, not introduced here. |
| FE-10 | Tenant ID in query keys | PASS | `useSeed.ts:62-67` — `transportRoutesSeedStatusKey`, `transportVesselsSeedStatusKey`, `instanceRoutesSeedStatusKey` all key on `tenantId: string`, and the 3 query hooks (`useSeed.ts:441-444, 457-460, 473-476`) fall back to a literal `"none"` tenant segment when `activeTenant` is null — functionally equivalent to the documented `tenant?.id \|\| 'no-tenant'` idiom, consistent with the other 8 status keys in the same file |
| FE-11 | Error handling with `createErrorFromUnknown` | WARN (pre-existing/inherited) | `grep -n '\.catch('` over all 6 files — zero matches (this codebase's seed mutations rely on React Query's built-in error state, not raw `.catch()`, so the literal FE-11 grep doesn't directly apply). However: `SetupPage.tsx:115-118` (`handleSeed`), the handler used by **all 11** seed rows including the 3 new Transport rows, fires `mutation.mutate()` with only an immediate `toast.info("Seeding ${label}...")` — there is no `onError` callback anywhere in the call chain (not in `handleSeed`, not in `useSeedTransportRoutes`/`useSeedTransportVessels`/`useSeedInstanceRoutes` at `useSeed.ts:201-245`, and not in any of the 8 sibling seed-mutation hooks either). If `seedService.seedRoutes()`/`seedVessels()`/`seedInstanceRoutes()` rejects (e.g. the tenant-config service 4xx/5xx's), the user sees no error toast and no error state — only the button leaving its pending state, contrasted with `handleRunProcessing` (`SetupPage.tsx:144-153`) and `handleRestoreBaseline` (`:155-173`) which do wire `onError` → `toast.error`. This is a real gap but it is the identical, faithfully-mirrored idiom shared by all 8 pre-existing seed rows — not something task-189 introduced. Flagged non-blocking. |
| FE-12 | JSON:API model shape | PASS (N/A) | `TransportRoutesSeedStatus`/`TransportVesselsSeedStatus`/`InstanceRoutesSeedStatus` (`seed.service.ts:74-87`) are plain read-projection interfaces, not `types/models/` JSON:API resources — consistent with all 8 sibling `*SeedStatus` interfaces in the same file, none of which follow `{id, attributes}` either (the raw `/seed/status` wire response is deliberately *not* a JSON:API envelope, per the comment at `seed.service.ts:89-93`) |

## Architecture Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-13 | Service extends `BaseService` (when applicable) | PASS | `SeedService` (`seed.service.ts:187`) uses the documented "Direct API Client Pattern (Simple Resources)" — no validation/transformation needs, consistent with all of its own pre-existing methods |
| FE-14 | Query key factory uses `as const` | PASS | `useSeed.ts:62-67` — `transportRoutesSeedStatusKey`, `transportVesselsSeedStatusKey`, `instanceRoutesSeedStatusKey` all end `as const` |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | PASS (N/A) | No form UI added by this change |
| FE-16 | Schema in `lib/schemas/` with inferred type | PASS (N/A) | No Zod schema added by this change |

## Testing Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | **FAIL** | The three new **status query hooks** — `useTransportRoutesSeedStatus`, `useTransportVesselsSeedStatus`, `useInstanceRoutesSeedStatus` (`useSeed.ts:436-482`) — are never imported or exercised in `lib/hooks/api/__tests__/useSeed.test.tsx`. Confirmed via `grep -rn "useTransportRoutesSeedStatus\|useTransportVesselsSeedStatus\|useInstanceRoutesSeedStatus" src/`: the only test-tree hits are `pages/__tests__/SetupPage.test.tsx:65-67`, where they are **mocked away** (`() => emptyStatus`), not exercised. Every one of the 8 sibling status hooks (`useDropsSeedStatus`, `useGachaponsSeedStatus`, …, `useMapActionScriptsSeedStatus`) gets direct coverage via the `describe.each` block at `useSeed.test.tsx:84-175` — "enables polling and keys by tenant id when a tenant is active" + "disables polling when no tenant is active". The 3 new query hooks were not added to that table (or an equivalent), so their `enabled: !!activeTenant` guard and their `transportRoutesSeedStatusKey`/etc. tenant-keying (FE-09/FE-10, asserted PASS above by code inspection) have zero unit-test verification — only the 3 new **mutation** hooks got a dedicated "transport seed mutations" block (`useSeed.test.tsx:269-302`). This is a real, introduced-by-this-branch coverage gap, not an inherited one — the describe.each table existed before this branch and simply wasn't extended for the 3 new query hooks. |
| FE-18 | Mocks updated when services changed | PASS | `pages/__tests__/SetupPage.test.tsx:39-69` — the `@/lib/hooks/api/useSeed` mock was extended with all 6 new hooks (`useSeedTransportRoutes/Vessels/InstanceRoutes`, `useTransportRoutesSeedStatus`/`useTransportVesselsSeedStatus`/`useInstanceRoutesSeedStatus`); `seed.service.test.ts:433-520` adds a `"transport configuration seed status"` describe block covering all 3 new `get*SeedStatus` projections plus the zero-count/never-seeded case |

## Summary

### Blocking (must fix)
- **FE-17** — Add tenant-guard + polling + query-key coverage for `useTransportRoutesSeedStatus`, `useTransportVesselsSeedStatus`, and `useInstanceRoutesSeedStatus` in `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx`, matching the existing `describe.each` pattern used for the 8 sibling status hooks (`useSeed.test.tsx:84-175`) — either extend that table with the 3 new hooks or add an equivalent dedicated block. Currently these 3 hooks are exercised only through `SetupPage.test.tsx`, where they are mocked out entirely, so their tenant-enablement/tenant-keying logic has no real unit-test proof.

### Non-Blocking (should fix)
- **FE-11 (pre-existing/inherited, all 11 seed rows including the 3 new Transport rows)** — `handleSeed` (`SetupPage.tsx:115-118`) has no `onError` handler anywhere in its call chain, so a failed `seedRoutes()`/`seedVessels()`/`seedInstanceRoutes()` (or any of the 8 sibling seed mutations) surfaces no error toast/state to the user, unlike `handleRunProcessing`/`handleRestoreBaseline` which do. Not introduced by task-189, but worth a follow-up since it now applies to 3 more rows.
- **FE-09 (pre-existing/inherited)** — the seed-row `<Button>` `disabled={row.mutation.isPending}` (`SetupPage.tsx:450`) does not also disable on `!activeTenant`, unlike the Upload/Process/Restore buttons in the same file (`:357, :383, :415`). All 11 seed rows share this gap; the 3 new rows just inherit it via the same generic `seedRows.map()` render loop.
- **FE-13 / multi-tenancy (pre-existing/inherited)** — `seedRoutes()`, `seedVessels()`, `seedInstanceRoutes()` (`seed.service.ts:224-234`) take no explicit `tenant` parameter, relying instead on the API client's ambient tenant state set by `TenantProvider`. This deviates from `patterns-service-layer.md`'s documented rule that every tenant-scoped service method take `tenant` as an explicit first parameter, but it is byte-for-byte the same shape as the 8 pre-existing seed-write methods in the same class (`seedDrops`, `seedGachapons`, etc.), faithfully mirrored per the dispatch brief's instruction, not a new defect.

---

# Backend Audit — task-189-tenant-config-seed-provisioning

- **Scope:** Go files changed on `task-189-tenant-config-seed-provisioning` vs `24c78aa85` (base), 43 files across `libs/atlas-seeder`, `libs/atlas-outbox`, `libs/atlas-tenant`, `services/atlas-tenants/atlas.com/tenants`, `services/atlas-transports/atlas.com/transports`, `tools/catalog-lint`.
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*, SUB-*, FILE-*, EXT-*, SCAFFOLD-*)
- **Date:** 2026-08-03
- **Build:** PASS (all 5 modules + `docker buildx bake atlas-tenants atlas-transports`)
- **Tests:** PASS — `go test -race ./... -count=1` clean in `atlas-tenants` and `atlas-transports`; `go test ./... -count=1` clean in `libs/atlas-seeder`, `libs/atlas-outbox`, `libs/atlas-tenant`
- **Overall:** NEEDS-WORK (one Important structural finding; two non-blocking notes)

## Build & Test Results

```
libs/atlas-seeder        go build ./...  EXIT 0   go test ./... -count=1  ok  (0.035s)
libs/atlas-outbox        go build ./...  EXIT 0   go test ./... -count=1  ok  (0.072s)
libs/atlas-tenant        go build ./...  EXIT 0   go test ./... -count=1  ok  (0.003s)
atlas-tenants (svc)      go build ./...  EXIT 0   go test -race ./... -count=1  ok, all packages
atlas-transports (svc)   go build ./...  EXIT 0   go test -race ./... -count=1  ok, all packages
tools/catalog-lint       go build ./...  EXIT 0

go vet ./...              clean in all 5 modules
tools/goroutine-guard.sh  exit 0 (no bare `go` statements in changed packages)
tools/redis-key-guard.sh  exit 0
tools/service-registration-guard.sh  "service-registration-guard: clean"
docker buildx bake atlas-tenants atlas-transports   both images built and exported successfully
```

## Domain / Package Checklist Results

None of the five changed Go modules contain an `internal/` domain-package tree with `model.go` — `libs/atlas-seeder`, `libs/atlas-outbox`, `libs/atlas-tenant` are shared libraries, and the `atlas-tenants`/`atlas-transports` changes land inside a pre-existing domain package (`configuration`, `transport/config`, `instance/config`) whose `model.go`/`entity.go`/`builder.go` were **not** touched by this branch. The applicable checks are therefore the File Responsibilities Checklist (run on every touched package) plus the specific DOM items that are triggered by what changed (tenant/context threading, Kafka emit, goroutines, Dockerfile).

### File Responsibilities Checklist

| ID | Check | Package | Status | Evidence |
|----|-------|---------|--------|----------|
| FILE-01 | Processor logic in `processor.go` | `configuration` | PASS | All 18 `tenantCtx`-threading edits stay inside `services/atlas-tenants/atlas.com/tenants/configuration/processor.go` (e.g. `CreateRouteAndEmit` processor.go:296-311, `DeleteRankingsAndEmit` processor.go:1689). No `ProcessorImpl` method added outside processor.go. |
| FILE-02 | RestModel + Transform in `rest.go` | `configuration` | PASS | New `Uuid` field and `tenant.DerivedId` call added to `RouteRestModel`/`VesselRestModel`/`InstanceRouteRestModel` and their `Transform*` functions, all in `services/atlas-tenants/atlas.com/tenants/configuration/rest.go:10-22,188-215,378-461`. |
| FILE-02 | RestModel + Extract in `rest.go` | `transport/config`, `instance/config` | PASS | `ExtractRouteFor` (replacing `ExtractRoute`) lives in `services/atlas-transports/atlas.com/transports/transport/config/rest.go:43-95` and `.../instance/config/rest.go:32-79`, alongside the `RestModel` structs it maps. |
| FILE-03 | Cross-service request funcs in `requests.go` | `transport/config`, `instance/config` | PASS | `requests.go` in both packages was not modified; `ExtractRouteFor` is still consumed from `processor.go` via `requests.DrainProvider(...)`, not called directly from `resource.go`. |
| FILE-04 | Entity + Migration in `entity.go` | `configuration` | PASS | `entity.go` untouched by this branch; `AppendConfigurationEntries`/`CountConfigurationEntries` operate on the existing `Entity` type via `administrator.go` and `provider.go`, not by declaring new entity shapes. |
| FILE-05 | Write funcs in `administrator.go` | `configuration` | PASS | `AppendConfigurationEntries` and `CountConfigurationEntries` (both new) are in `services/atlas-tenants/atlas.com/tenants/configuration/administrator.go:95-174`. |
| **FILE-06** | **No package-named catch-all / no ≥2-responsibility collapse** | `configuration/seed` (new package) | **FAIL (Important)** | See finding 1 below. |

### Finding 1 (Important) — `configuration/seed/groups.go` collapses route registration and Kafka-emission business logic into one file

`services/atlas-tenants/atlas.com/tenants/configuration/seed/groups.go` contains, in one file:

- `InitResource` (lines 33-44) — registers HTTP routes via `seeder.RegisterRoutes(router, db, l, src, g)`. This is `resource.go`'s documented responsibility ("Route registration and handler definitions for REST endpoints", file-responsibilities.md `resource.go`), and its own signature (`func InitResource(db *gorm.DB) server.RouteInitializer`) is a byte-for-byte match of the `InitializeRoutes(...) func(db *gorm.DB) server.RouteInitializer` convention documented in patterns-rest-jsonapi.md.
- `afterSeed` (lines 96-126) — business logic that composes `database.ExecuteTransaction`, `outbox.EmitProvider`, and `message.Emit` to build and enqueue a Kafka event. This is `processor.go`/`producer.go` territory ("Pure curried business functions plus `AndEmit` variants for Kafka emission" / "Kafka message creation using context-aware header decorators", file-responsibilities.md).

Per the File Responsibilities Checklist, a single file carrying ≥2 of the documented per-file responsibilities is a FILE-06 finding regardless of how thin either half is. Route registration belongs in a `resource.go`-named file; the `AfterSeed` Kafka-emission callback belongs in a `processor.go`/`producer.go`-named file (or, given this is a fairly small handler, a distinctly named `emit.go`/`events.go` would also satisfy the separation).

**Context, not an exemption:** this exact shape (`Groups()`/`newGroup()`/`afterSeed()` alongside route init, all in one `groups.go`) is the pre-existing convention used by eight other `libs/atlas-seeder` consumers already in the repo (e.g. `services/atlas-npc-shops/atlas.com/npc/seed/groups.go`, `services/atlas-drop-information/atlas.com/dis/seed/groups.go`, `services/atlas-party-quests/atlas.com/party-quests/definition/groups.go`). Per this audit's explicit instructions, prevalence across the codebase does not turn a File-Responsibilities deviation into a pass — no guideline text documents an exception for seeder-adapter files, so this remains a finding here. Fixing it for `atlas-tenants` alone (without touching the other eight pre-existing instances, which predate this branch and are out of scope) would mean splitting `groups.go` into e.g. `resource.go` (InitResource) and `processor.go` or `producer.go` (Groups/newGroup/afterSeed).

By contrast, `services/atlas-tenants/atlas.com/tenants/configuration/seed/subdomain.go` implements exactly one external contract (`seeder.Subdomain[Entry, Entry]`) as a single cohesive adapter type and is **not** flagged — it is the "genuine single-purpose utility" the FILE-06 exception clause allows.

## Notes (non-blocking)

**DOM-20 (table-driven tests):** most new test files in this branch (`libs/atlas-seeder/catalog_test.go`, `libs/atlas-seeder/seed_test.go`, `libs/atlas-seeder/handlers_test.go`, `services/atlas-tenants/atlas.com/tenants/configuration/emit_tenant_test.go`, `services/atlas-transports/atlas.com/transports/instance/config/rest_test.go`) use discrete `func TestXxx(t *testing.T)` functions rather than the `tests := []struct{...}{...}` + `t.Run` table-driven idiom. Only `libs/atlas-tenant/id_test.go:43` (`TestDerivedId_PinnedVectors`) uses a literal table. testing-guide.md states this as a "Prefer," not a hard requirement, and every new test asserts real, substantive behavior (outbox row contents, derived-uuid values, route-shadowing regressions) rather than being a shallow mock check — so this is graded Minor/non-blocking, not a FAIL.

**DOM-10 (test DB tenant callbacks) — graded N/A, not FAIL:** `services/atlas-tenants/atlas.com/tenants/test/database.go:14-40` (`SetupTestDB`) does not call `database.RegisterTenantCallbacks`. This is deliberate and documented, not an oversight: `administrator_test.go:12-16` states explicitly "the append and count administrators scope by an explicit tenant_id predicate, so no tenant GORM callback registration is needed here," and `configuration/provider.go` (untouched by this branch — confirmed via `git diff`, zero hunks) filters every query with an explicit `map[string]interface{}{"tenant_id": tenantID, ...}` WHERE clause rather than relying on `db.WithContext(ctx)` + the automatic GORM callback. Because Task 7's `tenantCtx` helper (`configuration/processor.go:186-198`) now threads a real, non-zero tenant into `p.db.WithContext(ctx)` for every `...AndEmit` call, the automatic callback (registered in production via `database.Connect()`) **is** live on those code paths — but context.md §7 ("Traps") explicitly verified that the callback-injected `tenant_id` predicate and the explicit one are always identical by construction (both derive from the same path-scoped `tenantId`), so this is a verified non-issue, not a live bug this audit is flagging blind.

## What was verified and passed (selected, with citations)

- **F4 fix (configuration-status events carry tenant headers):** `configuration/processor.go:186-198` (`tenantCtx`) is called from all 18 `...AndEmit` methods — verified by count: `grep -c "p.tenantCtx(tenantId)"` = 18, matching `grep -c "AndEmit(tenantId"` interface-method-count of 18 (`processor.go:296,389,426,554,647,684,812,905,942,1061,1150,1186,1314,1407,1444,1606,1660,1689`). Regression test: `configuration/emit_tenant_test.go:82-119` (`TestCreateRouteAndEmit_OutboxRowCarriesTenantHeaders`) asserts all four tenant headers land on the outbox row, and `emit_tenant_test.go:124-145` asserts an unknown tenant aborts the write with zero outbox rows (no tenant-free emit).
- **Derived UUID identity (atlas-tenants and atlas-transports compute it the same way):** `libs/atlas-tenant/id.go:22-24` (`DerivedId`); `configuration/rest.go:108,209,461` calls it with resource names `"routes"`/`"vessels"`/`"instance-routes"`; `atlas-transports/transport/config/rest.go` uses `routeResourceName = "routes"` and `atlas-transports/instance/config/rest.go` uses `resourceName = "instance-routes"` — both string constants match the atlas-tenants side exactly. `libs/atlas-tenant/id_test.go:43-59` pins literal SHA1 vectors so a formula change fails loudly rather than silently re-deriving.
- **Load-then-clear-then-add reconcile (fixes duplicate Redis registry rows):** `services/atlas-transports/atlas.com/transports/kafka/consumer/configuration/consumer.go:40-72` loads before calling `ClearTenant()`; `services/atlas-transports/atlas.com/transports/bootstrap.go:44-64` (`reconcileScheduled`/`reconcileInstance`) applies the same ordering at startup, replacing the old main.go inline loop that used to `AddTenant` with empty slices on a load error (silently wiping registries) — confirmed via `git diff ... main.go`, the destructive fallback is gone.
- **Nil-tenant guard on the consumer side:** `kafka/consumer/configuration/consumer.go:41-50` refuses to act when `t.Id() == uuid.Nil`, tested by `consumer_test.go:21-46` (`TestHandleConfigurationStatus_NilTenantSkipsReload`), which also proves no registry mutation occurred (the registries are un-initialized singletons in the test, so any call would panic — a passing test is proof of the guard).
- **Outbox producer-side WARN:** `libs/atlas-outbox/bridge.go:26-35` logs a Warn (not silent) when `tenant.ID` is absent from resolved headers; `bridge_test.go:129-150` (`TestEnqueueBuffer_WarnsWhenContextHasNoTenant`) and `bridge_test.go:152-163` (silent-when-present counterpart) both exist.
- **F1/F5 zero-behavior-change guarantee for the 9 pre-existing `libs/atlas-seeder` consumers:** `libs/atlas-seeder/catalog_test.go:248` (`TestFilesystemCatalogSource_Roots_PlainSourceStillSingleRoot`) and `catalog_test.go:258` (`TestRevisionFor_SingleRootIsUnchanged`) directly test that a single-root `CatalogSource` (the only kind the other 9 consumers construct) is unaffected by the new shared-root machinery.
- **Seed data shape (F2/F3):** spot-checked `deploy/seed/shared/all/instance-routes/flight-temple-of-time-leafre.json` — top-level `{"data": {"id": "temple-of-time-return-flight", "type": "instance-routes", "attributes": {...}}}`, confirming filename ≠ entity id and no extra `{"data": {"data": ...}}` double-wrap.
- **Route stand-in endpoints correctly scoped to POST only:** `configuration/resource.go` replaces the three deleted `Seed*Handler` registrations with `r.HandleFunc(".../seed", http.NotFound).Methods(http.MethodPost)`, and `configuration/seed/groups_test.go:136-150` (`TestSeedStandInDoesNotShadowOtherVerbs`) distinguishes the stand-in's literal `"404 page not found\n"` body from the CRUD handler's bare-status 404 to prove GET/PATCH/DELETE on an id literally named `"seed"` still reach the CRUD handlers.
- **DOM-22 / Dockerfile:** `atlas-seeder` (newly a *direct* require of `atlas-tenants`, confirmed via `git diff .../go.mod` — no `// indirect` comment) appears in the root `Dockerfile` at all three places this repo's build actually uses (mod-only COPY `Dockerfile:47`, source COPY `Dockerfile:77`, generated `go.work use()` block `Dockerfile:96`) — this repo does not use a fourth `go mod edit -replace` block (it generates `go.work` directly inside the image instead), so 3/3 applicable mentions is complete. Confirmed empirically: `docker buildx bake atlas-tenants atlas-transports` both built and exported successfully.
- **DOM-26 (no bare `go` statements):** `tools/goroutine-guard.sh` exits 0; the one background dispatch this branch relies on is `routine.Go(l, ctx, func(_ context.Context) {...})` at `libs/atlas-seeder/handlers.go:51`, not a bare `go`.
- **Anti-pattern avoided:** neither `services/atlas-tenants/atlas.com/tenants/main.go` nor `services/atlas-transports/atlas.com/transports/main.go` calls `database.RegisterTenantCallbacks` (that call is reserved for SQLite test setups per patterns-multitenancy-context.md; `database.Connect()` already registers it in production) — grep confirms zero matches in both `main.go` files.
- **Mock synchronization (Interface Change Workflow):** the three removed `Seed{Routes,InstanceRoutes,Vessels}` interface methods are removed from both `configuration/processor.go`'s `Processor` interface and `configuration/mock/processor.go` in the same commit (`git diff` shows matched removal, no orphaned mock methods); confirmed by a clean `go build ./...`.

## Summary

### Blocking (must fix)
- FILE-06: `services/atlas-tenants/atlas.com/tenants/configuration/seed/groups.go` combines `resource.go`-responsibility route registration (`InitResource`) with `processor.go`/`producer.go`-responsibility business logic and Kafka emission (`afterSeed`, `Groups`, `newGroup`) in one file. Split route registration into a `resource.go` and the seed-group/emit logic into a `processor.go` (or equivalently-scoped file) within `configuration/seed/`.

### Non-Blocking (should fix)
- DOM-20: prefer table-driven (`tests := []struct{...}{...}` + `t.Run`) tests for the new test files listed above, per testing-guide.md's stated preference.
- DOM-10: `services/atlas-tenants/atlas.com/tenants/test/database.go`'s `SetupTestDB` does not register tenant GORM callbacks. Currently safe because the whole `configuration` package filters explicitly rather than via context, and context.md §7 already verified the one place (Task 7's `tenantCtx`) where the automatic callback now *is* live in production resolves to an identical predicate — but if a future change in this package starts relying on `db.WithContext(ctx)` for tenant isolation without an explicit predicate, the test suite would not catch a regression. Consider adding `database.RegisterTenantCallbacks` to `SetupTestDB` proactively.

---

# Plan Audit — task-189-tenant-config-seed-provisioning

**Plan Path:** docs/tasks/task-189-tenant-config-seed-provisioning/plan.md
**Audit Date:** 2026-08-04
**Branch:** task-189-tenant-config-seed-provisioning
**Base Branch:** main (merge-base `24c78aa85`)
**Head:** `afcf458b8`

## Executive Summary

All 13 planned tasks were faithfully implemented; the code on disk matches the plan's literal file contents almost verbatim in every module checked (`libs/atlas-seeder`, `libs/atlas-tenant`, `libs/atlas-outbox`, `services/atlas-tenants`, `services/atlas-transports`, `services/atlas-ui`). All five Go modules build, vet, and pass `go test -race -count=1` clean; atlas-ui passes `npm test` (1383/1383) and `npm run build` (`tsc -b` + `vite build`); all three repo guards (`redis-key-guard.sh`, `goroutine-guard.sh`, `service-registration-guard.sh`) exit 0; `go run ./tools/catalog-lint deploy/seed` exits 0; and a full tree-wide `tools/lint.sh --check` reports "0 issues" for every one of the ~90 Go modules it checked (its only failure, `ui:node-missing`, is an environment artifact of that particular shell not having Node on `PATH` — independently confirmed clean below via `prettier --check`/`eslint` with Node loaded). Note: `plan.md`'s checkboxes are all literally `- [ ]` (never checked off) despite every task being commit-verified done — a plan-hygiene gap, not an implementation gap; the 13 feature commits map 1:1 onto the 13 tasks by commit message and diff content. Three genuine, well-documented deviations from the plan's literal text were found, all fixes discovered during the implementers' own review pass rather than silent scope cuts — this is adherence, not a violation. The existing backend-guidelines-reviewer audit in this file flags one Blocking structural finding (FILE-06 on `configuration/seed/groups.go`); note that the plan's own File Structure section for Task 6 explicitly specifies exactly the file layout the reviewer flags, so this is a plan-vs-guideline tension for the reader to weigh, not a case of the implementers deviating from what was planned.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `libs/atlas-seeder` — shared catalog root | DONE | `libs/atlas-seeder/catalog.go:22-79` (`filesystemSource.sharedRel`, `NewFilesystemCatalogSourceWithShared`), `seed.go:88-155` (`runSubdomain` multi-root merge, `revisionFor`), `status.go:22-25` (routed through `revisionFor`). Commit `828ed6d54`. Tests pass: `go test -race -count=1 ./...` → `ok`. |
| 2 | `libs/atlas-seeder` — `Group.AfterSeed` hook | DONE | `seeder.go:10-20` (`AfterSeed` field), `handlers.go:52-77` (invoked after successful `Seed`, error logged not returned). Commit `a2792ef34`. |
| 3 | `libs/atlas-tenant` — `DerivedId` formula | DONE | `libs/atlas-tenant/id.go:22-24`, exact `uuid.NewSHA1(tenantId, strings.Join(parts,"/"))` formula. Commit `6318719de`. Pinned-vector tests present in `id_test.go`. |
| 4 | Move transport seed data into shared catalog | DONE | `deploy/seed/shared/all/{routes,vessels,instance-routes}/` — 12/6/12 files, counts verified (`ls ... \| wc -l`). Source dirs deleted: `git rm -r services/atlas-tenants/configurations/{routes,vessels,instance-routes}` confirmed via `git show --stat fe41a7ca9`; `ls services/atlas-tenants/configurations/` on HEAD shows only `rps-rewards` remaining (Task 4 Step 3's exact expected outcome — `mts-configs` never existed as a directory, matching the plan's own caveat). `tools/catalog-lint/subdomains.go:26-33` gained the three `pattern: nil` rules. `go run ./tools/catalog-lint deploy/seed` exits 0. `Dockerfile` untouched (`git diff main...HEAD -- Dockerfile` empty), as required. Commit `fe41a7ca9`. |
| 5 | atlas-tenants — append/count administrators | DONE | `configuration/administrator.go:102` (`AppendConfigurationEntries`), `:153` (`CountConfigurationEntries`). Commit `cc2c9c868`. `administrator_test.go` includes the cross-tenant-isolation test. `go test -race -count=1 ./configuration/...` → `ok`. |
| 6 | atlas-tenants adopts `libs/atlas-seeder` | DONE | `configuration/seed/subdomain.go`, `configuration/seed/groups.go` (new, matches plan's `Subdomain`/`Groups`/`InitResource` shapes exactly — this is the same file the backend-guidelines-reviewer flags as FILE-06 below, per a layout the plan itself specifies). All three bespoke `Seed*` deletions confirmed by grep returning zero matches: `Processor.SeedRoutes/SeedVessels/SeedInstanceRoutes` gone from `processor.go`, `mock/processor.go`, and `resource.go`. `configuration/seed.go` retains `LoadRpsRewardFiles`/`LoadMtsConfigFiles` (untouched, as required). `main.go:89-90` registers `seed.InitResource(db)` **before** `configuration.RegisterRoutes` as specified. Commit `2becc2f9f`, hardened by review-round commit `0240e3456`. |
| 7 | atlas-tenants — real tenant in every emit context | DONE | `configuration/processor.go:188-196` (`tenantCtx` helper). Verified programmatically (not spot-checked): **all 18** `…AndEmit` methods (`CreateRouteAndEmit` … `DeleteRankingsAndEmit`) call `tenantCtx(...)` and **none** reference `p.ctx` directly inside their bodies. Commit `7c422964f`. `emit_tenant_test.go` covers route + rankings emit paths with tenant-header assertions. |
| 8 | atlas-tenants — stable UUID on read models | DONE | `configuration/rest.go` — `RouteRestModel`, `VesselRestModel`, `InstanceRouteRestModel` all gained `Uuid string` populated via `tenant.DerivedId(tenantId, "<resource>", id)` (lines 111, 212, 464). All **12** `Transform*` call sites in `resource.go` (4 each for `TransformRoute`/`TransformVessel`/`TransformInstanceRoute`) pass `tenantId` — verified by grep, all 12 present. Commit `24389e9d1`. `VesselRestModel` gaining `Uuid` is a documented, in-scope addition beyond design.md's narrower "vessels not affected" line (context.md §5) — correctly implemented, not a violation. |
| 9 | `libs/atlas-outbox` — observable silent tenant drop | DONE | `bridge.go:21-35` (`EnqueueBuffer` WARNs per topic when `tenant.ID` header absent). Commit `128d99e39`. `bridge_test.go` covers it. |
| 10 | atlas-transports — stable UUID as route id | DONE | `instance/config/rest.go` and `transport/config/rest.go` byte-match the plan's literal code: `Uuid` field, `ExtractRouteFor(l, t)`, `resolveRouteId` (parse-then-derive fallback with WARN). `Processor.GetInstanceRoutes(t)`, `GetRoutes(t)`, `GetVessels(t)` all take `tenant.Model` per plan. `ExtractVessel` deliberately unchanged. Commit `a4180a8e4`. |
| 11 | atlas-transports — nil-tenant guard + load-then-clear-then-add | DONE | `bootstrap.go` (new) — `reconcileScheduled`/`reconcileInstance`, both load-first, clear-then-add only on success. `kafka/consumer/configuration/consumer.go:44-53` — `t.Id() == uuid.Nil` guard logs ERROR and returns before any registry mutation; reload reordered load→clear→add (`:61-68`, `:72-79`). File content is a byte-for-byte match to the plan's literal listing. Commit `99a924cd6`. |
| 12 | atlas-ui — three transport seed rows | DONE | `seed.service.ts` (`seedRoutes/seedVessels/seedInstanceRoutes`, `getTransportRoutesSeedStatus`/etc., `TransportRoutesSeedStatus`/etc. types), `useSeed.ts` (`useSeedTransportRoutes/Vessels/InstanceRoutes`, `useTransportRoutesSeedStatus`/etc.), `SetupPage.tsx:288-317` (three `seedRows` entries with `Ship`/`Anchor`/`Plane` icons). Commit `ba2daedaf`. `npm test` 1383/1383 pass, `npm run build` succeeds. See the frontend-guidelines-reviewer's FE-17 finding elsewhere in this file — a real but narrow test-coverage gap on the 3 new *status query hooks* (not specified in the plan's own Task 12 Step 1 test listing either, so it's a pre-existing plan-scope gap, not a task-12 execution deviation). |
| 13 | Deploy wiring, runbook, full verification sweep | DONE | `deploy/k8s/base/atlas-tenants.yaml:5-6` — `atlas.seed-catalog: "true"` label added, matching `atlas-drop-information.yaml`'s pattern exactly. `kubectl kustomize deploy/k8s/base` confirms the git-sync sidecar, `seed-catalog` volume/mount, and `SEED_CATALOG_ROOT=/var/run/seed-catalog/catalog/deploy/seed` all land on the atlas-tenants Deployment. `docs/tasks/task-189-tenant-config-seed-provisioning/runbook.md` (156 lines) covers rollout order, post-deploy verification, provisioning instructions, manual fallback, and the known startup window exactly as specified. Commit `b3013b4bc`, followed by a genuine post-sweep hardening fix in `afcf458b8` (see Deviations below). |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 13 tasks are fully implemented with file:line evidence matching the plan's specified interfaces and, in most cases, its literal code listings.

## Deliberate Deviations From the Plan's Literal Text (adherence, not violations)

1. **Commit `0240e3456` ("review round 1")** — after Task 6 landed, the implementers found their own initial cut let `POST /tenants/{tenantId}/configurations/{routes,vessels,instance-routes}/seed` fall through to the CRUD `{id}` handler and swallow non-POST verbs on a resource whose id happens to be literally `"seed"`. Fixed by scoping the three 404 stand-ins to `.Methods(http.MethodPost)` (`resource.go:1205,1216,1228`) and adding `TestSeedStandInDoesNotShadowOtherVerbs` (`configuration/seed/groups_test.go:136-150`) plus a full `substance_test.go` (247 lines) exercising `seeder.Seed` against the real `deploy/seed/shared/all` catalog end-to-end. This is the "self-review catches a regression, fixes it in-branch" pattern CLAUDE.md's "No Deferring Producible Work" section asks for — not a scope cut.
2. **Commit `afcf458b8` (last commit, post Task-13 sweep)** — discovered that `libs/atlas-seeder`'s `runSubdomain` deletes a resource's rows *before* walking the catalog, so a seed run against a missing/unmounted catalog mount scores `Created=0, Deleted=N` and `classifyOutcome` calls that "success." The original `afterSeed` (Task 6) would have emitted the configuration-status event from that state, telling atlas-transports' `ClearTenant()+reload` consumer to wipe a live, healthy Redis registry over what is actually a transient mount failure. Fixed by guarding `afterSeed` on `Created==0 && Deleted>0`: log ERROR, return an error, skip the emit (`configuration/seed/groups.go:96-126`), covered by two new tests in `substance_test.go` (`TestSeed_AfterSeedGuardSkipsEmitOnDeleteWithoutCreate`, `TestSeed_AfterSeedGuardAllowsEmitOnLegitimatelyEmptyFirstSeed`) and documented in the runbook. This is genuine data-safety hardening found during final verification, not a plan violation — it makes the implementation *more* conservative than the plan's literal Task 6 text, in the direction FR-5's "no silent registry wipe" intent already pointed.
3. **`VesselRestModel` gains `Uuid` (Task 8)** — design.md §3.7 says vessels are "not affected" for *registry identity* purposes (true: `ExtractVessel` already used a stable slug id, so vessels never drifted), but the PRD's §5 "Modified" list explicitly includes the vessel read model among those needing a stable UUID surfaced. context.md §5 records this as a deliberate, in-scope addition; the implementation follows context.md, not the narrower design.md line — correctly, since context.md is this task's authoritative "decisions already made" record.

None of these are silent scope cuts; all three are documented in commit messages and/or context.md, and each is covered by new or extended tests.

## Build & Test Results

| Service / Module | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| `libs/atlas-seeder` | PASS | PASS | PASS | `go test -race -count=1 ./...` → `ok` (1.07s) |
| `libs/atlas-tenant` | PASS | PASS | PASS | `go test -race -count=1 ./...` → `ok` (1.01s) |
| `libs/atlas-outbox` | PASS | PASS | PASS | `go test -race -count=1 ./...` → `ok` (1.11s) |
| `services/atlas-tenants/atlas.com/tenants` | PASS | PASS | PASS | all packages `ok` (`configuration`, `configuration/seed`, `tenant`, etc.) |
| `services/atlas-transports/atlas.com/transports` | PASS | PASS | PASS | all packages `ok`, including `instance/config`, `transport/config`, `kafka/consumer/configuration` |
| `services/atlas-ui` | PASS (`tsc -b` + `vite build`) | — | PASS | 192 test files / 1383 tests passed via `npm run test -- --run` |

**Repo guards:** `tools/redis-key-guard.sh` exit 0, `tools/goroutine-guard.sh` exit 0, `tools/service-registration-guard.sh` exit 0 (all run from repo root). `go run ./tools/catalog-lint deploy/seed` exit 0. `kubectl kustomize deploy/k8s/base` confirms the atlas-tenants seed-catalog patch applies.

**Lint (`tools/lint.sh --check`, full tree-wide run, completed):** every Go module in the repo reported `0 issues.` (golangci-lint `fmt` + `run --new-from-rev`), including all 5 modules this branch touched. The only failing target was `ui:node-missing` — the background shell that ran `tools/lint.sh` did not have Node on `PATH` (this repo requires `nvm use 22` first, per CLAUDE.md/project memory), so the script correctly refused to run the atlas-ui checks rather than silently skipping them. This is an environment artifact, not a code finding: independently re-run with Node loaded, `npx prettier --check` and `npx eslint` against all 6 changed/new atlas-ui files (`seed.service.ts`, `seed.service.test.ts`, `useSeed.ts`, `useSeed.test.tsx`, `SetupPage.tsx`, `SetupPage.test.tsx`) both came back clean. Two `generated_file_filter` warnings in the lint output reference files under **other** worktrees (`task-161-keydown-aura-broadcast-skills`, `task-147-attack-drain-hp-gain`, `task-152-mortal-blow`) that no longer exist on disk — pre-existing tool noise unrelated to task-189, not evidence of a problem in this branch.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** NEEDS_FIXES (per the pre-existing backend-guidelines-reviewer's Blocking FILE-06 finding and frontend-guidelines-reviewer's Blocking FE-17 finding elsewhere in this file — both are guideline-conformance issues, not plan-adherence gaps; every one of the plan's 13 tasks is itself fully and correctly implemented)

## Action Items

1. Resolve the backend-guidelines-reviewer's Blocking FILE-06 finding: split `services/atlas-tenants/atlas.com/tenants/configuration/seed/groups.go`'s route-registration responsibility (`InitResource`) from its Kafka-emission business logic (`Groups`/`newGroup`/`afterSeed`) into separately-named files, OR accept the finding as pre-existing convention (eight other `libs/atlas-seeder` consumers already share this exact shape) and explicitly note the guideline exception if that's the maintainers' call — the plan's own Task 6 File Structure section specified this single-file layout, so this is a plan-text-vs-guideline conflict for a maintainer to resolve, not something the implementers got wrong relative to the plan they were given.
2. Resolve the frontend-guidelines-reviewer's Blocking FE-17 finding: extend `services/atlas-ui/src/lib/hooks/api/__tests__/useSeed.test.tsx`'s `describe.each` table (currently covering 8 sibling status hooks at lines 84-175) to include `useTransportRoutesSeedStatus`, `useTransportVesselsSeedStatus`, `useInstanceRoutesSeedStatus`. Not a plan-adherence gap (the plan's own Task 12 Step 1 never specified these tests), but cheap to close and the sibling hooks all have it.
3. (Optional, hygiene) `plan.md`'s checkboxes were never checked off (`grep -c '^\- \[x\]' plan.md` → 0). Since every task is otherwise fully verified done, this has no bearing on the merge decision, but the executing agent should check the boxes for future auditors' convenience.
