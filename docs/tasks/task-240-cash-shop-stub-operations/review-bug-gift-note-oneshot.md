# review-bug-gift-note-oneshot

Reviewer: task-reviewer (Sonnet 5)
Scope: commit `009bdf195` only ("fix(cash-shop): gate replayed gift-note NOTE_ACTION SEND (Defect I)").
Excluded (already reviewed/approved elsewhere): `5ebad82cc`, `5b9c5be4e` — see
`docs/tasks/task-240-cash-shop-stub-operations/review-bug-gift-ack-and-redisplay.md`


Requirement: `## Defect I` in
`docs/tasks/task-240-cash-shop-stub-operations/bug-gift-ack-note-and-redisplay.md`,
including its "Known limitation to document in code, not to fix" and "Also in
this unit — finding 1" subsections.
Implementer report: `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-note-oneshot-report.md`.

## Method

`git show --stat 009bdf195` for the file list, then hunk-by-hunk `git show` per
file/pair (atlas-cashshop ↔ atlas-channel side by side), a full `Read` of the
two files that matter most for correctness (`note_gift_forward.go` and its
test), a build + full package test run for both services, and two live
mutation probes against the working tree (temporarily neutered each new gate,
confirmed the corresponding test goes RED, then restored the file from the
pre-edit backup and confirmed `git status`/`git diff` showed no residue).

## 1. `MARK_GIFT_NOTE_SENT` command body — byte-identical JSON tags

Verified independently (not taking the implementer's hand-check on trust):

- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`:
  `type MarkGiftNoteSentCommandBody struct { AccountId uint32 \`json:"accountId"\`; CashId int64 \`json:"cashId"\` }`
- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`:
  identical field names, types, and JSON tags.
- `CommandTypeMarkGiftNoteSent = "MARK_GIFT_NOTE_SENT"` is the same string
  literal in both files.

**PASS.**

## 2. `GiftNoteSent` round-trip, hop for hop against `GiftAcknowledged`'s template

Traced every hop that `5b9c5be4e` established for `GiftAcknowledged` and
confirmed `GiftNoteSent` is present at each one:

atlas-cashshop:
- `entity.go:80` — `GiftNoteSent bool \`gorm:"not null;default:false"\`` field, with a doc comment covering the known async-race limitation.
- `entity.go` `Make` (entity→model, `entity.go:110`) — `SetGiftNoteSent(e.GiftNoteSent)`.
- `model.go:25` — struct field; `model.go:101-105` — `GiftNoteSent()` getter; `Clone` (`model.go:126`) carries it; `ModelBuilder` struct (`model.go:146`), `SetGiftNoteSent` setter (`model.go:233-236`), and `Build()` (`model.go:257`) all carry it — **both** clone paths (`Clone` and the builder's own `Build`) are covered.
- `administrator.go:112-120` — `updateGiftNoteSent(db, compartmentId, cashId)`, scoped update, unknown cashId is a no-op (not an error), matching `updateGiftAcknowledged`'s shape.
- `processor.go:265-267` (asset package) — `Processor.MarkGiftNoteSent(compartmentId, cashId)`.
- `cashshop/processor_gift_note_sent.go` (new) — `MarkGiftNoteSentAndEmit(accountId, cashId)`, loops the account's compartments, logs+continues on a partial failure (matches `AcknowledgeGiftsAndEmit`'s pattern) — registered on the top-level `Processor` interface (`processor.go:120`).
- `rest.go` — `RestModel.GiftNoteSent \`json:"giftNoteSent"\`` field; `Transform`/`Extract` both directions.
- `rest_test.go` — `TestTransformExtractRoundTripGiftNoteSent` round-trips `Transform → Extract` and asserts equality; genuinely exercises the new field (confirmed by reading the test body, not just its name).
- `kafka/consumer/cashshop/consumer.go:112-128` — `handleCommandMarkGiftNoteSent`, registered in the same topic-registration block as `handleCommandAcknowledgeGifts` (verified by reading the surrounding loop, not just the diff hunk — both calls are inside the same `t` iteration).
- `consumer_test.go` — `TestHandleMarkGiftNoteSentInvokesProcessor` (named cashId flips, sibling untouched) and `TestHandleMarkGiftNoteSentIgnoresOtherCommandTypes` (type guard).

atlas-channel:
- `cashshop/inventory/asset/model.go` — `giftNoteSent` field + `GiftNoteSent()` getter.
- `cashshop/inventory/asset/builder.go` — `modelBuilder.giftNoteSent`, `CloneModel` carries it, `SetGiftNoteSent`, `Build()` carries it.
- `cashshop/inventory/asset/rest.go` — `RestModel.GiftNoteSent \`json:"giftNoteSent"\``, `Transform`/`Extract` both directions.
- `cashshop/producer.go` — `MarkGiftNoteSentCommandProvider(characterId, accountId, cashId)`.
- `cashshop/processor.go` — `Processor.MarkGiftNoteSent(accountId, characterId, cashId)` forwards the command on `EnvCommandTopic`.
- `socket/handler/note_gift_forward.go` — reads `GiftNoteSent` off the channel-side asset model via `findGiftAsset`, gates on it, and fires `MarkGiftNoteSent` on success.

No hop is missing on either side. **PASS.**

## 3. `handleNoteGiftForward` — ownership + `GiftFrom == ToName` intact, `GiftAcknowledged` NOT consulted, new gate additive

Read the full post-commit file
(`services/atlas-channel/atlas.com/channel/socket/handler/note_gift_forward.go`).
Gate order in `handleNoteGiftForward` (lines 111-150):

1. Load compartment (line 115) — unchanged in shape, only routed through the new `noteGiftForwardCompartmentFunc` test seam.
2. `findGiftAsset` — `!found` → reject (line 122-125) — unchanged.
3. `giftFrom == "" || giftFrom != sp.ToName()` → reject (line 126-129) — **the pre-existing anti-tamper gate, untouched**.
4. `giftNoteSent` → reject (line 130-133) — **the new gate, added after #3, not in place of it**.
5. Resolve gifter by name (line 135) — unchanged.
6. Build + create the one-step `saga.NoteSend` (lines 141-145) — unchanged (still no `DestroyAsset` step).
7. `noteGiftForwardMarkSentFunc` fired only on saga-creation success (line 147-149).

`grep -n "GiftAcknowledged"` over `note_gift_forward.go` and `note_operation.go`
returns exactly one hit — a comment (`note_gift_forward.go:103`) explicitly
stating `GiftAcknowledged` is "deliberately NOT consulted here." No code path
reads `GiftAcknowledged`. `note_operation.go` is untouched by this commit
(absent from `git show 009bdf195 --stat`), so Defect G's dispatch (`giftFlag==1`
routes here before the Note-item gate) is unchanged.

**PASS.**

## 4. Async-race limitation documented at the gate, not fixed with a new synchronous seam

`note_gift_forward.go:102-110`, directly above `handleNoteGiftForward`:

> "Known limitation, not fixed here: MarkGiftNoteSent is an asynchronous Kafka
> round trip, so two acknowledgement packets racing inside that window can
> both pass this gate before either write lands. This narrows the exposure
> from unbounded to a single race."

This is on the gate itself (the doc comment immediately preceding the function
containing the gate), matches the brief's required content, and no
synchronous write was introduced anywhere in the diff — `MarkGiftNoteSent` is
fired via the existing async Kafka producer path (`producer.go`), identical in
shape to every other command in this file. **PASS.**

(The same limitation is also documented, redundantly but harmlessly, on the
`GiftNoteSent` entity field in atlas-cashshop's `entity.go` — not a defect,
just duplication.)

## 5. Finding 1 — is the rewritten test genuinely load-bearing?

Read `note_gift_forward_test.go` in full. The rewrite adds three
package-level test seams (`noteGiftForwardCompartmentFunc`,
`noteGiftForwardCharacterFunc`, `noteGiftForwardMarkSentFunc`) alongside the
pre-existing `noteGiftForwardSagaCreateFunc`, and calls `handleNoteGiftForward`
directly with a real, codec-decoded `*notesb.OperationSend` (built via
`notesbOperationSend`, which round-trips a hand-built wire packet through the
actual `OperationSend.Decode`). `TestHandleNoteGiftForward_GiftFromMismatch`
and `TestHandleNoteGiftForward_AlreadySent` assert both `sagaCreated == false`
and `markSentCalled == false` via the seam recorders.

Did not take the implementer's RED claim on trust. Ran the actual mutation
myself against the working tree:

- Neutered the `GiftFrom == ToName` gate (`if false && (giftFrom == "" ||
  giftFrom != sp.ToName())`) and ran
  `go test ./socket/handler/... -run TestHandleNoteGiftForward_GiftFromMismatch`:
  **FAIL** — `note_gift_forward_test.go:161: expected no saga to be created on
  a giftFrom/toName mismatch`.
- Neutered the `giftNoteSent` gate (`if false && giftNoteSent`) and ran
  `go test ./socket/handler/... -run TestHandleNoteGiftForward_AlreadySent`:
  **FAIL** — `note_gift_forward_test.go:182: expected no saga to be created
  when GiftNoteSent is already true`.
- Restored the file from a pre-mutation backup after each probe;
  `git status --porcelain services/` and `git diff -- services/` confirmed no
  residue survived into the tree.

Both rewritten tests are genuinely load-bearing — they fail exactly the way
finding 1 required, closing the original gap (the old
`TestFindGiftAsset_GiftFromMismatch` never called the handler and passed with
the gate deleted).

**PASS.**

### Judging the scope decision (seam tests vs. HTTP JSON:API mocking)

The implementer's rationale (report, "Why this approach instead of full HTTP
JSON:API mocking") is that no precedent exists anywhere in the
`atlas-channel` test suite for mocking a JSON:API compartment with nested
asset/item `included` relationships over `httptest`, and that the
package-level test-seam pattern is already the established convention in this
exact file (`noteGiftForwardSagaCreateFunc`, and elsewhere in the package:
`scriptedItemSagaCreateFunc`, `dueyCouponSagaCreateFunc`,
`remoteMerchantSagaCreateFunc`, `npcItemUseSagaCreateFunc`).

Assessed on the merits, not on the claim: the seam tests exercise the real
`handleNoteGiftForward` function body — the exact code path
`NoteOperationHandleFunc` calls in production — with a real, wire-decoded
packet. The only thing mocked is the *transport* to two upstream services
(cash-shop compartment fetch, character-by-name fetch) and the *emission* of
two side effects (saga creation, MarkGiftNoteSent). This is a legitimate unit
boundary, not a re-derivation of the gate logic in the test (which is exactly
what finding 1 flagged the *old* test for doing). My mutation probes (above)
confirm the tests fail for the right reason. Judged as reasonable scope, not
a shortcut that reintroduces finding 1's problem.

## Additional checks

- `go build ./...` clean in both `services/atlas-channel/atlas.com/channel`
  and `services/atlas-cashshop/atlas.com/cashshop`.
- Full package test runs green: `atlas-channel/cashshop/...`,
  `atlas-channel/socket/handler`, `atlas-cashshop/cashshop/...`,
  `atlas-cashshop/kafka/...`.
- `handleCommandMarkGiftNoteSent` registration verified by reading the
  surrounding `consumer.go` loop (not just the diff hunk) — it sits in the
  same per-topic registration block as `handleCommandAcknowledgeGifts`, no
  misrouting to a different topic.
- No `TODO`/stub/placeholder introduced; the pre-existing "TODO select correct
  compartment" comment (`note_gift_forward.go:113-114`) is left verbatim, as
  the brief for Defect G/H required.
- No new domain type/constant collides with anything in
  `libs/atlas-constants/` — `CommandTypeMarkGiftNoteSent` is a cash-shop
  Kafka command string, the same category as the file's other constants.

## Not evaluable

- Whether `asset.Migration`'s `AutoMigrate` actually lands `gift_note_sent`
  defaulting to `false` on pre-existing rows in a live Postgres database was
  not independently verified in this review (no live database available in
  this surface) — the doc comment claims it, and this is the identical
  mechanism `GiftAcknowledged` used in `5b9c5be4e`, which was already reviewed
  and approved on that point. Deferred to that precedent rather than
  re-verified here.
- The genuine cross-service Kafka round trip (real broker, both consumers
  live) was not exercised — this is integration-test territory outside a
  single commit's unit-test surface, and is the same class of gap the async-
  race limitation itself documents as inherent to the design, not an
  implementation defect.

## Verdict

All four seam checks (byte-identical JSON tags, full round-trip, gate
ordering, and the documented limitation) pass with evidence. Finding 1 is
genuinely fixed — verified by independent mutation, not by trusting the
implementer's claim. No blocking defects found in this commit.

APPROVED.
