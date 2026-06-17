# MTS (MapleStory Trade System) — Feature Scaffold & Research Note

> **Status:** pre-spec research. No code written. **Hard dependency: task-096
> (cfield-packet-family) must byte-verify the MTS opcodes before MTS coding
> begins.** This note is the input to a future `/spec-task`.
>
> **Sources:** Cosmic server (`~/source/Cosmic`, treated as an *unfaithful*
> behavioral reference), task-096 packet enumeration
> (`docs/packets/evidence/*/field.clientbound.FieldMts*`,
> `docs/packets/registry/*.yaml`, `MapleStory Ops - *.csv`), and direct v83
> client decompilation via IDA (`CITC*` functions, GMS v83 instance port 13342).

## 1. Summary

A **per-world player marketplace** where characters list inventory
items/equipment for sale in cash currency (not meso) and buy them. Sellers
list at a value; the system marks the buyer price up 10% and pays the seller
the list value as **Maple Points**. Supports **fixed-price sale, timed auction
(with buy-now), wish-list (zzim), and a transfer/take-home holding area**.
Entry is a cash-shop-style migration out of the field.

> **Note on "cart":** Cosmic's `mts_cart` (its ops 9/10/17) is *Cosmic's*
> implementation of the wish-list — it stubbed the real wish-list ops
> (3/4/12/13/14) and built its own table instead. There is **one** saved-items
> mechanism, faithfully the **wish-list (zzim)**; "cart" is not a separate
> system and is not in the client protocol. Do not model a distinct cart.

## 2. Locked decisions

| # | Decision | Value |
|---|----------|-------|
| Scope | Full | fixed-price + timed auction + buy-now + wish-list (zzim) + take-home |
| Market reach | Per-world | `(tenant_id, world_id)` scoped |
| Currency | Two buckets (verified) | NX Prepaid + Maple Points (NX Credit excluded — cash-shop only) |
| Buyer pays | NX Prepaid | marked-up price (list × 1.10) |
| Seller receives | Maple Points | the list value |
| 10% markup | House commission / NX sink | configurable rate |
| Versions | All 5 templated | gms_v83 / v84 / v87 / v95 / jms_v185 |
| Verification | task-096 first | packet-verifier gates coding |

### Economic policy — all tenant-level config
| Param | Default | Source |
|-------|---------|--------|
| Listing fee | 5,000 meso | Cosmic parity |
| Commission | 10% markup (buyer pays list×1.10; seller credited list value in Maple Points) | faithful GMS |
| Max active listings / char | 10 | Cosmic parity |
| Min level to enter | 10 | Cosmic parity |
| Auction window | 24–168 hrs, 1-hr step | **IDA-verified** |
| Price floor | 110 NX | **IDA-verified** |
| Items per page | 16 | Cosmic parity |

## 3. Verified against v83 IDA (do not re-derive from Cosmic)

- **2-bucket wallet:** `CITC::OnQueryCashResult` (op 0x15B/347) reads exactly
  2× `Decode4`; `CCashShop::OnQueryCashResult` reads 3. MTS wallet = NX Prepaid
  + Maple Points.
- **Price floor 110 NX:** `CITC::OnRegisterSaleEntry` validates `price > 109`;
  error string `PLEASE ENTER THE PRICE OF OVER 110 NX`. Cosmic's "110 meso" is a
  mislabel — the unit is NX.
- **Auction window 24–168 hrs, 1-hr increment:** error string
  `THE AUCTION CAN BE SET UP 24~168 HRS ... IN 1 HR INCREMENT`. Cosmic's flat
  7-day duration is just the max (168 hrs).
- **Two sale types:** `OnRegisterSaleEntry(arg0)` — `arg0=0` fixed-price
  (serverbound mode `2`), `arg0=1` auction (serverbound mode `0x12`/18). Cosmic
  stubbed auction → buy-now only; full scope restores it.
- **`ITC_STATUS_CHARGE` (0xFB)** is a bodiless "open NX recharge" hook, not a
  trade op.

## 4. Protocol inventory (full scope)

### Architecture: 3 standalone opcodes + 2 symmetric mode-dispatchers
- **Serverbound `ITC_OPERATION`** (v83 0xFD) — leading `Encode1(mode)` selects op.
- **Clientbound `MTS_OPERATION`** (v83 0x15C) — `CITC::OnNormalItemResult`
  switches on `Decode1(mode)`, cases **21–62** (38 result types).
- Standalone: `ENTER_MTS`, `ITC_STATUS_CHARGE` (0xFB), `ITC_QUERY_CASH_REQUEST`
  (0xFC) → `MTS_OPERATION2` (0x15B, 2× i32 wallet).

Per-version opcodes (from `docs/packets/registry/*.yaml` + CSV):

| Op | Dir | v83 | v87 | v95 |
|----|-----|-----|-----|-----|
| ENTER_MTS | SB | 0x9C | 0xA4 | 0xB4 |
| ITC_STATUS_CHARGE | SB | 0xFB | 0x109 | 0x132 |
| ITC_QUERY_CASH_REQUEST | SB | 0xFC | 0x10A | 0x133 |
| ITC_OPERATION | SB | 0xFD | 0x10B | 0x134 |
| MTS_OPERATION2 | CB | 0x15B | 0x170 | 0x19B |
| MTS_OPERATION | CB | 0x15C | 0x171 | 0x19C |

### Serverbound `ITC_OPERATION` modes — verified bodies (v83)
| Mode | Operation | Body (read order) | Validation |
|------|-----------|-------------------|------------|
| `2` | Register fixed-price sale | item-slot, `qty`(i32), `commodityId`(i32), `price`(i32), `type`(u8), `flag`(u8) | `price > 110 NX`, `qty > 0` |
| `3` | Sale current (sell selected item) | `type`(u8), `slotPos`(i32), item-slot, `commodityId`(i32) | confirm dialog |
| `0x12` (18) | Register auction | item-slot, `qty`(i32), `commodityId`(i32), `=1`(i32), `buyNowPrice`(i32), `type`(u8), `flag`(u8), `durationHrs`(i32) | duration 24–168 hrs |

**Remaining serverbound modes** (bodies for task-096): buy, buy-now-on-auction,
place-bid, set/buy/delete zzim, view/buy/cancel/register wish, cancel-sale,
move-ITC-purchase-LtoS (take home), change-category, change-category-sub,
change-page.

### Clientbound `MTS_OPERATION` result-mode table (verified switch, cases 21–62)
| Case(s) | Result group (task-096 `FieldMtsResult*` names) |
|---------|--------------------------------------------------|
| **21** ✅ | `GetItcListDone` — browse/search page (count + per-item ITCITEM records, `tab`/`type` bytes) |
| 22–24 | `GetSearchItcListDone`, `GetUserSaleItemDone/Failed` |
| 29–32 | `RegisterSaleEntryDone/Failed`, `SaleCurrentItemToWishDone` |
| 33–38 | `BuyItemDone/Failed`, `BuyZzimItemDone/Failed` |
| **39** ✅ | `MoveItcPurchaseItemLtoSDone` — take-home confirm (`tab`(i32), `selectedNo`(i32)) |
| 40–49 | `RegisterWishItemDone/Failed`, `BuyWishDone/Failed`, `CancelWishFailed` |
| 50–56 | `SetZzimDone/Failed`, cancel-sale |
| 60–62 | `BidAuctionFailed`, auction-state notices |

(Exact case→name pinning + all read orders are task-096's byte-verification
deliverable; only the dispatch *structure* and case *count* are verified here.)

## 5. User-facing operation set
1. Enter/exit MTS (migration; level ≥ 10 gate)
2. List item — fixed price or auction (24–168h, with buy-now)
3. Sell selected inventory item (quick-sell, mode 3)
4. Browse / search / paginate — by tab, category, sub-category, page; by item or seller
5. Buy (direct buy-now)
6. Auction — place bid, buy-now-on-auction
7. Wish-list (zzim) — set, buy-from-wish, delete, view, register wish, cancel wish
   (this *is* the "saved items" mechanism; Cosmic's "cart" is its own version of this)
8. Take home (LtoS) — move purchased/unsold/cancelled/expired item from holding → inventory
9. Cancel sale — return active listing to holding
10. Query wallet / recharge — 2-bucket display + recharge hook

## 6. Asset custody & transactional integrity (dupe-safety) — CRITICAL

Real MapleStory MTS was a notorious source of item-dupe exploits. Custody
integrity is a **first-class, non-negotiable requirement**, not a detail.

**Single-custody invariant:** a listed/in-transit item exists in **exactly one
place at every instant**. MTS is the sole custodian for the entire middle of the
journey; the item is never simultaneously in an inventory and in MTS, and never
in two inventories.

```
seller inventory ──list──▶ MTS(listed) ──buy/win──▶ MTS(buyer holding) ──take-home──▶ buyer inventory
```

A purchase/auction-win does NOT push the item into the buyer's inventory — it
moves custody to the buyer's **holding inside MTS**; the buyer pulls it on demand
(LtoS). Every hop is a custody transfer between exactly two owners, never a copy.

**Dupe vectors to close:**
1. List-without-remove — item listed *and* still in inventory (non-atomic removal).
2. Grant-before-debit / double-grant-on-retry — buyer gets item before/without
   currency settling, or twice on a replayed delivery.
3. Cancel racing purchase — listing returned to seller *and* delivered to buyer.
4. Take-home replay — item pulled to inventory but not cleared from holding.

**Mechanisms (all via saga + inventory reserve, never optimistic writes):**
- **Reserve → confirm → commit.** Listing uses inventory `RequestReserve` then
  `Consume`/`Release` so the item leaves inventory atomically with MTS taking
  custody; failure compensates to exactly one copy. Same family as the cash-shop
  sagas (`TransferToCashShop`/`WithdrawFromCashShop`/`AcceptToCashShop`/
  `ReleaseFromCashShop`).
- **One saga per money-moving op** — debit-buyer / credit-seller / move-custody
  are all-or-nothing with compensation; no grant-before-debit.
- **Idempotency keyed by `transactionId`** on every step — a replayed
  delivery/take-home is a no-op, not a second item.
- **MTS is the single source of truth for custody.** `inventory | listed |
  holding` are mutually exclusive states the service enforces; cancel-vs-buy
  resolves to one winner via the listing state, not a race.

## 7. Atlas implementation shape (mirrors atlas-cashshop)
- **New service `atlas-mts`** — immutable model+builder for `Listing` (seller,
  world_id, item snapshot/equip stats, list value, marked-up price, type
  fixed/auction, tab/type, auction window + current bid, state
  `active|sold|cancelled|expired`, expiry) + per-character transfer inventory.
- **Entity** — UUID PK + `tenant_id`, `(tenant_id, id)` unique index, name-keyed
  columns (avoids slug-PK-collision and binary-COPY column-order bug families).
- **Processor** — Interface+Impl, `Method`/`MethodAndEmit`.
- **REST** — JSON:API listings + search + transfer inventory.
- **Kafka** — `COMMAND_TOPIC_MTS` / `EVENT_TOPIC_MTS_STATUS`.
- **Wallet** — reuse atlas-cashshop `wallet.AdjustCurrency` (prepaid debit,
  points credit) via `EVENT_TOPIC_WALLET_STATUS`.
- **Saga** — list / buy / take-home are multi-step atomic ops modeled on the
  cash-shop sagas (`TransferToMts`, `WithdrawFromMts`; buy = debit buyer NX →
  credit seller Maple Points → move custody to buyer holding); timeout scaled to
  step count. **See §6 — every money/item move is saga-coordinated and
  idempotent for dupe-safety; no direct inventory writes.**
- **atlas-channel** — register 4 serverbound handlers (with **validators** +
  per-version config-driven `operations` mode tables for *every* version) and 2
  clientbound writers (each a mode dispatcher); migration mirrors `EnterCashShop`.

## 8. Sequencing
- **Blocker:** task-096 byte-verifies the 6 MTS ops across all 5 versions
  (serverbound mode bodies + clientbound result-mode bodies). Per the
  dispatcher-family rule, *every mode arm* needs its own byte fixture —
  enumerating mode bytes is not verification.
- MTS coding starts only after those matrix cells promote.
