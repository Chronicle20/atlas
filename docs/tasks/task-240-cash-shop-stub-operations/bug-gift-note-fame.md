# Bug — cash-shop gift acceptance does not fame the gifter

Task: task-240-cash-shop-stub-operations
Branch: task-240-cash-shop-stub-operations
Reported against: atlas-pr-1426
Reported by: user (product ruling), 2026-08-27

## Reproduced

Not reproduced live — this is a **missing behavior**, established by reading the
code path end to end rather than by observing a failure. No live cluster access
was used; nothing in the flow below is inferred from remembered game behavior,
it is all read from repo source at the commits on this branch.

Path traced:

1. Client accepts a cash-shop gift →
   `CCashShop::OnCashItemResLoadGiftDone` writes NOTE_ACTION SEND with
   `giftFlag == 1`.
2. `services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go:45`
   routes that to `handleNoteGiftForward`.
3. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go:106`
   gates on (a) the asset existing in the account's cash compartment by
   `GiftSN`, (b) `GiftFrom == ToName`, (c) `GiftNoteSent` not already set —
   then creates a `NoteSend` saga with one `create_note` step
   (`buildGiftForwardSaga`, same file line 29) and fires `MARK_GIFT_NOTE_SENT`.
4. `buildGiftForwardSaga` sets `SenderId = s.CharacterId()` (the gift
   **recipient**, who typed the thank-you message) and
   `ReceiverId = gifter.Id()` (the character who **gifted** the item).

Nothing anywhere in that path awards fame.

## Observed

Accepting a cash-shop gift creates the thank-you note and nothing else. The
gifter's fame is unchanged.

The only fame that can ever result is *backwards*: atlas-notes awards +1 fame
to a note's **sender** when the note's owner discards it
(`services/atlas-notes/atlas.com/notes/note/processor.go:308` →
`buildFameAwardSaga`, line 321). So if the gifter later deletes the thank-you
note, the **gift recipient** gains +1 fame — the opposite of the intended
direction. That generic discard mechanic is not changed by this fix; see
"Not yet answered".

## Expected

Accepting a cash-shop gift fames the character who **received the note** — i.e.
the character who gifted the item — by +1.

## Root cause

The gift-forward branch was specified (design §2.2, and the whole
`bug-round-2-gift-notice-*` series) purely as a note-delivery path. Fame was
never part of that spec: no task-240 artifact mentions fame
(`grep -rn -i fame docs/tasks/task-240-cash-shop-stub-operations/*.md` returns
nothing). `handleNoteGiftForward` therefore builds a one-step saga
(`create_note`) and stops. The `award_fame` saga action exists and is wired
end to end (`libs/atlas-saga/model.go:102`,
`services/atlas-saga-orchestrator/.../saga/handler.go:918` →
`handleAwardFame` → `character.AwardFame`), but atlas-channel never emits it.

Two supporting facts that shape the fix:

- `award_fame` routes to **atlas-character** directly, not through
  atlas-fame's `REQUEST_CHANGE`. It therefore bypasses atlas-fame's level-15
  gate, its once-per-day / once-per-target-per-month limits, and writes no
  fame log row. This matches the existing note-discard fame precedent
  (`atlas-notes` uses the same `award_fame` action for the same reason: an
  earned fame is not a player-spent fame). **Assumption taken:** gift fame
  behaves the same way — uncapped, unlogged, not consuming the daily fame.
- `atlas-channel/saga/model.go` does **not** currently alias `AwardFame` or
  `AwardFamePayload` from `sharedsaga`; both aliases must be added.

## Fix

Award +1 fame to the gifter at gift-acceptance time, as a **separate**
single-step saga, fired from `handleNoteGiftForward` after the note saga is
created.

Why a separate saga rather than a second step on the existing `NoteSend`
saga: `compensateNoteSend`
(`services/atlas-saga-orchestrator/.../saga/compensator.go:2246`) terminates
the whole saga and emits `StatusEventTypeFailed` carrying the sender's
characterId, which the channel turns into a `MEMO_RESULT SEND_ERROR` announce.
The client has already shown SP_2713 "The note has successfully been sent."
unconditionally before any server reply, so a fame failure must not be able to
drag the note saga into that error path. A standalone
`InventoryTransaction` saga carrying one `award_fame` step is the established
precedent for exactly this — see `atlas-notes`' `buildFameAwardSaga`.

### File inventory

1. `services/atlas-channel/atlas.com/channel/saga/model.go`
   - Add `AwardFamePayload = sharedsaga.AwardFamePayload` to the payload alias
     block (alongside `CreateNotePayload`, line 36).
   - Add `AwardFame = sharedsaga.AwardFame` to the action constant block
     (alongside `CreateNote`, line 135).

2. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`
   - New builder `buildGiftFameSaga(transactionId uuid.UUID, now time.Time,
     gifterId uint32, worldId world.Id, channelId channel.Id) saga.Saga`,
     mirroring `buildGiftForwardSaga`'s shape: `SagaType: saga.InventoryTransaction`,
     `InitiatedBy: "NOTE_ACTION_GIFT_FORWARD_FAME"`, one `saga.Pending` step
     with `Action: saga.AwardFame` and
     `saga.AwardFamePayload{CharacterId: gifterId, WorldId: ..., ChannelId: ...,
     Amount: 1}`. Doc-comment it with the "why a separate saga" reason above.
   - In `handleNoteGiftForward`, after the existing
     `noteGiftForwardSagaCreateFunc` call succeeds and **before**
     `noteGiftForwardMarkSentFunc`, create the fame saga through the **same**
     `noteGiftForwardSagaCreateFunc` seam (do not add a second seam — it is
     the same "create a saga" dependency). A failure is logged at Error and
     must **not** return early: the note already exists and the one-shot flag
     still has to be set.
   - Guard: skip the fame saga when `gifter.Id() == s.CharacterId()`
     (self-gift), mirroring atlas-notes' self-note skip
     (`processor.go:329`). Log at Debug.
   - `worldId`/`channelId` come from `s.WorldId()` / `s.ChannelId()`
     (`session/model.go:179,183`).

3. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go`
   - `noteGiftForwardSagaCreateFunc` now fires twice on the success path;
     change the existing capture from a single `saga.Saga` to a slice and
     update `TestHandleNoteGiftForward_Success` (line 191) accordingly.
   - New unit test for `buildGiftFameSaga` (shape: type, action, payload,
     `Amount == 1`), mirroring `TestBuildGiftForwardSaga` (line 32).
   - New test: success path creates exactly two sagas, the second being the
     `award_fame` one targeting the **gifter's** id with the session's world
     and channel.
   - New test: self-gift (gifter id == session character id) creates the note
     saga but **no** fame saga.
   - Existing reject-path tests (`_GiftFromMismatch` line 150,
     `_AlreadySent` line 171, and the unknown-SN / compartment-error paths)
     must assert **zero** sagas created — i.e. no fame leaks past a failed
     gate.

Scope note: no change to `libs/atlas-saga`, atlas-saga-orchestrator,
atlas-notes, atlas-fame, or any packet codec. No wire change. No contract
change — `award_fame` is an existing, consumed action.

## Not yet answered

1. ~~**Reverse fame on discard.**~~ **RESOLVED** — the reporter ruled: "In this
   case, the note sender should not receive a fame. only the giftee." Fixed in
   `d5281504e` via a server-only `GiftNote` field threaded from
   `buildGiftForwardSaga` to atlas-notes' `buildFameAwardSaga`. See
   `bug-gift-note-discard-fame.md`. Original text follows.

   The generic atlas-notes mechanic still fames a
   note's sender when the owner discards it, so the gifter deleting the
   thank-you note will fame the gift recipient. Making gift-forward notes
   exempt would require a gift marker on the note (atlas-notes `Model`/entity
   currently has no gift concept — `grep -rni gift services/atlas-notes`
   returns nothing), i.e. a schema + Kafka contract change. Deliberately **not**
   done here; needs a product ruling on whether it is wanted at all.
2. **Amount.** +1 is assumed, matching the note-discard precedent. Not
   specified by the reporter, not found in any repo source.
3. **Live confirmation.** The fame direction ("gifter gains fame on
   acceptance") is the reporter's product ruling; it is not derived from
   client source. The client is not involved in the award — fame is entirely
   server-side — so there is no IDA evidence to check against.

## Resolution

- Fix commit: `175879987` — "fix(atlas-channel): award fame to gifter on
  cash-shop gift acceptance". Report: `bug-gift-note-fame-report.md`.
- Gate: `tools/verify.sh --quick --base c84a8b26a` exited **0**. Passing blocks:
  go build/vet (services/atlas-channel/atlas.com/channel), go analyzer guards,
  skill/job id guard, scope guard, producer seam guard, operator cancel path
  guard, env domain guard, lint & format guard (1 module). The run's own closing
  line: "All checks passed, but docker bake was skipped — not a pre-PR pass."
  **A flagless `tools/verify.sh` still has to exit 0 before this branch is
  called done** — `--quick` skips the bake and `-race`.
- Review: `review-bug-gift-note-fame.md` — **APPROVED_WITH_FINDINGS**, 0
  blocking. Verified the cross-service seam: the emitted `AwardFamePayload`
  matches atlas-saga-orchestrator's `handleAwardFame` consumer field-for-field
  and type-for-type; dispatch and event-acceptance are action-keyed rather than
  saga-type-keyed, so the standalone `InventoryTransaction` saga cannot misfire
  compensation onto the sibling `note_send` saga (independent records,
  independent transaction ids); all four reject gates return before either saga
  is built; the fame-create failure path does not return early, so
  `MarkGiftNoteSent`'s one-shot ordering is preserved.
  - One non-blocking finding: the unknown-SN and compartment-error reject paths
    were not given the zero-saga assertions this file's inventory item 3 asks
    for. Not a functional defect (both paths `return` before any saga is built).
    Closed by a follow-up commit; see `bug-gift-note-fame-testgap-report.md`.
  - One item the reviewer marked not-evaluable: whether atlas-character's
    `AwardFameAndEmit` actually emits the `EventKindCharacterStatChanged` the
    orchestrator's event-acceptance map expects for `AwardFame`. atlas-character
    is outside this commit's diff and outside atlas-channel's module; it is
    treated as already-proven by atlas-notes' discard-fame saga, which uses the
    same action in production. **Unverified here** — worth confirming on the
    live re-test.
- Live re-test: **not done** (no live cluster access from this session). The
  end-to-end confirmation — gift a cash item, accept it on the recipient,
  observe the gifter's fame increment by 1 — has not been run. See "Not yet
  answered" #3.
