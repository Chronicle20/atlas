# Review: Task 3 — Gate drop-reason distinguishability

Range reviewed: `15910ad9e..2072f881e` (`.superpowers/sdd/plan/review-15910ad9e..2072f881e.diff`)
Brief: `.superpowers/sdd/plan/task-3-brief.md`
Report: `.superpowers/sdd/plan/task-3-report.md`

## Scope

`git diff --stat 15910ad9e..2072f881e`:

```
libs/atlas-kafka/consumer/gate.go      |  31 +++++--
libs/atlas-kafka/consumer/gate_test.go | 148 ++++++++++++++++++++++++++++++---
libs/atlas-kafka/consumer/manager.go   |   9 +-
3 files changed, 165 insertions(+), 23 deletions(-)
```

Exactly the three files the brief named, at the call site the brief named
(`manager.go:628-641`). No scope drift.

## Requirement-by-requirement

1. **`gateReason` type + six constants** (`gate.go:19-32`) — present verbatim,
   same FR/design comments as the brief's Step 2 snippet. PASS.

2. **`decide` returns `(gateVerdict, gateReason)`, no condition/ordering/verdict
   change** (`gate.go:67-86`) — diffed against the brief's exact snippet;
   every `return` gained only a second value, no `if` was touched. Confirmed
   by reading the diff hunk directly (`git diff 15910ad9e..2072f881e --
   gate.go`): each condition line is unchanged, only the `return` statements
   grew a second value. PASS.

3. **`gateDroppedUnresolvable` widened to `{"service","environment","reason"}`,
   `gateProcessed`/`gateSkippedNotOwner` left at two labels**
   (`gate.go:34-58`) — confirmed. PASS.

4. **Cardinality bound** — only `gateDropUnresolvable`'s three arms
   (`reasonMismatched`, `reasonStale`, `reasonNotActive`) ever reach the
   3-label counter (`manager.go:631-634`); all three come from the closed
   `gateReason` enum, never a formatted string carrying tenant/env/topic
   data. No unbounded-cardinality risk. PASS.

5. **Call site** (`manager.go:628-641`) — `verdict, reason := decide(...)`,
   `switch verdict`, `string(reason)` as the third label value on the drop
   arm, `.WithField("reason", ...)` added to the existing `Error` log
   (message text unchanged, matching the brief's instruction), and the
   previously-silent `gateSkipNotOwner` arm now logs at `Debug` with
   `environment`, `topic`, `reason` fields before `return true`. Matches the
   brief's snippet exactly, including the Debug-not-Info level rationale.
   PASS.

6. **`libs/atlas-constants/` reuse check** — swept `libs/atlas-constants/`
   for any existing reason/label constant; its subpackages are game-domain
   (asset, channel, character, item, job, map, monster, ...), nothing
   overlapping a gate/consumer drop-reason concept. `gateReason` is a
   legitimately new, package-local type. PASS.

7. **FR-4.6 regression, by test not inspection** — brief required
   `TestGateWithNoRecordsProjectedIsUnchanged` with a registry that never
   received `r.Apply`. Present at `gate_test.go:213-222`, asserting both
   `(gateProcess, reasonLegacy)` for `msgEnv=""` and
   `(gateDropUnresolvable, reasonNotActive)` for `msgEnv="pr-999"`. Verified
   by hand against `libs/atlas-env/registry.go:126-133`
   (`MapRegistry.Stale()` returns `false` when `lastSeen.IsZero()` — i.e. a
   registry that never called `Apply`/`Observe`) and
   `libs/atlas-env/registry.go:179-187` (`IsActive("pr-999")` → `false`
   because the record isn't present) — so `decide` skips the stale branch
   entirely and lands on `reasonNotActive`, exactly as asserted. This
   correctly pins that a never-observed registry is legacy mode, not
   spuriously stale. Ran the full gate test subset locally
   (`go test ./consumer/... -run 'TestGate|TestExactlyOne' -v`) — all pass,
   including this one. PASS.

8. **Task 2 interaction (`env.Reconcile`)** — Task 2's `Reconcile` change in
   `libs/atlas-env` is not exercised by any of this task's new/changed tests
   (they all construct `MapRegistry` directly and call `Apply`/nothing, never
   `Reconcile`). The FR-4.6 regression path depends only on `Stale()` and
   `IsActive()`, both read directly above; neither was touched by Task 2 (its
   diff is out of this range and not in the reviewed diff). No interaction
   found between Task 2's arm and this task's drop-reason plumbing for the
   legacy-empty-environment case.

9. **All existing `decide(...)` call sites updated to `got, _ := decide(...)`**
   — 8 sites in `gate_test.go`, confirmed by direct read: lines 20, 30, 44,
   54, 68, 77, 92, 95, 239 (the last via `verdict, _ :=` inline). No assertion
   logic changed on any of them. PASS.

10. **New table-driven `TestGateDropReasons`** (`gate_test.go:103-191`) — six
    subtests matching the brief's table row-for-row (mismatched, stale, not
    active, not owner, owner, legacy), each asserting both verdict and
    reason. Registry setup mirrors the pre-existing same-named scenarios.
    PASS.

11. **`TestGateMismatchReasonWinsOverStaleness`** (`gate_test.go:196-207`) —
    builds the stale registry, passes `mismatched=true`, asserts
    `(gateDropUnresolvable, reasonMismatched)`. PASS.

12. **Metric assertion on `TestGateDropUnresolvableIncrementsCounterAndSkipsHandler`**
    (`gate_test.go:278,297`) — both `testutil.ToFloat64` reads now use
    `gateDroppedUnresolvable.WithLabelValues("atlas-monsters", "pr-999",
    string(reasonNotActive))`. This is a genuine test of label population:
    the 3-label counter requires an exact label-value match, so a bug that
    populated the wrong or empty reason value would leave `before == after`
    for this specific combination. Confirmed by running it — passes, and the
    Debug-level `Error` log line in the test's own output shows
    `reason=not_active`. PASS.

## Cross-service seam check (`libs/atlas-kafka` is consumed by every service)

- `decide` is unexported (`func decide(...)`, package `consumer`) — its
  signature change is invisible outside `libs/atlas-kafka/consumer`.
  Confirmed no external caller exists:
  `grep -rn "consumer\.decide\|\.decide(" --include="*.go" .` (from repo
  root) returns nothing outside `gate.go`/`manager.go`/`gate_test.go`.
- No other exported symbol in the diff changed shape. `gateProcessed`,
  `gateSkippedNotOwner`, `gateDroppedUnresolvable` are all package-private
  `var`s; widening `gateDroppedUnresolvable`'s label set is a
  Prometheus-registration-time change, not a Go API change, and cannot break
  an out-of-module caller's compile.
- `go build ./...` and `go vet ./consumer/...` both clean from the module
  root, confirming no other in-module caller was missed.

## Verification re-run

```
cd libs/atlas-kafka && go build ./...   # clean
gofmt -l consumer/gate.go consumer/manager.go consumer/gate_test.go   # no output
go vet ./consumer/...   # no output
go test ./consumer/... -run 'TestGate|TestExactlyOne' -v   # all PASS, including
  TestGateDropReasons/{mismatched,stale,not_active,not_owner,owner,legacy},
  TestGateMismatchReasonWinsOverStaleness,
  TestGateWithNoRecordsProjectedIsUnchanged,
  TestGateDropUnresolvableIncrementsCounterAndSkipsHandler
```

Matches the implementer's reported output.

## Backend guideline spot-check (code actually touched)

- No new mutable/exported struct — `gateReason` is an immutable string enum,
  consistent with `gateVerdict`'s existing pattern in the same file.
- No test-only constructor added; `gate_test.go` reuses `env.NewMapRegistry`
  and `env.Record` builders that already existed pre-task.
- Functional style preserved — `decide` remains a pure function, no I/O or
  side effects added to it.

## Not evaluable

- None. The reviewed surface (three named files, one call site, six new/
  extended test cases) was fully inspectable within scope; no dependency on
  code outside the brief's three files was needed to evaluate correctness.

## Verdict rationale

Every brief requirement is implemented exactly as specified, byte-for-byte
matching the brief's own code snippets. The FR-4.6 regression test is
present, is a genuine test (traced by hand against `libs/atlas-env`'s
`Stale()`/`IsActive()` to confirm it exercises the claimed path rather than
trivially passing), and the metric-label test genuinely proves population
rather than mere presence. No scope drift, no cross-service compile risk, no
unbounded cardinality, no constants-reuse violation. No findings.
