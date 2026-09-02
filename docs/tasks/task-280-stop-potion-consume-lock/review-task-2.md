# Review: task-280 Task 2 — ErrPotionLocked sentinel and POTION_LOCKED wire value

Range reviewed: `1161f19d6..8baa153fc` (commit `8baa153fc`)
Brief: `.superpowers/sdd/plan/task-2-brief.md`
Report: `.superpowers/sdd/plan/task-2-report.md`

## Scope

Diff touches exactly three files, all under `services/atlas-consumables/atlas.com/consumables`:

- `consumable/processor.go` (+7/-0)
- `consumable/processor_test.go` (+24/-0)
- `kafka/message/consumable/kafka.go` (+5/-0)

This matches the brief's file list exactly. No `atlas-channel` files touched (that is Task 4's scope, correctly excluded). No shared kafka contract package introduced.

## Checks

1. **Wire value** — `kafka.go:122` adds `ErrorTypePotionLocked = "POTION_LOCKED"` additively to the existing `ErrorType*` const group, with a doc comment referencing task-280 FR-6. Matches the binding constraint exactly (byte-for-byte the brief's snippet). PASS.

2. **Sentinel error** — `processor.go:63-66` adds `ErrPotionLocked = errors.New("potion use locked")` to the existing `var (...)` block alongside `ErrPetCannotConsume`/`ErrPetCannotLearn`. PASS.

3. **`consumeErrorType` arm and existing-arm preservation** — `processor.go:452-454` adds:
   ```go
   if errors.Is(err, ErrPotionLocked) {
       return consumable.ErrorTypePotionLocked
   }
   ```
   placed above the `return consumable.ErrorTypeConsumeFailed` fallthrough, after the pre-existing `ErrPetCannotConsume`/`ErrPetCannotLearn` arms, which are unmodified. Verified via `go test ./consumable/ -run 'TestConsumeErrorType' -v`: `TestConsumeErrorType_GenericFailure`, `_PetCannotConsume`, `_PetCannotLearn`, and the new `_PotionLocked` all PASS. PASS.

4. **Test honesty** — `TestConsumeErrorType_PotionLocked` asserts `consumeErrorType(ErrPotionLocked) == consumable.ErrorTypePotionLocked` and pins the literal wire string `"POTION_LOCKED"`; this would fail without the implementation (undefined symbols pre-change, per the report's RED-by-inspection reasoning, and would fail post-implementation-with-wrong-value since the literal is pinned). `TestErrorEventProviderPotionLocked` exercises `ErrorEventProvider` end-to-end through JSON marshal/unmarshal and asserts `e.Body.Error == "POTION_LOCKED"` — a genuine, non-trivial assertion, not a tautology. PASS.

5. **Build/format** — `go build ./...` succeeds; `go test ./consumable/...` passes (all cases, including pre-existing ones); `gofmt -l` on the three touched files reports no diffs. PASS.

6. **No scope creep** — `git diff --stat` confirms only the three brief-listed files changed; no edits to `character/buff/model.go` (Task 1's territory) or any `atlas-channel` file (Task 4's territory). PASS.

## Findings

None. Blocking: 0. Non-blocking: 0. Not evaluable: 0 (the full change is self-contained and testable within this unit; Task 3's consumption of `ErrPotionLocked`/`ErrorTypePotionLocked` is out of scope for this review and correctly deferred).

## Verdict

APPROVED.
