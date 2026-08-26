# bug-cash-shop-live-testing — implementation report

Branch: `task-240-cash-shop-stub-operations` (existing branch, no new worktree/branch created)

## Commits

- `f64b454a3` — `fix(atlas-cashshop): table-qualify purchase-record upsert, fix inventory-slot type math` (Defect A + Defect B + Testing-notes answers)
- `bcb403bd7` — `fix(atlas-ui): unwrap the JSON:API envelope from account PATCH` (Defect C, separable per ruling)

## Defect A — ambiguous `count` column in ON CONFLICT DO UPDATE

`services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator.go`

- Changed `gorm.Expr("count + 1")` to `gorm.Expr(e.TableName() + ".count + 1")`, i.e. `cash_purchase_records.count + 1`, per the brief. Referenced `entity.TableName()` rather than a second hard-coded literal.
- Refactored `Record` to delegate to a new unexported `recordTx` that returns the `*gorm.DB` chain result (instead of only `.Error`), so a test can inspect the generated SQL without duplicating the clause construction. `Record`'s signature and behavior are unchanged.

`services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator_test.go`

- Added `TestRecordConflictUpdateIsTableQualified`, which uses `db.ToSQL(func(tx *gorm.DB) *gorm.DB { return recordTx(tx, ...) })` (GORM's dry-run SQL builder) to assert the generated `ON CONFLICT ... DO UPDATE` right-hand side is `cash_purchase_records.count + 1` and specifically that the old unqualified form (`SET "count"=count + 1`) is absent. This is the only assertion available without a live Postgres, per the brief — sqlite silently accepts either form when executed.

## Defect B — inventory-slot-by-item purchase computes a garbage inventory type

`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`

- Fixed the operator-precedence bug: `inventory.Type(ci.ItemId() - 9110000/1000)` → `inventory.Type((ci.ItemId() - 9110000) / 1000)`.
- Added `ErrInvalidInventoryType` alongside the existing `ErrInsufficientFunds`/`ErrMaxSlots`/`ErrAssetAlreadyReserved` sentinels.
- Added `isValidInventoryType(t inventory.Type) bool`, a membership check against `inventory.Types` (`libs/atlas-constants/inventory/constants.go`) — no existing helper for this existed in the file (`TypeFromItemId` in the constants lib is an unrelated equip-item-id-prefix mapping, not applicable here).
- `PurchaseInventoryIncrease` now rejects an out-of-range computed inventory type **before** the transaction (and therefore the wallet debit) starts: it logs, fires `ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", uuid.Nil)` on the direct producer path (matching the existing `txErr` reject pattern for this function), and returns `ErrInvalidInventoryType`. This applies to both `PurchaseInventoryIncreaseByItemAndEmit` (the defect's actual trigger) and `PurchaseInventoryIncreaseByTypeAndEmit` (defense in depth against a malformed wire type; the "working" case in the bug report is unaffected since its type is always a valid, well-known constant sent by the client).

`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_inventoryincrease_test.go` (new file)

- `TestPurchaseInventoryIncreaseByItemComputesETCType` — itemId `9114000` (the reported ETC slot-expansion item, serial `50200095`, price 6800) resolves to inventory type 4 (`inventory.TypeValueETC`), the purchase succeeds, the wallet is debited by exactly the price, and the outbox carries one `INVENTORY_CAPACITY_INCREASED` event with `inventoryType == 4`.
- `TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem` — itemId `9104000` (computed type `-6`, out of range) is rejected with a `UNKNOWN_ERROR` direct-producer event, **and the wallet balance is asserted unchanged** (the exact loss the tester hit — 6800 cash debited for nothing).
- These tests do not stub `INVENTORY_SERVICE_URL` (the remote character-inventory lookup): `requests.RootUrlFor` errors on the unset env var, and `character.ProcessorImpl.InventoryDecorator` swallows that error, returning the character model with an empty compartment map — exactly the same effective state the live pod logs show (`CompartmentByType` misses → capacity 0). `IncreaseCapacity` is a local Kafka-buffer write (`character/compartment.ProcessorImpl`), not a remote call, so it needs no stub either. This is documented in the test file's header comment.

## Defect C — birthday PATCH toast reports failure after success

`services/atlas-ui/src/services/api/accounts.service.ts`

- Imported `ApiSingleResponse` from `@/types/api/responses` (same source `coupons.service.ts` uses).
- `updateAccountBirthDate` now calls `api.patch<ApiSingleResponse<Account>>(...)` and unwraps `.data` before handing it to `transformAccount`, matching the established idiom at `coupons.service.ts:247`.

`services/atlas-ui/src/services/api/__tests__/accounts.service.test.ts`

- Added a `patch` mock to the `@/lib/api/client` mock (it was missing entirely; only `get` was mocked before).
- Added `describe("accountsService.updateAccountBirthDate", ...)` with a case that resolves `patchMock` against a mocked JSON:API envelope (`{ data: { ...account, attributes: {...} } }`) and asserts `updateAccountBirthDate` resolves successfully with the transformed fields, rather than throwing on `undefined.loggedIn`.

## Testing notes — answered from repo source

Updated `docs/tasks/task-240-cash-shop-stub-operations/bug-cash-shop-live-testing.md`'s "Testing notes" section with the concrete in-game trigger for each command, read from `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` and the corresponding `libs/atlas-packet/cash/serverbound` wire types / `derivation.md`:

- **ENABLE_EQUIP_SLOT** (mode 9) — buying a cash item from the **Equip Slot Increase** category (a distinct item class from the ETC/USE/SETUP inventory-slot coupons). Wire type `ShopOperationEnableEquipSlot`, fname `CCashShop::SendEnableEquipSlotExt`/`OnEnableEquipSlotExt`; handler routes to `RequestEquipSlotIncrease`.
- **REBATE_LOCKER_ITEM** (mode 26) — selecting a purchased item in the Cash Inventory/Locker and choosing **Rebate/Return item**; the client gates it on the account birthday (v83.1) before sending. fname `CCashShop::OnRebateLockerItem`.
- **APPLY_WISHLIST** (mode 35 on the v95+ split; grouped with `SET_WISHLIST` on this tenant's v83.1 per `context.md:34`) — a button in the Cash Shop's **"Best" (best-sellers) window**, per `derivation.md` D2a's decompilation of `CCashShop::ApplyWishListEvent`, sole caller `CCSWnd_Best::OnMouseButton`. Body is confirmed empty (Encode1 only, no other fields).
- **GET_PURCHASE_RECORD** (mode 40) — the precise UI trigger (which button/hover state) is **not** covered by this repo's decompilation notes (`derivation.md`/`context.md`), so I did not invent one; flagged as unverified in the file. What is confirmed from the request/response shape (`ShopOperationGetPurchaseRecord`, `purchaserecord.GetForAccount`) is that it is a per-serial-number "has this account ever bought this item" query. It reads `cash_purchase_records`, the table Defect A failed to write to, so it cannot be meaningfully tested until Defect A is fixed and a purchase has landed.

## Testing

### atlas-cashshop (module-local)

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```
All packages pass, including the two new/changed test files:
```
ok  	atlas-cashshop	0.016s
ok  	atlas-cashshop/cashshop	8.391s
...
ok  	atlas-cashshop/purchaserecord	0.033s
...
(full list: all `ok`, zero failures)
```
Targeted runs during development:
- `go test ./purchaserecord/... -v` — `TestRecordConflictUpdateIsTableQualified` PASS, all existing `TestRecordUpsertsAndCounts`/`TestBackfill*` subtests PASS.
- `go test ./cashshop/... -run TestPurchaseInventoryIncrease -v` — `TestPurchaseInventoryIncreaseByItemComputesETCType` PASS, `TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem` PASS.

### atlas-ui (module-local)

```
cd services/atlas-ui && npm run build   # tsc -b && vite build — type-checks tests too
```
Build succeeded, no type errors.

```
cd services/atlas-ui && npm run test
```
`Test Files 259 passed (259)`, `Tests 2132 passed (2132)`, output pristine (one pre-existing jsdom "Not implemented: navigation to another Document" info line, unrelated to this change and present before it).

## Files changed

- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator.go`
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator_test.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_inventoryincrease_test.go` (new)
- `services/atlas-ui/src/services/api/accounts.service.ts`
- `services/atlas-ui/src/services/api/__tests__/accounts.service.test.ts`
- `docs/tasks/task-240-cash-shop-stub-operations/bug-cash-shop-live-testing.md` (Testing notes section answered)

## Self-review

- Defect A: table-qualification matches the brief exactly; used `entity.TableName()` rather than a second literal, as instructed. Regression test asserts on generated SQL via dry-run, the only method available without live Postgres.
- Defect B: fixed the precedence bug exactly as specified; the reject-before-debit guard was placed at the earliest point (before the transaction opens at all) rather than merely before the wallet `Update` call, which is a slightly stronger fix than the brief's minimum ask but consistent with its intent ("reject...rather than proceeding"). Applied to both By-Item and By-Type entry points since both funnel through the same `PurchaseInventoryIncrease`; this is defense-in-depth, not scope creep, since it's the same one-line guard already required for By-Item.
- Defect C: matches `coupons.service.ts`'s established idiom exactly; added the missing `patch` mock to the test file's `@/lib/api/client` mock rather than leaving `updateAccountBirthDate` untestable.
- Testing notes: GET_PURCHASE_RECORD's exact UI trigger was not fabricated — flagged as unverified per CLAUDE.md's grounding requirement, since no decompilation evidence for its caller exists in this repo's audit docs.
- No `UNKNOWN_ERROR` template-mapping entry was added anywhere, per the ruling.

## Concerns

None outstanding. All three defects reproduce and fix per the brief's own reproduction evidence; module-local build/test is clean for both touched modules.

## Review fix — `<commit-sha>` uint32 underflow guard in `PurchaseInventoryIncreaseByItemAndEmit`

Addressed the single non-blocking finding from `review-bug-cash-shop-live-testing.md`: `ci.ItemId() - 9110000` is `uint32` arithmetic in `PurchaseInventoryIncreaseByItemAndEmit`, so for a hypothetical commodity with `ItemId() < 9110000` the subtraction underflows before truncation to `inventory.Type` (`int8`), and the truncated byte could coincidentally land in `inventory.Types`, silently defeating `isValidInventoryType`. Only reachable via the server's own commodity table, not client input.

`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`

- Added a guard in `PurchaseInventoryIncreaseByItemAndEmit`, immediately after `p.comP.GetById(serialNumber)` and before the `ci.ItemId() - 9110000` subtraction: if `ci.ItemId() < 9110000`, log, fire `ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", uuid.Nil)` on the direct producer path (same pattern as the existing `isValidInventoryType` reject added in `f64b454a3`), and return `ErrInvalidInventoryType` without ever opening the transaction (so the wallet is never touched). This is defense in depth ahead of the subtraction, not a replacement for the existing post-hoc `isValidInventoryType` check in `PurchaseInventoryIncrease`, which is unchanged.
- Ran `gofumpt -l`/`-d` over the file per the brief; it flagged a pre-existing import-grouping issue (`context`/`errors` mixed into the `atlas-cashshop/...` group) unrelated to this edit. Applied `gofumpt -w` to keep the file gofumpt-clean, since I was already touching it.

`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_inventoryincrease_test.go`

- Added `TestPurchaseInventoryIncreaseByItemRejectsItemBelowBaseOffset`, following the shape of `TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem`. Uses itemId `95704` — chosen (via brute-force search over `0..9110000`) so that the pre-fix wrapped/truncated computation `int8(uint32(95704-9110000)/1000)` equals `1` (`inventory.TypeValueEquip`), a *member* of `inventory.Types`. This means the existing post-hoc `isValidInventoryType` check alone would NOT have caught it — only the new lower-bound guard does, which is exactly the scenario the review finding describes. Asserts: `PurchaseInventoryIncreaseByItemAndEmit` returns an error, the wallet balance is unchanged, no outbox entries are written, and exactly one direct-producer `UNKNOWN_ERROR` status event fires.

### Testing

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./cashshop/...
```
```
ok  	atlas-cashshop/cashshop	7.739s
?   	atlas-cashshop/cashshop/commodity	[no test files]
ok  	atlas-cashshop/cashshop/inventory	(cached)
ok  	atlas-cashshop/cashshop/inventory/asset	(cached)
ok  	atlas-cashshop/cashshop/inventory/asset/reservation	(cached)
ok  	atlas-cashshop/cashshop/inventory/compartment	(cached)
```

Targeted run:
```
cd services/atlas-cashshop/atlas.com/cashshop && go test ./cashshop/ -run TestPurchaseInventoryIncreaseByItem -v
```
```
=== RUN   TestPurchaseInventoryIncreaseByItemComputesETCType
--- PASS: TestPurchaseInventoryIncreaseByItemComputesETCType (0.27s)
=== RUN   TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem
--- PASS: TestPurchaseInventoryIncreaseByItemRejectsOutOfRangeItem (0.00s)
=== RUN   TestPurchaseInventoryIncreaseByItemRejectsItemBelowBaseOffset
--- PASS: TestPurchaseInventoryIncreaseByItemRejectsItemBelowBaseOffset (0.00s)
PASS
ok  	atlas-cashshop/cashshop	0.288s
```

`gofumpt -l` over both touched files after the fix: clean (no output).

### Files changed (this round)

- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go`
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor_inventoryincrease_test.go`

### Self-review

- Guard placed at the earliest possible point (before the subtraction, before the transaction opens), matching the brief's "reject before the subtraction can underflow" instruction exactly.
- Both guards kept: the new lower-bound check and the existing `isValidInventoryType` post-hoc check in `PurchaseInventoryIncrease` are both present and independently exercised by tests.
- Scope held to `processor.go` and its test, per the brief's limits; did not touch `services/atlas-configurations/seed-data/templates/` or the already-committed Defect A/B/C fixes.
- Did not touch other gofumpt-flagged files outside this round's diff.

### Concerns

None outstanding.
