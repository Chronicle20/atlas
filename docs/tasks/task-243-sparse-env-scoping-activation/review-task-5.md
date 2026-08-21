# Review — Task 5: atlas-login readiness component (FR-1.5 / D6)

Commit range: `4f5742892..d24377d11`
Brief: `.superpowers/sdd/plan/task-5-brief.md`
Report: `.superpowers/sdd/plan/task-5-report.md`

## Scope

`git diff --stat 4f5742892..d24377d11` touches exactly the three files the
brief names:

- `services/atlas-login/atlas.com/login/configuration/projection/state.go` (+12)
- `services/atlas-login/atlas.com/login/configuration/projection/projection_test.go` (+50)
- `services/atlas-login/atlas.com/login/main.go` (+3)

No other files changed in this range. Scope matches the brief. Reviewed the
full diff plus `libs/atlas-service/bootstrap.go` (`WithReadinessGate`,
`Runtime.Ready`, `Bootstrap`) since correctness of the deadlock question
depends on that contract, and `subscriber.go`'s foreign-row filter referenced
by the new doc comment/test comment.

## Findings

### 1. `HasService` implementation — PASS

`state.go:78-88` (post-diff) matches the brief's exact contract: RLock, return
`s.service != nil`. `s.service` is set in `ApplyService` (`state.go:36`) and
cleared in `ApplyServiceTombstone` (`state.go:51`), both under `s.mu.Lock()`.
Same RWMutex discipline as `Snapshot`. No new state, no new locking primitive.

### 2. `main.go` wiring — PASS

`main.go:69-71`: `service.WithReadinessGate(state.HasService)` added
immediately after the existing `service.WithReadinessGate(caughtUp.CaughtUpNow)`,
with a one-line FR-1.5/D6 comment. `state := projection.NewState()` is
declared at `main.go:53`, before `service.Bootstrap` is called, so `state` is
in scope without any closure restructuring — confirmed by reading the file
rather than trusting the report.

### 3. Deadlock check — PASS, no deadlock possible

Traced `WithReadinessGate` (`libs/atlas-service/bootstrap.go:32`) and
`Runtime.Ready` (`bootstrap.go:102-110`): `Ready()` is a **non-blocking**
poll — it iterates `r.gates` and calls each `func() bool` synchronously, no
channel, no wait. `state.HasService` itself takes an `RLock`, checks a
pointer, returns — it cannot block on anything upstream. If the service row
never arrives, `HasService()` returns `false` forever and `/readyz` simply
stays 503; nothing in the startup path (`main.go`) blocks on `Ready()` — the
only blocking call is `rt.AwaitProjectionCatchUp()` (`main.go:87`), which is
gated on Kafka catch-up (an existing, unrelated mechanism), not on
`HasService`. A pod whose row never arrives boots fully, serves no traffic
readiness-wise, and does not wedge startup. This matches the brief's intent
(a permanently non-Ready pod, not a hung one).

### 4. Test honesty — PASS, none are trivially-passing

Mentally mutated the implementation for each test:

- `TestHasServiceIsFalseBeforeAnyServiceIsApplied`: if `HasService` returned
  `true` unconditionally (or checked the wrong field), this fails —
  `NewState()` never calls `ApplyService`.
- `TestHasServiceIsTrueAfterTheMatchingServiceIsApplied`: if `HasService`
  always returned `false`, or if `ApplyService` were changed to not set
  `s.service`, this fails.
- `TestHasServiceIsFalseAgainAfterATombstone`: if `ApplyServiceTombstone`
  did not clear `s.service` (e.g. a no-op stub), this fails — the test
  asserts `true` first, then `false` after the tombstone, so a `HasService`
  that ignores tombstones is caught.
- `TestHasServiceIsFalseAfterOnlyATenantIsApplied`: if `HasService`
  accidentally checked `len(s.tenants) > 0` instead of `s.service != nil`
  (a plausible copy-paste bug given the sibling tenant map), this fails.

Ran the targeted suite directly rather than trusting the report:
`go test ./configuration/projection/... -run TestHasService -v` → all 4 pass.
Ran full module build + test suite (`go build ./... && go test ./...`) →
clean, no failures, output matches report.

The "different service's row" case is correctly *not* re-tested here — the
brief and the test-file comment both point at `subscriber.go:111-113`
(`env.Id != s.ServiceId.String()` early return), which is the actual
enforcement point; a foreign row never reaches `State.ApplyService`. This is
a legitimate scope decision, not a dropped case, since `HasService` itself
has no way to distinguish "own service" from "foreign service" — that
filtering is entirely the subscriber's job, one layer up, outside this
diff's surface.

### 5. Mirror-cleanliness for Task 6 (atlas-channel) — PASS, no red flags

- `HasService`'s doc comment references `subscriber.go:111`, a path that
  will differ in atlas-channel only in module prefix, not shape — the
  subscriber pattern (`env.Id != s.ServiceId.String()` early return) is
  named generically enough in the comment to mirror without editing the
  claim (still needs the mirrored subscriber to exist and behave the same
  way, which is Task 6's job to verify independently, not this review's).
- The `main.go` insertion point is keyed off the existing
  `service.WithReadinessGate(caughtUp.CaughtUpNow)` call, a pattern task 6's
  brief presumably also names — nothing atlas-login-specific leaked into the
  gate registration line itself.
- Test file structure (fixed UUID literal, `ApplyService`/`ApplyTenant`
  envelope construction reusing existing helpers) is generic and requires no
  atlas-login-specific machinery beyond what `atlas-channel`'s equivalent
  `projection` package should already have, per the brief's design.

No shape issue found that would fail to mirror cleanly.

### 6. Repo conventions — PASS

- No new domain type/constant introduced (nothing to check against
  `libs/atlas-constants`).
- No `*_testhelpers.go` file created; test reuses existing `projection`
  package constructors (`NewState`, envelope struct literals) as the brief
  directed.
- Comment density and doc-comment style consistent with the file's existing
  functions (`ApplyServiceTombstone`, `Snapshot`).

## Not evaluable

- Runtime behavior of `/readyz` actually flipping a pod's k8s-visible status
  is Task 7's manifest half, explicitly out of scope per the assignment.
- Whether Task 6's atlas-channel mirror will in fact reuse this shape
  cleanly can only be confirmed once Task 6 lands; this review only checked
  that nothing in Task 5's diff would obstruct a clean mirror.

## Verdict

APPROVED. Implementation matches the brief exactly, the readiness gate is
non-blocking so it cannot deadlock startup, and all four new tests are
non-trivial — each is falsified by kicking out from under it the single
transition it is asserting.
