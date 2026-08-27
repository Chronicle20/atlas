# Bug — discarding a gift thank-you note fames the gift recipient

Task: task-240-cash-shop-stub-operations
Branch: task-240-cash-shop-stub-operations
Reported against: atlas-pr-1426
Reported by: user (product ruling), 2026-08-27

Follow-up to `bug-gift-note-fame.md`, whose "Not yet answered" item 1 this
resolves. The reporter's ruling, verbatim: *"In this case, the note sender
should not receive a fame. only the giftee"* — i.e. for a cash-shop gift
acknowledgement, the **only** fame that may be awarded is the +1 to the gifter
at acceptance time (already landed in `175879987`). The note's sender must not
also be famed when the gifter later discards the note.

## Reproduced

Not reproduced live — established by reading the code path, same as the parent
bug. No live cluster access was used.

Path traced:

1. `175879987` made gift acceptance award +1 fame to the gifter
   (`handleNoteGiftForward` → standalone `award_fame` saga).
2. The thank-you note itself is created with
   `SenderId = ` the gift **recipient**, `ReceiverId = ` the **gifter**
   (`services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`,
   `buildGiftForwardSaga`).
3. When the gifter later discards that note, atlas-notes' generic mechanic
   awards +1 fame to the note's **sender**:
   `services/atlas-notes/atlas.com/notes/note/processor.go:308` (`Discard`)
   → `buildFameAwardSaga` (line 321), which skips only two cases — sender
   id 0 (system note) and sender == recipient (self-note).

## Observed

A single cash-shop gift can yield **two** fame awards in opposite directions:
+1 to the gifter on acceptance, and +1 to the gift recipient whenever the
gifter deletes the thank-you note.

## Expected

One fame award per gift, to the gifter only. Discarding a gift-originated
thank-you note awards nothing.

## Root cause

atlas-notes has no concept of a gift-originated note — `grep -rni gift
services/atlas-notes` returns nothing. `note.Model` carries only
`id / characterId / senderId / message / timestamp / flag`, so `Discard` cannot
distinguish a note whose fame was already settled at creation from an ordinary
player-sent note, and applies the blanket sender-fame rule to both.

The existing `flag` byte is **not** usable as the marker: it is a wire value
the client interprets for memo render templates (0 = plain, and
`discardSpecialFlag` = wedding invitation, see
`libs/atlas-packet/note/serverbound/operation_discard.go:22`). Repurposing it
would change client-side rendering. A separate, server-only field is required.

## Fix

Thread a server-only `GiftNote bool` from the note's creation site through to
persistence, and have `buildFameAwardSaga` decline when it is set.

Naming rationale: the field records the *fact* (this note originated from a
cash-shop gift acknowledgement), not the policy (don't fame). The fame rule
derives from the fact at the point of decision, so a later policy change does
not require another migration.

### File inventory

1. `libs/atlas-saga/payloads.go`
   - `CreateNotePayload`: add `GiftNote bool \`json:"giftNote,omitempty"\``,
     doc-commented as "note originated from a cash-shop gift acknowledgement;
     its fame was settled at acceptance time".
   - No change needed in `unmarshal.go` (line 737 already unmarshals the whole
     struct), but confirm the round-trip test at `unmarshal_test.go:1275`
     still passes and extend it to cover the new field.

2. `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`
   - `buildGiftForwardSaga`: set `GiftNote: true` in the `CreateNotePayload`.
   - Leave `note_send.go`'s `buildNoteSendSaga` alone — an ordinary player note
     keeps the `false` zero value and the existing fame behavior.
   - Update `note_gift_forward_test.go`'s `TestBuildGiftForwardSaga` to assert
     the flag is set; add/extend a `note_send_test.go` assertion that the
     plain path leaves it false.

3. `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`
   - `note/processor.go`: `CreateNote(transactionId, receiverId, senderId,
     message string, flag byte)` gains a trailing `giftNote bool` parameter
     (interface at line 15, impl at line 37).
   - `note/producer.go`: `CreateNoteCommandProvider` gains the same parameter
     and puts it in the command body.
   - `note/mock/processor.go`: update `CreateNoteFunc` and the method.
   - `saga/handler.go:3790`: pass `payload.GiftNote`.

4. `services/atlas-notes/atlas.com/notes/`
   - `kafka/message/note/kafka.go`: `CommandCreateBody` add
     `GiftNote bool \`json:"giftNote,omitempty"\``. **Do not** add it to
     `StatusEventCreatedBody` — no consumer needs it on the read path, and the
     clientbound note display does not render it.
   - `kafka/consumer/note/consumer.go:51`: pass `c.Body.GiftNote` through.
   - `note/model.go`: add the `giftNote` field and a `GiftNote() bool`
     accessor.
   - `note/builder.go`: add `SetGiftNote(bool) *Builder` (the builder is the
     project's test-setup pattern — no test-only constructor).
   - `note/entity.go`: add `GiftNote bool` to `Entity`; thread it through
     `Make` and `MakeEntity`. `Migration` is `AutoMigrate`, so the column is
     added automatically — no hand-written migration, and no tenant-tables
     manifest to update (verified: nothing under `docs/` or `tools/` enumerates
     the `notes` table's columns).
   - `note/processor.go`: `Create` (line 79) / `CreateAndEmit` (line 113) gain
     the `giftNote` parameter and set it on the built model.
   - `note/processor.go` `buildFameAwardSaga` (line 321): add a third skip —
     when the note is a gift note, return `ok=false` with a Debug log, ahead of
     or alongside the existing system-note and self-note skips. This requires
     `Discard` (line 308) to pass the loaded model's `GiftNote()` through;
     prefer widening the call to take the `Model` it already has in hand over
     adding a fourth positional bool.

5. Tests
   - atlas-notes: a `Discard` test proving a gift note produces **no** pending
     fame award while a sibling ordinary note in the same batch still does —
     the mixed batch is the case that catches a misplaced `continue`.
   - atlas-notes: entity/model round-trip through `Make`/`MakeEntity`.
   - atlas-saga-orchestrator: `handleCreateNote` forwards `GiftNote`.

### Known gap, deliberately not closed

`updateNote` (`services/atlas-notes/atlas.com/notes/note/administrator.go:24`)
uses `tx.Updates(&entity)` with a struct, which GORM treats as
non-zero-fields-only — a `GiftNote` of `false` would not be written by an
update. Gift notes are only ever created, never updated, so this cannot bite
today. Flagged rather than fixed because changing `Updates` semantics touches
every note update path and is out of this bug's scope.

## Not yet answered

1. **Retroactive notes.** Notes already persisted before this change get
   `GiftNote = false` from the AutoMigrate default, so any gift thank-you note
   created between `175879987` and this fix will still fame its sender when
   discarded. No backfill is proposed — the window is a single unreleased
   branch, not production data. Confirm that assumption if this branch has
   already been deployed anywhere.
2. **Amount.** Unchanged from the parent bug: +1, matching the note-discard
   precedent. Not specified in any repo source.

## Resolution

- Fix commit: `d5281504e` — "fix(notes): suppress sender fame on discard of
  gift-originated notes". Report: `bug-gift-note-discard-fame-report.md`.
  Follow-up `bce9b22a0` — "fix(atlas-notes): strengthen gift-note mixed-batch
  fame test with distinct sender ids"; report:
  `bug-gift-note-discard-fame-testgap-report.md`.
- Gate: `tools/verify.sh --quick --base 8ada25e79` exited **0** — 98 checks
  green, including go build/vet across all 91 modules (the `libs/atlas-saga`
  change fans out repo-wide), go analyzer guards, skill/job id guard, scope
  guard, producer seam guard, operator cancel path guard, env domain guard, and
  the lint & format guard over 91 modules. The run's own closing line: "All
  checks passed, but docker bake was skipped — not a pre-PR pass."
  **A flagless `tools/verify.sh` still has to exit 0 before this branch is
  called done** — `--quick` skips the bake and `-race`.
- Review: `review-bug-gift-note-discard-fame.md` — **APPROVED_WITH_FINDINGS**,
  0 blocking. The reviewer hand-traced `GiftNote` across all 25 changed files
  and confirmed: JSON tag parity on both sides of every Kafka hop (a mismatch
  here would fail *open*, i.e. fame still awarded); the `buildFameAwardSaga`
  skip fires for gift notes and does not suppress ordinary ones, including in a
  mixed batch; backward compatibility — `omitempty` on a `false` bool plus the
  AutoMigrate zero-value means an in-flight command or a pre-existing row
  decodes to `false` and keeps today's behavior; `note_send.go`'s ordinary
  player-note path is untouched.
  - Its one non-blocking finding — both notes in the mixed-batch test shared a
    sender id, so the surviving payload's `CharacterId` could not distinguish a
    correct skip from an inverted one — is closed by `bce9b22a0`, which gives
    them distinct ids and was confirmed to fail when the skip condition is
    inverted and pass when it is correct.
- Live re-test: **not done** (no live cluster access from this session). The
  end-to-end confirmation — gift a cash item, accept it, discard the resulting
  thank-you note as the gifter, observe that the gift recipient's fame does
  **not** change while the gifter's earlier +1 stands — has not been run.
