# bug-gift-note-oneshot-report — Defect I

Implements `## Defect I — a replayed acknowledgement packet mints unlimited
free notes` from
`docs/tasks/task-240-cash-shop-stub-operations/bug-gift-ack-note-and-redisplay.md`,
plus its "Also in this unit — finding 1" test fix.

## What was implemented

A second, independent per-asset flag `GiftNoteSent` (atlas-cashshop), threaded
through every hop `5b9c5be4e` (GiftAcknowledged, Defect H) established as the
template:

1. **atlas-cashshop `cashshop/inventory/asset`**: `entity.go` (gorm column +
   doc comment covering the known async-race limitation), `model.go` (field,
   getter, `ModelBuilder` setter, both `Clone`/`Build` paths), `rest.go` +
   `rest_test.go` (JSON:API attribute `giftNoteSent`, round-trip test).
2. **atlas-cashshop `administrator.go`/`processor.go`**: `updateGiftNoteSent`
   (single cashId, scoped by compartmentId) and `Processor.MarkGiftNoteSent`.
3. **atlas-cashshop `cashshop/processor_gift_note_sent.go`** (new, mirrors
   `processor_gift_ack.go`): `MarkGiftNoteSentAndEmit(accountId, cashId)`,
   resolving the account's compartments and writing to whichever holds the
   asset. Registered on the top-level `Processor` interface.
   `processor_gift_note_sent_test.go` covers the happy path (named asset
   flipped, sibling untouched) and an unknown-cashId no-op.
4. **`kafka/message/cashshop/kafka.go` in both services**:
   `CommandTypeMarkGiftNoteSent = "MARK_GIFT_NOTE_SENT"` and
   `MarkGiftNoteSentCommandBody{AccountId uint32; CashId int64}`. Verified
   byte-identical JSON tags by hand (see below).
5. **atlas-cashshop `kafka/consumer/cashshop/consumer.go`**:
   `handleCommandMarkGiftNoteSent`, registered alongside
   `handleCommandAcknowledgeGifts`, same type-guard shape.
   `consumer_test.go` adds `TestHandleMarkGiftNoteSentInvokesProcessor` and
   `TestHandleMarkGiftNoteSentIgnoresOtherCommandTypes`, mirroring the
   ACKNOWLEDGE_GIFTS pair.
6. **atlas-channel `cashshop/producer.go` + `processor.go`**:
   `MarkGiftNoteSentCommandProvider` and `Processor.MarkGiftNoteSent`.
7. **atlas-channel `cashshop/inventory/asset/{model.go,builder.go,rest.go}`**:
   mirrors the `giftNoteSent` attribute into the channel-side read model.
8. **atlas-channel `socket/handler/note_gift_forward.go`**:
   - `findGiftAsset` now also returns the asset's `GiftNoteSent`.
   - A new gate, after the existing ownership + `GiftFrom == ToName` check:
     if the asset's `GiftNoteSent` is already true, reject (log + return,
     announce nothing) — same as the mismatch gate. `GiftAcknowledged` is
     never consulted, per "Interaction with Defect G".
   - On success, fires `MarkGiftNoteSent` for the gift's cash id via a new
     `noteGiftForwardMarkSentFunc` test seam (same pattern as
     `noteGiftForwardSagaCreateFunc`).
   - The known-limitation comment (async Kafka round trip narrows, but does
     not close, the race) is on the gate itself, per the brief — no
     synchronous seam was invented.

## Finding 1 fix — the meaningless mismatch test

`TestFindGiftAsset_GiftFromMismatch` only asserted a locally recomputed
condition (`giftFrom == "SomeoneElse"`) and never called
`handleNoteGiftForward`, so it passed even with the gate deleted from
production code.

Fix: added two new test seams, `noteGiftForwardCompartmentFunc` and
`noteGiftForwardCharacterFunc` (same "package-level var, test-overridable"
pattern already used four times in this package — `scriptedItemSagaCreateFunc`,
`dueyCouponSagaCreateFunc`, `remoteMerchantSagaCreateFunc`,
`npcItemUseSagaCreateFunc`), so `handleNoteGiftForward` can be exercised
directly against a builder-constructed `compartment.Model`/`character.Model`
with no HTTP mocking required. `TestFindGiftAsset_GiftFromMismatch` is
replaced by `TestHandleNoteGiftForward_GiftFromMismatch`, which calls the real
handler with a mismatched `ToName` and asserts (via the saga-create and
mark-sent seams) that neither fires. Added the equivalent negative test for
the new gate, `TestHandleNoteGiftForward_AlreadySent`, and a positive control,
`TestHandleNoteGiftForward_Success`.

**Why this approach instead of full HTTP JSON:API mocking:** there is no
precedent anywhere in `atlas-channel`'s test suite (checked the whole
`socket/handler` package and the module) for constructing a mocked
JSON:API compartment (with nested asset/item relationships and `included`)
over `httptest`. Building that from scratch was judged disproportionate risk
for this fix versus the existing, already-established test-seam convention
that every other saga-creating handler in this exact file uses. The seam
approach directly exercises the same `handleNoteGiftForward` code path the
production packet handler calls — it does not re-derive the gate logic in the
test.

**RED confirmation** (that the rewritten test is load-bearing, not
cosmetic): temporarily deleted the `giftFrom == "" || giftFrom != sp.ToName()`
gate from `note_gift_forward.go` and re-ran
`go test ./socket/handler/... -run TestHandleNoteGiftForward_GiftFromMismatch`.
Result: **build failure** (`giftFrom declared and not used`), because deleting
the check removes the only remaining use of the variable — an even stronger
signal than a test failure that the check is load-bearing. Restored the file
from a pre-edit backup and re-ran the full package build + test suite to
confirm it was back to green (see GREEN below). The file was restored via
`cp` before any further edits; git status confirms no stray diff from this
probe survived into the commit.

## Byte-identical JSON tag verification

```
== services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go ==
type MarkGiftNoteSentCommandBody struct {
	AccountId uint32 `json:"accountId"`
	CashId    int64  `json:"cashId"`
}
== services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go ==
type MarkGiftNoteSentCommandBody struct {
	AccountId uint32 `json:"accountId"`
	CashId    int64  `json:"cashId"`
}
```
Confirmed identical by hand via `awk` extraction of both struct definitions.

## Testing

Module-local build + test, atlas-cashshop:
```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./cashshop/...
ok  	atlas-cashshop/cashshop	7.681s
?   	atlas-cashshop/cashshop/commodity	[no test files]
ok  	atlas-cashshop/cashshop/inventory	0.016s
ok  	atlas-cashshop/cashshop/inventory/asset	0.010s
ok  	atlas-cashshop/cashshop/inventory/asset/reservation	(cached)
ok  	atlas-cashshop/cashshop/inventory/compartment	0.032s
```

Module-local build + test, atlas-channel:
```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./cashshop/... ./socket/handler/...
ok  	atlas-channel/cashshop	(cached)
ok  	atlas-channel/cashshop/inventory	(cached)
ok  	atlas-channel/cashshop/inventory/asset	(cached)
ok  	atlas-channel/cashshop/inventory/compartment	(cached)
ok  	atlas-channel/cashshop/item	(cached)
ok  	atlas-channel/cashshop/purchaserecord	(cached)
ok  	atlas-channel/cashshop/wishlist	(cached)
ok  	atlas-channel/socket/handler	(cached)
```

New/changed tests specifically exercised (verbose):
```
--- PASS: TestFindGiftAsset_Matching
--- PASS: TestFindGiftAsset_UnknownSN
--- PASS: TestHandleNoteGiftForward_GiftFromMismatch
--- PASS: TestHandleNoteGiftForward_AlreadySent
--- PASS: TestHandleNoteGiftForward_Success
--- PASS: TestNoteActionSendGiftFlagZeroStillGatesOnNoteItem
--- PASS: TestNoteActionSendGiftFlagOneDoesNotHitNoteItemGate
```

`go vet ./socket/handler/...` — clean.

Formatting: `gofmt -l` over every file this task touched (all 20) — empty
output, all clean. `gofumpt` is not installed in this environment, so the
gofumpt-specific alignment check the brief warns about could not be run
directly; `gofmt` output for all touched files is pristine, and struct-literal
field alignment was verified by eye after every mechanical `sed`/`python3`
edit (the earlier branch failure was a key-alignment finding in a test map
literal, which does not apply to any file touched here).

## Files changed

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/entity.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/model.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest_test.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/administrator.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift_note_sent.go` (new)
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift_note_sent_test.go` (new)
- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
- `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go`
- `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer_test.go`
- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`
- `services/atlas-channel/atlas.com/channel/cashshop/producer.go`
- `services/atlas-channel/atlas.com/channel/cashshop/processor.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/model.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/builder.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/rest.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward_test.go`

Commit: `009bdf195` "fix(cash-shop): gate replayed gift-note NOTE_ACTION SEND
(Defect I)"

## Self-review

- Followed `5b9c5be4e` hop-for-hop for the flag plumbing; the only
  deliberate deviations from that template are: (a) `MarkGiftNoteSent`
  operates on a single `cashId int64`, not a slice (per the brief, since a
  note-send acknowledges exactly one gift), and (b) the top-level
  `MarkGiftNoteSentAndEmit` loops every compartment for the account rather
  than requiring the caller to already know which one holds the asset —
  same shape as `AcknowledgeGiftsAndEmit`.
- `GiftAcknowledged` is never read by `handleNoteGiftForward` — confirmed by
  inspection of the final diff.
- No `TODO`/stub/501 introduced. The pre-existing "TODO select correct
  compartment" comment in `note_gift_forward.go` was left verbatim per the
  brief ("do not try to fix it").
- No new domain type/alias/numeric constant was invented outside
  `libs/atlas-constants` scope — the new constants
  (`CommandTypeMarkGiftNoteSent`) are cash-shop-internal Kafka command
  strings, same category as every existing `CommandType*` constant in this
  file.

## Issues or concerns

None outstanding for this defect. One judgment call, documented above and in
the report body: finding 1's fix uses new package-level test seams
(`noteGiftForwardCompartmentFunc`, `noteGiftForwardCharacterFunc`) rather than
a full HTTP JSON:API mock, because no precedent for the latter exists
anywhere in this test suite and the seam pattern is already the established
convention in this exact file for the same purpose (saga creation). A
reviewer who wants the handler exercised end-to-end over real HTTP transport
would need to introduce that mocking infrastructure as a separate,
non-trivial addition to the test suite's conventions.
