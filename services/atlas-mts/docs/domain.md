## Listing

### Responsibility

Owns marketplace listings — fixed-price sales, auctions, and want-ad offers
— from creation through cancel, buy, bid, settle, and expire, including the
associated interactions with the Bid, Holding, and Transaction domains.

### Core Models

- `Model` (`listing/model.go`): `id`, `tenantId`, `worldId`, `serial` (the
  ITC serial), `sellerId`, `sellerAccountId`, `sellerName`, `saleType`,
  `state`, an item snapshot (`templateId`, `quantity`, and the full equip
  stat block: strength, dexterity, intelligence, luck, hp, mp, weaponAttack,
  magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands,
  speed, jump, slots, level, itemLevel, itemExp, ringId, viciousCount,
  flags), sale fields (`listValue`, `buyNowPrice`, `commissionRate`,
  `category`, `subCategory`), offer linkage (`offerWishSerial`,
  `offerWishOwnerId`), auction fields (`endsAt`, `currentBid`,
  `highBidderId`, `minIncrement`, `bidCount`), and `createdAt`/`updatedAt`.
  Constructed via `Builder` (`listing/builder.go`).
- `SaleType`: `fixed` | `auction` | `offer` (an offer is an item escrowed
  against a want-ad, sellable only to that want-ad's poster).
- `State`: `active` | `settling` | `sold` | `cancelled` | `expired`.
- `BrowseFilter` (`listing/provider.go`): the optional public-browse filter
  set — category, subCategory, saleType, itemId/itemIds, serial/serials,
  sellerId, excludeSellerId, sellerName, offerWishSerial, excludeOffers,
  page, pageSize.

### Invariants

- A listing's `serial` is unique within `(tenantId, worldId)` and drawn
  from the shared per-(tenant, world) serial counter (the Serial domain)
  also used by Holding and Wish rows, so a serial never resolves to more
  than one row within a world.
- A listing stores the seller's BASE price (`listValue`, `buyNowPrice`,
  `currentBid`); the buyer-facing marked-up price is computed at read/settle
  time via `MarkedUp(amount, commissionRate, commissionBase) =
  ceil(amount*(1+commissionRate)) + commissionBase` and is never persisted.
- `state` only ever moves `active -> {settling, sold, cancelled, expired}`;
  `settling` only moves to `sold`, or reverts to `active` if the settle
  saga fails to emit.
- Every `active -> {cancelled, expired, settling}` and `{active,
  settling} -> sold` transition is a race-safe conditional update (the
  transition requires the row's current state to still equal the expected
  `from` state), so concurrent callers (e.g. a cancel racing a buy) resolve
  to exactly one winner.
- An auction's `currentBid`/`highBidderId` only advance through a
  compare-and-swap keyed on the caller's previously-read prior bid/bidder;
  a concurrent bid that already advanced the row makes a stale caller lose.
- An offer listing (`saleType=offer`) is exempt from the seller's
  active-listing cap and the seller listing fee.
- A `settling` listing is excluded from the expiration sweep's discovery
  set (`state='active' AND ends_at<now`), preventing a second sweep tick
  from emitting a duplicate seller credit for the same auction.

### State Transitions

- `active -> settling -> sold` — auction settle path (`SettleAuction`
  claims the row synchronously, the async custody move completes it).
- `active -> sold` — buy / buy-now path (`Buy` emits the settle saga; the
  async custody move performs the transition).
- `active -> cancelled` — seller `Cancel`.
- `active -> expired` — `Expire`, applied by the periodic sweep to a
  no-bid auction or a fixed-price listing past its sale term.
- `settling -> active` — reverted if the settle saga fails to emit.
- `sold -> active` — `RestoreFromHolding`, a late-compensation reversal of
  a settlement move.

### Processors

- `Processor` (`listing/processor.go`, `listing/processor_custody.go`):
  `GetAll`/`GetById`/`GetBySerial`/`Create`, `Browse`/`CountBrowse`,
  `TransitionState`, `UpdateAuction`, `Cancel`/`CancelForSeller`/
  `CancelBySerial`, `Expire`, `List` (validates a list request against
  tenant configuration — price floor, active-listing cap, auction
  duration, sell-level gate, item tradability guards — and emits a
  `TransferToMts` saga), `Buy` (settles a buy/buy-now: computes the
  marked-up price, pre-checks the buyer's prepaid balance, emits a
  debit-first `MtsSettlePurchase` saga), `PlaceBid` (validates and
  advances an auction bid under a compare-and-swap, emits
  `MtsBidEscrow` hold/release sagas), `SettleAuction` (settles an expired
  auction to its winner, or expires it when there were no bids),
  `ReleaseSiblingOffers`, `ReleaseHighBidEscrow`, `Accept` (creates the
  row from a custody-carried snapshot), `SettleMove` (settles a purchase
  in one local transaction), `RemoveSpuriousActive`/`RestoreFromHolding`
  (late-compensation inverses of `Accept`/`SettleMove`).

## Bid

### Responsibility

Records escrowed bids placed on auction listings and their held/released/won
lifecycle. Owned and driven entirely by the Listing domain's bid/cancel/
settle flows.

### Core Models

- `Model` (`bid/model.go`): `id`, `tenantId`, `listingId`, `bidderId`,
  `bidderAccountId`, `amount` (the raw base bid), `escrowTxnId`, `state`,
  `createdAt`. Constructed via `Builder` (`bid/builder.go`).
- `State`: `held` | `released` | `won`.

### Invariants

- A bid's `state` only moves `held -> released` or `held -> won` through
  `UpdateState`'s race-safe conditional transition (requires the row's
  current state to still equal `from`).
- `amount` is the raw base bid (matching the listing's `currentBid` at bid
  time); the escrowed NX held for it is the marked-up amount, computed by
  the owning Listing flow and never stored on the bid row itself.

### Processors

- `Processor` (`bid/processor.go`): `GetAll`/`GetById`/`Create`,
  `GetByListingId`, `TransitionState`. There is no REST resource for bids
  — the Listing domain is the sole caller, placing and releasing bids as
  part of its bid/cancel/settle flows.

## Holding

### Responsibility

Owns the take-home custody bucket an item enters when it leaves a listing
without going directly to inventory (purchased, unsold/expired, cancelled),
until its owner withdraws it.

### Core Models

- `Model` (`holding/model.go`): `id`, `tenantId`, `worldId`, `serial` (the
  ITC serial), `ownerId`, `origin`, an item snapshot identical in shape to
  Listing's (`templateId`, `quantity`, and the full equip stat block), and
  `createdAt`. Constructed via `Builder` (`holding/builder.go`).
- `Origin`: `purchased` | `unsold` | `cancelled` | `expired`.

### Invariants

- A holding's `serial` is unique within `(tenantId, worldId)`, drawn from
  the same shared serial counter as Listing and Wish rows.
- Release (take-home) is a soft delete (`deleted_at`); a replayed release
  affects zero rows and is treated as success, not an error.
- A settlement-move buyer holding's id is deterministically derived from
  `(listingId, buyerId)` (`listing.MoveHoldingId`), so a replayed
  settlement move is idempotent.

### State Transitions

- `live -> released` — soft-deleted via `Release`, triggered by the
  take-home saga.
- `released -> live` — `RestoreHolding`, the compensating inverse of
  `Release`.

### Processors

- `Processor` (`holding/processor.go`, `holding/processor_custody.go`):
  `GetById`/`GetBySerial`/`Create`, `GetByOwner`/`GetByCharacter`,
  `ByOwnerPagedProvider`/`ByCharacterPagedProvider`, `TakeHome` (emits a
  `WithdrawFromMts` saga), `Release` (custody-driven soft delete),
  `RestoreHolding` (compensating un-delete).

## Wish

### Responsibility

Owns a character's standing interest in an item template: either a cart
entry (a favorited listing) or a wanted entry (a want-ad other players can
fulfill by creating an offer listing against it).

### Core Models

- `Model` (`wish/model.go`): `id`, `tenantId`, `worldId`, `serial` (the
  ITC serial), `characterId`, `itemId`, `listingSerial` (cart entries
  only), `wishType`, `price`, `count`, `expiresAt` (wanted entries only),
  `createdAt`. Constructed via `Builder` (`wish/builder.go`).
- `Type`: `cart` | `wanted`.

### Invariants

- At most one wish entry exists per `(tenantId, worldId, characterId,
  itemId, type)` — enforced by a unique index and `CreateWish`'s idempotent
  existence check (a duplicate create returns the existing row and
  consumes no new serial).
- A wish entry's `serial` is unique within `(tenantId, worldId)`, drawn
  from the same shared serial counter as Listing and Holding rows.
- A wanted entry's `price` is derived from the poster's commission-inclusive
  typed total via `WantAdBaseFromTotal` (Listing domain) down to the BASE
  the offerer nets; a cart entry's `price` is the favorited listing's list
  value, stored as-is.
- A wanted entry's `expiresAt` is set to the tenant's fixed-sale term at
  create time; a cart entry never expires (`expiresAt` is nil).
- `count` floors to 1 at create time.

### Processors

- `Processor` (`wish/processor.go`, `wish/processor_register.go`):
  `GetById`/`GetBySerial`/`Create`, `GetByCharacter`/
  `GetByCharacterAndType`/`GetWantedByWorld` and their paged variants,
  `Delete`/`DeleteBySerial`, `RegisterWish` (row-create plus price/expiry
  derivation), `RemoveWish`.

## Transaction

### Responsibility

Records the settled-fact purchase/sale/bid-lost/cancelled history a
character accumulates from MTS activity (My Page -> History).

### Core Models

- `Model` (`transaction/model.go`): `id`, `tenantId`, `worldId`,
  `characterId`, `counterpartyId`, `itemId`, `quantity`, `totalPrice`,
  `kind`, `createdAt`. Constructed via `Builder` (`transaction/builder.go`).
- `Kind`: `purchase` | `sale` | `bid_lost` | `cancelled`.

### Invariants

- A single settle writes exactly two rows — the buyer's purchase row and
  the seller's sale row — each owned by its own `characterId` with the
  other party recorded as `counterpartyId`.
- Rows are write-once, append-only history; there is no update or delete
  surface.

### Processors

- `Processor` (`transaction/processor.go`): `GetByCharacter`, `Create`,
  `ByCharacterPagedProvider`. Invoked by the Listing/Bid flows at
  settle/cancel/outbid time; never created independently.

## Configuration

### Responsibility

Resolves and caches the per-tenant economic knobs — listing fee, commission
rate/base, active-listing cap, sell-level gate, auction duration bounds,
fixed-sale term, price floor, page size, and minimum bid increment — that
gate and price every list/buy/bid flow.

### Core Models

- `Model` (`configuration/model.go`): the immutable resolved knob set, with
  `DefaultConfig()` supplying the fallback values used when a tenant has no
  configuration resource seeded.

### Invariants

- `Extract` (`configuration/rest.go`) substitutes `DefaultConfig`'s value
  for any knob the fetched resource left at its zero value, so a partial
  upstream configuration never yields a nonsensical zero (e.g. a 0%
  commission or a 0-hour auction).
- `Registry` (`configuration/registry.go`) is a per-tenant, process-wide
  cache (singleton via `sync.Once`): a fetch miss or error is cached as
  `DefaultConfig`, so a tenant without a seeded configuration never
  hard-fails a request and repeated lookups stay cheap.

### Processors

- `Registry.GetTenantConfig` is the sole read path. There is no
  processor/REST surface for writing configuration from atlas-mts — the
  resource is owned and seeded by atlas-tenants.

## Wallet

### Responsibility

Provides a read-only view of a cash-shop account's NX Prepaid and Maple
Points balances for the buy flow's pre-check and the wallet REST
passthrough endpoint. atlas-mts holds no wallet balance of its own.

### Core Models

- No locally persisted model. `RestModel` (`wallet/rest.go`) is the wire
  shape of atlas-cashshop's wallet resource (`accountId`, `credit`,
  `points`, `prepaid`), used only to decode the upstream REST response.

### Invariants

- atlas-mts never mutates a wallet directly; every balance change flows
  through the saga orchestrator's `AwardCurrency`/`MtsBidEscrow` steps
  executed against atlas-cashshop. This package only reads.

### Processors

- `Processor` (`wallet/processor.go`): `PrepaidBalance`, `Balance`,
  `EnsureWallet` (an idempotent wallet-bootstrap used only by the
  test-seed flow).

## Serial

### Responsibility

Assigns the persistent, per-(tenant, world) monotonic ITC serial (the
client's `nITCSN`) shared by Listing, Holding, and Wish rows, so a serial
addresses exactly one row across all three within a world.

### Core Models

- The `mts_serials` counter row (`serial/entity.go`): `(tenantId,
  worldId) -> nextSerial`. There is no exported domain `Model` type;
  `Next(db, tenantId, worldId)` is the sole entry point.

### Invariants

- The counter advance and the row insert it serves commit or roll back
  together — `Next` must be called inside the same transaction as the
  caller's row insert.
- The increment (`next_serial = next_serial + 1`) is computed by the
  database, never in application code, so two concurrent callers on the
  same `(tenant, world)` cannot compute the same next value.

### Processors

- No `Processor` type; `Next` is called directly by `CreateListing`,
  `CreateHolding`, and `CreateWish`.
