# Review — bug-round-2 Defect F (commit ae4042260)

Scope: commit `ae4042260` only ("fix(cash-shop): surface gift sender/message in the
locker and entry burst (Defect F)"). Commits `648310b18`/`59916a651` are out of
scope (already approved in `review-bug-round-2.md`).

Requirement: `docs/tasks/task-240-cash-shop-stub-operations/bug-round-2-defect-f-brief.md`
Symptom: `docs/tasks/task-240-cash-shop-stub-operations/bug-cash-shop-live-testing-round-2.md` § Defect F

## Verdict

APPROVED

## Findings

None blocking, none non-blocking.

## Checklist against the brief

### 1. GiftFrom/GiftMessage round-trip end to end

Traced the full chain by hand:

- **Persisted entity → atlas-cashshop model**: `services/atlas-cashshop/.../inventory/asset/entity.go:77-78`
  already builds the domain model with `SetGiftFrom(e.GiftFrom).SetGiftMessage(e.GiftMessage)`
  (pre-existing, unchanged by this commit — confirmed root-cause-chain step 1 was
  already true).
- **atlas-cashshop REST Transform/Extract**: `services/atlas-cashshop/.../inventory/asset/rest.go`
  diff adds `GiftFrom`/`GiftMessage` to `RestModel` (json tags `giftFrom`/`giftMessage`),
  carries both through `Transform` (`GiftFrom: a.GiftFrom()`, `GiftMessage: a.GiftMessage()`)
  and `Extract` (`giftFrom: rm.GiftFrom`, `giftMessage: rm.GiftMessage`). New test
  `rest_test.go:TestTransformExtractRoundTripGift` asserts a non-empty sender/message
  survives Transform→Extract; `TestTransformExtractRoundTripNonGift` pins that a
  purchased asset round-trips empty rather than leaking a stale value. Both tests pass
  (`go test ./cashshop/inventory/asset/...` → ok).
- **atlas-channel asset model/builder/rest**: `services/atlas-channel/.../cashshop/inventory/asset/{model,builder,rest}.go`
  mirror the same pattern — `Model.giftFrom/giftMessage` fields with `GiftFrom()`/`GiftMessage()`
  accessors, `modelBuilder.SetGiftFrom/SetGiftMessage`, `CloneModel` and `Build()` both
  carry the two fields, and `rest.go`'s `Transform`/`Extract` carry the REST fields
  through symmetrically with the cashshop side.
- **cash_shop_entry.go projections**: `CashInventoryItem.GiftFrom` now reads
  `as.GiftFrom()` (line 96, was the literal `""`), and `buildGiftListEntries`
  (lines 151-165) builds one `cashcb.GiftListEntry{SN, ItemId, BuyCharacterName:
  as.GiftFrom(), Text: as.GiftMessage()}` per asset with a non-empty sender,
  skipping the rest.

No break found anywhere in the chain. Every link either changed correctly or was
already correct and left alone (verified via `git show ae4042260 -- <file>` for
each hop plus reading pre-existing code the diff depends on).

### 2. Unbound-arm guard actually protects the announce

`loadGiftDoneConfigured` (cash_shop_entry.go:174-182) resolves
`writer.TenantWriterOptions(t.Id(), cashcb.CashShopOperationWriter)` — the same
`writer` name (`"CashShopOperation"`, `shop_operation_result.go:13`) and the same
per-tenant table (`options_registry.go`, populated at `main.go:438` via
`RegisterTenantWriterOptions(t.Id(), tenantCfg.Socket.Writers)`) that
`CashShopLoadGiftDoneBody` itself resolves against
(`atlas_packet.WithResolvedCode("operations", CashShopOperationLoadGiftDone, ...)`,
`shop_operation_body.go:549`). It then calls `atlaspacket.CodeConfigured(opts,
"operations", cashcb.CashShopOperationLoadGiftDone)` — exactly the `property`/`key`
pair the writer resolves. Confirmed by hand against the seed templates:

- `template_gms_12_1.json`: the `CashShopOperation` writer entry has **no** `options`
  key at all → `TenantWriterOptions` returns `ok=false` → guard returns `false`.
- `template_gms_48_1.json`: has `options.operations` but no `LOAD_GIFT_SUCCESS` key
  → `CodeConfigured` returns `false`.
- `template_gms_61_1.json`: has `options.operations.LOAD_GIFT_SUCCESS: 49` →
  `CodeConfigured` returns `true`.

The guard is reached (`cash_shop_entry.go:105`) strictly before the announce call
(line 107), and returning `false` skips the announce entirely rather than calling
`Announce` and letting `ResolveCode` fall through to its 99 sentinel. This closes
the exact defect the brief warned about. `TestLoadGiftDoneConfigured`
(`cash_shop_entry_test.go:85-124`) pins bound/unbound/unregistered cases.

### 3. GiftListEntry fixed-width fields cannot overflow

`GiftListEntry.EncodeBytes` (`shop_operation_result_gift.go:39-46`, unchanged by
this commit — pre-existing) calls `model.WritePaddedString(w, m.BuyCharacterName,
13)` and `model.WritePaddedString(w, m.Text, 73)`. `WritePaddedString`
(`libs/atlas-packet/model/padded_string.go:10-17`) truncates when
`len(str) > number` (`w.WriteByteArray([]byte(str)[:number])`) rather than
overflowing the fixed 98-byte `GW_GiftList` record. The DB columns
(`entity.go:52` `GiftFrom string `gorm:"size:13"``, `entity.go:57` `GiftMessage
string `gorm:"size:73"``) already constrain the values at the persistence layer to
the same widths, but the encode path is independently safe even if a value longer
than the column bound ever reached it (e.g. via direct REST write). No overflow
risk.

### 4. Entry burst arm order unchanged / not displaced

Read `cash_shop_entry.go:69-143` in full. Order is: `CashShopOpenWriter` (line 69)
→ `CashShopOperationWriter`/`CashShopCashInventoryBody` (line 100) → **new**
guarded `CashShopLoadGiftDoneBody` (lines 105-111) → `CashShopWishListLoadBody`
(line 122) → `CashQueryResultWriter`/`NewCashQueryResult` (line 132) →
`cashshop.NewProcessor(...).Enter` (line 139). The new announce is inserted
strictly after the cash-inventory announce and before the wishlist load, matching
the brief; none of the pre-existing arms were reordered, removed, or had their
call sites altered beyond the `GiftFrom` field literal at line 96.

### 5. `GiftFrom: ""` literals in `kafka/consumer/cashshop/consumer.go` left alone

`git show ae4042260 -- services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
produces an empty diff — the file is untouched by this commit. Correct per the
brief: those are the buyer's own just-purchased items (no sender), and this
commit's diff stat (`git show --stat ae4042260`) confirms the file is absent from
the 9 changed files.

## Other checks

- **Build/tests**: `go build ./...` and `go test ./cashshop/inventory/asset/...`
  in atlas-cashshop, and `go build ./...` and `go test ./socket/handler/...
  ./cashshop/...` in atlas-channel — both green, all packages `ok`.
- **Docs**: `services/atlas-cashshop/docs/domain.md` and `docs/rest.md` updated
  with the two new fields and their example values, consistent with the code
  change.
- **Builder pattern / no test-only constructors**: `cash_shop_entry_test.go`'s
  `newTestAsset` helper uses `asset.NewModelBuilder(...).SetGiftFrom(...).Build()`,
  not a bespoke constructor; `rest_test.go` uses the pre-existing
  `NewBuilder(...).SetGiftFrom(...).SetGiftMessage(...).Build()`. Compliant with
  repo convention.
- **Scope**: diff touches exactly the 9 files listed in the brief's file
  inventory (cashshop rest.go + test, channel asset rest/model/builder, channel
  handler + test, two docs files). No unrelated changes found.

## Not evaluable

None — full review surface was reachable from the commit diff plus the files it
structurally depends on (entity.go, model.go, resolve.go, options_registry.go,
padded_string.go, shop_operation_result_gift.go, the three seed templates).
