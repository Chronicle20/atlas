# Bug fix report — cash-shop gift acceptance does not fame the gifter

Task: task-240-cash-shop-stub-operations
Bug: `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-fame.md`

## What was implemented

Followed the bug's `## Fix` section exactly:

1. `services/atlas-channel/atlas.com/channel/saga/model.go`
   - Added `AwardFamePayload = sharedsaga.AwardFamePayload` to the payload
     alias block (alongside `CreateNotePayload`).
   - Added `AwardFame = sharedsaga.AwardFame` to the action constant block
     (alongside `CreateNote`).

2. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`
   - Added `buildGiftFameSaga(transactionId uuid.UUID, now time.Time,
     gifterId uint32, worldId world.Id, channelId channel.Id) saga.Saga`,
     mirroring `buildGiftForwardSaga`'s shape: `SagaType:
     saga.InventoryTransaction`, `InitiatedBy:
     "NOTE_ACTION_GIFT_FORWARD_FAME"`, one `saga.Pending` step with `Action:
     saga.AwardFame` and `saga.AwardFamePayload{CharacterId: gifterId,
     WorldId, ChannelId, Amount: 1}`. Doc-commented with the "why a separate
     saga" reasoning from the bug writeup (compensateNoteSend /
     StatusEventTypeFailed / SP_2713 already shown).
   - In `handleNoteGiftForward`, after the existing
     `noteGiftForwardSagaCreateFunc` call for the note saga succeeds and
     before `noteGiftForwardMarkSentFunc`, the fame saga is created through
     the same `noteGiftForwardSagaCreateFunc` seam. A failure is logged at
     Error and does not return early — the note already exists and the
     one-shot flag still has to be set.
   - Guard: self-gift (`gifter.Id() == s.CharacterId()`) skips the fame saga
     entirely, logged at Debug, mirroring atlas-notes' self-note skip.
   - `worldId`/`channelId` sourced from `s.WorldId()` / `s.ChannelId()`.
   - Added the `atlas-constants/channel` and `atlas-constants/world` imports
     needed for the new function's parameter types.

3. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go`
   - `withNoteGiftForwardSeams`'s saga-creation seam now records every saga
     created (`*[]saga.Saga`, append-in-call-order) instead of a single bool,
     so tests can assert saga count and each saga's shape.
   - `TestHandleNoteGiftForward_Success` updated: asserts exactly 2 sagas are
     created — the first is `saga.NoteSend`, the second is
     `saga.InventoryTransaction` with one `award_fame` step whose payload has
     `CharacterId == gifterId`, `Amount == 1`, and `WorldId`/`ChannelId`
     matching the session's.
   - New `TestHandleNoteGiftForward_SelfGift`: gifter id == session character
     id creates the note saga but no fame saga (1 saga total), and still
     marks the note sent.
   - New `TestBuildGiftFameSaga`: pins the fame saga's shape (type, one
     `award_fame` step, `AwardFamePayload` with the right `CharacterId` and
     `Amount == 1`), mirroring `TestBuildGiftForwardSaga`.
   - `TestHandleNoteGiftForward_GiftFromMismatch` and
     `TestHandleNoteGiftForward_AlreadySent` updated to assert `len(*sagasCreated) == 0`
     (zero sagas, note or fame, on a failed gate).
   - Added `atlas-constants/channel` and `atlas-constants/world` imports for
     `TestBuildGiftFameSaga`.

4. `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-fame.md`
   - Filled in the `## Resolution` section (gate result, pointer to this
     report; live re-test left `_pending_` as noted in "Not yet answered" #3
     — no live cluster access available in this environment).

No changes to `libs/atlas-saga`, atlas-saga-orchestrator, atlas-notes,
atlas-fame, or any packet codec, per the bug's scope note.

## Testing

From `services/atlas-channel/atlas.com/channel`:

```
$ go build ./...
(no output, exit 0)

$ go test ./socket/handler/... -run 'GiftForward|GiftFame' -v
=== RUN   TestBuildGiftForwardSaga
--- PASS: TestBuildGiftForwardSaga (0.00s)
=== RUN   TestHandleNoteGiftForward_GiftFromMismatch
--- PASS: TestHandleNoteGiftForward_GiftFromMismatch (0.00s)
=== RUN   TestHandleNoteGiftForward_AlreadySent
--- PASS: TestHandleNoteGiftForward_AlreadySent (0.00s)
=== RUN   TestHandleNoteGiftForward_Success
--- PASS: TestHandleNoteGiftForward_Success (0.00s)
=== RUN   TestHandleNoteGiftForward_SelfGift
--- PASS: TestHandleNoteGiftForward_SelfGift (0.00s)
=== RUN   TestBuildGiftFameSaga
--- PASS: TestBuildGiftFameSaga (0.00s)
PASS
ok  	atlas-channel/socket/handler	0.014s

$ go test ./...
ok for every package with tests (full module suite), no failures
```

## Files changed

- `services/atlas-channel/atlas.com/channel/saga/model.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go`
- `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-fame.md`

## Self-review

- Fame saga is fired independently of the note saga's outcome path per the
  bug's rationale (separate `InventoryTransaction` saga, not a second step) —
  confirmed the code creates it only after the note saga create call
  succeeds, matching the brief ("after ... succeeds and before
  `noteGiftForwardMarkSentFunc`").
- Self-gift guard placed before firing the fame saga, not before building it
  — matches the brief's "skip the fame saga," no unused builder call.
- A fame-saga-create failure is logged at Error and does not return early;
  `noteGiftForwardMarkSentFunc` still runs afterward — verified by reading
  the final function body.
- No new test-only constructor was added; test setup reuses the existing
  `withNoteGiftForwardSeams`/`giftAsset`/`newCashItemUseTestSession` helpers,
  extending the existing seam's return type rather than adding a parallel
  one.
- Checked `libs/atlas-constants` before introducing `world.Id`/`channel.Id`
  parameter types — reused the existing aliases already used throughout
  `session/model.go`, no new constant defined.

## Issues or concerns

None. The two open items in the bug's "Not yet answered" section (reverse
fame on discard, live confirmation) are explicitly out of scope for this fix
and require a separate product ruling / live access, as stated in the bug
report.
