# Transports Surface in atlas-ui — Design

Task: task-198-transports-ui-surface
PRD: [`prd.md`](./prd.md) (v1, approved)
Status: Draft for review
Created: 2026-08-06

---

## 1. Scope and framing

The PRD asks for a read-only **Operations → Transports** surface plus the minimum backend
work that makes it possible. This document settles the architecture, resolves the PRD's four
open questions against verified source, and records the alternatives that were rejected and why.

Nothing here changes warping, scheduling, or state-transition behaviour. The only behavioural
backend change is FR-6.1 — the instance status endpoint starts returning data it should always
have returned.

### 1.1 Facts verified during design

Everything below was read, not recalled. These are the load-bearing facts the design rests on.

| Fact | Evidence |
|---|---|
| The day's schedule is computed once per reconcile, not continuously | `bootstrap.go:44-52` calls `registry.AddTenant`, which calls `NewScheduler(...).ComputeSchedule()` (`transport/processor.go:70`); `ComputeSchedule` anchors on `startOfDay` of the day it runs (`transport/scheduler.go:22-24`) |
| The 1-second ticker only re-derives state; it never recomputes the schedule | `main.go:111` ticker → `transport.NewProcessor(l, ctx).UpdateRoutes()` → `UpdateRoute` → `route.UpdateState(now)` (`transport/processor.go:128-143`) |
| State comparison is time-of-day only — trip timestamps carry a stale date | `transport/model.go:116` strips the date from `now`, and lines 122-123 / 172-175 strip it from every trip boundary |
| Shared vessels are resolved **by route name**, and an unresolved vessel silently zeroes both routes | `transport/scheduler.go:91-97` matches `route.Name() == vessel.RouteAID()`; lines 100-102 return an empty schedule when either side is missing |
| A route with no trips reads `out_of_service` | `transport/model.go:167-169` |
| The instance status endpoint reads the nil-tenant Redis set | `instance/resource.go:107` passes `uuid.Nil`; writes go through `p.t.Id()` (`instance/processor.go:120` → `instance/instance_registry.go:136`) |
| `createdAt` **is** persisted on an instance, so instance age survives a restart | `instance/instance_registry.go:23,34,45` |
| `boardingUntil = createdAt + boardingWindow`, `arrivalAt = boardingUntil + travelDuration` | `instance/instance_registry.go:166-168` |
| Stuck-timeout force-warp compares `now - createdAt` against `MaxLifetime()` | `instance/instance_registry.go:249-253`, `instance/model.go:73` |
| Route ids are tenant-derived and stable across restarts/replicas | `transport/config/rest.go:79-89` |
| Scheduled-route config durations are **minutes-as-integer**; vessel turnaround is **seconds-as-integer** | `transport/config/rest.go:61-64` (`* time.Minute`), `:123` (`* time.Second`) |
| Instance-route config durations are already `…Seconds`-suffixed | `instance/config/rest.go:27-28,58-59` |
| Sparse fieldsets **cannot** suppress the `included` array | `api2go@v1.0.4/jsonapi/helpers.go:116-123` — `FilterSparseFields` iterates `document.Included` and rewrites each entry's attributes; it never removes an entry. An empty `fields[trip-schedule]=` yields a wrong-field 400 (`helpers.go:161-171`, `:125-141`) |
| Nothing outside atlas-transports consumes the route `schedule` relationship | Consumers are atlas-channel (`transport/route/`) and atlas-query-aggregator (`transport/`). Query-aggregator's `RestModel` declares no references at all. Channel decodes schedule ids but no code outside its own `rest.go` reads `Model.Schedule()` |
| The UI already reads `/api/tenants/{tenantId}/configurations/{resource}` | `services/api/mts-config.service.ts:36` — the exact precedent for the vessels read |

---

## 2. Architecture

```
atlas-ui  (Operations → Transports)
│
├── TransportsPage ─ tabs: Scheduled │ Instance │ Vessels   (tab in ?tab=)
│     ├── Scheduled ── useScheduledRoutes()   ── GET /api/transports/routes            (30s)
│     ├── Instance  ── useInstanceRoutes()    ── GET /api/transports/instance-routes   (30s)
│     │               useInstanceStatuses()   ── GET …/instance-routes/{id}/status ×N  (30s)
│     └── Vessels   ── useVessels()           ── GET /api/tenants/{t}/configurations/vessels
│
└── TransportRouteDetailPage
      └── useScheduledRoute(routeId)          ── GET /api/transports/routes/{id}?include=schedule
            ├── MapFlowRail        (SVG)
            └── VesselTimeline     (SVG, 1 or 2 lanes)

shared: useCountdown() → subscribes to a single 1-second clock store (one interval per tab)
```

Three principles govern the split:

1. **The server owns state and time.** The UI never derives a route state, and never does
   scheduler arithmetic. It renders `state`, and counts down to a server-supplied absolute
   instant.
2. **Time-of-day never crosses the wire as an instant.** Trip rows keep their stale-date
   timestamps (unchanged), and the UI formats only their time component. The one field the UI
   treats as an absolute instant — `nextTransitionAt` — is computed server-side precisely so no
   date-bearing value has to be trusted.
3. **The board is independent of the schedule.** The Scheduled tab renders entirely from route
   attributes. Trip rows are fetched only on the detail page, where exactly one route's worth of
   them is needed.

---

## 3. Backend design — atlas-transports

### 3.1 FR-6.1 / FR-6.2 — instance status tenant scoping

`GetInstanceRouteStatusHandler` derives the tenant from the request context, matching every
other tenant-scoped read in the service:

```go
t := tenant.MustFromContext(d.Context())
instances := ir.GetInstancesByRoute(t.Id(), routeId)
```

`rest.RegisterHandler` already installs the tenant in the request context (the same context the
sibling handlers' `NewProcessor(d.Logger(), d.Context())` calls depend on), so no plumbing
change is needed. `MustFromContext` panics on a missing tenant, which is the established
convention here — an untenanted request never reaches these handlers.

**Tests.**

- `TestGetInstanceRouteStatusPaginates` (`instance/resource_paginate_test.go:151`) is re-pointed:
  seed the three instances under a concrete `tenantA := uuid.New()` and issue the requests with
  that same tenant. The comment block at lines 149-159 that documents the `uuid.Nil` quirk is
  deleted with it — it is no longer true, and leaving stale "pre-existing quirk" prose is worse
  than no comment.
- New `TestGetInstanceRouteStatusIsTenantScoped`: create an instance under tenant A, request the
  same route's status as tenant B, assert `data` is empty and `meta.total` is 0; then request as
  tenant A and assert the instance is present. This is the regression guard the acceptance
  criteria call for, and it fails against today's code in both directions.

### 3.2 FR-6.4 — `nextTransitionAt` / `nextState` (the core backend change)

**Decision: accept the PRD's addition.** Open question 2 is resolved in favour of computing the
transition server-side. The alternative — reconstructing it in TypeScript — requires
reimplementing the time-of-day comparison in `processStateChange` *and*
`computeSharedVesselSchedule`'s alternation-plus-turnaround arithmetic
(`scheduler.go:104-136`), in a language whose `Date` has no time-of-day type, against rows whose
date component is a lie. That is a scheduler fork, and a second place for the mechanic to drift.

**Refactor.** `processStateChange` today computes the governing trip and its four boundaries and
then throws all of it away except the state. It is restructured into one evaluation that returns
both, with `processStateChange` retained as a thin wrapper so `UpdateState` and the existing
`state_test.go` cases are untouched:

```go
// transport/model.go
type Transition struct {
    State     RouteState
    NextState RouteState
    NextAt    time.Time // zero when State == OutOfService
}

func (m Model) Evaluate(now time.Time) Transition
func (m Model) processStateChange(now time.Time) RouteState { return m.Evaluate(now).State }
```

`Evaluate` keeps the existing trip-selection and branch structure verbatim. Each branch that
returns a state now also names its boundary and successor:

| Branch (normal case) | `NextState` | Boundary |
|---|---|---|
| before `boardingOpen` → `awaiting_return` | `open_entry` | `boardingOpen` |
| before `boardingClosed` → `open_entry` | `locked_entry` | `boardingClosed` |
| before `departure` → `locked_entry` | `in_transit` | `departure` |
| before `arrival` → `in_transit` | `awaiting_return` | `arrival` |
| past arrival, future trip exists → `awaiting_return` | `open_entry` | the next trip's `boardingOpen` |
| no trip at all → `out_of_service` | — | zero |

The midnight-crossing branch (`model.go:178-188`) gets the same treatment against its own
boundaries.

**Time-of-day → absolute instant.** Boundaries are time-of-day values. `NextAt` is materialised
as: today's date (in UTC, matching the scheduler's `startOfDay`) at the boundary's time-of-day;
if that instant is not strictly after `now`, add 24h. When the governing trip yields no boundary
later in the day, wrap to the earliest `boardingOpen` across the route's schedule, plus 24h. This
is a faithful projection of what the running comparison already does — the state machine is
already daily-periodic by construction, because it only ever compares times of day.

**Transform.** `Transform` needs a `now`. The package already carries `var timeNow = time.Now`
(`transport/scheduler.go:7`) as its seam for tests; `Transform` uses the same symbol rather than
introducing a second clock or changing the `func(Model) (RestModel, error)` signature that
`model.SliceMap` composes against.

`nextTransitionAt` is serialised RFC3339 (`""` when `out_of_service`), `nextState` as the bare
`RouteState` string (`""` when `out_of_service`).

**Tests.** Table-driven in `transport/state_test.go`'s style, with `timeNow` pinned: one case per
branch asserting `(State, NextState, NextAt)`; one midnight-crossing case; one same-time-of-day
wrap case; one `out_of_service` case asserting a zero `NextAt`; one shared-vessel case built from
a real `ComputeSchedule` run asserting the boundary lands on the *other* lane's turnaround.

### 3.3 FR-6.5 — one duration rule

**Decision: additive, integer seconds, explicit `…Seconds` suffix.** This matches the convention
`instance/config/rest.go:27-28` already established, and it is the only form that survives JSON
without a decoder.

Added to `transport.RestModel`:

| Field | Type | Source |
|---|---|---|
| `boardingWindowSeconds` | `uint32` | `m.BoardingWindowDuration()` |
| `preDepartureSeconds` | `uint32` | `m.PreDepartureDuration()` |
| `travelDurationSeconds` | `uint32` | `m.TravelDuration()` |
| `cycleIntervalSeconds` | `uint32` | `m.CycleInterval()` |

Added to `instance.RouteRestModel`:

| Field | Type | Source |
|---|---|---|
| `boardingWindowSeconds` | `uint32` | `m.BoardingWindow()` |
| `travelDurationSeconds` | `uint32` | `m.TravelDuration()` |

The pre-existing nanosecond-valued fields — `cycleInterval` on `routes`, `boardingWindow` /
`travelDuration` on `instance-routes` — are **kept as-is and marked legacy in `docs/rest.md`**.
atlas-channel decodes `cycleInterval` into `route.Model` (`transport/route/rest.go:143`) and
atlas-saga-orchestrator decodes the instance-route pair; none of them *reads* those values, so
retyping in place would very likely be harmless — but "very likely harmless" is not a reason to
change a wire contract mid-task. The UI reads only the `…Seconds` fields; the legacy fields get
one line of doc each saying they are nanoseconds and superseded.

Note the PRD's FR-6.3 named the new fields `boardingWindowDuration` / `preDepartureDuration` /
`travelDuration`. Those names collide with the legacy semantics on the sibling resource and carry
no unit; the `…Seconds` names are used instead, which is what FR-6.5 asks for when it says "one
rule."

`Extract` (`transport/rest.go:127`) is extended to round-trip the three new duration fields — it
currently drops them entirely, so a `RestModel → Model → RestModel` round trip loses the route's
shape. This is a latent bug in the existing code that the new fields would otherwise inherit.

### 3.4 Open question 1 — suppressing the schedule payload on the board

**Answer: sparse fieldsets cannot do it, verified.** `jsonapi.FilterSparseFields`
(`api2go@v1.0.4/jsonapi/helpers.go:84-144`) filters *attributes within* each entry of
`document.Included`; it never drops an entry. `fields[trip-schedule]=` with an empty value
produces a wrong-field error and a 400. Sparse fieldsets are a dead end here.

**Decision: make the schedule opt-in on the list endpoint.**

- `GET /transports/routes` — the `schedule` relationship is empty by default. `Transform` is
  split: `Transform` (unchanged, schedule attached) and `TransformSummary` (identical minus the
  schedule slice). The list handler uses `TransformSummary` unless `include=schedule` is present
  in the query, in which case it uses `Transform`.
- `GET /transports/routes/{routeId}` — unchanged, always attaches the schedule.

This is standard JSON:API semantics (`include` is what controls compound documents; a document
with no `include` is not required to carry one) and it is safe: the only external consumers are
atlas-channel's `IsBoatInMap` sweep, which reads `state` and `startMapId`
(`transport/route/processor.go:98-125`), and atlas-query-aggregator, whose `RestModel` declares
no relationships at all. Both keep working; the board drops from ~1,000 included resources per
poll to twelve route objects.

A regression test asserts the list response's `included` is absent/empty by default and populated
when `?include=schedule` is passed.

### 3.5 `createdAt` on `instance-status`

FR-3.3 flags an instance past two-thirds of `MaxLifetime()`. The server force-warps on
`now - createdAt > MaxLifetime()` (`instance/instance_registry.go:249-253`), so the UI must flag
on the same quantity or the warning and the action disagree. `createdAt` is already persisted
(`instance/instance_registry.go:23`) but not exposed.

**Decision: add `createdAt` (RFC3339) to `InstanceStatusRestModel`.** This is a one-line
addition beyond the PRD's FR-6 list, recorded in §7 as a deliberate deviation. The alternative —
deriving `createdAt = boardingUntil − boardingWindow`, which is exact by construction
(`instance_registry.go:166`) — was rejected because it makes the UI's fault threshold depend on
an invariant it cannot see, and it silently drifts if instance creation ever changes.

### 3.6 Documentation

`services/atlas-transports/docs/rest.md` gains the new attributes on `routes` and
`instance-routes`, the `include=schedule` parameter and its default, the legacy-field notes, and
a corrected description of the status endpoint's tenant scoping. `docs/domain.md` gains
`Model.Evaluate` / `Transition`. The `.bruno` collections under `services/atlas-transports/.bruno/`
gain one request exercising `?include=schedule`.

---

## 4. Frontend design — atlas-ui

### 4.1 Module layout

```
src/types/models/transport.ts                  types only
src/services/api/transports.service.ts         HTTP adapters
src/lib/hooks/api/useTransports.ts             query keys + hooks
src/lib/utils/clock.ts                         shared 1-second store
src/pages/TransportsPage.tsx                   board shell + tabs
src/pages/transports-columns.tsx               Scheduled tab columns
src/pages/TransportRouteDetailPage.tsx         route detail
src/components/features/transports/
    RouteStatePill.tsx
    Countdown.tsx
    MapFlowRail.tsx
    VesselTimeline.tsx
    InstanceRoutesTable.tsx
    VesselsTable.tsx
    FreshnessIndicator.tsx
    transport-format.ts                        pure helpers (labels, sorting, windowing)
```

Every figure and table is its own file with a single purpose, and the arithmetic they need
(sort order, transition labels, timeline windowing, fault detection) lives in
`transport-format.ts` as pure functions so it can be unit-tested without rendering.

### 4.2 Types

`src/types/models/transport.ts` declares the JSON:API shapes: `ScheduledRoute`,
`TripSchedule`, `InstanceRoute`, `InstanceStatus`, `Vessel`, plus a `RouteState` union
(`"out_of_service" | "in_transit" | "locked_entry" | "open_entry" | "awaiting_return"`) and an
`InstanceState` union. Durations are typed as `number` with `Seconds` in the field name; the
legacy nanosecond fields are deliberately **not** declared, so nothing can read them by accident.

### 4.3 Service layer

`transports.service.ts` is a thin adapter over `lib/api/client`, following
`mts-config.service.ts` and `maps.service.ts`:

| Method | Request |
|---|---|
| `getScheduledRoutes()` | `fetchAll<ScheduledRoute>("/api/transports/routes")` |
| `getScheduledRoute(routeId)` | `api.getOne("/api/transports/routes/{id}?include=schedule")` — returns the route plus its trip rows, read from the document's `included` |
| `getInstanceRoutes()` | `fetchAll<InstanceRoute>("/api/transports/instance-routes")` |
| `getInstanceStatuses(routeId)` | `fetchAll<InstanceStatus>("/api/transports/instance-routes/{id}/status")` |
| `getVessels(tenantId)` | `fetchAll<Vessel>("/api/tenants/{tenantId}/configurations/vessels")` |

The detail call needs the compound document, so it uses `api.getListDocument`-style access to
`included` rather than `getOne`'s data-only projection; the service normalises it to
`{ route, schedule }` so no component ever touches a raw JSON:API document.

### 4.4 Hooks and polling

`useTransports.ts` exports a `transportKeys` factory (`all` / `scheduled` / `scheduledDetail(id)`
/ `instanceRoutes` / `instanceStatus(routeId)` / `vessels(tenantId)`) and one hook per query.
Every hook: `enabled: !!activeTenant`, `refetchInterval: 30_000`,
`refetchIntervalInBackground: false`, `placeholderData: keepPreviousData` so a poll does not blank
the table. Tenant switching is already handled by `TenantProvider`'s cache clear; no
transport-specific invalidation exists.

**Instance statuses are a fan-out.** FR-3.1 needs a live count for every instance route, so the
status endpoint is queried once per route via `useQueries` — twelve small requests per 30s poll
(0.4 rps), each normally returning an empty collection. Alternatives considered:
fetching status only for expanded rows (rejected — FR-3.1 requires counts for collapsed rows
too), and adding an aggregate `/transports/instance-status` endpoint (rejected — a new endpoint
and a new Redis scan for a twelve-item list is more backend surface than the payload justifies;
worth revisiting only if instance-route counts grow by an order of magnitude).

### 4.5 The clock, and countdowns that don't re-render the table

`lib/utils/clock.ts` exposes a module-level store with `subscribe(cb)` / `getSnapshot()` at
1-second granularity: one `setInterval`, started on first subscriber and cleared on last.
`Countdown` is a leaf component that reads it via `useSyncExternalStore`, so each tick re-renders
only the countdown cells — never the table, never the page. This satisfies the NFR directly and
avoids both a context provider (which would re-render every consumer's subtree) and a
per-cell interval (twelve timers instead of one).

`Countdown` takes `targetAt: string | null` and a `label`. Behaviour: renders `mm:ss`, switching
to `h:mm:ss` past one hour; clamps at `0:00` and never goes negative; renders an em dash for
`null`. It never emits a state change of its own — a countdown reaching zero does not update the
route state; the next refetch does.

### 4.6 Board page

`/transports`, title "Transports", one entry appended to the Operations group in
`app-sidebar-items.ts`. Tabs are `Scheduled` / `Instance` / `Vessels`, each with a count badge;
the active tab lives in `?tab=` (`useSearchParams`, `replace: true`), defaulting to `scheduled` —
the pattern `MonstersPage.tsx:48-60` already uses for `?q=`.

`DataTable` (`components/data-table.tsx`) has no sorting row model, so FR-1.3's ordering is
applied to the data in a `useMemo` before it reaches the table: state severity
(`out_of_service` → `in_transit` → `locked_entry` → `open_entry` → `awaiting_return`), then route
name. The severity rank is a pure exported constant so the ordering is unit-testable.

Columns: route name (link to detail) · state pill · next change · start map · destination map ·
vessel · cycle interval. Maps render through the existing `MapCell` (`components/map-cell.tsx`),
which is already process-cached and gives the copyable-id tooltip for free. The vessel column
links to `/transports?tab=vessels#vessel-{slug}`.

The "next change" label is derived from `nextState` alone: `open_entry` → "boards in",
`locked_entry` → "closes in", `in_transit` → "departs in", `awaiting_return` → "arrives in";
`out_of_service` → em dash.

`FreshnessIndicator` in the page header reads `dataUpdatedAt` / `isFetching` / `isError` from the
queries and renders a live dot, the age of the last success (ticking off the same clock store),
and an explicit stale treatment on error.

### 4.7 Route detail

`/transports/routes/:routeId`. Header: name, state pill, countdown.

**Map-flow rail** — an SVG `role="img"` with an `aria-label` naming the chain, rendering
`startMap → stagingMap → enRouteMapIds[…] → destinationMap` as stops joined by captioned legs
("walk in", "warp on departure", "warp on arrival"). Stops render `MapCell` in a foreignObject-free
layout: the SVG draws the rail and the leg captions, and the stop badges are HTML positioned over
it, so `MapCell`'s tooltip and link behaviour are unchanged. When the route is `in_transit` the
en-route segment is emphasised. `observationMapId` is an annotation beneath the rail, explicitly
labelled as where ARRIVED/DEPARTED effects fire — not a stop.

**Key/value strip** — observation map, boarding window, pre-departure, travel duration, cycle
interval, trips scheduled today, shared vessel.

**Vessel timeline** — an SVG `role="img"` strip with a NOW marker. Each trip is three contiguous
segments (boarding open / locked / in transit) positioned proportionally. A route on a shared
vessel gets two lanes (its own and its partner's, so the alternation and the turnaround gap are
visible); an independent route gets one. The partner route's trips come from a second detail
fetch, keyed by the partner's id, resolved through the vessel list.

Under the last lane sits a **time axis**: a rule with ticks at round wall-clock times across the
window (`timelineAxisTicks` picks the coarsest interval from 1m…1h that keeps the window under six
gridlines) and a `HH:MM` label per tick. Without it, horizontal position — the strip's whole
subject — carries no absolute value, and every real time is locked inside a segment's tooltip. The
NOW marker is stamped with the second it is drawn at (`NOW HH:MM:SS`) rather than the bare word:
it is the one time on the strip that moves, and the countdowns beside it tick every second.

Trip times are formatted **time-of-day only**, everywhere, with no date component — enforced by
`formatTimeOfDay()` in `transport-format.ts` (and its `formatTimeOfDayMs` / `formatClockMs`
siblings, which take milliseconds-since-UTC-midnight for the axis and the marker) that all
timeline code must go through. Axis ticks past UTC midnight keep counting up so the axis stays
monotonic; only their labels wrap, so a window straddling midnight reads `23:50 · 00:00 · 00:10`
rather than `24:00`.

**Open question 4 — window size. Decision: derive it from the schedule, don't hard-code ±30 min.**
A fixed window puts two legs on screen for the 15-minute boat and ten for the 6-minute plane. The
rule: let `spacing` be the median gap between consecutive `boardingOpen` times across the lanes in
view; the half-window is `clamp(1.5 × spacing, 10 min, 30 min)`. This is derived from the data
rather than from `cycleInterval`, which is the right input because a shared vessel's real spacing
is driven by arrival-plus-turnaround (`scheduler.go:134`), not by either route's configured cycle.
It degenerates to roughly ±22 min for the boats and ±10 min for the subway and plane, and it is a
pure function with its own unit tests.

**Fault state (FR-2.8)** — a route with no trips replaces the timeline with a message naming the
two producible causes: a `cycleInterval`/`travelDuration` combination that leaves no trip fitting
inside the day (`scheduler.go:66`, `:120` — a trip is dropped unless `arrival.Before(endOfDay)`),
or membership in a vessel whose partner does not resolve (`scheduler.go:100-102`). The UI
distinguishes them: it knows from the vessel list whether this route is on an unresolved vessel.

### 4.8 Instance tab

One row per instance route: name, live count, capacity, boarding window, travel duration, start
map, destination map. Rows with `count > 0` expand to per-instance rows: truncated id with a
copyable full-UUID tooltip, state, character count, and a countdown (to `boardingUntil` while
boarding, to `arrivalAt` while in transit). `count === 0` renders `0`, is not expandable, and
carries no error styling — it is the steady state.

The stuck flag (FR-3.3) is `now - createdAt > (2/3) × MaxLifetime`, with
`MaxLifetime = 2 × (boardingWindowSeconds + travelDurationSeconds)` from the instance route. Both
inputs are server-supplied; the UI does no other instance arithmetic.

### 4.9 Vessels tab

**Open question 3 — keep the Vessels tab.** It costs one table and one query, and it is the only
place FR-4.4's unpaired-vessel fault can live: that fault belongs to the *vessel*, not to either
route, and folding it into the Scheduled tab would either duplicate it across two route rows or
hide it entirely. It also makes the alternation legible in one glance, which is the PRD's stated
primary goal.

Columns: vessel name, route A, route B, turnaround delay. Routes are resolved **by name** against
the scheduled-route list — the same rule `scheduler.go:91-97` uses — and rendered with their
current state pills. Each row carries `id="vessel-{slug}"` so the Scheduled tab's vessel column
can link to it. An unresolvable `routeAID`/`routeBID` renders as a fault row stating that both of
the vessel's routes will be `out_of_service` until the reference is fixed.

Vessel ids are slugs, not UUIDs, and slug ≠ name even though the seed data currently makes them
equal (`configuration/rest.go:159-172`, `transport/config/rest.go:119-121`). Resolution matches on
`name`; the slug is used only for anchors.

### 4.10 Cross-cutting

- **Accessibility.** Every state pill carries its text label — colour is never the sole encoding.
  Both SVG figures are `role="img"` with an `aria-label` stating what the figure shows. Both
  themes via existing tokens; no hard-coded colours.
- **Responsiveness.** Tables and both figures live in `overflow-x: auto` containers; the page body
  never scrolls horizontally.
- **Routing.** Two `lazyWithReload` routes in `App.tsx` (never bare `React.lazy` — see
  `App.tsx:17,22-25`).
- **Errors.** Failed refetches surface through the existing error logger plus the stale treatment;
  no new observability.

### 4.11 Frontend tests

Vitest, alongside the existing `__tests__` folders:

- `transport-format.test.ts` — severity sort, transition labels, `formatTimeOfDay` (asserting no
  date leaks), the timeline window rule at 15/6/5-minute spacings, the stuck-instance threshold,
  the vessel-resolution and unresolved-vessel cases.
- `Countdown.test.tsx` — ticks, clamps at `0:00`, never negative, em dash for null.
- `transports.service.test.ts` — request URLs including `?include=schedule`, and the
  `{ route, schedule }` normalisation from a compound-document fixture.
- `TransportsPage.test.tsx` — tab ↔ `?tab=` round trip, counts, `0` instance routes render
  non-expandable.

---

## 5. Alternatives considered and rejected

| Decision | Chosen | Rejected alternative | Why |
|---|---|---|---|
| Next transition | Server-computed `nextTransitionAt` / `nextState` | Client reconstructs it | Requires forking the time-of-day comparison *and* the shared-vessel alternation into TypeScript, against date-lying timestamps. Two places for one mechanic. |
| Board payload | `include=schedule` opt-in on the list | Sparse fieldsets | Verified impossible: `FilterSparseFields` never drops an `included` entry, and an empty field list 400s. |
| Board payload | `include=schedule` opt-in on the list | New `/transports/routes/summary` endpoint | A second endpoint for the same resource, with its own pagination and drift risk, to express what `include` already expresses. |
| Duration wire format | Additive `…Seconds` integer fields | Retype `cycleInterval` in place | No consumer reads the value today, so it would very likely be harmless — but the change is invisible until it isn't, and additive costs four struct fields. |
| Instance age | Expose `createdAt` | Derive from `boardingUntil − boardingWindow` | Exact today, but it makes the UI's fault threshold depend on an invariant it cannot see, and it diverges silently if creation changes. |
| Instance counts | `useQueries` fan-out over routes | Aggregate status endpoint | Twelve tiny, usually-empty responses per 30s. New endpoint + Redis scan is more surface than the payload justifies. |
| Countdown ticking | One shared clock store + `useSyncExternalStore` | Context provider, or a timer per cell | Context re-renders every consumer subtree; per-cell timers multiply intervals. The store re-renders only the leaves that subscribe. |
| Timeline window | Derived from median trip spacing, clamped 10–30 min | Fixed ±30 min | Fixed shows two legs for boats and ten for the plane. Derived also handles shared vessels, whose spacing is turnaround-driven, not cycle-driven. |
| Vessels | Own tab | Fold into Scheduled tab | The unpaired-vessel fault belongs to the vessel, not to a route; folding it either duplicates or hides it. |

---

## 6. Risks

| Risk | Mitigation |
|---|---|
| `Evaluate`'s refactor changes derived state as a side effect | `processStateChange` stays as a wrapper over `Evaluate().State`; the existing `state_test.go` suite must pass unchanged before any new assertion is added. |
| `include=schedule` default-off breaks an unseen consumer | Verified against both consumers by grep and by reading their models; neither reads schedule data. The detail endpoint is unchanged, so any consumer that genuinely needs trips has an unmodified path. |
| `nextTransitionAt` disagrees with the state the same response reports | Both come from one `Evaluate(now)` call on one `now` — they cannot diverge within a response. |
| Schedule staleness across a day boundary (schedule computed yesterday) | Pre-existing and out of scope: the time-of-day comparison makes it work, and `nextTransitionAt` is projected onto today, so the UI is *more* correct than a raw trip row. Called out in `docs/rest.md`. |
| The map-flow rail's HTML-over-SVG layout drifts on narrow viewports | Rail lives in its own `overflow-x: auto` container with a min-width, so it scrolls rather than reflows. |

---

## 7. Deviations from the PRD

1. **New attribute names.** FR-6.3's `boardingWindowDuration` / `preDepartureDuration` /
   `travelDuration` become `boardingWindowSeconds` / `preDepartureSeconds` / `travelDurationSeconds`,
   plus `cycleIntervalSeconds`. FR-6.5 asks for one unit-explicit rule; these names are it, and the
   unsuffixed names would collide with the legacy nanosecond semantics on the sibling resource.
2. **`createdAt` added to `instance-status`** (§3.5). PRD §5 said the shape was unchanged; this is
   one additive field, justified by FR-3.3 needing the same quantity the server force-warps on.
3. **`instance-routes` gains `…Seconds` fields** (§3.3). Implied by FR-6.5's "scheduled and
   instance routes alike," not enumerated in FR-6.3.
4. **`Extract` round-trip fix** (§3.3). `transport/rest.go:127` silently drops the route's
   durations today; the new fields would inherit the gap.

Everything else follows the PRD as written. All four open questions in PRD §9 are resolved above:
Q1 §3.4, Q2 §3.2, Q3 §4.9, Q4 §4.7.

---

## 8. Verification plan

Per CLAUDE.md §Build & Verification, from the worktree root:

- `go test -race ./...`, `go vet ./...`, `go build ./...` in `services/atlas-transports`.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` (nvm 22 on PATH).
- `npm run build` **and** `npm run test` in `services/atlas-ui` — the build is what type-checks.
- No `go.mod` change is expected, so no `docker buildx bake`; if one appears, bake
  `atlas-transports`.
- Code review before the PR: `plan-adherence-reviewer`, `backend-guidelines-reviewer`,
  `frontend-guidelines-reviewer`.

Manual acceptance against a live tenant: the Scheduled tab shows twelve routes with ticking
countdowns; opening a shared-vessel route shows two lanes with a visible turnaround gap; the
instance status endpoint returns live instances (it returns an empty list on `main` for the same
request — that diff is the FR-6.1 proof).
