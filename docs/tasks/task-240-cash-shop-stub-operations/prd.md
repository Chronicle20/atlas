# Cash Shop Stub Operations — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-19
---

## 1. Overview

`CashShopOperationHandleFunc` in `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
dispatches the client's `CCashShop::OnCashItemRequest` opcode to one arm per operation
mode. Ten of those modes currently decode their serverbound packet, write an
`l.Infof` line, and return — the player sees nothing happen, no currency moves, and
no result packet comes back. One further mode (`BUY_OTHER_PACKAGE`, 31) has a
declared constant but no dispatch arm at all, so it falls through to the terminal
`Unhandled Cash Shop Operation` warning.

This task implements those arms end to end: the channel handler routes to a
processor, the processor issues a Kafka command to `atlas-cashshop`, the service
mutates the durable state (locker asset, wallet, wishlist, character slot), and
the resulting event is announced back to the client through the existing
`CashShopOperationWriter` result family established by `task-183-cashshop-result-family`.
Three of the arms require new domain surface that does not exist anywhere in the
monorepo today: a cash-package catalogue in `atlas-data` (BUY_PACKAGE /
BUY_OTHER_PACKAGE), and a ring-pairing domain (BUY_COUPLE / BUY_FRIENDSHIP).

The task also closes an adjacent configuration gap: the `gms_95` seed template does
not register the `CashShopOpen` writer, so `CashShopEntryHandleFunc` cannot announce
the cash shop open packet on a v95 tenant at all — every operation below is
unreachable on that version until it is fixed.

### Scope corrections against the originating backlog entry

The backlog item that prompted this task is stale in three respects. These are
settled facts, verified against the tree at `d9ec287b8`, and the requirements below
reflect them:

1. **`BUY_NAME_CHANGE` (46) and `BUY_WORLD_TRANSFER` (49) are already implemented.**
   `cash_shop_operation.go` dispatches both to `handleBuyNameChange` and
   `handleBuyWorldTransfer`, landed by `task-227-cash-name-change-world-transfer`.
   They are **out of scope**.
2. **The `atlas-cashshop` command list is longer than the backlog states.**
   `kafka/message/cashshop/kafka.go` declares nine command types, not seven —
   `OPEN_SURPRISE` (task-207) and `REQUEST_COUPON_REDEMPTION` (task-206) also exist.
   New commands added here extend that set.
3. **`BUY_OTHER_PACKAGE` (31) is not "log-only" — it is entirely unrouted.**
   `CashShopOperationBuyOtherPackage` is declared at `cash_shop_operation.go:40`
   and referenced nowhere else in the repository.

## 2. Goals

Primary goals:

- Replace every log-only arm in `cash_shop_operation.go` with a real implementation
  that mutates durable state and announces a result packet to the client.
- Route `BUY_OTHER_PACKAGE` (31), which today has no dispatch arm.
- Introduce a cash-package catalogue domain in `atlas-data`, sourced from
  `Etc.wz/CashPackage.img`, so a package serial number resolves to its member
  serial numbers.
- Introduce a ring-pairing domain so couple and friendship ring purchases create a
  paired record and deliver the partner's ring to the recipient's locker.
- Register the `CashShopOpen` writer for `gms_95`, or record a verified `n-a` proof
  if v95 genuinely does not carry that packet.
- Every new arm returns a typed failure result to the client on the rejection paths
  (insufficient funds, unknown serial number, recipient not found, locker full)
  rather than failing silently.

Non-goals:

- Re-implementing or modifying `BUY_NAME_CHANGE` / `BUY_WORLD_TRANSFER` (task-227).
- The MTS marketplace (`task-102-mts-marketplace`).
- A cash shop UI surface in `atlas-ui` for administering packages or rings. The
  catalogue is ingested from WZ data; administration is out of scope.
- Marriage/wedding content — the ring domain here covers item pairing and delivery
  only, not wedding maps, ceremonies, or the marriage quest line.
- The "quest-item tab" mentioned in the backlog entry. No equivalent was located in
  either codebase; it is not specified here.
- Backfilling coverage-matrix cells for packet versions unrelated to the arms
  implemented here.

## 3. User Stories

- As a player, I want to gift a cash item to another character by name so that it
  arrives in that character's cash locker with my message attached.
- As a player, I want to buy a cash package so that every item in the package lands
  in my locker in a single transaction, and none of them land if the purchase fails.
- As a player, I want to buy a couple ring or friendship ring for another character
  so that we each receive our half of the pair.
- As a player, I want to return a locker item I bought by mistake so that the
  currency is refunded to the wallet it was drawn from.
- As a player, I want to apply my saved wishlist so that the shop reflects the items
  I previously flagged.
- As a player, I want to see the purchase record for an item so that I know whether
  I already own it before spending again.
- As a player, I want to buy the extra pendant slot so that my character can equip a
  second pendant.
- As a v95-tenant player, I want the cash shop to open at all, so that every
  operation above is reachable.

## 4. Functional Requirements

### 4.1 GIFT (mode 4)

`ShopOperationGift` decodes `birthday`, `spw`, `serialNumber`, `oneADay`, `name`,
`message` (`libs/atlas-packet/cash/serverbound/shop_operation_gift.go:23`).

- FR-GIFT-1: The handler MUST resolve `name` to a character in the same world. An
  unknown name returns a failure result naming the recipient-not-found reason; no
  currency is charged.
- FR-GIFT-2: The handler MUST validate `birthday` against the purchasing account's
  stored date of birth before charging. A mismatch returns a failure result and
  charges nothing. (Cosmic's `CashOperationHandler` action 4 performs the same check.)
- FR-GIFT-3: A gift MUST NOT be sent to a character on the sending account. Attempting
  it returns a failure result.
- FR-GIFT-4: On success the commodity's item is created as a locker asset owned by the
  **recipient's** account, carrying the sender's character name and `message`.
- FR-GIFT-5: The charge and the recipient-side asset creation MUST be atomic within
  `atlas-cashshop`: if asset creation fails, the wallet is not debited.
- FR-GIFT-6: The sender receives a gift-succeeded result packet. Delivery to an
  offline recipient is durable — the asset exists in the locker regardless of whether
  the recipient is connected.
- FR-GIFT-7: `oneADay` and `spw` handling MUST be derived from the v95 client before
  being specified as behavior; see Open Questions.

### 4.2 ENABLE_EQUIP_SLOT (mode 9)

`ShopOperationEnableEquipSlot` decodes `pointType`, `currency`, `flag`,
`serialNumber`.

- FR-SLOT-1: The purchase MUST charge through the same `REQUEST_PURCHASE` pipeline
  every other `BUY_*` arm uses, using the decoded `pointType`/`currency`.
- FR-SLOT-2: On success the character's extra-pendant-slot expansion MUST be persisted
  in `atlas-character`, with an expiration where the commodity carries one.
- FR-SLOT-3: The equip-slot expansion MUST be reflected in the character's equipped
  inventory capacity such that a second pendant can be equipped and survives a
  channel change.
- FR-SLOT-4: Buying the expansion while it is already active extends rather than
  duplicates it.

### 4.3 BUY_NORMAL (mode 20)

`ShopOperationBuyNormal` decodes `serialNumber`, `spw`, `name`, `message`.

- FR-BUYN-1: The arm MUST route to the existing `RequestPurchase` pipeline
  (`services/atlas-channel/atlas.com/channel/cashshop/processor.go:98`).
- FR-BUYN-2: The struct carries no `isPoints`/`currency` fields. The currency to charge
  MUST be derived rather than invented — the design phase MUST establish the derivation
  from the v95 client, following the precedent set by task-227's
  `buy-currency-derivation.md` for the same problem on `BUY_NAME_CHANGE`.
- FR-BUYN-3: On success the item lands in the purchaser's locker and a purchase result
  packet is announced.

### 4.4 REBATE_LOCKER_ITEM (mode 26)

`ShopOperationRebateLockerItem` decodes `birthday`, `spw`, `unk` (uint64).

- FR-REB-1: `unk` MUST be identified during design — the uint64 width and Cosmic's
  refund-locker-item behavior both suggest a cash asset id. It MUST NOT be assumed.
- FR-REB-2: The arm MUST validate the referenced asset exists, belongs to the
  requesting account, and is refund-eligible before mutating anything.
- FR-REB-3: On success the asset is removed from the locker and the original purchase
  price is credited back to the wallet the purchase was drawn from.
- FR-REB-4: An asset that has been moved into a character inventory, has expired, or
  was not purchased with currency (a gift, a coupon reward, a surprise-box drop) MUST
  be rejected with a typed failure result.
- FR-REB-5: The refund MUST be idempotent — a replayed command refunds once.
- FR-REB-6: `birthday` MUST be validated as in FR-GIFT-2.

### 4.5 BUY_COUPLE (29) and BUY_FRIENDSHIP (35)

`ShopOperationBuyCouple` decodes `isPoints`, `currency`, `birthday`, `spw`, `option`,
`serialNumber`, `name`, `message`. `ShopOperationBuyFriendship` decodes the same plus
`flag`.

No ring, pairing, or marriage domain exists in the monorepo today.

- FR-RING-1: A new ring-pairing domain MUST be introduced that records a pair: the two
  character ids, the ring item template, a shared pair id, and the ring type
  (couple or friendship).
- FR-RING-2: The recipient `name` MUST resolve to an online-or-offline character in the
  same world. Unknown recipient returns a typed failure and charges nothing.
- FR-RING-3: On success, two ring assets are created — one in each account's locker —
  bound to the same pair id, and the recipient receives the sender's `message`.
- FR-RING-4: The charge and both asset creations MUST be atomic. A partial pair MUST
  NOT be persistable.
- FR-RING-5: `birthday` MUST be validated as in FR-GIFT-2.
- FR-RING-6: The two ring types MUST be distinguishable in the data model; friendship
  rings and couple rings have distinct item ranges and distinct client-side effects.
- FR-RING-7: The paired state MUST be queryable by character id so that a later
  consumer (ring effects, buddy display) can resolve a character's partner.
- FR-RING-8: `option` and `flag` semantics MUST be derived from the v95 client before
  being specified; see Open Questions.
- FR-RING-9: Breaking a pair (either ring destroyed, expired, or unequipped) is out of
  scope for the effect surface but the data model MUST NOT preclude it — pair records
  carry a state field rather than being delete-only.

### 4.6 BUY_PACKAGE (30) and BUY_OTHER_PACKAGE (31)

`ShopOperationBuyPackage` decodes `pointType`, `option`, `serialNumber`.
`BUY_OTHER_PACKAGE` has **no dispatch arm today**.

- FR-PKG-1: A cash-package catalogue domain MUST be added to `atlas-data`, ingesting
  `Etc.wz/CashPackage.img`. The source shape is confirmed present in the WZ dumps:
  each package is an `imgdir` keyed by package id (e.g. `9100000`) containing an `SN`
  `imgdir` of member serial numbers.
- FR-PKG-2: The domain MUST follow the established `atlas-data` pattern — reader,
  registry entry, processor, REST resource — consistent with the sibling `commodity`
  and `cash` domains.
- FR-PKG-3: A package purchase MUST resolve the package serial number to its member
  serial numbers, then create one locker asset per member.
- FR-PKG-4: The purchase MUST be atomic across all members: if any member cannot be
  created (unknown commodity, locker capacity), nothing is created and nothing is
  charged.
- FR-PKG-5: The charge is the package's own price, charged once — not the sum of member
  prices.
- FR-PKG-6: Locker capacity MUST be checked against the full member count *before*
  charging.
- FR-PKG-7: `BUY_OTHER_PACKAGE` (31) MUST be routed. Whether it reuses
  `ShopOperationBuyPackage` or requires its own serverbound struct MUST be determined
  from the v95 client during design — there is no `shop_operation_buy_other_package.go`
  in `libs/atlas-packet/cash/serverbound/` today.
- FR-PKG-8: The behavioral difference between mode 30 and mode 31 MUST be derived from
  the client, not assumed to be identical.

### 4.7 APPLY_WISHLIST (mode 33)

The current arm decodes nothing at all — it logs and returns. There is no
`shop_operation_apply_wishlist.go` in `libs/atlas-packet/cash/serverbound/`.

- FR-WISH-1: The serverbound payload (if any) MUST be derived from the v95 client and,
  if non-empty, a codec added following `IMPLEMENTING_A_PACKET.md`.
- FR-WISH-2: The arm MUST read the character's stored wishlist via the existing
  `wishlist` domain in `atlas-cashshop` and announce it to the client using the
  existing `CashShopWishListUpdateBody` writer already used by `SET_WISHLIST`.
- FR-WISH-3: A character with an empty wishlist MUST receive a well-formed empty
  response, not an error and not silence.

### 4.8 GET_PURCHASE_RECORD (mode 40)

`ShopOperationGetPurchaseRecord` decodes `serialNumber`.

- FR-REC-1: The response semantics MUST be derived from the v95 client's
  `CCashShop::OnCashItemResult` arm before behavior is specified.
- FR-REC-2: The arm MUST answer whether the requesting account has previously
  purchased the given serial number, backed by durable purchase history rather than
  a computed guess from current locker contents — a consumed or discarded item still
  counts as purchased.
- FR-REC-3: If no durable purchase history exists in `atlas-cashshop` today, the design
  MUST specify the record to persist and the point in the purchase pipeline that
  writes it. This requirement MUST NOT be satisfied by a stubbed or hard-coded response.

### 4.9 gms_95 CashShopOpen registration

Verified at `d9ec287b8`: `CashShopOpen` occurrence counts across the GMS seed
templates in `services/atlas-configurations/seed-data/templates/` are
gms_12/48/61/72/79/83/84/87/92 = 1 each, **gms_95 = 0**.

- FR-V95-1: The v95 opcode for the cash shop open packet MUST be derived from the
  GMS v95.1 IDB, not copied from v92.
- FR-V95-2: If the packet exists on v95, `template_gms_95_1.json` MUST register the
  `CashShopOpen` writer so `CashShopEntryHandleFunc`
  (`cash_shop_entry.go:67`) can announce it.
- FR-V95-3: If v95 genuinely does not carry the packet, an `n-a` proof MUST be recorded
  in the coverage matrix rather than the gap being left undocumented.
- FR-V95-4: The fix MUST be validated by confirming a v95 tenant can open the cash shop,
  not by template diff alone.

### 4.10 Cross-cutting

- FR-X-1: No arm may remain a log-and-return. After this task, every declared
  `CashShopOperation*` constant either dispatches to a real implementation or is
  removed.
- FR-X-2: Every failure path MUST announce a typed result to the client. A silent
  return leaves the client's cash shop dialog wedged.
- FR-X-3: All new arms MUST use the existing `isCashShopOperation` config-resolved
  mode comparison. Mode bytes MUST NOT be hard-coded.
- FR-X-4: Version-divergent packet fields MUST use the `MajorAtLeast` idiom, never a
  raw `> N` comparison.
- FR-X-5: New writers/readers MUST be registered across all applicable seed templates,
  not only gms_95.

## 5. API Surface

### 5.1 New Kafka commands — `atlas-cashshop`

Extending `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
(currently nine command types). Each follows the existing
`Command[E]{CharacterId, Type, Body}` envelope and carries a `TransactionId uuid.UUID`
correlation id in the body, matching the `RequestPurchaseCommandBody` precedent.

| Command type | Body fields |
|---|---|
| `REQUEST_GIFT_PURCHASE` | `TransactionId`, `SerialNumber`, `RecipientCharacterId`, `Message` |
| `REQUEST_PACKAGE_PURCHASE` | `TransactionId`, `Currency`, `PackageSerialNumber` |
| `REQUEST_RING_PURCHASE` | `TransactionId`, `Currency`, `SerialNumber`, `PartnerCharacterId`, `Message`, `RingType` |
| `REQUEST_LOCKER_REBATE` | `TransactionId`, `AssetId` |
| `REQUEST_EQUIP_SLOT_INCREASE` | `TransactionId`, `Currency`, `SerialNumber` |

Exact naming and field sets are settled at design time; the constraint is that each
follows the existing envelope and correlation-id convention rather than inventing a
new shape.

### 5.2 New events — `atlas-cashshop`

Each command emits a success event and reuses the existing `ErrorEventBody` (echoing
`TransactionId`) for failures, consistent with the purchase pipeline. Gift and ring
purchases emit an additional recipient-directed event so the recipient's locker view
refreshes if they are connected.

### 5.3 New REST resource — `atlas-data`

JSON:API, following the sibling `commodity` resource:

- `GET /api/data/cashPackages` — paginated list
- `GET /api/data/cashPackages/{packageId}` — one package with its member serial numbers

### 5.4 New REST resource — ring pairing

- `GET /api/.../rings?filter[characterId]=` — the character's ring pairs
- `GET /api/.../rings/{ringId}` — one pair

Owning service is an open question (§9).

## 6. Data Model

All new tables carry `tenant_id` and are scoped by it in every query.

### 6.1 `cash_packages` (atlas-data, ingested)

| Column | Type | Notes |
|---|---|---|
| `tenant_id` | uuid | scoping |
| `package_id` | uint32 | the `imgdir` key, e.g. `9100000` |
| `serial_numbers` | uint32[] | member SNs from the `SN` imgdir |

Ingested from `Etc.wz/CashPackage.img` on the existing atlas-data ingest run. Not
user-mutable.

### 6.2 `rings`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | scoping |
| `pair_id` | uuid | shared by both halves |
| `character_id` | uint32 | owner of this half |
| `partner_character_id` | uint32 | the other half's owner |
| `item_template_id` | uint32 | ring item |
| `ring_type` | string | `couple` \| `friendship` |
| `state` | string | supports later break/expire (FR-RING-9) |
| `created_at` | timestamp | |

Constraint: both halves of a `pair_id` MUST be inserted in one transaction (FR-RING-4).

### 6.3 Purchase history

Required by FR-REC-2 if no durable record exists today. The design phase MUST confirm
whether `atlas-cashshop` already persists purchase history; if not, a
`purchase_records` table keyed by (`tenant_id`, `account_id`, `serial_number`) is
added and written by the purchase pipeline.

### 6.4 Equip-slot expansion

Extends the character record in `atlas-character` with the extra-pendant-slot
expansion and its expiration. Migration is additive; the existing pattern from
task-227's pending-change work is the reference for non-destructive character schema
change.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-channel` | Replace ten log-only arms in `cash_shop_operation.go`; add the missing `BUY_OTHER_PACKAGE` arm; extend `cashshop/processor.go` with the new request methods; consume the new events and announce results |
| `atlas-cashshop` | New commands, handlers, and events; gift/rebate/package/ring transaction logic; new ring domain (or a client of it); purchase-record persistence if absent |
| `atlas-data` | New `cashpackage` domain — reader, registry entry, processor, REST resource — ingesting `Etc.wz/CashPackage.img` |
| `atlas-character` | Equip-slot expansion persistence and capacity effect |
| `atlas-configurations` | `template_gms_95_1.json` gains the `CashShopOpen` writer registration; any new packet routed across all applicable templates |
| `libs/atlas-packet` | Codecs for `APPLY_WISHLIST` and `BUY_OTHER_PACKAGE` if the client carries payloads for them |

## 8. Non-Functional Requirements

- **Multi-tenancy:** every new entity carries `tenant_id`; every query and every Kafka
  consumer resolves tenant from context. No cross-tenant read is reachable.
- **Atomicity:** gift, package, ring, and rebate flows are all-or-nothing. A partial
  package, a half-created ring pair, or a debit without delivery is a correctness bug,
  not an acceptable degraded mode.
- **Idempotency:** every new command is safe to replay — a redelivered Kafka message
  MUST NOT double-charge or double-deliver. This follows the `task-208-command-idempotency`
  precedent.
- **Evidence:** no opcode, field order, or mode byte is invented. Every wire-level fact
  is derived from the GMS v95.1 IDB or the checked-in export. Where a field's meaning
  cannot be established, it is recorded as unverified rather than guessed.
- **Observability:** each new arm logs the character id, operation, and outcome. Failure
  paths log the typed reason.
- **Testing:** byte-fixture tests for new codecs with `packet-audit:verify` markers;
  service tests use the project's Builder pattern, not `*_testhelpers.go`.
- **Constants:** check `libs/atlas-constants/` before defining any new domain type,
  alias, or numeric constant.

## 9. Open Questions

1. **Which service owns the ring domain?** `atlas-cashshop` creates the pair, but ring
   *effects* are a character/buddy concern. Candidates: a new `atlas-ring` service, a
   domain inside `atlas-character`, or a domain inside `atlas-cashshop`. Resolve in
   design.
2. **`spw` semantics** — present on GIFT, BUY_NORMAL, REBATE, COUPLE, FRIENDSHIP, with
   inconsistent width (`string` on most, `uint32` on `ShopOperationBuyNormal`). Is it a
   secondary password? If so, is it validated server-side, and against what?
3. **`ShopOperationRebateLockerItem.unk` (uint64)** — presumed cash asset id, unverified.
4. **`option` / `flag`** on the couple, friendship, package, and equip-slot structs —
   meaning unknown.
5. **`oneADay`** on `ShopOperationGift` — a per-day gift limit flag? Enforcement
   unspecified.
6. **BUY_NORMAL currency derivation** (FR-BUYN-2) — the struct carries no
   `isPoints`/`currency`. Task-227 hit the identical problem on `BUY_NAME_CHANGE` and
   resolved it to `isPoints=false, currency=0`; whether that generalizes to mode 20 is
   unverified.
7. **Mode 30 vs mode 31** — what distinguishes `BUY_PACKAGE` from `BUY_OTHER_PACKAGE`?
8. **Does `atlas-cashshop` already persist purchase history** (FR-REC-2)?
9. **Is the gms_95 `CashShopOpen` omission intentional?** Unverified — FR-V95-1 settles it.
10. **Recipient locker capacity for gifts** — reject the gift, or queue it?

## 10. Acceptance Criteria

- [ ] No arm in `cash_shop_operation.go` is a decode-log-return. Verified by reading
      every arm.
- [ ] `CashShopOperationBuyOtherPackage` is dispatched; `grep` shows it referenced
      beyond its declaration.
- [ ] GIFT delivers a locker asset to the named recipient's account with the sender's
      message; an unknown name charges nothing and returns a typed failure.
- [ ] A gift with a mismatched birthday charges nothing and returns a typed failure.
- [ ] BUY_PACKAGE creates one locker asset per member serial number, charges the
      package price once, and creates nothing when any member fails.
- [ ] `atlas-data` exposes cash packages over JSON:API; a known package id from
      `Etc.wz/CashPackage.img` resolves to its member serial numbers.
- [ ] BUY_COUPLE and BUY_FRIENDSHIP each create two paired ring assets sharing a
      `pair_id`; a failure creates neither.
- [ ] A character's ring pair is queryable by character id.
- [ ] REBATE_LOCKER_ITEM removes the asset and credits the original price; a replay
      credits once; an ineligible asset is rejected with a typed failure.
- [ ] APPLY_WISHLIST announces the stored wishlist; an empty wishlist yields a
      well-formed empty response.
- [ ] GET_PURCHASE_RECORD answers from durable purchase history, not from current
      locker contents.
- [ ] ENABLE_EQUIP_SLOT lets the character equip a second pendant, and the expansion
      survives a channel change.
- [ ] A v95 tenant can open the cash shop — confirmed by live behavior, not template
      diff alone. Or an `n-a` proof is recorded for v95 with derivation evidence.
- [ ] All new codecs have byte-fixture tests carrying `packet-audit:verify` markers,
      and the affected coverage-matrix cells promote.
- [ ] Every new entity is `tenant_id`-scoped; no query omits it.
- [ ] Every new Kafka command is idempotent under redelivery.
- [ ] No open question from §9 is resolved by an invented value; each is either derived
      with evidence or explicitly recorded as unverified.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
