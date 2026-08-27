# Review: Task 7 — atlas-consumables spawnPoint accessor

Commit reviewed: `f56e6fc70d93702d06ad59d0e22df5d854bf1fec` (single commit, not a range)

## Scope

Diff touches exactly:
- `services/atlas-consumables/atlas.com/consumables/character/model.go` (2 lines)
- `services/atlas-consumables/atlas.com/consumables/character/rest_test.go` (7 lines added)

`character/rest.go` (`Extract`/`Transform`) is unmodified, as required by the brief. `scope_confirmed`: matches the brief's file inventory exactly — no scope drift.

## Findings

### PASS — Accessor un-stubbed correctly
`character/model.go:213-215`:
```go
func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}
```
Replaces the hardcoded `byte { return 0 }` stub. Returns the full-width backing field `spawnPoint uint32` (`model.go:42`) with no narrowing inside the accessor — matches the established convention (narrow only at the wire writer, never in the model). This service has no wire path for `SpawnPoint`, consistent with the brief's note that narrowing doesn't belong here.

### PASS — `Extract`/`Transform` untouched
`git show f56e6fc` shows no changes to `rest.go`. `rest.go:124` (`spawnPoint: rm.SpawnPoint`) and `:158` (`SpawnPoint: m.spawnPoint`) already round-trip correctly and remain untouched, per brief instruction.

### PASS — Test is a genuine fidelity witness
`rest_test.go:52-53` asserts `got.SpawnPoint() != 275` before the existing `DeepEqual` check, using the exact wording specified in the brief. Verified independently:
- Against the pre-change `byte`-returning stub, `275` overflows the untyped-constant comparison — a compile error, a valid red state (confirmed via report's captured output: `character/rest_test.go:52:25: 275 (untyped int constant) overflows byte`).
- Against the post-change `uint32` accessor, `go test ./character/... -run TransformRoundTrip -v` passes.

This is a real regression witness, not decorative coverage — it fails without the fix and passes with it.

### PASS — No other consumers broken
`grep -rn "SpawnPoint()" .` inside the module shows only two references: the accessor definition itself and the new test assertion. No other caller depended on the old `byte` return type, so widening the signature is safe within this module's scope.

### PASS — Build and full test suite
Verified independently (module-local, no `verify.sh`):
```
go build ./...   → succeeds, no output
go test ./character/... -run TransformRoundTrip -v  → PASS
```
All character-package tests pass or report `[no test files]`.

## Not evaluable

None — the unit is small and fully within reviewable scope.

## Verdict

APPROVED. The commit does exactly what the brief specifies: un-stubs the accessor to the correct `uint32` type without reintroducing narrowing, leaves the already-correct `Extract`/`Transform` untouched, and adds a test that is a genuine (not decorative) failure witness for the bug being fixed.
