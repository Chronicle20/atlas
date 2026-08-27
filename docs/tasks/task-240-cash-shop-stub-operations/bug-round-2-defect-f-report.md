# Defect F implementation report

## Summary

Implemented both F1 and F2 as scoped in `bug-round-2-defect-f-brief.md`, committed
together as a single commit on `task-240-cash-shop-stub-operations`.

- **F1** — `services/atlas-channel/.../socket/handler/cash_shop_entry.go` now
  populates each `cashcb.CashInventoryItem.GiftFrom` from `as.GiftFrom()`
  instead of the literal `""`. The buyer's-own-purchase literals in
  `kafka/consumer/cashshop/consumer.go` were left untouched, per the brief.
- **F2** — after the existing `CashShopCashInventoryBody` announce, the entry
  handler now builds one `cashcb.GiftListEntry` per locker asset with a
  non-empty `GiftFrom` (via a new pure helper, `buildGiftListEntries`) and
  announces it with `cashcb.CashShopLoadGiftDoneBody`, gated by a new
  `loadGiftDoneConfigured` predicate that mirrors the established
  `transferFailureReasonConfigured` idiom in `cash_shop_operation.go`:
  `writer.TenantWriterOptions(tenantId, cashcb.CashShopOperationWriter)` +
  `atlaspacket.CodeConfigured(opts, "operations", cashcb.CashShopOperationLoadGiftDone)`.
  This is the "established idiom for skipping an unbound arm" the brief asked
  me to find rather than invent — confirmed by grepping every `CodeConfigured`
  call site in `services/atlas-channel`.

To carry the gift sender/message end to end, added `GiftFrom`/`GiftMessage`
to:
- `services/atlas-cashshop/.../cashshop/inventory/asset/rest.go` — new
  `RestModel` fields, threaded through `Transform`/`Extract`. (`Model` and
  `Entity` already had the fields, per the brief's root-cause chain — only the
  REST layer was missing them.)
- `services/atlas-channel/.../cashshop/inventory/asset/model.go`,
  `rest.go`, `builder.go` — the channel-side asset didn't have these fields
  at all (unlike atlas-cashshop's asset, whose `Model` already carried them).
  Added `giftFrom`/`giftMessage` directly on `asset.Model` (not nested inside
  `item.Model`, matching how the channel-side `RestModel` already flattens
  several `item.Model`-sourced fields onto its own struct) with
  `GiftFrom()`/`GiftMessage()` accessors, `RestModel` JSON fields, and
  `SetGiftFrom`/`SetGiftMessage` builder methods (project Builder pattern, no
  test-only constructors).

## Tests

New tests, run module-local per Contract 2:

1. `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest_test.go`
   — added `TestTransformExtractRoundTripGift` and
   `TestTransformExtractRoundTripNonGift`, following the existing
   `TestTransformExtractRoundTrip` fixture style in the same file.

   ```
   cd services/atlas-cashshop/atlas.com/cashshop && go test ./cashshop/inventory/asset/... -run "TestTransformExtractRoundTripGift|TestTransformExtractRoundTripNonGift" -v
   --- PASS: TestTransformExtractRoundTripGift (0.00s)
   --- PASS: TestTransformExtractRoundTripNonGift (0.00s)
   PASS
   ```

2. `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry_test.go`
   (new file) — `TestBuildGiftListEntries` / `TestBuildGiftListEntriesNoGifts`
   pin the F2 filtering rule (gifted assets produce an entry with sender +
   message; purchased assets are omitted, not emitted with an empty sender).
   `TestLoadGiftDoneConfigured` (bound / unbound / no-options-registered
   subtests) pins the version guard, using the same
   `writer.RegisterTenantWriterOptions` + `opcodes.WriterConfig` pattern
   `character_damage_test.go` already uses for this kind of options-table
   test.

   ```
   cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run "TestBuildGiftListEntries|TestLoadGiftDoneConfigured" -v
   --- PASS: TestBuildGiftListEntries (0.00s)
   --- PASS: TestBuildGiftListEntriesNoGifts (0.00s)
   --- PASS: TestLoadGiftDoneConfigured (0.00s)
       --- PASS: TestLoadGiftDoneConfigured/bound (0.00s)
       --- PASS: TestLoadGiftDoneConfigured/unbound_(template_gms_12_1_/_template_gms_48_1_shape) (0.00s)
       --- PASS: TestLoadGiftDoneConfigured/no_writer_options_registered_at_all (0.00s)
   PASS
   ```

Full module-local build + test, both modules:

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
=> all packages ok (0 failures)

cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
=> all packages ok, including ok atlas-channel/socket/handler 0.900s
```

Formatting (constraints named `tools/lint.sh`, no flags, as the authority —
scoped to the touched files, not a tree-wide sweep):

```
tools/lint.sh --go <9 touched .go files>
lint.sh: OK
```

## Files changed

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/rest_test.go`
- `services/atlas-cashshop/docs/domain.md` (asset `Model`/`ModelBuilder` field
  list)
- `services/atlas-cashshop/docs/rest.md` (asset REST example payloads, 3
  response examples updated; the POST-request example intentionally left
  alone — gift fields are server-derived, never client-supplied on create)
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/builder.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/model.go`
- `services/atlas-channel/atlas.com/channel/cashshop/inventory/asset/rest.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry_test.go` (new)

## Self-review

- Confirmed the buyer's-own-purchase `GiftFrom: ""` literals in
  `kafka/consumer/cashshop/consumer.go` (3 call sites) were **not** touched —
  brief explicitly scopes only the entry-burst projection.
- Confirmed the packet layer (`CashShopLoadGiftDoneBody`, `GiftListEntry`,
  `CashShopOperationLoadGiftDone`) was not modified — used as-is.
- Confirmed the version guard actually skips the announce (not just logs a
  warning) when `LOAD_GIFT_SUCCESS` is unconfigured — `loadGiftDoneConfigured`
  gates the entire `if` block around the announce call.
- Confirmed no `libs/atlas-packet` changes were made, per the brief.
- `buildGiftListEntries` was factored out as a pure function specifically so
  F2's filtering rule is testable without standing up the full entry
  handler's account/character/buddylist/minigame/compartment/storage/
  wishlist/wallet dependency chain — consistent with how `giftRejectionReason`
  is tested in isolation in `cash_shop_gift_test.go`.
- No new domain constants were needed; `GiftListEntry`/`CashShopOperationLoadGiftDone`
  already existed in `libs/atlas-packet`.

## Concerns

None. Both modules build and test clean; formatting gate passes on the
touched files.
