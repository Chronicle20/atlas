# Domain

## Shop

### Responsibility

Represents a character shop or hired merchant placed in a Free Market room.

### Core Models

**Model** (`shop/model.go`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Shop identifier |
| characterId | uint32 | Owner character |
| shopType | ShopType | CharacterShop (1) or HiredMerchant (2) |
| state | State | Current lifecycle state |
| title | string | Shop display name |
| mapId | uint32 | Map where the shop is placed |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| permitItemId | uint32 | Permit item used for placement |
| createdAt | time.Time | Creation timestamp |
| expiresAt | *time.Time | Expiration time (hired merchants: 24h) |
| closedAt | *time.Time | Closure timestamp |
| closeReason | CloseReason | Reason for closure |
| mesoBalance | uint32 | Accumulated meso balance (hired merchants) |

**ShopType**

| Value | Name |
|---|---|
| 1 | CharacterShop |
| 2 | HiredMerchant |

**CloseReason**

| Value | Name |
|---|---|
| 0 | None |
| 1 | SoldOut |
| 2 | ManualClose |
| 3 | Disconnect |
| 4 | Expired |
| 5 | ServerRestart |
| 6 | Empty |

**PurchaseResult** (`shop/processor.go`)

Returned from PurchaseBundle. Contains listing details, cost breakdown (TotalCost, Fee, NetAmount), buyer/owner info, and whether the shop closed as a result.

### Invariants

- Shop ID, characterId, shopType, and state are required (builder validation).
- One active shop per type per character.
- Shop must be in a Free Market room (validated against hardcoded map ID set).
- Shop cannot be placed within 130px (Manhattan distance) of a portal.
- Shop cannot be placed within 100px (Manhattan distance) of an existing shop.
- Shop cannot be placed if the owner has items or mesos pending at Frederick.
- Maximum 16 listings per shop.
- Maximum 3 visitors per shop.
- Listings require pricePerBundle >= 1, bundleSize >= 1, bundleCount >= 1.
- Pets, cash items, and untradeable items cannot be listed.
- Purchase uses optimistic locking on listing version to prevent concurrent conflicts.

### State Transitions

```
Draft -> Open       (requires at least one listing)
Open -> Maintenance
Open -> Closed
Maintenance -> Open (if listings remain)
Maintenance -> Closed (automatic if no listings remain)
Draft -> Closed
```

Maintenance ejects all visitors. Closing ejects all visitors, removes registry entries, and removes map index entries. For hired merchants, closing moves unsold items and mesos to Frederick.

### Processors

**shop.Processor** (`shop/processor.go`)

- GetById, GetByCharacterId, GetByMapId, GetListings, GetExpired
- CreateShop — validates placement, creates entity, registers in Redis
- OpenShop — transitions Draft to Open, adds to map index
- EnterMaintenance — transitions Open to Maintenance, ejects visitors
- ExitMaintenance — transitions Maintenance to Open (or Closed if empty)
- CloseShop — transitions to Closed, cleans up registry/index/visitors, stores to Frederick (hired merchants)
- AddListing, RemoveListing, UpdateListing — manage listings in Draft or Maintenance states
- PurchaseBundle — deducts bundles with optimistic lock, calculates fee, closes shop if sold out, accumulates meso balance (hired merchants)
- EnterShop, ExitShop, EjectAllVisitors, GetVisitors — visitor management via Redis

**Fee Calculation** (`shop/processor.go:GetFee`)

| Meso Amount | Fee Rate |
|---|---|
| < 100,000 | 0% |
| 100,000 - 999,999 | 0.8% |
| 1,000,000 - 4,999,999 | 1.8% |
| 5,000,000 - 9,999,999 | 3% |
| 10,000,000 - 24,999,999 | 4% |
| 25,000,000 - 99,999,999 | 5% |
| >= 100,000,000 | 6% |

Fee is calculated using integer division.

**Validation** (`shop/validation.go`)

- IsFreemarketRoom — checks against hardcoded Free Market room map IDs (Henesys, Perion, El Nath, Ludibrium, Hidden Street)
- IsNearPortal — fetches portal positions from atlas-data REST, checks Manhattan distance < 130
- IsNearExistingShop — checks Manhattan distance < 100 against shops on same map
- IsListableItem — rejects pets (ClassificationPet), cash items (TypeValueCash), untradeable items (FlagUntradeable)

**Expiration Reaper** (`shop/reaper.go`)

Background goroutine (30s interval). Queries expired shops across all tenants using `WithoutTenantFilter`. Reconstructs tenant context from entity columns and calls CloseShop with CloseReasonExpired.

---

## Listing

### Responsibility

Represents an item listed for sale within a shop.

### Core Models

**Model** (`listing/model.go`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Listing identifier |
| shopId | uuid.UUID | Parent shop |
| itemId | uint32 | Item template ID |
| itemType | byte | Item type |
| quantity | uint16 | Total quantity (bundleSize * bundlesRemaining) |
| bundleSize | uint16 | Items per bundle |
| bundlesRemaining | uint16 | Bundles available |
| pricePerBundle | uint32 | Meso price per bundle |
| itemSnapshot | json.RawMessage | Serialized item data |
| displayOrder | uint16 | Position in shop display |
| listedAt | time.Time | Time listed |

### Invariants

- id, shopId are required (builder validation).
- pricePerBundle >= 1, bundleSize >= 1 (builder validation).
- Display order is maintained contiguously; removing a listing decrements orders of subsequent listings.
- Optimistic locking via version column on bundle updates during purchase.

### Processors

Listing operations are performed through the shop.Processor. The listing package exposes exported provider functions (exports.go) for cross-package access.

---

## Frederick

### Responsibility

Stores unsold items and accumulated mesos from closed hired merchants. Items and mesos are held until the owner retrieves them or they expire after 100 days.

### Core Models

**ItemModel** (`frederick/model.go`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Stored item identifier |
| characterId | uint32 | Owning character |
| itemId | uint32 | Item template ID |
| itemType | byte | Item type |
| quantity | uint16 | Item quantity |
| itemSnapshot | json.RawMessage | Serialized item data |

**MesoModel** (`frederick/model.go`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Stored meso record identifier |
| characterId | uint32 | Owning character |
| amount | uint32 | Meso amount |

**StoredItem** (`frederick/processor.go`)

| Field | Type |
|---|---|
| ItemId | uint32 |
| ItemType | byte |
| Quantity | uint16 |
| ItemSnapshot | []byte |

### Invariants

- Items and mesos expire after 100 days.
- Notifications follow tiered schedule: 2, 5, 10, 15, 30, 60, 90 days after storage.
- A character cannot place a new shop while items or mesos are pending at Frederick.

### Processors

**frederick.Processor** (`frederick/processor.go`)

- StoreItems — saves unsold listing items for a character
- StoreMesos — saves meso balance for a character
- GetItems, GetMesos — retrieve stored items/mesos
- ClearItems, ClearMesos, ClearNotifications — remove records after retrieval
- CreateNotification — creates notification record starting at day 2

**Cleanup Reaper** (`frederick/processor.go:StartCleanupReaper`)

Background goroutine (6h interval). Deletes items and mesos older than 100 days across all tenants using `WithoutTenantFilter`.

**Notification Scheduler** (`frederick/notification.go:StartNotificationScheduler`)

Background goroutine (1h interval). Queries due notifications across all tenants. Produces `FREDERICK_NOTIFICATION` status event. Advances to next tier or deletes notification record if final tier reached.

---

## Message

### Responsibility

Persists chat messages sent within a shop.

### Core Models

**Model** (`message/model.go`)

| Field | Type | Description |
|---|---|---|
| id | uuid.UUID | Message identifier |
| shopId | uuid.UUID | Parent shop |
| characterId | uint32 | Sender character |
| content | string | Message text |
| sentAt | time.Time | Timestamp |

### Processors

**message.Processor** (`message/processor.go`)

- SendMessage — persists a chat message for a shop
- GetMessages — retrieves all messages for a shop ordered by sent time

---

## Visitor

### Responsibility

Tracks characters currently visiting a shop using Redis.

### Processors

**visitor.Registry** (`visitor/registry.go`)

- AddVisitor — adds character to shop visitor set and reverse lookup
- RemoveVisitor — removes character from shop visitor set and reverse lookup
- GetVisitors, GetVisitorCount — query visitors for a shop
- RemoveAllVisitors — bulk remove all visitors from a shop
- GetShopForCharacter — reverse lookup: character to shop
