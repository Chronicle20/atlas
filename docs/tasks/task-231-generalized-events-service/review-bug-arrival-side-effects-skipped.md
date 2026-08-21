# Review: bug-arrival-side-effects-skipped — commit `77b87db91`

Reviewer: atlas-reviewer (Sonnet)
Scope: commit `77b87db91` (`fix(atlas-transports): run arrival side effects
regardless of landing state`) against `## Fix` in
`docs/tasks/task-231-generalized-events-service/bug-arrival-side-effects-skipped.md`.
Background reworked: `aeb23694b` (`bug-voyage-arrived-identity-mismatch.md`).

## Scope confirmed

Diff touches exactly the five files the brief scopes: `model.go`,
`processor.go`, `evaluate_test.go`, `processor_test.go`, and the bug doc
itself. `atlas-events` is untouched (`git diff --stat 77b87db91~1 77b87db91`
shows no `services/atlas-events` entry) — matches "Do not touch
`atlas-events`". No scope mismatch.

## Findings

### 1. `model.go` — `ArrivedTripId`/`ArrivedDepartedAt` on every `Evaluate` return path — PASS

Read the full function (`model.go:174-310`). Six return statements:

- `model.go:252-258` — early `nextTrip == nil` → `OutOfService`. Sets
  `ArrivedTripId`/`ArrivedDepartedAt` from `arrivedTripId`/`arrivedDepartedAt`,
  which are computed at `model.go:245-250` **before** this return, i.e. the
  selection loop's result is reachable here as the brief required ("do not
  move the selection loop below it" — confirmed not moved; the loop at
  `model.go:187-230` still precedes the early return).
- `model.go:260-270` — the `to(...)` closure, used by all four
  state-transition returns (`AwaitingReturn`×2, `OpenEntry`×1 via
  `LockedEntry`, `InTransit`×2, midnight-crossing variants). Sets both fields
  from the same closed-over `arrivedTripId`/`arrivedDepartedAt`.
- `model.go:305-309` — the final `OutOfService` return (past the last
  scheduled arrival). Sets both fields.

Every path is covered; none is missed. `ArrivedDepartedAt` uses
`materializeDeparture(now, timeOfDay(justArrivedTrip.Departure()))`, matching
the fix's spec. `toAwaitingReturn` was removed and both `AwaitingReturn` call
sites now call plain `to(AwaitingReturn, OpenEntry, boardingOpen)`
(`model.go:287`, `model.go:297`) — `TripId`/`DepartedAt` are uniformly
`nextTrip`'s again, matching the brief.

### 2. Pre-existing fixture edit (`inTransitRouteAboutToArrive`) — claim verified independently

Traced `Evaluate` by hand against the **old** fixture (single trip: boards
12:30, closes 12:35, departs 12:37, arrives 12:47; `now` = 12:00, i.e.
*before* the trip's own boarding window, let alone its arrival):

- `justArrivedTrip` selection requires `!nowTimeOfDay.Before(tripArrival)`;
  at `now`=12:00 vs `arrival`=12:47 this is false, so `justArrivedTrip` stays
  `nil`.
- `nextTrip` = `futureTrip` = the same (only) trip.
- Evaluate lands in `AwaitingReturn` (`nowTimeOfDay.Before(boardingOpen)` at
  12:00 < 12:30) via `toAwaitingReturn`, whose `justArrivedTrip != nil` guard
  is false, so it falls back to `nextTrip`'s identity — which is the same
  trip as if `justArrivedTrip` had been picked, purely because there is only
  one trip on the schedule.

This independently confirms the implementer's claim: the old fixture never
exercised the `justArrivedTrip` branch; it passed only via the fallback,
coincidentally matching because there was nothing else to disagree with.

Verified the **new** fixture (`processor_test.go`, two trips, `now` =
13:31) by hand too: at 13:31, trip A (arrives 13:30) satisfies
`!nowTimeOfDay.Before(arrival)` → `justArrivedTrip` = trip A. Trip B's
boarding (14:00) is not yet open, so `nextTrip` = trip B (`futureTrip`), and
`Evaluate` returns `AwaitingReturn` counting down to trip B's boarding, with
`TripId` = trip B and `ArrivedTripId` = trip A — genuinely two different
trips this time, exactly what the fixture comment claims.

**Regression check on the two pre-existing tests that reuse this fixture**
(`TestArrivalEmitsVoyageArrivedPerChannel`, unchanged assertions):
still asserts 2 events (one per channel), non-nil shared `VoyageId`, matching
`DestinationMapId`. None of these assertions depended on the specific trip
selected, so widening the fixture to two trips does not weaken what this test
was checking — it strengthens it, because the event now flows through the
genuinely-populated `ArrivedTripId` path rather than the coincidental
fallback. Confirmed empirically in §5 below that this test still passes.

`TestArrivalAcrossMidnightUsesDepartureDayVoyageId` uses its own
single-trip midnight-crossing fixture (untouched by this commit) and asserts
an exact `VoyageId`/`DepartedAt` value. Hand-traced: at `now`=00:40 the
following day, `justArrivedTrip` = the one trip (arrival 00:30 satisfies
`!now.Before(arrival)`), so `ArrivedTripId`/`ArrivedDepartedAt` equal
`nextTrip`'s values in this single-trip fixture too — the test now validates
the same exact-value invariant through the new `ArrivedTripId` code path
(since `processor.go` now emits `VOYAGE_ARRIVED` from `tr.ArrivedTripId`, not
`tr.TripId`), so this test is not weakened; if anything it now covers
midnight-crossing `ArrivedDepartedAt` derivation that it previously covered
only for `DepartedAt`.

### 3. `route.State() == InTransit || r.State() == AwaitingReturn` gate — PASS, no double/spurious fire in the normal case

Read `processor.go:140-238`. The whole arrival block sits inside
`if changed {`, gated on `UpdateStateWithTransition` reporting a state
change this tick, so it can only run at most once per state transition.

- `route` (the function's pre-update parameter, in scope since `route.go`
  parameter) is the state *before* this tick's evaluation; `r` is the state
  *after*. On the ordinary arrival tick: `route.State() == InTransit` is true
  (pre-update), so the block fires once regardless of what `r.State()` lands
  in (`AwaitingReturn`, `OpenEntry`, or `OutOfService` — all three now
  covered, per Finding 1).
- On the *next* tick after an ordinary arrival (e.g. `AwaitingReturn` →
  `OpenEntry`), `route.State()` is `AwaitingReturn` (not `InTransit`) and
  `r.State()` is `OpenEntry` (not `AwaitingReturn`) — gate is false, no
  re-fire. No double-fire in the steady-state case.
- The `r.State() == AwaitingReturn` disjunct is redundant with
  `route.State() == InTransit` in the steady-state case and only fires
  independently on a restart where the registry's prior in-memory `route`
  state is not `InTransit` (this restart-recovery behavior is pre-existing —
  the old code already gated solely on `r.State() == AwaitingReturn`, so this
  commit does not introduce new restart risk; it is unchanged from before).
  The brief explicitly calls a duplicate emission on that seam "harmless" and
  does not ask this commit to fix restart-recovery semantics; verifying that
  claim (idempotent consumer-side `UPDATE`) is outside this diff's surface
  (`atlas-events`, untouched) and is asserted, not re-verified, in the brief
  — noted as not independently re-verified here since it is out of scope for
  this producer-side diff.

### 4. Departure path / `rest.go:152` — PASS, unaffected

`processor.go:234` still calls `emitVoyageEvent(mb)(r, tr.TripId,
tr.DepartedAt, VoyageDepartedStatusEventProvider)` — same fields as before,
now passed explicitly rather than via the removed shared `Transition`
parameter. `rest.go:152` reads `transition.TripId`/`transition.DepartedAt`
directly from a fresh `Evaluate` call and was not touched; since `TripId`/
`DepartedAt` reverted to uniformly meaning `nextTrip` (Finding 1), `rest.go`'s
reading is unchanged in meaning from both before `aeb23694b` and after this
commit — `toAwaitingReturn`'s removal does not regress it, matching the
brief's explicit call-out.

`emitVoyageEvent`'s signature change (`processor.go:267`) splits the caller
into passing `tripId`/`departedAt` explicitly instead of a shared
`Transition`, exactly as specified ("do not thread a boolean flag through
it").

The zero-uuid guard (`processor.go:208-213`) logs and skips only the
`VOYAGE_ARRIVED` emission while the warp loop above it (`processor.go:194-203`)
still runs unconditionally — matches "still run the warp" / "players must
never be left on a phantom boat".

### 5. Do the new tests fail without the production change? — Verified empirically

Built a throwaway worktree at the parent commit (`aeb23694b`, pre-`77b87db91`)
and copied in this commit's test files to isolate the claim:

- Copying **all** new test files onto `aeb23694b`'s production code fails to
  build (`evaluate_test.go` references `Transition.ArrivedTripId`/
  `ArrivedDepartedAt`, which don't exist pre-fix) — confirms the model-level
  test cannot even compile, let alone pass, without the `model.go` change.
- Isolated further: copied only the new `model.go` (field population) onto
  `aeb23694b`, keeping `processor.go`'s old `r.State() == AwaitingReturn`
  gate and old `emitVoyageEvent(r, tr, provider)` signature. Result
  (`go test ./transport/... -run ...`):
  - `TestEvaluateArrivalTickReportsSameIdentityAsDepartureTick` (all 4
    subtests, including the two new skip-shape subtests) — **PASS**. This is
    expected: this test only exercises `Evaluate`, which the isolated
    `model.go` change alone fixes.
  - `TestArrivalIntoOpenEntryWarpsAndEmitsVoyageArrivedPerChannel` —
    **FAIL**: `expected the arrival warp to move 1 character, got 0`
    (`processor_test.go:849`).
  - `TestArrivalIntoOutOfServiceWarpsAndEmitsVoyageArrivedPerChannel` —
    **FAIL**: same shape, `processor_test.go:877`.
  - `TestArrivalVoyageIdMatchesDepartureVoyageIdEndToEnd` — **FAIL**:
    `"[]" should have 1 item(s), but has 0` (`processor_test.go:931`) — zero
    `VOYAGE_ARRIVED` messages emitted.

  This isolates the claim precisely: the three processor-level tests fail
  specifically because of the `processor.go` gate change (Finding 3), not
  because of the `model.go` change, and only pass once both pieces of the
  fix are in place together.
- Against the full commit (`go build ./... && go test ./...` at
  `services/atlas-transports/atlas.com/transports`) — all packages **PASS**,
  including `transport` (all existing + new tests).

## Not evaluable

- Restart-recovery double-fire behavior (the `r.State() == AwaitingReturn`
  disjunct's actual necessity, and whether the guarded `UPDATE` on the
  `atlas-events` consumer is genuinely idempotent) is asserted by the code
  comment and the bug doc but not independently re-verified here — it
  pre-dates this commit (the old gate was `r.State() == AwaitingReturn`
  alone) and the consumer side is `atlas-events`, which this diff does not
  touch and which is explicitly out of scope per the brief ("Do not touch
  `atlas-events`"). Flagging as not evaluated within this diff's surface,
  not as a defect.
- Whether any seeded route schedule can actually reach gap 1 or gap 2 in
  production remains unverified, per the bug doc's own "Not yet answered"
  section — this is explicitly deferred by the brief itself, not a gap in
  this review.

## Verdict rationale

All five review-brief focus areas check out: the `Transition` fields are
populated on every `Evaluate` return path (including both the early and
final `OutOfService` returns), the fixture-change claim is independently
confirmed correct and does not weaken any pre-existing assertion, the arrival
gate fires exactly once in the normal case with no new double-fire risk, the
departure path and `rest.go:152` are unaffected, and the new tests are proven
(empirically, via a parent-commit rebuild) to fail without this commit's
`processor.go` change. `go build ./... && go test ./...` passes clean at
HEAD. No blocking findings.
