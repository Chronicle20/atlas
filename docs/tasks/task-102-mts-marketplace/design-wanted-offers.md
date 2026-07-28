# MTS Wanted → Offer/Accept Fulfillment — Design

Status: Approved (decisions 1–3 confirmed by owner)
Created: 2026-07-10
Scope: close the want-ad ("Wanted") fulfillment loop in atlas-mts + atlas-channel.

## 1. Problem

A player posts a **want-ad** (a standing buy order: "I want item X for N NX") via
`REGISTER_WISH_ENTRY`. Other players who own X should be able to **offer** their
specific item; the poster **reviews the offers** (each item's real stats) and
**buys** the one they want. Today the loop is broken: `SALE_CURRENT_ITEM` creates
a generic public For-Sale listing (not attached to the want-ad), `VIEW_WISH`
returns the poster's own wishlist (not the offers), and `BUY_WISH` misses.

## 2. Client protocol (IDA-verified, v95 `CITC::*`)

`ITC_OPERATION`, serverbound opcode 308, mode byte from the tenant `operations` table.

| Client fn | v95 | Mode | Wire (serverbound) | Result (clientbound) |
|---|---|---|---|---|
| `OnSaleCurrentItem` | 0x5731a0 | 3 | `nItemTI, nSlotPos, GW_ItemSlotBase(item), nITCSN=want-ad serial` | `SaleCurrentItemToWishDone/Failed` (0x575d20/0x575d70) |
| `OnViewWish` | 0x5735c0 | 0xB | `nITCSN=want-ad serial` | `LoadWishSaleListDone` (0x5769a0) = offers on that want-ad |
| `OnBuyWish` | 0x573660 | 0xC | `nITCSN=offer serial` | `BuyWishDone/Failed` (0x576270/0x5762a0) |
| `OnCancelSaleItem` | 0x5737a0 | 7 | `nITCSN=item serial`, client guard `nBidCount==0` | `CancelSaleItemDone/Failed` |
| `OnRegisterWishEntry` | 0x573c10 | 4 | `itemId, price(≥110), count, duration, feeOption, desc` | `RegisterWishItemDone/Failed` |
| `OnGetUserSaleItemDone` | 0x576870 | — | server-populated Not-Yet-Sold (`m_wndSale`) | — |

Grounded conclusions:
- The **offer carries the full item** (`GW_ItemSlotBase`), so the poster reviews
  real stats. `VIEW_WISH` is keyed by the **want-ad serial**; `BUY_WISH` by the
  **offer serial**.
- **Seller reclaim is option B only**: `OnChangedCategorySub` (0x5739a0) shows
  every My-Page sub-tab sends a generic `GET_ITC_LIST`; per-item buttons there are
  hardwired (binary sub-flag in `sub_5BC1D5`) to Cart or Wish actions. **No My-Page
  sub-tab can emit `CANCEL_SALE`** — only the "Not Yet Sold" window does. Option A
  (a dedicated My-Page cancel view) does not exist in the client.

## 3. Model: an offer IS a listing (reuse, don't rebuild)

An offer is an item escrowed in custody, sold to one predetermined buyer (the
want-ad poster). That is a listing with a narrower audience — so reuse the
`listing` domain and its money-tested sagas instead of a parallel domain:

- New `SaleType` value **`offer`** (alongside `fixed`/`auction`).
- Two new `listing` columns: `offer_wish_serial` (the target want-ad's serial) and
  `offer_wish_owner_id` (the poster — captured at offer time so VIEW_WISH is a
  direct filter). Both 0/absent for normal listings (AutoMigrate adds them).
- `list_value` = the want-ad's **base** price (what the offerer nets). No listing
  fee for offers.

Everything else reuses existing machinery:

| Concern | Reuse |
|---|---|
| Escrow the offerer's item → custody | `TransferToMts` saga (`release_from_character` + `accept_to_mts_listing`); `AcceptToMtsListing` creates the `offer` listing from the payload snapshot (payload gains `SaleType/OfferWishSerial/OfferWishOwnerId`; fee step = 0). |
| Poster buys an offer (`BUY_WISH`) | `Buy(offerSerial)` with **buyer = poster** → existing `MtsSettlePurchase` (poster pays `MarkedUp(base)`, offerer nets base, item → poster holding). |
| Seller cancels an offer (`CANCEL_SALE`) | Existing cancel-listing path (`transitionToSellerHolding`) — an offer is a listing, so `GetBySerial` + owner-check + un-escrow to the offerer's holding all work unchanged. |
| Not-Yet-Sold shows offers | Already the seller's own active listings; offers are active listings owned by the offerer → they appear automatically. |
| History rows | `MtsSettlePurchase` custody settle already writes Purchased(poster)/Sold(offerer); cancel/release already writes Cancelled. |

Genuinely new logic is small: create an `offer` listing, browse offers by
want-ad serial, exclude `offer` from the public browse, and on an offer sale
**consume the want-ad + release sibling offers**.

## 4. Flows

### 4.1 Offer (`SALE_CURRENT_ITEM`, mode 3)
Channel resolves the target want-ad (`wish_serial` → owner id) and emits
`CREATE_LISTING` with `saleType=offer`, `offerWishSerial`, `offerWishOwnerId`,
`listValue = the want-ad's base price`, and the offerer's chosen item/asset.
atlas-mts runs `TransferToMts` (no fee) → `AcceptToMtsListing` creates the offer
listing. Result → `SaleCurrentItemToWishDone`; refresh the offerer's Not-Yet-Sold.
Reject if the want-ad serial no longer resolves (`SaleCurrentItemToWishFailed`).

### 4.2 View offers (`VIEW_WISH`, mode 0xB)
Channel `VIEW_WISH(wantAdSerial)` → browse listings where `saleType=offer AND
offer_wish_serial=wantAdSerial` → render each with the **full item snapshot** and
`nITCSN = offer.serial` → `LoadWishSaleListDone`. Replaces today's own-wishlist
rendering. Scoped to the want-ad owner (the requester).

### 4.3 Accept/buy (`BUY_WISH`, mode 0xC)
Channel `BUY_WISH(offerSerial)` → `Buy(serial, buyer = poster session)` (BuyNow=false).
Existing settle: poster prepaid −`MarkedUp(base)`, offerer points +base, item →
poster's Transfer Inventory. **After** the offer listing transitions sold (in the
custody move-to-holding tx): consume the target want-ad (remove the `wanted` wish),
and **release every sibling offer** on that want-ad → each un-escrows to its
offerer's holding with a Cancelled history row. Emit `BuyWishDone`.

### 4.4 Cancel offer (`CANCEL_SALE`, mode 7)
Unchanged wire; an offer is a listing so the existing cancel path un-escrows it to
the offerer's Transfer Inventory. Result `CancelSaleItemDone`.

### 4.5 Browse audience
- Public For-Sale / Auction browse: **exclude `saleType=offer`**.
- Not-Yet-Sold (offerer's own listings): **include** offers.

## 5. Money model (Option B, buyer-initiated)
The want-ad's stored `price` is the **base** (offerer nets it); the poster pays
`MarkedUp(base)`; commission is the sink. Same formula/direction as a normal sale —
the poster is the buyer. (Assumption to confirm in testing: `REGISTER_WISH_ENTRY`'s
price is the base, not the total; if it is the total, convert once at offer
creation.)

## 6. Task breakdown
1. `listing`: add `SaleTypeOffer`, `offer_wish_serial`, `offer_wish_owner_id`
   (entity/model/builder/provider/rest + AutoMigrate); browse filter gains
   `OfferWishSerial` and an `ExcludeOffers` flag; public browse sets ExcludeOffers.
2. `libs/atlas-saga` `TransferToMtsPayload` + atlas-mts `AcceptToMtsListing`:
   carry & persist `SaleType/OfferWishSerial/OfferWishOwnerId`; fee 0 for offers.
3. atlas-mts processor: `OfferItem` (create offer listing via TransferToMts);
   on offer sale — consume want-ad + release siblings (in the custody settle tx).
4. atlas-channel: rewire `SALE_CURRENT_ITEM` (→ CREATE_LISTING offer), `VIEW_WISH`
   (→ offers-by-wish browse), `BUY_WISH` (→ Buy as poster); status→client handlers
   `SaleCurrentItemToWishDone/Failed`, `LoadWishSaleListDone`, `BuyWishDone/Failed`;
   Not-Yet-Sold already unions offers.
5. Tests: offer escrow, view-offers, buy-wish settle (money split), cancel/release
   un-escrow, public-browse excludes offers.

## 7. Edge cases
- Concurrency: the `offered→sold` listing-state CAS is the single-buyer arbiter
  (existing settle guard).
- Want-ad cancelled while it has offers: release all its offers.
- Offer against a removed want-ad: reject at command time.
- `nBidCount` on an offer is 0, so the client's `CANCEL_SALE` guard permits cancel.
