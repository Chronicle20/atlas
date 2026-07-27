## Endpoints

All routes are mounted under the service base path `/api/`. Requests and
responses use the JSON:API convention (`jsonapi`/`api2go`); list endpoints
use the repo-wide `page[number]`/`page[size]` paging convention
(`paginate.ParseParams`).

### GET /worlds/{worldId}/listings

Browse/search active marketplace listings.

- Parameters: `worldId` (path, byte). Query: `page[number]`/`page[size]`
  (default page size 16, capped at `paginate.MaxPageSize`; invalid values
  are rejected, never silently clamped), `category`, `subCategory`,
  `saleType` (or legacy alias `type`), `sellerName`, `itemId`, `itemIds`
  (comma-separated template ids), `serial`, `serials` (comma-separated ITC
  serials), `sellerId`, `excludeSellerId`, `offerWishSerial`,
  `excludeOffers=true`.
- Request model: none.
- Response model: a paginated JSON:API list of listing `RestModel`
  (resource type `listings`), restricted to `state=active` rows; each
  entry is stamped with `contractFee` (the buyer-visible commission on top
  of the current price).
- Error conditions: 400 invalid `page[number]`/`page[size]`; 500 on a
  browse or count failure.

### POST /worlds/{worldId}/listings

Initiate a listing (validates against tenant configuration and emits the
fee-debit + custody-transfer saga). Does not create the listing row.

- Parameters: `worldId` (path, byte).
- Request model: `CreateListingRestModel` (resource type `listings`) —
  `sellerId`, `sellerAccountId`, `sellerName`, `sellerLevel`, `itemId`,
  `saleType`, `sourceInventoryType`, `assetId`, `quantity`, `listValue`,
  `buyNowPrice` (optional), `durationHours` (optional, auctions only),
  `category`, `subCategory`.
- Response model: 202 Accepted, `CreateListingRestModel` echoing the
  request plus the pre-allocated listing `id` (the row does not exist yet
  — it is created when the custody saga's accept step lands).
- Error conditions: 400 on any List validation failure (item not
  tradable/sellable, seller below the minimum level, price below the
  configured floor, seller at the active-listing cap, auction duration
  outside the configured range) or a saga-emit failure.

### GET /worlds/{worldId}/listings/{listingId}

Listing detail.

- Parameters: `worldId` (path, byte), `listingId` (path, UUID).
- Request model: none.
- Response model: listing `RestModel`, stamped with `contractFee`.
- Error conditions: 404 if the listing does not exist; 500 on any other
  read failure.

### DELETE /worlds/{worldId}/listings/{listingId}

Seller cancel of an active listing (race-safe `active -> holding(seller)`
transition).

- Parameters: `worldId` (path, byte), `listingId` (path, UUID);
  `characterId` (query, required — identifies the requesting seller for
  the owner-only check).
- Request model: none.
- Response model: no body.
- Error conditions: 400 missing/malformed `characterId`; 403 if the
  requester is not the listing's seller; 404 if the listing does not
  exist; 409 if the cancel lost the cancel-vs-buy race (the listing was no
  longer active); 204 on success.

### GET /characters/{characterId}/mts/holding

List a character's take-home holdings.

- Parameters: `characterId` (path, uint32). Query: optional `worldId`
  (narrows to one world; absent returns holdings across all worlds),
  `page[number]`/`page[size]` (default and cap both
  `paginate.MaxPageSize`).
- Request model: none.
- Response model: a paginated JSON:API list of holding `RestModel`
  (resource type `holdings`).
- Error conditions: 400 invalid `worldId` or invalid paging params; 500 on
  a read failure.

### POST /characters/{characterId}/mts/holding/{holdingId}/take-home

Initiate withdrawal of a holding into the owner's inventory (emits a
`WithdrawFromMts` saga). Does not soft-delete the holding directly.

- Parameters: `characterId` (path, uint32), `holdingId` (path, UUID).
- Request model: `TakeHomeRestModel` — `inventoryType` (destination
  inventory), `slot` (advisory target slot; not propagated to the saga).
- Response model: 202 Accepted, `TakeHomeRestModel` echoing the request
  plus the allocated saga transaction id in `id`.
- Error conditions: 404 if the holding does not exist; 403 if the
  requesting `characterId` is not the holding's owner; 500 on a
  saga-emit failure.

### GET /characters/{characterId}/mts/wishlist

List a character's wish-list entries.

- Parameters: `characterId` (path, uint32). Query: optional `type`
  (`cart` or `wanted`, narrows the result; absent returns the full
  wishlist), `page[number]`/`page[size]` (default and cap both
  `paginate.MaxPageSize`).
- Request model: none.
- Response model: a paginated JSON:API list of wish `RestModel` (resource
  type `wish-entries`).
- Error conditions: 400 invalid paging params; 500 on a read failure.

### POST /characters/{characterId}/mts/wishlist

Add a wish-list entry.

- Parameters: `characterId` (path, uint32).
- Request model: `RestModel` (resource type `wish-entries`) — only
  `worldId` and `itemId` are read from the request attributes;
  `characterId` comes from the path, and `id`/`serial`/`createdAt` are
  server-assigned.
- Response model: 201 Created, wish `RestModel`.
- Error conditions: 400 if the model fails to build; 500 on a create
  failure.

### DELETE /characters/{characterId}/mts/wishlist/{wishId}

Remove a wish-list entry.

- Parameters: `characterId` (path, uint32), `wishId` (path, UUID).
- Request model: none.
- Response model: no body.
- Error conditions: 400 malformed `wishId`; 404 if the entry does not
  exist; 204 on success.

### GET /worlds/{worldId}/mts/wishlist

Every want-ad (`type=wanted`) in a world, across all characters.

- Parameters: `worldId` (path, byte). Query: `page[number]`/`page[size]`
  (default and cap both `paginate.MaxPageSize`).
- Request model: none.
- Response model: a paginated JSON:API list of wish `RestModel`.
- Error conditions: 400 invalid paging params; 500 on a read failure.

### GET /characters/{characterId}/mts/transactions

A character's settled purchase/sale/bid-lost/cancelled history
(My Page -> History), newest-first. Read-only — rows are written
server-side at settle.

- Parameters: `characterId` (path, uint32). Query: `page[number]`/
  `page[size]` (default `paginate.DefaultPageSize`, capped at
  `paginate.MaxPageSize`).
- Request model: none.
- Response model: a paginated JSON:API list of transaction `RestModel`
  (resource type `transactions`).
- Error conditions: 400 invalid paging params; 500 on a read failure.

### GET /accounts/{accountId}/mts/wallet

Read-through view of an account's two MTS wallet buckets (NX Prepaid and
Maple Points) from atlas-cashshop.

- Parameters: `accountId` (path, uint32).
- Request model: none.
- Response model: `WalletRestModel` (resource type `wallets`) —
  `prepaid`, `points`.
- Error conditions: 500 if the upstream cash-shop read fails (the bare
  credit bucket, currencyType 1, is not an MTS bucket and is intentionally
  absent from the response).

## Non-Public Routes

`testsupport.InitResource` registers an additional `/test/*` route set
(listing seed, expire, sweep, simulated purchase/bid) that is compiled in
but mounted only when the `MTS_TEST_ROUTES_ENABLED` environment variable is
`"true"`. These routes are never exposed through ingress and are not part
of the service's public HTTP interface.
