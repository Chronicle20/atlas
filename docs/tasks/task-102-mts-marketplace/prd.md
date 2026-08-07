# MTS (MapleStory Trade System) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-17
---

> Primary input: `research-scaffold.md` (same folder) — IDA-verified protocol
> facts, opcodes, mode tables, and the custody model. This PRD is self-contained;
> the scaffold is the supporting protocol reference.

## 1. Overview

MTS (MapleStory Trade System) is a **per-world player marketplace** where
characters list their own inventory items/equipment for sale and buy items other
players have listed, priced in cash currency rather than meso. It is the in-game
analogue of an auction site that sits beside the Cash Shop: players migrate into
a dedicated MTS stage (the same lifecycle as entering the Cash Shop), browse and
search listings, sell via fixed price or timed auction, maintain a wish-list, and
collect purchased/unsold items from a holding area.

The economic loop is the point of the feature: a **buyer pays in NX Prepaid**, a
**seller is paid in Maple Points** (which they then spend in the Cash Shop), and
the marketplace keeps a configurable commission as a currency sink. This gives
players a sanctioned way to trade for cash currency and gives operators a meso/NX
sink lever.

Item-custody integrity is the defining constraint. Real MapleStory MTS was a
notorious source of item-dupe exploits; therefore every item and currency
movement is **saga-coordinated, idempotent, and governed by a single-custody
invariant** (an item in transit exists in exactly one place at every instant).
The feature must support all five templated client versions
(gms_v83/v84/v87/v95, jms_v185).

This builds on task-096, which byte-verified the **clientbound** MTS result
packets (`MTS_OPERATION`, `MTS_OPERATION2`). The **serverbound** request packets
(`ENTER_MTS`, `ITC_STATUS_CHARGE`, `ITC_QUERY_CASH_REQUEST`, `ITC_OPERATION`) are
**not yet verified** and are addressed in Phase 0 of this task.

## 2. Goals

Primary goals:
- A new `atlas-mts` Go service owning marketplace listings, bids, wish-lists, and
  per-character transfer (take-home) holding, scoped per `(tenant_id, world_id)`.
- Full-scope trading: fixed-price sale, timed auction (24–168h) with buy-now,
  real-time competitive bidding (protocol permitting — see §9), wish-list (zzim),
  browse/search/paginate, take-home, cancel-sale, and automatic expiration sweep.
- Two-currency settlement: buyer debited NX Prepaid for the marked-up price;
  seller credited the list value in Maple Points; commission retained as sink.
- Strict single-custody dupe-safety: all item/money moves via the saga
  orchestrator with reserve→confirm→commit and `transactionId` idempotency.
- All economic parameters are per-tenant configuration (no hard-coded values).
- `atlas-channel` wiring for all five templated versions: serverbound handlers
  (with validators + per-version `operations` mode tables) and clientbound
  writers (mode dispatchers), plus cash-shop-style migration.
- An `atlas-ui` surface: tenant configuration pages for the economic knobs, and a
  read listings browser.
- Phase 0: byte-verify the serverbound MTS/ITC packets across all five versions
  before implementing the in-game flows.

Non-goals (this task):
- The "cart" and "history (past purchases)" My-Page tabs — semantics unresolved
  from the binary (StringPool labels live in `String.wz`); tabled for play-testing.
- Extracting the wallet out of `atlas-cashshop` into its own service — recorded as
  a recommended follow-up (see §9); task-102 routes through cash-shop as-is.
- Cross-world or global (cross-tenant) markets.
- Real-money NX purchasing (the existing cash-charge flow; `ITC_STATUS_CHARGE`
  only re-opens it).
- jms_v185 in-game MTS flows where the clientbound result packets are
  version-absent (⬜) — jms support is limited to whatever its opcode set defines.

## 3. User Stories

- As a **seller**, I want to list an item at a fixed price so that another player
  can buy it and I receive Maple Points.
- As a **seller**, I want to list an item as a timed auction with an optional
  buy-now price so that buyers compete and I get the best price.
- As a **seller**, I want to cancel an unsold listing so the item returns to my
  holding for take-home.
- As a **buyer**, I want to browse and search listings by category, item, or
  seller, paginated, so I can find what I want.
- As a **buyer**, I want to buy a fixed-price listing instantly so I receive the
  item in my holding to take home.
- As a **bidder**, I want to place bids on an auction and be notified if I'm
  outbid (protocol permitting) so I can compete in real time.
- As a **player**, I want a wish-list (zzim) of items I'm watching so I can act on
  them later.
- As a **player**, I want to pull purchased, unsold, cancelled, or expired items
  from my MTS holding into my inventory on demand.
- As a **player**, I want to see my MTS wallet balance (NX Prepaid + Maple Points).
- As an **operator/admin**, I want to configure the economic knobs (fee,
  commission, listing cap, level gate, auction window, price floor) per tenant via
  `atlas-ui`.
- As an **operator**, I want confidence that no trade can duplicate an item or
  desynchronize currency, even under crashes, retries, or races.

## 4. Functional Requirements

### 4.0 Phase 0 — Serverbound packet verification (prerequisite)
- Byte-verify and promote in the coverage matrix, for gms_v83/v84/v87/v95 (and
  jms_v185 where present): `ENTER_MTS`, `ITC_STATUS_CHARGE`,
  `ITC_QUERY_CASH_REQUEST`, and every `ITC_OPERATION` sub-mode (OnRegisterSaleEntry,
  OnSaleCurrentItem, OnBuy, OnBuyAuctionImm, OnSetZzim/OnBuyZzim/OnDeleteZzim,
  OnViewWish/OnBuyWish/OnCancelWish/OnRegisterWishEntry, OnCancelSaleItem,
  OnMoveITCPurchaseItemLtoS, OnChangedCategory/Sub, OnChangedPage, bid).
- Per the dispatcher-family rule, **each mode arm needs its own byte fixture** —
  enumerating mode bytes is not verification.
- Phase 0 also determines whether the protocol supports **server-pushed auction
  state** (live outbid notifications) — this gates the real-time-bidding decision
  in §9.

### 4.1 Entry / migration
- `ENTER_MTS` migrates the character out of the field into the MTS stage, mirroring
  `EnterCashShop` (save character, leave channel/map, open MTS).
- Entry is gated on a configurable minimum level (default 10) and the same
  map/event eligibility checks as cash-shop entry.
- On entry the client receives: the initial browse page, the character's active
  listings, their transfer (holding) inventory, and wallet balance.

### 4.2 Listing — fixed price
- A seller selects an inventory item, quantity (≥1), and list value (≥ price
  floor, default 110 NX).
- A configurable **listing fee** (default 5,000 meso) is debited from the
  character's meso at listing time.
- A character may hold at most a configurable number of active listings (default 10).
- The item is removed from inventory and taken into MTS custody atomically (§8).
- Listing carries the full item snapshot; equipment carries its complete stat
  block (str/dex/int/luk/hp/mp/watk/matk/wdef/mdef/acc/avoid/hands/speed/jump,
  upgrade slots, level, item level, item exp, ring id, vicious count, flags).

### 4.3 Listing — auction
- As 4.2, plus a buy-now price and a duration within a configurable window
  (default 24–168 hours, 1-hour increments).
- Auction tracks current high bid and high bidder; buy-now ends the auction early.

### 4.4 Browse / search / paginate
- Listings are browsable by tab and item category/sub-category, paginated at a
  configurable page size (default 16).
- Search by item (name/id) or by seller name, within the current world.
- All queries scoped to `(tenant_id, world_id)` and exclude items already in
  holding (`transfer` state).

### 4.5 Buy (fixed price / buy-now)
- Buyer must have NX Prepaid ≥ marked-up price (`list_value × (1 + commission)`).
- On purchase: buyer debited marked-up price (NX Prepaid); seller credited
  `list_value` (Maple Points); commission retained; listing custody moves to the
  buyer's holding. All steps are one saga (§8).
- Buyer is never granted the item directly into inventory — it lands in holding.

### 4.6 Auction bidding
- A bidder places a bid above the current high bid (and any configured minimum
  increment); bid funds are held in escrow (NX Prepaid).
- If outbid, the prior bidder's escrow is released; (real-time outbid notification
  is delivered if the protocol supports server push — §9).
- At expiry (or buy-now), the high bidder wins and settlement proceeds as 4.5.
- If an auction ends with no bids, the listing returns to the seller's holding.

### 4.7 Wish-list (zzim)
- Players can add, view, and remove wish-list entries, register a wish entry, and
  buy directly from the wish-list. This is the only "saved items" mechanism;
  Cosmic's "cart" is its own implementation of this and is not modeled separately.

### 4.8 Take-home (LtoS)
- A player moves a purchased/unsold/cancelled/expired item from their MTS holding
  into a chosen inventory slot. The item is granted to inventory and cleared from
  holding atomically and idempotently (§8) — a replayed take-home is a no-op.

### 4.9 Cancel sale
- A seller cancels an active (un-bid / fixed-price) listing; the item moves to the
  seller's holding for take-home. Cancel resolves against the listing state so it
  cannot race a concurrent purchase (§8).

### 4.10 Expiration sweep
- A ticker periodically moves expired listings to the seller's holding (the piece
  Cosmic omits). Expiry is enforced server-side, not display-only.

### 4.11 Wallet / recharge
- `ITC_QUERY_CASH_REQUEST` returns the two-bucket balance (NX Prepaid + Maple
  Points) via `MTS_OPERATION2`.
- `ITC_STATUS_CHARGE` re-opens the existing NX recharge flow (no new currency
  purchase logic).

### 4.12 Tenant configuration
- Economic knobs are loaded from an `atlas-tenants` configuration resource and are
  adjustable per tenant: listing fee, commission rate, commission model
  (buyer-markup), max active listings, min level, auction min/max hours, price
  floor, page size, min bid increment.
- Socket handler/writer opcodes and the per-version `operations` mode tables for
  the MTS packets are seeded into tenant config for every templated version.

## 5. API Surface

JSON:API conventions (`api2go/jsonapi`); all resources tenant-scoped via context;
world scoping via path/query.

### 5.1 `atlas-mts` REST (new)
- `GET /worlds/{worldId}/listings` — browse/search (query: `category`,
  `subCategory`, `type`, `page`, `pageSize`, `itemId`, `sellerName`, `saleType`).
- `GET /worlds/{worldId}/listings/{listingId}` — listing detail (incl. auction
  state: current bid, high bidder, ends-at).
- `POST /worlds/{worldId}/listings` — create listing (JSON:API envelope;
  attributes: `saleType`, `itemSnapshot`, `quantity`, `listValue`, `buyNowPrice?`,
  `durationHours?`). Initiates the list saga.
- `DELETE /worlds/{worldId}/listings/{listingId}` — cancel sale (seller only).
- `POST /worlds/{worldId}/listings/{listingId}/bids` — place bid.
- `POST /worlds/{worldId}/listings/{listingId}/buy` — buy / buy-now.
- `GET /characters/{characterId}/mts/holding` — transfer (take-home) inventory.
- `POST /characters/{characterId}/mts/holding/{holdingId}/take-home` — LtoS
  (attributes: target `inventoryType`, `slot`).
- `GET /characters/{characterId}/mts/wishlist` + `POST` / `DELETE`.
- `GET /characters/{characterId}/mts/wallet` — passthrough/aggregate of the
  cash-shop wallet's prepaid+points (read).
- `GET /tenants/{tenantId}/mts/config` + `PATCH` — economic knobs (admin).
- Error cases: insufficient funds (NX), price below floor, listing cap reached,
  level too low, item not owned/already reserved, listing not active (cancel vs
  buy race), duration out of range, bid below current+increment.

### 5.2 Kafka
- `COMMAND_TOPIC_MTS` — commands: CreateListing, CancelListing, PlaceBid, Buy,
  TakeHome, ExpireListing, RegisterWish, RemoveWish.
- `EVENT_TOPIC_MTS_STATUS` — events: ListingCreated, ListingCancelled, BidPlaced,
  Outbid, ListingSold, ListingExpired, ItemMovedToHolding, ItemTakenHome,
  WishAdded, WishRemoved (all carry `transactionId`, `worldId`).
- Reuses `EVENT_TOPIC_WALLET_STATUS` (cash-shop) for prepaid debit / points credit
  via the saga; inventory compartment commands (RequestReserve, Consume, Release,
  CreateAsset) for item custody; a character meso-debit command for the listing fee.

### 5.3 `atlas-channel` socket
- Serverbound handlers (each with a validator): `ENTER_MTS`, `ITC_STATUS_CHARGE`,
  `ITC_QUERY_CASH_REQUEST`, `ITC_OPERATION` (config-driven `operations` mode table,
  per version).
- Clientbound writers: `MTS_OPERATION` (result-mode dispatcher, cases 21–62),
  `MTS_OPERATION2` (wallet, 2× i32). Migration handler mirrors `EnterCashShop`.

### 5.4 `atlas-ui`
- Tenant config page surfacing the §4.12 economic knobs (react-hook-form + Zod).
- Read-only listings browser (per world, search/paginate) over `atlas-mts` REST.

## 6. Data Model

All entities: UUID PK + `tenant_id`, with `(tenant_id, id)` unique index and
**explicit name-keyed columns** (avoid the slug-only-PK and binary-COPY
column-order bug families). World scoping via a `world_id` column.

### 6.1 `Listing`
- `id`, `tenant_id`, `world_id`, `seller_id`, `seller_name`
- `sale_type` (`fixed` | `auction`), `state` (`active` | `sold` | `cancelled` |
  `expired`)
- `item_snapshot` (template id, quantity, and full equip stat block when equipment)
- `list_value` (NX), `buy_now_price` (NX, nullable), `commission_rate` (captured at
  list time), `category`, `sub_category`
- auction: `ends_at`, `current_bid`, `high_bidder_id`, `min_increment`
- `created_at`, `updated_at`
- Indexes: `(tenant_id, world_id, state, category)` for browse;
  `(tenant_id, seller_id, state)` for "my listings"; `(tenant_id, world_id,
  ends_at)` for the expiration sweep.

### 6.2 `Bid` (auction)
- `id`, `tenant_id`, `listing_id`, `bidder_id`, `amount`, `escrow_txn_id`,
  `state` (`held` | `released` | `won`), `created_at`.

### 6.3 `Holding` (transfer / take-home)
- `id`, `tenant_id`, `world_id`, `owner_id`, `item_snapshot`, `origin`
  (`purchased` | `unsold` | `cancelled` | `expired`), `created_at`.

### 6.4 `WishEntry`
- `id`, `tenant_id`, `character_id`, `item_id` / criteria, `created_at`.

### 6.5 Config (in `atlas-tenants`)
- Resource `"mts-config"`: listing fee, commission rate/model, max listings, min
  level, auction min/max hours, price floor, page size, min bid increment.

**Migration notes:** new tables created in `atlas-mts` via AutoMigrate with
explicit DDL for the unique indexes; no changes to existing tables. New
`atlas-tenants` resource follows the generic JSONB configuration pattern.

## 7. Service Impact

- **`atlas-mts` (new):** owns listings, bids, holding, wish-list; REST + Kafka;
  expiration ticker; saga participant. Registered in `services.json` AND
  `docker-bake.hcl` `go_services`; `go.work` + root `Dockerfile` COPY lines if a
  new shared lib is introduced; k8s manifests.
- **`atlas-channel`:** new serverbound handlers + clientbound writers + MTS
  migration; per-version `operations` mode tables; saga initiation for trade ops.
- **`atlas-cashshop`:** wallet debit (prepaid) / credit (points) consumed via
  Kafka/saga (no API change expected; confirm against the
  `legacy-atlas-cashshop-remediation` work).
- **`atlas-inventory`:** item custody via reserve/consume/release/create-asset
  commands.
- **`atlas-saga-orchestrator`:** new MTS saga types (TransferToMts, WithdrawFromMts,
  Buy/Settle, TakeHome) with compensation.
- **`atlas-tenants`:** new `"mts-config"` resource + the MTS socket opcode/mode
  tables for all five versions.
- **`atlas-character` (or meso owner):** meso debit for the listing fee.
- **`atlas-ui`:** tenant config page + listings browser.
- **deploy/k8s:** new service manifests; confirm whether MTS needs its own socket
  ports or runs as a channel-migrated stage (cash-shop model → no new ports).

## 8. Non-Functional Requirements

### 8.1 Asset custody & dupe-safety (CRITICAL)
- **Single-custody invariant:** a listed/in-transit item is in exactly one of
  `inventory | mts-listed | mts-holding | inventory(buyer)` at every instant —
  never two. MTS is the sole custodian for the middle of the journey.
- A purchase/win moves custody to the buyer's **holding inside MTS**, never
  directly to inventory; the buyer pulls on demand (LtoS).
- All item and currency moves are **saga-coordinated** with
  reserve→confirm→commit and compensation; **no optimistic/direct inventory
  writes**.
- Every saga step is **idempotent, keyed by `transactionId`**; replayed
  deliveries/take-homes are no-ops, not duplicates.
- Cancel-vs-buy and bid races resolve via the authoritative listing state (single
  source of truth), not timing.
- Test plan must include: crash-mid-list, grant-before-debit, double-grant replay,
  cancel-racing-purchase, and take-home replay.

### 8.2 Multi-tenancy & versioning
- All data and config tenant-scoped; market scoped per world.
- Behavior correct across gms_v83/v84/v87/v95 and jms_v185; version differences
  (opcodes, mode tables) come from tenant config, never hard-coded.

### 8.3 Performance & observability
- Browse/search served from indexed queries; pagination bounded.
- Expiration sweep bounded and logged (no silent truncation of swept listings).
- Structured logs + metrics on every saga step and trade event; failed
  settlements surfaced, not swallowed.

### 8.4 Security
- Sellers can only cancel/modify their own listings; buyers can only take-home
  from their own holding (tenant + owner checks).
- Server-authoritative pricing/commission (client-supplied prices validated
  against floor and config).

## 9. Open Questions

1. **Real-time bidding (protocol support):** Phase 0 must confirm whether the
   verified packet set includes a server-pushed auction-state / outbid packet. If
   yes, implement live competitive bidding (outbid notifications; consider
   anti-snipe extension). **If not, fall back to escrowed highest-bid-wins-at-expiry
   with buy-now early-end** (still full auction, just no live push).
2. **Wallet extraction follow-up:** routing MTS currency through `atlas-cashshop`
   is a deliberate design smell. **Recommended follow-up task:** extract the wallet
   (credit/points/prepaid) into its own service consumed by both cash-shop and
   MTS. Out of scope for task-102; to be filed after.
3. **Cart / history My-Page tabs:** semantics unresolved from the binary; confirm
   during play-testing, then spec if real.
4. **jms_v185 scope:** clientbound MTS result packets are version-absent (⬜) for
   jms; determine the supported jms surface in Phase 0.
5. **Listing-fee currency owner:** confirm which service owns character meso and
   the debit command for the 5,000-meso fee.

## 10. Acceptance Criteria

- [ ] Phase 0: all serverbound MTS/ITC packets byte-verified and matrix-promoted
      for gms_v83/v84/v87/v95 (+jms where present), each mode arm fixtured.
- [ ] Phase 0: real-time-bidding protocol support determined and recorded; §9.1
      resolved to live-push or escrow-at-expiry.
- [ ] `atlas-mts` service exists with Listing/Bid/Holding/WishEntry models,
      processors, JSON:API REST, Kafka command/event topics, and the expiration
      ticker; registered in `services.json` + `docker-bake.hcl`; builds via
      `docker buildx bake atlas-mts`.
- [ ] A character can list (fixed + auction), and the item leaves inventory into
      MTS custody atomically; the listing fee is debited; listing caps and price
      floor enforced.
- [ ] A buyer can buy-now and bid; settlement debits buyer NX Prepaid (marked-up),
      credits seller Maple Points (list value), retains commission, and moves the
      item to the buyer's holding — all in one saga with compensation.
- [ ] Take-home moves an item from holding to inventory idempotently; replay is a
      no-op.
- [ ] Cancel and expiration return items to the seller's holding; cancel cannot
      race a purchase to duplicate an item.
- [ ] Wish-list add/view/remove/buy works.
- [ ] Economic knobs are tenant-configurable via `atlas-tenants` and editable in
      `atlas-ui`; a listings browser renders per-world listings.
- [ ] All trade flows function across all five templated versions; opcodes/mode
      tables sourced from tenant config; every socket handler has a validator.
- [ ] Dupe-safety test suite passes: crash-mid-list, grant-before-debit,
      double-grant replay, cancel-vs-buy race, take-home replay — none duplicate an
      item or desync currency.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...`,
      `docker buildx bake` for changed services, and `tools/redis-key-guard.sh`
      all clean.
