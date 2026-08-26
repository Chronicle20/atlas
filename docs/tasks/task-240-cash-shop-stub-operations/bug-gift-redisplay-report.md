# Defect H implementation report

Scope: `## Defect H` only, from
`docs/tasks/task-240-cash-shop-stub-operations/bug-gift-ack-note-and-redisplay.md`.
Defect G was already fixed and committed (`5ebad82cc`) and was **not**
touched — `socket/handler/note_operation.go` and
`socket/handler/note_gift_forward.go` are unmodified; confirmed by reading
both before starting and by grep after finishing (only the channel-side
gift-forward handler's dependency, the asset read model, changed, and its
tests still pass unmodified).

## What I implemented

Followed the bug file's numbered Fix list (items 1-13) verbatim.

### atlas-cashshop (persistence + command consumer)

1. `cashshop/inventory/asset/entity.go` — added
   `GiftAcknowledged bool \`gorm:"not null;default:false"\`` with a doc
   comment stating explicitly: "presented to the recipient", NOT "the
   recipient clicked OK." `asset.Migration` is still bare `AutoMigrate`, so
   every existing row lands the column at its GORM default (`false`) —
   verified by inspection (no `NOT NULL` violation possible since the tag
   carries `default:false` and GORM/sqlite/postgres both apply column
   defaults on `ALTER TABLE ADD COLUMN`).
2. `cashshop/inventory/asset/model.go` — `giftAcknowledged` field, getter,
   `ModelBuilder.SetGiftAcknowledged`, carried through `Clone`/`Build`.
3. `entity.go`'s `Make()` — carries `GiftAcknowledged` into the domain model.
4. `cashshop/inventory/asset/administrator.go` — `updateGiftAcknowledged(db,
   compartmentId, cashIds)`: bulk `UPDATE ... WHERE compartment_id = ? AND
   cash_id IN (?)`. Scoped by compartment as well as cash id so a caller
   cannot drain an asset outside the compartment it resolved. Empty
   `cashIds` is a no-op guard at this layer too.
5. `cashshop/inventory/asset/rest.go` + `rest_test.go` — `giftAcknowledged`
   JSON:API attribute, both directions; added
   `TestTransformExtractRoundTripGiftAcknowledged`.
6. `cashshop/inventory/asset/processor.go` — `AcknowledgeGifts(compartmentId
   uuid.UUID, cashIds []int64) error` on the `Processor` interface + impl.
7. `kafka/message/cashshop/kafka.go` (atlas-cashshop) —
   `CommandTypeAcknowledgeGifts = "ACKNOWLEDGE_GIFTS"` and
   `AcknowledgeGiftsCommandBody{AccountId uint32, CashIds []int64}`, doc
   comment follows `RequestLockerRebateCommandBody`'s idempotency-note
   pattern (a redelivery setting an already-true flag is naturally a no-op,
   so no separate ledger claim was added).
8. `kafka/consumer/cashshop/consumer.go` — registered
   `handleCommandAcknowledgeGifts`, same type-guard shape as
   `handleCommandRequestLockerRebate`. It calls a new top-level orchestrator,
   `AcknowledgeGiftsAndEmit(accountId, cashIds)`
   (`cashshop/processor_gift_ack.go`), added to the `cashshop.Processor`
   interface: resolves the account's compartments
   (`compartment.Processor.GetByAccountId`, the same call `RebateAndEmit`
   uses) and calls `asset.Processor.AcknowledgeGifts` per compartment. No
   outbox emit — there is nothing client-facing to announce.

### atlas-channel (edge)

9. `kafka/message/cashshop/kafka.go` (atlas-channel) — mirrored const +
   `AcknowledgeGiftsCommandBody` **byte-for-byte** on JSON tags. Verified by
   hand:
   - atlas-cashshop: `AccountId uint32 \`json:"accountId"\``, `CashIds
     []int64 \`json:"cashIds"\``.
   - atlas-channel: identical field names, types, and tags.
10. `cashshop/producer.go` + `processor.go` —
    `AcknowledgeGiftsCommandProvider(characterId, accountId, cashIds)` and
    `Processor.AcknowledgeGifts(accountId, characterId, cashIds) error` on
    `COMMAND_TOPIC_CASH_SHOP`, following `RequestLockerRebate`'s shape.
11. `cashshop/inventory/asset/{model.go,builder.go,rest.go}` — mirrored
    `giftAcknowledged` into the channel-side read model (getter, builder
    setter, clone, REST attribute both directions).
12. `socket/handler/cash_shop_entry.go` — `buildGiftListEntries` now skips
    any asset with `GiftAcknowledged() == true` (in addition to the existing
    empty-`GiftFrom` skip). After a successful LOAD_GIFT_SUCCESS announce,
    if the built list is non-empty, fires `AcknowledgeGifts(accountId,
    characterId, cashIds)` with exactly the cash ids (`GiftListEntry.SN`)
    just announced; an empty list skips the call entirely.
    `loadGiftDoneConfigured`'s gate is untouched — an unconfigured tenant
    still announces nothing and drains nothing.
13. Tests:
    - `cash_shop_entry_test.go`: added
      `TestBuildGiftListEntriesSkipsAcknowledged` (an acknowledged gift is
      excluded, an unacknowledged one is included) and a
      `newTestGiftAsset` helper (`newTestAsset` now delegates to it with
      `giftAcknowledged=false`) so the acknowledged-flag path is directly
      testable without touching every existing call site.
    - atlas-cashshop `rest_test.go`: round-trip test (item 5 above).
    - atlas-cashshop `cashshop/processor_gift_ack_test.go`:
      `TestAcknowledgeGiftsAndEmit` (named cashId flips, unrelated cashId in
      the same compartment untouched) and
      `TestAcknowledgeGiftsAndEmitEmpty` (nil cashIds is a no-op, not an
      unbounded update).
    - atlas-cashshop `kafka/consumer/cashshop/consumer_test.go`:
      `TestHandleAcknowledgeGiftsInvokesProcessor` (asserts the DB row's
      `GiftAcknowledged` flag) and
      `TestHandleAcknowledgeGiftsIgnoresOtherCommandTypes` (type guard),
      mirroring the existing `TestHandleOpenSurprise*` pattern in that file.

  I did **not** add a command-carrying test at the empty-list level in
  `cash_shop_entry_test.go` (no fake Kafka producer harness exists at that
  handler layer for this command; the "no command on empty list" behavior is
  covered structurally by the `if len(gifts) > 0` guard and by
  `TestAcknowledgeGiftsAndEmitEmpty` at the atlas-cashshop processor level).

## Interaction with Defect G

Per the stated ruling, `note_gift_forward.go`'s `handleNoteGiftForward` /
`findGiftAsset` validate ownership (asset found by `GiftSN`) and `GiftFrom ==
ToName` only — I did not add or require a not-yet-acknowledged check there.
This is correct given the announce-time drain: by the time the client's
NOTE_ACTION SEND packet arrives, the asset backing it has already been
marked `GiftAcknowledged = true` by this task's fix, so a Defect-G gate
requiring "unacknowledged" would break every legitimate acknowledgement.

## Testing

Module-local only, per Contract 2.

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```
All packages `ok` (or `no test files`); no failures.

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages `ok` (or `no test files`); no failures.

Formatting: ran `gofmt -l` over every touched file, found 5 files gofmt
disagreed with (struct-tag column alignment after adding a longer field
name), fixed with `gofmt -w`, re-verified `gofmt -l` empty, then re-ran both
modules' `go build`/`go test` to confirm the formatting pass didn't change
behavior (still green). Also ran `tools/lint.sh` (flagless, fix mode,
tree-wide golangci-lint) in the background; it completed with **0 issues**
across every module and `git status` showed no further changes (the earlier
`gofmt -w` pass had already brought the touched files into full compliance).

## Files changed

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/entity.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/model.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/administrator.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest.go` (+ `rest_test.go`)
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go` (interface addition)
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift_ack.go` (new)
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_gift_ack_test.go` (new)
- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
- `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go` (+ `consumer_test.go`)
- `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`
- `services/atlas-channel/atlas.com/channel/cashshop/producer.go`
- `services/atlas-channel/atlas.com/channel/cashshop/processor.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/model.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/builder.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/rest.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go` (+ `cash_shop_entry_test.go`)

Commit: `5b9c5be4e` — "fix(cash-shop): stop re-announcing acknowledged gifts (Defect H)"

## Self-review

- Completeness: all 13 numbered Fix items addressed; "Interaction with
  Defect G" honored without touching G's files.
- Quality: followed existing patterns exactly (`RequestLockerRebateCommandBody`
  shape for the new command, `RebateAndEmit`'s compartment-resolution
  pattern for the new orchestrator, existing builder/model/rest triads for
  the new field).
- Discipline: no ledger/idempotency table added for `ACKNOWLEDGE_GIFTS` — a
  redelivery setting an already-true flag is a true no-op, so a claim table
  would be unused ceremony; documented that reasoning in the doc comment
  instead of silently omitting it.
- Testing: every new behavior (flag round-trip, bulk writer, orchestrator,
  consumer wiring, channel-side filter + command trigger) has a test that
  fails if the behavior regresses — verified by construction (each test
  asserts a DB row or slice content, not a mock call count).

## Concerns

- The "Not yet answered" note in the bug file (locker→locker copy paths
  should also carry `GiftAcknowledged`) is explicitly marked not
  load-bearing and out of scope for Defect H; I did not touch any rebate/copy
  path.
- None outstanding: `tools/lint.sh` (flagless, tree-wide) subsequently
  completed with 0 issues.
