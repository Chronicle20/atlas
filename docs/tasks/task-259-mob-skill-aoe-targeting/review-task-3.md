# Review: task-259 Task 3 — the I/O shell and the positionFn seam

Commit range: `08f8567..51c5242` (single commit `51c5242d7`)
Module: `services/atlas-monsters/atlas.com/monsters`

## Scope confirmed

Diff touches exactly the three files named in the brief:
- `monster/processor.go` (+12/-33)
- `monster/disease_targets.go` (+77, new content appended)
- `monster/disease_targets_shell_test.go` (+223, new file)

No files outside the brief's list were touched. Matches Task 3's declared scope; no Task 4 seams (`p.emit` routing for `executeDebuff`/`executeBanish`/`executeDispel`, `testInformationLookup`, `SetBanish`) were touched, consistent with the scope boundary given.

## Brief compliance

1. **`positionFn` seam** — `monster/processor.go:100-101` (field, `func(characterId uint32) (int16, int16, error)`), wired in `NewProcessor` at `monster/processor.go:127-129` to `position.NewProcessor(p.l, p.ctx).GetPosition`, import added at `monster/processor.go:6` (`"atlas-monsters/character/position"`). Matches brief exactly. PASS.

2. **Old selector deleted** — the old `getDiseaseTargets` (with `rand.Shuffle` and direct `_map.NewProcessor(...).CharacterIdsInFieldProvider` call) is gone from `processor.go` (confirmed via diff hunk `-1278,32 +1283,6`). `rand.Shuffle` no longer appears anywhere in `processor.go`; `rand.Intn` remains at `monster/processor.go:714` for the basic-attack damage formula, and `math/rand` stays imported. `_map` import is still used elsewhere. PASS.

3. **`skillId` threaded through callers** — `executeDebuff`'s dispel/banish branches and its own `getDiseaseTargets` call, plus `executeBanish`/`executeDispel` signatures and their internal `getDiseaseTargets` calls, all now pass the real `skillId byte` rather than re-deriving a skill-type constant. Verified via diff hunks at `processor.go:1221-1273` (new line numbers). PASS.

4. **`disease_targets.go` shell** — `positionLookupConcurrency = 8` is a named constant (`disease_targets.go:51`), used at the semaphore-channel construction site (`disease_targets.go:67`) — never a bare literal at the call site. `resolvePositions` (`disease_targets.go:65-92`) uses `sync.WaitGroup` + a buffered channel of size `positionLookupConcurrency` as the semaphore; no `errgroup` import anywhere in the diff or `go.mod` changes (`golang.org/x/sync v0.22.0 // indirect` unchanged). Results are assembled into `slots := make([]*positionedCharacter, len(ids))` indexed by input position, so field-listing order survives goroutine interleaving — index-based assembly, not append-under-lock. PASS.

5. **No field-wide fallback** — `getDiseaseTargets` (`disease_targets.go:102-121`): the `!sd.HasBoundingBox()` branch returns only the controller or `nil`, never the field list. The `p.inFieldFn` error path returns `nil`, not the (nonexistent) partial list. `resolvePositions` drops unresolved characters (logs at `Warn`, does not populate the slot) — the candidate set can only shrink from `len(ids)` down to `len(candidates)`, never widen. No code path anywhere in the diff falls back to "everyone in the field." PASS.

6. **Boxless skill → controller-only, no field listing, no position lookup** — `disease_targets.go:103-108`: `if !sd.HasBoundingBox() { ... return []uint32{m.ControlCharacterId()} }` returns before `p.inFieldFn` or `p.resolvePositions` is ever called. Confirmed by test (`TestGetDiseaseTargets_BoxlessWithMultiCountReturnsControllerOnly`, `disease_targets_shell_test.go:51-66`) which asserts `len(positionCalls) == 0`. PASS.

7. **No literal `128` for seduce** — `disease_targets.go:41` uses `monster2.SkillTypeSeduce` (`selectDiseaseTargets`, part of Task 2's file but re-verified here since Task 3's tests exercise it through the shell); the shell test at `disease_targets_shell_test.go:180` also uses the named constant. `grep 128` across both files: no match. PASS.

8. **Test setup / Builder pattern** — Uses `Clone(Model{}).SetX(...).SetY(...).SetControlCharacterId(...).Build()` and `mobskill.NewModelBuilder()...Build()` throughout, per the project's Builder pattern. No `*_testhelpers.go` file was created; `diseaseTargetProcessor`/`diseaseTargetTenant` live inside `disease_targets_shell_test.go`. PASS.

9. **All 8 named tests from the brief's table are present** and, individually, assert the behavior specified (bounding-box filter, order preservation, position-failure exclusion, field-listing-failure short-circuit, seduce cap, concurrent-order determinism). Verified by reading each test body against the brief's table — matches. PASS on content.

## Deviation adjudication: `diseaseTargetTenant()`

The brief's `diseaseTargetProcessor` signature has no `*testing.T` parameter, yet its prose says it's "built on `recordingProcessor(context.Background(), newTestTenant(t), &emitted)`" — and `newTestTenant` (`monster/cooldown_test.go:28`) requires `t *testing.T` and calls `t.Fatalf` on error. This is a genuine signature conflict in the brief, not an implementer shortcut.

The implementer's `diseaseTargetTenant()` (`disease_targets_shell_test.go:22-28`) calls `tenant.Create(uuid.New(), "GMS", 83, 1)` — the exact same literals `newTestTenant` uses — and panics on error instead of `t.Fatalf`. Since `tenant.Create` only errors on malformed input and the literals are fixed and known-good (the same ones every other test in this package uses successfully), this cannot silently pass a broken tenant into the test; a panic during test setup fails the test just as loudly as `t.Fatalf` would, only with a different failure mode (interrupts the whole test binary run for that test rather than a clean per-test failure). It does not weaken any assertion — the tenant produced is behaviorally identical to `newTestTenant(t)`'s. Judged: **correct**, and the right resolution of a genuinely underspecified brief. Non-blocking note: a `panic` instead of `t.Fatalf` means a future error here would kill the whole `go test` binary rather than failing just this test; worth a one-line follow-up if this helper is ever extended, but not a defect against the brief as given.

## Defect found: data race in the shared test helper

`go test ./monster/... -run TestGetDiseaseTargets -race -count=1`, run 5 times:

```
run 1: ok
run 2: ok
run 3: ok
run 4: --- FAIL: TestGetDiseaseTargets_PositionFailureExcludesOnlyThatCharacter (race detected during execution of test)
run 5: ok
```

`diseaseTargetProcessor`'s `positionFn` closure, at `disease_targets_shell_test.go:40-47`:

```go
p.positionFn = func(id uint32) (int16, int16, error) {
    *positionCalls = append(*positionCalls, id)
    ...
}
```

This closure is invoked concurrently by `resolvePositions`'s goroutine fan-out (`disease_targets.go:71-82`) whenever a test built via `diseaseTargetProcessor` supplies more than one `inField` id (`TestGetDiseaseTargets_FiltersByBoundingBox`, `_PreservesFieldListingOrder`, `_PositionFailureExcludesOnlyThatCharacter`, `_SeduceCapsAcrossTheShell` all supply 2-4 ids, all launched as separate goroutines gated only by an 8-wide semaphore — i.e., unserialized). The `append(*positionCalls, id)` write is unsynchronized: concurrent goroutines race on the same slice header/backing array with no mutex, unlike the two tests the implementer wrote directly against `recordingProcessor` (`TestGetDiseaseTargets_FieldListingFailureReturnsNothing`, which never launches goroutines because the field-listing error returns before fan-out, and `TestGetDiseaseTargets_ConcurrentLookupsPreserveOrder`, which the implementer correctly guarded with `sync.Mutex` at `disease_targets_shell_test.go:188/202-204`).

The implementer's own report claims:
> `go test ./monster/... -race -run TestGetDiseaseTargets` → PASS, no race reports.

That claim does not hold under repeated runs; it is flaky (roughly 1-in-5 to 1-in-3 in local sampling), so a single green run — including the implementer's own — is exactly the kind of report the "Never claim verified from a flagged or partial run" rule exists for. The production code (`resolvePositions`) itself is race-free — it writes to disjoint indices of a preallocated slice and never touches `positionCalls`; this is purely a test-harness defect, but it is squarely inside Task 3's own deliverable (the brief's own Step 8 requires this exact command to pass clean) and it will intermittently fail `tools/verify.sh` (which runs `-race`) for reasons unrelated to the code under test — a source of confusing, hard-to-reproduce CI flakes for whoever hits it next.

**Fix is small**: guard `*positionCalls = append(...)` in `diseaseTargetProcessor`'s `positionFn` with the same `sync.Mutex` pattern already used in `TestGetDiseaseTargets_ConcurrentLookupsPreserveOrder`.

## Correctness of `resolvePositions` / `getDiseaseTargets` (production code)

- Semaphore + WaitGroup fan-out is textbook-correct: `sem <- struct{}{}` acquired before the position lookup, released via `defer`; `wg.Wait()` blocks until all goroutines finish before assembling `out`. No goroutine leak, no unbounded fan-out.
- Index-based `slots []*positionedCharacter` assembly is race-free by construction — each goroutine writes exactly one slot at its own index, never touching another's. Confirmed no `-race` failures against the compiled `resolvePositions`/`getDiseaseTargets` logic itself in isolation (the only failures observed are the test-harness race above).
- Degrade-don't-abort is honored: a `positionFn` error logs and returns from that one goroutine without touching `slots[i]` (stays `nil`), which is filtered out at collection time — never propagated as a fatal error for the whole cast.
- `getDiseaseTargets`'s two early-return paths (`!sd.HasBoundingBox()`, `inFieldFn` error) both skip the entire fan-out — correct per FR-2.1/FR-5.4.

## Full-suite / build check

```
$ go build ./... (module root)          — clean
$ go test ./... (module root)           — ok, all packages, no regressions
$ go test ./monster/... -race -run TestGetDiseaseTargets -count=1  — flaky (see defect above); 4/5 clean, 1/5 fails on the exact test the brief singles out
```

## Findings summary

- Blocking: 1 — the unsynchronized `positionCalls` append in the shared `diseaseTargetProcessor` test helper produces a real, reproducible (if intermittent) data race under `-race`, contradicting both the brief's explicit Step 8 requirement and the implementer's report of a clean race run.
- Non-blocking: 1 — `diseaseTargetTenant()`'s use of `panic` instead of a `t.Fatalf`-style per-test failure is a reasonable resolution of the brief's own signature conflict, but is worth a one-line note if the helper is ever reused for a case where `tenant.Create` could plausibly fail.
- Not evaluable: 0 — everything in scope for Task 3 was directly readable and testable from this worktree.
