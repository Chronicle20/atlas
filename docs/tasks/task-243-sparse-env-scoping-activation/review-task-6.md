# Review: Task 6 — atlas-channel readiness component

Range: `d24377d11..e96378598` (single commit in range: `e96378598`)
Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`

## Scope

`git show --stat e96378598` — 3 files changed, 65 insertions, 0 deletions:

- `services/atlas-channel/atlas.com/channel/configuration/projection/state.go` (+12)
- `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` (+50)
- `services/atlas-channel/atlas.com/channel/main.go` (+3)

No `deploy/k8s/` files touched — clean of Task 7's concurrent surface.
`git log --oneline d24377d11..e96378598` shows exactly one commit in range,
matching the brief. Scope confirmed.

## Requirement 1 — mirrors Task 5's shape (the item Task 5's review left
not-evaluable)

Diffed `git show d24377d11` (Task 5, atlas-login) against `git show
e96378598` (Task 6, atlas-channel) directly, file by file. Outside of
package paths, commit metadata, the two derived service UUIDs, and the
`subscriber.go` line-number reference in the doc comment (111-113 in
atlas-login vs. 112 in atlas-channel — both correct for their own file,
verified below), the diffs are **line-for-line identical**:

- `state.go`: same `HasService() bool` body (`s.mu.RLock/RUnlock`; `return
  s.service != nil`), same doc-comment structure, inserted at the same
  point (between `ApplyTenantTombstone` and `Snapshot`).
- `main.go`: same `service.WithReadinessGate(state.HasService)` line with
  the same FR-1.5/D6 comment, inserted immediately after the existing
  `WithReadinessGate(caughtUp.CaughtUpNow)` call, before
  `WithEnvironmentRegistry`.
- `projection_test.go`: the same four test cases, same names, same
  assertions, same structure — only the id literal and (in the tenant
  case) the already-consistent `tenant.RestModel` field set change.

**Verdict: clean mirror, no divergence.** This closes the item Task 5's
review left open.

Cross-checked the `subscriber.go:112` reference against the actual file:
`atlas-channel/configuration/projection/subscriber.go:112` is
`if env.Id != s.ServiceId.String() {` — matches the comment exactly (`file:
services/atlas-channel/atlas.com/channel/configuration/projection/subscriber.go:112`).

Cross-checked the derived channel-service UUID
(`5a86d8e6-3167-5e74-9fc5-021d94001da2`) against
`tools/derive-service-id_test.sh:34`:
`assert_eq "channel-service pr-1411" "5a86d8e6-3167-5e74-9fc5-021d94001da2" ...`
— pinned and consistent with the brief's instruction.

## Requirement 2 — `State.HasService` structural correctness

`state.go:16-19` — `State.service *configuration.RestModel`, same nil-until-
first-`ApplyService` contract as atlas-login's `State`. `ApplyService`
(`state.go:28-40`) sets `s.service = &cfg` under `s.mu.Lock()`;
`ApplyServiceTombstone` (`state.go:47-51`) resets it to `nil`. `HasService`
(`state.go:79-83`) reads under `s.mu.RLock()` and returns `s.service != nil`.
Consistent, race-safe (RWMutex already used by every other accessor).

## Requirement 3 — test honesty (mutation check, not just static reading)

Ran the new tests, then mutated `HasService`'s body
(`s.service != nil` → `s.service == nil`) and re-ran:

```
--- FAIL: TestHasServiceIsTrueAfterTheMatchingServiceIsApplied
--- FAIL: TestHasServiceIsFalseAgainAfterATombstone
--- FAIL: TestHasServiceIsFalseAfterOnlyATenantIsApplied
```

3 of 4 cases fail under the mutation (the 4th, "false before any service is
applied," is trivially satisfied by either polarity on a freshly-constructed
`State` with `service == nil`, same as in Task 5 — not a gap, just an
unavoidable base case). Reverted the mutation immediately after
(`git checkout -- services/atlas-channel/.../state.go`); `git status
--porcelain services/atlas-channel` confirmed clean afterward. Tests are not
trivially-passing.

Also ran the suite unmutated:

```
cd services/atlas-channel/atlas.com/channel && go build ./...   # clean
go test ./configuration/... -count=1                             # ok
```

## Requirement 4 — readiness gate cannot deadlock atlas-channel's startup

Re-traced atlas-channel's own `main.go`, not inherited from Task 5:

- `main.go:179-200` — `state := projection.NewState()` and `caughtUp :=
  projection.NewCaughtUp()` are constructed before `service.Bootstrap`.
  `service.Bootstrap` is called with `WithConfigProjection` (which wires the
  subscriber referencing `state`/`caughtUp`), then two
  `WithReadinessGate` calls (`caughtUp.CaughtUpNow`, `state.HasService`),
  then `WithEnvironmentRegistry`. All of `main()`'s subsequent setup
  (Redis connect, consumer registration, REST server, etc. through at least
  line 260+) proceeds unconditionally after `Bootstrap` returns.
- Read `libs/atlas-service/bootstrap.go:51-89` (`Bootstrap`) and
  `:100-112` (`Ready`): `WithReadinessGate` only appends `fn` to
  `cfg.gates` / `rt.gates` (`bootstrap.go:32-34`). `Bootstrap` never calls
  any gate during startup — it returns `rt` unconditionally
  (`bootstrap.go:88`). `Ready()` is a pure poll function invoked
  out-of-band by the `/readyz` handler; it is not called anywhere in the
  hot startup path. There is no blocking wait on `HasService` (or on
  `CaughtUpNow`) inside `main()`.
- Therefore: a pod whose SERVICE_ID row never arrives keeps running and
  keeps trying (the subscriber's Kafka consumption is unaffected by
  readiness state) but simply never flips `/readyz` to 200 — it cannot wedge
  `main()`'s init sequence, which is the failure mode task-243 is designed
  to prevent, not the failure mode being newly introduced. Confirmed by
  reading the lib directly for atlas-channel's own call path, not assumed
  from Task 5's login trace. `libs/atlas-service` is unmodified in this
  commit — its correctness was in-scope only as a dependency the change
  must not misuse, and it is not misused here (same as Task 5's finding).

## Non-blocking notes

- None beyond what's already covered by "PASS" items above.

## Not evaluable

- None. All items in the brief and the extra checks requested (mirror-shape
  diff, test-mutation honesty, atlas-channel-specific deadlock trace) were
  directly evaluable within this commit's diff plus the shared
  `libs/atlas-service` contract it depends on.

## Summary

Task 6 is a faithful, line-for-line mirror of Task 5's already-approved
atlas-login change, adapted only for atlas-channel's package paths, its own
pinned service UUID, and its own (correct) subscriber.go line reference.
Tests are non-trivial (mutation-confirmed). The readiness gate is
non-blocking by construction in the shared `atlas-service` bootstrap lib and
cannot deadlock atlas-channel's startup, independently re-traced through
atlas-channel's own `main.go`.
