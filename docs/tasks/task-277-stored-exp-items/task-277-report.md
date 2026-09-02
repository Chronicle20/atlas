# Task 277: Solomon no-effect / no-feedback fix — report

## Implemented

### Fix section 1 — re-ingest follow-up recorded
- `docs/TODO.md`: expanded the existing (incomplete) task-277 follow-up entry into
  a full section mirroring task-219's morph-coupon precedent in shape/wording —
  states which documents are stale, why every Writ is rejected until re-ingest,
  and gives two unchecked action items (re-ingest Item.wz Consume for every
  provisioned tenant + canonical GMS/83/1; verify per-tenant via a live GET).
- `docs/tasks/task-277-stored-exp-items/prd.md:331`: ticked the acceptance box
  now that the entry exists.

### Fix section 2 — distinct rejection messages
- `services/atlas-consumables/.../consumable/solomon.go`: replaced the three
  `errors.New(...)` literals with package-level sentinels `ErrSolomonNoExperience`,
  `ErrSolomonLevelExceeded`, `ErrSolomonBalanceNotEmpty`. Ordering/Warn logs
  untouched; every check still rejects before `ConsumeItem`.
- `services/atlas-consumables/.../kafka/message/consumable/kafka.go`: added
  producer-side constants `ErrorTypeSolomonNoExperience` = `"SOLOMON_NO_EXPERIENCE"`,
  `ErrorTypeSolomonLevelExceeded` = `"SOLOMON_LEVEL_EXCEEDED"`,
  `ErrorTypeSolomonBalanceNotEmpty` = `"SOLOMON_BALANCE_NOT_EMPTY"`.
- `services/atlas-consumables/.../consumable/processor.go` (`consumeErrorType`):
  added three `errors.Is` arms ahead of the `ErrorTypeConsumeFailed` fallthrough.
- `services/atlas-channel/.../kafka/message/consumable/kafka.go`: hand-mirrored
  the same three constants, byte-for-byte, with a comment pointing at the
  producer copy and the pin test.
- `services/atlas-channel/.../kafka/consumer/consumable/consumer.go`:
  - `consumableErrorAction` routes all three new types to a new
    `actionSolomonRejected` errorAction (kept the existing `POTION_LOCKED`
    explicit-unstick case and the `CONSUME_FAILED`/default unstick untouched).
  - New switch arm in `handleErrorConsumableEvent` for `actionSolomonRejected`:
    announces `CharacterStatusMessageWriter` +
    `CharacterStatusMessageOperationSystemMessageBody(...)` (Water of Life
    precedent), then calls `unstick`.
  - Added `solomonRejectionMessage(errorType string) string` to resolve which
    of the three exported message consts to show (one action arm, three texts,
    since only the message differs per type).
  - Declared three exported message consts in `WaterOfLifeFailedMessage`'s
    comment style — **flagged for user approval, not verified game text**:
    - `SolomonNoExperienceMessage = "The Writ of Solomon has no effect."`
    - `SolomonLevelExceededMessage = "You are not experienced enough to use the Writ of Solomon."`
    - `SolomonBalanceNotEmptyMessage = "You already have stored EXP banked. Use it before using another Writ of Solomon."`

    Note on register: the brief's own example ("...has been returned to you.")
    fits Water of Life's post-consumption-refund case. Solomon's three checks
    all reject *before* `ConsumeItem`, so the item is never destroyed/refunded —
    I used the "has no effect" pre-check register (`waterOfLifeNoEffectMessage`
    precedent in the same file) for the no-experience case instead, and wrote
    the level/balance texts to match that same register. Please review/reword.

### Tests
- `services/atlas-channel/.../kafka/consumer/consumable/consumer_test.go`:
  added 3 rows to `TestConsumableErrorAction` (all three types → `actionSolomonRejected`),
  a new `TestSolomonErrorWireValues` pinning the three literal strings, and a new
  `TestSolomonRejectionMessage` pinning each type → its own message text (and the
  unrecognized-type fallback).
- `services/atlas-consumables/.../kafka/message/consumable/kafka_test.go` (new file):
  `TestSolomonErrorWireValues` pins the producer-side literals — this is the
  "pin the strings in a test on both sides" requirement from the brief.
- `services/atlas-consumables/.../consumable/solomon_test.go`: added
  `TestConsumeSolomonRejectionErrorTypes` — for each of the three rejection
  paths, asserts `consumeErrorType(err)` returns the correct producer-side
  type AND that the item is never consumed and no credit is issued.

## Tested

```
cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./...
```
All packages `ok` (consumable, kafka/message/consumable, and the rest of the
module unaffected — 24 packages, all pass).

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages `ok` (kafka/consumer/consumable and the rest of the module —
~150 packages, all pass, output pristine).

## Files changed
- `docs/TODO.md`
- `docs/tasks/task-277-stored-exp-items/prd.md`
- `docs/tasks/task-277-stored-exp-items/bug-solomon-no-effect-no-feedback.md` (added, the brief itself — untracked in worktree, now committed)
- `services/atlas-consumables/atlas.com/consumables/consumable/solomon.go`
- `services/atlas-consumables/atlas.com/consumables/consumable/solomon_test.go`
- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`
- `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
- `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka_test.go` (new)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer_test.go`
- `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go`

## Self-review
- Ordering contract preserved: all three eligibility checks in `consumeSolomon`
  still run and reject before `d.compartment.ConsumeItem`; the tests assert
  the item is never consumed on any rejection path.
- `CONSUME_FAILED` → `actionUnstick` default left untouched, as instructed.
- The cross-service seam (producer/consumer string literals) has no shared
  constant, as flagged in the brief; both sides now have a pinning test.
- Did not touch section 3 (operational re-ingest) — no kubectl/environment
  commands were run.
- Did not narrow the "distinct messages" ruling to only the level gate — all
  three reasons get their own text, per the explicit ruling.

## Concerns
- **Message wording is new, unverified copy** — needs the user's explicit
  approval/rewording per the brief's own "Not yet answered" section. Called
  out above and in-code via comments.
