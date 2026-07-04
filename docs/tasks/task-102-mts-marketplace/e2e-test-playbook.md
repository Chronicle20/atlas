# MTS E2E test playbook (task-102)

This playbook drives the `atlas-mts` marketplace end-to-end against a deployed
PR environment, using a real v83 client for the parts a human must observe and
the env-gated `/test/*` routes (`services/atlas-mts/atlas.com/mts/testsupport/`)
for the parts that need time-travel or a second actor. It exercises 8 scenarios;
each lists the exact curl(s), what to look for in the client, what to look for
over REST, and what to look for in logs.

No scenario here modifies `deploy/shared/routes.conf` — the `/test/*` routes
are **not** exposed through the shared ingress by design (they are reachable
only via a direct port-forward to the `atlas-mts` pod).

## Prereqs

1. A PR env with `atlas-mts` deployed, and a real v83 client logged into it
   with two characters available:
   - **Main** — the character you play from the client throughout.
   - **Alt** — a second character on the same tenant/world, used only as the
     "fake actor" for the `/test/purchases` and `/test/bids` routes (task-102
     never needs a second live client — the alt's actions are simulated).

2. Enable the test routes on the deployment (env-gated; there is deliberately
   no ingress route for them — see `testsupport/resource.go` package doc):

   ```bash
   kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED=true
   ```

3. Port-forward directly to the `atlas-mts` pod (bypasses the ingress
   entirely, so both the production mts REST routes and the `/test/*` routes
   are reachable at the same base URL):

   ```bash
   kubectl port-forward svc/atlas-mts 8080:8080
   ```

4. Header block and base URLs, reused verbatim across every curl below:

   ```bash
   H=(-H "TENANT_ID: <uuid>" -H "REGION: GMS" -H "MAJOR_VERSION: 83" -H "MINOR_VERSION: 1" -H "Content-Type: application/json")
   MTS="http://localhost:8080/api"

   # atlas-cashshop is NOT port-forwarded in this playbook (only its wallet
   # top-up is needed, once, in step 5) — reach it through the PR env's own
   # ingress. Replace <your-pr-env-ingress-host> with that env's actual host
   # (e.g. from `kubectl get ingress`); never hardcode a shared cluster host.
   INGRESS="https://<your-pr-env-ingress-host>"
   ```

   IDs you'll reuse throughout (fill in from your client session / DB):

   ```bash
   WORLD_ID=0
   MY_CHARACTER_ID=<your main character's id>
   MY_ACCOUNT_ID=<your main character's account id>
   ALT_CHARACTER_ID=<your alt character's id>
   ALT_ACCOUNT_ID=<your alt character's account id>
   ```

5. Top up the alt's NX so it can afford to buy/bid as the fake actor. The
   `PATCH /api/accounts/{accountId}/wallet` route on atlas-cashshop
   (`services/atlas-cashshop/atlas.com/cashshop/wallet/resource.go:82-105`,
   `wallet.ProcessorImpl.UpdateAndEmit`,
   `services/atlas-cashshop/atlas.com/cashshop/wallet/processor.go:115-117`)
   sets `credit`/`points`/`prepaid` as **absolute replacement values, not a
   delta** — so read the current wallet first and echo `credit`/`points`
   back unchanged, only raising `prepaid`:

   ```bash
   # 1) Read the alt's current wallet (field names verified in
   #    wallet/rest.go:7-13 — RestModel{AccountId, Credit, Points, Prepaid}).
   curl -s "${H[@]}" "$INGRESS/api/accounts/${ALT_ACCOUNT_ID}/wallet"
   # => {"data":{"type":"wallets","id":"...","attributes":{"accountId":...,"credit":<C>,"points":<P>,"prepaid":<X>}}}

   # 2) PATCH it, keeping credit/points as read and setting prepaid to 50000
   #    (comfortably above the tenant's default 110 NX price floor plus the
   #    commission markup — up to 10%+500 NX on seeded listings, tenant-config
   #    rate (default 7%)+500 NX on client-listed ones; see the economics
   #    cheat-sheet below and configuration/model.go:143-156 DefaultConfig).
   curl -s "${H[@]}" -X PATCH "$INGRESS/api/accounts/${ALT_ACCOUNT_ID}/wallet" \
     -d '{"data":{"type":"wallets","attributes":{"credit":<C>,"points":<P>,"prepaid":50000}}}'
   ```

   Scenario 4 also needs YOUR (main) account to have enough NX prepaid to
   place a bid from the client — top it up the same way against
   `${MY_ACCOUNT_ID}` if needed.

6. When you're done, disable the test routes:

   ```bash
   kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED-
   ```

### Economics cheat-sheet (informational only)

Per `services/atlas-mts/atlas.com/mts/configuration/model.go` `DefaultConfig()`
(used when the tenant has no MTS configuration resource — your tenant may
override these):

| Knob | Default | Meaning |
|---|---|---|
| `priceFloor` | 110 NX | minimum `listValue` for a listing |
| `commissionRate` | 0.07 | buyer markup rate (client-listed listings only — see below) |
| `commissionBase` | 500 NX | flat NX added to the markup |
| `maxActiveListings` | 10 | active listings per seller |
| `auctionMinHours` / `auctionMaxHours` | 24 / 168 | allowed auction duration via the real client flow (`/test/listings/seed` bypasses this — see Scenario 1) |

A buyer/bidder pays `ceil(priceBasis * (1 + rate)) + 500`; the seller is
credited exactly `priceBasis` (the fixed listValue, the buy-now price, or the
winning bid); the difference is the sink and is never credited to anyone
(`listing/processor.go` `Buy`/`markedUp` doc comments).

**Which rate applies:** `rate` is the listing ROW's captured
`commissionRate`, not the tenant config at buy/bid time — `Buy` and
`PlaceBid` both read `lm.CommissionRate()` from the row
(`listing/processor.go:554,739,760`). **Seeded listings carry a hardcoded
10% rate** (`testsupport/resource.go:272` `SetCommissionRate(0.10)`);
listings created through the real client flow capture the tenant config
rate (default 7%). Only `commissionBase` (500) comes from config at
buy/bid time. So all arithmetic below against seeded listings uses 0.10.

---

## Scenario 1 — Browse volume: seed 60 mixed listings, browse every tab/page

**Purpose:** populate enough listings to exercise the client's pagination
(default page size 16 — `listing/provider.go:65` `DefaultPageSize = 16`) across
both the "For Sale" and "Auction" tabs and across item sub-categories.

**Curl:**

```bash
curl -s "${H[@]}" -X POST "$MTS/test/listings/seed" -d '{
  "data": {
    "type": "test-seeds",
    "attributes": {
      "worldId": 0,
      "entries": [
        {"saleType": "fixed",   "count": 20, "templateId": 1302000, "listValue": 1500},
        {"saleType": "fixed",   "count": 20, "templateId": 2000000, "quantity": 50, "listValue": 800},
        {"saleType": "auction", "count": 20, "templateId": 1302000, "listValue": 1000, "durationSeconds": 3600}
      ]
    }
  }
}' | jq .
```

Body shape verified against `testsupport/resource_test.go` `TestSeedListings`
(`services/atlas-mts/atlas.com/mts/testsupport/resource_test.go:68-74`) —
identical `entries` fields (`saleType`, `count`, `templateId`, `quantity`,
`listValue`, `durationSeconds`). `templateId 1302000` (one-handed sword,
equip) and `2000000` (generic consumable) are the same ids the repo's own
tests use (`libs/atlas-constants/item/constants_test.go:45` names 1302000 a
"one-handed sword"; `2000000` is asserted a "generic consumable" at
`constants_test.go:19-20`). Total requested = 60, well under the 200 cap
(`testsupport/rest.go` / `resource.go:36` `seedMaxListings`).

**Expected REST observation:**
- `201 Created`; `data` is a JSON:API array of 60 `"listings"` resources
  (`listing/rest.go` `RestModel`, `GetName()=="listings"`), each with a
  distinct non-zero `itcSn` (real ITC serial via `listing.CreateListing`) and
  `state:"active"`. Fixed rows have `category:"1"`; auction rows have
  `category:"3"` and a non-null `endsAt` (`testsupport/resource.go:250-260`
  category derivation).
- Browse everything: `curl -s "${H[@]}" "$MTS/worlds/${WORLD_ID}/listings?pageSize=100"` →
  60 rows. Narrow to auctions only:
  `"$MTS/worlds/${WORLD_ID}/listings?saleType=auction"` → 20 rows
  (`listing/resource.go:167-243` `handleBrowseListings` filters).

**Expected client observation:** log in as Main, open the MTS (via the MTS
NPC / cash-shop entrance). "For Sale" tab shows 40 fixed listings from seller
"TestSeller" split across the sword/consumable sub-tabs (their
`inventory.TypeFromItemId` differs, so they land in different item
sub-categories), paginated at ~16/page (3 pages). "Auction" tab shows 20
auction rows, each with an "ends in ~1h" style countdown.

**Expected log lines:** one line per seed call, at Info level:

```
[TEST ROUTE] Seeded [60] listings in world [0] for tenant [<tenant-uuid>].
```

(`testsupport/resource.go:310`.)

---

## Scenario 2 — I buy a seeded fixed listing (real client, full saga)

**Purpose:** exercise the genuine client-driven buy path (not a `/test/*`
simulate route) end-to-end: `CITC` buy packet → channel → atlas-mts
`CommandBuy` → `handleBuy` → `listing.Buy` → `MtsSettlePurchase` saga
(debit buyer prepaid `markedUp`, credit seller points `listValue`, move item
to buyer holding `origin=purchased`) — `listing/processor.go:497-586` doc
comment, `kafka/consumer/mts/consumer.go:250-290` `handleBuy`.

**Steps (client-driven, no curl for the buy itself):**
1. Read your baseline wallet before buying:
   ```bash
   curl -s "${H[@]}" "$MTS/accounts/${MY_ACCOUNT_ID}/mts/wallet"
   ```
   (`wallet/resource.go:52-73` — GET route is account-keyed, not
   character-keyed, because the authoritative wallet lives in atlas-cashshop
   and is account-scoped.)
2. In the client, browse "For Sale", pick one of the Scenario-1 fixed
   listings (seller "TestSeller"), and buy it.

**Expected client observation:** `BuyItemDone` success notice; the listing
disappears from browse; the purchased item shows up in the MTS "Item"/My
Page pending-pickup list (a `holding` row — see Scenario 8 for take-home).

**Expected REST observation:**
- `curl -s "${H[@]}" "$MTS/accounts/${MY_ACCOUNT_ID}/mts/wallet"` — your
  `prepaid` dropped by `ceil(listValue*1.10)+500` from step 1's baseline
  (seeded listings carry the hardcoded 10% rate — e.g. for the Scenario-1
  1500 NX sword: `1650+500 = 2150` NX).
- `curl -s "${H[@]}" "$MTS/characters/${MY_CHARACTER_ID}/mts/holding"`
  (`holding/resource.go:18-38`) — one new row, `origin:"purchased"`.
- The listing's detail GET (`"$MTS/worlds/${WORLD_ID}/listings/<listingId>"`,
  works regardless of state, unlike browse which only shows `active`) now
  shows `state:"sold"`.

**Expected log lines:** atlas-mts logs nothing on a *successful* buy
(`handleBuy` only calls `Errorf`/emits `BUY_FAILED` on failure —
`kafka/consumer/mts/consumer.go:259-289`); confirm success via the REST
checks above, or watch `atlas-saga-orchestrator`'s logs for the
`mts_settle_purchase` saga's steps completing.

---

## Scenario 3 — Fake buyer buys MY listing (`/test/purchases`)

**Purpose:** verify the seller side of a sale without needing a second live
client — the alt "buys" a listing you created for real.

**Steps:**
1. In the client, list one of your own items for sale (fixed price) via the
   normal "Register For Sale" flow.
2. Find its listing id:
   ```bash
   curl -s "${H[@]}" "$MTS/worlds/${WORLD_ID}/listings?sellerId=${MY_CHARACTER_ID}" | jq .
   LISTING_ID=<id from the response>
   ```
3. Simulate the alt buying it:

   ```bash
   curl -s "${H[@]}" -X POST "$MTS/test/purchases" -d '{
     "data": {
       "type": "test-purchases",
       "attributes": {
         "listingId": "'"$LISTING_ID"'",
         "buyerId": '"$ALT_CHARACTER_ID"',
         "buyerAccountId": '"$ALT_ACCOUNT_ID"',
         "buyNow": false
       }
     }
   }'
   ```

   Body shape verified against
   `testsupport/simulate_test.go:163-168`
   (`TestSimulatePurchaseEmitsBuyCommand`) — identical field set
   (`listingId`, `buyerId`, `buyerAccountId`, `buyNow`).

**Expected response:** `202 Accepted` — command emitted, not yet settled
(`testsupport/resource.go:96-101` doc comment).

**Expected client observation:** on your NEXT MTS panel refresh (or
re-opening My Page / re-browsing) the listing is gone (sold), and your NX
Points balance (visible in the cash shop / MTS wallet display) increased.
There is no client-visible "someone bought it" popup for an already-closed
client session — the notice (`BuyItemDone`-style seller refresh) only fires
if your client is actively subscribed; the REST checks below are the
authoritative confirmation either way.

**Expected REST observation:**
- `curl -s "${H[@]}" "$MTS/accounts/${MY_ACCOUNT_ID}/mts/wallet"` — your
  `points` increased by the listing's `listValue` (seller is credited
  points, never prepaid — `listing/processor.go` `Buy` doc comment: "the
  seller still receives only listValue").
- `curl -s "${H[@]}" "$MTS/worlds/${WORLD_ID}/listings/${LISTING_ID}"` →
  `state:"sold"`.
- `curl -s "${H[@]}" "$MTS/characters/${MY_CHARACTER_ID}/mts/transactions"`
  (`transaction/resource.go:15-30`) — one new row, `kind:"sale"`,
  `counterpartyId:${ALT_CHARACTER_ID}`, `totalPrice` = the listing's
  `listValue`.

**Expected log lines:**

```
[TEST ROUTE] Emitted BUY txn [<uuid>] — buyer [<ALT_CHARACTER_ID>] listing [<LISTING_ID>] serial [<n>] buyNow [false].
```

(`testsupport/resource.go:136`, Info level.) If the buy is rejected
downstream (e.g. alt underfunded), atlas-mts instead logs, at Error level:
`Failed to settle buy for listing [...] (serial [...]), buyer [...], transaction [...].`
(`kafka/consumer/mts/consumer.go:284`) and emits `BUY_FAILED` — re-check the
alt's wallet top-up (Prereqs step 5) if you see this.

---

## Scenario 4 — Outbid escrow release

**Purpose:** verify that when the alt outbids you on an auction, YOUR
escrowed NX is released back to your wallet
(`listing/processor.go:614-635` `PlaceBid` doc comment: "On an outbid it
releases the prior bidder's escrow").

**Steps:**
1. Seed an auction:
   ```bash
   curl -s "${H[@]}" -X POST "$MTS/test/listings/seed" -d '{
     "data": {"type": "test-seeds", "attributes": {"worldId": 0, "entries": [
       {"saleType": "auction", "templateId": 1302000, "listValue": 1000, "durationSeconds": 1800}
     ]}}
   }' | jq .
   LISTING_ID=<id from the response>
   ```
2. In the client, find the seeded auction ("TestSeller") and place a bid of
   1200 NX. Make sure `${MY_ACCOUNT_ID}` has enough prepaid first
   (`ceil(1200*1.10)+500 = 1820` NX — seeded listings carry the hardcoded
   10% rate) — read your wallet:
   ```bash
   curl -s "${H[@]}" "$MTS/accounts/${MY_ACCOUNT_ID}/mts/wallet"
   ```
3. Simulate the alt outbidding you:

   ```bash
   curl -s "${H[@]}" -X POST "$MTS/test/bids" -d '{
     "data": {
       "type": "test-bids",
       "attributes": {
         "listingId": "'"$LISTING_ID"'",
         "bidderId": '"$ALT_CHARACTER_ID"',
         "bidderAccountId": '"$ALT_ACCOUNT_ID"',
         "amount": 1500
       }
     }
   }'
   ```

   Body shape verified against `testsupport/simulate_test.go:226-231`
   (`TestSimulateBidEmitsPlaceBidCommand`).

**Expected response:** `202 Accepted`.

**Expected client observation:** your wallet's NX Prepaid balance recovers
by the amount that had been held for your 1200 bid (`ceil(1200*1.10)+500 =
1820`); depending on client version/UI, an "outbid" notice may appear on
your next panel refresh.

**Expected REST observation:**
- `curl -s "${H[@]}" "$MTS/accounts/${MY_ACCOUNT_ID}/mts/wallet"` — `prepaid`
  is back to (approximately) its pre-bid level — the escrow hold for your
  1200 bid was released.
- `curl -s "${H[@]}" "$MTS/worlds/${WORLD_ID}/listings/${LISTING_ID}"` →
  `currentBid:1500`, `highBidderId:${ALT_CHARACTER_ID}`.

**Expected log lines:**

```
[TEST ROUTE] Emitted PLACE_BID txn [<uuid>] — bidder [<ALT_CHARACTER_ID>] listing [<LISTING_ID>] serial [<n>] amount [1500].
```

(`testsupport/resource.go:173`.) A rejected bid (e.g. below the floor
`currentBid + minIncrement`) instead logs, Error level: `Failed to place bid
for listing [...] (serial [...]), bidder [...], transaction [...].`
(`kafka/consumer/mts/consumer.go:335`) with `BID_FAILED` emitted.

---

## Scenario 5 — I win an auction

**Purpose:** verify the settle-to-winner path: seller credited, winner's
item lands in their holding (`origin:"purchased"` — same origin as a plain
buy, since the settle step reuses the same `mts_move_listing_to_holding`
custody command — `kafka/consumer/custody/consumer.go:378-381`), and the
winner is NOT re-debited at settle (their prepaid was already escrowed at
bid time — `listing/processor.go:780-830` `SettleAuction` doc comment).

**Steps:**
1. Seed a short-lived auction:
   ```bash
   curl -s "${H[@]}" -X POST "$MTS/test/listings/seed" -d '{
     "data": {"type": "test-seeds", "attributes": {"worldId": 0, "entries": [
       {"saleType": "auction", "templateId": 1302000, "listValue": 500, "durationSeconds": 60}
     ]}}
   }' | jq .
   LISTING_ID=<id from the response>
   ```
2. In the client, place ONE bid on it as Main (e.g. 600 NX — clears the
   500 listValue floor for a first bid; the escrow held from your prepaid is
   `ceil(600*1.10)+500 = 1160` NX, the seeded 10% rate). Confirm you are the
   high bidder:
   ```bash
   curl -s "${H[@]}" "$MTS/worlds/${WORLD_ID}/listings/${LISTING_ID}" | jq .
   ```
3. Force it to expire (rewrites `endsAt` to 1s ago — you do not need to
   actually wait out `durationSeconds`) and run one sweep:
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' "${H[@]}" -X POST "$MTS/test/listings/${LISTING_ID}/expire"
   curl -s "${H[@]}" -X POST "$MTS/test/sweep" | jq .
   ```

**Expected response:** expire → `204 No Content`; sweep → `200 OK` with body
`{"data":{"type":"test-sweeps","attributes":{"swept":<n>}}}` where `n>=1`
(may be higher than 1 if other auctions from earlier scenarios also expired
in the same tick — `testsupport/rest.go:37-50` `SweepResultRestModel`).

**Expected client observation:** on your next MTS/My Page refresh, the item
appears in your pending pickup (holding) list — see Scenario 8 for taking
it home.

**Expected REST observation:**
- `curl -s "${H[@]}" "$MTS/characters/${MY_CHARACTER_ID}/mts/holding"` — a
  new row, `origin:"purchased"`, `templateId:1302000`.
- Seller ("TestSeller", synthetic — no wallet to check) was credited
  `listValue` NX points via `award_currency`; you (winner) were NOT
  re-debited (your prepaid stayed at the level it was after step 2's bid
  escrow).
- Listing detail → `state:"sold"`.

**Expected log lines** (from the periodic sweep, `task/periodic.go`):

```
MTS expiration sweep: settled auction [<LISTING_ID>] -> winner [<MY_CHARACTER_ID>] holding (tenant [<uuid>]).
```

at Debug level (`task/periodic.go:149`) — requires the pod's log level at
Debug to see it directly; at Info level, one of the two summary lines
appears (`task/periodic.go:173-178` picks based on whether the 500-listing
batch cap deferred any items — normally the second):

```
MTS expiration sweep: expired/settled [<n>] listings this tick; [<m>] remain past the [500] batch cap and will be processed next tick.
MTS expiration sweep: expired/settled [<n>] of [<n>] discovered listings.
```

---

## Scenario 6 — Fake bidder wins MY auction

**Purpose:** the mirror of Scenario 5 from the seller's side — the alt wins
an auction you (Main) listed for real, and you are the one credited.

**Steps:**
1. In the client, list one of your own items as an auction (real "Register
   Auction" flow), short duration if the client allows, or accept whatever
   the client's minimum is.
2. Find its listing id:
   ```bash
   curl -s "${H[@]}" "$MTS/worlds/${WORLD_ID}/listings?sellerId=${MY_CHARACTER_ID}&saleType=auction" | jq .
   LISTING_ID=<id from the response>
   ```
3. Simulate the alt bidding (first bid, must clear `listValue`):
   ```bash
   curl -s "${H[@]}" -X POST "$MTS/test/bids" -d '{
     "data": {
       "type": "test-bids",
       "attributes": {
         "listingId": "'"$LISTING_ID"'",
         "bidderId": '"$ALT_CHARACTER_ID"',
         "bidderAccountId": '"$ALT_ACCOUNT_ID"',
         "amount": <amount clearing listValue>
       }
     }
   }'
   ```
4. Force-expire + sweep (as in Scenario 5):
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' "${H[@]}" -X POST "$MTS/test/listings/${LISTING_ID}/expire"
   curl -s "${H[@]}" -X POST "$MTS/test/sweep" | jq .
   ```

**Expected client observation:** your listing disappears from your active
listings; your NX Points balance increases by the winning bid amount on
your next panel refresh.

**Expected REST observation:**
- `curl -s "${H[@]}" "$MTS/accounts/${MY_ACCOUNT_ID}/mts/wallet"` — `points`
  increased by the alt's winning bid amount.
- The alt's holding row (not reachable through your character's GET —
  confirm via DB or, if you have a second client, the alt's own
  `GET "$MTS/characters/${ALT_CHARACTER_ID}/mts/holding"`) — one new row,
  `origin:"purchased"`.
- `curl -s "${H[@]}" "$MTS/characters/${MY_CHARACTER_ID}/mts/transactions"` —
  one new row, `kind:"sale"`, `counterpartyId:${ALT_CHARACTER_ID}`.

**Expected log lines:** same sweep lines as Scenario 5
(`task/periodic.go:149,173-178`), plus the bid's `[TEST ROUTE] Emitted
PLACE_BID ...` line (`testsupport/resource.go:173`) from step 3.

---

## Scenario 7 — No-bid expiry: item back in seller Transfer Inventory

**Purpose:** verify an auction with zero bids returns to the SELLER's
holding with `origin:"expired"` (not "purchased") and no money moves at all
(`listing/processor.go:297` `Expire` → `holding.OriginExpired`;
`task/periodic.go:159-167`).

**Steps:**
1. Seed an auction and let no one bid on it:
   ```bash
   curl -s "${H[@]}" -X POST "$MTS/test/listings/seed" -d '{
     "data": {"type": "test-seeds", "attributes": {"worldId": 0, "entries": [
       {"saleType": "auction", "templateId": 1302000, "listValue": 500, "durationSeconds": 60}
     ]}}
   }' | jq .
   LISTING_ID=<id from the response>
   ```
2. Expire + sweep with zero bids:
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' "${H[@]}" -X POST "$MTS/test/listings/${LISTING_ID}/expire"
   curl -s "${H[@]}" -X POST "$MTS/test/sweep" | jq .
   ```

**Expected client observation:** if this was YOUR listing (repeat step 1
via the real client "Register Auction" flow instead of seeding, to see it in
your own client), the item reappears in your MTS pending-pickup list with no
sale notice — nothing sold, no NX changed hands.

**Expected REST observation:**
- Listing detail → `state:"expired"` (not `"sold"`).
- The seller's holding (`999000001`'s synthetic seller for a seeded listing,
  or your own `GET "$MTS/characters/${MY_CHARACTER_ID}/mts/holding"` if you
  listed it yourself) — new row, `origin:"expired"`.
- No transaction-history rows are written for this listing (settle-side
  `transaction.CreateTransaction` calls only run in the has-a-winner arm of
  `handleMtsMoveListingToHolding` — the no-bid path never reaches there).

**Expected log lines:**

```
MTS expiration sweep: expired listing [<LISTING_ID>] -> seller [<sellerId>] holding (tenant [<uuid>]).
```

at Debug level (`task/periodic.go:166`), plus one of the two Info-level
summary lines (`task/periodic.go:173-178` — see Scenario 5).

---

## Scenario 8 — Take-home after each settle + transaction history for every scenario

**Purpose:** confirm the terminal step of every settle above — taking the
item out of MTS holding into your real inventory — and that every settle
wrote the transaction-history rows Scenarios 2/3/5/6 checked for.

**Steps — take home each pending holding (repeat per holding row):**
1. List your current holdings:
   ```bash
   curl -s "${H[@]}" "$MTS/characters/${MY_CHARACTER_ID}/mts/holding" | jq .
   HOLDING_ID=<id of the row you want to take home>
   ```
2. Either take it home in the client (My Page → take item), or drive it
   directly over REST:
   ```bash
   curl -s "${H[@]}" -X POST "$MTS/characters/${MY_CHARACTER_ID}/mts/holding/${HOLDING_ID}/take-home" -d '{
     "data": {
       "type": "holdings",
       "attributes": {
         "inventoryType": 1,
         "slot": 0
       }
     }
   }'
   ```
   `inventoryType:1` is Equip (`libs/atlas-constants/inventory/constants.go:12`
   `TypeValueEquip`) — the right destination for the seeded 1302000 sword;
   `slot` is advisory only (the grant auto-slots —
   `holding/rest.go:50-59` `TakeHomeRestModel` doc comment).

**Expected response:** `202 Accepted` — a `WithdrawFromMts` saga was
initiated, not yet completed (`holding/resource.go:40-47` doc comment).

**Expected client observation:** the item leaves the MTS pending-pickup list
and appears in your real inventory (Equip tab, since `inventoryType:1`).

**Expected REST observation:**
- `curl -s "${H[@]}" "$MTS/characters/${MY_CHARACTER_ID}/mts/holding"` — the
  taken-home row is gone (released by the saga's
  `ReleaseFromMtsHolding` custody step).

**Transaction history — verify every scenario above left the expected rows:**

```bash
curl -s "${H[@]}" "$MTS/characters/${MY_CHARACTER_ID}/mts/transactions" | jq .
```

(`transaction/resource.go:15-30`, `RestModel` fields `worldId`,
`characterId`, `counterpartyId`, `itemId`, `quantity`, `totalPrice`, `kind`,
`createdAt` — `transaction/rest.go:8-18`.) Expect, newest first:

| From | `kind` | `counterpartyId` | `totalPrice` |
|---|---|---|---|
| Scenario 2 (you bought) | `purchase` | the listing's seller (`999000001` for a seeded row) | the listing's `listValue`/buy-now price |
| Scenario 3 (alt bought yours) | `sale` | `${ALT_CHARACTER_ID}` | the listing's `listValue` |
| Scenario 5 (you won) | `purchase` | the seller (`999000001`) | your winning bid |
| Scenario 6 (alt won yours) | `sale` | `${ALT_CHARACTER_ID}` | the alt's winning bid |

Scenario 4 (outbid, no settle) and Scenario 7 (no-bid expiry) write **no**
transaction rows — both are confirmed by their absence here, consistent
with `kafka/consumer/custody/consumer.go:414-451` (the two
`transaction.CreateTransaction` calls live inside the has-a-winner /
has-a-buyer settle path only).

---

## Cleanup

```bash
kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED-
```
