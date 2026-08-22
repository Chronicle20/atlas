# Review: Task 7 — `intervalSet` level-gate interval union

Commit range: `72b4f7332..e22deb1cb` (single commit `e22deb1cb`, "feat(atlas-monster-death): add interval-union level gate primitive (FR-6.2)")
Brief: `.superpowers/sdd/plan/task-7-brief.md`
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

`git show --stat` confirms the commit touches exactly the two files the brief names:

- `services/atlas-monster-death/atlas.com/monster/monster/interval.go` (new, 67 lines)
- `services/atlas-monster-death/atlas.com/monster/monster/interval_test.go` (new, 141 lines)

No other files changed. No scope creep (no touch to `ExperienceConfig`, `processor.go`, or any exported surface). `scope_confirmed`: matches the given range and brief exactly.

The worktree's current HEAD is `e22deb1cb` itself, and `git status` shows no pending modification to either `interval.go` or `interval_test.go` (untracked files present are for other, later tasks — `config.go`, `review-task-*.md`, `agent-ledger.tsv`), so building/testing the working tree is equivalent to building/testing the reviewed commit.

## Findings

### 1. PASS — Merge semantics for adjacent vs overlapping intervals

`interval.go:39-52` (`build()`): sorts by `lo`, then merges with condition `iv.lo <= merged[last].hi+1`. This correctly handles both:
- **Overlapping**: `(0,10)+(5,20)` → `5 <= 10+1` → merges to `[0,20]`. Confirmed by `TestIntervalSet_BuildMergesRanges` (interval_test.go:118-133), which asserts `built.ivs` structurally equals `[{0,20},{100,105}]`, not just the observable `contains` behavior — this pins the merge itself, exactly as the brief specifies.
- **Adjacent**: `(0,10)+(11,20)` → `11 <= 10+1` → merges to `[0,20]`. Confirmed by the `"adjacent intervals merge"` subtest (interval_test.go:38-46): probes at 10, 11, 20 all `true`, 21 `false`.
- **Disjoint**: `(0,5)+(100,105)` → `100 <= 5+1=6` is false → stays two bands. Confirmed by `"disjoint intervals stay disjoint"` subtest (interval_test.go:56-64).

### 2. PASS — `lo < 0 → 0` clamp and int-vs-unsigned arithmetic

`interval.go:23-25` (`add`): `if lo < 0 { lo = 0 }` runs before the interval is stored, and the doc comment (interval.go:18-20) explicitly states the rationale from the brief (mob level `uint32`, member level `byte`, `lo-5` computed in `int` before the clamp so it never wraps). Confirmed by the `"negative lo clamps to zero"` subtest (interval_test.go:66-72): `add(-5, 3)` then probes 0→true, 3→true, 4→false. `add`'s parameters are `int`, matching the brief's interface (`add(lo, hi int)`), so no caller-side unsigned arithmetic reaches this function; the wrap-around scenario the brief warns about is structurally impossible given this signature.

### 3. PASS — Every table case in the brief's test matrix is asserted

All 8 rows of the brief's `TestIntervalSet_Contains` table are present verbatim in `interval_test.go:9-107`, including the PRD worked example (probes 32→true, 70→false, 25/35/115/130→true, 36/114→false) that specifically falsifies a naive `[min-5, max+5]` band implementation — a min/max band over `{25,35,115,125,120,130}` would be `[20,135]` and would wrongly admit 70. Both `TestIntervalSet_BuildMergesRanges` and `TestIntervalSet_BuildIsIdempotent` are present and match the brief's descriptions (structural pin on `s.ivs`; `reflect.DeepEqual` idempotency check on a rebuild of an already-built set).

### 4. PASS — Interface conformance

`intervalSet` is unexported, in package `monster`, with exactly the three brief-specified methods: `add(lo, hi int)` (pointer receiver, mutates), `build() intervalSet` (value receiver, returns a new value), `contains(v int) bool` (value receiver). `build()` copies `s.ivs` into a freshly allocated slice (interval.go:33-34) before sorting/merging, so the receiver's backing array is never mutated — this is what makes idempotency hold and matches the brief's explicit requirement ("must not mutate the receiver's backing array").

### 5. PASS — Test honesty

Ran the actual committed test suite against the committed implementation:
```
cd services/atlas-monster-death/atlas.com/monster && go build ./... && go test ./monster/ -run TestIntervalSet -v
```
All 8 `Contains` subtests, `BuildMergesRanges`, and `BuildIsIdempotent` pass. `go vet ./monster/...` is clean. The PRD-worked-example row is a genuine differentiator (see finding 3) — a plausible-but-wrong min/max implementation would fail it, so this is not a test that passes either way.

## Non-blocking observations

- `add` silently drops a band where `hi < lo` after the clamp (interval.go:26-28). No test in the brief's table exercises this path, and the brief's skeleton includes this exact line, so it's not a deviation — noting only that this behavior is unverified by any test, should a future caller ever pass an inverted range.
- `contains`/`build` are value-receiver methods on `intervalSet` while `add` is a pointer receiver; this mixed-receiver style is unusual but matches the brief's skeleton exactly (`func (s intervalSet) build()`, `func (s intervalSet) contains(v int) bool`) and is intentional — `add` mutates via `append`, `build`/`contains` do not need mutation.

## Not evaluable

None — the unit is fully self-contained (two new files, no external dependents yet since this primitive is not wired into `processor.go` until a later task), and every item in the brief's checklist was directly verifiable from the diff plus a fresh module-local build/test run.

## Verdict

APPROVED. No blocking findings.
