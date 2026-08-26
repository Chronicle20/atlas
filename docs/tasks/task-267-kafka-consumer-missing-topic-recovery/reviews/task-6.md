# Review: Task 6 — Log preset-creation failures (atlas-character-factory)

Range reviewed: `dc17e62d9..b77baa5` (single commit `b77baa5`, `fix(atlas-character-factory): log preset-creation failures`)
Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`

## Scope

`git diff --stat dc17e62d9..b77baa5`:
```
.../factory/resource.go      | 1 +
.../factory/resource_test.go | 43 ++++++++++++++++++++++
2 files changed, 44 insertions(+)
```
Matches the brief's file list exactly (`resource.go`, `resource_test.go`; `processor.go` untouched, confirmed read-only). No other files touched, no drift into the concurrent `libs/atlas-kafka` work in the worktree.

## Findings

### PASS — log line added exactly as specified (FR-4.1, FR-4.3)

`resource.go:62` (new line, inside `handleCreateFromPreset`'s error branch):
```go
d.Logger().WithError(err).Error("Error creating character from preset.")
```
Placed immediately before `statusCode := categorizePresetError(err)`, matching the brief's Step 3 verbatim. `categorizePresetError` (`resource.go:35-53`) and its call site are byte-for-byte unchanged in the diff — status-code mapping behavior is preserved (FR-4.3).

### PASS — message differs from the seed handler's, satisfying log-search separability (FR-4.2)

Seed handler's existing log call, `resource.go:127`: `"Error creating character from seed."`. New preset handler call: `"Error creating character from preset."`. Confirmed distinct strings; the new test asserts on this distinction directly (`resource_test.go`, the `if e.Message == "Error creating character from seed." { t.Fatalf(...) }` branch).

### PASS — test is genuinely meaningful, not a same-branch-passes-either-way test

The implementer flagged DONE_WITH_CONCERNS because the test and fix were written in the same edit pass, with no literal captured RED run. I verified meaningfulness by reading the code paths rather than re-running TDD:

- `processor.go:271-275` — `CreateFromPreset` with `presetId: "not-a-valid-uuid"` fails `uuid.Parse` and returns `("", ErrInvalidPresetId)` immediately. No logging occurs anywhere in this path before the error returns to `handleCreateFromPreset` (confirmed via `grep` for `.Error(`/`.Warn(`/`.Info(`/`.Debug(` in `processor.go` — no matches on that path).
- Without the fix, `handleCreateFromPreset`'s only action on `err != nil` would be `categorizePresetError` + `w.WriteHeader`; `hook.AllEntries()` would be empty; the test's `if found == nil { t.Fatal(...) }` at the end would fire.
- I independently ran the full test file with the fix present: `TestHandleCreateFromPreset_LogsErrorWithPresetMessage` passes, along with the three pre-existing `TestHandleCreateFromPreset_*` tests and `TestCategorizePresetError` (7 subtests) — all green, matching the report.

Verdict on this test: it does exercise a real assertion path that fails without the fix. The missing literal RED transcript is a process-hygiene gap (TDD discipline), not a correctness or coverage gap — the assertions themselves are sound and the failure mode was traceable by inspection.

### PASS — pre-existing tests unaffected

Ran `go test ./factory/ -run 'TestHandleCreateFromPreset|TestCategorizePresetError' -v` in the module: all pre-existing tests (`_BadJSON`, `_MissingPresetId`, `_InvalidPresetIdFormat`, `TestCategorizePresetError`) still pass with their original status-code assertions; output shows `TestHandleCreateFromPreset_MissingPresetId` and `_InvalidPresetIdFormat` now also emit the new log line (expected side effect, not asserted against by those tests, no assertion breakage).

### Non-blocking — TDD sequencing deviation, self-reported

The report explicitly and accurately discloses that RED was not captured as a standalone failing run (test and fix written together). This is honestly reported (not hidden) and, per the above analysis, doesn't undermine the test's validity. Noting it here as the one item that keeps this from an unqualified APPROVED, per the reviewer's mandate to flag process deviations even when the resulting artifact is sound.

## Not evaluable

None. The full surface (both changed files, plus the read-only `processor.go` contract the fix depends on) was reviewable within this review's scope.

## Verdict

APPROVED_WITH_FINDINGS — the change is correct, minimal, matches the brief exactly, and the new test is genuinely load-bearing (verified by code-path inspection, not just "looks correct"). The only mark against it is the disclosed but non-blocking TDD-sequencing deviation.
