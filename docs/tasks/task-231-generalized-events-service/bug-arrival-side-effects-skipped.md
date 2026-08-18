# bug: voyage arrival side effects are skipped when the route does not land in AwaitingReturn

Task: task-231-generalized-events-service
Follow-up to `bug-voyage-arrived-identity-mismatch.md` (fixed in `aeb23694b`).
Both gaps below were recorded there under `## Not yet answered`; the user has
since asked for both to be fixed.

## Reproduced

Gap 1 yes, gap 2 by inspection only — see each below.

## Observed

Two distinct paths where a voyage arrives but none of the arrival side effects
run: players are **not** warped off the en-route map, and **no**
`VOYAGE_ARRIVED` is emitted. A CRIMSON_BALROG occurrence for that voyage stays
`ACTIVE` forever — the same user-visible symptom as the identity bug, reached a
different way.

**Gap 1 — arrival lands in `OpenEntry`.** If the next trip's `boardingOpen` is
at or before the arrival instant, `Evaluate` returns `OpenEntry` at the arrival
tick rather than `AwaitingReturn`. Reproduced during the identity-bug
investigation: with trip A (boarding 12:00, departs 13:00, arrives 13:30) and
trip B (boarding 13:00, departs 14:00), the tick at 13:31 returned
`state=open_entry`. Not currently hit by the seeded boat routes — the live
Ellinia→Orbis schedule ran a ~30-minute cycle against a 10-minute travel
duration, leaving slack — but nothing in the model prevents it, and
`boat-ellinia-orbis.json` as seeded (`travelDuration: 10`, `cycleInterval: 15`)
has far less slack than what was observed live.

**Gap 2 — arrival lands in `OutOfService`.** When no future trip remains,
`Evaluate` returns early at `model.go:229-231` (`nextTrip == nil` →
`Transition{State: OutOfService}`), and the same happens at the final
`return Transition{State: OutOfService}` past the last arrival. Either way the
route leaves `InTransit` without passing through `AwaitingReturn`. Not
reproduced against a live schedule; `ComputeSchedule` (`scheduler.go:22-24`)
builds trips for one calendar day, so whether the day boundary leaves a
trailing arrival uncovered is still **unverified**. Fix it regardless — the
model permits it and the failure is silent.

## Expected

Arrival side effects run whenever a voyage actually ends, regardless of which
state the route lands in afterwards, and the `VOYAGE_ARRIVED` they emit carries
the arrived voyage's identity.

## Root cause

One root cause with two surfaces: **arrival is an event (a boundary crossing)
but the code infers it from a state.** `processor.go:188` gates the whole
arrival block on `r.State() == AwaitingReturn`, so any tick where the route
leaves `InTransit` for some other state loses the arrival entirely.

The identity fix in `aeb23694b` has the same shape problem. It parks the
arrived-trip identity inside the `AwaitingReturn` branch of `Evaluate`
(`toAwaitingReturn`, `model.go:244-258`), so the correct identity is only
available in exactly the state that gap 1 and gap 2 fail to reach. That
placement was right for the bug it fixed and is wrong for this one.

## Fix

Scope: `atlas-transports` only. Part of this deliberately reworks `aeb23694b` —
that commit is not being reverted for being wrong, it is being generalized,
because arrival identity must be available in states other than
`AwaitingReturn`.

- `services/atlas-transports/atlas.com/transports/transport/model.go`
  - Add two fields to `Transition`: `ArrivedTripId uuid.UUID` and
    `ArrivedDepartedAt time.Time` — the identity of the trip that has just
    landed (`justArrivedTrip`), or zero when there is none. Document them as
    the arrival counterpart to `TripId`/`DepartedAt`: `TripId` names the trip
    the transition's *state* is about, `ArrivedTripId` names the voyage that
    just *ended*, and the two are independent.
  - Populate both from `justArrivedTrip` on **every** return path of
    `Evaluate`, independent of state — including the early
    `nextTrip == nil` → `OutOfService` return at `model.go:229-231` and the
    final `OutOfService` return. `ArrivedDepartedAt` uses the same
    `materializeDeparture(now, timeOfDay(justArrivedTrip.Departure()))` the
    current `toAwaitingReturn` uses. Note the early return currently fires
    *before* `justArrivedTrip` is consulted at all, so it needs the selection
    loop's result to be reachable there — do not move the selection loop below
    it.
  - Remove `toAwaitingReturn` and restore both `AwaitingReturn` return sites to
    plain `to(AwaitingReturn, OpenEntry, boardingOpen)`. `TripId`/`DepartedAt`
    go back to uniformly meaning "the trip `Evaluate` selected" (`nextTrip`),
    which is what `rest.go:152` and the departure emission already assume.
    Keeping both mechanisms would leave two fields carrying the same value in
    one state, which will drift.

- `services/atlas-transports/atlas.com/transports/transport/processor.go`
  - Change the arrival gate at `:188` from `r.State() == AwaitingReturn` to
    the union: the route **left** `InTransit` (`route.State() == InTransit`,
    the pre-update model already in scope at `:175`) **or** it landed in
    `AwaitingReturn` (`r.State() == AwaitingReturn`). Keep both disjuncts. The
    second is not redundant: on a restart mid-voyage the registry's prior state
    is not `InTransit`, and the `AwaitingReturn` arm is what still emits the
    arrival then. A duplicate `VOYAGE_ARRIVED` is harmless — `atlas-events`
    completes via a guarded `UPDATE` that no-ops for the loser, and the warp
    loop is a no-op when the en-route map is empty.
  - `emitVoyageEvent` (`:254`) must use `tr.ArrivedTripId` /
    `tr.ArrivedDepartedAt` for the **arrival** event and keep `tr.TripId` /
    `tr.DepartedAt` for the **departure** event. The cleanest split is to stop
    passing the provider into one shared method and give arrival and departure
    their own identity source; do not thread a boolean flag through it.
  - The body's `DepartedAt` field must carry the arrived voyage's departure
    instant for `VOYAGE_ARRIVED` (it is the same value that feeds `VoyageId`,
    and `atlas-events` reads it).
  - If `ArrivedTripId` is the zero uuid on an arrival path, log an error and
    skip **only** the `VOYAGE_ARRIVED` emission — still run the warp. A wrong
    voyage id is worse than a missing event, but players must never be left on
    a phantom boat.
  - Leave the `OpenEntry` / `LockedEntry` / `InTransit` chain at `:206-232`
    alone. It is a separate `if`, so the unified arrival block composes with it
    — when a tick is both an arrival and a boarding opening, both must run.

- `services/atlas-transports/atlas.com/transports/transport/evaluate_test.go`
  - Rewrite `TestEvaluateArrivalTickReportsSameIdentityAsDepartureTick` to
    assert the arrival tick's `ArrivedTripId`/`ArrivedDepartedAt` equal the
    departure tick's `TripId`/`DepartedAt`. Keep both existing subtests
    (same-day, midnight-crossing) and keep the equality-between-two-ticks
    framing — that is the invariant `VoyageId` depends on. Drop the assertion
    that the arrival tick's state is `AwaitingReturn`; that is no longer the
    property under test.
  - Add subtests for the two skip shapes, asserting the same identity equality
    holds: one where the arrival tick evaluates to `OpenEntry` (next trip's
    boarding already open — the gap-1 repro shape above), and one where it
    evaluates to `OutOfService` (no future trip). Assert the state in each, so
    the test fails loudly if the schedule shape stops reaching that branch.

- `services/atlas-transports/atlas.com/transports/transport/processor_test.go`
  - There is an existing harness with channel/map/character mocks
    (`TestArrivalEmitsVoyageArrivedPerChannel` at `:624`,
    `TestArrivalAcrossMidnightUsesDepartureDayVoyageId` at `:681`) — extend it,
    do not build a second one.
  - Add a test that the arrival warp **and** one `VOYAGE_ARRIVED` per channel
    are emitted when the route goes `InTransit` → `OpenEntry`, and another for
    `InTransit` → `OutOfService`.
  - Add a test that the emitted `VOYAGE_ARRIVED`'s `voyageId` equals the
    `voyageId` of the `VOYAGE_DEPARTED` emitted for the same trip — asserted
    end-to-end through the processor, not just through `Evaluate`. This is the
    assertion that would have caught the original live bug.

Do not change `VoyageId`'s derivation or `voyageNamespace`. Do not touch
`atlas-events`. Do not alter the state machine's state assignments — no making
`AwaitingReturn` win over `OpenEntry`; that would delay legitimate boarding.

Module-local verification: `go build ./... && go test ./...` from
`services/atlas-transports/atlas.com/transports`.

## Not yet answered

- Whether any seeded route's schedule can actually produce gap 1 or gap 2 in
  production is still unverified — the fix makes both harmless rather than
  proving they occur. Worth a follow-up pass over the seeded route timings
  (`deploy/seed/shared/all/routes/*.json`) against `ComputeSchedule`'s
  day-boundary behaviour if the question matters beyond robustness.

## Resolution

_Pending._
