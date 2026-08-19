# Review: bug-voyage-arrived-identity-mismatch

Commit under review: `aeb23694b` (only commit in range `d728af737..HEAD`).
Requirement: `docs/tasks/task-231-generalized-events-service/bug-voyage-arrived-identity-mismatch.md`, `## Fix` section.

## Scope

`git diff --stat d728af737..HEAD`:

```
.../bug-voyage-arrived-identity-mismatch.md        | 177 +++++++++++++++++++++
 .../transports/transport/evaluate_test.go          |  44 +++++
 .../atlas.com/transports/transport/model.go        |  51 +++++-
 3 files changed, 264 insertions(+), 8 deletions(-)
```

Matches the brief exactly: `atlas-transports` only, `model.go` + `evaluate_test.go`, no touch to `atlas-events` or `VoyageId`/`voyageNamespace`. `scope_confirmed`: reviewed the full diff of `model.go` and `evaluate_test.go`, plus `processor.go`'s `emitVoyageEvent` (read-only, to confirm the seam consumes `Transition.TripId`/`DepartedAt` and needed no change).

## Findings

### 1. `justArrivedTrip` selector — same-day and midnight-crossing — PASS

`model.go:168-220`. Two independent per-trip predicates now run alongside the existing `inTransitTrip`/`futureTrip` accumulation, and are proven disjoint from `inTransitTrip`'s window (complementary `Before`/`!Before` on the same boundary), so no trip is double-classified:

- Non-crossing trip (`model.go:208-212`): `!nowTimeOfDay.Before(arrival)` — "latest arrival at or before now" — correctly picks the most-recently-landed trip.
- Crossing trip (`model.go:193-201`): `!nowTimeOfDay.Before(arrival) && nowTimeOfDay.Before(departure)` — the region between the low-side arrival mark and the high-side departure mark, i.e. the complement of the existing wraparound in-transit window (`model.go:188`). Confirmed by hand: for a trip departing 23:30/arriving 00:30, an observation at 00:31 satisfies `!Before(00:30) && Before(23:30)` → selected, matching the brief's framing exactly.

### 2. `materializeDeparture` day projection at the arrival tick — PASS (verified by hand, both shapes)

Same-day: departure/arrival tick both use `materializeDeparture(now, departureTimeOfDay)`; at the arrival tick, `departureTimeOfDay` (e.g. 13:00) is still `<= now`'s time-of-day (13:31), so `at` is not `After(now)` and stays on today's date — same date the departure tick reported. Confirmed with the trace in `evaluate_test.go:270-285` reasoning above: both ticks land on `2026-08-15 13:00:00`.

Midnight-crossing: at the arrival tick (`now` = 00:31 on day D+1), `departureTimeOfDay` (23:30) *is* `After(now)` on D+1's date, so `materializeDeparture` subtracts 24h and lands on day D — exactly the date the departure tick (observed at 23:40 on day D) reported. Confirmed with the trace above: both ticks land on `2026-08-15 23:30:00`.

### 3. `toAwaitingReturn` fallback (no just-arrived trip found) — PASS

`model.go:249-256`: when `justArrivedTrip == nil`, `tr` is returned exactly as `to(...)` computed it from `nextTrip` — unchanged from pre-fix behaviour. This is exercised (for state/NextAt, not identity) by `TestEvaluate_Branches/Before_boarding_opens_counts_down_to_boarding_open` (single trip, no earlier arrival — `evaluate_test.go:84-91`), which still passes. No new code path; matches the brief's "behaviour-preserving fallback" requirement.

### 4. `InTransit` / `OpenEntry` / `LockedEntry` / `OutOfService` identity — PASS, unchanged

`to()` (`model.go:233-241`) is untouched and still derives identity from `nextTrip` alone; `justArrivedTrip` is only read inside the new `toAwaitingReturn` wrapper, which is only called from the two `AwaitingReturn` return sites (`model.go:273`, `model.go:283`, formerly `to(AwaitingReturn, OpenEntry, boardingOpen)`). `OpenEntry`/`LockedEntry` sites (`model.go:275`, `287`) and `OutOfService` sites (`model.go:230`, `291`) call `to()` or return the zero `Transition` directly, as before. `TestEvaluate_Branches` (`evaluate_test.go:49-170`) covers all four states' `State`/`NextState`/`NextAt` and passes unmodified.

### 5. Test honesty — PASS, empirically verified (not just trusted)

Reverted `model.go` to the pre-fix version (`git show d728af737:...model.go`) and re-ran the new test in isolation:

```
--- FAIL: TestEvaluateArrivalTickReportsSameIdentityAsDepartureTick (0.00s)
    --- FAIL: .../same-day_trip: TripId not equal; DepartedAt 2026-08-15 13:00:00 vs 2026-08-14 15:00:00
    --- FAIL: .../midnight-crossing_trip: TripId not equal; DepartedAt 2026-08-15 23:30:00 vs 2026-08-15 09:00:00
```

Both sub-tests fail against pre-fix code, on exactly the invariant the bug describes (arrival tick reporting the *next* trip's id/departure, dated wrong). Restored the fixed `model.go` (diff against working tree comes back empty) and re-ran: all of `TestEvaluate*` pass, including the new test's `AwaitingReturn`-state assertions (`evaluate_test.go:280`, `299`), which guard against the schedule shape silently missing the `AwaitingReturn` branch.

The assertion itself (`evaluate_test.go:282-284`, `301-303`) is exactly the invariant that matters — equality between the two ticks' `TripId`/`DepartedAt` — not a restatement of hard-coded values, per the brief's explicit instruction ("write it as an equality between the two ticks rather than against hard-coded values").

### 6. Doc comment update — PASS

`model.go:114` ("TripId names the trip this transition is about") matches the brief's requested wording exactly, replacing "the trip Evaluate selected."

### 7. Cross-service seam — PASS, traced by hand

`processor.go:257`: `voyageId := VoyageId(t, r.Id(), tr.TripId, tr.DepartedAt)`, and `processor.go:267`: `DepartedAt: tr.DepartedAt` in the emitted `VoyageStatusEventBody`. Both come straight from the `Transition` `Evaluate` returns; `emitVoyageEvent`/`processor.go` were not touched by this commit and needed no change — the fix at the `Evaluate` layer is sufficient to repair the `VOYAGE_ARRIVED` payload. `atlas-events`' `crimsonbalrog.ArrivalProcessorImpl.OnVoyageArrived` was correctly left untouched per the brief ("Do not touch atlas-events"); no new consumer-side test was required by the brief and none was added — consistent with root cause being entirely producer-side.

### 8. Build/test — PASS

`go build ./... && go test ./...` from `services/atlas-transports/atlas.com/transports` is green (all packages `ok`, including `transport` with the new test).

Note: `git status` in the worktree shows unrelated pre-existing modifications (`deploy/k8s/overlays/pr-sparse/patches/consumer-group-env.yaml`, `go.work.sum`) that are not part of commit `aeb23694b` and outside this review's scope.

## Not evaluable

None — the full fix surface (model.go, evaluate_test.go) and its only real consumer (processor.go's read of `Transition`) were reviewed and hand-verified, including empirical pre-fix/post-fix test execution.

## Verdict

APPROVED. No blocking findings. No non-blocking findings either — the fix matches the brief requirement-by-requirement, the day-projection arithmetic checks out by hand for both same-day and midnight-crossing shapes, the fallback path is genuinely unchanged, the four untouched RouteState paths are verified unchanged, and the new test was empirically confirmed to fail pre-fix and pass post-fix while asserting the invariant that matters.
