# Review: task-259 Task 3 — Fix round 1 for race condition in test harness

Commit: `f5fa66c` (range: `9b9c0b9..f5fa66c`)
Module: `services/atlas-monsters/atlas.com/monsters`

## Scope confirmed

Diff touches exactly one file:
- `monster/disease_targets_shell_test.go` (+3 lines)

Fix is scoped to the open blocking finding: the unsynchronized `*positionCalls` append in `diseaseTargetProcessor`'s `positionFn` closure. No production code touched; no files owned by Task 4 touched.

## The open finding: ADDRESSED

**Original finding:** `diseaseTargetProcessor`'s `positionFn` closure did an unsynchronized `*positionCalls = append(*positionCalls, id)` while `resolvePositions` invokes it concurrently from up to 8 goroutines (`positionLookupConcurrency`). The brief's Step 8 requires `go test ./monster/... -race -run TestGetDiseaseTargets` to pass clean; the original review observed 1-in-5 race failures across repeated runs.

**Fix applied:** `disease_targets_shell_test.go:35, 42, 44`:
```go
func diseaseTargetProcessor(inField []uint32, positions map[uint32][2]int16, positionErr map[uint32]error, positionCalls *[]uint32) *ProcessorImpl {
    var mu sync.Mutex              // line 35, new
    // ...
    p.positionFn = func(id uint32) (int16, int16, error) {
        mu.Lock()                   // line 42, new
        *positionCalls = append(*positionCalls, id)
        mu.Unlock()                 // line 44, new
        // ...
    }
}
```

The mutex is created once at function entry (not inside the closure), guarding the shared slice write. This matches the exact pattern already used in `TestGetDiseaseTargets_ConcurrentLookupsPreserveOrder` (`disease_targets_shell_test.go:191, 205-207`).

**Independently verified fix:**
```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run TestGetDiseaseTargets -race -count=5
```

Result: **all 40 tests passed across all 5 runs, zero race reports**. The brief's explicit Step 8 requirement is now met.

Quote from actual output:
```
Go test: 40 passed in 9 packages
```

## Assertions unchanged

All test assertions remain at their original strictness:
- `TestGetDiseaseTargets_BoxlessWithMultiCountReturnsControllerOnly` (line 63): still asserts exact slice match via `reflect.DeepEqual(got, []uint32{7})`
- `TestGetDiseaseTargets_PreservesFieldListingOrder` (line 121): still asserts order-sensitive exact match `reflect.DeepEqual(got, []uint32{3, 1})` — proves index-based assembly, not completion-order
- `TestGetDiseaseTargets_FiltersByBoundingBox` (line 102): still asserts exact match `reflect.DeepEqual(got, []uint32{1, 3})`
- `TestGetDiseaseTargets_PositionFailureExcludesOnlyThatCharacter` (line 140): still asserts exact exclusion `reflect.DeepEqual(got, []uint32{1, 3})`
- `TestGetDiseaseTargets_SeduceCapsAcrossTheShell` (line 185): still asserts seduce cap `reflect.DeepEqual(got, []uint32{1, 2})`
- `TestGetDiseaseTargets_ConcurrentLookupsPreserveOrder` (line 223): still asserts ascending order `reflect.DeepEqual(got, want)` for `1..20`

The mutex guards only the recording of which ids were looked up, not the actual return values or slot assembly. The production code (`resolvePositions`, `getDiseaseTargets`) is race-free by construction and untouched by this fix.

## Import verification

`sync` is imported at `disease_targets_shell_test.go:8`.

## Summary

The fix is minimal, precisely targeted, and successfully addresses the open blocking finding. The required Step 8 race-detector check now passes cleanly across 5 consecutive runs with no regressions. Assertions remain at their original precision — no weakening of test coverage.

---

**Verdict: APPROVED**

The one blocking finding from the original review is fully addressed. The fix is sound, minimal, and verified by independent re-run with `-count=5`.
