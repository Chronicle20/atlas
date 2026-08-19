# Cash Shop Stub Operations — Design

Task: `task-240-cash-shop-stub-operations`
PRD: [prd.md](./prd.md) (v1, approved)
Tree state at design time: worktree branch `task-240-cash-shop-stub-operations`, branched from `d9ec287b8`.

---

## 0. What changed between PRD and design

Four PRD open questions are **settled from evidence already in this repository** — no
IDB session needed. They are recorded here so the plan phase does not re-litigate them.

| PRD §9 | Question | Settled answer | Evidence |
|---|---|---|---|
| 2 | `spw` semantics | It is `ask_SPW` — the account's **PIC** (secondary password). It occupies the same leading wire slot as the v83 `birthday` int; v95+ replaced the int with a length-prefixed string. | Doc comments on `libs/atlas-packet/cash/serverbound/shop_operation_gift.go:17-22`, `shop_operation_buy_couple.go:17-20`, `shop_operation_rebate_locker_item.go:17-21` — all name `ask_SPW` explicitly and all describe the v83-int → v95-string substitution in the same slot. |
| 3 | `ShopOperationRebateLockerItem.unk` (uint64) | It is the **locker cash serial** (`GW_ItemSlotBase::liCashItemSN`), i.e. `cash_assets.CashId`. | `shop_operation_rebate_locker_item.go:18-21`: "The trailing 8-byte locker serial (EncodeBuffer 8) is identical across versions, modeled here as the uint64 unk". `asset.Model.CashId()` is `int64` (`services/atlas-cashshop/.../inventory/asset/model.go:12`). |
| 6 | BUY_NORMAL currency derivation | Same answer as task-227's `BUY_NAME_CHANGE`: `isPoints=false, currency=0`. On v83+ the whole body is `serialNumber` (4 bytes) — there is nothing else on the wire. The v48-only legacy path adds spw/name/message, still no currency. | `shop_operation_buy_normal.go:23-28` and its `Decode`. |
| 7 (partial) | `BUY_OTHER_PACKAGE` (31) vs `BUY_PACKAGE` (30) | **Strong hypothesis, still requires IDB confirmation:** mode 31 is *gift a package to another character*. The client's result family carries a matched `GIFT_PACKAGE_SUCCESS` / `GIFT_PACKAGE_FAILED` pair that no other serverbound arm can produce, and its success body takes a recipient name. | `shop_operation_body.go:41,42,61,62,577` — `CashShopGiftPackageDoneBody(recipientName string, packageId int32, unused1, unused2 uint16, nxCashSpent int32)`. Every other `*_FAILED`/`*_DONE` pair maps 1:1 to a serverbound arm; GIFT_PACKAGE is the only orphan, and BUY_OTHER_PACKAGE is the only unrouted arm. |

Two further PRD statements are **corrected**:

- **`BUY_OTHER_PACKAGE` is already routed in configuration.** Every GMS template's
  `CashShopOperationHandle` `operations` table already binds it (`gms_95` = 33,
  `gms_83` = 31). The gap is purely the missing Go dispatch arm, not a config gap.
- **The entire clientbound result family already exists.** `shop_operation_body.go`
  provides `CashShopGiftDoneBody`, `CashShopGiftFailedBody`, `CashShopCoupleDoneBody`,
  `CashShopCoupleFailedBody`, `CashShopFriendshipDoneBody`, `CashShopFriendshipFailedBody`,
  `CashShopBuyPackageDoneBody`, `CashShopBuyPackageFailedBody`, `CashShopGiftPackageDoneBody`,
  `CashShopGiftPackageFailedBody`, `CashShopBuyNormalDoneBody`, `CashShopBuyNormalFailedBody`,
  `CashShopRebateDoneBody`, `CashShopRebateFailedBody`, `CashShopPurchaseRecordDoneBody`,
  `CashShopPurchaseRecordFailedBody`, `CashShopEnableEquipSlotExtSuccessBody`,
  `CashShopEnableEquipSlotExtFailedBody`, and `CashShopWishListLoadBody`. `gms_95`'s
  `CashShopOperation` writer options bind a mode byte for every one of them, and the
  `errors` table binds every reason code this task needs. **No new clientbound codec is
  required.** This collapses a large slice of the PRD's `libs/atlas-packet` impact.

The remaining `libs/atlas-packet` work is at most **one new serverbound struct**
(`ShopOperationBuyOtherPackage`) and **possibly one more** (`ShopOperationApplyWishlist`,
if mode 33/35 carries a payload). Both are gated on derivation (§9).

---

## 1. Architecture

### 1.1 The shape every arm takes

All ten arms collapse onto **two skeletons**. Choosing between them per arm is the
single most load-bearing decision in this design.

**Skeleton A — mutating arm (currency moves, or durable state changes):**

```
channel handler arm
  → decode serverbound struct
  → secondary-credential gate (§2)
  → cheap local rejections that need no service state (self-gift, unresolvable commodity)
  → cashshop.Processor.Request<X>(...)  — Kafka command, transactionId minted here
       → atlas-cashshop consumer
       → one DB transaction: validate → debit wallet → mutate locker/ring/slot
       → outbox: success StatusEvent   |  direct producer: typed failure StatusEvent
  → channel kafka/consumer/cashshop handler
  → session.Announce(CashShopOperationWriter)(<arm>DoneBody | <arm>FailedBody)
```

**Skeleton B — read-only arm (no state changes):**

```
channel handler arm
  → decode serverbound struct
  → REST read against the owning service (atlas-cashshop)
  → session.Announce(CashShopOperationWriter)(<arm>DoneBody | <arm>FailedBody)
```

**The rule: currency-moving or durably-mutating arms take Skeleton A; pure reads take
Skeleton B.** Skeleton B is not a shortcut — a read has no transaction to protect, no
idempotency requirement, and no redelivery hazard, so routing it through Kafka would add
a topic round-trip and a correlation-id dance for nothing. The repository already mixes
both: `SET_WISHLIST` reads/writes over REST (`cash_shop_operation.go:69-88`) while every
`BUY_*` arm goes over Kafka.

Arm assignment:

| Arm | Mode (v83/v95) | Skeleton | Rationale |
|---|---|---|---|
| GIFT | 4 / 4 | A | debits wallet, writes recipient's locker |
| ENABLE_EQUIP_SLOT | 9 / 10 | A | debits wallet, writes character state |
| BUY_NORMAL | 20 / 34 | A | debits wallet, writes locker |
| REBATE_LOCKER_ITEM | 26 / 28 | A | credits wallet, deletes locker asset |
| BUY_COUPLE | 29 / 31 | A | debits wallet, writes two lockers + pair |
| BUY_PACKAGE | 30 / 32 | A | debits wallet, writes N locker rows |
| BUY_OTHER_PACKAGE | 31 / 33 | A | debits wallet, writes recipient's locker |
| APPLY_WISHLIST | 33 / 35 | B | reads the stored wishlist |
| BUY_FRIENDSHIP | 35 / 37 | A | same as BUY_COUPLE |
| GET_PURCHASE_RECORD | 40 / 44 | B | reads purchase history |

### 1.2 Service responsibilities

| Service | Owns | Does not own |
|---|---|---|
| `atlas-channel` | packet decode; mode dispatch; the secondary-credential gate; minting `transactionId`; announcing every result | any validation that needs locker/wallet state — that is re-done service-side |
| `atlas-cashshop` | wallet debit/credit; locker assets; **ring pairs**; **purchase records**; atomicity of every multi-row purchase | the equip-slot capacity effect |
| `atlas-character` | equip-slot extension record and its effect on equipped capacity | anything cash-shop-transactional |
| `atlas-data` | **cash package catalogue** ingested from `Etc.wz/CashPackage.img` | package pricing (that is the package commodity's own `Price`) |
| `atlas-account` | `BirthDate`, `PIC` — read-only for this task | — |

**Nothing new is validated only at the edge.** The channel's checks are latency
optimizations that produce a fast typed failure; atlas-cashshop repeats every one of them
inside its transaction, because the edge does not own the state and a Kafka command can
arrive stale or replayed.

---

## 2. The secondary-credential gate

Six arms carry the `ask_SPW` slot: GIFT, BUY_COUPLE, BUY_FRIENDSHIP, REBATE_LOCKER_ITEM
(and, on legacy v48, BUY_NORMAL). Today none of them look at it.

**Design:** one helper in `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`:

```go
// verifySecondaryCredential gates the arms that carry the client's ask_SPW slot.
func verifySecondaryCredential(l, ctx) func(s session.Model, spw string, birthday uint32) error
```

Behavior, in order:

1. Resolve the session's account via `account.NewProcessor(l, ctx).GetById(s.AccountId())`.
2. If the tenant is GMS major ≥ 95 (`MajorAtLeast`, never a raw `> N`), compare `spw`
   against `a.PIC()`. Plaintext comparison, matching the established precedent in
   `services/atlas-login/atlas.com/login/socket/handler/character_selected_pic.go:49`
   (`if a.PIC() != p.Pic()`) — this task does not change how PICs are stored.
3. Otherwise compare `birthday` against `a.BirthDate()`.
4. **If the account has no credential set** (`PIC == ""`, or `BirthDate == 0`), the gate
   **passes**. A server that has never collected the value cannot meaningfully check it,
   and failing closed would make every gifted item unpurchasable on a fresh tenant.
   This is logged at debug, once per call, so an operator can see the gate is inert.
5. A mismatch returns a sentinel error; each arm maps it to its own `*_FAILED` body with
   the errors-table key `INVALID_BIRTHDAY` (bound in every GMS template's `errors` table)
   and charges nothing.

Failed attempts are recorded through the existing
`account.Processor.RecordPicAttempt(id, success, ip, hwid)` — already on the channel's
account processor interface (`account/processor.go:23`) and already wired to the ban
escalation in atlas-account. This is a free win: the cash shop becomes the second PIC
surface without adding a rate limiter.

**Alternative rejected:** validating SPW inside atlas-cashshop. It would need an
account-service read inside the purchase transaction for a check that has nothing to do
with the transaction's invariants, and the failure is a client-input failure, not a
state failure — the edge is the right place to reject it.

---

## 3. New Kafka surface — `atlas-cashshop`

Extends `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
(and its mirror in `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/`).
Every body carries `TransactionId uuid.UUID` as both correlation id and **idempotency
key**, exactly as `OpenSurpriseCommandBody` does.

```go
CommandTypeRequestGiftPurchase       = "REQUEST_GIFT_PURCHASE"
CommandTypeRequestPackagePurchase    = "REQUEST_PACKAGE_PURCHASE"
CommandTypeRequestRingPurchase       = "REQUEST_RING_PURCHASE"
CommandTypeRequestLockerRebate       = "REQUEST_LOCKER_REBATE"
CommandTypeRequestEquipSlotIncrease  = "REQUEST_EQUIP_SLOT_INCREASE"
```

| Body | Fields |
|---|---|
| `RequestGiftPurchaseCommandBody` | `TransactionId`, `SerialNumber uint32`, `RecipientCharacterId uint32`, `Message string` |
| `RequestPackagePurchaseCommandBody` | `TransactionId`, `Currency uint32`, `SerialNumber uint32`, `RecipientCharacterId uint32` (zero = buy for self, non-zero = mode 31 gift) |
| `RequestRingPurchaseCommandBody` | `TransactionId`, `Currency uint32`, `SerialNumber uint32`, `PartnerCharacterId uint32`, `Message string`, `RingType string` |
| `RequestLockerRebateCommandBody` | `TransactionId`, `CashId int64` |
| `RequestEquipSlotIncreaseCommandBody` | `TransactionId`, `Currency uint32`, `SerialNumber uint32` |

**One command covers both package modes**, discriminated by `RecipientCharacterId`.
Modes 30 and 31 differ only in *who receives the members* — the resolution, capacity,
atomicity, and pricing logic is identical, and two commands would duplicate all of it.
The channel picks the result body (`BuyPackageDone` vs `GiftPackageDone`) from the event's
own `RecipientCharacterId`, so the wire distinction survives to the announce.

**One command covers couple and friendship**, discriminated by `RingType`. Same reasoning.

New status events:

```go
StatusEventTypeGiftPurchased      = "GIFT_PURCHASED"
StatusEventTypePackagePurchased   = "PACKAGE_PURCHASED"
StatusEventTypeRingPurchased      = "RING_PURCHASED"
StatusEventTypeLockerRebated      = "LOCKER_REBATED"
StatusEventTypeEquipSlotIncreased = "EQUIP_SLOT_INCREASED"
```

| Body | Fields | Feeds |
|---|---|---|
| `GiftPurchasedBody` | `TransactionId`, `RecipientName string`, `TemplateId uint32`, `Quantity uint16`, `Price uint32`, `RecipientCharacterId uint32` | `CashShopGiftDoneBody(recipientName, itemId, quantity, nxCashSpent)` |
| `PackagePurchasedBody` | `TransactionId`, `CompartmentId uuid.UUID`, `AssetIds []uint32`, `PackageTemplateId uint32`, `Price uint32`, `RecipientCharacterId uint32`, `RecipientName string` | `CashShopBuyPackageDoneBody(items, trailingCount)` or `CashShopGiftPackageDoneBody(...)` |
| `RingPurchasedBody` | `TransactionId`, `CompartmentId`, `AssetId uint32`, `PartnerName string`, `TemplateId uint32`, `Quantity uint16`, `RingType string`, `PairId uuid.UUID` | `CashShopCoupleDoneBody` / `CashShopFriendshipDoneBody` |
| `LockerRebatedBody` | `TransactionId`, `CashId int64`, `Amount int32` | `CashShopRebateDoneBody(sn, amount)` |
| `EquipSlotIncreasedBody` | `TransactionId`, `SlotIndex uint16`, `Days uint16` | `CashShopEnableEquipSlotExtSuccessBody(slotIndex, days)` |

Failures **reuse the existing `ErrorEventBody`** (`Error`, `CashItemId`, `TransactionId`).
It already echoes `TransactionId`, and the channel already consumes it. The channel cannot
tell from `ErrorEventBody` alone *which* failure arm to answer on — see §7.

Recipient-directed refresh (gift, mode-31 package, ring): rather than a second event type,
the recipient's session is refreshed by the channel's existing locker path. When
`RecipientCharacterId` is non-zero and differs from `CharacterId`, the channel's handler
also looks up the recipient's session via `session.NewProcessor(...).IfPresentByCharacterId`
and announces `CashShopRefreshLockerBody(item)` — mode 162, already in the family and
already bound in `gms_95`. An offline recipient simply has no session; the asset is durable
in their locker either way (FR-GIFT-6).

---

## 4. Ring pairing domain

### 4.1 Placement — decided: **inside `atlas-cashshop`**

Three candidates were weighed:

| Option | Pro | Con |
|---|---|---|
| **New `atlas-ring` service** | clean long-term home for marriage/effects | full `docs/adding-a-new-service.md` onboarding; a *third* service in a transaction that FR-RING-4 requires to be atomic; enormous cost for one table |
| Domain inside `atlas-character` | ring *effects* (buddy display, equip effects) are character-shaped | breaks FR-RING-4: the two ring assets live in atlas-cashshop's DB, so the pair record would be in a different service and a partial pair becomes persistable |
| **Domain inside `atlas-cashshop`** ✅ | the pair and both its assets commit in **one transaction**, satisfying FR-RING-4 exactly; the domain follows the existing `wishlist`/`coupon` layout verbatim | ring effects (out of scope here) will read it cross-service over REST |

The deciding argument is FR-RING-4. Cross-service atomicity would require a saga, and a
saga for a two-row insert is not a trade this task should make. The "effects belong to
character" objection is answered by the query resource: a future effects consumer reads
`GET /rings?filter[characterId]=` — a normal cross-service read, not internals access.

Recorded for the future: if a marriage/wedding task later needs pair lifecycle beyond
cash-locker concerns, the domain moves as a unit. The `state` field (FR-RING-9) is what
makes that move non-destructive.

### 4.2 Model

`services/atlas-cashshop/atlas.com/cashshop/ring/` — `model.go`, `entity.go`,
`administrator.go`, `provider.go`, `processor.go`, `rest.go`, `resource.go`, matching
`wishlist/` file-for-file.

```
Entity "cash_rings"
  Id                  uuid.UUID  PK
  TenantId            uuid.UUID  not null, indexed
  PairId              uuid.UUID  not null, indexed   -- shared by both halves
  CharacterId         uint32     not null, indexed   -- owner of THIS half
  PartnerCharacterId  uint32     not null
  AssetId             uint32     not null            -- the cash_assets row for this half
  ItemTemplateId      uint32     not null
  RingType            string     not null            -- "COUPLE" | "FRIENDSHIP"
  State               string     not null            -- "ACTIVE" | "BROKEN" | "EXPIRED"
  CreatedAt           time.Time  not null
```

Two rows per pair, inserted in the same `database.ExecuteTransaction` closure as the two
assets and the wallet debit. `AssetId` is the join back to the locker row that carries the
ring, so a later rebate/expire path can find and mark the pair.

REST: `GET /rings?filter[characterId]=N` (paginated, tenant-scoped) and `GET /rings/{id}`.
Both live on atlas-cashshop's existing router (`rest/handler.go`), so the PRD's §5.4
"owning service is an open question" is closed.

**`RingType` is a `libs/atlas-constants` candidate.** Before defining it, check
`libs/atlas-constants/` for an existing ring or item-classification equivalent; only add
one if none exists, and put it there rather than in the service.

### 4.3 Item template selection

The couple/friendship arms carry a `serialNumber` (one commodity), but a pair needs
**two** ring items. Two sub-cases, and the design must not guess between them:

- If the commodity's `ItemId` is itself the paired ring template (both halves are the same
  template id), both assets are created from the same commodity — the common case for
  friendship rings.
- If the couple ring has distinct male/female templates, the second half's template must be
  derived. **This is an unresolved derivation** — see §9, OQ-R1. The implementation must
  not invent a `+1` or gender-based offset.

Until OQ-R1 is resolved, the plan implements the same-template path and rejects the
divergent case with a typed `COUPLE_FAILED` / `FRIENDSHIP_FAILED` rather than guessing.

---

## 5. Cash package catalogue — `atlas-data`

### 5.1 Resolution chain

The key insight, which the PRD's data model half-states: **the package id is an item id,
not a serial number.** `CashPackage.img` is keyed by the package *item* (e.g. `9100000`),
while the client sends a *commodity serial number*. The chain is therefore:

```
sp.SerialNumber()  →  commodity(SN)  →  commodity.ItemId  ==  packageId
                   →  cashPackage(packageId).SerialNumbers[]  (member commodity SNs)
                   →  for each: commodity(memberSN) → ItemId, Count, Period
                   →  one cash_assets row per member
```

The **price charged once is the package commodity's own `Price`** (FR-PKG-5) — it is
already on the commodity resolved in step 1, so no extra lookup is needed and the "sum of
members" mistake is structurally impossible.

### 5.2 Domain

`services/atlas-data/atlas.com/data/cashpackage/` mirroring `commodity/` exactly:

- `rest.go` — `RestModel{ Id uint32 (json:"-"), SerialNumbers []uint32 }`, `GetName() == "cashPackages"`
- `registry.go` — `document.NewRegistry[string, RestModel]()`
- `reader.go` — for each child `imgdir` of the root, `Id` = the child's name parsed as
  uint32, `SerialNumbers` = the integer values under its `SN` child imgdir
- `processor.go` — `NewStorage(l, db)` with document key `"CASH_PACKAGE"`, and
  `RegisterCashPackage(path string)` following `RegisterCommodity` verbatim (per-row
  commit, no outer transaction — task-076 F2)
- `resource.go` — `GET /data/cashPackages` (paginated) and `GET /data/cashPackages/{packageId}`

Ingest wiring: one line in `services/atlas-data/atlas.com/data/data/processor.go`
alongside `:176`, `RegisterFileData(path, filepath.Join("Etc.wz", "CashPackage.img.xml"), ...)`,
plus a worker under `data/workers/` mirroring `workers/commodity.go`.

**Ingest must be tolerant of the file's absence.** `Etc.wz/CashPackage.img.xml` is not in
this repository (WZ dumps are external), and not every tenant's dump will carry it. Follow
whatever `RegisterFileData` already does for a missing file; if it hard-fails, the
cashpackage registration must be the tolerant variant, because a tenant without the file
must still boot atlas-data. This is a **verification item**, not an assumption — the plan
checks `RegisterFileData`'s missing-file behavior before wiring.

Consumer side: a thin `data/cashpackage` REST client in `atlas-cashshop`, mirroring
`cashshop/commodity/requests.go`.

---

## 6. Purchase records

**Confirmed: `atlas-cashshop` persists no purchase history today.** `cash_assets` carries
`CommodityId`, `PurchasedBy`, and `CreatedAt`, but it is soft-deleted on withdrawal to a
character inventory and on rebate — so "did this account ever buy SN X" cannot be answered
from live locker contents (FR-REC-2 says so explicitly).

New table in atlas-cashshop, `services/atlas-cashshop/atlas.com/cashshop/purchaserecord/`:

```
Entity "cash_purchase_records"
  Id            uuid.UUID  PK
  TenantId      uuid.UUID  not null
  AccountId     uint32     not null
  SerialNumber  uint32     not null
  Count         uint32     not null   -- times purchased
  FirstAt       time.Time  not null
  LastAt        time.Time  not null
  unique index (TenantId, AccountId, SerialNumber)
```

**Written inside the same transaction as every successful purchase** — the existing
`Purchase(mb)` closure, plus each new gift/package/ring/equip-slot closure. An upsert on
the unique index (`Count = Count + 1`, `LastAt = now`). For a package, one record per
member SN **and** one for the package SN itself, because the client can ask about either.

A record is **not** removed on rebate. "Purchased" is a historical fact; FR-REC-2's own
wording ("a consumed or discarded item still counts as purchased") settles it.

Read path (Skeleton B): `GET /accounts/{accountId}/purchaseRecords/{serialNumber}` returns
`{ serialNumber, purchased bool, count }`; a miss is a `200` with `purchased: false`, not a
404 — the client needs an answer, not an error. The channel announces
`CashShopPurchaseRecordDoneBody(goodsSN int32, purchased byte)`.

**Backfill:** existing accounts have no records, so `GET_PURCHASE_RECORD` will answer
"not purchased" for pre-task purchases. A one-shot backfill from live `cash_assets` rows
(`CommodityId` + owning account, including soft-deleted rows) is **in scope** and cheap —
it makes the answer correct for everything still recorded. Rows already hard-gone are
unrecoverable and that is stated, not hidden.

---

## 7. Failure routing — the one genuinely new mechanism

Every mutating arm must answer on **its own** `*_FAILED` mode byte (FR-X-2). The existing
`ErrorEventBody` carries a reason string and a `TransactionId` but not *which arm* failed,
and the channel's existing `handleStatusEventError` unconditionally announces
`CashShopInventoryCapacityIncreaseFailedBody` — the wrong mode for eight of the ten arms.

Two options:

| Option | Assessment |
|---|---|
| A new `*Failed` event type per arm (five more types) | Symmetric with `SurpriseFailedEventBody` / `CouponFailedBody`, both of which exist precisely because ERROR announces the wrong mode. But five near-identical event types is a lot of duplication. |
| **Add an `Operation string` discriminator to `ErrorEventBody`** ✅ | One field; the channel switches on it to pick the failure body. Backward compatible — an empty `Operation` keeps today's behavior byte-for-byte, so `REQUEST_PURCHASE`, `EXPIRE`, and the inventory-increase paths are untouched. |

**Decision: extend `ErrorEventBody` with `Operation string`.** New producers set it
(`"GIFT"`, `"BUY_PACKAGE"`, `"GIFT_PACKAGE"`, `"COUPLE"`, `"FRIENDSHIP"`, `"REBATE"`,
`"BUY_NORMAL"`, `"ENABLE_EQUIP_SLOT"`); existing producers leave it empty and keep the
current arm. The channel's `handleStatusEventError` grows a switch with today's behavior
as its `default`.

`ErrorEventBody.Error` continues to carry an **errors-table key**, never display text —
`INVENTORY_FULL`, `NOT_ENOUGH_CASH`, `INCORRECT_NAME`, `CANNOT_GIFT_TO_OWN_ACCOUNT`,
`CANNOT_GIFT_RECIPIENT_INVENTORY_FULL`, `INVALID_BIRTHDAY`, `NOT_AVAILABLE_FOR_PURCHASE`,
`UNKNOWN_ERROR` — all already bound in every GMS template's `errors` table. Before a
failure is announced, the channel checks the tenant binds the key via the existing
`atlaspacket.CodeConfigured(opts, "errors", reason)` predicate (the
`transferFailureReasonConfigured` pattern at `cash_shop_operation.go`), logs a warning
when it does not, and sends anyway so the dialog is never left wedged.

---

## 8. Idempotency

Every new command is replay-safe (NFR, task-208 precedent). Mechanism per command:

| Command | Idempotency key | Enforcement |
|---|---|---|
| `REQUEST_GIFT_PURCHASE`, `REQUEST_PACKAGE_PURCHASE`, `REQUEST_RING_PURCHASE`, `REQUEST_EQUIP_SLOT_INCREASE` | `TransactionId` | A `cash_command_ledger` row (`TenantId`, `TransactionId` unique) inserted **first inside the transaction**. A duplicate insert violates the unique index → the transaction rolls back and the handler returns without re-charging. Same shape as `surprise/opening/duplicate.go`. |
| `REQUEST_LOCKER_REBATE` | `TransactionId` **and** the asset's own existence | Belt and braces: the ledger stops the replay, and the asset lookup fails on the second pass anyway because the first deleted it. FR-REB-5 is satisfied twice over. |

The ledger is a single shared table, not one per command — the key is globally unique and
the row's only job is to be a uniqueness claim. Row retention: leave the rows; they are
small and the table is the audit trail.

**`transactionId` is minted per click at the channel**, never derived from character id or
timestamp. A genuine second click gets a new id and legitimately charges twice; only a
Kafka redelivery replays the same id. This is exactly `OpenSurpriseCommandBody`'s contract.

---

## 9. Equip-slot extension (ENABLE_EQUIP_SLOT) — the honest gap

The purchase half is ordinary Skeleton A. The **effect** half is not, and this design will
not pretend otherwise.

What is verified:

- The wire body on v83+ is `pointType bool` + `serialNumber uint32` — no currency int
  (`shop_operation_enable_equip_slot.go:63-72`). So the purchase is requested with
  `resolvePurchaseCurrency(pointType, 0)`, which steers a points buy to wallet currency 2
  and a credit buy to 0, matching `cashshop/processor.go:120-127`.
- The success body is `CashShopEnableEquipSlotExtSuccessBody(slotIndex uint16, days uint16)`
  — the client wants a slot index and a duration in days.
- `libs/atlas-constants/inventory/slot/constants.go` has **no extra-pendant slot**. The
  highest-numbered entries stop at `pet3ItemIgnore` (-48); there is no -59 or equivalent.
- `atlas-character` has **no** pendant, slot-extension, or equipped-capacity concept
  anywhere (verified by sweep: zero matches for pendant/slotExt in the service).

What is **not** verified and must be derived from the GMS v95.1 IDB before implementation:

- **OQ-E1** — the `slotIndex` the client expects in `CashShopEnableEquipSlotExtSuccessBody`,
  and the equipped-inventory slot position the extended pendant occupies.
- **OQ-E2** — how the client learns the extension is active across a relog/channel change:
  a field on `GW_CharacterStat`, a re-sent `CashShopEnableEquipSlotExtSuccess`, or an
  avatar-look consequence. FR-SLOT-3 ("survives a channel change") cannot be satisfied
  without this.

**Design position:** the atlas-character side is a new
`services/atlas-character/atlas.com/character/equipslot/` domain —
`(TenantId, CharacterId, SlotIndex, ExpiresAt)` with an upsert that **extends** rather than
duplicates (FR-SLOT-4: `ExpiresAt = max(now, ExpiresAt) + period`). That much is safe and
version-independent. The slot constant and the client-facing propagation are blocked on
OQ-E1/OQ-E2 and are the **first derivation the plan phase must schedule**, because they
also decide whether `libs/atlas-constants/inventory/slot/constants.go` gains an entry —
a shared-library change that ripples.

If OQ-E1/OQ-E2 cannot be resolved from the IDB, the correct outcome is to land the
purchase, the persistence, and a typed failure for the un-derivable path — **not** an
invented slot number. That is a partial arm with a stated reason, not a stub.

---

## 10. `gms_95` CashShopOpen

Verified: `CashShopOpen` appears exactly once in every GMS template and in
`template_jms_185_1.json`, and **zero times** in `template_gms_95_1.json`. The v92
registration is `{"opCode":"0x8E","writer":"CashShopOpen","fname":"CStage::OnSetCashShop","services":["channel"]}`.

Plan: derive the v95 opcode for `CStage::OnSetCashShop` from the GMS v95.1 IDB
(`func_query` per `docs/reverse-engineering.md`), then add the registration to
`template_gms_95_1.json` with the derived opcode and the same fname. **The opcode is not
copied from v92** (FR-V95-1) — v95's writer table already diverges heavily from v92's
(the `CashShopOperation` writer moved from 0x145 on v83 to 0x180 on v95).

If the derivation shows v95 does not send the packet at all, record the `n-a` proof in the
coverage matrix (FR-V95-3). Validation is a live v95 tenant opening the cash shop, not a
template diff (FR-V95-4).

This is **the first slice to land** — every other arm in this task is unreachable on a v95
tenant until it does, so it is also the enabler for all live validation.

---

## 11. Testing

| Layer | What | How |
|---|---|---|
| Packet | any new serverbound codec (`ShopOperationBuyOtherPackage`, possibly `ShopOperationApplyWishlist`) | byte-fixture tests with `packet-audit:verify` markers, per `docs/packets/audits/VERIFYING_A_PACKET.md`; affected matrix cells promote |
| Channel handler | each arm's gate, rejection, and command emission | table-driven tests over `CashShopOperationHandleFunc` with a fake writer producer, following the existing `handleBuyNameChange` test precedent; the gate is tested with PIC set / PIC empty / mismatch |
| atlas-cashshop | atomicity and idempotency | transaction tests: package purchase with an unresolvable member creates **nothing**; ring purchase with a failing second insert leaves **no** pair; a replayed `TransactionId` charges once. Builder pattern per repo convention — no `*_testhelpers.go`. |
| atlas-data | reader | `reader_test.go` against a fixture `CashPackage.img.xml` node tree, mirroring `commodity/reader_test.go` |
| Cross-service | the seam | for each new event, trace it by hand into the channel consumer and assert a test covers the **new** contract (CLAUDE.md "Done means verified") |

Failure-path coverage is not optional: FR-X-2 means every arm needs at least one test that
asserts a typed failure body reaches the writer.

---

## 12. Sequencing

The task is large; the slices are ordered so each lands independently verifiable and
nothing later is blocked on an unresolved derivation.

| # | Slice | Depends on | Notes |
|---|---|---|---|
| 1 | `gms_95` CashShopOpen registration | IDB derivation only | unblocks all live validation |
| 2 | Failure routing (`ErrorEventBody.Operation` + channel switch) | — | prerequisite for every arm's rejection path |
| 3 | Secondary-credential gate | — | prerequisite for GIFT / RING / REBATE |
| 4 | Purchase records (table + write-in-tx + REST + backfill) | — | |
| 5 | **GET_PURCHASE_RECORD** (Skeleton B) | 4 | first arm to land end-to-end |
| 6 | **APPLY_WISHLIST** (Skeleton B) | derivation of payload (§13, OQ-W1) | trivial once payload is known |
| 7 | **BUY_NORMAL** (Skeleton A, reuses `RequestPurchase`) | 2 | smallest mutating arm; validates the skeleton |
| 8 | **REBATE_LOCKER_ITEM** | 2, 3, 8-ledger | introduces the idempotency ledger |
| 9 | **GIFT** | 2, 3, 7 | introduces `GiftFrom` on `cash_assets` and recipient-directed refresh |
| 10 | `atlas-data` cashpackage domain + atlas-cashshop client | — | parallelizable with 5–9 |
| 11 | **BUY_PACKAGE** (mode 30) | 10 | |
| 12 | **BUY_OTHER_PACKAGE** (mode 31) | 11 + OQ-P1 derivation | reuses slice 11's command |
| 13 | Ring domain + **BUY_COUPLE** / **BUY_FRIENDSHIP** | 2, 3, + OQ-R1 | |
| 14 | **ENABLE_EQUIP_SLOT** | OQ-E1/OQ-E2 | last, because it is the most derivation-blocked |

Slice 9 requires an **additive migration on `cash_assets`**: `GiftFrom string` (the sender's
character name, ≤13 chars per `CashInventoryItem`'s padded encode) and `GiftMessage string`
(≤73 chars per `GiftListEntry`). Both default empty; existing rows are unaffected.

---

## 13. Open questions still requiring derivation

Each names the artifact that resolves it and the slice it blocks. **None may be closed by
an invented value** (PRD acceptance criterion).

| Id | Question | Resolves via | Blocks |
|---|---|---|---|
| OQ-P1 | Does `BUY_OTHER_PACKAGE` (mode 31/33) carry the same body as mode 30, or its own (recipient name + message)? The `GIFT_PACKAGE_*` result pair strongly suggests a gift shape, but the serverbound layout is underived. | GMS v95.1 IDB, `CCashShop::OnBuyPackage` and the mode-31 sender | slice 12 |
| OQ-W1 | Does `APPLY_WISHLIST` (mode 33/35) carry a serverbound payload at all? There is no `shop_operation_apply_wishlist.go` and the current arm reads nothing. | GMS v95.1 IDB, the `CCashShop` sender for that mode | slice 6 |
| OQ-E1 | The `slotIndex` value the client expects, and the equipped slot position the extra pendant occupies. | GMS v95.1 IDB, `CCashShop::OnEnableEquipSlotExt` result handler | slice 14 |
| OQ-E2 | How the extension is re-communicated across relog/channel change (`GW_CharacterStat` field? re-sent packet?). | GMS v95.1 IDB | slice 14 (FR-SLOT-3) |
| OQ-R1 | For a couple ring, are both halves the same item template, or are there distinct male/female templates? If distinct, how is the partner's template derived? | `Etc.wz` / `Character.wz` ring data + the client's ring-pair handling | slice 13 |
| OQ-G1 | `oneADay` (`m_bRequestBuyOneADay`) on `ShopOperationGift` — is it a per-day gift limit the server enforces, or a client-side UI flag? | GMS v95.1 IDB, `CCashShop::SendGiftsPacket` caller | slice 9 (behavior only; the field is already decoded) |
| OQ-O1 | `option` on `ShopOperationBuyPackage` / `ShopOperationBuyCouple` / `ShopOperationBuyFriendship`. | GMS v95.1 IDB | slices 11, 13 (currently ignored; must be *proven* ignorable, not assumed) |

`flag` on `ShopOperationBuyFriendship` is **not** listed: the codec already documents it as
a v48-only constant byte the client hard-codes to 1 (`shop_operation_buy_friendship.go:26`,
`:83-90`), absent on v83+. Nothing to derive.

**Gift-recipient locker capacity (PRD OQ 10) is decided, not deferred:** reject the gift
with `CANNOT_GIFT_RECIPIENT_INVENTORY_FULL` (already bound in every template's errors
table) and charge nothing. Queuing would require a durable pending-gift store with no
client surface to display it.

---

## 14. Risks

| Risk | Mitigation |
|---|---|
| The task is large enough that a single branch grows unreviewable | The §12 slices are independently verifiable; each ends at a state where `tools/verify.sh` passes and a reviewer can read one arm end to end |
| A derivation (OQ-E1/E2, OQ-P1) comes back inconclusive and an arm is left half-done | Stated up front: land the verifiable half plus a typed failure, and record the gap. Never a stub, never an invented value |
| `ErrorEventBody.Operation` changes a message shape every existing consumer reads | Additive field with empty-string default; the channel's `default:` branch is today's exact behavior; a test pins the empty-Operation path |
| `cash_assets` migration on a live table | Two nullable/defaulted string columns, additive only, no backfill needed |
| Package purchase creating N assets blows locker capacity mid-way | FR-PKG-6: capacity is checked against the full member count **before** the wallet is debited, inside the same transaction |
| SPW gate locks players out of the cash shop on tenants that never collected a PIC | The gate passes when no credential is set (§2 step 4), and logs that it did |
| `Etc.wz/CashPackage.img.xml` absent from a tenant's dump | Ingest must tolerate the missing file; verified against `RegisterFileData`'s actual behavior before wiring, not assumed |
