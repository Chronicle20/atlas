# Domain

## Shop

### Responsibility

Represents a personal (character) shop or hired merchant placed in a Free Market room, and owns its lifecycle, listings, occupancy, blacklist/visit enforcement, and purchase settlement.

### Core Models

**Model** (`shop/model.go:11-29`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Shop identifier |
| characterId | uint32 | Owner character |
| shopType | ShopType | CharacterShop (1) or HiredMerchant (2) |
| state | State | Current lifecycle state |
| title | string | Shop display name |
| worldId | world.Id | World placement |
| channelId | channel.Id | Channel placement |
| mapId | uint32 | Map where the shop is placed |
| instanceId | uuid.UUID | Field instance |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| permitItemId | uint32 | Permit item used for placement |
| createdAt | time.Time | Creation timestamp |
| expiresAt | *time.Time | Expiration time (hired merchants: 24h) |
| closedAt | *time.Time | Closure timestamp |
| closeReason | CloseReason | Reason for closure |
| mesoBalance | uint32 | Accumulated meso balance (hired merchants) |

**ShopType** (`shop/state.go:8-13`) — alias of `merchantconst.ShopType` (`libs/atlas-constants/merchant/shop.go`)

| Value | Name |
|---|---|
| 1 | CharacterShop |
| 2 | HiredMerchant |

**State** (`shop/state.go:15-22`) — alias of `merchantconst.ShopState` (`libs/atlas-constants/merchant/shop.go`)

| Value | Name |
|---|---|
| 1 | Draft |
| 2 | Open |
| 3 | Maintenance |
| 4 | Closed |

**CloseReason** (`shop/state.go:24-34`)

| Value | Name |
|---|---|
| 0 | None |
| 1 | SoldOut |
| 2 | ManualClose |
| 3 | Disconnect |
| 4 | Expired |
| 5 | ServerRestart |
| 6 | Empty |

**LogoutOutcome** (`shop/state.go:38-49`) — policy result for a shop on owner logout: `LogoutNone` (0), `LogoutClose` (1), `LogoutExitMaintenance` (2). The `LogoutAction` policy (`shop/state.go:56-71`): closed shops are no-ops; any non-closed character shop closes; a hired merchant closes when Draft, exits maintenance when Maintenance, and is otherwise left running.

**PurchaseResult** (`shop/processor.go`)

Returned from PurchaseBundle. Contains listing details, cost breakdown (TotalCost, Fee, NetAmount), buyer/owner info, and whether the shop closed as a result.

### Invariants

- Shop id, characterId, shopType, and state are required (`shop/builder.go:121-133`).
- One active (non-Closed) shop per (character, shopType). A second placement returns `ErrShopLimitReached`.
- Shop must be placed in a Free Market room (`IsFreemarketRoom`, hardcoded map-id set in `shop/validation.go`).
- Shop cannot be placed within the portal proximity threshold (euclidean, `portalProximityThreshold = 120`) of a blocking portal (teleport-type portals with a real target map). Portal-fetch failure fails open.
- Shop cannot be placed within the shop proximity threshold (Manhattan, `shopProximityThreshold = 100`) of an existing shop on the same map. Provider failure fails open.
- Shop cannot be placed while the owner has items or mesos pending at Frederick (`ErrFrederickPending`).
- Maximum 16 listings per shop (`MaxListings`).
- Maximum 3 concurrent visitors per shop (`MaxVisitors`); at capacity, entry is rejected as capacity-full rather than erroring.
- Listing search returns at most 200 rows (`MaxSearchResults`).
- Listings require pricePerBundle >= 1 and bundleSize >= 1 (builder validation).
- Pets, cash-inventory items, and untradeable items cannot be listed (`IsListableItem`).
- Purchase uses optimistic locking on the listing version to prevent concurrent conflicts (`ErrVersionConflict`).
- Owner-only mutations are guarded by `requireOwner` (returns `ErrNotOwner` when the caller is not the owner): OpenShop, EnterMaintenance, ExitMaintenance, CloseShop, AddListing, RemoveListing, WithdrawMeso, OrganizeListings, AddToBlacklist, RemoveFromBlacklist. UpdateListing, PurchaseBundle, EnterShop, ExitShop, SendMessage, and RetrieveFrederick are deliberately not owner-guarded.

### State Transitions

```
(new)       -> Draft         (CreateShop)
Draft       -> Open          (OpenShop; requires at least one listing)
Open        -> Maintenance   (EnterMaintenance; ejects visitors)
Maintenance -> Open          (ExitMaintenance, if listings remain)
Maintenance -> Closed        (ExitMaintenance, CloseReasonEmpty, if no listings)
Open        -> Closed        (CloseShop; also SoldOut auto-close on last bundle)
Draft       -> Closed        (CloseShop)
Maintenance -> Closed        (CloseShop)
```

Listings may be added, removed, updated, or organized only in Draft or Maintenance. Entering maintenance and closing eject all visitors. Closing removes registry and map-index entries; for character shops the unsold items are returned to the owner's inventory (unless closed on disconnect), and for hired merchants the unsold items and accumulated meso balance are stored to Frederick. Visitor entry is rejected when the shop is not Open (distinguishing an under-maintenance shop from an otherwise-closed room) or when the visitor is blacklisted.

### Processors

**shop.Processor** (`shop/processor.go`)

Read/query: GetById, ByIdProvider, GetByCharacterId, GetByCharacterIdPaged, GetByField, GetByFieldPaged, GetAllOpenPaged, GetListingCounts, SearchListingsByItemIdPaged, GetListings, GetListingsPaged, GetExpired, GetBlacklistPaged, GetVisitsPaged, GetVisitors, GetShopForCharacter.

Mutating (each has an `...AndEmit` wrapper that runs the mutation in a transaction and emits via the outbox, except the Redis-only EnterShop/ExitShop paths which emit directly):

- CreateShop — validates placement, active-shop limit, and Frederick-pending; inserts a Draft shop and seeds the Redis registry.
- OpenShop — Draft to Open; requires at least one listing.
- EnterMaintenance — Open to Maintenance; ejects visitors.
- ExitMaintenance — Maintenance to Open, or to Closed (Empty) when no listings remain.
- CloseShop — transitions to Closed; returns items to owner (character shop) or stores items and meso balance to Frederick (hired merchant).
- AddListing, RemoveListing, UpdateListing, OrganizeListings — manage listings in Draft or Maintenance.
- WithdrawMeso — hired merchant only; zeroes the accumulated meso balance and credits the owner.
- PurchaseBundle — optimistic-locked bundle decrement, fee calculation, sold-out auto-close, meso settlement.
- EnterShop, ExitShop, EjectAllVisitors, GetVisitors, GetShopForCharacter — visitor occupancy (Redis).
- AddToBlacklist, RemoveFromBlacklist — maintain the shop blacklist; adding a currently-present banned character ejects them.
- SendMessage — persists a chat message and reports the sender's slot.
- RetrieveFrederick — grants pending Frederick items and mesos to the character.

**Fee Calculation** (`shop/processor.go` `GetFee`)

| Meso Amount | Fee Rate |
|---|---|
| < 100,000 | 0% |
| 100,000 – 999,999 | 0.8% |
| 1,000,000 – 4,999,999 | 1.8% |
| 5,000,000 – 9,999,999 | 3% |
| 10,000,000 – 24,999,999 | 4% |
| 25,000,000 – 99,999,999 | 5% |
| >= 100,000,000 | 6% |

Fee uses integer arithmetic. The buyer is charged the full total; the fee is retained by the server and the net amount is credited to the owner (hired merchants accumulate net into the meso balance).

**Occupancy Resolution** (`shop/processor.go` `GetShopForCharacter`)

Resolves the shop a character occupies via the transient visitor registry, then the active-shop registry (validated by `isOwnerOccupied`: a character shop is occupied while not Closed; a hired merchant is occupied only in Draft/Maintenance since an Open hired merchant runs detached), then a database fallback.

**Registry** (`shop/registry.go`) — Redis owner-occupancy cache: `activeShops` (per-character `ActiveShopEntry`) and `mapPlacement` (map-id to shop-id index).

**Expiration Task** (`shop/task.go`)

Background task (30s interval). Queries shops whose `expiresAt` has passed and whose state is Draft, Open, or Maintenance, across all tenants; reconstructs tenant context and closes each with CloseReasonExpired. Only hired merchants receive an `expiresAt` (now + 24h at creation).

---

## Listing

### Responsibility

Represents an item listed for sale within a shop, as a set of bundles with a frozen item snapshot.

### Core Models

**Model** (`listing/model.go:10-23`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Listing identifier |
| shopId | uuid.UUID | Parent shop |
| itemId | uint32 | Item template id |
| itemType | byte | Item type |
| quantity | uint16 | Total units (bundleSize * bundlesRemaining) |
| bundleSize | uint16 | Units per bundle |
| bundlesRemaining | uint16 | Bundles available |
| pricePerBundle | uint32 | Meso price per bundle |
| itemSnapshot | asset.AssetData | Frozen item payload |
| displayOrder | uint16 | Position in shop display |
| version | uint32 | Optimistic-lock counter |
| listedAt | time.Time | Time listed |

### Invariants

- id and shopId are required; pricePerBundle >= 1 and bundleSize >= 1 (`listing/builder.go:90-102`).
- On creation, quantity = bundleSize * bundleCount and version starts at 1.
- Bundle decrements during purchase are gated on the expected version (`WHERE id = ? AND version = ?`); a zero-row update signals a version conflict.
- Display order is compacted after a removal (subsequent listings' display_order decremented).

### Processors

**listing.Processor** (`listing/processor.go`) — pure database operations, no messaging: GetByShopId, GetByShopIdPaged, GetByShopIdAndDisplayOrder, CountByShopId, CountByShopIds, Create, Delete, DeleteByShopId, UpdateBundles (optimistic), DecrementDisplayOrderAfter, SetDisplayOrder, UpdateFields. Listing mutations are orchestrated by the shop processor.

---

## Blacklist

### Responsibility

Persistent per-shop deny list, keyed by character name.

### Core Models

The domain surface is a set of names plus the persisted record (id, tenant, shopId, name). There is no separate model struct; `blacklist.Processor.Names` returns `[]string`.

### Invariants

- Uniqueness per (tenant, shop, name); adding an already-present name is an idempotent no-op (insert on-conflict do-nothing).

### Processors

**blacklist.Processor** (`blacklist/processor.go`)

- Add — add a name to a shop's blacklist (idempotent).
- Remove — remove a name from a shop's blacklist.
- NamesPaged — list a shop's blacklisted names, paged (ordered by name).
- IsBlacklisted — whether a name is blacklisted for a shop.

Enforcement is in `shop.EnterShop`: a blacklisted visitor name is denied entry (a lookup error fails open, admitting the visitor). Adding a name via the shop processor also ejects that character if they are currently present.

---

## Visit

### Responsibility

Persistent per-shop visit tally keyed by visitor name, backing the hired-merchant visit-list view.

### Core Models

**Model** (`visit/model.go:4-10`)

| Field | Type | Description |
|---|---|---|
| name | string | Visitor character name |
| count | uint32 | Cumulative visit count |

### Invariants

- Uniqueness per (tenant, shop, name).
- Record is an atomic increment upsert (insert with count 1, on conflict increment count). An empty name is a no-op.

### Processors

**visit.Processor** (`visit/processor.go`)

- Record — record a visit for a name (atomic increment). Called best-effort on successful shop entry; a failure is logged, not fatal.
- ListPaged — list a shop's visits, paged (ordered by count descending).

---

## SearchCount

### Responsibility

Per-(tenant, world, item) counter of item-listing searches, backing the top-searches ranking.

### Core Models

**Model** (`searchcount/model.go:3-14`)

| Field | Type | Description |
|---|---|---|
| itemId | uint32 | Searched item template id |
| count | uint64 | Cumulative search count |

### Invariants

- Uniqueness per (tenant, world, item).
- RecordSearch is an atomic increment upsert (insert with count 1, on conflict increment count).

### Processors

**searchcount.Processor** (`searchcount/processor.go`)

- RecordSearch — increment the counter for a (world, item). Driven by the record-item-search command.
- GetTop — return the highest-count items for a world, ordered by count descending, limited by the caller-supplied limit.

---

## Frederick

### Responsibility

Stores unsold items and accumulated mesos from closed hired merchants until the owner retrieves them or they expire.

### Core Models

**ItemModel** (`frederick/model.go:9-23`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Stored item identifier |
| characterId | uint32 | Owning character |
| itemId | uint32 | Item template id |
| itemType | byte | Item type |
| quantity | uint16 | Item quantity |
| itemSnapshot | asset.AssetData | Frozen item payload |

**MesoModel** (`frederick/model.go:25-33`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Stored meso record identifier |
| characterId | uint32 | Owning character |
| amount | uint32 | Meso amount |

### Invariants

- Items and mesos expire after 100 days (`CleanupAge`).
- Notifications follow a tiered schedule: 2, 5, 10, 15, 30, 60, 90 days after storage.
- A character cannot place a new shop while items or mesos are pending at Frederick (`HasItemsOrMesos` / `HasPending`).

### Processors

**frederick.Processor** (`frederick/processor.go`)

- StoreItems, StoreMesos — persist unsold items / meso balance for a character.
- GetItems, GetMesos — retrieve stored items/mesos.
- ClearItems, ClearMesos, ClearNotifications — remove records after retrieval.
- CreateNotification — create a notification record starting at day 2.
- HasPending — whether the character has any pending items or mesos.

**Cleanup Task** (`frederick/task.go`)

Background task (6h interval). Deletes items and mesos older than 100 days across all tenants.

**Notification Task** (`frederick/notification_task.go`)

Background task (1h interval). Emits a Frederick-notification event per due record, then advances to the next tier or deletes the record when the final tier is reached.

---

## Message

### Responsibility

Persists chat messages sent within a shop.

### Core Models

**Model** (`message/model.go:9-31`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Message identifier |
| shopId | uuid.UUID | Parent shop |
| characterId | uint32 | Sender character |
| content | string | Message text |
| sentAt | time.Time | Timestamp |

### Processors

**message.Processor** (`message/processor.go`)

- SendMessage — persist a chat message for a shop.
- GetMessages — retrieve a shop's messages.

The shop processor's SendMessage wraps this and computes the sender's slot (0 = owner, otherwise the visitor's 1-indexed position).

---

## Visitor

### Responsibility

Tracks the characters currently present inside a shop, using Redis. This is transient occupancy, distinct from the persistent blacklist (deny list) and visit list (visit tally).

### Processors

**visitor.Registry** (`visitor/registry.go`)

- AddVisitor — add a character to a shop's visitor set (scored by insertion time) and reverse lookup.
- RemoveVisitor — remove a character from a shop's visitor set and reverse lookup.
- GetVisitors, GetVisitorCount — query a shop's current visitors (ordered by insertion).
- RemoveAllVisitors — bulk-remove all visitors from a shop, returning the ejected ids.
- GetShopForCharacter — reverse lookup from character to shop.

Visitor slots are 1-indexed from insertion order; slot 0 is reserved for the owner. Entry is capacity-limited to `MaxVisitors` (3).
</content>
