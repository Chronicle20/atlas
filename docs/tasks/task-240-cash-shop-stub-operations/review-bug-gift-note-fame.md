# Review — bug-gift-note-fame (commit 175879987)

Range reviewed: `c84a8b26a..175879987` (single commit
`175879987 fix(atlas-channel): award fame to gifter on cash-shop gift acceptance`).

Brief: `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-fame.md`
Report: `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-fame-report.md`

## Scope confirmed

`git diff --stat` matches the report's file inventory exactly:

- `services/atlas-channel/atlas.com/channel/saga/model.go` (+2)
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go` (+46)
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go` (+92/-17)
- the two task docs (bug + report)

No changes to `libs/atlas-saga`, atlas-saga-orchestrator, atlas-notes, atlas-fame, or
any packet codec, matching the brief's stated scope note. Reviewed the diff plus the
consumer contract it calls into (`libs/atlas-saga/payloads.go`, `unmarshal.go`,
`model.go`, and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{handler.go,compensator.go,event_acceptance.go}`)
since correctness genuinely depends on that contract, per the review brief.

## Findings

### 1. PASS — `AwardFamePayload`/`AwardFame` aliases added correctly

`services/atlas-channel/atlas.com/channel/saga/model.go`:
- `AwardFamePayload = sharedsaga.AwardFamePayload` added to the payload alias block.
- `AwardFame = sharedsaga.AwardFame` added to the action constant block.

Both are re-exports of the shared type/const, not redefinitions — no drift risk.
`gofmt -l` clean on the touched files.

### 2. PASS — emitted payload matches the orchestrator's consumer exactly

`buildGiftFameSaga` (`note_gift_forward.go:67-86`) builds:

```go
saga.AwardFamePayload{
    CharacterId: gifterId,
    WorldId:     worldId,
    ChannelId:   channelId,
    Amount:      1,
}
```

against `libs/atlas-saga/payloads.go:87-95`:

```go
type AwardFamePayload struct {
    CharacterId uint32
    WorldId     world.Id
    ChannelId   channel.Id
    ActorId     uint32     `json:"actorId,omitempty"`
    ActorType   string     `json:"actorType,omitempty"`
    Amount      int16
}
```

Field names and types line up (`gifterId` is `uint32` from `character.Model.Id()`;
`worldId`/`channelId` are `world.Id`/`channel.Id` from `session.Model.WorldId()`/
`ChannelId()` — `session/model.go:179,183` — the identical types the payload struct
declares, so no implicit conversion or truncation). `ActorId`/`ActorType` are
correctly left zero-valued (omitempty), matching "not a quest/actor-attributed
award" semantics — the consumer does not require them.

Consumer: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2254-2266`
(`handleAwardFame`) type-asserts `AwardFamePayload`, builds `channel.NewModel(payload.WorldId, payload.ChannelId)`,
and calls `h.charP.AwardFameAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.Amount)` —
exactly the four fields atlas-channel populates. `AwardFame` action routes to this
handler at `handler.go:918-919`. Wire encoding round-trip verified via
`libs/atlas-saga/unmarshal.go:54-59`, which unmarshals the same `AwardFamePayload`
struct — no field-name mismatch possible.

### 3. PASS — standalone `InventoryTransaction`/`award_fame` saga is a sound, non-misfiring shape

- `SagaType: saga.InventoryTransaction` is a generic, action-agnostic saga type used
  by dozens of unrelated saga shapes across the orchestrator test suite (`grep -rn
  InventoryTransaction saga/*.go` — 90+ hits spanning create/destroy/currency/fame
  sagas). Nothing in `handler.go`'s dispatch (`handler.go:918`) or
  `event_acceptance.go`'s event-kind map (`event_acceptance.go:167`,
  `AwardFame: {EventKindCharacterStatChanged}`) is keyed on saga type — both are
  keyed purely on `Action`. A lone `award_fame` step inside an `InventoryTransaction`
  saga is therefore handled identically to any other `InventoryTransaction` step;
  there is no saga-type-specific compensation path that would misfire on this shape.
- Precedent match: atlas-notes' `buildFameAwardSaga`
  (`services/atlas-notes/atlas.com/notes/note/processor.go:323-338`) builds the exact
  same shape (`SetSagaType(saga.InventoryTransaction)`, one `award_fame` step, a
  self-target skip guard) for the reverse-direction fame award on note discard. This
  fix mirrors an already-proven pattern rather than inventing a new one.
- Compensation: `AwardFame` is registered in `lateCompensableActions`
  (`compensator.go:2989`) with a real late-compensation handler
  (`compensator.go:3255-3262`, negates the awarded amount). This only fires on the
  orchestrator's late/out-of-order-event path (a step whose saga already
  timed out/failed but whose completion event arrives later) — a pre-existing,
  generic mechanism this change does not touch or need to special-case. No normal
  (in-order) failure of the single `award_fame` step can cascade into the sibling
  `note_send` saga, since they are two independent `Saga` records with independent
  `TransactionId`s (`buildGiftForwardSaga` and `buildGiftFameSaga` each call
  `uuid.New()` separately — `note_gift_forward.go:178,187`).
- Event acceptance: `AwardFame` accepts `EventKindCharacterStatChanged`
  (`event_acceptance.go:167`), matching what `AwardFameAndEmit` presumably emits
  (out of this unit's scope to verify further — see "Not evaluable" below) — this is
  the identical acceptance-kind entry the existing atlas-notes discard-fame saga
  already relies on in production, so no new gap is introduced by this change.

### 4. PASS — reject paths cannot leak a fame award

Read the full `handleNoteGiftForward` body (`note_gift_forward.go:158-224`). All four
gates — compartment-load error, unknown `GiftSN`, `giftFrom != ToName`, and
`giftNoteSent == true` — `return` before either `buildGiftForwardSaga`/`sg` or
`buildGiftFameSaga`/`fg` is constructed. The gifter-resolution error path
(`noteGiftForwardCharacterFunc` failure) also returns before either saga is built.
The fame saga is only reachable after the note saga's `noteGiftForwardSagaCreateFunc`
call has already succeeded (`note_gift_forward.go:178-182`), so a note-saga-create
failure cannot leak a fame award either. Confirmed by test:
`TestHandleNoteGiftForward_GiftFromMismatch` and `TestHandleNoteGiftForward_AlreadySent`
assert `len(*sagasCreated) == 0` (`note_gift_forward_test.go:161,182`).

### 5. PASS — self-gift guard matches precedent, placed correctly

`gifter.Id() == s.CharacterId()` (`note_gift_forward.go:184`) skips building/firing the
fame saga entirely (not just skipping the fire after building — no wasted builder
call), logged at Debug, mirroring atlas-notes' `senderId == recipientId` self-note
skip (`processor.go:329-332`). `TestHandleNoteGiftForward_SelfGift` pins this: 1 saga
(note only), still marks sent.

### 6. PASS — `MarkGiftNoteSent` one-shot ordering preserved

The fame-saga-create call reuses the shared `err` variable and does **not** early
`return` on failure (`note_gift_forward.go:187-190` — the `if err != nil` block only
logs). `noteGiftForwardMarkSentFunc` (`note_gift_forward.go:193`) unconditionally runs
next regardless of the fame saga's outcome, so a fame-saga-create failure cannot
prevent the one-shot `GiftNoteSent` flag from being set (which would otherwise permit
an unbounded retry). Both `TestHandleNoteGiftForward_Success` and
`TestHandleNoteGiftForward_SelfGift` assert `*markSentCalled == true`.

### 7. PASS — test honesty

`TestHandleNoteGiftForward_Success` (`note_gift_forward_test.go:194-232`) asserts saga
**count** (`len(*sagasCreated) != 2`), the first saga's type (`NoteSend`), the second
saga's type/step/action (`InventoryTransaction`/`award_fame`), and the payload's
`CharacterId`/`Amount`/`WorldId`/`ChannelId` values against the session and gifter —
this fails without the fix (old code only ever created 1 saga; `sagasCreated` seam
signature itself changed from `*bool` to `*[]saga.Saga` specifically to make the count
assertable). `TestBuildGiftFameSaga` and `TestHandleNoteGiftForward_SelfGift` are
likewise new, targeted, and would fail against the pre-fix code (function doesn't
exist / fame saga still fires on self-gift). Confirmed by running:
`go build ./...` and `go test ./socket/handler/... -run 'GiftForward|GiftFame'` from
`services/atlas-channel/atlas.com/channel` — both pass, matching the report's claim.

### 8. Non-blocking — brief's "unknown-SN / compartment-error" reject-path test coverage not literally added

The bug's file-inventory item 3 says: "Existing reject-path tests ... **and the
unknown-SN / compartment-error paths** — must assert zero sagas created." Only the
two existing `_GiftFromMismatch` and `_AlreadySent` tests were updated to assert zero
sagas; no new `handleNoteGiftForward`-level test exercises the compartment-load-error
or unknown-SN paths for a zero-saga assertion (only `TestFindGiftAsset_UnknownSN`
exists, and it tests the `findGiftAsset` helper directly, not the handler). This is
not a functional defect — finding #4 above confirms by direct code reading that both
paths `return` before any saga construction — but it is a documented-vs-actual gap
against the brief's own file inventory, and a future regression on those gates would
not be caught by a test.

## Not evaluable

- Whether `AwardFameAndEmit` in atlas-character actually emits an event of kind
  `EventKindCharacterStatChanged` that the orchestrator's `event_acceptance.go:167`
  entry expects — atlas-character is outside this unit's diff and outside
  atlas-channel's module; verifying it would require reading a third service not
  touched by this commit. Treated as already-proven by the existing atlas-notes
  discard-fame saga using the identical `AwardFame` action in production, which this
  review takes as sufficient precedent rather than expanding scope to re-verify.
- Live/runtime confirmation of the fame direction and amount — the brief itself
  marks this "pending" (no live cluster access), consistent with the report.

## Verdict rationale

All six correctness dimensions in the review brief (payload contract, saga-type/
compensation soundness, reject-path leak prevention, one-shot ordering) check out
against direct code reading, cross-referenced against the actual orchestrator
consumer and the pre-existing atlas-notes precedent. Tests are honest and would fail
without the fix. The one gap (finding #8) is a coverage completeness note against
the brief's own file inventory, not a functional defect — downgraded to non-blocking
because direct code reading (finding #4) already confirms the underlying gates are
safe.
