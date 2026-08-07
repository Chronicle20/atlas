# Transports Surface in atlas-ui — Implementation Context

Companion to [`plan.md`](./plan.md). Everything below was read from source during planning, not recalled.

- **Task:** task-198-transports-ui-surface
- **Worktree:** `.worktrees/task-198-transports-ui-surface/`, branch `task-198-transports-ui-surface`
- **PRD:** [`prd.md`](./prd.md) · **Design:** [`design.md`](./design.md)
- **Services touched:** `atlas-transports` (Go), `atlas-ui` (TypeScript). `atlas-tenants` gains a second consumer of an existing endpoint, no code change.

---

## 1. Load-bearing facts, with evidence

Paths below are relative to `services/atlas-transports/atlas.com/transports/` unless stated otherwise.

| Fact | Evidence |
|---|---|
| The day's schedule is computed once per reconcile, anchored on UTC start-of-day | `transport/scheduler.go:21-24` — `ComputeSchedule` takes `timeNow().UTC()` and derives `startOfDay`/`endOfDay` |
| The 1-second ticker only re-derives state; it never recomputes the schedule | `main.go:109-120` ticker → `transport.NewProcessor(...).UpdateRoutes()` → `UpdateRoute` (`transport/processor.go:137-139`), which calls `route.UpdateState(time.Now())` |
| State comparison is **time-of-day only**, so trip timestamps carry a stale date | `transport/model.go:116` strips the date from `now`; `:122-123` and `:172-175` strip it from every trip boundary |
| A route with no trips reads `out_of_service` | `transport/model.go:167-169` |
| Shared vessels resolve **by route name**; an unresolved side silently zeroes both routes | `transport/scheduler.go:91-97` matches `route.Name() == vessel.RouteAID()`; `:100-102` returns an empty schedule when either side is missing |
| A trip is dropped unless it arrives before end of day | `transport/scheduler.go:66` and `:120` — `if arrival.Before(endOfDay)` |
| The instance status endpoint reads the **nil-tenant** Redis set | `instance/resource.go:107` passes `uuid.Nil`; writes go through the creating tenant (`instance/instance_registry.go:136-171`, `storeMetadata` adds to the tenant-keyed per-route set at `:95`) |
| `createdAt` is persisted on an instance and has a getter | `instance/instance_registry.go:23,34,45`; `instance/model.go:143` (`CreatedAt()`) |
| `boardingUntil = createdAt + boardingWindow`; `arrivalAt = boardingUntil + travelDuration` | `instance/instance_registry.go:167-169` |
| The stuck sweep force-warps on `now - createdAt > MaxLifetime` | `instance/instance_registry.go:249-253`; `MaxLifetime() = 2 * (boardingWindow + travelDuration)` at `instance/model.go:73` |
| Route ids are tenant-derived and stable across restarts and replicas | `transport/config/rest.go:79-89` — `tenant.DerivedId(t.Id(), "routes", slug)` |
| Scheduled-route config durations are **minutes-as-integer**; vessel turnaround is **seconds** | `transport/config/rest.go:61-64` (`* time.Minute`), `:123` (`* time.Second`) |
| Instance-route config durations already use a `…Seconds` naming convention | `instance/config/rest.go:27-28,58-59` |
| `time.Duration` marshals as an integer nanosecond count | `transport/rest.go:23` — `CycleInterval time.Duration` reaches a client as e.g. `900000000000` |
| `Extract` silently drops the route's three configured durations | `transport/rest.go:141-150` — `NewBuilder(...)` never calls `SetBoardingWindowDuration`/`SetPreDepartureDuration`/`SetTravelDuration` |
| Sparse fieldsets **cannot** drop an `included` entry | `api2go@v1.0.4/jsonapi/helpers.go:84-144` — `FilterSparseFields` iterates `document.Included` and rewrites each entry's attributes; an empty field list yields a wrong-field 400 |
| The vessels configuration resource exists and is paginated | `services/atlas-tenants/atlas.com/tenants/configuration/resource.go:220-270`, routed at `:1217`; model at `configuration/rest.go:159-172` (`uuid`, `name`, `routeAID`, `routeBID`, `turnaroundDelay uint32`) |
| The UI already reads `/api/tenants/{tenantId}/configurations/{resource}` | `services/atlas-ui/src/services/api/mts-config.service.ts:36` — the exact precedent for the vessels read |

---

## 2. Key files

### atlas-transports

| File | Why it matters |
|---|---|
| `transport/model.go` | The state machine. `processStateChange` (108-207) becomes `Evaluate`; `Model` is immutable (private fields + `Builder()` at :83) |
| `transport/rest.go` | `RestModel` (14-25), `Transform` (107-125), `Extract` (127-151), `TripScheduleRestModel` (154-201) |
| `transport/resource.go` | `GetAllRoutesHandler` (50-112) has two `Transform` call sites — the `filter[startMapId]` branch and the all-routes branch. Both need the transformer swap |
| `transport/scheduler.go` | `var timeNow = time.Now` (:7) is the package's existing clock seam — reuse it, do not add a second |
| `transport/state.go` | The five `RouteState` constants |
| `transport/builder.go` | `NewBuilder(name)` + setters; `NewTripScheduleBuilder()`; `NewSharedVesselBuilder()` |
| `instance/resource.go` | `GetInstanceRouteStatusHandler` (97-127) — the `uuid.Nil` bug |
| `instance/rest.go` | `RouteRestModel` (13-22), `InstanceStatusRestModel` (54-61), both transforms |
| `instance/model.go` | `RouteModel.MaxLifetime()` (:71), `TransportInstance.CreatedAt()` (:144) |

### atlas-transports test harnesses (reuse, don't reinvent)

- `transport/resource_paginate_test.go:17-38` — `testServerInformation`, `doGetRoutes(t, router, tenantId, path)`. `setupTransportTestRegistry(t)` and `newTestTenantContext(t) (tenant.Model, context.Context)` live in `transport/processor_test.go:24,31`.
- `instance/resource_paginate_test.go:19-42` — `testServerInformation`, `doGetInstance(t, router, tenantId, path)`.
- `instance/instance_registry_test.go:16-22` — `setupInstanceTestRegistry(t)` spins a `miniredis` and calls `InitInstanceRegistry`.
- `transport/state_test.go` — the table-driven style the new `Evaluate` tests follow. **It must pass unedited** after Task 1; that is the guard that the refactor changed no derived state.

### atlas-ui

| File | Why it matters |
|---|---|
| `src/lib/api/client.ts:353-407` | `api.get` / `getOne` / `getList` / `getListDocument`. `getOne` projects to `data` only and **drops `included`** — the route detail read must use `api.get` on the raw document |
| `src/services/api/pagination.ts:68-92` | `fetchAll<T>(url, size?, options?)` drains `page[number]`/`page[size]`; a response with no `meta` is treated as the whole collection |
| `src/lib/api/query-params.ts:14-36` | `ServiceOptions extends ApiRequestOptions`, `QueryOptions extends ServiceOptions` — `ServiceOptions` is accepted by `fetchAll` |
| `src/components/map-cell.tsx` | `<MapCell mapId={string} tenant={Tenant \| null} />` — process-wide name cache, copyable-id tooltip. `mapId` is a **string**; route attributes are numbers, so `String(...)` at the call site |
| `src/components/data-table.tsx:42-60` | `DataTable` has **no sorting row model** — ordering must be applied to the data before it reaches the table |
| `src/pages/MerchantsPage.tsx:192-200` | The `Tabs` + `useSearchParams` pattern to follow |
| `src/pages/maps-columns.tsx` | The colocated `*-columns.tsx` convention |
| `src/context/tenant-context.tsx:64` | `queryClient.clear()` on tenant switch — no transport-specific invalidation is needed |
| `src/App.tsx:17,22-25` | `lazyWithReload`, never bare `React.lazy` |
| `src/components/app-sidebar-items.ts:24-43` | The Operations group (tenant-switched, which is correct for tenant-scoped transport data) |
| `src/lib/hooks/api/__tests__/useMaps.test.tsx:28-60` | The hook-test mocking harness (mock `@/context/tenant-context` and the service module) |
| `services/atlas-ui/vitest config` | jsdom, `setupFiles: ./src/test/setup.ts` |

---

## 3. Decisions carried into the plan

| Decision | Where | Rationale |
|---|---|---|
| Compute the next transition **server-side** | Task 1, 3 | Reconstructing it client-side means reimplementing the time-of-day comparison *and* `computeSharedVesselSchedule`'s alternation/turnaround, in a language with no time-of-day type, against date-lying rows |
| `processStateChange` stays as a wrapper over `Evaluate().State` | Task 1 | One implementation of the state machine; `state_test.go` passes unedited as the guard |
| `NextAt` is materialised in `now`'s own date and location | Task 1 | Keeps the boundary in the exact frame the comparison uses, so state and countdown cannot disagree. `Transform` passes `timeNow().UTC()`, so the REST path is explicitly UTC |
| Additive `…Seconds` fields; legacy nanosecond fields untouched | Task 2, 6 | No consumer reads the legacy values today, so retyping would *probably* be harmless — but "probably harmless" is not a reason to change a wire contract mid-task |
| `include=schedule` opt-in rather than sparse fieldsets or a `/summary` endpoint | Task 4 | Sparse fieldsets verified impossible; a second endpoint for the same resource would carry its own pagination and drift risk |
| Expose `createdAt` rather than deriving it from `boardingUntil − boardingWindow` | Task 6 | The derivation is exact today but makes the UI's fault threshold depend on an invariant it cannot see |
| `useQueries` fan-out for instance statuses | Task 11 | FR-3.1 needs counts for collapsed rows too. Twelve usually-empty responses per 30s is 0.4 rps — cheaper than a new aggregate endpoint plus a Redis scan |
| One shared clock store + `useSyncExternalStore` | Task 9 | A context provider re-renders every consumer subtree; a timer per cell multiplies intervals |
| Timeline window derived from median trip spacing, clamped 10–30 min | Task 10 | A fixed ±30 min shows two legs for the 15-min boat and ten for the 6-min plane. A shared vessel's real spacing is arrival-plus-turnaround, not either route's cycle |
| Keep the Vessels tab | Task 15 | The unpaired-vessel fault belongs to the *vessel*; folding it into Scheduled would duplicate it across two route rows or hide it |
| **Schedule times render as UTC, labelled** | Task 10, 17 | Resolved during planning — the design says "format the time component only" without naming a frame. The schedule is anchored on UTC midnight, so its UTC components *are* the real boarding times. Converting to browser-local would shift the day boundary and desync the NOW marker from the trips |

---

## 4. Dependency order

```
Backend (independent of the frontend):
  1 Evaluate ──► 3 nextTransitionAt ──► 4 include=schedule
  2 …Seconds + Extract ──┘
  5 tenant scoping (independent)
  6 instance …Seconds + createdAt (independent)
  7 docs (after 1-6)

Frontend:
  8 types + service
        │
        ├──► 10 transport-format ──► 9 Countdown ──┐
        │                                          │
        ├──► 11 hooks ─────────────────────────────┤
        │                                          ▼
        └──► 12 pill + freshness ──────────► 13 board (Scheduled)
                                                   │
                                     ┌─────────────┼─────────────┐
                                     ▼             ▼             ▼
                                14 Instance   15 Vessels    16 MapFlowRail
                                                                 │
                                                            17 VesselTimeline
                                                                 │
                                                            18 route detail

19 verification (everything)
```

Task 9 imports `formatCountdown` from Task 10's module — **implement Task 10 before Task 9** even though the numbering runs the other way (Task 9 introduces the clock, which reads better first).

---

## 5. Verification commands

From the worktree root unless stated otherwise. Per `CLAUDE.md` §Build & Verification.

```bash
# Go — from services/atlas-transports/atlas.com/transports
go test -race ./...
go vet ./...
go build ./...

# Repo guards
tools/redis-key-guard.sh
tools/goroutine-guard.sh

# Lint & format (needs nvm 22 on PATH or the atlas-ui half false-fails)
tools/lint.sh          # fix mode, rewrites in place
tools/lint.sh --check  # must be clean

# atlas-ui — from services/atlas-ui
npm run test    # vitest run
npm run build   # tsc -b && vite build — THIS is what type-checks
```

Not applicable to this task unless something unexpected changes:

- `docker buildx bake atlas-transports` — only if `go.mod` changes (not expected; no new dependency).
- `tools/service-registration-guard.sh` — no new service.
- `tools/template-opcode-order-guard.sh`, `tools/skill-job-id-guard.sh` — no templates, no job/skill id comparisons.

Before the PR: `superpowers:requesting-code-review` → `plan-adherence-reviewer` + `backend-guidelines-reviewer` + `frontend-guidelines-reviewer`, pinned to Sonnet/Haiku.

---

## 6. Traps

1. **`api.getOne` drops `included`.** The route detail read needs the compound document — use `api.get` on the raw shape. A silent empty timeline is the failure mode.
2. **`MapCell` takes a string `mapId`.** Route attributes are numbers. `String(...)` at every call site.
3. **`DataTable` cannot sort.** FR-1.3's severity order must be applied in a `useMemo` before the data reaches the table.
4. **`useSyncExternalStore`'s `getSnapshot` must be cached.** Returning `Date.now()` directly loops forever; the module holds `snapshot` and updates it inside the tick.
5. **Do not add a second clock seam in atlas-transports.** `transport/scheduler.go:7` already has `var timeNow = time.Now`; `Transform` uses it so tests can pin it.
6. **`GetAllRoutesHandler` has two `Transform` call sites.** Missing the `filter[startMapId]` branch leaves a payload hole that no test in the plan would catch except the include test's default case.
7. **`instance/resource_paginate_test.go` passes `uuid.New()` per subtest** (three call sites) — all must move to the seeded tenant, or the re-pointed test fails in a confusing way.
8. **Vessel ids are slugs, not names.** Resolution matches on `attributes.name` against `routeAID`/`routeBID`; the slug is only for the `#vessel-{slug}` anchor. The seed data currently makes them equal, which will hide the bug locally.
9. **`tools/lint.sh --check` false-fails without nvm 22 on PATH**, and under cross-worktree golangci-lint lock contention. Confirm the failure is real before chasing it.
10. **`npm run test` alone is not verification.** The build is what type-checks, and it type-checks the test files too.
11. **Line endings.** Do not let an editor normalize CRLF→LF; it inflates the diff with spurious changes.
