# Cash Shop Stub Operations — Implementation Context

Companion to [plan.md](./plan.md). Everything here was verified against the worktree at plan time; nothing is remembered.

---

## 1. Corrections and decisions made at plan time

These change what the design said. They are settled, with evidence, so the executing phase does not re-litigate them.

### 1.1 `REFRESH_LOCKER` is not bound in ANY GMS template — recipient live-refresh is dropped

Design §3 says the recipient's session is refreshed with `CashShopRefreshLockerBody`, "mode 162, already in the family and already bound in `gms_95`". It is **not bound**:

```
grep -c REFRESH_LOCKER template_gms_83_1.json  -> 0
grep -c REFRESH_LOCKER template_gms_95_1.json  -> 0
```

The `operations` table on gms_95's `CashShopOperation` writer has no `REFRESH_LOCKER` key, so `atlas_packet.ResolveCode` would fall through to its sentinel. Note also that mode 162 on gms_95 is `FRIENDSHIP_SUCCESS`, not `REFRESH_LOCKER` — the "mode 162" in the codec's doc comment is the v83 value.

**Decision:** recipient-directed live refresh is out of scope (Task 14 Step 4). No FR requires it — FR-GIFT-6 requires only that the sender gets a result and that delivery to an offline recipient is durable, which it is: the asset is a committed `cash_assets` row. The recipient sees it on their next locker load. Binding a new mode across nine templates would need a per-version derivation this task has no requirement to fund.

### 1.2 `BUY_OTHER_PACKAGE` is already bound in configuration

Confirmed: `template_gms_95_1.json`'s `CashShopOperationHandle` `operations` table binds `"BUY_OTHER_PACKAGE": 33`. The PRD's framing ("entirely unrouted") is right about Go and wrong about config. The gap is only the missing dispatch arm.

Full gms_95 serverbound mode table, read at plan time:

```
BUY 3, GIFT 4, SET_WISHLIST 5, INCREASE_INVENTORY 6, INCREASE_STORAGE 7,
INCREASE_CHARACTER_SLOT 8, ENABLE_EQUIP_SLOT 10, MOVE_FROM_CASH_INVENTORY 14,
MOVE_TO_CASH_INVENTORY 15, REBATE_LOCKER_ITEM 28, BUY_COUPLE 31, BUY_PACKAGE 32,
BUY_OTHER_PACKAGE 33, BUY_NORMAL 34, APPLY_WISHLIST 35, BUY_FRIENDSHIP 37,
GET_PURCHASE_RECORD 44, BUY_NAME_CHANGE 50, BUY_WORLD_TRANSFER 53
```

### 1.3 `RegisterFileData` already tolerates a missing WZ file

Design §5.2 flagged this as a verification item. Verified — `services/atlas-data/atlas.com/data/data/processor.go:298-303`:

```go
func (p *ProcessorImpl) RegisterFileData(rootDir string, wzFileName string, rf RegisterFunc) Worker {
	return func() error {
		rf(filepath.Join(rootDir, wzFileName))
		return nil
	}
}
```

The register function's return value is discarded and the worker always returns `nil`. A tenant whose dump lacks `CashPackage.img.xml` still boots. The **worker** path (`data/workers/commodity.go:38`) does *not* discard errors, so the cash-package call added there must be logged-and-continued explicitly.

### 1.4 Cash-package ingest rides the existing `Commodity` worker

Design §5.2 proposed a new worker under `data/workers/`. Both `Commodity.img.xml` and `CashPackage.img.xml` live under `Etc.wz`, which the `Commodity` worker already downloads and serializes (`workers/commodity.go:30-36`). A second worker would re-fetch the whole archive for one small file. One extra register call in `Commodity.Run` and one extra `RegisterFileData` line in the `WorkerCommodity` branch of `data/processor.go` is the whole change, and `workers/registry.go` needs no edit.

### 1.5 The whole clientbound result family already exists

Verified in `libs/atlas-packet/cash/clientbound/shop_operation_body.go`. Every result arm this task announces has a constructor and a bound gms_95 mode byte:

| Arm | Constructor | gms_95 mode |
|---|---|---|
| GIFT_SUCCESS / FAILED | `CashShopGiftDoneBody(recipientName string, itemId int32, quantity uint16, nxCashSpent int32)` / `CashShopGiftFailedBody(message string)` | 107 / 108 |
| ENABLE_EQUIP_SLOT_EXT_SUCCESS / FAILED | `CashShopEnableEquipSlotExtSuccessBody(slotIndex uint16, days uint16)` / `...FailedBody` | 117 / 118 |
| REBATE_SUCCESS / FAILED | `CashShopRebateDoneBody(sn int64, amount int32)` / `...FailedBody` | 150 / 151 |
| COUPLE_SUCCESS / FAILED | `CashShopCoupleDoneBody(item CashInventoryItem, recipientName string, itemId int32, quantity uint16)` / `...FailedBody` | 152 / 153 |
| BUY_PACKAGE_SUCCESS / FAILED | `CashShopBuyPackageDoneBody(items []CashInventoryItem, trailingCount uint16)` / `...FailedBody` | 154 / 155 |
| GIFT_PACKAGE_SUCCESS / FAILED | `CashShopGiftPackageDoneBody(recipientName string, packageId int32, unused1, unused2 uint16, nxCashSpent int32)` / `...FailedBody` | 156 / 157 |
| BUY_NORMAL_SUCCESS / FAILED | `CashShopBuyNormalDoneBody(refs []PackedCashItemRef)` / `...FailedBody` | 158 / 159 |
| FRIENDSHIP_SUCCESS / FAILED | `CashShopFriendshipDoneBody(item CashInventoryItem, recipientName string, itemId int32, quantity uint16)` / `...FailedBody` | 162 / 163 |
| PURCHASE_RECORD / FAILED | `CashShopPurchaseRecordDoneBody(goodsSN int32, purchased byte)` / `...FailedBody` | 175 / 176 |
| LOAD_WISHLIST / UPDATE_WISHLIST | `CashShopWishListLoadBody(sns []uint32)` / `CashShopWishListUpdateBody(sns []uint32)` | 92 / 98 |

**No new clientbound codec is required.** At most two serverbound codecs are, both gated on Task 1's derivation.

### 1.6 Errors-table keys used by this plan are all bound

Verified in `template_gms_95_1.json`'s `errors` table: `NOT_ENOUGH_CASH` 3, `CANNOT_GIFT_TO_OWN_ACCOUNT` 6, `INCORRECT_NAME` 7, `CANNOT_GIFT_RECIPIENT_INVENTORY_FULL` 9, `INVENTORY_FULL` 25, `NOT_AVAILABLE_FOR_PURCHASE` 26, `INVALID_BIRTHDAY` 34, `unknown_error` 69. Two templates are known to be thinner — `template_gms_12_1.json` has no `errors` table at all and `template_jms_185_1.json` carries a different set — which is exactly why every failure announce runs the reason through `atlaspacket.CodeConfigured` first and logs before sending anyway (the `transferFailureReasonConfigured` pattern from task-227).

### 1.7 `item.ClassificationRing` exists; a ring *pair* type does not

`libs/atlas-constants/item/constants.go:24` defines `ClassificationRing = Classification(111)` — an item classification, not a pairing type. A sweep of `libs/atlas-constants/` found no ring-pair, marriage, or equip-slot-extension constant. `ring.Type` is therefore a service-local typed string (Task 18), which is the correct placement for a cash-shop domain concept.

---

## 2. Key files, by service

### `atlas-channel` (`services/atlas-channel/atlas.com/channel`)

| File | Role |
|---|---|
| `socket/handler/cash_shop_operation.go` | 497 lines. The dispatcher (`:50`), the 19 mode constants (`:26-46`), the two implemented arms `handleBuyNameChange` (`:243`) / `handleBuyWorldTransfer` (`:329`), the announce helpers, and `isCashShopOperation` (`:479`) |
| `cashshop/processor.go` | The `Processor` interface (`:19`), `RequestPurchase` (`:98`), `resolvePurchaseCurrency` (`:118`) |
| `cashshop/producer.go` | Command providers; `OpenSurpriseCommandProvider` (`:140`) is the transaction-id-carrying template |
| `kafka/message/cashshop/kafka.go` | The mirror of atlas-cashshop's message types |
| `kafka/consumer/cashshop/consumer.go` | `handleStatusEventPurchase` (`:142`), `handleStatusEventError` (`:290`) — the one that answers on the wrong arm today |
| `account/processor.go` | `GetById` (`:54`), `RecordPicAttempt` (`:85`) |
| `character/processor.go` | `GetByName` (`:41`) — the recipient lookup; atlas-cashshop has no equivalent |

### `atlas-cashshop` (`services/atlas-cashshop/atlas.com/cashshop`)

| File | Role |
|---|---|
| `cashshop/processor.go` | `Purchase` (`:100`) — the canonical transaction, including the `rejectEmit` idiom (`:105-113`) that fires rejections on the direct producer path because `message.Emit` only flushes on a nil return |
| `kafka/message/cashshop/kafka.go` | Nine command types, eight status event types, `ErrorEventBody`, `PurchaseEventBody` |
| `kafka/producer/cashshop/producer.go` | `ErrorStatusEventProvider` (`:13`), `PurchaseStatusEventProvider` (`:40`) |
| `kafka/consumer/cashshop/consumer.go` | `InitHandlers` (`:35`) — where new command handlers register |
| `surprise/opening/` | The idempotency-ledger template: `entity.go`, `administrator.go`, `duplicate.go` (dual-driver duplicate-key detection) |
| `wishlist/` | The seven-file domain template: `model/entity/administrator/provider/processor/rest/resource` |
| `cashshop/inventory/asset/` | `cash_assets`; `Create`/`CreateWithCashId` on the processor, `Model.CashId()` as `int64` |
| `main.go:62` | The migration list every new domain appends to |

### `atlas-data` (`services/atlas-data/atlas.com/data`)

| File | Role |
|---|---|
| `commodity/` | The six-file domain template `cashpackage/` copies: `rest/registry/reader/processor/resource` + tests |
| `data/processor.go:175-176` | The `WorkerCommodity` branch; `:298` `RegisterFileData` |
| `data/workers/commodity.go` | The `Etc.wz` worker the cash-package register call joins |
| `main.go:192` | Where `commodity.InitResource` mounts; `cashpackage` mounts beside it |

### `libs/atlas-packet`

| File | Role |
|---|---|
| `cash/clientbound/shop_operation_body.go` | Every result arm constructor and its mode-name constant |
| `cash/serverbound/shop_operation_*.go` | The decoded bodies; doc comments on `gift`, `buy_couple`, `rebate_locker_item` name `ask_SPW` explicitly and describe the v83-int → v95-string substitution in the same wire slot |

---

## 3. Task sizing

Sizes were chosen so an implementer stays under the 120 tool-call budget and a reviewer can read one arm end to end.

**Deliberately larger than the ~6-file guideline, with reasons:**

- **Task 15 (`atlas-data` cashpackage)** — six new files plus three one-line wirings. They are one domain, copied file-for-file from `commodity/`, and splitting a reader from its registry produces two tasks neither of which compiles alone.
- **Task 18 (ring domain)** — six new files, same argument, copied file-for-file from `wishlist/`.
- **Tasks 3, 7, 9, 23** each touch two services, and `plan-lint` flags 7 and 23 for it. Task 3 must span both, because it changes a Kafka message shape that only means anything once both ends agree — a single-service half would land a field nothing reads. Task 7 is a REST resource on one side and a thin four-file client plus one handler arm on the other, both copied file-for-file from `wishlist`/`purchaserecord` siblings; splitting it would leave a resource with no consumer for a review cycle. Tasks 9 and 23 are small on each side. None of the four approaches the 120 tool-call budget.

**Deliberately split where the design treated it as one slice:**

- REBATE, GIFT, PACKAGE, and RING each split into a `atlas-cashshop` task and an `atlas-channel` task (11/12, 13/14, 16/17, 19/20). The service-side transaction carries all the atomicity and idempotency risk and deserves its own review gate; the channel side is decode-gate-emit-announce.
- Purchase records split from their backfill (5/6) — the backfill runs on every boot and its idempotency is a distinct failure mode.
- The two derivation batches (1, 21) are separate because Task 21's answers block only ENABLE_EQUIP_SLOT, the last arm, while Task 1's block the first.

---

## 4. Open items carried into execution

These are real and must not be closed by an invented value.

| Id | Question | Resolved by | Blocks | If UNRESOLVED |
|---|---|---|---|---|
| D1 | The v95 opcode for `CStage::OnSetCashShop` | Task 1 | Task 2 | Record an `n-a` proof in the coverage matrix (FR-V95-3) |
| D2a/D2b | APPLY_WISHLIST's serverbound body and response arm | Task 1 | Task 8 | Implement the no-payload read answering on `LOAD_WISHLIST`, and say it is unconfirmed |
| D3a/D3b | BUY_OTHER_PACKAGE's body and result arm | Task 1 | Task 17 | Route the arm to a typed `GIFT_PACKAGE_FAILED` with a logged warning — never leave it falling through to `Unhandled Cash Shop Operation` |
| D4a | `option` on the package/couple/friendship bodies | Task 1 | Tasks 17, 20 | State in the doc comment that it is ignored and unproven |
| D4b | `oneADay` on `ShopOperationGift` | Task 1 | Task 14 | Same |
| E1 | The `slotIndex` the client expects | Task 21 | Tasks 22, 23 | Land the purchase + persistence; answer `ENABLE_EQUIP_SLOT_EXT_FAILED` rather than invent a slot number |
| E2 | How the extension survives a relog | Task 21 | Task 22 | FR-SLOT-3 goes unsatisfied and is recorded as such |
| R1 | Couple rings: same template for both halves, or distinct male/female? | Task 19, from `Etc.wz`/`Character.wz` ring data | Task 19 | Implement the same-template path; reject the divergent case with a typed failure. **Never** a `+1` or gender-derived offset |
| C1 | Which wallet currency a purchase was drawn from, for the rebate credit | Task 11 | Task 11 | Not deferrable — the task must pick a mechanism (read from the purchase record, or add a currency column to `cash_assets`) and state which |

`flag` on `ShopOperationBuyFriendship` is **not** an open item: `shop_operation_buy_friendship.go:26,83-90` already documents it as a v48-only constant byte the client hard-codes to 1, absent on v83+.

---

## 5. Things that stay true across every task

- The `rejectEmit` idiom in `cashshop/processor.go:105-113` exists because `message.Emit` only flushes its buffer when the wrapped closure returns `nil`. Every new failing transaction must fire its error event on the **direct producer path**, outside the transaction closure. A rejection `mb.Put` inside a failing transaction is silently dropped — this is a real bug that has already been fixed once in this file, and the pattern is there to be copied, not re-derived.
- `transactionId` is minted **at the channel, per click**, never derived from a character id or a timestamp. A genuine second click gets a new id and legitimately charges twice; only a Kafka redelivery replays the same one.
- `uuid.Nil` means "no correlation" on `RequestPurchaseCommandBody`. The ledger must reject it outright rather than let it become a shared uniqueness claim that blocks every uncorrelated command (Task 10).
- Every failure path announces something. A silent return leaves the client's cash shop dialog wedged, and that is the failure mode this whole task exists to remove.
