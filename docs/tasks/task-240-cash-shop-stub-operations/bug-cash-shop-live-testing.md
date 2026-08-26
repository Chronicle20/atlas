# bug-cash-shop-live-testing

PR: atlas-pr-1426 · branch `task-240-cash-shop-stub-operations` · HEAD at triage `bd7e2e11b`
Environment: namespace `atlas-pr-1426`, tenant `f3fc852d-555a-45b1-80d8-578ea3b9f401`, region GMS, client version **83.1** (confirmed from `atlas-channel` log field `ms.version":"83.1","region":"GMS"` and from the tenant config broadcast on `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`).
Pods read: `atlas-cashshop-9d954445-jxhvc`, `atlas-channel-54b95d899d-q974s`, `atlas-character-7c856c6b59-p9z6v`.

Six symptoms were reported. They resolve to **three** independent defects.

---

## Defect A — every purchase that writes a purchase record fails on Postgres

Covers reported items **2** (gift Bunny Ears), **4** (normal item, Desert Fox Hat), **5** (Friendship Ring / Couple Ring) and **6** (Robot package).

### Reproduced
Yes, four distinct code paths, from live pod logs.

### Observed
`atlas-cashshop` emits, for each purchase:

```
2026/08/26 17:26:48 /app/services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator.go:38 ERROR: column reference "count" is ambiguous (SQLSTATE 42702)
[2.205ms] [rows:0] INSERT INTO "cash_purchase_records" ("id","tenant_id","account_id","serial_number","count","first_at","last_at") VALUES ('42296f46-6652-4e58-90cc-7de2eacc493c','f3fc852d-555a-45b1-80d8-578ea3b9f401',1,10002308,1,'2026-08-26 17:26:48.335','2026-08-26 17:26:48.335') ON CONFLICT ("tenant_id","account_id","serial_number") DO UPDATE SET "count"=count + 1,"last_at"='2026-08-26 17:26:48.335'
```

followed by the four distinct log messages, one per path:

| time | log message | reported item |
|---|---|---|
| 17:26:48 / 17:28:46 | `Unable to record gift purchase for sender [1].` | 2 |
| 17:31:27 | `Unable to record purchase for character [1].` | 4 |
| 17:32:04 / 17:32:34 / 17:33:01 | `Unable to record ring purchase for character [1].` | 5 |
| 17:33:20 | `Unable to record package purchase for character [1].` | 6 |

The failure aborts the enclosing transaction — at 17:31:27 the very next line is
`/app/libs/atlas-outbox/outbox.go:38 ERROR: current transaction is aborted, commands ignored until end of transaction block (SQLSTATE 25P02)` — so the purchase rolls back and a `UNKNOWN_ERROR` status event is emitted instead.

`atlas-channel` then logs, on the *same* `trace.id` for each of those seven purchases:
`Code [UNKNOWN_ERROR] not configured in property [errors]. Defaulting to 99 which will likely cause a client crash.`
which is the "due to an unknown error, the request for cash shop has failed" the tester saw.

### Expected
The upsert succeeds; the purchase commits; the item lands in the locker / is gifted / the ring or package is created.

### Root cause
`services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator.go:35`

```go
"count": gorm.Expr("count + 1"),
```

renders as `ON CONFLICT (...) DO UPDATE SET "count"=count + 1`. In Postgres, inside `ON CONFLICT ... DO UPDATE`, the right-hand side of a `SET` is resolved against **both** the target table and the `excluded` pseudo-relation, so an unqualified `count` is ambiguous (SQLSTATE 42702). The column must be table-qualified.

This is a **regression introduced by this branch** — `purchaserecord/` does not exist at the base commit `32d55cb21`.

It was not caught by the unit tests because `purchaserecord/administrator_test.go` runs against sqlite in-memory (`databasetest`), and sqlite resolves the unqualified name to the target table without complaint.

---

## Defect B — inventory-slot-by-item purchase computes a garbage inventory type

Covers reported item **3** (bought an ETC slot expansion item; client was disconnected with "an error occurred while communicating with the server", cash was taken, no ETC slots awarded).

Reported item 3's other half — "not sure how to test ENABLE_EQUIP_SLOT" — is **not a defect**; it is a testing question, answered at the bottom of this file.

### Reproduced
Yes, from live pod logs.

### Observed
`atlas-cashshop` at 17:29:58, command `REQUEST_INVENTORY_INCREASE_BY_ITEM` `{"currency":1,"serialNumber":50200095}`:

- commodity lookup returns `{"itemId":9114000,"count":1,"price":6800,...}`
- `Character [1] attempting to purchase inventory [-6] increase using currency [1]. Cost is [6800].`
- wallet is debited: `Updating wallet information for account [1]. Credit [89200], ...` (was 96000)
- `Character [1] purchased inventory [-6] increase. New capacity will be [4].`
- 17:29:59 — `{"account_id":1,"name":"Atlas","status":"LOGGED_OUT"}`, i.e. the client dropped immediately after.

The working case for contrast, 17:29:13, `REQUEST_INVENTORY_INCREASE_BY_TYPE` `{"currency":1,"inventoryType":2}` → `inventory [2]`, `New capacity will be [32]`. That path takes the type straight off the wire and is unaffected.

### Expected
`inventoryType` = 4 (`inventory.TypeValueETC`), capacity 24 → 28, client stays connected.

### Root cause
`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:317`

```go
inventoryType := inventory.Type(ci.ItemId() - 9110000/1000)
```

Go operator precedence binds `/` tighter than `-`, so this evaluates `ci.ItemId() - 9110` — not `(ci.ItemId() - 9110000) / 1000`. For itemId `9114000` the intended value is `(9114000-9110000)/1000 = 4` = `inventory.TypeValueETC` (`libs/atlas-constants/inventory/constants.go:15`); the actual value narrows to `-6`.

Downstream, `CompartmentByType(-6)` has no compartment so capacity reads 0, the `slots+amount > 96` guard passes, the wallet is debited, and `IncreaseCapacity` plus the `InventoryCapacityIncreasedStatusEventProvider` are issued for a nonexistent compartment type — which is what the client cannot parse.

This defect **pre-exists on `main`** (identical line at `32d55cb21:.../processor.go:250`); it is not a task-240 regression. It is in the task's blast radius and was reported against this PR, so it is fixed here.

---

## Defect C — birthday update toast reports a failure after a successful PATCH

Covers reported item **1**.

### Reproduced
By code inspection; no server-side error exists for it (the PATCH succeeds, which is why the birthday does change).

### Observed
Toast: `Failed to update birthday: cannot read properties of undefined (reading 'loggedIn')`.

### Expected
Toast: `Birthday updated`.

### Root cause
`services/atlas-ui/src/services/api/accounts.service.ts:128`

```go
const updated = await api.patch<Account>(`${BASE_PATH}/${account.id}`, body, options);
return transformAccount(updated);
```

`api.patch` (`services/atlas-ui/src/lib/api/client.ts:389`) returns the **raw JSON:API envelope** — unlike `api.getOne` (`client.ts:374`), which unwraps `.data`. `atlas-account`'s PATCH handler (`services/atlas-account/atlas.com/account/account/resource.go:45-71`) marshals a full JSON:API single-resource document, so `updated` is `{ data: {...} }`, `updated.attributes` is `undefined`, and `transformAccount`'s first field access — `Number(data.attributes.loggedIn)` at `accounts.service.ts:30` — throws. The mutation has already succeeded server-side by then, hence "seems to work".

The established idiom in this codebase is `api.patch<ApiSingleResponse<T>>(...)` then `.data` — see `services/api/coupons.service.ts:247` and `services/api/events.service.ts:122`. `accounts.service.ts` is the outlier.

This defect **is unrelated to task-240** (the branch changes no TypeScript; `task-facts.sh` reports `ts_changed=false`). It is fixed on this branch because it was reported against this PR and is two lines; call it out in the PR description as an unrelated drive-by.

---

## Fix

### Defect A
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator.go` — table-qualify the increment: `gorm.Expr("cash_purchase_records.count + 1")`. Both Postgres and sqlite accept a table-qualified reference in `ON CONFLICT DO UPDATE SET`, so the existing sqlite tests keep working. Do not hard-code the table name in a second place — `entity.TableName()` (`entity.go:29`) already owns it; reference it.
- `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/administrator_test.go` — the existing sqlite test cannot distinguish the broken form from the fixed one. Add a regression assertion on the **generated SQL** (GORM dry-run / `Session{DryRun: true}` and inspect `Statement.SQL.String()`), asserting the DO UPDATE right-hand side is table-qualified. That is the only assertion available without a live Postgres.

### Defect B
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:317` — `inventory.Type((ci.ItemId() - 9110000) / 1000)`.
- Reject an out-of-range result rather than proceeding: if the computed type is not one of `inventory.Types` (`libs/atlas-constants/inventory/constants.go:19`), emit the reject path instead of debiting the wallet. `PurchaseInventoryIncrease` currently debits before it can discover the compartment does not exist — the tester lost 6800 cash to this. `inventory.Types` / the existing `TypeFromByte`-style helper in that file is the right guard; check what is already there before adding one.
- Add a unit test in `services/atlas-cashshop/atlas.com/cashshop/cashshop/` covering `PurchaseInventoryIncreaseByItemAndEmit` for itemId `9114000` → type 4, and one out-of-range item id → reject with no wallet mutation.

### Defect C
- `services/atlas-ui/src/services/api/accounts.service.ts:116-133` — type the PATCH as `ApiSingleResponse<Account>` and unwrap `.data` before `transformAccount`, matching `coupons.service.ts:247`. Import the type from wherever `coupons.service.ts` imports it.
- `services/atlas-ui/src/services/api/__tests__/accounts.service.test.ts` — add a case asserting `updateAccountBirthDate` resolves against a mocked envelope response `{ data: { id, type, attributes } }`.

## Not yet answered

- **`UNKNOWN_ERROR` has no entry in the cash-shop `errors` table for any template.** `libs/atlas-packet/resolve.go:70` logs `Code [UNKNOWN_ERROR] not configured in property [errors]. Defaulting to 99 which will likely cause a client crash.` The `gms_83_1` template's `socket/writers[215]` (`opCode 0x145`, `CashShopOperation`) `options.errors` map has a lowercase `"unknown_error": 231` (part of the world-transfer key group) but no uppercase `UNKNOWN_ERROR`; `resolve.go` is case-sensitive. In practice the 99 fallback rendered a usable message ("due to an unknown error, the request for cash shop has failed") and did **not** crash the client, so this is cosmetic-plus-log-noise rather than the failure itself. The `UNKNOWN_ERROR` emission pre-exists on `main` (9 occurrences in `cashshop/processor.go` at `32d55cb21`), so it is not a task-240 regression. **Picking the correct per-version byte needs client evidence (IDA / the v83 error string table) — do not guess a value. Out of scope for this fix; raise it separately.**
- Whether the 6800 cash the tester lost to Defect B should be refunded in the PR environment. Not a code question.

## Testing notes for the operations the tester could not exercise

All four commands originate from `atlas-channel`'s `CashShopOperationHandleFunc` (`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`), which dispatches on the mode byte of the `CASHSHOP_OPERATION` (opcode `0x113`) client packet decoded via `libs/atlas-packet/cash/serverbound`.

- **`ENABLE_EQUIP_SLOT`** (mode 9, `services/atlas-channel/.../cash_shop_operation.go:129-140`) — this is **not** the inventory-slot coupon path (`REQUEST_INVENTORY_INCREASE_BY_TYPE`/`_BY_ITEM`, mode 6/7, `processor.go`). The wire type is `ShopOperationEnableEquipSlot` (`libs/atlas-packet/cash/serverbound/shop_operation_enable_equip_slot.go`), sourced from `CCashShop::SendEnableEquipSlotExt`/`OnEnableEquipSlotExt`, carrying `pointType` + `serialNumber` (a commodity serial, exactly like a normal purchase). The handler calls `cashshop.NewProcessor(l, ctx).RequestEquipSlotIncrease(...)`, which is the equip-slot-extension purchase path (`services/atlas-cashshop/.../cashshop/processor_equipslot.go`). **In-game trigger:** in the Cash Shop, purchase a cash item from the **Equip Slot Increase** category (a distinct item class from the ETC/USE/SETUP inventory-slot-expansion coupons) — buying that item is what sends mode 9, not a menu action outside a purchase.
- **`REBATE_LOCKER_ITEM`** (mode 26, `cash_shop_operation.go:174-193`) — wire type `ShopOperationRebateLockerItem` (`libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`), sourced from `CCashShop::OnRebateLockerItem`, carrying a secondary-password/birthday gate (`birthday` on v83, `spw` on v95+) plus the target locker item's 8-byte cash serial (`unk`). The handler first calls `verifySecondaryCredential`, then `RequestLockerRebate`. **In-game trigger:** open the Cash Shop's Cash Inventory/Locker, select an already-purchased (not-yet-withdrawn) item, and choose the **Rebate/Return item** option — the client prompts for the account's birthday (v83.1, this tenant's version) before sending the packet.
- **`APPLY_WISHLIST`** (mode 35 on v95+; this PR's client is v83.1, where the same operation is exposed under the shared `SET_WISHLIST`/`APPLY_WISHLIST` group per `context.md:34`) — the body is empty (`derivation.md` D2a, confirmed against `CCashShop::ApplyWishListEvent @ 0x482ea0`: `Encode1(0x23)` then `SendPacket`, no other encoded fields). **In-game trigger:** its sole caller is `CCSWnd_Best::OnMouseButton` (`derivation.md` D2a) — a button in the Cash Shop's **"Best" (best-sellers) window** that submits the currently-selected wishlist entries to the server. The reply the client expects is `UPDATE_WISHLIST` (mode 98, `derivation.md` D2b), which is what the handler (`cash_shop_operation.go:212-244`) sends back via `cashcb.CashShopWishListUpdateBody`.
- **`GET_PURCHASE_RECORD`** (mode 40, `cash_shop_operation.go:251-267`) — wire type `ShopOperationGetPurchaseRecord` (`libs/atlas-packet/cash/serverbound/shop_operation_get_purchase_record.go`), sourced from `CCashShop::SendGetPurchaseRecord`/`RequestCashPurchaseRecord`, carrying a single `serialNumber`. The handler calls `purchaserecord.NewProcessor(l, ctx).GetForAccount(...)`, which reads `cash_purchase_records` — the very table Defect A failed to write to — and answers with `CashShopPurchaseRecordDoneBody(serialNumber, purchased-flag)`. This repo's decompilation notes (`derivation.md`, `context.md`) do not carry the caller of `CCashShop::SendGetPurchaseRecord`, so the precise UI action (which button/hover state in the shop fires it) is **unverified — do not treat as confirmed**; by the request/response shape, it is a per-serial-number "has this account ever bought this item" query, consistent with the client checking purchase-record eligibility for a specific commodity (e.g. a "one per account" item) before or during a purchase attempt. It cannot be meaningfully tested until Defect A is fixed and at least one purchase for that serial has landed.

## Outcome

Fixed on branch `task-240-cash-shop-stub-operations`.

| commit | content |
|---|---|
| `f64b454a3` | Defect A (table-qualified upsert) + Defect B (inventory-slot type math, range guard before the wallet debit) |
| `bcb403bd7` | Defect C (JSON:API envelope unwrap in `accounts.service.ts`) |
| `6097885e7` | formatting attempt using bare `gofumpt` — **wrong tool**, superseded by `56108b790`; squash this out at PR time |
| `e6dffd8f6` | uint32 underflow guard in the slot math (review finding, user-approved) |
| `a41fd08cd` | report SHA fill-in |
| `56108b790` | correct formatting via `tools/lint.sh` fix mode |

**Review:** `review-bug-cash-shop-live-testing.md` — APPROVED_WITH_FINDINGS, 0 blocking, 1 non-blocking. The one finding (uint32 underflow at the `ci.ItemId() - 9110000` subtraction) was approved by the user and fixed in `e6dffd8f6`. The reviewer verified Defect A by reverting the fix locally and confirming the new test genuinely fails against the old expression, and hand-traced that Defect B's guard runs before `database.ExecuteTransaction` — i.e. before the wallet debit.

**Gate:** `tools/verify.sh --quick --base bd7e2e11b` → **PASS** (exit 0) at `56108b790`. Checks run: go build/vet, go analyzer guards, skill/job id guard, scope guard, producer seam guard, env domain guard, lint & format guard (1 Go module + atlas-ui).

Two earlier gate runs failed on the lint & format guard only. Cause: bare `gofumpt` is **not** the repo's formatting authority — `tools/lint.sh:184` runs `golangci-lint fmt -c .golangci.yml`, i.e. gofumpt **plus** goimports with `local-prefixes: github.com/Chronicle20/atlas`. Module-local `atlas-cashshop/*` imports do not match that prefix and are grouped differently by the two tools. Use `tools/lint.sh` (no flags = fix mode); do not verify Go formatting with `gofumpt -l`.

### Still outstanding

- **The flagless `tools/verify.sh` has NOT been run.** `--quick` skips the bake and `-race` and does not satisfy the "done means verified" bar. Run it before opening/updating the PR.
- **No live re-test yet.** All six reported symptoms remain unconfirmed-fixed until re-tested in `atlas-pr-1426` against tenant `f3fc852d-555a-45b1-80d8-578ea3b9f401` on GMS 83.1, with the images rebuilt to include these commits. Re-test: gift, normal item, ring and package purchases (Defect A); the ETC slot-expansion item 9114000 / serial 50200095 (Defect B); the birthday dialog toast (Defect C).
- The `UNKNOWN_ERROR` template-mapping gap under `## Not yet answered` is untouched and still needs client evidence.
- `GET_PURCHASE_RECORD`'s in-game trigger remains unverified (see the Testing notes above).
