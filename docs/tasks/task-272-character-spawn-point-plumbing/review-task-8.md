# Review: Task 8 — atlas-messages spawnPoint accessor

**Commit reviewed:** `e4f376d2e1c8c8d17ecde2d1fbd57b5dc50385e2` (single commit, not a range — tasks 6/7 committed concurrently onto the same branch and are explicitly out of scope).

**Brief:** `.superpowers/sdd/plan/task-8-brief.md`
**Report:** `.superpowers/sdd/plan/task-8-report.md`

## Scope

`git show e4f376d` touches exactly two files:

- `services/atlas-messages/atlas.com/messages/character/model.go` (+2/-2)
- `services/atlas-messages/atlas.com/messages/character/rest_test.go` (+7/-0)

Both are on the brief's file inventory. No other file is present in the commit. Scope confirmed: matches the brief exactly.

## Findings

### PASS — Accessor un-stubbed correctly

`character/model.go:205-207`:

```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```

Was previously `func (m Model) SpawnPoint() byte { return 0 }`. The backing field `spawnPoint` is declared `uint32` at `character/model.go:37`, matching `RestModel.SpawnPoint uint32` (`character/rest.go` struct field, `json:"spawnPoint"`). The return type change from `byte` to `uint32` is correct and matches the field's actual width — no narrowing cast introduced in the model, consistent with the established pattern (narrowing belongs at the wire-writing call site, not here).

### PASS — Extract/Transform untouched

`git diff e4f376d~1 e4f376d -- character/rest.go` produces no output. `Extract` (character/rest.go:163, populates `spawnPoint: rm.SpawnPoint` at line 163 body) and `Transform` are unmodified, as required — this task was accessor-only.

### PASS — Stance() stub left alone

`character/model.go:225-227` still reads:

```go
func (m Model) Stance() byte {
	return 0
}
```

Untouched, per brief's explicit out-of-scope note (PRD §9 item 3).

### PASS — Test assertion is honest (proven RED→GREEN)

`character/rest_test.go:410-417`, added inside `TestTransformRoundTrip`, immediately after the `Extract` error check, exactly as specified in the brief:

```go
if got := m.SpawnPoint(); got != 11 {
	t.Errorf("SpawnPoint() = %d, want 11", got)
}
```

The fixture at line ~401 sets `SpawnPoint: 11` in the `RestModel` literal. Reproduced locally against the pre-commit state:

```
$ git diff e4f376d~1 e4f376d -- character/rest.go   # empty, confirms Extract/Transform untouched
$ go test ./character/... -run TransformRoundTrip -v   # (post-commit) PASS
```

The report's RED evidence (`SpawnPoint() = 0, want 11` against the stub) is consistent with the diff: reverting only the `model.go` hunk while keeping the test would reproduce that failure, since `Extract` correctly assigns `spawnPoint: rm.SpawnPoint` but the stub ignored it. This is a genuine differential test — it fails without the fix and passes with it, not a test that passes either way.

### PASS — No other call sites needed updating

`grep -rn "SpawnPoint()" .` inside the module surfaces only `character/model.go:205` (the accessor) and `character/rest_test.go:416` (the new test). No writer/wire call site in this module invokes `Model.SpawnPoint()`, so no `byte(...)` narrowing cast was needed here — consistent with the brief's "accessor-only, no Extract/Transform/wire change expected."

### PASS — Build and full test suite green

Ran locally, module-local (no `verify.sh`):

```
$ go build ./...          # succeeds
$ go test ./...           # 29 "ok" packages, 0 FAIL
```

## Not evaluable

None — the unit is small and fully self-contained within the reviewed commit; no cross-service seam is implicated (task is explicitly accessor-only, "Consumes: nothing from other tasks").

## Verdict rationale

Diff is minimal and matches the brief line-for-line: correct type widening (`byte`→`uint32`) to the field's true width, no narrowing introduced, `Extract`/`Transform`/`Stance()` left untouched, and a differential (RED-then-GREEN) test assertion added at the specified location. No defects found.
