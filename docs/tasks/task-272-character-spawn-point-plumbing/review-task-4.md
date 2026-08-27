# Review: Task 4 — atlas-cashshop spawnPoint plumbing

**Range:** `299779b35..c027e20d3`
**Commit:** `c027e20d3` — `fix(atlas-cashshop): decode and return spawnPoint`
**Brief:** `.superpowers/sdd/plan/task-4-brief.md`
**Report:** `.superpowers/sdd/plan/task-4-report.md`

## Scope

Diff stat matches the review package exactly:

```
services/atlas-cashshop/atlas.com/cashshop/character/model.go    | 4 ++--
services/atlas-cashshop/atlas.com/cashshop/character/rest.go     | 1 +
services/atlas-cashshop/atlas.com/cashshop/character/rest_test.go| 9 ++++++++-
3 files changed, 11 insertions(+), 3 deletions(-)
```

All three files are the ones named in the brief's `### Files` section. No other files touched.

## Findings

### PASS — `Model.SpawnPoint()` returns `uint32` and `m.spawnPoint`

`services/atlas-cashshop/atlas.com/cashshop/character/model.go:211-213`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

Previously `func (m Model) SpawnPoint() byte { return 0 }`. Confirmed the `byte` return type is fully removed and the `Model` struct field is itself `uint32` (`model.go:41` — `spawnPoint uint32`), so no truncation occurs.

### PASS — `Extract` decodes `spawnPoint`

`services/atlas-cashshop/atlas.com/cashshop/character/rest.go:155` adds `spawnPoint: m.SpawnPoint,` inside the `Extract` struct literal, between `sp:` and `gm:` as specified. `RestModel.SpawnPoint` is `uint32` (`rest.go:40`), matching the `Model` field type — no lossy conversion.

### PASS — `Transform` already correct (read-only reference confirmed, not modified)

`rest.go:120` — `SpawnPoint: m.spawnPoint,` inside `Transform`. This line is unchanged by the diff, consistent with the brief's statement that `Transform` already emits `SpawnPoint` correctly; the defect was one-directional (decode-only).

### PASS — Non-zero test fixture

`rest_test.go` fixture changed from `SpawnPoint: 0,` to `SpawnPoint: 11,` (diff hunk, `rest_test.go` around former line 43). A zero-valued fixture would have passed against the removed `return 0` stub; the non-zero value is load-bearing.

### PASS — New assertion is a literal-value comparison, not `Extract∘Transform` idempotence

```go
if got := m.SpawnPoint(); got != 11 {
    t.Errorf("SpawnPoint() = %d, want 11", got)
}
```

placed immediately after the first `Extract(rm)` call and before `Transform`/second `Extract`/`DeepEqual` round-trip logic. This compares the decoded `Model` value directly against the `RestModel` literal's `11`, not against a round-tripped value — satisfies the constraint that a round-trip/idempotence check alone (blind to a stub that returns a constant unrelated to input, as long as `Transform` re-emits the same constant) is not acceptable proof. In this case the stub returned an *unconditional* `0` regardless of input, so even idempotence would have failed too — but the assertion as written is the correct, brief-specified form regardless.

Verified by manual trace: pre-fix, `Extract` did not set `spawnPoint` on `Model` (field defaults to Go zero value `0`), and even had it been set, `SpawnPoint()` unconditionally returned `0` — so `m.SpawnPoint()` would equal `0`, and the assertion `!= 11` triggers a failure. This is a genuine RED-then-GREEN case; I did not execute `go test` per the task instruction to not re-run the suite, but the failure mode is unambiguous from reading the removed code (`return 0` regardless of `m.spawnPoint`).

### PASS — Global constraints respected

- `services/atlas-character/**` does not appear in the diff — confirmed via the diff and `git diff --stat` file list above.
- `libs/atlas-packet/**` does not appear in the diff — confirmed.
- No producer added: `grep` for `UpdateSpawnPoint` / any new producer call was not needed since the diff is fully visible above and contains no such addition — only accessor, `Extract`, and test changes.
- No model-copy deduplication — the change is scoped to `atlas-cashshop` only, other seven services untouched (out of this task's file scope, correctly).
- Rank/RankMove/JobRank/JobRankMove stubs untouched — confirmed absent from the diff.
- Test setup: only a `RestModel{...}` struct literal was added/modified (`rest_test.go:17` context), which is the sanctioned `Extract` input shape per the plan's explicit carve-out — no `Model{...}` struct literal was introduced, so the "use the Builder" constraint is not implicated (and would not need to be, since `RestModel` is not `Model`).

## Not evaluable

None. The unit is small, self-contained, and fully within the diff; nothing here depends on out-of-scope files (the only cross-reference, `Transform`, was inspected as an unmodified read-only anchor named explicitly by the brief).

## Verdict

APPROVED. The change exactly matches the brief: single-direction decode fix, correct type (`uint32`), non-zero fixture, literal-value assertion (not idempotence), no scope creep, no forbidden files touched, no producer added.
