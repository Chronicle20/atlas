# Transports Surface in atlas-ui — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only Operations → Transports surface to atlas-ui (departure board, route detail with map-flow rail and vessel timeline, live-instance panel) plus the atlas-transports changes that make it possible.

**Architecture:** The server owns state and time — atlas-transports gains a `Model.Evaluate` that returns the current state *and* the absolute instant of the next transition, so the UI never reimplements the scheduler. The route list endpoint stops attaching the day's ~96 trips per route by default (opt-in via `?include=schedule`). The UI polls at 30s and ticks countdowns locally off a single shared clock store, so only countdown leaves re-render.

**Tech Stack:** Go 1.24 (atlas-transports, api2go JSON:API, miniredis-backed tests, testify), React 19 + TypeScript + Vite + TanStack Query/Table + shadcn/ui + Tailwind + Vitest/RTL (atlas-ui).

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-198-transports-ui-surface/` on branch `task-198-transports-ui-surface`. Never edit the main repo.
- **No TODOs, stubs, or 501s in landed commits.** Finish the bounded work or escalate explicitly.
- **Immutable domain models.** Private fields + getters + Builder. Never add a setter to `transport.Model` or `instance.RouteModel`.
- **Test setup uses the project's Builder pattern.** No `*_testhelpers.go` files with test-only constructors.
- **Reuse `libs/atlas-constants`** for map/world/channel ids (`_map.Id` etc.). Do not redefine.
- **Read-only surface.** No new mutation endpoints, no new secrets, no new service-to-service auth.
- **Additive wire changes only.** The pre-existing nanosecond-valued fields (`cycleInterval` on `routes`; `boardingWindow`/`travelDuration` on `instance-routes`) stay exactly as they are and are documented as legacy. Never retype a field in place.
- **Duration rule (FR-6.5):** every *new* duration field is an integer count of seconds with an explicit `…Seconds` suffix — `boardingWindowSeconds`, `preDepartureSeconds`, `travelDurationSeconds`, `cycleIntervalSeconds`. One rule, scheduled and instance routes alike.
- **Trip-schedule timestamps carry a stale date.** Only their time-of-day component is meaningful. No UI code may render a schedule timestamp with a date component. The single absolute instant the UI trusts is `nextTransitionAt`.
- **Repo-relative paths only** in committed files. Never write `/home/<user>/...`.
- **Preserve line endings.** Do not normalize CRLF→LF as a side effect of an edit.
- **Prefer per-file Edit/Write** over shell patch loops.
- Commit after every task. Conventional-commit prefixes (`feat:`, `fix:`, `test:`, `docs:`, `refactor:`).

## File Structure

**atlas-transports** (`services/atlas-transports/atlas.com/transports/`)

| File | Responsibility |
|---|---|
| `transport/model.go` | Add `Transition` type, `Model.Evaluate`, `timeOfDay`, `materializeBoundary`; `processStateChange` becomes a wrapper |
| `transport/rest.go` | Four `…Seconds` fields, `nextTransitionAt`/`nextState`, `TransformSummary`, `Extract` round-trip fix |
| `transport/resource.go` | `include=schedule` opt-in on the list handler |
| `transport/evaluate_test.go` | **new** — table-driven `Evaluate` branch coverage |
| `transport/rest_test.go` | **new** — `Transform` field population + `Extract` round trip |
| `transport/resource_include_test.go` | **new** — `included` absent by default, present with `?include=schedule` |
| `instance/resource.go` | Tenant-scoping fix (FR-6.1) |
| `instance/rest.go` | `…Seconds` fields on `RouteRestModel`, `createdAt` on `InstanceStatusRestModel` |
| `instance/resource_paginate_test.go` | Re-point off `uuid.Nil` |
| `instance/resource_tenant_test.go` | **new** — cross-tenant regression guard |
| `instance/rest_test.go` | **new** — `…Seconds` + `createdAt` transform assertions |
| `docs/rest.md`, `docs/domain.md`, `.bruno/Routes/Get Routes With Schedule.bru` | Documentation |

**atlas-ui** (`services/atlas-ui/src/`)

| File | Responsibility |
|---|---|
| `types/models/transport.ts` | JSON:API shapes + `RouteState`/`InstanceState` unions |
| `services/api/transports.service.ts` | HTTP adapters (5 reads) |
| `lib/hooks/api/useTransports.ts` | `transportKeys` + query hooks, tenant-gated, 30s poll |
| `lib/utils/clock.ts` | One 1-second store + `useClock` |
| `components/features/transports/transport-format.ts` | All pure arithmetic: sorting, labels, time-of-day, windowing, fault detection |
| `components/features/transports/Countdown.tsx` | Leaf countdown, subscribes to the clock alone |
| `components/features/transports/RouteStatePill.tsx` | State → labelled badge |
| `components/features/transports/FreshnessIndicator.tsx` | Live dot + fetch age + stale/error treatment |
| `components/features/transports/MapFlowRail.tsx` | Stop/leg rail for route detail |
| `components/features/transports/VesselTimeline.tsx` | 1–2 lane SVG trip timeline |
| `components/features/transports/InstanceRoutesTable.tsx` | Instance tab, expandable rows |
| `components/features/transports/VesselsTable.tsx` | Vessels tab, name-resolution + fault row |
| `pages/TransportsPage.tsx`, `pages/transports-columns.tsx` | Board shell, tabs, Scheduled columns |
| `pages/TransportRouteDetailPage.tsx` | Route detail |
| `App.tsx`, `components/app-sidebar-items.ts` | Two lazy routes + one sidebar entry |

---

## Task 1: `transport.Transition` and `Model.Evaluate`

Restructure `processStateChange` so the branch that decides the state also names the boundary it is counting down to. This is a pure refactor plus new return data — derived state must not change.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/transport/model.go:108-207`
- Test: `services/atlas-transports/atlas.com/transports/transport/evaluate_test.go` (create)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `type Transition struct { State RouteState; NextState RouteState; NextAt time.Time }`
  - `func (m Model) Evaluate(now time.Time) Transition`
  - `func (m Model) processStateChange(now time.Time) RouteState` (unchanged signature, now a wrapper)

- [ ] **Step 1: Write the failing test**

Create `services/atlas-transports/atlas.com/transports/transport/evaluate_test.go`:

```go
package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// evaluateTestRoute builds a route with a single-trip schedule whose
// boundaries are expressed as offsets from `base`.
func evaluateTestRoute(t *testing.T, routeId uuid.UUID, trips []TripScheduleModel) Model {
	t.Helper()
	m, err := NewBuilder("Evaluate Route").
		SetId(routeId).
		SetStartMapId(_map.Id(100)).
		SetStagingMapId(_map.Id(101)).
		SetEnRouteMapIds([]_map.Id{_map.Id(102)}).
		SetDestinationMapId(_map.Id(103)).
		SetObservationMapId(_map.Id(104)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		SetSchedule(trips).
		Build()
	require.NoError(t, err)
	return m
}

func evaluateTestTrip(t *testing.T, routeId uuid.UUID, open, closed, departure, arrival time.Time) TripScheduleModel {
	t.Helper()
	trip, err := NewTripScheduleBuilder().
		SetTripId(uuid.New()).
		SetRouteId(routeId).
		SetBoardingOpen(open).
		SetBoardingClosed(closed).
		SetDeparture(departure).
		SetArrival(arrival).
		Build()
	require.NoError(t, err)
	return trip
}

func TestEvaluate_Branches(t *testing.T) {
	routeId := uuid.New()
	// The schedule's own date is deliberately NOT the evaluation date -
	// this is the stale-date property Evaluate has to be immune to.
	sched := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	at := func(h, m int) time.Time {
		return time.Date(sched.Year(), sched.Month(), sched.Day(), h, m, 0, 0, time.UTC)
	}
	todayAt := func(h, m int) time.Time {
		return time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC)
	}

	// One trip: boards 12:30, closes 12:35, departs 12:37, arrives 12:47.
	trip := func() []TripScheduleModel {
		return []TripScheduleModel{evaluateTestTrip(t, routeId, at(12, 30), at(12, 35), at(12, 37), at(12, 47))}
	}

	tests := []struct {
		name          string
		now           time.Time
		trips         []TripScheduleModel
		expectedState RouteState
		expectedNext  RouteState
		expectedAt    time.Time
	}{
		{
			name:          "No trips is out of service with a zero boundary",
			now:           now,
			trips:         []TripScheduleModel{},
			expectedState: OutOfService,
			expectedNext:  "",
			expectedAt:    time.Time{},
		},
		{
			name:          "Before boarding opens counts down to boarding open",
			now:           todayAt(12, 0),
			trips:         trip(),
			expectedState: AwaitingReturn,
			expectedNext:  OpenEntry,
			expectedAt:    todayAt(12, 30),
		},
		{
			name:          "Boarding open counts down to boarding closed",
			now:           todayAt(12, 31),
			trips:         trip(),
			expectedState: OpenEntry,
			expectedNext:  LockedEntry,
			expectedAt:    todayAt(12, 35),
		},
		{
			name:          "Locked entry counts down to departure",
			now:           todayAt(12, 36),
			trips:         trip(),
			expectedState: LockedEntry,
			expectedNext:  InTransit,
			expectedAt:    todayAt(12, 37),
		},
		{
			name:          "In transit counts down to arrival",
			now:           todayAt(12, 40),
			trips:         trip(),
			expectedState: InTransit,
			expectedNext:  AwaitingReturn,
			expectedAt:    todayAt(12, 47),
		},
		{
			name:          "Past the last arrival with no other trips is out of service",
			now:           todayAt(13, 0),
			trips:         trip(),
			expectedState: OutOfService,
			expectedNext:  "",
			expectedAt:    time.Time{},
		},
		{
			name: "Past an arrival with a later trip counts down to that trip's boarding open",
			now:  todayAt(13, 0),
			trips: []TripScheduleModel{
				evaluateTestTrip(t, routeId, at(12, 30), at(12, 35), at(12, 37), at(12, 47)),
				evaluateTestTrip(t, routeId, at(14, 0), at(14, 5), at(14, 7), at(14, 17)),
			},
			expectedState: AwaitingReturn,
			expectedNext:  OpenEntry,
			expectedAt:    todayAt(14, 0),
		},
		{
			name: "Midnight-crossing trip in transit counts down to arrival after midnight",
			now:  todayAt(23, 50),
			trips: []TripScheduleModel{
				evaluateTestTrip(t, routeId, at(23, 30), at(23, 35), at(23, 40), at(0, 20)),
			},
			expectedState: InTransit,
			expectedNext:  AwaitingReturn,
			expectedAt:    todayAt(0, 20).Add(24 * time.Hour),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := evaluateTestRoute(t, routeId, tc.trips)
			got := m.Evaluate(tc.now)
			assert.Equal(t, tc.expectedState, got.State)
			assert.Equal(t, tc.expectedNext, got.NextState)
			assert.True(t, got.NextAt.Equal(tc.expectedAt),
				"NextAt = %s, want %s", got.NextAt, tc.expectedAt)
		})
	}
}

// TestEvaluate_StateMatchesProcessStateChange pins the refactor: the wrapper
// must keep returning exactly what Evaluate decided.
func TestEvaluate_StateMatchesProcessStateChange(t *testing.T) {
	routeId := uuid.New()
	sched := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)
	m := evaluateTestRoute(t, routeId, []TripScheduleModel{
		evaluateTestTrip(t, routeId, sched, sched.Add(5*time.Minute), sched.Add(7*time.Minute), sched.Add(17*time.Minute)),
	})
	for _, offset := range []time.Duration{0, 6 * time.Minute, 8 * time.Minute, 20 * time.Minute} {
		now := time.Date(2026, 8, 6, 12, 30, 0, 0, time.UTC).Add(offset)
		assert.Equal(t, m.Evaluate(now).State, m.processStateChange(now))
	}
}

// TestEvaluate_NextAtIsAlwaysInTheFuture guards the materialization rule.
func TestEvaluate_NextAtIsAlwaysInTheFuture(t *testing.T) {
	routeId := uuid.New()
	sched := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	trips := []TripScheduleModel{
		evaluateTestTrip(t, routeId,
			sched.Add(8*time.Hour), sched.Add(8*time.Hour+5*time.Minute),
			sched.Add(8*time.Hour+7*time.Minute), sched.Add(8*time.Hour+17*time.Minute)),
	}
	m := evaluateTestRoute(t, routeId, trips)
	base := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	for minute := 0; minute < 24*60; minute += 7 {
		now := base.Add(time.Duration(minute) * time.Minute)
		got := m.Evaluate(now)
		require.Equal(t, OpenEntry != "", true)
		if got.State == OutOfService {
			continue
		}
		assert.True(t, got.NextAt.After(now), "at %s NextAt = %s is not in the future", now, got.NextAt)
		assert.True(t, got.NextAt.Sub(now) <= 24*time.Hour, "at %s NextAt = %s is more than a day out", now, got.NextAt)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run 'TestEvaluate' -v
```

Expected: FAIL — `m.Evaluate undefined (type Model has no field or method Evaluate)`.

- [ ] **Step 3: Implement `Transition`, `Evaluate`, and the helpers**

In `transport/model.go`, replace the whole of `processStateChange` (lines 108-207) with the following, and add the `Transition` type just above `UpdateState`:

```go
// Transition is the result of evaluating a route's schedule against a moment
// in time: the state the route is in now, the state it moves to next, and the
// absolute instant of that move.
//
// Trip-schedule timestamps carry the date of the day the schedule was computed
// and only their time-of-day component is meaningful (the schedule is computed
// once per reconcile; the 1-second ticker only re-derives state from it). NextAt
// is that time-of-day boundary projected onto the first instant strictly after
// `now`, so - unlike a raw trip row - it is safe to render as an absolute
// timestamp. When State is OutOfService there is no boundary: NextState is ""
// and NextAt is the zero time.
type Transition struct {
	State     RouteState
	NextState RouteState
	NextAt    time.Time
}

// timeOfDay strips the date from t, leaving only a comparable time of day.
// Every schedule comparison in this file goes through it.
func timeOfDay(t time.Time) time.Time {
	return time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// materializeBoundary projects a time-of-day boundary onto the first instant
// strictly after now, in now's own date and location - the same frame the
// time-of-day comparisons use, so the state and the boundary can never
// disagree about which side of a transition we are on.
func materializeBoundary(now time.Time, boundary time.Time) time.Time {
	at := time.Date(now.Year(), now.Month(), now.Day(),
		boundary.Hour(), boundary.Minute(), boundary.Second(), boundary.Nanosecond(),
		now.Location())
	if !at.After(now) {
		at = at.Add(24 * time.Hour)
	}
	return at
}

// Evaluate derives the route's state at `now` together with the transition it
// is counting down to. The trip-selection and branch structure is the state
// machine this service has always run; each branch now also names the boundary
// it is waiting on.
func (m Model) Evaluate(now time.Time) Transition {
	var nextTrip *TripScheduleModel
	var inTransitTrip *TripScheduleModel
	var futureTrip *TripScheduleModel

	nowTimeOfDay := timeOfDay(now)

	for i := range m.Schedule() {
		trip := m.schedule[i]
		if trip.RouteId() != m.Id() {
			continue
		}

		tripDepartureTimeOfDay := timeOfDay(trip.Departure())
		tripArrivalTimeOfDay := timeOfDay(trip.Arrival())

		if tripArrivalTimeOfDay.Before(tripDepartureTimeOfDay) {
			// Arrival is before departure in time of day: the trip crosses midnight.
			if nowTimeOfDay.After(tripDepartureTimeOfDay) || nowTimeOfDay.Before(tripArrivalTimeOfDay) {
				if inTransitTrip == nil || tripDepartureTimeOfDay.After(timeOfDay(inTransitTrip.Departure())) {
					inTransitTrip = &trip
				}
			}
		} else {
			if nowTimeOfDay.After(tripDepartureTimeOfDay) && nowTimeOfDay.Before(tripArrivalTimeOfDay) {
				if inTransitTrip == nil || tripDepartureTimeOfDay.After(timeOfDay(inTransitTrip.Departure())) {
					inTransitTrip = &trip
				}
			}
		}

		if tripDepartureTimeOfDay.After(nowTimeOfDay) {
			if futureTrip == nil || tripDepartureTimeOfDay.Before(timeOfDay(futureTrip.Departure())) {
				futureTrip = &trip
			}
		}
	}

	// Prioritize in-transit trips over future trips
	if inTransitTrip != nil {
		nextTrip = inTransitTrip
	} else {
		nextTrip = futureTrip
	}

	if nextTrip == nil {
		return Transition{State: OutOfService}
	}

	to := func(state RouteState, next RouteState, boundary time.Time) Transition {
		return Transition{State: state, NextState: next, NextAt: materializeBoundary(now, boundary)}
	}

	boardingOpen := timeOfDay(nextTrip.BoardingOpen())
	boardingClosed := timeOfDay(nextTrip.BoardingClosed())
	departure := timeOfDay(nextTrip.Departure())
	arrival := timeOfDay(nextTrip.Arrival())

	if arrival.Before(departure) {
		// Midnight-crossing trip.
		if nowTimeOfDay.Before(boardingOpen) && nowTimeOfDay.After(arrival) {
			return to(AwaitingReturn, OpenEntry, boardingOpen)
		} else if nowTimeOfDay.Before(boardingClosed) {
			return to(OpenEntry, LockedEntry, boardingClosed)
		} else if nowTimeOfDay.Before(departure) {
			return to(LockedEntry, InTransit, departure)
		}
		return to(InTransit, AwaitingReturn, arrival)
	}

	if nowTimeOfDay.Before(boardingOpen) {
		return to(AwaitingReturn, OpenEntry, boardingOpen)
	} else if nowTimeOfDay.Before(boardingClosed) {
		return to(OpenEntry, LockedEntry, boardingClosed)
	} else if nowTimeOfDay.Before(departure) {
		return to(LockedEntry, InTransit, departure)
	} else if nowTimeOfDay.Before(arrival) {
		return to(InTransit, AwaitingReturn, arrival)
	}
	return Transition{State: OutOfService}
}

func (m Model) processStateChange(now time.Time) RouteState {
	return m.Evaluate(now).State
}
```

- [ ] **Step 4: Run the new test and the pre-existing state suite**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run 'TestEvaluate|TestStateMachine' -v
```

Expected: PASS. `TestStateMachine_UpdateState` must pass **without being edited** — it is the guard that the refactor did not change derived state.

- [ ] **Step 5: Run the whole module with the race detector**

```bash
cd services/atlas-transports/atlas.com/transports && go test -race ./... && go vet ./...
```

Expected: PASS, no vet findings.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/transport/model.go \
        services/atlas-transports/atlas.com/transports/transport/evaluate_test.go
git commit -m "refactor(transports): derive next transition alongside route state via Model.Evaluate"
```

---

## Task 2: `…Seconds` duration fields on `routes` + `Extract` round-trip fix

`time.Duration` marshals as an integer nanosecond count, so `cycleInterval` reaches a client as `900000000000`. Add unit-explicit second-valued fields alongside it and fix `Extract`, which today silently drops the route's three configured durations.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/transport/rest.go:14-25` (struct), `:107-151` (`Transform`, `Extract`)
- Test: `services/atlas-transports/atlas.com/transports/transport/rest_test.go` (create)

**Interfaces:**
- Consumes: nothing from Task 1.
- Produces: `RestModel` gains `CycleIntervalSeconds`, `BoardingWindowSeconds`, `PreDepartureSeconds`, `TravelDurationSeconds` — all `uint32`, JSON `cycleIntervalSeconds` / `boardingWindowSeconds` / `preDepartureSeconds` / `travelDurationSeconds`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-transports/atlas.com/transports/transport/rest_test.go`:

```go
package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func restTestRoute(t *testing.T) Model {
	t.Helper()
	m, err := NewBuilder("Rest Route").
		SetId(uuid.New()).
		SetStartMapId(_map.Id(100)).
		SetStagingMapId(_map.Id(101)).
		SetEnRouteMapIds([]_map.Id{_map.Id(102)}).
		SetDestinationMapId(_map.Id(103)).
		SetObservationMapId(_map.Id(104)).
		SetState(AwaitingReturn).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(15 * time.Minute).
		Build()
	require.NoError(t, err)
	return m
}

func TestTransform_PopulatesSecondsFields(t *testing.T) {
	rm, err := Transform(restTestRoute(t))
	require.NoError(t, err)

	assert.Equal(t, uint32(300), rm.BoardingWindowSeconds)
	assert.Equal(t, uint32(120), rm.PreDepartureSeconds)
	assert.Equal(t, uint32(600), rm.TravelDurationSeconds)
	assert.Equal(t, uint32(900), rm.CycleIntervalSeconds)

	// The legacy nanosecond field is untouched.
	assert.Equal(t, 15*time.Minute, rm.CycleInterval)
}

func TestExtract_RoundTripsDurations(t *testing.T) {
	original := restTestRoute(t)
	rm, err := Transform(original)
	require.NoError(t, err)

	back, err := Extract(rm)
	require.NoError(t, err)

	assert.Equal(t, original.BoardingWindowDuration(), back.BoardingWindowDuration())
	assert.Equal(t, original.PreDepartureDuration(), back.PreDepartureDuration())
	assert.Equal(t, original.TravelDuration(), back.TravelDuration())
	assert.Equal(t, original.CycleInterval(), back.CycleInterval())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run 'TestTransform_PopulatesSecondsFields|TestExtract_RoundTripsDurations' -v
```

Expected: FAIL — `rm.BoardingWindowSeconds undefined`.

- [ ] **Step 3: Add the fields and wire them**

In `transport/rest.go`, replace the `RestModel` struct (lines 14-25) with:

```go
// RestModel is the JSON:API resource for a transport route.
//
// CycleInterval is a time.Duration and therefore serialises as an integer
// nanosecond count. It is retained unchanged for existing consumers; new
// consumers read the unit-explicit *Seconds fields below.
type RestModel struct {
	ID                    uuid.UUID               `json:"-"`
	Name                  string                  `json:"name"`
	StartMapID            _map.Id                 `json:"startMapId"`
	StagingMapID          _map.Id                 `json:"stagingMapId"`
	EnRouteMapIDs         []_map.Id               `json:"enRouteMapIds"`
	DestinationMapID      _map.Id                 `json:"destinationMapId"`
	ObservationMapID      _map.Id                 `json:"observationMapId"`
	State                 string                  `json:"state"`
	CycleInterval         time.Duration           `json:"cycleInterval"`
	BoardingWindowSeconds uint32                  `json:"boardingWindowSeconds"`
	PreDepartureSeconds   uint32                  `json:"preDepartureSeconds"`
	TravelDurationSeconds uint32                  `json:"travelDurationSeconds"`
	CycleIntervalSeconds  uint32                  `json:"cycleIntervalSeconds"`
	Schedule              []TripScheduleRestModel `json:"-"`
}
```

In `Transform` (line 113), add the four fields to the returned literal, just after `CycleInterval`:

```go
		CycleInterval:         m.CycleInterval(),
		BoardingWindowSeconds: uint32(m.BoardingWindowDuration().Seconds()),
		PreDepartureSeconds:   uint32(m.PreDepartureDuration().Seconds()),
		TravelDurationSeconds: uint32(m.TravelDuration().Seconds()),
		CycleIntervalSeconds:  uint32(m.CycleInterval().Seconds()),
		Schedule:              schedule,
```

In `Extract` (line 141), add the three missing setters — the route's configured shape is currently dropped on the way back:

```go
	return NewBuilder(r.Name).
		SetStartMapId(r.StartMapID).
		SetStagingMapId(r.StagingMapID).
		SetEnRouteMapIds(r.EnRouteMapIDs).
		SetDestinationMapId(r.DestinationMapID).
		SetObservationMapId(r.ObservationMapID).
		SetState(RouteState(r.State)).
		SetSchedule(schedule).
		SetBoardingWindowDuration(time.Duration(r.BoardingWindowSeconds) * time.Second).
		SetPreDepartureDuration(time.Duration(r.PreDepartureSeconds) * time.Second).
		SetTravelDuration(time.Duration(r.TravelDurationSeconds) * time.Second).
		SetCycleInterval(r.CycleInterval).
		Build()
```

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./transport/ -v
```

Expected: PASS, including the pre-existing paginate and scheduler tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/transport/rest.go \
        services/atlas-transports/atlas.com/transports/transport/rest_test.go
git commit -m "feat(transports): expose unit-explicit second durations on routes and round-trip them in Extract"
```

---

## Task 3: `nextTransitionAt` / `nextState` on the route resource

Expose Task 1's `Transition` on the wire so the board can count down without reimplementing the scheduler.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/transport/rest.go` (struct + `Transform`)
- Test: `services/atlas-transports/atlas.com/transports/transport/rest_test.go` (extend)

**Interfaces:**
- Consumes: `Model.Evaluate(now) Transition` from Task 1; the `RestModel` layout from Task 2.
- Produces: `RestModel.NextTransitionAt string` (JSON `nextTransitionAt`, RFC3339 or `""`), `RestModel.NextState string` (JSON `nextState`, a `RouteState` value or `""`).

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-transports/atlas.com/transports/transport/rest_test.go`:

```go
// pinTimeNow swaps the package clock seam for the duration of a test.
func pinTimeNow(t *testing.T, at time.Time) {
	t.Helper()
	previous := timeNow
	timeNow = func() time.Time { return at }
	t.Cleanup(func() { timeNow = previous })
}

func TestTransform_PopulatesNextTransition(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	pinTimeNow(t, now)

	routeId := uuid.New()
	sched := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(h, m int) time.Time {
		return time.Date(sched.Year(), sched.Month(), sched.Day(), h, m, 0, 0, time.UTC)
	}
	trip, err := NewTripScheduleBuilder().
		SetTripId(uuid.New()).
		SetRouteId(routeId).
		SetBoardingOpen(at(12, 30)).
		SetBoardingClosed(at(12, 35)).
		SetDeparture(at(12, 37)).
		SetArrival(at(12, 47)).
		Build()
	require.NoError(t, err)

	m, err := NewBuilder("Next Transition Route").
		SetId(routeId).
		SetStartMapId(_map.Id(100)).
		SetStagingMapId(_map.Id(101)).
		SetDestinationMapId(_map.Id(103)).
		SetObservationMapId(_map.Id(104)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(15 * time.Minute).
		SetSchedule([]TripScheduleModel{trip}).
		Build()
	require.NoError(t, err)

	rm, err := Transform(m)
	require.NoError(t, err)

	assert.Equal(t, string(OpenEntry), rm.NextState)
	assert.Equal(t, "2026-08-06T12:30:00Z", rm.NextTransitionAt)
	// state is whatever Evaluate decided on the same `now` - they cannot diverge.
	assert.Equal(t, string(AwaitingReturn), rm.State)
}

func TestTransform_OutOfServiceHasNoTransition(t *testing.T) {
	pinTimeNow(t, time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC))

	rm, err := Transform(restTestRoute(t))
	require.NoError(t, err)

	assert.Equal(t, string(OutOfService), rm.State)
	assert.Equal(t, "", rm.NextState)
	assert.Equal(t, "", rm.NextTransitionAt)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run 'TestTransform_PopulatesNextTransition|TestTransform_OutOfServiceHasNoTransition' -v
```

Expected: FAIL — `rm.NextState undefined`.

- [ ] **Step 3: Add the fields and compute them in `Transform`**

In `transport/rest.go`, add two fields to `RestModel` immediately after `CycleIntervalSeconds`:

```go
	// NextTransitionAt is the absolute instant of the next state change,
	// projected from the schedule's time-of-day boundaries onto the first
	// instant after the server's `now`. Empty when the route is out of
	// service. Clients count down to this rather than reconstructing the
	// scheduler.
	NextTransitionAt string `json:"nextTransitionAt"`
	NextState        string `json:"nextState"`
```

Replace `Transform` (lines 107-125) with a summary/full pair. `Transform` keeps its `func(Model) (RestModel, error)` signature so `model.SliceMap` composition is unchanged:

```go
// TransformSummary converts a Model to a RestModel without its trip schedule.
// The board renders entirely from these attributes; a full day's schedule is
// ~96 rows per route and is fetched only where it is actually read.
//
// State and NextState/NextTransitionAt all come from one Evaluate call on one
// `now`, so a response can never report a state that disagrees with its own
// countdown.
func TransformSummary(m Model) (RestModel, error) {
	transition := m.Evaluate(timeNow().UTC())

	nextAt := ""
	nextState := ""
	if transition.State != OutOfService && !transition.NextAt.IsZero() {
		nextAt = transition.NextAt.Format(time.RFC3339)
		nextState = string(transition.NextState)
	}

	return RestModel{
		ID:                    m.Id(),
		Name:                  m.Name(),
		StartMapID:            m.StartMapId(),
		StagingMapID:          m.StagingMapId(),
		EnRouteMapIDs:         m.EnRouteMapIds(),
		DestinationMapID:      m.DestinationMapId(),
		ObservationMapID:      m.ObservationMapId(),
		State:                 string(transition.State),
		CycleInterval:         m.CycleInterval(),
		BoardingWindowSeconds: uint32(m.BoardingWindowDuration().Seconds()),
		PreDepartureSeconds:   uint32(m.PreDepartureDuration().Seconds()),
		TravelDurationSeconds: uint32(m.TravelDuration().Seconds()),
		CycleIntervalSeconds:  uint32(m.CycleInterval().Seconds()),
		NextTransitionAt:      nextAt,
		NextState:             nextState,
	}, nil
}

// Transform converts a Model to a RestModel with its trip schedule attached.
func Transform(m Model) (RestModel, error) {
	rm, err := TransformSummary(m)
	if err != nil {
		return RestModel{}, err
	}

	schedule, err := model.SliceMap(TransformSchedule)(model.FixedProvider(m.Schedule()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}
	rm.Schedule = schedule

	return rm, nil
}
```

Note `State` now comes from `transition.State` rather than `m.State()`. These agree — the 1-second ticker persists exactly this derivation — and taking it from the same `Evaluate` call is what makes the "cannot diverge" guarantee true rather than merely likely.

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-transports/atlas.com/transports && go test -race ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/transport/rest.go \
        services/atlas-transports/atlas.com/transports/transport/rest_test.go
git commit -m "feat(transports): serve nextTransitionAt and nextState on the route resource"
```

---

## Task 4: `include=schedule` opt-in on the route list

`Transform` attaches the full day's trips to every route; a 15-minute cycle yields 96 trips, so a twelve-route board fetch carries ~1,000 included resources every 30 seconds. Sparse fieldsets cannot suppress them (`api2go@v1.0.4/jsonapi/helpers.go:116-123` rewrites each `included` entry's attributes and never removes an entry; an empty field list 400s). Make the compound document opt-in instead, which is what JSON:API's `include` means.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/transport/resource.go:50-112`
- Test: `services/atlas-transports/atlas.com/transports/transport/resource_include_test.go` (create)

**Interfaces:**
- Consumes: `TransformSummary` / `Transform` from Task 3.
- Produces: the list endpoint's default response carries no `included`; `?include=schedule` restores it. The detail endpoint is unchanged.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-transports/atlas.com/transports/transport/resource_include_test.go`:

```go
package transport

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// TestGetAllRoutesScheduleIsOptIn proves the board's payload guarantee: the
// list endpoint does not ship a day's worth of trip rows unless asked.
func TestGetAllRoutesScheduleIsOptIn(t *testing.T) {
	setupTransportTestRegistry(t)
	tm, ctx := newTestTenantContext(t)

	routeId := uuid.MustParse("00000000-0000-0000-0000-0000000004a0")
	trip, err := NewTripScheduleBuilder().
		SetTripId(uuid.New()).
		SetRouteId(routeId).
		SetBoardingOpen(time.Date(2023, 1, 1, 8, 0, 0, 0, time.UTC)).
		SetBoardingClosed(time.Date(2023, 1, 1, 8, 5, 0, 0, time.UTC)).
		SetDeparture(time.Date(2023, 1, 1, 8, 7, 0, 0, time.UTC)).
		SetArrival(time.Date(2023, 1, 1, 8, 17, 0, 0, time.UTC)).
		Build()
	if err != nil {
		t.Fatalf("seed trip build failed: %v", err)
	}

	m, err := NewBuilder("Include Route").
		SetId(routeId).
		SetStartMapId(_map.Id(100000000)).
		SetStagingMapId(_map.Id(100000001)).
		SetEnRouteMapIds([]_map.Id{_map.Id(100000002)}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		SetSchedule([]TripScheduleModel{trip}).
		Build()
	if err != nil {
		t.Fatalf("seed build failed: %v", err)
	}
	getRouteRegistry().AddTenant(ctx, []Model{m})

	logger, _ := logtest.NewNullLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(router, logger)

	read := func(path string) []json.RawMessage {
		rr := doGetRoutes(t, router, tm.Id(), path)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Included []json.RawMessage `json:"included"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		return doc.Included
	}

	t.Run("DefaultOmitsIncluded", func(t *testing.T) {
		if included := read("/transports/routes"); len(included) != 0 {
			t.Fatalf("len(included) = %d, want 0", len(included))
		}
	})

	t.Run("IncludeScheduleAttachesTrips", func(t *testing.T) {
		if included := read("/transports/routes?include=schedule"); len(included) != 1 {
			t.Fatalf("len(included) = %d, want 1", len(included))
		}
	})

	t.Run("DetailAlwaysIncludesSchedule", func(t *testing.T) {
		rr := doGetRoutes(t, router, tm.Id(), "/transports/routes/"+routeId.String())
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Included []json.RawMessage `json:"included"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Included) != 1 {
			t.Fatalf("len(included) = %d, want 1", len(doc.Included))
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./transport/ -run TestGetAllRoutesScheduleIsOptIn -v
```

Expected: FAIL on `DefaultOmitsIncluded` — `len(included) = 1, want 0`.

- [ ] **Step 3: Make the schedule opt-in**

In `transport/resource.go`, add this helper just above `GetAllRoutesHandler`:

```go
// wantsSchedule reports whether the JSON:API `include` parameter asks for the
// route's trip schedule. The list endpoint attaches a day's worth of trip rows
// (~96 per route on a 15-minute cycle) only on request; consumers that read
// only route attributes - the board, atlas-channel's IsBoatInMap sweep,
// atlas-query-aggregator - get a document a fraction of the size. The detail
// endpoint always attaches the schedule.
func wantsSchedule(include string) bool {
	for _, name := range strings.Split(include, ",") {
		if strings.TrimSpace(name) == "schedule" {
			return true
		}
	}
	return false
}
```

Add `"strings"` to the import block.

Inside `GetAllRoutesHandler`, immediately after `startMapIdFilter := query.Get("filter[startMapId]")`, select the transformer:

```go
		transformer := TransformSummary
		if wantsSchedule(query.Get("include")) {
			transformer = Transform
		}
```

Then replace the two `Transform` call sites in that handler:

- line ~82 (`restModel, transformErr := Transform(route)`) becomes `restModel, transformErr := transformer(route)`
- line ~95 (`model.SliceMap(Transform)(...)`) becomes `model.SliceMap(transformer)(...)`

`GetRouteHandler` keeps calling `Transform` — leave it untouched.

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-transports/atlas.com/transports && go test -race ./... && go vet ./...
```

Expected: PASS, including `TestGetAllRoutesPaginates`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/transport/resource.go \
        services/atlas-transports/atlas.com/transports/transport/resource_include_test.go
git commit -m "perf(transports): make the route list's trip schedule opt-in via include=schedule"
```

---

## Task 5: Fix instance-status tenant scoping

`GetInstanceRouteStatusHandler` passes `uuid.Nil` into `GetInstancesByRoute`, so it reads the nil-tenant Redis set while `FindOrCreateInstance` writes under the real tenant id. For any live tenant the endpoint returns an empty list.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/resource.go:97-127`
- Modify: `services/atlas-transports/atlas.com/transports/instance/resource_paginate_test.go:149-251`
- Test: `services/atlas-transports/atlas.com/transports/instance/resource_tenant_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `GET /transports/instance-routes/{routeId}/status` returns the requesting tenant's instances.

- [ ] **Step 1: Write the failing regression test**

Create `services/atlas-transports/atlas.com/transports/instance/resource_tenant_test.go`:

```go
package instance

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// TestGetInstanceRouteStatusIsTenantScoped is the cross-tenant guard. An
// instance created under tenant A must never surface for tenant B: the
// handler reads the per-route Redis set, which is tenant-keyed, and
// FindOrCreateInstance writes under the creating tenant's id.
func TestGetInstanceRouteStatusIsTenantScoped(t *testing.T) {
	setupInstanceTestRegistry(t)

	route, err := NewRouteBuilder("tenant-scoped-route").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(3).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()
	if err != nil {
		t.Fatalf("seed build failed: %v", err)
	}

	tenantA := uuid.New()
	tenantB := uuid.New()

	reg := getInstanceRegistry()
	inst := reg.FindOrCreateInstance(tenantA, route, time.Now())
	reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: 1})

	logger, _ := logtest.NewNullLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(router, logger)

	path := "/transports/instance-routes/" + route.Id().String() + "/status"

	readIds := func(tenantId uuid.UUID) ([]string, int) {
		rr := doGetInstance(t, router, tenantId, path)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct {
				Id string `json:"id"`
			} `json:"data"`
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		ids := make([]string, 0, len(doc.Data))
		for _, d := range doc.Data {
			ids = append(ids, d.Id)
		}
		return ids, doc.Meta.Total
	}

	t.Run("CreatingTenantSeesTheInstance", func(t *testing.T) {
		ids, total := readIds(tenantA)
		if len(ids) != 1 || ids[0] != inst.InstanceId().String() {
			t.Fatalf("tenant A ids = %v, want [%s]", ids, inst.InstanceId())
		}
		if total != 1 {
			t.Fatalf("tenant A meta.total = %d, want 1", total)
		}
	})

	t.Run("OtherTenantSeesNothing", func(t *testing.T) {
		ids, total := readIds(tenantB)
		if len(ids) != 0 {
			t.Fatalf("tenant B ids = %v, want []", ids)
		}
		if total != 0 {
			t.Fatalf("tenant B meta.total = %d, want 0", total)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run TestGetInstanceRouteStatusIsTenantScoped -v
```

Expected: FAIL on `CreatingTenantSeesTheInstance` — the handler reads the `uuid.Nil` set, so tenant A's request returns an empty list.

- [ ] **Step 3: Derive the tenant from the request context**

In `instance/resource.go`, add the tenant import to the grouped atlas imports:

```go
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
```

In `GetInstanceRouteStatusHandler`, replace lines 106-107:

```go
			ir := getInstanceRegistry()
			instances := ir.GetInstancesByRoute(uuid.Nil, routeId)
```

with:

```go
			// Instances are written under the creating tenant's id
			// (Processor.StartTransport -> FindOrCreateInstance), and the
			// per-route set is tenant-keyed. Read under the same tenant the
			// request carries - rest.RegisterHandler has already installed it
			// in the context, exactly as the sibling handlers' NewProcessor
			// calls rely on.
			t := tenant.MustFromContext(d.Context())

			ir := getInstanceRegistry()
			instances := ir.GetInstancesByRoute(t.Id(), routeId)
```

- [ ] **Step 4: Re-point the paginate test off `uuid.Nil`**

In `instance/resource_paginate_test.go`, replace the stale comment block (lines 149-159) and the seeding loop with a concrete tenant. Replace lines 149-182 with:

```go
// TestGetInstanceRouteStatusPaginates proves
// GET /transports/instance-routes/{routeId}/status is now paginated.
// Instances are seeded under a concrete tenant and read back as that same
// tenant, matching how the handler scopes its read. The route's capacity is 1
// and each instance is filled to capacity before the next
// FindOrCreateInstance call, forcing 3 distinct instances instead of one
// reused instance. Instance ids are server-generated (uuid.New()), so the
// determinism assertion sorts the expected ids the same way the handler does
// rather than asserting fixed literals.
func TestGetInstanceRouteStatusPaginates(t *testing.T) {
	setupInstanceTestRegistry(t)

	route, err := NewRouteBuilder("status-route").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(1).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()
	if err != nil {
		t.Fatalf("seed build failed: %v", err)
	}

	tenantId := uuid.New()

	reg := getInstanceRegistry()
	now := time.Now()
	seededIds := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		inst := reg.FindOrCreateInstance(tenantId, route, now)
		reg.AddCharacter(inst.InstanceId(), CharacterEntry{CharacterId: uint32(i + 1)})
		seededIds = append(seededIds, inst.InstanceId().String())
	}
```

Then replace every `doGetInstance(t, router, uuid.New(), path...)` call inside that test function (three of them, at the original lines 191, 230 and 237) with `doGetInstance(t, router, tenantId, path...)`.

`seededIds` is retained because the surrounding subtests reference it; if the compiler reports it unused after your edit, that means a subtest was dropped — restore it rather than deleting the variable.

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-transports/atlas.com/transports && go test -race ./instance/ -v
```

Expected: PASS, both `TestGetInstanceRouteStatusIsTenantScoped` and `TestGetInstanceRouteStatusPaginates`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/resource.go \
        services/atlas-transports/atlas.com/transports/instance/resource_paginate_test.go \
        services/atlas-transports/atlas.com/transports/instance/resource_tenant_test.go
git commit -m "fix(transports): scope instance status reads to the requesting tenant"
```

---

## Task 6: `…Seconds` on instance routes, `createdAt` on instance status

The UI's stuck-instance flag must compare the same quantity the server force-warps on: `now - createdAt` against `MaxLifetime() = 2 × (boardingWindow + travelDuration)`. `createdAt` is persisted but not exposed, and the two duration inputs are nanosecond-valued.

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/rest.go:13-93`
- Test: `services/atlas-transports/atlas.com/transports/instance/rest_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `RouteRestModel` gains `BoardingWindowSeconds`/`TravelDurationSeconds` (`uint32`, JSON `boardingWindowSeconds`/`travelDurationSeconds`); `InstanceStatusRestModel` gains `CreatedAt string` (JSON `createdAt`, RFC3339).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-transports/atlas.com/transports/instance/rest_test.go`:

```go
package instance

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestTransformRoute_PopulatesSecondsFields(t *testing.T) {
	route, err := NewRouteBuilder("seconds-route").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(5).
		SetBoardingWindow(90 * time.Second).
		SetTravelDuration(3 * time.Minute).
		Build()
	require.NoError(t, err)

	rm, err := TransformRoute(route)
	require.NoError(t, err)

	assert.Equal(t, uint32(90), rm.BoardingWindowSeconds)
	assert.Equal(t, uint32(180), rm.TravelDurationSeconds)
	// Legacy nanosecond fields are untouched.
	assert.Equal(t, 90*time.Second, rm.BoardingWindow)
	assert.Equal(t, 3*time.Minute, rm.TravelDuration)
}

func TestTransformInstanceStatus_ExposesCreatedAt(t *testing.T) {
	boardingUntil := time.Date(2026, 8, 6, 12, 0, 30, 0, time.UTC)
	arrivalAt := boardingUntil.Add(30 * time.Second)
	inst := NewTransportInstance(uuid.New(), uuid.New(), uuid.New(), boardingUntil, arrivalAt)

	rm, err := TransformInstanceStatus(inst)
	require.NoError(t, err)

	assert.Equal(t, inst.CreatedAt().Format(time.RFC3339), rm.CreatedAt)
	assert.NotEmpty(t, rm.CreatedAt)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-transports/atlas.com/transports && go test ./instance/ -run 'TestTransformRoute_PopulatesSecondsFields|TestTransformInstanceStatus_ExposesCreatedAt' -v
```

Expected: FAIL — `rm.BoardingWindowSeconds undefined`.

- [ ] **Step 3: Add the fields**

In `instance/rest.go`, replace the `RouteRestModel` struct (lines 13-22) with:

```go
// RouteRestModel is the JSON:API resource for an instance transport route.
//
// BoardingWindow and TravelDuration are time.Duration and therefore serialise
// as integer nanosecond counts. They are retained unchanged for existing
// consumers; new consumers read the unit-explicit *Seconds fields.
type RouteRestModel struct {
	ID                    uuid.UUID     `json:"-"`
	Name                  string        `json:"name"`
	StartMapId            _map.Id       `json:"startMapId"`
	TransitMapIds         []_map.Id     `json:"transitMapIds"`
	DestinationMapId      _map.Id       `json:"destinationMapId"`
	Capacity              uint32        `json:"capacity"`
	BoardingWindow        time.Duration `json:"boardingWindow"`
	TravelDuration        time.Duration `json:"travelDuration"`
	BoardingWindowSeconds uint32        `json:"boardingWindowSeconds"`
	TravelDurationSeconds uint32        `json:"travelDurationSeconds"`
}
```

In `TransformRoute` (line 42), add to the returned literal:

```go
		BoardingWindow:        m.BoardingWindow(),
		TravelDuration:        m.TravelDuration(),
		BoardingWindowSeconds: uint32(m.BoardingWindow().Seconds()),
		TravelDurationSeconds: uint32(m.TravelDuration().Seconds()),
```

In `InstanceStatusRestModel` (lines 54-61), add:

```go
	// CreatedAt is the instance's creation instant. The stuck-timeout sweep
	// force-warps on now - createdAt exceeding the route's MaxLifetime, so a
	// client that wants to warn ahead of that has to compare the same
	// quantity rather than infer it from boardingUntil.
	CreatedAt string `json:"createdAt"`
```

In `TransformInstanceStatus` (line 85), add:

```go
		CreatedAt:     inst.CreatedAt().Format(time.RFC3339),
```

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-transports/atlas.com/transports && go test -race ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/rest.go \
        services/atlas-transports/atlas.com/transports/instance/rest_test.go
git commit -m "feat(transports): expose second durations and instance createdAt on the instance resources"
```

---

## Task 7: Backend documentation

**Files:**
- Modify: `services/atlas-transports/docs/rest.md`
- Modify: `services/atlas-transports/docs/domain.md`
- Create: `services/atlas-transports/.bruno/Routes/Get Routes With Schedule.bru`

**Interfaces:**
- Consumes: everything from Tasks 1-6. No code depends on this task.

- [ ] **Step 1: Update the `routes` list section of `rest.md`**

In `services/atlas-transports/docs/rest.md`, under `### GET /transports/routes`, add an `include` row to the parameters table:

```markdown
| include | query | string | No | Comma-separated relationship names. `schedule` attaches the day's trip rows; omitted by default (see note) |
```

Replace that section's response-model table with:

```markdown
| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier (tenant-derived, stable across restarts and replicas) |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| stagingMapId | map.Id | Staging map ID |
| enRouteMapIds | []map.Id | En-route map IDs |
| destinationMapId | map.Id | Destination map ID |
| observationMapId | map.Id | Observation map ID |
| state | string | Current route state |
| cycleInterval | time.Duration | **Legacy.** Serialises as an integer nanosecond count. Superseded by `cycleIntervalSeconds` |
| boardingWindowSeconds | uint32 | Boarding window, in seconds |
| preDepartureSeconds | uint32 | Pre-departure hold, in seconds |
| travelDurationSeconds | uint32 | Travel duration, in seconds |
| cycleIntervalSeconds | uint32 | Cycle interval, in seconds |
| nextTransitionAt | string | Absolute instant (RFC3339) of the next state change; empty when `out_of_service` |
| nextState | string | The state the route moves to at `nextTransitionAt`; empty when `out_of_service` |

**Schedule is opt-in.** `Transform` attaches a full day of trip rows (~96 per
route on a 15-minute cycle), so a twelve-route list would carry ~1,000 included
resources. The list endpoint therefore uses a summary transform by default and
attaches the `schedule` relationship only when `include=schedule` is passed.
The detail endpoint always attaches it. Sparse fieldsets cannot express this:
api2go's `FilterSparseFields` rewrites each `included` entry's attributes and
never removes an entry, and an empty field list is a 400.

**Time semantics.** The day's schedule is computed once per reconcile and the
1-second ticker only re-derives state from it, comparing *time of day* only.
Trip-schedule timestamps therefore carry the computing day's date, and only
their time-of-day component is meaningful. `nextTransitionAt` exists so clients
never have to reconstruct that: it is the governing boundary projected onto the
first instant after the server's `now`. `state` and `nextState`/`nextTransitionAt`
come from a single evaluation on a single `now` and cannot disagree.
```

Apply the same field table (minus the schedule-is-opt-in note) to the `### GET /transports/routes/{routeId}` section, and add to it: `The schedule relationship is always attached on this endpoint.`

- [ ] **Step 2: Update the instance-route and instance-status sections of `rest.md`**

In both `### GET /transports/instance-routes` and `### GET /transports/instance-routes/{routeId}`, change the `boardingWindow` / `travelDuration` descriptions to mark them legacy and add the two new rows:

```markdown
| boardingWindow | time.Duration | **Legacy.** Integer nanosecond count. Superseded by `boardingWindowSeconds` |
| travelDuration | time.Duration | **Legacy.** Integer nanosecond count. Superseded by `travelDurationSeconds` |
| boardingWindowSeconds | uint32 | Boarding window, in seconds |
| travelDurationSeconds | uint32 | Travel duration, in seconds |
```

In `### GET /transports/instance-routes/{routeId}/status`, add the `createdAt` row and a scoping note:

```markdown
| createdAt | string | Instance creation instant (RFC3339). The stuck-timeout sweep force-warps when `now - createdAt` exceeds the route's `MaxLifetime()` = `2 × (boardingWindow + travelDuration)` |

**Tenant scoping.** Instances are stored in a per-route, tenant-keyed Redis set
and are read back under the tenant the request carries. A tenant only ever sees
its own instances.
```

- [ ] **Step 3: Document `Evaluate` / `Transition` in `domain.md`**

In `services/atlas-transports/docs/domain.md`, under the `### Model (transport/model.go)` section, append:

```markdown
**Transition (transport/model.go)**

`Model.Evaluate(now)` returns a `Transition` — the state the route is in, the
state it moves to next, and `NextAt`, the absolute instant of that move.
`processStateChange` is a thin wrapper over `Evaluate(now).State`, so the state
machine has exactly one implementation.

| Field | Type | Notes |
|-------|------|-------|
| State | RouteState | The state at `now` |
| NextState | RouteState | The state at `NextAt`; empty when `State` is `out_of_service` |
| NextAt | time.Time | Zero when `State` is `out_of_service` |

Schedule comparisons are time-of-day only (the schedule is computed once per
reconcile and carries that day's date). `NextAt` is the governing time-of-day
boundary projected onto the first instant strictly after `now`, in `now`'s own
date and location, which is what makes it safe to serialise as an absolute
timestamp.
```

- [ ] **Step 4: Add the Bruno request**

Create `services/atlas-transports/.bruno/Routes/Get Routes With Schedule.bru`:

```
meta {
  name: Get Routes With Schedule
  type: http
  seq: 5
}

get {
  url: {{scheme}}://{{host}}:{{port}}/api/transports/routes?include=schedule
  body: none
  auth: inherit
}

params:query {
  include: schedule
}
```

- [ ] **Step 5: Verify the docs describe the code**

```bash
cd services/atlas-transports/atlas.com/transports && grep -n 'json:"' transport/rest.go instance/rest.go
```

Expected: every JSON field name printed appears in `docs/rest.md`. Fix any that do not.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-transports/docs/rest.md \
        services/atlas-transports/docs/domain.md \
        "services/atlas-transports/.bruno/Routes/Get Routes With Schedule.bru"
git commit -m "docs(transports): document seconds durations, next-transition fields, include=schedule and tenant scoping"
```

---

## Task 8: atlas-ui transport types and service layer

**Files:**
- Create: `services/atlas-ui/src/types/models/transport.ts`
- Create: `services/atlas-ui/src/services/api/transports.service.ts`
- Test: `services/atlas-ui/src/services/api/__tests__/transports.service.test.ts` (create)

**Interfaces:**
- Consumes: the wire fields added in Tasks 2, 3 and 6.
- Produces (every later frontend task imports from these two modules):
  - `RouteState`, `InstanceState`, `ScheduledRoute`, `TripSchedule`, `ScheduledRouteDetail`, `InstanceRoute`, `InstanceStatus`, `Vessel` from `@/types/models/transport`
  - `transportsService.getScheduledRoutes()`, `.getScheduledRoute(routeId)`, `.getInstanceRoutes()`, `.getInstanceStatuses(routeId)`, `.getVessels(tenantId)` from `@/services/api/transports.service`

- [ ] **Step 1: Write the types**

Create `services/atlas-ui/src/types/models/transport.ts`:

```ts
/**
 * JSON:API shapes for the atlas-transports surface.
 *
 * Durations are always the server's unit-explicit `…Seconds` integers. The
 * legacy nanosecond-valued fields (`cycleInterval` on routes, `boardingWindow`
 * and `travelDuration` on instance routes) are deliberately NOT declared here,
 * so nothing can read them by accident.
 */

export type RouteState =
  | "out_of_service"
  | "in_transit"
  | "locked_entry"
  | "open_entry"
  | "awaiting_return";

export type InstanceState = "boarding" | "in_transit";

export interface ScheduledRouteAttributes {
  name: string;
  startMapId: number;
  stagingMapId: number;
  enRouteMapIds: number[];
  destinationMapId: number;
  observationMapId: number;
  state: RouteState;
  boardingWindowSeconds: number;
  preDepartureSeconds: number;
  travelDurationSeconds: number;
  cycleIntervalSeconds: number;
  /** Absolute RFC3339 instant of the next state change; "" when out of service. */
  nextTransitionAt: string;
  /** The state reached at nextTransitionAt; "" when out of service. */
  nextState: RouteState | "";
}

export interface ScheduledRoute {
  id: string;
  attributes: ScheduledRouteAttributes;
}

/**
 * Trip boundaries carry the date of the day the schedule was computed; only
 * their time-of-day component is meaningful. Render them through
 * `formatTimeOfDay` and never as a date.
 */
export interface TripScheduleAttributes {
  boardingOpen: string;
  boardingClosed: string;
  departure: string;
  arrival: string;
}

export interface TripSchedule {
  id: string;
  attributes: TripScheduleAttributes;
}

export interface ScheduledRouteDetail {
  route: ScheduledRoute;
  schedule: TripSchedule[];
}

export interface InstanceRouteAttributes {
  name: string;
  startMapId: number;
  transitMapIds: number[];
  destinationMapId: number;
  capacity: number;
  boardingWindowSeconds: number;
  travelDurationSeconds: number;
}

export interface InstanceRoute {
  id: string;
  attributes: InstanceRouteAttributes;
}

export interface InstanceStatusAttributes {
  routeId: string;
  state: InstanceState;
  characters: number;
  boardingUntil: string;
  arrivalAt: string;
  createdAt: string;
}

export interface InstanceStatus {
  id: string;
  attributes: InstanceStatusAttributes;
}

/**
 * Vessels are pure tenant configuration served by atlas-tenants. The resource
 * id is the config slug; `routeAID`/`routeBID` are route **names**, which is
 * what the backend scheduler matches on.
 */
export interface VesselAttributes {
  uuid: string;
  name: string;
  routeAID: string;
  routeBID: string;
  /** Turnaround delay in seconds. */
  turnaroundDelay: number;
}

export interface Vessel {
  id: string;
  attributes: VesselAttributes;
}
```

- [ ] **Step 2: Write the failing service test**

Create `services/atlas-ui/src/services/api/__tests__/transports.service.test.ts`:

```ts
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/lib/api/client";
import { transportsService } from "@/services/api/transports.service";

vi.mock("@/lib/api/client", () => ({
  api: { get: vi.fn() },
}));

const mockedGet = vi.mocked(api.get);

function pagedDocument<T>(data: T[]) {
  return { data, meta: { total: data.length, page: { number: 1, size: 250, last: 1 } } };
}

describe("transportsService", () => {
  beforeEach(() => {
    mockedGet.mockReset();
  });

  it("drains the scheduled route list without asking for the schedule", async () => {
    mockedGet.mockResolvedValueOnce(pagedDocument([{ id: "r1", attributes: {} }]));

    const routes = await transportsService.getScheduledRoutes();

    expect(routes).toHaveLength(1);
    const url = mockedGet.mock.calls[0]?.[0] as string;
    expect(url).toContain("/api/transports/routes");
    expect(url).not.toContain("include=schedule");
  });

  it("asks for the compound document on the detail read and normalises included trips", async () => {
    mockedGet.mockResolvedValueOnce({
      data: { id: "r1", type: "routes", attributes: { name: "Orbis" } },
      included: [
        { id: "t1", type: "trip-schedule", attributes: { boardingOpen: "2023-01-01T08:00:00Z" } },
        { id: "x1", type: "something-else", attributes: {} },
      ],
    });

    const detail = await transportsService.getScheduledRoute("r1");

    expect(mockedGet.mock.calls[0]?.[0]).toBe(
      "/api/transports/routes/r1?include=schedule",
    );
    expect(detail.route.id).toBe("r1");
    expect(detail.schedule).toEqual([
      { id: "t1", attributes: { boardingOpen: "2023-01-01T08:00:00Z" } },
    ]);
  });

  it("returns an empty schedule when the document carries no included array", async () => {
    mockedGet.mockResolvedValueOnce({ data: { id: "r1", attributes: {} } });

    const detail = await transportsService.getScheduledRoute("r1");

    expect(detail.schedule).toEqual([]);
  });

  it("reads instance statuses from the per-route status endpoint", async () => {
    mockedGet.mockResolvedValueOnce(pagedDocument([]));

    await transportsService.getInstanceStatuses("ir1");

    expect(mockedGet.mock.calls[0]?.[0]).toContain(
      "/api/transports/instance-routes/ir1/status",
    );
  });

  it("reads vessels from the tenant configuration endpoint", async () => {
    mockedGet.mockResolvedValueOnce(pagedDocument([]));

    await transportsService.getVessels("tenant-1");

    expect(mockedGet.mock.calls[0]?.[0]).toContain(
      "/api/tenants/tenant-1/configurations/vessels",
    );
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/services/api/__tests__/transports.service.test.ts
```

Expected: FAIL — cannot resolve `@/services/api/transports.service`.

- [ ] **Step 4: Write the service**

Create `services/atlas-ui/src/services/api/transports.service.ts`:

```ts
import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";
import { fetchAll } from "@/services/api/pagination";
import type {
  InstanceRoute,
  InstanceStatus,
  ScheduledRoute,
  ScheduledRouteDetail,
  TripSchedule,
  TripScheduleAttributes,
  Vessel,
} from "@/types/models/transport";

const SCHEDULED_PATH = "/api/transports/routes";
const INSTANCE_PATH = "/api/transports/instance-routes";

/** JSON:API compound document for a single route plus its included trips. */
interface ScheduledRouteDocument {
  data: ScheduledRoute;
  included?: Array<{
    id: string;
    type: string;
    attributes: TripScheduleAttributes;
  }>;
}

/**
 * Read-only adapters for the transports surface.
 *
 * The list read deliberately omits `include=schedule`: the backend attaches a
 * full day of trip rows (~96 per route) only on request, and the board renders
 * entirely from route attributes.
 */
export const transportsService = {
  async getScheduledRoutes(
    options?: ServiceOptions,
  ): Promise<ScheduledRoute[]> {
    return fetchAll<ScheduledRoute>(SCHEDULED_PATH, undefined, options);
  },

  /**
   * One route plus its trip rows. Uses the raw document because the trips
   * arrive in `included`, which `api.getOne`'s data-only projection drops.
   */
  async getScheduledRoute(
    routeId: string,
    options?: ServiceOptions,
  ): Promise<ScheduledRouteDetail> {
    const doc = await api.get<ScheduledRouteDocument>(
      `${SCHEDULED_PATH}/${routeId}?include=schedule`,
      options,
    );
    const schedule: TripSchedule[] = (doc.included ?? [])
      .filter((resource) => resource.type === "trip-schedule")
      .map((resource) => ({
        id: resource.id,
        attributes: resource.attributes,
      }));
    return { route: doc.data, schedule };
  },

  async getInstanceRoutes(options?: ServiceOptions): Promise<InstanceRoute[]> {
    return fetchAll<InstanceRoute>(INSTANCE_PATH, undefined, options);
  },

  async getInstanceStatuses(
    routeId: string,
    options?: ServiceOptions,
  ): Promise<InstanceStatus[]> {
    return fetchAll<InstanceStatus>(
      `${INSTANCE_PATH}/${routeId}/status`,
      undefined,
      options,
    );
  },

  /**
   * Vessels are tenant configuration, not runtime state — atlas-transports
   * hands them to the scheduler and never stores them, so there is no vessel
   * registry to serve from. Same pattern the UI already uses for handlers,
   * writers and MTS config.
   */
  async getVessels(
    tenantId: string,
    options?: ServiceOptions,
  ): Promise<Vessel[]> {
    return fetchAll<Vessel>(
      `/api/tenants/${tenantId}/configurations/vessels`,
      undefined,
      options,
    );
  },
};
```

- [ ] **Step 5: Run the test**

```bash
cd services/atlas-ui && npx vitest run src/services/api/__tests__/transports.service.test.ts
```

Expected: PASS (5 tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/types/models/transport.ts \
        services/atlas-ui/src/services/api/transports.service.ts \
        services/atlas-ui/src/services/api/__tests__/transports.service.test.ts
git commit -m "feat(ui): add transport JSON:API types and read-only service adapters"
```

---

## Task 9: The shared clock and the `Countdown` leaf

One `setInterval` for the whole page. `Countdown` subscribes to it through `useSyncExternalStore`, so a tick re-renders only the countdown cells — never the table, never the page.

**Files:**
- Create: `services/atlas-ui/src/lib/utils/clock.ts`
- Create: `services/atlas-ui/src/components/features/transports/Countdown.tsx`
- Test: `services/atlas-ui/src/components/features/transports/__tests__/Countdown.test.tsx` (create)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `useClock(): number` (epoch ms, 1-second granularity) and `subscribeToClock(listener): () => void`, `getClockSnapshot(): number` from `@/lib/utils/clock`
  - `<Countdown targetAt={string | null} label={string | undefined} />` from `@/components/features/transports/Countdown`
  - `formatCountdown(msRemaining: number): string` — **defined in Task 10's `transport-format.ts`**; Task 9 depends on Task 10 for it. Implement Task 10 first if you are executing out of order.

- [ ] **Step 1: Write the clock store**

Create `services/atlas-ui/src/lib/utils/clock.ts`:

```ts
import { useSyncExternalStore } from "react";

/**
 * A single 1-second clock shared by every countdown on the page.
 *
 * Countdowns must tick between the 30-second polls without dragging a table
 * re-render along with them. One module-level interval plus
 * useSyncExternalStore gives that: only the leaf components that subscribe
 * re-render on a tick. A context provider would re-render every consumer's
 * subtree; a timer per cell would multiply intervals.
 *
 * The interval starts on the first subscriber and is cleared on the last, so
 * a page with no countdowns runs no timer.
 */

const TICK_MS = 1000;

let listeners: Array<() => void> = [];
let intervalId: ReturnType<typeof setInterval> | null = null;
let snapshot = Date.now();

function tick(): void {
  snapshot = Date.now();
  for (const listener of listeners) {
    listener();
  }
}

export function subscribeToClock(listener: () => void): () => void {
  listeners = [...listeners, listener];
  if (intervalId === null) {
    snapshot = Date.now();
    intervalId = setInterval(tick, TICK_MS);
  }
  return () => {
    listeners = listeners.filter((registered) => registered !== listener);
    if (listeners.length === 0 && intervalId !== null) {
      clearInterval(intervalId);
      intervalId = null;
    }
  };
}

/**
 * The cached tick value. It changes only when `tick` runs, which is what
 * useSyncExternalStore requires — returning `Date.now()` here would loop.
 */
export function getClockSnapshot(): number {
  return snapshot;
}

/** Current epoch milliseconds, re-rendering the caller once per second. */
export function useClock(): number {
  return useSyncExternalStore(
    subscribeToClock,
    getClockSnapshot,
    getClockSnapshot,
  );
}
```

- [ ] **Step 2: Write the failing `Countdown` test**

Create `services/atlas-ui/src/components/features/transports/__tests__/Countdown.test.tsx`:

```tsx
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";

import { Countdown } from "@/components/features/transports/Countdown";

describe("Countdown", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-06T12:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders mm:ss and ticks down once per second", () => {
    render(<Countdown targetAt="2026-08-06T12:00:30Z" label="departs in" />);

    expect(screen.getByText("0:30")).toBeInTheDocument();
    expect(screen.getByText("departs in")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(5000);
    });

    expect(screen.getByText("0:25")).toBeInTheDocument();
  });

  it("switches to h:mm:ss past one hour", () => {
    render(<Countdown targetAt="2026-08-06T13:30:05Z" />);
    expect(screen.getByText("1:30:05")).toBeInTheDocument();
  });

  it("clamps at 0:00 and never goes negative", () => {
    render(<Countdown targetAt="2026-08-06T12:00:02Z" />);

    act(() => {
      vi.advanceTimersByTime(10_000);
    });

    expect(screen.getByText("0:00")).toBeInTheDocument();
  });

  it("renders an em dash when there is no target", () => {
    render(<Countdown targetAt={null} label="departs in" />);

    expect(screen.getByText("—")).toBeInTheDocument();
    expect(screen.queryByText("departs in")).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/Countdown.test.tsx
```

Expected: FAIL — cannot resolve `@/components/features/transports/Countdown`.

- [ ] **Step 4: Write `Countdown`**

Create `services/atlas-ui/src/components/features/transports/Countdown.tsx`:

```tsx
import { useClock } from "@/lib/utils/clock";
import { formatCountdown } from "@/components/features/transports/transport-format";

interface CountdownProps {
  /** Absolute RFC3339 instant to count down to; null renders an em dash. */
  targetAt: string | null;
  /** What the transition is, e.g. "departs in". Hidden when there is no target. */
  label?: string;
}

/**
 * A countdown leaf. It subscribes to the shared clock on its own so a tick
 * re-renders this cell and nothing else.
 *
 * Reaching zero is not a state change: the pill next to this component keeps
 * showing the server's state until the next refetch supplies a new one.
 */
export function Countdown({ targetAt, label }: CountdownProps) {
  const now = useClock();

  if (!targetAt) {
    return <span className="text-muted-foreground">—</span>;
  }

  const target = Date.parse(targetAt);
  if (Number.isNaN(target)) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <span className="inline-flex items-baseline gap-1.5 tabular-nums">
      {label ? (
        <span className="text-xs text-muted-foreground">{label}</span>
      ) : null}
      <span>{formatCountdown(target - now)}</span>
    </span>
  );
}
```

- [ ] **Step 5: Run the test**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/Countdown.test.tsx
```

Expected: PASS (4 tests). If `formatCountdown` does not exist yet, complete Task 10 first.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/lib/utils/clock.ts \
        services/atlas-ui/src/components/features/transports/Countdown.tsx \
        services/atlas-ui/src/components/features/transports/__tests__/Countdown.test.tsx
git commit -m "feat(ui): add a shared 1-second clock store and the Countdown leaf"
```

---

## Task 10: `transport-format.ts` — every pure helper

All the arithmetic the surface needs, in one unit-testable module with no React in it: sorting, labels, time-of-day formatting, timeline windowing, the stuck-instance threshold, and vessel resolution.

**Time frame decision.** Trip boundaries are UTC time-of-day (the scheduler anchors on `startOfDay` in UTC). `formatTimeOfDay` renders the **UTC** time-of-day and the timeline positions everything in "milliseconds since UTC midnight", so trips and the NOW marker share one frame and no day-boundary shift can be introduced. Timeline and schedule surfaces label their times `UTC`.

**Files:**
- Create: `services/atlas-ui/src/components/features/transports/transport-format.ts`
- Test: `services/atlas-ui/src/components/features/transports/__tests__/transport-format.test.ts` (create)

**Interfaces:**
- Consumes: types from Task 8.
- Produces (exact names used by Tasks 9, 13, 14, 15, 17, 18):
  - `STATE_SEVERITY: Record<RouteState, number>`
  - `compareRoutesBySeverityThenName(a: ScheduledRoute, b: ScheduledRoute): number`
  - `stateLabel(state: RouteState): string`
  - `transitionLabel(nextState: RouteState | ""): string | null`
  - `formatCountdown(msRemaining: number): string`
  - `formatDurationSeconds(seconds: number): string`
  - `formatTimeOfDay(iso: string): string`
  - `utcTimeOfDayMs(iso: string): number`
  - `nowUtcTimeOfDayMs(now: number): number`
  - `timelineHalfWindowMs(boardingOpenMs: number[]): number`
  - `segmentSpan(startMs: number, endMs: number, nowMs: number, halfWindowMs: number): { left: number; right: number } | null`
  - `isInstanceStuck(createdAt: string, boardingWindowSeconds: number, travelDurationSeconds: number, now: number): boolean`
  - `resolveVesselRoutes(vessel: Vessel, routes: ScheduledRoute[]): { routeA: ScheduledRoute | null; routeB: ScheduledRoute | null; unresolved: boolean }`
  - `findVesselForRoute(route: ScheduledRoute, vessels: Vessel[]): Vessel | null`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/transports/__tests__/transport-format.test.ts`:

```ts
import { describe, expect, it } from "vitest";

import {
  compareRoutesBySeverityThenName,
  findVesselForRoute,
  formatCountdown,
  formatDurationSeconds,
  formatTimeOfDay,
  isInstanceStuck,
  nowUtcTimeOfDayMs,
  resolveVesselRoutes,
  segmentSpan,
  timelineHalfWindowMs,
  transitionLabel,
  utcTimeOfDayMs,
} from "@/components/features/transports/transport-format";
import type {
  RouteState,
  ScheduledRoute,
  Vessel,
} from "@/types/models/transport";

const MINUTE = 60_000;

function route(name: string, state: RouteState): ScheduledRoute {
  return {
    id: `id-${name}`,
    attributes: {
      name,
      startMapId: 1,
      stagingMapId: 2,
      enRouteMapIds: [3],
      destinationMapId: 4,
      observationMapId: 5,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "",
      nextState: "",
    },
  };
}

function vessel(routeAID: string, routeBID: string): Vessel {
  return {
    id: "vessel-slug",
    attributes: {
      uuid: "vessel-uuid",
      name: "Shared Hull",
      routeAID,
      routeBID,
      turnaroundDelay: 60,
    },
  };
}

describe("compareRoutesBySeverityThenName", () => {
  it("sorts out_of_service above every other state", () => {
    const sorted = [
      route("Bravo", "awaiting_return"),
      route("Alpha", "in_transit"),
      route("Zulu", "out_of_service"),
    ].sort(compareRoutesBySeverityThenName);

    expect(sorted.map((r) => r.attributes.name)).toEqual([
      "Zulu",
      "Alpha",
      "Bravo",
    ]);
  });

  it("orders the remaining states by severity then name", () => {
    const sorted = [
      route("D", "awaiting_return"),
      route("C", "open_entry"),
      route("B", "locked_entry"),
      route("A", "in_transit"),
    ].sort(compareRoutesBySeverityThenName);

    expect(sorted.map((r) => r.attributes.name)).toEqual(["A", "B", "C", "D"]);
  });

  it("breaks severity ties on name", () => {
    const sorted = [
      route("Zeta", "open_entry"),
      route("Alpha", "open_entry"),
    ].sort(compareRoutesBySeverityThenName);

    expect(sorted.map((r) => r.attributes.name)).toEqual(["Alpha", "Zeta"]);
  });
});

describe("transitionLabel", () => {
  it("names the transition from the state being moved to", () => {
    expect(transitionLabel("open_entry")).toBe("boards in");
    expect(transitionLabel("locked_entry")).toBe("closes in");
    expect(transitionLabel("in_transit")).toBe("departs in");
    expect(transitionLabel("awaiting_return")).toBe("arrives in");
  });

  it("has no label for a route with no transition", () => {
    expect(transitionLabel("")).toBeNull();
    expect(transitionLabel("out_of_service")).toBeNull();
  });
});

describe("formatCountdown", () => {
  it("renders mm:ss below an hour", () => {
    expect(formatCountdown(30_000)).toBe("0:30");
    expect(formatCountdown(5 * MINUTE + 3000)).toBe("5:03");
  });

  it("renders h:mm:ss at or above an hour", () => {
    expect(formatCountdown(3600_000 + 5 * MINUTE + 4000)).toBe("1:05:04");
  });

  it("clamps at zero and never goes negative", () => {
    expect(formatCountdown(0)).toBe("0:00");
    expect(formatCountdown(-90_000)).toBe("0:00");
  });
});

describe("formatDurationSeconds", () => {
  it("renders minutes and seconds", () => {
    expect(formatDurationSeconds(900)).toBe("15m");
    expect(formatDurationSeconds(90)).toBe("1m 30s");
    expect(formatDurationSeconds(45)).toBe("45s");
    expect(formatDurationSeconds(3900)).toBe("1h 5m");
  });
});

describe("formatTimeOfDay", () => {
  it("renders the UTC time component and never a date", () => {
    // The date here is the schedule's stale computing day - it must not leak.
    expect(formatTimeOfDay("2023-01-01T08:07:00Z")).toBe("08:07");
    expect(formatTimeOfDay("2023-01-01T23:59:00Z")).toBe("23:59");
    expect(formatTimeOfDay("2023-01-01T08:07:00Z")).not.toContain("2023");
  });
});

describe("utcTimeOfDayMs / nowUtcTimeOfDayMs", () => {
  it("measures milliseconds since UTC midnight", () => {
    expect(utcTimeOfDayMs("2023-01-01T00:00:00Z")).toBe(0);
    expect(utcTimeOfDayMs("2023-01-01T01:30:00Z")).toBe(90 * MINUTE);
    expect(nowUtcTimeOfDayMs(Date.parse("2026-08-06T12:15:00Z"))).toBe(
      12 * 60 * MINUTE + 15 * MINUTE,
    );
  });
});

describe("timelineHalfWindowMs", () => {
  it("derives the window from median trip spacing, clamped to 10-30 minutes", () => {
    const boats = [0, 15, 30, 45].map((m) => m * MINUTE);
    // median gap 15m * 1.5 = 22.5m, inside the clamp
    expect(timelineHalfWindowMs(boats)).toBe(22.5 * MINUTE);

    const plane = [0, 6, 12, 18].map((m) => m * MINUTE);
    // 6m * 1.5 = 9m, clamped up to 10m
    expect(timelineHalfWindowMs(plane)).toBe(10 * MINUTE);

    const slow = [0, 60, 120].map((m) => m * MINUTE);
    // 60m * 1.5 = 90m, clamped down to 30m
    expect(timelineHalfWindowMs(slow)).toBe(30 * MINUTE);
  });

  it("falls back to the widest window when spacing cannot be measured", () => {
    expect(timelineHalfWindowMs([])).toBe(30 * MINUTE);
    expect(timelineHalfWindowMs([5 * MINUTE])).toBe(30 * MINUTE);
  });
});

describe("segmentSpan", () => {
  const now = 12 * 60 * MINUTE;
  const half = 30 * MINUTE;

  it("spans a segment fully inside the window", () => {
    const span = segmentSpan(now - 10 * MINUTE, now + 10 * MINUTE, now, half);
    expect(span).not.toBeNull();
    expect(span!.left).toBeCloseTo(1 / 3);
    expect(span!.right).toBeCloseTo(2 / 3);
  });

  it("puts a zero-length segment at now in the centre", () => {
    const span = segmentSpan(now, now, now, half);
    expect(span!.left).toBeCloseTo(0.5);
    expect(span!.right).toBeCloseTo(0.5);
  });

  it("clips a segment that overhangs an edge", () => {
    const span = segmentSpan(now - 60 * MINUTE, now, now, half);
    expect(span!.left).toBeCloseTo(0);
    expect(span!.right).toBeCloseTo(0.5);
  });

  it("returns null for a segment entirely outside the window", () => {
    expect(segmentSpan(now + 40 * MINUTE, now + 50 * MINUTE, now, half)).toBeNull();
  });

  it("wraps a segment across UTC midnight", () => {
    const nearMidnight = 23 * 60 * MINUTE + 55 * MINUTE; // 23:55
    // 00:05 - 00:15 the next day is 10 to 20 minutes after 23:55
    const span = segmentSpan(5 * MINUTE, 15 * MINUTE, nearMidnight, half);
    expect(span).not.toBeNull();
    expect(span!.left).toBeCloseTo((10 * MINUTE + half) / (2 * half));
  });
});

describe("isInstanceStuck", () => {
  // MaxLifetime = 2 * (60 + 120) = 360s; two thirds of that is 240s.
  const created = "2026-08-06T12:00:00Z";

  it("is false below two thirds of MaxLifetime", () => {
    expect(
      isInstanceStuck(created, 60, 120, Date.parse("2026-08-06T12:03:00Z")),
    ).toBe(false);
  });

  it("is true past two thirds of MaxLifetime", () => {
    expect(
      isInstanceStuck(created, 60, 120, Date.parse("2026-08-06T12:05:00Z")),
    ).toBe(true);
  });
});

describe("resolveVesselRoutes", () => {
  const orbis = route("Orbis to Ellinia", "open_entry");
  const ellinia = route("Ellinia to Orbis", "awaiting_return");

  it("resolves both sides by route name", () => {
    const result = resolveVesselRoutes(
      vessel("Orbis to Ellinia", "Ellinia to Orbis"),
      [orbis, ellinia],
    );

    expect(result.routeA?.id).toBe(orbis.id);
    expect(result.routeB?.id).toBe(ellinia.id);
    expect(result.unresolved).toBe(false);
  });

  it("flags a vessel whose reference matches no route", () => {
    const result = resolveVesselRoutes(
      vessel("Orbis to Ellinia", "Typo to Nowhere"),
      [orbis, ellinia],
    );

    expect(result.routeA?.id).toBe(orbis.id);
    expect(result.routeB).toBeNull();
    expect(result.unresolved).toBe(true);
  });

  it("does not match on the vessel slug", () => {
    const result = resolveVesselRoutes(vessel("vessel-slug", "vessel-slug"), [
      orbis,
    ]);

    expect(result.unresolved).toBe(true);
  });
});

describe("findVesselForRoute", () => {
  const orbis = route("Orbis to Ellinia", "open_entry");

  it("finds the vessel a route belongs to by name", () => {
    const found = findVesselForRoute(orbis, [
      vessel("Ellinia to Orbis", "Orbis to Ellinia"),
    ]);
    expect(found?.id).toBe("vessel-slug");
  });

  it("returns null for an independent route", () => {
    expect(findVesselForRoute(orbis, [vessel("A", "B")])).toBeNull();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/transport-format.test.ts
```

Expected: FAIL — cannot resolve `@/components/features/transports/transport-format`.

- [ ] **Step 3: Write the module**

Create `services/atlas-ui/src/components/features/transports/transport-format.ts`:

```ts
import type {
  RouteState,
  ScheduledRoute,
  Vessel,
} from "@/types/models/transport";

const MS_PER_DAY = 86_400_000;
const MIN_HALF_WINDOW_MS = 10 * 60_000;
const MAX_HALF_WINDOW_MS = 30 * 60_000;

/**
 * Board ordering: faults first, then the states closest to a change, so a bad
 * route and an imminent departure both sort to the top. `out_of_service` means
 * the scheduler produced no trips at all for that route today.
 */
export const STATE_SEVERITY: Record<RouteState, number> = {
  out_of_service: 0,
  in_transit: 1,
  locked_entry: 2,
  open_entry: 3,
  awaiting_return: 4,
};

export function compareRoutesBySeverityThenName(
  a: ScheduledRoute,
  b: ScheduledRoute,
): number {
  const severity =
    STATE_SEVERITY[a.attributes.state] - STATE_SEVERITY[b.attributes.state];
  if (severity !== 0) return severity;
  return a.attributes.name.localeCompare(b.attributes.name);
}

export function stateLabel(state: RouteState): string {
  switch (state) {
    case "out_of_service":
      return "Out of service";
    case "in_transit":
      return "In transit";
    case "locked_entry":
      return "Boarding closed";
    case "open_entry":
      return "Boarding";
    case "awaiting_return":
      return "Awaiting return";
  }
}

/**
 * The countdown's caption, derived from the state being moved *to*. A route
 * about to enter open_entry "boards in"; one about to enter in_transit
 * "departs in".
 */
export function transitionLabel(nextState: RouteState | ""): string | null {
  switch (nextState) {
    case "open_entry":
      return "boards in";
    case "locked_entry":
      return "closes in";
    case "in_transit":
      return "departs in";
    case "awaiting_return":
      return "arrives in";
    default:
      return null;
  }
}

/** `mm:ss`, or `h:mm:ss` at an hour or more. Clamps at `0:00`. */
export function formatCountdown(msRemaining: number): string {
  const total = Math.max(0, Math.floor(msRemaining / 1000));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  const pad = (value: number) => String(value).padStart(2, "0");
  if (hours > 0) return `${hours}:${pad(minutes)}:${pad(seconds)}`;
  return `${minutes}:${pad(seconds)}`;
}

/** Human-readable configured duration, e.g. `15m`, `1m 30s`, `1h 5m`. */
export function formatDurationSeconds(seconds: number): string {
  if (seconds <= 0) return "0s";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const rest = seconds % 60;
  if (hours > 0) return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  if (minutes > 0) return rest > 0 ? `${minutes}m ${rest}s` : `${minutes}m`;
  return `${rest}s`;
}

/**
 * Renders a trip boundary's UTC time of day and nothing else.
 *
 * Trip-schedule timestamps carry the date of the day the schedule was computed
 * — a stale date whose only meaningful part is the time. The schedule is
 * anchored on UTC midnight, so the UTC components are the real boarding and
 * departure times; every schedule surface goes through this function and
 * labels its times UTC.
 */
export function formatTimeOfDay(iso: string): string {
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return "—";
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${pad(parsed.getUTCHours())}:${pad(parsed.getUTCMinutes())}`;
}

/** Milliseconds since UTC midnight for a schedule timestamp. */
export function utcTimeOfDayMs(iso: string): number {
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) return 0;
  return (
    parsed.getUTCHours() * 3_600_000 +
    parsed.getUTCMinutes() * 60_000 +
    parsed.getUTCSeconds() * 1000 +
    parsed.getUTCMilliseconds()
  );
}

/** Milliseconds since UTC midnight for an epoch instant. */
export function nowUtcTimeOfDayMs(now: number): number {
  return ((now % MS_PER_DAY) + MS_PER_DAY) % MS_PER_DAY;
}

/**
 * Half-width of the timeline window, derived from the median gap between
 * consecutive boarding-open times rather than from `cycleInterval`: a shared
 * vessel's real spacing is arrival-plus-turnaround, not either route's
 * configured cycle. Clamped to 10-30 minutes so the 6-minute plane does not
 * put ten legs on screen and the 15-minute boat does not show two.
 */
export function timelineHalfWindowMs(boardingOpenMs: number[]): number {
  if (boardingOpenMs.length < 2) return MAX_HALF_WINDOW_MS;

  const sorted = [...boardingOpenMs].sort((a, b) => a - b);
  const gaps: number[] = [];
  for (let i = 1; i < sorted.length; i++) {
    const gap = sorted[i]! - sorted[i - 1]!;
    if (gap > 0) gaps.push(gap);
  }
  if (gaps.length === 0) return MAX_HALF_WINDOW_MS;

  gaps.sort((a, b) => a - b);
  const middle = Math.floor(gaps.length / 2);
  const median =
    gaps.length % 2 === 0
      ? (gaps[middle - 1]! + gaps[middle]!) / 2
      : gaps[middle]!;

  return Math.min(
    MAX_HALF_WINDOW_MS,
    Math.max(MIN_HALF_WINDOW_MS, 1.5 * median),
  );
}

/**
 * Fractional [left, right] extent of a time-of-day segment inside the window
 * centred on `nowMs`, clipped to the window's edges. Null when the segment
 * falls entirely outside.
 *
 * The segment is anchored on whichever ±1-day placement of its start lands
 * closest to now, which is what lets a window spanning UTC midnight draw
 * late-evening and early-morning trips on one strip.
 */
export function segmentSpan(
  startMs: number,
  endMs: number,
  nowMs: number,
  halfWindowMs: number,
): { left: number; right: number } | null {
  const duration =
    endMs >= startMs ? endMs - startMs : endMs + MS_PER_DAY - startMs;

  let startOffset = startMs - nowMs;
  for (const candidate of [startMs - MS_PER_DAY, startMs, startMs + MS_PER_DAY]) {
    const offset = candidate - nowMs;
    if (Math.abs(offset) < Math.abs(startOffset)) {
      startOffset = offset;
    }
  }

  const endOffset = startOffset + duration;
  if (endOffset < -halfWindowMs || startOffset > halfWindowMs) return null;

  const toFraction = (offset: number) =>
    (Math.min(halfWindowMs, Math.max(-halfWindowMs, offset)) + halfWindowMs) /
    (2 * halfWindowMs);

  return { left: toFraction(startOffset), right: toFraction(endOffset) };
}

/**
 * Whether an instance is approaching the stuck-timeout force-warp.
 *
 * The server sweeps on `now - createdAt > MaxLifetime`, where MaxLifetime is
 * `2 × (boardingWindow + travelDuration)`. Warning at two thirds of the same
 * quantity keeps the warning and the action in agreement.
 */
export function isInstanceStuck(
  createdAt: string,
  boardingWindowSeconds: number,
  travelDurationSeconds: number,
  now: number,
): boolean {
  const created = Date.parse(createdAt);
  if (Number.isNaN(created)) return false;
  const maxLifetimeMs =
    2 * (boardingWindowSeconds + travelDurationSeconds) * 1000;
  if (maxLifetimeMs <= 0) return false;
  return now - created > (2 / 3) * maxLifetimeMs;
}

/**
 * Resolves a vessel's two sides against the scheduled-route list **by name**,
 * which is the rule the backend scheduler itself uses. An unresolved side is
 * not cosmetic: the scheduler returns an empty schedule for the vessel, which
 * drives *both* of its routes to out_of_service.
 *
 * Vessel ids are configuration slugs, not names — do not match on `vessel.id`
 * even where the seed data happens to make them equal.
 */
export function resolveVesselRoutes(
  vessel: Vessel,
  routes: ScheduledRoute[],
): {
  routeA: ScheduledRoute | null;
  routeB: ScheduledRoute | null;
  unresolved: boolean;
} {
  const byName = (name: string) =>
    routes.find((route) => route.attributes.name === name) ?? null;

  const routeA = byName(vessel.attributes.routeAID);
  const routeB = byName(vessel.attributes.routeBID);

  return { routeA, routeB, unresolved: routeA === null || routeB === null };
}

/** The shared vessel a route belongs to, or null when it runs independently. */
export function findVesselForRoute(
  route: ScheduledRoute,
  vessels: Vessel[],
): Vessel | null {
  return (
    vessels.find(
      (vessel) =>
        vessel.attributes.routeAID === route.attributes.name ||
        vessel.attributes.routeBID === route.attributes.name,
    ) ?? null
  );
}
```

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/
```

Expected: PASS — `transport-format.test.ts` plus `Countdown.test.tsx` from Task 9.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/transports/transport-format.ts \
        services/atlas-ui/src/components/features/transports/__tests__/transport-format.test.ts
git commit -m "feat(ui): add pure transport formatting, sorting, windowing and vessel-resolution helpers"
```

---

## Task 11: `useTransports` query hooks

**Files:**
- Create: `services/atlas-ui/src/lib/hooks/api/useTransports.ts`
- Modify: `services/atlas-ui/src/lib/hooks/api/index.ts`
- Test: `services/atlas-ui/src/lib/hooks/api/__tests__/useTransports.test.tsx` (create)

**Interfaces:**
- Consumes: `transportsService` from Task 8.
- Produces:
  - `TRANSPORT_POLL_MS = 30_000`
  - `transportKeys.{all, scheduled(tenantId), scheduledDetail(tenantId, routeId), instanceRoutes(tenantId), instanceStatus(tenantId, routeId), vessels(tenantId)}`
  - `useScheduledRoutes()`, `useScheduledRoute(routeId)`, `useInstanceRoutes()`, `useInstanceStatuses(routeIds: string[])`, `useVessels()`
  - `useInstanceStatuses` returns `UseQueryResult<InstanceStatus[], Error>[]`, index-aligned with the `routeIds` argument.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/lib/hooks/api/__tests__/useTransports.test.tsx`:

```tsx
import { vi, beforeEach, describe, expect, it } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import type { Tenant } from "@/types/models/tenant";
import {
  transportKeys,
  useInstanceStatuses,
  useScheduledRoute,
  useScheduledRoutes,
  useVessels,
  TRANSPORT_POLL_MS,
} from "../useTransports";
import { transportsService } from "@/services/api/transports.service";

const mockTenant: Tenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

let activeTenant: Tenant | null = mockTenant;

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
}));

vi.mock("@/services/api/transports.service", () => ({
  transportsService: {
    getScheduledRoutes: vi.fn(),
    getScheduledRoute: vi.fn(),
    getInstanceRoutes: vi.fn(),
    getInstanceStatuses: vi.fn(),
    getVessels: vi.fn(),
  },
}));

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("useTransports", () => {
  beforeEach(() => {
    activeTenant = mockTenant;
    vi.mocked(transportsService.getScheduledRoutes).mockReset();
    vi.mocked(transportsService.getScheduledRoute).mockReset();
    vi.mocked(transportsService.getInstanceStatuses).mockReset();
    vi.mocked(transportsService.getVessels).mockReset();
  });

  it("polls the scheduled routes every 30 seconds", () => {
    expect(TRANSPORT_POLL_MS).toBe(30_000);
  });

  it("scopes every key to the active tenant", () => {
    expect(transportKeys.scheduled("tenant-1")).toContain("tenant-1");
    expect(transportKeys.scheduledDetail("tenant-1", "r1")).toContain("r1");
    expect(transportKeys.instanceStatus("tenant-1", "ir1")).toContain("ir1");
    expect(transportKeys.vessels("tenant-1")).toContain("tenant-1");
  });

  it("fetches scheduled routes when a tenant is active", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    const { result } = renderHook(() => useScheduledRoutes(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(transportsService.getScheduledRoutes).toHaveBeenCalled();
  });

  it("does not fetch without an active tenant", () => {
    activeTenant = null;
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    const { result } = renderHook(() => useScheduledRoutes(), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
    expect(transportsService.getScheduledRoutes).not.toHaveBeenCalled();
  });

  it("does not fetch a route detail without a routeId", () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: { id: "r1", attributes: {} as never },
      schedule: [],
    });

    const { result } = renderHook(() => useScheduledRoute(""), { wrapper });

    expect(result.current.fetchStatus).toBe("idle");
  });

  it("fans out one status query per instance route", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);

    const { result } = renderHook(
      () => useInstanceStatuses(["ir1", "ir2", "ir3"]),
      { wrapper },
    );

    await waitFor(() => expect(result.current).toHaveLength(3));
    await waitFor(() =>
      expect(transportsService.getInstanceStatuses).toHaveBeenCalledTimes(3),
    );
    expect(transportsService.getInstanceStatuses).toHaveBeenCalledWith("ir2");
  });

  it("reads vessels from the active tenant's configuration", async () => {
    vi.mocked(transportsService.getVessels).mockResolvedValue([]);

    const { result } = renderHook(() => useVessels(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(transportsService.getVessels).toHaveBeenCalledWith("tenant-1");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/lib/hooks/api/__tests__/useTransports.test.tsx
```

Expected: FAIL — cannot resolve `../useTransports`.

- [ ] **Step 3: Write the hooks**

Create `services/atlas-ui/src/lib/hooks/api/useTransports.ts`:

```ts
/**
 * React Query hooks for the transports surface.
 *
 * Every query is gated on an active tenant and polls on a 30-second interval.
 * Countdowns tick locally off the shared clock store between polls, so this is
 * the only network cadence on the page. Tenant switching is already handled by
 * TenantProvider's cache clear — there is no transport-specific invalidation.
 */

import {
  keepPreviousData,
  useQueries,
  useQuery,
  type UseQueryResult,
} from "@tanstack/react-query";

import { useTenant } from "@/context/tenant-context";
import { transportsService } from "@/services/api/transports.service";
import type {
  InstanceRoute,
  InstanceStatus,
  ScheduledRoute,
  ScheduledRouteDetail,
  Vessel,
} from "@/types/models/transport";

export const TRANSPORT_POLL_MS = 30_000;

export const transportKeys = {
  all: ["transports"] as const,
  scheduled: (tenantId: string) =>
    [...transportKeys.all, "scheduled", tenantId] as const,
  scheduledDetail: (tenantId: string, routeId: string) =>
    [...transportKeys.all, "scheduled", tenantId, "detail", routeId] as const,
  instanceRoutes: (tenantId: string) =>
    [...transportKeys.all, "instance-routes", tenantId] as const,
  instanceStatus: (tenantId: string, routeId: string) =>
    [...transportKeys.all, "instance-status", tenantId, routeId] as const,
  vessels: (tenantId: string) =>
    [...transportKeys.all, "vessels", tenantId] as const,
};

const pollDefaults = {
  refetchInterval: TRANSPORT_POLL_MS,
  refetchIntervalInBackground: false,
  placeholderData: keepPreviousData,
} as const;

export function useScheduledRoutes(): UseQueryResult<ScheduledRoute[], Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.scheduled(tenantId),
    queryFn: () => transportsService.getScheduledRoutes(),
    enabled: !!activeTenant,
    ...pollDefaults,
  });
}

export function useScheduledRoute(
  routeId: string,
): UseQueryResult<ScheduledRouteDetail, Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.scheduledDetail(tenantId, routeId),
    queryFn: () => transportsService.getScheduledRoute(routeId),
    enabled: !!activeTenant && !!routeId,
    ...pollDefaults,
  });
}

export function useInstanceRoutes(): UseQueryResult<InstanceRoute[], Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.instanceRoutes(tenantId),
    queryFn: () => transportsService.getInstanceRoutes(),
    enabled: !!activeTenant,
    ...pollDefaults,
  });
}

/**
 * One status query per instance route, index-aligned with `routeIds`.
 *
 * The Instance tab needs a live count for every route, including collapsed
 * rows, so fetching only for expanded rows will not do. Twelve small,
 * usually-empty responses per 30s is 0.4 rps — cheaper than adding an
 * aggregate endpoint and a Redis scan for a twelve-item list.
 */
export function useInstanceStatuses(
  routeIds: string[],
): UseQueryResult<InstanceStatus[], Error>[] {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQueries({
    queries: routeIds.map((routeId) => ({
      queryKey: transportKeys.instanceStatus(tenantId, routeId),
      queryFn: () => transportsService.getInstanceStatuses(routeId),
      enabled: !!activeTenant,
      ...pollDefaults,
    })),
  });
}

export function useVessels(): UseQueryResult<Vessel[], Error> {
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.id ?? "no-tenant";

  return useQuery({
    queryKey: transportKeys.vessels(tenantId),
    queryFn: () => transportsService.getVessels(tenantId),
    enabled: !!activeTenant,
    ...pollDefaults,
  });
}
```

- [ ] **Step 4: Re-export from the hooks barrel**

Open `services/atlas-ui/src/lib/hooks/api/index.ts` and add an export line matching the file's existing style, e.g.:

```ts
export * from "./useTransports";
```

Place it in the alphabetical position the file already uses. If the barrel exports named symbols explicitly rather than star-exporting, follow that form instead and export `transportKeys`, `TRANSPORT_POLL_MS`, `useScheduledRoutes`, `useScheduledRoute`, `useInstanceRoutes`, `useInstanceStatuses`, `useVessels`.

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-ui && npx vitest run src/lib/hooks/api/__tests__/useTransports.test.tsx
```

Expected: PASS (7 tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/useTransports.ts \
        services/atlas-ui/src/lib/hooks/api/index.ts \
        services/atlas-ui/src/lib/hooks/api/__tests__/useTransports.test.tsx
git commit -m "feat(ui): add tenant-gated transport query hooks with a 30s poll"
```

---

## Task 12: `RouteStatePill` and `FreshnessIndicator`

**Files:**
- Create: `services/atlas-ui/src/components/features/transports/RouteStatePill.tsx`
- Create: `services/atlas-ui/src/components/features/transports/FreshnessIndicator.tsx`
- Test: `services/atlas-ui/src/components/features/transports/__tests__/RouteStatePill.test.tsx` (create)

**Interfaces:**
- Consumes: `stateLabel` from Task 10, `useClock` from Task 9.
- Produces:
  - `<RouteStatePill state={RouteState} />`
  - `<FreshnessIndicator dataUpdatedAt={number} isFetching={boolean} isError={boolean} />`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/transports/__tests__/RouteStatePill.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import type { RouteState } from "@/types/models/transport";

describe("RouteStatePill", () => {
  it.each<[RouteState, string]>([
    ["out_of_service", "Out of service"],
    ["in_transit", "In transit"],
    ["locked_entry", "Boarding closed"],
    ["open_entry", "Boarding"],
    ["awaiting_return", "Awaiting return"],
  ])("labels %s in text, never colour alone", (state, label) => {
    render(<RouteStatePill state={state} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("gives out_of_service a fault treatment distinct from the rest", () => {
    const { container: fault } = render(
      <RouteStatePill state="out_of_service" />,
    );
    const { container: normal } = render(<RouteStatePill state="open_entry" />);

    expect(fault.firstElementChild?.className).not.toBe(
      normal.firstElementChild?.className,
    );
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/RouteStatePill.test.tsx
```

Expected: FAIL — cannot resolve `@/components/features/transports/RouteStatePill`.

- [ ] **Step 3: Write both components**

Create `services/atlas-ui/src/components/features/transports/RouteStatePill.tsx`:

```tsx
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { stateLabel } from "@/components/features/transports/transport-format";
import type { RouteState } from "@/types/models/transport";

/**
 * State is never encoded by colour alone — the pill always carries its text
 * label. `out_of_service` gets the destructive treatment because it means the
 * scheduler produced no trips for the route today, which is a fault, not a
 * quiet status.
 */
const STATE_VARIANT: Record<
  RouteState,
  { variant: "default" | "secondary" | "outline" | "destructive"; className?: string }
> = {
  out_of_service: { variant: "destructive" },
  in_transit: { variant: "default" },
  locked_entry: { variant: "secondary" },
  open_entry: { variant: "outline", className: "border-primary text-primary" },
  awaiting_return: { variant: "outline" },
};

export function RouteStatePill({ state }: { state: RouteState }) {
  const { variant, className } = STATE_VARIANT[state];
  return (
    <Badge variant={variant} className={cn("whitespace-nowrap", className)}>
      {stateLabel(state)}
    </Badge>
  );
}
```

Create `services/atlas-ui/src/components/features/transports/FreshnessIndicator.tsx`:

```tsx
import { AlertTriangle } from "lucide-react";

import { cn } from "@/lib/utils";
import { useClock } from "@/lib/utils/clock";

interface FreshnessIndicatorProps {
  /** React Query's dataUpdatedAt for the page's primary query. */
  dataUpdatedAt: number;
  isFetching: boolean;
  isError: boolean;
}

/**
 * Says how fresh the board is. The age ticks off the same shared clock as the
 * countdowns, so this adds no timer of its own.
 */
export function FreshnessIndicator({
  dataUpdatedAt,
  isFetching,
  isError,
}: FreshnessIndicatorProps) {
  const now = useClock();

  if (isError) {
    return (
      <span className="inline-flex items-center gap-1.5 text-sm text-destructive">
        <AlertTriangle className="h-4 w-4" aria-hidden="true" />
        Stale — last refresh failed
      </span>
    );
  }

  if (!dataUpdatedAt) {
    return <span className="text-sm text-muted-foreground">Loading…</span>;
  }

  const ageSeconds = Math.max(0, Math.floor((now - dataUpdatedAt) / 1000));

  return (
    <span className="inline-flex items-center gap-1.5 text-sm text-muted-foreground">
      <span
        className={cn(
          "h-2 w-2 rounded-full bg-emerald-500",
          isFetching && "animate-pulse",
        )}
        aria-hidden="true"
      />
      <span>Updated {ageSeconds}s ago</span>
    </span>
  );
}
```

- [ ] **Step 4: Run the test**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/RouteStatePill.test.tsx
```

Expected: PASS (6 assertions across 2 tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/transports/RouteStatePill.tsx \
        services/atlas-ui/src/components/features/transports/FreshnessIndicator.tsx \
        services/atlas-ui/src/components/features/transports/__tests__/RouteStatePill.test.tsx
git commit -m "feat(ui): add the route state pill and board freshness indicator"
```

---

## Task 13: The Transports board — Scheduled tab, routing, sidebar

**Files:**
- Create: `services/atlas-ui/src/pages/transports-columns.tsx`
- Create: `services/atlas-ui/src/pages/TransportsPage.tsx`
- Modify: `services/atlas-ui/src/App.tsx`
- Modify: `services/atlas-ui/src/components/app-sidebar-items.ts`
- Test: `services/atlas-ui/src/pages/__tests__/TransportsPage.test.tsx` (create)

**Interfaces:**
- Consumes: Tasks 8-12.
- Produces:
  - `createScheduledRouteColumns({ tenant, vessels }): ColumnDef<ScheduledRoute>[]` from `@/pages/transports-columns`
  - `TransportsPage` (named export) from `@/pages/TransportsPage`
  - Route `/transports`; sidebar entry "Transports" under Operations.

- [ ] **Step 1: Write the failing page test**

Create `services/atlas-ui/src/pages/__tests__/TransportsPage.test.tsx`:

```tsx
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { TransportsPage } from "@/pages/TransportsPage";
import type { Tenant } from "@/types/models/tenant";
import type {
  ScheduledRoute,
  RouteState,
} from "@/types/models/transport";
import { transportsService } from "@/services/api/transports.service";

const mockTenant: Tenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: mockTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
}));

vi.mock("@/components/map-cell", () => ({
  MapCell: ({ mapId }: { mapId: string }) => <span>map:{mapId}</span>,
}));

vi.mock("@/services/api/transports.service", () => ({
  transportsService: {
    getScheduledRoutes: vi.fn(),
    getScheduledRoute: vi.fn(),
    getInstanceRoutes: vi.fn(),
    getInstanceStatuses: vi.fn(),
    getVessels: vi.fn(),
  },
}));

function scheduledRoute(
  name: string,
  state: RouteState,
  overrides: Partial<ScheduledRoute["attributes"]> = {},
): ScheduledRoute {
  return {
    id: `route-${name}`,
    attributes: {
      name,
      startMapId: 104000000,
      stagingMapId: 104000001,
      enRouteMapIds: [104000002],
      destinationMapId: 200000100,
      observationMapId: 104000003,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "",
      nextState: "",
      ...overrides,
    },
  };
}

function renderPage(initialEntry = "/transports") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <QueryClientProvider client={client}>
        <TransportsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("TransportsPage", () => {
  beforeEach(() => {
    vi.mocked(transportsService.getScheduledRoutes).mockReset();
    vi.mocked(transportsService.getInstanceRoutes).mockReset();
    vi.mocked(transportsService.getInstanceStatuses).mockReset();
    vi.mocked(transportsService.getVessels).mockReset();
    vi.mocked(transportsService.getInstanceRoutes).mockResolvedValue([]);
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([]);
  });

  it("lists scheduled routes with a state pill and map cells", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Orbis to Ellinia", "open_entry"),
    ]);

    renderPage();

    expect(await screen.findByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.getByText("Boarding")).toBeInTheDocument();
    expect(screen.getByText("map:104000000")).toBeInTheDocument();
    expect(screen.getByText("map:200000100")).toBeInTheDocument();
  });

  it("sorts out_of_service above every other state", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Aaa Working", "open_entry"),
      scheduledRoute("Zzz Broken", "out_of_service"),
    ]);

    renderPage();

    await screen.findByText("Zzz Broken");
    const names = screen
      .getAllByRole("row")
      .map((row) => row.textContent ?? "")
      .filter((text) => text.includes("Broken") || text.includes("Working"));

    expect(names[0]).toContain("Zzz Broken");
    expect(names[1]).toContain("Aaa Working");
  });

  it("shows an em dash for an out_of_service route's next change", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Broken", "out_of_service"),
    ]);

    renderPage();

    await screen.findByText("Broken");
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
  });

  it("links a route name to its detail page", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("Orbis to Ellinia", "open_entry"),
    ]);

    renderPage();

    const link = await screen.findByRole("link", { name: "Orbis to Ellinia" });
    expect(link).toHaveAttribute(
      "href",
      "/transports/routes/route-Orbis to Ellinia",
    );
  });

  it("defaults to the Scheduled tab and reflects a selection in the URL", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    renderPage();

    const scheduledTab = await screen.findByRole("tab", { name: /Scheduled/ });
    expect(scheduledTab).toHaveAttribute("aria-selected", "true");
  });

  it("honours ?tab= on load", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);

    renderPage("/transports?tab=instance");

    await waitFor(() =>
      expect(screen.getByRole("tab", { name: /Instance/ })).toHaveAttribute(
        "aria-selected",
        "true",
      ),
    );
  });

  it("carries a count on each tab label", async () => {
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("A", "open_entry"),
      scheduledRoute("B", "open_entry"),
    ]);

    renderPage();

    expect(
      await screen.findByRole("tab", { name: /Scheduled\s*2/ }),
    ).toBeInTheDocument();
  });
});
```

Note: the `?tab=instance` assertion passes from this task because the tab trigger exists; its pane is filled in by Task 14.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/pages/__tests__/TransportsPage.test.tsx
```

Expected: FAIL — cannot resolve `@/pages/TransportsPage`.

- [ ] **Step 3: Write the Scheduled columns**

Create `services/atlas-ui/src/pages/transports-columns.tsx`:

```tsx
import { type ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router-dom";

import { MapCell } from "@/components/map-cell";
import { Countdown } from "@/components/features/transports/Countdown";
import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import {
  findVesselForRoute,
  formatDurationSeconds,
  transitionLabel,
} from "@/components/features/transports/transport-format";
import type { Tenant } from "@/types/models/tenant";
import type { ScheduledRoute, Vessel } from "@/types/models/transport";

interface ScheduledRouteColumnDeps {
  tenant: Tenant | null;
  vessels: Vessel[];
}

export function createScheduledRouteColumns({
  tenant,
  vessels,
}: ScheduledRouteColumnDeps): ColumnDef<ScheduledRoute>[] {
  return [
    {
      id: "name",
      header: "Route",
      cell: ({ row }) => (
        <Link
          to={`/transports/routes/${row.original.id}`}
          className="font-medium hover:underline"
        >
          {row.original.attributes.name}
        </Link>
      ),
    },
    {
      id: "state",
      header: "State",
      cell: ({ row }) => <RouteStatePill state={row.original.attributes.state} />,
    },
    {
      id: "nextChange",
      header: "Next change",
      cell: ({ row }) => {
        const { nextState, nextTransitionAt } = row.original.attributes;
        const label = transitionLabel(nextState);
        if (!label || !nextTransitionAt) {
          return <span className="text-muted-foreground">—</span>;
        }
        return <Countdown targetAt={nextTransitionAt} label={label} />;
      },
    },
    {
      id: "startMap",
      header: "Start map",
      cell: ({ row }) => (
        <MapCell
          mapId={String(row.original.attributes.startMapId)}
          tenant={tenant}
        />
      ),
    },
    {
      id: "destinationMap",
      header: "Destination map",
      cell: ({ row }) => (
        <MapCell
          mapId={String(row.original.attributes.destinationMapId)}
          tenant={tenant}
        />
      ),
    },
    {
      id: "vessel",
      header: "Vessel",
      cell: ({ row }) => {
        const vessel = findVesselForRoute(row.original, vessels);
        if (!vessel) return <span className="text-muted-foreground">—</span>;
        return (
          <Link
            to={`/transports?tab=vessels#vessel-${vessel.id}`}
            className="hover:underline"
          >
            {vessel.attributes.name}
          </Link>
        );
      },
    },
    {
      id: "cycleInterval",
      header: "Cycle",
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatDurationSeconds(row.original.attributes.cycleIntervalSeconds)}
        </span>
      ),
    },
  ];
}
```

- [ ] **Step 4: Write the board page**

Create `services/atlas-ui/src/pages/TransportsPage.tsx`:

```tsx
import { Suspense, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { Ship } from "lucide-react";

import { DataTable } from "@/components/data-table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTenant } from "@/context/tenant-context";
import {
  useScheduledRoutes,
  useVessels,
} from "@/lib/hooks/api/useTransports";
import { FreshnessIndicator } from "@/components/features/transports/FreshnessIndicator";
import { compareRoutesBySeverityThenName } from "@/components/features/transports/transport-format";
import { createScheduledRouteColumns } from "@/pages/transports-columns";

const TABS = ["scheduled", "instance", "vessels"] as const;
type TransportTab = (typeof TABS)[number];

function isTransportTab(value: string | null): value is TransportTab {
  return value !== null && (TABS as readonly string[]).includes(value);
}

export function TransportsPage() {
  return (
    <Suspense>
      <TransportsPageContent />
    </Suspense>
  );
}

function TransportsPageContent() {
  const { activeTenant } = useTenant();
  const [searchParams, setSearchParams] = useSearchParams();

  const requestedTab = searchParams.get("tab");
  const activeTab: TransportTab = isTransportTab(requestedTab)
    ? requestedTab
    : "scheduled";

  const scheduledQuery = useScheduledRoutes();
  const vesselsQuery = useVessels();

  const routes = useMemo(
    () => [...(scheduledQuery.data ?? [])].sort(compareRoutesBySeverityThenName),
    [scheduledQuery.data],
  );
  const vessels = useMemo(() => vesselsQuery.data ?? [], [vesselsQuery.data]);

  const scheduledColumns = useMemo(
    () => createScheduledRouteColumns({ tenant: activeTenant, vessels }),
    [activeTenant, vessels],
  );

  const handleTabChange = (value: string) => {
    const next = new URLSearchParams(searchParams);
    if (value === "scheduled") {
      next.delete("tab");
    } else {
      next.set("tab", value);
    }
    setSearchParams(next, { replace: true });
  };

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <Ship className="h-6 w-6" />
          <h2 className="text-2xl font-bold tracking-tight">Transports</h2>
        </div>
        <FreshnessIndicator
          dataUpdatedAt={scheduledQuery.dataUpdatedAt}
          isFetching={scheduledQuery.isFetching}
          isError={scheduledQuery.isError}
        />
      </div>

      <Tabs
        value={activeTab}
        onValueChange={handleTabChange}
        className="flex-1 min-h-0 flex flex-col"
      >
        <TabsList>
          <TabsTrigger value="scheduled">Scheduled {routes.length}</TabsTrigger>
          <TabsTrigger value="instance">Instance</TabsTrigger>
          <TabsTrigger value="vessels">Vessels {vessels.length}</TabsTrigger>
        </TabsList>

        <TabsContent value="scheduled" className="flex-1 min-h-0 overflow-x-auto">
          <DataTable
            columns={scheduledColumns}
            data={routes}
            onRefresh={() => void scheduledQuery.refetch()}
            isRefreshing={scheduledQuery.isFetching}
          />
        </TabsContent>

        <TabsContent value="instance" className="flex-1 min-h-0 overflow-x-auto" />

        <TabsContent value="vessels" className="flex-1 min-h-0 overflow-x-auto" />
      </Tabs>
    </div>
  );
}
```

The two empty `TabsContent` panes are filled in by Tasks 14 and 15 — do not ship this task without completing them.

- [ ] **Step 5: Wire the route and the sidebar entry**

In `services/atlas-ui/src/App.tsx`, add the lazy import alongside the others (alphabetical position, matching the file's existing style):

```tsx
const TransportsPage = lazyWithReload(() =>
  import("@/pages/TransportsPage").then((m) => ({ default: m.TransportsPage })),
);
```

And add the route next to the other Operations routes (near `/maps`):

```tsx
                    <Route path="/transports" element={<TransportsPage />} />
```

In `services/atlas-ui/src/components/app-sidebar-items.ts`, append one entry to the Operations group's `children` array (after `{ title: "Reward Pools", url: "/reward-pools" }`):

```ts
      { title: "Transports", url: "/transports" },
```

- [ ] **Step 6: Run the tests and the type-checking build**

```bash
cd services/atlas-ui && npx vitest run src/pages/__tests__/TransportsPage.test.tsx && npm run build
```

Expected: PASS. The `?tab=instance` test passes because the trigger exists; its pane fills in next. `npm run build` is what type-checks — vitest alone does not.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/pages/TransportsPage.tsx \
        services/atlas-ui/src/pages/transports-columns.tsx \
        services/atlas-ui/src/pages/__tests__/TransportsPage.test.tsx \
        services/atlas-ui/src/App.tsx \
        services/atlas-ui/src/components/app-sidebar-items.ts
git commit -m "feat(ui): add the Transports board with the Scheduled tab, route and sidebar entry"
```

---

## Task 14: Instance tab

**Files:**
- Create: `services/atlas-ui/src/components/features/transports/InstanceRoutesTable.tsx`
- Modify: `services/atlas-ui/src/pages/TransportsPage.tsx`
- Test: `services/atlas-ui/src/components/features/transports/__tests__/InstanceRoutesTable.test.tsx` (create)

**Interfaces:**
- Consumes: `useInstanceRoutes`, `useInstanceStatuses` (Task 11); `isInstanceStuck`, `formatDurationSeconds` (Task 10); `Countdown` (Task 9).
- Produces: `<InstanceRoutesTable tenant={Tenant | null} />` — self-contained, owns its own queries. Exposes `instanceRouteCount` via the `onCountChange` callback so the tab label can carry a count.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/transports/__tests__/InstanceRoutesTable.test.tsx`:

```tsx
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { InstanceRoutesTable } from "@/components/features/transports/InstanceRoutesTable";
import { transportsService } from "@/services/api/transports.service";
import type { Tenant } from "@/types/models/tenant";
import type { InstanceRoute, InstanceStatus } from "@/types/models/transport";

const mockTenant: Tenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: mockTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
}));

vi.mock("@/components/map-cell", () => ({
  MapCell: ({ mapId }: { mapId: string }) => <span>map:{mapId}</span>,
}));

vi.mock("@/services/api/transports.service", () => ({
  transportsService: {
    getInstanceRoutes: vi.fn(),
    getInstanceStatuses: vi.fn(),
  },
}));

const route: InstanceRoute = {
  id: "ir1",
  attributes: {
    name: "Ereve Sky Ferry",
    startMapId: 130000000,
    transitMapIds: [130000010],
    destinationMapId: 130000200,
    capacity: 10,
    boardingWindowSeconds: 60,
    travelDurationSeconds: 120,
  },
};

function status(overrides: Partial<InstanceStatus["attributes"]> = {}): InstanceStatus {
  return {
    id: "11111111-2222-3333-4444-555555555555",
    attributes: {
      routeId: "ir1",
      state: "boarding",
      characters: 3,
      boardingUntil: "2026-08-06T12:01:00Z",
      arrivalAt: "2026-08-06T12:03:00Z",
      createdAt: "2026-08-06T12:00:00Z",
      ...overrides,
    },
  };
}

function renderTable() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <InstanceRoutesTable tenant={mockTenant} />
    </QueryClientProvider>,
  );
}

describe("InstanceRoutesTable", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(new Date("2026-08-06T12:00:30Z"));
    vi.mocked(transportsService.getInstanceRoutes).mockReset();
    vi.mocked(transportsService.getInstanceStatuses).mockReset();
    vi.mocked(transportsService.getInstanceRoutes).mockResolvedValue([route]);
  });

  it("lists every instance route with its capacity and durations", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);

    renderTable();

    expect(await screen.findByText("Ereve Sky Ferry")).toBeInTheDocument();
    expect(screen.getByText("10")).toBeInTheDocument();
    expect(screen.getByText("map:130000000")).toBeInTheDocument();
  });

  it("renders zero live instances as a plain 0 with no expander", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([]);

    renderTable();

    await screen.findByText("Ereve Sky Ferry");
    expect(screen.queryByRole("button", { name: /expand/i })).toBeNull();
  });

  it("expands a route with live instances to per-instance rows", async () => {
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([status()]);

    renderTable();

    const expander = await screen.findByRole("button", { name: /expand/i });
    await userEvent.click(expander);

    // truncated instance id
    expect(screen.getByText(/11111111/)).toBeInTheDocument();
    // character count
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("flags an instance past two thirds of its route's MaxLifetime", async () => {
    // MaxLifetime = 2 * (60 + 120) = 360s; two thirds = 240s.
    vi.setSystemTime(new Date("2026-08-06T12:05:00Z"));
    vi.mocked(transportsService.getInstanceStatuses).mockResolvedValue([
      status({ state: "in_transit" }),
    ]);

    renderTable();

    const expander = await screen.findByRole("button", { name: /expand/i });
    await userEvent.click(expander);

    expect(screen.getByText(/approaching stuck timeout/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/InstanceRoutesTable.test.tsx
```

Expected: FAIL — cannot resolve `@/components/features/transports/InstanceRoutesTable`.

- [ ] **Step 3: Write the table**

Create `services/atlas-ui/src/components/features/transports/InstanceRoutesTable.tsx`:

```tsx
import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, ChevronDown, ChevronRight } from "lucide-react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Badge } from "@/components/ui/badge";
import { MapCell } from "@/components/map-cell";
import { Countdown } from "@/components/features/transports/Countdown";
import {
  formatDurationSeconds,
  isInstanceStuck,
} from "@/components/features/transports/transport-format";
import { useClock } from "@/lib/utils/clock";
import {
  useInstanceRoutes,
  useInstanceStatuses,
} from "@/lib/hooks/api/useTransports";
import type { Tenant } from "@/types/models/tenant";
import type { InstanceRoute, InstanceStatus } from "@/types/models/transport";

interface InstanceRoutesTableProps {
  tenant: Tenant | null;
  /** Reports the instance-route count so the tab label can carry it. */
  onCountChange?: (count: number) => void;
}

export function InstanceRoutesTable({
  tenant,
  onCountChange,
}: InstanceRoutesTableProps) {
  const routesQuery = useInstanceRoutes();
  const routes = useMemo(() => routesQuery.data ?? [], [routesQuery.data]);
  const routeIds = useMemo(() => routes.map((route) => route.id), [routes]);

  const statusQueries = useInstanceStatuses(routeIds);

  const statusesByRouteId = useMemo(() => {
    const map = new Map<string, InstanceStatus[]>();
    routeIds.forEach((routeId, index) => {
      map.set(routeId, statusQueries[index]?.data ?? []);
    });
    return map;
  }, [routeIds, statusQueries]);

  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  useEffect(() => {
    onCountChange?.(routes.length);
  }, [routes.length, onCountChange]);

  const toggle = (routeId: string) => {
    setExpanded((previous) => {
      const next = new Set(previous);
      if (next.has(routeId)) {
        next.delete(routeId);
      } else {
        next.add(routeId);
      }
      return next;
    });
  };

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10" />
            <TableHead>Route</TableHead>
            <TableHead>Live</TableHead>
            <TableHead>Capacity</TableHead>
            <TableHead>Boarding window</TableHead>
            <TableHead>Travel</TableHead>
            <TableHead>Start map</TableHead>
            <TableHead>Destination map</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {routes.map((route) => {
            const statuses = statusesByRouteId.get(route.id) ?? [];
            const isExpanded = expanded.has(route.id);
            return (
              <InstanceRouteRows
                key={route.id}
                route={route}
                statuses={statuses}
                tenant={tenant}
                isExpanded={isExpanded}
                onToggle={() => toggle(route.id)}
              />
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

function InstanceRouteRows({
  route,
  statuses,
  tenant,
  isExpanded,
  onToggle,
}: {
  route: InstanceRoute;
  statuses: InstanceStatus[];
  tenant: Tenant | null;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const expandable = statuses.length > 0;

  return (
    <>
      <TableRow>
        <TableCell>
          {expandable ? (
            <button
              type="button"
              onClick={onToggle}
              aria-label={
                isExpanded
                  ? `Collapse ${route.attributes.name}`
                  : `Expand ${route.attributes.name}`
              }
              aria-expanded={isExpanded}
            >
              {isExpanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </button>
          ) : null}
        </TableCell>
        <TableCell className="font-medium">{route.attributes.name}</TableCell>
        {/* Zero live instances is the steady state, not an error. */}
        <TableCell className="tabular-nums">{statuses.length}</TableCell>
        <TableCell className="tabular-nums">
          {route.attributes.capacity}
        </TableCell>
        <TableCell>
          {formatDurationSeconds(route.attributes.boardingWindowSeconds)}
        </TableCell>
        <TableCell>
          {formatDurationSeconds(route.attributes.travelDurationSeconds)}
        </TableCell>
        <TableCell>
          <MapCell mapId={String(route.attributes.startMapId)} tenant={tenant} />
        </TableCell>
        <TableCell>
          <MapCell
            mapId={String(route.attributes.destinationMapId)}
            tenant={tenant}
          />
        </TableCell>
      </TableRow>

      {isExpanded
        ? statuses.map((status) => (
            <LiveInstanceRow key={status.id} status={status} route={route} />
          ))
        : null}
    </>
  );
}

function LiveInstanceRow({
  status,
  route,
}: {
  status: InstanceStatus;
  route: InstanceRoute;
}) {
  const now = useClock();
  const stuck = isInstanceStuck(
    status.attributes.createdAt,
    route.attributes.boardingWindowSeconds,
    route.attributes.travelDurationSeconds,
    now,
  );

  const boarding = status.attributes.state === "boarding";

  return (
    <TableRow className="bg-muted/40">
      <TableCell />
      <TableCell colSpan={2}>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-mono text-xs">
                {status.id.slice(0, 8)}…
              </span>
            </TooltipTrigger>
            <TooltipContent copyable>
              <p>{status.id}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </TableCell>
      <TableCell>
        <Badge variant={boarding ? "outline" : "default"}>
          {boarding ? "Boarding" : "In transit"}
        </Badge>
      </TableCell>
      <TableCell className="tabular-nums">
        {status.attributes.characters}
      </TableCell>
      <TableCell colSpan={2}>
        <Countdown
          targetAt={
            boarding
              ? status.attributes.boardingUntil
              : status.attributes.arrivalAt
          }
          label={boarding ? "closes in" : "arrives in"}
        />
      </TableCell>
      <TableCell>
        {stuck ? (
          <span className="inline-flex items-center gap-1.5 text-destructive text-xs">
            <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
            Approaching stuck timeout
          </span>
        ) : null}
      </TableCell>
    </TableRow>
  );
}
```

- [ ] **Step 4: Mount it in the Instance tab**

In `services/atlas-ui/src/pages/TransportsPage.tsx`:

Add the imports:

```tsx
import { useCallback, useState } from "react";
import { InstanceRoutesTable } from "@/components/features/transports/InstanceRoutesTable";
```

(merge `useCallback`/`useState` into the existing `react` import rather than adding a second one).

Add the count state inside `TransportsPageContent`, above the return:

```tsx
  const [instanceRouteCount, setInstanceRouteCount] = useState(0);
  const handleInstanceCountChange = useCallback(
    (count: number) => setInstanceRouteCount(count),
    [],
  );
```

Replace the Instance trigger and its empty pane:

```tsx
          <TabsTrigger value="instance">Instance {instanceRouteCount}</TabsTrigger>
```

```tsx
        <TabsContent value="instance" className="flex-1 min-h-0 overflow-x-auto">
          <InstanceRoutesTable
            tenant={activeTenant}
            onCountChange={handleInstanceCountChange}
          />
        </TabsContent>
```

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/ src/pages/__tests__/TransportsPage.test.tsx
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/transports/InstanceRoutesTable.tsx \
        services/atlas-ui/src/components/features/transports/__tests__/InstanceRoutesTable.test.tsx \
        services/atlas-ui/src/pages/TransportsPage.tsx
git commit -m "feat(ui): add the Instance tab with live instances and the stuck-timeout flag"
```

---

## Task 15: Vessels tab

**Files:**
- Create: `services/atlas-ui/src/components/features/transports/VesselsTable.tsx`
- Modify: `services/atlas-ui/src/pages/TransportsPage.tsx`
- Test: `services/atlas-ui/src/components/features/transports/__tests__/VesselsTable.test.tsx` (create)

**Interfaces:**
- Consumes: `resolveVesselRoutes`, `formatDurationSeconds` (Task 10); `RouteStatePill` (Task 12).
- Produces: `<VesselsTable vessels={Vessel[]} routes={ScheduledRoute[]} />` — a pure presentational table; the page owns the queries.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/transports/__tests__/VesselsTable.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { VesselsTable } from "@/components/features/transports/VesselsTable";
import type {
  RouteState,
  ScheduledRoute,
  Vessel,
} from "@/types/models/transport";

function route(name: string, state: RouteState): ScheduledRoute {
  return {
    id: `route-${name}`,
    attributes: {
      name,
      startMapId: 1,
      stagingMapId: 2,
      enRouteMapIds: [3],
      destinationMapId: 4,
      observationMapId: 5,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "",
      nextState: "",
    },
  };
}

const vessel: Vessel = {
  id: "orbis-ellinia-boat",
  attributes: {
    uuid: "vessel-uuid",
    name: "Orbis–Ellinia Boat",
    routeAID: "Orbis to Ellinia",
    routeBID: "Ellinia to Orbis",
    turnaroundDelay: 60,
  },
};

describe("VesselsTable", () => {
  const outbound = route("Orbis to Ellinia", "open_entry");
  const inbound = route("Ellinia to Orbis", "awaiting_return");

  it("resolves both routes by name and shows their state pills", () => {
    render(<VesselsTable vessels={[vessel]} routes={[outbound, inbound]} />);

    expect(screen.getByText("Orbis–Ellinia Boat")).toBeInTheDocument();
    expect(screen.getByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.getByText("Ellinia to Orbis")).toBeInTheDocument();
    expect(screen.getByText("Boarding")).toBeInTheDocument();
    expect(screen.getByText("Awaiting return")).toBeInTheDocument();
  });

  it("anchors each row on the vessel slug so the board can deep link to it", () => {
    const { container } = render(
      <VesselsTable vessels={[vessel]} routes={[outbound, inbound]} />,
    );

    expect(container.querySelector("#vessel-orbis-ellinia-boat")).not.toBeNull();
  });

  it("flags a vessel whose route reference resolves to nothing", () => {
    render(<VesselsTable vessels={[vessel]} routes={[outbound]} />);

    expect(
      screen.getByText(/both of this vessel's routes will be out of service/i),
    ).toBeInTheDocument();
  });

  it("renders the turnaround delay", () => {
    render(<VesselsTable vessels={[vessel]} routes={[outbound, inbound]} />);

    expect(screen.getByText("1m")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/VesselsTable.test.tsx
```

Expected: FAIL — cannot resolve `@/components/features/transports/VesselsTable`.

- [ ] **Step 3: Write the table**

Create `services/atlas-ui/src/components/features/transports/VesselsTable.tsx`:

```tsx
import { AlertTriangle } from "lucide-react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import {
  formatDurationSeconds,
  resolveVesselRoutes,
} from "@/components/features/transports/transport-format";
import type { ScheduledRoute, Vessel } from "@/types/models/transport";

interface VesselsTableProps {
  vessels: Vessel[];
  routes: ScheduledRoute[];
}

/**
 * Six vessels over twelve routes: the tab exists because the unpaired-vessel
 * fault belongs to the *vessel*, not to either of its routes, and it makes the
 * alternation legible in one glance.
 */
export function VesselsTable({ vessels, routes }: VesselsTableProps) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Vessel</TableHead>
            <TableHead>Route A</TableHead>
            <TableHead>Route B</TableHead>
            <TableHead>Turnaround</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {vessels.map((vessel) => {
            const { routeA, routeB, unresolved } = resolveVesselRoutes(
              vessel,
              routes,
            );
            return (
              <TableRow key={vessel.id} id={`vessel-${vessel.id}`}>
                <TableCell className="font-medium">
                  {vessel.attributes.name}
                  {unresolved ? (
                    <div className="mt-1 inline-flex items-start gap-1.5 text-xs text-destructive">
                      <AlertTriangle
                        className="mt-0.5 h-3.5 w-3.5 shrink-0"
                        aria-hidden="true"
                      />
                      <span>
                        Unresolved route reference — both of this vessel&apos;s
                        routes will be out of service until it is fixed.
                      </span>
                    </div>
                  ) : null}
                </TableCell>
                <VesselRouteCell
                  name={vessel.attributes.routeAID}
                  route={routeA}
                />
                <VesselRouteCell
                  name={vessel.attributes.routeBID}
                  route={routeB}
                />
                <TableCell>
                  {formatDurationSeconds(vessel.attributes.turnaroundDelay)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

/**
 * Routes are matched by **name**, which is the rule the backend scheduler
 * uses. `name` here is the vessel's raw reference, shown even when it resolves
 * to nothing so the operator can see what the bad value is.
 */
function VesselRouteCell({
  name,
  route,
}: {
  name: string;
  route: ScheduledRoute | null;
}) {
  if (!route) {
    return (
      <TableCell>
        <span className="text-destructive">{name || "—"}</span>
        <span className="ml-1 text-xs text-muted-foreground">(no match)</span>
      </TableCell>
    );
  }
  return (
    <TableCell>
      <div className="flex items-center gap-2">
        <span>{route.attributes.name}</span>
        <RouteStatePill state={route.attributes.state} />
      </div>
    </TableCell>
  );
}
```

- [ ] **Step 4: Mount it in the Vessels tab**

In `services/atlas-ui/src/pages/TransportsPage.tsx`, add the import:

```tsx
import { VesselsTable } from "@/components/features/transports/VesselsTable";
```

and replace the empty Vessels pane:

```tsx
        <TabsContent value="vessels" className="flex-1 min-h-0 overflow-x-auto">
          <VesselsTable vessels={vessels} routes={routes} />
        </TabsContent>
```

- [ ] **Step 5: Run the tests**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/ src/pages/__tests__/TransportsPage.test.tsx
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/transports/VesselsTable.tsx \
        services/atlas-ui/src/components/features/transports/__tests__/VesselsTable.test.tsx \
        services/atlas-ui/src/pages/TransportsPage.tsx
git commit -m "feat(ui): add the Vessels tab with name resolution and the unpaired-vessel fault"
```

---

## Task 16: `MapFlowRail`

The ordered map chain a character actually traverses, with each leg captioned by what moves them across it. The observation map is an annotation, not a stop — it is where ARRIVED/DEPARTED effects fire, not somewhere anyone travels.

**Files:**
- Create: `services/atlas-ui/src/components/features/transports/MapFlowRail.tsx`
- Test: `services/atlas-ui/src/components/features/transports/__tests__/MapFlowRail.test.tsx` (create)

**Interfaces:**
- Consumes: `MapCell`; `ScheduledRoute` from Task 8.
- Produces: `<MapFlowRail route={ScheduledRoute} tenant={Tenant | null} />`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/transports/__tests__/MapFlowRail.test.tsx`:

```tsx
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";

import { MapFlowRail } from "@/components/features/transports/MapFlowRail";
import type { RouteState, ScheduledRoute } from "@/types/models/transport";

vi.mock("@/components/map-cell", () => ({
  MapCell: ({ mapId }: { mapId: string }) => <span>map:{mapId}</span>,
}));

function route(state: RouteState): ScheduledRoute {
  return {
    id: "r1",
    attributes: {
      name: "Orbis to Ellinia",
      startMapId: 200000100,
      stagingMapId: 200000110,
      enRouteMapIds: [200090010, 200090011],
      destinationMapId: 101000300,
      observationMapId: 200000111,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "",
      nextState: "",
    },
  };
}

describe("MapFlowRail", () => {
  it("renders the whole chain in order", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    const stops = screen
      .getAllByText(/^map:/)
      .map((node) => node.textContent);

    expect(stops).toEqual([
      "map:200000100",
      "map:200000110",
      "map:200090010",
      "map:200090011",
      "map:101000300",
      // the observation map is rendered separately, after the chain
      "map:200000111",
    ]);
  });

  it("captions each leg with what moves a character across it", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    expect(screen.getByText("walk in")).toBeInTheDocument();
    expect(screen.getAllByText("warp on departure").length).toBeGreaterThan(0);
    expect(screen.getByText("warp on arrival")).toBeInTheDocument();
  });

  it("annotates the observation map as an effect origin, not a stop", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    expect(
      screen.getByText(/ARRIVED\/DEPARTED effects fire/i),
    ).toBeInTheDocument();
  });

  it("exposes the rail to assistive tech as a labelled figure", () => {
    render(<MapFlowRail route={route("open_entry")} tenant={null} />);

    const figure = screen.getByRole("img", { name: /map flow/i });
    expect(figure).toBeInTheDocument();
  });

  it("emphasises the en-route segment while the route is in transit", () => {
    const { container } = render(
      <MapFlowRail route={route("in_transit")} tenant={null} />,
    );

    expect(container.querySelector("[data-en-route-active='true']")).not.toBeNull();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/MapFlowRail.test.tsx
```

Expected: FAIL — cannot resolve `@/components/features/transports/MapFlowRail`.

- [ ] **Step 3: Write the rail**

Create `services/atlas-ui/src/components/features/transports/MapFlowRail.tsx`:

```tsx
import { MapCell } from "@/components/map-cell";
import { cn } from "@/lib/utils";
import type { Tenant } from "@/types/models/tenant";
import type { ScheduledRoute } from "@/types/models/transport";

interface MapFlowRailProps {
  route: ScheduledRoute;
  tenant: Tenant | null;
}

interface Stop {
  mapId: number;
  enRoute: boolean;
}

interface Leg {
  caption: string;
  enRoute: boolean;
}

/**
 * The ordered chain of maps a character traverses:
 * start → staging → en-route… → destination.
 *
 * The connectors are SVG (role="img", labelled) so the figure reads as one
 * thing to assistive tech, while the stop badges stay ordinary HTML — MapCell
 * carries a link and a copyable tooltip, and role="img" would hide them.
 */
export function MapFlowRail({ route, tenant }: MapFlowRailProps) {
  const {
    name,
    startMapId,
    stagingMapId,
    enRouteMapIds,
    destinationMapId,
    observationMapId,
    state,
  } = route.attributes;

  const stops: Stop[] = [
    { mapId: startMapId, enRoute: false },
    { mapId: stagingMapId, enRoute: false },
    ...enRouteMapIds.map((mapId) => ({ mapId, enRoute: true })),
    { mapId: destinationMapId, enRoute: false },
  ];

  // One leg between each adjacent pair of stops. The caption names the
  // mechanism that moves a character across it.
  const legs: Leg[] = stops.slice(1).map((stop, index) => {
    const previous = stops[index]!;
    if (!previous.enRoute && !stop.enRoute && index === 0) {
      return { caption: "walk in", enRoute: false };
    }
    if (stop.enRoute) {
      return { caption: "warp on departure", enRoute: true };
    }
    if (previous.enRoute) {
      return { caption: "warp on arrival", enRoute: false };
    }
    return { caption: "warp on departure", enRoute: false };
  });

  const inTransit = state === "in_transit";

  return (
    <div className="space-y-3">
      <div className="overflow-x-auto">
        <div className="flex min-w-max items-start gap-2">
          {stops.map((stop, index) => (
            <div key={`${stop.mapId}-${index}`} className="flex items-start gap-2">
              <div className="flex flex-col items-center gap-1 pt-1">
                <MapCell mapId={String(stop.mapId)} tenant={tenant} />
              </div>
              {index < legs.length ? (
                <RailLeg
                  leg={legs[index]!}
                  active={inTransit && legs[index]!.enRoute}
                />
              ) : null}
            </div>
          ))}
        </div>
      </div>

      <p className="text-xs text-muted-foreground">
        Observation map — where ARRIVED/DEPARTED effects fire; characters never
        travel here.{" "}
        <span className="align-middle">
          <MapCell mapId={String(observationMapId)} tenant={tenant} />
        </span>
      </p>

      <span className="sr-only">
        Map flow for {name}: {stops.map((stop) => stop.mapId).join(" then ")}
      </span>
    </div>
  );
}

function RailLeg({ leg, active }: { leg: Leg; active: boolean }) {
  return (
    <div className="flex flex-col items-center gap-0.5">
      <svg
        role="img"
        aria-label={`Map flow leg: ${leg.caption}`}
        width="72"
        height="12"
        viewBox="0 0 72 12"
        data-en-route-active={active ? "true" : undefined}
        className={cn(
          "text-muted-foreground",
          active && "text-primary",
        )}
      >
        <line
          x1="0"
          y1="6"
          x2="64"
          y2="6"
          stroke="currentColor"
          strokeWidth={active ? 3 : 1.5}
          strokeDasharray={leg.enRoute ? "4 3" : undefined}
        />
        <path d="M64 2 L72 6 L64 10 Z" fill="currentColor" />
      </svg>
      <span className="text-[10px] text-muted-foreground whitespace-nowrap">
        {leg.caption}
      </span>
    </div>
  );
}
```

- [ ] **Step 4: Run the test**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/MapFlowRail.test.tsx
```

Expected: PASS (5 tests). The `role="img"` query matches the leg SVGs' `aria-label`, which all begin "Map flow leg: …".

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/transports/MapFlowRail.tsx \
        services/atlas-ui/src/components/features/transports/__tests__/MapFlowRail.test.tsx
git commit -m "feat(ui): add the route map-flow rail with captioned legs and observation-map annotation"
```

---

## Task 17: `VesselTimeline`

A windowed strip of trips around now. A shared-vessel route shows two lanes so the alternation and the turnaround gap are visible; an independent route shows one.

**Files:**
- Create: `services/atlas-ui/src/components/features/transports/VesselTimeline.tsx`
- Test: `services/atlas-ui/src/components/features/transports/__tests__/VesselTimeline.test.tsx` (create)

**Interfaces:**
- Consumes: `timelineHalfWindowMs`, `utcTimeOfDayMs`, `nowUtcTimeOfDayMs`, `segmentSpan`, `formatTimeOfDay` (all Task 10); `TripSchedule` (Task 8).
- Produces:
  - `interface TimelineLane { label: string; trips: TripSchedule[]; emphasised?: boolean }`
  - `<VesselTimeline lanes={TimelineLane[]} nowEpochMs={number} />`

- [ ] **Step 1: Write the failing timeline test**

Create `services/atlas-ui/src/components/features/transports/__tests__/VesselTimeline.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { VesselTimeline } from "@/components/features/transports/VesselTimeline";
import type { TripSchedule } from "@/types/models/transport";

function trip(id: string, open: string, closed: string, departure: string, arrival: string): TripSchedule {
  // The 2023 date is the schedule's stale computing day; only the time matters.
  const at = (hhmm: string) => `2023-01-01T${hhmm}:00Z`;
  return {
    id,
    attributes: {
      boardingOpen: at(open),
      boardingClosed: at(closed),
      departure: at(departure),
      arrival: at(arrival),
    },
  };
}

const nowEpochMs = Date.parse("2026-08-06T12:00:00Z");

describe("VesselTimeline", () => {
  const outbound = [
    trip("a1", "11:45", "11:50", "11:52", "12:02"),
    trip("a2", "12:15", "12:20", "12:22", "12:32"),
  ];
  const inbound = [trip("b1", "12:03", "12:08", "12:10", "12:20")];

  it("renders one lane for an independent route", () => {
    render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(screen.getByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.queryByText("Ellinia to Orbis")).toBeNull();
  });

  it("renders both lanes for a shared vessel", () => {
    render(
      <VesselTimeline
        lanes={[
          { label: "Orbis to Ellinia", trips: outbound, emphasised: true },
          { label: "Ellinia to Orbis", trips: inbound },
        ]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(screen.getByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.getByText("Ellinia to Orbis")).toBeInTheDocument();
  });

  it("is a labelled figure with a NOW marker", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(screen.getByRole("img", { name: /trip timeline/i })).toBeInTheDocument();
    expect(container.querySelector("[data-now-marker]")).not.toBeNull();
  });

  it("labels trip times as time of day only, never with a date", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: outbound }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(container.textContent).not.toContain("2023");
    expect(container.textContent).toContain("11:45");
  });

  it("renders three segments per in-window trip", () => {
    const { container } = render(
      <VesselTimeline
        lanes={[{ label: "Orbis to Ellinia", trips: [outbound[1]!] }]}
        nowEpochMs={nowEpochMs}
      />,
    );

    expect(container.querySelectorAll("[data-segment]")).toHaveLength(3);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/__tests__/VesselTimeline.test.tsx
```

Expected: FAIL — cannot resolve `@/components/features/transports/VesselTimeline`.

- [ ] **Step 3: Write the timeline**

Create `services/atlas-ui/src/components/features/transports/VesselTimeline.tsx`:

```tsx
import { useMemo } from "react";

import {
  formatTimeOfDay,
  nowUtcTimeOfDayMs,
  segmentSpan,
  timelineHalfWindowMs,
  utcTimeOfDayMs,
} from "@/components/features/transports/transport-format";
import type { TripSchedule } from "@/types/models/transport";

export interface TimelineLane {
  label: string;
  trips: TripSchedule[];
  /** The route being viewed, drawn with more weight than its partner. */
  emphasised?: boolean;
}

interface VesselTimelineProps {
  lanes: TimelineLane[];
  /** Epoch milliseconds; the strip is centred on this instant's UTC time of day. */
  nowEpochMs: number;
}

const WIDTH = 720;
const LANE_HEIGHT = 34;
const LANE_GAP = 10;
const TOP_PAD = 18;

const SEGMENT_STYLE = {
  open: { className: "fill-emerald-500/70", label: "boarding open" },
  locked: { className: "fill-amber-500/70", label: "boarding closed" },
  transit: { className: "fill-sky-500/70", label: "in transit" },
} as const;

/**
 * A windowed strip of trips around now.
 *
 * Trip boundaries are UTC times of day (the schedule is anchored on UTC
 * midnight and its date component is stale), so everything here — including the
 * NOW marker — is positioned in milliseconds since UTC midnight. Times are
 * labelled UTC for the same reason.
 *
 * The window's half-width is derived from the trips' own median spacing rather
 * than a fixed ±30 minutes, because a shared vessel's real spacing comes from
 * arrival-plus-turnaround, not from either route's configured cycle.
 */
export function VesselTimeline({ lanes, nowEpochMs }: VesselTimelineProps) {
  const nowMs = nowUtcTimeOfDayMs(nowEpochMs);

  const halfWindowMs = useMemo(
    () =>
      timelineHalfWindowMs(
        lanes.flatMap((lane) =>
          lane.trips.map((trip) => utcTimeOfDayMs(trip.attributes.boardingOpen)),
        ),
      ),
    [lanes],
  );

  const height = TOP_PAD + lanes.length * (LANE_HEIGHT + LANE_GAP);

  return (
    <div className="space-y-2">
      <div className="overflow-x-auto">
        <svg
          role="img"
          aria-label={`Trip timeline for ${lanes
            .map((lane) => lane.label)
            .join(" and ")}, covering ${Math.round(
            halfWindowMs / 60_000,
          )} minutes either side of now (times UTC)`}
          viewBox={`0 0 ${WIDTH} ${height}`}
          width="100%"
          style={{ minWidth: `${WIDTH / 2}px` }}
        >
          {lanes.map((lane, laneIndex) => {
            const y = TOP_PAD + laneIndex * (LANE_HEIGHT + LANE_GAP);
            return (
              <g key={lane.label}>
                <rect
                  x={0}
                  y={y}
                  width={WIDTH}
                  height={LANE_HEIGHT}
                  className="fill-muted"
                  rx={4}
                />
                {lane.trips.map((trip) => (
                  <TripSegments
                    key={trip.id}
                    trip={trip}
                    y={y}
                    nowMs={nowMs}
                    halfWindowMs={halfWindowMs}
                    emphasised={lane.emphasised ?? false}
                  />
                ))}
              </g>
            );
          })}

          <line
            data-now-marker=""
            x1={WIDTH / 2}
            y1={0}
            x2={WIDTH / 2}
            y2={height}
            className="stroke-foreground"
            strokeWidth={1.5}
            strokeDasharray="3 3"
          />
          <text
            x={WIDTH / 2 + 4}
            y={12}
            className="fill-foreground"
            fontSize={10}
          >
            NOW
          </text>
        </svg>
      </div>

      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {lanes.map((lane) => (
          <span key={lane.label} className="font-medium text-foreground">
            {lane.label}
          </span>
        ))}
        <span>boarding open</span>
        <span>boarding closed</span>
        <span>in transit</span>
        <span>
          times UTC
          {lanes[0]?.trips[0]
            ? ` · first trip boards ${formatTimeOfDay(
                lanes[0].trips[0].attributes.boardingOpen,
              )}`
            : ""}
        </span>
      </div>
    </div>
  );
}

function TripSegments({
  trip,
  y,
  nowMs,
  halfWindowMs,
  emphasised,
}: {
  trip: TripSchedule;
  y: number;
  nowMs: number;
  halfWindowMs: number;
  emphasised: boolean;
}) {
  const { boardingOpen, boardingClosed, departure, arrival } = trip.attributes;

  const parts = [
    {
      kind: "open" as const,
      start: utcTimeOfDayMs(boardingOpen),
      end: utcTimeOfDayMs(boardingClosed),
    },
    {
      kind: "locked" as const,
      start: utcTimeOfDayMs(boardingClosed),
      end: utcTimeOfDayMs(departure),
    },
    {
      kind: "transit" as const,
      start: utcTimeOfDayMs(departure),
      end: utcTimeOfDayMs(arrival),
    },
  ];

  const laneY = emphasised ? y + 4 : y + 10;
  const laneHeight = emphasised ? LANE_HEIGHT - 8 : LANE_HEIGHT - 20;

  return (
    <>
      {parts.map((part) => {
        const span = segmentSpan(part.start, part.end, nowMs, halfWindowMs);
        if (!span || span.right <= span.left) return null;
        return (
          <rect
            key={`${trip.id}-${part.kind}`}
            data-segment={part.kind}
            x={span.left * WIDTH}
            y={laneY}
            width={(span.right - span.left) * WIDTH}
            height={laneHeight}
            rx={2}
            className={SEGMENT_STYLE[part.kind].className}
          >
            <title>
              {`${SEGMENT_STYLE[part.kind].label} ${formatTimeOfDay(
                part.kind === "open"
                  ? boardingOpen
                  : part.kind === "locked"
                    ? boardingClosed
                    : departure,
              )} UTC`}
            </title>
          </rect>
        );
      })}
    </>
  );
}
```

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-ui && npx vitest run src/components/features/transports/
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/transports/VesselTimeline.tsx \
        services/atlas-ui/src/components/features/transports/__tests__/VesselTimeline.test.tsx
git commit -m "feat(ui): add the windowed vessel trip timeline with one or two lanes"
```

---

## Task 18: Route detail page

**Files:**
- Create: `services/atlas-ui/src/pages/TransportRouteDetailPage.tsx`
- Modify: `services/atlas-ui/src/App.tsx`
- Test: `services/atlas-ui/src/pages/__tests__/TransportRouteDetailPage.test.tsx` (create)

**Interfaces:**
- Consumes: everything from Tasks 8-17.
- Produces: `TransportRouteDetailPage` (named export); route `/transports/routes/:routeId`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/pages/__tests__/TransportRouteDetailPage.test.tsx`:

```tsx
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { TransportRouteDetailPage } from "@/pages/TransportRouteDetailPage";
import { transportsService } from "@/services/api/transports.service";
import type { Tenant } from "@/types/models/tenant";
import type {
  RouteState,
  ScheduledRoute,
  TripSchedule,
} from "@/types/models/transport";

const mockTenant: Tenant = {
  id: "tenant-1",
  attributes: {
    name: "Test Tenant",
    region: "GMS",
    majorVersion: 83,
    minorVersion: 1,
  },
};

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: mockTenant,
    tenants: [mockTenant],
    loading: false,
    setActiveTenant: vi.fn(),
    refreshTenants: vi.fn(),
    refreshAndSelectTenant: vi.fn(),
    fetchTenantConfiguration: vi.fn(),
  }),
}));

vi.mock("@/components/map-cell", () => ({
  MapCell: ({ mapId }: { mapId: string }) => <span>map:{mapId}</span>,
}));

vi.mock("@/services/api/transports.service", () => ({
  transportsService: {
    getScheduledRoute: vi.fn(),
    getScheduledRoutes: vi.fn(),
    getVessels: vi.fn(),
  },
}));

function scheduledRoute(state: RouteState): ScheduledRoute {
  return {
    id: "r1",
    attributes: {
      name: "Orbis to Ellinia",
      startMapId: 200000100,
      stagingMapId: 200000110,
      enRouteMapIds: [200090010],
      destinationMapId: 101000300,
      observationMapId: 200000111,
      state,
      boardingWindowSeconds: 300,
      preDepartureSeconds: 120,
      travelDurationSeconds: 600,
      cycleIntervalSeconds: 900,
      nextTransitionAt: "2026-08-06T12:05:00Z",
      nextState: "locked_entry",
    },
  };
}

const trips: TripSchedule[] = [
  {
    id: "t1",
    attributes: {
      boardingOpen: "2023-01-01T11:45:00Z",
      boardingClosed: "2023-01-01T11:50:00Z",
      departure: "2023-01-01T11:52:00Z",
      arrival: "2023-01-01T12:02:00Z",
    },
  },
];

function renderDetail() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <MemoryRouter initialEntries={["/transports/routes/r1"]}>
      <QueryClientProvider client={client}>
        <Routes>
          <Route
            path="/transports/routes/:routeId"
            element={<TransportRouteDetailPage />}
          />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("TransportRouteDetailPage", () => {
  beforeEach(() => {
    vi.mocked(transportsService.getScheduledRoute).mockReset();
    vi.mocked(transportsService.getScheduledRoutes).mockReset();
    vi.mocked(transportsService.getVessels).mockReset();
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([]);
  });

  it("shows the route name, state pill and countdown", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("open_entry"),
      schedule: trips,
    });

    renderDetail();

    expect(await screen.findByText("Orbis to Ellinia")).toBeInTheDocument();
    expect(screen.getByText("Boarding")).toBeInTheDocument();
    expect(screen.getByText("closes in")).toBeInTheDocument();
  });

  it("renders the map chain and the key/value strip", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("open_entry"),
      schedule: trips,
    });

    renderDetail();

    await screen.findByText("Orbis to Ellinia");
    expect(screen.getByText("map:200000100")).toBeInTheDocument();
    expect(screen.getByText("map:101000300")).toBeInTheDocument();
    expect(screen.getByText("Trips scheduled today")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("never renders a schedule date", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("open_entry"),
      schedule: trips,
    });

    const { container } = renderDetail();

    await screen.findByText("Orbis to Ellinia");
    expect(container.textContent).not.toContain("2023");
  });

  it("replaces the timeline with a fault message when there are no trips", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("out_of_service"),
      schedule: [],
    });

    renderDetail();

    await screen.findByText("Orbis to Ellinia");
    expect(screen.getByText(/no trips were scheduled/i)).toBeInTheDocument();
    expect(screen.getByText(/scheduler drops any trip/i)).toBeInTheDocument();
  });

  it("names the unresolved-vessel cause when the route's vessel does not pair", async () => {
    vi.mocked(transportsService.getScheduledRoute).mockResolvedValue({
      route: scheduledRoute("out_of_service"),
      schedule: [],
    });
    vi.mocked(transportsService.getScheduledRoutes).mockResolvedValue([
      scheduledRoute("out_of_service"),
    ]);
    vi.mocked(transportsService.getVessels).mockResolvedValue([
      {
        id: "boat",
        attributes: {
          uuid: "u",
          name: "Boat",
          routeAID: "Orbis to Ellinia",
          routeBID: "Missing Route",
          turnaroundDelay: 60,
        },
      },
    ]);

    renderDetail();

    await screen.findByText("Orbis to Ellinia");
    expect(
      await screen.findByText(/vessel .*does not resolve/i),
    ).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-ui && npx vitest run src/pages/__tests__/TransportRouteDetailPage.test.tsx
```

Expected: FAIL — cannot resolve `@/pages/TransportRouteDetailPage`.

- [ ] **Step 3: Write the page**

Create `services/atlas-ui/src/pages/TransportRouteDetailPage.tsx`:

```tsx
import { useMemo, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import { AlertTriangle, ArrowLeft } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useTenant } from "@/context/tenant-context";
import {
  useScheduledRoute,
  useScheduledRoutes,
  useVessels,
} from "@/lib/hooks/api/useTransports";
import { Countdown } from "@/components/features/transports/Countdown";
import { MapFlowRail } from "@/components/features/transports/MapFlowRail";
import { RouteStatePill } from "@/components/features/transports/RouteStatePill";
import {
  VesselTimeline,
  type TimelineLane,
} from "@/components/features/transports/VesselTimeline";
import {
  findVesselForRoute,
  formatDurationSeconds,
  resolveVesselRoutes,
  transitionLabel,
} from "@/components/features/transports/transport-format";
import { useClock } from "@/lib/utils/clock";
import { MapCell } from "@/components/map-cell";

export function TransportRouteDetailPage() {
  const { routeId = "" } = useParams();
  const { activeTenant } = useTenant();
  const now = useClock();

  const detailQuery = useScheduledRoute(routeId);
  const routesQuery = useScheduledRoutes();
  const vesselsQuery = useVessels();

  const detail = detailQuery.data;
  const routes = useMemo(() => routesQuery.data ?? [], [routesQuery.data]);
  const vessels = useMemo(() => vesselsQuery.data ?? [], [vesselsQuery.data]);

  const vessel = useMemo(
    () => (detail ? findVesselForRoute(detail.route, vessels) : null),
    [detail, vessels],
  );

  const partner = useMemo(() => {
    if (!detail || !vessel) return null;
    const { routeA, routeB } = resolveVesselRoutes(vessel, routes);
    if (routeA && routeA.id !== detail.route.id) return routeA;
    if (routeB && routeB.id !== detail.route.id) return routeB;
    return null;
  }, [detail, vessel, routes]);

  const partnerDetailQuery = useScheduledRoute(partner?.id ?? "");

  const vesselUnresolved = useMemo(() => {
    if (!vessel) return false;
    return resolveVesselRoutes(vessel, routes).unresolved;
  }, [vessel, routes]);

  const lanes: TimelineLane[] = useMemo(() => {
    if (!detail) return [];
    const own: TimelineLane = {
      label: detail.route.attributes.name,
      trips: detail.schedule,
      emphasised: true,
    };
    const partnerSchedule = partnerDetailQuery.data;
    if (partner && partnerSchedule) {
      return [
        own,
        { label: partner.attributes.name, trips: partnerSchedule.schedule },
      ];
    }
    return [own];
  }, [detail, partner, partnerDetailQuery.data]);

  if (detailQuery.isLoading || !detail) {
    return (
      <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  const attributes = detail.route.attributes;
  const label = transitionLabel(attributes.nextState);

  return (
    <div className="flex flex-col flex-1 min-h-0 space-y-6 p-10 pb-16">
      <Link
        to="/transports"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:underline"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Transports
      </Link>

      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-2xl font-bold tracking-tight">{attributes.name}</h2>
        <RouteStatePill state={attributes.state} />
        {label && attributes.nextTransitionAt ? (
          <Countdown targetAt={attributes.nextTransitionAt} label={label} />
        ) : null}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Map flow</CardTitle>
        </CardHeader>
        <CardContent>
          <MapFlowRail route={detail.route} tenant={activeTenant} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-2 gap-x-8 gap-y-3 sm:grid-cols-3">
            <Field label="Observation map">
              <MapCell
                mapId={String(attributes.observationMapId)}
                tenant={activeTenant}
              />
            </Field>
            <Field label="Boarding window">
              {formatDurationSeconds(attributes.boardingWindowSeconds)}
            </Field>
            <Field label="Pre-departure">
              {formatDurationSeconds(attributes.preDepartureSeconds)}
            </Field>
            <Field label="Travel duration">
              {formatDurationSeconds(attributes.travelDurationSeconds)}
            </Field>
            <Field label="Cycle interval">
              {formatDurationSeconds(attributes.cycleIntervalSeconds)}
            </Field>
            <Field label="Trips scheduled today">
              {String(detail.schedule.length)}
            </Field>
            <Field label="Shared vessel">
              {vessel ? vessel.attributes.name : "—"}
            </Field>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Trip timeline</CardTitle>
        </CardHeader>
        <CardContent>
          {detail.schedule.length === 0 ? (
            <ScheduleFault
              vesselName={vessel?.attributes.name ?? null}
              vesselUnresolved={vesselUnresolved}
            />
          ) : (
            <VesselTimeline lanes={lanes} nowEpochMs={now} />
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 text-sm">{children}</dd>
    </div>
  );
}

/**
 * A route with no trips is a configuration fault, and there are exactly two
 * producible causes: a cycle-interval / travel-duration combination that
 * leaves no trip fitting inside the day (the scheduler drops a trip unless it
 * arrives before end of day), or membership in a vessel whose partner does not
 * resolve (which returns an empty schedule for both sides).
 */
function ScheduleFault({
  vesselName,
  vesselUnresolved,
}: {
  vesselName: string | null;
  vesselUnresolved: boolean;
}) {
  return (
    <div className="flex items-start gap-2 text-sm text-destructive">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="space-y-1">
        <p>No trips were scheduled for this route today.</p>
        {vesselUnresolved && vesselName ? (
          <p>
            Its shared vessel <strong>{vesselName}</strong> does not resolve —
            one of its route references matches no route, which zeroes the
            schedule for both sides. Fix the vessel configuration.
          </p>
        ) : (
          <p>
            Check the route&apos;s cycle interval and travel duration: the
            scheduler drops any trip that would not arrive before the end of the
            day, so a cycle longer than the day&apos;s remaining room produces
            no trips at all.
          </p>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Wire the route**

In `services/atlas-ui/src/App.tsx`, add the lazy import beside `TransportsPage`:

```tsx
const TransportRouteDetailPage = lazyWithReload(() =>
  import("@/pages/TransportRouteDetailPage").then((m) => ({
    default: m.TransportRouteDetailPage,
  })),
);
```

and the route immediately after `/transports`:

```tsx
                    <Route
                      path="/transports/routes/:routeId"
                      element={<TransportRouteDetailPage />}
                    />
```

- [ ] **Step 5: Run the tests and the build**

```bash
cd services/atlas-ui && npx vitest run src/pages/__tests__/TransportRouteDetailPage.test.tsx && npm run build
```

Expected: PASS and a clean build.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/pages/TransportRouteDetailPage.tsx \
        services/atlas-ui/src/pages/__tests__/TransportRouteDetailPage.test.tsx \
        services/atlas-ui/src/App.tsx
git commit -m "feat(ui): add the transport route detail page with map flow, config strip and timeline"
```

---

## Task 19: Full verification sweep

Nothing in this task is optional. Report what actually ran, with its output — a skipped step is a false "verified".

**Files:** none created or modified unless a check fails.

**Interfaces:**
- Consumes: all prior tasks.

- [ ] **Step 1: Go module checks**

```bash
cd services/atlas-transports/atlas.com/transports
go test -race ./...
go vet ./...
go build ./...
```

Expected: all three clean. Fix anything that is not before continuing.

- [ ] **Step 2: Repo guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
```

Run both from the worktree root. Expected: exit 0, no findings.

- [ ] **Step 3: Lint and format**

```bash
tools/lint.sh          # fix mode - rewrites files in place
tools/lint.sh --check  # must be clean
```

`tools/lint.sh` needs nvm 22 on PATH for its atlas-ui half; without it the check false-fails. Run the fix pass first, then re-run `--check` and confirm it exits 0. If the fix pass modified files, include them in the commit at Step 6.

- [ ] **Step 4: atlas-ui tests and build**

```bash
cd services/atlas-ui
npm run test
npm run build
```

Both are required — `npm run build` (`tsc -b && vite build`) is what type-checks, and it type-checks the tests too. `npm run test` alone will not catch a type error.

- [ ] **Step 5: Confirm no `go.mod` changed**

```bash
git diff --name-only origin/main...HEAD -- '*go.mod' '*go.sum'
```

Expected: empty. If atlas-transports' `go.mod` did change, run `docker buildx bake atlas-transports` from the worktree root and confirm it succeeds — `go build` against the workspace will not catch a missing `COPY libs/...` in the shared Dockerfile.

- [ ] **Step 6: Commit any lint/format fallout**

```bash
git status --short
git add -A
git commit -m "chore: apply lint and format fixes"
```

Skip the commit if the tree is clean.

- [ ] **Step 7: Code review before the PR**

Run `superpowers:requesting-code-review`. Both Go and TypeScript changed, so it must dispatch:

- `plan-adherence-reviewer` — every task in this plan actually implemented
- `backend-guidelines-reviewer` — DOM-* checklist over atlas-transports
- `frontend-guidelines-reviewer` — FE-* checklist over the atlas-ui changes

Pin the reviewer subagents to Sonnet or Haiku. Each writes to `docs/tasks/task-198-transports-ui-surface/audit.md`. Address findings before opening the PR; verify the worktree is clean after the reviewer runs.

- [ ] **Step 8: Manual acceptance against a live tenant**

The FR-6.1 proof is a diff, not an assertion: `GET /api/transports/instance-routes/{routeId}/status` returns an empty list on `main` for the same request and returns live instances on this branch. Also confirm by eye:

- the Scheduled tab lists every seeded route with a ticking countdown
- an `out_of_service` route sorts to the top with the fault pill
- a shared-vessel route's detail page shows two lanes with a visible turnaround gap
- an instance route with no live instances shows `0` and does not expand
- switching tenants clears and refetches everything, with no cross-tenant rows

Record what you observed. If a live tenant is unavailable, say so explicitly rather than implying the check passed.

---

## Spec Coverage

| Requirement | Task |
|---|---|
| FR-1.1 route + sidebar entry | 13 |
| FR-1.2 three tabs, counts, `?tab=` | 13, 14, 15 |
| FR-1.3 Scheduled columns + severity sort | 10, 13 |
| FR-1.4 state pill, fault treatment | 12 |
| FR-1.5 labelled countdown, em dash | 9, 10, 13 |
| FR-1.6 `MapCell` | 13 |
| FR-1.7 route name links to detail | 13 |
| FR-1.8 vessel column + anchor link | 13, 15 |
| FR-2.1 detail route on the stable derived id | 18 |
| FR-2.2 header name/pill/countdown | 18 |
| FR-2.3 map-flow rail, captioned legs, en-route emphasis | 16 |
| FR-2.4 observation map as annotation | 16 |
| FR-2.5 key/value strip | 18 |
| FR-2.6 windowed timeline, three segments, NOW marker | 17 |
| FR-2.7 two lanes for a shared vessel | 17, 18 |
| FR-2.8 no-trips fault message | 18 |
| FR-3.1 instance route list | 14 |
| FR-3.2 expandable live instances | 14 |
| FR-3.3 stuck-timeout flag | 6, 10, 14 |
| FR-3.4 truncated id + copyable tooltip | 14 |
| FR-3.5 zero instances is not an error | 14 |
| FR-4.1–4.3 vessels table, name resolution, pills | 15 |
| FR-4.4 unpaired-vessel fault | 10, 15 |
| FR-5.1 30s `refetchInterval` | 11 |
| FR-5.2 local 1-second tick from an absolute instant | 9 |
| FR-5.3 clamp at `0:00` | 9, 10 |
| FR-5.4 freshness/stale treatment | 12, 13 |
| FR-5.5 tenant gating | 11 |
| FR-6.1 tenant-scoping fix | 5 |
| FR-6.2 re-pointed test + regression guard | 5 |
| FR-6.3 route duration attributes | 2 |
| FR-6.4 `nextTransitionAt` / `nextState` | 1, 3 |
| FR-6.5 one duration rule, scheduled and instance alike | 2, 6 |
| FR-6.6 no vessels endpoint; read from configuration | 8 |
| Open Q1 sparse fieldsets → `include=schedule` | 4 |
| Open Q2 server-computed transition | 1, 3 |
| Open Q3 keep the Vessels tab | 15 |
| Open Q4 derived timeline window | 10, 17 |
| Design §3.5 `createdAt` on instance status | 6 |
| Design §3.6 docs + Bruno | 7 |
| Verification plan | 19 |
