# Transports Surface in atlas-ui — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-06
---

## 1. Overview

`atlas-transports` runs two unrelated transit models — scheduled routes (ferries, trains,
planes, driven entirely by the wall clock) and instance routes (Ereve sky-ferries, ephemeral
per-boarding instances) — and already serves both over REST. Nothing in atlas-ui reads any of
it. An operator today has no way to answer "is the Orbis boat boarding right now", "why is this
route permanently `awaiting_return`", or "is a player stuck on a ferry" without reading Redis or
service logs.

This task adds a read-only **Operations → Transports** surface: a live departure board, a route
detail page that draws the map flow and the shared-vessel timeline, and a live-instance panel.
It also lands the three backend changes that make those views possible — most importantly a
tenant-scoping bug that currently makes the live-instance endpoint return an empty list for every
real tenant.

The surface is deliberately read-only. Route, vessel, and instance-route definitions live in
tenant configuration alongside handlers and writers; editing them is a separate, larger surface
that belongs under Deployment, not under Operations.

## 2. Goals

Primary goals:

- Give an operator a single tenant-scoped page that shows every transport route and its current
  state, refreshed live.
- Make the shared-vessel mechanic legible. Six of the twelve scheduled routes are paired onto
  three-plus hulls that alternate directions; this is the single most confusing property of the
  domain and the reason half the board sits in `awaiting_return` at any instant.
- Make live instance transports observable, including instances approaching their stuck-timeout
  force-warp.
- Fix `instance/resource.go`'s `uuid.Nil` tenant scoping so the instance status endpoint returns
  real data.
- Surface configuration faults that are currently silent — an `out_of_service` route, and a
  vessel whose `routeAID`/`routeBID` matches no route.

Non-goals:

- Editing routes, vessels, or instance-route configuration from this surface.
- A world-map or geographic overlay. Map ids render through the existing `MapCell` component
  (name badge + copyable id tooltip) and link to `/maps/:id`; that is the whole map treatment.
- Kafka-driven or WebSocket push. Freshness comes from polling.
- Any change to warping, scheduling, or state-transition behaviour.
- Exposing character ids aboard an instance. Counts only (decided during spec).
- A full-day trip table on the route detail page. A windowed timeline only (decided during spec).

## 3. User Stories

- As an operator, I want to see every scheduled route with its current state and a countdown to
  its next state change, so I can confirm transport is running without logging into the game.
- As an operator, I want to see which routes share a hull and how their trips interleave, so I
  understand why a route I expect to be boarding is `awaiting_return`.
- As an operator, I want to open a route and see the exact map chain a player traverses, so I can
  cross-check a report of "the boat dropped me in the wrong place".
- As an operator, I want to see live instance transports and how long each has been aloft, so I
  can spot a character stuck on a transit map before the stuck-timeout force-warps them.
- As an operator, I want a route with no scheduled trips today to read as a fault rather than a
  quiet status, so a bad `cycleInterval` is caught rather than ignored.
- As an operator, I want a vessel that references a non-existent route to be flagged, because
  that silently zeroes both of its routes' schedules.

## 4. Functional Requirements

### FR-1 — Transports board (index)

- **FR-1.1** A new route `/transports` renders a page titled "Transports", reachable from a single
  new sidebar entry under the existing **Operations** group in
  `src/components/app-sidebar-items.ts`. Operations is tenant-switched, which is correct: all
  transport data is tenant-scoped.
- **FR-1.2** The page presents three tabs: **Scheduled**, **Instance**, **Vessels**. Each tab
  label carries a count. Scheduled is the default tab. Tab selection is reflected in the URL
  (e.g. `/transports?tab=instance`) so a tab is linkable and survives reload.
- **FR-1.3** The Scheduled tab is a table with columns: route name, state, next change, start map,
  destination map, vessel, cycle interval. Rows are sorted by state severity
  (`out_of_service` first, then `in_transit`, `locked_entry`, `open_entry`, `awaiting_return`),
  then by route name, so faults and imminent changes sort to the top.
- **FR-1.4** State renders as a pill whose colour and label encode the `RouteState` value.
  `out_of_service` renders in a fault treatment distinct from the other four, because it means the
  scheduler produced no trips for that route today.
- **FR-1.5** The "next change" column shows a countdown (`mm:ss`, or `h:mm:ss` past one hour) to the
  next state transition, with a label naming the transition ("closes in", "departs in",
  "arrives in", "boards in"). For `out_of_service` it shows an em dash.
- **FR-1.6** Start and destination map ids render via the existing `MapCell` component, which
  resolves the map name, shows the raw id in a copyable tooltip, and is already cached
  process-wide.
- **FR-1.7** Route name is a link to the route detail page at `/transports/routes/:routeId`.
- **FR-1.8** The vessel column shows the shared vessel a route belongs to, or an em dash for an
  independent route. It links to the Vessels tab anchored on that vessel.

### FR-2 — Route detail

- **FR-2.1** `/transports/routes/:routeId` renders one scheduled route. The `routeId` is the
  route's UUID, which is stable across restarts and replicas — it is derived via
  `tenant.DerivedId(tenantId, "routes", slug)` (`transport/config/rest.go:88`), so deep links are
  durable and may be bookmarked.
- **FR-2.2** The header shows the route name, its current state pill, and the countdown to its
  next transition.
- **FR-2.3** A **map-flow rail** renders the ordered map chain as stops joined by legs:
  `startMap → stagingMap → enRouteMapIds[0..n] → destinationMap`. Each leg is captioned with what
  moves a character across it ("walk in", "warp on departure", "warp on arrival"). Each stop
  renders its map through `MapCell`. When the route is `in_transit`, the en-route segment is
  visually emphasised.
- **FR-2.4** The `observationMapId` is shown as an annotation, not as a stop in the chain — it is
  where ARRIVED/DEPARTED effects fire, not somewhere a character travels.
- **FR-2.5** A key/value strip shows: observation map, boarding window, pre-departure duration,
  travel duration, cycle interval, trips scheduled today, and the shared vessel (if any).
- **FR-2.6** A **vessel timeline** renders a windowed strip covering roughly the current hour
  (approximately −30 to +30 minutes). Each trip renders as three contiguous segments — boarding
  open, locked, in transit — positioned proportionally in time, with a NOW marker. Times come from
  the route's `trip-schedule` relationship, never from client-side scheduler arithmetic
  (see FR-6.4).
- **FR-2.7** For a route belonging to a shared vessel, the timeline shows **both** directions as
  two lanes on one figure, so the alternation and the turnaround gap are visible. For an
  independent route it shows one lane.
- **FR-2.8** When a route has no scheduled trips today, the timeline area is replaced by an
  explicit fault message naming the likely cause (a `cycleInterval`/`travelDuration` combination
  that leaves no trip fitting inside the day, or an unpaired vessel).

### FR-3 — Instance tab and live instances

- **FR-3.1** The Instance tab lists every instance route with: name, live instance count,
  capacity, boarding window, travel duration, start map, destination map.
- **FR-3.2** A route row with one or more live instances is expandable. Expanding it lists each
  live instance: short instance id, state (`boarding` / `in_transit`), characters aboard as a
  count, and a countdown — to boarding close while boarding, to arrival while in transit.
- **FR-3.3** An instance whose age exceeds two thirds of its route's `MaxLifetime()`
  (`2 × (boardingWindow + travelDuration)`, `instance/model.go:73`) is flagged, because it is
  approaching the stuck-timeout that force-warps its occupants back to the start map.
- **FR-3.4** Instance ids are shown truncated with the full UUID available in a copyable tooltip.
- **FR-3.5** A route with zero live instances renders its count as `0` and is not expandable. This
  is the normal steady state and must not read as an error.

### FR-4 — Vessels tab

- **FR-4.1** The Vessels tab lists each shared vessel with: vessel name, route A, route B, and
  turnaround delay.
- **FR-4.2** Route A and route B are resolved to the corresponding scheduled routes and rendered
  with their current state pills, so the alternation is visible at a glance.
- **FR-4.3** Resolution is by route **name**, matching what the backend itself does
  (`Scheduler.computeSharedVesselSchedule` compares `route.Name() == vessel.RouteAID()`,
  `transport/scheduler.go:92`). The UI must not invent a different matching rule.
- **FR-4.4** A vessel whose `routeAID` or `routeBID` resolves to no route is flagged as a
  configuration fault. This is not cosmetic: `scheduler.go:100-102` returns an empty schedule
  for an unresolved vessel, which drives **both** of its routes to `out_of_service`.

### FR-5 — Freshness

- **FR-5.1** All transport queries use React Query with a 30-second `refetchInterval`, and
  continue refetching while the tab is focused.
- **FR-5.2** Countdowns tick locally once per second between refetches, derived from a
  server-supplied absolute timestamp, so the board reads as live without a 1-second poll.
- **FR-5.3** A countdown that reaches zero holds at `0:00` until the next refetch supplies a new
  target. It must not go negative and must not roll over speculatively.
- **FR-5.4** The page header shows the freshness state: a live indicator, the age of the last
  successful fetch, and an explicit stale/error treatment when a refetch fails.
- **FR-5.5** Queries are gated on `activeTenant` being set, consistent with every other hook in
  `src/lib/hooks/api/`. A tenant switch clears the cache via the existing `TenantProvider`
  effect; no transport-specific invalidation is needed.

### FR-6 — Backend changes

- **FR-6.1** **Fix the instance status tenant scoping.** `GetInstanceRouteStatusHandler`
  (`instance/resource.go:107`) passes `uuid.Nil` into `GetInstancesByRoute`, so it reads the
  nil-tenant Redis set while `FindOrCreateInstance` writes under the real tenant id
  (`instance/instance_registry.go:136`). For any live tenant the endpoint returns an empty list.
  The handler must derive the tenant from the request context
  (`tenant.MustFromContext(d.Context())`) like every other tenant-scoped read in the service.
- **FR-6.2** `TestGetInstanceRouteStatusPaginates` (`instance/resource_paginate_test.go:151`)
  currently documents this as "a pre-existing quirk" and seeds its fixtures under `uuid.Nil` so the
  handler can find them. It must be re-pointed at a real tenant, and a regression test added that
  proves an instance created under tenant A is **not** returned for tenant B.
- **FR-6.3** Add `boardingWindowDuration`, `preDepartureDuration`, and `travelDuration` to
  `transport.RestModel` (`transport/rest.go:14`), which currently exposes only `cycleInterval`.
  These are the route's configured shape and must remain readable even when the route has no
  trips scheduled — precisely the `out_of_service` case where they cannot be derived from
  schedule rows.
- **FR-6.4** Add a server-computed `nextTransitionAt` (absolute RFC3339 timestamp) and
  `nextState` to `transport.RestModel`. **Rationale, and why this is not optional:**
  - `Model.processStateChange` (`transport/model.go:108`) compares only *time-of-day*; it strips
    the date from both `now` and every trip time. The schedule itself is computed once at startup
    for that day (`main.go:100-107`; the 1-second ticker only re-derives state from the existing
    schedule). The consequence is that `TripScheduleModel` timestamps carry a **stale date** —
    only their time-of-day component is meaningful. A client that renders `departure` as an
    absolute instant will display the deploy day's date.
  - Reconstructing the next transition client-side therefore requires reproducing both the
    time-of-day comparison *and*, for shared-vessel routes, `computeSharedVesselSchedule`'s
    alternation and turnaround arithmetic. That is the scheduler reimplemented in TypeScript.
  - `processStateChange` already locates the governing trip and its four boundaries; returning the
    next boundary as an absolute instant alongside the state is a small, local change and removes
    the entire class of client-side date bugs.
- **FR-6.5** Serialised durations: `time.Duration` marshals as an integer nanosecond count, so
  `cycleInterval` currently reaches a client as `900000000000`. The three fields added in FR-6.3
  must not repeat that trap. Either serialise seconds under explicitly named fields
  (`travelDurationSeconds`, matching the convention `instance-routes` configuration already uses)
  or provide a single documented duration decoder in the UI. One rule, applied to scheduled and
  instance routes alike.
- **FR-6.6** No new endpoint is added for vessels. The UI reads them from the existing
  configuration endpoint (see §5).

## 5. API Surface

### Modified — `GET /api/transports/routes` and `/api/transports/routes/{routeId}`

Resource type `routes`. Existing attributes: `name`, `startMapId`, `stagingMapId`,
`enRouteMapIds`, `destinationMapId`, `observationMapId`, `state`, `cycleInterval`. To-many
relationship `schedule` → `trip-schedule`.

Added attributes:

| Field | Type | Notes |
|---|---|---|
| `boardingWindowDuration` | duration | Per FR-6.5 serialisation rule |
| `preDepartureDuration` | duration | Per FR-6.5 serialisation rule |
| `travelDuration` | duration | Per FR-6.5 serialisation rule |
| `nextTransitionAt` | string (RFC3339) | Absolute instant of the next state change; empty when `out_of_service` |
| `nextState` | string | The `RouteState` the route transitions to at `nextTransitionAt`; empty when `out_of_service` |

Unchanged: pagination (`page[number]`, `page[size]`, default 50, max 250), the
`filter[startMapId]` filter, and the stable derived route UUID as the resource id.

**Payload note.** `Transform` attaches the full day's `trip-schedule` rows to every route
(`transport/rest.go:108`), and a 15-minute cycle yields 96 trips per route. A board fetch of all
twelve routes therefore carries on the order of 1,000 included resources every 30 seconds. The
board must not require that payload — with `nextTransitionAt` it does not need any schedule rows
at all. See §9 for the sparse-fieldset question.

### Fixed — `GET /api/transports/instance-routes/{routeId}/status`

Resource type `instance-status`. Shape unchanged: `routeId`, `state`, `characters` (count),
`boardingUntil`, `arrivalAt`. Behaviour changes from "always empty for a real tenant" to
"returns that tenant's live instances". Paginated as today.

### Unchanged — `GET /api/transports/instance-routes`, `/{routeId}`

Resource type `instance-routes`: `name`, `startMapId`, `transitMapIds`, `destinationMapId`,
`capacity`, `boardingWindow`, `travelDuration`. Consumed as-is.

### Reused — `GET /api/tenants/{tenantId}/configurations/vessels`

Served by atlas-tenants (`configuration/resource.go:220`), routed by nginx at
`deploy/shared/routes.conf:512`. Resource type `vessels`, resource id is the config slug.
Attributes: `uuid`, `name`, `routeAID`, `routeBID`, `turnaroundDelay` (seconds, `uint32`).

Vessels are pure configuration with no runtime state: `LoadConfigurationsForTenant` hands them to
`NewScheduler(...)` (`transport/processor.go:70`) and never stores them, so there is no vessel
registry in atlas-transports to serve from. Reading them from configuration is the same pattern
the UI already uses for handlers, writers, worlds, and MTS config. This endpoint is paginated;
the UI drains it with the existing `fetchAll` helper.

### Not added

`GET /transports/vessels`. Decided during spec — it would require introducing Redis persistence
in atlas-transports for data that has no runtime state.

## 6. Data Model

No schema changes. No migrations. No new persisted entities in any service.

Facts the implementation must respect:

- **Route identity is derived and stable.** `tenant.DerivedId(tenantId, "routes", slug)` produces
  the same UUID on every replica and across restarts (`transport/config/rest.go:88`; the same
  scheme applies to `instance-routes`). Route detail deep links are durable.
- **Vessel identity is a slug, not a UUID.** `SharedVesselModel.Id()` is the configuration slug.
  Vessels reference their routes by **name**, and the scheduler matches on name. Do not assume
  slug and name are interchangeable even though the seed data currently makes them equal.
- **Trip-schedule timestamps are time-of-day, not instants.** See FR-6.4. Any UI that formats a
  schedule row must format the time component only, and any comparison against "now" must be a
  time-of-day comparison.
- **Instance state lives in Redis, keyed by tenant.** Instance metadata and per-instance character
  sets are separate keys; the per-route set is tenant-scoped, which is exactly what FR-6.1 fixes.

## 7. Service Impact

### atlas-ui

- `src/services/api/transports.service.ts` — new. Route list/detail, instance-route list, instance
  status, and the vessels read from the tenant configuration endpoint. Thin adapters over
  `lib/api/client`, using `fetchAll` from `services/api/pagination` for the collections.
- `src/lib/hooks/api/useTransports.ts` — new. `transportKeys` factory plus query hooks, gated on
  `activeTenant`, with the 30s `refetchInterval`.
- `src/types/models/transport.ts` — new. JSON:API data/attribute types for `routes`,
  `trip-schedule`, `instance-routes`, `instance-status`, and `vessels`.
- `src/pages/TransportsPage.tsx`, `src/pages/TransportRouteDetailPage.tsx`, and colocated
  `transports-columns.tsx` — new, following the existing page/columns convention.
- Feature components under `src/components/features/transports/` — the map-flow rail, the vessel
  timeline, the state pill, and the countdown hook.
- `src/App.tsx` — two lazy routes via `lazyWithReload` (not bare `React.lazy`).
- `src/components/app-sidebar-items.ts` — one entry appended to the Operations group.

### atlas-transports

- `transport/rest.go` — five added attributes on `RestModel` (FR-6.3, FR-6.4) plus the
  serialisation decision from FR-6.5.
- `transport/model.go` — expose the next-transition boundary alongside the derived state, so
  `Transform` can populate `nextTransitionAt`/`nextState` without duplicating the comparison
  logic.
- `instance/resource.go` — tenant-scoping fix (FR-6.1).
- `instance/resource_paginate_test.go` — re-point off `uuid.Nil` (FR-6.2).
- `docs/rest.md` and `docs/domain.md` — regenerate for the changed REST models.

### atlas-tenants

No code change. Its existing vessels configuration endpoint gains a second consumer.

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every request carries the four tenant headers via `api.setTenant`. No
  transport data may be rendered for a tenant other than `activeTenant`. FR-6.1's regression test
  is the backend half of this guarantee.
- **Payload.** A board refresh must stay well under the ~1,000-included-resource shape described
  in §5. Target: a Scheduled-tab refresh transfers no more than the twelve route resources
  themselves.
- **Polling cost.** 30-second interval across at most four concurrent queries. Countdown ticking is
  local and must not trigger network activity or re-render the whole table — only the countdown
  cells.
- **Correctness over liveness.** `state` is always the server's value. The UI never derives a
  state from a countdown reaching zero; it waits for the next refetch.
- **Accessibility.** State is never encoded by colour alone — every pill carries its text label.
  The map-flow rail and vessel timeline are SVG with `role="img"` and an `aria-label` stating what
  the figure shows. Both themes are supported, per the existing `ThemeProvider`.
- **Responsiveness.** Tables and both figures scroll inside their own `overflow-x: auto`
  containers; the page body never scrolls horizontally.
- **Observability.** No new metrics. Failed refetches surface in the UI's existing error logger
  and in the page's stale treatment.
- **Security.** Read-only. No new mutation endpoints, no new secrets, no new service-to-service
  auth surface.

## 9. Open Questions

1. **Can the board suppress included `trip-schedule` resources?** JSON:API sparse fieldsets are
   parsed (`jsonapi.ParseQueryFields`), but whether api2go will omit the `included` array for the
   list response needs verifying in the design phase. If it cannot, the fallback is an explicit
   opt-out parameter on the list endpoint. Either way FR-6.4's `nextTransitionAt` is what makes
   the board independent of schedule rows.
2. **`nextTransitionAt` / `nextState` are an addition to the approved Tier 3 scope**, arrived at
   during spec (FR-6.4). They replace, rather than add to, a larger client-side burden — without
   them the UI must reimplement the shared-vessel scheduler. Confirm before planning.
3. **Should the Vessels tab exist, or fold into the Scheduled tab?** Six vessels over twelve
   routes is a small list, and the Scheduled tab already carries a vessel column. Keeping it
   separate is proposed mainly to give FR-4.4's unpaired-vessel fault somewhere to live.
4. **Timeline window size.** ±30 minutes suits the 15-minute boat cycle and the 5-minute subway,
   but the airplane's 6-minute cycle puts ten legs in view. A per-route window derived from cycle
   interval may read better; deferred to design.

## 10. Acceptance Criteria

Functional:

- [ ] A single **Transports** entry appears under Operations and routes to `/transports`.
- [ ] The Scheduled tab lists all twelve seeded routes with a state pill, a labelled countdown, and
      `MapCell`-rendered start/destination maps.
- [ ] Fault sorting holds: an `out_of_service` route sorts above all other states.
- [ ] Route name links to `/transports/routes/:routeId`; the link survives a service restart
      (stable derived id).
- [ ] Route detail renders the full map chain in order, with the observation map annotated
      separately from the traversed maps.
- [ ] For a shared-vessel route, the timeline shows both directions and their turnaround gap;
      for an independent route it shows one lane.
- [ ] No schedule timestamp is rendered with a date component anywhere in the UI.
- [ ] The Instance tab lists all twelve instance routes; a route with live instances expands to
      per-instance rows with countdowns; a route with none shows `0` and does not expand.
- [ ] An instance past two thirds of `MaxLifetime()` is visibly flagged.
- [ ] The Vessels tab resolves both routes of each vessel by name and shows their state pills; an
      unresolvable vessel is flagged as a fault.
- [ ] Countdowns tick every second between 30-second refetches, hold at `0:00`, and never go
      negative.
- [ ] Switching tenants clears and refetches all transport data; no cross-tenant leakage.

Backend:

- [ ] `GET /api/transports/instance-routes/{routeId}/status` returns live instances for a real
      tenant.
- [ ] A regression test proves an instance created under tenant A is not returned for tenant B.
- [ ] `resource_paginate_test.go` no longer seeds under `uuid.Nil`.
- [ ] The route resource exposes `boardingWindowDuration`, `preDepartureDuration`,
      `travelDuration`, `nextTransitionAt`, and `nextState`.
- [ ] Duration serialisation follows one documented rule across scheduled and instance routes.
- [ ] `services/atlas-transports/docs/rest.md` and `domain.md` reflect the changed models.

Verification (per CLAUDE.md §Build & Verification):

- [ ] `go test -race ./...` clean in atlas-transports.
- [ ] `go vet ./...` clean in atlas-transports.
- [ ] `go build ./...` clean in atlas-transports.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` clean from the repo root.
- [ ] `tools/lint.sh --check` clean from the repo root (requires nvm 22 on PATH for the atlas-ui
      half).
- [ ] `npm run build` and `npm run test` clean in `services/atlas-ui` — build, not vitest alone,
      because the build is what type-checks.
- [ ] `docker buildx bake atlas-transports` only if that service's `go.mod` changed (not expected
      for this task).
- [ ] Code review run before opening the PR: `plan-adherence-reviewer`,
      `backend-guidelines-reviewer`, and `frontend-guidelines-reviewer`.
