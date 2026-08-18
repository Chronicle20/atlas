# bug: VOYAGE_ARRIVED derives a different voyage id than VOYAGE_DEPARTED

Task: task-231-generalized-events-service
Found in: live testing on `atlas-pr-1375` (2026-08-18), occurrence
`fd1d0c3a-cdad-4192-8b9f-987efd3896fe`.

## Reproduced

Yes — twice, independently.

**1. Unit-level, in-repo.** A throwaway test against
`services/atlas-transports/atlas.com/transports/transport/model.go`'s `Evaluate`
with two same-day trips (A: boarding 12:00, departs 13:00, arrives 13:30;
B: boarding 14:00, departs 15:00):

```
DEPART tick (13:10): state=in_transit      tripId=26546022-… departedAt=2026-08-15 13:00:00 +0000 UTC
ARRIVE tick (13:31): state=awaiting_return tripId=8c7e1e64-… departedAt=2026-08-14 15:00:00 +0000 UTC
```

The arrival tick reports **trip B's** id and a departure instant on the
**previous calendar day**. (The scratch test was removed; the fix must land its
own permanent version — see `## Fix`.)

**2. In the live namespace.** Every `VOYAGE_ARRIVED` on the topic carries a
`voyageId` that matches no `VOYAGE_DEPARTED`. For the Ellinia→Orbis route
(`stagingMapId: 101000301`), from `atlas-events` consumer logs:

```
VOYAGE_ARRIVED   voyageId=481c79af-…  departedAt=2026-08-17T13:39:30Z
VOYAGE_DEPARTED  voyageId=022449ed-…  departedAt=2026-08-18T13:39:30Z
VOYAGE_ARRIVED   voyageId=0d93b8ea-…  departedAt=2026-08-17T14:09:40Z
VOYAGE_DEPARTED  voyageId=d7d348ea-…  departedAt=2026-08-18T14:09:40Z
VOYAGE_ARRIVED   voyageId=6a8e46dd-…  departedAt=2026-08-17T14:39:50Z
VOYAGE_DEPARTED  voyageId=e16cd0fc-…  departedAt=2026-08-18T14:39:50Z
```

Each arrival reports the *next* trip's departure time-of-day, dated **yesterday**
(the run was on 2026-08-18). No arrival id ever equals a departure id. The same
pattern holds for every other route sampled (e.g. `ab987172-…`: six
ARRIVED/DEPARTED pairs, twelve distinct voyage ids).

`atlas-transports` logs confirm the arrival transition itself fires normally
("Transport for route […] has arrived at […]"), and the reporter was warped off
the boat — so the `AwaitingReturn` branch ran; only the identity it carried was
wrong.

## Observed

The CRIMSON_BALROG occurrence stays `ACTIVE` forever. The Balrog is never
despawned and the boat-shake visual is never hidden. The player is warped to the
destination map (the warp is in the same branch and is unaffected).

## Expected

`VOYAGE_ARRIVED` carries the same `voyageId` as the `VOYAGE_DEPARTED` of the trip
that is arriving, so `crimsonbalrog.ArrivalProcessorImpl.OnVoyageArrived` →
`occurrence.Processor.GetActiveByVoyage` matches the live occurrence and
completes it (FR-B17/FR-B19).

## Root cause

`Model.Evaluate` (`services/atlas-transports/atlas.com/transports/transport/model.go`)
reports `Transition.TripId` / `Transition.DepartedAt` from `nextTrip`, which is
`inTransitTrip` when one exists and otherwise `futureTrip`.

At the arrival instant the trip that just landed is no longer in transit — the
same-day selector requires `nowTimeOfDay.After(departure) && nowTimeOfDay.Before(arrival)`
(model.go:~185), which is false one tick past arrival. So `inTransitTrip == nil`,
`nextTrip` becomes the **next** trip, and the `AwaitingReturn` branch
(`nowTimeOfDay.Before(boardingOpen)`, model.go:~246) returns `to(AwaitingReturn, …)`
carrying that next trip's id.

`DepartedAt` is compounded by `materializeDeparture` (model.go:~150): the next
trip's departure time-of-day is still in the future relative to `now`, so it
subtracts 24h and lands on yesterday. That helper is only correct under the
in-transit reading it was written for.

`emitVoyageEvent` (`transport/processor.go:~250`) then computes
`VoyageId(t, r.Id(), tr.TripId, tr.DepartedAt)` from those two wrong inputs, so
the id is a pure function of the wrong trip — deterministically never equal to
the departure's id.

Test coverage missed it: `evaluate_test.go`'s
`TestEvaluateReportsSelectedTripAndDepartureInstant` and
`TestEvaluateDepartureInstantForSameDayTrip` assert trip identity **only for the
`InTransit` state**. Neither exercises the `AwaitingReturn` tick — which is the
only tick that emits `VOYAGE_ARRIVED`. design §18 risk 1 named this exact failure
and the guard written for it covers the wrong half of it.

Nothing in `atlas-events` is at fault; its matching (`GetActiveByVoyage`, scoped
to `state = ACTIVE` plus world/channel) is correct and correctly no-ops on zero
matches.

## Fix

Scope: `atlas-transports` only.

- `services/atlas-transports/atlas.com/transports/transport/model.go`
  - In `Evaluate`, additionally select the **just-arrived trip**: among this
    route's trips, the one whose arrival time-of-day is the latest at or before
    `nowTimeOfDay`, handling the midnight-crossing case the same way the
    existing in-transit selector does (a crossing trip's arrival sits on the low
    side of the zero-date axis).
  - When the derived state is `AwaitingReturn`, report that trip's `TripId` and
    `DepartedAt = materializeDeparture(now, timeOfDay(justArrived.Departure()))`
    instead of `nextTrip`'s. Verify by hand that `materializeDeparture` then
    lands on the correct day for both a same-day trip (departure before `now` →
    today) and a midnight-crossing trip (departure after `now` → yesterday).
  - If no just-arrived trip is found, keep the current `nextTrip` values — a
    behaviour-preserving fallback, not a new code path.
  - Leave `OpenEntry` / `LockedEntry` identity as-is; no voyage event reads it.
  - Update the `Transition` doc comment so `TripId`/`DepartedAt` are documented
    as "the trip this transition is about", not "the trip Evaluate selected".

- `services/atlas-transports/atlas.com/transports/transport/evaluate_test.go`
  - Add a test that ticks the *same* `Model` at the in-transit instant and again
    one tick past arrival, and asserts both ticks report the same `TripId` and
    the same `DepartedAt`. This is the assertion that actually pins the
    invariant (`VoyageId` is a pure function of that pair), so write it as an
    equality between the two ticks rather than against hard-coded values.
  - Cover both a same-day trip and a midnight-crossing trip (arrival after
    midnight, next trip later the same day).
  - Assert the arrival tick's state is `AwaitingReturn` in both, so the test
    fails loudly if the schedule shape stops reaching that branch.

Do **not** change `VoyageId`'s derivation or `voyageNamespace` — the derivation
is correct; its inputs were wrong. Do not touch `atlas-events`.

Module-local verification: `go build ./... && go test ./...` from
`services/atlas-transports/atlas.com/transports`.

## Not yet answered

Two adjacent gaps in the same state machine were noticed while diagnosing. Both
produce the same user-visible symptom (occurrence never completes) by a
different mechanism. **Neither is in scope for this fix** — recorded so they are
not rediscovered:

1. **Arrival can skip `AwaitingReturn` entirely.** If the next trip's
   `boardingOpen` is at or before the arrival instant, `Evaluate` returns
   `OpenEntry` at that tick, so the `r.State() == AwaitingReturn` block in
   `processor.go` never runs — no warp off the boat and no `VOYAGE_ARRIVED` at
   all. Confirmed reachable in a unit repro (state was `open_entry` when trips
   were spaced so boarding reopened at arrival). Not currently hit by the seeded
   boat routes: `boat-ellinia-orbis` has `travelDuration: 10`, `cycleInterval: 15`
   as seeded, but the live schedule observed a ~30-minute cycle, leaving slack.
   Whether any seeded route can hit it is **unverified**.
2. **Last trip of the day yields `OutOfService`.** When no future trip remains,
   `nextTrip == nil` and `Evaluate` returns `Transition{State: OutOfService}` —
   again bypassing the `AwaitingReturn` block, so the final voyage of a schedule
   day may never emit `VOYAGE_ARRIVED`. `ComputeSchedule` builds trips for a
   calendar day (`scheduler.go:22-24`); whether the day boundary leaves a
   trailing arrival uncovered is **unverified**.

Both are state-machine shape questions rather than the identity defect reported
here, so they need a decision before being fixed.

## Resolution

Fixed in `services/atlas-transports/atlas.com/transports/transport/model.go`.
`Evaluate` now also tracks `justArrivedTrip` — among the route's trips, the one
whose arrival time-of-day is the latest at or before `nowTimeOfDay`, mirroring
the in-transit selector's midnight-crossing handling (a crossing trip's
arrival sits on the low side of the zero-date axis; "already arrived" is the
complement of the in-transit wraparound window). Both `AwaitingReturn` return
sites now go through a `toAwaitingReturn` helper that reports
`justArrivedTrip`'s `TripId`/`DepartedAt` when one was found, falling back to
`nextTrip`'s values (the prior, buggy behaviour) otherwise — a
behaviour-preserving fallback, not a new code path.

`transport/evaluate_test.go` gained
`TestEvaluateArrivalTickReportsSameIdentityAsDepartureTick`, which ticks the
same `Model` at the in-transit instant and again one tick past arrival and
asserts equal `TripId`/`DepartedAt` (and that the arrival tick's state is
`AwaitingReturn`), for both a same-day trip and a midnight-crossing trip. It
fails against the pre-fix code (see report) and passes after the fix.
