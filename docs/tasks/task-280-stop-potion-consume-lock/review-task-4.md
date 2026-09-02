# Review: task-280 Task 4 — atlas-channel explicit POTION_LOCKED routing

Range reviewed: `61d74c97b..ee37c5eb7` (single commit `ee37c5eb7`)
Brief: `.superpowers/sdd/plan/task-4-brief.md`
Report: `.superpowers/sdd/plan/task-4-report.md`

## Scope

Diff touches exactly three files, all under `services/atlas-channel/atlas.com/channel/`:

- `kafka/message/consumable/kafka.go` (+5)
- `kafka/consumer/consumable/consumer.go` (92 changed lines: extraction + one new arm)
- `kafka/consumer/consumable/consumer_test.go` (new, +45)

No file under `services/atlas-consumables/` appears in the diff. This matches the task's binding constraint (channel-only) and the brief's file list exactly. No scope mismatch.

## Findings

### PASS — wire constant matches producer exactly

`services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go:97`: `ErrorTypePotionLocked = "POTION_LOCKED"`.
`services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go:127`: `ErrorTypePotionLocked = "POTION_LOCKED"`.
Both strings are `"POTION_LOCKED"`, verified with `grep` against both files. `TestPotionLockedWireValue` (consumer_test.go:41-44) pins the channel-side value.

### PASS — exactly four `errorAction` values, POTION_LOCKED reached via explicit case

`consumer.go:93-108`: `const (actionUnstick errorAction = iota; actionPetCashFoodError; actionInventoryFull; actionVegaInvalid)` — four values, matches the constraint exactly.

`consumer.go:110-126`: `consumableErrorAction` has an explicit `case consumable2.ErrorTypePotionLocked: return actionUnstick` (lines 121-125), distinct from the `default: return actionUnstick` (lines 126-127) that CONSUME_FAILED and unrecognized types fall through to. This satisfies the FR-7 recognition requirement — POTION_LOCKED is not merely reaching the catch-all by accident, it is an explicit, testable branch.

### PASS — no `ErrorTypeConsumeFailed` constant added

Grep of `kafka.go` confirms only `ErrorTypePetCannotConsume`, `ErrorTypeInventoryFull`, `ErrorTypeVegaInvalid`, `ErrorTypePotionLocked` exist in the `ErrorType*` group. `consumer_test.go:22` uses the string literal `"CONSUME_FAILED"` for that test row, as required.

### PASS — no channel-side buff/lock check added (FR-1 routing-only)

`consumableErrorAction` is a pure `switch` on the input string with no session, storage, or buff-processor calls. `handleErrorConsumableEvent`'s only behavioral change is switching on the classifier's return value instead of chained `if`s; no new read of buff/lock state was introduced.

### PASS — existing arms keep their exact effects (verified against diff pre-image)

Comparing removed (`-`) lines against added (`+`) lines in the consumer.go hunk:

- `PET_CANNOT_CONSUME` → `sp.IfPresentByCharacterId(...)(..., session.Announce(l)(ctx)(wp)(petpkt.PetCashFoodResultWriter)(petpkt.NewPetCashFoodResultError().Encode))` — identical call, now under `case actionPetCashFoodError`.
- `INVENTORY_FULL` → same `CharacterStatusMessageWriter`/`CharacterStatusMessageDropPickUpInventoryFullBody` call followed by the same `StatChangedWriter`/`NewStatChanged(make([]statpkt.Update, 0), true)` unstick, now via the shared `unstick` closure. Statement bodies identical.
- `VEGA_INVALID` → same `VegaScrollWriter`/`VegaScrollInvalidBody` call followed by the same unstick. The explanatory comment about dialog/exclusive-request behavior is preserved (wording trimmed slightly, meaning unchanged — comment, not a statement).
- Catch-all (previously the final unconditional block, now `default`) → same bare `StatChangedWriter`/`NewStatChanged(...)` unstick call via the `unstick` closure.

The only observable behavioral difference across all four arms is the error-log message: previously `INVENTORY_FULL` logged `"Unable to process inventory-full event for character [%d]."` while the other three logged `"Unable to process error event for character [%d]."`; now all four log the unified `"Unable to process error event for character [%d]."` (consumer.go, final `if err != nil` block, line ~171). This is a log-text-only change (not a client-facing statement) and is explicitly specified by the brief's own Step 3 code snippet, which the report calls out under Self-review. Not a defect.

### PASS — POTION_LOCKED sends no client message

POTION_LOCKED routes to `actionUnstick` → `default` case → `sp.IfPresentByCharacterId(...)(..., unstick)`, i.e. a bare `StatChanged` unstick with no preceding announce call. Matches "POTION_LOCKED performs the bare StatChanged unstick and sends NO client message."

### PASS — test table matches the authoritative Step 1 count

`consumer_test.go` contains `TestConsumableErrorAction` with 7 subtests (pet cannot consume, inventory full, vega invalid, potion locked, consume failed falls through, empty falls through, unrecognized falls through) plus `TestPotionLockedWireValue`, matching Step 1's enumerated table exactly. No ninth-case finding raised per the controller's ruling.

Ran `go build ./...` and `go test ./kafka/consumer/consumable/... -v` in `services/atlas-channel/atlas.com/channel`: build succeeds, both tests pass, all 7 subtests pass.

### Non-blocking note — comment/log deltas beyond content are non-substantive

`consumer.go`'s Vega-invalid comment changed the em-dash (`—`) to a double-hyphen (`--`) and reflowed the "design §2.3/§4.7" reference out. This is comment-only, has zero effect on behavior or the constraint checklist, and is not evaluated further as a defect. Noted for completeness only.

## Not evaluable

None. The full diff (230 lines) was read in full; both touched files' pre- and post-images were compared hunk-by-hunk; the module builds and the new/existing tests were executed directly in the worktree.

## Verdict

APPROVED. Every binding constraint for Task 4 is satisfied: channel-only scope, exact wire-string match, routing-only (no buff/lock read), all four existing arms preserved verbatim in their client-facing statements, POTION_LOCKED reached via an explicit `case` (not default) and produces no client message, no `ErrorTypeConsumeFailed` constant added, and the test table matches the authoritative Step 1 count. Build and tests pass.
